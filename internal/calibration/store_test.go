package calibration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agbruneau/taugo/internal/calibration"
)

func TestStore_SaveAndLoad_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := calibration.NewStore(dir)
	p := calibration.DefaultProfile()

	storedPath, err := s.Save(p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	expected := s.Path(p.ID, p.Version)
	if storedPath != expected {
		t.Errorf("Save returned %q; want %q", storedPath, expected)
	}

	got, err := s.Load(p.ID, p.Version)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Compare fields that must survive the round-trip exactly.
	assertProfileEqual(t, p, got)
}

func TestStore_SaveTwiceSameProfile_ByteIdentical(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := calibration.NewStore(dir)
	p := calibration.DefaultProfile()

	path1, err := s.Save(p)
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}

	// Modify p.Version to produce a second distinct file.
	p2 := p
	p2.Version = "0.1.1"
	path2, err := s.Save(p2)
	if err != nil {
		t.Fatalf("second Save (v0.1.1): %v", err)
	}

	// Save p (original version) a second time to a fresh store (different dir)
	// to ensure byte-identical output from MarshalCanonical.
	dir2 := t.TempDir()
	s2 := calibration.NewStore(dir2)
	pathA, err := s2.Save(p)
	if err != nil {
		t.Fatalf("Save to dir2: %v", err)
	}

	sum1 := mustSHA256(t, path1)
	sumA := mustSHA256(t, pathA)
	if sum1 != sumA {
		t.Errorf("byte-identity failed: sha256(%q)=%s sha256(%q)=%s",
			path1, sum1, pathA, sumA)
	}

	// path2 must differ (different Version field).
	sum2 := mustSHA256(t, path2)
	if sum1 == sum2 {
		t.Error("expected different sha256 for different Version, got identical")
	}
}

func TestStore_CurrentJSONPointsToLatest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := calibration.NewStore(dir)
	p := calibration.DefaultProfile()

	if _, err := s.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.LoadCurrent()
	if err != nil {
		t.Fatalf("LoadCurrent: %v", err)
	}
	assertProfileEqual(t, p, got)
}

func TestStore_LoadCurrent_NoFileError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := calibration.NewStore(dir)

	_, err := s.LoadCurrent()
	if err == nil {
		t.Fatal("expected error when current.json absent, got nil")
	}
}

func TestStore_Save_OverwritesCurrentOnUpdate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := calibration.NewStore(dir)

	p1 := calibration.DefaultProfile()
	p1.Version = "0.1.0"

	p2 := calibration.DefaultProfile()
	p2.Version = "0.2.0"

	if _, err := s.Save(p1); err != nil {
		t.Fatalf("Save v0.1.0: %v", err)
	}
	if _, err := s.Save(p2); err != nil {
		t.Fatalf("Save v0.2.0: %v", err)
	}

	got, err := s.LoadCurrent()
	if err != nil {
		t.Fatalf("LoadCurrent: %v", err)
	}
	if got.Version != "0.2.0" {
		t.Errorf("LoadCurrent.Version = %q; want 0.2.0", got.Version)
	}
}

// assertProfileEqual compares the stable fields of two profiles.
// CreatedAt is excluded because DefaultProfile stamps time.Now().
func assertProfileEqual(t *testing.T, want, got calibration.Profile) {
	t.Helper()
	if want.ID != got.ID {
		t.Errorf("ID: got %q, want %q", got.ID, want.ID)
	}
	if want.Version != got.Version {
		t.Errorf("Version: got %q, want %q", got.Version, want.Version)
	}
	if want.VersionMonographie != got.VersionMonographie {
		t.Errorf("VersionMonographie: got %q, want %q", got.VersionMonographie, want.VersionMonographie)
	}
	if want.Thresholds != got.Thresholds {
		t.Errorf("Thresholds mismatch: got %+v, want %+v", got.Thresholds, want.Thresholds)
	}
}

// mustSHA256 reads the file at path and returns its hex SHA-256 digest.
func mustSHA256(t *testing.T, path string) string {
	t.Helper()
	s := calibration.NewStore(filepath.Dir(path))
	sum, err := s.ExportSHA256(path)
	if err != nil {
		t.Fatalf("sha256(%q): %v", path, err)
	}
	return sum
}

// TestStore_Save_CurrentJSONExists checks that current.json is created after Save.
func TestStore_Save_CurrentJSONExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := calibration.NewStore(dir)

	if _, err := s.Save(calibration.DefaultProfile()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	currentPath := filepath.Join(dir, "current.json")
	info, err := os.Stat(currentPath)
	if err != nil {
		t.Fatalf("current.json must exist after Save: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("current.json must be non-empty")
	}
}
