# 00 — Bootstrap d'audit (orchestrateur)

> Journal de bootstrap produit par le thread orchestrateur **avant** dispatch des 6 sous-agents.
> FR-CA. Marqueurs : `[confirmé]` `[probable]` `[hypothèse]` `[à vérifier]`.

## Conclusion de bootstrap (pyramide inversée)

**[confirmé]** Environnement *quasi complet* pour un audit statique + dynamique de haute fidélité, **à une exception majeure** : `go test -race` **indisponible** (`CGO_ENABLED=0`, aucun compilateur C) → l'axe concurrence (SA3) bascule en analyse statique. Tout le reste (build, tests, fuzz, benchmarks, lint épinglé v1.64.8, staticcheck, gosec, couverture) est exécutable nativement.

## Cible & état

| Élément | Valeur |
|---|---|
| Dépôt | `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo` · module `github.com/agbruneau/taugo` |
| `go.mod` | `go 1.25.0` · `toolchain go1.26.3` |
| HEAD | `1948a7b` · `git describe` `v0.1.0-17-g1948a7b` |
| État | v0.1.2-pre (retrait CI/CD, ADR-0010, pure-local) ; arbre git propre |
| Plateforme | Windows 11 / PowerShell 7 / amd64 / 24 cœurs |

## Toolchain — disponibilité × repli

| Outil | Statut | Version | Repli |
|---|---|---|---|
| git | OK | 2.54.0 | — |
| go | OK | **1.26.3** | — |
| golangci-lint | OK | **v1.64.8** (pin exact, schéma v1) | — *(aucun écart)* |
| staticcheck | OK | 2026.1 (v0.7.0) | — |
| gosec | OK | dev | — |
| python | OK | 3.14.5 | — |
| gofmt / go vet | OK | bundle Go 1.26.3 | — |
| gcc / clang | **ABSENT** | — | `-race` indisponible → SA3 statique |
| make | **ABSENT** | — | `go` / `golangci-lint` directs |

`go env` : `CGO_ENABLED=0` · `GOOS=windows` · `GOARCH=amd64` · `GOTOOLCHAIN=auto`.

## Inventaire (16 packages)

`cmd/{generate-corpus,tau}` · `examples/quickstart` · `internal/{app,bridge/agentmeshkafka,bridge/llm,calibration,errors,orchestration,tau,tau/dimensions,tau/invariants,testutil,thresholds}` · `test/e2e`.

- Fuzz : 5 cibles `FuzzI1`…`FuzzI5` (`internal/tau/invariants/`). Build tags : `integration`, `e2e`, `empirical`.
- Golden : `tests/calibration/golden-corpus.jsonl`. ADRs : 0001-0010 présents. `.github/` absent (ADR-0010 confirmé).

## Vérifications de base (HEAD)

`go build ./...` exit 0 (~1,3 s) · `go vet ./...` exit 0 · `golangci-lint version` v1.64.8. [confirmé]

## Adaptations (prompt écrit pour Claude Code on the web / Linux → poste Windows local)

1. Pas de `make` → cibles traduites en `go`/`golangci-lint` directs.
2. Pas de `-race` (CGO off) → SA3 statique ; tests sans `-race`.
3. sha256 : `Get-FileHash -Algorithm SHA256` (PowerShell).
4. Binaire `tau.exe` ; sorties temporaires confinées à `audit/`.
5. Go déjà 1.26.3, golangci-lint déjà v1.64.8 → aucune installation, aucun écart.

## Orchestration (3 vagues, max 3 agents actifs)

- **Vague 1** : SA1 conformité τ · SA2 invariants/épistémique · SA3 concurrence.
- **Vague 2** : SA5 idiomatique · SA6 architecture/tests.
- **Vague 3** : SA4 performance **solo** *(machine au repos → fidélité benchmarks ; déviation assumée vs les 2 vagues du prompt)*.

Partage d'état par sortie structurée ; l'orchestrateur écrit les 6 rapports d'axe + `RAPPORT_FINAL.md`. Lecture seule sur code et golden corpus.
