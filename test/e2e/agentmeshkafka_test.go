//go:build integration
// +build integration

package e2e

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
	"github.com/agbruneau/taugo/internal/tau"
)

// testdataPath resolves a path under internal/bridge/agentmeshkafka/testdata
// relative to this file's location, which is at test/e2e/ within the module.
func testdataPath(filename string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile is .../test/e2e/agentmeshkafka_test.go
	// testdata is at .../internal/bridge/agentmeshkafka/testdata/
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(root, "internal", "bridge", "agentmeshkafka", "testdata", filename)
}

// validRegimes is the set of all regime values defined by PRD §3.
var validRegimes = map[tau.Regime]struct{}{
	tau.Deterministe: {},
	tau.Probabiliste: {},
	tau.Refus:        {},
}

// TestE2E_AgentMeshKafka_FullPipeline exercises the full M4 pipeline:
// FileAdapter -> StreamAsTauExchanges -> Dispatcher.Decide against the
// golden-3.jsonl corpus. Verifies that every Exchange produces a Decision
// with a matching ExchangeID and a valid Regime.
func TestE2E_AgentMeshKafka_FullPipeline(t *testing.T) {
	t.Parallel()

	adapter, err := agentmeshkafka.NewFileAdapter(testdataPath("golden-3.jsonl"))
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	defer adapter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exchanges, errc := app.StreamAsTauExchanges(ctx, adapter, []string{"taugo-traces"})

	d := app.NewDispatcher()
	var decisions []tau.Decision
	for x := range exchanges {
		dec, err := d.Decide(ctx, x)
		if err != nil {
			t.Fatalf("Decide(%s): %v", x.ID, err)
		}
		if dec.Trace.ExchangeID != x.ID {
			t.Errorf("Decision.Trace.ExchangeID = %q, want %q", dec.Trace.ExchangeID, x.ID)
		}
		if _, ok := validRegimes[dec.Regime]; !ok {
			t.Errorf("Decision.Regime = %v, not in {Deterministe, Probabiliste, Refus}", dec.Regime)
		}
		decisions = append(decisions, dec)
	}

	for adapterErr := range errc {
		t.Errorf("adapter error: %v", adapterErr)
	}

	// golden-3.jsonl has 3 lines; topic filter "taugo-traces" passes none of
	// them (source_topic is "agentic.bfsi" / "agentic.support"). Test without
	// topic filter to verify full pipeline behaviour.
	t.Logf("decisions collected with topic filter 'taugo-traces': %d", len(decisions))
}

// TestE2E_AgentMeshKafka_FullPipeline_NoTopicFilter repeats the pipeline
// without a topic filter so all 3 corpus lines are processed.
func TestE2E_AgentMeshKafka_FullPipeline_NoTopicFilter(t *testing.T) {
	t.Parallel()

	adapter, err := agentmeshkafka.NewFileAdapter(testdataPath("golden-3.jsonl"))
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	defer adapter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exchanges, errc := app.StreamAsTauExchanges(ctx, adapter, nil)

	d := app.NewDispatcher()
	var decisions []tau.Decision
	for x := range exchanges {
		dec, err := d.Decide(ctx, x)
		if err != nil {
			t.Fatalf("Decide(%s): %v", x.ID, err)
		}
		if dec.Trace.ExchangeID != x.ID {
			t.Errorf("Decision.Trace.ExchangeID = %q, want %q", dec.Trace.ExchangeID, x.ID)
		}
		if _, ok := validRegimes[dec.Regime]; !ok {
			t.Errorf("Decision.Regime = %v, not in {Deterministe, Probabiliste, Refus}", dec.Regime)
		}
		decisions = append(decisions, dec)
	}

	for adapterErr := range errc {
		t.Errorf("adapter error: %v", adapterErr)
	}

	if len(decisions) < 3 {
		t.Fatalf("got %d decisions, want >= 3 (golden-3 corpus has 3 lines)", len(decisions))
	}

	regimeCounts := map[tau.Regime]int{}
	for _, dec := range decisions {
		regimeCounts[dec.Regime]++
	}
	t.Logf("regime distribution: Det=%d, Prob=%d, Refus=%d",
		regimeCounts[tau.Deterministe], regimeCounts[tau.Probabiliste], regimeCounts[tau.Refus])
}

// TestE2E_AgentMeshKafka_MalformedCorpus verifies resilient stream behaviour:
// 1 malformed line reports an error without halting the stream; the 2 valid
// lines still produce decisions.
func TestE2E_AgentMeshKafka_MalformedCorpus(t *testing.T) {
	t.Parallel()

	adapter, err := agentmeshkafka.NewFileAdapter(testdataPath("golden-3-malformed.jsonl"))
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	defer adapter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exchanges, errc := app.StreamAsTauExchanges(ctx, adapter, nil)

	d := app.NewDispatcher()
	var decisions []tau.Decision
	for x := range exchanges {
		dec, err := d.Decide(ctx, x)
		if err != nil {
			t.Fatalf("Decide(%s): %v", x.ID, err)
		}
		decisions = append(decisions, dec)
	}

	var errCount int
	for range errc {
		errCount++
	}

	if errCount < 1 {
		t.Errorf("got %d adapter errors, want >= 1 (malformed corpus has 1 bad line)", errCount)
	}
	if len(decisions) < 2 {
		t.Errorf("got %d decisions, want >= 2 (malformed corpus has 2 valid lines)", len(decisions))
	}
	t.Logf("malformed corpus: %d decisions, %d errors", len(decisions), errCount)
}
