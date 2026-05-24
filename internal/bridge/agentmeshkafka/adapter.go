package agentmeshkafka

import (
	"context"
	"time"
)

// AgentMeshExchange is the neutral DTO produced by the adapter. Field names
// mirror the canonical tau.Exchange; conversion lives in internal/app/agentmesh.go
// (cf. ADR-0005). The two types are intentionally distinct to preserve the
// arch_test.go étanchéité (bridge → tau direct: forbidden).
type AgentMeshExchange struct {
	ID                          string                `json:"id"`
	IntentDescription           string                `json:"intent_description"`
	DiscoveredAt                time.Time             `json:"discovered_at"`
	Initiator                   AgentMeshPrincipal    `json:"initiator"`
	Target                      AgentMeshCapability   `json:"target"`
	AttestationInstitutionnelle *AgentMeshAttestation `json:"attestation_institutionnelle,omitempty"`
	Context                     map[string]any        `json:"context,omitempty"`
	// Sourcing metadata — neutral to the τ kernel.
	SourceTopic     string `json:"source_topic,omitempty"`
	SourceOffset    int64  `json:"source_offset,omitempty"`
	SourcePartition int32  `json:"source_partition,omitempty"`
}

// AgentMeshPrincipal mirrors tau.Principal without importing it.
type AgentMeshPrincipal struct {
	ID              string `json:"id"`
	HumanInLoop     bool   `json:"human_in_loop"`
	Organization    string `json:"organization"`
	DelegationDepth int    `json:"delegation_depth"`
}

// AgentMeshCapability mirrors tau.Capability. DiscoveryMode is a free-form
// string (e.g. "static" | "dynamic_mcp" | "dynamic_a2a" | "dynamic_agntcy");
// the app-layer converter maps it to the typed tau.DiscoveryMode.
type AgentMeshCapability struct {
	ID            string `json:"id"`
	DiscoveryMode string `json:"discovery_mode"`
	ContractURI   string `json:"contract_uri,omitempty"`
}

// AgentMeshAttestation mirrors tau.Attestation without importing it.
type AgentMeshAttestation struct {
	Emetteur   string    `json:"emetteur"`
	Reference  string    `json:"reference"`
	Marqueur   string    `json:"marqueur"`
	AssertedAt time.Time `json:"asserted_at"`
}

// Adapter streams AgentMesh traces. Two-method ISP-conforming interface.
// V1 ships FileAdapter (JSONL); a real Kafka adapter lands in M4-bis.
type Adapter interface {
	// Stream opens a flow of exchanges over the given topics. Returns two
	// channels: exchanges and non-fatal errors. The exchanges channel is
	// closed cleanly on ctx.Done() or after Close().
	Stream(ctx context.Context, topics []string) (<-chan AgentMeshExchange, <-chan error)

	// Close releases resources. Idempotent. Blocks until in-flight drain.
	Close() error
}
