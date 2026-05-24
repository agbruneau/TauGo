package main_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// buildCLI compiles the cmd/tau binary into a temp file and returns its path.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "tau")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "github.com/agbruneau/taugo/cmd/tau")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

// runDecide pipes input on stdin and decodes the JSON Decision from stdout.
func runDecide(t *testing.T, bin, input string) map[string]any {
	t.Helper()
	cmd := exec.Command(bin, "decide")
	cmd.Stdin = bytes.NewBufferString(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tau decide failed: %v\nstderr: %s", err, stderr.String())
	}
	var dec map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &dec); err != nil {
		t.Fatalf("decode decision: %v\nraw: %s", err, stdout.String())
	}
	return dec
}

// Regime constants (mirror of tau.Regime iota; kept here to avoid the
// black-box main_test importing internal/tau, which would couple the
// E2E test to the internal package layout).
const (
	regimeDeterministe = 1.0
	regimeProbabiliste = 2.0
)

// TestEndToEnd_DecideDeterministe — intent "creative generation" hashes
// (FNV-1a 32-bit) to 0.262, below the 0.35 Deterministe threshold.
func TestEndToEnd_DecideDeterministe(t *testing.T) {
	t.Parallel()
	bin := buildCLI(t)
	dec := runDecide(t, bin, `{"id":"t1","intent_description":"creative generation"}`)
	r, _ := dec["regime"].(float64)
	if r != regimeDeterministe {
		t.Fatalf("regime = %v, want %v (Deterministe). Full decision: %v", r, regimeDeterministe, dec)
	}
	trace, _ := dec["trace"].(map[string]any)
	if id, _ := trace["exchange_id"].(string); id != "t1" {
		t.Fatalf("trace.exchange_id = %q, want \"t1\"", id)
	}
}

// TestEndToEnd_DecideProbabiliste — intent "hello world" hashes
// (FNV-1a 32-bit) to 0.807, above the 0.65 Probabiliste threshold.
func TestEndToEnd_DecideProbabiliste(t *testing.T) {
	t.Parallel()
	bin := buildCLI(t)
	dec := runDecide(t, bin, `{"id":"t2","intent_description":"hello world"}`)
	r, _ := dec["regime"].(float64)
	if r != regimeProbabiliste {
		t.Fatalf("regime = %v, want %v (Probabiliste). Full decision: %v", r, regimeProbabiliste, dec)
	}
}
