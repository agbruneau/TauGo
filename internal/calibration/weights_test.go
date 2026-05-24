package calibration

import (
	"math"
	"testing"
)

// TestCalibrateWeights_V1IsPassthrough verifies that V1 CalibrateWeights
// returns the input Weights unchanged (identity property).
func TestCalibrateWeights_V1IsPassthrough(t *testing.T) {
	t.Parallel()
	base := DefaultProfile().Weights
	got := CalibrateWeights(nil, 0, base)
	if got.DSens != base.DSens || got.DAuthority != base.DAuthority || got.DInvariant != base.DInvariant {
		t.Fatalf("V1 CalibrateWeights must be identity: got %+v, want %+v", got, base)
	}
	for k, want := range base.SensProbes {
		if got.SensProbes[k] != want {
			t.Fatalf("SensProbes[%s]: got %v, want %v", k, got.SensProbes[k], want)
		}
	}
}

// TestCalibrateWeights_PreservesSumToOne verifies that the returned Weights
// still satisfy the sum-to-one invariant on dimension weights.
func TestCalibrateWeights_PreservesSumToOne(t *testing.T) {
	t.Parallel()
	const epsilon = 1e-9
	base := DefaultProfile().Weights
	got := CalibrateWeights(nil, 42, base)
	sum := got.DSens + got.DAuthority + got.DInvariant
	if math.Abs(sum-1.0) > epsilon {
		t.Fatalf("dimension weights must sum to 1.0 ± %v, got %.15f", epsilon, sum)
	}
}

// TestWeightHook_AllowsCustomStrategy verifies that a caller-supplied
// WeightHook is honored and can return an entirely different Weights value.
// This exercises the V2 injection path without mutating the production hook.
func TestWeightHook_AllowsCustomStrategy(t *testing.T) {
	t.Parallel()
	zeroSensHook := WeightHook(func(_ []CorpusEntry, _ int64, base Weights) Weights {
		out := base
		out.DSens = 0.0
		out.DAuthority = 0.5
		out.DInvariant = 0.5
		return out
	})
	base := DefaultProfile().Weights
	got := CalibrateWeights(nil, 0, base, zeroSensHook)
	if got.DSens != 0.0 {
		t.Fatalf("custom hook: DSens should be 0.0, got %v", got.DSens)
	}
	if got.DAuthority != 0.5 || got.DInvariant != 0.5 {
		t.Fatalf("custom hook: DAuthority/DInvariant should be 0.5, got %v/%v", got.DAuthority, got.DInvariant)
	}
	// Base must not be mutated.
	if base.DSens != 0.4 {
		t.Fatal("hook must not mutate base Weights in place")
	}
}

// TestWeightCalibrationStrategy_IsV1 ensures the exported constant is set
// to the expected V1 identifier so callers can log/inspect the active strategy.
func TestWeightCalibrationStrategy_IsV1(t *testing.T) {
	t.Parallel()
	const want = "v1-passthrough"
	if WeightCalibrationStrategy != want {
		t.Fatalf("WeightCalibrationStrategy = %q, want %q", WeightCalibrationStrategy, want)
	}
}
