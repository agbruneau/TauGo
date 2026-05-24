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
var defaultDimensionWeights = struct{ DSens, DAuthority, DInvariant float64 }{ //nolint:gochecknoglobals // read-only after init; package-private calibration default, see PRD §11.1
	DSens:      0.4,
	DAuthority: 0.3,
	DInvariant: 0.3,
}

// Compile-time assertion that Dispatcher satisfies the tau.Kernel interface.
var _ tau.Kernel = (*Dispatcher)(nil)

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
//
// SECURITY NOTE: This constructor disables the profile-expiry guard
// (PRD §7.3 case 4, anti-pattern #3). Reserved for internal tests only.
// Production code MUST use NewDispatcherWithProfile, or the app.NewDispatcher
// wrapper which injects a default profile automatically.
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

// dimensionWeights returns the composite dimension weights in effect for this
// dispatch. When a Profile with non-zero Weights is injected (T-017), its
// values override the package-level defaults.
func (d *Dispatcher) dimensionWeights() struct{ DSens, DAuthority, DInvariant float64 } {
	if d.profile != nil && (d.profile.Weights.DSens+d.profile.Weights.DAuthority+d.profile.Weights.DInvariant) > 0 {
		return struct{ DSens, DAuthority, DInvariant float64 }{
			DSens:      d.profile.Weights.DSens,
			DAuthority: d.profile.Weights.DAuthority,
			DInvariant: d.profile.Weights.DInvariant,
		}
	}
	return defaultDimensionWeights
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
// early-exit guards in Decide (steps 1, 3) where no scores are available yet.
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

// refusDecisionWithScores builds a Refus Decision carrying available ventilated
// scores. Used by step 2 (DAuthority known) and step 5 (all three scores known).
func refusDecisionWithScores(x tau.Exchange, diag string, f tau.FrontierCheck, tt tau.TraceThresholds, ns int64,
	auth, sens, inv *tau.Score) tau.Decision {
	return tau.Decision{
		Regime:     tau.Refus,
		Diagnostic: diag,
		Trace: tau.Trace{
			ExchangeID: x.ID,
			Frontier:   f,
			Thresholds: tt,
			DurationNs: ns,
			DAuthority: auth,
			DSens:      sens,
			DInvariant: inv,
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
	frontier := x.FrontierCheck()
	if !frontier.Inside() {
		return refusDecision(x, tau.DiagFrontiereFranchie, frontier, tt, durationNs(start)), nil
	}

	// Step 2 — Ontological guard D-AUTORITE (PRD §4.4, I3).
	// Score is computed before the guard to populate Trace.DAuthority on Refus.
	authScore, err := dimensions.ScoreDAuthority(ctx, x, dimensions.DefaultAuthorityWeights())
	if err != nil {
		return tau.Decision{}, err
	}
	authPtr := &tau.Score{Value: authScore.Value, Probes: authScore.Probes, Weights: authScore.Weights, ComputedAt: authScore.ComputedAt}
	if authScore.Value >= d.thresholds.AuthBlock && x.AttestationInstitutionnelle == nil {
		return refusDecisionWithScores(x, tau.DiagVerrouOntologique, frontier, tt, durationNs(start),
			authPtr, nil, nil), nil
	}

	// Step 3 — Profile expiry guard (PRD §10, §7.1 C3, anti-pattern #3).
	// Only active when a Profile was supplied via NewDispatcherWithProfile.
	// Uses After (strict): today == dateRevision is NOT a hard refusal here.
	// calibration.CheckDrift uses !Before (>=), firing one day earlier as an
	// early warning. This asymmetry is deliberate; see drift.go CheckDrift.
	if d.profile != nil && !d.profile.DateRevision.IsZero() && d.now().After(d.profile.DateRevision) {
		return refusDecision(x, tau.DiagPeremptionProfile, frontier, tt, durationNs(start)), nil
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
	sensPtr := &tau.Score{Value: sensScore.Value, Probes: sensScore.Probes, Weights: sensScore.Weights, ComputedAt: sensScore.ComputedAt}
	invPtr := &tau.Score{Value: invScore.Value, Probes: invScore.Probes, Weights: invScore.Weights, ComputedAt: invScore.ComputedAt}

	// Step 5 — I4 coherence guard (PRD §6.1): low D-SENS with high D-INVARIANT.
	if invScore.Value >= d.thresholds.InvCoherence && sensScore.Value < d.thresholds.SensCoherence {
		return refusDecisionWithScores(x, tau.DiagIncoherenceI4, frontier, tt, durationNs(start),
			authPtr, sensPtr, invPtr), nil
	}

	// Step 6 — Weighted composite tau_score (Profile.Weights injected if available, T-017).
	w := d.dimensionWeights()
	tauScore := w.DSens*sensScore.Value + w.DAuthority*authScore.Value + w.DInvariant*invScore.Value

	// Step 7 — Hysteresis decision (M2 default: Deterministe in the band).
	regime := tau.Deterministe
	if tauScore >= d.thresholds.Probabiliste {
		regime = tau.Probabiliste
	}

	decision := tau.Decision{
		Regime: regime,
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   tauScore,
			Frontier:   frontier,
			Thresholds: tt,
			DurationNs: durationNs(start),
			DAuthority: authPtr,
			DSens:      sensPtr,
			DInvariant: invPtr,
		},
	}
	if d.profile != nil {
		decision.ProfileVersion = d.profile.Version
		decision.DateRevision = d.profile.DateRevision
	} else {
		// Profile non injecté : valeur sentinel pour tests internes uniquement.
		// La trap péremption est gardée par TestApp_NewDispatcher_ChargeProfilParDefaut.
		decision.ProfileVersion = "M3-default-no-profile"
		decision.DateRevision = time.Time{}
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
