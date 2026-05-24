// internal/tau/operator_test.go
package tau_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// --- T-023: Regime String/MarshalJSON/UnmarshalJSON ---

func TestRegime_String(t *testing.T) {
	t.Parallel()
	cases := []struct {
		r    tau.Regime
		want string
	}{
		{tau.RegimeUnknown, "RegimeUnknown"},
		{tau.Deterministe, "Deterministe"},
		{tau.Probabiliste, "Probabiliste"},
		{tau.Refus, "Refus"},
		{tau.Regime(99), "Unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if got := tc.r.String(); got != tc.want {
				t.Errorf("Regime(%d).String() = %q, want %q", int(tc.r), got, tc.want)
			}
		})
	}
}

func TestRegime_MarshalJSON_StringRepresentation(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(tau.Deterministe)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(b) != `"Deterministe"` {
		t.Errorf("MarshalJSON(Deterministe) = %s, want %q", b, "Deterministe")
	}
}

func TestRegime_UnmarshalJSON_AccepteStringEtInt(t *testing.T) {
	t.Parallel()
	t.Run("string PascalCase", func(t *testing.T) {
		t.Parallel()
		var r tau.Regime
		if err := json.Unmarshal([]byte(`"Probabiliste"`), &r); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if r != tau.Probabiliste {
			t.Errorf("got %v, want Probabiliste", r)
		}
	})
	t.Run("string lowercase legacy", func(t *testing.T) {
		t.Parallel()
		var r tau.Regime
		if err := json.Unmarshal([]byte(`"refus"`), &r); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if r != tau.Refus {
			t.Errorf("got %v, want Refus", r)
		}
	})
	t.Run("int legacy", func(t *testing.T) {
		t.Parallel()
		var r tau.Regime
		if err := json.Unmarshal([]byte(`1`), &r); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if r != tau.Deterministe {
			t.Errorf("got %v, want Deterministe", r)
		}
	})
	t.Run("string inconnue retourne erreur", func(t *testing.T) {
		t.Parallel()
		var r tau.Regime
		if err := json.Unmarshal([]byte(`"invalid"`), &r); err == nil {
			t.Error("expected error for unknown string, got nil")
		}
	})
	t.Run("JSON invalide retourne erreur", func(t *testing.T) {
		t.Parallel()
		var r tau.Regime
		if err := json.Unmarshal([]byte(`{}`), &r); err == nil {
			t.Error("expected error for invalid JSON type, got nil")
		}
	})
}

// --- T-024: DiscoveryMode String/MarshalJSON/UnmarshalJSON ---

func TestDiscoveryMode_String(t *testing.T) {
	t.Parallel()
	cases := []struct {
		d    tau.DiscoveryMode
		want string
	}{
		{tau.Static, "Static"},
		{tau.DynamicMCP, "DynamicMCP"},
		{tau.DynamicA2A, "DynamicA2A"},
		{tau.DynamicAGNTCY, "DynamicAGNTCY"},
		{tau.DiscoveryMode(99), "Unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if got := tc.d.String(); got != tc.want {
				t.Errorf("DiscoveryMode(%d).String() = %q, want %q", int(tc.d), got, tc.want)
			}
		})
	}
}

func TestDiscoveryMode_MarshalJSON_StringRepresentation(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(tau.DynamicMCP)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(b) != `"DynamicMCP"` {
		t.Errorf("MarshalJSON(DynamicMCP) = %s, want %q", b, "DynamicMCP")
	}
}

func TestDiscoveryMode_UnmarshalJSON_AccepteStringEtInt(t *testing.T) {
	t.Parallel()
	t.Run("string PascalCase", func(t *testing.T) {
		t.Parallel()
		var d tau.DiscoveryMode
		if err := json.Unmarshal([]byte(`"DynamicA2A"`), &d); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if d != tau.DynamicA2A {
			t.Errorf("got %v, want DynamicA2A", d)
		}
	})
	t.Run("string snake_case legacy", func(t *testing.T) {
		t.Parallel()
		var d tau.DiscoveryMode
		if err := json.Unmarshal([]byte(`"dynamic_mcp"`), &d); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if d != tau.DynamicMCP {
			t.Errorf("got %v, want DynamicMCP", d)
		}
	})
	t.Run("int legacy", func(t *testing.T) {
		t.Parallel()
		var d tau.DiscoveryMode
		if err := json.Unmarshal([]byte(`0`), &d); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if d != tau.Static {
			t.Errorf("got %v, want Static", d)
		}
	})
	t.Run("string inconnue retourne erreur", func(t *testing.T) {
		t.Parallel()
		var d tau.DiscoveryMode
		if err := json.Unmarshal([]byte(`"nope"`), &d); err == nil {
			t.Error("expected error for unknown string, got nil")
		}
	})
	t.Run("JSON invalide retourne erreur", func(t *testing.T) {
		t.Parallel()
		var d tau.DiscoveryMode
		if err := json.Unmarshal([]byte(`{}`), &d); err == nil {
			t.Error("expected error for invalid JSON type, got nil")
		}
	})
}

// TestTrace_MarshalJSON_ScoresVentilesOmitemptySiVides verifies that a Trace
// with nil Score pointers omits d_sens, d_authority, d_invariant from JSON,
// and that non-nil scores appear correctly (ADR-0008).
func TestTrace_MarshalJSON_ScoresVentilesOmitemptySiVides(t *testing.T) {
	t.Parallel()

	t.Run("scores absents si nil", func(t *testing.T) {
		t.Parallel()
		tr := tau.Trace{
			ExchangeID: "test-x",
			TauScore:   0.5,
			DurationNs: 1,
			// DSens, DAuthority, DInvariant left nil
		}
		b, err := json.Marshal(tr)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		out := string(b)
		for _, key := range []string{`"d_sens"`, `"d_authority"`, `"d_invariant"`} {
			if strings.Contains(out, key) {
				t.Errorf("expected key %s to be absent for nil Score, but found in: %s", key, out)
			}
		}
	})

	t.Run("scores présents si peuplés", func(t *testing.T) {
		t.Parallel()
		ts := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
		tr := tau.Trace{
			ExchangeID: "test-x",
			TauScore:   0.5,
			DurationNs: 1,
			DSens:      &tau.Score{Value: 0.42, ComputedAt: ts},
			DAuthority: &tau.Score{Value: 0.75, ComputedAt: ts},
			DInvariant: &tau.Score{Value: 0.30, ComputedAt: ts},
		}
		b, err := json.Marshal(tr)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		out := string(b)
		for _, key := range []string{`"d_sens"`, `"d_authority"`, `"d_invariant"`} {
			if !strings.Contains(out, key) {
				t.Errorf("expected key %s to be present for non-nil Score, not found in: %s", key, out)
			}
		}
	})
}
