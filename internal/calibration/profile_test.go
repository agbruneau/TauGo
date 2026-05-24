package calibration_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/calibration"
)

func TestDefaultProfile_FieldsNonEmpty(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	if p.ID == "" {
		t.Error("Profile.ID must not be empty")
	}
	if p.Version == "" {
		t.Error("Profile.Version must not be empty")
	}
	if p.VersionMonographie == "" {
		t.Error("Profile.VersionMonographie must not be empty")
	}
}

func TestDefaultProfile_DateRevisionAfterCreation(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	if !p.DateRevision.After(p.CreatedAt) {
		t.Fatalf("DateRevision (%v) must be after CreatedAt (%v)", p.DateRevision, p.CreatedAt)
	}
}

func TestDefaultProfile_DateRevisionAtLeast6MonthsAhead(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	minRevision := time.Now().UTC().AddDate(0, 6, 0)
	if p.DateRevision.Before(minRevision) {
		t.Fatalf("DateRevision %v is less than 6 months ahead of now", p.DateRevision)
	}
}

func TestDefaultProfile_WeightsSumToOne(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	const eps = 1e-9

	// Composite dimension weights
	compositeSum := p.Weights.DSens + p.Weights.DAuthority + p.Weights.DInvariant
	if compositeSum < 1.0-eps || compositeSum > 1.0+eps {
		t.Fatalf("composite weights sum = %f, want 1.0", compositeSum)
	}

	// Per-dimension probe weights
	probeSums := map[string]float64{}
	for _, v := range p.Weights.SensProbes {
		probeSums["sens"] += v
	}
	for _, v := range p.Weights.AuthorityProbes {
		probeSums["authority"] += v
	}
	for _, v := range p.Weights.InvariantProbes {
		probeSums["invariant"] += v
	}
	for dim, sum := range probeSums {
		if sum < 1.0-eps || sum > 1.0+eps {
			t.Errorf("%s probe weights sum = %f, want 1.0", dim, sum)
		}
	}
}

func TestDefaultProfile_ThresholdOrderingInvariant(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	if p.Thresholds.Deterministe > p.Thresholds.Probabiliste {
		t.Fatalf("Deterministe (%f) > Probabiliste (%f): ordering violated",
			p.Thresholds.Deterministe, p.Thresholds.Probabiliste)
	}
}
