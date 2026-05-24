package dimensions_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

// sensWeights returns weights summing to 1.0 for D-SENS per PRD §5.1.
func sensWeights() dimensions.SensWeights {
	return dimensions.SensWeights{
		Contract:         0.35,
		RuntimeResolve:   0.30,
		CapabilityDiscov: 0.20,
		ReasonerIntent:   0.15,
	}
}

func newStaticExchange() tau.Exchange {
	return tau.Exchange{
		ID:                "x-static",
		IntentDescription: "call payment service",
		DiscoveredAt:      time.Now(),
		Target: tau.Capability{
			ID:            "payment-svc",
			DiscoveryMode: tau.Static,
			ContractURI:   "https://api.example.com/openapi.yaml",
		},
		Initiator: tau.Principal{
			ID:           "agent-1",
			HumanInLoop:  true,
			Organization: "org-a",
		},
	}
}

func newDynamicExchange() tau.Exchange {
	return tau.Exchange{
		ID:                "x-dynamic",
		IntentDescription: "discover and invoke best available tool",
		DiscoveredAt:      time.Now(),
		Target: tau.Capability{
			ID:            "",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
		Initiator: tau.Principal{
			ID:           "agent-2",
			HumanInLoop:  false,
			Organization: "org-b",
		},
	}
}

func TestDSens_Bounded(t *testing.T) {
	t.Parallel()
	w := sensWeights()
	cases := []tau.Exchange{newStaticExchange(), newDynamicExchange()}
	for _, x := range cases {
		t.Run(x.ID, func(t *testing.T) {
			t.Parallel()
			score, err := dimensions.ScoreDSens(context.Background(), x, w, nil)
			if err != nil {
				t.Fatalf("ScoreDSens error: %v", err)
			}
			if score.Value < 0 || score.Value > 1 {
				t.Fatalf("ScoreDSens value %f out of [0,1]", score.Value)
			}
		})
	}
}

func TestDSens_StaticLowerThanDynamic(t *testing.T) {
	t.Parallel()
	w := sensWeights()
	static, err := dimensions.ScoreDSens(context.Background(), newStaticExchange(), w, nil)
	if err != nil {
		t.Fatalf("static: %v", err)
	}
	dynamic, err := dimensions.ScoreDSens(context.Background(), newDynamicExchange(), w, nil)
	if err != nil {
		t.Fatalf("dynamic: %v", err)
	}
	if static.Value >= dynamic.Value {
		t.Fatalf("expected static (%f) < dynamic (%f)", static.Value, dynamic.Value)
	}
}

func TestDSens_ProbeWeightsSumToOne(t *testing.T) {
	t.Parallel()
	w := sensWeights()
	sum := w.Contract + w.RuntimeResolve + w.CapabilityDiscov + w.ReasonerIntent
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("probe weights sum = %f, want 1.0", sum)
	}
}

func TestDSens_ProbesMapPopulated(t *testing.T) {
	t.Parallel()
	w := sensWeights()
	score, err := dimensions.ScoreDSens(context.Background(), newDynamicExchange(), w, nil)
	if err != nil {
		t.Fatalf("ScoreDSens error: %v", err)
	}
	expected := []string{"S_contract", "S_runtime_resolve", "S_capability_discovery", "S_reasoner_intent"}
	for _, k := range expected {
		if _, ok := score.Probes[k]; !ok {
			t.Errorf("probe %q missing from score.Probes", k)
		}
	}
}
