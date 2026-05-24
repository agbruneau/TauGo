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
		{"universOuvert=false", FrontierCheck{false, true, true, true}},
		{"compositionVariable=false", FrontierCheck{true, false, true, true}},
		{"pairProbabiliste=false", FrontierCheck{true, true, false, true}},
		{"coutNonBorne=false", FrontierCheck{true, true, true, false}},
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
