package dimensions_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

func authorityWeights() dimensions.AuthorityWeights {
	return dimensions.AuthorityWeights{
		ChainDepth:        0.25,
		CrossOrg:          0.25,
		HumanAnchor:       0.25,
		DynamicResolution: 0.25,
	}
}

func newShortChainExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-short-chain",
		DiscoveredAt: time.Now(),
		Initiator: tau.Principal{
			ID:              "human-user",
			HumanInLoop:     true,
			Organization:    "org-a",
			DelegationDepth: 0,
		},
		Target: tau.Capability{
			ID:            "internal-svc",
			DiscoveryMode: tau.Static,
			ContractURI:   "https://internal/api",
		},
	}
}

func newLongChainExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-long-chain",
		DiscoveredAt: time.Now(),
		Initiator: tau.Principal{
			ID:              "agent-orchestrator",
			HumanInLoop:     false,
			Organization:    "org-b",
			DelegationDepth: 5,
		},
		Target: tau.Capability{
			ID:            "external-api",
			DiscoveryMode: tau.DynamicA2A,
			ContractURI:   "",
		},
	}
}

func TestDAuthority_Bounded(t *testing.T) {
	t.Parallel()
	w := authorityWeights()
	for _, x := range []tau.Exchange{newShortChainExchange(), newLongChainExchange()} {
		t.Run(x.ID, func(t *testing.T) {
			t.Parallel()
			score, err := dimensions.ScoreDAuthority(context.Background(), x, w)
			if err != nil {
				t.Fatalf("ScoreDAuthority error: %v", err)
			}
			if score.Value < 0 || score.Value > 1 {
				t.Fatalf("ScoreDAuthority value %f out of [0,1]", score.Value)
			}
		})
	}
}

func TestDAuthority_ShortChainLowerThanLong(t *testing.T) {
	t.Parallel()
	w := authorityWeights()
	short, err := dimensions.ScoreDAuthority(context.Background(), newShortChainExchange(), w)
	if err != nil {
		t.Fatalf("short: %v", err)
	}
	long, err := dimensions.ScoreDAuthority(context.Background(), newLongChainExchange(), w)
	if err != nil {
		t.Fatalf("long: %v", err)
	}
	if short.Value >= long.Value {
		t.Fatalf("expected short-chain (%f) < long-chain (%f)", short.Value, long.Value)
	}
}

func TestDAuthority_ProbeWeightsSumToOne(t *testing.T) {
	t.Parallel()
	w := authorityWeights()
	sum := w.ChainDepth + w.CrossOrg + w.HumanAnchor + w.DynamicResolution
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("authority probe weights sum = %f, want 1.0", sum)
	}
}

func TestDAuthority_ProbesMapPopulated(t *testing.T) {
	t.Parallel()
	w := authorityWeights()
	score, err := dimensions.ScoreDAuthority(context.Background(), newLongChainExchange(), w)
	if err != nil {
		t.Fatalf("ScoreDAuthority error: %v", err)
	}
	expected := []string{"A_chain_depth", "A_cross_org", "A_human_anchor", "A_dynamic_resolution"}
	for _, k := range expected {
		if _, ok := score.Probes[k]; !ok {
			t.Errorf("probe %q missing from score.Probes", k)
		}
	}
}
