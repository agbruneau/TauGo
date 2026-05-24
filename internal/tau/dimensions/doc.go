// Package dimensions implements the three scored dimensions of the τ operator:
// D-SENS, D-AUTORITÉ, and D-INVARIANT (chap. III.8.4).
//
// Each dimension exposes a Score function that aggregates its probes using
// the calibrated weights from the active Profile. Dimensions are orthogonal
// in value (scores are independent); they are coupled only by the I4 coherence
// constraint enforced at the orchestration layer.
//
// Architecture rule: this package may import internal/tau and internal/bridge/llm
// but must NOT import internal/orchestration or any other dimension package.
package dimensions
