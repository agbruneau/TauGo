package calibration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	taugoerrors "github.com/agbruneau/taugo/internal/errors"
)

// CorpusEntry is one labeled exchange used for grid-search calibration.
// Each line of a JSONL golden corpus deserializes to a CorpusEntry.
// Pre-computed dimension scores are produced by the corpus generator so
// that Calibrate never calls an LLM or imports bridge/tau packages.
type CorpusEntry struct {
	ID             string  `json:"id"`
	SensScore      float64 `json:"sens_score"`
	AuthorityScore float64 `json:"authority_score"`
	InvariantScore float64 `json:"invariant_score"`
	HumanInLoop    bool    `json:"human_in_loop"`
	HasAttestation bool    `json:"has_attestation"`
	// LabeledRegime is one of: "deterministe" | "probabiliste" |
	// "refus_authority" | "refus_i4".
	LabeledRegime string `json:"labeled_regime,omitempty"`
	// Deprecated: utiliser LabeledRegime. Conservé pour rétro-compat
	// des corpus JSON v0.1.0. Retrait V0.2.
	ExpectedRegime string `json:"expected_regime,omitempty"`
}

// validRegimes is the set of accepted LabeledRegime values.
//
//nolint:gochecknoglobals // immutable lookup set for corpus validation
var validRegimes = map[string]struct{}{
	"deterministe":    {},
	"probabiliste":    {},
	"refus_authority": {},
	"refus_i4":        {},
}

// migrate applique la rétro-compat : si LabeledRegime est vide mais ExpectedRegime non,
// copie ExpectedRegime dans LabeledRegime.
func (e *CorpusEntry) migrate() {
	if e.LabeledRegime == "" && e.ExpectedRegime != "" {
		e.LabeledRegime = e.ExpectedRegime
	}
}

// Validate vérifie la cohérence de l'entrée de corpus.
// Retourne *taugoerrors.CalibrationError si l'entrée est invalide.
func (e *CorpusEntry) Validate() error {
	if e.ID == "" {
		return &taugoerrors.CalibrationError{Cause: fmt.Errorf("CorpusEntry.ID vide")}
	}
	if e.SensScore < 0 || e.SensScore > 1 {
		return &taugoerrors.CalibrationError{Cause: fmt.Errorf("SensScore hors [0,1]: %f", e.SensScore)}
	}
	if e.AuthorityScore < 0 || e.AuthorityScore > 1 {
		return &taugoerrors.CalibrationError{Cause: fmt.Errorf("AuthorityScore hors [0,1]: %f", e.AuthorityScore)}
	}
	if e.InvariantScore < 0 || e.InvariantScore > 1 {
		return &taugoerrors.CalibrationError{Cause: fmt.Errorf("InvariantScore hors [0,1]: %f", e.InvariantScore)}
	}
	if _, ok := validRegimes[e.LabeledRegime]; !ok {
		return &taugoerrors.CalibrationError{
			Cause: fmt.Errorf("LabeledRegime invalide : %q", e.LabeledRegime),
		}
	}
	return nil
}

// LoadCorpus lit un fichier JSONL de corpus, applique la migration rétro-compat
// (ExpectedRegime → LabeledRegime) et valide chaque entrée.
// Retourne *CalibrationError si un fichier est illisible ou une entrée invalide.
func LoadCorpus(path string) ([]CorpusEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, &taugoerrors.CalibrationError{Cause: fmt.Errorf("open corpus %q: %w", path, err)}
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	out := make([]CorpusEntry, 0, 64)
	for dec.More() {
		var e CorpusEntry
		if err := dec.Decode(&e); err != nil {
			return nil, &taugoerrors.CalibrationError{Cause: fmt.Errorf("decode corpus entry: %w", err)}
		}
		e.migrate()
		if err := e.Validate(); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// Calibrate runs the V1 grid-search algorithm against the labeled corpus.
// It returns a Profile whose Thresholds maximize agreement with the corpus
// labels. Weights are kept as-is from in.Weights (V1 scope — weight
// calibration is deferred; see docs/algorithms/calibration.md).
//
// Determinism: same (corpus, seed, in.Weights) always yields the same
// Profile.Thresholds. Ties are broken lexicographically by
// (Deterministe, HysteresisGap, AuthBlock, SensCoherence) ascending so
// that the most-conservative combination wins.
//
// The seed parameter is reserved for future stochastic extensions (e.g.
// random restarts in V2). V1 ignores it; the grid is exhaustive and
// order-deterministic.
func Calibrate(corpus []CorpusEntry, _ int64, in Profile) Profile {
	best := -1
	bestT := in.Thresholds

	// Grid ranges (milli-unit integers to avoid float64 accumulation drift).
	// Deterministe: [0.10..0.90], HysteresisGap: [0.05..0.20],
	// AuthBlock: [0.70..0.95], SensCoherence: [0.30..0.70].
	// V1 simplification: InvCoherence = SensCoherence (see docs/algorithms/calibration.md).
	for dM := 100; dM <= 900; dM += 50 {
		for gM := 50; gM <= 200; gM += 50 {
			pM := dM + gM
			if pM > 950 {
				continue
			}
			for aM := 700; aM <= 950; aM += 50 {
				for sM := 300; sM <= 700; sM += 50 {
					t := Thresholds{
						Deterministe:  fromMillis(int64(dM)),
						Probabiliste:  fromMillis(int64(pM)),
						HysteresisGap: fromMillis(int64(gM)),
						AuthBlock:     fromMillis(int64(aM)),
						SensCoherence: fromMillis(int64(sM)),
						InvCoherence:  fromMillis(int64(sM)),
					}
					score := countAgreement(corpus, t)
					if score > best {
						best = score
						bestT = t
					}
					// Ties: lexicographic (d, g, a, s) ascending — smaller values
					// are already encountered first, so first-wins = most conservative.
				}
			}
		}
	}

	out := in
	out.Thresholds = bestT
	// Route Weights through CalibrateWeights so a V2 hook can intercept
	// without changing Calibrate's signature (PRD §11.1, M5.2).
	out.Weights = CalibrateWeights(corpus, 0, in.Weights)
	return out
}

// countAgreement returns the number of corpus entries where simulate
// predicts the same regime as the entry's LabeledRegime.
func countAgreement(corpus []CorpusEntry, t Thresholds) int {
	n := 0
	for i := range corpus {
		if simulate(corpus[i], t) == corpus[i].LabeledRegime {
			n++
		}
	}
	return n
}

// simulate applies the threshold rules to a pre-scored CorpusEntry and
// returns the predicted regime string. This is a lightweight projection
// of the dispatcher logic (M3); it intentionally omits LLM calls because
// all scores are pre-computed in the corpus.
//
// Naming note: "Deterministe" and "Probabiliste" reflect confidence in the
// decision, not the raw score magnitude. A high SensScore means the kernel is
// confident enough to commit deterministically; a moderate score keeps the
// regime probabilistic. PRD §4 (frontier) and §10 (algorithm) define the
// semantics.
//
// Rule order mirrors PRD §10 dispatcher steps 2–7:
//  1. refus_authority — AuthorityScore >= AuthBlock without attestation
//  2. refus_i4        — incoherence: low SensScore and high InvariantScore
//  3. deterministe    — SensScore >= Probabiliste threshold (high confidence)
//  4. probabiliste    — SensScore >= Deterministe threshold (moderate confidence)
//  5. default         — probabiliste
func simulate(e CorpusEntry, t Thresholds) string {
	if e.AuthorityScore >= t.AuthBlock && !e.HasAttestation {
		return "refus_authority"
	}
	if e.SensScore < t.SensCoherence && e.InvariantScore >= t.InvCoherence {
		return "refus_i4"
	}
	if e.SensScore >= t.Probabiliste {
		return "deterministe"
	}
	if e.SensScore >= t.Deterministe {
		return "probabiliste"
	}
	return "probabiliste"
}

// MarshalCanonical serializes p as byte-identical JSON: 2-space indented,
// trailing newline, all map keys explicitly sorted at every level. Safe
// against any future change to encoding/json's map-encoding order.
//
// PRD §17 critère #10: two MarshalCanonical calls on equal Profile values
// must produce byte-for-byte identical output.
func MarshalCanonical(p Profile) ([]byte, error) {
	// First pass: standard marshal to obtain a []byte we can decode.
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("calibration: MarshalCanonical first pass: %w", err)
	}

	// Second pass: decode into generic map, re-encode with sorted keys.
	var generic any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&generic); err != nil {
		return nil, fmt.Errorf("calibration: MarshalCanonical decode: %w", err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(sortedAny(generic)); err != nil {
		return nil, fmt.Errorf("calibration: MarshalCanonical encode: %w", err)
	}
	// json.Encoder.Encode appends '\n'; buf already ends with '\n'.
	return buf.Bytes(), nil
}

// UnmarshalCanonical deserializes data produced by MarshalCanonical back
// into a Profile. It is the inverse of MarshalCanonical.
func UnmarshalCanonical(data []byte) (Profile, error) {
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, fmt.Errorf("calibration: UnmarshalCanonical: %w", err)
	}
	return p, nil
}

// sortedAny recursively sorts map keys so that the JSON encoding is
// deterministic regardless of Go runtime map iteration order.
func sortedAny(v any) any {
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(sortedMap, 0, len(val))
		for _, k := range keys {
			out = append(out, kv{k, sortedAny(val[k])})
		}
		return out
	case []any:
		for i, elem := range val {
			val[i] = sortedAny(elem)
		}
		return val
	default:
		return v
	}
}

// sortedMap is an ordered list of key-value pairs that encodes as a JSON
// object with keys in the order they were inserted.
type sortedMap []kv

type kv struct {
	Key   string
	Value any
}

// MarshalJSON encodes sortedMap as a JSON object preserving key order.
func (m sortedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, pair := range m {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, err := json.Marshal(pair.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')
		valBytes, err := json.Marshal(pair.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
