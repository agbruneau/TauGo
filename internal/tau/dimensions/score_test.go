package dimensions

import (
	"math"
	"testing"
)

// White-box tests for clamp01. Package dimensions (not dimensions_test) so the
// unexported helper is directly accessible.

func TestClamp01_BelowZero(t *testing.T) {
	t.Parallel()
	if got := clamp01(-0.5); got != 0 {
		t.Fatalf("clamp01(-0.5) = %v, want 0", got)
	}
}

func TestClamp01_AboveOne(t *testing.T) {
	t.Parallel()
	if got := clamp01(1.5); got != 1 {
		t.Fatalf("clamp01(1.5) = %v, want 1", got)
	}
}

func TestClamp01_Zero(t *testing.T) {
	t.Parallel()
	if got := clamp01(0); got != 0 {
		t.Fatalf("clamp01(0) = %v, want 0", got)
	}
}

func TestClamp01_One(t *testing.T) {
	t.Parallel()
	if got := clamp01(1); got != 1 {
		t.Fatalf("clamp01(1) = %v, want 1", got)
	}
}

// TestClamp01_NaN verifies pass-through behavior: NaN fails both comparisons
// so it is returned unchanged. Callers that produce NaN expose a bug upstream;
// this test documents the actual behavior rather than asserting correctness.
func TestClamp01_NaN(t *testing.T) {
	t.Parallel()
	nan := math.NaN()
	got := clamp01(nan)
	if !math.IsNaN(got) {
		t.Fatalf("clamp01(NaN) = %v, want NaN (pass-through)", got)
	}
}
