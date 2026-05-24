// Tests for runMain — the testable entry point of the tau CLI.
// These unit tests cover paths that the E2E binary tests cannot reach.
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunMain_NoArgs_AfficheUsage(t *testing.T) {
	t.Parallel()
	var out, stderr bytes.Buffer
	code := runMain([]string{}, strings.NewReader(""), &out, &stderr)
	if code != 2 {
		t.Fatalf("runMain() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "tau") {
		t.Errorf("stderr missing usage text; got: %s", stderr.String())
	}
}

func TestRunMain_CommandeInconnue_Exit2(t *testing.T) {
	t.Parallel()
	var out, stderr bytes.Buffer
	code := runMain([]string{"foobar"}, strings.NewReader(""), &out, &stderr)
	if code != 2 {
		t.Fatalf("runMain(foobar) = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "foobar") {
		t.Errorf("stderr should mention unknown command; got: %s", stderr.String())
	}
}

func TestRunMain_Version_Exit0(t *testing.T) {
	t.Parallel()
	var out, stderr bytes.Buffer
	code := runMain([]string{"--version"}, strings.NewReader(""), &out, &stderr)
	if code != 0 {
		t.Fatalf("runMain(--version) = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "tau") {
		t.Errorf("stdout missing version text; got: %s", out.String())
	}
}

func TestRunMain_DecideAvecExchangeStdin_Exit0(t *testing.T) {
	t.Parallel()
	var out, stderr bytes.Buffer
	code := runMain([]string{"decide"}, strings.NewReader(validExchangeJSON), &out, &stderr)
	if code != 0 {
		t.Fatalf("runMain(decide) = %d, want 0; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "regime") {
		t.Errorf("stdout missing 'regime' key; got: %s", out.String())
	}
}

func TestRunMain_CalibrateAvecArgsValides_Exit0(t *testing.T) {
	t.Parallel()
	corpus := writeTempCorpus(t, 3)
	output := t.TempDir() + "/profile.json"
	var out, stderr bytes.Buffer
	code := runMain([]string{
		"calibrate",
		"--corpus", corpus,
		"--output", output,
		"--date-revision", "2026-11-23",
		"--seed", "42",
		"--created-at", "1970-01-01T00:00:00Z",
	}, strings.NewReader(""), &out, &stderr)
	if code != 0 {
		t.Fatalf("runMain(calibrate) = %d, want 0; stderr: %s", code, stderr.String())
	}
}
