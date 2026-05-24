package orchestration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// TestDispatcher_TraceContientScoresVentiles verifies that a non-Refus Decision
// carries all three ventilated dimension scores (DSens, DAuthority, DInvariant)
// with Value > 0, per ADR-0008 T-016 requirement.
func TestDispatcher_TraceContientScoresVentiles(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.50}, orchestration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	})
	x := tau.Exchange{
		ID:                "e-scores-ventiles",
		IntentDescription: "test intent with no contract",
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

	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("decide error: %v", err)
	}
	if dec.Regime == tau.Refus {
		t.Skipf("exchange was refused upstream (diag=%q); cannot test scores", dec.Diagnostic)
	}

	if dec.Trace.DAuthority == nil {
		t.Fatal("Trace.DAuthority is nil on non-Refus decision, want non-nil")
	}
	if dec.Trace.DAuthority.Value <= 0 {
		t.Errorf("Trace.DAuthority.Value = %.4f, want > 0", dec.Trace.DAuthority.Value)
	}
	if dec.Trace.DSens == nil {
		t.Fatal("Trace.DSens is nil on non-Refus decision, want non-nil")
	}
	if dec.Trace.DSens.Value <= 0 {
		t.Errorf("Trace.DSens.Value = %.4f, want > 0", dec.Trace.DSens.Value)
	}
	if dec.Trace.DInvariant == nil {
		t.Fatal("Trace.DInvariant is nil on non-Refus decision, want non-nil")
	}
	if dec.Trace.DInvariant.Value <= 0 {
		t.Errorf("Trace.DInvariant.Value = %.4f, want > 0", dec.Trace.DInvariant.Value)
	}
}

// TestEvaluateI3_LitDAuthorityVentile verifies that EvaluateI3 reads
// DAuthority directly from Trace when ventilated (ADR-0008 T-016).
// A Probabiliste decision with DAuthority.Value >= AuthBlock and no attestation
// must return Violated.
func TestEvaluateI3_LitDAuthorityVentile(t *testing.T) {
	t.Parallel()
	// Forge a Probabiliste decision with high DAuthority — simulating a bypass
	// where step 2 failed to catch the ontological lock.
	authValue := 0.90
	authBlock := 0.85
	dec := tau.Decision{
		Regime: tau.Probabiliste,
		Trace: tau.Trace{
			ExchangeID: "e-i3-ventile",
			TauScore:   0.70,
			DAuthority: &tau.Score{Value: authValue},
			Thresholds: tau.TraceThresholds{AuthBlock: authBlock},
		},
	}
	x := tau.Exchange{} // no AttestationInstitutionnelle

	got := invariants.EvaluateI3(x, dec)
	if got != invariants.Violated {
		t.Fatalf("EvaluateI3 with DAuthority=%.2f >= AuthBlock=%.2f and no attestation = %v, want Violated",
			authValue, authBlock, got)
	}
}

// TestEvaluateI4_DetecteByPassSilencieux verifies that EvaluateI4 detects a
// silent bypass when ventilated scores form an incoherent pair (ADR-0008 T-016).
// A Deterministe decision where DInvariant >= InvCoherence and DSens < SensCoherence
// indicates the step-5 guard should have fired but did not.
func TestEvaluateI4_DetecteByPassSilencieux(t *testing.T) {
	t.Parallel()
	dec := tau.Decision{
		Regime: tau.Deterministe,
		Trace: tau.Trace{
			ExchangeID: "e-i4-bypass-silencieux",
			TauScore:   0.40,
			DSens:      &tau.Score{Value: 0.20}, // < SensCoherence(0.50)
			DInvariant: &tau.Score{Value: 0.70}, // >= InvCoherence(0.50)
			Thresholds: tau.TraceThresholds{
				SensCoherence: 0.50,
				InvCoherence:  0.50,
			},
		},
	}

	got := invariants.EvaluateI4(tau.Exchange{}, dec)
	if got != invariants.Violated {
		t.Fatalf("EvaluateI4 with DSens=0.20 < SensCoherence=0.50 and DInvariant=0.70 >= InvCoherence=0.50 = %v, want Violated",
			got)
	}
}
