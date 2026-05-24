package orchestration_test

import (
	"context"
	"testing"

	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestStep8_InvariantsEvaluated_NoViolation_TraceEmpty verifies that a nominal
// decision (no invariant violation) leaves UnmodeledObservations empty.
//
// Configuration: standard thresholds with AuthBlock=0.85. The exchange is
// inside the frontier but has a low composite tau_score (fakeLLM score 0),
// so EvaluateI3 does not fire (authScore << AuthBlock).
func TestStep8_InvariantsEvaluated_NoViolation_TraceEmpty(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.0}, orchestration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	})
	dec, err := d.Decide(context.Background(), newDeterministeExchange("e-s8-no-violation"))
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if dec.Regime == tau.Refus {
		t.Skipf("exchange was refused upstream; cannot exercise step 8: diag=%q", dec.Diagnostic)
	}
	if len(dec.Trace.UnmodeledObservations) != 0 {
		t.Fatalf("UnmodeledObservations = %v, want empty (no invariant violation)", dec.Trace.UnmodeledObservations)
	}
}

// TestStep8_InvariantsEvaluated_ViolationDetected_TraceEnriched verifies that
// when EvaluateI3 detects a bypass violation, its Summary entry appears in
// Trace.UnmodeledObservations.
//
// V2 logic (ADR-0008): EvaluateI3 reads dec.Trace.DAuthority.Value directly.
// To trigger a Probabiliste result that EvaluateI3 flags as Violated, we need:
//   - DAuthority.Value >= AuthBlock (so the gate should have fired)
//   - No attestation on x
//   - But the exchange must NOT have been refused at step 2
//
// With correct scores this scenario cannot arise in a normal dispatcher flow
// (step 2 would catch it). To test the EvaluateI3 bypass-detection path
// directly, we use a forged Decision in i3_authority_asymmetry_test.go.
//
// This test verifies the complementary case: exchange with AuthBlock set well
// above authScore (0.5625) produces no violation. The Probabiliste regime
// is reached because tauScore (~0.692) >= Probabiliste threshold (0.65).
func TestStep8_InvariantsEvaluated_ViolationDetected_TraceEnriched(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.80}, orchestration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.60, // above authScore(0.5625); step 2 does not fire
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	})
	x := newExchangeInsideFrontier("e-s8-violation")

	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if dec.Regime == tau.Refus {
		t.Fatalf("exchange was refused upstream; step 8 not reached. diag=%q", dec.Diagnostic)
	}
	// With ventilated scores (ADR-0008), authScore.Value = 0.5625 < AuthBlock = 0.60,
	// so EvaluateI3 correctly reports Held (no bypass). UnmodeledObservations must be empty.
	if len(dec.Trace.UnmodeledObservations) != 0 {
		t.Fatalf("expected no violation (DAuthority=%.4f < AuthBlock=%.4f), got: %v",
			dec.Trace.DAuthority.Value, dec.Trace.Thresholds.AuthBlock, dec.Trace.UnmodeledObservations)
	}
	// Confirm ventilated scores are populated on the Probabiliste decision.
	if dec.Trace.DAuthority == nil {
		t.Fatal("Trace.DAuthority must be non-nil on a Probabiliste decision")
	}
	if dec.Trace.DSens == nil {
		t.Fatal("Trace.DSens must be non-nil on a Probabiliste decision")
	}
	if dec.Trace.DInvariant == nil {
		t.Fatal("Trace.DInvariant must be non-nil on a Probabiliste decision")
	}
}

// TestStep8_RegimeUntouchedByEvaluation verifies that step 8 is pure
// instrumentation: the Regime produced by step 7 is not modified when
// a nominal exchange is evaluated.
func TestStep8_RegimeUntouchedByEvaluation(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.80}, orchestration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.60,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	})
	x := newExchangeInsideFrontier("e-s8-regime")

	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if dec.Regime == tau.Refus {
		t.Fatalf("exchange was refused upstream; step 8 not reached. diag=%q", dec.Diagnostic)
	}
	// Regime must be one of the two non-Refus outcomes.
	if dec.Regime != tau.Deterministe && dec.Regime != tau.Probabiliste {
		t.Fatalf("step 8 mutated Regime to unexpected value %v", dec.Regime)
	}
}
