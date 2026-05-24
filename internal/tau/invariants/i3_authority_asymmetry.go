package invariants

import (
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// I3PerimptionLimite returns the cut-off date beyond which any calibration
// profile is considered expired and triggers Refus (anti-pattern #3, PRD §7.2).
// The value is fixed at compile time; modifying it requires an ADR.
// Status: Probable. Dated 2026-05-24; next review 2027-01-01.
func I3PerimptionLimite() time.Time {
	return time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
}

// diagI3OntologicalRefus is the exact diagnostic string emitted by the
// dispatcher (orchestration/dispatcher.go step 2) when the ontological
// D-AUTORITÉ guard fires.
const diagI3OntologicalRefus = tau.DiagVerrouOntologique

// diagI3ExpiryRefus is the diagnostic string expected when the dispatcher
// fires the profile-expiry refus (step 3, landed in M3). It is matched
// when checking whether the expiry guard was properly enforced.
const diagI3ExpiryRefus = tau.DiagPeremptionProfile

// IsProfileExpired reports whether dec.DateRevision is in the past relative
// to now. A zero DateRevision is treated as never-expired (no revision date
// set on the profile implies the dispatcher did not inject one).
//
// This helper is the testable building block for the anti-patron #3 guard
// (atemporel: expired profile tolerated without a Refus).
func IsProfileExpired(dec tau.Decision, now time.Time) bool {
	if dec.DateRevision.IsZero() {
		return false
	}
	return now.After(dec.DateRevision)
}

// EvaluateI3WithClock returns the I3 verdict using the supplied clock.
// The clock is injectable so unit tests can control "today" without sleeping.
// Prefer EvaluateI3 in production code.
//
// Logic (in priority order):
//  1. Anti-patron #3 (expiry): if the profile is expired AND the decision is
//     not a Refus("profil périmé — veille requise"), the expiry guard was
//     bypassed → Violated.
//  2. Ontological lock: if Probabiliste AND no attestation AND
//     tau_score >= AuthBlock threshold, the D-AUTORITÉ guard was bypassed →
//     Violated.
//  3. Explicit I3 refusals (ontological or expiry) → Held.
//  4. All other cases → Held (nominal path outside the risk zone).
func EvaluateI3WithClock(x tau.Exchange, dec tau.Decision, now time.Time) Status {
	// (1) Profile expiry check (anti-patron #3).
	if IsProfileExpired(dec, now) {
		expiryRefused := dec.Regime == tau.Refus && dec.Diagnostic == diagI3ExpiryRefus
		if !expiryRefused {
			return Violated
		}
	}

	switch dec.Regime {
	case tau.Refus:
		if dec.Diagnostic == diagI3OntologicalRefus || dec.Diagnostic == diagI3ExpiryRefus {
			return Held
		}
		// Other Refus diagnostics do not exercise the I3 gates.
		return NotApplicable

	case tau.Probabiliste:
		// (2) Ontological bypass check: read D-AUTORITÉ score directly from the
		// ventilated Trace field (ADR-0008). Falls back to the TauScore proxy
		// only when DAuthority is not yet populated (nil — e.g. stub traces in
		// older test fixtures). A score >= AuthBlock without attestation means
		// the ontological gate should have fired and was bypassed.
		authValue := dec.Trace.TauScore // fallback proxy
		if dec.Trace.DAuthority != nil {
			authValue = dec.Trace.DAuthority.Value
		}
		if x.AttestationInstitutionnelle == nil &&
			dec.Trace.Thresholds.AuthBlock > 0 &&
			authValue >= dec.Trace.Thresholds.AuthBlock {
			return Violated
		}
		return Held

	case tau.Deterministe:
		// Deterministe is below the probabilistic zone; D-AUTORITÉ gate not
		// triggered by definition in V1 (tau_score < Probabiliste threshold).
		return Held

	default:
		return NotApplicable
	}
}

// EvaluateI3 returns the I3 verdict for (x, decision) using time.Now() as
// the reference clock.
//
// PRD §6.1 I3: D-AUTORITÉ(x) ≥ θ_auth_block ∧ Attestation == nil ⇒ Refus.
// Status: Probable, dated 2026-05-16. Quarterly review tracked in PRD §16.
// Reads ventilated trace.DAuthority since v0.1.1 (ADR-0008); falls back to
// composite TauScore proxy if the ventilated score is nil for backward-compat.
func EvaluateI3(x tau.Exchange, dec tau.Decision) Status {
	return EvaluateI3WithClock(x, dec, time.Now())
}
