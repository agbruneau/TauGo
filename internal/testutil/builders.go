// Package testutil contains shared test helpers (fixtures, factories,
// assertions) used across TauGo's _test.go files.
//
// Use BuildExchange for fluent construction of tau.Exchange values:
//
//	x := testutil.BuildExchange(
//	    testutil.WithID("ex-1"),
//	    testutil.WithHumanInLoop(true),
//	    testutil.WithDelegationDepth(0),
//	    testutil.WithDiscoveryMode(tau.DynamicMCP),
//	)
package testutil

import (
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// ExchangeOption mutates an Exchange under construction.
type ExchangeOption func(*tau.Exchange)

// WithID sets the Exchange.ID.
func WithID(id string) ExchangeOption {
	return func(x *tau.Exchange) { x.ID = id }
}

// WithIntentDescription sets Exchange.IntentDescription.
func WithIntentDescription(s string) ExchangeOption {
	return func(x *tau.Exchange) { x.IntentDescription = s }
}

// WithDiscoveryMode sets Exchange.Target.DiscoveryMode.
func WithDiscoveryMode(m tau.DiscoveryMode) ExchangeOption {
	return func(x *tau.Exchange) { x.Target.DiscoveryMode = m }
}

// WithContractURI sets Exchange.Target.ContractURI.
func WithContractURI(uri string) ExchangeOption {
	return func(x *tau.Exchange) { x.Target.ContractURI = uri }
}

// WithHumanInLoop sets Exchange.Initiator.HumanInLoop.
func WithHumanInLoop(v bool) ExchangeOption {
	return func(x *tau.Exchange) { x.Initiator.HumanInLoop = v }
}

// WithDelegationDepth sets Exchange.Initiator.DelegationDepth.
func WithDelegationDepth(d int) ExchangeOption {
	return func(x *tau.Exchange) { x.Initiator.DelegationDepth = d }
}

// WithAttestation sets Exchange.AttestationInstitutionnelle.
// Pass nil to leave it unset.
func WithAttestation(a *tau.Attestation) ExchangeOption {
	return func(x *tau.Exchange) { x.AttestationInstitutionnelle = a }
}

// WithContext merges a single key/value pair into Exchange.Context.
func WithContext(key string, value any) ExchangeOption {
	return func(x *tau.Exchange) {
		if x.Context == nil {
			x.Context = make(map[string]any)
		}
		x.Context[key] = value
	}
}

// BuildExchange constructs a tau.Exchange with sensible defaults and
// applies each option in order.
//
// Defaults: ID="ex-test", HumanInLoop=false, DelegationDepth=1,
// DiscoveryMode=DynamicMCP, IntentDescription="test intent",
// DiscoveredAt=2026-05-24T12:00:00Z, no Attestation, no Context.
//
// The defaults model the "inside-frontier" case (all 4 classical
// conditions violated) so tests can focus on a single variable.
func BuildExchange(opts ...ExchangeOption) tau.Exchange {
	x := tau.Exchange{
		ID:                "ex-test",
		IntentDescription: "test intent",
		DiscoveredAt:      time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
		Initiator: tau.Principal{
			ID:              "agent-test",
			HumanInLoop:     false,
			Organization:    "org-test",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "cap-test",
			DiscoveryMode: tau.DynamicMCP,
		},
	}
	for _, o := range opts {
		o(&x)
	}
	return x
}
