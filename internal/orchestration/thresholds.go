package orchestration

// Thresholds holds the complete set of decision thresholds for the dispatcher.
// M1 had Deterministe and Probabiliste only; M2 adds the guard thresholds.
type Thresholds struct {
	Deterministe  float64 // tau_score < theta -> Deterministe
	Probabiliste  float64 // tau_score >= theta -> Probabiliste
	AuthBlock     float64 // D-AUTORITE >= AuthBlock && Attestation==nil -> Refus (I3)
	SensCoherence float64 // I4 guard: D-SENS must be >= SensCoherence when D-INVARIANT >= InvCoherence
	InvCoherence  float64 // I4 guard: D-INVARIANT threshold that triggers the coherence check
}

// Ordered reports the ordering invariant.
// Must hold at all times: Deterministe <= Probabiliste.
func (t Thresholds) Ordered() bool { return t.Deterministe <= t.Probabiliste }

// DefaultThresholds returns the initial thresholds from PRD §11.1.
// Status: Hypothesis — to be corroborated by M4 empirical calibration.
func DefaultThresholds() Thresholds {
	return Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	}
}
