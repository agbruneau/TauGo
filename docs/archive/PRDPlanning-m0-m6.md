# PRDPlanning — TauGo (Plan d'implémentation V1)

> **Pour les agents exécutants** : ce plan est exécuté par **agent teams** *(cf. [`CLAUDE.md` §Agent Teams](CLAUDE.md))*. Chaque tâche est assignable à un sous-agent identifié dans la colonne « Agent ». Le coordinateur (thread principal) **dispatche, n'implémente pas**. Les étapes utilisent la syntaxe checkbox `- [ ]` pour le tracking — `superpowers:subagent-driven-development` ou `superpowers:executing-plans` pour la mécanique.

**Objectif** : livrer TauGo V1 (chap. III.8 → kernel Go validé empiriquement contre `AgentMeshKafka`) en 7 milestones M0-M6, conformément au [`PRD.md` §16](PRD.md).

**Architecture du programme** : pipeline strict M0 → M6 ; documentation théorique et lint éditorial parallélisables ; chaque milestone produit un livrable testable de bout en bout *(critère d'acceptation falsifiable)*.

**Tech stack** : Go 1.25+ · golangci-lint v1.64.8 · `gopter` (property-based) · `go test -fuzz` (I1-I5) · GitHub Actions (3 OS matrix) · Makefile · pas de framework (cf. PRD §3.3, §13).

**Référence canonique** : `agbruneau/InteroperabiliteAgentique` v2.4.3, chap. III.8.

---

## A. Orchestration agent teams

### A.1 Cartographie agent → rôle

| Agent | Rôle dans TauGo | Quand l'invoquer |
|---|---|---|
| `Plan` | Architecte logiciel | Avant chaque milestone : raffiner le sous-plan détaillé (M1+) ; toute décision d'architecture non triviale |
| `ruflo-swarm:architect` | Architecte système | Design interfaces et contrats inter-couches ; ADR avant changement structurel |
| `ruflo-swarm:coordinator` | Coordinateur swarm | Quand ≥ 3 agents tournent en parallèle sur tâches indépendantes |
| `Explore` | Recherche read-only | Localiser patterns FibGo à calquer ; rechercher symboles dans la monographie |
| `ruflo-core:researcher` | Pathfinder | Vérifier alignement théorie ↔ code (chap. III.8 ↔ Go) ; trouver le verbatim d'un invariant |
| `ruflo-core:coder` | Implémentation | Écriture TDD du code Go conforme aux conventions PRD §14 |
| `ruflo-core:reviewer` | Revue de code | Gate avant merge — vérifie invariants, anti-patrons, étanchéité Clean Arch |
| `understand-anything:project-scanner` | Inventaire | Avant M6 : scanner du repo pour rapport d'audit final |
| `understand-anything:architecture-analyzer` | Analyse architecture | Vérifier que les couches livrées correspondent à PRD §8 |
| `general-purpose` | Tâches multi-étapes ouvertes | Recherche comparative inter-projets (FibGo vs FibRust patterns, etc.) |

### A.2 Pattern d'exécution par milestone

```
1. RECHERCHE (parallèle)
   ├─ Explore         → patterns FibGo (Claude.md, arch_test.go, calibration/)
   └─ ruflo-core:researcher → alignement chap. III.8 ↔ tâches Go

2. ARCHITECTURE
   └─ Plan ou ruflo-swarm:architect → sous-plan détaillé du milestone

3. IMPLÉMENTATION (parallèle si tâches indépendantes)
   └─ ruflo-core:coder × N → code TDD conforme

4. REVUE
   └─ ruflo-core:reviewer → gate invariants + anti-patrons + étanchéité

5. INTÉGRATION (thread principal)
   → tests CI verts → tag → commit conventionnel signé
```

### A.3 Coordinateur — règles d'orchestration

- **Le thread principal ne code pas**. Il dispatche, intègre, valide.
- **Parallélisme par défaut** quand les tâches sont indépendantes (research vs implementation, par ex.).
- **Sérialisation imposée** pour : commits, tags, intégration finale, décisions ADR.
- **Briefing complet** à chaque agent : pas de référence implicite à la conversation principale ; chaque dispatch contient un contexte autoportant.
- **Vérification systématique** : après chaque agent, le coordinateur lit le diff produit avant de relancer la suite. Ne pas faire confiance aveuglement à un rapport d'agent.

---

## B. Dépendances entre milestones

```
M0  squelette + CI
 │
 ├──> M1  dispatcher 2 régimes + stub LLM            (besoin de cmd/tau + tau/operator skeleton)
 │       │
 │       └──> M2  3 dimensions + gardes ontol. + I4 (besoin du dispatcher)
 │                │
 │                └──> M3  fuzz I1-I5                (besoin des dimensions + gardes)
 │                        │
 │                        └──> M4  AgentMeshKafka + campagne empirique I4
 │                                │
 │                                └──> M5  calibration adaptative + drift
 │                                        │
 │                                        └──> M6  docs + typographie + release v0.1.0
 │
 └──> [parallèle] docs/theory/{03..07}.md  rédigés au fil ; renvois chap. III.8
```

**Parallélismes inter-milestone exploitables** :

| Parallélisme | Justification |
|---|---|
| M0 squelette + `docs/theory/03-operateur-tau.md` | Pas de dépendance code ; agent `ruflo-core:researcher` écrit les renvois pendant que `ruflo-core:coder` pose les fichiers |
| M3 fuzz I1-I5 + début préparation mock AgentMeshKafka | Mock peut être stabilisé pendant que fuzz court tourne |
| M5 calibration + ébauche M6 `README.md`, `CHANGELOG.md` | Documentation finale rédigeable sur état figé après M5 |

---

# Milestone M0 — Squelette + CI (détail bite-sized)

**Objectif** : `git init` opérationnel, premier commit signé vert, tag `v0.0.1-alpha`, `TestRefusHorsFrontiere` passe. Aucun comportement métier ; structure + gardes architecturales seulement.

**Critère d'acceptation global** :
```bash
go build ./... && go test -race -short ./... && golangci-lint run ./...
```
…passe vert sur 3 OS. Tag `v0.0.1-alpha` créé.

### Tâche M0.1 — Bootstrap module Go

**Files :**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `LICENSE` *(Apache-2.0 verbatim)*

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Initialiser go.mod**

```bash
go mod init github.com/agbruneau/taugo
```

Vérifier la version `go 1.25` dans le fichier généré.

- [ ] **Étape 2 — Créer .gitignore**

```gitignore
# binaries
/bin/
/dist/
tau
tau.exe

# go
*.test
*.out
coverage.html
coverage.txt

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# temp build artifacts
*.prof
*.pgo
default.pgo

# config local
.env
.env.local
```

- [ ] **Étape 3 — Copier LICENSE Apache-2.0**

Récupérer le texte verbatim Apache-2.0 (calque FibGo `LICENSE`).

- [ ] **Étape 4 — Commit M0.1**

```bash
git add go.mod .gitignore LICENSE
git commit -m "chore(bootstrap): initialize Go module and license

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.2 — golangci-lint config (calque FibGo)

**Files :**
- Create: `.golangci.yml`

**Agent :** `Explore` (récupérer `.golangci.yml` FibGo) → `ruflo-core:coder` (adapter)

- [ ] **Étape 1 — Récupérer le `.golangci.yml` FibGo de référence**

Briefing pour `Explore` : « Récupère le fichier `.golangci.yml` de `agbruneau/FibGo` (branche `main`) verbatim. Rapporte le contenu intégral. »

- [ ] **Étape 2 — Adapter pour TauGo**

Conserver les 24 linters, complexité max 15/30, longueur 100 LOC / 50 statements. Ajuster les exclusions de path (`cmd/generate-golden/` à exclure des stricts ; `test/golden/` idem).

- [ ] **Étape 3 — Vérifier**

```bash
golangci-lint run ./...
```

Doit passer sans warning *(repo presque vide, donc trivialement vert)*.

- [ ] **Étape 4 — Commit M0.2**

```bash
git add .golangci.yml
git commit -m "chore(lint): add golangci-lint config (24 linters, FibGo calque)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.3 — Makefile

**Files :**
- Create: `Makefile`

**Agent :** `ruflo-core:coder` (calque FibGo Makefile)

- [ ] **Étape 1 — Rédiger Makefile complet**

```makefile
.PHONY: all build test test-short coverage benchmark lint fuzz fuzz-long \
        calibrate build-reproducible build-pgo build-all clean

GO ?= go
BIN := tau
PKG := ./cmd/tau

all: lint test build

build:
	$(GO) build -trimpath -buildvcs=true -o $(BIN) $(PKG)

build-reproducible:
	$(GO) build -trimpath -buildvcs=true \
		-ldflags="-buildid= -X main.buildTimestamp=1778889600" \
		-o $(BIN) $(PKG)

build-pgo:
	$(GO) build -trimpath -pgo=default.pgo -o $(BIN) $(PKG)

build-all:
	GOOS=linux   GOARCH=amd64 $(GO) build -trimpath -o dist/tau-linux-amd64   $(PKG)
	GOOS=linux   GOARCH=arm64 $(GO) build -trimpath -o dist/tau-linux-arm64   $(PKG)
	GOOS=darwin  GOARCH=amd64 $(GO) build -trimpath -o dist/tau-darwin-amd64  $(PKG)
	GOOS=darwin  GOARCH=arm64 $(GO) build -trimpath -o dist/tau-darwin-arm64  $(PKG)
	GOOS=windows GOARCH=amd64 $(GO) build -trimpath -o dist/tau-windows-amd64.exe $(PKG)

test:
	$(GO) test -v -race -cover ./...

test-short:
	$(GO) test -v -short ./...

coverage:
	$(GO) test -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html

benchmark:
	$(GO) test -bench=. -benchmem -run=^$$ ./internal/tau/...

lint:
	golangci-lint run ./...

fuzz:
	$(GO) test -fuzz=. -fuzztime=30s ./internal/tau/invariants/

fuzz-long:
	$(GO) test -fuzz=. -fuzztime=24h ./internal/tau/invariants/

calibrate:
	$(GO) run $(PKG) calibrate $(ARGS)

clean:
	rm -f $(BIN) coverage.txt coverage.html
	rm -rf dist/
```

- [ ] **Étape 2 — Vérifier `make all` ne plante pas**

À ce stade, `make all` doit échouer proprement (pas de code à builder). Acceptable jusqu'à M0.5.

- [ ] **Étape 3 — Commit M0.3**

```bash
git add Makefile
git commit -m "chore(build): add Makefile (calque FibGo)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.4 — Squelette de packages internal/

**Files :**
- Create: `internal/tau/doc.go`
- Create: `internal/orchestration/doc.go`
- Create: `internal/calibration/doc.go`
- Create: `internal/bridge/llm/doc.go`
- Create: `internal/bridge/agentmeshkafka/doc.go`
- Create: `internal/app/doc.go`
- Create: `internal/config/doc.go`
- Create: `internal/errors/doc.go`
- Create: `internal/metrics/doc.go`
- Create: `internal/testutil/doc.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Écrire `internal/tau/doc.go`**

```go
// Package tau implements the operator τ defined in chap. III.8 of
// `InteroperabiliteAgentique/Monographie.md` v2.4.3.
//
// It is the core of TauGo: it decides the call regime (Deterministe,
// Probabiliste, or Refus) at the agentic interoperability boundary,
// under the five invariants I1-I5.
//
// τ never predicts behavior; it never executes the call. The exclusive
// public entry point is Kernel.Decide.
package tau
```

- [ ] **Étape 2 — Écrire les autres `doc.go`** *(une ligne descriptive par package)*

Exemple `internal/orchestration/doc.go` :
```go
// Package orchestration dispatches decisions across the deterministic
// and probabilistic regimes. It owns the Decision and Trace types.
package orchestration
```

Idem pour `calibration`, `bridge/llm`, `bridge/agentmeshkafka`, `app`, `config`, `errors`, `metrics`, `testutil`. Chaque `doc.go` < 5 lignes.

- [ ] **Étape 3 — Vérifier compilation**

```bash
go build ./...
```

Attendu : pas d'erreur, pas de package buildé *(seulement des `doc.go`)*.

- [ ] **Étape 4 — Commit M0.4**

```bash
git add internal/
git commit -m "feat(scaffold): create internal/ package skeleton with doc.go

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.5 — FrontierCheck + test

**Files :**
- Create: `internal/tau/frontier.go`
- Create: `internal/tau/frontier_test.go`

**Agent :** `ruflo-core:coder` (TDD : test rouge d'abord)

- [ ] **Étape 1 — Écrire le test rouge**

```go
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.f.Inside() {
				t.Fatalf("expected Inside()=false when %s (one classical condition still holds)", tc.name)
			}
		})
	}
}
```

- [ ] **Étape 2 — Vérifier que le test échoue à la compilation**

```bash
go test ./internal/tau/...
```

Attendu : `undefined: FrontierCheck`.

- [ ] **Étape 3 — Implémenter FrontierCheck**

```go
// internal/tau/frontier.go
package tau

// FrontierCheck encodes the four classical conditions whose simultaneous
// violation defines the agentic boundary where τ applies (chap. III.8.3.2).
type FrontierCheck struct {
	UniversOuvert       bool // capabilities discovered at runtime
	CompositionVariable bool // composition resolved at runtime
	PairProbabiliste    bool // peer is a probabilistic reasoner (LLM or equivalent)
	CoutNonBorne        bool // error cost unbounded and/or irreversible
}

// Inside reports whether the exchange falls within τ's domain of validity.
// τ applies if and only if all four classical conditions are simultaneously
// violated; one condition still holding rules out τ application.
func (f FrontierCheck) Inside() bool {
	return f.UniversOuvert && f.CompositionVariable &&
		f.PairProbabiliste && f.CoutNonBorne
}
```

- [ ] **Étape 4 — Vérifier que tous les tests passent**

```bash
go test -race ./internal/tau/...
```

Attendu : `ok internal/tau 0.123s`.

- [ ] **Étape 5 — Commit M0.5**

```bash
git add internal/tau/frontier.go internal/tau/frontier_test.go
git commit -m "feat(tau): add FrontierCheck encoding the 4-conditions boundary

Encodes chap. III.8.3.2: τ applies if and only if all four classical
conditions (univers ouvert, composition variable, pair probabiliste,
coût non borné) are simultaneously violated. Single condition holding
rules out τ application — guards anti-patron #2 (hors frontière).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.6 — Squelette `internal/tau/operator.go` (panic not implemented)

**Files :**
- Create: `internal/tau/operator.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Écrire les types stubs et la signature Decide**

```go
// internal/tau/operator.go
package tau

import (
	"context"
	"time"
)

// Regime is the discrete output of τ. Never a behavior, never a result.
type Regime int

const (
	RegimeUnknown Regime = iota
	Deterministe
	Probabiliste
	Refus
)

// Exchange is the interoperability exchange submitted to τ.
type Exchange struct {
	ID                          string
	IntentDescription           string
	DiscoveredAt                time.Time
	AttestationInstitutionnelle *Attestation
	Context                     map[string]any
	// Principal and Capability fields intentionally omitted in M0;
	// added in M2 alongside the dimensions.
}

// Attestation is the opposable reference that populates the "execution"
// pole of D-AUTORITÉ (chap. III.8.4.2.bis, Searle 1995).
type Attestation struct {
	Emetteur   string
	Reference  string
	Marqueur   string
	AssertedAt time.Time
}

// Decision is the full output of Kernel.Decide. Always traced.
type Decision struct {
	Regime         Regime
	Diagnostic     string // non-empty iff Regime == Refus
	ProfileVersion string
	DateRevision   time.Time
	// Trace field intentionally omitted in M0; added in M1.
}

// Kernel is the public face of the τ operator. Single entry point: Decide.
type Kernel interface {
	Decide(ctx context.Context, x Exchange) (Decision, error)
}
```

- [ ] **Étape 2 — Vérifier compilation**

```bash
go build ./internal/tau/
```

Attendu : pas d'erreur.

- [ ] **Étape 3 — Commit M0.6**

```bash
git add internal/tau/operator.go
git commit -m "feat(tau): add operator skeleton (Kernel.Decide signature)

M0 stub: types Regime, Exchange, Attestation, Decision, and the Kernel
interface. Trace + Principal + Capability deferred to M1/M2.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.7 — Architecture test (étanchéité des 4 couches)

**Files :**
- Create: `internal/arch_test.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Écrire arch_test.go avec règles d'étanchéité**

```go
// internal/arch_test.go
package internal_test

import (
	"go/build"
	"strings"
	"testing"
)

type rule struct {
	from string
	deny []string
}

var archRules = []rule{
	{from: "github.com/agbruneau/taugo/internal/tau", deny: []string{
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/bridge",
		"github.com/agbruneau/taugo/internal/app",
	}},
	{from: "github.com/agbruneau/taugo/internal/tau/dimensions", deny: []string{
		"github.com/agbruneau/taugo/internal/tau/invariants",
	}},
	{from: "github.com/agbruneau/taugo/internal/tau/invariants", deny: []string{
		"github.com/agbruneau/taugo/internal/tau/dimensions",
	}},
	{from: "github.com/agbruneau/taugo/internal/bridge", deny: []string{
		"github.com/agbruneau/taugo/internal/tau",
	}},
}

func TestArchitectureLayering(t *testing.T) {
	t.Parallel()
	for _, r := range archRules {
		r := r
		t.Run(strings.ReplaceAll(r.from, "/", "_"), func(t *testing.T) {
			t.Parallel()
			pkg, err := build.Default.Import(r.from, ".", build.ImportComment)
			if err != nil {
				// package may not exist yet in M0; skip without failing
				t.Skipf("package %s not built yet: %v", r.from, err)
			}
			imports := append([]string{}, pkg.Imports...)
			imports = append(imports, pkg.TestImports...)
			for _, imp := range imports {
				for _, denied := range r.deny {
					if imp == denied || strings.HasPrefix(imp, denied+"/") {
						t.Errorf("forbidden import: %s imports %s", r.from, imp)
					}
				}
			}
		})
	}
}
```

- [ ] **Étape 2 — Vérifier que les règles passent (packages vides à ce stade)**

```bash
go test -v ./internal/...
```

Attendu : `TestArchitectureLayering` passe (avec `SKIP` pour les packages non encore peuplés).

- [ ] **Étape 3 — Commit M0.7**

```bash
git add internal/arch_test.go
git commit -m "test(arch): add layer-tightness rules for Clean Architecture

Encodes PRD §8.1: tau/* cannot import orchestration/bridge/app;
dimensions ↔ invariants forbidden (orthogonality); bridge cannot
import tau directly. Skipped for packages not yet populated.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.8 — `cmd/tau/main.go` squelette CLI

**Files :**
- Create: `cmd/tau/main.go`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Écrire main.go minimal avec `--help`**

```go
// Command tau is the TauGo CLI. M0: --help only.
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	buildTimestamp = "dev" // set by `make build-reproducible`
	version        = "0.0.1-alpha"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "--version" {
		fmt.Printf("tau %s (build %s)\n", version, buildTimestamp)
		os.Exit(0)
	}
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `tau — TauGo kernel CLI (V0.1)

USAGE:
    tau <command> [flags]

COMMANDS:
    decide      Decide a regime for one exchange (M1+)
    calibrate   Run adaptive calibration on a corpus (M5+)
    --version   Print version

Specification: PRD.md
`)
	}
	flag.Parse()
	flag.Usage()
}
```

- [ ] **Étape 2 — Vérifier build et `--help`**

```bash
go build -o tau ./cmd/tau
./tau --help
./tau --version
```

Attendus :
- `--help` → texte d'aide
- `--version` → `tau 0.0.1-alpha (build dev)`

- [ ] **Étape 3 — Commit M0.8**

```bash
git add cmd/tau/main.go
git commit -m "feat(cli): add CLI skeleton with --help and --version

M0 stub: no command implemented yet. \"decide\" and \"calibrate\"
will land in M1 and M5 respectively.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.9 — CI GitHub Actions (build + test + lint)

**Files :**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/coverage.yml`

**Agent :** `Explore` (calque FibGo) → `ruflo-core:coder` (adapter)

- [ ] **Étape 1 — Récupérer les workflows FibGo de référence**

Briefing pour `Explore` : « Récupère `.github/workflows/ci.yml` et `.github/workflows/coverage.yml` de `agbruneau/FibGo` branche `main`. Rapporte verbatim. »

- [ ] **Étape 2 — Adapter `ci.yml` pour TauGo**

Calque FibGo : matrice `{ubuntu-latest, macos-latest, windows-latest}` × Go `1.25.x`. Jobs : `lint`, `test (race, short)`, `build`. Cache `~/.cache/go-build`. Race detector activé sur Linux/macOS (CGO disponible).

- [ ] **Étape 3 — Adapter `coverage.yml`**

Gate `MIN_COVERAGE=80` global. Upload à codecov (optionnel). Pour TauGo : ajouter étape « couverture par package » avec gate ≥ 90 % sur `tau/*` *(à activer dès M1 où le package a du code)*.

- [ ] **Étape 4 — Vérifier syntaxe YAML**

```bash
# si actionlint dispo localement :
actionlint .github/workflows/*.yml
```

Sinon : pousser une branche temporaire pour validation GitHub.

- [ ] **Étape 5 — Commit M0.9**

```bash
git add .github/workflows/
git commit -m "ci: add GitHub Actions (build/test/lint + coverage)

Matrix: ubuntu-latest, macos-latest, windows-latest × Go 1.25.x.
Race detector on Linux/macOS via CGO. Coverage gate 80% global.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.10 — README minimal + CHANGELOG

**Files :**
- Create: `README.md`
- Create: `CHANGELOG.md`

**Agent :** `ruflo-core:researcher` (rédaction alignée monographie + PRD)

- [ ] **Étape 1 — Rédiger `README.md` minimal**

```markdown
# TauGo

Kernel exécutable Go de l'opérateur τ et validateur empirique des invariants I1-I5 du modèle théorique défini au chapitre III.8 de la monographie *Interopérabilité Agentique en Écosystème d'Entreprise*.

> **État** : V0.1 — pré-implémentation. Le code n'existe pas encore. Spec dans [`PRD.md`](PRD.md). Conventions dans [`CLAUDE.md`](CLAUDE.md). Plan dans [`PRDPlanning.md`](PRDPlanning.md).

## Quick start

```bash
git clone https://github.com/agbruneau/taugo
cd taugo
make all
./tau --help
```

## Documentation

- [`PRD.md`](PRD.md) — spécification canonique V0.2
- [`CLAUDE.md`](CLAUDE.md) — conventions de rédaction et d'ingénierie
- [`PRDPlanning.md`](PRDPlanning.md) — plan d'exécution M0-M6
- [`docs/theory/`](docs/theory/) — renvois vers chap. III.8 de la monographie

## Licence

Apache-2.0. Voir [LICENSE](LICENSE).
```

- [ ] **Étape 2 — Rédiger `CHANGELOG.md` initial (format Keep-a-Changelog)**

```markdown
# Changelog

Conforme à [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et au [Versionnage Sémantique](https://semver.org/lang/fr/).

## [Non publié]

## [0.0.1-alpha] - 2026-05-XX

### Ajouté
- Squelette du module Go (`go.mod`, Apache-2.0)
- Makefile avec cibles essentielles (`all`, `test`, `lint`, `fuzz`, `build-reproducible`, etc.)
- `.golangci.yml` (24 linters, calque FibGo)
- Squelette `internal/` (packages avec `doc.go`)
- `FrontierCheck` encodant la frontière de validité de τ (chap. III.8.3.2) + tests
- `Kernel.Decide` signature, types `Regime`, `Exchange`, `Attestation`, `Decision`
- `internal/arch_test.go` — étanchéité des 4 couches Clean Architecture
- `cmd/tau/main.go` — CLI minimal (`--help`, `--version`)
- CI GitHub Actions (3 OS matrix, race detector, lint, coverage 80% gate)

### Spec
- `PRD.md` V0.2 (refactorisé)
- `CLAUDE.md` V0.2 (refactorisé)
- `PRDPlanning.md` initial
```

- [ ] **Étape 3 — Commit M0.10**

```bash
git add README.md CHANGELOG.md
git commit -m "docs: add README and CHANGELOG (Keep-a-Changelog format)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.11 — `docs/theory/03-operateur-tau.md` (en parallèle de M0.5-M0.6)

**Files :**
- Create: `docs/theory/03-operateur-tau.md`

**Agent :** `ruflo-core:researcher` (alignement chap. III.8.3)

**Parallélisable** avec M0.5-M0.6.

- [ ] **Étape 1 — Rédiger les renvois croisés monographie ↔ TauGo**

```markdown
# 03 — L'opérateur τ — renvoi vers chap. III.8.3

*Document de renvoi croisé. Le verbatim canonique vit dans `InteroperabiliteAgentique/Monographie.md` v2.4.3.*

## Définition (chap. III.8.3.1)

τ déplace l'instant de fixation des grandeurs d'interopérabilité (sens, autorité, support d'invariant) de l'avant-interaction vers l'interaction :

`τ : t_fix(g) ≺ t_int ↦ t_fix(g) ≈ t_int`

## Encodage TauGo

| Concept monographie | Encodage Go | Renvoi PRD |
|---|---|---|
| Grandeur `g` | Trois dimensions : D-SENS, D-AUTORITÉ, D-INVARIANT (`internal/tau/dimensions/`) | §5 |
| Instant `t_fix` | Position sur l'axe `[0, 1]` de chaque dimension | §2.2 |
| Frontière de validité | `FrontierCheck.Inside()` (`internal/tau/frontier.go`) | §4.3 |
| Asymétrie ontologique | `Attestation` requise pour D-AUTORITÉ ≥ θ_auth_block | §4.4 |

## Propriétés exploitables (chap. III.8.3.1)

1. τ opère sur `t_fix`, jamais sur le contenu de `g` → base I1 (conservation)
2. τ non trivial seulement si `t_fix(g) ≺ t_int` peut être violé → base I2 (irréductibilité)
3. Application de τ à une grandeur n'entraîne pas son application à une autre → base orthogonalité

## Statut

*Probable. Daté 2026-05-23. Aligné monographie v2.4.3.*
```

- [ ] **Étape 2 — Commit M0.11**

```bash
git add docs/theory/03-operateur-tau.md
git commit -m "docs(theory): add III.8.3 cross-reference for operator τ

Encodes mapping monographie ↔ TauGo for the τ operator definition.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Tâche M0.12 — Validation finale M0 et tag

**Agent :** thread principal (intégration), `ruflo-core:reviewer` (revue)

- [ ] **Étape 1 — Lancer la suite complète localement**

```bash
make all
```

Vérifier : build vert, tests passent, lint sans warning, `TestRefusHorsFrontiere` (renommé `TestFrontierCheck_*`) passe, `TestArchitectureLayering` passe.

- [ ] **Étape 2 — Briefing reviewer**

Briefing pour `ruflo-core:reviewer` : « Revue intégrée du commit range `M0.1..HEAD`. Vérifier : (1) absence d'anti-patron (pas de `Predict*` exporté, pas d'import LLM concret dans `tau/`), (2) `arch_test.go` couvre les 4 règles d'étanchéité PRD §8.1, (3) FrontierCheck correspond verbatim au chap. III.8.3.2 (4 conditions, conjonction stricte), (4) conventions de code calquent FibGo (interfaces étroites, `t.Parallel()`, pas d'emoji). Rapport bref. »

- [ ] **Étape 3 — Tag `v0.0.1-alpha`**

```bash
git tag -a v0.0.1-alpha -m "M0: skeleton + CI complete

- Go module, Apache-2.0, .gitignore
- Makefile, .golangci.yml (24 linters)
- internal/ skeleton with doc.go
- FrontierCheck (chap. III.8.3.2 encoded)
- Kernel.Decide signature + Exchange/Decision types
- arch_test.go (4 layer-tightness rules)
- cmd/tau CLI skeleton
- CI 3-OS matrix + coverage 80% gate
- README, CHANGELOG, docs/theory/03

Spec: PRD.md V0.2. Plan: PRDPlanning.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin v0.0.1-alpha
```

- [ ] **Étape 4 — Vérifier CI verte sur le tag**

Ouvrir GitHub Actions, confirmer les jobs `ci.yml` et `coverage.yml` verts sur 3 OS.

**Sortie M0** : repo initialisé, CI verte, premier test métier vert, étanchéité gardée, première trace théorie ↔ code. M1 peut démarrer.

---

# Milestone M1 — Dispatcher minimal + stub LLM (résumé)

**Objectif** : `tau decide --input fixture.json` rend une `Decision` instrumentée avec `Regime ∈ {Deterministe, Probabiliste}`. Pas encore de dimensions calculables (M2) ; le régime est tiré d'un seuil naïf sur un score factice.

**Critère d'acceptation** :
```bash
echo '{"id":"test-1","intent_description":"trivial echo"}' | ./tau decide
# → JSON Decision avec Regime, Trace, ProfileVersion
```

**Tâches de haut niveau** *(à détailler au démarrage de M1 par `Plan`)* :

| # | Tâche | Agent |
|---|---|---|
| M1.1 | Étendre `Decision` avec `Trace` immuable | `ruflo-core:coder` |
| M1.2 | Implémenter `internal/orchestration/dispatcher.go` (étapes 1, 6, 7 du pseudo-algo PRD §10) | `ruflo-core:coder` |
| M1.3 | Stub LLM déterministe (`internal/bridge/llm/stub.go`) avec mapping intent→score checked-in | `ruflo-core:coder` |
| M1.4 | Injection LLM en `internal/app/` (config + factory) | `ruflo-core:coder` |
| M1.5 | Commande `tau decide` avec parsing JSON stdin et sortie JSON | `ruflo-core:coder` |
| M1.6 | Test E2E `cmd/tau` (TestEndToEnd_DecideDeterministe / TestEndToEnd_DecideProbabiliste) | `ruflo-core:coder` |
| M1.7 | `TestDefaultLLMIsStub` (anti-patron : refuser appel LLM externe en test) | `ruflo-core:coder` |
| M1.8 | `TestDecisionAlwaysTraced`, `TestRefusImpliesDiagnostic`, `TestTraceImmutable` | `ruflo-core:coder` |
| M1.9 | Revue + tag `v0.0.2-alpha` | thread principal + `ruflo-core:reviewer` |

**Dépendances** : M0 complet. **Recherche préalable** : `Explore` localise `bigfft/pool.go` FibGo pour le pattern d'interface étroite ; `ruflo-core:researcher` confirme l'alignement PRD §10 / §12.2.

---

# Milestone M2 — Trois dimensions + gardes ontologique D-AUTORITÉ et I4 (résumé)

**Objectif** : les sondes D-SENS, D-AUTORITÉ, D-INVARIANT calculent un score `[0, 1]`. La garde ontologique §4.4 et la garde I4 §6.1 sont actives. Le pseudo-algorithme PRD §10 est complet (étapes 1-7).

**Critère d'acceptation** :
```bash
go test -race ./... && \
  go test -run TestRefusOntologiqueDAUTORITE ./internal/tau/ && \
  go test -run TestI4_IncoherenceDetectee ./internal/tau/
```
…vert. Rapport `docs/empirical/M2-sample-decisions.md` avec ≥ 10 décisions tracées et leurs scores ventilés.

**Tâches de haut niveau** :

| # | Tâche | Agent |
|---|---|---|
| M2.1 | `Principal`, `Capability` types + extension `Exchange` | `ruflo-core:coder` |
| M2.2 | `internal/tau/dimensions/dsens.go` + 4 sondes + tests | `ruflo-core:coder` |
| M2.3 | `internal/tau/dimensions/dauthority.go` + 4 sondes + test ontologique | `ruflo-core:coder` |
| M2.4 | `internal/tau/dimensions/dinvariant.go` + 4 sondes + tests | `ruflo-core:coder` |
| M2.5 | Garde ontologique D-AUTORITÉ dans dispatcher (étape 2) + `TestRefusOntologiqueDAUTORITE` | `ruflo-core:coder` |
| M2.6 | Garde I4 dans dispatcher (étape 5) + `TestI4_IncoherenceDetectee` | `ruflo-core:coder` |
| M2.7 | `internal/calibration/profile.go` minimal + chargement par défaut | `ruflo-core:coder` |
| M2.8 | `internal/calibration/thresholds.go` avec pattern `atomic.Int64` (calque FibGo) | `ruflo-core:coder` |
| M2.9 | `docs/theory/04-dimensions.md` (renvoi III.8.4) | `ruflo-core:researcher` |
| M2.10 | Rapport `docs/empirical/M2-sample-decisions.md` | `ruflo-core:researcher` |
| M2.11 | Revue + tag `v0.0.3-alpha` | thread principal + `ruflo-core:reviewer` |

**Recherche préalable** : `Explore` extrait le pattern `atomic.Int64` du `bigfft/fft.go` FibGo ; `ruflo-core:researcher` valide les pondérations initiales contre PRD §5.

---

# Milestone M3 — Cinq invariants comme cibles fuzz (résumé)

**Objectif** : `go test -fuzz=. -fuzztime=30s ./internal/tau/invariants/` vert sur I1-I5.

**Critère d'acceptation** : aucune panique, aucun crash sur 30 s/cible × 5 cibles. Rapport `docs/empirical/fuzz-summary.md` avec : nombre d'entrées explorées, couverture de la corpus, exceptions épinglées, marqueur statut.

**Tâches de haut niveau** :

| # | Tâche | Agent |
|---|---|---|
| M3.1 | `internal/tau/invariants/i1_conservation.go` + helper `Conserve(x, τ(x))` | `ruflo-core:coder` |
| M3.2 | `internal/tau/invariants/i2_irreductibility.go` + helpers `Residu`, `Recablage` | `ruflo-core:coder` |
| M3.3 | `internal/tau/invariants/i3_authority_asymmetry.go` + clause péremption | `ruflo-core:coder` |
| M3.4 | `internal/tau/invariants/i4_coherence.go` + détecteur incohérence | `ruflo-core:coder` |
| M3.5 | `internal/tau/invariants/i5_composition.go` + API d'agrégation M(π) (calcul V2) | `ruflo-core:coder` |
| M3.6 | `internal/tau/invariants/fuzz_targets.go` — `FuzzI1` à `FuzzI5` | `ruflo-core:coder` |
| M3.7 | Corpus initial fuzz (`testdata/fuzz/FuzzI*/*`) | `ruflo-core:researcher` |
| M3.8 | Étape 8 du dispatcher : `EvaluateInvariants` sur la trace | `ruflo-core:coder` |
| M3.9 | `TestNoPredictiveAPI`, `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported` | `ruflo-core:coder` |
| M3.10 | `docs/theory/05-invariants.md` + `docs/empirical/fuzz-summary.md` | `ruflo-core:researcher` |
| M3.11 | Revue + tag `v0.0.4-alpha` | thread principal + `ruflo-core:reviewer` |

**Recherche préalable** : `Explore` étudie les cibles fuzz FibGo (`bigfft/FuzzMul`, `bigfft/FuzzSqr`, `fibonacci/FuzzFastDoublingConsistency`) pour le pattern.

---

# Milestone M4 — Adaptateur AgentMeshKafka + campagne empirique I4 (résumé)

**Objectif** : trace empirique end-to-end ; rapport I4 sur ≥ 100 traces analysées.

**Critère d'acceptation** :
```bash
go test -race -tags=integration ./test/e2e/agentmeshkafka_test.go
```
…vert (peut nécessiter Kafka local ou mock fidèle). Rapport `docs/empirical/I4-report.md` avec : nombre de cas incohérents détectés, faux positifs, faux négatifs, marqueur statut (Hypothèse → Probable visé).

**Tâches de haut niveau** :

| # | Tâche | Agent |
|---|---|---|
| M4.1 | `internal/bridge/agentmeshkafka/adapter.go` + interface `Adapter` | `ruflo-core:coder` |
| M4.2 | Mock fidèle (TestContainers ou Sarama mock) | `ruflo-core:coder` |
| M4.3 | E2E `test/e2e/agentmeshkafka_test.go` | `ruflo-core:coder` |
| M4.4 | Campagne empirique I4 : ingestion ≥ 100 traces réelles + classification | `ruflo-core:researcher` |
| M4.5 | Rapport `docs/empirical/I4-report.md` (Hypothèse → Probable si confirmé) | `ruflo-core:researcher` |
| M4.6 | Rapport `docs/empirical/unmodeled.md` initial | `ruflo-core:researcher` |
| M4.7 | Revue + tag `v0.0.5-alpha` | thread principal + `ruflo-core:reviewer` |

**Risque #1** (PRD §18) : `AgentMeshKafka` peut ne pas être prêt. Plan de contingence : continuer avec mock seul ; reporter campagne empirique à M4.bis ; M5 peut démarrer en parallèle sur le stub LLM uniquement.

---

# Milestone M5 — Calibration adaptative + drift (résumé)

**Objectif** : `tau calibrate` produit un profil reproductible byte-identique sur corpus fixé ; drift détecté invalide le profil.

**Critère d'acceptation** :
```bash
tau calibrate --corpus tests/calibration/golden-corpus.jsonl --output /tmp/p1.json
tau calibrate --corpus tests/calibration/golden-corpus.jsonl --output /tmp/p2.json
sha256sum /tmp/p1.json /tmp/p2.json
# → mêmes hashes
```
`TestCalibrationDeterministic` passe.

**Tâches de haut niveau** :

| # | Tâche | Agent |
|---|---|---|
| M5.1 | Algo de calibration des seuils (`internal/calibration/calibrate.go`) | `ruflo-core:coder` |
| M5.2 | Algo de calibration des poids (`internal/calibration/weights.go`) | `ruflo-core:coder` |
| M5.3 | `internal/calibration/drift.go` (5 critères PRD §11.4) | `ruflo-core:coder` |
| M5.4 | Persistance JSON versionnée (`internal/calibration/store.go`) + `current.json` symlink | `ruflo-core:coder` |
| M5.5 | Commande `tau calibrate` complète | `ruflo-core:coder` |
| M5.6 | `TestCalibrationDeterministic`, `TestExpiredProfileRefuses` | `ruflo-core:coder` |
| M5.7 | `docs/algorithms/calibration.md` + `docs/algorithms/drift.md` | `ruflo-core:researcher` |
| M5.8 | Revue + tag `v0.0.6-alpha` | thread principal + `ruflo-core:reviewer` |

---

# Milestone M6 — Docs + typographie + release v0.1.0 (résumé)

**Objectif** : release `v0.1.0`. Typographie française appliquée (U+00A0). Documentation complète. Tous critères de succès PRD §17 vérifiés.

**Critère d'acceptation** : checklist PRD §17 verte sur 10/10 items. Audit textuel final : aucun emoji, aucune fabrication, aucune citation non sourçée.

**Tâches de haut niveau** :

| # | Tâche | Agent |
|---|---|---|
| M6.1 | Typographie française dans `PRD.md`, `CLAUDE.md`, `PRDPlanning.md`, `docs/` | `ruflo-core:researcher` |
| M6.2 | `docs/theory/06-conditions-validite.md`, `docs/theory/07-anti-patrons.md` | `ruflo-core:researcher` |
| M6.3 | `docs/algorithms/dispatch.md` complet | `ruflo-core:researcher` |
| M6.4 | ADRs : `0001-clean-architecture-4-layers.md`, `0002-go-1.25-toolchain.md`, `0003-llm-client-injection.md`, `0004-agentmeshkafka-empirical-bridge.md` | `ruflo-core:researcher` |
| M6.5 | `README.md` final (badges CI, exemples d'usage, schéma) | `ruflo-core:researcher` |
| M6.6 | Cas BFSI anonymisé (`docs/empirical/case-study-bfsi.md`) | `ruflo-core:researcher` |
| M6.7 | Audit final : `understand-anything:project-scanner` + `understand-anything:architecture-analyzer` | agents `understand-anything:*` |
| M6.8 | Revue finale + tag `v0.1.0` | thread principal + `ruflo-core:reviewer` |

**Audit final** *(parallélisable)* :

```
3 agents en parallèle :
  ├─ understand-anything:project-scanner    → inventaire LOC, packages, dépendances
  ├─ understand-anything:architecture-analyzer → vérif couches conformes PRD §8
  └─ ruflo-core:reviewer                     → checklist PRD §17 + audit anti-patrons
```

---

# Annexes

## Annexe X.1 — Self-review du plan (à exécuter avant commit du plan)

- ☐ **Couverture spec** : chaque section du PRD §1-§20 a-t-elle une tâche correspondante ?
  - §1-§3 (cadrage) → couvert par M0 docs + M6 README
  - §4 (opérateur τ) → M0.5 (FrontierCheck) + M0.6 (Kernel.Decide stub) + M2.5 (ontologique)
  - §5 (dimensions) → M2.2, M2.3, M2.4
  - §6 (invariants) → M3.1-M3.6
  - §7 (conditions/anti-patrons) → M0.5, M2.5, M3.9, M4.6
  - §8 (architecture) → M0.7 (arch_test) + chaque tâche d'impl
  - §9 (modèle données) → M0.6 (stubs) + M1.1 (Trace) + M2.1 (Principal/Capability)
  - §10 (algorithme dispatch) → M1.2 + M2.5 + M2.6 + M3.8
  - §11 (calibration) → M2.7, M2.8, M5.1-M5.4
  - §12 (bridges) → M1.3, M1.4, M4.1
  - §13 (stack) → M0.1, M0.2, M0.3
  - §14 (conventions) → appliquées partout ; lint via golangci-lint
  - §15 (tests) → chaque tâche TDD + M3 (fuzz) + M4 (e2e)
  - §16 (roadmap) → ce document est la décomposition de §16
  - §17 (critères de succès) → checklist M6.8
  - §18 (risques) → noté en M4 contingence ; gardes CI partout
  - §19 (glossaire) → maintenu dans PRD ; pas de tâche dédiée
  - §20 (prochaines étapes) → M0 démarrage
- ☐ **Placeholder scan** : pas de TBD, TODO, « à compléter » résiduel dans les tâches M0 *(M1-M6 résumés volontairement haut niveau)*
- ☐ **Cohérence des types** : `Decision`, `Exchange`, `FrontierCheck`, `Kernel`, `Regime` cohérents entre M0.5, M0.6, M1.1, M1.2
- ☐ **Anti-patrons gardés** :
  - #1 prédictif → `TestNoPredictiveAPI` en M3.9
  - #2 hors frontière → `TestFrontierCheck_*` en M0.5
  - #3 atemporel → `TestI3_DateRevisionRespectee`, `TestExpiredProfileRefuses` en M3.9, M5.6
  - #4 clos → `TestUnmodeledObservationsReported` en M3.9

## Annexe X.2 — Modes d'exécution disponibles

À la clôture de chaque milestone, le coordinateur (thread principal) choisit :

1. **Subagent-driven** *(recommandé)* — `superpowers:subagent-driven-development` : un sous-agent frais par tâche, revue entre tâches, itération rapide. **Privilégier pour M0-M3.**
2. **Inline execution** — `superpowers:executing-plans` : exécution dans la session courante avec checkpoints. **À envisager pour M4-M6 si le contexte théorique nécessite continuité.**

## Annexe X.3 — Mise à jour de ce plan

Ce document est **vivant**. À l'ouverture de chaque milestone M1-M6, l'agent `Plan` produit un **sous-plan détaillé** par milestone, suivant la même granularité bite-sized que M0. Le sous-plan est commité dans `docs/superpowers/plans/YYYY-MM-DD-M{N}-<feature>.md`. Le présent `PRDPlanning.md` reste le **plan-cadre** et est mis à jour seulement pour : changement de séquence des milestones, changement de critère d'acceptation, changement d'agent assigné.

---

*Plan V0.1 — 2026-05-23. Référence : `PRD.md` V0.2. Coordinateur : Claude Code thread principal. Exécutants : agent teams.*
