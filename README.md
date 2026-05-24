# TauGo — Kernel exécutable Go de l'opérateur τ

[![CI](https://github.com/agbruneau/taugo/actions/workflows/ci.yml/badge.svg)](https://github.com/agbruneau/taugo/actions/workflows/ci.yml)
[![Coverage](https://github.com/agbruneau/taugo/actions/workflows/coverage.yml/badge.svg)](https://github.com/agbruneau/taugo/actions/workflows/coverage.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/agbruneau/taugo.svg)](https://pkg.go.dev/github.com/agbruneau/taugo)
[![Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

> **État** — `v0.1.0` taggé · `v0.1.1-pre` livré *(refactor consolidation post-audit, 2026-05-24, commit `2cf560c`)*. Tag `v0.1.1` à apposer après revue humaine. Source du refactor : [`AUDIT.md`](AUDIT.md) · [`AUDITPlan.md`](AUDITPlan.md).

---

## 1. Doctrine

TauGo implémente le **kernel exécutable de l'opérateur τ** défini au chapitre III.8 de la monographie *Interopérabilité Agentique en Écosystème d'Entreprise* (`agbruneau/InteroperabiliteAgentique` v2.4.3, *chap. III.8*).

L'API publique unique est :

```go
// Decide est l'unique point de décision public. Renvoie Deterministe,
// Probabiliste ou Refus — jamais un comportement du pair appelé.
// La trace expose scores ventilés (D-SENS, D-AUTORITÉ, D-INVARIANT),
// invariants, seuils, profil de calibration.
func (k *Kernel) Decide(ctx context.Context, x Exchange) (Decision, error)
```

**Régimes de sortie** : `Deterministe | Probabiliste | Refus`.

**Opérateur τ** — déplace l'instant de fixation `t_fix(g)`, jamais le contenu de la grandeur `g`. τ décide *où* le sens, l'autorité et le support se fixent, donc *avec quoi* appeler.

**Trois dimensions** *(III.8.4)* :

| Dimension | Sémantique | Plage |
|---|---|---|
| D-SENS | Lieu de fixation du sens (avant / pendant) | [0,1] |
| D-AUTORITÉ | Portée de la chaîne de délégation | [0,1] |
| D-INVARIANT | Support des invariants d'intégration | [0,1] |

Scores ventilés exposés dans `Decision.Trace.{DSens, DAuthority, DInvariant}` *(v0.1.1, ADR-0008)*.

**Cinq invariants réfutables** *(III.8.5)* : I1 conservation, I2 irréductibilité, I3 asymétrie d'autorité, I4 cohérence, I5 composition conjonctive. Statut épistémique : voir [§7 Statut des invariants](#7-statut-des-invariants).

**Frontière de validité** — τ ne s'applique qu'aux échanges satisfaisant simultanément les quatre violations des conditions classiques. Hors frontière → `Refus` avec diagnostic. Aucun fallback silencieux. La méthode canonique est `x.FrontierCheck()` *(v0.1.1)*.

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

Calibration reproductible byte-identique :

```bash
./tau calibrate --corpus tests/calibration/golden-corpus.jsonl \
                --output current.json \
                --seed 1 \
                --date-revision 2027-12-01
```

Remarque : `-race` nécessite CGO (gcc). Les runners Linux/macOS CI l'activent ; sous Windows sans gcc local, utiliser `go test -short ./...`.

---

## 4. Architecture

Clean Architecture, quatre couches strictes, calque structurel de `agbruneau/FibGo`. Étanchéité gardée par `internal/arch_test.go` *(7 règles depuis v0.1.1)*. Détail dans [docs/adr/0001-clean-architecture-4-layers.md](docs/adr/0001-clean-architecture-4-layers.md).

```
+-----------------------------------------------------+
|  cmd/tau                    (CLI -- decide, calibrate)|
+------------------------+----------------------------+
                         |
                         v
+-----------------------------------------------------+
|  internal/app           (lifecycle + injection LLM, |
|                          DefaultProfile par défaut) |
+------------------------+----------------------------+
                         |
           +-------------+-------------+
           |                           |
           v                           v
+----------------------+  +-------------------------+
|  internal/           |  |  internal/              |
|  orchestration       |  |  calibration            |
|  (dispatcher 8 étapes|  |  (Profile, drift,       |
|   Decision, Trace)   |  |   AtomicThresholds,     |
|                      |  |   DefaultProfile)       |
+----------+-----------+  +----------+--------------+
           |                         |
           +------------+------------+
                        |
                        v
+-----------------------------------------------------+
|  internal/tau           (COEUR : opérateur τ,       |
|    operator.go, frontier.go, diagnostics.go         |
|    dimensions/{dsens, dauthority, dinvariant, score}|
|    invariants/{i1..i5, evaluator} + fuzz_targets)   |
+-----------------------------------------------------+
            ^                            ^
            |                            |
+-----------+--------+   +---------------+--------------+
| internal/bridge/llm|   | internal/bridge/             |
| (interface Client, |   | agentmeshkafka               |
|  Stub déterministe)|   | (FileAdapter, DTO neutre)    |
+--------------------+   +------------------------------+

   +----------------------+   +----------------------+
   |  internal/thresholds |   |  internal/errors     |
   |  (type valeur        |   |  (typed errors +     |
   |   transverse, V0.1.1)|   |   sentinels, V0.1.1) |
   +----------------------+   +----------------------+

   +----------------------+
   |  internal/testutil   |
   |  (BuildExchange, ... |
   |   helpers de test)   |
   +----------------------+
```

Règles d'étanchéité (7) :
1. `tau/* → orchestration` interdit
2. `tau/* → bridge` interdit
3. `bridge → tau/*` direct interdit
4. `dimensions ↔ invariants` interdit (orthogonalité 3D vs I1-I5)
5. **LLM concret hors `app/` et `bridge/llm/`** : interdit *(`TestArchNoConcreteLLMInDomain`, v0.1.1)*
6. **`calibration → tau/orch/bridge`** : interdit *(v0.1.1)*
7. **`internal/thresholds → *`** : interdit (couche descendante, ADR-0006)

Violation = `arch_test.go` rouge.

---

## 5. Exemples d'usage

### A. `tau decide` — dispatch JSON en/out

```bash
# Échange hors frontière → Refus
echo '{"id":"e-hors","intent_description":"hello","universe_open":false}' \
  | ./tau decide

# Échange dans frontière → Probabiliste ou Deterministe selon scores
echo '{"id":"e-in","intent_description":"creative generation with open capabilities",
      "universe_open":true,"composition_variable":true,
      "peer_probabilistic":true,"cost_unbounded":true,
      "initiator":{"id":"svc-a","name":"Service A","roles":["caller"]},
      "target":{"id":"svc-b","name":"Service B","roles":["callee"]}}' \
  | ./tau decide
```

Codes de sortie : `0` = succès, `2` = entrée invalide, `3` = Refus, `4` = erreur interne.

Sortie JSON v0.1.1 :

```json
{
  "id": "e-in",
  "regime": "Deterministe",
  "trace": {
    "d_sens":      {"value": 0.42, "probes": [...]},
    "d_authority": {"value": 0.31, "probes": [...]},
    "d_invariant": {"value": 0.55, "probes": [...]},
    "tau_score": 0.46,
    "profile_version": "0.1.0",
    "date_revision": "2026-12-01T00:00:00Z",
    ...
  }
}
```

Le `Regime` est désormais une string PascalCase *(v0.1.1, ADR-0008)*. La désérialisation reste rétro-compatible avec les corpus v0.1.0 (int legacy accepté).

### B. `tau calibrate` — reproductibilité byte-identique

```bash
# Calibration sur corpus annoté
go run ./cmd/tau calibrate \
  --corpus tests/calibration/golden-corpus.jsonl \
  --output profiles/v1-seed42.json \
  --seed 42 \
  --date-revision 2027-12-01 \
  --version-monographie v2.4.3

# Vérification byte-identique (deux runs → même sha256)
sha256sum profiles/v1-seed42.json
```

Validation corpus : chaque `CorpusEntry` est validée à `LoadCorpus` (4 valeurs `LabeledRegime` admises ; rétro-compat `ExpectedRegime`).

### C. Intégration Go — embed `Kernel.Decide` dans un service hôte

```go
import (
    "context"
    "github.com/agbruneau/taugo/internal/app"
    "github.com/agbruneau/taugo/internal/tau"
)

// app.NewDispatcher charge calibration.DefaultProfile() par défaut
// (v0.1.1 — active la garde de péremption sur le chemin standard).
dispatcher := app.NewDispatcher()
x := tau.Exchange{
    ID:                  "e-1",
    IntentDescription:   "schedule approval with external authority",
    UniversOuvert:       true,
    CompositionVariable: true,
    PairProbabiliste:    true,
    CoutNonBorne:        true,
}
dec, err := dispatcher.Decide(context.Background(), x)
// dec.Regime         : tau.Deterministe | tau.Probabiliste | tau.Refus
// dec.Trace.DSens    : *tau.Score (nil si pas calculé)
// dec.Trace.DAuthority, DInvariant : idem
// dec.Trace          : tau_score composite, profil, invariants
```

Voir `internal/app/` pour l'injection d'un client LLM réel (`TAUGO_LLM_BACKEND=real`).

### D. Fuzz I1-I5 et tests E2E

```bash
make fuzz                  # fuzz 30 s sur I1-I5 (CI)
make fuzz-long             # fuzz 24 h (CI nocturne)
make e2e                   # intégration FileAdapter → Dispatcher (tag 'integration')
make e2e-calibration       # déterminisme byte-identique (tag 'e2e')
make empirical-i4          # campagne empirique I4 sur 120 traces (tag 'empirical')
```

---

## 6. État v0.1.0 + refactor v0.1.1

Tous les milestones M0-M6 ont été livrés sous tag `v0.1.0` (2026-05-24).

| Milestone | Date | Tag | Contenu | Statut |
|---|---|---|---|---|
| M0 | 2026-05-23 | `v0.0.1-alpha` | Squelette, CI, arch_test, frontier | livré |
| M1 | 2026-05-23 | `v0.0.2-alpha` | Dispatcher minimal, stub LLM, `tau decide` | livré |
| M2 | 2026-05-23 | `v0.0.3-alpha` | 3 dimensions, gardes I3/I4, étapes 1-7 dispatcher | livré |
| M3 | 2026-05-24 | `v0.0.4-alpha` | 5 invariants, fuzz I1-I5, étape 8 dispatcher | livré |
| M4 | 2026-05-24 | `v0.0.5-alpha` | Bridge AgentMeshKafka, campagne empirique I4 | livré |
| M5 | 2026-05-24 | `v0.0.6-alpha` | Calibration adaptative, drift, `tau calibrate` | livré |
| M6 | 2026-05-24 | `v0.1.0` | ADRs 0001-0005, docs theory/empirical, README final | livré |

### Refactor v0.1.1-pre *(2026-05-24, commit `2cf560c`)*

Consolidation post-audit orchestrée par Agent teams selon [`AUDITPlan.md`](AUDITPlan.md) — 42 tâches, 4 vagues parallèles, 72 fichiers modifiés (+4199/-347 LOC).

**Highlights** :
- **P0-01 fermé** : nouvelle garde `TestArchNoConcreteLLMInDomain` (anti-patron #6 désormais en CI).
- **P0-02 fermé** : `app.NewDispatcher()` charge `calibration.DefaultProfile()` par défaut (garde péremption active sur chemin CLI).
- **Trace ventilée** *(ADR-0008)* : `Trace.DSens / DAuthority / DInvariant` peuplés ; `EvaluateI3`/`EvaluateI4` lisent les scores ventilés.
- **Profile.Weights injecté** au runtime à l'étape 6 du dispatcher.
- **Packages ajoutés** : `internal/thresholds` (ADR-0006), `internal/errors` peuplé (ADR-0009), `internal/testutil.BuildExchange`.
- **4 ADRs nouvelles** : 0006 thresholds transverses · 0007 hystérèse V1 · 0008 Trace ventilée · 0009 erreurs typées.
- **Gate CI per-package** activé : `internal/tau/*` ≥ 90 %, global ≥ 80 %.
- **Couverture globale** : 92.1 % *(était 90.9 %)*.
- **Anti-patrons §7.2** : 7/7 gardés *(était 6/7)*.
- **Purge agressive** : 10 `cov*.out`, 2 `*.exe`, 6 plans M0-M6 archivés, `ruvector.db` désindexé, 3 packages morts supprimés (`config`, `metrics`, `testutil/doc.go`).

Détail complet : [`CHANGELOG.md`](CHANGELOG.md) section v0.1.1-pre.

**V0.2+ envisagé** : KafkaAdapter réel (bascule Régime A), calibration des poids par gradient (V2 `CalibrateWeights`), fenêtre glissante drift, TUI Bubble Tea (`tau-stack`), mécanisation Lean 4 (`cia-runtime`, ADR-0010 à créer), hystérèse complète avec `LastRegime` (cf. ADR-0007).

---

## 7. Statut des invariants

Marqueurs épistémiques conformes à `InteroperabiliteAgentique/CLAUDE.md §1.4`. Détail et conditions de réfutation : [docs/theory/05-invariants.md](docs/theory/05-invariants.md) *(chap. III.8.5)*.

| # | Énoncé court | Statut | Cible fuzz | Débit smoke |
|---|---|---|---|---|
| I1 | τ conserve la grandeur (déplace `t_fix`, pas le contenu) | Probable | `FuzzI1_Conservation` | ~8.6 M exec/s |
| I2 | Résidu migrant non vide, non recâblable hors ligne | Confirmé | `FuzzI2_Irreductibilite` | ~8.6 M exec/s |
| I3 | D-AUTORITÉ asymétrique (fait institutionnel — Searle 1995) ; sans `AttestationInstitutionnelle` → refus ontologique. Veille trimestrielle ; date 2026-05-16. Lit `Trace.DAuthority` ventilé depuis v0.1.1. | Probable | `FuzzI3_AsymetrieAutorite` | ~8.2 M exec/s |
| I4 | `i ≈ pendant ⟹ s ≈ pendant` ; configuration incohérente → refus. Détecte le bypass silencieux via scores ventilés depuis v0.1.1. | Hypothèse | `FuzzI4_CoherenceContrainte` | ~9.5 M exec/s |
| I5 | Pile composée hérite de la conjonction ; `M(π) >= max(\|Ai\|)`. `BoundsHold` optimisé -46 % ns/op en v0.1.1. | Probable | `FuzzI5_CompositionConjonctive` | ~1.1 M exec/s |

I4 reste à statut **Hypothèse** : le corpus synthétique M4 n'injecte pas les clés `Context` qui pilotent D-INVARIANT au-dessus du seuil `θ_inv`. v0.1.1 rend la détection ventilée opérationnelle ; corroboration empirique différée à M7. Voir [docs/empirical/I4-report.md](docs/empirical/I4-report.md).

---

## 8. Documentation

### Spécification et planification

- [`PRD.md`](PRD.md) — spécification canonique V0.2 (911 l., 20 sections, glossaire)
- [`CLAUDE.md`](CLAUDE.md) — conventions d'ingénierie, agent teams, anti-patrons
- [`PRDPlanning.md`](PRDPlanning.md) — plan d'exécution M0-M6 par agent teams
- [`AUDIT.md`](AUDIT.md) — audit consolidé v0.1.0 → v0.1.1
- [`AUDITPlan.md`](AUDITPlan.md) — plan refactor 42 tâches
- [`CHANGELOG.md`](CHANGELOG.md) — historique Keep-a-Changelog

### Théorie (renvois chap. III.8)

- [`docs/theory/03-operateur-tau.md`](docs/theory/03-operateur-tau.md) — définition formelle τ, frontière de validité *(III.8.3)*
- [`docs/theory/04-dimensions.md`](docs/theory/04-dimensions.md) — D-SENS, D-AUTORITÉ, D-INVARIANT, sondes *(III.8.4)*
- [`docs/theory/05-invariants.md`](docs/theory/05-invariants.md) — I1-I5, reformulations exécutables, conditions de réfutation *(III.8.5)*
- [`docs/theory/06-conditions-validite.md`](docs/theory/06-conditions-validite.md) — conditions de validité V1 *(III.8.6)*
- [`docs/theory/07-anti-patrons.md`](docs/theory/07-anti-patrons.md) — 7 anti-patrons interdits *(III.8.7)*

### Algorithmes

- [`docs/algorithms/calibration.md`](docs/algorithms/calibration.md) — grid search, encodage milli-unités, tie-break, byte-identité
- [`docs/algorithms/drift.md`](docs/algorithms/drift.md) — 5 critères de drift, skip empty-fingerprint, bascule Refus
- [`docs/algorithms/dispatch.md`](docs/algorithms/dispatch.md) — pseudo-algorithme 8 étapes

### Décisions d'architecture (ADR)

- [`docs/adr/0001-clean-architecture-4-layers.md`](docs/adr/0001-clean-architecture-4-layers.md) — 4 couches, règles d'étanchéité
- [`docs/adr/0002-go-1.25-toolchain.md`](docs/adr/0002-go-1.25-toolchain.md) — Go 1.25+, toolchain 1.26.x, golangci-lint v1.64.8
- [`docs/adr/0003-llm-client-injection.md`](docs/adr/0003-llm-client-injection.md) — interface Client, injection dans app/, stub déterministe
- [`docs/adr/0004-agentmeshkafka-empirical-bridge.md`](docs/adr/0004-agentmeshkafka-empirical-bridge.md) — bridge empirique, régime contingence
- [`docs/adr/0005-agentmeshkafka-dto.md`](docs/adr/0005-agentmeshkafka-dto.md) — DTO neutre, pivot app/agentmesh.go
- [`docs/adr/0006-types-valeur-transverses.md`](docs/adr/0006-types-valeur-transverses.md) — package `internal/thresholds/` *(v0.1.1)*
- [`docs/adr/0007-hysteresis-v1-simplifiee.md`](docs/adr/0007-hysteresis-v1-simplifiee.md) — hystérèse V1 simplifiée, cible V0.2 *(v0.1.1)*
- [`docs/adr/0008-trace-ventilee-scores-dimensions.md`](docs/adr/0008-trace-ventilee-scores-dimensions.md) — `Trace.DSens/DAuthority/DInvariant` *(v0.1.1)*
- [`docs/adr/0009-types-erreurs-typees.md`](docs/adr/0009-types-erreurs-typees.md) — `DispatchError`/`RefusError`/`CalibrationError` *(v0.1.1)*

### Empirique

- [`docs/empirical/M2-sample-decisions.md`](docs/empirical/M2-sample-decisions.md) — 10 décisions tracées, ventilation scores par dimension
- [`docs/empirical/fuzz-summary.md`](docs/empirical/fuzz-summary.md) — rapport fuzz M3 : résultats I1-I5, 0 crash
- [`docs/empirical/I4-report.md`](docs/empirical/I4-report.md) — campagne I4 : 120 traces, statut Hypothèse inchangé
- [`docs/empirical/I4-regime.md`](docs/empirical/I4-regime.md) — régime B (contingence), conditions de bascule vers A
- [`docs/empirical/unmodeled.md`](docs/empirical/unmodeled.md) — 3 observations non modélisées (OBS-001 à OBS-003)
- [`docs/empirical/case-study-bfsi.md`](docs/empirical/case-study-bfsi.md) — cas d'usage BFSI anonymisé (M6)

### Archives

- [`docs/archive/plans-m0-m6/`](docs/archive/plans-m0-m6/) — plans détaillés M1-M6 archivés v0.1.1 (9 824 LOC)

---

## 9. Stack technique

| Composant | Version | Note |
|---|---|---|
| Go | 1.25+ (toolchain 1.26.x) | Aligné FibGo ; `go.mod` authoritative |
| golangci-lint | v1.64.8 | Épinglé ; 24 linters ; calque FibGo |
| Fuzz | natif Go (`testing.F`) | I1-I5 ; 30 s CI, 24 h nocturne |
| Build reproductible | `-trimpath -ldflags="-buildid= -X main.buildTimestamp=..."` | Byte-identique à timestamp gelé |
| Cross-compile | linux × {amd64,arm64}, darwin × {amd64,arm64}, windows × amd64 | `make build-all` |
| CI | GitHub Actions — 3 OS (ubuntu, windows, macos) | `make test && make lint && make fuzz` + **gate per-package ≥ 90 % `tau/*`** |
| Race detector | CGO requis | Actif Linux/macOS CI ; Windows couvert via runner GH |
| Couverture | global ≥ 80 %, per-package `tau/*` ≥ 90 % | gate CI `coverage.yml` *(actif v0.1.1)* |

---

## 10. Cadence de revue

Conformément au PRD §16 :

- **Mensuelle (scope)** : revue des limites V1 contre nouvelles traces empiriques ; mise à jour de `docs/empirical/unmodeled.md`.
- **Trimestrielle (I3)** : veille sur l'invariant I3 (asymétrie d'autorité — Searle 1995). Le profil porte `DateRevision` ; un profil périmé entraîne `Refus` automatique (anti-patron #3, étape 3 dispatcher). Alerte CI à 30 jours avant péremption. **v0.1.1** : `app.NewDispatcher()` charge un profil par défaut, activant la garde même sans calibration explicite.

---

## 11. Licence

Apache-2.0. Voir [LICENSE](LICENSE).

---

## 12. Cosignature IA

TauGo a été conçu et réalisé en collaboration avec Claude (Anthropic), en tant que co-auteur technique de l'implémentation. Chaque commit produit par un agent IA porte la mention :

```
Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

Cette pratique est conforme à la politique éditoriale du projet (`CLAUDE.md §Conventions de code`) et documentée dans l'historique `git log`.

---

*TauGo v0.1.1-pre — 2026-05-24. Référence canonique : `agbruneau/InteroperabiliteAgentique` v2.4.3, chap. III.8. Spec complète : [`PRD.md`](PRD.md). Refactor v0.1.1 : [`AUDIT.md`](AUDIT.md) · [`AUDITPlan.md`](AUDITPlan.md).*
