package invariants

import (
	"sort"

	"github.com/agbruneau/taugo/internal/tau"
)

// AngleMort names the blind-spots of a single layer of an agentic stack.
// V1 keeps them as opaque string identifiers; semantics are stack-specific.
type AngleMort []string

// Pile is a composed agentic stack: an ordered list of layers, each with
// its own set of blind-spots.
type Pile []AngleMort

// Aggregate returns M(π) — the union of blind-spots across the stack.
// Output is deterministically ordered (lex-sorted, deduplicated).
//
// PRD §6.1 I5: M(π) = |⋃ Aᵢ| with bounds (chap. III.8.6.3):
//   - len(M(π)) ≥ max(len(Aᵢ))   (lower bound: union contains the largest layer)
//   - len(M(π)) ≤ Σ len(Aᵢ)      (upper bound: no ex-nihilo creation; equality iff disjoint)
//
// Status: Probable (chap. III.8.5). PRD §6.1 promised V2 for the computation;
// M3.6 delivers it in V1: Aggregate is calculatory, not merely declared.
func Aggregate(p Pile) []string {
	seen := map[string]struct{}{}
	for _, layer := range p {
		for _, a := range layer {
			seen[a] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// BoundsHold reports whether Aggregate respects the two I5 bounds on p.
// Cheap finite check: never panics, never allocates beyond Aggregate's cost.
func BoundsHold(p Pile) bool {
	agg := Aggregate(p)
	if len(p) == 0 {
		return len(agg) == 0
	}
	maxLayer := 0
	sumLayers := 0
	for _, layer := range p {
		if len(layer) > maxLayer {
			maxLayer = len(layer)
		}
		sumLayers += len(layer)
	}
	return len(agg) >= maxLayer && len(agg) <= sumLayers
}

// EvaluateI5 returns the I5 verdict for (x, decision).
//
// V1 pipeline: no layer stack is reified in tau.Decision yet; that is a V2
// concern. EvaluateI5 therefore returns Held in the normal dispatcher pipeline —
// the invariant holds by construction because Aggregate is total and both bounds
// are proved algebraically (see BoundsHold).
//
// FuzzI5_CompositionConjonctive exercises BoundsHold directly on generated
// stacks without passing through EvaluateI5. This design avoids a speculative
// coupling between tau.Decision and the layer-stack concept before the Trace
// type is extended.
//
// Status: Probable (chap. III.8.5).
func EvaluateI5(_ tau.Exchange, _ tau.Decision) Status {
	// V1: stack not reified in Trace. Held by construction; fuzz verifies
	// BoundsHold on arbitrary generated stacks independently.
	return Held
}
