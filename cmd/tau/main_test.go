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

// TestEndToEnd_DecideDeterministe — composite tau_score falls in the
// hysteresis zone (Deterministe default) for an exchange with static
// contract URI. Intent "creative generation" hashes (FNV-1a 32-bit) to
// 0.262 (S_reasoner_intent). ContractURI lowers D-SENS; resulting composite
// is in [0.35, 0.65) => Deterministe (M2 hysteresis default).
func TestEndToEnd_DecideDeterministe(t *testing.T) {
	t.Parallel()
	bin := buildCLI(t)
	input := `{"id":"t1","intent_description":"creative generation","initiator":{"id":"agent","organization":"org-a","delegation_depth":1},"target":{"id":"svc","discovery_mode":1,"contract_uri":"https://api.example.com/v1"}}`
	dec := runDecide(t, bin, input)
	r, _ := dec["regime"].(float64)
	if r != regimeDeterministe {
		t.Fatalf("regime = %v, want %v (Deterministe). Full decision: %v", r, regimeDeterministe, dec)
	}
	trace, _ := dec["trace"].(map[string]any)
	if id, _ := trace["exchange_id"].(string); id != "t1" {
		t.Fatalf("trace.exchange_id = %q, want \"t1\"", id)
	}
}

// TestEndToEnd_DecideProbabiliste — composite tau_score >= 0.65 for an
// exchange with dynamic discovery and no contract. Intent "hello world"
// hashes (FNV-1a 32-bit) to 0.807 (S_reasoner_intent), keeping D-SENS high.
func TestEndToEnd_DecideProbabiliste(t *testing.T) {
	t.Parallel()
	bin := buildCLI(t)
	input := `{"id":"t2","intent_description":"hello world","initiator":{"id":"agent","organization":"org-a","delegation_depth":1},"target":{"id":"svc","discovery_mode":1}}`
	dec := runDecide(t, bin, input)
	r, _ := dec["regime"].(float64)
	if r != regimeProbabiliste {
		t.Fatalf("regime = %v, want %v (Probabiliste). Full decision: %v", r, regimeProbabiliste, dec)
	}
}
