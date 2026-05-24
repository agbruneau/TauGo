package calibration

import "time"

// Weights holds the composite and per-probe weights for the three dimensions.
// DSens + DAuthority + DInvariant must sum to 1.0.
// Each probe map (SensProbes, AuthorityProbes, InvariantProbes) must sum to 1.0.
type Weights struct {
	DSens           float64            `json:"d_sens"`
	DAuthority      float64            `json:"d_authority"`
	DInvariant      float64            `json:"d_invariant"`
	SensProbes      map[string]float64 `json:"sens_probes,omitempty"`
	AuthorityProbes map[string]float64 `json:"authority_probes,omitempty"`
	InvariantProbes map[string]float64 `json:"invariant_probes,omitempty"`
}

// Thresholds holds the full set of calibrated decision thresholds.
// Mirrors orchestration.Thresholds; separate to avoid calibration -> orchestration import.
type Thresholds struct {
	Deterministe  float64 `json:"deterministe"`
	Probabiliste  float64 `json:"probabiliste"`
	AuthBlock     float64 `json:"auth_block"`
	SensCoherence float64 `json:"sens_coherence"`
	InvCoherence  float64 `json:"inv_coherence"`
	HysteresisGap float64 `json:"hysteresis_gap"`
}

// Profile is the versioned, opposable calibration record for the tau operator.
// Every Profile carries fingerprints of the environment in which it was produced;
// a changed fingerprint invalidates the profile (PRD §11.4).
type Profile struct {
	ID                  string     `json:"id"`
	Version             string     `json:"version"`
	CreatedAt           time.Time  `json:"created_at"`
	DateRevision        time.Time  `json:"date_revision"`       // expiry date (PRD §7.1 C3)
	VersionMonographie  string     `json:"version_monographie"` // pinned monograph tag
	CPUFingerprint      string     `json:"cpu_fingerprint"`
	ModelLLMFingerprint string     `json:"model_llm_fingerprint"`
	CorpusFingerprint   string     `json:"corpus_fingerprint"`
	Thresholds          Thresholds `json:"thresholds"`
	Weights             Weights    `json:"weights"`
}

// DefaultProfile returns the initial profile with PRD §11.1 values.
// Status: Hypothesis — thresholds and weights to be corroborated in M4/M5.
func DefaultProfile() Profile {
	now := time.Now().UTC()
	// DateRevision: 6 months ahead per PRD §11.4 minimum. Initial value 2026-12-01
	// (chosen with margin over the 2026-05-23 authoring date to satisfy >= 6 months strictly).
	dateRevision := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	return Profile{
		ID:                  "default",
		Version:             "0.1.0",
		CreatedAt:           now,
		DateRevision:        dateRevision,
		VersionMonographie:  "v2.4.3",
		CPUFingerprint:      "",
		ModelLLMFingerprint: "stub:v0",
		CorpusFingerprint:   "",
		Thresholds: Thresholds{
			Deterministe:  0.35,
			Probabiliste:  0.65,
			AuthBlock:     0.85,
			SensCoherence: 0.50,
			InvCoherence:  0.50,
			HysteresisGap: 0.10,
		},
		Weights: Weights{
			DSens:      0.4,
			DAuthority: 0.3,
			DInvariant: 0.3,
			SensProbes: map[string]float64{
				"S_contract":             0.35,
				"S_runtime_resolve":      0.30,
				"S_capability_discovery": 0.20,
				"S_reasoner_intent":      0.15,
			},
			AuthorityProbes: map[string]float64{
				"A_chain_depth":        0.25,
				"A_cross_org":          0.25,
				"A_human_anchor":       0.25,
				"A_dynamic_resolution": 0.25,
			},
			InvariantProbes: map[string]float64{
				"I_event_registry":       0.30,
				"I_idempotency_derived":  0.25,
				"I_capability_mediation": 0.25,
				"I_enumerated_plan":      0.20,
			},
		},
	}
}
