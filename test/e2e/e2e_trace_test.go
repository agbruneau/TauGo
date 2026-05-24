//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/calibration"
	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/testutil"
)

// TestE2E_TauDecide_TraceVentileeEtPoidsProfil exercises the full Decide path
// using a deterministic Exchange constructed via testutil.BuildExchange.
// It verifies that:
//
//	(a) All three ventilated scores (DSens, DAuthority, DInvariant) are non-nil
//	    and positive in the Trace (ADR-0008, T-015/T-016).
//	(b) Profile.Weights are applied to step 6, producing a TauScore that differs
//	    from a Dispatcher without a custom profile (T-017).
func TestE2E_TauDecide_TraceVentileeEtPoidsProfil(t *testing.T) {
	t.Parallel()

	x := testutil.BuildExchange(
		testutil.WithID("e2e-trace-ventilee"),
		testutil.WithIntentDescription("e2e integration test intent"),
		testutil.WithDiscoveryMode(tau.DynamicMCP),
		testutil.WithHumanInLoop(false),
		testutil.WithDelegationDepth(1),
	)

	th := orchestration.DefaultThresholds()

	// Dispatcher without custom profile (uses defaultDimensionWeights 0.4/0.3/0.3).
	dDefault := orchestration.NewDispatcher(llm.Stub{}, th)

	// Profile with non-uniform weights (0.6/0.2/0.2) and valid DateRevision.
	pCustom := calibration.DefaultProfile()
	pCustom.Weights.DSens = 0.60
	pCustom.Weights.DAuthority = 0.20
	pCustom.Weights.DInvariant = 0.20
	pCustom.DateRevision = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	dCustom := orchestration.NewDispatcherWithProfile(llm.Stub{}, th, &pCustom)

	// --- (a) Ventilated scores on custom dispatcher ---
	decCustom, err := dCustom.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("dCustom.Decide error: %v", err)
	}
	if decCustom.Regime == tau.Refus {
		t.Fatalf("E2E exchange refused unexpectedly: diag=%q", decCustom.Diagnostic)
	}
	assertScoreNonNilAndPositive(t, "DSens", decCustom.Trace.DSens)
	assertScoreNonNilAndPositive(t, "DAuthority", decCustom.Trace.DAuthority)
	assertScoreNonNilAndPositive(t, "DInvariant", decCustom.Trace.DInvariant)

	// --- (b) TauScore divergence between default and custom profile ---
	decDefault, err := dDefault.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("dDefault.Decide error: %v", err)
	}
	if decDefault.Regime == tau.Refus {
		t.Fatalf("E2E default exchange refused unexpectedly: diag=%q", decDefault.Diagnostic)
	}

	if decDefault.Trace.TauScore == decCustom.Trace.TauScore {
		t.Fatalf(
			"TauScore identical (%.6f) between default and custom profile dispatchers; "+
				"expected divergence. defaultWeights=(0.4,0.3,0.3) customWeights=(0.6,0.2,0.2)",
			decDefault.Trace.TauScore,
		)
	}
	t.Logf("E2E default TauScore=%.6f  custom TauScore=%.6f", decDefault.Trace.TauScore, decCustom.Trace.TauScore)
}

// assertScoreNonNilAndPositive fails the test if the Score pointer is nil or
// if Value is not strictly positive.
func assertScoreNonNilAndPositive(t *testing.T, name string, s *tau.Score) {
	t.Helper()
	if s == nil {
		t.Fatalf("Trace.%s is nil, want non-nil Score", name)
	}
	if s.Value <= 0 {
		t.Fatalf("Trace.%s.Value = %.4f, want > 0", name, s.Value)
	}
}
