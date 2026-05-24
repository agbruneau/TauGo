package dimensions

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/tau"
)

// SensWeights holds the calibrated weights for the four D-SENS probes.
// The sum must equal 1.0 (enforced at construction by the calibration layer).
// Initial values from PRD §5.1: {0.35, 0.30, 0.20, 0.15}.
type SensWeights struct {
	Contract         float64 // weight for S_contract
	RuntimeResolve   float64 // weight for S_runtime_resolve
	CapabilityDiscov float64 // weight for S_capability_discovery
	ReasonerIntent   float64 // weight for S_reasoner_intent
}

// DefaultSensWeights returns the initial weights from PRD §5.1.
// Status: Hypothesis — to be corroborated on AgentMeshKafka traces in M4.
func DefaultSensWeights() SensWeights {
	return SensWeights{
		Contract:         0.35,
		RuntimeResolve:   0.30,
		CapabilityDiscov: 0.20,
		ReasonerIntent:   0.15,
	}
}

// ScoreDSens computes the D-SENS dimension score for exchange x.
// llmClient may be nil; in that case S_reasoner_intent returns 0.
// Returns a Score with Value in [0,1] and all probe values populated.
func ScoreDSens(ctx context.Context, x tau.Exchange, w SensWeights, llmClient llm.Client) (Score, error) {
	sContract := probeContract(x)
	sRuntime := probeRuntimeResolve(x)
	sDiscov := probeCapabilityDiscovery(x)
	sReasoner, err := probeReasonerIntent(ctx, x, llmClient)
	if err != nil {
		return Score{}, err
	}

	value := w.Contract*sContract +
		w.RuntimeResolve*sRuntime +
		w.CapabilityDiscov*sDiscov +
		w.ReasonerIntent*sReasoner

	return Score{
		Value: clamp01(value),
		Probes: map[string]float64{
			"S_contract":             sContract,
			"S_runtime_resolve":      sRuntime,
			"S_capability_discovery": sDiscov,
			"S_reasoner_intent":      sReasoner,
		},
		Weights: map[string]float64{
			"S_contract":             w.Contract,
			"S_runtime_resolve":      w.RuntimeResolve,
			"S_capability_discovery": w.CapabilityDiscov,
			"S_reasoner_intent":      w.ReasonerIntent,
		},
		ComputedAt: time.Now(),
	}, nil
}

// probeContract (S_contract) — presence of a published, versioned, opposable
// contract for the target capability (PRD §5.1). Returns 0 if a non-empty
// ContractURI is present (contract wired = fixed before interaction = pole 0).
// Returns 1 if no contract (meaning is negotiated at runtime = pole 1).
func probeContract(x tau.Exchange) float64 {
	if x.Target.ContractURI != "" {
		return 0.0
	}
	return 1.0
}

// probeRuntimeResolve (S_runtime_resolve) — runtime semantic resolution
// (embedding, NL parsing). Returns 1 if the exchange has a non-empty
// IntentDescription that suggests NL-level interpretation, 0 if intent is empty
// (implying a static protocol invocation).
func probeRuntimeResolve(x tau.Exchange) float64 {
	if x.IntentDescription == "" {
		return 0.0
	}
	return 1.0
}

// probeCapabilityDiscovery (S_capability_discovery) — dynamic discovery
// via MCP list_tools, A2A, or AGNTCY (PRD §5.1). Returns 1 if DiscoveryMode
// is anything other than Static.
func probeCapabilityDiscovery(x tau.Exchange) float64 {
	if x.Target.DiscoveryMode == tau.Static {
		return 0.0
	}
	return 1.0
}

// probeReasonerIntent (S_reasoner_intent) — probabilistic reasoner intent
// interpretation (PRD §5.1). Delegates to the LLM client's Interpret method.
// Returns 0 if llmClient is nil (no reasoner available).
func probeReasonerIntent(ctx context.Context, x tau.Exchange, c llm.Client) (float64, error) {
	if c == nil {
		return 0.0, nil
	}
	return c.Interpret(ctx, x.IntentDescription)
}
