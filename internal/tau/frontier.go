// internal/tau/frontier.go
package tau

// FrontierCheck encodes the four classical conditions whose simultaneous
// violation defines the agentic boundary where τ applies (chap. III.8.3.2).
type FrontierCheck struct {
	UniversOuvert       bool `json:"univers_ouvert"`       // capabilities discovered at runtime
	CompositionVariable bool `json:"composition_variable"` // composition resolved at runtime
	PairProbabiliste    bool `json:"pair_probabiliste"`    // peer is a probabilistic reasoner (LLM or equivalent)
	CoutNonBorne        bool `json:"cout_non_borne"`       // error cost unbounded and/or irreversible
}

// Inside reports whether the exchange falls within τ's domain of validity.
// τ applies if and only if all four classical conditions are simultaneously
// violated; one condition still holding rules out τ application.
func (f FrontierCheck) Inside() bool {
	return f.UniversOuvert && f.CompositionVariable &&
		f.PairProbabiliste && f.CoutNonBorne
}
