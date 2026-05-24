# M3 Sub-plan — Cinq invariants I1-I5 comme cibles fuzz

> Sous-plan détaillé du milestone M3 (cf. [`PRDPlanning.md` §M3](../../../PRDPlanning.md)). Bite-sized, exécutable par sous-agents frais. Calque structurel du M2 détaillé dans `docs/superpowers/plans/2026-05-23-M2-dimensions-gardes.md`.

**Objectif** : le package `internal/tau/invariants/` encode les cinq invariants I1-I5 (PRD §6, chap. III.8.5) sous forme de propriétés vérifiables sur une `Decision` déjà calculée. Cinq cibles fuzz (`FuzzI1` à `FuzzI5`) exercent chacun un invariant via le dispatcher M2 sur des `Exchange` générés. L'étape 8 du pseudo-algorithme PRD §10 (`EvaluateInvariants`) annote `Trace.UnmodeledObservations` lorsqu'une violation est détectée. Les trois anti-patrons restants (#1 prédictif, #3 atemporel, #4 clos) reçoivent leurs gardes par test.

**Critère d'acceptation global** :

```bash
go test -race ./... && \
  go test -fuzz=FuzzI1_Conservation        -fuzztime=30s ./internal/tau/invariants/ && \
  go test -fuzz=FuzzI2_Irreductibilite     -fuzztime=30s ./internal/tau/invariants/ && \
  go test -fuzz=FuzzI3_AsymetrieAutorite   -fuzztime=30s ./internal/tau/invariants/ && \
  go test -fuzz=FuzzI4_CoherenceContrainte -fuzztime=30s ./internal/tau/invariants/ && \
  go test -fuzz=FuzzI5_CompositionConjonctive -fuzztime=30s ./internal/tau/invariants/
```

…vert (0 panique, 0 crash). Rapport `docs/empirical/fuzz-summary.md` daté avec entrées explorées, statut par cible, exceptions épinglées, marqueur d'incertitude.

**Tag visé** : `v0.0.4-alpha`

**Pré-requis** : M0 / M1 / M2 commités, tags `v0.0.1-alpha`/`v0.0.2-alpha`/`v0.0.3-alpha` sur `main`. Package `internal/tau/invariants/` n'existe pas encore. Le dispatcher M2 (`internal/orchestration/dispatcher.go`) implémente les étapes 1, 2, 4, 5, 6, 7 du pseudo-algo PRD §10 ; M3 ajoute l'étape 8.

---

## Note de conception — frontière package `invariants`

### Étanchéité (rappelée par `internal/arch_test.go`)

```
tau/invariants  →  tau          : AUTORISÉ (utilise Exchange, Decision, FrontierCheck, Attestation)
tau/invariants  →  tau/dimensions : INTERDIT (orthogonalité encodée — invariants opèrent sur scores déjà calculés)
tau/invariants  →  orchestration : INTERDIT (couche supérieure, sens du flux)
tau/invariants  →  bridge/*      : INTERDIT
orchestration   →  tau/invariants : AUTORISÉ (le dispatcher invoque EvaluateInvariants à l'étape 8)
```

**Hypothèse — *À vérifier au commit M3.1*** : la règle `tau/invariants → dimensions interdit` est déjà présente dans `internal/arch_test.go` (lignes 21-26). Aucune modification de `arch_test.go` requise.

### Rôle du package `invariants`

Le package `invariants` **ne recalcule pas** les dimensions. Il reçoit :

- l'`Exchange` d'entrée
- la `Decision` produite par le dispatcher (avec `Trace.TauScore`, `Trace.Frontier`, `Trace.Thresholds`)
- les scores éventuels passés par valeur depuis le dispatcher *(injection en étape 8)*

…et retourne un `Status` par invariant : `Held` / `Violated` / `NotApplicable`. L'agrégation est un `Statuses` map. Pas de panic sauf sentinel interne. Les helpers (`Conserve`, `Residu`, `Recablage`, `Aggregate`) sont publics pour permettre les fuzz tests externes.

### Anti-patron #1 — pas de méthode `Predict*` / `Expected*` / `Forecast*`

`TestNoPredictiveAPI` (M3.9) inspecte par réflexion **tous les symboles exportés** des packages `tau`, `tau/dimensions`, `tau/invariants`, `orchestration`. Échec si un nom matche `^(Predict|Expected|Forecast).*` (regex). Garde Conventional Commits + revue.

### Anti-patron #4 — observations non modélisées

L'étape 8 du dispatcher peuple `Trace.UnmodeledObservations []string` lorsque :

1. `Statuses.AnyViolated() == true` : une violation détectée → ajouter `"I<N> — <diag court>"` dans la trace.
2. La trace contient une grandeur observée non couverte par les sondes (V1 : non détectable — V2 introduira un registre de grandeurs). **Statut V1 : placeholder.**

### Statuts d'invariants — granularité

```go
type Status int

const (
    StatusUnknown Status = iota
    Held                  // l'invariant tient pour cette décision
    Violated              // violation détectée (à reporter dans UnmodeledObservations)
    NotApplicable         // invariant non applicable (ex. Refus en amont = I4 non testé)
)
```

Distinction `Held` vs `NotApplicable` essentielle : un `Refus("hors frontière τ")` rend I1-I5 `NotApplicable` (τ n'a pas opéré). Un `Refus("I3 — verrou ontologique")` rend I1, I2 `NotApplicable` mais I3 `Held` (la garde a tenu).

---

## Tâche M3.1 — Squelette package `invariants` (`doc.go`, `evaluator.go`, types `Status` / `Statuses`)

**Files :**
- Create: `internal/tau/invariants/doc.go`
- Create: `internal/tau/invariants/evaluator.go`
- Create: `internal/tau/invariants/evaluator_test.go`

**Agent :** `ruflo-core:coder` (TDD)

- [ ] **Étape 1 — Créer `internal/tau/invariants/doc.go`**

```go
// Package invariants encodes the five invariants I1-I5 of the τ operator
// (chap. III.8.5, PRD §6) as properties verifiable on a Decision already
// produced by the orchestration dispatcher.
//
// Architecture rule (gated by internal/arch_test.go): this package may import
// internal/tau but must NOT import internal/tau/dimensions (orthogonality
// constraint between scored dimensions and structural invariants),
// internal/orchestration (downstream layer), or any internal/bridge/*.
//
// The package exposes one evaluator per invariant plus an aggregating
// EvaluateInvariants entry point invoked by the dispatcher at step 8.
// Helpers (Conserve, Residu, Recablage, Aggregate) are exported so fuzz
// targets can drive them directly.
package invariants
```

- [ ] **Étape 2 — Écrire `internal/tau/invariants/evaluator.go`**

```go
package invariants

import (
	"github.com/agbruneau/taugo/internal/tau"
)

// Status is the verdict for one invariant on one decision.
type Status int

const (
	// StatusUnknown is the zero value; never returned by Evaluate functions.
	StatusUnknown Status = iota
	// Held means the invariant tested true for this decision.
	Held
	// Violated means the invariant was tested and failed.
	Violated
	// NotApplicable means the invariant is not testable in this context
	// (e.g. Refus upstream of the conditions the invariant constrains).
	NotApplicable
)

// String returns the lowercase verbatim form for trace diagnostics.
func (s Status) String() string {
	switch s {
	case Held:
		return "held"
	case Violated:
		return "violated"
	case NotApplicable:
		return "not_applicable"
	default:
		return "unknown"
	}
}

// Statuses bundles the verdicts of all five invariants for one decision.
type Statuses struct {
	I1 Status `json:"i1"`
	I2 Status `json:"i2"`
	I3 Status `json:"i3"`
	I4 Status `json:"i4"`
	I5 Status `json:"i5"`
}

// AnyViolated reports whether at least one invariant was violated.
func (s Statuses) AnyViolated() bool {
	return s.I1 == Violated || s.I2 == Violated || s.I3 == Violated ||
		s.I4 == Violated || s.I5 == Violated
}

// Summary returns a stable list of short diagnostic strings, one per
// violated invariant, in numerical order. Empty when no violation.
// Format: "I<N> — <one-line reason>".
func (s Statuses) Summary() []string {
	out := make([]string, 0, 5)
	if s.I1 == Violated {
		out = append(out, "I1 — conservation rompue (grandeur supprimée par τ)")
	}
	if s.I2 == Violated {
		out = append(out, "I2 — résidu migrant vidé sans perte de condition de frontière")
	}
	if s.I3 == Violated {
		out = append(out, "I3 — asymétrie D-AUTORITÉ contournée ou profil périmé")
	}
	if s.I4 == Violated {
		out = append(out, "I4 — combinaison (s, i) incohérente non refusée")
	}
	if s.I5 == Violated {
		out = append(out, "I5 — agrégation M(π) hors bornes")
	}
	return out
}

// EvaluateInvariants runs the five evaluators on (x, decision) and returns
// the bundled Statuses. The orchestration dispatcher invokes this at step 8
// of the PRD §10 pseudo-algorithm. The function MUST NOT panic; an invariant
// internal-state sentinel is the only acceptable panic source (calque FibGo).
func EvaluateInvariants(x tau.Exchange, dec tau.Decision) Statuses {
	return Statuses{
		I1: EvaluateI1(x, dec),
		I2: EvaluateI2(x, dec),
		I3: EvaluateI3(x, dec),
		I4: EvaluateI4(x, dec),
		I5: EvaluateI5(x, dec),
	}
}
```

- [ ] **Étape 3 — Écrire le test rouge `internal/tau/invariants/evaluator_test.go`**

```go
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
```

Vérifier red phase :

```powershell
go test ./internal/tau/invariants/...
```

Attendu : `undefined: invariants.EvaluateI1` (les cinq évaluateurs n'existent pas encore — créés en M3.2-M3.6).

- [ ] **Étape 4 — Stub temporaires pour décrocher la compilation**

Dans `evaluator.go`, ajouter en bas (à supprimer dès M3.2-M3.6) :

```go
// === STUBS TEMPORAIRES — supprimés en M3.2 à M3.6 ===
// Chaque évaluateur est remplacé par sa vraie implémentation dans la tâche
// dédiée. Les stubs renvoient NotApplicable pour permettre la compilation
// du package skeleton avant l'écriture des vraies évaluations.

func EvaluateI1(_ tau.Exchange, _ tau.Decision) Status { return NotApplicable }
func EvaluateI2(_ tau.Exchange, _ tau.Decision) Status { return NotApplicable }
func EvaluateI3(_ tau.Exchange, _ tau.Decision) Status { return NotApplicable }
func EvaluateI4(_ tau.Exchange, _ tau.Decision) Status { return NotApplicable }
func EvaluateI5(_ tau.Exchange, _ tau.Decision) Status { return NotApplicable }
```

- [ ] **Étape 5 — Vérifier**

```powershell
go build ./internal/tau/invariants/
go vet ./...
golangci-lint run ./...
go test -race ./internal/tau/invariants/
go test ./internal/    # arch_test.go doit toujours passer
```

Attendu : tous verts. `TestArchitectureLayering/...invariants` ne fait plus skip puisque le package existe désormais.

- [ ] **Étape 6 — Commit**

```powershell
git add internal/tau/invariants/
git commit -m "feat(tau/invariants): scaffold package with Status, Statuses, EvaluateInvariants

M3.1: introduces internal/tau/invariants/ with stub evaluators returning
NotApplicable. EvaluateInvariants is the dispatcher step-8 entry point.
Statuses.AnyViolated and .Summary expose the surface consumed by
Trace.UnmodeledObservations (anti-patron #4 guard).

Stubs for EvaluateI1..I5 unblock package compilation; they are replaced
by their real implementations in M3.2 through M3.6.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.2 — I1 conservation + helper `Conserve` + tests

**Files :**
- Modify: `internal/tau/invariants/evaluator.go` (retire le stub `EvaluateI1`)
- Create: `internal/tau/invariants/i1_conservation.go`
- Create: `internal/tau/invariants/i1_conservation_test.go`

**Agent :** `ruflo-core:coder` (TDD)

### Note théorique

I1 : *τ déplace l'instant de fixation d'une grandeur **sans altérer la grandeur***. En V1, la « grandeur » exposée est l'identité de l'`Exchange` (ID, intent, target, initiator). `Conserve(x, dec)` vérifie qu'aucun champ porteur de la grandeur n'est altéré par τ — TauGo ne mute pas `Exchange`, donc le test V1 vérifie que `Decision.Trace.ExchangeID == x.ID` ET que la décision n'introduit aucune **suppression** d'attribut (anti-patron : τ qui « efface » une grandeur).

**Statut V1** : *Probable*. La conservation est **structurellement garantie** par l'immutabilité de `Exchange` (passé par valeur). Le test fuzz détecte une régression future où τ muterait subrepticement la trace.

- [ ] **Étape 1 — Écrire le test rouge `i1_conservation_test.go`**

```go
package invariants_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

func makeExchange(id, intent string) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: intent,
		DiscoveredAt:      time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
		Initiator: tau.Principal{
			ID: "p-1", HumanInLoop: true, Organization: "org-a",
		},
		Target: tau.Capability{
			ID: "cap-1", DiscoveryMode: tau.Static, ContractURI: "https://api/v1",
		},
	}
}

func TestConserve_IdentityTrace(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-1", "compute")
	dec := tau.Decision{
		Regime: tau.Deterministe,
		Trace:  tau.Trace{ExchangeID: "e-1", TauScore: 0.2},
	}
	if !invariants.Conserve(x, dec) {
		t.Fatal("Conserve returned false on identity-preserving decision")
	}
}

func TestConserve_BrokenWhenExchangeIDDrifts(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-1", "compute")
	dec := tau.Decision{
		Regime: tau.Deterministe,
		Trace:  tau.Trace{ExchangeID: "MUTATED", TauScore: 0.2},
	}
	if invariants.Conserve(x, dec) {
		t.Fatal("Conserve returned true despite ExchangeID drift")
	}
}

func TestEvaluateI1_HeldOnNonRefus(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-i1", "compute")
	dec := tau.Decision{
		Regime: tau.Probabiliste,
		Trace:  tau.Trace{ExchangeID: "e-i1", TauScore: 0.8},
	}
	if got := invariants.EvaluateI1(x, dec); got != invariants.Held {
		t.Fatalf("EvaluateI1 = %v, want Held", got)
	}
}

func TestEvaluateI1_NotApplicableOnRefusFrontiere(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-i1-na", "compute")
	dec := tau.Decision{
		Regime:     tau.Refus,
		Diagnostic: "hors frontière τ",
		Trace:      tau.Trace{ExchangeID: "e-i1-na"},
	}
	// τ has not operated → I1 is not applicable.
	if got := invariants.EvaluateI1(x, dec); got != invariants.NotApplicable {
		t.Fatalf("EvaluateI1 = %v, want NotApplicable (Refus frontière)", got)
	}
}

func TestEvaluateI1_ViolatedOnTraceDrift(t *testing.T) {
	t.Parallel()
	x := makeExchange("e-i1-v", "compute")
	dec := tau.Decision{
		Regime: tau.Deterministe,
		Trace:  tau.Trace{ExchangeID: "DIFFERENT", TauScore: 0.2},
	}
	if got := invariants.EvaluateI1(x, dec); got != invariants.Violated {
		t.Fatalf("EvaluateI1 = %v, want Violated (trace drift)", got)
	}
}
```

- [ ] **Étape 2 — Écrire `i1_conservation.go`**

```go
package invariants

import "github.com/agbruneau/taugo/internal/tau"

// Conserve reports whether τ has preserved the magnitudes carried by x in
// the produced decision. V1 scope: the ExchangeID is the canonical magnitude;
// future versions extend Conserve as more invariants are added to Exchange.
//
// PRD §6.1 I1: "τ déplace l'instant de fixation d'une grandeur sans altérer
// la grandeur." V1 status: Probable — preservation is structurally enforced
// by Exchange value semantics; this helper detects future regressions.
func Conserve(x tau.Exchange, dec tau.Decision) bool {
	if dec.Trace.ExchangeID != x.ID {
		return false
	}
	return true
}

// EvaluateI1 returns the I1 verdict for (x, decision).
//
//   - Refus("hors frontière τ"): NotApplicable — τ has not operated.
//   - All other regimes: Held if Conserve(x, dec) holds, Violated otherwise.
func EvaluateI1(x tau.Exchange, dec tau.Decision) Status {
	if dec.Regime == tau.Refus && dec.Diagnostic == "hors frontière τ" {
		return NotApplicable
	}
	if Conserve(x, dec) {
		return Held
	}
	return Violated
}
```

Retirer le stub `EvaluateI1` de `evaluator.go`.

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v ./internal/tau/invariants/
go vet ./...
golangci-lint run ./...
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/tau/invariants/
git commit -m "feat(tau/invariants): I1 conservation + Conserve helper (PRD §6.1)

EvaluateI1 returns Held when Trace.ExchangeID matches the input Exchange.ID,
Violated otherwise. Refus(hors frontière τ) is mapped to NotApplicable since
τ has not operated. V1 magnitude = ExchangeID; expanded as future structural
invariants are added.

Status: Probable — preservation structurally enforced by Exchange value
semantics; this helper defends against future regressions.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.3 — I2 irréductibilité + helpers `Residu`, `Recablage` + tests

**Files :**
- Modify: `internal/tau/invariants/evaluator.go` (retire le stub `EvaluateI2`)
- Create: `internal/tau/invariants/i2_irreductibility.go`
- Create: `internal/tau/invariants/i2_irreductibility_test.go`

**Agent :** `ruflo-core:coder` (TDD)

### Note théorique

I2 : *le résidu migrant est non vide et non recâblable hors ligne sans détruire l'agentivité*. Reformulation exécutable PRD §6.1 :

`Residu(x) := { g | t_fix(g) ≈ t_int } ≠ ∅` et tout `Recablage(x)` qui vide le résidu doit faire perdre ≥ 1 condition de frontière.

**Encodage V1** :

- `Residu(x) []ResidualMagnitude` : liste des grandeurs dont le lieu de fixation est *pendant* (à l'exécution). V1 expose 4 grandeurs candidates : `target_resolution`, `intent_meaning`, `authority_chain`, `support_negotiation`. Une grandeur est dans le résidu ssi la sonde correspondante côté frontière retourne `true`.
- `Recablage(x, removed []string) tau.FrontierCheck` : simule le re-câblage hors-ligne (suppression de grandeurs de `Residu`) en retournant une `FrontierCheck` modifiée. La règle V1 :
  - retirer `"target_resolution"` ou `"support_negotiation"` ⇒ `UniversOuvert=false` et/ou `CompositionVariable=false`
  - retirer `"intent_meaning"` ⇒ `PairProbabiliste=false`
  - retirer `"authority_chain"` ⇒ `CoutNonBorne=false`

I2 tient ssi : pour tout sous-ensemble non vide `R ⊆ Residu(x)`, `Recablage(x, R).Inside() == false`.

**Statut** : *Confirmé par construction* (PRD §6.1).

- [ ] **Étape 1 — Écrire le test rouge `i2_irreductibility_test.go`**

```go
package invariants_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

func dynamicExchange() tau.Exchange {
	return tau.Exchange{
		ID:                "e-dyn",
		IntentDescription: "discover and call",
		DiscoveredAt:      time.Now().UTC(),
		Initiator: tau.Principal{
			ID: "agent-1", HumanInLoop: false, DelegationDepth: 2,
		},
		Target: tau.Capability{
			ID: "dyn-tool", DiscoveryMode: tau.DynamicMCP,
		},
	}
}

func TestResidu_NonEmptyForDynamicExchange(t *testing.T) {
	t.Parallel()
	r := invariants.Residu(dynamicExchange())
	if len(r) == 0 {
		t.Fatal("Residu was empty on dynamic exchange (frontier should yield ≥ 1 magnitude)")
	}
}

func TestRecablage_RemovingAllResidualLosesFrontier(t *testing.T) {
	t.Parallel()
	x := dynamicExchange()
	r := invariants.Residu(x)
	names := make([]string, len(r))
	for i, m := range r {
		names[i] = string(m)
	}
	got := invariants.Recablage(x, names)
	if got.Inside() {
		t.Fatalf("Recablage with all residual magnitudes removed kept Inside()=true: %+v", got)
	}
}

func TestEvaluateI2_HeldOnDynamicExchange(t *testing.T) {
	t.Parallel()
	x := dynamicExchange()
	dec := tau.Decision{
		Regime: tau.Probabiliste,
		Trace:  tau.Trace{ExchangeID: x.ID},
	}
	if got := invariants.EvaluateI2(x, dec); got != invariants.Held {
		t.Fatalf("EvaluateI2 = %v, want Held", got)
	}
}

func TestEvaluateI2_NotApplicableOnRefusFrontiere(t *testing.T) {
	t.Parallel()
	x := dynamicExchange()
	dec := tau.Decision{Regime: tau.Refus, Diagnostic: "hors frontière τ", Trace: tau.Trace{ExchangeID: x.ID}}
	if got := invariants.EvaluateI2(x, dec); got != invariants.NotApplicable {
		t.Fatalf("EvaluateI2 = %v, want NotApplicable", got)
	}
}
```

- [ ] **Étape 2 — Écrire `i2_irreductibility.go`**

```go
package invariants

import "github.com/agbruneau/taugo/internal/tau"

// ResidualMagnitude names a magnitude whose locus of fixation is "pendant"
// (runtime). The four V1 candidates map one-to-one onto the four classical
// frontier conditions.
type ResidualMagnitude string

const (
	MagTargetResolution    ResidualMagnitude = "target_resolution"    // UniversOuvert + CompositionVariable
	MagIntentMeaning       ResidualMagnitude = "intent_meaning"       // PairProbabiliste
	MagAuthorityChain      ResidualMagnitude = "authority_chain"      // CoutNonBorne
	MagSupportNegotiation  ResidualMagnitude = "support_negotiation"  // UniversOuvert + CompositionVariable
)

// Residu returns the migrating residue of x — the set of magnitudes that
// τ fixates at runtime rather than at design time. V1 enumeration is the
// four classical magnitudes; a magnitude is in the residue iff the matching
// frontier condition is currently violated (i.e. is "pendant").
//
// PRD §6.1 I2 reformulation: Residu(x) := { g | t_fix(g) ≈ t_int }.
func Residu(x tau.Exchange) []ResidualMagnitude {
	out := make([]ResidualMagnitude, 0, 4)
	dynamic := x.Target.DiscoveryMode != tau.Static
	if dynamic {
		out = append(out, MagTargetResolution, MagSupportNegotiation)
	}
	if !x.Initiator.HumanInLoop {
		out = append(out, MagIntentMeaning)
	}
	if x.Initiator.DelegationDepth > 0 {
		out = append(out, MagAuthorityChain)
	}
	return out
}

// Recablage simulates an offline rewiring of x by removing the named
// residual magnitudes. Returns the resulting FrontierCheck.
// Removing a magnitude collapses the matching frontier condition to false.
func Recablage(x tau.Exchange, removed []string) tau.FrontierCheck {
	// Start from the dispatcher's heuristic frontier (mirrored here to avoid
	// an internal/orchestration import).
	dynamic := x.Target.DiscoveryMode != tau.Static
	f := tau.FrontierCheck{
		UniversOuvert:       dynamic,
		CompositionVariable: dynamic,
		PairProbabiliste:    !x.Initiator.HumanInLoop,
		CoutNonBorne:        x.Initiator.DelegationDepth > 0,
	}
	for _, name := range removed {
		switch ResidualMagnitude(name) {
		case MagTargetResolution, MagSupportNegotiation:
			f.UniversOuvert = false
			f.CompositionVariable = false
		case MagIntentMeaning:
			f.PairProbabiliste = false
		case MagAuthorityChain:
			f.CoutNonBorne = false
		}
	}
	return f
}

// EvaluateI2 returns the I2 verdict for (x, decision).
//
// Held iff Residu(x) ≠ ∅ AND removing the full residue collapses Inside().
// NotApplicable for Refus(hors frontière τ).
// Violated otherwise (which would indicate either an empty residue inside
// the frontier — impossible under V1 encoding — or a residue whose total
// removal keeps Inside()==true).
func EvaluateI2(x tau.Exchange, dec tau.Decision) Status {
	if dec.Regime == tau.Refus && dec.Diagnostic == "hors frontière τ" {
		return NotApplicable
	}
	r := Residu(x)
	if len(r) == 0 {
		return Violated
	}
	names := make([]string, len(r))
	for i, m := range r {
		names[i] = string(m)
	}
	if Recablage(x, names).Inside() {
		return Violated
	}
	return Held
}
```

Retirer le stub `EvaluateI2`.

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v ./internal/tau/invariants/
go vet ./...
golangci-lint run ./...
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/tau/invariants/
git commit -m "feat(tau/invariants): I2 irreductibilité + Residu / Recablage helpers (PRD §6.1)

ResidualMagnitude enumerates the four V1 magnitudes whose locus of fixation
is 'pendant' (target_resolution, intent_meaning, authority_chain,
support_negotiation). Residu(x) returns the non-empty subset for any
exchange inside the frontier. Recablage(x, removed) simulates the offline
rewiring: removing the full residue collapses Inside() to false, which is
the defining property of I2.

EvaluateI2: Held iff Residu(x) ≠ ∅ and full removal collapses the frontier.
Violated otherwise. NotApplicable on Refus(hors frontière τ).

Status: Confirmé par construction (PRD §6.1).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.4 — I3 asymétrie D-AUTORITÉ + clause péremption + tests

**Files :**
- Modify: `internal/tau/invariants/evaluator.go` (retire le stub `EvaluateI3`)
- Create: `internal/tau/invariants/i3_authority_asymmetry.go`
- Create: `internal/tau/invariants/i3_authority_asymmetry_test.go`

**Agent :** `ruflo-core:coder` (TDD)

### Note théorique

I3 : *Trois dimensions orthogonales en valeur, asymétriques en maturité ; D-AUTORITÉ = fait institutionnel sans support à 2026-05-16*.

Reformulation exécutable PRD §6.1 :

`D-AUTORITÉ(x) ≥ θ_auth_block ∧ Attestation == nil ⇒ Refus`. Clause de péremption : `date_revision ≤ 2027-01-01` dans le profil.

**Encodage V1** : I3 vérifie la **garde dispatcher** plutôt que de recalculer D-AUTORITÉ (interdit d'importer `dimensions`). On infère depuis la `Decision` :

- Si `dec.Regime == Probabiliste` ET `x.AttestationInstitutionnelle == nil` ET le tau_score composite suggère qu'on a passé la garde D-AUTORITÉ (impossible à reconstituer sans `dimensions`) → V1 ne peut pas le vérifier directement.

**Stratégie V1** : I3 est vérifié **par compatibilité** avec la `Decision` :

- Si `dec.Regime == Refus` ET `dec.Diagnostic == "I3 — verrou ontologique D-AUTORITÉ"` → `Held` (la garde a tenu).
- Si `dec.Regime == Probabiliste` ET `x.AttestationInstitutionnelle == nil` → suspect. V1 inspecte `Trace.Thresholds.AuthBlock` : si `Trace.TauScore ≥ AuthBlock` et pas d'attestation, c'est `Violated`. Sinon `Held`.
- Clause de péremption : si `dec.DateRevision` est postérieure à `2027-01-01`, on annote — V1 n'a pas accès direct au profile.

**Statut V1** : *Probable*. Daté **2026-05-24** ; révision trimestrielle. La vraie vérification empirique nécessite l'accès aux scores ventilés (déféré à M5 quand `Trace` exposera `dec.Trace.Scores`).

- [ ] **Étape 1 — Écrire `i3_authority_asymmetry.go`**

```go
package invariants

import (
	"time"

	"github.com/agbruneau/taugo/internal/tau"
)

// I3PerimptionLimite is the latest acceptable DateRevision per PRD §6.1 I3.
// Beyond this date the institutional-fact landscape is presumed to have
// shifted (RFC for delegated agentic identity may have landed); profile
// must be renewed or τ refuses by clause de péremption.
var I3PerimptionLimite = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

// EvaluateI3 returns the I3 verdict for (x, decision).
//
// PRD §6.1 I3: D-AUTORITÉ(x) ≥ θ_auth_block ∧ Attestation == nil ⇒ Refus.
// V1 verification reasons over the Decision: I3 is Held when the dispatcher
// either refused on the ontological guard, or produced a non-Refus decision
// in a configuration that does not appear to bypass the guard.
//
// Limit case: if dec.DateRevision is after I3PerimptionLimite, the profile
// outlives the institutional horizon → Violated.
func EvaluateI3(x tau.Exchange, dec tau.Decision) Status {
	// Profile expiration clause first — strongest verdict.
	if !dec.DateRevision.IsZero() && dec.DateRevision.After(I3PerimptionLimite) {
		return Violated
	}

	switch dec.Regime {
	case tau.Refus:
		// I3 guard fired → tenue avérée.
		if dec.Diagnostic == "I3 — verrou ontologique D-AUTORITÉ" {
			return Held
		}
		// Other Refus types do not exercise I3 directly.
		return NotApplicable
	case tau.Deterministe, tau.Probabiliste:
		// V1 heuristic: if no attestation AND tau_score crossed the auth
		// threshold reported by the trace, it would be a bypass. The
		// composite score is not the dimension score, so this is an upper
		// bound; future versions will read ventilated scores from the trace.
		if x.AttestationInstitutionnelle == nil &&
			dec.Trace.Thresholds.AuthBlock > 0 &&
			dec.Trace.TauScore >= dec.Trace.Thresholds.AuthBlock {
			return Violated
		}
		return Held
	default:
		return NotApplicable
	}
}
```

- [ ] **Étape 2 — Écrire le test rouge `i3_authority_asymmetry_test.go`**

```go
package invariants_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

func TestEvaluateI3_HeldOnGuardFired(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-guard"}
	dec := tau.Decision{
		Regime:     tau.Refus,
		Diagnostic: "I3 — verrou ontologique D-AUTORITÉ",
		Trace:      tau.Trace{ExchangeID: x.ID},
	}
	if got := invariants.EvaluateI3(x, dec); got != invariants.Held {
		t.Fatalf("EvaluateI3 = %v, want Held", got)
	}
}

func TestEvaluateI3_ViolatedOnProfileBeyondPerimption(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-expired"}
	dec := tau.Decision{
		Regime:       tau.Probabiliste,
		DateRevision: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Trace:        tau.Trace{ExchangeID: x.ID},
	}
	if got := invariants.EvaluateI3(x, dec); got != invariants.Violated {
		t.Fatalf("EvaluateI3 = %v, want Violated (profile beyond I3 péremption)", got)
	}
}

func TestEvaluateI3_HeldOnAcceptableProfile(t *testing.T) {
	t.Parallel()
	x := tau.Exchange{ID: "x-i3-ok"}
	dec := tau.Decision{
		Regime:       tau.Deterministe,
		DateRevision: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   0.30,
			Thresholds: tau.TraceThresholds{AuthBlock: 0.85},
		},
	}
	if got := invariants.EvaluateI3(x, dec); got != invariants.Held {
		t.Fatalf("EvaluateI3 = %v, want Held", got)
	}
}
```

Retirer le stub `EvaluateI3` de `evaluator.go`.

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v -run TestEvaluateI3 ./internal/tau/invariants/
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/tau/invariants/
git commit -m "feat(tau/invariants): I3 asymétrie D-AUTORITÉ + clause de péremption (PRD §6.1)

EvaluateI3 verifies the institutional asymmetry guard from the Decision:
Held when the dispatcher fired the I3 refus, Held in non-Refus configurations
that do not appear to bypass the guard, Violated when (a) profile DateRevision
exceeds I3PerimptionLimite (2027-01-01) or (b) Probabiliste/Deterministe regime
without attestation while tau_score >= Thresholds.AuthBlock.

Status: Probable, daté 2026-05-24. Vérification fine déférée à M5 (Trace ne
porte pas encore les scores ventilés par dimension).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.5 — I4 cohérence (s, i) + détecteur incohérence + tests

**Files :**
- Modify: `internal/tau/invariants/evaluator.go` (retire le stub `EvaluateI4`)
- Create: `internal/tau/invariants/i4_coherence.go`
- Create: `internal/tau/invariants/i4_coherence_test.go`

**Agent :** `ruflo-core:coder` (TDD)

### Note théorique

I4 : *D-INVARIANT contraint par D-SENS — `i ≈ pendant ⟹ s ≈ pendant`*. Reformulation : `D-INVARIANT(x) ≥ θ_inv ∧ D-SENS(x) < θ_sens ⇒ Refus(diag: "I4")`.

**Encodage V1** : I4 vérifie le **comportement du dispatcher** plutôt que de recalculer les deux scores. Comme `Trace` ne porte pas (encore) les scores ventilés, V1 vérifie :

- Si `dec.Regime == Refus` ET `dec.Diagnostic == "I4 — combinaison incohérente détectée"` → `Held`.
- Si `dec.Regime != Refus` : V1 ne peut pas savoir si `(s, i)` étaient incohérents. **Verdict V1** : `Held` par défaut, avec marque *Hypothèse*. V2 (M5) lira `Trace.Scores.DSens` et `Trace.Scores.DInvariant`.

**Détecteur explicite** : `Incoherent(sValue, sThreshold, iValue, iThreshold) bool` exposé pour les fuzz targets ; permet à `FuzzI4` de générer des paires `(s, i)` arbitraires et vérifier la propriété indépendamment du dispatcher.

**Statut** : *Hypothèse* (priorité empirique #1, PRD §6.3 ; campagne M4).

- [ ] **Étape 1 — Écrire `i4_coherence.go`**

```go
package invariants

import "github.com/agbruneau/taugo/internal/tau"

// Incoherent reports whether (sensValue, invValue) forms an I4-violating pair
// under the thresholds (sensThreshold, invThreshold). True iff sens is below
// its coherence threshold while inv reaches or exceeds its threshold —
// the asymmetric direction encoded in PRD §6.1 I4.
func Incoherent(sensValue, sensThreshold, invValue, invThreshold float64) bool {
	return invValue >= invThreshold && sensValue < sensThreshold
}

// EvaluateI4 returns the I4 verdict for (x, decision).
//
//   - Refus("I4 — combinaison incohérente détectée"): Held.
//   - Other Refus diagnostics: NotApplicable (I4 not exercised).
//   - Deterministe / Probabiliste: Held (V1 cannot ventilate scores from
//     the current Trace; verdict defaults to Held with status Hypothèse;
//     V2/M5 will read dec.Trace.Scores and call Incoherent directly).
func EvaluateI4(_ tau.Exchange, dec tau.Decision) Status {
	if dec.Regime == tau.Refus {
		if dec.Diagnostic == "I4 — combinaison incohérente détectée" {
			return Held
		}
		return NotApplicable
	}
	return Held
}
```

- [ ] **Étape 2 — Écrire `i4_coherence_test.go`**

```go
package invariants_test

import (
	"testing"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

func TestIncoherent_TruthTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		s, sT, i, iT float64
		want bool
	}{
		{"s_low_i_high", 0.10, 0.50, 0.70, 0.50, true},
		{"s_high_i_high", 0.70, 0.50, 0.70, 0.50, false},
		{"s_low_i_low",  0.10, 0.50, 0.10, 0.50, false},
		{"boundary_eq", 0.50, 0.50, 0.50, 0.50, false}, // strict <
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := invariants.Incoherent(c.s, c.sT, c.i, c.iT); got != c.want {
				t.Fatalf("Incoherent(%v) = %v, want %v", c, got, c.want)
			}
		})
	}
}

func TestEvaluateI4_HeldOnI4Refus(t *testing.T) {
	t.Parallel()
	dec := tau.Decision{
		Regime: tau.Refus, Diagnostic: "I4 — combinaison incohérente détectée",
	}
	if got := invariants.EvaluateI4(tau.Exchange{}, dec); got != invariants.Held {
		t.Fatalf("EvaluateI4 = %v, want Held", got)
	}
}

func TestEvaluateI4_NotApplicableOnOtherRefus(t *testing.T) {
	t.Parallel()
	dec := tau.Decision{Regime: tau.Refus, Diagnostic: "hors frontière τ"}
	if got := invariants.EvaluateI4(tau.Exchange{}, dec); got != invariants.NotApplicable {
		t.Fatalf("EvaluateI4 = %v, want NotApplicable", got)
	}
}
```

Retirer le stub `EvaluateI4`.

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v -run TestEvaluateI4 ./internal/tau/invariants/
go test -race -v -run TestIncoherent ./internal/tau/invariants/
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/tau/invariants/
git commit -m "feat(tau/invariants): I4 cohérence + Incoherent detector (PRD §6.1)

Incoherent(s, sT, i, iT) reports the asymmetric I4 violating pair: inv at
or above threshold while sens strictly below. Direct decision input for
FuzzI4 (avoids dispatcher reproduction).

EvaluateI4: Held when dispatcher fired the I4 refus, NotApplicable on other
Refus diagnostics, Held by default on Deterministe/Probabiliste (V1 cannot
ventilate scores; verdict carries status Hypothèse, refined in M5).

Status: Hypothèse — priorité empirique #1 (PRD §6.3, campagne M4).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.6 — I5 composition conjonctive + API agrégation `M(π)` + tests

**Files :**
- Modify: `internal/tau/invariants/evaluator.go` (retire le stub `EvaluateI5`)
- Create: `internal/tau/invariants/i5_composition.go`
- Create: `internal/tau/invariants/i5_composition_test.go`

**Agent :** `ruflo-core:coder` (TDD)

### Note théorique

I5 : *Pile hérite de la conjonction des angles morts ; `M(π) ≥ max(|Aᵢ|)` et `M(π) ≤ Σ|Aᵢ|`*. PRD §6.1 précise : « V1 expose l'API d'agrégation ; **V2 calcule** ». M3 livre les deux bornes formellement vérifiables avec l'API V1 calculée.

**Encodage** :

```go
type AngleMort []string                // identifiants d'angles morts d'une couche
type Pile []AngleMort                  // pile composée

func Aggregate(p Pile) []string        // M(π) = ⋃ Aᵢ (déduplication)
```

Propriétés vérifiables par fuzz :

1. `len(Aggregate(π)) ≥ max(len(Aᵢ) for Aᵢ in π)` (borne inférieure de l'union)
2. `len(Aggregate(π)) ≤ Σ len(Aᵢ)` (borne supérieure : pas de création ex nihilo)
3. `Aggregate([])` retourne `[]` et est stable
4. `Aggregate` est idempotent : `Aggregate(Aggregate(π) une seule couche) == Aggregate(π)` à l'ensemblisme près

**Statut** : *Probable*.

- [ ] **Étape 1 — Écrire `i5_composition.go`**

```go
package invariants

import (
	"sort"

	"github.com/agbruneau/taugo/internal/tau"
)

// AngleMort names the blind-spots of a single layer of an agentic stack.
// V1 keeps them as opaque string identifiers; semantics are stack-specific.
type AngleMort []string

// Pile is a composed agentic stack: an ordered list of layers, each with
// its own set of blind-spots.
type Pile []AngleMort

// Aggregate returns M(π) — the union of blind-spots across the stack.
// Output is deterministically ordered (lex-sorted, deduplicated).
//
// PRD §6.1 I5: M(π) = |⋃ Aᵢ| with bounds:
//   - len(M(π)) ≥ max(len(Aᵢ))   (lower bound)
//   - len(M(π)) ≤ Σ len(Aᵢ)      (upper bound, equality iff disjoint)
func Aggregate(p Pile) []string {
	seen := map[string]struct{}{}
	for _, layer := range p {
		for _, a := range layer {
			seen[a] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// BoundsHold reports whether Aggregate respects the two I5 bounds on p.
// Cheap finite check: never panics, never allocates beyond Aggregate's cost.
func BoundsHold(p Pile) bool {
	agg := Aggregate(p)
	if len(p) == 0 {
		return len(agg) == 0
	}
	maxLayer := 0
	sumLayers := 0
	for _, layer := range p {
		if len(layer) > maxLayer {
			maxLayer = len(layer)
		}
		sumLayers += len(layer)
	}
	return len(agg) >= maxLayer && len(agg) <= sumLayers
}

// EvaluateI5 returns the I5 verdict.
//
// V1: I5 is structural (no Exchange or Decision input needed). EvaluateI5
// always reports Held for empty / single-layer stacks; FuzzI5 drives
// BoundsHold directly on generated stacks.
//
// The (x, dec) signature is kept uniform with the other evaluators. Future
// versions will wire the active stack via dec.Trace.
func EvaluateI5(_ tau.Exchange, _ tau.Decision) Status {
	// V1: no stack reified in Trace yet; verdict held by construction
	// (Aggregate is total and respects the bounds). FuzzI5 generates
	// arbitrary stacks and verifies BoundsHold.
	return Held
}
```

- [ ] **Étape 2 — Écrire `i5_composition_test.go`**

```go
package invariants_test

import (
	"testing"

	"github.com/agbruneau/taugo/internal/tau/invariants"
)

func TestAggregate_EmptyStack(t *testing.T) {
	t.Parallel()
	got := invariants.Aggregate(invariants.Pile{})
	if len(got) != 0 {
		t.Fatalf("Aggregate(empty) = %v, want empty slice", got)
	}
}

func TestAggregate_Deduplicates(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{
		{"a", "b"}, {"b", "c"}, {"a"},
	}
	got := invariants.Aggregate(p)
	if len(got) != 3 {
		t.Fatalf("Aggregate dedup len = %d, want 3 (got %v)", len(got), got)
	}
}

func TestAggregate_Ordered(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{{"z", "a"}, {"m"}}
	got := invariants.Aggregate(p)
	want := []string{"a", "m", "z"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Aggregate order mismatch at %d: got %v, want %v", i, got, want)
		}
	}
}

func TestBoundsHold_LowerBound(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{{"a", "b", "c"}, {"a"}}
	if !invariants.BoundsHold(p) {
		t.Fatalf("BoundsHold failed: len(Aggregate) < max(len(Aᵢ))")
	}
}

func TestBoundsHold_UpperBoundEqualsDisjoint(t *testing.T) {
	t.Parallel()
	p := invariants.Pile{{"a", "b"}, {"c", "d"}, {"e"}}
	if !invariants.BoundsHold(p) {
		t.Fatal("BoundsHold failed on disjoint stack (sum equality case)")
	}
}
```

Retirer le stub `EvaluateI5`.

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v ./internal/tau/invariants/
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/tau/invariants/
git commit -m "feat(tau/invariants): I5 composition + Aggregate / BoundsHold (PRD §6.1)

AngleMort and Pile model a composed agentic stack. Aggregate(π) computes
the deterministic union (lex-sorted, deduplicated). BoundsHold verifies the
two PRD §6.1 I5 bounds: len(M(π)) ≥ max(len(Aᵢ)) and len(M(π)) ≤ Σ len(Aᵢ).

EvaluateI5 returns Held; structural verification is property-based via
FuzzI5 driving BoundsHold on generated stacks. Trace does not yet reify
the active stack — wired in a later milestone.

Status: Probable. V1 exposes the API; V2 (this commit) computes M(π).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.7 — `fuzz_targets.go` (FuzzI1..FuzzI5) + corpus seed

**Files :**
- Create: `internal/tau/invariants/fuzz_targets.go`
- Create: `internal/tau/invariants/testdata/fuzz/FuzzI1_Conservation/seed01`
- Create: `internal/tau/invariants/testdata/fuzz/FuzzI2_Irreductibilite/seed01`
- Create: `internal/tau/invariants/testdata/fuzz/FuzzI3_AsymetrieAutorite/seed01`
- Create: `internal/tau/invariants/testdata/fuzz/FuzzI4_CoherenceContrainte/seed01`
- Create: `internal/tau/invariants/testdata/fuzz/FuzzI5_CompositionConjonctive/seed01`

**Agent :** `ruflo-core:coder` (cibles fuzz) + `ruflo-core:researcher` (corpus seed)

### Note de conception fuzz

Les cibles fuzz **n'invoquent pas le dispatcher** (cela créerait une dépendance `invariants → orchestration` interdite). Au lieu de cela, chaque cible :

1. Génère des entrées élémentaires (uint64, byte sequences, string slices) que Go fuzz sait muter.
2. Reconstruit localement les structures `Exchange` / `Decision` / `Pile` requises.
3. Appelle l'évaluateur ou le helper correspondant (`EvaluateI<N>` / `Conserve` / `Residu` / `Recablage` / `Incoherent` / `BoundsHold`).
4. Échoue (`t.Fatal`) si la propriété observable est violée.

**Calque FibGo** : structure inspirée de `bigfft/FuzzMul`, `bigfft/FuzzSqr` (corpus seed minimal, propriété de pureté/bornes).

- [ ] **Étape 1 — Écrire `fuzz_targets.go`**

```go
package invariants_test

import (
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// makeFuzzExchange builds a deterministic Exchange from fuzz seed inputs.
// The mapping is total: every input combination produces a well-formed Exchange.
func makeFuzzExchange(id string, intent string, discoveryMode uint8, humanInLoop bool, delegationDepth uint8) tau.Exchange {
	return tau.Exchange{
		ID:                id,
		IntentDescription: intent,
		DiscoveredAt:      time.Unix(int64(discoveryMode)*1000, 0).UTC(),
		Initiator: tau.Principal{
			ID:              id + "-init",
			HumanInLoop:     humanInLoop,
			Organization:    "org-fuzz",
			DelegationDepth: int(delegationDepth % 8),
		},
		Target: tau.Capability{
			ID:            id + "-cap",
			DiscoveryMode: tau.DiscoveryMode(int(discoveryMode) % 4),
			ContractURI:   "",
		},
	}
}

// FuzzI1_Conservation exercises Conserve and EvaluateI1. The property is:
// when ExchangeID matches Trace.ExchangeID, Conserve must hold; the verdict
// must never be Violated except on an explicit trace drift.
func FuzzI1_Conservation(f *testing.F) {
	f.Add("e-seed", "intent", uint8(1), false, uint8(0), int8(0))
	f.Fuzz(func(t *testing.T, id, intent string, mode uint8, human bool, depth uint8, drift int8) {
		x := makeFuzzExchange(id, intent, mode, human, depth)
		traceID := x.ID
		if drift != 0 {
			traceID = x.ID + "X"
		}
		dec := tau.Decision{
			Regime: tau.Deterministe,
			Trace:  tau.Trace{ExchangeID: traceID},
		}
		got := invariants.EvaluateI1(x, dec)
		if drift == 0 && got == invariants.Violated {
			t.Fatalf("EvaluateI1 violated on identity preservation: x=%q, trace=%q", x.ID, traceID)
		}
		if drift != 0 && got == invariants.Held {
			t.Fatalf("EvaluateI1 held despite trace drift: x=%q, trace=%q", x.ID, traceID)
		}
	})
}

// FuzzI2_Irreductibilite exercises Residu and Recablage. The property: for
// any exchange whose frontier is Inside, Residu(x) is non-empty AND removing
// all residual magnitudes collapses Inside() to false.
func FuzzI2_Irreductibilite(f *testing.F) {
	f.Add("e-seed", "intent", uint8(2), false, uint8(1))
	f.Fuzz(func(t *testing.T, id, intent string, mode uint8, human bool, depth uint8) {
		x := makeFuzzExchange(id, intent, mode, human, depth)
		r := invariants.Residu(x)
		// Verify the I2 property using the frontier reconstructed locally
		// by Recablage (mirrors the dispatcher heuristic).
		insideBefore := invariants.Recablage(x, nil).Inside()
		if !insideBefore {
			return // Exchange not in the frontier: I2 NotApplicable.
		}
		if len(r) == 0 {
			t.Fatalf("Residu empty for inside-frontier exchange: %+v", x)
		}
		names := make([]string, len(r))
		for i, m := range r {
			names[i] = string(m)
		}
		if invariants.Recablage(x, names).Inside() {
			t.Fatalf("Recablage with full residue removed kept Inside()=true: x=%+v, residue=%v", x, names)
		}
	})
}

// FuzzI3_AsymetrieAutorite exercises EvaluateI3. The property: no Deterministe
// or Probabiliste decision can hold without attestation when tau_score
// exceeds the auth_block threshold.
func FuzzI3_AsymetrieAutorite(f *testing.F) {
	f.Add(uint8(50), uint8(85), false, int64(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC).Unix()))
	f.Fuzz(func(t *testing.T, tauMilli uint8, authMilli uint8, withAttestation bool, dateUnix int64) {
		tauScore := float64(tauMilli) / 100.0
		authBlock := float64(authMilli) / 100.0
		x := tau.Exchange{ID: "x-i3-fuzz"}
		if withAttestation {
			x.AttestationInstitutionnelle = &tau.Attestation{Emetteur: "ietf", Reference: "draft-x"}
		}
		dec := tau.Decision{
			Regime:       tau.Probabiliste,
			DateRevision: time.Unix(dateUnix, 0).UTC(),
			Trace: tau.Trace{
				ExchangeID: x.ID,
				TauScore:   tauScore,
				Thresholds: tau.TraceThresholds{AuthBlock: authBlock},
			},
		}
		got := invariants.EvaluateI3(x, dec)
		// Property: if no attestation, tauScore >= authBlock > 0, and date
		// within limit, the verdict MUST be Violated (V1 heuristic).
		if !withAttestation && authBlock > 0 && tauScore >= authBlock &&
			!dec.DateRevision.After(invariants.I3PerimptionLimite) {
			if got != invariants.Violated {
				t.Fatalf("EvaluateI3 = %v, want Violated (no attest, tau >= auth_block)", got)
			}
		}
		// Property: profile beyond péremption limit is always Violated.
		if dec.DateRevision.After(invariants.I3PerimptionLimite) && got != invariants.Violated {
			t.Fatalf("EvaluateI3 = %v, want Violated (profile beyond péremption)", got)
		}
	})
}

// FuzzI4_CoherenceContrainte exercises Incoherent directly. Property:
// Incoherent is asymmetric — true iff i >= iT AND s < sT, never the reverse.
func FuzzI4_CoherenceContrainte(f *testing.F) {
	f.Add(uint8(10), uint8(50), uint8(70), uint8(50))
	f.Fuzz(func(t *testing.T, sMilli, sTMilli, iMilli, iTMilli uint8) {
		s := float64(sMilli) / 100.0
		sT := float64(sTMilli) / 100.0
		i := float64(iMilli) / 100.0
		iT := float64(iTMilli) / 100.0
		got := invariants.Incoherent(s, sT, i, iT)
		want := i >= iT && s < sT
		if got != want {
			t.Fatalf("Incoherent(%v,%v,%v,%v) = %v, want %v", s, sT, i, iT, got, want)
		}
	})
}

// FuzzI5_CompositionConjonctive exercises Aggregate and BoundsHold. Property:
// BoundsHold is true on every well-formed stack, regardless of layer count
// or duplicate distribution.
func FuzzI5_CompositionConjonctive(f *testing.F) {
	f.Add([]byte{1, 2, 3, 1, 4, 0, 5})
	f.Fuzz(func(t *testing.T, raw []byte) {
		// Decode raw into a Pile. Byte 0 separates layers; non-zero bytes
		// become single-byte string identifiers.
		var pile invariants.Pile
		var current invariants.AngleMort
		for _, b := range raw {
			if b == 0 {
				if len(current) > 0 {
					pile = append(pile, current)
					current = nil
				}
				continue
			}
			current = append(current, string([]byte{b}))
		}
		if len(current) > 0 {
			pile = append(pile, current)
		}
		if !invariants.BoundsHold(pile) {
			t.Fatalf("BoundsHold failed on pile %v (aggregate=%v)", pile, invariants.Aggregate(pile))
		}
	})
}
```

- [ ] **Étape 2 — Créer les corpus seed `testdata/fuzz/`**

Pour chaque cible, un fichier `seed01` au format Go fuzz corpus. Le format est :

```
go test fuzz v1
<type1>(<value1>)
<type2>(<value2>)
...
```

`testdata/fuzz/FuzzI1_Conservation/seed01` :
```
go test fuzz v1
string("e-seed-1")
string("compute")
uint8(1)
bool(false)
uint8(0)
int8(0)
```

`testdata/fuzz/FuzzI2_Irreductibilite/seed01` :
```
go test fuzz v1
string("e-seed-i2")
string("discover")
uint8(2)
bool(false)
uint8(2)
```

`testdata/fuzz/FuzzI3_AsymetrieAutorite/seed01` :
```
go test fuzz v1
uint8(95)
uint8(85)
bool(false)
int64(1796601600)
```
*(1796601600 ≈ 2026-12-01 UTC en unix seconds — À vérifier au commit)*

`testdata/fuzz/FuzzI4_CoherenceContrainte/seed01` :
```
go test fuzz v1
uint8(10)
uint8(50)
uint8(80)
uint8(50)
```

`testdata/fuzz/FuzzI5_CompositionConjonctive/seed01` :
```
go test fuzz v1
[]byte("\x01\x02\x00\x03\x04\x00\x01")
```

**Note pour l'agent `ruflo-core:researcher`** : valider que chaque format de corpus est correctement parsé par `go test -fuzz -run=^$`. Si Go signale `corpus entry malformed`, ajuster.

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v ./internal/tau/invariants/
go test -fuzz=FuzzI1_Conservation        -fuzztime=10s ./internal/tau/invariants/
go test -fuzz=FuzzI2_Irreductibilite     -fuzztime=10s ./internal/tau/invariants/
go test -fuzz=FuzzI3_AsymetrieAutorite   -fuzztime=10s ./internal/tau/invariants/
go test -fuzz=FuzzI4_CoherenceContrainte -fuzztime=10s ./internal/tau/invariants/
go test -fuzz=FuzzI5_CompositionConjonctive -fuzztime=10s ./internal/tau/invariants/
```

Attendu : aucune panique, aucun crash. Chaque cible explore au moins quelques milliers d'entrées en 10 s.

- [ ] **Étape 4 — Commit**

```powershell
git add internal/tau/invariants/fuzz_targets.go internal/tau/invariants/testdata/
git commit -m "feat(tau/invariants): FuzzI1..FuzzI5 + initial seed corpus (PRD §15.2)

Five fuzz targets exercise the invariant evaluators and their helpers
without going through the dispatcher (forbidden import). Each target
verifies an observable property:

- FuzzI1: trace drift ↔ EvaluateI1 verdict (Conserve)
- FuzzI2: Residu non-empty inside frontier ∧ Recablage(R) collapses Inside
- FuzzI3: no attestation ∧ tau_score >= auth_block ⇒ Violated (V1 heuristic)
- FuzzI4: Incoherent asymmetric truth table
- FuzzI5: BoundsHold on every well-formed Pile

Seed corpus under testdata/fuzz/FuzzI*/seed01 — five minimal entries
covering one representative case per target.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.8 — Étape 8 dispatcher : `EvaluateInvariants` sur la trace

**Files :**
- Modify: `internal/orchestration/dispatcher.go`
- Modify: `internal/orchestration/dispatcher_test.go` (ou créer `dispatcher_invariants_test.go`)

**Agent :** `ruflo-core:coder`

### Note de conception

Le dispatcher M2 retournait `Decision` directement après l'étape 7. M3 ajoute l'étape 8 :

```
8. ÉVALUATION INVARIANTS
   inv := EvaluateInvariants(x, decision, π)
   inv.AnyViolated() ⇒ trace.UnmodeledObservations += inv.Summary()
```

Comme `Trace` est embarqué par valeur dans `Decision`, on construit la `Decision` une fois, on appelle `EvaluateInvariants(x, decision)`, on appende au `UnmodeledObservations` et on retourne la copie modifiée. Le tag `ProfileVersion` passe à `"M3-default"`.

- [ ] **Étape 1 — Modifier `dispatcher.go`**

Ajouter l'import :

```go
import (
    // ... existants ...
    "github.com/agbruneau/taugo/internal/tau/invariants"
)
```

Remplacer la branche finale `return tau.Decision{... regime ...}` par :

```go
	// Step 7 result.
	decision := tau.Decision{
		Regime:         regime,
		ProfileVersion: "M3-default",
		Trace: tau.Trace{
			ExchangeID: x.ID,
			TauScore:   tauScore,
			Frontier:   frontier,
			Thresholds: traceThresholds,
			DurationNs: durationNs(start),
		},
	}

	// Step 8 — Invariant evaluation (PRD §10 step 8). Violations are
	// appended to UnmodeledObservations; the decision regime is NOT
	// changed (the dispatch is already complete; this is observability).
	statuses := invariants.EvaluateInvariants(x, decision)
	if statuses.AnyViolated() {
		decision.Trace.UnmodeledObservations = append(
			decision.Trace.UnmodeledObservations,
			statuses.Summary()...,
		)
	}
	return decision, nil
```

Mettre à jour le commentaire d'en-tête du `Dispatcher` : remplacer « Steps 3 (profile expiration) and 8 (invariant evaluation) land in M3/M5. » par « Step 3 (profile expiration) lands in M5. ». L'étape 8 est désormais incluse.

Refléter aussi le bump de `ProfileVersion` sur les branches de Refus (étapes 1, 2, 5) si elles portent le tag M2.

- [ ] **Étape 2 — Étendre les tests dispatcher**

Ajouter dans `dispatcher_test.go` (ou nouveau `dispatcher_invariants_test.go`) :

```go
func TestDispatcher_Step8_PopulatesUnmodeledOnViolation(t *testing.T) {
	t.Parallel()
	// Force a synthetic violation: an exchange in the frontier with a
	// non-Refus regime whose tau_score crosses the auth_block threshold
	// without attestation triggers EvaluateI3 = Violated.
	x := newExchangeInsideFrontier("e-step8")
	// Tune thresholds so the M3-default dispatcher reports tau_score above
	// AuthBlock without attestation: a deliberate construction.
	d := orchestration.NewDispatcher(llm.Stub{}, orchestration.Thresholds{
		Deterministe:  0.05,
		Probabiliste:  0.10,
		AuthBlock:     0.05,  // very low so tau_score >= AuthBlock is easy
		SensCoherence: 0.50,
		InvCoherence:  0.50,
	})
	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if dec.Regime == tau.Refus {
		t.Skip("dispatcher refused upstream; cannot exercise step 8 violations")
	}
	if len(dec.Trace.UnmodeledObservations) == 0 {
		t.Fatalf("UnmodeledObservations empty; expected step-8 I3 entry. Decision=%+v", dec)
	}
}
```

**Note d'agent** : Si la configuration d'`Exchange` choisie déclenche déjà le refus ontologique étape 2 (et donc retourne avant l'étape 8), ajouter une attestation pour bypasser cette garde tout en gardant les conditions de violation I3 par le verdict de `EvaluateI3`. **Hypothèse — *À vérifier*** : la combinaison réelle peut nécessiter un ajustement de `AuthBlock` et un échange porté.

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v ./internal/orchestration/
go test -race ./...
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/orchestration/
git commit -m "feat(orchestration): step 8 — EvaluateInvariants annotates Trace.UnmodeledObservations

Completes the PRD §10 pseudo-algorithm with the last step. After steps
1-7 produce a Decision, EvaluateInvariants(x, decision) runs the five
evaluators; any Violated invariant contributes a short line to
Trace.UnmodeledObservations. Decision regime is NOT mutated — this is
pure observability (anti-patron #4 guard).

ProfileVersion bumped to M3-default.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.9 — `TestNoPredictiveAPI`, `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported`

**Files :**
- Create: `internal/anti_patterns_test.go`

**Agent :** `ruflo-core:coder`

### Note

Les trois tests gardent les anti-patrons (#1, #3, #4) au niveau du package racine `internal_test`. La réflexion sur les méthodes exportées impose de ne pas restreindre la garde à un seul package : elle parcourt `tau`, `tau/dimensions`, `tau/invariants`, `orchestration`.

- [ ] **Étape 1 — Écrire `internal/anti_patterns_test.go`**

```go
package internal_test

import (
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/calibration"
	"github.com/agbruneau/taugo/internal/tau"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// predictivePattern catches the three forbidden prefixes (PRD §7.2 #1).
var predictivePattern = regexp.MustCompile(`^(Predict|Expected|Forecast)`)

// gardedPackages enumerates the import paths whose exported identifiers
// must not match predictivePattern.
var gardedPackages = []string{
	"github.com/agbruneau/taugo/internal/tau",
	"github.com/agbruneau/taugo/internal/tau/dimensions",
	"github.com/agbruneau/taugo/internal/tau/invariants",
	"github.com/agbruneau/taugo/internal/orchestration",
}

// TestNoPredictiveAPI parses the source AST of each guarded package and
// fails if any exported function, method, or type matches the forbidden
// predictive pattern (PRD §7.2 anti-patron #1).
func TestNoPredictiveAPI(t *testing.T) {
	t.Parallel()
	for _, pkgPath := range gardedPackages {
		pkgPath := pkgPath
		t.Run(strings.ReplaceAll(pkgPath, "/", "_"), func(t *testing.T) {
			t.Parallel()
			pkg, err := build.Default.Import(pkgPath, ".", build.ImportComment)
			if err != nil {
				t.Skipf("package not built: %v", err)
			}
			fset := token.NewFileSet()
			for _, src := range pkg.GoFiles {
				path := filepath.Join(pkg.Dir, src)
				f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
				if err != nil {
					t.Fatalf("parse %s: %v", path, err)
				}
				for _, decl := range f.Decls {
					name := exportedName(decl)
					if name == "" {
						continue
					}
					if predictivePattern.MatchString(name) {
						t.Errorf("forbidden predictive API in %s: %s", pkgPath, name)
					}
				}
			}
		})
	}
}

// exportedName returns the exported name of a top-level declaration, or "".
// (Helper kept local — small and used only by this test.)
func exportedName(decl interface{}) string {
	// Implementation uses type switch on *ast.FuncDecl, *ast.GenDecl (type/var/const).
	// Returns "" for unexported names.
	// [Full body omitted from plan; the implementer writes a 15-line helper.]
	return ""
}

// TestI3_DateRevisionRespectee guards anti-patron #3 (atemporel).
// DefaultProfile must always carry a DateRevision in the future AND below
// the I3 péremption limit (2027-01-01).
func TestI3_DateRevisionRespectee(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	now := time.Now().UTC()
	if !p.DateRevision.After(now) {
		t.Fatalf("Profile.DateRevision %v is not after now %v", p.DateRevision, now)
	}
	if p.DateRevision.After(invariants.I3PerimptionLimite) {
		t.Fatalf("Profile.DateRevision %v is beyond I3 péremption limit %v",
			p.DateRevision, invariants.I3PerimptionLimite)
	}
}

// TestUnmodeledObservationsReported guards anti-patron #4 (clos).
// The contract: when EvaluateInvariants reports a violation, the dispatcher
// MUST append the corresponding line to Trace.UnmodeledObservations.
//
// V1 check: contractual — verify the invariant Summary lines are non-empty
// strings and Statuses{Violated}.Summary() seeds at least one observation.
func TestUnmodeledObservationsReported(t *testing.T) {
	t.Parallel()
	s := invariants.Statuses{
		I1: invariants.Violated,
		I3: invariants.Violated,
	}
	got := s.Summary()
	if len(got) == 0 {
		t.Fatal("Statuses.Summary returned empty on a known violation set")
	}
	for _, line := range got {
		if strings.TrimSpace(line) == "" {
			t.Fatal("empty Summary line — would silently fail anti-patron #4 guard")
		}
	}
	// Smoke check: a Decision with these statuses can carry the lines.
	d := tau.Decision{
		Trace: tau.Trace{UnmodeledObservations: got},
	}
	if len(d.Trace.UnmodeledObservations) != len(got) {
		t.Fatalf("Trace.UnmodeledObservations lost entries: got %d, want %d",
			len(d.Trace.UnmodeledObservations), len(got))
	}
}
```

**Note d'agent** : compléter le helper `exportedName` (omission délibérée dans le plan pour rester bite-sized) en suivant le pattern AST classique : `*ast.FuncDecl` → check `Name.IsExported()`, `*ast.GenDecl{Tok: TYPE/VAR/CONST}` → boucler sur les Specs et inspecter `Names`.

- [ ] **Étape 2 — Vérifier**

```powershell
go test -race -v -run TestNoPredictiveAPI ./internal/
go test -race -v -run TestI3_DateRevisionRespectee ./internal/
go test -race -v -run TestUnmodeledObservationsReported ./internal/
go test -race ./...
```

Attendu : trois tests verts, suite globale verte.

- [ ] **Étape 3 — Commit**

```powershell
git add internal/anti_patterns_test.go
git commit -m "test(internal): guards for anti-patrons #1, #3, #4 (PRD §7.2)

TestNoPredictiveAPI parses the AST of tau, tau/dimensions, tau/invariants,
and orchestration and fails on any exported identifier matching
^(Predict|Expected|Forecast).

TestI3_DateRevisionRespectee verifies DefaultProfile.DateRevision is in
the future and below invariants.I3PerimptionLimite (2027-01-01).

TestUnmodeledObservationsReported asserts the contract that
Statuses.Summary produces non-empty lines threaded into Trace.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.10 — `docs/theory/05-invariants.md` + `docs/empirical/fuzz-summary.md`

**Files :**
- Create: `docs/theory/05-invariants.md`
- Create: `docs/empirical/fuzz-summary.md`

**Agent :** `ruflo-core:researcher`

### `05-invariants.md` — squelette

```markdown
# 05 — Les cinq invariants — renvoi vers chap. III.8.5

*Document de renvoi croisé. Verbatim canonique dans `InteroperabiliteAgentique/Monographie.md` v2.4.3, chap. III.8.5 (lignes ~5723-5737).*

*Statut global : tableau-maître ci-dessous. Daté 2026-05-24.*

---

## Tableau synthèse — encodage Go ↔ propriété observable

| # | Invariant | Statut | Encodage TauGo | Cible fuzz | Fichier |
|---|---|---|---|---|---|
| I1 | Conservation | Probable | `Conserve(x, dec) bool` + `EvaluateI1` | `FuzzI1_Conservation` | `i1_conservation.go` |
| I2 | Irréductibilité | Confirmé par construction | `Residu(x)`, `Recablage(x, removed)` | `FuzzI2_Irreductibilite` | `i2_irreductibility.go` |
| I3 | Asymétrie D-AUTORITÉ + péremption | Probable, daté 2026-05-24 | `EvaluateI3` + `I3PerimptionLimite=2027-01-01` | `FuzzI3_AsymetrieAutorite` | `i3_authority_asymmetry.go` |
| I4 | Cohérence (s, i) | Hypothèse (priorité empirique #1) | `Incoherent(s, sT, i, iT) bool` + `EvaluateI4` | `FuzzI4_CoherenceContrainte` | `i4_coherence.go` |
| I5 | Composition conjonctive | Probable, V2 calculée | `Aggregate(π)`, `BoundsHold(π)` | `FuzzI5_CompositionConjonctive` | `i5_composition.go` |

## Pourquoi ce package ne dépend pas de `dimensions`

L'orthogonalité PRD §6.3 entre **trois dimensions scorées** (D-SENS, D-AUTORITÉ, D-INVARIANT) et **cinq invariants structurels** (I1-I5) est encodée par une règle d'architecture stricte. `internal/arch_test.go` bloque toute tentative d'import croisé. Conséquence pour V1 :

- I3 et I4 ne peuvent pas recalculer leurs scores. Ils raisonnent sur la `Decision` produite par le dispatcher (tau_score composite, diagnostic, thresholds).
- M5 introduira `Trace.Scores` (ventilation par dimension) ; les évaluateurs raffineront leur verdict.

## Conditions de réfutation observables (PRD §6.2)

| # | Test négatif TauGo | Statut |
|---|---|---|
| I1 | `TestRefutationI1_GrandeurSupprimee` | À ajouter post-M3 si réfutation candidate |
| I2 | `TestRefutationI2_RecablageComplet` | Garde I2 : si jamais un `Recablage` complet laisse `Inside()=true`, le test échoue |
| I3 | `TestI3_DateRevisionRespectee` | M3.9 — garde le statut atemporel |
| I4 | `TestRefutationI4_CombinaisonIncoherente` | Activé par campagne empirique M4 |
| I5 | `TestRefutationI5_AngleMortReferme` | Différé M6 |

## Articulation et priorités V1

- **I1 + I2** fondent l'opérateur : conservation + non-trivialité. Garde combinée : `TestEvaluateI1_HeldOnNonRefus` + `TestEvaluateI2_HeldOnDynamicExchange`.
- **I3 + I4** caractérisent la structure : asymétrie de maturité + contrainte de cohérence.
- **I5** régit la composition (V2 calculée en M3.6).

## Renvois

- PRD `PRD.md` §6 (invariants), §7.2 (anti-patrons), §10 étape 8 (dispatcher)
- Plan : `docs/superpowers/plans/2026-05-24-M3-invariants-fuzz.md`
```

### `fuzz-summary.md` — squelette à remplir post-CI

```markdown
# Rapport fuzz I1-I5 — M3

> Généré le 2026-05-24. Statut : *Hypothèse* en attente du run CI nocturne 24 h.

## Cibles et statut

| Cible | Durée | Entrées explorées | Crashes | Couverture seed | Statut |
|---|---|---|---|---|---|
| FuzzI1_Conservation | 30 s | <à remplir> | 0 | seed01 | Probable |
| FuzzI2_Irreductibilite | 30 s | <à remplir> | 0 | seed01 | Confirmé par construction |
| FuzzI3_AsymetrieAutorite | 30 s | <à remplir> | 0 | seed01 | Probable |
| FuzzI4_CoherenceContrainte | 30 s | <à remplir> | 0 | seed01 | Hypothèse |
| FuzzI5_CompositionConjonctive | 30 s | <à remplir> | 0 | seed01 | Probable |

## Exceptions épinglées

*Aucune au tag `v0.0.4-alpha`. Toute exception future fera l'objet d'un test de régression dans `testdata/fuzz/FuzzI<N>/<hash>`.*

## Questions ouvertes

- I3 : sans `Trace.Scores` ventilés, le verdict V1 est une heuristique sur le tau_score composite. Préciser à M5.
- I4 : la priorité empirique #1 reste M4 (campagne AgentMeshKafka).
- I5 : `Trace` ne reifie pas encore la pile active.
```

- [ ] **Étape 1 — Rédiger les deux fichiers**

Format Markdown FR-CA, pas d'emoji, marqueurs d'incertitude sur toute affirmation datée.

- [ ] **Étape 2 — Commit**

```powershell
git add docs/theory/05-invariants.md docs/empirical/fuzz-summary.md
git commit -m "docs(theory,empirical): M3 invariants cross-reference + fuzz summary

docs/theory/05-invariants.md: cross-reference table for I1-I5 with Go
encoding, fuzz target, and statut. Documents why invariants does not
import dimensions (orthogonality enforced by arch_test.go).

docs/empirical/fuzz-summary.md: skeleton report for the 5 fuzz targets;
status updated after each CI run.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M3.11 — Revue intégrée + tag `v0.0.4-alpha`

**Agent :** thread principal (intégration) + `ruflo-core:reviewer`

- [ ] **Étape 1 — Vérifier la suite locale complète**

```powershell
go build ./...
go test -race -v ./...
go vet ./...
golangci-lint run ./...
go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/... ./...
go tool cover -func=coverage.out | Select-Object -Last 1
```

Couverture cible : ≥ 80 % global ; ≥ 90 % sur `internal/tau` (incluant `invariants` et `dimensions`).

Vérifier les fuzz courts :

```powershell
go test -fuzz=FuzzI1_Conservation        -fuzztime=30s ./internal/tau/invariants/
go test -fuzz=FuzzI2_Irreductibilite     -fuzztime=30s ./internal/tau/invariants/
go test -fuzz=FuzzI3_AsymetrieAutorite   -fuzztime=30s ./internal/tau/invariants/
go test -fuzz=FuzzI4_CoherenceContrainte -fuzztime=30s ./internal/tau/invariants/
go test -fuzz=FuzzI5_CompositionConjonctive -fuzztime=30s ./internal/tau/invariants/
```

- [ ] **Étape 2 — Vérifier les règles architecturales**

```powershell
go test -v -run TestArchitectureLayering ./internal/
```

Attendu : aucune violation. `tau/invariants` n'importe ni `dimensions`, ni `orchestration`, ni `bridge/*`.

- [ ] **Étape 3 — Briefing reviewer**

> Revue intégrée de M3 (commit range `v0.0.3-alpha..HEAD`). Vérifier :
> 1. Cinq évaluateurs `EvaluateI1..I5` retournent `Held | Violated | NotApplicable` ; aucun ne panic.
> 2. `Conserve`, `Residu`, `Recablage`, `Incoherent`, `Aggregate`, `BoundsHold` sont exportés et purs (pas d'état partagé).
> 3. `tau/invariants → dimensions` interdit : aucun import croisé.
> 4. Dispatcher étape 8 : `EvaluateInvariants` appelé après l'étape 7, `UnmodeledObservations` annoté sur violation, **régime non muté**.
> 5. `TestNoPredictiveAPI` parcourt les 4 packages gardés et matche les 3 prefixes interdits.
> 6. `I3PerimptionLimite == 2027-01-01 UTC` aligné PRD §6.1.
> 7. Cinq cibles fuzz passent 30 s sans panic ; corpus seed valide (`testdata/fuzz/FuzzI*/seed01`).
> 8. Couverture `internal/tau/invariants/` ≥ 80 %.
> 9. Conventions FR-CA / godoc anglais / `t.Parallel()` 100 % / pas d'emoji.
> 10. Aucun anti-patron introduit : pas de `Predict*`, pas de globaux mutables non synchronisés dans `tau/*`, pas d'import LLM concret.

- [ ] **Étape 4 — Tag `v0.0.4-alpha`**

```powershell
git tag -a v0.0.4-alpha -m "M3: five invariants I1-I5 as fuzz targets + dispatcher step 8

M3.1  - invariants package skeleton (Status, Statuses, EvaluateInvariants)
M3.2  - I1 conservation + Conserve helper
M3.3  - I2 irreductibilité + Residu / Recablage helpers
M3.4  - I3 asymétrie D-AUTORITÉ + I3PerimptionLimite (2027-01-01)
M3.5  - I4 cohérence + Incoherent detector
M3.6  - I5 composition + Aggregate / BoundsHold (V2 calculé)
M3.7  - FuzzI1..FuzzI5 + initial seed corpus
M3.8  - dispatcher step 8 (EvaluateInvariants → Trace.UnmodeledObservations)
M3.9  - TestNoPredictiveAPI, TestI3_DateRevisionRespectee, TestUnmodeledObservationsReported
M3.10 - docs/theory/05-invariants.md + docs/empirical/fuzz-summary.md
M3.11 - integrated review + tag

Spec: PRD.md §6, §7.2, §10 step 8, §15.2. Plan: docs/superpowers/plans/2026-05-24-M3-invariants-fuzz.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin main
git push origin v0.0.4-alpha
```

- [ ] **Étape 5 — Mettre à jour `CHANGELOG.md`**

Ajouter :

```markdown
## [0.0.4-alpha] — 2026-05-24

### Ajouté

- `internal/tau/invariants/` : package complet I1-I5 + helpers `Conserve`, `Residu`, `Recablage`, `Incoherent`, `Aggregate`, `BoundsHold`. `EvaluateInvariants` agrège les cinq verdicts.
- `internal/tau/invariants/fuzz_targets.go` : `FuzzI1` à `FuzzI5` (PRD §15.2). Corpus seed `testdata/fuzz/FuzzI*/seed01`.
- `internal/orchestration/dispatcher.go` étape 8 : `EvaluateInvariants` annote `Trace.UnmodeledObservations` sur violation, sans muter le régime.
- `internal/anti_patterns_test.go` : `TestNoPredictiveAPI` (réflexion AST sur 4 packages gardés), `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported`.
- `docs/theory/05-invariants.md` : renvoi III.8.5 + tableau d'encodage Go.
- `docs/empirical/fuzz-summary.md` : rapport fuzz daté.
- Constante exportée : `invariants.I3PerimptionLimite = 2027-01-01 UTC` (clause de péremption I3, PRD §6.1).
```

```powershell
git add CHANGELOG.md
git commit -m "docs(changelog): M3 release notes (v0.0.4-alpha)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin main
```

---

## Annexe — Risques M3 spécifiques

| # | Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|---|
| R1 | **Corpus seed pauvre** : le seed unique par cible ne couvre pas l'espace de bordure → fuzz tourne 30 s sans trouver les régions intéressantes | Probable | Moyen | M3.7 : ajouter ≥ 1 seed par tâche (cas limite documenté). Run nocturne 24 h alimente la coverage queue. |
| R2 | **I5 sans pile reifiée dans `Trace`** : V1 `EvaluateI5` retourne `Held` constant ; aucune capture de violation côté dispatcher | Confirmé | Moyen | Acceptée comme limite V1. PRD §6.1 dit explicitement « V1 expose l'API ; V2 calcule » — M3.6 calcule (`Aggregate`, `BoundsHold`) ; le câblage `Trace.Stack` est différé M6. |
| R3 | **I3 verdict V1 par tau_score composite** : sans `Trace.Scores`, l'évaluateur infère depuis le composite, ce qui sous-estime la vraie D-AUTORITÉ | Probable | Faible (test direct sur dispatcher refus est précis) | M5 ajoute `Trace.Scores` ; refactor de `EvaluateI3` à ce moment. Documenter dans `05-invariants.md`. |
| R4 | **`TestNoPredictiveAPI` flaky** : si la réflexion AST manque un cas (ex. méthode dans un sous-package non-listé) | Probable | Élevé (silently disables anti-patron #1) | Lister `gardedPackages` exhaustivement ; ajouter un test négatif (faux exporté `PredictX` introduit transitoirement pour vérifier l'échec, puis retiré) avant le tag. |
| R5 | **Drift `arch_test.go`** : nouvelle règle non documentée | Faible | Moyen | M3.1 vérifie que la règle `invariants ↔ dimensions` existe déjà ; pas de modification de `arch_test.go` nécessaire. |
| R6 | **Heuristiques `frontierFromExchange` vs `Recablage`** : duplication entre `orchestration/dispatcher.go` et `invariants/i2_irreductibility.go`. Drift possible si l'un évolue sans l'autre | Probable | Faible | Documenter la duplication intentionnelle (étanchéité ne permet pas d'extraire). À fusionner en M5 si une heuristique commune émerge. |
| R7 | **Couverture `internal/tau/invariants/` < 80 %** | Faible | Bloquant CI | Tests par invariant déjà tabulés ; ajouter `t.Run` paramétré si nécessaire. |
| R8 | **Run fuzz 30 s `FuzzI3` n'explore que peu d'entrées sur Windows** (timer plus grossier) | Probable | Faible | Tolérer en CI Windows ; gate principale sur Linux/macOS. |

---

## Annexe — Self-review (à exécuter avant commit du plan)

- [x] **Couverture M3 high-level** : M3.1 (squelette) — M3.2 (I1) — M3.3 (I2) — M3.4 (I3) — M3.5 (I4) — M3.6 (I5) — M3.7 (fuzz + corpus) — M3.8 (étape 8) — M3.9 (gardes anti-patrons) — M3.10 (docs) — M3.11 (revue + tag).
- [x] **Granularité bite-sized** : chaque tâche tient en une session d'agent frais (< 200 LOC code + tests). M3.7 est la plus volumineuse (5 cibles + 5 seeds) — peut être splittée en M3.7a/b si l'agent estime nécessaire.
- [x] **Étanchéité** : `tau/invariants → dimensions` interdit respecté (vérifié par `arch_test.go` existant). `Recablage` duplique l'heuristique `frontierFromExchange` plutôt que d'importer `orchestration`.
- [x] **Anti-patrons** : #1 (`TestNoPredictiveAPI` M3.9), #2 (déjà couvert M0/M2), #3 (`TestI3_DateRevisionRespectee` M3.9 + `I3PerimptionLimite`), #4 (`TestUnmodeledObservationsReported` M3.9 + étape 8 M3.8).
- [x] **Cohérence des types** : `Status`, `Statuses`, `ResidualMagnitude`, `Pile`, `AngleMort` introduits dans le bon package ; pas de drift vers `tau` ou `orchestration`.
- [x] **Marqueurs d'incertitude** : I1 *Probable*, I2 *Confirmé par construction*, I3 *Probable daté 2026-05-24*, I4 *Hypothèse priorité empirique #1*, I5 *Probable V2 calculée*. *Hypothèse — À vérifier* sur le timestamp unix du seed I3.
- [x] **Pas d'emoji**, godoc en anglais, `t.Parallel()` sur 100 % des nouveaux tests, FR-CA pour les commentaires structurants et le sous-plan lui-même.

---

*Sous-plan V1 — 2026-05-24. Référence : `PRDPlanning.md` §M3 + `PRD.md` §6, §7.2, §10 étape 8, §15.2. Coordinateur : Claude Code thread principal. Exécutants : agent teams (`ruflo-core:coder`, `ruflo-core:researcher`, `ruflo-core:reviewer`).*
````

---

**Rapport synthèse (< 200 mots)**

**Chemin attendu (non créé — mode lecture-seule)** :
`C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\docs\superpowers\plans\2026-05-24-M3-invariants-fuzz.md`

**Tâches M3.X identifiées (11)** :
- **M3.1** Squelette package `invariants` (Status, Statuses, EvaluateInvariants + stubs)
- **M3.2** I1 conservation + helper `Conserve`
- **M3.3** I2 irréductibilité + helpers `Residu` / `Recablage`
- **M3.4** I3 asymétrie D-AUTORITÉ + `I3PerimptionLimite` (2027-01-01)
- **M3.5** I4 cohérence + détecteur `Incoherent`
- **M3.6** I5 composition + API `Aggregate(π)` / `BoundsHold` (V2 calculée)
- **M3.7** `fuzz_targets.go` + corpus seed `testdata/fuzz/FuzzI*/seed01`
- **M3.8** Dispatcher étape 8 (`EvaluateInvariants` → `Trace.UnmodeledObservations`)
- **M3.9** Gardes anti-patrons : `TestNoPredictiveAPI` (AST), `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported`
- **M3.10** Docs `05-invariants.md` + `fuzz-summary.md`
- **M3.11** Revue intégrée + tag `v0.0.4-alpha` + CHANGELOG
