package invariants_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// refTime is an arbitrary fixed instant used as "now" across I3 tests.
var refTime = time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

// pastRevision is a DateRevision that is before refTime (already expired).
var pastRevision = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// futureRevision is a DateRevision that is after refTime (still valid).
var futureRevision = time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

// --- IsProfileExpired ---

func TestIsProfileExpired_ZeroDateRevision(t *testing.T) {
	t.Parallel()
	dec := tau.Decision{}
	if invariants.IsProfileExpired(dec, refTime) {
		t.Fatal("IsProfileExpired returned true for zero DateRevision, want false")
	}
}

func TestIsProfileExpired_FutureRevision(t *testing.T) {
	t.Parallel()
	dec := tau.Decision{DateRevision: futureRevision}
	if invariants.IsProfileExpired(dec, refTime) {
		t.Fatal("IsProfileExpired returned true for future DateRevision, want false")
	}
}

func TestIsProfileExpired_PastRevision(t *testing.T) {
	t.Parallel()
	dec := tau.Decision{DateRevision: pastRevision}
	if !invariants.IsProfileExpired(dec, refTime) {
		t.Fatal("IsProfileExpired returned false for past DateRevision, want true")
	}
}

// --- EvaluateI3 nominal ---

// TestEvaluateI3_NominalHeld verifies that a non-Refus decision with no
// expiry and no attestation issue yields Held.
func TestEvaluateI3_NominalHeld(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-nominal"}
	dec := tau.Decision{
		Regime:       tau.Deterministe,
		DateRevision: futureRevision,
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   0.30,
			Thresholds: tau.TraceThresholds{AuthBlock: 0.85},
		},
	}
	// Use real EvaluateI3 — today (2026-05-24) < futureRevision so no expiry.
	if got := invariants.EvaluateI3(x, dec); got != invariants.Held {
		t.Fatalf("EvaluateI3 = %v, want Held (nominal deterministe, non-expired)", got)
	}
}

// --- Anti-patron #3: profile expired but not refused ---

// TestEvaluateI3_ProfileExpiredButNotRefused_Violated checks that when the
// profile is expired AND the decision is not Refus("profil périmé — veille
// requise"), EvaluateI3 returns Violated (expiry guard was bypassed).
func TestEvaluateI3_ProfileExpiredButNotRefused_Violated(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-expired-bypass"}
	dec := tau.Decision{
		Regime:       tau.Deterministe,
		DateRevision: pastRevision, // expired relative to refTime
		Trace:        tau.Trace{ExchangeID: x.ID},
	}
	// Drive with injectable clock so the test is deterministic.
	got := invariants.EvaluateI3WithClock(x, dec, refTime)
	if got != invariants.Violated {
		t.Fatalf("EvaluateI3WithClock = %v, want Violated (profile expired, not refused)", got)
	}
}

// TestEvaluateI3_ExpiryRefusal_Held verifies that a proper expiry Refus is
// treated as Held (the guard tient, expiry was correctly enforced).
func TestEvaluateI3_ExpiryRefusal_Held(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-expiry-refus"}
	dec := tau.Decision{
		Regime:       tau.Refus,
		Diagnostic:   "profil périmé — veille requise",
		DateRevision: pastRevision,
		Trace:        tau.Trace{ExchangeID: x.ID},
	}
	got := invariants.EvaluateI3WithClock(x, dec, refTime)
	if got != invariants.Held {
		t.Fatalf("EvaluateI3WithClock = %v, want Held (proper expiry refus)", got)
	}
}

// --- Ontological lock ---

// TestEvaluateI3_AuthBlockedWithoutAttestation_Violated checks that a
// Probabiliste decision with tau_score >= AuthBlock and no attestation is
// flagged Violated (the D-AUTORITÉ guard was bypassed).
func TestEvaluateI3_AuthBlockedWithoutAttestation_Violated(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{
		ID:                          "x-i3-auth-bypass",
		AttestationInstitutionnelle: nil, // no attestation
	}
	dec := tau.Decision{
		Regime:       tau.Probabiliste,
		DateRevision: futureRevision,
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   0.90, // >= AuthBlock
			Thresholds: tau.TraceThresholds{AuthBlock: 0.85},
		},
	}
	got := invariants.EvaluateI3WithClock(x, dec, refTime)
	if got != invariants.Violated {
		t.Fatalf("EvaluateI3WithClock = %v, want Violated (Probabiliste, score>=AuthBlock, no attestation)", got)
	}
}

// TestEvaluateI3_AuthBlockedRefusal_Held verifies that the ontological Refus
// emitted by the dispatcher yields Held (the guard fired correctly).
func TestEvaluateI3_AuthBlockedRefusal_Held(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-guard-fired"}
	dec := tau.Decision{
		Regime:       tau.Refus,
		Diagnostic:   "I3 — verrou ontologique D-AUTORITÉ",
		DateRevision: futureRevision,
		Trace:        tau.Trace{ExchangeID: x.ID},
	}
	got := invariants.EvaluateI3WithClock(x, dec, refTime)
	if got != invariants.Held {
		t.Fatalf("EvaluateI3WithClock = %v, want Held (ontological refus fired)", got)
	}
}

// TestEvaluateI3_OtherRefus_NotApplicable confirms that a Refus unrelated to
// I3 (e.g. hors frontière) returns NotApplicable.
func TestEvaluateI3_OtherRefus_NotApplicable(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-other-refus"}
	dec := tau.Decision{
		Regime:       tau.Refus,
		Diagnostic:   "hors frontière τ",
		DateRevision: futureRevision,
		Trace:        tau.Trace{ExchangeID: x.ID},
	}
	got := invariants.EvaluateI3WithClock(x, dec, refTime)
	if got != invariants.NotApplicable {
		t.Fatalf("EvaluateI3WithClock = %v, want NotApplicable (unrelated Refus)", got)
	}
}

// TestEvaluateI3_ProbabilisteBelowAuthBlock_Held verifies that a Probabiliste
// decision with tau_score < AuthBlock and no attestation remains Held (not a
// bypass — score did not reach the guard threshold).
func TestEvaluateI3_ProbabilisteBelowAuthBlock_Held(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-prob-low-score"}
	dec := tau.Decision{
		Regime:       tau.Probabiliste,
		DateRevision: futureRevision,
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   0.60, // < AuthBlock
			Thresholds: tau.TraceThresholds{AuthBlock: 0.85},
		},
	}
	got := invariants.EvaluateI3WithClock(x, dec, refTime)
	if got != invariants.Held {
		t.Fatalf("EvaluateI3WithClock = %v, want Held (Probabiliste, score<AuthBlock)", got)
	}
}
