package calibration

// WeightCalibrationStrategy identifies the active weight-calibration algorithm.
// V1 is a deliberate pass-through: PRD §11.1 marks initial weights as
// "Hypothèse"; M5 scope is bounded by PRD §17 critère #10 (reproducibility,
// not quality). A real algorithm requires an ADR (M6+).
const WeightCalibrationStrategy = "v1-passthrough"

// WeightHook is the extension point for future weight-calibration strategies.
// A V2 implementation (gradient descent or Bayesian optimisation) can be
// injected by replacing the default hook without changing Calibrate's
// signature. The hook receives the same seed that Calibrate exposes so that
// stochastic strategies remain deterministic under test.
type WeightHook func(corpus []CorpusEntry, seed int64, base Weights) Weights

// defaultWeightHook is the V1 pass-through; it satisfies WeightHook.
func defaultWeightHook(_ []CorpusEntry, _ int64, base Weights) Weights { return base }

// CalibrateWeights applies the active weight-calibration strategy to corpus
// and returns the resulting Weights.
//
// V1: the function is intentionally an identity — it returns base unchanged.
// Rationale: the M2 DefaultProfile weights have not yet been challenged by
// sufficient empirical signal (M4 I4-report deferred). Mutating weights
// before that signal is available would violate PRD §11.1 ("Hypothèse")
// and PRD §17 critère #10 (reproducibility).
//
// V2 hook: pass a custom WeightHook to substitute a gradient-descent or
// Bayesian-optimisation strategy. The hook must not mutate base in place;
// it must return a new Weights value.
//
// Determinism: for a given (corpus, seed, base, hook), the output is
// identical across calls — enforced by V1's identity property and required
// of any V2 implementation.
func CalibrateWeights(corpus []CorpusEntry, seed int64, base Weights, hooks ...WeightHook) Weights {
	hook := defaultWeightHook
	if len(hooks) > 0 && hooks[0] != nil {
		hook = hooks[0]
	}
	return hook(corpus, seed, base)
}
