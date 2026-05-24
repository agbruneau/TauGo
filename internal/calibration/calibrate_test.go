package calibration_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/agbruneau/taugo/internal/calibration"
)

// loadMiniCorpus reads testdata/mini-corpus.jsonl and returns all entries.
func loadMiniCorpus(t *testing.T) []calibration.CorpusEntry {
	t.Helper()
	f, err := os.Open("testdata/mini-corpus.jsonl")
	if err != nil {
		t.Fatalf("open mini-corpus: %v", err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	out := make([]calibration.CorpusEntry, 0, 32)
	for dec.More() {
		var e calibration.CorpusEntry
		if err := dec.Decode(&e); err != nil {
			t.Fatalf("decode corpus entry: %v", err)
		}
		out = append(out, e)
	}
	return out
}

// countAgreementHelper replicates the agreement count for assertions.
func countAgreementHelper(corpus []calibration.CorpusEntry, t calibration.Thresholds) int {
	n := 0
	for _, e := range corpus {
		if simulateHelper(e, t) == e.ExpectedRegime {
			n++
		}
	}
	return n
}

func simulateHelper(e calibration.CorpusEntry, t calibration.Thresholds) string {
	if e.AuthorityScore >= t.AuthBlock && !e.HasAttestation {
		return "refus_authority"
	}
	if e.SensScore < t.SensCoherence && e.InvariantScore >= t.InvCoherence {
		return "refus_i4"
	}
	if e.SensScore >= t.Probabiliste {
		return "deterministe"
	}
	return "probabiliste"
}

func TestCalibrate_GridSearchReturnsBestThresholds(t *testing.T) {
	t.Parallel()
	corpus := loadMiniCorpus(t)
	p := calibration.Calibrate(corpus, 1, calibration.DefaultProfile())
	th := p.Thresholds
	if th.Deterministe <= 0 {
		t.Errorf("Deterministe must be > 0, got %f", th.Deterministe)
	}
	if th.Probabiliste <= th.Deterministe {
		t.Errorf("Probabiliste (%f) must be > Deterministe (%f)", th.Probabiliste, th.Deterministe)
	}
	if th.AuthBlock <= 0 {
		t.Errorf("AuthBlock must be > 0, got %f", th.AuthBlock)
	}
}

func TestCalibrate_DeterministicSameInputSameOutput(t *testing.T) {
	t.Parallel()
	corpus := loadMiniCorpus(t)
	p1 := calibration.Calibrate(corpus, 1, calibration.DefaultProfile())
	p2 := calibration.Calibrate(corpus, 1, calibration.DefaultProfile())
	if p1.Thresholds != p2.Thresholds {
		t.Fatalf("Calibrate not deterministic:\n  run1: %+v\n  run2: %+v", p1.Thresholds, p2.Thresholds)
	}
}

func TestCalibrate_ImprovesOrMaintainsAgreement(t *testing.T) {
	t.Parallel()
	corpus := loadMiniCorpus(t)
	baseline := countAgreementHelper(corpus, calibration.DefaultProfile().Thresholds)
	out := calibration.Calibrate(corpus, 1, calibration.DefaultProfile())
	after := countAgreementHelper(corpus, out.Thresholds)
	if after < baseline {
		t.Fatalf("Calibrate worsened agreement: %d (baseline) → %d (after)", baseline, after)
	}
}

func TestCalibrate_PreservesWeightsV1(t *testing.T) {
	t.Parallel()
	corpus := loadMiniCorpus(t)
	in := calibration.DefaultProfile()
	out := calibration.Calibrate(corpus, 1, in)
	if !reflect.DeepEqual(out.Weights, in.Weights) {
		t.Fatal("Calibrate V1 must not mutate Weights")
	}
}

func TestMarshalCanonical_KeysSorted(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	// Inject probe keys deliberately out of alphabetical order.
	p.Weights.SensProbes = map[string]float64{"z_probe": 0.5, "a_probe": 0.5}
	b, err := calibration.MarshalCanonical(p)
	if err != nil {
		t.Fatalf("MarshalCanonical: %v", err)
	}
	aIdx := bytes.Index(b, []byte(`"a_probe"`))
	zIdx := bytes.Index(b, []byte(`"z_probe"`))
	if aIdx <= 0 || zIdx <= 0 || aIdx >= zIdx {
		t.Fatalf(`expected "a_probe" before "z_probe" in canonical output:\n%s`, b)
	}
}

func TestMarshalCanonical_RoundTripIdentity(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	b1, err := calibration.MarshalCanonical(p)
	if err != nil {
		t.Fatalf("first marshal: %v", err)
	}
	p2, err := calibration.UnmarshalCanonical(b1)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	b2, err := calibration.MarshalCanonical(p2)
	if err != nil {
		t.Fatalf("second marshal: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("round-trip not byte-identical:\n  first:  %s\n  second: %s", b1, b2)
	}
}

func TestMarshalCanonical_ByteIdentical(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	b1, err := calibration.MarshalCanonical(p)
	if err != nil {
		t.Fatalf("first marshal: %v", err)
	}
	b2, err := calibration.MarshalCanonical(p)
	if err != nil {
		t.Fatalf("second marshal: %v", err)
	}
	h1 := sha256.Sum256(b1)
	h2 := sha256.Sum256(b2)
	if h1 != h2 {
		t.Fatal("MarshalCanonical not idempotent: sha256 differs")
	}
}

func TestMarshalCanonical_TrailingNewline(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	b, err := calibration.MarshalCanonical(p)
	if err != nil {
		t.Fatalf("MarshalCanonical: %v", err)
	}
	if len(b) == 0 || b[len(b)-1] != '\n' {
		t.Fatalf("canonical encoding must end with '\\n', got last byte 0x%02x", b[len(b)-1])
	}
}

func TestMarshalCanonical_ValidJSON(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	b, err := calibration.MarshalCanonical(p)
	if err != nil {
		t.Fatalf("MarshalCanonical: %v", err)
	}
	if !json.Valid(b) {
		t.Fatalf("MarshalCanonical output is not valid JSON:\n%s", b)
	}
}

func TestUnmarshalCanonical_InvalidInput(t *testing.T) {
	t.Parallel()
	_, err := calibration.UnmarshalCanonical([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON input")
	}
	if !strings.Contains(err.Error(), "UnmarshalCanonical") {
		t.Fatalf("error should mention UnmarshalCanonical, got: %v", err)
	}
}
