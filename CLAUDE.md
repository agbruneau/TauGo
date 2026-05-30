# CLAUDE.md — TauGo

Kernel Go de l'opérateur τ. Pont théorie (`agbruneau/InteroperabiliteAgentique` v2.4.3, chap. III.8) ↔ empirie (`agbruneau/AgentMeshKafka`) ↔ ingénierie (`agbruneau/FibGo`). Spec complète : [`PRD.md`](PRD.md). Ce fichier ne contient que les règles opérationnelles ; tout détail théorique, modèle de données ou pseudo-algorithme renvoie au PRD.

> **État v0.1.2-pre** *(2026-05-24)*. M0-M6 clos sous tag `v0.1.0` ; refactor v0.1.1 (consolidation post-audit, 4 ADRs, packages `internal/{thresholds,errors,testutil}` peuplés, `Trace` ventilée, anti-patron #6 gardé) livré sous commit `2cf560c`. **v0.1.2 retire tout l'outillage CI/CD** (workflows GitHub Actions + cibles Make CI-only + références doc) — projet *pure-local* (cf. ADR-0010). Source de vérité du refactor v0.1.1 (archivée) : [`docs/archive/audits/2026-05-24-AUDIT-v0.1.0-to-v0.1.1.md`](docs/archive/audits/2026-05-24-AUDIT-v0.1.0-to-v0.1.1.md) et [`docs/archive/audits/2026-05-24-AUDITPlan-v0.1.1.md`](docs/archive/audits/2026-05-24-AUDITPlan-v0.1.1.md).

---

## Projet

- **Module** : `github.com/agbruneau/taugo` *(Confirmé `go.mod`)*
- **Go** : 1.25+ (toolchain 1.26.x) — aligné FibGo
- **Licence** : Apache-2.0
- **Lint** : `golangci-lint v1.64.8` épinglé, 24 linters (calque FibGo)
- **Validation** : locale uniquement depuis v0.1.2 (ADR-0010 — retrait CI/CD). Cible : `make test && make lint && make fuzz`. Objectifs (vérifiables via `make coverage`) : `internal/tau/*` ≥ 90 %, global ≥ 80 %. `-race` requiert CGO (Linux/macOS).
- **Référence canonique** : `InteroperabiliteAgentique` v2.4.3 (2026-05-21), chap. III.8

---

## Doctrine

**TauGo EST** : un kernel qui décide d'un *régime d'appel* (`Deterministe | Probabiliste | Refus`) à la frontière agentique, sous les cinq invariants I1-I5 du chap. III.8.5. Sortie unique : `Kernel.Decide(ctx, Exchange) → Decision`.

**TauGo N'EST PAS** : un framework agentique · un orchestrateur · un wrapper LLM · un service réseau · un RAG · un prédicteur de comportement.

`τ : t_fix(g) ≺ t_int ↦ t_fix(g) ≈ t_int` — déplace l'instant de fixation, jamais le contenu. Détail [`PRD.md` §2, §4](PRD.md).

Toute PR qui érode les frontières ci-dessus exige mise à jour explicite du [`PRD.md` §3.3](PRD.md) — sinon rejet.

---

## Agent teams — règle d'exploitation obligatoire

**Toute planification et toute exécution de tâche passent par des sous-agents.** Le thread principal **coordonne, dispatche, intègre, valide** — il **n'implémente pas directement**. Cette règle est non négociable.

### Cartographie agent → rôle

| Agent | Rôle TauGo | Quand l'invoquer |
|---|---|---|
| `Plan` | Architecte logiciel | Avant chaque milestone : raffiner le sous-plan. Toute décision d'architecture non triviale. |
| `ruflo-swarm:architect` | Architecte système | Design des interfaces et contrats inter-couches. ADR avant changement structurel. |
| `ruflo-swarm:coordinator` | Coordinateur swarm | Quand ≥ 3 agents tournent en parallèle sur tâches indépendantes. |
| `Explore` | Recherche read-only | Localiser patterns FibGo. Rechercher symboles dans la monographie. |
| `ruflo-core:researcher` | Pathfinder théorie ↔ code | Vérifier l'alignement chap. III.8 ↔ Go. Rédiger ADRs et `docs/theory/`. |
| `ruflo-core:coder` | Implémentation TDD | Écriture du code Go conforme aux conventions §Conventions de code. |
| `ruflo-core:reviewer` | Revue de code | Gate avant merge : conformité invariants, anti-patrons, étanchéité Clean Arch. |
| `understand-anything:project-scanner` | Inventaire repo | Avant audit final ou release majeure. |
| `understand-anything:architecture-analyzer` | Analyse couches | Vérifier que les couches livrées correspondent à PRD §8. |
| `general-purpose` | Tâches ouvertes | Recherche comparative inter-projets, exploration ambiguë, audit code complet. |

### Pattern d'exécution par tâche

```
1. RECHERCHE (parallèle si possible)
   ├─ Explore  → patterns FibGo, code de référence
   └─ ruflo-core:researcher → alignement théorie ↔ implémentation

2. ARCHITECTURE (si tâche non triviale)
   └─ Plan ou ruflo-swarm:architect → décomposition bite-sized + ADR

3. IMPLÉMENTATION (parallèle pour tâches indépendantes)
   └─ ruflo-core:coder × N

4. REVUE
   └─ ruflo-core:reviewer

5. INTÉGRATION (thread principal)
   → `make test && make lint && make fuzz` verts en local → commit conventionnel signé → tag si milestone
```

### Règles d'orchestration

1. **Le thread principal ne code pas.** Il dispatche, lit les diffs produits, intègre.
2. **Parallélisme par défaut** quand tâches indépendantes. Invocations multiples dans **un seul message** pour exécution concurrente.
3. **Sérialisation imposée** pour : commits, tags, intégration finale, décisions ADR.
4. **Briefing autoportant** : chaque dispatch contient le contexte complet — pas de référence implicite à la conversation principale.
5. **Vérification post-agent** : le coordinateur **lit le diff réel** avant de relancer. Pas de confiance aveugle au rapport d'un agent.
6. **Coordination écriture** : si plusieurs agents touchent le même fichier, sérialiser ou batch (cas typique : `dispatcher.go`, `operator.go`).

Détail complet de la stratégie : [`docs/archive/PRDPlanning-m0-m6.md` §A](docs/archive/PRDPlanning-m0-m6.md) *(plan-cadre M0-M6 archivé après clôture v0.1.0)*.

---

## Anti-patrons interdits (7 — tous gardés par test depuis v0.1.1)

| # | Anti-patron | Garde |
|---|---|---|
| 1 | Méthode `Predict*` / `Expected*` / `Forecast*` exportée | `TestNoPredictiveAPI` |
| 2 | Bypass de `FrontierCheck.Inside()` *(4 conditions classiques toutes violées)* | `TestFrontierCheck_Inside_*` |
| 3 | Profil de calibration périmé toléré (`today > date_revision`) | `TestExpiredProfileRefuses`, `TestI3_DateRevisionRespectee`, `TestApp_NewDispatcher_*` *(v0.1.1 : `app.NewDispatcher()` charge `calibration.DefaultProfile()`, activant la garde sur le chemin CLI standard)* |
| 4 | Observation non modélisée passée sous silence | `TestUnmodeledObservationsReported` |
| 5 | Citation/chiffre/API/DOI fabriqué dans `docs/` | Audit + PR sans marqueur d'incertitude sur affirmation datée → reject |
| 6 | Import LLM concret (`anthropic`, `openai`, …) dans `internal/tau/*` ou `internal/orchestration/*` | `TestArchNoConcreteLLMInDomain` *(walk AST sur 12 substrings interdites — actif depuis v0.1.1)* |
| 7 | Globaux mutables non synchronisés dans `internal/tau/*` | `gochecknoglobals` + revue PR *(v0.1.1 : `I3PerimptionLimite` converti en getter)* |

Détail et raisonnement dans [`PRD.md` §7.2](PRD.md) et chap. III.8.7.

---

## Architecture (Clean Architecture, 4 couches strictes)

```
cmd/{tau, generate-corpus}/
internal/
  app/                 # lifecycle, injection LLM concret, app.NewDispatcher
  tau/                 # CŒUR — n'importe pas orchestration/, bridge/
    {operator,frontier,diagnostics}.go
    dimensions/{dsens,dauthority,dinvariant,score}.go
    invariants/{i1..i5,evaluator}.go + fuzz_targets_test.go
  orchestration/       # dispatcher (8 étapes), Thresholds alias, Decision/Trace
  calibration/         # Profile, drift, atomic thresholds, DefaultProfile, Validate
  bridge/{agentmeshkafka, llm}/
  thresholds/          # Type valeur transverse (ADR-0006)
  errors/              # DispatchError, RefusError, CalibrationError, sentinels (ADR-0009)
  testutil/            # BuildExchange + Option(...) helpers
docs/{theory, algorithms, adr, empirical, archive/plans-m0-m6}/
test/e2e/
tests/calibration/golden-corpus.jsonl   # golden corpus réel (répertoire tests/, distinct de test/)
```

**Étanchéité gardée par `internal/arch_test.go`** *(7 règles depuis v0.1.1)* :

- `tau/* → orchestration` · `tau/* → bridge` · `bridge → tau/*` direct : interdits
- `dimensions ↔ invariants` : interdit (orthogonalité I1-I5 vs 3 dimensions encodée)
- LLM concret hors `app/` et `bridge/llm/` : interdit *(`TestArchNoConcreteLLMInDomain`)*
- `calibration → tau/*` `orchestration` `bridge/*` : interdit *(v0.1.1 nouvelle règle V-A2)*
- `thresholds` → aucun package taugo : étanchéité descendante (ADR-0006)

Détail [`PRD.md` §8](PRD.md) et [`docs/adr/0006-types-valeur-transverses.md`](docs/adr/0006-types-valeur-transverses.md).

---

## Invariants & dimensions — résumé exécutable

*Verbatim théorique : `InteroperabiliteAgentique/Monographie.md` chap. III.8.5. Reformulation exécutable et conditions de réfutation : [`PRD.md` §6](PRD.md).*

| # | Énoncé court | Statut | Cible fuzz | Débit *(fonction-propriété scalaire ; [À vérifier])* |
|---|---|---|---|---|
| I1 | τ conserve la grandeur (déplace `t_fix`, pas le contenu) | Probable | `FuzzI1_Conservation` | ~8.6 M exec/s |
| I2 | Résidu migrant non vide, non recâblable hors ligne | Confirmé | `FuzzI2_Irreductibilite` | ~8.6 M exec/s |
| I3 | D-AUTORITÉ asymétrique (fait institutionnel — Searle 1995) ; sans `AttestationInstitutionnelle` → refus ontologique. **Veille trimestrielle ; daté 2026-05-16.** | Probable | `FuzzI3_AsymetrieAutorite` | ~8.2 M exec/s |
| I4 | `i ≈ pendant ⟹ s ≈ pendant` ; configuration incohérente → refus | Hypothèse | `FuzzI4_CoherenceContrainte` | ~9.5 M exec/s |
| I5 | Pile composée hérite de la conjonction ; `M(π) ≥ max(\|Aᵢ\|)` | Probable | `FuzzI5_CompositionConjonctive` | ~1.1 M exec/s *(v0.1.1 : `BoundsHold` optim -46 % ns/op)* |

> **Méthodologie débit** *(audit 2026-05-29, P4-01/P4-07)*. La colonne **Débit** ci-dessus mesure le **débit de la fonction-propriété scalaire isolée** (~8.2–9.5 M exec/s I1-I4, hôte historique) — pas le débit du moteur de fuzzing. Le **débit moteur** `go test -fuzz` mesuré sur ce poste (Windows, `CGO_ENABLED=0`, Go 1.26.3) est de **~1,4 M exec/s** (I1-I4) et **~1,1 M/s** (I5, confirmé). 0 crash sur ~200 M exécutions cumulées (30 s/cible).

**Trois dimensions** *(détail [`PRD.md` §5](PRD.md))* : `D-SENS` [0,1] (lieu de fixation du sens) · `D-AUTORITÉ` [0,1] (portée de la chaîne de délégation) · `D-INVARIANT` [0,1] (support des invariants d'intégration).

**Trace ventilée** *(v0.1.1, ADR-0008)* : `Decision.Trace.{DSens, DAuthority, DInvariant} *tau.Score` peuplés par le dispatcher aux étapes 2 et 4. `EvaluateI3`/`EvaluateI4` lisent désormais ces champs au lieu du proxy `TauScore`.

---

## Refus — décision de premier rang

`Decide` retourne `Refus(diag, trace)` dans cinq cas :

1. **Hors frontière τ** (≥ 1 des 4 conditions classiques tenue)
2. **Verrou ontologique D-AUTORITÉ** (`score ≥ θ_auth_block` sans attestation)
3. **Profil périmé** (`today > date_revision`) — actif sur chemin CLI par défaut depuis v0.1.1
4. **Incohérence I4** (`s < θ_sens ∧ i ≥ θ_inv`)
5. **Observation non modélisée à fort impact** (rapportée dans `Trace.UnmodeledObservations`, anti-patron §7.2 #4)

Diagnostics canoniques : constantes `tau.DiagFrontiereFranchie`, `DiagVerrouOntologique`, `DiagPeremptionProfile`, `DiagIncoherenceI4` *(v0.1.1 — résout duplication littéraux)*.

Refus n'est pas un échec : c'est une décision pleine, instrumentée, opposable. Détail [`PRD.md` §7.3](PRD.md).

---

## Commandes essentielles

```bash
make all                 # build + test
make test                # go test -v -race -cover ./...  (CGO requis pour -race)
make test-short          # go test -v -short ./...
make coverage            # HTML — objectifs locaux ≥ 80 % global, ≥ 90 % sur tau/*
make benchmark           # go test -bench=. -benchmem ./internal/tau/...
make lint                # golangci-lint run ./...
make fuzz                # -fuzztime=30s sur I1-I5
make calibrate           # tau calibrate --corpus … --output …
make build               # -trimpath -buildvcs=true ./cmd/tau
make build-pgo           # PGO (profil checked-in après M3)
make build-all           # cross linux/darwin/windows × amd64/arm64
```

Cibles retirées en v0.1.2 (ADR-0010) — utiliser `go test` direct :

```bash
go test -fuzz=. -fuzztime=24h ./internal/tau/invariants/                        # fuzz long
go test -tags=integration ./test/e2e/...                                        # E2E intégration
go test -tags=e2e ./test/e2e/... -run="TestCalibration|TestCalibrate|TestExpiredProfileRefuses"   # E2E calibration
go test -tags=empirical ./internal/bridge/agentmeshkafka/... -run=^TestEmpiricalI4Campaign$$ -count=1   # campagne I4
# build-reproducible : retiré (timestamp gelé CI-only)
```

`-race` exige CGO (gcc) — disponible Linux/macOS ; sous Windows sans gcc, utiliser `make test-short` ou `go test -short ./...`.

---

## Conventions éditoriales

*Condensé `InteroperabiliteAgentique/CLAUDE.md` §1.1-§1.8.*

- **FR-CA** pour `PRD.md`, `CLAUDE.md`, `docs/`, commentaires structurants — **godoc en anglais**.
- **Typographie française** : espaces insécables U+00A0 avant `: ; ? ! »` et après `«` ; guillemets `« … »`. Cible v0.1.0 atteinte ; commentaires structurants `.go` couverts depuis v0.1.1.
- **Marqueurs d'incertitude obligatoires** sur toute affirmation datée ou évolutive : `Confirmé` · `Probable` · `Hypothèse` · `À vérifier` · `Je ne sais pas (avec piste)`.
- **Zéro fabrication.** Aucune citation, version, API, date, DOI, nom propre inventés. Une fabrication détectée invalide le livrable concerné, sans appel.
- **Renvois croisés monographie** : toute décision théorique cite `*(chap. III.8.X.Y)*` en italique.
- **Patrons de raisonnement** : recommandation = (1) compromis principal · (2) ≥ 1 alternative crédible · (3) conditions de retournement.
- **Anonymisation** : aucun cas Desjardins identifiable ; références publiques nommées librement.
- **Pas d'emoji** sauf demande explicite.
- **Pas de BOM UTF-8** dans les fichiers source : un BOM en milieu de payload bloque `go test -coverpkg`. Strip via `python -c "..."` ou éditeur configuré « UTF-8 sans BOM ».

---

## Conventions de code (calque FibGo)

- Packages par responsabilité, jamais par feature.
- Interfaces étroites (ISP), ≤ 5 méthodes publiques.
- Erreurs typées : `*errors.DispatchError`, `*errors.RefusError`, `*errors.CalibrationError` (package `internal/errors/`, ADR-0009) — `fmt.Errorf("%w", err)` pour wrap. **Pas de panic** sauf invariant interne cassé (sentinel re-propagé, calque `bigfft/fermat.go`).
- Sentinels d'erreur : `errors.Is`/`errors.As`-compatibles (`ErrFrontiereFranchie`, `ErrPeremptionProfile`, `ErrIncoherenceI4`, `ErrVerrouOntologique`).
- `t.Parallel()` systématique (cible 100 % atteinte v0.1.0).
- Complexité max : cyclomatique 15, cognitive 30 ; fonction ≤ 100 LOC / 50 statements. Exception documentée par `//nolint:gocognit` avec raison.
- `doc.go` par package public — peut être fusionné dans le fichier principal si le package contient ≥ 1 fichier non-vide.
- Commentaires : *pourquoi*, jamais *quoi*. Pas de référence au caller ni à la tâche courante.
- **Pas de globaux mutables non synchronisés** dans `internal/tau/*`. Lookup tables immutables `//nolint:gochecknoglobals` admis avec justification (ex. `regimeStrings`, `discoveryModeStrings`).
- **JSON enums** : `MarshalJSON`/`UnmarshalJSON` retournent une string PascalCase ; `UnmarshalJSON` accepte aussi l'int legacy pour rétro-compat corpus v0.1.0.
- Helpers de test : préférer `testutil.BuildExchange(opts...)` aux constructions ad-hoc.

**Commits — Conventional Commits** : `<type>(<scope>): <description>` avec types `feat · fix · perf · refactor · test · docs · chore · theory` (`theory` = mise à jour `docs/theory/` motivée par révision monographie).

**Co-signature IA** :

```
Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

---

## Directives projet

> Les guidelines `~/.claude/CLAUDE.md` (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution) s'appliquent **en plus**.

1. **Anti-platform discipline** — voir Doctrine. PR qui érode les anti-objectifs : reject ou mise à jour explicite du PRD §3.3.
2. **Frontière non négociable** — aucun bypass de `FrontierCheck.Inside()`. Drapeau « skip » = reject. La méthode canonique est `x.FrontierCheck()` (`internal/tau/operator.go`).
3. **Étanchéité Clean Architecture** — gardée par `arch_test.go` (7 règles). Toute violation = test rouge.
4. **Stub LLM par défaut** — tout test sans `TAUGO_LLM_BACKEND=real` utilise le stub déterministe. Le code de production n'appelle jamais de service LLM externe sans configuration explicite.
5. **Performance critique** — modifs dans `tau/*` ou `calibration/*` : `make benchmark` avant + après. Régression > 5 % = blocage. *(Note audit 2026-05-29, P4-02 : les benchmarks `Decide`/calibration de bout en bout manquent encore ; la garde 5 % ne couvre aujourd'hui que les micro-benchmarks `internal/tau/...`.)*
6. **Golden tests immuables** (V1.1+) — pas de flag `-update` checked-in ; modification = ADR.
7. **Modifications chirurgicales** — diff minimal. Refactor > 50 LOC sur > 2 fichiers : motiver dans le commit. Bug touché en passant = `fix(...)` isolé AVANT de poursuivre.
8. **Veille active I3** — profil porte `date_revision` ; périmé → `Refus`. `app.NewDispatcher()` injecte `calibration.DefaultProfile()` pour activer cette garde par défaut (v0.1.1, P0-02). **v0.1.2** : l'alerte automatisée 30 jours avant péremption (qui passait par CI) bascule en check manuel ou cron externe (cf. ADR-0010).
9. **Renvois croisés** — toute décision dans `docs/theory/` cite `chap. III.8.X.Y` ; lint manuel à chaque clôture de milestone.
10. **Validation locale obligatoire** *(v0.1.2 — CI retirée, ADR-0010)* — `make test && make lint && make fuzz` en local avant tout commit. Gate per-package ≥ 90 % `tau/*` reste un *objectif vérifiable* via `make coverage` (HTML rapport), mais n'est plus automatisé. Race detector requiert CGO (Linux/macOS) ; sous Windows utiliser `go test -short ./...`.
11. **Agent teams obligatoires** — toute planification et toute exécution passent par sous-agents *(cf. §Agent teams)*. Le thread principal coordonne, dispatche, intègre. Plan-cadre M0-M6 archivé : [`docs/archive/PRDPlanning-m0-m6.md`](docs/archive/PRDPlanning-m0-m6.md) ; refactor v0.1.1 : [`docs/archive/audits/2026-05-24-AUDITPlan-v0.1.1.md`](docs/archive/audits/2026-05-24-AUDITPlan-v0.1.1.md).

---

## Workflow

```
1. Branche : <type>/<description-courte>
2. Test rouge → fix minimal → vert + golden (+ benchmark si perf-sensitive)
3. go test ./<pkg>/... -count=1 -race && go vet ./... && golangci-lint run ./<pkg>/...
4. Si invariant touché : go test -fuzz=FuzzI<N>_... -fuzztime=30s ./internal/tau/invariants/
5. Commit conventionnel + co-signature
6. PR, review, merge
```

**Modification d'invariant ou de dimension** *(workflow strict)* :

```
ADR docs/adr/NNNN-<motif>.md → MAJ PRD §5 ou §6 → MAJ docs/theory/04|05 →
  implémentation → extension fuzz → vérif renvois chap. III.8 → commit theory(...)
```

---

## Modules sensibles (v0.1.1)

| Fichier | Raison |
|---|---|
| `internal/tau/operator.go` | Point d'entrée public unique ; types `Exchange`, `Decision`, `Trace`, `Score`, `Regime`, `DiscoveryMode`. Évolution = rupture API potentielle. |
| `internal/tau/frontier.go` | Garde de premier rang ; bypass interdit (anti-patron #2). |
| `internal/tau/diagnostics.go` | Constantes `Diag*` canoniques — toute nouvelle valeur de diagnostic doit y être ajoutée. |
| `internal/tau/invariants/i3_authority_asymmetry.go` | Encode Searle 1995 ; refus ontologique non contournable. `I3PerimptionLimite()` getter (date 2027-01-01). |
| `internal/tau/invariants/i4_coherence.go` | Détecte la rupture silencieuse via scores ventilés (chap. III.7). |
| `internal/orchestration/dispatcher.go` | Ordre des 8 étapes du pseudo-algo non arbitraire — détail [`PRD.md` §10](PRD.md). Lit `Profile.Weights` à l'étape 6 (v0.1.1). |
| `internal/calibration/profile.go` | Sérialisation reproductible byte-identique ; migration de schéma = ADR. `DefaultProfile()` injecté par `app.NewDispatcher`. |
| `internal/calibration/drift.go` | Un drift non détecté = profil silencieusement périmé (anti-patron #3). |

---

## Dette technique résolue *(audit 2026-05-29)*

- **C1-01 — golden corpus de calibration au mauvais schéma** *(RÉSOLU 2026-05-29, [ADR-0012](docs/adr/0012-golden-corpus-calibration-schema.md) ; runtime `Decide` n’a jamais été affecté — `DefaultProfile()`)*. `tests/calibration/golden-corpus.jsonl` était en schéma `Exchange` (`expected_regime` PascalCase) et non `CorpusEntry` → `tau calibrate` produisait un profil dégénéré (no-op depuis M5).
  - **Résolution livrée** : corpus régénéré au schéma `CorpusEntry` via `cmd/generate-corpus --scored` (**170 lignes**, scores ventilés réels, `probabiliste` 90 / `deterministe` 50 / `refus_authority` 30 ; `refus_i4` 0 — limitation I4 connue). Profil non dégénéré (seuils Det 0,45 / Prob 0,65 / Auth 0,70 / Sens 0,30 / Inv 0,30 ; plancher antérieur 0,10/0,15). Hash re-épinglé `goldenCorpusCanonicalHash = 8e5dc2fc…caa4`. Validation CLI rétablie : `cmd/tau/calibrate.go:loadCorpus` délègue à `calibration.LoadCorpus` (migration + `Validate`) ; corpus invalide → exit 2. Détail : [`PRD.md` §20.4](PRD.md), [ADR-0012](docs/adr/0012-golden-corpus-calibration-schema.md), audit [`01_conformite_tau.md`](docs/archive/audits/2026-05-29-AUDIT-v0.1.2-pre/01_conformite_tau.md).

---

## Références

- [`PRD.md`](PRD.md) — spec canonique V0.3
- [`docs/archive/PRDPlanning-m0-m6.md`](docs/archive/PRDPlanning-m0-m6.md) — plan d'exécution M0-M6 (archivé après clôture v0.1.0)
- [`docs/archive/audits/2026-05-24-AUDIT-v0.1.0-to-v0.1.1.md`](docs/archive/audits/2026-05-24-AUDIT-v0.1.0-to-v0.1.1.md) — audit consolidé v0.1.0 → v0.1.1 (2026-05-24)
- [`docs/archive/audits/2026-05-24-AUDITPlan-v0.1.1.md`](docs/archive/audits/2026-05-24-AUDITPlan-v0.1.1.md) — plan refactor 42 tâches T-001..T-040
- [`docs/archive/audits/2026-05-29-AUDIT-v0.1.2-pre/RAPPORT_FINAL.md`](docs/archive/audits/2026-05-29-AUDIT-v0.1.2-pre/RAPPORT_FINAL.md) — audit de régression v0.1.2-pre (2026-05-29 ; 6 axes 01..06, 00_bootstrap, CONVENTIONS) : 0 critique, 10 majeur, 16 mineur, 15 informatif ; verdict — kernel sain, fragilités épistémiques/documentaires corrigées dans ce lot
- [`docs/archive/README.md`](docs/archive/README.md) — index des documents archivés
- [`CHANGELOG.md`](CHANGELOG.md) — historique Keep-a-Changelog
- **Monographie** : `agbruneau/InteroperabiliteAgentique` v2.4.3 (2026-05-21), chap. III.8 — épinglée dans chaque `Profile`
- **Ingénierie** : `agbruneau/FibGo` — `Claude.md`, `.golangci.yml`, `arch_test.go`, `internal/calibration/`
- **Empirie** : `agbruneau/AgentMeshKafka` — bridge en `internal/bridge/agentmeshkafka/` (M4)
- **HGL** (V2+) : `InteroperabiliteAgentique/RechercheFondamentale.md` — mécanisation Lean en dépôt compagnon (ADR-0011 à créer ; l'ADR-0010 a été allouée au retrait CI/CD v0.1.2)
- `docs/theory/` — renvois systématiques chap. III.8.*
- `docs/adr/` — ADRs 0001-0010, 0012 *(0001 Clean Arch · 0002 Go 1.25 · 0003 LLM injection · 0004 AgentMeshKafka · 0005 DTO neutre · 0006 thresholds transverses · 0007 hystérèse V1 · 0008 Trace ventilée · 0009 erreurs typées · 0010 retrait CI/CD · 0012 golden corpus CorpusEntry)*
- `docs/archive/plans-m0-m6/` — plans détaillés M0-M6 archivés v0.1.1

---

*CLAUDE.md V0.5 — 2026-05-24. Alignement v0.1.2 : retrait outillage CI/CD complet (ADR-0010), réécriture §Projet "Validation", §Commandes essentielles, directives 8 et 10, références ADR-0010. Précédent V0.4 : alignement post-refactor v0.1.1 (anti-patrons #6, packages thresholds/errors, Trace ventilée). Document vivant : déviation matérielle = mise à jour de ce fichier ET du `PRD.md`, AVANT le code.*

*2026-05-29 — alignement post-audit de régression v0.1.2-pre (survente couverture/débits corrigée, arborescence resynchronisée, spec canonique V0.3, audit indexé en §Références).*
