package dimensions

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// AuthorityWeights holds the calibrated weights for the four D-AUTORITE probes.
// Initial values from PRD §5.2: {0.25, 0.25, 0.25, 0.25} (equal weighting).
type AuthorityWeights struct {
	ChainDepth        float64 // weight for A_chain_depth
	CrossOrg          float64 // weight for A_cross_org
	HumanAnchor       float64 // weight for A_human_anchor (inverted probe)
	DynamicResolution float64 // weight for A_dynamic_resolution
}

// DefaultAuthorityWeights returns the initial equal weights from PRD §5.2.
// Status: Hypothesis — to be corroborated on AgentMeshKafka traces in M4.
func DefaultAuthorityWeights() AuthorityWeights {
	return AuthorityWeights{
		ChainDepth:        0.25,
		CrossOrg:          0.25,
		HumanAnchor:       0.25,
		DynamicResolution: 0.25,
	}
}

// ScoreDAuthority computes the D-AUTORITE dimension score for exchange x.
// Returns a Score with Value in [0,1] and all probe values populated.
// Note: the attestation check (ontological guard) is NOT performed here;
// it is enforced at the orchestration layer (PRD §4.4, step 2 of dispatch).
func ScoreDAuthority(_ context.Context, x tau.Exchange, w AuthorityWeights) (Score, error) {
	aChain := probeChainDepth(x)
	aCross := probeCrossOrg(x)
	aHuman := probeHumanAnchor(x)
	aDynamic := probeDynamicResolution(x)

	value := w.ChainDepth*aChain +
		w.CrossOrg*aCross +
		w.HumanAnchor*aHuman +
		w.DynamicResolution*aDynamic

	return Score{
		Value: clamp01(value),
		Probes: map[string]float64{
			"A_chain_depth":        aChain,
			"A_cross_org":          aCross,
			"A_human_anchor":       aHuman,
			"A_dynamic_resolution": aDynamic,
		},
		Weights: map[string]float64{
			"A_chain_depth":        w.ChainDepth,
			"A_cross_org":          w.CrossOrg,
			"A_human_anchor":       w.HumanAnchor,
			"A_dynamic_resolution": w.DynamicResolution,
		},
		ComputedAt: time.Now(),
	}, nil
}

// probeChainDepth (A_chain_depth) — delegation chain depth. Returns a
// normalized value in [0,1] using a saturation function: depth 0 = 0.0,
// depth 1 = 0.25, depth 2 = 0.50, depth >= 4 = 1.0.
func probeChainDepth(x tau.Exchange) float64 {
	d := x.Initiator.DelegationDepth
	if d <= 0 {
		return 0.0
	}
	if d >= 4 {
		return 1.0
	}
	return float64(d) / 4.0
}

// probeCrossOrg (A_cross_org) — whether the exchange crosses an organizational
// boundary. Returns 1 if Initiator.Organization is empty (unknown = assumed
// cross-org) or DelegationDepth > 1 (implying multi-hop cross-org delegation).
func probeCrossOrg(x tau.Exchange) float64 {
	if x.Initiator.Organization == "" {
		return 1.0
	}
	if x.Initiator.DelegationDepth > 1 {
		return 1.0
	}
	return 0.0
}

// probeHumanAnchor (A_human_anchor) — inverted probe: human in the loop
// reduces the D-AUTORITE score (short chain, anchored authority = pole 0).
// Returns 0 if HumanInLoop is true (human anchor present), 1 if absent.
func probeHumanAnchor(x tau.Exchange) float64 {
	if x.Initiator.HumanInLoop {
		return 0.0
	}
	return 1.0
}

// probeDynamicResolution (A_dynamic_resolution) — authority resolved at
// runtime rather than pre-wired. Returns 1 if DiscoveryMode != Static
// (capability identity itself resolved dynamically = authority chain unknown
// at design time).
func probeDynamicResolution(x tau.Exchange) float64 {
	if x.Target.DiscoveryMode == tau.Static {
		return 0.0
	}
	return 1.0
}
