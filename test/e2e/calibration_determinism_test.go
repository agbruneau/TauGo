//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/calibration"
	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

// goldenCorpusCanonicalHash is the expected sha256 of the profile produced by
// `tau calibrate` with the golden corpus (200 lines), seed 42,
// date_revision 2026-11-23, version_monographie v2.4.3, created_at frozen to
// 1970-01-01T00:00:00Z. Changing this value signals a regression in the
// calibration algorithm or the canonical marshaller (PRD §17 criterion #10).
const goldenCorpusCanonicalHash = "d753245b87933f97c6324f54df1572fab7cc68c52bc49baa1b891ab97abff6c7"

// goldenCorpusPath resolves the path to tests/calibration/golden-corpus.jsonl
// relative to the module root, regardless of where the test binary runs.
func goldenCorpusPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile is .../test/e2e/calibration_determinism_test.go
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	p := filepath.Join(root, "tests", "calibration", "golden-corpus.jsonl")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("golden corpus not found at %s: %v", p, err)
	}
	return p
}

// buildTauE2E compiles the cmd/tau binary into t.TempDir() and returns its path.
// It is kept separate from the integration-tag buildCLI helper in
// agentmeshkafka_test.go to avoid build-tag coupling.
func buildTauE2E(t *testing.T) string {
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

// sha256HexFile computes the lower-hex sha256 of the file at path.
func sha256HexFile(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// runCalibrateCanonical runs `tau calibrate` with the fixed canonical flags
// against the golden corpus and writes the profile to output.
func runCalibrateCanonical(t *testing.T, bin, corpus, output string) {
	t.Helper()
	cmd := exec.Command(bin,
		"calibrate",
		"--corpus", corpus,
		"--output", output,
		"--date-revision", "2026-11-23",
		"--seed", "42",
		"--created-at", "1970-01-01T00:00:00Z",
		"--version-monographie", "v2.4.3",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tau calibrate failed: %v\n%s", err, out)
	}
}

// TestCalibrationDeterministic verifies PRD §17 criterion #10:
// two consecutive `tau calibrate` runs on the golden corpus with the same
// fixed flags produce byte-identical Profile JSON, confirmed via sha256.
func TestCalibrationDeterministic(t *testing.T) {
	t.Parallel()
	bin := buildTauE2E(t)
	corpus := goldenCorpusPath(t)
	dir := t.TempDir()
	p1 := filepath.Join(dir, "p1.json")
	p2 := filepath.Join(dir, "p2.json")

	runCalibrateCanonical(t, bin, corpus, p1)
	runCalibrateCanonical(t, bin, corpus, p2)

	h1 := sha256HexFile(t, p1)
	h2 := sha256HexFile(t, p2)
	if h1 != h2 {
		t.Fatalf("PRD §17 #10 violated: sha256 differ:\n  run1: %s\n  run2: %s", h1, h2)
	}
}

// TestCalibrate_GoldenCorpus_FixedHash pins the canonical sha256 of the profile
// produced from the golden corpus with the canonical flags. A hash change signals
// a regression in the calibration algorithm or the canonical marshaller.
func TestCalibrate_GoldenCorpus_FixedHash(t *testing.T) {
	t.Parallel()
	bin := buildTauE2E(t)
	corpus := goldenCorpusPath(t)
	out := filepath.Join(t.TempDir(), "canonical.json")

	runCalibrateCanonical(t, bin, corpus, out)

	got := sha256HexFile(t, out)
	if got != goldenCorpusCanonicalHash {
		t.Fatalf("canonical sha256 changed — calibration regression detected:\n  got:  %s\n  want: %s",
			got, goldenCorpusCanonicalHash)
	}
}

// TestExpiredProfileRefuses verifies PRD §15.1 end-to-end: a Profile whose
// DateRevision is in the past causes the dispatcher to return Refus with the
// canonical "profil périmé" diagnostic (anti-pattern #3 guarded).
//
// Implementation note: `tau decide` does not yet expose a --profile flag.
// This test exercises the same production code path via in-process dispatch
// with a frozen clock, matching what the subprocess would invoke once the
// flag lands (the guard is wired at dispatcher step 3, M5.5).
func TestExpiredProfileRefuses(t *testing.T) {
	t.Parallel()

	// Build an expired profile — DateRevision set to a fixed past date.
	p := calibration.DefaultProfile()
	p.DateRevision = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Freeze the clock to a date clearly after the expiry.
	frozenNow := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)

	d := orchestration.NewDispatcherWithProfile(
		llm.Stub{},
		orchestration.DefaultThresholds(),
		&p,
	).WithClock(func() time.Time { return frozenNow })

	// Construct an Exchange inside the frontier (all four M2 conditions hold):
	// DiscoveryMode != Static => UniversOuvert=true, CompositionVariable=true;
	// HumanInLoop=false => PairProbabiliste=true; DelegationDepth > 0 => CoutNonBorne=true.
	// An attestation is provided so the D-AUTORITÉ guard (step 2) does not trigger first.
	x := tau.Exchange{
		ID:                "e2e-expired-01",
		IntentDescription: "dispatch notification",
		DiscoveredAt:      frozenNow,
		Initiator: tau.Principal{
			ID:              "agent-a",
			Organization:    "org-a",
			HumanInLoop:     false,
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "svc-b",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1",
		},
		AttestationInstitutionnelle: &tau.Attestation{
			Emetteur:   "desjardins-iam",
			Reference:  "ref-e2e",
			Marqueur:   "Confirmé",
			AssertedAt: frozenNow,
		},
	}

	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Refus {
		t.Fatalf("expected Refus, got regime=%v diagnostic=%q", dec.Regime, dec.Diagnostic)
	}
	if !strings.Contains(dec.Diagnostic, "profil périmé") {
		t.Fatalf("expected 'profil périmé' in diagnostic, got %q", dec.Diagnostic)
	}
}
