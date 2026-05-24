package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
	"github.com/agbruneau/taugo/internal/tau"
)

// AnnotatedEntry wraps an AgentMeshExchange with its expected_regime,
// computed by piping the exchange through app.ToTauExchange + Dispatcher.Decide.
type AnnotatedEntry struct {
	agentmeshkafka.AgentMeshExchange
	ExpectedRegime string `json:"expected_regime"`
}

// Annotator converts AgentMeshExchange records to AnnotatedEntry by running
// each through the production Dispatcher. It satisfies the interface used by
// GenerateAnnotated; callers provide the concrete dispatcher via app.NewDispatcher.
type Annotator interface {
	Decide(ctx context.Context, x tau.Exchange) (tau.Decision, error)
}

// regimeString converts tau.Regime to its canonical string form.
func regimeString(r tau.Regime) string {
	switch r {
	case tau.Deterministe:
		return "Deterministe"
	case tau.Probabiliste:
		return "Probabiliste"
	default:
		return "Refus"
	}
}

// DistributionProfile selects the mixing rule for generated traces.
type DistributionProfile string

const (
	ProfileBalanced   DistributionProfile = "balanced"
	ProfileI4Heavy    DistributionProfile = "i4-heavy"
	ProfileRefusHeavy DistributionProfile = "refus-heavy"
)

// branchWeights holds the integer percentage weights for the six branches.
// Weights must sum to 100.
type branchWeights struct {
	refusFrontiere int // Refus hors frontière (4 conditions classiques)
	refusI3        int // Refus I3 ontologique (D-AUTORITÉ sans attestation)
	refusI4        int // Refus I4 incohérence (s < θ_sens ∧ i ≥ θ_inv)
	deterministe   int // Régime Deterministe
	probabiliste   int // Régime Probabiliste
	hysteresis     int // Zone d'hystérèse / bord ambigu
}

func weightsFor(p DistributionProfile) (branchWeights, error) {
	switch p {
	case ProfileBalanced:
		return branchWeights{15, 15, 10, 25, 25, 10}, nil
	case ProfileI4Heavy:
		// I4 incohérence at 60 %; remaining spread over the others.
		return branchWeights{10, 10, 60, 5, 5, 10}, nil
	case ProfileRefusHeavy:
		// ~50 % total refus (all three kinds), rest split between D and P.
		return branchWeights{20, 20, 10, 20, 20, 10}, nil
	default:
		return branchWeights{}, fmt.Errorf("unknown distribution profile %q", p)
	}
}

// Generator produces a deterministic stream of AgentMeshExchange records.
// Same seed + same count + same profile → byte-identical output.
type Generator struct {
	rng *rand.Rand
}

// NewGenerator returns a Generator seeded with the given value.
// Uses PCG, which is part of the standard math/rand/v2 API and produces
// deterministic output independent of Go version upgrades (PCG spec is fixed).
func NewGenerator(seed int64) *Generator {
	src := rand.NewPCG(uint64(seed), uint64(seed)^0xdeadbeef_cafebabe)
	return &Generator{rng: rand.New(src)} //nolint:gosec // intentional: reproducibility requires math/rand, not crypto/rand
}

// GenerateAnnotated writes n exchanges to w as JSONL, each enriched with an
// expected_regime field derived by running app.ToTauExchange + a.Decide on
// every AgentMeshExchange. The base RNG stream is identical to Generate so
// that non-annotated fields are byte-for-byte reproducible for the same seed.
func (g *Generator) GenerateAnnotated(ctx context.Context, w io.Writer, n int, profile DistributionProfile, a Annotator) error {
	wts, err := weightsFor(profile)
	if err != nil {
		return err
	}
	if n < 1 {
		return fmt.Errorf("count must be >= 1")
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	plan := buildPlan(n, wts, g)
	g.rng.Shuffle(len(plan), func(i, j int) { plan[i], plan[j] = plan[j], plan[i] })

	base := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	for idx, fn := range plan {
		x := fn(idx)
		x.DiscoveredAt = base.Add(time.Duration(idx) * time.Minute)
		x.SourceTopic = "agentic.synth"
		x.SourceOffset = int64(idx)

		d, derr := a.Decide(ctx, app.ToTauExchange(x))
		if derr != nil {
			return fmt.Errorf("annotate line %d: %w", idx, derr)
		}
		entry := AnnotatedEntry{
			AgentMeshExchange: x,
			ExpectedRegime:    regimeString(d.Regime),
		}
		if err := enc.Encode(&entry); err != nil {
			return fmt.Errorf("encode line %d: %w", idx, err)
		}
	}
	return nil
}

// Generate writes n exchanges to w as JSONL.
func (g *Generator) Generate(w io.Writer, n int, profile DistributionProfile) error {
	wts, err := weightsFor(profile)
	if err != nil {
		return err
	}
	if n < 1 {
		return fmt.Errorf("count must be >= 1")
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	plan := buildPlan(n, wts, g)

	// Shuffle deterministically so branch order does not correlate with index.
	g.rng.Shuffle(len(plan), func(i, j int) { plan[i], plan[j] = plan[j], plan[i] })

	base := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	for idx, fn := range plan {
		x := fn(idx)
		x.DiscoveredAt = base.Add(time.Duration(idx) * time.Minute)
		x.SourceTopic = "agentic.synth"
		x.SourceOffset = int64(idx)
		if err := enc.Encode(&x); err != nil {
			return fmt.Errorf("encode line %d: %w", idx, err)
		}
	}
	return nil
}

// buildPlan allocates a slice of build functions proportional to weights.
// Remainder rows are filled with hysteresis entries.
func buildPlan(n int, wts branchWeights, g *Generator) []func(int) agentmeshkafka.AgentMeshExchange {
	plan := make([]func(int) agentmeshkafka.AgentMeshExchange, 0, n)
	addN := func(count int, fn func(int) agentmeshkafka.AgentMeshExchange) {
		for i := 0; i < count; i++ {
			plan = append(plan, fn)
		}
	}
	addN(n*wts.refusFrontiere/100, g.buildRefusFrontiere)
	addN(n*wts.refusI3/100, g.buildRefusI3)
	addN(n*wts.refusI4/100, g.buildRefusI4)
	addN(n*wts.deterministe/100, g.buildDeterministe)
	addN(n*wts.probabiliste/100, g.buildProbabiliste)
	// Fill remainder with hysteresis to reach exactly n entries.
	for len(plan) < n {
		plan = append(plan, g.buildHysteresis)
	}
	return plan
}

// intents is a fixed-size list used to vary IntentDescription deterministically
// without calling rng at construction time (preserves shuffle determinism).
var intents = [8]string{ //nolint:gochecknoglobals // compile-time constant table; immutable after init
	"query customer profile",
	"transfer file to downstream agent",
	"validate policy document",
	"dispatch notification to subscriber",
	"aggregate sensor readings",
	"update delegation chain",
	"invoke compliance check",
	"retrieve audit log segment",
}

func intentFor(i int) string { return intents[i%len(intents)] }

// Branch builders — pure functions of (index, g.rng).

func (g *Generator) buildRefusFrontiere(i int) agentmeshkafka.AgentMeshExchange {
	// HumanInLoop=true + static discovery → violates Inside() condition.
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-rf-%06d", i),
		IntentDescription: intentFor(i),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              fmt.Sprintf("agent-rf-%06d", i),
			HumanInLoop:     true,
			Organization:    "org-synth",
			DelegationDepth: 0,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "tool-static",
			DiscoveryMode: "static",
			ContractURI:   "https://api.example.org/v1/op",
		},
	}
}

func (g *Generator) buildRefusI3(i int) agentmeshkafka.AgentMeshExchange {
	// DynamicMCP + DelegationDepth >= 3 + no attestation → refus I3 ontologique.
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-r3-%06d", i),
		IntentDescription: intentFor(i),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              fmt.Sprintf("agent-r3-%06d", i),
			HumanInLoop:     false,
			Organization:    fmt.Sprintf("org-x-%02d", i%5),
			DelegationDepth: 3 + (i % 3),
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "tool-dyn-mcp",
			DiscoveryMode: "dynamic_mcp",
		},
	}
}

func (g *Generator) buildRefusI4(i int) agentmeshkafka.AgentMeshExchange {
	// Very short intent (low D-SENS score) + fully enumerated contract
	// (high D-INVARIANT) → incoherent pair → refus I4.
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-r4-%06d", i),
		IntentDescription: "x",
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              fmt.Sprintf("agent-r4-%06d", i),
			HumanInLoop:     false,
			Organization:    "org-i4",
			DelegationDepth: 1,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "tool-i4-strict",
			DiscoveryMode: "dynamic_mcp",
			ContractURI:   "https://api.example.org/v1/strict",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur:   "ietf",
			Reference:  "draft-i4-test",
			Marqueur:   "Hypothèse",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func (g *Generator) buildDeterministe(i int) agentmeshkafka.AgentMeshExchange {
	// Attestation present + DelegationDepth <= 1 → low composite scores → Deterministe.
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-d-%06d", i),
		IntentDescription: intentFor(i),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              fmt.Sprintf("agent-d-%06d", i),
			HumanInLoop:     false,
			Organization:    "org-d",
			DelegationDepth: 1,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "tool-d",
			DiscoveryMode: "dynamic_mcp",
			ContractURI:   "https://api.example.org/v1/op",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur:   "desjardins-iam",
			Reference:  "ref-d-stable",
			Marqueur:   "Probable",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func (g *Generator) buildProbabiliste(i int) agentmeshkafka.AgentMeshExchange {
	// Longer, high-cardinality intent string → high D-SENS → Probabiliste.
	suffix := g.rng.Int64()
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-p-%06d", i),
		IntentDescription: fmt.Sprintf("multi-step %s orchestration task %d", intentFor(i), suffix),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              fmt.Sprintf("agent-p-%06d", i),
			HumanInLoop:     false,
			Organization:    "org-p",
			DelegationDepth: 2,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "tool-p",
			DiscoveryMode: "dynamic_a2a",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur:   "desjardins-iam",
			Reference:  "ref-p-dyn",
			Marqueur:   "Probable",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func (g *Generator) buildHysteresis(i int) agentmeshkafka.AgentMeshExchange {
	// Mid-range intent near the (Deterministe, Probabiliste) threshold.
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-h-%06d", i),
		IntentDescription: fmt.Sprintf("boundary case %d", i),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              fmt.Sprintf("agent-h-%06d", i),
			HumanInLoop:     false,
			Organization:    "org-h",
			DelegationDepth: 1,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "tool-h",
			DiscoveryMode: "dynamic_mcp",
			ContractURI:   "https://api.example.org/v1/op",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur:   "desjardins-iam",
			Reference:  "ref-h-bnd",
			Marqueur:   "Probable",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}
