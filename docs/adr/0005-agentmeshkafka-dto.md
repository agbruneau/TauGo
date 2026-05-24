# ADR-0005 — AgentMeshKafka adapter retourne un DTO neutre

*Statut : Accepté · Daté 2026-05-24 · Auteurs : thread principal + ruflo-core:coder*

## Contexte

PRD §12.1 spécifie initialement :

```go
Stream(ctx context.Context, topics []string) (<-chan tau.Exchange, <-chan error)
```

Cette signature viole `internal/arch_test.go` qui interdit explicitement
`bridge/agentmeshkafka → tau` (lignes 32-34), règle d'étanchéité tirée de
PRD §8.1 (Clean Architecture 4 couches). ADR-0001 pose le principe directeur ;
la règle arch_test en est la garde automatisée.

## Décision

L'adaptateur expose un **DTO local** `AgentMeshExchange` (et ses trois
sous-types `AgentMeshPrincipal`, `AgentMeshCapability`,
`AgentMeshAttestation`). Les champs sont des miroirs nominaux de ceux de
`tau.Exchange`, mais le type est **délibérément distinct** — aucun import de
`internal/tau` ni de `internal/tau/*` n'est autorisé dans le package
`bridge/agentmeshkafka`.

La conversion `AgentMeshExchange → tau.Exchange` est hébergée en
`internal/app/agentmesh.go` (M4.3), seule couche autorisée à voir
`bridge/*` ET `tau/*` simultanément.

PRD §12.1 est marqué pour révision en M6 (passe typographie + ADR-track),
afin de refléter la signature corrigée :

```go
Stream(ctx context.Context, topics []string) (<-chan AgentMeshExchange, <-chan error)
```

## Conséquences

**Positives :**
- Étanchéité Clean Architecture préservée ; ADR-0001 tient sans modification.
- `arch_test.go` n'est pas modifié — aucune règle supprimée.
- Le package `bridge/agentmeshkafka` est indépendant du kernel τ et peut
  évoluer sans risque de couplage rétrospectif.

**Négatives :**
- Conversion explicite à maintenir (un test golden viendra verrouiller la
  bijection en M4.3).
- `DiscoveryMode` passe comme `string` brut dans le DTO ; la validation du
  vocabulaire est déléguée au convertisseur `app/`.

## Alternatives rejetées

1. **Import direct de `tau` dans `bridge/agentmeshkafka`** — violation de
   la règle arch_test ; ouvre la porte à un couplage bidirectionnel
   `tau ↔ bridge` qui compromet l'orthogonalité des couches.

2. **Suppression de la règle arch_test** — inacceptable ; ADR-0001 exige
   une ADR pour toute suppression. Le coût de la conversion est moindre que
   celui d'une étanchéité dégradée.

3. **Interface générique `any`** — perd la typage statique et rend le
   convertisseur non-vérifiable à la compilation.

## Renvois

- PRD §8.1 (Clean Architecture), §12.1 (signature à réviser en M6), §18 risque #10
- ADR-0001 (Clean Arch 4 couches)
- `internal/arch_test.go` lignes 32-34
- `internal/bridge/agentmeshkafka/adapter.go` (DTO + interface)
- `internal/app/agentmesh.go` (convertisseur — M4.3)
- Plan : `docs/archive/plans-m0-m6/2026-05-24-M4-agentmeshkafka-bridge.md`

Statut : *Confirmé par construction.*
