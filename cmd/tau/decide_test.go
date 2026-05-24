// Tests for runDecide in the tau CLI.
// Uses bytes.Buffer for in/out so the coverage tool sees these paths.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/agbruneau/taugo/internal/tau"
)

// validExchangeJSON is a minimal Exchange that passes all frontier checks and
// produces a successful Decision (exit 0). Uses a dynamic-MCP target with
// a contract URI and an institutional attestation so D-AUTORITÉ is satisfied.
const validExchangeJSON = `{
	"id": "unit-t1",
	"intent_description": "dispatch notification to subscriber",
	"initiator": {
		"id": "agent-unit",
		"organization": "org-unit",
		"delegation_depth": 1
	},
	"target": {
		"id": "tool-unit",
		"discovery_mode": 2,
		"contract_uri": "https://api.example.org/v1/op"
	},
	"attestation_institutionnelle": {
		"emetteur": "desjardins-iam",
		"reference": "ref-unit-stable",
		"marqueur": "Probable",
		"asserted_at": "2026-01-01T00:00:00Z"
	},
	"discovered_at": "2026-04-01T00:03:00Z"
}`

func TestRunDecide_ValidExchange_Exit0(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(validExchangeJSON)
	var out bytes.Buffer
	code := runDecide(in, &out)
	if code != 0 {
		t.Fatalf("runDecide returned %d, want 0; output: %s", code, out.String())
	}
}

func TestRunDecide_InvalidJSON_Exit2(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(`{bad json`)
	var out bytes.Buffer
	code := runDecide(in, &out)
	if code != 2 {
		t.Fatalf("runDecide returned %d, want 2", code)
	}
}

func TestRunDecide_EmptyStdin_Exit2(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(``)
	var out bytes.Buffer
	code := runDecide(in, &out)
	if code != 2 {
		t.Fatalf("runDecide returned %d, want 2 (empty input is invalid JSON)", code)
	}
}

func TestRunDecide_ValidExchange_OutputIsValidJSON(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(validExchangeJSON)
	var out bytes.Buffer
	if code := runDecide(in, &out); code != 0 {
		t.Fatalf("runDecide returned %d; output: %s", code, out.String())
	}
	var d tau.Decision
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatalf("output is not a valid Decision JSON: %v\nraw: %s", err, out.String())
	}
	if d.Trace.ExchangeID != "unit-t1" {
		t.Errorf("trace.exchange_id = %q, want \"unit-t1\"", d.Trace.ExchangeID)
	}
}

func TestRunDecide_ValidExchange_RegimeIsSet(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(validExchangeJSON)
	var out bytes.Buffer
	if code := runDecide(in, &out); code != 0 {
		t.Fatalf("runDecide returned %d; output: %s", code, out.String())
	}
	var d tau.Decision
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatalf("decode decision: %v", err)
	}
	if d.Regime == tau.RegimeUnknown {
		t.Errorf("regime is RegimeUnknown (0), want a concrete regime")
	}
}
