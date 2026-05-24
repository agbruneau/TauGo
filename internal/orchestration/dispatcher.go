package orchestration

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/calibration"
	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// defaultDimensionWeights holds the composite weights for tau_score = w_s*D_SENS + w_a*D_AUTH + w_i*D_INV.
// Initial values per PRD §11.1: (0.4, 0.3, 0.3). Status: Hypothesis.
var defaultDimensionWeights = struct{ DSens, DAuthority, DInvariant float64 }{
	DSens:      0.4,
	DAuthority: 0.3,
	DInvariant: 0.3,
}

// Dispatcher implements the PRD §10 tau pseudo-algorithm:
// steps 1 (frontier), 2 (ontological guard D-AUTORITE), 3 (profile expiry),
// 4 (dimension scores), 5 (I4 coherence guard), 6 (weighted composite),
// 7 (hysteresis decision), and 8 (invariant evaluation).
type Dispatcher struct {
	llm        llm.Client
	thresholds Thresholds
	profile    *calibration.Profile // nil => step 3 disabled (backward-compat)
	now        func() time.Time     // injectable clock for testing
}

// NewDispatcher constructs a Dispatcher with the given LLM client and thresholds.
// Step 3 (profile expiry) is disabled — use NewDispatcherWithProfile to enable it.
// Panics on ordering invariant violation (calque FibGo: invariant casse = panic interne).
func NewDispatcher(client llm.Client, t Thresholds) *Dispatcher {
	if !t.Ordered() {
		panic("orchestration: thresholds out of order (Deterministe > Probabiliste)")
	}
	return &Dispatcher{llm: client, thresholds: t, now: time.Now}
}

// NewDispatcherWithProfile constructs a Dispatcher that enforces step 3
// (profile expiry guard, PRD §10, anti-pattern #3). When today is past
// p.DateRevision, Decide returns Refus with "profil périmé — veille requise".
// Panics on ordering invariant violation.
func NewDispatcherWithProfile(client llm.Client, t Thresholds, p *calibration.Profile) *Dispatcher {
	d := NewDispatcher(client, t)
	d.profile = p
	return d
}

// WithClock replaces the Dispatcher's clock function. Used in tests to inject
// a fixed time for profile-expiry assertions.
func (d *Dispatcher) WithClock(c func() time.Time) *Dispatcher {
	d.now = c
	return d
}

// durationNs returns elapsed nanoseconds since start, guaranteeing at least 1
// to satisfy the Trace.DurationNs > 0 invariant on platforms (e.g. Windows)
// where the timer resolution may be coarser than 1 ns.
func durationNs(start time.Time) int64 {
	if d := time.Since(start).Nanoseconds(); d > 0 {
		return d
	}
	return 1
}

// refusDecision builds a Refus Decision with a minimal trace. Used by the
// early-exit guards in Decide (steps 1, 2, 3, 5) to keep each guard concise.
func refusDecision(x tau.Exchange, diag string, f tau.FrontierCheck, tt tau.TraceThresholds, ns int64) tau.Decision {
	return tau.Decision{
		Regime:     tau.Refus,
		Diagnostic: diag,
		Trace: tau.Trace{
			ExchangeID: x.ID,
			Frontier:   f,
			Thresholds: tt,
			DurationNs: ns,
		},
	}
}

// Decide implements PRD §10 steps 1, 2, 3, 4, 5, 6, 7, 8.
func (d *Dispatcher) Decide(ctx context.Context, x tau.Exchange) (tau.Decision, error) {
	start := time.Now()

	tt := tau.TraceThresholds{
		Deterministe:  d.thresholds.Deterministe,
		Probabiliste:  d.thresholds.Probabiliste,
		AuthBlock:     d.thresholds.AuthBlock,
		SensCoherence: d.thresholds.SensCoherence,
		InvCoherence:  d.thresholds.InvCoherence,
	}

	// Step 1 — Frontier check (M2 heuristic from DiscoveryMode and HumanInLoop).
	frontier := frontierFromExchange(x)
	if !frontier.Inside() {
		return refusDecision(x, "hors frontière τ", frontier, tt, durationNs(start)), nil
	}

	// Step 2 — Ontological guard D-AUTORITE (PRD §4.4, I3).
	authScore, err := dimensions.ScoreDAuthority(ctx, x, dimensions.DefaultAuthorityWeights())
	if err != nil {
		return tau.Decision{}, err
	}
	if authScore.Value >= d.thresholds.AuthBlock && x.AttestationInstitutionnelle == nil {
		return refusDecision(x, "I3 — verrou ontologique D-AUTORITÉ", frontier, tt, durationNs(start)), nil
	}

	// Step 3 — Profile expiry guard (PRD §10, §7.1 C3, anti-pattern #3).
	// Only active when a Profile was supplied via NewDispatcherWithProfile.
	if d.profile != nil && !d.profile.DateRevision.IsZero() && d.now().After(d.profile.DateRevision) {
		return refusDecision(x, "profil périmé — veille requise", frontier, tt, durationNs(start)), nil
	}

	// Step 4 — Dimension scores (D-SENS and D-INVARIANT; D-AUTORITE already computed).
	sensScore, err := dimensions.ScoreDSens(ctx, x, dimensions.DefaultSensWeights(), d.llm)
	if err != nil {
		return tau.Decision{}, err
	}
	invScore, err := dimensions.ScoreDInvariant(ctx, x, dimensions.DefaultInvariantWeights())
	if err != nil {
		return tau.Decision{}, err
	}

	// Step 5 — I4 coherence guard (PRD §6.1): low D-SENS with high D-INVARIANT.
	if invScore.Value >= d.thresholds.InvCoherence && sensScore.Value < d.thresholds.SensCoherence {
		return refusDecision(x, "I4 — combinaison incohérente détectée", frontier, tt, durationNs(start)), nil
	}

	// Step 6 — Weighted composite tau_score.
	tauScore := defaultDimensionWeights.DSens*sensScore.Value +
		defaultDimensionWeights.DAuthority*authScore.Value +
		defaultDimensionWeights.DInvariant*invScore.Value

	// Step 7 — Hysteresis decision (M2 default: Deterministe in the band).
	regime := tau.Deterministe
	if tauScore >= d.thresholds.Probabiliste {
		regime = tau.Probabiliste
	}

	decision := tau.Decision{
		Regime:         regime,
		ProfileVersion: "M3-default",
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   tauScore,
			Frontier:   frontier,
			Thresholds: tt,
			DurationNs: durationNs(start),
		},
	}

	// Step 8 — Invariant evaluation (observability only; Regime and Diagnostic unchanged).
	statuses := invariants.EvaluateInvariants(x, decision)
	if statuses.AnyViolated() {
		decision.Trace.UnmodeledObservations = append(
			decision.Trace.UnmodeledObservations,
			statuses.Summary()...,
		)
	}
	return decision, nil
}

// frontierFromExchange derives a FrontierCheck from the Exchange fields.
// This is a placeholder heuristic until M5 empirical calibration.
// Rules (all placeholder, documented as such):
//   - Target.DiscoveryMode != Static  => UniversOuvert=true, CompositionVariable=true
//   - !Initiator.HumanInLoop          => PairProbabiliste=true
//   - Initiator.DelegationDepth > 0   => CoutNonBorne=true
func frontierFromExchange(x tau.Exchange) tau.FrontierCheck {
	dynamic := x.Target.DiscoveryMode != tau.Static
	return tau.FrontierCheck{
		UniversOuvert:       dynamic,
		CompositionVariable: dynamic,
		PairProbabiliste:    !x.Initiator.HumanInLoop,
		CoutNonBorne:        x.Initiator.DelegationDepth > 0,
	}
}
