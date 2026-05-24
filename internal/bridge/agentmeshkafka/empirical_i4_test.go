//go:build empirical

package agentmeshkafka_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

// empiricalI4Report is the JSON structure written to testdata/empirical-i4-results.json.
// Fields are ordered alphabetically to ensure stable JSON output.
type empiricalI4Report struct {
	CorpusPath              string                            `json:"corpus_path"`
	CorpusSHA256            string                            `json:"corpus_sha256"`
	ObservationsNonModelisees []string                        `json:"observations_non_modelisees,omitempty"`
	ProfileVersion          string                            `json:"profile_version"`
	Sensitivity             float64                           `json:"sensitivity"`
	Specificity             float64                           `json:"specificity"`
	Stats                   agentmeshkafka.EmpiricalI4Stats   `json:"stats"`
	Timestamp               string                            `json:"timestamp"`
	TotalDecisions          int                               `json:"total_decisions"`
}

// TestEmpiricalI4Campaign ingests the 120-line synthetic corpus, dispatches
// every exchange, classifies each Decision against the I4 invariant, and writes
// a JSON report to testdata/empirical-i4-results.json.
//
// The test always passes for the empirical build tag — findings are captured
// in the report for M4.7. Sensitivity below 0.7 is logged, not fatal.
func TestEmpiricalI4Campaign(t *testing.T) {
	t.Parallel()

	// Resolve corpus path relative to the package directory (os.Getwd in
	// Go tests is set to the package being tested).
	pkgDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// pkgDir = .../TauGo/internal/bridge/agentmeshkafka — 3 levels up to repo root.
	repoRoot := filepath.Join(pkgDir, "..", "..", "..")
	corpusPath := filepath.Join(repoRoot, "cmd", "generate-corpus", "testdata",
		"synthetic-corpus-120-seed42-balanced.jsonl")

	if _, err := os.Stat(corpusPath); err != nil {
		t.Fatalf("corpus not found at %s: %v", corpusPath, err)
	}

	corpusSHA, err := sha256File(corpusPath)
	if err != nil {
		t.Fatalf("sha256 corpus: %v", err)
	}

	adapter, err := agentmeshkafka.NewFileAdapter(corpusPath)
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	defer adapter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exchanges, errc := app.StreamAsTauExchanges(ctx, adapter, nil)
	dispatcher := orchestration.NewDispatcher(nil, orchestration.DefaultThresholds())

	var (
		decisions  []agentmeshkafka.EmpiricalDecision
		unmodeled  []string
		streamErrs []string
	)

	for x := range exchanges {
		dec, decErr := dispatcher.Decide(ctx, x)
		if decErr != nil {
			t.Fatalf("Decide(%s): %v", x.ID, decErr)
		}

		ed := toEmpiricalDecision(ctx, x, dec, orchestration.DefaultThresholds())
		decisions = append(decisions, ed)

		if len(dec.Trace.UnmodeledObservations) > 0 {
			for _, obs := range dec.Trace.UnmodeledObservations {
				unmodeled = append(unmodeled, fmt.Sprintf("%s: %s", x.ID, obs))
			}
		}
	}

	// Drain the error channel (non-fatal parse errors are logged only).
	for err := range errc {
		streamErrs = append(streamErrs, err.Error())
	}
	if len(streamErrs) > 0 {
		t.Logf("stream errors (non-fatal): %v", streamErrs)
	}

	stats := agentmeshkafka.EmpiricalI4Summary(decisions)

	// Assert: total decisions must equal corpus size.
	if stats.Total != 120 {
		t.Errorf("total decisions = %d, want 120", stats.Total)
	}

	// Report sensitivity: below 0.7 is informational, not fatal.
	if stats.Sensitivity >= 0 && stats.Sensitivity < 0.7 {
		t.Logf("NOTICE: sensitivity = %.4f (< 0.70); current D-INV probes may not yet capture I4 pattern — expected finding for M4.7", stats.Sensitivity)
	}

	// Write JSON report.
	report := empiricalI4Report{
		Timestamp:               time.Now().UTC().Format(time.RFC3339),
		ProfileVersion:          "M3-default",
		CorpusPath:              corpusPath,
		CorpusSHA256:            corpusSHA,
		TotalDecisions:          stats.Total,
		Stats:                   stats,
		Sensitivity:             stats.Sensitivity,
		Specificity:             stats.Specificity,
		ObservationsNonModelisees: unmodeled,
	}

	outDir := filepath.Join(pkgDir, "testdata")
	if err := writeReport(t, report, outDir); err != nil {
		t.Fatalf("write report: %v", err)
	}
}

// toEmpiricalDecision computes dimension scores for exchange x (mirroring what
// the dispatcher does) and builds the EmpiricalDecision used for classification.
// The external test package may import tau/dimensions directly: arch_test.go
// (line 93) excludes _test.go files from the bridge-isolation walk, so this
// import is permitted.
func toEmpiricalDecision(ctx context.Context, x tau.Exchange, dec tau.Decision, th orchestration.Thresholds) agentmeshkafka.EmpiricalDecision {
	sensScore, _ := dimensions.ScoreDSens(ctx, x, dimensions.DefaultSensWeights(), nil)
	invScore, _ := dimensions.ScoreDInvariant(ctx, x, dimensions.DefaultInvariantWeights())
	return agentmeshkafka.EmpiricalDecision{
		RegimeStr:             regimeString(dec.Regime),
		Diagnostic:            dec.Diagnostic,
		DSensValue:            sensScore.Value,
		DInvariantValue:       invScore.Value,
		SensCoherence:         th.SensCoherence,
		InvCoherence:          th.InvCoherence,
		UnmodeledObservations: dec.Trace.UnmodeledObservations,
	}
}

func regimeString(r tau.Regime) string {
	switch r {
	case tau.Deterministe:
		return "deterministe"
	case tau.Probabiliste:
		return "probabiliste"
	case tau.Refus:
		return "refus"
	default:
		return "unknown"
	}
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeReport(t *testing.T, report empiricalI4Report, outDir string) error {
	t.Helper()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir testdata: %w", err)
	}
	outPath := filepath.Join(outDir, "empirical-i4-results.json")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	t.Logf("empirical I4 report written to %s", outPath)
	return nil
}
