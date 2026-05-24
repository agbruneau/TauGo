// internal/tau/frontier_test.go
package tau

import "testing"

func TestFrontierCheck_Inside_AllConditionsViolated(t *testing.T) {
	t.Parallel()
	f := FrontierCheck{
		UniversOuvert:       true,
		CompositionVariable: true,
		PairProbabiliste:    true,
		CoutNonBorne:        true,
	}
	if !f.Inside() {
		t.Fatal("expected Inside()=true when all 4 conditions are violated")
	}
}

func TestFrontierCheck_Inside_OneConditionMet_Refused(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		f    FrontierCheck
	}{
		{"allFalse", FrontierCheck{}}, // zero value — ordinary deterministic call, outside frontier
		{"universOuvert=false", FrontierCheck{
			UniversOuvert: false, CompositionVariable: true, PairProbabiliste: true, CoutNonBorne: true,
		}},
		{"compositionVariable=false", FrontierCheck{
			UniversOuvert: true, CompositionVariable: false, PairProbabiliste: true, CoutNonBorne: true,
		}},
		{"pairProbabiliste=false", FrontierCheck{
			UniversOuvert: true, CompositionVariable: true, PairProbabiliste: false, CoutNonBorne: true,
		}},
		{"coutNonBorne=false", FrontierCheck{
			UniversOuvert: true, CompositionVariable: true, PairProbabiliste: true, CoutNonBorne: false,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.f.Inside() {
				t.Fatalf("expected Inside()=false when %s (one classical condition still holds)", tc.name)
			}
		})
	}
}
