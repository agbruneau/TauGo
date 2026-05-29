package invariants

import "github.com/agbruneau/taugo/internal/tau"

// ResidualMagnitude names a magnitude whose locus of fixation is "pendant"
// (runtime). The four V1 candidates map one-to-one onto the four classical
// frontier conditions (chap. III.8.5.2).
type ResidualMagnitude string

const (
	// MagTargetResolution covers UniversOuvert + CompositionVariable.
	MagTargetResolution ResidualMagnitude = "target_resolution"
	// MagIntentMeaning covers PairProbabiliste.
	MagIntentMeaning ResidualMagnitude = "intent_meaning"
	// MagAuthorityChain covers CoutNonBorne.
	MagAuthorityChain ResidualMagnitude = "authority_chain"
	// MagSupportNegotiation covers UniversOuvert + CompositionVariable.
	MagSupportNegotiation ResidualMagnitude = "support_negotiation"
)

// Residu returns the migrating residue of x — the set of magnitudes whose
// locus of fixation is "pendant" (runtime rather than design time).
//
// PRD §6.1 I2 reformulation: Residu(x) := { g | t_fix(g) ≈ t_int }.
// V1 enumerates four candidates; a magnitude is in the residue iff the
// matching frontier condition is currently violated (i.e. is "pendant").
func Residu(x tau.Exchange) []ResidualMagnitude {
	out := make([]ResidualMagnitude, 0, 4)
	dynamic := x.Target.DiscoveryMode != tau.Static
	if dynamic {
		out = append(out, MagTargetResolution, MagSupportNegotiation)
	}
	if !x.Initiator.HumanInLoop {
		out = append(out, MagIntentMeaning)
	}
	if x.Initiator.DelegationDepth > 0 {
		out = append(out, MagAuthorityChain)
	}
	return out
}

// Recablage simulates an offline rewiring of x by removing the named
// residual magnitudes. Returns the resulting FrontierCheck.
//
// Removing a magnitude collapses its matching frontier condition to false.
// If Inside() remains true after removing the full residue, I2 is violated —
// the residue would not have been necessary to maintain the frontier.
func Recablage(x tau.Exchange, removed []string) tau.FrontierCheck {
	f := x.FrontierCheck()
	for _, name := range removed {
		switch ResidualMagnitude(name) {
		case MagTargetResolution, MagSupportNegotiation:
			f.UniversOuvert = false
			f.CompositionVariable = false
		case MagIntentMeaning:
			f.PairProbabiliste = false
		case MagAuthorityChain:
			f.CoutNonBorne = false
		}
	}
	return f
}

// EvaluateI2 returns the I2 verdict for (x, decision).
//
// Held iff Residu(x) is non-empty AND removing the full residue collapses
// Inside() to false (defining property of I2: the residue is necessary).
// NotApplicable for Refus("hors frontiere τ") — τ has not operated.
// Violated if Residu is empty or if removing it keeps Inside()==true.
//
// PRD §6.1 I2. Status: Confirmed by construction.
func EvaluateI2(x tau.Exchange, dec tau.Decision) Status {
	if dec.Regime == tau.Refus && dec.Diagnostic == tau.DiagFrontiereFranchie {
		return NotApplicable
	}
	r := Residu(x)
	if len(r) == 0 {
		return Violated
	}
	names := make([]string, len(r))
	for i, m := range r {
		names[i] = string(m)
	}
	if Recablage(x, names).Inside() {
		return Violated
	}
	return Held
}
