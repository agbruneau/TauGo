# TauGo

Kernel exécutable Go de l'opérateur τ et validateur empirique des invariants I1-I5 du modèle théorique défini au chapitre III.8 de la monographie *Interopérabilité Agentique en Écosystème d'Entreprise* (`agbruneau/InteroperabiliteAgentique` v2.4.3).

> **État** : V0.1 — pré-implémentation. Le squelette M0 est posé ; la logique de décision arrive en M1+. Spec canonique dans [`PRD.md`](PRD.md) (911 l.). Conventions dans [`CLAUDE.md`](CLAUDE.md) (287 l.). Plan d'exécution dans [`PRDPlanning.md`](PRDPlanning.md) (1113 l., M0-M6).

## Quick start

```bash
git clone https://github.com/agbruneau/taugo
cd taugo
make all           # build + test (Linux/macOS) ; sous Windows, voir Makefile pour équivalents `go`
./tau --version
./tau --help
```

Sans `make` (ex. Windows sans gcc) :

```bash
go build -o tau ./cmd/tau
go test -race ./...   # nécessite CGO ; absent sur Windows sans gcc
golangci-lint run ./...
```

## Qu'est-ce que TauGo ?

TauGo est un **kernel** qui décide d'un *régime d'appel* (`Deterministe | Probabiliste | Refus`) à la frontière agentique d'entreprise, sous cinq invariants réfutables. L'API publique unique est :

```go
func (k *Kernel) Decide(ctx context.Context, x Exchange) (Decision, error)
```

τ décide *où* le sens, l'autorité et le support se fixent, donc *avec quoi* appeler — jamais ce que le pair répondra.

**TauGo n'est pas** : un framework agentique · un orchestrateur · un wrapper LLM · un service réseau · un RAG · un prédicteur de comportement. Voir [`PRD.md` §3.3](PRD.md).

## Architecture

Clean Architecture, quatre couches strictes, calque structurel de `agbruneau/FibGo` :

```
cmd/tau/                  # CLI principale
internal/
  app/                    # lifecycle, injection LLM
  tau/                    # CŒUR (opérateur τ, dimensions, invariants)
  orchestration/          # dispatcher, Decision, Trace
  calibration/            # Profile, drift, thresholds
  bridge/{llm,agentmeshkafka}/
  ...
```

Étanchéité gardée par `internal/arch_test.go`. Détail [`PRD.md` §8](PRD.md).

## Documentation

- [`PRD.md`](PRD.md) — spécification canonique V0.2 (911 l., 20 sections, glossaire)
- [`CLAUDE.md`](CLAUDE.md) — conventions de rédaction et d'ingénierie + règle d'orchestration agent teams
- [`PRDPlanning.md`](PRDPlanning.md) — plan d'exécution M0-M6 par agent teams
- [`CHANGELOG.md`](CHANGELOG.md) — historique des versions (Keep-a-Changelog)
- `docs/theory/` — renvois croisés vers chap. III.8 de la monographie (M0.11+)

## Références

- **Monographie source** : `agbruneau/InteroperabiliteAgentique` v2.4.3 (2026-05-21), chap. III.8
- **Référence d'ingénierie** : `agbruneau/FibGo` (Clean Architecture, calibration adaptative, fuzz, déterminisme byte-identique)
- **Substrat empirique** : `agbruneau/AgentMeshKafka` (validation end-to-end M4+)

## Licence

Apache-2.0. Voir [LICENSE](LICENSE).
