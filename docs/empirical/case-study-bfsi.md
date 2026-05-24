# Cas d'étude — Secteur BFSI : virement interbancaire sous délégation

> **Statut** : Probable — cas synthétique pédagogique inspiré de patterns BFSI publics.
> Ce document illustre l'application du dispatcher τ à une catégorie de transactions BFSI typique.
> Il ne s'appuie sur aucun cas client réel, aucune donnée opérationnelle réelle, et aucune
> institution identifiable. Aucun montant présenté ici n'est empirique.
> Renvois : *(PRD §17 critère #1, §1.1, §4.4, §6, §10, chap. III.8)*
>
> Daté : 2026-05-24. Pour les traces empiriques réelles, voir
> `docs/empirical/I4-report.md` (M4, corpus AgentMeshKafka).

---

## §1 Contexte fictif

InstitutionΑ est une institution financière fictive opérant dans la juridiction QC-CA.
Elle déploie un assistant agentique BFSI chargé d'initier des virements interbancaires
pour le compte de clients, sous délégation d'un orchestrateur humain supervisant de
manière asynchrone.

Le cadre réglementaire fictif combine :
- **RGPD-like** (protection des données personnelles) ;
- **LCBA** — Loi Concernant le Blanchiment d'Argent (fictif/générique) : seuil
  déclaratif fixé à 7 000 CAD dans ce scénario (valeur illustrative).

L'agent agit avec `delegation_depth=2` (l'humain a délégué à un orchestrateur qui
a lui-même délégué à l'agent), sans supervision en boucle synchrone. La cible est
`InstitutionΒ`, institution partenaire distincte — franchissement de frontière
inter-organisationnelle.

Ce cas explore la garde ontologique D-AUTORITÉ *(chap. III.8.4)* dans un contexte où
l'autorité déléguée est profonde, trans-org, et où aucune attestation institutionnelle
opposable n'a été fournie.

---

## §2 Échange soumis à τ

```json
{
  "id": "vir-2026-05-24-001",
  "intent_description": "Initier un virement interbancaire de 8500 CAD vers InstitutionΒ pour client X",
  "initiator": {
    "id": "agent-bfsi-01",
    "human_in_loop": false,
    "organization": "InstitutionΑ",
    "delegation_depth": 2
  },
  "target": {
    "id": "bank-transfer-svc",
    "capability": "bank_transfer_initiation",
    "discovery_mode": 1,
    "contract_uri": ""
  },
  "attestation_institutionnelle": null,
  "context": {
    "juridiction": "QC-CA",
    "montant_cad": 8500,
    "lcba_threshold_breached": true,
    "destination_org": "InstitutionΒ"
  }
}
```

**Points saillants** :
- `human_in_loop=false` : supervision asynchrone uniquement — aucun humain dans la boucle synchrone.
- `delegation_depth=2` : chaîne humain → orchestrateur → agent.
- `discovery_mode=1` (DynamicMCP) : la capacité est résolue dynamiquement à l'exécution.
- `contract_uri=""` : aucun contrat statique préenregistré.
- `attestation_institutionnelle=null` : **point critique** — aucune attestation opposable fournie.
- `lcba_threshold_breached=true` : le montant dépasse le seuil LCBA fictif (valeur illustrative).

---

## §3 Application de τ — Trace pas-à-pas

### Étape 1 — Vérification de frontière (`FrontierCheck`)

Le dispatcher évalue `frontierFromExchange(x)` *(PRD §10, étape 1)* :

| Condition | Valeur | Motif |
|---|---|---|
| `UniversOuvert` | `true` | `DiscoveryMode=DynamicMCP != Static` |
| `CompositionVariable` | `true` | `DiscoveryMode=DynamicMCP != Static` |
| `PairProbabiliste` | `true` | `HumanInLoop=false` |
| `CoutNonBorne` | `true` | `DelegationDepth=2 > 0` |

`Inside() = true` — les quatre conditions classiques d'agentic boundary sont toutes
violées *(chap. III.8.5.1)*. L'échange est dans la frontière τ. Le dispatcher
**ne refuse pas** à cette étape.

### Étape 2 — Garde ontologique D-AUTORITÉ (I3) *(PRD §4.4, §6, chap. III.8.4)*

Le dispatcher calcule `ScoreDAuthority(x)` avec les poids par défaut PRD §5.2
(0.25 chacun — valeurs illustratives) :

| Sonde | Valeur | Motif |
|---|---|---|
| `A_chain_depth` | 0.500 | `DelegationDepth=2` → 2/4 = 0.500 |
| `A_cross_org` | 1.000 | `DelegationDepth=2 > 1` → frontière inter-org présumée |
| `A_human_anchor` | 1.000 | `HumanInLoop=false` → sonde inversée, pôle 1 |
| `A_dynamic_resolution` | 1.000 | `DynamicMCP != Static` |
| **D-AUTH** | **0.875** | `0.25 × (0.500 + 1.000 + 1.000 + 1.000)` |

`D-AUTH = 0.875 >= AuthBlock = 0.85` **ET** `AttestationInstitutionnelle = nil`.

La condition de la garde I3 est satisfaite *(PRD §6, I3)* :

> D-AUTORITÉ asymétrique (fait institutionnel — Searle 1995) ; sans
> `AttestationInstitutionnelle` → refus ontologique. *(chap. III.8.4)*

Le dispatcher retourne immédiatement **Refus I3**. Les étapes 3 à 8 ne sont pas atteintes.

---

## §4 Décision instrumentée

```json
{
  "regime": 3,
  "diagnostic": "I3 — verrou ontologique D-AUTORITÉ",
  "profile_version": "M3-default",
  "date_revision": "2026-08-16T00:00:00Z",
  "trace": {
    "exchange_id": "vir-2026-05-24-001",
    "tau_score": 0,
    "frontier": {
      "univers_ouvert": true,
      "composition_variable": true,
      "pair_probabiliste": true,
      "cout_non_borne": true
    },
    "thresholds": {
      "deterministe": 0.35,
      "probabiliste": 0.65,
      "auth_block": 0.85,
      "sens_coherence": 0.50,
      "inv_coherence": 0.50
    },
    "duration_ns": 1
  }
}
```

**Notes sur les valeurs illustratives** :
- `tau_score=0` : normal — la garde I3 est préemptive ; D-SENS et D-INVARIANT ne sont
  pas calculés (étape 2 coupe avant étape 4). Ce comportement est identique aux
  fixtures f04 de `M2-sample-decisions.md` — Confirmé.
- `date_revision="2026-08-16T00:00:00Z"` : valeur illustrative (trimestrielle à partir
  de 2026-05-16 — statut I3 Probable, veille trimestrielle). À vérifier en production.
- `duration_ns=1` : plancher garanti par `durationNs()` sur Windows (résolution timer).
- `profile_version="M3-default"` : valeur du dispatcher courant (`v0.0.x-alpha`).
  Probable — à confirmer en M4 après calibration.

---

## §5 Interprétation

τ ne prédit pas que le virement va échouer. τ refuse parce que le **fait institutionnel
est absent** *(chap. III.8.4, Searle 1995)* : l'autorité déléguée à cet agent pour
initier un virement LCBA-flaggé vers InstitutionΒ n'a pas été rendue opposable par
une attestation.

La distinction est fondamentale :

| Ce que τ n'affirme pas | Ce que τ affirme |
|---|---|
| « Ce virement va échouer » | « Je n'ai pas de fait institutionnel pour autoriser ceci » |
| « Le client est frauduleux » | « La chaîne de délégation dépasse le seuil D-AUTORITÉ sans attestation » |
| « InstitutionΒ est suspecte » | « `lcba_threshold_breached=true` aggrave l'exposition ; l'attestation est la seule sortie » |

Le refus est une **décision pleine, instrumentée, opposable** *(PRD §7.3)* — pas un
échec technique. Il invite l'institution à fournir une attestation conforme avant de
réexécuter l'échange *(voir §7 variante V1)*.

**Pourquoi D-AUTH = 0.875 franchit le seuil** : la combinaison `depth=2` (chaîne
non triviale), `cross-org` (InstitutionΑ → InstitutionΒ), `no-human-anchor` et
`dynamic-resolution` cumule 3,5 unités de contribution sur 4 possibles. Une chaîne
plus courte ou un humain en boucle abaisserait D-AUTH sous 0.85 (voir §7 variantes).

---

## §6 Anti-patrons évités

| # | Anti-patron | Situation dans ce cas | Résultat |
|---|---|---|---|
| **#2** | Bypass de `FrontierCheck.Inside()` | `Inside()=true` → frontière franchie, pas de refus prématuré | Conforme |
| **#1** | Méthode `Predict*` exportée | τ ne dit pas « ça va échouer » — il dit « je refuse car fait institutionnel absent » | Conforme |
| **#3** | Profil périmé toléré | `date_revision` vérifié à l'étape 3 ; profil valide au 2026-05-24 | Conforme (Probable) |
| **#6** | Import LLM concret dans `tau/*` | Garde I3 s'exécute avant `ScoreDSens` qui appelle le LLM stub — pas d'appel LLM réel | Conforme |

---

## §7 Cas dérivés

### V1 — Attestation présente → Probabiliste (valeurs illustratives)

Si l'opérateur humain fournit une attestation institutionnelle :

```json
"attestation_institutionnelle": {
  "emetteur": "InstitutionΑ-ComplianceΟfficer",
  "reference": "mandat-vir-2026-Q2-007",
  "marqueur": "Hypothèse",
  "asserted_at": "2026-05-24T09:00:00Z"
}
```

La garde I3 ne se déclenche pas (`Attestation != nil`). Le calcul continue :
- D-SENS (valeur illustrative) ≈ 0.880 (intent riche, DynamicMCP, pas de contrat).
- D-INV (valeur illustrative) ≈ 0.450 (médiation dynamique, sans event_registry ni idempotency).
- Garde I4 : D-INV(0.450) < InvCoherence(0.50) → ne se déclenche pas.
- τ_score ≈ 0.4×0.880 + 0.3×0.875 + 0.3×0.450 ≈ **0.750** >= 0.65 → **Probabiliste**.

Statut V1 : Probable — les sondes D-SENS dépendent du stub LLM ; à vérifier M4.

### V2 — Montant 200 CAD (LCBA non breached), depth=2, intra-org

Si le montant passe à 200 CAD et que l'opération reste intra-org
(`organization="InstitutionΑ"`, `delegation_depth=2`), le contexte change
(`lcba_threshold_breached=false`) mais les sondes D-AUTORITÉ **ne lisent pas le
contexte** directement — elles restent pilotées par `depth` et `discovery_mode`.
D-AUTH reste 0.875 (valeur illustrative). L'attestation reste requise.

Enseignement : τ évalue la **structure de délégation**, pas le contenu métier.
Un virement de 200 CAD avec la même chaîne et sans attestation reçoit le même Refus I3.
La conformité LCBA est une couche applicative en amont de τ.

Statut V2 : Hypothèse — comportement structurel, non calibré sur données réelles.

### V3 — Humain direct (delegation_depth=0, human_in_loop=true)

Si le virement est initié directement par l'humain (`delegation_depth=0`, `human_in_loop=true`) :

| Sonde | Valeur | Motif |
|---|---|---|
| `A_chain_depth` | 0.000 | `depth=0` |
| `A_cross_org` | 0.000 | `depth=0 ≤ 1` et org non vide |
| `A_human_anchor` | 0.000 | `HumanInLoop=true` → pôle 0 |
| `A_dynamic_resolution` | 0.000 ou 1.0 | Selon `discovery_mode` (Static → 0) |
| **D-AUTH** | **0.000** (Static) | Sous AuthBlock=0.85 |

Frontière : `PairProbabiliste=false` (`HumanInLoop=true`), `CoutNonBorne=false`
(`depth=0`) → `Inside()=false` → **Refus hors frontière τ** (étape 1).

τ ne s'applique pas au cas humain direct — c'est une opération hors frontière agentique.
Si `discovery_mode=DynamicMCP` et `delegation_depth=0` mais `human_in_loop=true` :
`PairProbabiliste=false` suffit à rendre `Inside()=false` (toutes les 4 conditions
doivent être vraies simultanément).

Statut V3 : Confirmé — comportement déterministe par construction booléenne.

---

## §8 Limites du cas

1. **Cas synthétique non généralisable** : aucune trace réelle d'InstitutionΑ n'existe.
   Ce cas est pédagogique ; la seule source empirique disponible est `docs/empirical/I4-report.md`
   (M4, corpus AgentMeshKafka, 5 traces). (Hypothèse)

2. **Juridiction simplifiée** : la LCBA fictive et le RGPD-like sont des simplifications
   grossières. Un déploiement réel nécessiterait une analyse juridique complète. (À vérifier)

3. **Sondes V1 coarses** : les sondes D-SENS et D-INVARIANT de la version M3 ne lisent pas
   `lcba_threshold_breached` ni `destination_org` du contexte. La connexion entre
   données métier BFSI et dimensions τ passe par le calibrage M4-M5, pas par
   l'ingénierie directe des sondes. (Hypothèse)

4. **D-AUTORITÉ insensible au montant** : comme montré en V2, τ évalue la structure de
   délégation, pas la valeur transactionnelle. Les contrôles de conformité métier
   (seuils LCBA, limites de virement) sont de la responsabilité de la couche applicative
   en amont du dispatcher. (Confirmé)

5. **LLM stub** : les valeurs illustratives D-SENS supposent le stub FNV-1a (`stub:v0`).
   Un backend LLM réel donnerait des valeurs de `S_reasoner_intent` différentes. (Hypothèse)

---

## §9 Renvois

| Document | Section |
|---|---|
| `PRD.md` | §4.4 (D-AUTORITÉ asymétrique), §6 (invariant I3), §10 (pseudo-algo 8 étapes), §17 (critères publication) |
| `docs/theory/05-invariants.md` | I3 (statut Probable, veille 2026-05-16) |
| `docs/empirical/M2-sample-decisions.md` | f04 (refus I3 analogue, D-AUTH=1.000) |
| `docs/empirical/I4-report.md` | Corpus empirique M4 (seule source empirique réelle) |
| Monographie | chap. III.8.4 (D-AUTORITÉ), chap. III.8.5 (invariants I1-I5) |

*Note* : `docs/algorithms/dispatch.md` (cible M6.3) n'était pas encore créé au moment
de la rédaction de ce document. Les 8 étapes du dispatcher sont documentées dans
`internal/orchestration/dispatcher.go` (commentaires godoc) et `PRD.md §10`.

---

## §10 Statut épistémique

| Élément | Statut | Justification |
|---|---|---|
| Mécanisme τ — garde I3 préemptive | Confirmé | Construction booléenne déterministe ; couvert par `TestRefusI3AuthBlock` et fixture f04 |
| Valeurs de seuils (AuthBlock=0.85) | Probable | Valeurs initiales PRD §11.1 ; calibration M4 non encore terminée |
| Scores D-AUTH illustratifs (0.875) | Probable | Calcul analytique exact selon implémentation M3 ; dépend des poids PRD §5.2 |
| Pertinence du scénario BFSI | Hypothèse | Synthétique ; aucune trace réelle disponible pour ce type de transaction |
| Comportement V1 (attestation levant I3) | Probable | Analogue à f06 de M2-sample-decisions ; construction booléenne `Attestation != nil` |
| Comportement V3 (hors frontière) | Confirmé | `Inside()` est booléen ; `HumanInLoop=true` → `PairProbabiliste=false` → refus étape 1 |

**Satisfait** : PRD §17 critère #1 — cas d'étude BFSI anonymisé documentant le
dispatch τ instrumenté sur un échange réaliste.

---

*BFSI case study V0.1 — 2026-05-24. Aucune institution réelle. Aucun chiffre empirique.
Marqueur global : Probable — cas synthétique pédagogique.*
