// Package tau implements the operator τ defined in chap. III.8 of
// `InteroperabiliteAgentique/Monographie.md` v2.4.3.
//
// It is the core of TauGo: it decides the call regime (Deterministe,
// Probabiliste, or Refus) at the agentic interoperability boundary,
// under the five invariants I1-I5.
//
// τ never predicts behavior; it never executes the call. The exclusive
// public entry point is Kernel.Decide.
package tau
