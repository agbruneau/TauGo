# Observations non modélisées — registre vivant

*Anti-patron #4 PRD §7.2. Document vivant. Daté 2026-05-24. Sera enrichi à chaque milestone.*
*Statut : **Initial** — campagne M4, Régime B (synthétique).*
**(chap. III.8.7 / PRD §7.2 #4)*

---

## §1 Mission du document

L'anti-patron #4 *(PRD §7.2)* interdit de laisser passer sous silence une observation non modélisée à fort impact. Ce registre est le point de collecte centralisé de toutes les observations dont le modèle actuel (kernel τ, dimensions, invariants I1-I5) ne rend pas compte de façon satisfaisante.

Chaque entrée est :
- identifiée par un ID unique (`OBS-<AAAA-MM>-<NNN>`) ;
- catégorisée selon la taxonomie §3 ;
- assortie d'une piste de modélisation et d'un statut.

Le garde opérationnelle est `TestUnmodeledObservationsReported` *(PRD §7.2 #4)*.

---

## §2 Observations M4 (campagne Régime B — 2026-05-24)

### OBS-2026-05-001 — Absence de signaux `Context` dans le générateur synthétique

| Champ | Valeur |
|---|---|
| Date | 2026-05-24 |
| Milestone | M4 |
| Source | `cmd/generate-corpus` / `internal/bridge/agentmeshkafka/empirical_i4_test.go` |
| Diagnostic | Les sondes D-INVARIANT lisent les champs `Context` (`event_registry`, `idempotency_key_mode`, profondeur de délégation) de l'exchange pour calculer un score > 0.25. Le générateur synthétique n'injecte aucun de ces champs — D-INVARIANT reste à `0.25` pour toutes les 120 traces, sous `θ_inv = 0.50`. La garde I4 ne se déclenche donc jamais. |
| Catégorie | Limite d'instrumentation synthétique |
| Impact | Empêche toute validation empirique d'I4 ; campagne M4 inconclusive sur I4 — **Hypothèse** *(chap. III.8.5.4)*. |
| Piste | Enrichir `cmd/generate-corpus` en M4-bis : ajouter un champ `context_config` dans le JSON de corpus, propagé vers `agentmeshkafka.Exchange.Context`, lu par les sondes D-INVARIANT. Viser D-INVARIANT ∈ [0.60, 0.90] pour les traces `r4-`. |
| Statut | **Ouvert** — M4-bis |

---

### OBS-2026-05-002 — Heuristique `frontierFromExchange` rejette 30 % du corpus avant scoring

| Champ | Valeur |
|---|---|
| Date | 2026-05-24 |
| Milestone | M4 |
| Source | `internal/bridge/agentmeshkafka/` — `FrontierCheck.Inside()` |
| Diagnostic | Sur 120 traces, 36 (30 %) sont rejetées en `other_refusal` par `FrontierCheck` avant que les sondes de dimension soient invoquées. Ces 36 entrées correspondent aux préfixes `rf-` (18 refus frontière) et `r3-` (18 refus I3). Bien que ce comportement soit **attendu** pour les traces `rf-`, le comportement pour les traces `r3-` mérite vérification : elles devraient idéalement passer la frontière et être refusées par la garde I3, pas par la frontière. La classification actuelle les fusionne en `other_refusal`. |
| Catégorie | Biais d'instrumentation — granularité insuffisante du classifieur |
| Impact | Réduit l'échantillon utile pour l'évaluation des invariants I3 et I4. La métrique `other_refusal` agrège deux causes différentes, masquant un signal I3 potentiellement utile — **Hypothèse**. |
| Piste | M4-bis ou M5 : différencier `refus_frontiere` et `refus_i3` dans le classifieur `classifyI4()`. Ajouter un champ `refusal_cause` au JSON de résultats. |
| Statut | **Ouvert** — M4-bis ou M5 |

---

### OBS-2026-05-003 — Dépendance externe AgentMeshKafka non disponible (risque #1 réalisé)

| Champ | Valeur |
|---|---|
| Date | 2026-05-24 |
| Milestone | M4 |
| Source | Audit M4.0 (`Explore`) |
| Diagnostic | Le dépôt `agbruneau/AgentMeshKafka` est inexistant en local et sur GitHub. Cette dépendance externe est identifiée comme risque #1 dans *(PRD §18)*. Son absence force le passage en Régime B (synthétique) et empêche l'acquisition de traces réelles portant des valeurs `Context` authentiques. Ce n'est pas une observation « non modélisée » au sens strict (le risque était anticipé), mais l'impact sur la campagne est direct et nécessite une traçabilité. |
| Catégorie | Dépendance externe — risque réalisé |
| Impact | Campagne réelle reportée. Statut I4 ne peut pas progresser au-delà de *Hypothèse* en l'absence de traces réelles *(PRD §18 risque #1)*. |
| Piste | Surveiller la création du dépôt. Dès disponibilité : pivoter en Régime A, acquérir ≥ 30 traces avec D-INVARIANT mesuré. Définir un schéma d'exportation minimal pour les fixtures de test. |
| Statut | **Ouvert** — dépendance externe, hors contrôle TauGo |

---

## §3 Catégorisation

| Catégorie | Description | Exemples M4 |
|---|---|---|
| Limite d'instrumentation synthétique | Le générateur ou le harness ne produit pas les signaux nécessaires pour exercer une garde | OBS-001 |
| Biais d'instrumentation | Le classifieur ou le pipeline agrège des causes distinctes, masquant un signal | OBS-002 |
| Dépendance externe — risque réalisé | Une ressource externe prévue dans le plan est indisponible | OBS-003 |
| Hors modèle théorique | Comportement observé non prévu par les invariants I1-I5 ou les dimensions | *(aucun en M4)* |
| Régression de performance | Dégradation au-delà du seuil 5 % *(PRD §12.3)* | *(aucun en M4)* |

---

## §4 Plan d'action

| ID | Action | Responsable | Milestone | Issue |
|---|---|---|---|---|
| OBS-001 | Enrichir `cmd/generate-corpus` avec champs `Context` injectables | `ruflo-core:coder` | M4-bis | À ouvrir |
| OBS-002 | Différencier `refus_frontiere` / `refus_i3` dans `classifyI4()` | `ruflo-core:coder` | M4-bis ou M5 | À ouvrir |
| OBS-003 | Surveiller disponibilité AgentMeshKafka — pas d'action code possible | Thread principal | En continu | N/A |

---

## §5 Statut et cadence de revue

Ce registre est revu à chaque clôture de milestone *(PRD §16)*. Fréquence minimale : mensuelle.

| Revue | Date | Responsable | Observations ajoutées | Observations actées |
|---|---|---|---|---|
| Initiale (M4) | 2026-05-24 | `ruflo-core:researcher` | 3 (OBS-001 à 003) | 0 |
| M5 | À planifier | Thread principal | — | — |

---

## Renvois

- *(chap. III.8.5.4)* — Invariant I4 : cohérence contrainte
- *(chap. III.8.7)* — Anti-patrons (dont #4 : observation non modélisée passée sous silence)
- PRD §6.1 (I4), §7.2 (anti-patron #4), §16 (cadence de revue), §18 (risque #1)
- `docs/empirical/I4-report.md` — rapport principal de la campagne M4
- `docs/empirical/I4-regime.md` — décision de régime A/B
