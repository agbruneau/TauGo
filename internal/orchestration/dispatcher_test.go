package orchestration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

type fakeLLM struct{ score float64 }

func (f fakeLLM) Fingerprint() string                                    { return "fake" }
func (f fakeLLM) Interpret(_ context.Context, _ string) (float64, error) { return f.score, nil }

func newExchange(id string) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: "test intent",
		DiscoveredAt:      time.Now(),
	}
}

func TestDispatcher_Decide_Deterministe(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.20}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	dec, err := d.Decide(context.Background(), newExchange("t-det"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Deterministe {
		t.Fatalf("regime = %v, want Deterministe", dec.Regime)
	}
	if dec.Trace.ExchangeID != "t-det" {
		t.Fatalf("trace ExchangeID = %q, want \"t-det\"", dec.Trace.ExchangeID)
	}
}

func TestDispatcher_Decide_Probabiliste(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.80}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	dec, err := d.Decide(context.Background(), newExchange("t-prob"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Probabiliste {
		t.Fatalf("regime = %v, want Probabiliste", dec.Regime)
	}
}

func TestDispatcher_Decide_HysteresisDefaultsToDeterministe(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.50}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	dec, _ := d.Decide(context.Background(), newExchange("t-hyst"))
	if dec.Regime != tau.Deterministe {
		t.Fatalf("hysteresis zone: regime = %v, want Deterministe (M1 default)", dec.Regime)
	}
}
