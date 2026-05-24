package orchestration

// Thresholds — minimal set required for the M1 dispatcher.
// Full Thresholds (with AuthBlock, SensCoherence, etc.) lands in M2/M5.
type Thresholds struct {
	Deterministe float64 // tau_score < theta -> Deterministe
	Probabiliste float64 // tau_score >= theta -> Probabiliste
}

// Ordered reports the ordering invariant.
// Must hold at all times: Deterministe <= Probabiliste.
func (t Thresholds) Ordered() bool { return t.Deterministe <= t.Probabiliste }
