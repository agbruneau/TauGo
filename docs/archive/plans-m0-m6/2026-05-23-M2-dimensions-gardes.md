# M2 Sub-plan — Trois dimensions + gardes ontologique D-AUTORITÉ et I4

> Sous-plan détaillé du milestone M2 (cf. [`PRDPlanning.md` §M2](../../../PRDPlanning.md)). Bite-sized, exécutable par sous-agents frais. Calque structurel du M1 détaillé dans `docs/superpowers/plans/2026-05-23-M1-dispatcher-stub-llm.md`.

**Objectif** : les sondes D-SENS, D-AUTORITÉ, D-INVARIANT calculent un score `[0, 1]` réel. La garde ontologique D-AUTORITÉ (étape 2 du pseudo-algo PRD §10) et la garde de cohérence I4 (étape 5) sont actives dans le dispatcher. Le composite τ est le résultat pondéré des trois dimensions, non plus le stub LLM direct.

**Critère d'acceptation global** :
```bash
go test -race ./... && \
  go test -run TestRefusOntologiqueDAUTORITE ./internal/orchestration/ && \
  go test -run TestI4_IncoherenceDetectee ./internal/orchestration/
```
…vert. Rapport `docs/empirical/M2-sample-decisions.md` avec 10 décisions tracées, scores ventilés par dimension et par sonde.

**Tag visé** : `v0.0.3-alpha`

**Pré-requis** : M1 complet (tag `v0.0.2-alpha` sur main). `internal/orchestration/dispatcher.go`, `internal/orchestration/thresholds.go`, `internal/bridge/llm/stub.go`, `internal/app/app.go`, `cmd/tau/main.go` existent et compilent.

---

## Note de conception — refactoring du dispatcher en M2

### Remplacement du score naïf par le composite pondéré

M1 utilise `tauScore := d.llm.Interpret(ctx, x.IntentDescription)` directement comme composite. M2 remplace cette ligne par :

```
tauScore = w_s * D_SENS(x) + w_a * D_AUTH(x) + w_i * D_INV(x)
```

Le client LLM (`d.llm`) est conservé mais utilisé uniquement par la sonde `S_reasoner_intent` de D-SENS — pas dans le corps principal du dispatcher.

### Dérivation de FrontierCheck depuis Exchange

Le M1 dispatcher construit un `FrontierCheck` en dur avec toutes les conditions à `true` (placeholder). M2 le remplace par une dérivation depuis `Exchange.Initiator` et `Exchange.Target` via une fonction helper `frontierFromExchange` dans `internal/orchestration/`. Heuristiques :

- `Target.DiscoveryMode != Static` → `UniversOuvert = true`, `CompositionVariable = true`
- `!Initiator.HumanInLoop` → `PairProbabiliste = true`
- `Initiator.DelegationDepth > 0` → `CoutNonBorne = true`

Ces règles sont des placeholders documentés jusqu'à la calibration empirique M5.

### Contraintes architecturales

- `internal/tau/dimensions/` peut importer `internal/tau` (parent) et `internal/bridge/llm` (pour `S_reasoner_intent`). Ne peut pas importer `internal/orchestration`.
- Les trois fichiers de dimension (`dsens.go`, `dauthority.go`, `dinvariant.go`) ne s'importent pas entre eux.
- `internal/orchestration/dispatcher.go` importe `internal/tau/dimensions` pour calculer les scores.
- `internal/calibration/` n'importe pas `internal/tau/dimensions` directement.

---

## Tâche M2.1 — Types `Principal`, `Capability`, `DiscoveryMode` + extension `Exchange`

**Files :**
- Modify: `internal/tau/operator.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Modifier `internal/tau/operator.go`**

Ajouter les nouveaux types après la déclaration de `Attestation`, avant `TraceThresholds`. Ajouter les champs `Initiator` et `Target` à `Exchange`. Supprimer le commentaire `// Principal and Capability fields intentionally omitted in M0`.

```go
// DiscoveryMode describes how a Capability is discovered at the boundary.
type DiscoveryMode int

const (
	// Static means the capability is known and wired at design time.
	Static DiscoveryMode = iota
	// DynamicMCP means the capability is discovered via MCP list_tools at runtime.
	DynamicMCP
	// DynamicA2A means the capability is discovered via A2A protocol at runtime.
	DynamicA2A
	// DynamicAGNTCY means the capability is discovered via AGNTCY registry at runtime.
	DynamicAGNTCY
)

// Principal is the initiating agent of an interoperability exchange.
type Principal struct {
	ID              string `json:"id"`
	HumanInLoop     bool   `json:"human_in_loop"`
	Organization    string `json:"organization"`
	DelegationDepth int    `json:"delegation_depth"` // 0 = direct human mandate
}

// Capability is the target capability being invoked in the exchange.
type Capability struct {
	ID            string        `json:"id"`
	DiscoveryMode DiscoveryMode `json:"discovery_mode"`
	ContractURI   string        `json:"contract_uri,omitempty"` // empty = no contract
}
```

Modifier `Exchange` pour inclure les nouveaux champs :

```go
// Exchange is the interoperability exchange submitted to τ.
type Exchange struct {
	ID                          string         `json:"id"`
	Initiator                   Principal      `json:"initiator"`
	Target                      Capability     `json:"target"`
	IntentDescription           string         `json:"intent_description"`
	DiscoveredAt                time.Time      `json:"discovered_at"`
	AttestationInstitutionnelle *Attestation   `json:"attestation_institutionnelle,omitempty"`
	Context                     map[string]any `json:"context,omitempty"`
}
```

Ajouter à `TraceThresholds` le champ `AuthBlock` introduit en M2 :

```go
// TraceThresholds is the immutable snapshot of the thresholds in effect
// at the time of the decision. Mirrors orchestration.Thresholds; kept here
// to avoid a tau -> orchestration import (forbidden by arch_test).
type TraceThresholds struct {
	Deterministe   float64 `json:"deterministe"`
	Probabiliste   float64 `json:"probabiliste"`
	AuthBlock      float64 `json:"auth_block"`
	SensCoherence  float64 `json:"sens_coherence"`
	InvCoherence   float64 `json:"inv_coherence"`
}
```

- [ ] **Étape 2 — Vérifier la compilation**

```bash
go build ./internal/tau/
go vet ./...
golangci-lint run ./...
go test -race ./internal/tau/...
```

Tous les tests `TestFrontierCheck_*` doivent encore passer. Pas de test cassé.

- [ ] **Étape 3 — Commit**

```bash
git add internal/tau/operator.go
git commit -m "feat(tau): add Principal, Capability, DiscoveryMode; extend Exchange and TraceThresholds

M2.1: Principal (HumanInLoop, Organization, DelegationDepth) and Capability
(DiscoveryMode, ContractURI) encode the two poles of an Exchange boundary.
DiscoveryMode drives the M2 frontier heuristic (Static vs Dynamic*).
TraceThresholds gains AuthBlock, SensCoherence, InvCoherence to snapshot
the full M2 guard configuration at decision time.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.2 — `internal/tau/dimensions/dsens.go` + 4 sondes + tests

**Files :**
- Create: `internal/tau/dimensions/dsens.go`
- Create: `internal/tau/dimensions/dsens_test.go`
- Create: `internal/tau/dimensions/doc.go`

**Agent :** `ruflo-core:coder` (TDD)

- [ ] **Étape 1 — Créer `internal/tau/dimensions/doc.go`**

```go
// Package dimensions implements the three scored dimensions of the τ operator:
// D-SENS, D-AUTORITÉ, and D-INVARIANT (chap. III.8.4).
//
// Each dimension exposes a Score function that aggregates its probes using
// the calibrated weights from the active Profile. Dimensions are orthogonal
// in value (scores are independent); they are coupled only by the I4 coherence
// constraint enforced at the orchestration layer.
//
// Architecture rule: this package may import internal/tau and internal/bridge/llm
// but must NOT import internal/orchestration or any other dimension package.
package dimensions
```

- [ ] **Étape 2 — Écrire le test rouge `internal/tau/dimensions/dsens_test.go`**

```go
package dimensions_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

// probeWeights returns weights summing to 1.0 for D-SENS per PRD §5.1.
func sensWeights() dimensions.SensWeights {
	return dimensions.SensWeights{
		Contract:         0.35,
		RuntimeResolve:   0.30,
		CapabilityDiscov: 0.20,
		ReasonerIntent:   0.15,
	}
}

func newStaticExchange() tau.Exchange {
	return tau.Exchange{
		ID:                "x-static",
		IntentDescription: "call payment service",
		DiscoveredAt:      time.Now(),
		Target: tau.Capability{
			ID:            "payment-svc",
			DiscoveryMode: tau.Static,
			ContractURI:   "https://api.example.com/openapi.yaml",
		},
		Initiator: tau.Principal{
			ID:          "agent-1",
			HumanInLoop: true,
			Organization: "org-a",
		},
	}
}

func newDynamicExchange() tau.Exchange {
	return tau.Exchange{
		ID:                "x-dynamic",
		IntentDescription: "discover and invoke best available tool",
		DiscoveredAt:      time.Now(),
		Target: tau.Capability{
			ID:            "",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
		Initiator: tau.Principal{
			ID:          "agent-2",
			HumanInLoop: false,
			Organization: "org-b",
		},
	}
}

func TestDSens_Bounded(t *testing.T) {
	t.Parallel()
	w := sensWeights()
	cases := []tau.Exchange{newStaticExchange(), newDynamicExchange()}
	for _, x := range cases {
		x := x
		t.Run(x.ID, func(t *testing.T) {
			t.Parallel()
			score, err := dimensions.ScoreDSens(context.Background(), x, w, nil)
			if err != nil {
				t.Fatalf("ScoreDSens error: %v", err)
			}
			if score.Value < 0 || score.Value > 1 {
				t.Fatalf("ScoreDSens value %f out of [0,1]", score.Value)
			}
		})
	}
}

func TestDSens_StaticLowerThanDynamic(t *testing.T) {
	t.Parallel()
	w := sensWeights()
	static, err := dimensions.ScoreDSens(context.Background(), newStaticExchange(), w, nil)
	if err != nil {
		t.Fatalf("static: %v", err)
	}
	dynamic, err := dimensions.ScoreDSens(context.Background(), newDynamicExchange(), w, nil)
	if err != nil {
		t.Fatalf("dynamic: %v", err)
	}
	if static.Value >= dynamic.Value {
		t.Fatalf("expected static (%f) < dynamic (%f)", static.Value, dynamic.Value)
	}
}

func TestDSens_ProbeWeightsSumToOne(t *testing.T) {
	t.Parallel()
	w := sensWeights()
	sum := w.Contract + w.RuntimeResolve + w.CapabilityDiscov + w.ReasonerIntent
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("probe weights sum = %f, want 1.0", sum)
	}
}

func TestDSens_ProbesMapPopulated(t *testing.T) {
	t.Parallel()
	w := sensWeights()
	score, err := dimensions.ScoreDSens(context.Background(), newDynamicExchange(), w, nil)
	if err != nil {
		t.Fatalf("ScoreDSens error: %v", err)
	}
	expected := []string{"S_contract", "S_runtime_resolve", "S_capability_discovery", "S_reasoner_intent"}
	for _, k := range expected {
		if _, ok := score.Probes[k]; !ok {
			t.Errorf("probe %q missing from score.Probes", k)
		}
	}
}
```

Vérifier red phase :
```bash
go test ./internal/tau/dimensions/...
```
Attendu : `undefined: dimensions.ScoreDSens` (ou similaire).

- [ ] **Étape 3 — Écrire `internal/tau/dimensions/dsens.go`**

```go
package dimensions

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/tau"
)

// SensWeights holds the calibrated weights for the four D-SENS probes.
// The sum must equal 1.0 (enforced at construction by the calibration layer).
// Initial values from PRD §5.1: {0.35, 0.30, 0.20, 0.15}.
type SensWeights struct {
	Contract         float64 // weight for S_contract
	RuntimeResolve   float64 // weight for S_runtime_resolve
	CapabilityDiscov float64 // weight for S_capability_discovery
	ReasonerIntent   float64 // weight for S_reasoner_intent
}

// DefaultSensWeights returns the initial weights from PRD §5.1.
// Status: Hypothesis — to be corroborated on AgentMeshKafka traces in M4.
func DefaultSensWeights() SensWeights {
	return SensWeights{
		Contract:         0.35,
		RuntimeResolve:   0.30,
		CapabilityDiscov: 0.20,
		ReasonerIntent:   0.15,
	}
}

// ScoreDSens computes the D-SENS dimension score for exchange x.
// llmClient may be nil; in that case S_reasoner_intent returns 0.
// Returns a Score with Value in [0,1] and all probe values populated.
func ScoreDSens(ctx context.Context, x tau.Exchange, w SensWeights, llmClient llm.Client) (Score, error) {
	sContract := probeContract(x)
	sRuntime := probeRuntimeResolve(x)
	sDiscov := probeCapabilityDiscovery(x)
	sReasoner, err := probeReasonerIntent(ctx, x, llmClient)
	if err != nil {
		return Score{}, err
	}

	value := w.Contract*sContract +
		w.RuntimeResolve*sRuntime +
		w.CapabilityDiscov*sDiscov +
		w.ReasonerIntent*sReasoner

	return Score{
		Value: clamp01(value),
		Probes: map[string]float64{
			"S_contract":            sContract,
			"S_runtime_resolve":     sRuntime,
			"S_capability_discovery": sDiscov,
			"S_reasoner_intent":     sReasoner,
		},
		Weights: map[string]float64{
			"S_contract":            w.Contract,
			"S_runtime_resolve":     w.RuntimeResolve,
			"S_capability_discovery": w.CapabilityDiscov,
			"S_reasoner_intent":     w.ReasonerIntent,
		},
		ComputedAt: time.Now(),
	}, nil
}

// probeContract (S_contract) — presence of a published, versioned, opposable
// contract for the target capability (PRD §5.1). Returns 0 if a non-empty
// ContractURI is present (contract wired = fixed before interaction = pôle 0).
// Returns 1 if no contract (meaning is negotiated at runtime = pôle 1).
func probeContract(x tau.Exchange) float64 {
	if x.Target.ContractURI != "" {
		return 0.0
	}
	return 1.0
}

// probeRuntimeResolve (S_runtime_resolve) — runtime semantic resolution
// (embedding, NL parsing). Returns 1 if the exchange has a non-empty
// IntentDescription that suggests NL-level interpretation, 0 if intent is empty
// (implying a static protocol invocation).
func probeRuntimeResolve(x tau.Exchange) float64 {
	if x.IntentDescription == "" {
		return 0.0
	}
	return 1.0
}

// probeCapabilityDiscovery (S_capability_discovery) — dynamic discovery
// via MCP list_tools, A2A, or AGNTCY (PRD §5.1). Returns 1 if DiscoveryMode
// is anything other than Static.
func probeCapabilityDiscovery(x tau.Exchange) float64 {
	if x.Target.DiscoveryMode == tau.Static {
		return 0.0
	}
	return 1.0
}

// probeReasonerIntent (S_reasoner_intent) — probabilistic reasoner intent
// interpretation (PRD §5.1). Delegates to the LLM client's Interpret method.
// Returns 0 if llmClient is nil (no reasoner available).
func probeReasonerIntent(ctx context.Context, x tau.Exchange, c llm.Client) (float64, error) {
	if c == nil {
		return 0.0, nil
	}
	return c.Interpret(ctx, x.IntentDescription)
}
```

- [ ] **Étape 4 — Créer `internal/tau/dimensions/score.go`** (type partagé entre les trois dimensions)

```go
package dimensions

import "time"

// Score is a normalized [0,1] score for a single dimension, with full
// probe-level traceability. Used by D-SENS, D-AUTORITÉ, and D-INVARIANT.
type Score struct {
	Value      float64            // composite value in [0,1]
	Probes     map[string]float64 // individual probe values
	Weights    map[string]float64 // weights in effect at compute time
	ComputedAt time.Time
}

// clamp01 clamps v to [0, 1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
```

- [ ] **Étape 5 — Vérifier**

```bash
go test -v ./internal/tau/dimensions/...
go vet ./...
golangci-lint run ./...
```

Attendu : `TestDSens_Bounded`, `TestDSens_StaticLowerThanDynamic`, `TestDSens_ProbeWeightsSumToOne`, `TestDSens_ProbesMapPopulated` passent.

- [ ] **Étape 6 — Commit**

```bash
git add internal/tau/dimensions/
git commit -m "feat(tau/dimensions): add D-SENS scorer with 4 probes (PRD §5.1)

ScoreDSens aggregates S_contract (0.35), S_runtime_resolve (0.30),
S_capability_discovery (0.20), S_reasoner_intent (0.15). LLM client
is injected for S_reasoner_intent; nil client returns 0 (deterministic CI).
Score type shared across all three dimensions. Probe values fully traced.

Tests: bounded, static < dynamic ordering, weights sum to 1.0, probes map
populated with all four keys.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.3 — `internal/tau/dimensions/dauthority.go` + 4 sondes + tests

**Files :**
- Create: `internal/tau/dimensions/dauthority.go`
- Create: `internal/tau/dimensions/dauthority_test.go`

**Agent :** `ruflo-core:coder` (TDD)

- [ ] **Étape 1 — Écrire le test rouge `internal/tau/dimensions/dauthority_test.go`**

```go
package dimensions_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

func authorityWeights() dimensions.AuthorityWeights {
	return dimensions.AuthorityWeights{
		ChainDepth:        0.25,
		CrossOrg:          0.25,
		HumanAnchor:       0.25,
		DynamicResolution: 0.25,
	}
}

func newShortChainExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-short-chain",
		DiscoveredAt: time.Now(),
		Initiator: tau.Principal{
			ID:              "human-user",
			HumanInLoop:     true,
			Organization:    "org-a",
			DelegationDepth: 0,
		},
		Target: tau.Capability{
			ID:            "internal-svc",
			DiscoveryMode: tau.Static,
			ContractURI:   "https://internal/api",
		},
	}
}

func newLongChainExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-long-chain",
		DiscoveredAt: time.Now(),
		Initiator: tau.Principal{
			ID:              "agent-orchestrator",
			HumanInLoop:     false,
			Organization:    "org-b",
			DelegationDepth: 5,
		},
		Target: tau.Capability{
			ID:            "external-api",
			DiscoveryMode: tau.DynamicA2A,
			ContractURI:   "",
		},
	}
}

func TestDAuthority_Bounded(t *testing.T) {
	t.Parallel()
	w := authorityWeights()
	for _, x := range []tau.Exchange{newShortChainExchange(), newLongChainExchange()} {
		x := x
		t.Run(x.ID, func(t *testing.T) {
			t.Parallel()
			score, err := dimensions.ScoreDAuthority(context.Background(), x, w)
			if err != nil {
				t.Fatalf("ScoreDAuthority error: %v", err)
			}
			if score.Value < 0 || score.Value > 1 {
				t.Fatalf("ScoreDAuthority value %f out of [0,1]", score.Value)
			}
		})
	}
}

func TestDAuthority_ShortChainLowerThanLong(t *testing.T) {
	t.Parallel()
	w := authorityWeights()
	short, err := dimensions.ScoreDAuthority(context.Background(), newShortChainExchange(), w)
	if err != nil {
		t.Fatalf("short: %v", err)
	}
	long, err := dimensions.ScoreDAuthority(context.Background(), newLongChainExchange(), w)
	if err != nil {
		t.Fatalf("long: %v", err)
	}
	if short.Value >= long.Value {
		t.Fatalf("expected short-chain (%f) < long-chain (%f)", short.Value, long.Value)
	}
}

func TestDAuthority_ProbeWeightsSumToOne(t *testing.T) {
	t.Parallel()
	w := authorityWeights()
	sum := w.ChainDepth + w.CrossOrg + w.HumanAnchor + w.DynamicResolution
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("authority probe weights sum = %f, want 1.0", sum)
	}
}

func TestDAuthority_ProbesMapPopulated(t *testing.T) {
	t.Parallel()
	w := authorityWeights()
	score, err := dimensions.ScoreDAuthority(context.Background(), newLongChainExchange(), w)
	if err != nil {
		t.Fatalf("ScoreDAuthority error: %v", err)
	}
	expected := []string{"A_chain_depth", "A_cross_org", "A_human_anchor", "A_dynamic_resolution"}
	for _, k := range expected {
		if _, ok := score.Probes[k]; !ok {
			t.Errorf("probe %q missing from score.Probes", k)
		}
	}
}
```

Vérifier red phase :
```bash
go test ./internal/tau/dimensions/...
```
Attendu : `undefined: dimensions.ScoreDAuthority`.

- [ ] **Étape 2 — Écrire `internal/tau/dimensions/dauthority.go`**

```go
package dimensions

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// AuthorityWeights holds the calibrated weights for the four D-AUTORITÉ probes.
// Initial values from PRD §5.2: {0.25, 0.25, 0.25, 0.25} (equal weighting).
type AuthorityWeights struct {
	ChainDepth        float64 // weight for A_chain_depth
	CrossOrg          float64 // weight for A_cross_org
	HumanAnchor       float64 // weight for A_human_anchor (inverted probe)
	DynamicResolution float64 // weight for A_dynamic_resolution
}

// DefaultAuthorityWeights returns the initial equal weights from PRD §5.2.
// Status: Hypothesis — to be corroborated on AgentMeshKafka traces in M4.
func DefaultAuthorityWeights() AuthorityWeights {
	return AuthorityWeights{
		ChainDepth:        0.25,
		CrossOrg:          0.25,
		HumanAnchor:       0.25,
		DynamicResolution: 0.25,
	}
}

// ScoreDAuthority computes the D-AUTORITÉ dimension score for exchange x.
// Returns a Score with Value in [0,1] and all probe values populated.
// Note: the attestation check (ontological guard) is NOT performed here;
// it is enforced at the orchestration layer (PRD §4.4, step 2 of dispatch).
func ScoreDAuthority(_ context.Context, x tau.Exchange, w AuthorityWeights) (Score, error) {
	aChain := probeChainDepth(x)
	aCross := probeCrossOrg(x)
	aHuman := probeHumanAnchor(x)
	aDynamic := probeDynamicResolution(x)

	value := w.ChainDepth*aChain +
		w.CrossOrg*aCross +
		w.HumanAnchor*aHuman +
		w.DynamicResolution*aDynamic

	return Score{
		Value: clamp01(value),
		Probes: map[string]float64{
			"A_chain_depth":        aChain,
			"A_cross_org":          aCross,
			"A_human_anchor":       aHuman,
			"A_dynamic_resolution": aDynamic,
		},
		Weights: map[string]float64{
			"A_chain_depth":        w.ChainDepth,
			"A_cross_org":          w.CrossOrg,
			"A_human_anchor":       w.HumanAnchor,
			"A_dynamic_resolution": w.DynamicResolution,
		},
		ComputedAt: time.Now(),
	}, nil
}

// probeChainDepth (A_chain_depth) — delegation chain depth. Returns a
// normalized value in [0,1] using a saturation function: depth 0 = 0.0,
// depth 1 = 0.25, depth 2 = 0.50, depth >= 4 = 1.0.
func probeChainDepth(x tau.Exchange) float64 {
	d := x.Initiator.DelegationDepth
	if d <= 0 {
		return 0.0
	}
	if d >= 4 {
		return 1.0
	}
	return float64(d) / 4.0
}

// probeCrossOrg (A_cross_org) — whether the exchange crosses an organizational
// boundary. Returns 1 if Initiator.Organization differs from Target.ID domain
// heuristic, or if Organization is empty (unknown = assumed cross-org).
// Simplified: returns 1 if Initiator.Organization == "" or DelegationDepth > 0
// with no explicit same-org marker. For V1, returns 1 when Organization is empty
// or DelegationDepth > 1 (implying multi-hop cross-org delegation).
func probeCrossOrg(x tau.Exchange) float64 {
	if x.Initiator.Organization == "" {
		return 1.0
	}
	if x.Initiator.DelegationDepth > 1 {
		return 1.0
	}
	return 0.0
}

// probeHumanAnchor (A_human_anchor) — inverted probe: human in the loop
// reduces the D-AUTORITÉ score (short chain, anchored authority = pôle 0).
// Returns 0 if HumanInLoop is true (human anchor present), 1 if absent.
func probeHumanAnchor(x tau.Exchange) float64 {
	if x.Initiator.HumanInLoop {
		return 0.0
	}
	return 1.0
}

// probeDynamicResolution (A_dynamic_resolution) — authority resolved at
// runtime rather than pre-wired. Returns 1 if DiscoveryMode != Static
// (capability identity itself resolved dynamically = authority chain unknown
// at design time).
func probeDynamicResolution(x tau.Exchange) float64 {
	if x.Target.DiscoveryMode == tau.Static {
		return 0.0
	}
	return 1.0
}
```

- [ ] **Étape 3 — Vérifier**

```bash
go test -v ./internal/tau/dimensions/...
go vet ./...
golangci-lint run ./...
```

Attendu : tous les tests `TestDAuthority_*` passent, les tests `TestDSens_*` de M2.2 restent verts.

- [ ] **Étape 4 — Commit**

```bash
git add internal/tau/dimensions/dauthority.go internal/tau/dimensions/dauthority_test.go
git commit -m "feat(tau/dimensions): add D-AUTORITÉ scorer with 4 probes (PRD §5.2)

ScoreDAuthority aggregates A_chain_depth (0.25), A_cross_org (0.25),
A_human_anchor inverted (0.25), A_dynamic_resolution (0.25). Equal
initial weights per PRD §5.2 — hypothesis, pending M4 empirical validation.

Note: the ontological guard (D-AUTORITÉ >= AuthBlock && Attestation == nil
=> Refus) is enforced at the orchestration layer (M2.5), not here.

Tests: bounded, short-chain < long-chain ordering, weights sum to 1.0,
probes map populated with all four keys.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.4 — `internal/tau/dimensions/dinvariant.go` + 4 sondes + tests

**Files :**
- Create: `internal/tau/dimensions/dinvariant.go`
- Create: `internal/tau/dimensions/dinvariant_test.go`

**Agent :** `ruflo-core:coder` (TDD)

- [ ] **Étape 1 — Écrire le test rouge `internal/tau/dimensions/dinvariant_test.go`**

```go
package dimensions_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

func invariantWeights() dimensions.InvariantWeights {
	return dimensions.InvariantWeights{
		EventRegistry:       0.30,
		IdempotencyDerived:  0.25,
		CapabilityMediation: 0.25,
		EnumeratedPlan:      0.20,
	}
}

func newFrozenSupportExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-frozen-support",
		DiscoveredAt: time.Now(),
		Target: tau.Capability{
			ID:            "batch-processor",
			DiscoveryMode: tau.Static,
			ContractURI:   "https://api.example.com/batch/v1",
		},
		Initiator: tau.Principal{
			ID:              "scheduler",
			HumanInLoop:     true,
			Organization:    "org-a",
			DelegationDepth: 0,
		},
		// No context: implies enumerated plan at design time
	}
}

func newTracedSupportExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-traced-support",
		DiscoveredAt: time.Now(),
		Target: tau.Capability{
			ID:            "dynamic-tool",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
		Initiator: tau.Principal{
			ID:              "llm-agent",
			HumanInLoop:     false,
			Organization:    "org-b",
			DelegationDepth: 3,
		},
		Context: map[string]any{
			"event_registry":        true,
			"idempotency_key_mode":  "derived",
			"capability_mediation":  true,
		},
	}
}

func TestDInvariant_Bounded(t *testing.T) {
	t.Parallel()
	w := invariantWeights()
	for _, x := range []tau.Exchange{newFrozenSupportExchange(), newTracedSupportExchange()} {
		x := x
		t.Run(x.ID, func(t *testing.T) {
			t.Parallel()
			score, err := dimensions.ScoreDInvariant(context.Background(), x, w)
			if err != nil {
				t.Fatalf("ScoreDInvariant error: %v", err)
			}
			if score.Value < 0 || score.Value > 1 {
				t.Fatalf("ScoreDInvariant value %f out of [0,1]", score.Value)
			}
		})
	}
}

func TestDInvariant_FrozenLowerThanTraced(t *testing.T) {
	t.Parallel()
	w := invariantWeights()
	frozen, err := dimensions.ScoreDInvariant(context.Background(), newFrozenSupportExchange(), w)
	if err != nil {
		t.Fatalf("frozen: %v", err)
	}
	traced, err := dimensions.ScoreDInvariant(context.Background(), newTracedSupportExchange(), w)
	if err != nil {
		t.Fatalf("traced: %v", err)
	}
	if frozen.Value >= traced.Value {
		t.Fatalf("expected frozen (%f) < traced (%f)", frozen.Value, traced.Value)
	}
}

func TestDInvariant_ProbeWeightsSumToOne(t *testing.T) {
	t.Parallel()
	w := invariantWeights()
	sum := w.EventRegistry + w.IdempotencyDerived + w.CapabilityMediation + w.EnumeratedPlan
	const eps = 1e-9
	if sum < 1.0-eps || sum > 1.0+eps {
		t.Fatalf("invariant probe weights sum = %f, want 1.0", sum)
	}
}

func TestDInvariant_ProbesMapPopulated(t *testing.T) {
	t.Parallel()
	w := invariantWeights()
	score, err := dimensions.ScoreDInvariant(context.Background(), newTracedSupportExchange(), w)
	if err != nil {
		t.Fatalf("ScoreDInvariant error: %v", err)
	}
	expected := []string{
		"I_event_registry",
		"I_idempotency_derived",
		"I_capability_mediation",
		"I_enumerated_plan",
	}
	for _, k := range expected {
		if _, ok := score.Probes[k]; !ok {
			t.Errorf("probe %q missing from score.Probes", k)
		}
	}
}
```

Vérifier red phase :
```bash
go test ./internal/tau/dimensions/...
```
Attendu : `undefined: dimensions.ScoreDInvariant`.

- [ ] **Étape 2 — Écrire `internal/tau/dimensions/dinvariant.go`**

```go
package dimensions

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// InvariantWeights holds the calibrated weights for the four D-INVARIANT probes.
// Initial values from PRD §5.3: {0.30, 0.25, 0.25, 0.20}.
type InvariantWeights struct {
	EventRegistry       float64 // weight for I_event_registry
	IdempotencyDerived  float64 // weight for I_idempotency_derived
	CapabilityMediation float64 // weight for I_capability_mediation
	EnumeratedPlan      float64 // weight for I_enumerated_plan (inverted probe)
}

// DefaultInvariantWeights returns the initial weights from PRD §5.3.
// Status: Hypothesis — to be corroborated on AgentMeshKafka traces in M4.
func DefaultInvariantWeights() InvariantWeights {
	return InvariantWeights{
		EventRegistry:       0.30,
		IdempotencyDerived:  0.25,
		CapabilityMediation: 0.25,
		EnumeratedPlan:      0.20,
	}
}

// ScoreDInvariant computes the D-INVARIANT dimension score for exchange x.
// Returns a Score with Value in [0,1] and all probe values populated.
// I4 coherence constraint (D-INVARIANT constrained by D-SENS) is enforced
// at the orchestration layer (step 5), not here.
func ScoreDInvariant(_ context.Context, x tau.Exchange, w InvariantWeights) (Score, error) {
	iRegistry := probeEventRegistry(x)
	iIdempotency := probeIdempotencyDerived(x)
	iMediation := probeCapabilityMediation(x)
	iEnumerated := probeEnumeratedPlan(x)

	value := w.EventRegistry*iRegistry +
		w.IdempotencyDerived*iIdempotency +
		w.CapabilityMediation*iMediation +
		w.EnumeratedPlan*iEnumerated

	return Score{
		Value: clamp01(value),
		Probes: map[string]float64{
			"I_event_registry":       iRegistry,
			"I_idempotency_derived":  iIdempotency,
			"I_capability_mediation": iMediation,
			"I_enumerated_plan":      iEnumerated,
		},
		Weights: map[string]float64{
			"I_event_registry":       w.EventRegistry,
			"I_idempotency_derived":  w.IdempotencyDerived,
			"I_capability_mediation": w.CapabilityMediation,
			"I_enumerated_plan":      w.EnumeratedPlan,
		},
		ComputedAt: time.Now(),
	}, nil
}

// probeEventRegistry (I_event_registry) — runtime-traced effect registry.
// Returns 1 if Context contains key "event_registry" with truthy bool value.
func probeEventRegistry(x tau.Exchange) float64 {
	if v, ok := x.Context["event_registry"]; ok {
		if b, isBool := v.(bool); isBool && b {
			return 1.0
		}
	}
	return 0.0
}

// probeIdempotencyDerived (I_idempotency_derived) — idempotency key derived
// from intent vs imposed at design time. Returns 1 if Context contains
// "idempotency_key_mode" == "derived".
func probeIdempotencyDerived(x tau.Exchange) float64 {
	if v, ok := x.Context["idempotency_key_mode"]; ok {
		if s, isStr := v.(string); isStr && s == "derived" {
			return 1.0
		}
	}
	return 0.0
}

// probeCapabilityMediation (I_capability_mediation) — capability mediation
// negotiated during the exchange. Returns 1 if Context contains
// "capability_mediation" with truthy bool value, or if DiscoveryMode != Static
// (dynamic discovery implies runtime mediation).
func probeCapabilityMediation(x tau.Exchange) float64 {
	if v, ok := x.Context["capability_mediation"]; ok {
		if b, isBool := v.(bool); isBool && b {
			return 1.0
		}
	}
	if x.Target.DiscoveryMode != tau.Static {
		return 1.0
	}
	return 0.0
}

// probeEnumeratedPlan (I_enumerated_plan) — inverted probe: an enumerated
// step plan known at design time reduces D-INVARIANT (support frozen at design
// time = pôle 0). Returns 0 if Context contains "enumerated_plan" == true,
// 1 if absent or false. Also returns 0 if ContractURI is present (contract
// implies pre-defined plan).
func probeEnumeratedPlan(x tau.Exchange) float64 {
	if x.Target.ContractURI != "" {
		return 0.0
	}
	if v, ok := x.Context["enumerated_plan"]; ok {
		if b, isBool := v.(bool); isBool && b {
			return 0.0
		}
	}
	return 1.0
}
```

- [ ] **Étape 3 — Vérifier**

```bash
go test -v ./internal/tau/dimensions/...
go vet ./...
golangci-lint run ./...
```

Attendu : tous les tests `TestDInvariant_*` passent. Suite complète verte.

- [ ] **Étape 4 — Commit**

```bash
git add internal/tau/dimensions/dinvariant.go internal/tau/dimensions/dinvariant_test.go
git commit -m "feat(tau/dimensions): add D-INVARIANT scorer with 4 probes (PRD §5.3)

ScoreDInvariant aggregates I_event_registry (0.30), I_idempotency_derived
(0.25), I_capability_mediation (0.25), I_enumerated_plan inverted (0.20).
Probe values read from Exchange.Context map entries and Target.DiscoveryMode.
I4 coherence constraint enforced at orchestration layer (M2.6), not here.

Tests: bounded, frozen < traced ordering, weights sum to 1.0, probes map
populated with all four keys.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.5 — Garde ontologique D-AUTORITÉ (étape 2 du dispatcher) + test

**Files :**
- Modify: `internal/orchestration/thresholds.go`
- Modify: `internal/orchestration/dispatcher.go`
- Create: `internal/orchestration/guards_test.go`

**Agent :** `ruflo-core:coder` (TDD)

- [ ] **Étape 1 — Étendre `internal/orchestration/thresholds.go`**

Ajouter `AuthBlock`, `SensCoherence`, `InvCoherence` à `Thresholds` :

```go
package orchestration

// Thresholds holds the complete set of decision thresholds for the dispatcher.
// M1 had Deterministe and Probabiliste only; M2 adds the guard thresholds.
type Thresholds struct {
	Deterministe  float64 // tau_score < theta -> Deterministe
	Probabiliste  float64 // tau_score >= theta -> Probabiliste
	AuthBlock     float64 // D-AUTORITÉ >= AuthBlock && Attestation==nil -> Refus (I3)
	SensCoherence float64 // I4 guard: D-SENS must be >= SensCoherence when D-INVARIANT >= InvCoherence
	InvCoherence  float64 // I4 guard: D-INVARIANT threshold that triggers the coherence check
}

// Ordered reports the ordering invariant.
// Must hold at all times: Deterministe <= Probabiliste.
func (t Thresholds) Ordered() bool { return t.Deterministe <= t.Probabiliste }

// DefaultThresholds returns the initial thresholds from PRD §11.1.
// Status: Hypothesis — to be corroborated by M4 empirical calibration.
func DefaultThresholds() Thresholds {
	return Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	}
}
```

- [ ] **Étape 2 — Écrire le test rouge `internal/orchestration/guards_test.go`**

```go
package orchestration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/orchestration"
	"github.com/agbruneau/taugo/internal/tau"
)

// highAuthorityExchange returns an exchange with D-AUTORITÉ likely above AuthBlock
// and no attestation: should trigger the ontological guard.
func highAuthorityExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-high-authority",
		DiscoveredAt: time.Now(),
		IntentDescription: "invoke external financial system without human oversight",
		Initiator: tau.Principal{
			ID:              "sub-agent-3",
			HumanInLoop:     false,
			Organization:    "",
			DelegationDepth: 5,
		},
		Target: tau.Capability{
			ID:            "external-fin",
			DiscoveryMode: tau.DynamicA2A,
			ContractURI:   "",
		},
		// AttestationInstitutionnelle intentionally nil
	}
}

// TestRefusOntologiqueDAUTORITE verifies that an exchange whose D-AUTORITÉ
// score reaches or exceeds AuthBlock without an institutional attestation
// is refused with diagnostic "I3 — verrou ontologique D-AUTORITÉ".
func TestRefusOntologiqueDAUTORITE(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
	x := highAuthorityExchange()
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Refus {
		t.Fatalf("regime = %v, want Refus (ontological guard should fire)", dec.Regime)
	}
	if dec.Diagnostic != "I3 — verrou ontologique D-AUTORITÉ" {
		t.Fatalf("diagnostic = %q, want \"I3 — verrou ontologique D-AUTORITÉ\"", dec.Diagnostic)
	}
}

// TestOntologicalGuardPassesWithAttestation verifies that the same high-authority
// exchange is NOT refused when an attestation is present.
func TestOntologicalGuardPassesWithAttestation(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
	x := highAuthorityExchange()
	x.AttestationInstitutionnelle = &tau.Attestation{
		Emetteur:   "IETF",
		Reference:  "draft-identity-delegation-00",
		Marqueur:   "Hypothèse",
		AssertedAt: time.Now(),
	}
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With attestation, the ontological guard must NOT fire.
	if dec.Regime == tau.Refus && dec.Diagnostic == "I3 — verrou ontologique D-AUTORITÉ" {
		t.Fatal("ontological guard fired despite attestation being present")
	}
}
```

Vérifier red phase (tests compilent mais échouent car le dispatcher ne calcule pas encore D-AUTORITÉ) :
```bash
go test -run TestRefusOntologiqueDAUTORITE ./internal/orchestration/...
```

- [ ] **Étape 3 — Refactorer `internal/orchestration/dispatcher.go`**

Remplacer le contenu complet par la version M2 qui :
1. Dérive `FrontierCheck` depuis l'`Exchange` (heuristique documentée)
2. Calcule D-AUTORITÉ pour la garde ontologique (étape 2)
3. Calcule D-SENS et D-INVARIANT pour la garde I4 (étape 5)
4. Remplace le composite naïf par le composite pondéré (étape 6)

```go
package orchestration

import (
	"context"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/llm"
	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/dimensions"
)

// defaultDimensionWeights holds the composite weights for tau_score = w_s*D_SENS + w_a*D_AUTH + w_i*D_INV.
// Initial values per PRD §11.1: (0.4, 0.3, 0.3). Status: Hypothesis.
var defaultDimensionWeights = struct{ DSens, DAuthority, DInvariant float64 }{
	DSens:      0.4,
	DAuthority: 0.3,
	DInvariant: 0.3,
}

// Dispatcher implements the M2 subset of the τ pseudo-algorithm (PRD §10):
// steps 1 (frontier), 2 (ontological guard D-AUTORITÉ), 4 (dimension scores),
// 5 (I4 coherence guard), 6 (weighted composite), and 7 (hysteresis decision).
// Steps 3 (profile expiration) and 8 (invariant evaluation) land in M3/M5.
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

// durationNs returns elapsed nanoseconds since start, guaranteeing at least 1
// to satisfy the Trace.DurationNs > 0 invariant on platforms (e.g. Windows)
// where the timer resolution may be coarser than 1 ns.
func durationNs(start time.Time) int64 {
	if d := time.Since(start).Nanoseconds(); d > 0 {
		return d
	}
	return 1
}

// Decide implements the M2 subset of PRD §10 (steps 1, 2, 4, 5, 6, 7).
func (d *Dispatcher) Decide(ctx context.Context, x tau.Exchange) (tau.Decision, error) {
	start := time.Now()

	traceThresholds := tau.TraceThresholds{
		Deterministe:  d.thresholds.Deterministe,
		Probabiliste:  d.thresholds.Probabiliste,
		AuthBlock:     d.thresholds.AuthBlock,
		SensCoherence: d.thresholds.SensCoherence,
		InvCoherence:  d.thresholds.InvCoherence,
	}

	// Step 1 — Frontier check derived from Exchange (M2: heuristic from
	// Capability.DiscoveryMode and Principal.HumanInLoop).
	frontier := frontierFromExchange(x)
	if !frontier.Inside() {
		return tau.Decision{
			Regime:     tau.Refus,
			Diagnostic: "hors frontière τ",
			Trace: tau.Trace{
				ExchangeID: x.ID,
				Frontier:   frontier,
				Thresholds: traceThresholds,
				DurationNs: durationNs(start),
			},
		}, nil
	}

	// Step 2 — Ontological guard D-AUTORITÉ (PRD §4.4, I3).
	authScore, err := dimensions.ScoreDAuthority(ctx, x, dimensions.DefaultAuthorityWeights())
	if err != nil {
		return tau.Decision{}, err
	}
	if authScore.Value >= d.thresholds.AuthBlock && x.AttestationInstitutionnelle == nil {
		return tau.Decision{
			Regime:     tau.Refus,
			Diagnostic: "I3 — verrou ontologique D-AUTORITÉ",
			Trace: tau.Trace{
				ExchangeID: x.ID,
				Frontier:   frontier,
				Thresholds: traceThresholds,
				DurationNs: durationNs(start),
			},
		}, nil
	}

	// Step 4 — Dimension scores (D-SENS and D-INVARIANT; D-AUTORITÉ already computed).
	sensScore, err := dimensions.ScoreDSens(ctx, x, dimensions.DefaultSensWeights(), d.llm)
	if err != nil {
		return tau.Decision{}, err
	}
	invScore, err := dimensions.ScoreDInvariant(ctx, x, dimensions.DefaultInvariantWeights())
	if err != nil {
		return tau.Decision{}, err
	}

	// Step 5 — I4 coherence guard (PRD §6.1):
	// D-INVARIANT >= InvCoherence AND D-SENS < SensCoherence => incoherent combination.
	if invScore.Value >= d.thresholds.InvCoherence && sensScore.Value < d.thresholds.SensCoherence {
		return tau.Decision{
			Regime:     tau.Refus,
			Diagnostic: "I4 — combinaison incohérente détectée",
			Trace: tau.Trace{
				ExchangeID: x.ID,
				Frontier:   frontier,
				Thresholds: traceThresholds,
				DurationNs: durationNs(start),
			},
		}, nil
	}

	// Step 6 — Weighted composite tau_score.
	tauScore := defaultDimensionWeights.DSens*sensScore.Value +
		defaultDimensionWeights.DAuthority*authScore.Value +
		defaultDimensionWeights.DInvariant*invScore.Value

	// Step 7 — Decision with hysteresis (M2: same default as M1 — Deterministe in the band).
	var regime tau.Regime
	switch {
	case tauScore >= d.thresholds.Probabiliste:
		regime = tau.Probabiliste
	default:
		// Covers tauScore < Deterministe and the hysteresis zone.
		// M2 default: Deterministe. Regime history tracking deferred to M5.
		regime = tau.Deterministe
	}

	return tau.Decision{
		Regime:         regime,
		ProfileVersion: "M2-default",
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   tauScore,
			Frontier:   frontier,
			Thresholds: traceThresholds,
			DurationNs: durationNs(start),
		},
	}, nil
}

// frontierFromExchange derives a FrontierCheck from the Exchange fields.
// This is a placeholder heuristic until M5 empirical calibration.
// Rules (all placeholder, documented as such):
//   - Target.DiscoveryMode != Static  => UniversOuvert=true, CompositionVariable=true
//   - !Initiator.HumanInLoop          => PairProbabiliste=true
//   - Initiator.DelegationDepth > 0   => CoutNonBorne=true
func frontierFromExchange(x tau.Exchange) tau.FrontierCheck {
	dynamic := x.Target.DiscoveryMode != tau.Static
	return tau.FrontierCheck{
		UniversOuvert:       dynamic,
		CompositionVariable: dynamic,
		PairProbabiliste:    !x.Initiator.HumanInLoop,
		CoutNonBorne:        x.Initiator.DelegationDepth > 0,
	}
}
```

- [ ] **Étape 4 — Vérifier que tous les tests existants passent encore**

```bash
go test -v ./internal/orchestration/...
go test -v ./internal/tau/...
go vet ./...
golangci-lint run ./...
```

Attendu : `TestRefusOntologiqueDAUTORITE` vert, tous les tests M1 (`TestDispatcher_*`, `TestDecisionAlways*`, `TestRefusImplies*`, `TestTrace*`) restent verts.

**Note pour l'agent** : les tests M1 utilisent `fakeLLM{score: X}` et `newExchangeInsideFrontier` qui crée un `Exchange{}` sans champs `Initiator`/`Target`. Avec le dispatcher M2, `frontierFromExchange({})` retourne `FrontierCheck{false, false, false, false}` → `Inside() == false` → `Refus`. Les tests M1 `TestDispatcher_Decide_Deterministe` et `TestDispatcher_Decide_Probabiliste` échoueront donc si l'exchange reste vide.

Mettre à jour `internal/orchestration/dispatcher_test.go` pour que `newExchangeInsideFrontier` retourne un exchange dont le frontier heuristique est `Inside() == true` :

```go
func newExchangeInsideFrontier(id string) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: "test intent",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent",
			HumanInLoop:     false,   // => PairProbabiliste=true
			Organization:    "org-x",
			DelegationDepth: 1,       // => CoutNonBorne=true
		},
		Target: tau.Capability{
			ID:            "target-svc",
			DiscoveryMode: tau.DynamicMCP, // => UniversOuvert=true, CompositionVariable=true
			ContractURI:   "",
		},
	}
}
```

- [ ] **Étape 5 — Commit**

```bash
git add internal/orchestration/thresholds.go internal/orchestration/dispatcher.go internal/orchestration/dispatcher_test.go internal/orchestration/guards_test.go
git commit -m "feat(orchestration): M2 dispatcher — ontological guard D-AUTORITÉ (step 2)

Replaces M1 naive LLM-direct composite with the 3-dimension weighted scorer
(w_s=0.4, w_a=0.3, w_i=0.3, PRD §11.1 hypothesis). Adds:

- frontierFromExchange: heuristic frontier derivation from Exchange fields
  (DiscoveryMode, HumanInLoop, DelegationDepth) — placeholder until M5.
- Step 2: D-AUTORITÉ >= AuthBlock (0.85) && Attestation==nil => Refus(I3).
  TestRefusOntologiqueDAUTORITE passes. TestOntologicalGuardPassesWithAttestation
  verifies attestation lifts the guard.
- Thresholds extended with AuthBlock, SensCoherence, InvCoherence and DefaultThresholds().

dispatcher_test.go: newExchangeInsideFrontier updated to produce an exchange
whose M2 frontier heuristic yields Inside()==true.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.6 — Garde I4 cohérence (étape 5 du dispatcher) + test

**Files :**
- Modify: `internal/orchestration/guards_test.go`

**Agent :** `ruflo-core:coder`

La garde I4 est déjà implémentée dans le dispatcher en M2.5 (étape 5). Cette tâche ajoute uniquement le test dédié.

- [ ] **Étape 1 — Ajouter `TestI4_IncoherenceDetectee` dans `guards_test.go`**

Appendre à la fin du fichier (après `TestOntologicalGuardPassesWithAttestation`) :

```go
// coherentInvariantHighSensExchange returns an exchange where D-INVARIANT is high
// but D-SENS is also high — coherent, must NOT trigger I4.
func coherentInvariantHighSensExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-coherent-i4",
		DiscoveredAt: time.Now(),
		IntentDescription: "dynamically negotiate and execute plan",
		Initiator: tau.Principal{
			ID:              "llm-orchestrator",
			HumanInLoop:     false,
			Organization:    "org-c",
			DelegationDepth: 2,
		},
		Target: tau.Capability{
			ID:            "adaptive-tool",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "",
		},
		Context: map[string]any{
			"event_registry":       true,
			"idempotency_key_mode": "derived",
			"capability_mediation": true,
		},
	}
}

// incoherentExchange returns an exchange where D-INVARIANT is high (dynamic
// support) but D-SENS is low (static contract) — incoherent per I4, must Refus.
func incoherentExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-incoherent-i4",
		DiscoveredAt: time.Now(),
		// No IntentDescription => S_runtime_resolve = 0; static contract => S_contract = 0
		Initiator: tau.Principal{
			ID:              "system-scheduler",
			HumanInLoop:     false,
			Organization:    "org-d",
			DelegationDepth: 2,
		},
		Target: tau.Capability{
			ID:            "static-svc",
			DiscoveryMode: tau.DynamicMCP, // dynamic => I_capability_mediation = 1
			ContractURI:   "https://api.example.com/v1", // static contract => S_contract = 0
		},
		Context: map[string]any{
			"event_registry":       true,
			"idempotency_key_mode": "derived",
			"capability_mediation": true,
			// enumerated_plan absent => I_enumerated_plan = 1
		},
		// High D-INVARIANT (tracé dynamique) but S_contract=0, no IntentDescription => low D-SENS
	}
}

// TestI4_IncoherenceDetectee verifies that an exchange with D-INVARIANT >=
// InvCoherence (0.50) and D-SENS < SensCoherence (0.50) is refused with
// diagnostic "I4 — combinaison incohérente détectée".
func TestI4_IncoherenceDetectee(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
	x := incoherentExchange()
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime != tau.Refus {
		t.Logf("D-SENS score may not be low enough with current fixtures; regime = %v", dec.Regime)
		t.Fatalf("expected Refus (I4 guard), got %v", dec.Regime)
	}
	if dec.Diagnostic != "I4 — combinaison incohérente détectée" {
		t.Fatalf("diagnostic = %q, want \"I4 — combinaison incohérente détectée\"", dec.Diagnostic)
	}
}

// TestI4_CoherentCombinationAccepted verifies that a coherent exchange
// (high D-INVARIANT AND high D-SENS) is not refused by the I4 guard.
func TestI4_CoherentCombinationAccepted(t *testing.T) {
	t.Parallel()
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
	x := coherentInvariantHighSensExchange()
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Regime == tau.Refus && dec.Diagnostic == "I4 — combinaison incohérente détectée" {
		t.Fatal("I4 guard fired on a coherent exchange")
	}
}
```

**Note pour l'agent** : l'exchange `incoherentExchange()` doit produire `D-INVARIANT >= 0.50` et `D-SENS < 0.50`. Vérifier avec des prints de debug avant de figer les fixtures si les tests échouent. Calculer manuellement :
- `D-INVARIANT`: `I_event_registry=1 (0.30) + I_idempotency_derived=1 (0.25) + I_capability_mediation=1 (0.25) + I_enumerated_plan=0 (0.20) = 0.80` → >= 0.50. OK.
- `D-SENS`: `S_contract=0 (0.35) + S_runtime_resolve=0 (0.30, no IntentDescription) + S_capability_discovery=1 (0.20, DynamicMCP) + S_reasoner_intent=stub(0.30 pour "" ou proche) * 0.15`. Le stub FNV-1a pour `""` retourne `float64(2166136261 % 1000) / 1000 = 0.261`. Score D-SENS = `0 + 0 + 0.20 + 0.261*0.15 = 0.239` → < 0.50. OK.

Si le test `TestI4_IncoherenceDetectee` échoue parce que l'ontological guard (étape 2) se déclenche en premier (exchange sans attestation, délégation 2), vérifier la valeur D-AUTORITÉ : `A_chain_depth=0.5 (depth=2/4) + A_cross_org=1 (depth>1) + A_human_anchor=1 (no human) + A_dynamic_resolution=1 (DynamicMCP) = 3.5/4 = 0.875 ≥ AuthBlock(0.85)`. L'exchange `incoherentExchange` déclenchera donc l'ontological guard avant I4. Corriger `incoherentExchange` pour qu'il porte une attestation mais reste incoherent :

```go
// Correction: ajouter une attestation pour bypasser le garde ontologique
// tout en gardant la combinaison D-INVARIANT élevé / D-SENS bas.
func incoherentExchange() tau.Exchange {
	return tau.Exchange{
		ID:           "x-incoherent-i4",
		DiscoveredAt: time.Now(),
		AttestationInstitutionnelle: &tau.Attestation{
			Emetteur:   "IETF",
			Reference:  "draft-delegation-scope",
			Marqueur:   "Hypothèse",
			AssertedAt: time.Now(),
		},
		Initiator: tau.Principal{
			ID:              "system-scheduler",
			HumanInLoop:     false,
			Organization:    "org-d",
			DelegationDepth: 2,
		},
		Target: tau.Capability{
			ID:            "static-svc",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1",
		},
		Context: map[string]any{
			"event_registry":       true,
			"idempotency_key_mode": "derived",
			"capability_mediation": true,
		},
	}
}
```

- [ ] **Étape 2 — Vérifier**

```bash
go test -v -run TestI4 ./internal/orchestration/...
go test -v -run TestRefus ./internal/orchestration/...
go test -v ./internal/orchestration/...
```

Attendu : `TestI4_IncoherenceDetectee` vert, `TestI4_CoherentCombinationAccepted` vert, tous les autres tests verts.

- [ ] **Étape 3 — Commit**

```bash
git add internal/orchestration/guards_test.go
git commit -m "test(orchestration): add I4 coherence guard tests (PRD §6.1)

TestI4_IncoherenceDetectee: exchange with D-INVARIANT >= 0.50 and
D-SENS < 0.50 (static contract, no IntentDescription) is refused with
'I4 — combinaison incohérente détectée'. Attestation present to bypass
the ontological guard so I4 is the active gate.

TestI4_CoherentCombinationAccepted: exchange with both dimensions high
passes the I4 guard without Refus.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.7 — `internal/calibration/profile.go`

**Files :**
- Create: `internal/calibration/profile.go`
- Create: `internal/calibration/profile_test.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Écrire `internal/calibration/profile.go`**

```go
package calibration

import "time"

// Weights holds the composite and per-probe weights for the three dimensions.
// DSens + DAuthority + DInvariant must sum to 1.0.
// Each probe map (SensProbes, AuthorityProbes, InvariantProbes) must sum to 1.0.
type Weights struct {
	DSens      float64            `json:"d_sens"`
	DAuthority float64            `json:"d_authority"`
	DInvariant float64            `json:"d_invariant"`
	SensProbes      map[string]float64 `json:"sens_probes,omitempty"`
	AuthorityProbes map[string]float64 `json:"authority_probes,omitempty"`
	InvariantProbes map[string]float64 `json:"invariant_probes,omitempty"`
}

// Thresholds holds the full set of calibrated decision thresholds.
// Mirrors orchestration.Thresholds; separate to avoid calibration -> orchestration import.
type Thresholds struct {
	Deterministe  float64 `json:"deterministe"`
	Probabiliste  float64 `json:"probabiliste"`
	AuthBlock     float64 `json:"auth_block"`
	SensCoherence float64 `json:"sens_coherence"`
	InvCoherence  float64 `json:"inv_coherence"`
	HysteresisGap float64 `json:"hysteresis_gap"`
}

// Profile is the versioned, opposable calibration record for the τ operator.
// Every Profile carries fingerprints of the environment in which it was produced;
// a changed fingerprint invalidates the profile (PRD §11.4).
type Profile struct {
	ID                  string     `json:"id"`
	Version             string     `json:"version"`
	CreatedAt           time.Time  `json:"created_at"`
	DateRevision        time.Time  `json:"date_revision"`         // expiry date (PRD §7.1 C3)
	VersionMonographie  string     `json:"version_monographie"`   // pinned monograph tag
	CPUFingerprint      string     `json:"cpu_fingerprint"`
	ModelLLMFingerprint string     `json:"model_llm_fingerprint"`
	CorpusFingerprint   string     `json:"corpus_fingerprint"`
	Thresholds          Thresholds `json:"thresholds"`
	Weights             Weights    `json:"weights"`
}

// DefaultProfile returns the initial profile with PRD §11.1 values.
// Status: Hypothesis — thresholds and weights to be corroborated in M4/M5.
func DefaultProfile() Profile {
	now := time.Now().UTC()
	// DateRevision: 6 months ahead per PRD §11.4 minimum. Initial value 2026-11-23.
	dateRevision := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	return Profile{
		ID:                  "default",
		Version:             "0.1.0",
		CreatedAt:           now,
		DateRevision:        dateRevision,
		VersionMonographie:  "v2.4.3",
		CPUFingerprint:      "",
		ModelLLMFingerprint: "stub:v0",
		CorpusFingerprint:   "",
		Thresholds: Thresholds{
			Deterministe:  0.35,
			Probabiliste:  0.65,
			AuthBlock:     0.85,
			SensCoherence: 0.50,
			InvCoherence:  0.50,
			HysteresisGap: 0.10,
		},
		Weights: Weights{
			DSens:      0.4,
			DAuthority: 0.3,
			DInvariant: 0.3,
			SensProbes: map[string]float64{
				"S_contract":            0.35,
				"S_runtime_resolve":     0.30,
				"S_capability_discovery": 0.20,
				"S_reasoner_intent":     0.15,
			},
			AuthorityProbes: map[string]float64{
				"A_chain_depth":        0.25,
				"A_cross_org":          0.25,
				"A_human_anchor":       0.25,
				"A_dynamic_resolution": 0.25,
			},
			InvariantProbes: map[string]float64{
				"I_event_registry":       0.30,
				"I_idempotency_derived":  0.25,
				"I_capability_mediation": 0.25,
				"I_enumerated_plan":      0.20,
			},
		},
	}
}
```

- [ ] **Étape 2 — Écrire `internal/calibration/profile_test.go`**

```go
package calibration_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/calibration"
)

func TestDefaultProfile_FieldsNonEmpty(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	if p.ID == "" {
		t.Error("Profile.ID must not be empty")
	}
	if p.Version == "" {
		t.Error("Profile.Version must not be empty")
	}
	if p.VersionMonographie == "" {
		t.Error("Profile.VersionMonographie must not be empty")
	}
}

func TestDefaultProfile_DateRevisionAfterCreation(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	if !p.DateRevision.After(p.CreatedAt) {
		t.Fatalf("DateRevision (%v) must be after CreatedAt (%v)", p.DateRevision, p.CreatedAt)
	}
}

func TestDefaultProfile_DateRevisionAtLeast6MonthsAhead(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	minRevision := time.Now().UTC().AddDate(0, 6, 0)
	if p.DateRevision.Before(minRevision) {
		t.Fatalf("DateRevision %v is less than 6 months ahead of now", p.DateRevision)
	}
}

func TestDefaultProfile_WeightsSumToOne(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	const eps = 1e-9

	// Composite dimension weights
	compositeSum := p.Weights.DSens + p.Weights.DAuthority + p.Weights.DInvariant
	if compositeSum < 1.0-eps || compositeSum > 1.0+eps {
		t.Fatalf("composite weights sum = %f, want 1.0", compositeSum)
	}

	// Per-dimension probe weights
	probeSums := map[string]float64{}
	for k, v := range p.Weights.SensProbes {
		probeSums["sens"] += v
		_ = k
	}
	for k, v := range p.Weights.AuthorityProbes {
		probeSums["authority"] += v
		_ = k
	}
	for k, v := range p.Weights.InvariantProbes {
		probeSums["invariant"] += v
		_ = k
	}
	for dim, sum := range probeSums {
		if sum < 1.0-eps || sum > 1.0+eps {
			t.Errorf("%s probe weights sum = %f, want 1.0", dim, sum)
		}
	}
}

func TestDefaultProfile_ThresholdOrderingInvariant(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	if p.Thresholds.Deterministe > p.Thresholds.Probabiliste {
		t.Fatalf("Deterministe (%f) > Probabiliste (%f): ordering violated",
			p.Thresholds.Deterministe, p.Thresholds.Probabiliste)
	}
}
```

- [ ] **Étape 3 — Vérifier**

```bash
go test -v ./internal/calibration/...
go vet ./...
golangci-lint run ./...
```

Attendu : `TestDefaultProfile_*` passent.

- [ ] **Étape 4 — Commit**

```bash
git add internal/calibration/profile.go internal/calibration/profile_test.go
git commit -m "feat(calibration): add Profile, Weights, Thresholds types with DefaultProfile()

PRD §11.3: versioned, opposable calibration record. DefaultProfile() returns
the initial M2 configuration (thresholds PRD §11.1, weights PRD §5.1-5.3).
DateRevision set to 2026-11-23 (>=6 months ahead). ModelLLMFingerprint
initialised to stub:v0 matching the default LLM backend.

Tests: fields non-empty, DateRevision after CreatedAt, >= 6 months ahead,
all weight maps sum to 1.0, Deterministe <= Probabiliste invariant.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.8 — `internal/calibration/thresholds.go` avec pattern `atomic.Int64`

**Files :**
- Create: `internal/calibration/thresholds_atomic.go`
- Create: `internal/calibration/thresholds_atomic_test.go`

**Agent :** `ruflo-core:coder` (calque FibGo `bigfft/fft.go`)

- [ ] **Étape 1 — Écrire `internal/calibration/thresholds_atomic.go`**

```go
package calibration

import (
	"sync/atomic"
)

// AtomicThresholds provides lock-free, concurrency-safe access to the
// Thresholds values using atomic.Int64 with milli-unit encoding.
// This is the calque of FibGo bigfft/fft.go threshold pattern.
//
// Encoding: float64 value stored as int64(v * 1000), i.e. milli-units.
// Range: [0, 1000] for [0.0, 1.0]. Resolution: 0.001.
type AtomicThresholds struct {
	deterministe  atomic.Int64
	probabiliste  atomic.Int64
	authBlock     atomic.Int64
	sensCoherence atomic.Int64
	invCoherence  atomic.Int64
	hysteresisGap atomic.Int64
}

// NewAtomicThresholds constructs an AtomicThresholds from a Thresholds value.
// Panics if the ordering invariant Deterministe <= Probabiliste is violated.
func NewAtomicThresholds(t Thresholds) *AtomicThresholds {
	if t.Deterministe > t.Probabiliste {
		panic("calibration: AtomicThresholds ordering violated (Deterministe > Probabiliste)")
	}
	at := &AtomicThresholds{}
	at.deterministe.Store(millis(t.Deterministe))
	at.probabiliste.Store(millis(t.Probabiliste))
	at.authBlock.Store(millis(t.AuthBlock))
	at.sensCoherence.Store(millis(t.SensCoherence))
	at.invCoherence.Store(millis(t.InvCoherence))
	at.hysteresisGap.Store(millis(t.HysteresisGap))
	return at
}

// Deterministe returns the current Deterministe threshold as float64.
func (at *AtomicThresholds) Deterministe() float64 {
	return fromMillis(at.deterministe.Load())
}

// Probabiliste returns the current Probabiliste threshold as float64.
func (at *AtomicThresholds) Probabiliste() float64 {
	return fromMillis(at.probabiliste.Load())
}

// AuthBlock returns the current AuthBlock threshold as float64.
func (at *AtomicThresholds) AuthBlock() float64 {
	return fromMillis(at.authBlock.Load())
}

// SensCoherence returns the current SensCoherence threshold as float64.
func (at *AtomicThresholds) SensCoherence() float64 {
	return fromMillis(at.sensCoherence.Load())
}

// InvCoherence returns the current InvCoherence threshold as float64.
func (at *AtomicThresholds) InvCoherence() float64 {
	return fromMillis(at.invCoherence.Load())
}

// HysteresisGap returns the current HysteresisGap as float64.
func (at *AtomicThresholds) HysteresisGap() float64 {
	return fromMillis(at.hysteresisGap.Load())
}

// Snapshot returns the current values as an immutable Thresholds copy.
func (at *AtomicThresholds) Snapshot() Thresholds {
	return Thresholds{
		Deterministe:  at.Deterministe(),
		Probabiliste:  at.Probabiliste(),
		AuthBlock:     at.AuthBlock(),
		SensCoherence: at.SensCoherence(),
		InvCoherence:  at.InvCoherence(),
		HysteresisGap: at.HysteresisGap(),
	}
}

// SetTuning atomically updates all thresholds in one coordinated call.
// Panics if the ordering invariant Deterministe <= Probabiliste would be violated.
func (at *AtomicThresholds) SetTuning(t Thresholds) {
	if t.Deterministe > t.Probabiliste {
		panic("calibration: SetTuning ordering violated (Deterministe > Probabiliste)")
	}
	at.deterministe.Store(millis(t.Deterministe))
	at.probabiliste.Store(millis(t.Probabiliste))
	at.authBlock.Store(millis(t.AuthBlock))
	at.sensCoherence.Store(millis(t.SensCoherence))
	at.invCoherence.Store(millis(t.InvCoherence))
	at.hysteresisGap.Store(millis(t.HysteresisGap))
}

// millis converts a float64 in [0,1] to milli-units int64.
func millis(v float64) int64 { return int64(v * 1000) }

// fromMillis converts milli-units int64 back to float64.
func fromMillis(v int64) float64 { return float64(v) / 1000.0 }
```

- [ ] **Étape 2 — Écrire `internal/calibration/thresholds_atomic_test.go`**

```go
package calibration_test

import (
	"sync"
	"testing"

	"github.com/agbruneau/taugo/internal/calibration"
)

func defaultThresholds() calibration.Thresholds {
	return calibration.Thresholds{
		Deterministe:  0.35,
		Probabiliste:  0.65,
		AuthBlock:     0.85,
		SensCoherence: 0.50,
		InvCoherence:  0.50,
		HysteresisGap: 0.10,
	}
}

func TestAtomicThresholds_Roundtrip(t *testing.T) {
	t.Parallel()
	at := calibration.NewAtomicThresholds(defaultThresholds())
	snap := at.Snapshot()
	const eps = 0.001 // milli-unit resolution
	if snap.Deterministe < 0.35-eps || snap.Deterministe > 0.35+eps {
		t.Errorf("Deterministe = %f, want ~0.35", snap.Deterministe)
	}
	if snap.AuthBlock < 0.85-eps || snap.AuthBlock > 0.85+eps {
		t.Errorf("AuthBlock = %f, want ~0.85", snap.AuthBlock)
	}
}

func TestAtomicThresholds_OrderingInvariant(t *testing.T) {
	t.Parallel()
	at := calibration.NewAtomicThresholds(defaultThresholds())
	if at.Deterministe() > at.Probabiliste() {
		t.Fatalf("ordering violated: Deterministe (%f) > Probabiliste (%f)",
			at.Deterministe(), at.Probabiliste())
	}
}

func TestAtomicThresholds_PanicOnOrderingViolation(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on ordering violation, got none")
		}
	}()
	_ = calibration.NewAtomicThresholds(calibration.Thresholds{
		Deterministe: 0.80,
		Probabiliste: 0.20, // violates Deterministe <= Probabiliste
	})
}

func TestAtomicThresholds_SetTuning(t *testing.T) {
	t.Parallel()
	at := calibration.NewAtomicThresholds(defaultThresholds())
	newT := calibration.Thresholds{
		Deterministe:  0.40,
		Probabiliste:  0.70,
		AuthBlock:     0.90,
		SensCoherence: 0.55,
		InvCoherence:  0.55,
		HysteresisGap: 0.15,
	}
	at.SetTuning(newT)
	snap := at.Snapshot()
	const eps = 0.001
	if snap.Deterministe < 0.40-eps || snap.Deterministe > 0.40+eps {
		t.Errorf("after SetTuning: Deterministe = %f, want ~0.40", snap.Deterministe)
	}
}

func TestAtomicThresholds_ConcurrentReadsSafe(t *testing.T) {
	t.Parallel()
	at := calibration.NewAtomicThresholds(defaultThresholds())
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = at.Snapshot()
			_ = at.Deterministe()
			_ = at.AuthBlock()
		}()
	}
	wg.Wait()
}
```

- [ ] **Étape 3 — Vérifier**

```bash
go test -race -v ./internal/calibration/...
go vet ./...
golangci-lint run ./...
```

Attendu : tous les tests `TestAtomicThresholds_*` et `TestDefaultProfile_*` passent avec le race detector activé.

- [ ] **Étape 4 — Commit**

```bash
git add internal/calibration/thresholds_atomic.go internal/calibration/thresholds_atomic_test.go
git commit -m "feat(calibration): add AtomicThresholds (calque FibGo bigfft/fft.go)

Lock-free, atomic.Int64-encoded thresholds for Deterministe, Probabiliste,
AuthBlock, SensCoherence, InvCoherence, HysteresisGap. Milli-unit encoding
(int64 = v*1000) per FibGo pattern. SetTuning coordinates all six fields.
Ordering invariant (Deterministe<=Probabiliste) enforced via panic on both
construction and SetTuning.

Tests: roundtrip, ordering, panic-on-violation, SetTuning, concurrent reads
(race detector clean).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.9 — `docs/theory/04-dimensions.md`

**Files :**
- Create: `docs/theory/04-dimensions.md`

**Agent :** `ruflo-core:researcher`

- [ ] **Étape 1 — Rédiger `docs/theory/04-dimensions.md`**

```markdown
# 04 — Les trois dimensions — renvoi vers chap. III.8.4

*Document de renvoi croisé. Le verbatim canonique vit dans `InteroperabiliteAgentique/Monographie.md` v2.4.3, chap. III.8.4.*

*Statut global : Hypothèse — pondérations initiales, à corroborer sur traces AgentMeshKafka M4. Daté 2026-05-23.*

---

## Vue synoptique (III.8.4)

| Dimension | Pôle 0 *(avant)* | Pôle 1 *(pendant)* | Nature |
|---|---|---|---|
| **D-SENS** | Contrat figé, publié, opposable | Capacité découverte, interprétée à l'exécution | Fait protocolaire |
| **D-AUTORITÉ** | Chaîne courte, intra-domaine, humain ancré | Chaîne longue, inter-org, sans humain | Fait institutionnel (Searle 1995) |
| **D-INVARIANT** | Support énuméré à la conception | Support tracé / négocié / observé pendant | Fait protocolaire |

τ applicable : D-SENS et D-INVARIANT — oui, coûteux. D-AUTORITÉ — conditionné à institution externe (§4.4).

---

## D-SENS — lieu de fixation du sens (III.8.4.1)

**Question opérante** : *le pair décide-t-il d'invoquer à partir d'une interprétation produite à l'exécution, ou d'un câblage produit à la conception ?*

### Sondes et poids initiaux

| Sonde | Indicateur TauGo | Poids | Encodage Go |
|---|---|---|---|
| `S_contract` | `Target.ContractURI == ""` → 1.0 | 0.35 | `probeContract()` dans `dsens.go` |
| `S_runtime_resolve` | `IntentDescription != ""` → 1.0 | 0.30 | `probeRuntimeResolve()` |
| `S_capability_discovery` | `Target.DiscoveryMode != Static` → 1.0 | 0.20 | `probeCapabilityDiscovery()` |
| `S_reasoner_intent` | Score LLM via `Client.Interpret()` | 0.15 | `probeReasonerIntent()` |

`D_SENS(x) = 0.35·S_contract + 0.30·S_runtime_resolve + 0.20·S_capability_discovery + 0.15·S_reasoner_intent`

**Fichier** : `internal/tau/dimensions/dsens.go`

---

## D-AUTORITÉ — portée de la chaîne de délégation (III.8.4.2)

**Question opérante** : *la chaîne est-elle longue, dynamique, inter-organisationnelle, sans humain ancré ?*

**Asymétrie ontologique (III.8.4.2.bis)** : D-AUTORITÉ est un fait institutionnel (Searle 1995). Déplacer la fixation d'autorité vers l'exécution exige une institution émettrice externe. Sans `Attestation` opposable, le score D-AUTORITÉ ≥ θ_auth_block déclenche un `Refus` ontologique, non un `Probabiliste`.

### Sondes et poids initiaux

| Sonde | Indicateur TauGo | Poids | Encodage Go |
|---|---|---|---|
| `A_chain_depth` | Profondeur normalisée : `DelegationDepth / 4.0` | 0.25 | `probeChainDepth()` |
| `A_cross_org` | `Organization == ""` ou `DelegationDepth > 1` → 1.0 | 0.25 | `probeCrossOrg()` |
| `A_human_anchor` | Inversé : `HumanInLoop == false` → 1.0 | 0.25 | `probeHumanAnchor()` |
| `A_dynamic_resolution` | `Target.DiscoveryMode != Static` → 1.0 | 0.25 | `probeDynamicResolution()` |

`D_AUTORITÉ(x) = 0.25·A_chain_depth + 0.25·A_cross_org + 0.25·A_human_anchor + 0.25·A_dynamic_resolution`

**Garde** (étape 2 du dispatcher) : `D_AUTORITÉ(x) ≥ 0.85 ∧ x.Attestation == nil ⇒ Refus("I3 — verrou ontologique D-AUTORITÉ")`.

**Fichier** : `internal/tau/dimensions/dauthority.go`

---

## D-INVARIANT — support des invariants d'intégration (III.8.4.3)

**Question opérante** : *le support repose-t-il sur un artefact figé avant l'interaction, ou tracé / négocié / observé pendant ?*

**Contrainte de cohérence I4 (III.8.4.5)** : `i ≈ pendant ⟹ s ≈ pendant`. Direction dissymétrique — D-SENS contraint D-INVARIANT. Une combinaison D-INVARIANT élevé / D-SENS bas est ontologiquement incohérente et déclenche un refus (étape 5 du dispatcher).

### Sondes et poids initiaux

| Sonde | Indicateur TauGo | Poids | Encodage Go |
|---|---|---|---|
| `I_event_registry` | `Context["event_registry"] == true` → 1.0 | 0.30 | `probeEventRegistry()` |
| `I_idempotency_derived` | `Context["idempotency_key_mode"] == "derived"` → 1.0 | 0.25 | `probeIdempotencyDerived()` |
| `I_capability_mediation` | `Context["capability_mediation"] == true` ou `DiscoveryMode != Static` | 0.25 | `probeCapabilityMediation()` |
| `I_enumerated_plan` | Inversé : `ContractURI == ""` et pas de plan explicite → 1.0 | 0.20 | `probeEnumeratedPlan()` |

`D_INVARIANT(x) = 0.30·I_event_registry + 0.25·I_idempotency_derived + 0.25·I_capability_mediation + 0.20·I_enumerated_plan`

**Fichier** : `internal/tau/dimensions/dinvariant.go`

---

## Score composite τ

```
τ_score = 0.4 · D_SENS(x) + 0.3 · D_AUTORITÉ(x) + 0.3 · D_INVARIANT(x)
```

Poids initiaux PRD §11.1 : `(0.4, 0.3, 0.3)`. Statut : Hypothèse.

---

## Encodage Go des types d'entrée

Les scores dépendent des types `Principal` et `Capability` ajoutés à `Exchange` en M2.1 :

```go
type Principal struct {
    ID              string
    HumanInLoop     bool          // false => PairProbabiliste, A_human_anchor = 1
    Organization    string        // "" => A_cross_org = 1
    DelegationDepth int           // 0 = humain direct ; >=4 => A_chain_depth = 1
}

type Capability struct {
    ID            string
    DiscoveryMode DiscoveryMode   // Static | DynamicMCP | DynamicA2A | DynamicAGNTCY
    ContractURI   string          // "" => S_contract = 1 (pas de contrat)
}
```

---

## Questions ouvertes (Hypothèse, 2026-05-23)

1. Les pondérations initiales {0.35, 0.30, 0.20, 0.15} pour D-SENS sont-elles robustes sur des traces AgentMeshKafka réelles ? *Réponse attendue : M4.*
2. L'heuristique `DelegationDepth >= 4 => A_chain_depth = 1.0` est-elle bien calibrée ? *À réviser avec données empiriques M4.*
3. Les clés de contexte (`event_registry`, `idempotency_key_mode`, `capability_mediation`) sont-elles portées par les messages AgentMeshKafka réels ? *À vérifier lors de l'intégration M4.*
4. La pondération égale (0.25 × 4) pour D-AUTORITÉ reflète-t-elle l'égale importance des quatre facteurs ? *Hypothèse de symétrie à tester M4.*

---

*Renvoi PRD : §5 (dimensions), §4.4 (asymétrie ontologique), §6.1 (I4). Plan M2 : `docs/superpowers/plans/2026-05-23-M2-dimensions-gardes.md`.*
```

- [ ] **Étape 2 — Commit**

```bash
git add docs/theory/04-dimensions.md
git commit -m "docs(theory): add III.8.4 cross-reference for three dimensions

Tables for D-SENS, D-AUTORITÉ, D-INVARIANT with probe names, weights,
Go encoding, and Go type references. Marks all weight values as Hypothèse.
Notes the I4 coherence direction (D-SENS constrains D-INVARIANT, not reverse).
Four open questions flagged for M4 empirical validation.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.10 — `docs/empirical/M2-sample-decisions.md`

**Files :**
- Create: `docs/empirical/M2-sample-decisions.md`

**Agent :** `ruflo-core:researcher`

- [ ] **Étape 1 — Générer les 10 décisions avec `tau decide`**

Construire 10 fixtures JSON couvrant : Deterministe bas, Probabiliste haut, Refus frontière, Refus I3, Refus I4, et 5 variantes intermédiaires. Lancer chaque fixture via le CLI et capturer la sortie :

```bash
go build -o tau ./cmd/tau

# Fixture 1 — Déterministe bas (static, humain, pas de délégation)
echo '{"id":"f01","intent_description":"call internal API","initiator":{"id":"user-1","human_in_loop":true,"organization":"org-a","delegation_depth":0},"target":{"id":"svc-a","discovery_mode":0,"contract_uri":"https://api.internal/v1"}}' | ./tau decide

# ... (idem pour f02 à f10)
rm -f tau tau.exe
```

- [ ] **Étape 2 — Rédiger `docs/empirical/M2-sample-decisions.md`**

Le rapport doit contenir pour chaque décision : fixture JSON (abrégée), sortie JSON Decision, tableau scores (D-SENS, D-AUTORITÉ, D-INVARIANT, τ_score), régime, diagnostic si Refus. Format :

```markdown
# M2 — Décisions de référence (10 échantillons)

> Généré le 2026-05-23 avec `tau decide` + dispatcher M2 (`v0.0.3-alpha`).
> Profil : M2-default (thresholds PRD §11.1 initiaux, poids PRD §5.1-5.3).
> Statut : Hypothèse — scores non calibrés (pré-M4).

## Tableau synthèse

| # | ID | D-SENS | D-AUTH | D-INV | τ_score | Régime | Diagnostic |
|---|---|---|---|---|---|---|---|
| 1 | f01 | 0.00 | 0.00 | 0.00 | 0.00 | Deterministe | — |
| ... |

## Décision f01 — Déterministe bas

**Fixture** : `{"id":"f01",...}` (static, humain ancré, contrat présent)

**Sondes** :
- S_contract = 0.00 (ContractURI présent)
- S_runtime_resolve = 1.00 (IntentDescription non vide)
- S_capability_discovery = 0.00 (Static)
- S_reasoner_intent = 0.XXX (stub FNV-1a)
- D-SENS = ...

**Décision JSON** :
\`\`\`json
{ ... sortie tau decide ... }
\`\`\`

...
```

**Note** : si le CLI produit des scores différents de ce qui est prévu manuellement, les valeurs réelles ont la priorité. Documenter toute surprise comme question ouverte.

- [ ] **Étape 3 — Commit**

```bash
git add docs/empirical/M2-sample-decisions.md
git commit -m "docs(empirical): M2 sample decisions report (10 traces, scores ventilated)

10 fixtures covering Deterministe low, Probabiliste high, Refus-frontier,
Refus-I3, Refus-I4, and 5 intermediate variants. All scores and probes
ventilated per decision. Marked as Hypothesis — pre-M4, not yet calibrated.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M2.11 — Revue intégrée + tag `v0.0.3-alpha`

**Agent :** thread principal (intégration) + `ruflo-core:reviewer` (revue)

- [ ] **Étape 1 — Vérifier la suite locale complète**

```bash
go build ./...
go test -race -v ./...
go vet ./...
golangci-lint run ./...
go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/... ./...
go tool cover -func=coverage.out | tail -1
```

Couverture cible : ≥ 80 % global sur `internal/*`. `internal/tau` ≥ 90 %. `internal/tau/dimensions` ≥ 80 %. `internal/orchestration` ≥ 80 %. `internal/calibration` ≥ 80 %.

Vérifier les tests nominaux M2 :
```bash
go test -run TestRefusOntologiqueDAUTORITE ./internal/orchestration/
go test -run TestI4_IncoherenceDetectee ./internal/orchestration/
go test -run TestDSens ./internal/tau/dimensions/
go test -run TestDAuthority ./internal/tau/dimensions/
go test -run TestDInvariant ./internal/tau/dimensions/
go test -run TestDefaultProfile ./internal/calibration/
go test -run TestAtomicThresholds ./internal/calibration/
```

- [ ] **Étape 2 — Vérifier les règles architecturales**

```bash
go test -v -run TestArchitectureLayering ./internal/
```

Vérifier que `internal/tau/dimensions` n'importe pas `internal/orchestration` ni d'autres packages de dimension entre eux.

- [ ] **Étape 3 — Briefing reviewer**

Dispatch un `ruflo-core:reviewer` avec le briefing suivant :

> Revue intégrée de M2 (commit range `v0.0.2-alpha..HEAD`). Vérifier :
> 1. Les trois scorers (`ScoreDSens`, `ScoreDAuthority`, `ScoreDInvariant`) retournent bien des valeurs dans `[0, 1]` pour tout `Exchange` valide. Les test d'ordre (static < dynamic) sont vrais.
> 2. Le dispatcher M2 implémente strictement les étapes 1, 2, 4, 5, 6, 7 du pseudo-algo PRD §10. Étapes 3 et 8 absentes (déférées M3/M5).
> 3. `frontierFromExchange` est documentée comme heuristique M2 placeholder, avec commentaire explicite `// placeholder until M5`.
> 4. La garde ontologique (étape 2) se déclenche AVANT le calcul de D-SENS et D-INVARIANT (ordre étapes respecté).
> 5. La garde I4 (étape 5) ne se déclenche PAS si l'attestation est absente mais D-AUTORITÉ < AuthBlock (les gardes sont indépendantes et ordonnées).
> 6. `internal/tau/dimensions/*` n'importe pas `internal/orchestration` ni d'autres packages de dimension (orthogonalité vérifiée par arch_test.go et manuellement).
> 7. `AtomicThresholds` est race-detector clean (`TestAtomicThresholds_ConcurrentReadsSafe`).
> 8. Tous les nouveaux types publics ont des tags JSON snake_case (`Principal`, `Capability`, `DiscoveryMode` en tant que champ `json:"discovery_mode"`).
> 9. `DefaultProfile().DateRevision` est ≥ 6 mois dans le futur depuis 2026-05-23.
> 10. Pas d'emoji, godoc en anglais, `t.Parallel()` sur 100 % des nouveaux tests.
> 11. Aucun anti-patron introduit : pas de méthode `Predict*`, pas d'import LLM concret dans `tau/*` ou `orchestration/*`.

- [ ] **Étape 4 — Tag `v0.0.3-alpha`**

```bash
git tag -a v0.0.3-alpha -m "M2: three dimensions + ontological guards D-AUTORITÉ and I4

M2.1 - Principal, Capability, DiscoveryMode types; Exchange extended; TraceThresholds extended
M2.2 - D-SENS scorer: 4 probes (S_contract, S_runtime_resolve, S_capability_discovery, S_reasoner_intent)
M2.3 - D-AUTORITÉ scorer: 4 probes (A_chain_depth, A_cross_org, A_human_anchor, A_dynamic_resolution)
M2.4 - D-INVARIANT scorer: 4 probes (I_event_registry, I_idempotency_derived, I_capability_mediation, I_enumerated_plan)
M2.5 - Dispatcher step 2: ontological guard D-AUTORITÉ (AuthBlock=0.85); frontierFromExchange heuristic
M2.6 - Dispatcher step 5: I4 coherence guard (InvCoherence=0.50, SensCoherence=0.50)
M2.7 - calibration.Profile with DefaultProfile() (PRD §11.3 initial values)
M2.8 - calibration.AtomicThresholds (calque FibGo bigfft/fft.go); lock-free milli-unit encoding
M2.9 - docs/theory/04-dimensions.md: III.8.4 cross-reference with probe tables
M2.10 - docs/empirical/M2-sample-decisions.md: 10 traced decisions, scores ventilated
M2.11 - integrated review + tag

Spec: PRD.md §4.4, §5, §6.1, §10, §11. Plan: docs/superpowers/plans/2026-05-23-M2-dimensions-gardes.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin main
git push origin v0.0.3-alpha
```

- [ ] **Étape 5 — Mettre à jour `CHANGELOG.md`**

Ajouter :

```markdown
## [0.0.3-alpha] — 2026-05-23

### Ajouté

- `internal/tau/operator.go` : types `Principal`, `Capability`, `DiscoveryMode` ; `Exchange` étendu avec `Initiator` et `Target` ; `TraceThresholds` étendu avec `AuthBlock`, `SensCoherence`, `InvCoherence`.
- `internal/tau/dimensions/dsens.go` : scoreur D-SENS, 4 sondes (`S_contract`, `S_runtime_resolve`, `S_capability_discovery`, `S_reasoner_intent`), poids initiaux PRD §5.1.
- `internal/tau/dimensions/dauthority.go` : scoreur D-AUTORITÉ, 4 sondes dont `A_human_anchor` inversé, poids égaux PRD §5.2.
- `internal/tau/dimensions/dinvariant.go` : scoreur D-INVARIANT, 4 sondes dont `I_enumerated_plan` inversé, poids initiaux PRD §5.3.
- `internal/orchestration/dispatcher.go` : étapes 1 (heuristique frontier), 2 (garde ontologique I3), 4 (scores), 5 (garde I4), 6 (composite pondéré), 7 (hystérèse). `Thresholds` étendu avec `AuthBlock`, `SensCoherence`, `InvCoherence` et `DefaultThresholds()`.
- `internal/calibration/profile.go` : types `Profile`, `Weights`, `Thresholds` ; `DefaultProfile()` avec valeurs initiales PRD §11.1/§11.3.
- `internal/calibration/thresholds_atomic.go` : `AtomicThresholds` — pattern `atomic.Int64` calque FibGo, 6 champs, milli-unit encoding, `SetTuning` coordonné.
- Tests : `TestRefusOntologiqueDAUTORITE`, `TestOntologicalGuardPassesWithAttestation`, `TestI4_IncoherenceDetectee`, `TestI4_CoherentCombinationAccepted`, suites `TestDSens_*`, `TestDAuthority_*`, `TestDInvariant_*`, `TestDefaultProfile_*`, `TestAtomicThresholds_*`.
- `docs/theory/04-dimensions.md` : renvoi III.8.4 avec tables sondes/poids/questions ouvertes.
- `docs/empirical/M2-sample-decisions.md` : 10 décisions tracées avec scores ventilés.
```

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): M2 release notes (v0.0.3-alpha)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin main
```

---

## Annexe — Self-review (à exécuter avant commit du plan)

- [x] **Couverture M2 high-level** :
  - M2.1 `Principal`, `Capability`, extension `Exchange` → Tâche M2.1 ✓
  - M2.2 `dsens.go` + 4 sondes + tests → Tâche M2.2 ✓
  - M2.3 `dauthority.go` + 4 sondes + tests → Tâche M2.3 ✓
  - M2.4 `dinvariant.go` + 4 sondes + tests → Tâche M2.4 ✓
  - M2.5 Garde ontologique D-AUTORITÉ (étape 2) + `TestRefusOntologiqueDAUTORITE` → Tâche M2.5 ✓
  - M2.6 Garde I4 (étape 5) + `TestI4_IncoherenceDetectee` → Tâche M2.6 ✓
  - M2.7 `calibration/profile.go` + `DefaultProfile()` → Tâche M2.7 ✓
  - M2.8 `calibration/thresholds_atomic.go` + pattern `atomic.Int64` → Tâche M2.8 ✓
  - M2.9 `docs/theory/04-dimensions.md` → Tâche M2.9 ✓
  - M2.10 `docs/empirical/M2-sample-decisions.md` → Tâche M2.10 ✓
  - M2.11 Revue + tag `v0.0.3-alpha` → Tâche M2.11 ✓

- [x] **Note de conception — refactoring dispatcher** :
  - `frontierFromExchange` heuristique documentée → M2.5 étape 3 ✓
  - Composite pondéré remplace score LLM direct → M2.5 étape 3 ✓
  - Client LLM utilisé uniquement dans `S_reasoner_intent` → M2.2 étape 3 ✓

- [x] **Placeholder scan** : aucun TBD/TODO résiduel dans le code fourni. Le seul TODO documenté est la mise à jour de `incoherentExchange()` si les scores ne correspondent pas (étape M2.6 étape 1) — c'est une instruction d'agent, pas un placeholder de code.

- [x] **Cohérence des types** :
  - `Principal`, `Capability`, `DiscoveryMode` déclarés dans `operator.go` (M2.1) et utilisés dans `dsens.go`, `dauthority.go`, `dinvariant.go` (M2.2-M2.4) ✓
  - `TraceThresholds` étendu en M2.1, utilisé dans `dispatcher.go` M2.5 ✓
  - `calibration.Thresholds` distinct de `orchestration.Thresholds` et `tau.TraceThresholds` — trois copies intentionnelles pour respecter l'étanchéité des couches ✓
  - `Score` type partagé dans `score.go` entre les trois scoreurs ✓

- [x] **Contraintes architecturales** :
  - `tau/dimensions → tau` : autorisé ✓
  - `tau/dimensions → bridge/llm` : autorisé (S_reasoner_intent) ✓
  - `tau/dimensions → orchestration` : interdit — non présent dans le code ✓
  - Dimensions ne s'importent pas entre elles : `dsens.go`, `dauthority.go`, `dinvariant.go` sont indépendants ✓
  - `orchestration → tau/dimensions` : autorisé (couche orchestration importe domaine) ✓

- [x] **Anti-patrons gardés (cumulés M0 + M1 + M2)** :
  - #1 prédictif → `TestNoPredictiveAPI` arrive en M3.9 ; aucun `Predict*` introduit ✓
  - #2 hors frontière → `frontierFromExchange` dérive correctement; `TestRefusHorsFrontiere` (M0) toujours actif ✓
  - #3 atemporel → `Profile.DateRevision` + test `TestDefaultProfile_DateRevisionAtLeast6MonthsAhead` ✓
  - #4 clos → `Trace.UnmodeledObservations` maintenu ; rapport empirique M2-sample-decisions daté ✓

- [x] **Tags JSON snake_case** : `Principal`, `Capability` tous avec tags `json:"..."` snake_case ✓

- [x] **Tests d'ordre** : propriété `static < dynamic` vérifiée pour D-SENS et D-AUTORITÉ ; `frozen < traced` pour D-INVARIANT ✓

---

*Sous-plan V1 — 2026-05-23. Référence : `PRDPlanning.md` §M2 + `PRD.md` §4.4, §5, §6.1, §10, §11. Coordinateur : Claude Code thread principal. Exécutants : agent teams (`ruflo-core:coder`, `ruflo-core:researcher`, `ruflo-core:reviewer`).*
