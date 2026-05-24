package orchestration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

// highAuthorityExchange returns an exchange with D-AUTORITE likely above AuthBlock
// and no attestation: should trigger the ontological guard.
func highAuthorityExchange() tau.Exchange {
	return tau.Exchange{
		ID:                "x-high-authority",
		DiscoveredAt:      time.Now(),
		IntentDescription: "invoke external financial system without human oversight",
		Initiator: tau.Principal{
			ID:              "sub-agent-3",
			HumanInLoop:     false,
			Organization:    "",
			DelegationDepth: 5,
		},
		Target: tau.Capability{
			ID:            "external-fin",
			DiscoveryMode: tau.DynamicA2A,
			ContractURI:   "",
		},
		// AttestationInstitutionnelle intentionally nil
	}
}

// TestRefusOntologiqueDAUTORITE verifies that an exchange whose D-AUTORITE
// score reaches or exceeds AuthBlock without an institutional attestation
// is refused with diagnostic "I3 — verrou ontologique D-AUTORITE".
func TestRefusOntologiqueDAUTORITE(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
	x := highAuthorityExchange()
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Refus {
		t.Fatalf("regime = %v, want Refus (ontological guard should fire)", dec.Regime)
	}
	if dec.Diagnostic != "I3 — verrou ontologique D-AUTORITÉ" {
		t.Fatalf("diagnostic = %q, want \"I3 — verrou ontologique D-AUTORITÉ\"", dec.Diagnostic)
	}
}

// TestOntologicalGuardPassesWithAttestation verifies that the same high-authority
// exchange is NOT refused when an attestation is present.
func TestOntologicalGuardPassesWithAttestation(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
	x := highAuthorityExchange()
	x.AttestationInstitutionnelle = &tau.Attestation{
		Emetteur:   "IETF",
		Reference:  "draft-identity-delegation-00",
		Marqueur:   "Hypothese",
		AssertedAt: time.Now(),
	}
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With attestation, the ontological guard must NOT fire.
	if dec.Regime == tau.Refus && dec.Diagnostic == "I3 — verrou ontologique D-AUTORITÉ" {
		t.Fatal("ontological guard fired despite attestation being present")
	}
}
