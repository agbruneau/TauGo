//go:build windows

package calibration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agbruneau/taugo/internal/calibration"
)

// TestSave_FallbackOnWindowsWhenSymlinkUnavailable verifies that after Save,
// current.json is non-empty regardless of whether os.Symlink succeeded
// (Developer Mode enabled) or fell back to a plain copy.
//
// When Developer Mode is OFF, Save writes a copy and a current.json.source
// sidecar. When Developer Mode is ON, Save writes a real symlink — the
// sidecar is then absent, which is also acceptable.
func TestSave_FallbackOnWindowsWhenSymlinkUnavailable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := calibration.NewStore(dir)
	p := calibration.DefaultProfile()

	_, err := s.Save(p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// os.Stat follows symlinks: works in both modes.
	info, err := os.Stat(filepath.Join(dir, "current.json"))
	if err != nil {
		t.Fatalf("current.json must exist after Save: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("current.json must be non-empty (copy or symlink-resolved)")
	}

	// Sidecar exists only in the fallback (copy) branch.
	sidecarPath := filepath.Join(dir, "current.json.source")
	if _, statErr := os.Stat(sidecarPath); statErr == nil {
		b, readErr := os.ReadFile(sidecarPath)
		if readErr != nil {
			t.Fatalf("read sidecar: %v", readErr)
		}
		if !strings.Contains(string(b), p.ID) {
			t.Fatalf("sidecar must record source filename containing %q, got %q", p.ID, b)
		}
	}
	// No sidecar → symlink branch — also acceptable; no assertion needed.
}
