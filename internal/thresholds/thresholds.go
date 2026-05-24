// Package thresholds defines the canonical Thresholds value type shared
// across tau, orchestration, and calibration packages.
//
// ADR-0006 (Types valeur transverses) — accepté 2026-05-24.
//
// This package has NO dependency on any other taugo package, to preserve
// Clean Architecture layering (arch_test.go enforces this).
package thresholds

// Thresholds groupe les seuils de décision du kernel τ.
//
//   - Deterministe    : tau_score < ce seuil → régime déterministe
//   - Probabiliste    : tau_score >= ce seuil → régime probabiliste
//   - AuthBlock       : seuil de D-AUTORITÉ déclenchant le verrou ontologique I3
//   - SensCoherence   : seuil bas de D-SENS au-dessous duquel I4 contrôle la cohérence
//   - InvCoherence    : seuil de D-INVARIANT déclenchant la garde de cohérence I4
//   - HysteresisGap   : largeur de la bande d'hystérèse autour de la frontière
type Thresholds struct {
	Deterministe  float64 `json:"deterministe"`
	Probabiliste  float64 `json:"probabiliste"`
	AuthBlock     float64 `json:"auth_block"`
	SensCoherence float64 `json:"sens_coherence"`
	InvCoherence  float64 `json:"inv_coherence"`
	HysteresisGap float64 `json:"hysteresis_gap"`
}

// Ordered reports whether the ordering invariant holds: Deterministe <= Probabiliste.
// Must be true at all times; a violation indicates a misconfiguration.
func (t Thresholds) Ordered() bool { return t.Deterministe <= t.Probabiliste }
