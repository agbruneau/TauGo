package invariants

import (
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// I3PerimptionLimite is the latest acceptable DateRevision per PRD §6.1 I3.
// Beyond this date the institutional-fact landscape is presumed to have
// shifted; the profile must be renewed or τ refuses by the expiry clause.
// Dated 2026-05-24; next review 2027-01-01. Status: Probable.
var I3PerimptionLimite = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // read-only sentinel, written once at init; see PRD §6.1 I3 veille trimestrielle

// diagI3OntologicalRefus is the exact diagnostic string emitted by the
// dispatcher (orchestration/dispatcher.go step 2) when the ontological
// D-AUTORITÉ guard fires.
const diagI3OntologicalRefus = "I3 — verrou ontologique D-AUTORITÉ"

// diagI3ExpiryRefus is the diagnostic string expected when the dispatcher
// fires the profile-expiry refus (step 3, landed in M3). It is matched
// when checking whether the expiry guard was properly enforced.
const diagI3ExpiryRefus = "profil périmé — veille requise"

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
		// (2) Ontological bypass heuristic: composite tau_score is used as a
		// proxy for D-AUTORITÉ since ventilated scores are not yet in Trace
		// (deferred to M5). A score >= AuthBlock without attestation indicates
		// the gate should have fired.
		if x.AttestationInstitutionnelle == nil &&
			dec.Trace.Thresholds.AuthBlock > 0 &&
			dec.Trace.TauScore >= dec.Trace.Thresholds.AuthBlock {
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
// Ventilated D-AUTORITÉ score deferred to M5 (Trace.Scores not yet exposed).
func EvaluateI3(x tau.Exchange, dec tau.Decision) Status {
	return EvaluateI3WithClock(x, dec, time.Now())
}
