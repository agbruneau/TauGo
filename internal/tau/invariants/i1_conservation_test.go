package invariants_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

func makeExchange(id, intent string) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: intent,
		DiscoveredAt:      time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
		Initiator: tau.Principal{
			ID: "p-1", HumanInLoop: true, Organization: "org-a",
		},
		Target: tau.Capability{
			ID: "cap-1", DiscoveryMode: tau.Static, ContractURI: "https://api/v1",
		},
	}
}

func TestConserve_IdentityTrace(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-1", "compute")
	dec := tau.Decision{
		Regime: tau.Deterministe,
		Trace:  tau.Trace{ExchangeID: "e-1", TauScore: 0.2},
	}
	if !invariants.Conserve(x, dec) {
		t.Fatal("Conserve returned false on identity-preserving decision")
	}
}

func TestConserve_BrokenWhenExchangeIDDrifts(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-1", "compute")
	dec := tau.Decision{
		Regime: tau.Deterministe,
		Trace:  tau.Trace{ExchangeID: "MUTATED", TauScore: 0.2},
	}
	if invariants.Conserve(x, dec) {
		t.Fatal("Conserve returned true despite ExchangeID drift")
	}
}

func TestEvaluateI1_HeldOnNonRefus(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-i1", "compute")
	dec := tau.Decision{
		Regime: tau.Probabiliste,
		Trace:  tau.Trace{ExchangeID: "e-i1", TauScore: 0.8},
	}
	if got := invariants.EvaluateI1(x, dec); got != invariants.Held {
		t.Fatalf("EvaluateI1 = %v, want Held", got)
	}
}

func TestEvaluateI1_NotApplicableOnRefusFrontiere(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-i1-na", "compute")
	dec := tau.Decision{
		Regime:     tau.Refus,
		Diagnostic: "hors frontière τ",
		Trace:      tau.Trace{ExchangeID: "e-i1-na"},
	}
	// τ has not operated → I1 is not applicable.
	if got := invariants.EvaluateI1(x, dec); got != invariants.NotApplicable {
		t.Fatalf("EvaluateI1 = %v, want NotApplicable (Refus frontière)", got)
	}
}

func TestEvaluateI1_ViolatedOnTraceDrift(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-i1-v", "compute")
	dec := tau.Decision{
		Regime: tau.Deterministe,
		Trace:  tau.Trace{ExchangeID: "DIFFERENT", TauScore: 0.2},
	}
	if got := invariants.EvaluateI1(x, dec); got != invariants.Violated {
		t.Fatalf("EvaluateI1 = %v, want Violated (trace drift)", got)
	}
}
