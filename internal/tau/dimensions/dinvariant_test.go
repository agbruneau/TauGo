package dimensions_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

func invariantWeights() dimensions.InvariantWeights {
	return dimensions.InvariantWeights{
		EventRegistry:       0.30,
		IdempotencyDerived:  0.25,
		CapabilityMediation: 0.25,
		EnumeratedPlan:      0.20,
	}
}

func newFrozenSupportExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-frozen-support",
		DiscoveredAt: time.Now(),
		Target: tau.Capability{
			ID:            "batch-processor",
			DiscoveryMode: tau.Static,
			ContractURI:   "https://api.example.com/batch/v1",
		},
		Initiator: tau.Principal{
			ID:              "scheduler",
			HumanInLoop:     true,
			Organization:    "org-a",
			DelegationDepth: 0,
		},
		// No context: implies enumerated plan at design time
	}
}

func newTracedSupportExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-traced-support",
		DiscoveredAt: time.Now(),
		Target: tau.Capability{
			ID:            "dynamic-tool",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
		Initiator: tau.Principal{
			ID:              "llm-agent",
			HumanInLoop:     false,
			Organization:    "org-b",
			DelegationDepth: 3,
		},
		Context: map[string]any{
			"event_registry":       true,
			"idempotency_key_mode": "derived",
			"capability_mediation": true,
		},
	}
}

func TestDInvariant_Bounded(t *testing.T) {
	t.Parallel()
	w := invariantWeights()
	for _, x := range []tau.Exchange{newFrozenSupportExchange(), newTracedSupportExchange()} {
		t.Run(x.ID, func(t *testing.T) {
			t.Parallel()
			score, err := dimensions.ScoreDInvariant(context.Background(), x, w)
			if err != nil {
				t.Fatalf("ScoreDInvariant error: %v", err)
			}
			if score.Value < 0 || score.Value > 1 {
				t.Fatalf("ScoreDInvariant value %f out of [0,1]", score.Value)
			}
		})
	}
}

func TestDInvariant_FrozenLowerThanTraced(t *testing.T) {
	t.Parallel()
	w := invariantWeights()
	frozen, err := dimensions.ScoreDInvariant(context.Background(), newFrozenSupportExchange(), w)
	if err != nil {
		t.Fatalf("frozen: %v", err)
	}
	traced, err := dimensions.ScoreDInvariant(context.Background(), newTracedSupportExchange(), w)
	if err != nil {
		t.Fatalf("traced: %v", err)
	}
	if frozen.Value >= traced.Value {
		t.Fatalf("expected frozen (%f) < traced (%f)", frozen.Value, traced.Value)
	}
}

func TestDInvariant_ProbeWeightsSumToOne(t *testing.T) {
	t.Parallel()
	w := invariantWeights()
	sum := w.EventRegistry + w.IdempotencyDerived + w.CapabilityMediation + w.EnumeratedPlan
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("invariant probe weights sum = %f, want 1.0", sum)
	}
}

func TestDInvariant_ProbesMapPopulated(t *testing.T) {
	t.Parallel()
	w := invariantWeights()
	score, err := dimensions.ScoreDInvariant(context.Background(), newTracedSupportExchange(), w)
	if err != nil {
		t.Fatalf("ScoreDInvariant error: %v", err)
	}
	expected := []string{
		"I_event_registry",
		"I_idempotency_derived",
		"I_capability_mediation",
		"I_enumerated_plan",
	}
	for _, k := range expected {
		if _, ok := score.Probes[k]; !ok {
			t.Errorf("probe %q missing from score.Probes", k)
		}
	}
}

// TestDefaultInvariantWeights_StructureAndSum verifies DefaultInvariantWeights
// returns a non-zero struct whose weights sum to 1.0 (PRD §5.3).
func TestDefaultInvariantWeights_StructureAndSum(t *testing.T) {
	t.Parallel()
	w := dimensions.DefaultInvariantWeights()
	if w.EventRegistry == 0 && w.IdempotencyDerived == 0 && w.CapabilityMediation == 0 && w.EnumeratedPlan == 0 {
		t.Fatal("DefaultInvariantWeights returned all-zero struct")
	}
	sum := w.EventRegistry + w.IdempotencyDerived + w.CapabilityMediation + w.EnumeratedPlan
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("DefaultInvariantWeights sum = %f, want 1.0", sum)
	}
}

// TestDInvariant_EnumeratedPlan_ExplicitTrue covers probeEnumeratedPlan returning 0
// when Context["enumerated_plan"] == true (inverted probe, PRD §5.3).
func TestDInvariant_EnumeratedPlan_ExplicitTrue(t *testing.T) {
	t.Parallel()
	x := newFrozenSupportExchange()
	x.Target.ContractURI = "" // remove ContractURI so the Context branch is reached
	x.Context = map[string]any{"enumerated_plan": true}
	w := invariantWeights()
	score, err := dimensions.ScoreDInvariant(context.Background(), x, w)
	if err != nil {
		t.Fatalf("ScoreDInvariant error: %v", err)
	}
	if score.Probes["I_enumerated_plan"] != 0 {
		t.Fatalf("expected I_enumerated_plan=0 with explicit true, got %f", score.Probes["I_enumerated_plan"])
	}
}

// TestDInvariant_CapabilityMediation_DynamicDiscovery covers probeCapabilityMediation
// returning 1 via DynamicMCP when context key is absent.
func TestDInvariant_CapabilityMediation_DynamicDiscovery(t *testing.T) {
	t.Parallel()
	x := newFrozenSupportExchange()
	x.Target.DiscoveryMode = tau.DynamicMCP
	x.Target.ContractURI = ""
	x.Context = nil // no capability_mediation key
	w := invariantWeights()
	score, err := dimensions.ScoreDInvariant(context.Background(), x, w)
	if err != nil {
		t.Fatalf("ScoreDInvariant error: %v", err)
	}
	if score.Probes["I_capability_mediation"] != 1.0 {
		t.Fatalf("expected I_capability_mediation=1.0 via DynamicMCP, got %f", score.Probes["I_capability_mediation"])
	}
}
