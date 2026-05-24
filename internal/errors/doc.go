// Package errors declares the typed error families (DispatchError,
// RefusError, CalibrationError) used across TauGo. It follows the
// FibGo pattern: structured errors, no panic except for internal
// invariant violations (cf. PRD §14.2).
package errors
