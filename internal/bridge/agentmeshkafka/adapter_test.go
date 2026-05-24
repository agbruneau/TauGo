package agentmeshkafka_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
)

func TestAdapter_InterfaceShape(t *testing.T) {
	t.Parallel()
	// Compile-time guarantee: any *FileAdapter is an Adapter.
	var _ agentmeshkafka.Adapter = (*agentmeshkafka.FileAdapter)(nil)
}

func TestAgentMeshExchange_FieldsPresent(t *testing.T) {
	t.Parallel()
	// Smoke: zero-value DTO does not panic; key fields are zero-init.
	x := agentmeshkafka.AgentMeshExchange{
		ID:                "e-0",
		IntentDescription: "noop",
		DiscoveredAt:      time.Unix(0, 0).UTC(),
	}
	if x.ID == "" || x.IntentDescription == "" {
		t.Fatal("zero-value AgentMeshExchange dropped fields unexpectedly")
	}
	if x.DiscoveredAt.IsZero() {
		t.Fatal("DiscoveredAt should not be zero for Unix(0,0)")
	}
}

func TestAdapter_StreamSignature(t *testing.T) {
	t.Parallel()
	// Compile-time: an Adapter.Stream returns the documented channel types.
	// Use a concrete *FileAdapter (which is always non-nil when allocated) to
	// exercise the type assertion without a dead nil-branch.
	//
	// The zero-value &FileAdapter{} has no open file; Stream with an empty
	// filename path calls os.Open(""), which fails silently on the error
	// channel and closes both channels immediately. This is intentional: the
	// test verifies the channel-type contract, not the streaming behavior.
	var a agentmeshkafka.Adapter = &agentmeshkafka.FileAdapter{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exCh, errCh := a.Stream(ctx, []string{"topic-1"})
	var (
		_ <-chan agentmeshkafka.AgentMeshExchange = exCh
		_ <-chan error                            = errCh
	)
}

func TestAgentMeshExchange_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	orig := agentmeshkafka.AgentMeshExchange{
		ID:                "e-roundtrip",
		IntentDescription: "test exchange",
		DiscoveredAt:      now,
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              "agent-a",
			HumanInLoop:     true,
			Organization:    "org-1",
			DelegationDepth: 2,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "cap-x",
			DiscoveryMode: "static",
			ContractURI:   "https://example.com/contract",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur:   "notaire",
			Reference:  "REF-001",
			Marqueur:   "valid",
			AssertedAt: now,
		},
		Context:         map[string]any{"key": "value"},
		SourceTopic:     "traces",
		SourceOffset:    42,
		SourcePartition: 1,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got agentmeshkafka.AgentMeshExchange
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got.ID != orig.ID {
		t.Errorf("ID: got %q want %q", got.ID, orig.ID)
	}
	if got.IntentDescription != orig.IntentDescription {
		t.Errorf("IntentDescription: got %q want %q", got.IntentDescription, orig.IntentDescription)
	}
	if !got.DiscoveredAt.Equal(orig.DiscoveredAt) {
		t.Errorf("DiscoveredAt: got %v want %v", got.DiscoveredAt, orig.DiscoveredAt)
	}
	if got.Initiator.ID != orig.Initiator.ID {
		t.Errorf("Initiator.ID: got %q want %q", got.Initiator.ID, orig.Initiator.ID)
	}
	if got.Initiator.DelegationDepth != orig.Initiator.DelegationDepth {
		t.Errorf("Initiator.DelegationDepth: got %d want %d", got.Initiator.DelegationDepth, orig.Initiator.DelegationDepth)
	}
	if got.Target.DiscoveryMode != orig.Target.DiscoveryMode {
		t.Errorf("Target.DiscoveryMode: got %q want %q", got.Target.DiscoveryMode, orig.Target.DiscoveryMode)
	}
	if got.AttestationInstitutionnelle == nil {
		t.Fatal("AttestationInstitutionnelle: got nil want non-nil")
	}
	if got.AttestationInstitutionnelle.Reference != orig.AttestationInstitutionnelle.Reference {
		t.Errorf("Attestation.Reference: got %q want %q",
			got.AttestationInstitutionnelle.Reference, orig.AttestationInstitutionnelle.Reference)
	}
	if got.SourceOffset != orig.SourceOffset {
		t.Errorf("SourceOffset: got %d want %d", got.SourceOffset, orig.SourceOffset)
	}
}

func TestAgentMeshExchange_ZeroValue(t *testing.T) {
	t.Parallel()
	// Zero-value struct must marshal and unmarshal without error.
	var x agentmeshkafka.AgentMeshExchange
	data, err := json.Marshal(x)
	if err != nil {
		t.Fatalf("marshal zero-value failed: %v", err)
	}
	var got agentmeshkafka.AgentMeshExchange
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal zero-value failed: %v", err)
	}
	if got.AttestationInstitutionnelle != nil {
		t.Error("zero-value AttestationInstitutionnelle should be nil after round-trip")
	}
}

func TestAgentMeshExchange_NoAttestationOmitted(t *testing.T) {
	t.Parallel()
	// When AttestationInstitutionnelle is nil, the field must be absent from JSON.
	x := agentmeshkafka.AgentMeshExchange{ID: "e-no-att"}
	data, err := json.Marshal(x)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if json.Valid(data) {
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		if _, present := m["attestation_institutionnelle"]; present {
			t.Error("attestation_institutionnelle key must be absent when nil (omitempty)")
		}
	}
}

func TestFileAdapter_ImplementsAdapter(t *testing.T) {
	t.Parallel()
	fa := &agentmeshkafka.FileAdapter{}
	ctx := context.Background()
	exCh, errCh := fa.Stream(ctx, nil)
	// Stub closes both channels immediately.
	for range exCh {
	}
	for range errCh {
	}
	if err := fa.Close(); err != nil {
		t.Errorf("Close: got %v want nil", err)
	}
}
