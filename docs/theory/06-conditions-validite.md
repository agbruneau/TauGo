# 06 — Conditions de validité — renvoi vers chap. III.8.6

*Document de renvoi croisé. Le verbatim canonique vit dans `agbruneau/InteroperabiliteAgentique` v2.4.3, `Monographie.md` chap. III.8.6.*

*Statut global : Confirmé par construction (M0.5 + M5.5). Daté 2026-05-24.*

---

## Vue synoptique *(chap. III.8.6.2)*

Trois conditions C1/C2/C3 définissent la **zone opératoire** de l'opérateur τ en environnement gouverné. Hors de cette zone, `Kernel.Decide` retourne `Refus` — décision pleine, instrumentée, opposable. *(PRD §7.1)*

| # | Condition *(III.8.6.2)* | Encodage TauGo | Garde principale |
|---|---|---|---|
| **C1** | La frontière agentique est réelle (4 conditions classiques toutes violées) | `FrontierCheck.Inside()` | `TestRefusHorsFrontiere` (M0.5) |
| **C2** | D-AUTORITÉ = facteur limitant, non résolu (I3 = contrainte de conception) | `θ_auth_block` conservateur (≤ 0.85) ; refus ontologique | `FuzzI3_AsymetrieAutorite` (M3) |
| **C3** | Modèle daté et révisable (horizon 2027-2030) | `Profile.DateRevision` ; CI échoue si `today > date_revision` | `TestExpiredProfileRefuses` (M5.6) |

*Confirmé pour les trois conditions par construction.*

---

## C1 — Frontière agentique réelle *(chap. III.8.6.2 — C1)*

τ n'opère qu'à la frontière où **les quatre conditions classiques sont simultanément violées** (chap. III.8.3.2) :

| Condition classique | Violation requise |
|---|---|
| Univers de capacités clos et énumérable | Univers ouvert, capacités découvertes à l'exécution |
| Composition fixe à la conception | Composition variable à l'exécution |
| Pair non probabiliste (déterministe sous contrat) | Pair réellement probabiliste |
| Coût d'erreur borné et réversible | Coût d'erreur non borné et/ou irréversible |

Si au moins une condition classique est tenue, la frontière n'est **pas** agentique : τ ne s'applique pas et `Decide` retourne `Refus("hors frontière τ")`.

**Encodage** : `internal/tau/frontier.go`, méthode `FrontierCheck.Inside()` (M0.5). Aucun drapeau « skip » toléré — contournement = anti-patron #2 *(CLAUDE.md §Anti-patrons interdits)*.

**Garde** : `TestRefusHorsFrontiere` (M0.5) ; fuzz court `FuzzI2_Irreductibilite` (M3).

*Confirmé par construction. *(chap. III.8.6.2 — C1, PRD §7.1, §4.3)*

---

## C2 — D-AUTORITÉ facteur limitant *(chap. III.8.6.2 — C2)*

D-AUTORITÉ est un **fait institutionnel** au sens de Searle (1995) : l'autorité déléguée ne peut pas être instaurée par accord in-band entre pairs probabilistes. Dans la transition 2027-2030, D-AUTORITÉ est la dimension dont le pôle « exécution » n'a aucun support normatif stable — c'est le facteur limitant de l'agentivité gouvernée *(I3, chap. III.8.5.3)*.

**Règle opératoire** :

```
D-AUTORITÉ(x) >= θ_auth_block ∧ Attestation == nil ⇒ Refus("I3 — verrou ontologique D-AUTORITÉ")
```

- `θ_auth_block` : défaut 0.85 (`internal/calibration/thresholds.go`, M2). Marqueur : Hypothèse — non calibré empiriquement.
- `Attestation` : structure portée par `Exchange`, fournie par une institution émettrice externe. Absente par défaut.
- Le refus est **ontologique**, non configurable : aucun seuil de remplacement admis sans ADR.

**Encodage** : `internal/tau/dimensions/dauthority.go` (M2) ; garde étape 2 du dispatcher `internal/orchestration/dispatcher.go` (M1, étendu M2.5).

**Gardes** : `FuzzI3_AsymetrieAutorite` (M3) ; `TestSpaceNonFlat` (M3).

**Veille** : `Profile.DateRevision` borne I3 au 2027-01-01 ; CI alerte 30 j avant péremption *(PRD §11.4)*. Revérification si RFC d'identité agentique déléguée (IETF) ou reconnaissance juridique. *(chap. III.8.5.3)*

*Confirmé pour la garde ; Probable pour I3 lui-même (daté 2026-05-16). *(chap. III.8.6.2 — C2, PRD §7.1, §4.4)*

---

## C3 — Modèle daté et révisable *(chap. III.8.6.2 — C3)*

τ est un **instrument de navigation temporellement situé**, pas une assertion intemporelle. Toute utilisation sans date de révision explicite transforme l'instrument en dogme — anti-patron #3. *(PRD §7.2)*

**Mécanisme** :

- `Profile.DateRevision` : champ obligatoire, porté par `internal/calibration/profile.go` (M2.9).
- Étape 3 du dispatcher : `today > Profile.DateRevision ⇒ Refus("profil périmé — veille requise")`.
- Lien `docs/algorithms/drift.md` (M5) : algorithme de détection de dérive avant péremption.

**Gardes** :
- `TestExpiredProfileRefuses` : `internal/calibration/drift_test.go` + `test/e2e/calibration_determinism_test.go` (M5.6).
- `TestI3_DateRevisionRespectee` : `internal/tau/invariants/` (M3.9).

*Confirmé par construction. *(chap. III.8.6.2 — C3, PRD §7.1, §11.4)*

---

## Articulation — zone opératoire de τ *(chap. III.8.6)*

Les trois conditions définissent un espace de validité conjonctif :

```
Zone(τ) = { x | C1(x) ∧ C2(x) ∧ C3(x) }
```

Toute violation d'une condition unique suffit à déclencher `Refus`. Il n'existe pas de mode « dégradé » :

| Condition violée | Diagnostic `Refus` |
|---|---|
| C1 | `"hors frontière τ"` |
| C2 (D-AUTORITÉ ≥ θ_auth_block, sans Attestation) | `"I3 — verrou ontologique D-AUTORITÉ"` |
| C3 (profil périmé) | `"profil périmé — veille requise"` |

Renvoi croisé : `docs/theory/03-operateur-tau.md` §Domaine de non-application ; `docs/theory/05-invariants.md` §I3 ; `docs/theory/07-anti-patrons.md` §AP#2 et §AP#3.

*Confirmé pour la structure conjonctive. *(chap. III.8.6, PRD §7.1, §7.3)*

---

## Statut épistémique

| Élément | Marqueur | Justification |
|---|---|---|
| Structure des 3 conditions C1/C2/C3 | Confirmé | Verbatim monographie v2.4.3, chap. III.8.6.2 |
| C1 — frontière réelle | Confirmé | `FrontierCheck.Inside()` livré M0.5, tests verts |
| C2 — D-AUTORITÉ facteur limitant (fait institutionnel) | Probable | I3 daté 2026-05-16 ; à revérifier 2026-12-01 |
| C2 — seuil `θ_auth_block = 0.85` | Hypothèse | Valeur initiale PRD §11.1 — non calibrée empiriquement |
| C3 — datation et péremption | Confirmé | `Profile.DateRevision`, étape 3 dispatcher, CI alerte, livré M5.6 |
| Zone opératoire conjonctive | Confirmé | Déductif des conditions ; implémenté dans le dispatcher |

---

*Renvoi PRD : §7 (conditions + anti-patrons), §4.3 (frontière), §4.4 (D-AUTORITÉ), §11.4 (veille I3). Calque structurel : `docs/theory/03-operateur-tau.md`.*

**Aligné monographie** : v2.4.3 (2026-05-21).
**Daté** : 2026-05-24.
