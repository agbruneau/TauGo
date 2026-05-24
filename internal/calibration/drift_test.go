package calibration_test

import (
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/calibration"
)

func alignedEnv(p calibration.Profile) calibration.Env {
	return calibration.Env{
		CurrentCPUFingerprint:    p.CPUFingerprint,
		CurrentLLMFingerprint:    p.ModelLLMFingerprint,
		CurrentCorpusFingerprint: p.CorpusFingerprint,
	}
}

func futureProfile() calibration.Profile {
	p := calibration.DefaultProfile()
	p.DateRevision = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	p.CPUFingerprint = "linux/amd64"
	p.ModelLLMFingerprint = "stub:v0"
	p.CorpusFingerprint = "sha256:abcd1234"
	return p
}

var now2026 = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

func TestCheckDrift_NoDrift(t *testing.T) {
	t.Parallel()
	p := futureProfile()
	r := calibration.CheckDrift(p, now2026, alignedEnv(p))
	if r.Detected {
		t.Fatalf("expected no drift, got criteria=%v details=%v", r.Criteria, r.Details)
	}
}

func TestCheckDrift_CPUMismatch(t *testing.T) {
	t.Parallel()
	p := futureProfile()
	env := alignedEnv(p)
	env.CurrentCPUFingerprint = "windows/amd64"
	r := calibration.CheckDrift(p, now2026, env)
	if !r.Detected {
		t.Fatal("expected drift detected")
	}
	if !containsCriterion(r.Criteria, calibration.DriftCPU) {
		t.Fatalf("expected DriftCPU in criteria, got %v", r.Criteria)
	}
	if r.Details[calibration.DriftCPU] == "" {
		t.Fatal("expected non-empty detail for DriftCPU")
	}
}

func TestCheckDrift_LLMMismatch(t *testing.T) {
	t.Parallel()
	p := futureProfile()
	env := alignedEnv(p)
	env.CurrentLLMFingerprint = "gpt-4o:2024-11"
	r := calibration.CheckDrift(p, now2026, env)
	if !r.Detected {
		t.Fatal("expected drift detected")
	}
	if !containsCriterion(r.Criteria, calibration.DriftModelLLM) {
		t.Fatalf("expected DriftModelLLM in criteria, got %v", r.Criteria)
	}
}

func TestCheckDrift_CorpusMismatch(t *testing.T) {
	t.Parallel()
	p := futureProfile()
	env := alignedEnv(p)
	env.CurrentCorpusFingerprint = "sha256:deadbeef"
	r := calibration.CheckDrift(p, now2026, env)
	if !r.Detected {
		t.Fatal("expected drift detected")
	}
	if !containsCriterion(r.Criteria, calibration.DriftCorpus) {
		t.Fatalf("expected DriftCorpus in criteria, got %v", r.Criteria)
	}
}

func TestCheckDrift_DateExpired(t *testing.T) {
	t.Parallel()
	p := futureProfile()
	p.DateRevision = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // in the past
	r := calibration.CheckDrift(p, now2026, alignedEnv(p))
	if !r.Detected {
		t.Fatal("expected drift detected for expired profile")
	}
	if !containsCriterion(r.Criteria, calibration.DriftDateExpired) {
		t.Fatalf("expected DriftDateExpired in criteria, got %v", r.Criteria)
	}
	if r.Details[calibration.DriftDateExpired] == "" {
		t.Fatal("expected non-empty detail for DriftDateExpired")
	}
}

func TestCheckDrift_MultipleCriteria(t *testing.T) {
	t.Parallel()
	p := futureProfile()
	p.DateRevision = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC) // expired
	env := calibration.Env{
		CurrentCPUFingerprint:    "darwin/arm64", // mismatches "linux/amd64"
		CurrentLLMFingerprint:    p.ModelLLMFingerprint,
		CurrentCorpusFingerprint: p.CorpusFingerprint,
	}
	r := calibration.CheckDrift(p, now2026, env)
	if !r.Detected {
		t.Fatal("expected drift detected")
	}
	if !containsCriterion(r.Criteria, calibration.DriftCPU) {
		t.Fatal("expected DriftCPU")
	}
	if !containsCriterion(r.Criteria, calibration.DriftDateExpired) {
		t.Fatal("expected DriftDateExpired")
	}
	if len(r.Criteria) < 2 {
		t.Fatalf("expected >= 2 criteria, got %d: %v", len(r.Criteria), r.Criteria)
	}
}

func TestCheckDrift_EmptyFingerprintsSkipped(t *testing.T) {
	t.Parallel()
	// A profile with empty fingerprints should not trigger CPU/corpus drift
	// even when the env has non-empty values (no fingerprint recorded = no check).
	p := calibration.DefaultProfile()
	p.DateRevision = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	// CPUFingerprint and CorpusFingerprint default to "" in DefaultProfile
	env := calibration.Env{
		CurrentCPUFingerprint:    "linux/amd64",
		CurrentLLMFingerprint:    p.ModelLLMFingerprint,
		CurrentCorpusFingerprint: "sha256:anything",
	}
	r := calibration.CheckDrift(p, now2026, env)
	if r.Detected {
		t.Fatalf("empty fingerprints should not trigger drift, got %v", r.Criteria)
	}
}

func TestFingerprintCorpus_Sha256Stable(t *testing.T) {
	t.Parallel()
	h1, err := calibration.FingerprintCorpus("testdata/drift-corpus-a.jsonl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := calibration.FingerprintCorpus("testdata/drift-corpus-a.jsonl")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("fingerprint unstable: %q vs %q", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Fatalf("expected sha256: prefix, got %q", h1)
	}
}

func TestFingerprintCorpus_DifferentFilesDistinct(t *testing.T) {
	t.Parallel()
	ha, err := calibration.FingerprintCorpus("testdata/drift-corpus-a.jsonl")
	if err != nil {
		t.Fatalf("corpus-a: %v", err)
	}
	hb, err := calibration.FingerprintCorpus("testdata/drift-corpus-b.jsonl")
	if err != nil {
		t.Fatalf("corpus-b: %v", err)
	}
	if ha == hb {
		t.Fatal("two distinct corpora produced identical fingerprints")
	}
}

func TestFingerprintCorpus_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := calibration.FingerprintCorpus("testdata/nonexistent.jsonl")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestFingerprintCPU_NonEmpty(t *testing.T) {
	t.Parallel()
	fp := calibration.FingerprintCPU()
	if fp == "" {
		t.Fatal("FingerprintCPU returned empty string")
	}
	if !strings.Contains(fp, "/") {
		t.Fatalf("expected GOOS/GOARCH format, got %q", fp)
	}
}

// containsCriterion reports whether c appears in criteria.
func containsCriterion(criteria []calibration.DriftCriterion, c calibration.DriftCriterion) bool {
	for _, v := range criteria {
		if v == c {
			return true
		}
	}
	return false
}
