package dimensions_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
	"github.com/agbruneau/taugo/internal/testutil"
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
	// Migration PoC (T-019): use testutil.BuildExchange for the short-chain case.
	shortX := testutil.BuildExchange(
		testutil.WithID("x-short-chain"),
		testutil.WithHumanInLoop(true),
		testutil.WithDelegationDepth(0),
		testutil.WithDiscoveryMode(tau.Static),
		testutil.WithContractURI("https://internal/api"),
	)
	short, err := dimensions.ScoreDAuthority(context.Background(), shortX, w)
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

// TestDefaultAuthorityWeights_StructureAndSum verifies DefaultAuthorityWeights
// returns a non-zero struct whose weights sum to 1.0 (PRD §5.2).
func TestDefaultAuthorityWeights_StructureAndSum(t *testing.T) {
	t.Parallel()
	w := dimensions.DefaultAuthorityWeights()
	if w.ChainDepth == 0 && w.CrossOrg == 0 && w.HumanAnchor == 0 && w.DynamicResolution == 0 {
		t.Fatal("DefaultAuthorityWeights returned all-zero struct")
	}
	sum := w.ChainDepth + w.CrossOrg + w.HumanAnchor + w.DynamicResolution
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("DefaultAuthorityWeights sum = %f, want 1.0", sum)
	}
}

// TestDAuthority_ChainDepthSaturation covers probeChainDepth saturation at depth >= 4.
func TestDAuthority_ChainDepthSaturation(t *testing.T) {
	t.Parallel()
	x := newLongChainExchange()
	x.Initiator.DelegationDepth = 4
	w := authorityWeights()
	score, err := dimensions.ScoreDAuthority(context.Background(), x, w)
	if err != nil {
		t.Fatalf("ScoreDAuthority error: %v", err)
	}
	if score.Probes["A_chain_depth"] != 1.0 {
		t.Fatalf("expected A_chain_depth=1.0 at depth=4, got %f", score.Probes["A_chain_depth"])
	}
}

// TestDAuthority_CrossOrg_EmptyOrg covers probeCrossOrg returning 1 for empty org.
func TestDAuthority_CrossOrg_EmptyOrg(t *testing.T) {
	t.Parallel()
	x := newShortChainExchange()
	x.Initiator.Organization = ""
	x.Initiator.DelegationDepth = 0
	w := authorityWeights()
	score, err := dimensions.ScoreDAuthority(context.Background(), x, w)
	if err != nil {
		t.Fatalf("ScoreDAuthority error: %v", err)
	}
	if score.Probes["A_cross_org"] != 1.0 {
		t.Fatalf("expected A_cross_org=1.0 for empty org, got %f", score.Probes["A_cross_org"])
	}
}
