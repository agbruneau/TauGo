package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
)

func TestGenerateCorpus_ReproducibleBytewise(t *testing.T) {
	t.Parallel()
	var a, b bytes.Buffer
	if err := NewGenerator(42).Generate(&a, 120, ProfileBalanced); err != nil {
		t.Fatal(err)
	}
	if err := NewGenerator(42).Generate(&b, 120, ProfileBalanced); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Fatal("same seed produced different output: determinism broken")
	}
}

func TestGenerateCorpus_RespectsCount(t *testing.T) {
	t.Parallel()
	for _, n := range []int{1, 10, 120} {
		var buf bytes.Buffer
		if err := NewGenerator(42).Generate(&buf, n, ProfileBalanced); err != nil {
			t.Fatalf("count=%d: %v", n, err)
		}
		got := strings.Count(buf.String(), "\n")
		if got != n {
			t.Errorf("count=%d: got %d lines, want %d", n, got, n)
		}
	}
}

func TestGenerateCorpus_DistributionBalanced_RoughlyEven(t *testing.T) {
	t.Parallel()
	const n = 120
	var buf bytes.Buffer
	if err := NewGenerator(42).Generate(&buf, n, ProfileBalanced); err != nil {
		t.Fatal(err)
	}

	// Count expected regimes by ID prefix.
	counts := map[string]int{
		"rf": 0, // refus-frontiere
		"r3": 0, // refus-i3
		"r4": 0, // refus-i4
		"d":  0, // deterministe
		"p":  0, // probabiliste
		"h":  0, // hysteresis
	}

	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		var x agentmeshkafka.AgentMeshExchange
		if err := json.Unmarshal(scanner.Bytes(), &x); err != nil {
			t.Fatalf("invalid JSON line: %v", err)
		}
		// IDs are "synth-<prefix>-NNNNNN"
		parts := strings.SplitN(x.ID, "-", 3)
		if len(parts) >= 2 {
			counts[parts[1]]++
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	// balanced weights: rf=15%, r3=15%, r4=10%, d=25%, p=25%, h fills remainder.
	// Allow ±10% of n tolerance (i.e., ±12 entries for n=120).
	const tolerance = 0.10
	checkBetween := func(label string, got, targetPct int) {
		t.Helper()
		lo := int(float64(n)*float64(targetPct)/100.0*(1-tolerance) + 0.5)
		hi := int(float64(n)*float64(targetPct)/100.0*(1+tolerance) + 0.5)
		if lo < 0 {
			lo = 0
		}
		if got < lo || got > hi {
			t.Errorf("branch %q: got %d, want [%d, %d] (target %d%%)", label, got, lo, hi, targetPct)
		}
	}

	checkBetween("rf", counts["rf"], 15)
	checkBetween("r3", counts["r3"], 15)
	checkBetween("r4", counts["r4"], 10)
	checkBetween("d", counts["d"], 25)
	checkBetween("p", counts["p"], 25)
}

func TestGenerateCorpus_DifferentSeeds_DifferentOutput(t *testing.T) {
	t.Parallel()
	var a, b bytes.Buffer
	if err := NewGenerator(42).Generate(&a, 10, ProfileBalanced); err != nil {
		t.Fatal(err)
	}
	if err := NewGenerator(99).Generate(&b, 10, ProfileBalanced); err != nil {
		t.Fatal(err)
	}
	if a.String() == b.String() {
		t.Fatal("seed=42 and seed=99 produced identical output")
	}
}

func TestGenerateCorpus_AllProfilesValid(t *testing.T) {
	t.Parallel()
	profiles := []DistributionProfile{ProfileBalanced, ProfileI4Heavy, ProfileRefusHeavy}
	for _, p := range profiles {
		var buf bytes.Buffer
		if err := NewGenerator(42).Generate(&buf, 20, p); err != nil {
			t.Errorf("profile %q: %v", p, err)
		}
		got := strings.Count(buf.String(), "\n")
		if got != 20 {
			t.Errorf("profile %q: got %d lines, want 20", p, got)
		}
	}
}

func TestGenerateCorpus_FrozenHash_Seed42_120_Balanced(t *testing.T) {
	t.Parallel()
	// Guards the checked-in corpus testdata/synthetic-corpus-120-seed42-balanced.jsonl.
	// Computed on first green run; fill the constant below.
	var buf bytes.Buffer
	if err := NewGenerator(42).Generate(&buf, 120, ProfileBalanced); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(buf.Bytes())
	got := hex.EncodeToString(h[:])

	const want = "a91d60cd9815d8183df57bfcf16bbe77d36360c4ed36e33fced9f12f70fd68ee"
	if got != want {
		t.Fatalf("frozen hash drift: got=%s want=%s", got, want)
	}
}

// TestGenerateCorpus_WithAnnotation_ProducesExpectedRegime verifies that
// GenerateAnnotated enriches every line with a valid expected_regime field.
func TestGenerateCorpus_WithAnnotation_ProducesExpectedRegime(t *testing.T) {
	t.Parallel()
	d := app.NewDispatcher()
	var buf bytes.Buffer
	if err := NewGenerator(42).GenerateAnnotated(context.Background(), &buf, 30, ProfileBalanced, d); err != nil {
		t.Fatal(err)
	}
	valid := map[string]bool{"Deterministe": true, "Probabiliste": true, "Refus": true}
	scanner := bufio.NewScanner(&buf)
	lineN := 0
	for scanner.Scan() {
		lineN++
		var entry AnnotatedEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("line %d: invalid JSON: %v", lineN, err)
		}
		if entry.ExpectedRegime == "" {
			t.Errorf("line %d: missing expected_regime", lineN)
		}
		if !valid[entry.ExpectedRegime] {
			t.Errorf("line %d: unexpected value %q", lineN, entry.ExpectedRegime)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if lineN != 30 {
		t.Fatalf("got %d lines, want 30", lineN)
	}
}

// TestGenerateCorpus_AnnotationDoesNotBreakBaselineHash confirms that the
// non-annotated Generate path is unaffected by the --annotate flag: same seed
// must produce the same sha256 as the M4 frozen value.
func TestGenerateCorpus_AnnotationDoesNotBreakBaselineHash(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := NewGenerator(42).Generate(&buf, 120, ProfileBalanced); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(buf.Bytes())
	got := hex.EncodeToString(h[:])
	const want = "a91d60cd9815d8183df57bfcf16bbe77d36360c4ed36e33fced9f12f70fd68ee"
	if got != want {
		t.Fatalf("baseline hash changed after annotation feature was added: got=%s want=%s", got, want)
	}
}

// TestGoldenCorpus_FrozenHash_Seed42_200_Balanced pins the sha256 of the
// checked-in golden calibration corpus (tests/calibration/golden-corpus.jsonl).
// Two successive generations must be identical; both must match the pinned constant.
func TestGoldenCorpus_FrozenHash_Seed42_200_Balanced(t *testing.T) {
	t.Parallel()
	d := app.NewDispatcher()

	hashGen := func() string {
		var buf bytes.Buffer
		if err := NewGenerator(42).GenerateAnnotated(context.Background(), &buf, 200, ProfileBalanced, d); err != nil {
			t.Fatalf("GenerateAnnotated: %v", err)
		}
		h := sha256.Sum256(buf.Bytes())
		return hex.EncodeToString(h[:])
	}

	run1 := hashGen()
	run2 := hashGen()
	if run1 != run2 {
		t.Fatalf("annotated generation is not reproducible: run1=%s run2=%s", run1, run2)
	}

	// Pinned after first green run.
	const want = "beb6c8d87911ef58d189c6f1c3d4adf9b71777e6dce328ed781e394614ac3a1b"
	if run1 != want {
		t.Fatalf("golden corpus hash drift: got=%s want=%s", run1, want)
	}

	// Also verify the checked-in file matches, if present.
	if data, err := os.ReadFile("../../tests/calibration/golden-corpus.jsonl"); err == nil {
		fh := sha256.Sum256(data)
		fgot := hex.EncodeToString(fh[:])
		if fgot != want {
			t.Errorf("checked-in golden-corpus.jsonl hash mismatch: got=%s want=%s", fgot, want)
		}
	}
}
