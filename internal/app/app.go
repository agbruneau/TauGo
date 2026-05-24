package app

import (
	"os"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/orchestration"
)

// defaultThresholds uses M2 values. Calibration in M5 will override these.
var defaultThresholds = orchestration.DefaultThresholds() //nolint:gochecknoglobals // read-only after init; single-point default for app wiring, see PRD §12.2

// NewDispatcher constructs the production Dispatcher.
// Default LLM: deterministic Stub (PRD §15.4).
// TAUGO_LLM_BACKEND=real switches to a real LLM (M5+; currently panics).
func NewDispatcher() *orchestration.Dispatcher {
	return orchestration.NewDispatcher(selectLLM(), defaultThresholds)
}

// selectLLM returns the LLM client based on the TAUGO_LLM_BACKEND env var.
// Default is the deterministic Stub. The real backend is not implemented
// in M1; setting TAUGO_LLM_BACKEND=real panics — explicit signal that
// CI must remain on the stub.
//
//nolint:unparam // M5+ will add a second return path (real LLM); false positive today.
func selectLLM() llm.Client {
	if os.Getenv("TAUGO_LLM_BACKEND") == "real" {
		panic("app: real LLM backend not implemented yet (M5+)")
	}
	return llm.Stub{}
}
