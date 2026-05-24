// internal/tau/operator.go
package tau

import (
	"context"
	"time"
)

// Regime is the discrete output of τ. Never a behavior, never a result.
type Regime int

const (
	RegimeUnknown Regime = iota
	Deterministe
	Probabiliste
	Refus
)

// Exchange is the interoperability exchange submitted to τ.
type Exchange struct {
	ID                          string         `json:"id"`
	IntentDescription           string         `json:"intent_description"`
	DiscoveredAt                time.Time      `json:"discovered_at"`
	AttestationInstitutionnelle *Attestation   `json:"attestation_institutionnelle,omitempty"`
	Context                     map[string]any `json:"context,omitempty"`
	// Principal and Capability fields intentionally omitted in M0;
	// added in M2 alongside the dimensions.
}

// Attestation is the opposable reference that populates the "execution"
// pole of D-AUTORITÉ (chap. III.8.4.2.bis, Searle 1995).
type Attestation struct {
	Emetteur   string    `json:"emetteur"`
	Reference  string    `json:"reference"`
	Marqueur   string    `json:"marqueur"`
	AssertedAt time.Time `json:"asserted_at"`
}

// TraceThresholds is the immutable snapshot of the thresholds in effect
// at the time of the decision. Mirrors orchestration.Thresholds; kept here
// to avoid a tau -> orchestration import (forbidden by arch_test).
type TraceThresholds struct {
	Deterministe float64 `json:"deterministe"`
	Probabiliste float64 `json:"probabiliste"`
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
