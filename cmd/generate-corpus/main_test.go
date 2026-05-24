package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

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
