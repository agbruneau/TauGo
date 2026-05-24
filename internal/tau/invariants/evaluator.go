package invariants

import (
	"github.com/agbruneau/taugo/internal/tau"
)

// Status is the verdict for one invariant on one decision.
type Status int

const (
	// StatusUnknown is the zero value; never returned by Evaluate functions.
	StatusUnknown Status = iota
	// Held means the invariant tested true for this decision.
	Held
	// Violated means the invariant was tested and failed.
	Violated
	// NotApplicable means the invariant is not testable in this context
	// (e.g. Refus upstream of the conditions the invariant constrains).
	NotApplicable
)

// String returns the lowercase verbatim form for trace diagnostics.
func (s Status) String() string {
	switch s {
	case Held:
		return "held"
	case Violated:
		return "violated"
	case NotApplicable:
		return "not_applicable"
	default:
		return "unknown"
	}
}

// Statuses bundles the verdicts of all five invariants for one decision.
type Statuses struct {
	I1 Status `json:"i1"`
	I2 Status `json:"i2"`
	I3 Status `json:"i3"`
	I4 Status `json:"i4"`
	I5 Status `json:"i5"`
}

// AnyViolated reports whether at least one invariant was violated.
func (s Statuses) AnyViolated() bool {
	return s.I1 == Violated || s.I2 == Violated || s.I3 == Violated ||
		s.I4 == Violated || s.I5 == Violated
}

// Summary returns a stable list of short diagnostic strings, one per
// violated invariant, in numerical order. Empty when no violation.
// Format: "I<N> — <one-line reason>".
func (s Statuses) Summary() []string {
	out := make([]string, 0, 5)
	if s.I1 == Violated {
		out = append(out, "I1 — conservation rompue (grandeur supprimée par τ)")
	}
	if s.I2 == Violated {
		out = append(out, "I2 — résidu migrant vidé sans perte de condition de frontière")
	}
	if s.I3 == Violated {
		out = append(out, "I3 — asymétrie D-AUTORITÉ contournée ou profil périmé")
	}
	if s.I4 == Violated {
		out = append(out, "I4 — combinaison (s, i) incohérente non refusée")
	}
	if s.I5 == Violated {
		out = append(out, "I5 — agrégation M(π) hors bornes")
	}
	return out
}

// EvaluateInvariants runs the five evaluators on (x, decision) and returns
// the bundled Statuses. The orchestration dispatcher invokes this at step 8
// of the PRD §10 pseudo-algorithm. The function MUST NOT panic; an invariant
// internal-state sentinel is the only acceptable panic source (calque FibGo).
func EvaluateInvariants(x tau.Exchange, dec tau.Decision) Statuses {
	return Statuses{
		I1: EvaluateI1(x, dec),
		I2: EvaluateI2(x, dec),
		I3: EvaluateI3(x, dec),
		I4: EvaluateI4(x, dec),
		I5: EvaluateI5(x, dec),
	}
}

// === STUBS TEMPORAIRES — supprimés en M3.5 à M3.6 ===
// EvaluateI3 est implémenté dans i3_authority_asymmetry.go (M3.4).
// Les stubs I4 et I5 renvoient NotApplicable jusqu'à M3.5/M3.6.

func EvaluateI4(_ tau.Exchange, _ tau.Decision) Status { return NotApplicable }
func EvaluateI5(_ tau.Exchange, _ tau.Decision) Status { return NotApplicable }
