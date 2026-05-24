//go:build !windows

package calibration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agbruneau/taugo/internal/calibration"
)

func TestSave_CreatesSymlinkOnUnix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := calibration.NewStore(dir)

	_, err := s.Save(calibration.DefaultProfile())
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Lstat(filepath.Join(dir, "current.json"))
	if err != nil {
		t.Fatalf("Lstat current.json: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("current.json must be a symlink on Unix")
	}
}
