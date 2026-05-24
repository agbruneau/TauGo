package orchestration

import "github.com/agbruneau/taugo/internal/thresholds"

// Thresholds is the canonical decision threshold type, defined in internal/thresholds.
// Aliased here to preserve existing usages in orchestration without import changes.
// ADR-0006 (Types valeur transverses) — accepté 2026-05-24.
type Thresholds = thresholds.Thresholds

// DefaultThresholds returns the initial thresholds from PRD §11.1.
// Status: Hypothesis — to be corroborated by M4 empirical calibration.
func DefaultThresholds() Thresholds {
	return Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
		HysteresisGap: 0.10,
	}
}
