package orchestration_test

import (
	"context"
	"testing"

	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestDecisionAlwaysTraced — every Decision must have a non-zero
// Trace.ExchangeID matching the input Exchange.ID, regardless of regime.
// Also verifies that DurationNs is positive.
func TestDecisionAlwaysTraced(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.50}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	dec, err := d.Decide(context.Background(), tau.Exchange{ID: "must-be-traced"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Trace.ExchangeID != "must-be-traced" {
		t.Fatalf("trace ExchangeID = %q, want \"must-be-traced\"", dec.Trace.ExchangeID)
	}
	if dec.Trace.DurationNs <= 0 {
		t.Fatalf("trace DurationNs = %d, want > 0", dec.Trace.DurationNs)
	}
}

// TestRefusImpliesDiagnostic — Decision.Regime == Refus iff Decision.Diagnostic != "".
// In M1, the only Refus path inside the dispatcher is "hors frontiere tau" but the
// frontier check is a placeholder that always returns Inside=true. To exercise
// the contract, we construct Decision values directly and verify the biconditional.
// M2 will add real Refus paths and dispatcher-driven assertions.
func TestRefusImpliesDiagnostic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		dec  tau.Decision
	}{
		{"refus_with_diagnostic", tau.Decision{Regime: tau.Refus, Diagnostic: "hors frontiere"}},
		{"deterministe_without_diagnostic", tau.Decision{Regime: tau.Deterministe, Diagnostic: ""}},
		{"probabiliste_without_diagnostic", tau.Decision{Regime: tau.Probabiliste, Diagnostic: ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			isRefus := tc.dec.Regime == tau.Refus
			hasDiag := tc.dec.Diagnostic != ""
			if isRefus != hasDiag {
				t.Fatalf("contract violated: Regime==Refus(%v) but Diagnostic!=\"\"(%v) for %s", isRefus, hasDiag, tc.name)
			}
		})
	}
}

// TestTraceImmutable — Trace is a value type embedded by value in Decision.
// Mutating the returned Decision's Trace must not leak back into the
// Dispatcher's internal state or affect subsequent Decisions.
func TestTraceImmutable(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.20}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	dec1, err := d.Decide(context.Background(), tau.Exchange{ID: "first"})
	if err != nil {
		t.Fatalf("first Decide failed: %v", err)
	}
	// Mutate the local copy of the first Decision's Trace.
	dec1.Trace.ExchangeID = "MUTATED-LOCAL"
	dec1.Trace.UnmodeledObservations = append(dec1.Trace.UnmodeledObservations, "injected")

	// A second Decide must produce an independent Trace.
	dec2, err := d.Decide(context.Background(), tau.Exchange{ID: "second"})
	if err != nil {
		t.Fatalf("second Decide failed: %v", err)
	}
	if dec2.Trace.ExchangeID != "second" {
		t.Fatalf("second decision Trace.ExchangeID = %q, want \"second\" (mutation leaked)", dec2.Trace.ExchangeID)
	}
	if len(dec2.Trace.UnmodeledObservations) != 0 {
		t.Fatalf("second decision UnmodeledObservations leaked: %v", dec2.Trace.UnmodeledObservations)
	}
}
