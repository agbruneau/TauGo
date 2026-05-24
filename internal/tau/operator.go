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
// at the time of the decision. Mirrors orchestration.Thresholds; kept here
// to avoid a tau -> orchestration import (forbidden by arch_test).
type TraceThresholds struct {
	Deterministe  float64 `json:"deterministe"`
	Probabiliste  float64 `json:"probabiliste"`
	AuthBlock     float64 `json:"auth_block"`
	SensCoherence float64 `json:"sens_coherence"`
	InvCoherence  float64 `json:"inv_coherence"`
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
