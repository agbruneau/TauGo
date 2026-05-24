// internal/tau/operator.go
package tau

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/agbruneau/taugo/internal/thresholds"
)

// Regime is the discrete output of τ. Never a behavior, never a result.
type Regime int

const (
	RegimeUnknown Regime = iota
	Deterministe
	Probabiliste
	Refus
)

// regimeStrings maps Regime values to their canonical PascalCase
// representation, used for JSON marshaling and human-readable logs.
//
//nolint:gochecknoglobals // immutable lookup table
var regimeStrings = map[Regime]string{
	RegimeUnknown: "RegimeUnknown",
	Deterministe:  "Deterministe",
	Probabiliste:  "Probabiliste",
	Refus:         "Refus",
}

// String returns the canonical PascalCase representation of r.
func (r Regime) String() string {
	if s, ok := regimeStrings[r]; ok {
		return s
	}
	return "Unknown"
}

// MarshalJSON serializes r as its String() representation.
func (r Regime) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// UnmarshalJSON accepts either a JSON string (v0.1.1+) or a JSON number
// (legacy v0.1.0) for backward compatibility.
func (r *Regime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		for k, v := range regimeStrings {
			if v == s {
				*r = k
				return nil
			}
		}
		// Tolerate legacy lowercase corpus values.
		for k, v := range regimeStrings {
			if strings.EqualFold(v, s) {
				*r = k
				return nil
			}
		}
		return fmt.Errorf("tau.Regime: valeur inconnue %q", s)
	}
	// Fallback: decode as int (retro-compat v0.1.0).
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("tau.Regime: JSON non décodable: %w", err)
	}
	*r = Regime(n)
	return nil
}

// Exchange is the interoperability exchange submitted to τ.
type Exchange struct {
	ID                          string         `json:"id"`
	Initiator                   Principal      `json:"initiator"`
	Target                      Capability     `json:"target"`
	IntentDescription           string         `json:"intent_description"`
	DiscoveredAt                time.Time      `json:"discovered_at"`
	AttestationInstitutionnelle *Attestation   `json:"attestation_institutionnelle,omitempty"`
	Context                     map[string]any `json:"context,omitempty"`
}

// Attestation is the opposable reference that populates the "execution"
// pole of D-AUTORITÉ (chap. III.8.4.2.bis, Searle 1995).
type Attestation struct {
	Emetteur   string    `json:"emetteur"`
	Reference  string    `json:"reference"`
	Marqueur   string    `json:"marqueur"`
	AssertedAt time.Time `json:"asserted_at"`
}

// DiscoveryMode describes how a Capability is discovered at the boundary.
type DiscoveryMode int

const (
	// Static means the capability is known and wired at design time.
	Static DiscoveryMode = iota
	// DynamicMCP means the capability is discovered via MCP list_tools at runtime.
	DynamicMCP
	// DynamicA2A means the capability is discovered via A2A protocol at runtime.
	DynamicA2A
	// DynamicAGNTCY means the capability is discovered via AGNTCY registry at runtime.
	DynamicAGNTCY
)

// discoveryModeStrings maps DiscoveryMode values to their canonical PascalCase
// representation, used for JSON marshaling and human-readable logs.
//
//nolint:gochecknoglobals // immutable lookup table
var discoveryModeStrings = map[DiscoveryMode]string{
	Static:        "Static",
	DynamicMCP:    "DynamicMCP",
	DynamicA2A:    "DynamicA2A",
	DynamicAGNTCY: "DynamicAGNTCY",
}

// String returns the canonical PascalCase representation of d.
func (d DiscoveryMode) String() string {
	if s, ok := discoveryModeStrings[d]; ok {
		return s
	}
	return "Unknown"
}

// MarshalJSON serializes d as its String() representation.
func (d DiscoveryMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON accepts either a JSON string (v0.1.1+) or a JSON number
// (legacy v0.1.0) for backward compatibility.
func (d *DiscoveryMode) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		for k, v := range discoveryModeStrings {
			if v == s {
				*d = k
				return nil
			}
		}
		// Tolerate legacy snake_case corpus values (e.g. "dynamic_mcp").
		sl := strings.ReplaceAll(strings.ToLower(s), "_", "")
		for k, v := range discoveryModeStrings {
			if strings.EqualFold(v, sl) {
				*d = k
				return nil
			}
		}
		return fmt.Errorf("tau.DiscoveryMode: valeur inconnue %q", s)
	}
	// Fallback: decode as int (retro-compat v0.1.0).
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("tau.DiscoveryMode: JSON non décodable: %w", err)
	}
	*d = DiscoveryMode(n)
	return nil
}

// Principal is the initiating agent of an interoperability exchange.
type Principal struct {
	ID              string `json:"id"`
	HumanInLoop     bool   `json:"human_in_loop"`
	Organization    string `json:"organization"`
	DelegationDepth int    `json:"delegation_depth"` // 0 = direct human mandate
}

// Capability is the target capability being invoked in the exchange.
type Capability struct {
	ID            string        `json:"id"`
	DiscoveryMode DiscoveryMode `json:"discovery_mode"`
	ContractURI   string        `json:"contract_uri,omitempty"` // empty = no contract
}

// TraceThresholds is the immutable snapshot of the thresholds in effect
// at the time of the decision. Aliased from internal/thresholds per ADR-0006
// (Types valeur transverses — accepté 2026-05-24). The alias preserves all
// existing field references; HysteresisGap is an additive field (zero by default).
type TraceThresholds = thresholds.Thresholds

// Score is a normalized [0,1] dimension score with full probe-level traceability.
// Promoted here from tau/dimensions to break the import cycle that would arise
// if tau.Trace carried tau/dimensions.Score directly (ADR-0008, accepté 2026-05-24).
// tau/dimensions declares: type Score = tau.Score (alias — no method divergence).
type Score struct {
	Value      float64            `json:"value"`
	Probes     map[string]float64 `json:"probes,omitempty"`
	Weights    map[string]float64 `json:"weights,omitempty"`
	ComputedAt time.Time          `json:"computed_at,omitempty"`
}

// Trace is the immutable instrumentation of a Decision.
// Once Decision is returned, the Trace must not be mutated.
type Trace struct {
	ExchangeID            string          `json:"exchange_id"`
	TauScore              float64         `json:"tau_score"`
	Frontier              FrontierCheck   `json:"frontier"`
	Thresholds            TraceThresholds `json:"thresholds"`
	UnmodeledObservations []string        `json:"unmodeled_observations,omitempty"`
	DurationNs            int64           `json:"duration_ns"`

	// Ventilated dimension scores (ADR-0008, accepté 2026-05-24).
	// DSens is the D-SENS score computed at dispatcher step 4.
	// DAuthority is the D-AUTORITÉ score computed at dispatcher step 2.
	// DInvariant is the D-INVARIANT score computed at dispatcher step 4.
	// Nil pointer indicates the score was not computed (early-exit Refus path).
	// Pointer semantics are required for omitempty to work on a struct type.
	DSens      *Score `json:"d_sens,omitempty"`
	DAuthority *Score `json:"d_authority,omitempty"`
	DInvariant *Score `json:"d_invariant,omitempty"`
}

// Decision is the full output of Kernel.Decide. Always traced.
type Decision struct {
	Regime         Regime    `json:"regime"`
	Diagnostic     string    `json:"diagnostic,omitempty"`
	ProfileVersion string    `json:"profile_version"`
	DateRevision   time.Time `json:"date_revision"`
	Trace          Trace     `json:"trace"`
}

// Kernel is the public face of the τ operator. Single entry point: Decide.
type Kernel interface {
	Decide(ctx context.Context, x Exchange) (Decision, error)
}

// FrontierCheck derives a FrontierCheck from this Exchange's fields using the
// four classical conditions (PRD §4.3). This is the canonical heuristic until
// M5 empirical calibration; both the dispatcher (step 1) and invariant I2's
// Recablage rely on it to avoid drift between the two derivations.
//   - Target.DiscoveryMode != Static => UniversOuvert=true, CompositionVariable=true
//   - !Initiator.HumanInLoop        => PairProbabiliste=true
//   - Initiator.DelegationDepth > 0 => CoutNonBorne=true
func (x Exchange) FrontierCheck() FrontierCheck {
	dynamic := x.Target.DiscoveryMode != Static
	return FrontierCheck{
		UniversOuvert:       dynamic,
		CompositionVariable: dynamic,
		PairProbabiliste:    !x.Initiator.HumanInLoop,
		CoutNonBorne:        x.Initiator.DelegationDepth > 0,
	}
}
