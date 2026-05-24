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
// so EvaluateI3 does not fire (tauScore << AuthBlock).
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
// when EvaluateI3 detects a violation, its Summary entry appears in
// Trace.UnmodeledObservations.
//
// I3 violation condition: x.AttestationInstitutionnelle == nil &&
// dec.Trace.Thresholds.AuthBlock > 0 && dec.Trace.TauScore >= AuthBlock.
//
// Construction:
//   - AuthBlock = 0.60, chosen so that authScore (0.5625 for this exchange) < 0.60
//     and therefore step 2 does NOT fire, while the composite tauScore (~0.692
//     with fakeLLM=0.80) >= 0.60 so EvaluateI3 flags Violated at step 8.
//   - Derivation: authScore = 0.25*0.25 + 0.25*0 + 0.25*1 + 0.25*1 = 0.5625
//     (depth=1→0.25, org non-empty depth=1→0, humanInLoop=false→1, DynamicMCP→1)
//     tauScore ≈ 0.4*0.97 + 0.3*0.5625 + 0.3*0.45 ≈ 0.692
//   - No attestation on x, so EvaluateI3 returns Violated.
func TestStep8_InvariantsEvaluated_ViolationDetected_TraceEnriched(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.80}, orchestration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.60, // above authScore(0.5625) but below tauScore(~0.692)
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	})
	// newExchangeInsideFrontier: no attestation, DelegationDepth=1, DynamicMCP.
	x := newExchangeInsideFrontier("e-s8-violation")

	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if dec.Regime == tau.Refus {
		t.Fatalf("exchange was refused upstream; step 8 not reached. diag=%q tau_score=%.4f authBlock=%.4f",
			dec.Diagnostic, dec.Trace.TauScore, dec.Trace.Thresholds.AuthBlock)
	}
	if len(dec.Trace.UnmodeledObservations) == 0 {
		t.Fatalf("UnmodeledObservations empty; expected I3 violation entry. tau_score=%.4f authBlock=%.4f Decision=%+v",
			dec.Trace.TauScore, dec.Trace.Thresholds.AuthBlock, dec)
	}
	// Verify I3 diagnostic string is present.
	foundI3 := false
	for _, obs := range dec.Trace.UnmodeledObservations {
		if len(obs) >= 2 && obs[:2] == "I3" {
			foundI3 = true
			break
		}
	}
	if !foundI3 {
		t.Fatalf("UnmodeledObservations does not contain I3 entry: %v", dec.Trace.UnmodeledObservations)
	}
}

// TestStep8_RegimeUntouchedByEvaluation verifies that step 8 is pure
// instrumentation: the Regime produced by step 7 is not modified when
// an invariant violation is detected.
func TestStep8_RegimeUntouchedByEvaluation(t *testing.T) {
	t.Parallel()
	// Same configuration as the violation test — regime must still be Probabiliste
	// (tauScore ~0.692 >= Probabiliste threshold 0.65), not Refus.
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
	// Confirm the I3 violation entry is present, proving step 8 ran and mutated only the trace.
	foundI3 := false
	for _, obs := range dec.Trace.UnmodeledObservations {
		if len(obs) >= 2 && obs[:2] == "I3" {
			foundI3 = true
			break
		}
	}
	if !foundI3 {
		t.Fatalf("step 8 I3 violation not in trace; cannot confirm step 8 ran. tau_score=%.4f authBlock=%.4f observations=%v",
			dec.Trace.TauScore, dec.Trace.Thresholds.AuthBlock, dec.Trace.UnmodeledObservations)
	}
}
