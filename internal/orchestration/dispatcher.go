package orchestration

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/tau"
)

// Dispatcher implements the M1 subset of the tau pseudo-algorithm
// (PRD §10): frontier check (step 1), naive composite from stub LLM
// score (step 6), and hysteresis decision (step 7). Steps 2 (ontological
// guard), 3 (profile expiration), 4 (full dimensional scores), 5 (I4
// coherence), and 8 (invariants evaluation) land in M2/M3/M5.
type Dispatcher struct {
	llm        llm.Client
	thresholds Thresholds
}

// NewDispatcher constructs a Dispatcher with the given LLM client and thresholds.
// Panics on ordering invariant violation (calque FibGo: invariant cassé = panic interne).
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

// Decide implements the M1 subset of PRD §10.
func (d *Dispatcher) Decide(ctx context.Context, x tau.Exchange) (tau.Decision, error) {
	start := time.Now()

	// Step 1 — Frontier check (M1: placeholder — assume Inside for any exchange;
	// the real frontier scoring lands in M2 when we have probes on Exchange).
	frontier := tau.FrontierCheck{
		UniversOuvert:       true,
		CompositionVariable: true,
		PairProbabiliste:    true,
		CoutNonBorne:        true,
	}
	if !frontier.Inside() {
		return tau.Decision{
			Regime:     tau.Refus,
			Diagnostic: "hors frontière τ",
			Trace: tau.Trace{
				ExchangeID: x.ID,
				Frontier:   frontier,
				Thresholds: tau.TraceThresholds{
					Deterministe: d.thresholds.Deterministe,
					Probabiliste: d.thresholds.Probabiliste,
				},
				DurationNs: durationNs(start),
			},
		}, nil
	}

	// Step 6 — Naive composite (M1: tau score = LLM stub score).
	tauScore, err := d.llm.Interpret(ctx, x.IntentDescription)
	if err != nil {
		return tau.Decision{}, err
	}

	// Step 7 — Decision with hysteresis (M1: defaults to Deterministe in the band).
	var regime tau.Regime
	switch {
	case tauScore >= d.thresholds.Probabiliste:
		regime = tau.Probabiliste
	default:
		// Covers tauScore < Deterministe and the hysteresis zone.
		// M1 default: Deterministe. M2 will track per-exchange history.
		regime = tau.Deterministe
	}

	return tau.Decision{
		Regime:         regime,
		ProfileVersion: "M1-default",
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   tauScore,
			Frontier:   frontier,
			Thresholds: tau.TraceThresholds{
				Deterministe: d.thresholds.Deterministe,
				Probabiliste: d.thresholds.Probabiliste,
			},
			DurationNs: durationNs(start),
		},
	}, nil
}
