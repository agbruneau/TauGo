package orchestration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/testutil"
)

// benchThresholds mirrors the canonical band used across dispatcher_test.go so
// the benchmarked decision path matches the regimes asserted in unit tests.
func benchThresholds() orchestration.Thresholds {
	return orchestration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	}
}

// BenchmarkDecide exercises the public Decide hot path across the three terminal
// regimes (Deterministe / Probabiliste / Refus). It makes the >5 % perf-regression
// guard (directive 5) measurable for the full eight-step pipeline, which until now
// had no benchmark coverage. The deterministic fakeLLM keeps the reasoner probe
// stable across iterations.
func BenchmarkDecide(b *testing.B) {
	ctx := context.Background()

	b.Run("Deterministe", func(b *testing.B) {
		// fakeLLM=0 + static contract + no intent => composite < Deterministe band.
		d := orchestration.NewDispatcher(fakeLLM{score: 0.0}, benchThresholds())
		x := newDeterministeExchange("b-det")
		benchDecide(ctx, b, d, x, tau.Deterministe)
	})

	b.Run("Probabiliste", func(b *testing.B) {
		// High LLM score + in-frontier exchange with no contract => composite above band.
		d := orchestration.NewDispatcher(fakeLLM{score: 0.80}, benchThresholds())
		x := testutil.BuildExchange(testutil.WithID("b-prob"))
		benchDecide(ctx, b, d, x, tau.Probabiliste)
	})

	b.Run("Refus", func(b *testing.B) {
		// Static, human-anchored, depth-0 exchange fails the M2 frontier (step 1),
		// exercising the first-rank refusal short path.
		d := orchestration.NewDispatcher(fakeLLM{score: 0.0}, benchThresholds())
		x := tau.Exchange{
			ID:           "b-refus",
			DiscoveredAt: time.Now(),
			Initiator:    tau.Principal{ID: "agent", HumanInLoop: true, Organization: "org-x"},
			Target:       tau.Capability{ID: "target-svc", DiscoveryMode: tau.Static, ContractURI: "https://api.example.com/v1"},
		}
		benchDecide(ctx, b, d, x, tau.Refus)
	})
}

// benchDecide asserts the expected regime once (fail-fast on a mis-built fixture)
// before running Decide in the timed loop.
func benchDecide(ctx context.Context, b *testing.B, d *orchestration.Dispatcher, x tau.Exchange, want tau.Regime) {
	b.Helper()

	dec, err := d.Decide(ctx, x)
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != want {
		b.Fatalf("expected %v, got %v", want, dec.Regime)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = d.Decide(ctx, x)
	}
}
