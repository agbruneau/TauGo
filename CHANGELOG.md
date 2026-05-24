# Changelog

Conforme à [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et au [Versionnage Sémantique](https://semver.org/lang/fr/).

## [Non publié]

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
