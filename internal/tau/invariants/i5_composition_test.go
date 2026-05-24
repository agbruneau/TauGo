package invariants_test

import (
	"testing"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// TestAggregate_EmptyStack verifies that Aggregate of an empty Pile returns
// a non-nil empty slice (lower bound trivially holds: max=0, |M(π)|=0).
func TestAggregate_EmptyStack(t *testing.T) {
	t.Parallel()
	got := invariants.Aggregate(invariants.Pile{})
	if len(got) != 0 {
		t.Fatalf("Aggregate(empty) = %v, want empty slice", got)
	}
}

// TestAggregate_SingleLayer verifies that a single-layer Pile returns that
// layer's blind-spots (sorted). Lower bound: |M(π)| == max(|Aᵢ|) == len(layer).
func TestAggregate_SingleLayer(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{{"c", "a", "b"}}
	got := invariants.Aggregate(p)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("Aggregate single layer len = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Aggregate single layer mismatch at %d: got %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestAggregate_MultipleLayers_StableSortedUnion verifies deterministic lex
// ordering across multiple layers.
func TestAggregate_MultipleLayers_StableSortedUnion(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{{"z", "a"}, {"m"}, {"b"}}
	got := invariants.Aggregate(p)
	want := []string{"a", "b", "m", "z"}
	if len(got) != len(want) {
		t.Fatalf("Aggregate multi-layer len = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Aggregate order mismatch at %d: got %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestAggregate_DuplicateBlindSpots_Deduplicated verifies that blind-spots
// shared across layers appear exactly once in M(π).
func TestAggregate_DuplicateBlindSpots_Deduplicated(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{
		{"a", "b"},
		{"b", "c"},
		{"a"},
	}
	got := invariants.Aggregate(p)
	if len(got) != 3 {
		t.Fatalf("Aggregate dedup len = %d, want 3 (got %v)", len(got), got)
	}
}

// TestM_LowerBoundMaxLayer verifies M(π) >= max(|Aᵢ|) (chap. III.8.6.3).
// Uses BoundsHold as the specification; also checks the cardinal directly.
func TestM_LowerBoundMaxLayer(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{{"a", "b", "c"}, {"a"}}
	agg := invariants.Aggregate(p)
	maxLen := 3 // largest layer: {"a","b","c"}
	if len(agg) < maxLen {
		t.Fatalf("M(π) = %d < max(|Aᵢ|) = %d: lower bound violated (got %v)", len(agg), maxLen, agg)
	}
	if !invariants.BoundsHold(p) {
		t.Fatalf("BoundsHold failed: len(M(π)) < max(len(Aᵢ))")
	}
}

// TestM_UpperBoundSum verifies M(π) <= Σ|Aᵢ| (chap. III.8.6.3).
// Equality holds for fully disjoint layers.
func TestM_UpperBoundSum(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{{"a", "b"}, {"c", "d"}, {"e"}}
	agg := invariants.Aggregate(p)
	sumLen := 2 + 2 + 1 // 5
	if len(agg) > sumLen {
		t.Fatalf("M(π) = %d > Σ|Aᵢ| = %d: upper bound violated (got %v)", len(agg), sumLen, agg)
	}
	if !invariants.BoundsHold(p) {
		t.Fatal("BoundsHold failed on disjoint stack (sum equality case)")
	}
}

// TestEvaluateI5_HeldInV1Pipeline documents that EvaluateI5 returns Held in
// the V1 dispatcher pipeline: no layer stack is reified in tau.Decision yet.
// BoundsHold (exercised by FuzzI5_CompositionConjonctive in M3.7) is the
// true property guard; this test pins the V1 pipeline behavior.
func TestEvaluateI5_HeldInV1Pipeline(t *testing.T) {
	t.Parallel()
	got := invariants.EvaluateI5(tau.Exchange{}, tau.Decision{})
	if got != invariants.Held {
		t.Fatalf("EvaluateI5 in V1 pipeline = %v, want Held (no stack in Trace yet)", got)
	}
}

// makeLargePile builds a Pile with layerCount layers × entriesPerLayer entries
// for benchmark purposes.
func makeLargePile(layerCount, entriesPerLayer int) invariants.Pile {
	p := make(invariants.Pile, layerCount)
	for i := range p {
		layer := make(invariants.AngleMort, entriesPerLayer)
		for j := range layer {
			layer[j] = string(rune('a'+((i*entriesPerLayer+j)%26))) + string(rune('0'+(j%10)))
		}
		p[i] = layer
	}
	return p
}

// BenchmarkI5_Aggregate measures Aggregate on a 10×50 pile (500 entries total).
func BenchmarkI5_Aggregate(b *testing.B) {
	p := makeLargePile(10, 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = invariants.Aggregate(p)
	}
}

// BenchmarkI5_BoundsHold measures BoundsHold (includes Aggregate) on the same pile.
func BenchmarkI5_BoundsHold(b *testing.B) {
	p := makeLargePile(10, 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = invariants.BoundsHold(p)
	}
}
