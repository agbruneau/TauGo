package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// cliBinaryPath holds the path of the binary built once by TestMain.
var cliBinaryPath string

// TestMain builds the cmd/tau binary once for the whole test run.
// All TestEndToEnd_* tests reuse cliBinaryPath instead of rebuilding.
func TestMain(m *testing.M) {
	code := buildAndRun(m)
	os.Exit(code)
}

// buildAndRun isole la séquence build + Run pour que les defer s'exécutent
// avant l'os.Exit (gocritic exitAfterDefer).
func buildAndRun(m *testing.M) int {
	dir, err := os.MkdirTemp("", "tau-cli-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: MkdirTemp:", err)
		return 2
	}
	defer os.RemoveAll(dir)

	cliBinaryPath = filepath.Join(dir, "tau"+exeSuffix())
	cmd := exec.Command("go", "build", "-o", cliBinaryPath, "github.com/agbruneau/taugo/cmd/tau")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: go build:", err)
		return 2
	}
	return m.Run()
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
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

// Regime constants (string representation from tau.Regime.String(), PascalCase).
// Kept here to avoid the black-box main_test importing internal/tau, which
// would couple the E2E test to the internal package layout.
const (
	regimeDeterministe = "Deterministe"
	regimeProbabiliste = "Probabiliste"
)

// TestEndToEnd_DecideDeterministe — composite tau_score falls in the
// hysteresis zone (Deterministe default) for an exchange with static
// contract URI. Intent "creative generation" hashes (FNV-1a 32-bit) to
// 0.262 (S_reasoner_intent). ContractURI lowers D-SENS; resulting composite
// is in [0.35, 0.65) => Deterministe (M2 hysteresis default).
func TestEndToEnd_DecideDeterministe(t *testing.T) {
	t.Parallel()
	input := `{"id":"t1","intent_description":"creative generation","initiator":{"id":"agent","organization":"org-a","delegation_depth":1},"target":{"id":"svc","discovery_mode":1,"contract_uri":"https://api.example.com/v1"}}`
	dec := runDecide(t, cliBinaryPath, input)
	r, _ := dec["regime"].(string)
	if r != regimeDeterministe {
		t.Fatalf("regime = %q, want %q (Deterministe). Full decision: %v", r, regimeDeterministe, dec)
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
	input := `{"id":"t2","intent_description":"hello world","initiator":{"id":"agent","organization":"org-a","delegation_depth":1},"target":{"id":"svc","discovery_mode":1}}`
	dec := runDecide(t, cliBinaryPath, input)
	r, _ := dec["regime"].(string)
	if r != regimeProbabiliste {
		t.Fatalf("regime = %q, want %q (Probabiliste). Full decision: %v", r, regimeProbabiliste, dec)
	}
}

// TestEndToEnd_Calibrate_ProducesValidProfile verifies that `tau calibrate`
// writes a non-empty JSON file with expected top-level keys, and that two
// runs with the same seed and fixed --created-at produce byte-identical output
// (PRD §17 criterion #10).
func TestEndToEnd_Calibrate_ProducesValidProfile(t *testing.T) {
	t.Parallel()

	// Locate mini-corpus relative to module root via __FILE__.
	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	corpus := filepath.Join(moduleRoot, "internal", "calibration", "testdata", "mini-corpus.jsonl")

	tmp := t.TempDir()
	out1 := filepath.Join(tmp, "p1.json")
	out2 := filepath.Join(tmp, "p2.json")

	runCalibrateCLI(t, cliBinaryPath, corpus, out1)
	runCalibrateCLI(t, cliBinaryPath, corpus, out2)

	// Assert non-empty and valid JSON with expected keys.
	data, err := os.ReadFile(out1)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("calibrate produced empty output")
	}
	var profile map[string]any
	if unmarshalErr := json.Unmarshal(data, &profile); unmarshalErr != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", unmarshalErr, data)
	}
	for _, key := range []string{"thresholds", "weights", "date_revision", "version_monographie"} {
		if _, ok := profile[key]; !ok {
			t.Errorf("missing key %q in profile", key)
		}
	}

	// Assert byte-identical output (determinism, PRD §17 #10).
	data2, err := os.ReadFile(out2)
	if err != nil {
		t.Fatalf("read second output: %v", err)
	}
	if !bytes.Equal(data, data2) {
		t.Fatalf("two runs with same seed produced different output\nrun1:\n%s\nrun2:\n%s", data, data2)
	}
}

// runCalibrateCLI invokes `tau calibrate` with fixed flags for reproducibility.
func runCalibrateCLI(t *testing.T, bin, corpus, output string) {
	t.Helper()
	cmd := exec.Command(bin, "calibrate",
		"--corpus", corpus,
		"--output", output,
		"--date-revision", "2026-11-23",
		"--seed", "42",
		"--created-at", "1970-01-01T00:00:00Z",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tau calibrate failed: %v\nstderr: %s", err, stderr.String())
	}
}
