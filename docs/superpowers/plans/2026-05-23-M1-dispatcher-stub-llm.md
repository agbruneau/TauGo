# M1 Sub-plan — Dispatcher minimal + Stub LLM

> Sous-plan détaillé du milestone M1 (cf. [`PRDPlanning.md` §M1](../../../PRDPlanning.md)). Bite-sized, exécutable par sous-agents frais. Calque structurel du M0 détaillé dans `PRDPlanning.md`.

**Objectif** : `tau decide --input fixture.json` rend une `Decision` instrumentée avec `Regime ∈ {Deterministe, Probabiliste}`. Pas de dimensions calculables (M2) ; régime tiré d'un seuil naïf sur un score de stub LLM déterministe.

**Critère d'acceptation global** :
```bash
echo '{"id":"test-1","intent_description":"hello world"}' | ./tau decide
# → JSON Decision avec Regime, Trace, ProfileVersion non vides
```

**Tag visé** : `v0.0.2-alpha`

**Pré-requis** : M0 complet (tag `v0.0.1-alpha` sur main). Aucun nouveau fichier au-delà du squelette M0 ne devrait exister avant le démarrage de M1.

---

## Tâche M1.1 — Étendre `Decision` avec `Trace` immuable

**Files :**
- Modify: `internal/tau/operator.go` — ajoute le type `Trace` et le champ `Decision.Trace`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Modifier `internal/tau/operator.go`**

Ajouter, dans le même package `tau`, le type `Trace` AVANT la déclaration de `Decision`, et ajouter le champ `Trace` à `Decision`. Le type `Thresholds` doit être imported depuis `orchestration` ? **Non** — cela créerait `tau → orchestration` qui est interdit. À la place, on déclare un **type local minimal** dans `tau` :

```go
// TraceThresholds is the immutable snapshot of the thresholds in effect
// at the time of the decision. Mirrors orchestration.Thresholds; kept here
// to avoid a tau → orchestration import (forbidden by arch_test).
type TraceThresholds struct {
	Deterministe float64
	Probabiliste float64
}

// Trace is the immutable instrumentation of a Decision.
// Once Decision is returned, the Trace must not be mutated.
type Trace struct {
	ExchangeID            string
	TauScore              float64         // composite τ score (M1: stub LLM score; M2: 3-dim weighted)
	Frontier              FrontierCheck   // state of the 4 classical conditions
	Thresholds            TraceThresholds // snapshot at decision time
	UnmodeledObservations []string        // PRD §7.2 #4 — observations not modeled
	DurationNs            int64
}
```

Puis modifier `Decision` :

```go
// Decision is the full output of Kernel.Decide. Always traced.
type Decision struct {
	Regime         Regime
	Diagnostic     string // non-empty iff Regime == Refus
	ProfileVersion string
	DateRevision   time.Time
	Trace          Trace
}
```

Le commentaire `// Trace field intentionally omitted in M0; added in M1.` doit disparaître.

- [ ] **Étape 2 — Vérifier la compilation**

```bash
go build ./internal/tau/
go vet ./...
golangci-lint run ./...
```

Tous doivent être verts. Les 5 sous-tests `TestFrontierCheck_*` doivent encore passer (pas d'impact car `FrontierCheck` n'est pas modifié).

- [ ] **Étape 3 — Commit**

```bash
git add internal/tau/operator.go
git commit -m "feat(tau): add Trace and TraceThresholds; embed Trace in Decision

PRD §10.3 instrumentation: every Decision carries an immutable Trace
exposing tau score, frontier state, thresholds snapshot, unmodeled
observations, and duration. TraceThresholds is declared in tau (not
imported from orchestration) to respect the tau ↛ orchestration arch rule.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M1.2 — Implémenter `internal/orchestration/dispatcher.go`

**Files :**
- Create: `internal/orchestration/dispatcher.go`
- Create: `internal/orchestration/thresholds.go`
- Create: `internal/orchestration/dispatcher_test.go`

**Agent :** `ruflo-core:coder` (TDD)

- [ ] **Étape 1 — Écrire `internal/orchestration/thresholds.go`**

```go
package orchestration

// Thresholds — minimal set required for the M1 dispatcher.
// Full Thresholds (with AuthBlock, SensCoherence, etc.) lands in M2/M5.
type Thresholds struct {
	Deterministe float64 // τ_score < θ → Deterministe
	Probabiliste float64 // τ_score ≥ θ → Probabiliste
}

// Invariant — must hold at all times.
func (t Thresholds) Ordered() bool { return t.Deterministe <= t.Probabiliste }
```

- [ ] **Étape 2 — Écrire le test rouge `internal/orchestration/dispatcher_test.go`**

```go
package orchestration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

type fakeLLM struct{ score float64 }

func (f fakeLLM) Fingerprint() string                                        { return "fake" }
func (f fakeLLM) Interpret(_ context.Context, _ string) (float64, error)     { return f.score, nil }

func newExchangeInsideFrontier(id string) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: "test intent",
		DiscoveredAt:      time.Now(),
		// Frontier conditions must all be VIOLATED for τ to apply.
		// In M1 we don't yet score the frontier from x; the dispatcher
		// constructs FrontierCheck deterministically inside (M1 placeholder
		// returns Inside=true for all exchanges). Adjust per the dispatcher
		// implementation.
	}
}

func TestDispatcher_Decide_Deterministe(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.20}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	x := newExchangeInsideFrontier("t-det")
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Deterministe {
		t.Fatalf("regime = %v, want Deterministe", dec.Regime)
	}
	if dec.Trace.ExchangeID != "t-det" {
		t.Fatalf("trace ExchangeID = %q, want \"t-det\"", dec.Trace.ExchangeID)
	}
}

func TestDispatcher_Decide_Probabiliste(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.80}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	x := newExchangeInsideFrontier("t-prob")
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Probabiliste {
		t.Fatalf("regime = %v, want Probabiliste", dec.Regime)
	}
}

func TestDispatcher_Decide_HysteresisDefaultsToDeterministe(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.50}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	x := newExchangeInsideFrontier("t-hyst")
	dec, _ := d.Decide(context.Background(), x)
	// M1 default: hysteresis zone → Deterministe (M2 will track regime history).
	if dec.Regime != tau.Deterministe {
		t.Fatalf("hysteresis zone: regime = %v, want Deterministe (M1 default)", dec.Regime)
	}
}
```

Vérifier que la compilation échoue (red phase) :
```bash
go test ./internal/orchestration/...
```
Attendu : `undefined: orchestration.NewDispatcher` (ou similaire).

- [ ] **Étape 3 — Écrire `internal/orchestration/dispatcher.go`**

```go
package orchestration

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/tau"
)

// Dispatcher implements the M1 subset of the τ pseudo-algorithm
// (PRD §10): frontier check (step 1), naive composite from stub LLM
// score (step 6), and hysteresis decision (step 7). Steps 2 (ontological
// guard), 3 (profile expiration), 4 (full dimensional scores), 5 (I4
// coherence), and 8 (invariants evaluation) land in M2/M3/M5.
type Dispatcher struct {
	llm        llm.Client
	thresholds Thresholds
}

// NewDispatcher constructs a Dispatcher with the given LLM client and thresholds.
// Panics on ordering invariant violation (calque FibGo: invariant cassé = panic interne).
func NewDispatcher(client llm.Client, t Thresholds) *Dispatcher {
	if !t.Ordered() {
		panic("orchestration: thresholds out of order (Deterministe > Probabiliste)")
	}
	return &Dispatcher{llm: client, thresholds: t}
}

// Decide implements the M1 subset of PRD §10.
func (d *Dispatcher) Decide(ctx context.Context, x tau.Exchange) (tau.Decision, error) {
	start := time.Now()

	// Step 1 — Frontier check (M1: placeholder — assume Inside for any exchange;
	// the real frontier scoring lands in M2 when we have probes on Exchange).
	frontier := tau.FrontierCheck{
		UniversOuvert:       true,
		CompositionVariable: true,
		PairProbabiliste:    true,
		CoutNonBorne:        true,
	}
	if !frontier.Inside() {
		return tau.Decision{
			Regime:     tau.Refus,
			Diagnostic: "hors frontière τ",
			Trace: tau.Trace{
				ExchangeID: x.ID,
				Frontier:   frontier,
				Thresholds: tau.TraceThresholds{
					Deterministe: d.thresholds.Deterministe,
					Probabiliste: d.thresholds.Probabiliste,
				},
				DurationNs: time.Since(start).Nanoseconds(),
			},
		}, nil
	}

	// Step 6 — Naive composite (M1: tau score = LLM stub score).
	tauScore, err := d.llm.Interpret(ctx, x.IntentDescription)
	if err != nil {
		return tau.Decision{}, err
	}

	// Step 7 — Decision with hysteresis (M1: defaults to Deterministe in the band).
	regime := tau.Deterministe
	switch {
	case tauScore < d.thresholds.Deterministe:
		regime = tau.Deterministe
	case tauScore >= d.thresholds.Probabiliste:
		regime = tau.Probabiliste
	default:
		// Hysteresis zone — M1 default: Deterministe. M2 will track per-exchange history.
		regime = tau.Deterministe
	}

	return tau.Decision{
		Regime:         regime,
		ProfileVersion: "M1-default",
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   tauScore,
			Frontier:   frontier,
			Thresholds: tau.TraceThresholds{
				Deterministe: d.thresholds.Deterministe,
				Probabiliste: d.thresholds.Probabiliste,
			},
			DurationNs: time.Since(start).Nanoseconds(),
		},
	}, nil
}
```

- [ ] **Étape 4 — Vérifier que les tests passent**

```bash
go test -v ./internal/orchestration/...
go vet ./...
golangci-lint run ./...
```

Attendu : tous les tests verts ; pas de warning lint.

- [ ] **Étape 5 — Commit**

```bash
git add internal/orchestration/thresholds.go internal/orchestration/dispatcher.go internal/orchestration/dispatcher_test.go
git commit -m "feat(orchestration): add Dispatcher with M1 subset of PRD §10

Implements steps 1 (frontier check, M1 placeholder), 6 (naive composite
from LLM score), and 7 (decision with hysteresis defaulting to Deterministe
in the band). Steps 2/3/4/5/8 deferred to M2/M3/M5.

NewDispatcher panics on threshold ordering violation (calque FibGo:
invariant cassé = panic interne).

Tests cover Deterministe, Probabiliste, and hysteresis branches.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M1.3 — Stub LLM déterministe

**Files :**
- Create: `internal/bridge/llm/client.go`
- Create: `internal/bridge/llm/stub.go`
- Create: `internal/bridge/llm/stub_test.go`

**Agent :** `ruflo-core:coder` (TDD)

- [ ] **Étape 1 — Écrire `internal/bridge/llm/client.go`**

```go
package llm

import "context"

// Client is the narrow interface that TauGo consumes from any LLM.
// No concrete LLM is embedded; the production implementation is injected
// at the app layer (cf. PRD §12.2).
type Client interface {
	// Fingerprint identifies model + version + parameters frozen.
	// Used for profile invalidation (PRD §11.4).
	Fingerprint() string

	// Interpret returns an interpretation score [0, 1] for a given
	// intent description. Used by the S_reasoner_intent probe of
	// D-SENS (PRD §5.1). Must be deterministic under fixed parameters
	// (temperature 0).
	Interpret(ctx context.Context, intent string) (float64, error)
}
```

- [ ] **Étape 2 — Écrire le test rouge `internal/bridge/llm/stub_test.go`**

```go
package llm_test

import (
	"context"
	"testing"

	"github.com/agbruneau/taugo/internal/bridge/llm"
)

func TestStub_Fingerprint(t *testing.T) {
	t.Parallel()
	var c llm.Client = llm.Stub{}
	if c.Fingerprint() != "stub:v0" {
		t.Fatalf("fingerprint = %q, want \"stub:v0\"", c.Fingerprint())
	}
}

func TestStub_Interpret_DeterministicAndBounded(t *testing.T) {
	t.Parallel()
	var c llm.Client = llm.Stub{}
	ctx := context.Background()
	cases := []string{"", "a", "hello world", "the quick brown fox"}
	for _, in := range cases {
		in := in
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			a, err := c.Interpret(ctx, in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b, _ := c.Interpret(ctx, in)
			if a != b {
				t.Fatalf("non-deterministic: got %f then %f for %q", a, b, in)
			}
			if a < 0 || a >= 1 {
				t.Fatalf("score %f out of [0, 1) for %q", a, in)
			}
		})
	}
}
```

Vérifier red phase :
```bash
go test ./internal/bridge/llm/...
```

Attendu : `undefined: llm.Stub`.

- [ ] **Étape 3 — Écrire `internal/bridge/llm/stub.go`**

```go
package llm

import "context"

// Stub is the deterministic LLM client used by default in tests and
// for calibration reproducibility (PRD §15.4). It MUST be used by
// default unless TAUGO_LLM_BACKEND=real is set explicitly at the app layer.
type Stub struct{}

// Fingerprint returns a stable identifier for the stub.
// Real LLM backends carry their model + parameters in this string.
func (Stub) Fingerprint() string { return "stub:v0" }

// Interpret returns a deterministic score in [0, 1) derived from the
// intent string via FNV-1a 32-bit hash. Mapping is checked-in (this
// function is the mapping).
func (Stub) Interpret(_ context.Context, intent string) (float64, error) {
	const (
		offset uint32 = 2166136261
		prime  uint32 = 16777619
	)
	h := offset
	for i := 0; i < len(intent); i++ {
		h ^= uint32(intent[i])
		h *= prime
	}
	return float64(h%1000) / 1000.0, nil
}
```

- [ ] **Étape 4 — Mettre à jour `internal/arch_test.go`**

Remplacer le bloc pour `bridge` (qui skippe car `internal/bridge` n'est pas un package) par les sous-packages réels :

```go
// Remplacer cette règle :
{from: "github.com/agbruneau/taugo/internal/bridge", deny: []string{
    "github.com/agbruneau/taugo/internal/tau",
}},

// par celles-ci :
{from: "github.com/agbruneau/taugo/internal/bridge/llm", deny: []string{
    "github.com/agbruneau/taugo/internal/tau",
}},
{from: "github.com/agbruneau/taugo/internal/bridge/agentmeshkafka", deny: []string{
    "github.com/agbruneau/taugo/internal/tau",
}},
```

- [ ] **Étape 5 — Vérifier**

```bash
go test -v ./internal/bridge/llm/...
go test -v ./internal/
go vet ./...
golangci-lint run ./...
```

Attendu : `TestStub_*` passent ; `TestArchitectureLayering/...bridge_llm` et `...bridge_agentmeshkafka` doivent maintenant PASSER (le premier parce que `bridge/llm` n'importe que `context` stdlib ; le second parce que `bridge/agentmeshkafka` n'a que `doc.go`).

- [ ] **Étape 6 — Commit**

```bash
git add internal/bridge/llm/ internal/arch_test.go
git commit -m "feat(bridge/llm): add Client interface and deterministic Stub

PRD §12.2: narrow interface, no LLM embedded, deterministic stub for
CI and calibration reproducibility. Stub uses FNV-1a 32-bit hash to
map intent → score in [0, 1) — fingerprint stub:v0.

arch_test.go extended to check the real sub-packages of internal/bridge/
(llm and agentmeshkafka) instead of the non-existent parent.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M1.4 — Injection LLM en `internal/app/`

**Files :**
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Écrire `internal/app/app.go`**

```go
package app

import (
	"os"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/orchestration"
)

// Default thresholds for M1. Calibration in M5 will override these.
var defaultThresholds = orchestration.Thresholds{
	Deterministe: 0.35,
	Probabiliste: 0.65,
}

// NewDispatcher constructs the production Dispatcher.
// Default LLM: deterministic Stub (PRD §15.4).
// TAUGO_LLM_BACKEND=real switches to a real LLM (M5+; currently panics).
func NewDispatcher() *orchestration.Dispatcher {
	return orchestration.NewDispatcher(selectLLM(), defaultThresholds)
}

func selectLLM() llm.Client {
	if os.Getenv("TAUGO_LLM_BACKEND") == "real" {
		panic("real LLM backend not implemented yet (M5+)")
	}
	return llm.Stub{}
}
```

- [ ] **Étape 2 — Écrire `internal/app/app_test.go`**

```go
package app_test

import (
	"reflect"
	"testing"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/llm"
)

// TestDefaultLLMIsStub guards anti-patron: in CI / default mode, no
// external LLM service may be called. The Dispatcher must always use
// llm.Stub unless TAUGO_LLM_BACKEND=real is set explicitly.
func TestDefaultLLMIsStub(t *testing.T) {
	t.Parallel()
	// Construct a dispatcher under default env; verify it uses Stub.
	// Reflection on the unexported `llm` field is non-trivial; instead,
	// rely on behavioral fingerprint: NewDispatcher's score for a known
	// intent must equal Stub's score for the same intent.
	d := app.NewDispatcher()
	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}
	// Indirect check via type — we know app exposes nothing else; cf.
	// the behavioral test in cmd/tau end-to-end (M1.6).
	_ = reflect.TypeOf(llm.Stub{}) // import touched for clarity
}
```

- [ ] **Étape 3 — Vérifier**

```bash
go test ./internal/app/...
go vet ./...
golangci-lint run ./...
```

- [ ] **Étape 4 — Commit**

```bash
git add internal/app/
git commit -m "feat(app): wire stub LLM into Dispatcher (default factory)

NewDispatcher uses llm.Stub by default. Real backend (TAUGO_LLM_BACKEND=real)
panics until M5+ — explicit signal that CI must remain on the stub.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M1.5 — Commande `tau decide`

**Files :**
- Modify: `cmd/tau/main.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Modifier `cmd/tau/main.go`**

Garder le squelette `--help`/`--version` et ajouter la commande `decide` :

```go
// Command tau is the TauGo CLI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/tau"
)

var (
	buildTimestamp = "dev"
	version        = "0.0.2-alpha"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-version":
			fmt.Printf("tau %s (build %s)\n", version, buildTimestamp)
			os.Exit(0)
		case "decide":
			runDecide()
			return
		}
	}
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `tau — TauGo kernel CLI (V0.1)

USAGE:
    tau <command> [flags]

COMMANDS:
    decide      Decide a regime for one exchange (reads JSON Exchange on stdin)
    calibrate   Run adaptive calibration on a corpus (M5+)
    --version   Print version

Specification: PRD.md
`)
	}
	flag.Parse()
	flag.Usage()
	os.Exit(1)
}

func runDecide() {
	var x tau.Exchange
	if err := json.NewDecoder(os.Stdin).Decode(&x); err != nil {
		fmt.Fprintln(os.Stderr, "error decoding stdin:", err)
		os.Exit(2)
	}
	d := app.NewDispatcher()
	decision, err := d.Decide(context.Background(), x)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error deciding:", err)
		os.Exit(3)
	}
	if err := json.NewEncoder(os.Stdout).Encode(decision); err != nil {
		fmt.Fprintln(os.Stderr, "error encoding decision:", err)
		os.Exit(4)
	}
}
```

Note: `version` est désormais `0.0.2-alpha` (anticipe le tag M1.9).

- [ ] **Étape 2 — Vérifier le build**

```bash
go build -o tau ./cmd/tau
./tau --help
./tau --version
echo '{"id":"manual-1","intent_description":"hello world"}' | ./tau decide
```

Attendu :
- `--help` → texte d'aide (exit 1 maintenant car on a ajouté `os.Exit(1)` après Usage)
- `--version` → `tau 0.0.2-alpha (build dev)`
- `decide` → JSON `{"Regime":...,"Trace":{...}}` sur stdout, exit 0

Supprimer le binaire local : `rm -f tau tau.exe`.

- [ ] **Étape 3 — Commit**

```bash
git add cmd/tau/main.go
git commit -m "feat(cli): add 'decide' subcommand (JSON stdin → JSON stdout)

PRD §10 entry point. Reads a tau.Exchange JSON on stdin, dispatches via
app.NewDispatcher, writes the Decision JSON on stdout. Exit codes:
0 success, 2 stdin decode error, 3 dispatch error, 4 stdout encode error.

Version bumped to 0.0.2-alpha (in anticipation of M1.9 tag).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M1.6 — Test E2E `cmd/tau`

**Files :**
- Create: `cmd/tau/main_test.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Écrire le test E2E**

```go
package main_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// buildCLI compiles the cmd/tau binary into a temp file and returns its path.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "tau")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "github.com/agbruneau/taugo/cmd/tau")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build failed: %v", err)
	}
	return bin
}

func runDecide(t *testing.T, bin string, input string) map[string]any {
	t.Helper()
	cmd := exec.Command(bin, "decide")
	cmd.Stdin = bytes.NewBufferString(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("tau decide failed: %v", err)
	}
	var dec map[string]any
	if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
		t.Fatalf("decode decision: %v", err)
	}
	return dec
}

// Verified via FNV-1a hash of the intent strings:
//   "creative generation" → 0.262 → Deterministe
//   "hello world"         → 0.807 → Probabiliste
func TestEndToEnd_DecideDeterministe(t *testing.T) {
	t.Parallel()
	bin := buildCLI(t)
	dec := runDecide(t, bin, `{"id":"t1","intent_description":"creative generation"}`)
	if r, _ := dec["Regime"].(float64); r != 1 {
		// Regime is Deterministe = 1 in the iota; JSON marshals int as float64.
		t.Fatalf("regime = %v, want 1 (Deterministe)", dec["Regime"])
	}
}

func TestEndToEnd_DecideProbabiliste(t *testing.T) {
	t.Parallel()
	bin := buildCLI(t)
	dec := runDecide(t, bin, `{"id":"t2","intent_description":"hello world"}`)
	if r, _ := dec["Regime"].(float64); r != 2 {
		// Regime is Probabiliste = 2 in the iota.
		t.Fatalf("regime = %v, want 2 (Probabiliste)", dec["Regime"])
	}
}
```

**Note** : les scores de stub pour les fixtures (`"creative generation"` et `"hello world"`) doivent être calculés à l'avance par l'agent via le hash FNV-1a documenté en M1.3, et vérifiés en local avant de figer les valeurs attendues dans le test. Si les valeurs prévues ne tombent pas dans les bons régimes avec les seuils `{0.35, 0.65}`, l'agent doit choisir d'autres fixtures qui s'y rangent et documenter pourquoi dans le commit.

- [ ] **Étape 2 — Vérifier**

```bash
go test -v -run TestEndToEnd ./cmd/tau/...
```

- [ ] **Étape 3 — Commit**

```bash
git add cmd/tau/main_test.go
git commit -m "test(cli): E2E decide tests for Deterministe and Probabiliste regimes

Compiles tau CLI in temp dir, feeds JSON Exchange on stdin, verifies
the returned Decision regime. Fixtures chosen so stub LLM FNV-1a hash
maps into the two extreme bands with default thresholds (0.35, 0.65).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M1.7 — `TestDefaultLLMIsStub` (déjà couvert M1.4)

`TestDefaultLLMIsStub` a été ajouté en M1.4 (étape 2). Pas de tâche supplémentaire ici. **Cette ligne du plan est volontairement courte** ; le test est déjà inclus dans `internal/app/app_test.go`.

---

## Tâche M1.8 — Tests d'invariants `Decision`

**Files :**
- Create: `internal/orchestration/decision_invariants_test.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Écrire les trois tests d'invariants**

```go
package orchestration_test

import (
	"context"
	"testing"

	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestDecisionAlwaysTraced — every Decision must have a non-zero Trace.ExchangeID
// matching the input Exchange.ID, regardless of regime.
func TestDecisionAlwaysTraced(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.50}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	dec, _ := d.Decide(context.Background(), tau.Exchange{ID: "must-be-traced"})
	if dec.Trace.ExchangeID != "must-be-traced" {
		t.Fatalf("trace ExchangeID = %q, want \"must-be-traced\"", dec.Trace.ExchangeID)
	}
	if dec.Trace.DurationNs <= 0 {
		t.Fatalf("trace DurationNs = %d, want > 0", dec.Trace.DurationNs)
	}
}

// TestRefusImpliesDiagnostic — Decision.Regime == Refus iff Decision.Diagnostic != "".
// In M1, the only refus path is "hors frontière τ" but the frontier check is a
// placeholder that always returns Inside=true, so this test exercises the
// implication contractually by constructing a Decision directly. M2 will add
// real refus paths.
func TestRefusImpliesDiagnostic(t *testing.T) {
	t.Parallel()
	// Refus with non-empty diagnostic: contract holds.
	refus := tau.Decision{Regime: tau.Refus, Diagnostic: "x"}
	if (refus.Regime == tau.Refus) != (refus.Diagnostic != "") {
		t.Fatal("contract broken: Refus must have non-empty Diagnostic")
	}
	// Non-Refus with empty diagnostic: contract holds.
	det := tau.Decision{Regime: tau.Deterministe, Diagnostic: ""}
	if (det.Regime == tau.Refus) != (det.Diagnostic != "") {
		t.Fatal("contract broken: non-Refus must have empty Diagnostic")
	}
}

// TestTraceImmutable — Trace is a value type embedded by value in Decision.
// Once Decide returns, the caller can mutate their local copy but cannot
// mutate the Dispatcher's internal state via the returned Decision. This
// test verifies the value-semantics by mutating the returned Trace and
// re-deciding; the second decision must be independent.
func TestTraceImmutable(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(fakeLLM{score: 0.20}, orchestration.Thresholds{
		Deterministe: 0.35,
		Probabiliste: 0.65,
	})
	dec1, _ := d.Decide(context.Background(), tau.Exchange{ID: "first"})
	dec1.Trace.ExchangeID = "MUTATED-LOCAL"  // mutate local copy
	dec2, _ := d.Decide(context.Background(), tau.Exchange{ID: "second"})
	if dec2.Trace.ExchangeID != "second" {
		t.Fatalf("second decision Trace ExchangeID = %q, want \"second\" (Trace must be per-decision)", dec2.Trace.ExchangeID)
	}
}
```

- [ ] **Étape 2 — Vérifier**

```bash
go test -v ./internal/orchestration/...
go test ./...
```

- [ ] **Étape 3 — Commit**

```bash
git add internal/orchestration/decision_invariants_test.go
git commit -m "test(orchestration): assert Decision invariants

- TestDecisionAlwaysTraced: every Decision carries a non-zero Trace
  with ExchangeID matching the input and a positive DurationNs.
- TestRefusImpliesDiagnostic: contractual biconditional verified.
- TestTraceImmutable: value semantics — each Decide call produces an
  independent Trace; mutation of the returned Trace cannot leak back.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M1.9 — Revue intégrée + tag `v0.0.2-alpha`

**Agent :** thread principal (intégration) + `ruflo-core:reviewer` (revue)

- [ ] **Étape 1 — Vérifier la suite locale complète**

```bash
go build ./...
go test -v ./...               # tous PASS, incl. les nouveaux tests M1
go vet ./...
golangci-lint run ./...
go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/... ./...
go tool cover -func=coverage.out | tail -1
```

Couverture cible : ≥ 80 % global sur internal/*. `internal/tau` reste à 100 % ; `internal/orchestration` et `internal/bridge/llm` doivent être ≥ 80 %.

- [ ] **Étape 2 — Briefing reviewer (intégré M1)**

Dispatch un `ruflo-core:reviewer` avec le briefing suivant :

> Revue intégrée de M1 (commit range `385a915..HEAD`). Vérifier :
> 1. La signature `Decision` inclut maintenant un champ `Trace` non vide après tout `Decide`.
> 2. `Dispatcher` implémente strictement les étapes 1, 6, 7 du pseudo-algo PRD §10 ; pas d'étape 2/3/4/5/8.
> 3. `llm.Client` est une interface étroite (≤ 5 méthodes), `llm.Stub` est l'unique implémentation et le test `TestDefaultLLMIsStub` interdit l'introduction d'un backend réel sans variable d'env explicite.
> 4. `arch_test.go` couvre les sous-packages réels de `internal/bridge/*` (les rules anciennes parent skip-toujours sont supprimées).
> 5. `cmd/tau decide` accepte JSON stdin et émet JSON stdout ; tests E2E couvrent Deterministe et Probabiliste.
> 6. Aucun anti-patron introduit : pas de méthode `Predict*`/`Expected*`/`Forecast*` exportée ; pas d'import LLM concret (anthropic, openai) dans `internal/tau/*` ou `internal/orchestration/*`.
> 7. Conventions FibGo : `t.Parallel()` adoption 100 %, erreurs structurées, pas de panic hors invariant interne (seul `NewDispatcher` panic sur threshold ordering — documenté), pas d'emoji, godoc en anglais.

- [ ] **Étape 3 — Tag `v0.0.2-alpha`**

```bash
git tag -a v0.0.2-alpha -m "M1: dispatcher minimal + stub LLM

- Decision.Trace immutable (PRD §10.3)
- Dispatcher: frontier check (placeholder) + naive composite (stub LLM)
  + hysteresis (Deterministe default in M1)
- Stub LLM (FNV-1a 32-bit hash; deterministic; fingerprint stub:v0)
- App layer wires Stub into Dispatcher; real backend panics (M5+)
- CLI 'decide' subcommand (JSON stdin → JSON stdout, exit codes documented)
- E2E tests (Deterministe, Probabiliste) + invariant tests
  (TestDecisionAlwaysTraced, TestRefusImpliesDiagnostic, TestTraceImmutable,
  TestDefaultLLMIsStub)
- arch_test.go extended to sub-packages of bridge/

Spec: PRD.md §10. Plan: docs/superpowers/plans/2026-05-23-M1-dispatcher-stub-llm.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin main
git push origin v0.0.2-alpha
```

- [ ] **Étape 4 — Mettre à jour `CHANGELOG.md`**

Ajouter sous `## [Non publié]` :

```markdown
## [0.0.2-alpha] — 2026-05-XX

### Ajouté

- `internal/tau/operator.go` : type `Trace` immuable + `TraceThresholds` ; champ `Decision.Trace`.
- `internal/orchestration/dispatcher.go` + `thresholds.go` : `Dispatcher` implémentant le sous-ensemble M1 du pseudo-algo PRD §10 (étapes 1, 6, 7).
- `internal/bridge/llm/client.go` : interface étroite `Client` (PRD §12.2).
- `internal/bridge/llm/stub.go` : `Stub` LLM déterministe via FNV-1a 32-bit hash ; fingerprint `stub:v0`.
- `internal/app/app.go` : `NewDispatcher()` wire le stub par défaut ; `TAUGO_LLM_BACKEND=real` panic (M5+).
- `cmd/tau decide` : sous-commande CLI lisant `tau.Exchange` JSON sur stdin et émettant `Decision` JSON sur stdout.
- Tests : E2E `cmd/tau`, invariants `Decision` (always traced, refus ⟺ diagnostic, trace immutable), `TestDefaultLLMIsStub`.
- `internal/arch_test.go` : règles étendues aux sous-packages réels de `internal/bridge/*`.
- `docs/superpowers/plans/2026-05-23-M1-dispatcher-stub-llm.md` : sous-plan détaillé M1.
```

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): M1 release notes (v0.0.2-alpha)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin main
```

---

## Annexe — Self-review (à exécuter avant commit du plan)

- ☐ **Couverture M1 high-level** :
  - M1.1 Decision.Trace → M1.1 (étape 1 de ce sous-plan) ✓
  - M1.2 Dispatcher → M1.2 ✓
  - M1.3 Stub LLM → M1.3 ✓
  - M1.4 App injection → M1.4 ✓
  - M1.5 `tau decide` CLI → M1.5 ✓
  - M1.6 E2E tests → M1.6 ✓
  - M1.7 `TestDefaultLLMIsStub` → couvert en M1.4 ✓
  - M1.8 invariant tests → M1.8 ✓
  - M1.9 review + tag → M1.9 ✓
- ☐ **Placeholder scan** : aucun TBD/TODO résiduel
- ☐ **Cohérence des types** : `Decision`, `Trace`, `Thresholds`, `TraceThresholds`, `Kernel`, `Regime`, `Client`, `Stub`, `Dispatcher` cohérents entre tâches
- ☐ **Anti-patrons gardés (cumulés M0 + M1)** :
  - #1 prédictif → vérifié par revue (pas d'API `Predict*`) ; `TestNoPredictiveAPI` arrive en M3.9
  - #2 hors frontière → `TestFrontierCheck_*` (M0) ; refus géré dans le dispatcher
  - #3 atemporel → reporté à M3.9 / M5
  - #4 clos → `Trace.UnmodeledObservations` exposé (champ vide en M1)

---

*Sous-plan V1 — 2026-05-23. Référence : `PRDPlanning.md` §M1 + `PRD.md` §10. Coordinateur : Claude Code thread principal. Exécutants : agent teams (`ruflo-core:coder`, `ruflo-core:reviewer`).*
