# Calibration adaptative — Algorithme V1

**Date :** 2026-05-24
**Statut :** Probable
**Renvois :** *(chap. III.8.4)* · *(PRD §11)* · *(PRD §17 critère #10)*

---

## §1 Objectif

`calibration.Calibrate` produit un `Profile` reproductible byte-identique dont les `Thresholds`
maximisent l'accord entre le corpus annoté et le dispatcher simulé. Ce profil ancre les seuils
opérationnels que l'orchestrateur (`internal/orchestration/`) lit au démarrage via
`AtomicThresholds` pour décider du régime (`Deterministe | Probabiliste | Refus`).

La reproductibilité byte-identique est une exigence contractuelle : PRD §17 critère #10 exige que
`sha256(profile_1) == sha256(profile_2)` pour tout doublet `(corpus, seed, created_at,
date_revision, version_monographie)` identique. La sérialisation canonique
(`calibration.MarshalCanonical`) est le mécanisme qui satisfait cette exigence.

---

## §2 Vue d'ensemble

Pipeline complet :

```
corpus JSONL
    │
    ▼  calibration.Calibrate(corpus, seed, in Profile) Profile
Grid search déterministe (§3)
    │  ─ maximise countAgreement(corpus, t)
    │  ─ Weights passthrough V1 (§4)
    ▼
Profile (Thresholds optimisés, Weights inchangés)
    │
    ▼  calibration.MarshalCanonical(p Profile) ([]byte, error)
JSON canonique (clés triées récursivement, §5)
    │
    ▼  calibration.Store.Save(p Profile) (string, error)
Dir/<ID>-<Version>.json  +  Dir/current.json (symlink ou copie, §5)
```

---

## §3 Algorithme V1 — Grid search déterministe

### 3.1 Domaines balayés

| Paramètre | Plage | Pas | Points |
|---|---|---|---|
| `Deterministe` | [0.10, 0.90] | 0.05 | 17 |
| `HysteresisGap` | [0.05, 0.20] | 0.05 | 4 |
| `AuthBlock` | [0.70, 0.95] | 0.05 | 6 |
| `SensCoherence` | [0.30, 0.70] | 0.05 | 9 |
| `InvCoherence` | = `SensCoherence` | — | — |
| `Probabiliste` | `Deterministe + HysteresisGap` | dérivé | — |

Contrainte : `Probabiliste <= 0.95` (entrées hors borne ignorées).
Simplification V1 : `InvCoherence = SensCoherence` — réduit l'espace de recherche et maintient
la cohérence I4 symétrique. Une disjonction des deux paramètres est réservée à V2 *(Hypothèse)*.

Complexité : O(17 × 4 × 6 × 9 × N) ≈ O(3 672 × N) pour un corpus de N entrées.
En pratique, les entrées hors borne (Probabiliste > 0.95) éliminées en cours de boucle réduisent
le compte réel à environ 2 295 × N *(À vérifier selon corpus)*.

### 3.2 Encodage milli-unités

Pour éviter l'accumulation d'erreur IEEE-754 lors des boucles d'incrémentation, chaque seuil est
stocké en entier milli-unités (int64) pendant le balayage :

```
dM := 100; dM <= 900; dM += 50   // 0.10 à 0.90 par pas 0.05
gM := 50;  gM <= 200; gM += 50   // 0.05 à 0.20 par pas 0.05
aM := 700; aM <= 950; aM += 50   // 0.70 à 0.95 par pas 0.05
sM := 300; sM <= 700; sM += 50   // 0.30 à 0.70 par pas 0.05
```

La conversion retour est `fromMillis(v int64) float64 = float64(v) / 1000.0`. Ce patron est calqué
sur `FibGo bigfft/fft.go` et implémenté dans `calibration.AtomicThresholds`.

### 3.3 Score d'accord

```
score = countAgreement(corpus, t)
      = #{e ∈ corpus | simulate(e, t) == e.ExpectedRegime}
```

`simulate` applique les règles du dispatcher (PRD §10 étapes 2-7) sur les scores pré-calculés de
chaque `CorpusEntry` — sans appel LLM. Ordre des règles :

1. `refus_authority` — `AuthorityScore >= AuthBlock && !HasAttestation`
2. `refus_i4` — `SensScore < SensCoherence && InvariantScore >= InvCoherence`
3. `deterministe` — `SensScore >= Probabiliste` *(gate supérieur)*
4. `probabiliste` — `SensScore >= Deterministe` *(zone intermédiaire)*
5. `probabiliste` — défaut

### 3.4 Tie-break conservateur

En cas d'égalité de score, la combinaison rencontrée en premier gagne. L'ordre de parcours est
`(Deterministe, HysteresisGap, AuthBlock, SensCoherence)` croissant, donc la combinaison de
seuils la plus petite l'emporte. Des seuils plus petits signifient des gardes moins hautes pour
`Deterministe` et `Probabiliste`, mais le choix préserve la conservation : aucun biais en faveur
de seuils plus permissifs ne peut s'introduire par tie-break.

---

## §4 Calibration des poids — V1 passthrough

`calibration.CalibrateWeights` retourne `base Weights` inchangé (stratégie `"v1-passthrough"`).

**Justification :** les poids initiaux de `DefaultProfile()` sont marqués `Hypothèse` dans PRD §11.1.
L'expérimentation empirique M4 (`docs/empirical/I4-report.md`) n'a pas produit assez de signal pour
challenger les pondérations `DSens=0.4, DAuthority=0.3, DInvariant=0.3`. Muter les poids avant
ce signal violerait à la fois le marqueur épistémique et PRD §17 critère #10 (reproductibilité).

**Hook V2 :** `calibration.WeightHook` est le point d'extension déclaré. Une implémentation V2
(descente de gradient ou optimisation bayésienne) peut être injectée sans modifier la signature de
`calibration.Calibrate`. La contrainte : le hook ne doit pas muter `base` en place ; il doit
retourner une nouvelle valeur `Weights`. Un ADR est requis avant toute implémentation V2 *(Hypothèse)*.

---

## §5 Sérialisation canonique

### 5.1 Stratégie

`calibration.MarshalCanonical` garantit l'ordre lexicographique des clés JSON à chaque niveau
de l'arbre, indépendamment de l'ordre d'itération aléatoire des maps Go :

```
1. json.Marshal(p)                           → []byte bruts
2. json.NewDecoder(...).UseNumber().Decode() → any (nombres préservés sans perte float64)
3. sortedAny(generic)                        → sortedMap (liste ordonnée de paires clé-valeur)
4. json.NewEncoder(...).SetIndent("", "  ").SetEscapeHTML(false).Encode()
                                             → []byte canoniques avec '\n' final
```

L'étape 3 est récursive : tout `map[string]any` imbriqué est trié. Les tableaux conservent leur
ordre d'origine (pas de tri sur les éléments).

### 5.2 Contrat de sortie

- Indentation 2 espaces, clés triées lexicographiquement à tous les niveaux.
- Caractères HTML non échappés (`SetEscapeHTML(false)`).
- Octet final : `'\n'` (ajouté par `json.Encoder.Encode`).

### 5.3 Stockage

`calibration.Store.Save` écrit `Dir/<ID>-<Version>.json` (permissions `0o600`) puis met à jour
`Dir/current.json` :

- Unix/macOS : lien symbolique relatif.
- Windows (sans Developer Mode) : copie plate + sidecar `current.json.source` qui enregistre le
  nom de la cible, préservant la traçabilité que le symlink aurait offerte.

---

## §6 Reproductibilité byte-identique

**Contrat PRD §17 critère #10 :**

```
sha256(MarshalCanonical(p1)) == sha256(MarshalCanonical(p2))
si p1 et p2 ont mêmes (corpus, seed, created_at, date_revision, version_monographie)
```

**Tests gardiens :**

| Test | Fichier | Ce qu'il vérifie |
|---|---|---|
| `TestCalibrate_DeterministicSameInputSameOutput` | `calibrate_test.go` | `Thresholds` identiques entre deux appels |
| `TestMarshalCanonical_ByteIdentical` | `calibrate_test.go` | SHA-256 identique entre deux marshals du même profil |
| `TestMarshalCanonical_KeysSorted` | `calibrate_test.go` | Clés `"a_probe"` avant `"z_probe"` dans la sortie |
| `TestMarshalCanonical_RoundTripIdentity` | `calibrate_test.go` | Marshal → Unmarshal → Marshal donne le même octet |
| `TestCalibrate_PreservesWeightsV1` | `calibrate_test.go` | `Weights` non mutés par `Calibrate` |

`Store.ExportSHA256` est l'utilitaire de test pour lire le digest d'un fichier écrit sur disque.

---

## §7 Limites V1

| Limite | Impact | Horizon |
|---|---|---|
| Grille grossière (pas 0.05) | Sous-optimalité possible si l'optimum est entre deux points | V2 fine-tuning post-grid |
| `Weights` non calibrés | Pondérations dimensionnelles non validées empiriquement | M6+ après signal I4 |
| `InvCoherence = SensCoherence` | Perd la disjonction I4 bi-paramètre | V2 si signal empirique le justifie |
| Distribution scores non implémentée | Aucune détection de dérive statistique des scores | V2 fenêtre glissante (renvoi `drift.md §6`) |
| Windows : symlink fallback | `current.json` est une copie, pas un lien — pas atomique | M6 si intégrité forte requise |

---

## §8 Prochaines étapes

- **V2 — Affinement post-grid :** descente de gradient locale autour du meilleur point grid.
- **V2 — Calibration des poids :** hook `WeightHook` avec gradient ou optimisation bayésienne.
- **V2 — Fenêtre glissante :** détection statistique de dérive des scores
  (voir `docs/algorithms/drift.md §5`).
- **V3 — Optimisation bayésienne :** remplacement du grid par une acquisition probabiliste.

Chaque changement d'algorithme requiert un ADR dans `docs/adr/` avant implémentation.

---

## §9 Marqueurs épistémiques

| Affirmation | Marqueur | Source |
|---|---|---|
| Grid search converge sur le corpus disponible | Probable | `TestCalibrate_ImprovesOrMaintainsAgreement` |
| Pondérations initiales (`DSens=0.4`, …) représentatives | Hypothèse | PRD §11.1 ; M4 I4-report.md |
| `InvCoherence = SensCoherence` suffisant en V1 | Hypothèse | Simplification documentée dans `calibrate.go` |
| Reproductibilité byte-identique garantie | Confirmé | Tests gardiens §6 + `MarshalCanonical` |
| V2 gradient/bayésien améliorera l'accord | À vérifier | Signal empirique requis (M4 déféré) |
