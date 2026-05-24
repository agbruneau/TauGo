package dimensions

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// InvariantWeights holds the calibrated weights for the four D-INVARIANT probes.
// Initial values from PRD §5.3: {0.30, 0.25, 0.25, 0.20}.
type InvariantWeights struct {
	EventRegistry       float64 // weight for I_event_registry
	IdempotencyDerived  float64 // weight for I_idempotency_derived
	CapabilityMediation float64 // weight for I_capability_mediation
	EnumeratedPlan      float64 // weight for I_enumerated_plan (inverted probe)
}

// DefaultInvariantWeights returns the initial weights from PRD §5.3.
// Status: Hypothesis — to be corroborated on AgentMeshKafka traces in M4.
func DefaultInvariantWeights() InvariantWeights {
	return InvariantWeights{
		EventRegistry:       0.30,
		IdempotencyDerived:  0.25,
		CapabilityMediation: 0.25,
		EnumeratedPlan:      0.20,
	}
}

// ScoreDInvariant computes the D-INVARIANT dimension score for exchange x.
// Returns a Score with Value in [0,1] and all probe values populated.
// I4 coherence constraint (D-INVARIANT constrained by D-SENS) is enforced
// at the orchestration layer (step 5), not here.
func ScoreDInvariant(_ context.Context, x tau.Exchange, w InvariantWeights) (Score, error) {
	iRegistry := probeEventRegistry(x)
	iIdempotency := probeIdempotencyDerived(x)
	iMediation := probeCapabilityMediation(x)
	iEnumerated := probeEnumeratedPlan(x)

	value := w.EventRegistry*iRegistry +
		w.IdempotencyDerived*iIdempotency +
		w.CapabilityMediation*iMediation +
		w.EnumeratedPlan*iEnumerated

	return Score{
		Value: clamp01(value),
		Probes: map[string]float64{
			"I_event_registry":       iRegistry,
			"I_idempotency_derived":  iIdempotency,
			"I_capability_mediation": iMediation,
			"I_enumerated_plan":      iEnumerated,
		},
		Weights: map[string]float64{
			"I_event_registry":       w.EventRegistry,
			"I_idempotency_derived":  w.IdempotencyDerived,
			"I_capability_mediation": w.CapabilityMediation,
			"I_enumerated_plan":      w.EnumeratedPlan,
		},
		ComputedAt: time.Now(),
	}, nil
}

// probeEventRegistry (I_event_registry) — runtime-traced effect registry.
// Returns 1 if Context contains key "event_registry" with truthy bool value.
func probeEventRegistry(x tau.Exchange) float64 {
	if v, ok := x.Context["event_registry"]; ok {
		if b, isBool := v.(bool); isBool && b {
			return 1.0
		}
	}
	return 0.0
}

// probeIdempotencyDerived (I_idempotency_derived) — idempotency key derived
// from intent vs imposed at design time. Returns 1 if Context contains
// "idempotency_key_mode" == "derived".
func probeIdempotencyDerived(x tau.Exchange) float64 {
	if v, ok := x.Context["idempotency_key_mode"]; ok {
		if s, isStr := v.(string); isStr && s == "derived" {
			return 1.0
		}
	}
	return 0.0
}

// probeCapabilityMediation (I_capability_mediation) — capability mediation
// negotiated during the exchange. Returns 1 if Context contains
// "capability_mediation" with truthy bool value, or if DiscoveryMode != Static
// (dynamic discovery implies runtime mediation).
func probeCapabilityMediation(x tau.Exchange) float64 {
	if v, ok := x.Context["capability_mediation"]; ok {
		if b, isBool := v.(bool); isBool && b {
			return 1.0
		}
	}
	if x.Target.DiscoveryMode != tau.Static {
		return 1.0
	}
	return 0.0
}

// probeEnumeratedPlan (I_enumerated_plan) — inverted probe: an enumerated
// step plan known at design time reduces D-INVARIANT (support frozen at design
// time = pole 0). Returns 0 if Context contains "enumerated_plan" == true,
// 1 if absent or false. Also returns 0 if ContractURI is present (contract
// implies pre-defined plan).
func probeEnumeratedPlan(x tau.Exchange) float64 {
	if x.Target.ContractURI != "" {
		return 0.0
	}
	if v, ok := x.Context["enumerated_plan"]; ok {
		if b, isBool := v.(bool); isBool && b {
			return 0.0
		}
	}
	return 1.0
}
