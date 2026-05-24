# ADR-0001 — Clean Architecture 4 couches

*Statut : Accepté · Daté 2026-05-24 (rétroactif M0) · Auteurs : thread principal + ruflo-swarm:architect*

## Contexte

PRD §8.1 prescrit une disposition en couches unidirectionnelles pour TauGo :

```
cmd/{tau, generate-golden}/
internal/
  app/                      # lifecycle, injection LLM concret
  orchestration/            # dispatcher, Decision, Trace (immuables)
  tau/                      # COEUR — n'importe pas orchestration/ ni bridge/
  calibration/              # Profile, drift, thresholds
  bridge/{agentmeshkafka, llm}/
  {config, errors, metrics, testutil}/
```

Ce choix est hérité du calque FibGo et motivé par trois exigences simultanées :

1. **Testabilité** — le kernel τ (`internal/tau/*`) doit être testable sans dépendance
   LLM, Kafka ou réseau. Une dépendance rétrograde (`bridge → tau`) ou latérale
   (`tau → orchestration`) rend cet objectif inatteignable sans mock à grande échelle.

2. **Traçabilité décisionnelle** — chaque invariant I1-I5 (chap. III.8.5) est
   encodé dans `internal/tau/invariants/` ; si ce package importe n'importe quelle
   couche d'infrastructure, la rupture devient silencieuse.

3. **Évolutivité** — les couches `bridge/` (AgentMeshKafka, LLM) évoluent
   indépendamment du kernel. Sans garde statique, la frontière s'érode à mesure
   que les développeurs cherchent le chemin le plus court.

Sans garde automatisée, la dette architecturale s'accumule entre les milestones sans
signal d'alarme. PRD §8.1 le formule : « étanchéité gardée par `internal/arch_test.go` ».

## Décision

TauGo adopte une **Clean Architecture en 4 couches strictes**, unidirectionnelles,
dont l'étanchéité est garantie par un test Go pur `internal/arch_test.go` exécuté
en CI sur les 3 OS cibles :

| Couche | Package(s) | Peut importer |
|---|---|---|
| Présentation | `cmd/*`, `internal/app/` | tout sauf `tau/*` direct hors interface |
| Orchestration | `internal/orchestration/` | `tau/`, `calibration/`, `errors/`, `metrics/` |
| Domaine | `internal/tau/` | `errors/`, `metrics/`, `config/` uniquement |
| Infrastructure | `internal/bridge/*`, `internal/calibration/` | `errors/`, `metrics/`, `config/` ; bridge interdit d'importer `tau/*` |

Règles négatives explicites (gardées par `arch_test.go`) :

- `tau/* → orchestration` : interdit.
- `tau/* → bridge` : interdit.
- `bridge/* → tau/*` direct : interdit (cf. ADR-0005 pour le contournement DTO).
- `dimensions ↔ invariants` croisé : interdit (orthogonalité I1-I5 vs 3 dimensions).
- LLM concret (`anthropic`, `openai`, …) hors `app/` et `bridge/llm/` : interdit.

Toute suppression d'une règle `arch_test.go` nécessite une ADR préalable. L'absence
d'ADR constitue un motif de rejet de la PR.

## Conséquences

**Positives :**
- Le kernel τ est testable de façon isolée ; la CI n'appelle jamais de service externe.
- Les invariants I1-I5 et les dimensions D-SENS / D-AUTORITÉ / D-INVARIANT sont
  protégés contre le couplage involontaire avec l'infrastructure.
- `arch_test.go` détecte toute violation dès `go test ./internal/...` — coût de
  détection nul en revue de code.
- Ajout d'un nouveau backend LLM ou d'un nouveau substrat agentique sans toucher
  le kernel.

**Négatives :**
- Toute communication légitime entre couches non adjacentes requiert un DTO neutre
  (cf. ADR-0005) et un convertisseur en couche `app/`. Cela ajoute une indirection
  par franchissement de frontière.
- Un besoin architectural non anticipé (ex. accès `orchestration` depuis `bridge`)
  exige un ADR, ce qui ralentit les décisions d'urgence.

**Neutres :**
- L'ensemble du code étant sous `internal/`, la règle ne peut pas être contournée
  via `go.mod` : tout module externe qui voudrait importer un package interne obtiendrait
  une erreur de compilation.

## Alternatives rejetées

1. **Architecture hexagonale 3 couches (ports & adapters)** — aurait fourni une
   séparation domaine/infrastructure correcte, mais la notion de « couche
   orchestration » distincte du domaine aurait été absente. La traçabilité du
   dispatcher (PRD §10 — 8 étapes ordonnées) aurait été perdue dans un port générique.
   Rejet formalisé en PRD §14.2.

2. **Flat `internal/` sans séparation** — structure la plus simple, mais aucune
   garde statique n'est possible. L'expérience FibGo montre que sans séparation
   explicite, les dépendances cycliques apparaissent dès le milestone 2. Rejet
   immédiat lors du design initial M0.

## Renvois

- PRD §8.1 (Clean Architecture et règles d'étanchéité)
- PRD §14.2 (alternatives architecturales rejetées)
- PRD §10 (dispatcher 8 étapes)
- CLAUDE.md §Architecture (résumé exécutable des couches)
- `internal/arch_test.go` (garde statique)
- ADR-0005 (DTO AgentMeshKafka — contournement conforme à la règle)
- FibGo `CLAUDE.md` (calque d'origine)

*Statut : Confirmé — règle active depuis M0, aucune exception accordée à ce jour.*
