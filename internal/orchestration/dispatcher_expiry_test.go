package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/calibration"
	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestDispatcher_Step3_ExpiredProfileRefuses verifies that Decide returns
// Refus with "profil périmé" when the profile's DateRevision is in the past.
func TestDispatcher_Step3_ExpiredProfileRefuses(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	p.DateRevision = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) // past

	d := orchestration.NewDispatcherWithProfile(
		fakeLLM{score: 0},
		orchestration.DefaultThresholds(),
		&p,
	).WithClock(func() time.Time {
		return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // after expiry
	})

	dec, err := d.Decide(context.Background(), newExchangeInsideFrontier("exp-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Refus {
		t.Fatalf("expected Refus, got %v", dec.Regime)
	}
	if !strings.Contains(dec.Diagnostic, "profil périmé") {
		t.Fatalf("expected 'profil périmé' in diagnostic, got %q", dec.Diagnostic)
	}
}

// TestDispatcher_Step3_NotExpiredProceeds verifies that Decide does not refuse
// when the profile's DateRevision is strictly after the clock value.
func TestDispatcher_Step3_NotExpiredProceeds(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	p.DateRevision = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC) // future

	d := orchestration.NewDispatcherWithProfile(
		fakeLLM{score: 0},
		orchestration.DefaultThresholds(),
		&p,
	).WithClock(func() time.Time {
		return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // before expiry
	})

	dec, err := d.Decide(context.Background(), newExchangeInsideFrontier("noexp-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must not refuse due to profile expiry; may refuse for other reasons.
	if dec.Regime == tau.Refus && strings.Contains(dec.Diagnostic, "profil périmé") {
		t.Fatalf("unexpected expiry refusal for non-expired profile: %v", dec.Diagnostic)
	}
}

// TestDispatcher_Step3_ZeroDateRevisionSkipsCheck verifies that a profile with
// zero DateRevision does not trigger the expiry guard (opt-in semantics).
func TestDispatcher_Step3_ZeroDateRevisionSkipsCheck(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	p.DateRevision = time.Time{} // zero value — guard must be skipped

	d := orchestration.NewDispatcherWithProfile(
		fakeLLM{score: 0},
		orchestration.DefaultThresholds(),
		&p,
	).WithClock(func() time.Time {
		return time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC) // far future
	})

	dec, err := d.Decide(context.Background(), newExchangeInsideFrontier("zero-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime == tau.Refus && strings.Contains(dec.Diagnostic, "profil périmé") {
		t.Fatalf("zero DateRevision triggered expiry guard: %v", dec.Diagnostic)
	}
}

// TestDispatcher_Step3_NilProfileSkipsCheck verifies backward-compatibility:
// NewDispatcher (no profile) never enforces step 3.
func TestDispatcher_Step3_NilProfileSkipsCheck(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0}, orchestration.DefaultThresholds()).
		WithClock(func() time.Time {
			return time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
		})

	dec, err := d.Decide(context.Background(), newExchangeInsideFrontier("nil-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime == tau.Refus && strings.Contains(dec.Diagnostic, "profil périmé") {
		t.Fatalf("nil profile triggered expiry guard: %v", dec.Diagnostic)
	}
}
