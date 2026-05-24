package dimensions

import (
	"github.com/agbruneau/taugo/internal/tau"
)

// Score is an alias for tau.Score, promoted to tau to break the import cycle
// that would arise if tau.Trace carried tau/dimensions.Score directly.
// ADR-0008 (Trace ventilée D-SENS / D-AUTORITÉ / D-INVARIANT) — accepté 2026-05-24.
//
// All existing code in this package that constructs Score{...} continues to work
// because alias types share the same underlying struct literal syntax.
type Score = tau.Score

// clamp01 clamps v to [0, 1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
