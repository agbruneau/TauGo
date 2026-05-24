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
// Encoding V2 (ADR-0008): when Trace carries ventilated scores (DSens,
// DInvariant non-nil), I4 can detect a silent bypass — a Deterministe or
// Probabiliste regime with an incoherent (sens, inv) pair that should have
// triggered a step-5 Refus. Without ventilated scores, verdict falls back to
// the V1 regime/diagnostic heuristic.
//
//   - Refus("I4 — combinaison incohérente détectée"): Held — the dispatcher's
//     step-5 guard fired correctly.
//   - Other Refus diagnostics: NotApplicable — I4 was not the deciding factor.
//   - Deterministe / Probabiliste with ventilated scores: check for silent bypass.
//   - Deterministe / Probabiliste without ventilated scores: Held by default (V1).
func EvaluateI4(_ tau.Exchange, dec tau.Decision) Status {
	if dec.Regime == tau.Refus {
		if dec.Diagnostic == tau.DiagIncoherenceI4 {
			return Held
		}
		return NotApplicable
	}

	// V2: detect silent bypass when ventilated scores are available.
	if dec.Trace.DSens != nil && dec.Trace.DInvariant != nil {
		th := dec.Trace.Thresholds
		if IsIncoherent(dec.Trace.DSens.Value, dec.Trace.DInvariant.Value, th.SensCoherence, th.InvCoherence) {
			return Violated
		}
	}

	return Held
}
