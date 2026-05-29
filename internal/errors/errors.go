// Package errors declares the typed error families (DispatchError,
// RefusError, CalibrationError) used across TauGo. It follows the
// FibGo pattern: structured errors, no panic except for internal
// invariant violations (cf. PRD §14.2).
//
// Three families:
//   - DispatchError : erreur de dispatch (étapes 1-8, hors refus)
//   - RefusError    : refus de premier rang (étapes 1, 2, 3, 5)
//   - CalibrationError : erreur de calibration / chargement profil
//
// Les sentinels exportés (ErrFrontiereFranchie, etc.) correspondent
// exactement aux chaînes Diagnostic utilisées dans le Dispatcher.
// Ils sont utilisables via errors.Is / errors.As.
//
// ADR-0009 (Types d'erreurs typées) — accepté 2026-05-24.
package errors

import (
	stderrors "errors"
	"fmt"
)

// DispatchError signals a failure during one of the 8 steps of the
// dispatcher pipeline (PRD §10) that does NOT escalate to Refus.
type DispatchError struct {
	Stage      int    // 1..8, ou 0 si pré-étape
	Cause      error  // erreur sous-jacente (peut être nil)
	ExchangeID string // ID de l'Exchange concerné (peut être vide)
	Detail     string // texte libre additionnel
}

func (e *DispatchError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("dispatch error at stage %d (exchange=%s): %s: %v",
			e.Stage, e.ExchangeID, e.Detail, e.Cause)
	}
	return fmt.Sprintf("dispatch error at stage %d (exchange=%s): %s",
		e.Stage, e.ExchangeID, e.Detail)
}

// Unwrap exposes the underlying cause for errors.Is / errors.As chaining.
func (e *DispatchError) Unwrap() error { return e.Cause }

// RefusError signals a Refus de premier rang (PRD §7.3).
// The Diagnostic field matches the sentinel strings below verbatim, and the
// Is method (see below) makes errors.Is(refus, ErrFrontiereFranchie) succeed
// when the Diagnostic equals that sentinel's message.
type RefusError struct {
	Stage      int    // étape où le refus est émis (1, 2, 3 ou 5)
	Diagnostic string // sentinel diagnostic, ex. "hors frontière τ"
	ExchangeID string
}

func (e *RefusError) Error() string {
	return fmt.Sprintf("refus at stage %d (exchange=%s): %s",
		e.Stage, e.ExchangeID, e.Diagnostic)
}

// Is reports whether target is a sentinel matching this refusal's Diagnostic.
// A RefusError carries no wrapped cause, so it cannot rely on Unwrap; instead
// it matches a sentinel by comparing the Diagnostic field to the sentinel's
// message verbatim. The package sentinels (ErrFrontiereFranchie, …) hold the
// exact Diagnostic strings emitted by the dispatcher, so errors.Is(refus,
// ErrFrontiereFranchie) is true whenever refus.Diagnostic == its message.
func (e *RefusError) Is(target error) bool {
	return target != nil && e.Diagnostic == target.Error()
}

// CalibrationError signals a failure during calibration corpus load,
// profile parse, drift detection, etc.
type CalibrationError struct {
	ProfileVersion string
	Cause          error // peut être nil
}

func (e *CalibrationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("calibration error (profile=%s): %v",
			e.ProfileVersion, e.Cause)
	}
	return fmt.Sprintf("calibration error (profile=%s)", e.ProfileVersion)
}

// Unwrap exposes the underlying cause for errors.Is / errors.As chaining.
func (e *CalibrationError) Unwrap() error { return e.Cause }

// Sentinel errors — utilisables via errors.Is.
// Les valeurs correspondent exactement aux champs Diagnostic produits
// par le Dispatcher (dispatcher.go, steps 1, 2, 3, 5).
var (
	ErrFrontiereFranchie = stderrors.New("hors frontière τ")
	ErrPeremptionProfile = stderrors.New("profil périmé — veille requise")
	ErrIncoherenceI4     = stderrors.New("I4 — combinaison incohérente détectée")
	ErrVerrouOntologique = stderrors.New("I3 — verrou ontologique D-AUTORITÉ")
)
