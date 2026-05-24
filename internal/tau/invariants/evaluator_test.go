package invariants_test

import (
	"testing"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

func TestStatuses_AnyViolated_ZeroValue(t *testing.T) {
	t.Parallel()
	var s invariants.Statuses
	if s.AnyViolated() {
		t.Fatal("zero-value Statuses (all StatusUnknown) reported AnyViolated=true")
	}
}

func TestStatuses_AnyViolated_OneSet(t *testing.T) {
	t.Parallel()
	s := invariants.Statuses{I3: invariants.Violated}
	if !s.AnyViolated() {
		t.Fatal("Statuses{I3:Violated} reported AnyViolated=false")
	}
}

func TestStatuses_Summary_OrderedAndShort(t *testing.T) {
	t.Parallel()
	s := invariants.Statuses{
		I1: invariants.Violated,
		I3: invariants.Violated,
		I5: invariants.Violated,
	}
	got := s.Summary()
	if len(got) != 3 {
		t.Fatalf("Summary len = %d, want 3", len(got))
	}
	// Numerical order: I1 before I3 before I5
	if got[0][:2] != "I1" || got[1][:2] != "I3" || got[2][:2] != "I5" {
		t.Fatalf("Summary order broken: %v", got)
	}
}

func TestEvaluateInvariants_NoPanicOnZeroExchange(t *testing.T) {
	t.Parallel()
	// Calque FibGo: invariant cassé = panic ; ici, le sentinel interne
	// ne doit jamais se déclencher sur une entrée vide bien formée.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("EvaluateInvariants panicked on zero Exchange/Decision: %v", r)
		}
	}()
	_ = invariants.EvaluateInvariants(tau.Exchange{}, tau.Decision{})
}
