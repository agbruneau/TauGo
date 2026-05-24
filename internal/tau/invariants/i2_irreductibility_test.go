package invariants_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

func dynamicExchange() tau.Exchange {
	return tau.Exchange{
		ID:                "e-dyn",
		IntentDescription: "discover and call",
		DiscoveredAt:      time.Now().UTC(),
		Initiator: tau.Principal{
			ID: "agent-1", HumanInLoop: false, DelegationDepth: 2,
		},
		Target: tau.Capability{
			ID: "dyn-tool", DiscoveryMode: tau.DynamicMCP,
		},
	}
}

func TestResidu_NonEmptyForDynamicExchange(t *testing.T) {
	t.Parallel()
	r := invariants.Residu(dynamicExchange())
	if len(r) == 0 {
		t.Fatal("Residu was empty on dynamic exchange (frontier should yield >= 1 magnitude)")
	}
}

func TestRecablage_RemovingAllResidualLosesFrontier(t *testing.T) {
	t.Parallel()
	x := dynamicExchange()
	r := invariants.Residu(x)
	names := make([]string, len(r))
	for i, m := range r {
		names[i] = string(m)
	}
	got := invariants.Recablage(x, names)
	if got.Inside() {
		t.Fatalf("Recablage with all residual magnitudes removed kept Inside()=true: %+v", got)
	}
}

func TestEvaluateI2_HeldOnDynamicExchange(t *testing.T) {
	t.Parallel()
	x := dynamicExchange()
	dec := tau.Decision{
		Regime: tau.Probabiliste,
		Trace:  tau.Trace{ExchangeID: x.ID},
	}
	if got := invariants.EvaluateI2(x, dec); got != invariants.Held {
		t.Fatalf("EvaluateI2 = %v, want Held", got)
	}
}

func TestEvaluateI2_NotApplicableOnRefusFrontiere(t *testing.T) {
	t.Parallel()
	x := dynamicExchange()
	dec := tau.Decision{Regime: tau.Refus, Diagnostic: "hors frontière τ", Trace: tau.Trace{ExchangeID: x.ID}}
	if got := invariants.EvaluateI2(x, dec); got != invariants.NotApplicable {
		t.Fatalf("EvaluateI2 = %v, want NotApplicable", got)
	}
}
