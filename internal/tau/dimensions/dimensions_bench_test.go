package dimensions_test

import (
	"context"
	"testing"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
	"github.com/agbruneau/taugo/internal/testutil"
)

// BenchmarkScoreDSens measures the D-SENS scorer on a representative in-frontier
// exchange, making the >5 % perf-regression guard (directive 5) measurable.
// The deterministic llm.Stub keeps probeReasonerIntent stable across runs.
func BenchmarkScoreDSens(b *testing.B) {
	ctx := context.Background()
	x := testutil.BuildExchange()
	w := dimensions.DefaultSensWeights()
	client := llm.Stub{}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = dimensions.ScoreDSens(ctx, x, w, client)
	}
}

// BenchmarkScoreDAuthority measures the D-AUTORITÉ scorer on a representative
// in-frontier exchange.
func BenchmarkScoreDAuthority(b *testing.B) {
	ctx := context.Background()
	x := testutil.BuildExchange()
	w := dimensions.DefaultAuthorityWeights()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = dimensions.ScoreDAuthority(ctx, x, w)
	}
}

// BenchmarkScoreDInvariant measures the D-INVARIANT scorer on a representative
// in-frontier exchange.
func BenchmarkScoreDInvariant(b *testing.B) {
	ctx := context.Background()
	x := testutil.BuildExchange()
	w := dimensions.DefaultInvariantWeights()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = dimensions.ScoreDInvariant(ctx, x, w)
	}
}
