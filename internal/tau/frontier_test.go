// internal/tau/frontier_test.go
package tau

import "testing"

// TestExchange_FrontierCheck_Reproduit verifies that Exchange.FrontierCheck()
// correctly derives the four classical conditions from the Exchange fields,
// covering the four rule branches documented in operator.go.
func TestExchange_FrontierCheck_Reproduit(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		x           Exchange
		wantInside  bool
		wantUnivers bool
		wantCompo   bool
		wantPair    bool
		wantCout    bool
	}{
		{
			name: "all_dynamic_noHuman_depth1",
			x: Exchange{
				Target:    Capability{DiscoveryMode: DynamicMCP},
				Initiator: Principal{HumanInLoop: false, DelegationDepth: 1},
			},
			wantInside: true, wantUnivers: true, wantCompo: true, wantPair: true, wantCout: true,
		},
		{
			name: "static_noHuman_depth1_notInside",
			x: Exchange{
				Target:    Capability{DiscoveryMode: Static},
				Initiator: Principal{HumanInLoop: false, DelegationDepth: 1},
			},
			wantInside: false, wantUnivers: false, wantCompo: false, wantPair: true, wantCout: true,
		},
		{
			name: "dynamic_humanInLoop_depth0_notInside",
			x: Exchange{
				Target:    Capability{DiscoveryMode: DynamicA2A},
				Initiator: Principal{HumanInLoop: true, DelegationDepth: 0},
			},
			wantInside: false, wantUnivers: true, wantCompo: true, wantPair: false, wantCout: false,
		},
		{
			name:       "zero_value_notInside",
			x:          Exchange{},
			wantInside: false, wantUnivers: false, wantCompo: false, wantPair: true, wantCout: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := tc.x.FrontierCheck()
			if f.UniversOuvert != tc.wantUnivers {
				t.Errorf("UniversOuvert = %v, want %v", f.UniversOuvert, tc.wantUnivers)
			}
			if f.CompositionVariable != tc.wantCompo {
				t.Errorf("CompositionVariable = %v, want %v", f.CompositionVariable, tc.wantCompo)
			}
			if f.PairProbabiliste != tc.wantPair {
				t.Errorf("PairProbabiliste = %v, want %v", f.PairProbabiliste, tc.wantPair)
			}
			if f.CoutNonBorne != tc.wantCout {
				t.Errorf("CoutNonBorne = %v, want %v", f.CoutNonBorne, tc.wantCout)
			}
			if f.Inside() != tc.wantInside {
				t.Errorf("Inside() = %v, want %v", f.Inside(), tc.wantInside)
			}
		})
	}
}

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
