package agentmeshkafka_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
)

func TestClassifyI4(t *testing.T) {
	t.Parallel()

	// theta_sens=0.50, theta_inv=0.50 mirrors DefaultThresholds().
	const sensThreshold = 0.50
	const invThreshold = 0.50

	cases := []struct {
		name string
		d    agentmeshkafka.EmpiricalDecision
		want agentmeshkafka.I4Class
	}{
		{
			name: "coherent_accepted_deterministe",
			d: agentmeshkafka.EmpiricalDecision{
				RegimeStr:       "deterministe",
				DSensValue:      0.60, // >= theta_sens
				DInvariantValue: 0.30, // < theta_inv
				SensCoherence:   sensThreshold,
				InvCoherence:    invThreshold,
			},
			want: agentmeshkafka.I4CoherentAccepted,
		},
		{
			name: "coherent_accepted_probabiliste",
			d: agentmeshkafka.EmpiricalDecision{
				RegimeStr:       "probabiliste",
				DSensValue:      0.70,
				DInvariantValue: 0.20,
				SensCoherence:   sensThreshold,
				InvCoherence:    invThreshold,
			},
			want: agentmeshkafka.I4CoherentAccepted,
		},
		{
			name: "i4_incoherent_refused_true_positive",
			d: agentmeshkafka.EmpiricalDecision{
				RegimeStr:       "refus",
				Diagnostic:      "I4 — combinaison incohérente détectée",
				DSensValue:      0.30, // < theta_sens
				DInvariantValue: 0.60, // >= theta_inv
				SensCoherence:   sensThreshold,
				InvCoherence:    invThreshold,
			},
			want: agentmeshkafka.I4IncoherentRefused,
		},
		{
			name: "i4_false_positive_refus_i4_but_condition_not_met",
			d: agentmeshkafka.EmpiricalDecision{
				RegimeStr:       "refus",
				Diagnostic:      "I4 — combinaison incohérente détectée",
				DSensValue:      0.70, // >= theta_sens — condition NOT met
				DInvariantValue: 0.60, // >= theta_inv
				SensCoherence:   sensThreshold,
				InvCoherence:    invThreshold,
			},
			want: agentmeshkafka.I4FalsePositive,
		},
		{
			name: "i4_false_negative_condition_met_but_accepted",
			d: agentmeshkafka.EmpiricalDecision{
				RegimeStr:       "deterministe",
				DSensValue:      0.20, // < theta_sens
				DInvariantValue: 0.80, // >= theta_inv — condition met but NOT refused
				SensCoherence:   sensThreshold,
				InvCoherence:    invThreshold,
			},
			want: agentmeshkafka.I4FalseNegative,
		},
		{
			name: "other_refusal_frontier",
			d: agentmeshkafka.EmpiricalDecision{
				RegimeStr:     "refus",
				Diagnostic:    "hors frontière τ",
				SensCoherence: sensThreshold,
				InvCoherence:  invThreshold,
			},
			want: agentmeshkafka.OtherRefusal,
		},
		{
			name: "other_refusal_i3",
			d: agentmeshkafka.EmpiricalDecision{
				RegimeStr:     "refus",
				Diagnostic:    "I3 — verrou ontologique D-AUTORITÉ",
				SensCoherence: sensThreshold,
				InvCoherence:  invThreshold,
			},
			want: agentmeshkafka.OtherRefusal,
		},
		{
			name: "coherent_accepted_on_threshold_boundary_inv_below",
			d: agentmeshkafka.EmpiricalDecision{
				RegimeStr:       "deterministe",
				DSensValue:      0.40, // < theta_sens
				DInvariantValue: 0.49, // strictly below theta_inv — no I4 condition
				SensCoherence:   sensThreshold,
				InvCoherence:    invThreshold,
			},
			want: agentmeshkafka.I4CoherentAccepted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := agentmeshkafka.ClassifyI4(tc.d)
			if got != tc.want {
				t.Errorf("ClassifyI4(%+v) = %d, want %d", tc.d, got, tc.want)
			}
		})
	}
}

// TestEmpiricalI4Summary_UnmodeledCounted verifies that EmpiricalI4Summary
// keeps Unmodeled at zero for the full set of modeled decision types.
// The Unmodeled bucket and the default: branch in EmpiricalI4Summary are
// dead code under the current ClassifyI4 implementation: its switch exhausts
// all four boolean combinations, making the trailing "return Unmodeled"
// unreachable. The test documents this expectation explicitly (anti-patron #4).
func TestEmpiricalI4Summary_UnmodeledCounted(t *testing.T) {
	t.Parallel()

	const sensThreshold = 0.50
	const invThreshold = 0.50

	// One representative decision for each of the five reachable classes.
	decisions := []agentmeshkafka.EmpiricalDecision{
		// I4CoherentAccepted
		{RegimeStr: "deterministe", DSensValue: 0.60, DInvariantValue: 0.30, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		// I4IncoherentRefused
		{RegimeStr: "refus", Diagnostic: "I4 — combinaison incohérente détectée", DSensValue: 0.30, DInvariantValue: 0.60, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		// I4FalsePositive
		{RegimeStr: "refus", Diagnostic: "I4 — combinaison incohérente détectée", DSensValue: 0.70, DInvariantValue: 0.60, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		// I4FalseNegative
		{RegimeStr: "deterministe", DSensValue: 0.20, DInvariantValue: 0.80, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		// OtherRefusal
		{RegimeStr: "refus", Diagnostic: "hors frontière τ", SensCoherence: sensThreshold, InvCoherence: invThreshold},
	}

	s := agentmeshkafka.EmpiricalI4Summary(decisions)

	if s.Total != 5 {
		t.Errorf("Total = %d, want 5", s.Total)
	}
	if s.Unmodeled != 0 {
		t.Errorf("Unmodeled = %d, want 0: every modeled decision must be classified", s.Unmodeled)
	}
}

func TestEmpiricalI4Summary(t *testing.T) {
	t.Parallel()

	const sensThreshold = 0.50
	const invThreshold = 0.50

	decisions := []agentmeshkafka.EmpiricalDecision{
		// 2 true positives (I4IncoherentRefused)
		{RegimeStr: "refus", Diagnostic: "I4 — combinaison incohérente détectée", DSensValue: 0.30, DInvariantValue: 0.60, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		{RegimeStr: "refus", Diagnostic: "I4 — combinaison incohérente détectée", DSensValue: 0.20, DInvariantValue: 0.70, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		// 3 true negatives (I4CoherentAccepted)
		{RegimeStr: "deterministe", DSensValue: 0.60, DInvariantValue: 0.30, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		{RegimeStr: "probabiliste", DSensValue: 0.70, DInvariantValue: 0.20, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		{RegimeStr: "deterministe", DSensValue: 0.80, DInvariantValue: 0.10, SensCoherence: sensThreshold, InvCoherence: invThreshold},
		// 1 other refusal
		{RegimeStr: "refus", Diagnostic: "hors frontière τ", SensCoherence: sensThreshold, InvCoherence: invThreshold},
	}

	s := agentmeshkafka.EmpiricalI4Summary(decisions)

	if s.Total != 6 {
		t.Errorf("Total = %d, want 6", s.Total)
	}
	if s.I4IncoherentRefused != 2 {
		t.Errorf("I4IncoherentRefused = %d, want 2", s.I4IncoherentRefused)
	}
	if s.I4CoherentAccepted != 3 {
		t.Errorf("I4CoherentAccepted = %d, want 3", s.I4CoherentAccepted)
	}
	if s.OtherRefusal != 1 {
		t.Errorf("OtherRefusal = %d, want 1", s.OtherRefusal)
	}
	// Sensitivity = 2 / (2+0) = 1.0
	if s.Sensitivity == nil || *s.Sensitivity != 1.0 {
		t.Errorf("Sensitivity = %v, want 1.0", s.Sensitivity)
	}
	// Specificity = 3 / (3+0) = 1.0
	if s.Specificity == nil || *s.Specificity != 1.0 {
		t.Errorf("Specificity = %v, want 1.0", s.Specificity)
	}
}

// TestEmpiricalI4Stats_SensitivityOmitemptySiNil vérifie que le champ Sensitivity
// est absent du JSON lorsque nil, et présent lorsqu'il a une valeur.
func TestEmpiricalI4Stats_SensitivityOmitemptySiNil(t *testing.T) {
	t.Parallel()

	t.Run("nil_absent_du_json", func(t *testing.T) {
		t.Parallel()
		// Zéro décisions → dénominateur nul → Sensitivity == nil.
		s := agentmeshkafka.EmpiricalI4Summary(nil)
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(data), `"sensitivity"`) {
			t.Errorf("JSON ne doit pas contenir \"sensitivity\" quand nil, obtenu : %s", data)
		}
	})

	t.Run("valeur_presente_dans_json", func(t *testing.T) {
		t.Parallel()
		const sens = 0.50
		const inv = 0.50
		// 1 TP, 0 FN → sensitivity = 1.0.
		decisions := []agentmeshkafka.EmpiricalDecision{
			{
				RegimeStr:       "refus",
				Diagnostic:      "I4 — combinaison incohérente détectée",
				DSensValue:      0.30,
				DInvariantValue: 0.60,
				SensCoherence:   sens,
				InvCoherence:    inv,
			},
		}
		s := agentmeshkafka.EmpiricalI4Summary(decisions)
		if s.Sensitivity == nil {
			t.Fatal("Sensitivity doit être non-nil quand TP > 0")
		}
		if *s.Sensitivity != 1.0 {
			t.Errorf("*Sensitivity = %f, want 1.0", *s.Sensitivity)
		}
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(data), `"sensitivity":1`) {
			t.Errorf("JSON doit contenir \"sensitivity\":1, obtenu : %s", data)
		}
	})
}
