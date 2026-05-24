# ADR-0004 — AgentMeshKafka comme bridge empirique avec branche contingence

*Statut : Accepté · Daté 2026-05-24 (rétroactif M4) · Auteurs : thread principal + ruflo-core:coder*

## Contexte

PRD §1 pose l'exigence fondatrice : le kernel τ doit être validé empiriquement sur
des échanges agentiques réels, pas seulement sur des corpus synthétiques. PRD §12.1
désigne explicitement `AgentMeshKafka` (`agbruneau/AgentMeshKafka`) comme substrat
de validation empirique cible.

Ce choix repose sur trois éléments :

1. **Historique empirique** — AgentMeshKafka a produit les traces ayant alimenté
   les hypothèses I1-I5 (chap. III.8.5). C'est le substrat naturel de validation
   continue.
2. **Architecture compatible** — le modèle publish/subscribe Kafka correspond à
   l'abstraction `Stream(ctx, topics) (<-chan Exchange, <-chan error)` de PRD §12.1.
3. **Réutilisation du bridge** — le package `internal/bridge/agentmeshkafka/` peut
   basculer du mock vers le vrai Kafka sans toucher le kernel.

**Risque PRD §18 #1 — réalisé en M4** : à la date de développement de M4, le repo
`AgentMeshKafka` n'est pas disponible localement ni sur GitHub dans un état intégrable.
Sans stratégie de contingence, M4 serait bloqué 4-6 semaines en attente du substrat.

La décision doit donc arbitrer entre deux valeurs contradictoires :
- Fidélité empirique (attendre le vrai substrat).
- Continuité de livraison (progresser avec un substrat synthétique reproductible).

## Décision

TauGo adopte une **stratégie bi-régime** pour le bridge AgentMeshKafka :

### Régime A — FileAdapter (branche contingence M4, actif)

`internal/bridge/agentmeshkafka/adapter.go` expose une interface `Adapter` avec
deux implémentations :

- **`FileAdapter`** — lit des fichiers JSONL locaux (corpus synthétique). Conforme
  à ADR-0005 (retourne `AgentMeshExchange`, jamais `tau.Exchange`). Utilisé par
  défaut en CI et dans tous les tests sans build tag `integration`.

- **`cmd/generate-corpus/`** — génère un corpus synthétique reproductible (seed
  fixe, 100 échanges couvrant I1-I5 et les 5 cas de refus). Ce corpus est
  checked-in dans `test/golden/corpus/`.

La conversion `AgentMeshExchange → tau.Exchange` est hébergée en `internal/app/agentmesh.go`
(cf. ADR-0005 pour la justification DTO).

### Régime B — KafkaAdapter (cible M5+, non actif)

`KafkaAdapter` implémentera la même interface `Adapter` en se connectant à un broker
Kafka réel. La bascule ne nécessite aucune modification de `internal/tau/*` ni de
`internal/orchestration/*` — uniquement un changement de binding dans `internal/app/`.

Les tests E2E de Régime B sont sous build tag `integration` et sont exclus de la CI
standard (`make test`). Ils sont lancés via `make test-integration` avec variable
`TAUGO_AMKAFKA_BROKER` définie.

### Garde architecturale

La règle ADR-0001 s'applique intégralement : `bridge/agentmeshkafka` n'importe pas
`internal/tau/*`. Toute violation est détectée par `arch_test.go` (lignes 32-34).

## Conséquences

**Positives :**
- M4 est livré sans bloquer sur la disponibilité d'AgentMeshKafka. Le corpus
  synthétique permet de valider les invariants I1-I5 sur 100 traces avec scores
  ventilés (cf. `docs/empirical/M2-sample-decisions.md`).
- La bascule vers le vrai Kafka (Régime B) est transparente pour le kernel —
  aucune modification dans `internal/tau/*` ou `internal/orchestration/*`.
- Le `FileAdapter` sert de documentation exécutable du format d'échange attendu
  par AgentMeshKafka.
- Reproductibilité garantie : corpus généré avec seed fixe, résultats identiques
  sur les 3 OS.

**Négatives :**
- La campagne empirique M4 est conduite sur corpus synthétique, pas sur trafic
  réel. Les invariants I4 et I5 restent au statut `Hypothèse` /`Probable` plutôt
  que `Confirmé` (cf. `docs/empirical/I4-report.md`).
- Deux implémentations d'`Adapter` à maintenir jusqu'à la disponibilité d'AgentMeshKafka.
- Le corpus synthétique ne capture pas les distributions réelles de trafic inter-agents.
  Risque de biais de confirmation sur I3 (chap. III.8.5).

**Neutres :**
- `docs/empirical/I4-regime.md` documente la distinction Régime A / Régime B et
  les conditions de bascule.

## Alternatives rejetées

1. **Bloquer M4 jusqu'à disponibilité d'AgentMeshKafka** — coût calendrier estimé
   à 4-6 semaines. Inacceptable : les milestones M5 (calibration adaptative) et M6
   (release) auraient glissé en cascade. La dette empirique est préférable à la
   dette calendrier pour un projet encore en phase alpha.

2. **Embarquer Kafka directement dans `internal/tau/*`** — viole ADR-0001 (étanchéité
   Clean Architecture). Le kernel deviendrait dépendant d'un broker Kafka pour
   s'initialiser, rendant les tests unitaires impossibles sans infrastructure.
   Rejet immédiat.

3. **NATS comme substrat de remplacement** — NATS est plus léger que Kafka et a une
   API Go mature. Cependant, les traces empiriques historiques proviennent
   d'AgentMeshKafka (Kafka). Remplacer le substrat aurait invalidé la comparabilité
   des mesures M4 avec les données pré-TauGo. Rejet : fidélité empirique compromise.

4. **Aucun bridge empirique en V1** — exécuter TauGo uniquement sur corpus goldens
   injectés manuellement. Viole PRD §1 (exigence de validation empirique) et
   PRD §12.1 (AgentMeshKafka désigné). Rejet : anti-objectif explicite.

## Renvois

- PRD §1 (exigence de validation empirique)
- PRD §12.1 (AgentMeshKafka comme substrat cible)
- PRD §18 risque #1 (AgentMeshKafka pas prêt en M4)
- ADR-0001 (Clean Architecture — règle bridge/tau)
- ADR-0005 (DTO `AgentMeshExchange` — contournement conforme)
- `internal/bridge/agentmeshkafka/adapter.go` (interface + FileAdapter)
- `internal/app/agentmesh.go` (convertisseur couche app)
- `cmd/generate-corpus/` (générateur corpus synthétique)
- `docs/empirical/I4-regime.md` (distinction Régime A / Régime B)
- `docs/empirical/I4-report.md` (campagne M4 — statut I4 Hypothèse)
- `test/golden/corpus/` (corpus checked-in)

*Statut : Confirmé (branche contingence Régime A active). Bascule Régime B conditionnée
à la disponibilité d'AgentMeshKafka — aucune date fixée à ce jour.*
