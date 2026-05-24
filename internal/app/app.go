package app

import (
	"fmt"
	"os"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/calibration"
	customerrors "github.com/agbruneau/taugo/internal/errors"
	"github.com/agbruneau/taugo/internal/orchestration"
)

// defaultThresholds uses M2 values. Calibration in M5 will override these.
var defaultThresholds = orchestration.DefaultThresholds() //nolint:gochecknoglobals // read-only after init; single-point default for app wiring, see PRD §12.2

// NewDispatcher constructs the production Dispatcher with a default calibration
// profile, ensuring the profile-expiry guard (PRD §10 step 3, anti-pattern #3)
// is always active. Default LLM: deterministic Stub (PRD §15.4).
// TAUGO_LLM_BACKEND=real switches to a real LLM (M5+; not yet implemented).
//
// P0-02 fix: uses NewDispatcherWithProfile so that Decide never silently
// tolerates an expired profile (PRD §7.3 case 4).
func NewDispatcher() *orchestration.Dispatcher {
	client, err := selectLLM(os.Getenv("TAUGO_LLM_BACKEND"))
	if err != nil {
		panic(fmt.Sprintf("app: selectLLM: %v", err))
	}
	p := calibration.DefaultProfile()
	return orchestration.NewDispatcherWithProfile(client, defaultThresholds, &p)
}

// selectLLM returns the LLM client corresponding to the given provider name.
// Returns *customerrors.DispatchError if the provider is unknown or not yet implemented.
func selectLLM(provider string) (llm.Client, error) {
	switch provider {
	case "stub", "":
		return llm.Stub{}, nil
	case "real":
		return nil, &customerrors.DispatchError{
			Stage:  0,
			Detail: fmt.Sprintf("LLM provider %q not implemented yet (M5+)", provider),
		}
	default:
		return nil, &customerrors.DispatchError{
			Stage:  0,
			Detail: fmt.Sprintf("LLM provider %q inconnu", provider),
		}
	}
}
