# TauGo -- Kernel exécutable Go de l'opérateur τ

[![CI](https://github.com/agbruneau/taugo/actions/workflows/ci.yml/badge.svg)](https://github.com/agbruneau/taugo/actions/workflows/ci.yml)
[![Coverage](https://github.com/agbruneau/taugo/actions/workflows/coverage.yml/badge.svg)](https://github.com/agbruneau/taugo/actions/workflows/coverage.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/agbruneau/taugo.svg)](https://pkg.go.dev/github.com/agbruneau/taugo)
[![Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

---

## 1. Doctrine

TauGo implémente le **kernel exécutable de l'opérateur τ** défini au chapitre III.8 de la monographie
*Interopérabilité Agentique en Écosystème d'Entreprise* (`agbruneau/InteroperabiliteAgentique` v2.4.3,
*(chap. III.8)*).

L'API publique unique est :

```go
// Decide est l'unique point de décision public. Renvoie Deterministe,
// Probabiliste ou Refus -- jamais un comportement du pair appelé.
// La trace expose scores, invariants, seuils, profil de calibration.
func (k *Kernel) Decide(ctx context.Context, x Exchange) (Decision, error)
```

**Régimes de sortie** : `Deterministe | Probabiliste | Refus`.

**Opérateur τ** -- déplace l'instant de fixation `t_fix(g)`, jamais le contenu de la grandeur `g`.
τ décide *où* le sens, l'autorité et le support se fixent, donc *avec quoi* appeler.

**Trois dimensions** *(III.8.4)* :

| Dimension | Sémantique | Plage |
|---|---|---|
| D-SENS | Lieu de fixation du sens (avant / pendant) | [0,1] |
| D-AUTORITE | Portée de la chaîne de délégation | [0,1] |
| D-INVARIANT | Support des invariants d'intégration | [0,1] |

**Cinq invariants réfutables** *(III.8.5)* : I1 conservation, I2 irréductibilité, I3 asymétrie
d'autorité, I4 cohérence, I5 composition conjonctive. Marqueurs épistémiques :
voir [§7 Statut des invariants](#7-statut-des-invariants).

**Frontière de validité** -- τ ne s'applique qu'aux échanges satisfaisant simultanément les
quatre violations des conditions classiques. Hors frontière → `Refus` avec diagnostic.
Aucun fallback silencieux.

---

## 2. Anti-objectifs

TauGo **n'est pas** *(PRD §3.3)* :

- un framework agentique ou un orchestrateur d'agents
- un wrapper LLM ou un service réseau
- un système RAG ou de recherche sémantique
- un prédicteur de comportement des pairs

Toute PR qui érode ces frontières exige une mise à jour explicite du `PRD.md §3.3`.

---

## 3. Quick start

```bash
git clone https://github.com/agbruneau/taugo
cd taugo
make all                     # lint + test + build (Linux/macOS)
./tau --version
./tau --help
```

Sous Windows sans `make` :

```bash
go build -trimpath -o tau.exe ./cmd/tau
go test -short ./...
golangci-lint run ./...
```

Décision depuis stdin JSON :

```bash
echo '{"id":"e-1","intent_description":"creative generation","universe_open":true,"composition_variable":true,"peer_probabilistic":true,"cost_unbounded":true}' | ./tau decide
```

Calibration reproductible byte-identique (M5) :

```bash
make calibrate ARGS="--corpus tests/calibration/golden-corpus.jsonl \
                     --output current.json \
                     --seed 1 \
                     --date-revision 2026-12-01"
```

Remarque : `-race` nécessite CGO (gcc). Les runners Linux/macOS CI l'activent ; sous
Windows sans gcc local, utiliser `go test -short ./...`.

---

## 4. Architecture

Clean Architecture, quatre couches strictes, calque structurel de `agbruneau/FibGo`.
Etancheite gardée par `internal/arch_test.go`. Détail dans [docs/adr/0001-clean-architecture-4-layers.md](docs/adr/0001-clean-architecture-4-layers.md).

```
+-----------------------------------------------------+
|  cmd/tau                    (CLI -- decide, calibrate)|
+------------------------+----------------------------+
                         |
                         v
+-----------------------------------------------------+
|  internal/app           (lifecycle + injection LLM) |
+------------------------+----------------------------+
                         |
           +-------------+-------------+
           |                           |
           v                           v
+----------------------+  +-------------------------+
|  internal/           |  |  internal/              |
|  orchestration       |  |  calibration            |
|  (dispatcher, Trace) |  |  (Profile, drift,       |
|                      |  |   AtomicThresholds)     |
+----------+-----------+  +----------+--------------+
           |                         |
           +------------+------------+
                        |
                        v
+-----------------------------------------------------+
|  internal/tau           (COEUR : operateur t,       |
|    dimensions/{dsens, dauthority, dinvariant}        |
|    invariants/{i1..i5} + fuzz_targets                |
|    frontier.go, operator.go)                         |
+-----------------------------------------------------+
                        ^
           +------------+------------+
           |                         |
+----------+-----------+  +----------+--------------+
|  internal/bridge/llm |  |  internal/bridge/       |
|  (interface Client,  |  |  agentmeshkafka         |
|   Stub deterministe) |  |  (FileAdapter, DTO)     |
+----------------------+  +-------------------------+
```

Regles d'etancheite : `tau/* → orchestration`, `tau/* → bridge`, `bridge → tau/*` direct,
et `dimensions <-> invariants` : tous interdits. Violation = `arch_test.go` rouge.

---

## 5. Exemples d'usage

### A. `tau decide` -- dispatch JSON en/out (M1+)

```bash
# Echange hors frontiere -> Refus
echo '{"id":"e-hors","intent_description":"hello","universe_open":false}' \
  | ./tau decide

# Echange dans frontiere -> Probabiliste ou Deterministe selon scores
echo '{"id":"e-in","intent_description":"creative generation with open capabilities",
      "universe_open":true,"composition_variable":true,
      "peer_probabilistic":true,"cost_unbounded":true,
      "initiator":{"id":"svc-a","name":"Service A","roles":["caller"]},
      "target":{"id":"svc-b","name":"Service B","roles":["callee"]}}' \
  | ./tau decide
```

Codes de sortie : `0` = succes, `2` = entree invalide, `3` = Refus, `4` = erreur interne.

### B. `tau calibrate` -- reproductibilite byte-identique (M5)

```bash
# Calibration sur corpus annote
go run ./cmd/tau calibrate \
  --corpus tests/calibration/golden-corpus.jsonl \
  --output profiles/v1-seed42.json \
  --seed 42 \
  --date-revision 2026-12-01 \
  --version-monographie v2.4.3

# Verification byte-identique (deux runs -> meme sha256)
sha256sum profiles/v1-seed42.json
```

### C. Intégration Go -- embed `Kernel.Decide` dans un service hote

```go
import (
    "context"
    "github.com/agbruneau/taugo/internal/app"
    "github.com/agbruneau/taugo/internal/tau"
)

dispatcher := app.NewDispatcher()   // injecte Stub LLM par defaut
x := tau.Exchange{
    ID:                  "e-1",
    IntentDescription:   "schedule approval with external authority",
    UniversOuvert:       true,
    CompositionVariable: true,
    PairProbabiliste:    true,
    CoutNonBorne:        true,
}
dec, err := dispatcher.Decide(context.Background(), x)
// dec.Regime : tau.Deterministe | tau.Probabiliste | tau.Refus
// dec.Trace  : scores, seuils, profil, invariants
```

Voir `internal/app/` pour l'injection d'un client LLM reel (`TAUGO_LLM_BACKEND=real`).

### D. Fuzz I1-I5 et tests E2E

```bash
make fuzz                  # fuzz 30 s sur I1-I5 (CI)
make fuzz-long             # fuzz 24 h (CI nocturne)
make e2e                   # integration FileAdapter -> Dispatcher (tag 'integration')
make e2e-calibration       # determinisme byte-identique (tag 'e2e')
make empirical-i4          # campagne empirique I4 sur 120 traces (tag 'empirical')
```

---

## 6. Etat V0.1.0

Tous les milestones M0-M6 ont ete livres. Version courante : `v0.1.0-alpha` (a tagger apres M6).

| Milestone | Date | Tag | Contenu | Statut |
|---|---|---|---|---|
| M0 | 2026-05-23 | v0.0.1-alpha | Squelette, CI, arch_test, frontier | livre |
| M1 | 2026-05-23 | v0.0.2-alpha | Dispatcher minimal, stub LLM, `tau decide` | livre |
| M2 | 2026-05-23 | v0.0.3-alpha | 3 dimensions, gardes I3/I4, etapes 1-7 dispatcher | livre |
| M3 | 2026-05-24 | v0.0.4-alpha | 5 invariants, fuzz I1-I5, etape 8 dispatcher | livre |
| M4 | 2026-05-24 | v0.0.5-alpha | Bridge AgentMeshKafka, campagne empirique I4 | livre |
| M5 | 2026-05-24 | v0.0.6-alpha | Calibration adaptative, drift, `tau calibrate` | livre |
| M6 | 2026-05-24 | v0.1.0 (cible) | ADR 0001-0005, docs theory/empirical, README final | en cours |

**V0.2+ envisage** : KafkaAdapter reel (bascule Regime A), calibration des poids par gradient
(V2 `CalibrateWeights`), fenetre glissante drift, TUI Bubble Tea (`tau-stack`).

---

## 7. Statut des invariants

Marqueurs epitemiques conformes a `InteroperabiliteAgentique/CLAUDE.md §1.4`.
Detail et conditions de refutation : [docs/theory/05-invariants.md](docs/theory/05-invariants.md)
*(chap. III.8.5)*.

| # | Enonce court | Statut | Cible fuzz |
|---|---|---|---|
| I1 | τ conserve la grandeur (deplace `t_fix`, pas le contenu) | Probable | `FuzzI1_Conservation` |
| I2 | Residu migrant non vide, non recablable hors ligne | Confirme | `FuzzI2_Irreductibilite` |
| I3 | D-AUTORITE asymetrique (fait institutionnel -- Searle 1995) ; sans `AttestationInstitutionnelle` → refus ontologique. Veille trimestrielle ; date 2026-05-16. | Probable | `FuzzI3_AsymetrieAutorite` |
| I4 | `i ≈ pendant ⟹ s ≈ pendant` ; configuration incoherente → refus | Hypothese | `FuzzI4_CoherenceContrainte` |
| I5 | Pile composee herite de la conjonction ; `M(π) >= max(|Ai|)` | Probable | `FuzzI5_CompositionConjonctive` |

I4 reste a statut **Hypothese** : le corpus synthétique M4 n'injecte pas les clés `Context`
qui pilotent D-INVARIANT au-dessus du seuil `θ_inv`. Voir [docs/empirical/I4-report.md](docs/empirical/I4-report.md).

---

## 8. Documentation

### Specification et planification

- [`PRD.md`](PRD.md) -- specification canonique V0.2 (911 l., 20 sections, glossaire)
- [`CLAUDE.md`](CLAUDE.md) -- conventions d'ingenierie, agent teams, anti-patrons
- [`PRDPlanning.md`](PRDPlanning.md) -- plan d'execution M0-M6 par agent teams
- [`CHANGELOG.md`](CHANGELOG.md) -- historique Keep-a-Changelog

### Theorie (renvois chap. III.8)

- [`docs/theory/03-operateur-tau.md`](docs/theory/03-operateur-tau.md) -- definition formelle τ, frontiere de validite *(III.8.3)*
- [`docs/theory/04-dimensions.md`](docs/theory/04-dimensions.md) -- D-SENS, D-AUTORITE, D-INVARIANT, sondes *(III.8.4)*
- [`docs/theory/05-invariants.md`](docs/theory/05-invariants.md) -- I1-I5, reformulations executables, conditions de refutation *(III.8.5)*
- [`docs/theory/06-conditions-validite.md`](docs/theory/06-conditions-validite.md) -- conditions de validite V1 *(III.8.6)*
- [`docs/theory/07-anti-patrons.md`](docs/theory/07-anti-patrons.md) -- 7 anti-patrons interdits *(III.8.7)*

### Algorithmes

- [`docs/algorithms/calibration.md`](docs/algorithms/calibration.md) -- grid search, encodage milli-unites, tie-break, byte-identite
- [`docs/algorithms/drift.md`](docs/algorithms/drift.md) -- 5 criteres de drift, skip empty-fingerprint, bascule Refus

### Decisions d'architecture (ADR)

- [`docs/adr/0001-clean-architecture-4-layers.md`](docs/adr/0001-clean-architecture-4-layers.md) -- 4 couches, regles d'etancheite
- [`docs/adr/0002-go-1.25-toolchain.md`](docs/adr/0002-go-1.25-toolchain.md) -- Go 1.25+, toolchain 1.26.x, golangci-lint v1.64.8
- [`docs/adr/0003-llm-client-injection.md`](docs/adr/0003-llm-client-injection.md) -- interface Client, injection dans app/, stub deterministe
- [`docs/adr/0004-agentmeshkafka-empirical-bridge.md`](docs/adr/0004-agentmeshkafka-empirical-bridge.md) -- bridge empirique, regime contingence
- [`docs/adr/0005-agentmeshkafka-dto.md`](docs/adr/0005-agentmeshkafka-dto.md) -- DTO neutre, pivot app/agentmesh.go

### Empirique

- [`docs/empirical/M2-sample-decisions.md`](docs/empirical/M2-sample-decisions.md) -- 10 decisions tracees, ventilation scores par dimension
- [`docs/empirical/fuzz-summary.md`](docs/empirical/fuzz-summary.md) -- rapport fuzz M3 : resultats I1-I5, 0 crash
- [`docs/empirical/I4-report.md`](docs/empirical/I4-report.md) -- campagne I4 : 120 traces, statut Hypothese inchange
- [`docs/empirical/I4-regime.md`](docs/empirical/I4-regime.md) -- regime B (contingence), conditions de bascule vers A
- [`docs/empirical/unmodeled.md`](docs/empirical/unmodeled.md) -- 3 observations non modelisees (OBS-001 a OBS-003)
- [`docs/empirical/case-study-bfsi.md`](docs/empirical/case-study-bfsi.md) -- cas d'usage BFSI anonymise (M6)

---

## 9. Stack technique

| Composant | Version | Note |
|---|---|---|
| Go | 1.25+ (toolchain 1.26.x) | Aligne FibGo ; `go.mod` authoritative |
| golangci-lint | v1.64.8 | Epingle ; 24 linters ; calque FibGo |
| Fuzz | natif Go (`testing.F`) | I1-I5 ; 30 s CI, 24 h nocturne |
| Build reproductible | `-trimpath -ldflags="-buildid= -X main.buildTimestamp=1778889600"` | Byte-identique a timestamp gele |
| Cross-compile | linux × {amd64,arm64}, darwin × {amd64,arm64}, windows × amd64 | `make build-all` |
| CI | GitHub Actions -- 3 OS (ubuntu, windows, macos) | `make test && make lint && make fuzz` |
| Race detector | CGO requis | Actif Linux/macOS CI ; Windows couvert via runner GH |

---

## 10. Cadence de revue

Conformement au PRD §16 :

- **Mensuelle (scope)** : revue des limites V1 contre nouvelles traces empiriques ; mise a jour de `docs/empirical/unmodeled.md`.
- **Trimestrielle (I3)** : veille sur l'invariant I3 (asymetrie d'autorite -- Searle 1995). Le profil porte `DateRevision` ; un profil perime entraîne `Refus` automatique (anti-patron #3, etape 3 dispatcher). Alerte CI a 30 jours avant peremption.

---

## 11. Licence

Apache-2.0. Voir [LICENSE](LICENSE).

---

## 12. Cosignature IA

TauGo a ete concu et realise en collaboration avec Claude (Anthropic), en tant que co-auteur
technique de l'implementation. Chaque commit produit par un agent IA porte la mention :

```
Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

Cette pratique est conforme a la politique editoriale du projet
(`CLAUDE.md §Conventions de code`) et documentee dans l'historique `git log`.

---

*TauGo V0.1.0 -- 2026-05-24. Reference canonique : `agbruneau/InteroperabiliteAgentique` v2.4.3,
chap. III.8. Spec complete : [`PRD.md`](PRD.md).*
