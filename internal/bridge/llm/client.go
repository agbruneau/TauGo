package llm

import "context"

// Client is the narrow interface that TauGo consumes from any LLM.
// No concrete LLM is embedded; the production implementation is injected
// at the app layer (cf. PRD §12.2).
type Client interface {
	// Fingerprint identifies model + version + parameters frozen.
	// Used for profile invalidation (PRD §11.4).
	Fingerprint() string

	// Interpret returns an interpretation score [0, 1] for a given
	// intent description. Used by the S_reasoner_intent probe of
	// D-SENS (PRD §5.1). Must be deterministic under fixed parameters
	// (temperature 0).
	Interpret(ctx context.Context, intent string) (float64, error)
}
