package testutil_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/testutil"
)

func TestBuildExchange_Defaults(t *testing.T) {
	t.Parallel()
	x := testutil.BuildExchange()

	if x.ID != "ex-test" {
		t.Errorf("ID = %q, want %q", x.ID, "ex-test")
	}
	if x.IntentDescription != "test intent" {
		t.Errorf("IntentDescription = %q, want %q", x.IntentDescription, "test intent")
	}
	if x.Initiator.HumanInLoop {
		t.Error("HumanInLoop = true, want false")
	}
	if x.Initiator.DelegationDepth != 1 {
		t.Errorf("DelegationDepth = %d, want 1", x.Initiator.DelegationDepth)
	}
	if x.Target.DiscoveryMode != tau.DynamicMCP {
		t.Errorf("DiscoveryMode = %v, want DynamicMCP", x.Target.DiscoveryMode)
	}
	want := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	if !x.DiscoveredAt.Equal(want) {
		t.Errorf("DiscoveredAt = %v, want %v", x.DiscoveredAt, want)
	}
	if x.AttestationInstitutionnelle != nil {
		t.Error("AttestationInstitutionnelle should be nil by default")
	}
	if x.Context != nil {
		t.Error("Context should be nil by default")
	}
}

func TestBuildExchange_AppliqueOptions(t *testing.T) {
	t.Parallel()
	att := &tau.Attestation{
		Emetteur:  "org-a",
		Reference: "REF-001",
		Marqueur:  "officiel",
	}
	x := testutil.BuildExchange(
		testutil.WithID("ex-custom"),
		testutil.WithIntentDescription("custom intent"),
		testutil.WithDiscoveryMode(tau.Static),
		testutil.WithContractURI("https://api.example.com/v1"),
		testutil.WithHumanInLoop(true),
		testutil.WithDelegationDepth(0),
		testutil.WithAttestation(att),
		testutil.WithContext("key", "val"),
	)

	if x.ID != "ex-custom" {
		t.Errorf("ID = %q, want %q", x.ID, "ex-custom")
	}
	if x.IntentDescription != "custom intent" {
		t.Errorf("IntentDescription = %q, want %q", x.IntentDescription, "custom intent")
	}
	if x.Target.DiscoveryMode != tau.Static {
		t.Errorf("DiscoveryMode = %v, want Static", x.Target.DiscoveryMode)
	}
	if x.Target.ContractURI != "https://api.example.com/v1" {
		t.Errorf("ContractURI = %q, want non-empty", x.Target.ContractURI)
	}
	if !x.Initiator.HumanInLoop {
		t.Error("HumanInLoop = false, want true")
	}
	if x.Initiator.DelegationDepth != 0 {
		t.Errorf("DelegationDepth = %d, want 0", x.Initiator.DelegationDepth)
	}
	if x.AttestationInstitutionnelle != att {
		t.Error("AttestationInstitutionnelle not set")
	}
	if v, ok := x.Context["key"]; !ok || v != "val" {
		t.Errorf("Context[\"key\"] = %v, want \"val\"", v)
	}
}
