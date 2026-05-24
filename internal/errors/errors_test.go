package errors_test

import (
	stderrors "errors"
	"testing"

	taugerrors "github.com/agbruneau/taugo/internal/errors"
)

func TestDispatchError_ErrorMessage(t *testing.T) {
	t.Parallel()

	t.Run("with_cause", func(t *testing.T) {
		t.Parallel()
		cause := stderrors.New("underlying")
		e := &taugerrors.DispatchError{Stage: 4, ExchangeID: "ex-1", Detail: "probe failed", Cause: cause}
		got := e.Error()
		want := "dispatch error at stage 4 (exchange=ex-1): probe failed: underlying"
		if got != want {
			t.Fatalf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without_cause", func(t *testing.T) {
		t.Parallel()
		e := &taugerrors.DispatchError{Stage: 2, ExchangeID: "ex-2", Detail: "auth score unavailable"}
		got := e.Error()
		want := "dispatch error at stage 2 (exchange=ex-2): auth score unavailable"
		if got != want {
			t.Fatalf("Error() = %q, want %q", got, want)
		}
	})
}

func TestDispatchError_UnwrapPropagation(t *testing.T) {
	t.Parallel()
	sentinel := stderrors.New("sentinel")
	e := &taugerrors.DispatchError{Stage: 1, Cause: sentinel}
	if !stderrors.Is(e, sentinel) {
		t.Fatal("errors.Is should find sentinel through DispatchError.Unwrap()")
	}
}

func TestRefusError_ErrorMessage(t *testing.T) {
	t.Parallel()
	e := &taugerrors.RefusError{
		Stage:      1,
		ExchangeID: "ex-r",
		Diagnostic: "hors frontière τ",
	}
	got := e.Error()
	want := "refus at stage 1 (exchange=ex-r): hors frontière τ"
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestCalibrationError_ErrorMessage(t *testing.T) {
	t.Parallel()

	t.Run("with_cause", func(t *testing.T) {
		t.Parallel()
		cause := stderrors.New("json: unexpected token")
		e := &taugerrors.CalibrationError{ProfileVersion: "0.1.0", Cause: cause}
		got := e.Error()
		want := "calibration error (profile=0.1.0): json: unexpected token"
		if got != want {
			t.Fatalf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without_cause", func(t *testing.T) {
		t.Parallel()
		e := &taugerrors.CalibrationError{ProfileVersion: "M3-default"}
		got := e.Error()
		want := "calibration error (profile=M3-default)"
		if got != want {
			t.Fatalf("Error() = %q, want %q", got, want)
		}
	})
}

func TestCalibrationError_UnwrapPropagation(t *testing.T) {
	t.Parallel()
	sentinel := stderrors.New("io: eof")
	e := &taugerrors.CalibrationError{ProfileVersion: "v1", Cause: sentinel}
	if !stderrors.Is(e, sentinel) {
		t.Fatal("errors.Is should find sentinel through CalibrationError.Unwrap()")
	}
}

func TestSentinels_IdentifieesViaErrorsIs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sentinel error
	}{
		{"ErrFrontiereFranchie", taugerrors.ErrFrontiereFranchie},
		{"ErrPeremptionProfile", taugerrors.ErrPeremptionProfile},
		{"ErrIncoherenceI4", taugerrors.ErrIncoherenceI4},
		{"ErrVerrouOntologique", taugerrors.ErrVerrouOntologique},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			wrapped := &taugerrors.DispatchError{Stage: 0, Cause: tc.sentinel}
			if !stderrors.Is(wrapped, tc.sentinel) {
				t.Fatalf("errors.Is(%v) returned false for sentinel %q", wrapped, tc.sentinel)
			}
		})
	}
}
