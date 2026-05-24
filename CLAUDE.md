# CLAUDE.md — TauGo

Kernel Go de l'opérateur τ. Pont théorie (`agbruneau/InteroperabiliteAgentique` v2.4.3, chap. III.8) ↔ empirie (`agbruneau/AgentMeshKafka`) ↔ ingénierie (`agbruneau/FibGo`). Spec complète : [`PRD.md`](PRD.md). Ce fichier ne contient que les règles opérationnelles ; tout détail théorique, modèle de données ou pseudo-algorithme renvoie au PRD.

> **État V0.1 — pré-implémentation.** Aucun code committé avant M0. Les règles ci-dessous gouvernent toute écriture à partir du premier commit signé `v0.0.1-alpha`.

---

## Projet

- **Module** : `github.com/agbruneau/taugo` *(à confirmer M0)*
- **Go** : 1.25+ (toolchain 1.26.x) — aligné FibGo
- **Licence** : Apache-2.0
- **Lint** : `golangci-lint v1.64.8` épinglé, config calque FibGo (24 linters)
- **CI** : matrice 3 OS (Linux/macOS/Windows), `-race` via CGO sur Linux/macOS, fuzz court 30 s sur I1-I5
- **Référence canonique** : `InteroperabiliteAgentique` v2.4.3 (2026-05-21), chap. III.8

---

## Doctrine

**TauGo EST** : un kernel qui décide d'un *régime d'appel* (`Deterministe | Probabiliste | Refus`) à la frontière agentique, sous les cinq invariants I1-I5 du chap. III.8.5. Sortie unique : `Kernel.Decide(ctx, Exchange) → Decision`.

**TauGo N'EST PAS** : un framework agentique · un orchestrateur · un wrapper LLM · un service réseau · un RAG · un prédicteur de comportement.

`τ : t_fix(g) ≺ t_int ↦ t_fix(g) ≈ t_int` — déplace l'instant de fixation, jamais le contenu. Détail [`PRD.md` §2, §4](PRD.md).

Toute PR qui érode les frontières ci-dessus exige mise à jour explicite du [`PRD.md` §3.3](PRD.md) — sinon rejet.

---

## Agent teams — règle d'exploitation obligatoire

**Toute planification et toute exécution de tâche passent par des sous-agents.** Le thread principal **coordonne, dispatche, intègre, valide** — il **n'implémente pas directement**. Cette règle est non négociable ; elle s'applique aux 7 milestones M0-M6 du [`PRDPlanning.md`](PRDPlanning.md) et à toute évolution ultérieure.

### Cartographie agent → rôle

| Agent | Rôle TauGo | Quand l'invoquer |
|---|---|---|
| `Plan` | Architecte logiciel | Avant chaque milestone : raffiner le sous-plan détaillé. Toute décision d'architecture non triviale. |
| `ruflo-swarm:architect` | Architecte système | Design des interfaces et contrats inter-couches. ADR avant changement structurel. |
| `ruflo-swarm:coordinator` | Coordinateur swarm | Quand ≥ 3 agents tournent en parallèle sur tâches indépendantes. |
| `Explore` | Recherche read-only | Localiser patterns FibGo à calquer. Rechercher symboles dans la monographie. |
| `ruflo-core:researcher` | Pathfinder théorie ↔ code | Vérifier l'alignement chap. III.8 ↔ Go. Récupérer verbatim d'invariant. Rédiger `docs/theory/`. |
| `ruflo-core:coder` | Implémentation TDD | Écriture du code Go conforme aux conventions §Conventions de code. |
| `ruflo-core:reviewer` | Revue de code | Gate avant merge : conformité invariants, anti-patrons, étanchéité Clean Arch. |
| `understand-anything:project-scanner` | Inventaire repo | Avant M6 : rapport d'audit final. |
| `understand-anything:architecture-analyzer` | Analyse couches | Vérifier que les couches livrées correspondent à PRD §8. |
| `general-purpose` | Tâches ouvertes | Recherche comparative inter-projets, exploration ambiguë. |

### Pattern d'exécution par tâche

```
1. RECHERCHE (parallèle si possible)
   ├─ Explore  → patterns FibGo, code de référence
   └─ ruflo-core:researcher → alignement théorie ↔ implémentation

2. ARCHITECTURE (si tâche non triviale)
   └─ Plan ou ruflo-swarm:architect → décomposition bite-sized

3. IMPLÉMENTATION (parallèle pour tâches indépendantes)
   └─ ruflo-core:coder × N

4. REVUE
   └─ ruflo-core:reviewer

5. INTÉGRATION (thread principal)
   → tests CI verts → commit conventionnel signé → tag si milestone
```

### Règles d'orchestration

1. **Le thread principal ne code pas.** Il dispatche, lit les diffs produits, intègre.
2. **Parallélisme par défaut** quand tâches indépendantes (recherche vs implémentation, plusieurs sondes, etc.). Invocations multiples dans **un seul message** pour exécution concurrente.
3. **Sérialisation imposée** pour : commits, tags, intégration finale, décisions ADR.
4. **Briefing autoportant** : chaque dispatch d'agent contient le contexte complet — pas de référence implicite à la conversation principale. L'agent ne voit rien d'autre que son prompt.
5. **Vérification post-agent** : après chaque retour, le coordinateur **lit le diff réel** avant de relancer. Ne pas faire confiance aveuglément au rapport d'un agent.
6. **Choix d'exécution** : `superpowers:subagent-driven-development` (recommandé) pour M0-M3 ; `superpowers:executing-plans` (inline avec checkpoints) pour M4-M6 si continuité contextuelle requise.

Détail complet de la stratégie : [`PRDPlanning.md` §A](PRDPlanning.md).

---

## Anti-patrons interdits (gardés par test)

| # | Anti-patron | Garde |
|---|---|---|
| 1 | Méthode `Predict*` / `Expected*` / `Forecast*` exportée | `TestNoPredictiveAPI` |
| 2 | Bypass de `FrontierCheck.Inside()` *(4 conditions classiques toutes violées)* | `TestRefusHorsFrontiere` |
| 3 | Profil de calibration périmé toléré (`today > date_revision`) | `TestExpiredProfileRefuses`, `TestI3_DateRevisionRespectee` |
| 4 | Observation non modélisée passée sous silence | `TestUnmodeledObservationsReported` |
| 5 | Citation/chiffre/API/DOI fabriqué dans `docs/` | Audit + PR sans marqueur d'incertitude sur affirmation datée → reject |
| 6 | Import LLM concret (`anthropic`, `openai`, …) dans `internal/tau/*` ou `internal/orchestration/*` | `TestArchNoConcreteLLMInDomain` |
| 7 | Globaux mutables non synchronisés dans `internal/tau/*` | `gochecknoglobals` + revue PR |

Détail et raisonnement dans [`PRD.md` §7.2](PRD.md) et chap. III.8.7.

---

## Architecture (Clean Architecture, 4 couches strictes)

```
cmd/{tau, generate-golden}/
internal/
  app/                 # lifecycle, injection LLM concret
  tau/                 # CŒUR — n'importe pas orchestration/, bridge/
    {operator,frontier}.go
    dimensions/{dsens,dauthority,dinvariant}.go + probes/
    invariants/{i1..i5}.go + fuzz_targets.go
  orchestration/       # dispatcher, Decision, Trace (immuables)
  calibration/         # Profile, drift, thresholds (atomic.Int64, calque FibGo)
  bridge/{agentmeshkafka, llm}/
  {config, errors, metrics, testutil}/
docs/{theory, algorithms, adr, empirical}/
test/{e2e, golden}/
```

**Étanchéité gardée par `internal/arch_test.go`** :

- `tau/* → orchestration` · `tau/* → bridge` · `bridge → tau/*` direct : interdits
- `dimensions ↔ invariants` : interdit (orthogonalité I1-I5 vs 3 dimensions encodée)
- LLM concret hors `app/` et `bridge/llm/` : interdit

Détail [`PRD.md` §8](PRD.md).

---

## Invariants & dimensions — résumé exécutable

*Verbatim théorique : `InteroperabiliteAgentique/Monographie.md` chap. III.8.5. Reformulation exécutable et conditions de réfutation : [`PRD.md` §6](PRD.md).*

| # | Énoncé court | Statut | Cible fuzz |
|---|---|---|---|
| I1 | τ conserve la grandeur (déplace `t_fix`, pas le contenu) | Probable | `FuzzI1_Conservation` |
| I2 | Résidu migrant non vide, non recâblable hors ligne | Confirmé | `FuzzI2_Irreductibilite` |
| I3 | D-AUTORITÉ asymétrique (fait institutionnel — Searle 1995) ; sans `AttestationInstitutionnelle` → refus ontologique. **Veille trimestrielle ; daté 2026-05-16.** | Probable | `FuzzI3_AsymetrieAutorite` |
| I4 | `i ≈ pendant ⟹ s ≈ pendant` ; configuration incohérente → refus | Hypothèse | `FuzzI4_CoherenceContrainte` |
| I5 | Pile composée hérite de la conjonction ; `M(π) ≥ max(|Aᵢ|)` | Probable | `FuzzI5_CompositionConjonctive` |

**Trois dimensions** *(détail [`PRD.md` §5](PRD.md))* : `D-SENS` [0,1] (lieu de fixation du sens) · `D-AUTORITÉ` [0,1] (portée de la chaîne de délégation) · `D-INVARIANT` [0,1] (support des invariants d'intégration).

---

## Refus — décision de premier rang

`Decide` retourne `Refus(diag, trace)` dans cinq cas :

1. **Hors frontière τ** (≥ 1 des 4 conditions classiques tenue)
2. **Verrou ontologique D-AUTORITÉ** (`score ≥ θ_auth_block` sans attestation)
3. **Incohérence I4** (`s < θ_sens ∧ i ≥ θ_inv`)
4. **Profil périmé** (`today > date_revision`)
5. **Observation non modélisée à fort impact** (anti-patron §7.2.4)

Refus n'est pas un échec : c'est une décision pleine, instrumentée, opposable. Détail [`PRD.md` §7.3](PRD.md).

---

## Commandes essentielles (Makefile cible M0)

```bash
make all                 # build + test
make test                # go test -v -race -cover ./...  (CGO requis pour -race)
make test-short          # go test -v -short ./...
make coverage            # HTML, gate ≥ 80% global, ≥ 90% sur tau/*
make benchmark           # go test -bench=. -benchmem ./internal/tau/...
make lint                # golangci-lint run ./...
make fuzz                # -fuzztime=30s sur I1-I5
make fuzz-long           # -fuzztime=24h (CI nocturne)
make calibrate           # tau calibrate --corpus … --output …
make build               # -trimpath ./cmd/tau
make build-reproducible  # timestamp gelé, vérif byte-identique
make build-pgo           # PGO (profil checked-in après M3)
make build-all           # cross linux/darwin/windows × amd64/arm64
```

`-race` exige CGO (gcc) — indisponible Windows sans gcc ; la CI Linux/macOS le couvre. Sans `make` : équivalents `go` directs.

---

## Conventions éditoriales

*Condensé `InteroperabiliteAgentique/CLAUDE.md` §1.1-§1.8.*

- **FR-CA** pour `PRD.md`, `CLAUDE.md`, `docs/`, commentaires structurants — **godoc en anglais**.
- **Typographie française** : espaces insécables U+00A0 avant `: ; ? ! »` et après `«` ; guillemets `« … »`. **Cible M6** ; M0-M5 acceptent espaces ordinaires.
- **Marqueurs d'incertitude obligatoires** sur toute affirmation datée ou évolutive : `Confirmé` · `Probable` · `Hypothèse` · `À vérifier` · `Je ne sais pas (avec piste)`.
- **Zéro fabrication.** Aucune citation, version, API, date, DOI, nom propre inventés. Une fabrication détectée invalide le livrable concerné, sans appel.
- **Renvois croisés monographie** : toute décision théorique cite `*(chap. III.8.X.Y)*` en italique.
- **Patrons de raisonnement** : recommandation = (1) compromis principal · (2) ≥ 1 alternative crédible · (3) conditions de retournement.
- **Anonymisation** : aucun cas Desjardins identifiable ; références publiques nommées librement.
- **Pas d'emoji** sauf demande explicite.

---

## Conventions de code (calque FibGo)

- Packages par responsabilité, jamais par feature.
- Interfaces étroites (ISP), ≤ 5 méthodes publiques.
- Erreurs : `fmt.Errorf("%w", err)` + types typés (`DispatchError`, `RefusError`, `CalibrationError`). **Pas de panic** sauf invariant interne cassé (sentinel re-propagé, calque `bigfft/fermat.go`).
- `t.Parallel()` systématique (cible 100 %).
- Complexité max : cyclomatique 15, cognitive 30 ; fonction ≤ 100 LOC / 50 statements.
- `doc.go` par package public, obligatoire M0 pour `tau`, `orchestration`, `calibration`.
- Commentaires : *pourquoi*, jamais *quoi*. Pas de référence au caller ni à la tâche courante.
- **Pas de globaux mutables non synchronisés** dans `internal/tau/*` — toute exception exige ADR.

**Commits — Conventional Commits** : `<type>(<scope>): <description>` avec types `feat · fix · perf · refactor · test · docs · chore · theory` (`theory` = mise à jour `docs/theory/` motivée par révision monographie).

**Co-signature IA** :

```
Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

---

## Directives projet

> Les guidelines `~/.claude/CLAUDE.md` (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution) s'appliquent **en plus**.

1. **Anti-platform discipline** — voir Doctrine. PR qui érode les anti-objectifs : reject ou mise à jour explicite du PRD §3.3.
2. **Frontière non négociable** — aucun bypass de `FrontierCheck.Inside()`. Drapeau « skip » = reject.
3. **Étanchéité Clean Architecture** — `cmd → app → orchestration → tau/{dimensions,invariants} → errors/metrics`. Gardée par `arch_test.go`.
4. **Stub LLM par défaut** — tout test sans `TAUGO_LLM_BACKEND=real` utilise le stub déterministe. CI n'appelle jamais de service LLM externe.
5. **Performance critique** — modifs dans `tau/*` ou `calibration/*` : `make benchmark` avant + après. Régression > 5 % = blocage.
6. **Golden tests immuables** (V1.1+) — pas de flag `-update` checked-in ; modification = ADR.
7. **Modifications chirurgicales** — diff minimal. Refactor > 50 LOC sur > 2 fichiers : motiver dans le commit. Bug touché en passant = `fix(...)` isolé AVANT de poursuivre.
8. **Veille active I3** — profil porte `date_revision` ; périmé → `Refus`. CI alerte à 30 jours avant péremption.
9. **Renvois croisés** — toute décision dans `docs/theory/` cite `chap. III.8.X.Y` ; lint manuel à chaque clôture de milestone.
10. **CI active** — `make test && make lint && make fuzz` en local avant PR. CI rejoue sur 3 OS. Pas de workflow concurrent.
11. **Agent teams obligatoires** — toute planification et toute exécution de tâche passent par sous-agents *(cf. §Agent teams ci-dessus)*. Le thread principal coordonne, dispatche, intègre — ne code pas directement. Plan d'exécution canonique : [`PRDPlanning.md`](PRDPlanning.md).

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

**Modification d'invariant ou de dimension** *(workflow strict)* :

```
ADR docs/adr/NNNN-<motif>.md → MAJ PRD §5 ou §6 → MAJ docs/theory/04|05 →
  implémentation → extension fuzz → vérif renvois chap. III.8 → commit theory(...)
```

---

## Modules sensibles (évolutif — cible M0+)

À créer en M0-M2, requièrent vigilance particulière une fois écrits :

| Fichier | Raison |
|---|---|
| `internal/tau/operator.go` | Point d'entrée public unique ; toute évolution = rupture API |
| `internal/tau/frontier.go` | Garde de premier rang ; bypass interdit (anti-patron #2) |
| `internal/tau/invariants/i3_authority_asymmetry.go` | Encode Searle 1995 ; refus ontologique non contournable |
| `internal/tau/invariants/i4_coherence.go` | Détecte la rupture silencieuse (chap. III.7) |
| `internal/orchestration/dispatcher.go` | Ordre des 8 étapes du pseudo-algo non arbitraire — détail [`PRD.md` §10](PRD.md) |
| `internal/calibration/profile.go` | Sérialisation reproductible byte-identique ; migration de schéma = ADR |
| `internal/calibration/drift.go` | Un drift non détecté = profil silencieusement périmé (anti-patron #3) |
| `internal/bridge/llm/client.go` | Interface étouffant la non-déterministe à `tau/*` |
| `internal/arch_test.go` | Étanchéité 4 couches ; suppression de règle = ADR obligatoire |

Tableau enrichi au fil de M1-M5 (modèle FibGo `Claude.md`).

---

## Références

- [`PRD.md`](PRD.md) — spec canonique V0.2 (911 l., 20 sections, glossaire)
- [`PRDPlanning.md`](PRDPlanning.md) — plan d'exécution M0-M6 par agent teams
- **Monographie** : `agbruneau/InteroperabiliteAgentique` v2.4.3 (2026-05-21), chap. III.8 — épingler dans chaque `Profile`
- **Ingénierie** : `agbruneau/FibGo` — `Claude.md`, `.golangci.yml`, `arch_test.go`, `internal/calibration/`
- **Empirie** : `agbruneau/AgentMeshKafka` — bridge en `internal/bridge/agentmeshkafka/` (M4)
- **HGL** (V2+) : `InteroperabiliteAgentique/RechercheFondamentale.md` — mécanisation Lean en dépôt compagnon à créer
- `docs/theory/` — renvois systématiques chap. III.8.*
- `docs/adr/` — ADR-0001 Clean Arch 4 couches · ADR-0002 Go 1.25 · ADR-0003 LLM injection · ADR-0004 AgentMeshKafka

---

*CLAUDE.md V0.3 — 2026-05-23. Ajout de la section §Agent teams et de la directive #11. Document vivant : déviation matérielle = mise à jour de ce fichier ET du `PRD.md`, AVANT le code.*
