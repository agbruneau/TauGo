# Détection de drift — Algorithme V1

**Date :** 2026-05-24
**Statut :** Probable
**Renvois :** *(PRD §11.4)* · *(PRD §7.1 C3)* · *(PRD §10 étape 3)*

---

## §1 Objectif

Un profil de calibration encode l'environnement dans lequel il a été produit
(architecture CPU, modèle LLM, corpus, date de révision). Dès que cet environnement change,
les seuils calibrés peuvent ne plus être valides. `calibration.CheckDrift` invalide le profil
de façon déterministe avant que le dispatcher ne l'utilise, en évaluant les cinq critères
PRD §11.4. L'invalidation est immédiate : aucun fallback silencieux, aucun profil périmé
toléré *(anti-patron #3, CLAUDE.md)*.

---

## §2 Les cinq critères PRD §11.4

| # | Constante Go | Champ surveillé | Condition de déclenchement |
|---|---|---|---|
| 1 | `DriftCPU` | `Profile.CPUFingerprint` | Fingerprint runtime ≠ fingerprint profilé |
| 2 | `DriftModelLLM` | `Profile.ModelLLMFingerprint` | Fingerprint LLM courant ≠ fingerprint profilé |
| 3 | `DriftCorpus` | `Profile.CorpusFingerprint` | SHA-256 corpus courant ≠ SHA-256 profilé |
| 4 | `DriftDateExpired` | `Profile.DateRevision` | `now >= DateRevision` |
| 5 | `DriftScoreDistribution` | *(placeholder V1)* | Jamais déclenché en V1 |

Le critère 4 est le seul dont l'effet est un `Refus` sans appel dans le dispatcher
*(PRD §10 étape 3)*. Les critères 1-3 et 5 déclenchent un rapport de health en V1
(log seulement ; promotion à `Refus` ou recalibration en arrière-plan est réservée à V2).

---

## §3 Algorithme

### 3.1 Logique de comparaison

`calibration.CheckDrift(current Profile, now time.Time, env Env) DriftReport`

Pour chaque critère fingerprint (1-3) :

```
si Profile.XFingerprint != "" && env.CurrentXFingerprint != Profile.XFingerprint:
    ajouter critère + message de diagnostic à DriftReport
```

La garde `!= ""` est cruciale : un profil sans fingerprint enregistré (cas premier démarrage,
`DefaultProfile()` retourne des chaînes vides pour CPU et corpus) ne déclenche pas de fausse
alarme. Le dispatcher peut démarrer proprement avant la première calibration complète.

Pour le critère date (4) :

```
si !now.Before(Profile.DateRevision):   // c.-à-d. now >= DateRevision
    ajouter DriftDateExpired
```

### 3.2 Structure de retour

```go
type DriftReport struct {
    Detected bool
    Criteria []DriftCriterion        // critères déclenchés, dans l'ordre de déclaration
    Details  map[DriftCriterion]string  // message humain pour chaque critère
}
```

`Detected = len(Criteria) > 0`. Les critères sont listés dans l'ordre de leur évaluation
(CPU → LLM → Corpus → DateExpired), ce qui rend le rapport déterministe et diffable.

### 3.3 Composition des critères

Tous les critères sont évalués indépendamment : un profil peut déclencher DriftCPU et
DriftDateExpired simultanément. Le rapport agrège l'ensemble ; l'appelant décide de l'action.
En V1, seul `DriftDateExpired` entraîne un `Refus` au niveau dispatcher *(PRD §10 étape 3)*.

---

## §4 Fingerprints V1

### 4.1 CPU — `calibration.FingerprintCPU()`

```
format : "GOOS/GOARCH"
exemple : "linux/amd64" · "darwin/arm64" · "windows/amd64"
source  : runtime.GOOS + "/" + runtime.GOARCH
```

Simplification V1 : l'identifiant d'architecture logicielle (`GOOS/GOARCH`) est suffisant pour
détecter les changements de plateforme croisée. Un fingerprint `cpuid` réel (fréquence, nombre
de coeurs logiques, jeu d'instructions) est différé à M6 si le besoin est confirmé empiriquement
*(Hypothèse)*.

### 4.2 LLM — `Profile.ModelLLMFingerprint`

La valeur est fournie par l'appelant via `llm.Client.Fingerprint()`. Le package `calibration`
n'importe pas `bridge/llm` (étanchéité Clean Architecture — `arch_test.go`). En mode stub :
`"stub:v0"`. Avec un modèle réel : l'identifiant du modèle fourni par le client concret.

`DefaultProfile()` pré-positionne `ModelLLMFingerprint = "stub:v0"` ; un changement de modèle
dans `internal/app/` doit mettre à jour ce champ lors de la création du profil.

### 4.3 Corpus — `calibration.FingerprintCorpus(jsonlPath string) (string, error)`

```
format : "sha256:<hex64>"
calcul : sha256(contenu_fichier)
```

Le même fichier produit toujours le même digest. Deux fichiers distincts produisent des digests
distincts (collision SHA-256 : négligeable). L'invalidation détecte tout ajout, suppression ou
modification d'entrée dans le corpus JSONL. Chemin vide dans `Env.CurrentCorpusFingerprint`
désactive la vérification (usage programmatique sans corpus sur disque).

---

## §5 Intégration au dispatcher

> `À vérifier` — audit 2026-05-29 (F-033/F-034) : en V1, le dispatcher **n’appelle pas** `CheckDrift` ; il réimplémente uniquement le test de date (`DateRevision`) en ligne à l’étape 3. `CheckDrift` (critères 1-3, 5) et le `slog.Warn` ci-dessous n’ont **aucun appelant de production** — ce pseudo-code décrit la cible d’intégration, non l’état câblé en V1.

Étape 3 du pseudo-algorithme PRD §10 :

```
CheckDrift(profil_courant, now, env)
    si DriftDateExpired détecté → retourner Refus("profil périmé — veille requise", trace)
    si autres critères détectés → émettre log de health (V1) ; continuer
```

Le diagnostic textuel canonique pour la péremption est `"profil périmé — veille requise"`
*(PRD §10 + §17 critère #10)*. Ce libellé est stable : le modifier sans ADR constitue une rupture
de contrat opposable.

**Critères 1, 2, 3, 5 en V1 :** le rapport est émis via `slog.Warn` mais n'interrompt pas le
traitement. La promotion à `Refus` (ou recalibration automatique en arrière-plan) est réservée à V2,
conditionnée à un ADR définissant la politique exacte *(À vérifier — choix de politique non arrêté)*.

**Veille active I3 :** la CI alerte à 30 jours avant péremption de `DateRevision`
(`today + 30 >= DateRevision` → avertissement build). Lien : CLAUDE.md §Directives #8.

---

## §6 Limites V1

| Limite | Conséquence | Horizon |
|---|---|---|
| `FingerprintCPU` grossier (GOOS/GOARCH seulement) | Ne détecte pas un changement de microarchitecture (ex. Haswell → Sapphire Rapids) | M6 si besoin empirique |
| `DriftScoreDistribution` non implémenté | Dérive statistique des scores non détectée | V2 fenêtre glissante |
| Critères 1-3 n'entraînent pas Refus | Un profil potentiellement invalide peut être utilisé | V2 politique + ADR |
| Recalibration en arrière-plan absente | Profil invalide nécessite intervention manuelle | V2 |

---

## §7 Marqueurs épistémiques

| Affirmation | Marqueur | Source |
|---|---|---|
| SHA-256 corpus détecte tout changement de fichier | Confirmé | `TestFingerprintCorpus_DifferentFilesDistinct` |
| `DriftCPU` GOOS/GOARCH suffisant en V1 | Probable | Commentaire `drift.go` ; besoin cpuid non démontré |
| Fingerprint vide → pas de fausse alarme | Confirmé | `TestCheckDrift_EmptyFingerprintsSkipped` |
| Promotion critères 1-3 à Refus pertinente en V2 | Hypothèse | Signal empirique requis avant ADR |
| Fenêtre glissante améliorera la détection de dérive statistique | À vérifier | Distribution scores non mesurée (M4 déféré) |
