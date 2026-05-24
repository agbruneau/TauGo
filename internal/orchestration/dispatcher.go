package orchestration

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

// defaultDimensionWeights holds the composite weights for tau_score = w_s*D_SENS + w_a*D_AUTH + w_i*D_INV.
// Initial values per PRD §11.1: (0.4, 0.3, 0.3). Status: Hypothesis.
var defaultDimensionWeights = struct{ DSens, DAuthority, DInvariant float64 }{
	DSens:      0.4,
	DAuthority: 0.3,
	DInvariant: 0.3,
}

// Dispatcher implements the M2 subset of the tau pseudo-algorithm (PRD §10):
// steps 1 (frontier), 2 (ontological guard D-AUTORITE), 4 (dimension scores),
// 5 (I4 coherence guard), 6 (weighted composite), and 7 (hysteresis decision).
// Steps 3 (profile expiration) and 8 (invariant evaluation) land in M3/M5.
type Dispatcher struct {
	llm        llm.Client
	thresholds Thresholds
}

// NewDispatcher constructs a Dispatcher with the given LLM client and thresholds.
// Panics on ordering invariant violation (calque FibGo: invariant casse = panic interne).
func NewDispatcher(client llm.Client, t Thresholds) *Dispatcher {
	if !t.Ordered() {
		panic("orchestration: thresholds out of order (Deterministe > Probabiliste)")
	}
	return &Dispatcher{llm: client, thresholds: t}
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

// Decide implements the M2 subset of PRD §10 (steps 1, 2, 4, 5, 6, 7).
func (d *Dispatcher) Decide(ctx context.Context, x tau.Exchange) (tau.Decision, error) {
	start := time.Now()

	traceThresholds := tau.TraceThresholds{
		Deterministe:  d.thresholds.Deterministe,
		Probabiliste:  d.thresholds.Probabiliste,
		AuthBlock:     d.thresholds.AuthBlock,
		SensCoherence: d.thresholds.SensCoherence,
		InvCoherence:  d.thresholds.InvCoherence,
	}

	// Step 1 — Frontier check derived from Exchange (M2: heuristic from
	// Capability.DiscoveryMode and Principal.HumanInLoop).
	frontier := frontierFromExchange(x)
	if !frontier.Inside() {
		return tau.Decision{
			Regime:     tau.Refus,
			Diagnostic: "hors frontière τ",
			Trace: tau.Trace{
				ExchangeID: x.ID,
				Frontier:   frontier,
				Thresholds: traceThresholds,
				DurationNs: durationNs(start),
			},
		}, nil
	}

	// Step 2 — Ontological guard D-AUTORITE (PRD §4.4, I3).
	authScore, err := dimensions.ScoreDAuthority(ctx, x, dimensions.DefaultAuthorityWeights())
	if err != nil {
		return tau.Decision{}, err
	}
	if authScore.Value >= d.thresholds.AuthBlock && x.AttestationInstitutionnelle == nil {
		return tau.Decision{
			Regime:     tau.Refus,
			Diagnostic: "I3 — verrou ontologique D-AUTORITÉ",
			Trace: tau.Trace{
				ExchangeID: x.ID,
				Frontier:   frontier,
				Thresholds: traceThresholds,
				DurationNs: durationNs(start),
			},
		}, nil
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

	// Step 5 — I4 coherence guard (PRD §6.1):
	// D-INVARIANT >= InvCoherence AND D-SENS < SensCoherence => incoherent combination.
	if invScore.Value >= d.thresholds.InvCoherence && sensScore.Value < d.thresholds.SensCoherence {
		return tau.Decision{
			Regime:     tau.Refus,
			Diagnostic: "I4 — combinaison incohérente détectée",
			Trace: tau.Trace{
				ExchangeID: x.ID,
				Frontier:   frontier,
				Thresholds: traceThresholds,
				DurationNs: durationNs(start),
			},
		}, nil
	}

	// Step 6 — Weighted composite tau_score.
	tauScore := defaultDimensionWeights.DSens*sensScore.Value +
		defaultDimensionWeights.DAuthority*authScore.Value +
		defaultDimensionWeights.DInvariant*invScore.Value

	// Step 7 — Decision with hysteresis (M2: same default as M1 — Deterministe in the band).
	var regime tau.Regime
	switch {
	case tauScore >= d.thresholds.Probabiliste:
		regime = tau.Probabiliste
	default:
		// Covers tauScore < Deterministe and the hysteresis zone.
		// M2 default: Deterministe. Regime history tracking deferred to M5.
		regime = tau.Deterministe
	}

	return tau.Decision{
		Regime:         regime,
		ProfileVersion: "M2-default",
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   tauScore,
			Frontier:   frontier,
			Thresholds: traceThresholds,
			DurationNs: durationNs(start),
		},
	}, nil
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
