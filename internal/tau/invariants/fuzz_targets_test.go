package invariants_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// makeFuzzExchange builds a deterministic Exchange from fuzz seed inputs.
// The mapping is total: every input combination produces a well-formed Exchange.
func makeFuzzExchange(id, intent string, discoveryMode, delegationDepth uint8, humanInLoop bool) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: intent,
		DiscoveredAt:      time.Unix(int64(discoveryMode)*1000, 0).UTC(),
		Initiator: tau.Principal{
			ID:              id + "-init",
			HumanInLoop:     humanInLoop,
			Organization:    "org-fuzz",
			DelegationDepth: int(delegationDepth % 8),
		},
		Target: tau.Capability{
			ID:            id + "-cap",
			DiscoveryMode: tau.DiscoveryMode(int(discoveryMode) % 4),
			ContractURI:   "",
		},
	}
}

// FuzzI1_Conservation exercises Conserve and EvaluateI1. Property:
// when drift==0 the ExchangeID is preserved and EvaluateI1 must not be
// Violated; when drift!=0 the trace ID diverges and EvaluateI1 must be Held
// only if the Decision is a Refus("hors frontière τ"), otherwise Violated.
func FuzzI1_Conservation(f *testing.F) {
	// seed 1: nominal — matching IDs, non-Refus
	f.Add("e-seed-1", "compute", uint8(1), uint8(0), false, int8(0))
	// seed 2: boundary — empty intent
	f.Add("e-seed-2", "", uint8(0), uint8(0), true, int8(0))
	// seed 3: degenerate — drift injected
	f.Add("x", "intent", uint8(3), uint8(5), false, int8(1))

	f.Fuzz(func(t *testing.T, id, intent string, mode, depth uint8, human bool, drift int8) {
		if len(id) > 256 || len(intent) > 4000 {
			return
		}
		x := makeFuzzExchange(id, intent, mode, depth, human)
		traceID := x.ID
		if drift != 0 {
			traceID = x.ID + "X"
		}
		dec := tau.Decision{
			Regime: tau.Deterministe,
			Trace:  tau.Trace{ExchangeID: traceID},
		}
		got := invariants.EvaluateI1(x, dec)
		if drift == 0 && got == invariants.Violated {
			t.Fatalf("EvaluateI1 Violated on identity-preserving exchange: id=%q trace=%q", x.ID, traceID)
		}
		if drift != 0 && got == invariants.Held {
			t.Fatalf("EvaluateI1 Held despite trace drift: id=%q trace=%q", x.ID, traceID)
		}
	})
}

// FuzzI2_Irreductibilite exercises Residu and Recablage. Property: for any
// exchange that is inside the frontier (Recablage(x, nil).Inside()), Residu(x)
// must be non-empty AND removing all magnitudes must collapse Inside() to false.
func FuzzI2_Irreductibilite(f *testing.F) {
	// seed 1: dynamic MCP, agent initiator
	f.Add("e-seed-i2", "discover", uint8(2), uint8(2), false)
	// seed 2: dynamic A2A, delegation depth 1
	f.Add("e-dyn-a2a", "negotiate", uint8(3), uint8(1), false)
	// seed 3: static target (may not be inside frontier)
	f.Add("e-static", "query", uint8(0), uint8(0), true)

	f.Fuzz(func(t *testing.T, id, intent string, mode, depth uint8, human bool) {
		if len(id) > 256 || len(intent) > 4000 {
			return
		}
		x := makeFuzzExchange(id, intent, mode, depth, human)

		// Check frontier before evaluating (mirrors plan: skip if not inside).
		insideBefore := invariants.Recablage(x, nil).Inside()
		if !insideBefore {
			return // Not inside frontier; I2 is NotApplicable.
		}

		r := invariants.Residu(x)
		if len(r) == 0 {
			t.Fatalf("Residu empty for inside-frontier exchange: %+v", x)
		}
		names := make([]string, len(r))
		for i, m := range r {
			names[i] = string(m)
		}
		if invariants.Recablage(x, names).Inside() {
			t.Fatalf("Recablage with full residue kept Inside()=true: x=%+v residue=%v", x, names)
		}
	})
}

// fuzzRefTime is the fixed clock used by FuzzI3 to make the expiry check
// deterministic. It is in the past relative to any future DateRevision.
var fuzzRefTime = time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

// FuzzI3_AsymetrieAutorite exercises EvaluateI3WithClock. Two properties are
// checked:
//  1. No attestation + tauScore >= authBlock (authBlock > 0) + Probabiliste
//     regime + non-expired profile => Violated.
//  2. Expired profile not refused => Violated.
func FuzzI3_AsymetrieAutorite(f *testing.F) {
	// seed 1: nominal — tauScore below authBlock, future revision, no attest
	f.Add(uint8(50), uint8(85), false, int64(1796601600))
	// seed 2: bypass condition — score above authBlock
	f.Add(uint8(95), uint8(85), false, int64(1796601600))
	// seed 3: with attestation — should be Held
	f.Add(uint8(95), uint8(85), true, int64(1796601600))

	f.Fuzz(func(t *testing.T, tauMilli, authMilli uint8, withAttestation bool, dateUnix int64) {
		tauScore := float64(tauMilli) / 100.0
		authBlock := float64(authMilli) / 100.0
		revisionDate := time.Unix(dateUnix, 0).UTC()

		x := tau.Exchange{ID: "x-i3-fuzz"}
		if withAttestation {
			x.AttestationInstitutionnelle = &tau.Attestation{
				Emetteur:  "ietf",
				Reference: "draft-x",
			}
		}
		dec := tau.Decision{
			Regime:       tau.Probabiliste,
			DateRevision: revisionDate,
			Trace: tau.Trace{
				ExchangeID: x.ID,
				TauScore:   tauScore,
				Thresholds: tau.TraceThresholds{AuthBlock: authBlock},
			},
		}

		got := invariants.EvaluateI3WithClock(x, dec, fuzzRefTime)

		// Property 1: expired profile not refused => Violated.
		profileExpired := invariants.IsProfileExpired(dec, fuzzRefTime)
		notExpiryRefused := !(dec.Regime == tau.Refus && dec.Diagnostic == "profil périmé — veille requise")
		if profileExpired && notExpiryRefused && got != invariants.Violated {
			t.Fatalf("EvaluateI3WithClock=%v want Violated: profile expired, not refused", got)
		}

		// Property 2: Probabiliste + no attest + score >= authBlock > 0 + non-expired => Violated.
		if !profileExpired &&
			!withAttestation &&
			authBlock > 0 &&
			tauScore >= authBlock &&
			dec.Regime == tau.Probabiliste &&
			got != invariants.Violated {
			t.Fatalf("EvaluateI3WithClock=%v want Violated: no attest, tau>=auth_block, non-expired", got)
		}
	})
}

// FuzzI4_CoherenceContrainte exercises Incoherent directly. Property:
// Incoherent(s, sT, i, iT) is exactly equivalent to (i >= iT && s < sT).
func FuzzI4_CoherenceContrainte(f *testing.F) {
	// seed 1: classic violating pair
	f.Add(uint8(10), uint8(50), uint8(70), uint8(50))
	// seed 2: boundary equality — strict < on s means NOT incoherent at s==sT
	f.Add(uint8(50), uint8(50), uint8(50), uint8(50))
	// seed 3: degenerate zero thresholds
	f.Add(uint8(0), uint8(0), uint8(0), uint8(0))

	f.Fuzz(func(t *testing.T, sMilli, sTMilli, iMilli, iTMilli uint8) {
		s := float64(sMilli) / 100.0
		sT := float64(sTMilli) / 100.0
		i := float64(iMilli) / 100.0
		iT := float64(iTMilli) / 100.0

		// IsIncoherent signature: (sensValue, invValue, sensCoherence, invCoherence)
		got := invariants.IsIncoherent(s, i, sT, iT)
		want := i >= iT && s < sT
		if got != want {
			t.Fatalf("IsIncoherent(%v,%v,%v,%v)=%v want %v", s, i, sT, iT, got, want)
		}
	})
}

// FuzzI5_CompositionConjonctive exercises Aggregate and BoundsHold. Property:
// BoundsHold must be true for every well-formed Pile, regardless of layer
// count, duplicate distribution, or element size.
func FuzzI5_CompositionConjonctive(f *testing.F) {
	// seed 1: two layers with overlap
	f.Add([]byte{1, 2, 3, 0, 1, 4, 0, 5})
	// seed 2: single layer
	f.Add([]byte{1, 2, 3})
	// seed 3: empty (no bytes -> empty pile)
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 4096 {
			return
		}
		// Decode raw into a Pile: 0x00 separates layers; non-zero bytes become
		// single-byte string identifiers within the current layer.
		var pile invariants.Pile
		var current invariants.AngleMort
		for _, b := range raw {
			if b == 0 {
				if len(current) > 0 {
					pile = append(pile, current)
					current = nil
				}
				continue
			}
			current = append(current, string([]byte{b}))
		}
		if len(current) > 0 {
			pile = append(pile, current)
		}
		if len(pile) > 50 {
			return // filter oversized stacks
		}
		if !invariants.BoundsHold(pile) {
			t.Fatalf("BoundsHold failed on pile %v (aggregate=%v)", pile, invariants.Aggregate(pile))
		}
	})
}
