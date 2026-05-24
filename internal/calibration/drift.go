package calibration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

// DriftCriterion identifies one of the five PRD §11.4 invalidation criteria.
type DriftCriterion int

const (
	// DriftCPU fires when the CPU fingerprint in the profile differs from the
	// current runtime environment.
	DriftCPU DriftCriterion = iota

	// DriftModelLLM fires when the LLM model fingerprint has changed since
	// the profile was calibrated.
	DriftModelLLM

	// DriftCorpus fires when the corpus fingerprint no longer matches the
	// profile's recorded value.
	DriftCorpus

	// DriftDateExpired fires when the current time is strictly after the
	// profile's DateRevision (PRD §7.1 C3).
	DriftDateExpired

	// DriftScoreDistribution is a V1 placeholder. The sliding-window score
	// distribution check is deferred to V2 (empirical layer, PRD §11.4 last
	// criterion). This criterion is never raised in V1.
	DriftScoreDistribution
)

// DriftReport is the result of CheckDrift. Detected is true when at least one
// criterion is triggered. Criteria lists the triggered criteria in declaration
// order. Details provides a human-readable explanation for each triggered
// criterion.
type DriftReport struct {
	Detected bool
	Criteria []DriftCriterion
	Details  map[DriftCriterion]string
}

// Env carries the current runtime fingerprints supplied by the caller.
// calibration does not import bridge/llm; the caller is responsible for
// collecting each value before calling CheckDrift.
type Env struct {
	// CurrentCPUFingerprint is the fingerprint of the current CPU environment.
	// Use FingerprintCPU() to obtain it at startup.
	CurrentCPUFingerprint string

	// CurrentLLMFingerprint identifies the active LLM model, e.g. "stub:v0"
	// or a real model ID. Provided by the caller via llm.Client.Fingerprint().
	CurrentLLMFingerprint string

	// CurrentCorpusFingerprint is the sha256 fingerprint of the calibration
	// corpus file in use. Empty string disables the corpus check.
	CurrentCorpusFingerprint string
}

// CheckDrift evaluates the five PRD §11.4 invalidation criteria against the
// current environment. It returns a DriftReport describing which criteria, if
// any, were triggered.
//
// The score-distribution criterion (DriftScoreDistribution) always returns
// false in V1. V2 will introduce the sliding-window detector once the
// empirical layer is available.
func CheckDrift(current Profile, now time.Time, env Env) DriftReport {
	report := DriftReport{Details: make(map[DriftCriterion]string)}

	if current.CPUFingerprint != "" && env.CurrentCPUFingerprint != current.CPUFingerprint {
		report.Criteria = append(report.Criteria, DriftCPU)
		report.Details[DriftCPU] = fmt.Sprintf(
			"cpu fingerprint changed: profile=%q env=%q",
			current.CPUFingerprint, env.CurrentCPUFingerprint,
		)
	}

	if current.ModelLLMFingerprint != "" && env.CurrentLLMFingerprint != current.ModelLLMFingerprint {
		report.Criteria = append(report.Criteria, DriftModelLLM)
		report.Details[DriftModelLLM] = fmt.Sprintf(
			"llm fingerprint changed: profile=%q env=%q",
			current.ModelLLMFingerprint, env.CurrentLLMFingerprint,
		)
	}

	if current.CorpusFingerprint != "" && env.CurrentCorpusFingerprint != current.CorpusFingerprint {
		report.Criteria = append(report.Criteria, DriftCorpus)
		report.Details[DriftCorpus] = fmt.Sprintf(
			"corpus fingerprint changed: profile=%q env=%q",
			current.CorpusFingerprint, env.CurrentCorpusFingerprint,
		)
	}

	// Deliberate asymmetry with the dispatcher (orchestration.Dispatcher step 3):
	// !now.Before(dateRevision) fires on the day of expiry itself (today == dateRev
	// triggers drift), whereas the dispatcher uses now.After(dateRevision) and
	// only blocks the day after. This gives one extra day of early warning before
	// the hard refusal kicks in (PRD §11.4 last sentence). See also:
	// internal/orchestration/dispatcher.go step 3 comment.
	if !now.Before(current.DateRevision) {
		report.Criteria = append(report.Criteria, DriftDateExpired)
		report.Details[DriftDateExpired] = fmt.Sprintf(
			"profile expired: date_revision=%s now=%s",
			current.DateRevision.Format(time.DateOnly), now.Format(time.DateOnly),
		)
	}

	// DriftScoreDistribution: V1 placeholder — always false.
	// V2 will introduce a sliding-window statistical check once the empirical
	// layer (PRD §11.4) is available. See docs/algorithms/drift.md.

	report.Detected = len(report.Criteria) > 0
	return report
}

// FingerprintCPU returns a coarse runtime identifier for the current CPU
// environment. In V1 this is "GOOS/GOARCH" — sufficient for cache
// invalidation when switching between architectures. A richer cpuid-based
// fingerprint is deferred to M6 if needed.
func FingerprintCPU() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// FingerprintCorpus computes a SHA-256 digest of the file at jsonlPath and
// returns it as "sha256:<hex>". The same file always produces the same string.
func FingerprintCorpus(jsonlPath string) (string, error) {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return "", fmt.Errorf("FingerprintCorpus: open %q: %w", jsonlPath, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("FingerprintCorpus: hash %q: %w", jsonlPath, err)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
