package thresholds_test

import (
	"testing"

	"github.com/agbruneau/taugo/internal/thresholds"
)

func TestThresholds_ZeroValue(t *testing.T) {
	t.Parallel()
	var th thresholds.Thresholds
	if th.Deterministe != 0 || th.Probabiliste != 0 || th.AuthBlock != 0 {
		t.Error("zero value should have all fields at 0")
	}
}

func TestThresholds_AllFields(t *testing.T) {
	t.Parallel()
	th := thresholds.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
		HysteresisGap: 0.10,
	}
	if th.Deterministe != 0.35 {
		t.Errorf("Deterministe: got %v, want 0.35", th.Deterministe)
	}
	if th.Probabiliste != 0.65 {
		t.Errorf("Probabiliste: got %v, want 0.65", th.Probabiliste)
	}
	if th.AuthBlock != 0.85 {
		t.Errorf("AuthBlock: got %v, want 0.85", th.AuthBlock)
	}
	if th.SensCoherence != 0.50 {
		t.Errorf("SensCoherence: got %v, want 0.50", th.SensCoherence)
	}
	if th.InvCoherence != 0.50 {
		t.Errorf("InvCoherence: got %v, want 0.50", th.InvCoherence)
	}
	if th.HysteresisGap != 0.10 {
		t.Errorf("HysteresisGap: got %v, want 0.10", th.HysteresisGap)
	}
}

func TestThresholds_OrderedInvariant(t *testing.T) {
	t.Parallel()
	th := thresholds.Thresholds{Deterministe: 0.35, Probabiliste: 0.65}
	if th.Deterministe > th.Probabiliste {
		t.Error("ordering invariant violated: Deterministe > Probabiliste")
	}
}
