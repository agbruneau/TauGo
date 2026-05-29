package calibration

import (
	"sync/atomic"
)

// AtomicThresholds provides lock-free, concurrency-safe access to the
// Thresholds values using atomic.Int64 with milli-unit encoding.
// This is the calque of FibGo bigfft/fft.go threshold pattern.
//
// Encoding: float64 value stored as int64(v * 1000), i.e. milli-units.
// Range: [0, 1000] for [0.0, 1.0]. Resolution: 0.001.
type AtomicThresholds struct {
	deterministe  atomic.Int64
	probabiliste  atomic.Int64
	authBlock     atomic.Int64
	sensCoherence atomic.Int64
	invCoherence  atomic.Int64
	hysteresisGap atomic.Int64
}

// NewAtomicThresholds constructs an AtomicThresholds from a Thresholds value.
// Panics if the ordering invariant Deterministe <= Probabiliste is violated.
func NewAtomicThresholds(t Thresholds) *AtomicThresholds {
	if t.Deterministe > t.Probabiliste {
		panic("calibration: AtomicThresholds ordering violated (Deterministe > Probabiliste)")
	}
	at := &AtomicThresholds{}
	at.deterministe.Store(millis(t.Deterministe))
	at.probabiliste.Store(millis(t.Probabiliste))
	at.authBlock.Store(millis(t.AuthBlock))
	at.sensCoherence.Store(millis(t.SensCoherence))
	at.invCoherence.Store(millis(t.InvCoherence))
	at.hysteresisGap.Store(millis(t.HysteresisGap))
	return at
}

// Deterministe returns the current Deterministe threshold as float64.
func (at *AtomicThresholds) Deterministe() float64 {
	return fromMillis(at.deterministe.Load())
}

// Probabiliste returns the current Probabiliste threshold as float64.
func (at *AtomicThresholds) Probabiliste() float64 {
	return fromMillis(at.probabiliste.Load())
}

// AuthBlock returns the current AuthBlock threshold as float64.
func (at *AtomicThresholds) AuthBlock() float64 {
	return fromMillis(at.authBlock.Load())
}

// SensCoherence returns the current SensCoherence threshold as float64.
func (at *AtomicThresholds) SensCoherence() float64 {
	return fromMillis(at.sensCoherence.Load())
}

// InvCoherence returns the current InvCoherence threshold as float64.
func (at *AtomicThresholds) InvCoherence() float64 {
	return fromMillis(at.invCoherence.Load())
}

// HysteresisGap returns the current HysteresisGap as float64.
func (at *AtomicThresholds) HysteresisGap() float64 {
	return fromMillis(at.hysteresisGap.Load())
}

// Snapshot returns the current values as an immutable Thresholds copy.
func (at *AtomicThresholds) Snapshot() Thresholds {
	return Thresholds{
		Deterministe:  at.Deterministe(),
		Probabiliste:  at.Probabiliste(),
		AuthBlock:     at.AuthBlock(),
		SensCoherence: at.SensCoherence(),
		InvCoherence:  at.InvCoherence(),
		HysteresisGap: at.HysteresisGap(),
	}
}

// SetTuning updates every threshold from t. Each individual Store is atomic,
// but SetTuning is NOT a single atomic transaction: the six Stores happen in
// sequence, so a concurrent Snapshot may observe a partially-updated state
// (some fields from t, some from the previous value). Callers requiring an
// all-or-nothing view must add external synchronization.
//
// NOTE: AtomicThresholds is not yet wired into the dispatcher; it is a
// forward-looking primitive for hot-reload tuning (cf. FibGo bigfft pattern).
//
// Panics if the ordering invariant Deterministe <= Probabiliste would be violated.
func (at *AtomicThresholds) SetTuning(t Thresholds) {
	if t.Deterministe > t.Probabiliste {
		panic("calibration: SetTuning ordering violated (Deterministe > Probabiliste)")
	}
	at.deterministe.Store(millis(t.Deterministe))
	at.probabiliste.Store(millis(t.Probabiliste))
	at.authBlock.Store(millis(t.AuthBlock))
	at.sensCoherence.Store(millis(t.SensCoherence))
	at.invCoherence.Store(millis(t.InvCoherence))
	at.hysteresisGap.Store(millis(t.HysteresisGap))
}

// millis converts a float64 in [0,1] to milli-units int64.
func millis(v float64) int64 { return int64(v * 1000) }

// fromMillis converts milli-units int64 back to float64.
func fromMillis(v int64) float64 { return float64(v) / 1000.0 }
