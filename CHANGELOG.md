# Changelog

Conforme à [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et au [Versionnage Sémantique](https://semver.org/lang/fr/).

## [Non publié]

## [0.0.5-alpha] — 2026-05-24

M4 : pont théorie ↔ empirie. Branche contingence active (PRD §18 risque #1 réalisé — AgentMeshKafka inexistant local/GitHub). DTO neutre + adaptateur fichier + convertisseur en couche `app/` + générateur de corpus synthétique reproductible byte-identique + harness empirique I4 + 3 rapports.

### Ajouté

- `internal/bridge/agentmeshkafka/adapter.go` : DTO neutre `AgentMeshExchange` (champs miroir de `tau.Exchange` sans import croisé), interface `Adapter` étroite (`Stream`, `Close` — ISP 2 méthodes).
- `internal/bridge/agentmeshkafka/file_adapter.go` : `FileAdapter` JSONL — lit ligne par ligne, résilient sur lignes malformées, `Close()` idempotent (`sync.Once`), respecte `ctx.Done()`.
- `internal/bridge/agentmeshkafka/testdata/{golden-3,golden-3-malformed}.jsonl` : corpus initial 3 lignes (nominal/sans attestation/contexte riche + variante malformée).
- `internal/bridge/agentmeshkafka/classifier.go` + tests : `I4Class` (6 classes : `i4_coherent_accepted`, `i4_incoherent_refused`, `i4_false_positive`, `i4_false_negative`, `other_refusal`, `unmodeled`), `EmpiricalDecision` neutre, `ClassifyI4`, `EmpiricalI4Summary` (sensitivity/specificity).
- `internal/bridge/agentmeshkafka/empirical_i4_test.go` (build tag `empirical`) : `TestEmpiricalI4Campaign` ingère 120 traces, écrit `testdata/empirical-i4-results.json`. Package externe `agentmeshkafka_test` autorise import croisé tau+app.
- `internal/app/agentmesh.go` : pivot unique bridge ↔ tau — `ToTauExchange` (pure totale) + `StreamAsTauExchanges` (wrapper streaming avec propagation d'erreurs et fermeture propre).
- `cmd/generate-corpus/` : CLI reproductible byte-identique (seed RNG explicite, sha256 frozen `a91d60cd9815d8183df57bfcf16bbe77d36360c4ed36e33fced9f12f70fd68ee` pinné dans `TestGenerateCorpus_FrozenHash_Seed42_120_Balanced`). 3 profils : `balanced`, `i4-heavy`, `refus-heavy`. Corpus checked-in : `synthetic-corpus-120-seed42-balanced.jsonl`.
- `test/e2e/agentmeshkafka_test.go` (build tag `integration`) : `TestE2E_AgentMeshKafka_FullPipeline` + variante `_NoTopicFilter` + `_MalformedCorpus`. 3 tests, full pipeline FileAdapter → StreamAsTauExchanges → Dispatcher.Decide.
- `internal/arch_test.go` : règle `bridge/agentmeshkafka` étendue (deny `tau`, `orchestration`, `app`) ; `TestBridgeNoTauImport` AST-walk sur tous `bridge/*` (exclut `_test.go`).
- `Makefile` : cibles `e2e` (tag integration) et `empirical-i4` (tag empirical).
- `docs/adr/0005-agentmeshkafka-dto.md` : décision DTO neutre + révision PRD §12.1 marquée pour M6.
- `docs/empirical/I4-report.md` (151 l.) : rapport campagne — 120 décisions classifiées, statut I4 **Hypothèse inchangée** (le générateur synthétique n'injecte pas les clés `Context` qui pilotent D-INVARIANT au-dessus du seuil ; la garde I4 n'a jamais été sollicitée).
- `docs/empirical/unmodeled.md` (108 l.) : 3 observations initiales (OBS-001 Context absent, OBS-002 frontière agrégée, OBS-003 AgentMeshKafka indisponible — risque #1 PRD §18 réalisé).
- `docs/empirical/I4-regime.md` (53 l.) : note d'audit — Régime B (contingence) sélectionné, conditions de bascule vers A documentées.
- `docs/superpowers/plans/2026-05-24-M4-agentmeshkafka-bridge.md` : sous-plan détaillé M4 (2134 l., 11 tâches).

### Notes

- **Découverte architecturale M4.1** : la signature PRD §12.1 `Stream(...) (<-chan tau.Exchange, ...)` viole `arch_test.go` (deny `bridge → tau`). Décision ADR-0005 : DTO neutre `AgentMeshExchange` dans `bridge/` + converter `ToTauExchange` dans `app/`. PRD §12.1 marqué pour révision M6.
- **Régime B activé** : `agbruneau/AgentMeshKafka` n'existe ni local ni sur GitHub (audit M4.0). Campagne empirique sur corpus synthétique reproductible. Bascule vers Régime A reportée à un éventuel M4-bis.
- **I4 inconclusif** : 84 `i4_coherent_accepted`, 36 `other_refusal`, 0 TP, 0 FN, 0 FP. Sensitivity = `-1` (dénominateur nul), Specificity = `1.0`. Cause racine documentée OBS-001 : le générateur ne peuple pas `Context.event_registry` ni `Context.idempotency_key_mode` → D-INVARIANT plafonne à 0.25 < `θ_inv = 0.50`.
- **Revue intégrée M4** : APPROVE_WITH_CONCERNS. 4 observations non-bloquantes (NB1-NB4) reportées à M4-bis ou M5 ; aucun bloquant code.
- Race detector indisponible sur Windows local (couvert par CI Linux/macOS). Le harness empirique court ~5s sur 120 traces.

## [0.0.4-alpha] — 2026-05-24

M3 : cinq invariants I1-I5 encodés et fuzzés, étape 8 dispatcher (`EvaluateInvariants → Trace.UnmodeledObservations`), gardes anti-patrons #1/#3/#4 par test. Smoke fuzz 5 s vert sur 5 cibles. Couverture `tau/invariants` 91.2 %.

### Ajouté

- `internal/tau/invariants/` (nouveau package) : `evaluator.go` (types `Status`, `Statuses`, `EvaluateInvariants`), `i1_conservation.go` + `Conserve`, `i2_irreductibility.go` + `Residu`/`Recablage`, `i3_authority_asymmetry.go` + `IsProfileExpired` + `EvaluateI3WithClock` + constante `I3PerimptionLimite` (2027-01-01 UTC), `i4_coherence.go` + `IsIncoherent`, `i5_composition.go` + `AngleMort`/`Pile`/`Aggregate`/`M`/`BoundsHold` (API d'agrégation calculée en V1 — la mention « V2 calcule » du PRD §6.1 est levée).
- `internal/tau/invariants/fuzz_targets_test.go` : 5 cibles fuzz `FuzzI1_Conservation`, `FuzzI2_Irreductibilite`, `FuzzI3_AsymetrieAutorite`, `FuzzI4_CoherenceContrainte`, `FuzzI5_CompositionConjonctive`. Smoke 5 s : I1 8.6M, I2 8.6M, I3 8.2M, I4 9.5M, I5 701K exécutions, 0 crash.
- `internal/tau/invariants/testdata/fuzz/FuzzI*/seed-*` : corpus seeds checked-in (3 seeds/cible) + 1 cas de régression FuzzI5 (`bf9c5ac437b95a58`).
- `internal/orchestration/dispatcher.go` : étape 8 du pseudo-algo PRD §10 — `invariants.EvaluateInvariants(x, dec)` ; `AnyViolated() → append(Summary())` dans `Trace.UnmodeledObservations`. Régime et Diagnostic intouchés.
- `internal/orchestration/dispatcher_invariants_test.go` : 3 tests (no violation, violation détectée, régime préservé).
- `internal/anti_patterns_test.go` : `TestNoPredictiveAPI` (parse AST des 4 packages, regex `^(Predict|Expected|Forecast)`), `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported`.
- `internal/arch_test.go` : règle `tau/invariants` étendue (3 deny : `tau/dimensions`, `orchestration`, `bridge`).
- `internal/tau/invariants/evaluator_test.go` : `TestStatus_String` couvre les 4 valeurs.
- `docs/theory/05-invariants.md` : renvoi croisé chap. III.8.5 — verbatims I1-I5, reformulations exécutables, conditions de réfutation, helpers Go, marqueurs épistémiques.
- `docs/empirical/fuzz-summary.md` : rapport empirique M3 — méthodologie, résultats par cible, découvertes, limites V1, prochaines étapes.
- `docs/superpowers/plans/2026-05-24-M3-invariants-fuzz.md` : sous-plan détaillé M3 (2047 l., 11 tâches bite-sized).

### Corrigé

- `internal/tau/invariants/i5_composition.go` : `BoundsHold` utilisait `len(layer)` (longueur slice) au lieu de la cardinalité ensembliste. Détecté par `FuzzI5_CompositionConjonctive` sur une pile `[["z","z"]]` (commit `7b4739c`). Calque le pattern FibGo « fuzz-discovered fix avant feat ».

### Notes

- Race detector indisponible sur Windows local (CGO/gcc) ; CI Linux/macOS couvre. Smoke fuzz 5 s sur dev, 30 s sur CI via `make fuzz`, 24 h hebdo via `make fuzz-long`.
- Étape 3 du dispatcher (péremption `today > date_revision`) reportée à M5 ; la propriété est gardée au niveau `Profile` en M3 (`TestI3_DateRevisionRespectee`).
- `EvaluateI5` retourne `Held` par construction V1 (pile d'angles morts non attachée à `Decision` avant V2) ; les bornes algébriques `max(|Aᵢ|) ≤ M(π) ≤ Σ|Aⱼ|` sont exercées directement par `FuzzI5` via `BoundsHold`.
- Revue intégrée M3 : APPROVE_WITH_CONCERNS. Observations résiduelles (info, non-bloquantes) reportées : couverture branche `EvaluateI2` zero-residue, vitesse FuzzI5 décodage byte-slice, couplage indirect `TestUnmodeledObservationsReported` ↔ `TestStep8_*`.
- Anti-patron #2 (hors frontière) toujours gardé par `TestFrontierCheck_*` (M0).

## [0.0.3-alpha] — 2026-05-23

M2 : trois dimensions (D-SENS, D-AUTORITÉ, D-INVARIANT) calculables, gardes ontologique I3 et cohérence I4 actives, pseudo-algo PRD §10 complet (étapes 1-7), Profile et AtomicThresholds. Couverture globale 92.2%.

### Ajouté

- `internal/tau/operator.go` : types `Principal`, `Capability`, `DiscoveryMode` (Static / DynamicMCP / DynamicA2A / DynamicAGNTCY) ; champs `Exchange.Initiator`, `Exchange.Target` ; `TraceThresholds` étendu (`AuthBlock`, `SensCoherence`, `InvCoherence`).
- `internal/tau/dimensions/` (nouveau package) : `score.go` (type `Score` partagé + `clamp01`), `dsens.go` + tests (4 sondes PRD §5.1), `dauthority.go` + tests (4 sondes PRD §5.2 + asymétrie ontologique), `dinvariant.go` + tests (4 sondes PRD §5.3 + contrainte I4).
- `internal/orchestration/dispatcher.go` : refonte M2 — `frontierFromExchange` heuristique (remplace placeholder M1) ; étape 2 garde ontologique D-AUTORITÉ (Refus I3) ; étape 4 scores des 3 dimensions ; étape 5 garde I4 ; étape 6 composite pondéré ; étape 7 hystérèse. Pseudo-algo PRD §10 étapes 1-7 complet.
- `internal/orchestration/thresholds.go` : étendu (`AuthBlock`, `SensCoherence`, `InvCoherence`) + `DefaultThresholds()`.
- `internal/orchestration/guards_test.go` : `TestRefusOntologiqueDAUTORITE`, `TestI4_IncoherenceDetectee`, `TestOntologicalGuardPassesWithAttestation`, `TestI4_CoherentCombinationAccepted`.
- `internal/calibration/profile.go` : `Profile`, `Weights`, `Thresholds` (PRD §11.3) + `DefaultProfile()` (DateRevision 2026-12-01, version monographie v2.4.3).
- `internal/calibration/thresholds_atomic.go` : `AtomicThresholds` calque FibGo `bigfft/fft.go` — `atomic.Int64` privés en milli-unités, accesseurs lecture, `SetTuning` coordonné, panic sentinel sur ordering violation, `Snapshot()` immuable.
- `internal/app/app.go` : utilise `orchestration.DefaultThresholds()` au lieu de hard-coded.
- `cmd/tau/main_test.go` : E2E adapté pour exchanges M2 (Initiator + Target inclus).
- `.golangci.yml` : termes français ajoutés au misspell ignore (combinaison, incohérente, détectée, frontière, verrou, ontologique).
- `docs/theory/04-dimensions.md` (170 l.) : renvoi croisé chap. III.8.4 — 3 dimensions, sondes, encodage Go, asymétrie ontologique (Searle 1995), contrainte I4.
- `docs/empirical/M2-sample-decisions.md` (397 l.) : 10 décisions tracées via `tau decide`, ventilation des scores par dimension, couvre tous les chemins (frontier refus, I3, I4, deterministe, probabiliste, hystérèse).
- `docs/superpowers/plans/2026-05-23-M2-dimensions-gardes.md` (2416 l.) : sous-plan détaillé M2.

### Modifié

- Tests dispatcher et invariants Decision adaptés pour le frontier heuristique M2 (les fixtures M1 fournissaient des Exchange sans Initiator/Target, qui maintenant tombent hors frontière par défaut).
- `TestDefaultLLMIsStub` adapté — vérification par déterminisme TauScore plutôt que comparaison directe au score Stub (le composite M2 ne se réduit plus à la sonde LLM seule).

### Notes

- Couverture par package : `tau/dimensions` 41.8 %, `orchestration` ≥ 80 %, `calibration` 100 %, `tau` 0.7 % (le code tau est mostly types/interface, sans logique testable directement — testée indirectement via dimensions et orchestration).
- M2.10 `docs/empirical/M2-sample-decisions.md` constitue le premier corpus de référence pour la calibration M5.
- Atomic accessors prêts pour la concurrence M5 (test `TestAtomicThresholds_ConcurrentReadsSafe` valide 100 goroutines).
- Anti-patrons à venir en M3 : `TestNoPredictiveAPI`, `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported`.

## [0.0.2-alpha] — 2026-05-23

M1 : dispatcher minimal + stub LLM. `tau decide` rend une `Decision` instrumentée. Cinq tests d'invariants `Decision`. Couverture globale 83.9 % (> 80 % gate).

### Ajouté

- `internal/tau/operator.go` : types `Trace` et `TraceThresholds` immuables ; champ `Decision.Trace` ; tags JSON snake_case sur tous les types publics.
- `internal/tau/frontier.go` : tags JSON snake_case sur `FrontierCheck`.
- `internal/orchestration/thresholds.go` : `Thresholds{Deterministe, Probabiliste}` avec invariant `Ordered()`.
- `internal/orchestration/dispatcher.go` : `Dispatcher` implémentant le sous-ensemble M1 du pseudo-algo PRD §10 (étapes 1, 6, 7) ; `NewDispatcher` panic sur ordering invariant ; clampage `DurationNs ≥ 1` pour résolution timer Windows.
- `internal/orchestration/dispatcher_test.go` : 3 tests (Deterministe, Probabiliste, hystérèse).
- `internal/orchestration/decision_invariants_test.go` : `TestDecisionAlwaysTraced`, `TestRefusImpliesDiagnostic` (3 subtests), `TestTraceImmutable`.
- `internal/bridge/llm/client.go` : interface étroite `Client` (PRD §12.2) — `Fingerprint()`, `Interpret(ctx, intent) (float64, error)`.
- `internal/bridge/llm/stub.go` : `Stub` déterministe via FNV-1a 32-bit hash ; fingerprint `stub:v0` ; score ∈ [0, 1) ; mappage checked-in.
- `internal/bridge/llm/stub_test.go` : fingerprint + déterminisme + bornes sur 4 cas (vide, 1 char, multi-mot, phrase).
- `internal/app/app.go` : `NewDispatcher()` factory ; sélection LLM via env `TAUGO_LLM_BACKEND` (défaut `Stub` ; `real` panic en M5+).
- `internal/app/app_test.go` : `TestDefaultLLMIsStub` (vérification comportementale TauScore vs Stub.Interpret).
- `internal/arch_test.go` : règle `internal/bridge` parent (skip-always) remplacée par règles concrètes sur `internal/bridge/llm` et `internal/bridge/agentmeshkafka`.
- `cmd/tau/main.go` : sous-commande `decide` (JSON stdin → JSON stdout, exit codes 0/2/3/4) ; version bumped à `0.0.2-alpha`.
- `cmd/tau/main_test.go` : tests E2E `TestEndToEnd_DecideDeterministe` (« creative generation » → 0.262) et `TestEndToEnd_DecideProbabiliste` (« hello world » → 0.807).
- `docs/superpowers/plans/2026-05-23-M1-dispatcher-stub-llm.md` : sous-plan détaillé M1 (1017 l., 9 tâches bite-sized).

### Corrigé

- Tags JSON manquants sur `tau.Exchange` : caché en M1.5, exposé par décodage silencieux de `intent_description` en chaîne vide (TauScore=0.261 = hash empty string). Fix `dff5565` aligne snake_case I/O.

### Spec et planification

- `PRDPlanning.md` reste référence ; sous-plan M1 commité séparément dans `docs/superpowers/plans/`.

### Notes

- Tous les sub-tasks M1 ont commit séparé. Couverture par package : `internal/tau` 100 %, `internal/orchestration` ≥ 90 %, `internal/bridge/llm` ≥ 80 %, `internal/app` ≥ 80 %.
- `tau decide` accepte stdin JSON ; sortie JSON snake_case ; régimes `0=Unknown, 1=Deterministe, 2=Probabiliste, 3=Refus` (marshaled comme nombre — M2+ peut ajouter `MarshalJSON`).
- Frontière de validité encore en mode placeholder (Inside=true toujours) ; les sondes réelles arrivent en M2.

## [0.0.1-alpha] — 2026-05-23

Premier tag. Squelette M0 du PRD : pas de logique métier, étanchéité architecturale gardée, CI verte sur 3 OS.

### Ajouté

- Squelette du module Go (`go.mod` `github.com/agbruneau/taugo`, `go 1.25.0`, `toolchain go1.26.3`).
- Licence Apache-2.0 (`LICENSE`).
- `.gitignore` (binaires, fichiers de test, IDE, artefacts agent runtime).
- `.golangci.yml` calque FibGo : 24 linters, complexité max 15/30, longueur fonction ≤ 100 LOC / 50 statements, `misspell` US + termes domaine FR-CA (Probabiliste, Deterministe, Refus, agentmeshkafka), `go: "1.25"` pour compatibilité golangci-lint v1.64.8.
- `Makefile` avec cibles `all`, `build`, `test`, `test-short`, `coverage`, `benchmark`, `lint`, `fuzz`, `fuzz-long`, `calibrate`, `build-reproducible`, `build-pgo`, `build-all`, `clean`. Timestamp gelé `1778889600` pour build reproductible (calque InteroperabiliteAgentique).
- Squelette `internal/` (10 packages avec `doc.go` descriptifs) : `tau`, `orchestration`, `calibration`, `bridge/{llm,agentmeshkafka}`, `app`, `config`, `errors`, `metrics`, `testutil`.
- `internal/tau/frontier.go` — `FrontierCheck` encodant les 4 conditions classiques de la frontière de validité de τ (chap. III.8.3.2) ; garde anti-patron #2 (« hors frontière »).
- `internal/tau/frontier_test.go` — 5 sous-tests TDD (all-true, all-false, 4× one-false), 100 % de couverture, `t.Parallel()` partout, init par champs nommés (anti-régression sur ajout de champ).
- `internal/tau/operator.go` — types `Regime`, `Exchange`, `Attestation`, `Decision` ; interface `Kernel` avec signature `Decide(ctx, Exchange) (Decision, error)`. Types `Trace`, `Principal`, `Capability` reportés à M1/M2.
- `internal/arch_test.go` — 4 règles d'étanchéité Clean Architecture (PRD §8.1) : `tau/* → orchestration/bridge/app` interdit ; `dimensions ↔ invariants` interdit (orthogonalité) ; `bridge → tau/*` direct interdit.
- `cmd/tau/main.go` — CLI minimale (`--help`, `--version`) ; points d'injection ldflags `main.version` et `main.buildTimestamp`.
- `.github/workflows/ci.yml` — matrice 3 OS (Linux/macOS/Windows) × Go 1.25.x : test (race CGO), lint (golangci-lint v1.64.8), build, cross-compile (linux/arm64, darwin/{amd64,arm64}), fuzz-smoke placeholder pour M3+.
- `.github/workflows/coverage.yml` — gate 80 % couverture globale ; per-package 90 % sur `tau/*` actif en M1+.
- `README.md` — point d'entrée FR-CA : quick start, doctrine (TauGo est / n'est pas), architecture résumée, liens vers PRD/CLAUDE/PRDPlanning.
- `docs/theory/03-operateur-tau.md` — premier renvoi croisé chap. III.8.3 : définition formelle `τ : t_fix(g) ≺ t_int ↦ t_fix(g) ≈ t_int`, table d'encodage TauGo, propriétés exploitables (bases I1, I2, orthogonalité), frontière de validité (4 conditions), anti-patrons cités.

### Spec et planification

- `PRD.md` V0.2 (refactorisé — 911 l., 20 sections, glossaire 16 termes).
- `CLAUDE.md` V0.3 (refactorisé + section Agent Teams + directive #11).
- `PRDPlanning.md` initial (1113 l., M0 détaillé bite-sized, M1-M6 résumés haut niveau).

### Notes

- Race detector requis sous CGO ; absent sur Windows sans gcc local — couvert par CI Linux/macOS.
- `golangci-lint` `run.go: "1.25"` requis car `golangci-lint v1.64.8` est construit avec Go 1.25 alors que `go.mod` carries `toolchain go1.26.3`.
- `.claude-flow/` (agent runtime artifacts) est ignoré par `.gitignore`.
- Drapeaux PRD §18 risques #1 (`AgentMeshKafka` not ready in M4) et #4 (scope creep) à surveiller.
