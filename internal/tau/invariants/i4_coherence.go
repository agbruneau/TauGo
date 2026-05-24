package invariants

import "github.com/agbruneau/taugo/internal/tau"

// IsIncoherent reports whether (sensValue, invValue) forms an I4-violating pair
// under the thresholds (sensCoherence, invCoherence). Returns true iff inv
// reaches or exceeds its coherence threshold while sens is strictly below its
// threshold — the asymmetric direction encoded in PRD §6.1 I4.
//
// This is the pure-function form of the dispatcher step-5 guard:
//
//	D-INVARIANT >= θ_inv ∧ D-SENS < θ_sens ⇒ incoherent
//
// Status: Hypothèse (chap. III.8.5.4). Empirical campaign deferred to M4.
func IsIncoherent(sensValue, invValue, sensCoherence, invCoherence float64) bool {
	return invValue >= invCoherence && sensValue < sensCoherence
}

// EvaluateI4 returns the I4 verdict for (x, decision).
//
// Encoding V1: I4 reasons over the Decision's regime and diagnostic rather
// than recomputing dimension scores (importing tau/dimensions is forbidden
// per arch_test.go). Ventilated scores deferred to M5.
//
//   - Refus("I4 — combinaison incohérente détectée"): Held — the dispatcher's
//     step-5 guard fired correctly.
//   - Other Refus diagnostics: NotApplicable — I4 was not the deciding factor.
//   - Deterministe / Probabiliste: Held by default (V1 cannot verify the
//     (s, i) pair without ventilated scores; verdict carries status Hypothèse).
//
// FuzzI4_CoherenceContrainte drives IsIncoherent directly on synthetic (s, i, θ)
// tuples without requiring dispatcher reproduction.
func EvaluateI4(_ tau.Exchange, dec tau.Decision) Status {
	if dec.Regime == tau.Refus {
		if dec.Diagnostic == "I4 — combinaison incohérente détectée" {
			return Held
		}
		return NotApplicable
	}
	return Held
}
