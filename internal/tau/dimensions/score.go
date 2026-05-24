package dimensions

import "time"

// Score is a normalized [0,1] score for a single dimension, with full
// probe-level traceability. Used by D-SENS, D-AUTORITÉ, and D-INVARIANT.
type Score struct {
	Value      float64            // composite value in [0,1]
	Probes     map[string]float64 // individual probe values
	Weights    map[string]float64 // weights in effect at compute time
	ComputedAt time.Time
}

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
