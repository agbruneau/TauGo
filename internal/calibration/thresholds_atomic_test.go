package calibration_test

import (
	"sync"
	"testing"

	"github.com/agbruneau/taugo/internal/calibration"
)

func defaultThresholds() calibration.Thresholds {
	return calibration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
		HysteresisGap: 0.10,
	}
}

func TestAtomicThresholds_Roundtrip(t *testing.T) {
	t.Parallel()
	at := calibration.NewAtomicThresholds(defaultThresholds())
	snap := at.Snapshot()
	const eps = 0.001 // milli-unit resolution
	if snap.Deterministe < 0.35-eps || snap.Deterministe > 0.35+eps {
		t.Errorf("Deterministe = %f, want ~0.35", snap.Deterministe)
	}
	if snap.AuthBlock < 0.85-eps || snap.AuthBlock > 0.85+eps {
		t.Errorf("AuthBlock = %f, want ~0.85", snap.AuthBlock)
	}
}

func TestAtomicThresholds_OrderingInvariant(t *testing.T) {
	t.Parallel()
	at := calibration.NewAtomicThresholds(defaultThresholds())
	if at.Deterministe() > at.Probabiliste() {
		t.Fatalf("ordering violated: Deterministe (%f) > Probabiliste (%f)",
			at.Deterministe(), at.Probabiliste())
	}
}

func TestAtomicThresholds_PanicOnOrderingViolation(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on ordering violation, got none")
		}
	}()
	_ = calibration.NewAtomicThresholds(calibration.Thresholds{
		Deterministe: 0.80,
		Probabiliste: 0.20, // violates Deterministe <= Probabiliste
	})
}

func TestAtomicThresholds_SetTuning(t *testing.T) {
	t.Parallel()
	at := calibration.NewAtomicThresholds(defaultThresholds())
	newT := calibration.Thresholds{
		Deterministe:  0.40,
		Probabiliste:  0.70,
		AuthBlock:     0.90,
		SensCoherence: 0.55,
		InvCoherence:  0.55,
		HysteresisGap: 0.15,
	}
	at.SetTuning(newT)
	snap := at.Snapshot()
	const eps = 0.001
	if snap.Deterministe < 0.40-eps || snap.Deterministe > 0.40+eps {
		t.Errorf("after SetTuning: Deterministe = %f, want ~0.40", snap.Deterministe)
	}
}

func TestAtomicThresholds_ConcurrentReadsSafe(t *testing.T) {
	t.Parallel()
	at := calibration.NewAtomicThresholds(defaultThresholds())
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = at.Snapshot()
			_ = at.Deterministe()
			_ = at.AuthBlock()
		}()
	}
	wg.Wait()
}
