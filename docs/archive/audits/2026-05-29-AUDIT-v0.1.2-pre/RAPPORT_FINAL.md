# RAPPORT FINAL — Audit Go multi-agents TauGo (v0.1.2-pre)

> **Cible** : kernel exécutable de l'opérateur τ — `github.com/agbruneau/taugo`, HEAD `1948a7b` (`v0.1.0-17`), état v0.1.2-pre.
> **Date** : 2026-05-29 · **Méthode** : 6 sous-agents read-only, 3 vagues, sorties structurées. **Plateforme** : Windows 11 / Go 1.26.3 / `CGO_ENABLED=0` / 24 cœurs.
> **Langue** : FR-CA. **Marqueurs** : `[confirmé]` `[probable]` `[hypothèse]` `[à vérifier]`.
> Détail par axe : [`01_conformite_tau.md`](01_conformite_tau.md) · [`02_invariants_epistemique.md`](02_invariants_epistemique.md) · [`03_concurrence.md`](03_concurrence.md) · [`04_performance.md`](04_performance.md) · [`05_idiomatique.md`](05_idiomatique.md) · [`06_architecture_tests.md`](06_architecture_tests.md). Méthodo : [`00_bootstrap.md`](00_bootstrap.md), [`CONVENTIONS.md`](CONVENTIONS.md).

---

## 1. Verdict global

**Le kernel τ est sain et conforme ; aucun défaut CRITIQUE.** [confirmé] (a) Conformité III.8 : frontière à 4 conditions simultanées correctement encodée, refus de premier rang avec diagnostic obligatoire, aucun fallback silencieux hors frontière. (b) Invariants : les 5 cibles fuzz passent sans crash sur ~200 M exécutions cumulées (I1 43M, I2 40M, I3 42M, I4 44M, I5 32M sur 30 s chacune) ; statuts épistémiques globalement honnêtes. (c) Architecture : 7 règles d'étanchéité vertes, gate `tau/* ≥ 90 %` tenu (tau 100 %, dimensions 98,7 %, invariants 92,7 %), outillage statique 100 % vert (gofmt/vet/staticcheck/golangci-lint v1.64.8). (d) État pure-local post-CI conforme à l'ADR-0010, veille I3 manuelle mitigée par garde runtime.

**Risque #1** : la fragilité n'est **pas** algorithmique mais **épistémique et documentaire** — plusieurs affirmations datées portent le marqueur « Confirmé » alors que la preuve ne le soutient pas (couverture 92,1 %, débits fuzz ~8,6 M/s), et la spec décrit une arborescence qui ne correspond plus au code. Pour un projet de recherche dont la *discipline épistémique est un livrable*, ces surventes sont le défaut le plus structurant. **Un seul défaut MAJEUR est fonctionnel** : la CLI `tau calibrate` contourne la validation du corpus et produit silencieusement un profil dégénéré (C1-01).

**Bilan chiffré** : **0 CRITIQUE · 10 MAJEUR · 16 MINEUR · 15 INFORMATIF** (41 constats).

---

## 2. Tableau de bord

### Constats par sévérité × axe

| Axe | CRITIQUE | MAJEUR | MINEUR | INFORMATIF |
|---|:--:|:--:|:--:|:--:|
| SA1 Conformité τ | 0 | 1 | 3 | 2 |
| SA2 Invariants/épistémique | 0 | 2 | 4 | 2 |
| SA3 Concurrence | 0 | 2 | 2 | 2 |
| SA4 Performance | 0 | 2 | 2 | 3 |
| SA5 Idiomatique | 0 | 1 | 2 | 3 |
| SA6 Architecture/tests | 0 | 2 | 3 | 3 |
| **Total** | **0** | **10** | **16** | **15** |

### Indicateurs clés

| Indicateur | Mesure | Marqueur |
|---|---|---|
| Déterminisme calibration | 2 runs → SHA256 identique == golden épinglé `d753245b…ff6c7` | [confirmé] |
| 7 règles d'étanchéité | toutes vertes (`TestArchitectureLayering`, `TestBridgeNoTauImport`, `TestArchNoConcreteLLMInDomain`, `TestNoPredictiveAPI`) | [confirmé] |
| Gate couverture `tau/*` ≥ 90 % | tau 100 % / dimensions 98,7 % / invariants 92,7 % | [confirmé] |
| Couverture globale | **89,2 %** (`-coverpkg=./...`) — *et non 92,1 %* (moyenne per-package pondérée v0.1.1) | [confirmé] (cf. A6-01) |
| Fuzz I1-I5 | 0 crash, ~200 M exécutions cumulées | [confirmé] |
| Débit fuzz mesuré (ce poste) | I1-I4 ~1,4 M/s · I5 ~1,1 M/s — *et non 8,2-9,5 M/s* | [confirmé] (cf. P4-01) |
| `go vet` / `staticcheck` / `golangci-lint v1.64.8` | exit 0, 0 alerte, 24 linters | [confirmé] |
| `go mod verify` | all modules verified | [confirmé] |
| Détecteur `-race` | **non exécuté** (CGO off, pas de compilateur C) | [à vérifier] |
| Outils dispo durant l'audit | go 1.26.3 ✓ · golangci-lint v1.64.8 ✓ (pin exact) · staticcheck v0.7.0 ✓ · gosec dev ✓ · make ✗ · gcc/clang ✗ | [confirmé] |

---

## 3. Validité scientifique (I1-I5)

Section dédiée — c'est l'axe central d'un projet de recherche.

| # | Statut annoncé | Verdict d'audit | Honnêteté |
|---|---|---|---|
| I1 conservation | Probable | Fuzz 43M, 0 crash. Propriété V1 (4 conditions de frontière). | **Honnête.** Réserve cosmétique : `EvaluateI1` compare un littéral codé en dur au lieu de la constante `tau.DiagFrontiereFranchie` (I2-04). |
| I2 irréductibilité | Confirmé | Fuzz 40M, 0 crash. La plus déductive des cinq. | **Honnête pour la propriété V1 encodée.** Écart V1 ↔ propriété monographique complète signalé dans `theory/05` mais pas dans le godoc (I2-08, asymétrie de traçabilité). |
| I3 asymétrie d'autorité | Probable | Garde ontologique active, péremption gardée sur chemin CLI. | **Honnête.** Incohérences de dates entre godocs (2026-05-16 vs 2026-05-24) et entre `DateRevision` profil (2026-12-01) / `I3PerimptionLimite` (2027-01-01) (I2-05). |
| I4 cohérence | Hypothèse | Campagne synthétique inconclusive : TP=0, FN=0, D-INVARIANT figé à 0,25 < θ_inv 0,50. | **Honnête et exemplaire** (I2-07) : le rapport distingue « absence de preuve » de « preuve d'absence ». |
| I5 composition | Probable | `BoundsHold` fuzzé (mono-passe confirmé). Mais `EvaluateI5` retourne toujours `Held`. | **Honnête à condition de lire « propriété mathématique fuzzée », non « invariant vérifié sur chaque décision »** (I2-06). |

**Promotion d'I4 (Hypothèse → Probable)** — ce qui manque, précisément : (a) un corpus dont les clés `Context` (`event_registry`, `idempotency_key_mode`, profondeur de délégation) font monter D-INVARIANT ≥ θ_inv ; (b) ≥ 10 vrais positifs I4 (refus déclenché par incohérence réelle, cf. profil `i4-heavy` enrichi) ; (c) idéalement des traces réelles (AgentMeshKafka inexistant — Régime B). La détection ventilée v0.1.1 (ADR-0008) améliore la **capacité du code** (testée) mais ne change **pas** le statut empirique tant que (a) n'est pas livré.

**Survente épistémique détectée (MAJEUR)** : les débits fuzz ~8,2-9,5 M/s (CLAUDE.md, PRD §15) conflatent deux métriques — débit de la fonction-propriété scalaire isolée vs débit du moteur `go test -fuzz` (instrumentation + 24 workers + mutation), ce dernier mesuré à ~1,4 M/s. Affirmation chiffrée datée **sans marqueur d'incertitude** → viole les conventions éditoriales du projet (P4-01).

---

## 4. Synthèse par axe (du plus critique au moins)

### SA1 — Conformité τ (1 MAJEUR)
- **C1-01 MAJEUR [CODE]** — `cmd/tau/calibrate.go:loadCorpus` n'appelle ni `migrate()` ni `Validate()` (contrairement à `calibration.LoadCorpus`). Un corpus à `labeled_regime` invalide, ou un corpus legacy (`expected_regime` seul, non migré), passe avec exit 0 et produit un profil dégénéré (plancher de grille `deterministe=0.1`). La rétro-compat annoncée v0.1.1 **n'est pas effective via la CLI**. Calibration de production silencieusement erronée.
- C1-02/03/04 MINEUR — exit code « 3=Refus » jamais atteint (Refus → exit 0) ; `M2-sample-decisions.md` montre l'ancien format `regime:3` (int) ; message d'erreur `Validate()` nomme `ExpectedRegime` au lieu de `LabeledRegime`.

### SA2 — Invariants & épistémique (2 MAJEUR)
- **I2-01 MAJEUR [DOC]** — `docs/theory/07-anti-patrons.md` ne documente que 4 anti-patrons et s'auto-déclare « Confirmé » alors que le projet en garde **7**.
- **I2-02 MAJEUR [CODE+DOC]** — l'artefact `testdata/empirical-i4-results.json` (suivi git) embarque `time.Now()` → re-exécution non byte-identique, contredisant l'affirmation « gelé » d'`I4-report.md`. De plus le `sensitivity:-1` checked-in est absent à la régénération (artefact édité à la main).
- I2-03..06 MINEUR — test I4 au nom mensonger + commentaire obsolète ; littéraux codés en dur dans I1/I2 ; dates I3 incohérentes ; `EvaluateI5` no-op (Held).

### SA3 — Concurrence (2 MAJEUR) — `-race` non exécuté
- **R3-01 MAJEUR [CODE]** — `internal/app/agentmesh.go` : `errOut` à perte silencieuse (`default:` drop si buffer de 8 plein) → perte d'observabilité sur erreurs non-fatales.
- **R3-02 MAJEUR [CODE/DOC]** — `SetTuning` fait 6 `Store` indépendants (fenêtre RMW non atomique) ; docstring « atomically updates all thresholds in one coordinated call » survendue. **Impact actuel nul** car `AtomicThresholds` est du **code mort** (R3-05).
- R3-03/04 MINEUR — `Decide` n'inspecte jamais `ctx.Err()`/`ctx.Done()` ; `*Profile` partagé non synchronisé (sûr aujourd'hui, risque futur hot-reload).
- **Le cœur `Decide` est concurremment sain par construction** (immuabilité, zéro goroutine) [confirmé au niveau statique].

### SA4 — Performance (2 MAJEUR)
- **P4-01 MAJEUR [DOC]** — débits fuzz ~6× au-dessus de la mesure reproductible, sans marqueur ni méthodologie (cf. §3).
- **P4-02 MAJEUR [CODE/DOC]** — **aucun benchmark** pour `Decide`, les 3 dimensions, l'orchestration ni la calibration → la directive 5 (« régression perf > 5 % bloquante sur `tau/*` et `calibration/*` ») est **non vérifiable de facto**.
- P4-03/04 MINEUR — `BoundsHold` alloue ~2× la mémoire d'`Aggregate` (map locale par couche) ; commentaire de benchmark périmé. Hot path `Decide` propre [confirmé].

### SA5 — Idiomatique Go (1 MAJEUR)
- **Q5-01 MAJEUR [CODE/DOC, touche ADR-0009]** — `*RefusError` n'a ni `Unwrap` ni `Is` → les 4 sentinels `Diagnostic` sont **non-matchables via `errors.Is`** sur un `RefusError`, contrairement à ce que promet le godoc/ADR-0009. (Impact runtime nul car le refus est une *Decision*, pas une *error* — Q5-02 : `internal/errors` quasi non adopté en production.)
- Q5-03/04 MINEUR — findings gosec G304/G301/G115 confinés au tooling, connus/acceptés. **Outillage statique 100 % vert** [confirmé].

### SA6 — Architecture & tests (2 MAJEUR)
- **A6-01 MAJEUR [DOC]** — couverture globale « 92,1 % Confirmé » (PRD/CHANGELOG) non reproductible : `-coverpkg=./...` = **89,2 %** (méthode différente — moyenne per-package pondérée vs dénominateur global). Survente.
- **A6-02 MAJEUR [DOC]** — divergences spec ↔ arborescence : `cmd/generate-golden` (inexistant ; réel = `generate-corpus`), table PRD §8.1 cite encore `config`/`metrics` (supprimés, contradiction interne au même doc), `test/golden` absent (golden corpus réel sous `tests/calibration/`).
- A6-03/04/05 MINEUR — dossiers vides `internal/config`+`internal/metrics` ; étanchéité asymétrique (pas de règle `from` pour app/errors/testutil) ; `-race` non vérifiable.

---

## 5. Régression (corrections de l'audit v0.1.1)

Toutes les corrections du refactor v0.1.1 **tiennent toujours** [confirmé] :

| Item | Garde | Statut |
|---|---|---|
| **P0-01** anti-patron #6 (LLM concret hors domaine) | `TestArchNoConcreteLLMInDomain` (`arch_test.go:140`, AST 12 substrings) | **vert** (SA2 + SA6) |
| **P0-02** `app.NewDispatcher()` charge `DefaultProfile()` | `app.go:28-29` + `TestApp_NewDispatcher_*` | **vert** (SA2) |
| 7 anti-patrons gardés par test | divers | **7/7 gardés** (seul `theory/07` ne *documente* que 4 — I2-01) |
| 7 règles d'étanchéité | `arch_test.go` | **7/7 vertes** (SA6) |
| Déterminisme + hash golden épinglé | `TestCalibrate_GoldenCorpus_FixedHash` | **non-régression confirmée** (SA1) |
| Trace ventilée (ADR-0008) lue par I3/I4 | dispatcher étapes 2/4 | **opérante** (SA1) |
| `BoundsHold` mono-passe (-46 % ns/op v0.1.1) | — | **mono-passe confirmé** ; delta exact [à vérifier] (pas de baseline) |

---

## 6. Plan d'action priorisé (Effort × Impact)

> Cet audit est **read-only**. Aucun correctif appliqué. Les items [CODE] ci-dessous sont des **suites recommandées**, hors du périmètre de la révision documentaire courante.

### P0 — Correctness (faire avant tout tag v0.1.2)
1. **C1-01 [CODE, impact élevé / effort faible]** — faire déléguer `cmd/tau/calibrate.go:loadCorpus` à `calibration.LoadCorpus` (migre + valide) ; retourner exit ≠ 0 sur corpus invalide. Ajouter `TestRunCalibrate_CorpusInvalidRegime_NonZero` + `_CorpusLegacyExpectedRegime_Migre`.

### P1 — Honnêteté épistémique (révision documentaire — *traitée dans ce lot*)
2. **A6-01 [DOC]** — corriger « 92,1 % Confirmé » → 89,2 % (`coverpkg`) avec méthode explicite (PRD, CHANGELOG, README).
3. **P4-01 [DOC]** — qualifier les débits fuzz (`[à vérifier]` + méthodologie, ou valeur moteur ~1,4 M/s) (CLAUDE.md, PRD, README).
4. **I2-01 [DOC]** — réconcilier `theory/07` (4 → 7 anti-patrons, ou note de périmètre explicite).
5. **A6-02 [DOC]** — aligner PRD §8/§8.1 + CLAUDE.md sur l'arborescence réelle (`generate-corpus`, retrait `config`/`metrics`, `tests/calibration`).
6. **C1-02 / C1-03 / I2-02(doc) / I2-05(doc) / P4-07** — exit codes README, format M2, claim reproductibilité I4-report, dates I3 theory/05, débit I5.

### P2 — Robustesse (suites [CODE])
7. **R3-01** — documenter (ou corriger) la sémantique « best-effort lossy » d'`errOut`.
8. **P4-02** — ajouter `BenchmarkDecide` (+ dimensions, calibration) pour rendre la directive 5 vérifiable.
9. **Q5-01** — ajouter `Is`/`Unwrap` à `RefusError` **ou** corriger le godoc/ADR-0009.
10. **R3-02/R3-05** — publier un snapshot immuable (`atomic.Pointer`) **ssi** un hot-reload est planifié, sinon corriger la docstring ou retirer `AtomicThresholds` (code mort).

### P3 — Cosmétique (suites [CODE])
11. I2-03 (renommer test I4), I2-04 (constante au lieu de littéral), C1-04 (message d'erreur), P4-04 (commentaire bench), A6-03 (supprimer dossiers vides + refs `arch_test.go`), A6-04 (règles `from` défensives), Q5-04 (`//nolint` G115).

### P4 — Plateforme (gate manuel)
12. **A6-05 / transverse** — exécuter `go test -race ./...` sur Linux/macOS avec CGO avant tout tag (la CI le couvrait ; documenter ce gate manuel). Optionnel : `uber-go/goleak` en `TestMain` (ne requiert pas CGO).

---

## 7. Annexe

### Commandes représentatives (adaptées Windows, sans `make`, sans `-race`)
```
go build ./...                                              # exit 0
go test ./... -count=1                                      # 12 packages ok
go test -cover ./...                                        # per-package
go test -coverpkg=./... -coverprofile=cover.out ./...       # global 89,2 %
go test -fuzz=FuzzI<N>_... -fuzztime=30s ./internal/tau/invariants/   # 0 crash
go test -tags=e2e ./test/e2e/... -run "TestCalibration..."  # déterminisme
golangci-lint run ./...                                     # v1.64.8, 24 linters, 0 alerte
```

### Versions
go 1.26.3 windows/amd64 · golangci-lint **v1.64.8** (pin ADR-0002, schéma config v1) · staticcheck 2026.1 (v0.7.0) · gosec « dev » · python 3.14.5.

### Limites du sandbox / éléments [à vérifier]
- **`-race` indisponible** (`CGO_ENABLED=0`, pas de gcc/clang sous Windows) → **aucune data race runtime vérifiée**. Les verdicts de concurrence sont plafonnés à [probable]/[à vérifier]. À couvrir sur Linux/macOS avec CGO avant release.
- **Pas de baseline perf v0.1.0** dans le sandbox → delta `-46 % ns/op` de `BoundsHold` non vérifiable ; chiffres actuels fournis comme référence reproductible.
- Débits fuzz mesurés sur Windows/no-CGO (~1,4 M/s) — l'écart avec les annonces tient à la plateforme **et** à la confusion de métrique (cf. P4-01).
- `gosec` rapporte la version « dev » (sans tag) → règles G* standard mais version non attestable.

### Hygiène
Lecture seule respectée par les 6 agents. `git status` propre avant/après (artefacts confinés à `audit/`, supprimés en fin ; `testdata/empirical-i4-results.json` régénéré par la campagne puis restauré via `git restore`). `ruvector.db` (outillage swarm) modifié mais git-ignoré — aucune contamination du dépôt.

---

*Rapport produit par 6 sous-agents (general-purpose) orchestrés en workflow dynamique — 187 appels d'outils, ~478k tokens, ~21 min. Thread principal : coordination, intégration, consolidation (règle Agent teams, CLAUDE.md §11).*
