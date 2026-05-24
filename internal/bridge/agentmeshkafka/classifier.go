package agentmeshkafka

// I4Class categorizes an empirical decision against the I4 coherence invariant.
// The classification operates on a neutral EmpiricalDecision record so that
// the agentmeshkafka package stays isolated from tau/* (arch_test.go rule).
// The external test package agentmeshkafka_test bridges tau.Decision to
// EmpiricalDecision under the `empirical` build tag.
type I4Class int

const (
	// I4CoherentAccepted: D-SENS >= theta_sens OR D-INV < theta_inv, and the
	// decision is not a Refus(I4). Expected happy-path outcome.
	I4CoherentAccepted I4Class = iota

	// I4IncoherentRefused: true positive — D-SENS < theta_sens AND D-INV >=
	// theta_inv, and the decision is Refus("I4 — combinaison incohérente
	// détectée"). Guard fired correctly.
	I4IncoherentRefused

	// I4FalsePositive: D-SENS >= theta_sens AND D-INV < theta_inv but the
	// decision is Refus(I4). Should never occur; flagged for investigation.
	I4FalsePositive

	// I4FalseNegative: D-SENS < theta_sens AND D-INV >= theta_inv but the
	// decision is NOT Refus(I4). Guard failed silently.
	I4FalseNegative

	// OtherRefusal: the decision is a Refus with a non-I4 diagnostic (frontier,
	// I3, expired profile, etc.). I4 classification is not applicable.
	//
	// V1 aggregates all non-I4 refusals under this single bucket. A finer
	// ventilation into distinct causes (e.g. frontier refusal vs. I3 ontological
	// lock vs. expired profile) is planned for M4-bis once the empirical corpus
	// provides enough samples per cause to measure their individual rates.
	OtherRefusal

	// Unmodeled: none of the above categories applies. Appended to the
	// UnmodeledObservations list (anti-patron #4).
	Unmodeled
)

// EmpiricalDecision is the neutral projection of tau.Decision used for
// classification. It carries only the fields required by ClassifyI4 so that
// this package stays free of tau imports.
type EmpiricalDecision struct {
	// RegimeStr is the string form of tau.Regime: "deterministe", "probabiliste",
	// or "refus".
	RegimeStr string
	// Diagnostic mirrors tau.Decision.Diagnostic.
	Diagnostic string
	// DSensValue is the D-SENS dimension score at decision time.
	DSensValue float64
	// DInvariantValue is the D-INVARIANT dimension score at decision time.
	DInvariantValue float64
	// SensCoherence is the I4 theta_sens threshold in effect at decision time.
	SensCoherence float64
	// InvCoherence is the I4 theta_inv threshold in effect at decision time.
	InvCoherence float64
	// UnmodeledObservations mirrors tau.Trace.UnmodeledObservations.
	UnmodeledObservations []string
}

const diagI4 = "I4 — combinaison incohérente détectée"

// ClassifyI4 returns the I4Class for a single empirical decision.
// It uses the thresholds embedded in the EmpiricalDecision so that
// classification is reproducible without re-running the dispatcher.
func ClassifyI4(d EmpiricalDecision) I4Class {
	isRefusI4 := d.RegimeStr == "refus" && d.Diagnostic == diagI4
	isOtherRefus := d.RegimeStr == "refus" && d.Diagnostic != diagI4

	if isOtherRefus {
		return OtherRefusal
	}

	// I4 condition: inv >= theta_inv AND sens < theta_sens.
	i4condition := d.DInvariantValue >= d.InvCoherence && d.DSensValue < d.SensCoherence

	switch {
	case i4condition && isRefusI4:
		return I4IncoherentRefused
	case i4condition && !isRefusI4:
		return I4FalseNegative
	case !i4condition && isRefusI4:
		return I4FalsePositive
	case !i4condition && !isRefusI4:
		return I4CoherentAccepted
	}
	// The switch above exhausts all four boolean combinations; this line is
	// unreachable in practice (anti-patron #4 sentinel kept for future changes).
	return Unmodeled // unreachable: switch above exhausts all four boolean combinations
}

// EmpiricalI4Stats aggregates classification counts and derived metrics for
// one empirical campaign.
type EmpiricalI4Stats struct {
	Total               int `json:"total"`
	I4CoherentAccepted  int `json:"i4_coherent_accepted"`
	I4IncoherentRefused int `json:"i4_incoherent_refused"`
	I4FalsePositive     int `json:"i4_false_positive"`
	I4FalseNegative     int `json:"i4_false_negative"`
	OtherRefusal        int `json:"other_refusal"`
	Unmodeled           int `json:"unmodeled"`
	// Sensitivity = TP / (TP + FN). nil if denominator is zero.
	Sensitivity *float64 `json:"sensitivity,omitempty"`
	// Specificity = TN / (TN + FP). nil if denominator is zero.
	Specificity *float64 `json:"specificity,omitempty"`
}

// EmpiricalI4Summary aggregates a slice of EmpiricalDecision records into
// EmpiricalI4Stats. Sensitivity and specificity are set to -1 when the
// denominator is zero.
func EmpiricalI4Summary(decisions []EmpiricalDecision) EmpiricalI4Stats {
	s := EmpiricalI4Stats{Total: len(decisions)}
	for _, d := range decisions {
		switch ClassifyI4(d) {
		case I4CoherentAccepted:
			s.I4CoherentAccepted++
		case I4IncoherentRefused:
			s.I4IncoherentRefused++
		case I4FalsePositive:
			s.I4FalsePositive++
		case I4FalseNegative:
			s.I4FalseNegative++
		case OtherRefusal:
			s.OtherRefusal++
		default:
			// Reachable only if ClassifyI4 returns Unmodeled, which requires
			// the switch in ClassifyI4 to not match — currently unreachable.
			// Kept as anti-patron #4 sentinel (PRD §7.2.4).
			s.Unmodeled++
		}
	}

	tp := float64(s.I4IncoherentRefused)
	fn := float64(s.I4FalseNegative)
	tn := float64(s.I4CoherentAccepted)
	fp := float64(s.I4FalsePositive)

	if tp+fn > 0 {
		v := tp / (tp + fn)
		s.Sensitivity = &v
	}
	if tn+fp > 0 {
		v := tn / (tn + fp)
		s.Specificity = &v
	}
	return s
}
