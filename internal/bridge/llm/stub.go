package llm

import "context"

// Stub is the deterministic LLM client used by default in tests and
// for calibration reproducibility (PRD §15.4). It MUST be used by
// default unless TAUGO_LLM_BACKEND=real is set explicitly at the app layer.
type Stub struct{}

// Fingerprint returns a stable identifier for the stub.
// Real LLM backends carry their model + parameters in this string.
func (s Stub) Fingerprint() string { return "stub:v0" }

// Interpret returns a deterministic score in [0, 1) derived from the
// intent string via FNV-1a 32-bit hash. Mapping is checked-in (this
// function is the mapping).
func (s Stub) Interpret(_ context.Context, intent string) (float64, error) {
	const (
		offset uint32 = 2166136261
		prime  uint32 = 16777619
	)
	h := offset
	for i := 0; i < len(intent); i++ {
		h ^= uint32(intent[i])
		h *= prime
	}
	return float64(h%1000) / 1000.0, nil
}
