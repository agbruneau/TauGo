package orchestration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

type fakeLLM struct{ score float64 }

func (f fakeLLM) Fingerprint() string                                    { return "fake" }
func (f fakeLLM) Interpret(_ context.Context, _ string) (float64, error) { return f.score, nil }

// newExchangeInsideFrontier returns an exchange whose M2 frontier heuristic yields Inside()==true.
// Rules: DiscoveryMode != Static => UniversOuvert=true, CompositionVariable=true;
// HumanInLoop=false => PairProbabiliste=true; DelegationDepth > 0 => CoutNonBorne=true.
func newExchangeInsideFrontier(id string) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: "test intent",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent",
			HumanInLoop:     false,
			Organization:    "org-x",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "target-svc",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
	}
}

// newDeterministeExchange returns an exchange with M2 composite tau_score < 0.35.
// D-SENS is reduced by a static contract URI and no intent; D-AUTORITÉ and D-INVARIANT
// are low (delegation depth 1, contract present). fakeLLM score of 0 is used.
func newDeterministeExchange(id string) tau.Exchange {
	return tau.Exchange{
		ID:           id,
		DiscoveredAt: time.Now(),
		// No IntentDescription => S_runtime_resolve = 0.
		Initiator: tau.Principal{
			ID:              "agent",
			HumanInLoop:     false,
			Organization:    "org-x",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "target-svc",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1",
		},
	}
}

// newHysteresisExchange returns an exchange with M2 composite tau_score in [0.35, 0.65).
// Uses a static contract URI to lower D-SENS while keeping IntentDescription set.
func newHysteresisExchange(id string) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: "test intent",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent",
			HumanInLoop:     false,
			Organization:    "org-x",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "target-svc",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1",
		},
	}
}

func TestDispatcher_Decide_Deterministe(t *testing.T) {
	t.Parallel()
	// fakeLLM score=0 keeps S_reasoner_intent=0; static contract keeps S_contract=0;
	// no intent keeps S_runtime_resolve=0. Resulting composite < 0.35 => Deterministe.
	d := orchestration.NewDispatcher(fakeLLM{score: 0.0}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
		AuthBlock:    0.85,
	})
	dec, err := d.Decide(context.Background(), newDeterministeExchange("t-det"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Deterministe {
		t.Fatalf("regime = %v, want Deterministe", dec.Regime)
	}
	if dec.Trace.ExchangeID != "t-det" {
		t.Fatalf("trace ExchangeID = %q, want \"t-det\"", dec.Trace.ExchangeID)
	}
}

func TestDispatcher_Decide_Probabiliste(t *testing.T) {
	t.Parallel()
	// All D-SENS probes high (no contract, intent set, DynamicMCP) + high LLM score => composite > 0.65.
	d := orchestration.NewDispatcher(fakeLLM{score: 0.80}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
		AuthBlock:    0.85,
	})
	dec, err := d.Decide(context.Background(), newExchangeInsideFrontier("t-prob"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Probabiliste {
		t.Fatalf("regime = %v, want Probabiliste", dec.Regime)
	}
}

func TestDispatcher_Decide_HysteresisDefaultsToDeterministe(t *testing.T) {
	t.Parallel()
	// Static contract + intent present + mid LLM score => composite in [0.35, 0.65) => hysteresis => Deterministe.
	d := orchestration.NewDispatcher(fakeLLM{score: 0.30}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
		AuthBlock:    0.85,
	})
	dec, _ := d.Decide(context.Background(), newHysteresisExchange("t-hyst"))
	if dec.Regime != tau.Deterministe {
		t.Fatalf("hysteresis zone: regime = %v, want Deterministe (M2 default)", dec.Regime)
	}
}
