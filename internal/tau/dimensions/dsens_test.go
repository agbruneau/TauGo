package dimensions_test

import (
	"context"
	"fmt"
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

// TestDefaultSensWeights_StructureAndSum verifies that DefaultSensWeights returns
// a non-zero struct whose weights sum to 1.0 (PRD §5.1).
func TestDefaultSensWeights_StructureAndSum(t *testing.T) {
	t.Parallel()
	w := dimensions.DefaultSensWeights()
	if w.Contract == 0 && w.RuntimeResolve == 0 && w.CapabilityDiscov == 0 && w.ReasonerIntent == 0 {
		t.Fatal("DefaultSensWeights returned all-zero struct")
	}
	sum := w.Contract + w.RuntimeResolve + w.CapabilityDiscov + w.ReasonerIntent
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("DefaultSensWeights sum = %f, want 1.0", sum)
	}
}

// TestDSens_EmptyIntent covers probeRuntimeResolve returning 0 (empty IntentDescription).
func TestDSens_EmptyIntent(t *testing.T) {
	t.Parallel()
	x := newStaticExchange()
	x.IntentDescription = ""
	w := sensWeights()
	score, err := dimensions.ScoreDSens(context.Background(), x, w, nil)
	if err != nil {
		t.Fatalf("ScoreDSens error: %v", err)
	}
	if score.Probes["S_runtime_resolve"] != 0 {
		t.Fatalf("expected S_runtime_resolve=0 for empty intent, got %f", score.Probes["S_runtime_resolve"])
	}
}

// TestDSens_LLMClientError covers probeReasonerIntent propagating an error.
func TestDSens_LLMClientError(t *testing.T) {
	t.Parallel()
	_, err := dimensions.ScoreDSens(context.Background(), newDynamicExchange(), sensWeights(), &errClient{})
	if err == nil {
		t.Fatal("expected error from errClient, got nil")
	}
}

// errClient is a test double that always returns an error from Interpret.
type errClient struct{}

func (e *errClient) Fingerprint() string { return "err-client-v0" }
func (e *errClient) Interpret(_ context.Context, _ string) (float64, error) {
	return 0, errClientErr
}

var errClientErr = fmt.Errorf("stub error")
