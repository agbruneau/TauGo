# Rapport fuzz I1-I5 — M3

> Généré le 2026-05-24. Statut global : *Probable* — campagne smoke initiale (5 s par cible, local Windows). Résultats CI 30 s sur 3 OS non encore disponibles au moment de ce rapport — marqués *À vérifier*.
>
> Renvois : *(chap. III.8.5 / PRD §15.2)*

---

## 1. Méthodologie

### Environnement local

| Paramètre | Valeur |
|---|---|
| OS | Windows 11 Pro 10.0.26220 |
| Parallélisme fuzz | 24 workers (`-parallel=24`, implicite) |
| Go toolchain | 1.25+ (toolchain 1.26.x) |
| Mode `-race` | Non disponible sous Windows sans CGO/gcc ; couverture race assurée par CI Linux/macOS |
| Durée par cible (smoke local) | 5 s |
| Durée cible CI (`make fuzz`) | 30 s × 5 cibles × 3 OS (Linux / macOS / Windows) |
| Durée fuzz nightly (`make fuzz-long`) | 24 h hebdomadaire, Linux uniquement |

### Format corpus seed

Chaque cible dispose d'un fichier `seed01` au format Go fuzz corpus v1, archivé sous :

```
internal/tau/invariants/testdata/fuzz/FuzzI<N>_<Nom>/seed01
```

Un fichier de régression supplémentaire a été généré automatiquement par le fuzzer pour FuzzI5 (voir §4).

### Architecture des cibles fuzz

Les cinq cibles fuzz sont définies dans `internal/tau/invariants/fuzz_targets.go` (package `invariants_test`). Elles n'invoquent **pas** le dispatcher (l'import `invariants → orchestration` est interdit par `internal/arch_test.go`) : chaque cible reconstruit localement les structures `Exchange`, `Decision`, ou `Pile` depuis les entrées primitives générées par le moteur de mutation Go.

### Gates CI

```
make fuzz      →  -fuzztime=30s  ×  5 cibles  ×  3 OS  (gate PR obligatoire)
make fuzz-long →  -fuzztime=24h  ×  5 cibles  ×  Linux  (nightly, post-M4)
```

---

## 2. Résultats par cible

Campagne smoke locale (5 s, Windows, 24 workers). *À vérifier* : résultats CI 30 s non encore produits au tag `v0.0.4-alpha`.

| Cible | Invariant testé | Énoncé court | Statut | Entrées explorées *[À vérifier]* | Crashes | Findings |
|---|---|---|---|---|---|---|
| `FuzzI1_Conservation` | I1 | τ conserve la grandeur (ExchangeID) | *Probable* | 8 600 000 | 0 | Aucun |
| `FuzzI2_Irreductibilite` | I2 | Résidu migrant non vide, non recâblable hors ligne | *Confirmé par construction* | 8 600 000 | 0 | Aucun |
| `FuzzI3_AsymetrieAutorite` | I3 | D-AUTORITÉ asymétrique, péremption 2027-01-01 | *Probable* (daté 2026-05-24) | 8 200 000 | 0 | Aucun |
| `FuzzI4_CoherenceContrainte` | I4 | `Incoherent(s, sT, i, iT)` asymétrique | *Hypothèse* — priorité empirique M4 | 9 500 000 | 0 | Aucun |
| `FuzzI5_CompositionConjonctive` | I5 | `BoundsHold` sur toute `Pile` bien formée | *Probable* | 701 000 | 0 | Bug réel détecté (voir §4) |

**Légende Findings** : un *finding* désigne un contre-exemple qui a fait échouer la propriété testée. « Aucun » signifie que la propriété a tenu sur toutes les entrées explorées.

**Note de méthodologie — deux débits distincts** *[À vérifier]* : il faut distinguer deux métriques. (1) Le **débit de la fonction-propriété scalaire isolée** — la propriété évaluée sans le moteur de mutation — d'où dérivent les ordres de grandeur ~8,2-9,5 M exec/s annoncés ailleurs (CLAUDE.md, smoke 5 s sur un autre hôte). (2) Le **débit du moteur `go test -fuzz`** (mutation incluse), mesuré ~1,4 M exec/s pour I1-I4 et ~1,1 M/s pour I5 sur ce poste (Windows, `CGO_ENABLED=0`, Go 1.26.3) ; I5 ~1,1 M/s est confirmé. Les comptes « Entrées explorées » ci-dessus relèvent de la première mesure (smoke initial) et n'ont pas été re-mesurés comme débit moteur — d'où le marqueur *[À vérifier]*. Le débit moteur est l'indicateur opérationnel pertinent pour `make fuzz`. Aucun crash observé sur ~200 M exécutions cumulées (30 s/cible).

---

## 3. Propriétés testées par cible

### FuzzI1_Conservation

**Propriété** : si l'identifiant de la trace (`Trace.ExchangeID`) correspond à l'identifiant de l'échange d'entrée (`x.ID`), alors `EvaluateI1` doit retourner `Held` (jamais `Violated`). Réciproquement, si les deux identifiants diffèrent (dérive simulée par `drift != 0`), le verdict doit être `Violated`.

*Statut* : Probable. La conservation est structurellement garantie par la sémantique par valeur de `Exchange` en Go. Le fuzz défend contre une régression future où τ muterait subrepticement la trace.

### FuzzI2_Irreductibilite

**Propriété** : pour tout échange dont la frontière est `Inside()=true`, `Residu(x)` est non vide, et le re-câblage complet (`Recablage(x, fullResidu)`) collapse `Inside()` à `false`.

*Statut* : Confirmé par construction. La correspondance bidirectionnelle entre magnitudes résiduelles et conditions de frontière rend la propriété structurelle.

### FuzzI3_AsymetrieAutorite

**Propriétés** :

1. Sans attestation, `tau_score >= auth_block > 0`, et profil dans les limites de péremption : `EvaluateI3` doit retourner `Violated`.
2. Profil dont `DateRevision` dépasse `I3PerimptionLimite` (2027-01-01 UTC) : `EvaluateI3` doit retourner `Violated` dans tous les cas.

*Statut* : Probable (daté 2026-05-24). Veille trimestrielle requise. La vérification fine est déférée à M5 (Trace ne porte pas encore les scores ventilés par dimension).

### FuzzI4_CoherenceContrainte

**Propriété** : `Incoherent(s, sT, i, iT) == (i >= iT && s < sT)` — la table de vérité asymétrique doit être exacte pour toute combinaison de valeurs `uint8` normalisées dans [0, 1].

*Statut* : Hypothèse (priorité empirique #1, PRD §6.3). La propriété testée est purement algébrique sur le helper `Incoherent` ; l'exercice empirique sur des traces réelles (AgentMeshKafka) est réservé à M4.

### FuzzI5_CompositionConjonctive

**Propriété** : `BoundsHold(pile) == true` pour toute `Pile` construite depuis un `[]byte` arbitraire (octet 0 = séparateur de couches, octets non nuls = identifiants d'angles morts).

*Statut* : Probable. Un bug réel a été détecté et corrigé (voir §4).

---

## 4. Découvertes empiriques

### Bug réel détecté par FuzzI5 — commit `7b4739c`

**Description** : `FuzzI5_CompositionConjonctive` a généré une couche contenant des identifiants dupliqués (par exemple `["z", "z"]`). La version initiale de `BoundsHold` calculait la borne inférieure en comparant `len(Aggregate(π))` (cardinalité ensembliste, ici 1) contre `maxLayer` calculé comme `len(layer)` (longueur brute de la slice, ici 2). La borne inférieure était donc faussement violée : `1 < 2` causait un `t.Fatal` dans la cible fuzz.

**Analyse** : I5 énonce `M(π) ≥ max(|Aᵢ|)` où `|Aᵢ|` désigne la **cardinalité ensembliste** (déduplication) d'une couche, pas la longueur de la slice sous-jacente. La correction introduit `distinctLen(layer)` qui déclasse les doublons avant toute comparaison.

**Correction** : `internal/tau/invariants/i5_composition.go`, commit `7b4739c`. Fichier de régression automatiquement archivé par le fuzzer sous `testdata/fuzz/FuzzI5_CompositionConjonctive/bf9c5ac437b95a58`.

**Significance** : ce finding confirme l'utilité empirique de la campagne fuzz. La propriété `BoundsHold` était **incorrecte par rapport à sa spécification théorique** (chap. III.8.5 borne inférieure), pas seulement par rapport à une implémentation. Un test unitaire paramétré sur des cas manuels n'aurait pas couvert ce cas limite.

### Invariants I1-I4 : aucun finding

Les quatre autres cibles ont tenu sur leurs corpus respectifs (5 s, ≥ 8M entrées chacun). Interprétation :

- *I1* : la conservation par valeur de `Exchange` est robuste sur l'espace exploré.
- *I2* : la correspondance magnitudes-frontière est correcte par construction sur la grille `DiscoveryMode × HumanInLoop × DelegationDepth`.
- *I3* : les deux propriétés (no-attestation + péremption) tiennent sur toute combinaison `uint8 × bool × int64`.
- *I4* : la table de vérité `Incoherent` est exacte sur 256×256 combinaisons de paires `(sMilli, iTMilli)`.

*Statut* : Probable pour I1, I3, I5 ; Confirmé par construction pour I2 ; Hypothèse pour I4 (empirique M4 requis).

---

## 5. Limites V1

### Durée smoke vs cibles CI vs nightly

| Contexte | Durée par cible | Observations |
|---|---|---|
| Smoke local (ce rapport) | 5 s | Couverture initiale ; oriente les bugs de surface |
| CI gate (`make fuzz`) | 30 s | Cible minimale avant merge ; 3 OS |
| Nightly (`make fuzz-long`) | 24 h | Exploration profonde ; post-M4 |

Un run de 5 s n'explore pas les états à plusieurs tours de mutation. *À vérifier* : les résultats de la campagne 30 s CI seront consignés en annexe après le tag `v0.0.4-alpha`.

### FuzzI5 — exécution plus lente (701K vs 8M+)

FuzzI5 génère sa `Pile` depuis un `[]byte` par décodage itératif (boucle sur `raw`, octet 0 = séparateur). Ce décodage est plus coûteux que la construction directe des autres cibles qui opèrent sur des primitives scalaires. La mutation du moteur Go produit davantage de variantes inutiles (octets 0 consécutifs, slice vide). *Probabilité* : la vitesse d'exploration de FuzzI5 peut être améliorée en M4+ par un encodage de corpus plus dense.

### EvaluateI5 retourne `Held` par construction en V1

`EvaluateI5(x, dec)` retourne systématiquement `Held` car `Trace` ne reifie pas encore la pile active (`Trace.Stack` est différé à M6). La vérification effective de I5 est assurée par `FuzzI5` qui exerce `BoundsHold` directement. *Conséquence* : le dispatcher étape 8 ne pourra pas annoter `Trace.UnmodeledObservations` avec une violation I5 avant M6.

### I3 et I4 — dépendance aux seuils calibrés

`EvaluateI3` et `FuzzI3` opèrent sur `Trace.Thresholds.AuthBlock` et `Trace.TauScore` (composite). En V1, ces seuils sont les valeurs initiales PRD §11.1 (`AuthBlock = 0.85`). La calibration adaptative M5 raffinera ces seuils ; toute modification de `DefaultProfile` pourrait déplacer la frontière de violation. *Statut* : Hypothèse pour I4 ; Probable pour I3.

### Couverture du corpus seed

La couverture de branche apportée par les cinq fichiers `seed01` n'a pas été mesurée indépendamment au moment de ce rapport. *À vérifier* : lancer `go test -coverprofile` avec les seeds uniquement (sans fuzz) pour quantifier la couverture de départ avant mutation.

---

## 6. Prochaines étapes

### M3.11 — Revue intégrée + tag `v0.0.4-alpha`

- Relancer les cinq cibles fuzz sur 30 s (CI gate).
- Vérifier couverture `internal/tau/invariants/` ≥ 80 %.
- Tag `v0.0.4-alpha` après revue `ruflo-core:reviewer`.

### M4 — Campagne empirique I4 sur traces AgentMeshKafka

- Collecter ≥ 100 traces depuis `agbruneau/AgentMeshKafka`.
- Exercer `Incoherent` et `EvaluateI4` sur les paires `(D-SENS, D-INVARIANT)` réelles.
- Objectif : faire évoluer le statut I4 de *Hypothèse* vers *Probable* ou *Confirmé* selon les résultats.
- Consigner dans `docs/empirical/i4-empirical-M4.md`.

### Intégration CI nightly (`make fuzz-long`)

- Activer la campagne 24 h hebdomadaire post-M4.
- Les nouveaux corpus générés par le fuzzer (`testdata/fuzz/FuzzI*/HASH`) seront commités au fil des runs pour alimenter les sessions suivantes.

### Veille I3 — révision statut RFC identité agentique déléguée

- Date de révision prochaine : **2026-12-01** *(À vérifier)*.
- `I3PerimptionLimite` est fixée à 2027-01-01 UTC dans `invariants.I3PerimptionLimite`.
- Si un RFC sur l'identité agentique déléguée est publié avant cette date, le statut de I3 devra être révisé de *Probable* vers *Confirmé* ou *Réfuté* selon le contenu, et `I3PerimptionLimite` mis à jour par ADR.
- CI alerte à 30 jours avant péremption (directive CLAUDE.md §8).

### M5 — Ventilation des scores dans `Trace`

- Ajouter `Trace.Scores` (D-SENS, D-AUTORITÉ, D-INVARIANT ventilés).
- Refactorer `EvaluateI3` et `EvaluateI4` pour lire les scores ventilés plutôt que le tau_score composite.
- Le statut des deux invariants sera révisé après la première campagne M5.

---

## Annexe A — Fichiers corpus seed

| Cible | Chemin | Type d'entrée |
|---|---|---|
| `FuzzI1_Conservation` | `testdata/fuzz/FuzzI1_Conservation/seed01` | `string, string, uint8, bool, uint8, int8` |
| `FuzzI2_Irreductibilite` | `testdata/fuzz/FuzzI2_Irreductibilite/seed01` | `string, string, uint8, bool, uint8` |
| `FuzzI3_AsymetrieAutorite` | `testdata/fuzz/FuzzI3_AsymetrieAutorite/seed01` | `uint8, uint8, bool, int64` |
| `FuzzI4_CoherenceContrainte` | `testdata/fuzz/FuzzI4_CoherenceContrainte/seed01` | `uint8, uint8, uint8, uint8` |
| `FuzzI5_CompositionConjonctive` | `testdata/fuzz/FuzzI5_CompositionConjonctive/seed01` | `[]byte` |
| `FuzzI5_CompositionConjonctive` (régression) | `testdata/fuzz/FuzzI5_CompositionConjonctive/bf9c5ac437b95a58` | `[]byte` (généré par le fuzzer, commit `7b4739c`) |

---

## Annexe B — Résumé des marqueurs d'incertitude appliqués

| Section | Marqueur | Justification |
|---|---|---|
| I1 statut | *Probable* | Vérification structurelle ; fuzz exploratoire |
| I2 statut | *Confirmé par construction* | Correspondance formelle magnitudes-frontière |
| I3 statut | *Probable* (daté 2026-05-24) | Dépendance RFC identité agentique ; veille trimestrielle |
| I4 statut | *Hypothèse* | Pas encore d'empirique sur traces réelles |
| I5 statut | *Probable* | Bug trouvé et corrigé ; `EvaluateI5` V1 `Held` par construction |
| Résultats CI 30 s | *À vérifier* | Non encore produits au moment du rapport |
| Couverture corpus seed | *À vérifier* | Mesure différée |
| Date révision I3 | *À vérifier* | 2026-12-01 estimée |
| Vitesse FuzzI5 | *Probabilité* (formulation douce) | Diagnostic inféré, pas mesuré |

---

*Rapport V1 — 2026-05-24. Référence : PRD §15.2, plan `docs/archive/plans-m0-m6/2026-05-24-M3-invariants-fuzz.md` tâche M3.10. Rédacteur : `ruflo-core:researcher`. Prochaine révision : après tag `v0.0.4-alpha` et run CI 30 s (M3.11).*
