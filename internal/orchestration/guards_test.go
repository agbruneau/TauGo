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

// coherentInvariantHighSensExchange returns an exchange where D-INVARIANT is high
// but D-SENS is also high — coherent, must NOT trigger I4.
// Attestation present because DelegationDepth=2 + empty-org heuristic raises
// D-AUTORITE to 0.875 >= AuthBlock(0.85); attestation bypasses I3.
func coherentInvariantHighSensExchange() tau.Exchange {
	return tau.Exchange{
		ID:                "x-coherent-i4",
		DiscoveredAt:      time.Now(),
		IntentDescription: "dynamically negotiate and execute plan",
		AttestationInstitutionnelle: &tau.Attestation{
			Emetteur:   "IETF",
			Reference:  "draft-coherent-scope",
			Marqueur:   "Hypothese",
			AssertedAt: time.Now(),
		},
		Initiator: tau.Principal{
			ID:              "llm-orchestrator",
			HumanInLoop:     false,
			Organization:    "org-c",
			DelegationDepth: 2,
		},
		Target: tau.Capability{
			ID:            "adaptive-tool",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
		Context: map[string]any{
			"event_registry":       true,
			"idempotency_key_mode": "derived",
			"capability_mediation": true,
		},
	}
}

// incoherentExchange returns an exchange where D-INVARIANT is high (dynamic
// support) but D-SENS is low (static contract, no IntentDescription) — incoherent
// per I4, must Refus.
// Attestation present because D-AUTORITE reaches 0.875 >= AuthBlock(0.85);
// attestation bypasses I3 so the I4 guard is the active gate.
//
// D-INVARIANT calculation: I_event_registry=1(0.30) + I_idempotency_derived=1(0.25)
// + I_capability_mediation=1(0.25) + I_enumerated_plan=0(0.20, ContractURI set) = 0.80
// D-SENS calculation: S_contract=0(0.35, ContractURI set) + S_runtime_resolve=0(0.30,
// no intent) + S_capability_discovery=1(0.20, DynamicMCP) + S_reasoner_intent~0.039
// (stub FNV-1a for "" * 0.15) = ~0.239 < SensCoherence(0.50).
func incoherentExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-incoherent-i4",
		DiscoveredAt: time.Now(),
		// No IntentDescription => S_runtime_resolve = 0.
		AttestationInstitutionnelle: &tau.Attestation{
			Emetteur:   "IETF",
			Reference:  "draft-delegation-scope",
			Marqueur:   "Hypothese",
			AssertedAt: time.Now(),
		},
		Initiator: tau.Principal{
			ID:              "system-scheduler",
			HumanInLoop:     false,
			Organization:    "org-d",
			DelegationDepth: 2,
		},
		Target: tau.Capability{
			ID:            "static-svc",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1", // static contract => S_contract = 0
		},
		Context: map[string]any{
			"event_registry":       true,
			"idempotency_key_mode": "derived",
			"capability_mediation": true,
			// enumerated_plan absent => I_enumerated_plan = 1 but ContractURI set overrides to 0
		},
	}
}

// TestI4_IncoherenceDetectee verifies that an exchange with D-INVARIANT >=
// InvCoherence (0.50) and D-SENS < SensCoherence (0.50) is refused with
// diagnostic "I4 — combinaison incohérente détectée".
func TestI4_IncoherenceDetectee(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
	x := incoherentExchange()
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Refus {
		t.Logf("D-SENS score may not be low enough with current fixtures; regime = %v", dec.Regime)
		t.Fatalf("expected Refus (I4 guard), got %v", dec.Regime)
	}
	if dec.Diagnostic != "I4 — combinaison incohérente détectée" {
		t.Fatalf("diagnostic = %q, want \"I4 — combinaison incohérente détectée\"", dec.Diagnostic)
	}
}

// TestI4_CoherentCombinationAccepted verifies that a coherent exchange
// (high D-INVARIANT AND high D-SENS) is not refused by the I4 guard.
func TestI4_CoherentCombinationAccepted(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
	x := coherentInvariantHighSensExchange()
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime == tau.Refus && dec.Diagnostic == "I4 — combinaison incohérente détectée" {
		t.Fatal("I4 guard fired on a coherent exchange")
	}
}
