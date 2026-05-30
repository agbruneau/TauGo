// Command tau calibrate runs the deterministic grid-search calibration on a
// labeled JSONL corpus and writes the resulting Profile as canonical JSON.
package main

import (
	"encoding/json"
	stderrors "errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/agbruneau/taugo/internal/calibration"
	customerrors "github.com/agbruneau/taugo/internal/errors"
)

// runCalibrate is the entry point for `tau calibrate`.
// Returns an exit code: 0 success, 1 runtime error, 2 bad flags/args.
// Kept as a standalone function so it can be called directly from tests.
func runCalibrate(args []string) int {
	fs := flag.NewFlagSet("calibrate", flag.ContinueOnError)
	corpusPath := fs.String("corpus", "", "path to pre-scored JSONL corpus (required)")
	outputPath := fs.String("output", "", "output profile JSON path (required)")
	dateRevStr := fs.String("date-revision", "", "profile DateRevision YYYY-MM-DD (required)")
	versionMono := fs.String("version-monographie", "v2.4.3", "pinned monograph version tag")
	seed := fs.Int64("seed", 42, "deterministic seed (reserved for V2 stochastic extensions)")
	createdAtStr := fs.String("created-at", "1970-01-01T00:00:00Z", "fixed CreatedAt for byte-identical reproducibility")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `tau calibrate — run adaptive calibration on a labeled JSONL corpus

USAGE:
    tau calibrate --corpus PATH --output PATH --date-revision YYYY-MM-DD [flags]

FLAGS:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if *corpusPath == "" || *outputPath == "" || *dateRevStr == "" {
		fmt.Fprintln(os.Stderr, "tau calibrate: --corpus, --output and --date-revision are required")
		fs.Usage()
		return 2
	}

	dateRev, err := parseDateRev(*dateRevStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tau calibrate: invalid --date-revision:", err)
		return 2
	}

	createdAt, err := time.Parse(time.RFC3339, *createdAtStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tau calibrate: invalid --created-at:", err)
		return 2
	}

	corpusFingerprint, err := calibration.FingerprintCorpus(*corpusPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tau calibrate:", err)
		return 1
	}

	entries, err := loadCorpus(*corpusPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tau calibrate: reading corpus:", err)
		return corpusErrExitCode(err)
	}

	profile := calibration.DefaultProfile()
	profile.DateRevision = dateRev
	profile.VersionMonographie = *versionMono
	profile.CreatedAt = createdAt
	profile.CorpusFingerprint = corpusFingerprint
	profile.CPUFingerprint = calibration.FingerprintCPU()
	profile.ModelLLMFingerprint = "stub:v0"

	out := calibration.Calibrate(entries, *seed, profile)

	b, err := calibration.MarshalCanonical(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tau calibrate: marshal:", err)
		return 1
	}

	if err := os.WriteFile(*outputPath, b, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "tau calibrate: write output:", err)
		return 1
	}
	return 0
}

// parseDateRev accepts RFC3339 or YYYY-MM-DD formats.
func parseDateRev(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// loadCorpus reads a JSONL file, migrates legacy ExpectedRegime entries and
// validates every entry (scores in [0,1], LabeledRegime among the four accepted
// values). It delegates to calibration.LoadCorpus so an invalid or legacy corpus
// is rejected at load time rather than silently producing a degenerate profile
// (AUDIT C1-01).
func loadCorpus(path string) ([]calibration.CorpusEntry, error) {
	return calibration.LoadCorpus(path)
}

// corpusErrExitCode classifies a corpus-load failure into a CLI exit code:
//   - exit 1 for I/O (missing file) and JSON syntax errors — operational faults;
//   - exit 2 for content validation errors (invalid regime, score out of range)
//     surfaced as a *CalibrationError without an underlying I/O or JSON cause —
//     i.e. bad input the caller must fix.
func corpusErrExitCode(err error) int {
	var pathErr *os.PathError
	if stderrors.As(err, &pathErr) {
		return 1
	}
	var synErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if stderrors.As(err, &synErr) || stderrors.As(err, &typeErr) ||
		stderrors.Is(err, io.ErrUnexpectedEOF) {
		return 1
	}
	var calErr *customerrors.CalibrationError
	if stderrors.As(err, &calErr) {
		return 2
	}
	return 1
}
