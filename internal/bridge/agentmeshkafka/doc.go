// Package agentmeshkafka adapts AgentMeshKafka traces into a neutral DTO
// (AgentMeshExchange) for downstream consumption by the app layer.
//
// Architecture rule (gated by internal/arch_test.go): this package must NOT
// import internal/tau/* or internal/orchestration/*. Conversion to
// tau.Exchange lives in internal/app/agentmesh.go (cf. ADR-0005).
//
// V1 ships a file-backed mock adapter (FileAdapter) that reads JSONL
// fixtures. A real Kafka adapter is deferred to M4-bis pending the
// stability of agbruneau/AgentMeshKafka (PRD §18 risque #1).
package agentmeshkafka
