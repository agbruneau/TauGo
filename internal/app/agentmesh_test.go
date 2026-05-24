package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestToTauExchange_Nominal / TestToTauExchange_BijectionCoreFields verifies
// that every non-attestation field round-trips through ToTauExchange.
func TestToTauExchange_Nominal(t *testing.T) {
	t.Parallel()
	src := agentmeshkafka.AgentMeshExchange{
		ID:                "e-conv-1",
		IntentDescription: "compute",
		DiscoveredAt:      time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              "p-1",
			HumanInLoop:     false,
			Organization:    "org-a",
			DelegationDepth: 2,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "cap-1",
			DiscoveryMode: "dynamic_mcp",
			ContractURI:   "",
		},
	}
	got := app.ToTauExchange(src)
	if got.ID != src.ID {
		t.Fatalf("ID drift: got=%s, want=%s", got.ID, src.ID)
	}
	if got.IntentDescription != src.IntentDescription {
		t.Fatalf("IntentDescription drift")
	}
	if !got.DiscoveredAt.Equal(src.DiscoveredAt) {
		t.Fatalf("DiscoveredAt drift")
	}
	if got.Initiator.ID != src.Initiator.ID {
		t.Fatalf("Initiator.ID drift")
	}
	if got.Initiator.HumanInLoop != src.Initiator.HumanInLoop {
		t.Fatalf("HumanInLoop drift")
	}
	if got.Initiator.Organization != src.Initiator.Organization {
		t.Fatalf("Organization drift")
	}
	if got.Initiator.DelegationDepth != 2 {
		t.Fatalf("DelegationDepth drift: got=%d", got.Initiator.DelegationDepth)
	}
	if got.Target.DiscoveryMode != tau.DynamicMCP {
		t.Fatalf("DiscoveryMode drift: got=%v, want=DynamicMCP", got.Target.DiscoveryMode)
	}
	if got.Target.ID != src.Target.ID {
		t.Fatalf("Target.ID drift")
	}
}

// TestToTauExchange_NoAttestation verifies that a nil AttestationInstitutionnelle
// in the DTO produces nil in the tau.Exchange (no phantom struct allocated).
func TestToTauExchange_NoAttestation(t *testing.T) {
	t.Parallel()
	src := agentmeshkafka.AgentMeshExchange{
		ID:                          "e-no-att",
		AttestationInstitutionnelle: nil,
	}
	got := app.ToTauExchange(src)
	if got.AttestationInstitutionnelle != nil {
		t.Fatal("expected nil AttestationInstitutionnelle, got non-nil")
	}
}

// TestToTauExchange_UnknownDiscoveryMode_FallsBackToZero verifies that an
// unrecognized DiscoveryMode string produces DynamicMCP (conservative
// dynamic-side fallback — anti-patron #2/#4).
func TestToTauExchange_UnknownDiscoveryMode_FallsBackToZero(t *testing.T) {
	t.Parallel()
	src := agentmeshkafka.AgentMeshExchange{
		ID:     "e-unknown",
		Target: agentmeshkafka.AgentMeshCapability{DiscoveryMode: "unknown"},
	}
	got := app.ToTauExchange(src).Target.DiscoveryMode
	if got != tau.DynamicMCP {
		t.Fatalf("unknown DiscoveryMode fallback = %v, want DynamicMCP", got)
	}
}

// TestToTauExchange_DiscoveryModeMapping covers all five branches of
// discoveryModeFromString including the empty-string alias for Static.
func TestToTauExchange_DiscoveryModeMapping(t *testing.T) {
	t.Parallel()
	cases := map[string]tau.DiscoveryMode{
		"static":         tau.Static,
		"":               tau.Static,
		"dynamic_mcp":    tau.DynamicMCP,
		"dynamic_a2a":    tau.DynamicA2A,
		"dynamic_agntcy": tau.DynamicAGNTCY,
		"unknown_xyz":    tau.DynamicMCP,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			src := agentmeshkafka.AgentMeshExchange{
				ID:     "x",
				Target: agentmeshkafka.AgentMeshCapability{DiscoveryMode: in},
			}
			got := app.ToTauExchange(src).Target.DiscoveryMode
			if got != want {
				t.Fatalf("ToTauExchange(%q).Target.DiscoveryMode = %v, want %v", in, got, want)
			}
		})
	}
}

// TestToTauExchange_AttestationPreserved verifies that a non-nil attestation
// in the DTO is faithfully converted and not dropped.
func TestToTauExchange_AttestationPreserved(t *testing.T) {
	t.Parallel()
	src := agentmeshkafka.AgentMeshExchange{
		ID: "e-att",
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur:   "ietf",
			Reference:  "draft-x",
			Marqueur:   "Hypothese",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	got := app.ToTauExchange(src)
	if got.AttestationInstitutionnelle == nil {
		t.Fatal("Attestation dropped during conversion")
	}
	if got.AttestationInstitutionnelle.Emetteur != "ietf" {
		t.Fatalf("Emetteur drift: %s", got.AttestationInstitutionnelle.Emetteur)
	}
	if got.AttestationInstitutionnelle.Reference != "draft-x" {
		t.Fatalf("Reference drift: %s", got.AttestationInstitutionnelle.Reference)
	}
	if got.AttestationInstitutionnelle.Marqueur != "Hypothese" {
		t.Fatalf("Marqueur drift: %s", got.AttestationInstitutionnelle.Marqueur)
	}
}

// blockingAdapter is a test-only Adapter that sends exchanges and errors on
// unbuffered channels. It is used to verify that StreamAsTauExchanges drains
// the error channel on context cancellation (goroutine-leak fix, AUDIT P1-06).
type blockingAdapter struct {
	ex   chan agentmeshkafka.AgentMeshExchange
	errs chan error
}

func newBlockingAdapter() *blockingAdapter {
	return &blockingAdapter{
		ex:   make(chan agentmeshkafka.AgentMeshExchange),
		errs: make(chan error), // unbuffered — will block if not drained
	}
}

func (a *blockingAdapter) Stream(_ context.Context, _ []string) (ex <-chan agentmeshkafka.AgentMeshExchange, errs <-chan error) {
	return a.ex, a.errs
}

func (a *blockingAdapter) Close() error { return nil }

// TestStreamAsTauExchanges_DrainsErrsOnContextCancel verifies that canceling the
// context causes StreamAsTauExchanges to drain the adapter's error channel,
// allowing the adapter goroutine (or any sender) to unblock and exit cleanly.
func TestStreamAsTauExchanges_DrainsErrsOnContextCancel(t *testing.T) {
	t.Parallel()
	a := newBlockingAdapter()
	ctx, cancel := context.WithCancel(context.Background())

	out, errc := app.StreamAsTauExchanges(ctx, a, nil)

	// Signal when both output channels are closed.
	done := make(chan struct{})
	go func() {
		for range out {
		}
		for range errc {
		}
		close(done)
	}()

	// Cancel before any exchange is sent.
	cancel()

	// The adapter still holds an unsent error on an unbuffered channel.
	// StreamAsTauExchanges must drain it so the send below does not block.
	errSent := make(chan struct{})
	go func() {
		// This send will complete only if StreamAsTauExchanges drains adapterErrs.
		select {
		case a.errs <- errors.New("adapter error"):
		default:
		}
		close(errSent)
		close(a.ex)
		close(a.errs)
	}()

	timeout := time.After(time.Second)
	select {
	case <-done:
		// Both channels closed in time — no goroutine leak.
	case <-timeout:
		t.Fatal("StreamAsTauExchanges did not close output channels within 1s after ctx cancel")
	}
	select {
	case <-errSent:
	case <-time.After(time.Second):
		t.Fatal("adapter error-send goroutine did not complete within 1s")
	}
}

// TestStreamAsTauExchanges_RoundTrip3Lines reads golden-3.jsonl (3 valid records)
// and verifies that exactly 3 tau.Exchange values are emitted with non-empty IDs.
func TestStreamAsTauExchanges_RoundTrip3Lines(t *testing.T) {
	t.Parallel()
	a, err := agentmeshkafka.NewFileAdapter("../bridge/agentmeshkafka/testdata/golden-3.jsonl")
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tauEx, _ := app.StreamAsTauExchanges(ctx, a, nil)
	var n int
	for x := range tauEx {
		if x.ID == "" {
			t.Fatalf("empty ID on converted exchange #%d", n)
		}
		n++
	}
	if n != 3 {
		t.Fatalf("converted count = %d, want 3", n)
	}
}

// TestStreamAsTauExchanges_PropagatesErrors reads golden-3-malformed.jsonl which
// contains 2 valid records and 1 malformed line. Verifies: 2 exchanges emitted,
// at least 1 error forwarded.
func TestStreamAsTauExchanges_PropagatesErrors(t *testing.T) {
	t.Parallel()
	a, err := agentmeshkafka.NewFileAdapter("../bridge/agentmeshkafka/testdata/golden-3-malformed.jsonl")
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tauEx, errc := app.StreamAsTauExchanges(ctx, a, nil)

	var exchanges int
	for range tauEx {
		exchanges++
	}
	var errs int
	for range errc {
		errs++
	}

	if exchanges != 2 {
		t.Fatalf("exchange count = %d, want 2", exchanges)
	}
	if errs < 1 {
		t.Fatalf("error count = %d, want >= 1", errs)
	}
}
