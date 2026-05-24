// Package main implements the generate-corpus command, a deterministic
// generator of synthetic AgentMeshExchange fixtures.
//
// It emits a JSONL file of AgentMeshExchange records, deterministically seeded
// so the output is byte-identical across runs (calque FibGo discipline,
// PRD §11). Three distribution profiles are supported: balanced, i4-heavy,
// and refus-heavy.
//
// Used by M4 (synthetic empirical campaign — Régime B contingency,
// PRD §18 risque #1).
package main
