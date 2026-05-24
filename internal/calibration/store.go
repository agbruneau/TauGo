package calibration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Store manages versioned Profile files under a single directory.
// The zero value is not usable; use NewStore or set Dir explicitly.
//
// File layout:
//
//	Dir/
//	  <ID>-<Version>.json     canonical Profile JSON
//	  current.json            symlink → <ID>-<Version>.json  (Unix)
//	                          plain copy                     (Windows fallback)
//	  current.json.source     records the target filename    (Windows fallback only)
type Store struct {
	// Dir is the directory where profiles are written. Must be non-empty.
	Dir string
}

// NewStore returns a Store rooted at dir.
func NewStore(dir string) *Store { return &Store{Dir: dir} }

// Path returns the absolute path for the given id+version pair without
// creating any file. Safe to call from tests.
func (s *Store) Path(id, version string) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%s-%s.json", id, version))
}

// Save serializes p using MarshalCanonical and writes it to
// Dir/<ID>-<Version>.json (permissions 0o600). It then refreshes
// Dir/current.json to point at the new file:
//   - On Linux/macOS: a relative symlink is used.
//   - On Windows (or when os.Symlink is unavailable): a plain copy is
//     written and a Dir/current.json.source sidecar records the target
//     filename so that the traceability the symlink would have given is
//     preserved.
//
// Save returns the path of the versioned file (not current.json).
func (s *Store) Save(p Profile) (string, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return "", fmt.Errorf("calibration Store.Save: mkdir %q: %w", s.Dir, err)
	}

	name := fmt.Sprintf("%s-%s.json", p.ID, p.Version)
	target := filepath.Join(s.Dir, name)

	b, err := MarshalCanonical(p)
	if err != nil {
		return "", fmt.Errorf("calibration Store.Save: marshal: %w", err)
	}

	if err := os.WriteFile(target, b, 0o600); err != nil {
		return "", fmt.Errorf("calibration Store.Save: write %q: %w", target, err)
	}

	if err := s.refreshCurrent(name, b); err != nil {
		return "", fmt.Errorf("calibration Store.Save: current.json: %w", err)
	}

	return target, nil
}

// refreshCurrent updates Dir/current.json. name is the bare filename
// (not the full path) of the new profile. b is the serialized content
// used for the copy fallback.
func (s *Store) refreshCurrent(name string, b []byte) error {
	currentPath := filepath.Join(s.Dir, "current.json")
	sidecarPath := currentPath + ".source"

	// Remove stale files; ignore errors (they may not exist).
	_ = os.Remove(currentPath)
	_ = os.Remove(sidecarPath)

	// Attempt a relative symlink (works on Unix and Windows with Developer Mode).
	if err := os.Symlink(name, currentPath); err == nil {
		return nil
	}

	// Symlink failed — fall back to a plain copy.
	slog.Info("calibration Store: symlink unavailable, using file copy for current.json",
		"dir", s.Dir, "target", name)

	if err := os.WriteFile(currentPath, b, 0o600); err != nil {
		return fmt.Errorf("write copy: %w", err)
	}
	if err := os.WriteFile(sidecarPath, []byte(name+"\n"), 0o600); err != nil {
		return fmt.Errorf("write sidecar: %w", err)
	}
	return nil
}

// Load reads Dir/<id>-<version>.json and deserializes it via UnmarshalCanonical.
func (s *Store) Load(id, version string) (Profile, error) {
	path := s.Path(id, version)
	return s.readProfile(path)
}

// LoadCurrent resolves Dir/current.json (symlink or plain copy) and
// deserializes the Profile it contains.
func (s *Store) LoadCurrent() (Profile, error) {
	currentPath := filepath.Join(s.Dir, "current.json")
	return s.readProfile(currentPath)
}

// readProfile reads and deserializes the Profile at path.
func (s *Store) readProfile(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, fmt.Errorf("calibration Store.readProfile %q: %w", path, err)
	}
	p, err := UnmarshalCanonical(data)
	if err != nil {
		return Profile{}, fmt.Errorf("calibration Store.readProfile %q: %w", path, err)
	}
	return p, nil
}

// ExportSHA256 returns the hex-encoded SHA-256 digest of the file at path.
// Exported for use in tests to assert byte-identical writes (PRD §17 #10).
func (s *Store) ExportSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("calibration Store.ExportSHA256: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("calibration Store.ExportSHA256: hash: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
