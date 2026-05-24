# Régime de campagne I4 — décision

*Daté 2026-05-24. Décision établie à M4.0 par l'agent `Explore`.*
*(PRD §18 risque #1 / PRDPlanning §M4)*

---

## §1 Régimes disponibles

| Régime | Description | Prérequis |
|---|---|---|
| **A** — Réel | Campagne sur traces acquises depuis `agbruneau/AgentMeshKafka` | Dépôt disponible, schéma événement documenté, ≥ 1 fixture exportable |
| **B** — Contingence | Campagne sur corpus synthétique généré par `cmd/generate-corpus` | Aucun (toujours disponible) |

---

## §2 Régime sélectionné M4

- [ ] Régime A — traces réelles AgentMeshKafka
- [x] **Régime B — corpus synthétique** (seed=42, profil `balanced`, 120 traces)

---

## §3 Justification

L'audit M4.0 a établi que le dépôt `agbruneau/AgentMeshKafka` est **inexistant** — ni en local ni sur GitHub au 2026-05-24. Aucune trace réelle ne peut donc être acquise. Le risque #1 *(PRD §18)* s'est matérialisé : passage automatique en Régime B.

Conséquence sur le statut I4 : le passage de *Hypothèse* à *Probable* est conditionné au Régime A *(PRDPlanning §M4)*. En Régime B, le statut I4 reste **Hypothèse** quel que soit le résultat de la campagne — une campagne synthétique inconclusive ne peut pas constituer une évidence empirique suffisante *(chap. III.8.5.4)*.

---

## §4 Conditions de bascule vers Régime A

Le régime A est activé dès que les trois conditions suivantes sont réunies :

1. Le dépôt `agbruneau/AgentMeshKafka` est créé et accessible (public ou accès direct).
2. Le schéma de l'événement Kafka est documenté et inclut les champs `Context` utilisés par les sondes D-INVARIANT (`event_registry`, `idempotency_key_mode`, profondeur de délégation).
3. Au moins une fixture de test est exportable (ou reproductible via un générateur piloté par le schéma réel).

Procédure de bascule : ouvrir un ADR compagnon (`docs/adr/`), mettre à jour ce fichier, re-exécuter le harness `TestEmpiricalI4Campaign` avec le corpus réel.

---

## §5 Tracé d'audit

| Attribut | Valeur |
|---|---|
| Date de sélection | 2026-05-24 |
| Sélectionné par | Agent `Explore` (audit M4.0) + `ruflo-core:researcher` (M4.7) |
| Corpus Régime B | `cmd/generate-corpus/testdata/synthetic-corpus-120-seed42-balanced.jsonl` |
| SHA-256 corpus | `a91d60cd9815d8183df57bfcf16bbe77d36360c4ed36e33fced9f12f70fd68ee` |
| Prochaine revue | Post-M5 (ou dès disponibilité d'AgentMeshKafka) |
| Plan de référence | `docs/superpowers/plans/2026-05-24-M4-agentmeshkafka-bridge.md` |
