package app_test

import (
	"context"
	"os"
	"testing"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestDefaultLLMIsStub — guards the anti-patron: in CI / default mode, no
// external LLM service may be called. The Dispatcher must always use
// llm.Stub unless TAUGO_LLM_BACKEND=real is set explicitly.
//
// Behavioral verification: NewDispatcher's Decide result for a given
// intent must match llm.Stub's score for the same intent. Direct
// reflection on the unexported llm field is intentionally avoided
// (it would be brittle and would test the wiring rather than the
// observable behavior).
func TestDefaultLLMIsStub(t *testing.T) {
	t.Parallel()
	if os.Getenv("TAUGO_LLM_BACKEND") == "real" {
		t.Skip("skipping: TAUGO_LLM_BACKEND=real explicitly set")
	}

	d := app.NewDispatcher()
	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}

	const intent = "test-default-stub-witness"
	stubScore, err := (llm.Stub{}).Interpret(context.Background(), intent)
	if err != nil {
		t.Fatalf("stub Interpret failed: %v", err)
	}

	dec, err := d.Decide(context.Background(), tau.Exchange{
		ID:                "witness",
		IntentDescription: intent,
	})
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}

	if dec.Trace.TauScore != stubScore {
		t.Fatalf("trace TauScore = %f, want %f (stub score for %q)", dec.Trace.TauScore, stubScore, intent)
	}
}
