package orchestration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/calibration"
	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestDispatcher_DecisionContientVersionProfileEtDateRevision verifies that a
// Dispatcher constructed with a Profile propagates the profile's Version and
// DateRevision into the Decision (T-039).
func TestDispatcher_DecisionContientVersionProfileEtDateRevision(t *testing.T) {
	t.Parallel()

	th := orchestration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	}

	knownDate := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	p := calibration.DefaultProfile()
	p.Version = "test-v0.1.1"
	p.DateRevision = knownDate

	d := orchestration.NewDispatcherWithProfile(fakeLLM{score: 0.50}, th, &p)

	x := tau.Exchange{
		ID:                "e-version-test",
		IntentDescription: "test intent",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent",
			HumanInLoop:     false,
			Organization:    "org-x",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "target-svc",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
	}

	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime == tau.Refus {
		t.Skipf("dispatcher refused exchange (diag=%q); cannot assert ProfileVersion", dec.Diagnostic)
	}
	if dec.ProfileVersion != "test-v0.1.1" {
		t.Fatalf("ProfileVersion = %q, want %q", dec.ProfileVersion, "test-v0.1.1")
	}
	if !dec.DateRevision.Equal(knownDate) {
		t.Fatalf("DateRevision = %v, want %v", dec.DateRevision, knownDate)
	}
}

// TestDispatcher_AppliqueProfileWeights verifies that a Dispatcher constructed
// with a Profile whose Weights are non-uniform produces a different TauScore
// than a Dispatcher without a Profile (which uses defaultDimensionWeights).
// Both dispatchers receive the same Exchange; the two TauScores must diverge
// when the profile weights differ from the defaults (T-017, ADR-0006).
func TestDispatcher_AppliqueProfileWeights(t *testing.T) {
	t.Parallel()

	th := orchestration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	}

	// Dispatcher without profile — uses defaultDimensionWeights (0.4, 0.3, 0.3).
	dDefault := orchestration.NewDispatcher(fakeLLM{score: 0.50}, th)

	// Profile with non-uniform weights (0.6, 0.2, 0.2) to ensure divergence.
	p := calibration.DefaultProfile()
	p.Weights.DSens = 0.60
	p.Weights.DAuthority = 0.20
	p.Weights.DInvariant = 0.20
	// DateRevision far in the future so the expiry guard does not fire.
	p.DateRevision = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	dProfile := orchestration.NewDispatcherWithProfile(fakeLLM{score: 0.50}, th, &p)

	x := tau.Exchange{
		ID:                "e-profile-weights",
		IntentDescription: "test intent",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent",
			HumanInLoop:     false,
			Organization:    "org-x",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "target-svc",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
	}

	decDefault, err := dDefault.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("dDefault.Decide error: %v", err)
	}
	decProfile, err := dProfile.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("dProfile.Decide error: %v", err)
	}

	if decDefault.Regime == tau.Refus {
		t.Skipf("default dispatcher refused exchange (diag=%q); cannot compare scores", decDefault.Diagnostic)
	}
	if decProfile.Regime == tau.Refus {
		t.Skipf("profile dispatcher refused exchange (diag=%q); cannot compare scores", decProfile.Diagnostic)
	}

	if decDefault.Trace.TauScore == decProfile.Trace.TauScore {
		t.Fatalf(
			"TauScore identical (%.6f) with different dimension weights; "+
				"expected divergence between defaultWeights(0.4,0.3,0.3) and profile(0.6,0.2,0.2)",
			decDefault.Trace.TauScore,
		)
	}
	t.Logf("default TauScore=%.6f  profile TauScore=%.6f", decDefault.Trace.TauScore, decProfile.Trace.TauScore)
}
