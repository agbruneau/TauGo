package app_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestSelectLLM_RealBackend_Panics asserts that NewDispatcher panics when
// TAUGO_LLM_BACKEND=real. The real backend is not implemented until M5+;
// the panic is a deliberate sentinel that prevents silent CI regressions
// (see PRD §15.4 and selectLLM inline comment).
func TestSelectLLM_RealBackend_Panics(t *testing.T) {
	t.Setenv("TAUGO_LLM_BACKEND", "real")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from selectLLM with TAUGO_LLM_BACKEND=real, got none")
		}
	}()
	app.NewDispatcher() // must panic
}

// TestDefaultLLMIsStub — guards the anti-patron: in CI / default mode, no
// external LLM service may be called. The Dispatcher must always use
// llm.Stub unless TAUGO_LLM_BACKEND=real is set explicitly.
//
// Behavioral verification (M2): the stub is deterministic — two calls with
// the same exchange must produce the same TauScore. A real LLM would be
// non-deterministic and would fail this property across calls.
// The exchange is constructed to be inside the M2 frontier (DiscoveryMode!=Static,
// HumanInLoop=false, DelegationDepth>0) so Decide actually reaches the
// LLM-backed D-SENS composite rather than bailing out at the frontier.
func TestDefaultLLMIsStub(t *testing.T) {
	t.Parallel()
	if os.Getenv("TAUGO_LLM_BACKEND") == "real" {
		t.Skip("skipping: TAUGO_LLM_BACKEND=real explicitly set")
	}

	d := app.NewDispatcher()
	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}

	x := tau.Exchange{
		ID:                "witness",
		IntentDescription: "test-default-stub-witness",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent-test",
			HumanInLoop:     false,
			Organization:    "org-test",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "svc-test",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1",
		},
	}

	dec1, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("first Decide failed: %v", err)
	}
	dec2, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("second Decide failed: %v", err)
	}

	// Stub is deterministic: same exchange must produce the same TauScore.
	if dec1.Trace.TauScore != dec2.Trace.TauScore {
		t.Fatalf("TauScore not deterministic: call1=%f call2=%f (expected same stub output)",
			dec1.Trace.TauScore, dec2.Trace.TauScore)
	}
}
