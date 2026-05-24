// Tests for runCalibrate, parseDateRev and loadCorpus in the tau CLI.
// Uses t.TempDir() corpora so no external file dependency is required.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// miniCorpusLine is one valid CorpusEntry in JSONL format (mini-corpus schema).
// Using the same schema as internal/calibration/testdata/mini-corpus.jsonl.
const miniCorpusLine = `{"id":"u01","sens_score":0.80,"authority_score":0.20,"invariant_score":0.75,"human_in_loop":true,"has_attestation":true,"expected_regime":"deterministe"}` + "\n"

// writeTempCorpus creates a temporary JSONL file with n copies of miniCorpusLine.
func writeTempCorpus(t *testing.T, n int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "corpus.jsonl")
	content := ""
	for i := 0; i < n; i++ {
		content += miniCorpusLine
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeTempCorpus: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------------
// runCalibrate tests
// ---------------------------------------------------------------------------

func TestRunCalibrate_HappyPath_Exit0(t *testing.T) {
	t.Parallel()
	corpus := writeTempCorpus(t, 5)
	output := filepath.Join(t.TempDir(), "profile.json")
	code := runCalibrate([]string{
		"--corpus", corpus,
		"--output", output,
		"--date-revision", "2026-11-23",
		"--seed", "42",
		"--created-at", "1970-01-01T00:00:00Z",
	})
	if code != 0 {
		t.Fatalf("runCalibrate returned %d, want 0", code)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("calibrate produced empty output")
	}
}

func TestRunCalibrate_MissingCorpusFlag_Exit2(t *testing.T) {
	t.Parallel()
	output := filepath.Join(t.TempDir(), "profile.json")
	code := runCalibrate([]string{
		"--output", output,
		"--date-revision", "2026-11-23",
	})
	if code != 2 {
		t.Fatalf("runCalibrate returned %d, want 2", code)
	}
}

func TestRunCalibrate_MissingOutputFlag_Exit2(t *testing.T) {
	t.Parallel()
	corpus := writeTempCorpus(t, 2)
	code := runCalibrate([]string{
		"--corpus", corpus,
		"--date-revision", "2026-11-23",
	})
	if code != 2 {
		t.Fatalf("runCalibrate returned %d, want 2", code)
	}
}

func TestRunCalibrate_MissingDateRevFlag_Exit2(t *testing.T) {
	t.Parallel()
	corpus := writeTempCorpus(t, 2)
	output := filepath.Join(t.TempDir(), "profile.json")
	code := runCalibrate([]string{
		"--corpus", corpus,
		"--output", output,
	})
	if code != 2 {
		t.Fatalf("runCalibrate returned %d, want 2", code)
	}
}

func TestRunCalibrate_BadDateRev_Exit2(t *testing.T) {
	t.Parallel()
	corpus := writeTempCorpus(t, 2)
	output := filepath.Join(t.TempDir(), "profile.json")
	code := runCalibrate([]string{
		"--corpus", corpus,
		"--output", output,
		"--date-revision", "not-a-date",
	})
	if code != 2 {
		t.Fatalf("runCalibrate returned %d, want 2", code)
	}
}

func TestRunCalibrate_BadCreatedAt_Exit2(t *testing.T) {
	t.Parallel()
	corpus := writeTempCorpus(t, 2)
	output := filepath.Join(t.TempDir(), "profile.json")
	code := runCalibrate([]string{
		"--corpus", corpus,
		"--output", output,
		"--date-revision", "2026-11-23",
		"--created-at", "not-a-timestamp",
	})
	if code != 2 {
		t.Fatalf("runCalibrate returned %d, want 2", code)
	}
}

func TestRunCalibrate_CorpusNotFound_Exit1(t *testing.T) {
	t.Parallel()
	output := filepath.Join(t.TempDir(), "profile.json")
	code := runCalibrate([]string{
		"--corpus", "/nonexistent/path/corpus.jsonl",
		"--output", output,
		"--date-revision", "2026-11-23",
	})
	if code != 1 {
		t.Fatalf("runCalibrate returned %d, want 1 (corpus not found)", code)
	}
}

func TestRunCalibrate_CorpusInvalidJSON_Exit1(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	badCorpus := filepath.Join(dir, "bad.jsonl")
	if err := os.WriteFile(badCorpus, []byte("{bad json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "profile.json")
	code := runCalibrate([]string{
		"--corpus", badCorpus,
		"--output", output,
		"--date-revision", "2026-11-23",
	})
	if code != 1 {
		t.Fatalf("runCalibrate returned %d, want 1 (invalid corpus JSON)", code)
	}
}

// ---------------------------------------------------------------------------
// parseDateRev tests
// ---------------------------------------------------------------------------

func TestParseDateRev_RFC3339_OK(t *testing.T) {
	t.Parallel()
	tm, err := parseDateRev("2026-11-23T00:00:00Z")
	if err != nil {
		t.Fatalf("parseDateRev RFC3339: %v", err)
	}
	if tm.Year() != 2026 || tm.Month() != 11 || tm.Day() != 23 {
		t.Errorf("unexpected time %v", tm)
	}
}

func TestParseDateRev_DateOnly_OK(t *testing.T) {
	t.Parallel()
	tm, err := parseDateRev("2026-11-23")
	if err != nil {
		t.Fatalf("parseDateRev date-only: %v", err)
	}
	if tm.Year() != 2026 || tm.Month() != 11 || tm.Day() != 23 {
		t.Errorf("unexpected time %v", tm)
	}
}

func TestParseDateRev_Invalid_Error(t *testing.T) {
	t.Parallel()
	if _, err := parseDateRev("not-a-date"); err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
}

// ---------------------------------------------------------------------------
// loadCorpus tests
// ---------------------------------------------------------------------------

func TestLoadCorpus_HappyPath(t *testing.T) {
	t.Parallel()
	path := writeTempCorpus(t, 3)
	entries, err := loadCorpus(path)
	if err != nil {
		t.Fatalf("loadCorpus: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("got %d entries, want 3", len(entries))
	}
}

func TestLoadCorpus_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := loadCorpus("/nonexistent/path/corpus.jsonl")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadCorpus_InvalidJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.jsonl")
	if err := os.WriteFile(bad, []byte("{bad json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadCorpus(bad)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
