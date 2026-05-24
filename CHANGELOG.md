# Changelog

Conforme à [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et au [Versionnage Sémantique](https://semver.org/lang/fr/).

## [Non publié]

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
