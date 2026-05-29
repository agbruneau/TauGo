package invariants

import "github.com/agbruneau/taugo/internal/tau"

// Conserve reports whether τ has preserved the magnitudes carried by x in
// the produced decision. V1 scope: the ExchangeID is the canonical magnitude;
// future versions extend Conserve as more invariants are added to Exchange.
//
// PRD §6.1 I1: "τ déplace l'instant de fixation d'une grandeur sans altérer
// la grandeur." V1 status: Probable — preservation is structurally enforced
// by Exchange value semantics; this helper detects future regressions.
func Conserve(x tau.Exchange, dec tau.Decision) bool {
	return dec.Trace.ExchangeID == x.ID
}

// EvaluateI1 returns the I1 verdict for (x, decision).
//
//   - Refus("hors frontière τ"): NotApplicable — τ has not operated.
//   - All other regimes: Held if Conserve(x, dec) holds, Violated otherwise.
func EvaluateI1(x tau.Exchange, dec tau.Decision) Status {
	if dec.Regime == tau.Refus && dec.Diagnostic == tau.DiagFrontiereFranchie {
		return NotApplicable
	}
	if Conserve(x, dec) {
		return Held
	}
	return Violated
}
