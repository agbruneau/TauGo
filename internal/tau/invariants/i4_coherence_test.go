package invariants_test

import (
	"testing"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// TestIsIncoherent_TableDriven verifies the pure IsIncoherent function on
// representative (s, i, θ_sens, θ_inv) tuples.
func TestIsIncoherent_TableDriven(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		s, i   float64
		sT, iT float64
		want   bool
	}{
		// I4-violating: inv high, sens low
		{"s_low_i_high", 0.10, 0.70, 0.50, 0.50, true},
		// Both high: no violation (sens is also high)
		{"s_high_i_high", 0.70, 0.70, 0.50, 0.50, false},
		// Both low: no violation (inv below threshold)
		{"s_low_i_low", 0.10, 0.10, 0.50, 0.50, false},
		// Boundary equality on sens (strict <): s == θ_sens is NOT a violation
		{"boundary_sens_eq", 0.50, 0.70, 0.50, 0.50, false},
		// Boundary equality on inv (>=): i == θ_inv IS a violation when s < θ_sens
		{"boundary_inv_eq_violating", 0.10, 0.50, 0.50, 0.50, true},
		// Zero thresholds: inv >= 0 always, sens < 0 never — no violation
		{"zero_thresholds", 0.0, 0.0, 0.0, 0.0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := invariants.IsIncoherent(c.s, c.i, c.sT, c.iT)
			if got != c.want {
				t.Fatalf(
					"IsIncoherent(sens=%v, inv=%v, θ_sens=%v, θ_inv=%v) = %v, want %v",
					c.s, c.i, c.sT, c.iT, got, c.want,
				)
			}
		})
	}
}

// TestEvaluateI4_NominalHeld verifies that a normal (non-Refus) decision returns Held.
func TestEvaluateI4_NominalHeld(t *testing.T) {
	t.Parallel()
	dec := tau.Decision{
		Regime: tau.Deterministe,
		Trace:  tau.Trace{ExchangeID: "e-i4-nominal"},
	}
	if got := invariants.EvaluateI4(tau.Exchange{}, dec); got != invariants.Held {
		t.Fatalf("EvaluateI4(Deterministe) = %v, want Held", got)
	}
}

// TestEvaluateI4_RefusalForI4_Held verifies that the dispatcher's I4 refus is
// recognized as Held (the guard fired correctly).
func TestEvaluateI4_RefusalForI4_Held(t *testing.T) {
	t.Parallel()
	dec := tau.Decision{
		Regime:     tau.Refus,
		Diagnostic: "I4 — combinaison incohérente détectée",
	}
	if got := invariants.EvaluateI4(tau.Exchange{}, dec); got != invariants.Held {
		t.Fatalf("EvaluateI4(Refus I4) = %v, want Held", got)
	}
}

// TestEvaluateI4_IncoherentNonRefused_Violated is the V1 limitation test:
// without ventilated scores in the Trace, EvaluateI4 cannot detect a bypassed
// guard from a non-Refus decision. This test documents the current Held
// verdict (status Hypothèse) and will be updated in M5 when scores are
// available in the Trace.
func TestEvaluateI4_IncoherentNonRefused_Violated(t *testing.T) {
	t.Parallel()
	// A Probabiliste decision where the (s, i) pair was incoherent but the
	// dispatcher's guard was somehow bypassed. V1 cannot observe this because
	// Trace does not carry ventilated dimension scores. EvaluateI4 returns Held
	// by construction (Hypothèse) — this test documents that limitation.
	dec := tau.Decision{
		Regime: tau.Probabiliste,
		Trace: tau.Trace{
			ExchangeID: "e-i4-bypass",
			TauScore:   0.75,
		},
	}
	// V1: Held (cannot detect from Decision alone without ventilated scores).
	// M5 will update this to Violated when Trace.DSens / Trace.DInvariant are
	// available and IsIncoherent can be called directly.
	got := invariants.EvaluateI4(tau.Exchange{}, dec)
	if got != invariants.Held {
		t.Fatalf(
			"EvaluateI4(Probabiliste bypass) = %v, want Held "+
				"(V1 limitation: ventilated scores absent from Trace — update in M5)",
			got,
		)
	}
}

// TestEvaluateI4_OtherRefusal_NotApplicable verifies that a Refus with a
// non-I4 diagnostic is mapped to NotApplicable.
func TestEvaluateI4_OtherRefusal_NotApplicable(t *testing.T) {
	t.Parallel()
	cases := []string{
		"hors frontière τ",
		"I3 — verrou ontologique D-AUTORITÉ",
		"profil périmé",
	}
	for _, diag := range cases {
		t.Run(diag, func(t *testing.T) {
			t.Parallel()
			dec := tau.Decision{Regime: tau.Refus, Diagnostic: diag}
			if got := invariants.EvaluateI4(tau.Exchange{}, dec); got != invariants.NotApplicable {
				t.Fatalf("EvaluateI4(Refus %q) = %v, want NotApplicable", diag, got)
			}
		})
	}
}
