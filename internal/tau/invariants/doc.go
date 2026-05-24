// Package invariants encodes the five invariants I1-I5 of the τ operator
// (chap. III.8.5, PRD §6) as properties verifiable on a Decision already
// produced by the orchestration dispatcher.
//
// Architecture rule (gated by internal/arch_test.go): this package may import
// internal/tau but must NOT import internal/tau/dimensions (orthogonality
// constraint between scored dimensions and structural invariants),
// internal/orchestration (downstream layer), or any internal/bridge/*.
//
// The package exposes one evaluator per invariant plus an aggregating
// EvaluateInvariants entry point invoked by the dispatcher at step 8.
// Helpers (Conserve, Residu, Recablage, Aggregate) are exported so fuzz
// targets can drive them directly.
package invariants
