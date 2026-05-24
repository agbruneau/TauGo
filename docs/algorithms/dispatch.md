# Algorithme de dispatch τ — V1

**Date :** 2026-05-24
**Statut :** Confirmé par construction
**Renvois :** *(chap. III.8.3 / PRD §10)*

---

## §1 Vue d'ensemble

Le dispatcher est l'orchestration centrale de TauGo. Il applique le pseudo-algorithme PRD §10.1
en séquence stricte de huit étapes, à partir d'une `Exchange` et d'un `Profile` de calibration,
pour produire une `Decision` — **toujours instrumentée**, quelque soit le régime retourné.

```
Exchange + Profile
        │
        ▼  orchestration.Dispatcher.Decide(ctx, x)
   ┌─────────────────────────────────┐
   │ 1. Frontière (FrontierCheck)    │ ──── Refus si hors domaine τ
   │ 2. Garde ontologique (I3)       │ ──── Refus si verrou D-AUTORITÉ
   │ 3. Garde péremption             │ ──── Refus si profil périmé
   │ 4. Scores 3 dimensions          │
   │ 5. Garde cohérence I4           │ ──── Refus si combinaison incohérente
   │ 6. Composite τ_score            │
   │ 7. Hystérèse → régime discret   │ ──── Deterministe | Probabiliste
   │ 8. Évaluation invariants (obs.) │
   └─────────────────────────────────┘
        │
        ▼
   Decision{Regime, Diagnostic, Trace}
```

**Positionnement dans la Clean Architecture :** le dispatcher réside dans `internal/orchestration/`
(couche `orchestration`). Il appelle vers le bas `internal/tau/` (dimensions, invariants, frontier)
et `internal/calibration/` (profil, seuils). Il ne remonte jamais vers `internal/app/` ni
vers `internal/bridge/` — étanchéité gardée par `internal/arch_test.go`. *(chap. III.8.3)*

**Entrée unique :** `tau.Exchange` — l'échange interopérabilité soumis à τ.
**Sortie unique :** `tau.Decision` — régime (`Deterministe | Probabiliste | Refus`), diagnostic
éventuel, trace immuable complète.

---

## §2 Pseudo-algorithme — 8 étapes (PRD §10.1)

```
ENTRÉE  : x Exchange, π Profile (calibration)
SORTIE  : d Decision (toujours instrumentée)

1. FRONTIÈRE              (§4.3, C1)
   ¬FrontierCheck(x).Inside() ⇒ return Refus(diag: "hors frontière τ")

2. GARDE ONTOLOGIQUE      (§4.4, I3)
   a := ScoreDAutorite(x, π)
   a.Value ≥ π.Thresholds.AuthBlock ∧ x.Attestation == nil
     ⇒ return Refus(diag: "I3 — verrou ontologique D-AUTORITÉ")

3. GARDE PÉREMPTION       (§7.1 C3, anti-patron #3)
   today > π.DateRevision ⇒ return Refus(diag: "profil périmé — veille requise")

4. SCORES                 (§5)
   s := ScoreDSens(x, π)
   i := ScoreDInvariant(x, π)

5. GARDE COHÉRENCE I4     (§6)
   i.Value ≥ π.Thresholds.InvCoherence ∧ s.Value < π.Thresholds.SensCoherence
     ⇒ return Refus(diag: "I4 — combinaison incohérente détectée")

6. COMPOSITE τ
   τ_score := π.Weights.DSens · s.Value
            + π.Weights.DAuthority · a.Value
            + π.Weights.DInvariant · i.Value

7. DÉCISION AVEC HYSTÉRÈSE (invariant : θ_d ≤ θ_p)
   τ_score ≥ θ_p              ⇒ return Probabiliste
   sinon (zone hystérèse)     ⇒ return Deterministe  (défaut M2, statut : Hypothèse)

8. ÉVALUATION INVARIANTS   (§6, observabilité pure)
   inv := EvaluateInvariants(x, decision)
   inv.AnyViolated() ⇒ trace.UnmodeledObservations += inv.Summary()
```

---

## §3 Étapes — description, garde, diagnostic, fichier Go

### Étape 1 — Frontière *(Confirmé)*

**Description :** `frontierFromExchange(x)` dérive un `tau.FrontierCheck` à partir des champs
`x.Target.DiscoveryMode` et `x.Initiator` selon l'heuristique M2. `FrontierCheck.Inside()` est
`true` si et seulement si les quatre conditions classiques sont simultanément tenues :
`UniversOuvert ∧ CompositionVariable ∧ PairProbabiliste ∧ CoutNonBorne`. *(chap. III.8.3.2)*

**Garde :** `!frontier.Inside()` → `Refus` immédiat. Anti-patron #2 : aucun bypass de
`FrontierCheck.Inside()` n'est toléré.

**Diagnostic en cas de refus :** `"hors frontière τ"`

**Fichiers Go :**
- `internal/tau/frontier.go` — type `FrontierCheck`, méthode `Inside()`
- `internal/orchestration/dispatcher.go` lignes 97-100 — appel `frontierFromExchange`, garde

**Test associé :** `TestRefusHorsFrontiere` (`internal/anti_patterns_test.go`)

---

### Étape 2 — Garde ontologique D-AUTORITÉ *(Probable)*

**Description :** `dimensions.ScoreDAuthority(ctx, x, dimensions.DefaultAuthorityWeights())`
retourne un score `[0, 1]` encodant l'asymétrie de la chaîne de délégation (Searle 1995).
Si `authScore.Value >= θ_auth_block` sans `AttestationInstitutionnelle`, l'échange est refusé
sur le verrou ontologique I3. *(chap. III.8.5.3)*

**Garde :** `authScore.Value >= d.thresholds.AuthBlock && x.AttestationInstitutionnelle == nil`

**Diagnostic en cas de refus :** `"I3 — verrou ontologique D-AUTORITÉ"`

**Fichiers Go :**
- `internal/tau/dimensions/dauthority.go` — `ScoreDAuthority`
- `internal/orchestration/dispatcher.go` lignes 103-109 — garde étape 2
- `internal/orchestration/thresholds.go` — `Thresholds.AuthBlock` (défaut : 0.85)

**Test associé :** `TestRefusOntologiqueDAUTORITE`, `TestOntologicalGuardPassesWithAttestation`
(`internal/orchestration/guards_test.go`)

---

### Étape 3 — Garde péremption *(Confirmé)*

**Description :** si un `Profile` a été fourni via `NewDispatcherWithProfile`, le dispatcher
compare la date courante (`d.now()`, horloge injectable) à `p.DateRevision`. La comparaison est
strictement `After` : `today == dateRevision` n'est pas un refus ici. En revanche,
`calibration.CheckDrift` utilise `>=` (un jour plus tôt), constituant une alerte précoce
distincte — l'asymétrie est intentionnelle (voir `docs/algorithms/drift.md`).

**Garde :** `d.profile != nil && !d.profile.DateRevision.IsZero() && d.now().After(d.profile.DateRevision)`

**Diagnostic en cas de refus :** `"profil périmé — veille requise"`

**Fichiers Go :**
- `internal/orchestration/dispatcher.go` lignes 112-119 — garde étape 3
- `internal/orchestration/dispatcher.go` lignes 47-51 — `NewDispatcherWithProfile`
- `internal/calibration/drift.go` — `CheckDrift`, critère `DriftDateExpired`

**Test associé :** `TestDispatcher_Step3_ExpiredProfileRefuses`,
`TestDispatcher_Step3_NotExpiredProceeds`, `TestDispatcher_Step3_ZeroDateRevisionSkipsCheck`,
`TestDispatcher_Step3_NilProfileSkipsCheck` (`internal/orchestration/dispatcher_expiry_test.go`),
`TestI3_DateRevisionRespectee` (`internal/anti_patterns_test.go`)

**Renvoi croisé :** `docs/algorithms/drift.md §2` — asymétrie `After` vs `>=`

---

### Étape 4 — Scores trois dimensions *(Probable)*

**Description :** D-AUTORITÉ a déjà été calculée à l'étape 2 et est réutilisée. Les deux
dimensions restantes sont calculées ici :

- `dimensions.ScoreDSens(ctx, x, dimensions.DefaultSensWeights(), d.llm)` — lieu de fixation du
  sens sémantique, [0, 1]. Fait appel au client LLM injecté pour la sonde `S_reasoner_intent`.
- `dimensions.ScoreDInvariant(ctx, x, dimensions.DefaultInvariantWeights())` — support des
  invariants d'intégration, [0, 1]. Entièrement déterministe.

**Garde :** aucune à cette étape — les scores sont calculés et transmis aux étapes suivantes.

**Fichiers Go :**
- `internal/tau/dimensions/dsens.go` — `ScoreDSens`
- `internal/tau/dimensions/dauthority.go` — `ScoreDAuthority` (calculée étape 2, réutilisée)
- `internal/tau/dimensions/dinvariant.go` — `ScoreDInvariant`
- `internal/tau/dimensions/score.go` — type `Score`, structure de poids
- `internal/orchestration/dispatcher.go` lignes 121-129 — appels étape 4

---

### Étape 5 — Garde cohérence I4 *(Hypothèse)*

**Description :** I4 détecte la rupture silencieuse *(chap. III.7)* : un D-INVARIANT élevé
combiné à un D-SENS faible signale une configuration où les invariants d'intégration exigent
un cadre que le sens sémantique ne peut pas ancrer. La combinaison est déclarée incohérente et
le dispatcher refuse plutôt que de produire une décision non fondée.

**Garde :** `invScore.Value >= d.thresholds.InvCoherence && sensScore.Value < d.thresholds.SensCoherence`
(défauts : `InvCoherence = 0.50`, `SensCoherence = 0.50`)

**Diagnostic en cas de refus :** `"I4 — combinaison incohérente détectée"`

**Fichiers Go :**
- `internal/orchestration/dispatcher.go` lignes 131-133 — garde étape 5
- `internal/tau/invariants/i4_coherence.go` — `I4CoherenceCheck` (évaluation I4 autonome)

**Test associé :** `TestI4_IncoherenceDetectee`, `TestI4_CoherentCombinationAccepted`
(`internal/orchestration/guards_test.go`)

---

### Étape 6 — Composite τ_score *(Hypothèse sur les poids)*

**Description :** les trois scores dimensionnels sont combinés en un scalaire `τ_score ∈ [0, 1]`
par somme pondérée :

```
τ_score = w_s · D_SENS + w_a · D_AUTORITÉ + w_i · D_INVARIANT
```

Les poids par défaut M2 sont `(0.4, 0.3, 0.3)` — valeurs initiales PRD §11.1,
statut `Hypothèse`, à corroborer par la calibration empirique M4-M5.

**Fichiers Go :**
- `internal/orchestration/dispatcher.go` lignes 136-139 — calcul composite
- `internal/orchestration/dispatcher.go` lignes 14-20 — `defaultDimensionWeights`
- `internal/calibration/weights.go` — type `Weights`, structure de poids calibrée

---

### Étape 7 — Hystérèse et décision discrète *(Hypothèse sur le défaut)*

**Description :** `τ_score` est mappé sur un régime discret par seuillage à double hystérèse :

```
τ_score >= θ_p  ⇒  Probabiliste
sinon           ⇒  Deterministe  (défaut M2, zone hystérèse)
```

L'invariant `θ_d ≤ θ_p` est vérifié à la construction (`Thresholds.Ordered()`) et une violation
déclenche un `panic` interne (calque `bigfft/fermat.go` de FibGo). Le défaut `Deterministe` dans
la zone d'hystérèse est un choix conservateur M2 — un mécanisme `LastRegime` basé sur
`Decision.LastRegime` est prévu en V2 pour la persistance du dernier régime connu.

**Statut du défaut :** `Hypothèse` — le choix `Deterministe` dans la bande `[θ_d, θ_p)` n'est
pas encore validé empiriquement.

**Fichiers Go :**
- `internal/orchestration/dispatcher.go` lignes 142-145 — seuillage
- `internal/orchestration/thresholds.go` — `DefaultThresholds()`, `Ordered()`
- `internal/tau/operator.go` — constantes `Deterministe`, `Probabiliste`, `Refus`

**Test associé :** `TestDispatcher_Decide_HysteresisDefaultsToDeterministe`
(`internal/orchestration/dispatcher_test.go`)

---

### Étape 8 — Évaluation des invariants *(Confirmé)*

**Description :** `invariants.EvaluateInvariants(x, decision)` est appelé après la décision de
régime. Son rôle est **exclusivement observationnel** : le `Regime` issu de l'étape 7 n'est jamais
modifié. Si des violations sont détectées, leurs résumés sont annexés à
`Trace.UnmodeledObservations`. Les violations à cette étape ne génèrent pas de `Refus` — elles
signalent des observations non modélisées pour audit ultérieur. *(chap. III.8.5)*

**Garde :** `statuses.AnyViolated()` → annexe dans `Trace.UnmodeledObservations` uniquement.

**Fichiers Go :**
- `internal/tau/invariants/evaluator.go` — `EvaluateInvariants`, `Statuses`, `AnyViolated`, `Summary`
- `internal/orchestration/dispatcher.go` lignes 159-166 — appel étape 8

**Test associé :** `TestStep8_InvariantsEvaluated_NoViolation_TraceEmpty`,
`TestStep8_InvariantsEvaluated_ViolationDetected_TraceEnriched`,
`TestStep8_RegimeUntouchedByEvaluation` (`internal/orchestration/dispatcher_invariants_test.go`)

**Renvoi croisé :** `docs/theory/05-invariants.md`

---

## §4 Ordre des étapes — non arbitraire

L'ordre des huit étapes est prescrit par PRD §10.1 ligne 539 :
`frontière → ontologie → péremption → scores → cohérence → composite → hystérèse → invariants`.
Chaque permutation brise au moins une garantie.

| Permutation hypothétique | Ce qui casse |
|---|---|
| Ontologie (2) avant frontière (1) | `ScoreDAuthority` s'exécute pour des échanges hors du domaine τ. Résultat : calcul inutile, score potentiellement non borné sur entrée non valide. C1 n'est plus le filtre de premier rang. |
| Péremption (3) avant ontologie (2) | Un profil périmé contient des seuils `AuthBlock` potentiellement obsolètes. Vérifier la péremption après l'ontologie signifie que la garde I3 a pu s'exécuter sur un `AuthBlock` invalide. |
| Scores (4) avant péremption (3) | `ScoreDSens` appelle le client LLM (réseau / I/O). Effectuer ce travail coûteux pour ensuite refuser sur péremption est un gaspillage explicitement interdit (anti-patron #3). |
| Cohérence I4 (5) avant scores (4) | Les scores `s` et `i` n'existent pas encore. La garde ne peut pas s'évaluer. |
| Composite (6) avant cohérence (5) | `τ_score` serait calculé pour une configuration incohérente. La décision hystérèse produirait un régime non fondé au lieu d'un `Refus`. |
| Hystérèse (7) avant composite (6) | `τ_score` n'existe pas. Pas de base scalaire pour le seuillage. |
| Invariants (8) avant hystérèse (7) | `EvaluateInvariants` reçoit un `Decision` incomplet (pas de `Regime`). L'évaluation I3 compare `TauScore` à `AuthBlock` — `TauScore` n'est pas encore dans la `Trace`. |

---

## §5 Trace immuable

Toute `Decision` porte une `Trace` construite par valeur (pas de pointeur partagé). Une fois
`Dispatcher.Decide` retourné, la `Trace` ne doit pas être mutée par l'appelant — son contenu
reflète l'état exact au moment de la décision.

Champs de `tau.Trace` accumulés au fil des étapes :

| Champ | Type | Rempli à l'étape |
|---|---|---|
| `ExchangeID` | `string` | 1 (garde refus) / 7 (nominal) |
| `Frontier` | `tau.FrontierCheck` | 1 |
| `Thresholds` | `tau.TraceThresholds` | 1 (snapshot immédiat) |
| `TauScore` | `float64` | 6 |
| `DurationNs` | `int64` | toutes (mis à jour à chaque `refusDecision` et au final) |
| `UnmodeledObservations` | `[]string` | 8 (si `AnyViolated()`) |

`tau.TraceThresholds` est un snapshot des seuils en vigueur au moment de l'appel — il reflète
`Thresholds.Deterministe`, `Probabiliste`, `AuthBlock`, `SensCoherence`, `InvCoherence`.

**Fichier Go :** `internal/tau/operator.go` — types `Trace`, `TraceThresholds`, `Decision`

---

## §6 Instrumentation par étape — tableau de contribution à Trace

| Étape | Régime retourné | Champs Trace renseignés | Test vérifiant la trace |
|---|---|---|---|
| 1 — Frontière | `Refus` ou poursuite | `ExchangeID`, `Frontier`, `Thresholds`, `DurationNs` | `TestDecisionAlwaysTraced` |
| 2 — Garde ontologique | `Refus` ou poursuite | idem étape 1 | `TestRefusOntologiqueDAUTORITE` |
| 3 — Garde péremption | `Refus` ou poursuite | idem étape 1 | `TestDispatcher_Step3_ExpiredProfileRefuses` |
| 4 — Scores | poursuite | (scores non exposés directement dans Trace V1) | — |
| 5 — Garde I4 | `Refus` ou poursuite | idem étape 1 | `TestI4_IncoherenceDetectee` |
| 6 — Composite | poursuite | `TauScore` | `TestDispatcher_Decide_Deterministe` |
| 7 — Hystérèse | `Deterministe` ou `Probabiliste` | `ExchangeID`, `TauScore`, `Frontier`, `Thresholds`, `DurationNs` | `TestDecisionAlwaysTraced`, `TestTraceImmutable` |
| 8 — Invariants | régime inchangé | `UnmodeledObservations` (si violation) | `TestStep8_InvariantsEvaluated_ViolationDetected_TraceEnriched` |

**Contrats vérifiés par les tests d'instrumentation :**

- `TestDecisionAlwaysTraced` : `Trace.ExchangeID == x.ID` et `DurationNs > 0` pour tout régime.
- `TestTraceImmutable` : la mutation locale de la `Trace` retournée ne se propage pas à la
  décision suivante.
- `TestRefusImpliesDiagnostic` : `Regime == Refus ⟺ Diagnostic != ""`.
- `TestUnmodeledObservationsReported` : `Statuses.Summary()` produit des lignes non vides
  ancrées sur les bons préfixes invariants (`I1`, `I3`, etc.).

---

## §7 Limites V1

### 7.1 Heuristique `frontierFromExchange` — précision partielle *(À vérifier)*

`frontierFromExchange` dérive les quatre conditions classiques par règles sur
`x.Target.DiscoveryMode` et `x.Initiator.HumanInLoop`/`DelegationDepth`. Ces règles sont
des heuristiques M2, marquées `placeholder` dans le code source
(`internal/orchestration/dispatcher.go` lignes 171-183).

Conséquence observée en M4 : environ 30 % des échanges du corpus empirique
`AgentMeshKafka` déclenchent un `Refus` à l'étape 1 via `Inside() == false`, classifiés en
`OBS-002` dans `docs/empirical/unmodeled.md`. Ces refus reflètent la limite de l'heuristique,
non une violation théorique. La calibration empirique M5 n'a pas encore affiné les règles de
`frontierFromExchange` — cela est prévu en V2.

### 7.2 Sondes coarses sur D-SENS *(Hypothèse)*

`ScoreDSens` utilise quatre sondes binaires ou quasi-binaires (`S_contract`, `S_runtime_resolve`,
`S_capability_discovery`, `S_reasoner_intent`). Les poids `DefaultSensWeights()` sont des valeurs
initiales non calibrées empiriquement. Les résultats M4 montrent que la sonde `S_reasoner_intent`
(dépendante du stub LLM en CI) domine les variations inter-échanges. V2 prévoira des sondes plus
fines. *(À vérifier par calibration M5+)*

### 7.3 Zone d'hystérèse sans persistance *(Hypothèse)*

Le défaut `Deterministe` dans la zone `[θ_d, θ_p)` est conservateur mais non validé. Le
mécanisme `LastRegime` prévu par PRD §10.1 (persistance du dernier régime pour l'`x.ID`) n'est
pas implémenté en V1 — `Decision.LastRegime` est réservé pour V2.

---

## §8 Historique par milestone

| Milestone | Étapes ajoutées | Fichiers clés livrés |
|---|---|---|
| M1 | 1 (frontière heuristique), 6 (composite), 7 (hystérèse sans persistance) | `dispatcher.go`, `frontier.go`, `operator.go` |
| M2 | 2 (garde ontologique I3), 4 (scores D-SENS, D-AUTORITÉ, D-INVARIANT), 5 (garde I4) | `dauthority.go`, `dsens.go`, `dinvariant.go`, `guards_test.go` |
| M3 | 8 (évaluation invariants, observabilité) | `evaluator.go`, `dispatcher_invariants_test.go` |
| M5 | 3 (garde péremption, `NewDispatcherWithProfile`, horloge injectable) | `dispatcher_expiry_test.go`, liens `drift.go` |

Statut de toutes les étapes listées : `Confirmé` (implémentation testée, CI verte 3 OS).

---

## §9 Anti-patrons et garde-fous

Les étapes 1, 2 et 3 correspondent directement aux trois premiers anti-patrons de CLAUDE.md :

| Anti-patron | Étape dispatcher | Garde test |
|---|---|---|
| #2 — Bypass de `FrontierCheck.Inside()` | Étape 1 | `TestRefusHorsFrontiere` |
| #3 — Profil périmé toléré | Étape 3 | `TestExpiredProfileRefuses`, `TestI3_DateRevisionRespectee` |
| #4 — Observation non modélisée passée sous silence | Étape 8 | `TestUnmodeledObservationsReported`, `TestStep8_InvariantsEvaluated_ViolationDetected_TraceEnriched` |

L'anti-patron #1 (API prédictive exportée `Predict*/Expected*/Forecast*`) est gardé
transversalement par `TestNoPredictiveAPI` (`internal/anti_patterns_test.go`) — il s'applique à
tous les packages `tau/`, `orchestration/` et non à une étape spécifique.

**Renvoi croisé :** `docs/theory/07-anti-patrons.md`

---

## §10 Renvois

- `PRD.md §10` — pseudo-algorithme canonique V1 (8 étapes, ordre, interface publique, instrumentation)
- `PRD.md §6` — invariants I1-I5 et conditions de réfutation
- `PRD.md §4` — opérateur τ formel (frontière, D-AUTORITÉ, D-SENS, D-INVARIANT)
- `PRD.md §5` — trois dimensions (D-SENS, D-AUTORITÉ, D-INVARIANT) et sondes
- `PRD.md §7.2` — anti-patrons #1-#7 et gardes
- `docs/algorithms/calibration.md` — calibration adaptative V1 (PRD §11)
- `docs/algorithms/drift.md` — détection de drift, asymétrie `After` vs `>=` (étape 3)
- `docs/theory/04-dimensions.md` — modèle formel D-SENS, D-AUTORITÉ, D-INVARIANT *(chap. III.8.4)*
- `docs/theory/05-invariants.md` — invariants I1-I5 *(chap. III.8.5)*
- `docs/theory/06-conditions-validite.md` — conditions de validité C1-C4 *(chap. III.8.6)*
- `docs/theory/07-anti-patrons.md` — anti-patrons et gardes *(chap. III.8.7)*
- `internal/orchestration/dispatcher.go` — implémentation complète de `Dispatcher.Decide`
- `internal/arch_test.go` — étanchéité Clean Architecture 4 couches
