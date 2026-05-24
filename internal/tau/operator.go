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
	ID                          string
	IntentDescription           string
	DiscoveredAt                time.Time
	AttestationInstitutionnelle *Attestation
	Context                     map[string]any
	// Principal and Capability fields intentionally omitted in M0;
	// added in M2 alongside the dimensions.
}

// Attestation is the opposable reference that populates the "execution"
// pole of D-AUTORITÉ (chap. III.8.4.2.bis, Searle 1995).
type Attestation struct {
	Emetteur   string
	Reference  string
	Marqueur   string
	AssertedAt time.Time
}

// TraceThresholds is the immutable snapshot of the thresholds in effect
// at the time of the decision. Mirrors orchestration.Thresholds; kept here
// to avoid a tau -> orchestration import (forbidden by arch_test).
type TraceThresholds struct {
	Deterministe float64
	Probabiliste float64
}

// Trace is the immutable instrumentation of a Decision.
// Once Decision is returned, the Trace must not be mutated.
type Trace struct {
	ExchangeID            string
	TauScore              float64         // composite tau score (M1: stub LLM score; M2: 3-dim weighted)
	Frontier              FrontierCheck   // state of the 4 classical conditions
	Thresholds            TraceThresholds // snapshot at decision time
	UnmodeledObservations []string        // PRD §7.2 #4 — observations not modeled
	DurationNs            int64
}

// Decision is the full output of Kernel.Decide. Always traced.
type Decision struct {
	Regime         Regime
	Diagnostic     string // non-empty iff Regime == Refus
	ProfileVersion string
	DateRevision   time.Time
	Trace          Trace
}

// Kernel is the public face of the τ operator. Single entry point: Decide.
type Kernel interface {
	Decide(ctx context.Context, x Exchange) (Decision, error)
}
