package app

import (
	"context"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
	"github.com/agbruneau/taugo/internal/tau"
)

// ToTauExchange converts a neutral AgentMeshExchange into a typed tau.Exchange.
// This conversion is hosted at the app layer (cf. ADR-0005) because arch_test.go
// forbids bridge/agentmeshkafka → tau imports.
//
// DiscoveryMode mapping falls back to DynamicMCP on unknown strings —
// conservative dynamic-side default prevents a silent frontier bypass
// (anti-patron #2, #4).
func ToTauExchange(x agentmeshkafka.AgentMeshExchange) tau.Exchange {
	out := tau.Exchange{
		ID:                x.ID,
		IntentDescription: x.IntentDescription,
		DiscoveredAt:      x.DiscoveredAt,
		Initiator: tau.Principal{
			ID:              x.Initiator.ID,
			HumanInLoop:     x.Initiator.HumanInLoop,
			Organization:    x.Initiator.Organization,
			DelegationDepth: x.Initiator.DelegationDepth,
		},
		Target: tau.Capability{
			ID:            x.Target.ID,
			DiscoveryMode: discoveryModeFromString(x.Target.DiscoveryMode),
			ContractURI:   x.Target.ContractURI,
		},
		Context: x.Context,
	}
	if x.AttestationInstitutionnelle != nil {
		out.AttestationInstitutionnelle = &tau.Attestation{
			Emetteur:   x.AttestationInstitutionnelle.Emetteur,
			Reference:  x.AttestationInstitutionnelle.Reference,
			Marqueur:   x.AttestationInstitutionnelle.Marqueur,
			AssertedAt: x.AttestationInstitutionnelle.AssertedAt,
		}
	}
	return out
}

// discoveryModeFromString maps the free-form DiscoveryMode string from the
// AgentMesh DTO to the typed tau.DiscoveryMode. Unknown values fall back to
// DynamicMCP (dynamic-side) rather than Static to avoid treating an unknown
// frontier as definitively outside τ.
func discoveryModeFromString(s string) tau.DiscoveryMode {
	switch s {
	case "", "static":
		return tau.Static
	case "dynamic_mcp":
		return tau.DynamicMCP
	case "dynamic_a2a":
		return tau.DynamicA2A
	case "dynamic_agntcy":
		return tau.DynamicAGNTCY
	default:
		return tau.DynamicMCP
	}
}

// StreamAsTauExchanges adapts a bridge Adapter to the kernel's typed input.
// It starts adapter.Stream and transforms each AgentMeshExchange to tau.Exchange
// in a goroutine. Errors from the adapter are forwarded verbatim. Both output
// channels are closed when the source stream drains or ctx is canceled.
func StreamAsTauExchanges(
	ctx context.Context,
	adapter agentmeshkafka.Adapter,
	topics []string,
) (exchanges <-chan tau.Exchange, errc <-chan error) {
	src, errs := adapter.Stream(ctx, topics)
	out := make(chan tau.Exchange)
	go func() {
		defer close(out)
		for x := range src {
			select {
			case out <- ToTauExchange(x):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errs
}
