# AUDITPlan.md — Plan d'exécution refactor TauGo v0.1.0 → v0.1.1

**Source** : `AUDIT.md` (2026-05-24, commit base `5a68c12`, branche `main`)
**Cible** : tag `v0.1.1` — refactor agressif autorisé, sans rupture API publique (Confirmé par AUDIT.md §19)
**Date** : 2026-05-24
**Coordinateur** : thread principal (dispatch only — *cf.* `CLAUDE.md` §Agent teams)
**Convention agents** : `ruflo-core:coder` (TDD), `ruflo-core:researcher` (théorie/ADR), `ruflo-swarm:architect` (design pré-code), `ruflo-core:reviewer` (gate final).

---

## Résumé exécutif

| Indicateur | Valeur |
|---|---|
| Tâches totales | **42** (T-001 → T-042) |
| Lots parallélisables | **7** (Lot 0 à Lot 6) |
| ADRs à produire | **4** (ADR-0006, 0007, 0008, 0009 ; ADR-0010 déferré V0.2 — *cf.* AUDIT.md §17) |
| Effort cumulé | ~12-15 jours-ingénieur (≈ 95-115 h) — alignement AUDIT.md §18 |
| Risque global | **Moyen** (impact API additif uniquement ; couverture protégée par gate CI) |
| Estimation calendaire en parallèle | **3-4 jours calendaires** sur 4-5 sous-agents `coder` concurrents |
| Anti-patrons §7.2 corrigés | #6 (P0-01) ; péremption #3 durcie (P0-02) |
| Rupture API | **Aucune** — `tau.Kernel`, `tau.Decide`, `tau.Decision`, `tau.Trace` étendus additivement (Confirmé AUDIT.md §19 footer) |

---

## Vue d'ensemble — DAG des lots

```
┌────────────────────────────────────────────────────────────────────────┐
│ Lot 0 — Purge agressive               (indépendant, immédiat)          │
│   T-001..T-004                                                          │
└──┬─────────────────────────────────────────────────────────────────────┘
   │
   ├─[parallèle]──> Lot 1 — P0 + ADRs préparatoires
   │                T-005..T-013   (R0-1, R0-2, ADR-0006..0009)
   │                       │
   │                       ▼
   ├──> Lot 2 — P1 architectural                Lot 3 — P1 quality + CI
   │     T-014..T-022 (R1, R3, R4, R5)           T-023..T-029 (R2, R6, R7, R8, R9)
   │     dépend de Lot 1                         dépend de Lot 1 (R6 indép. de Lot 1)
   │           │                                       │
   │           └──────────────┬────────────────────────┘
   │                          ▼
   │                 Lot 4 — P2 cosmétique
   │                 T-030..T-035 (R10..R15)
   │                          │
   │                          ▼
   │                 Lot 5 — P3 + docs
   │                 T-036..T-040 (R16..R19)
   │                          │
   │                          ▼
   │                 Lot 6 — Revue + CI + release v0.1.1   (sériel terminal)
   └─────────────────T-041..T-042
```

Chemin critique : `Lot 0` → `Lot 1` (R0-1, R0-2, ADR-0008) → `Lot 2` (R3 Trace ventilée) → `Lot 6`. Les autres lots sont parallèles à ce chemin.

---

## Lot 0 — Purge agressive (immédiat, indépendant)

**Objectif** : nettoyer ~10 K LOC + 1.6 MB tracked + ~7 MB orphelins avant tout refactor, conformément à AUDIT.md §16.

### T-001 — Purge Tier 1 (orphelins non trackés)
- **Source** : R20 (Tier 1)
- **Agent** : `ruflo-core:coder`
- **Description** : supprimer les artefacts orphelins listés à la racine et dans `scripts/`. Fichiers visés (Confirmé `ls` racine 2026-05-24) :
  - `cov-e2e.out`, `cov-empirical.out`, `cov-integration.out`, `cov-merged.out`, `cov-unit.out`, `cov.out`, `cov_after.out`, `cov_before.out`, `cover.out`, `cover_dims.out` (10 fichiers, ~350 KB)
  - `generate-corpus.exe`, `tau.exe` (~6.9 MB, régénérables par `make build`)
  - `scripts/__pycache__/` (bytecode Python, ~44 KB)
- **Critère succès** :
  - `git status --short` ne liste plus aucun de ces fichiers en `??` ni en modifications staged.
  - `make build` régénère `tau` sans erreur.
- **Dépendances** : aucune
- **Bloque** : T-004 (`.gitignore`)
- **Effort** : 15 min
- **Risque** : faible
- **Anti-patron / invariant** : non applicable

### T-002 — Purge Tier 2 (fichiers trackés morts)
- **Source** : R20 (Tier 2)
- **Agent** : `ruflo-core:coder`
- **Description** : `git rm` des fichiers suivants :
  - `internal/config/doc.go` (4 LOC, package vide jamais importé)
  - `internal/metrics/doc.go` (4 LOC, idem)
  - `ruvector.db` (1.6 MB, binaire SQLite trackée accidentellement commit `387c787`, exclu par PRD §3.2 — anti-RAG V1)
- **Important** : conserver `internal/testutil/doc.go` (à peupler par T-019). Conserver `internal/errors/doc.go` (à peupler par T-014).
- **Critère succès** :
  - `git status` propre après `git rm`.
  - `go build ./...` vert (aucun import résiduel — vérifier par `grep -r "internal/config" internal/ cmd/` et `grep -r "internal/metrics" internal/ cmd/`).
- **Dépendances** : aucune (peut s'exécuter en parallèle de T-001)
- **Bloque** : T-004
- **Effort** : 15 min
- **Risque** : faible

### T-003 — Archivage Tier 3 (plans M0-M6 → `docs/archive/`)
- **Source** : R20 (Tier 3)
- **Agent** : `ruflo-core:coder`
- **Description** : `git mv docs/superpowers/plans/2026-05-2[34]-M*.md docs/archive/plans-m0-m6/`. Six fichiers, 9 824 LOC totales (Confirmé AUDIT.md §16 Tier 3). Créer `docs/archive/plans-m0-m6/README.md` (≤ 30 lignes) qui pointe vers `CHANGELOG.md` pour la chronologie des milestones.
- **Critère succès** :
  - `docs/superpowers/plans/` vide ou ne contient plus que les plans actifs (M7+ si présents).
  - `docs/archive/plans-m0-m6/` contient les 6 plans + 1 README.
  - Aucun lien cassé : `grep -r "superpowers/plans/2026-05-2[34]" docs/ PRD.md CLAUDE.md PRDPlanning.md` retourne soit vide soit met à jour les renvois.
- **Dépendances** : aucune
- **Bloque** : T-004
- **Effort** : 30 min
- **Risque** : faible (archivage, pas suppression)

### T-004 — Mise à jour `.gitignore` (patterns globaux)
- **Source** : R20 (gitignore)
- **Agent** : `ruflo-core:coder`
- **Description** : ajouter au `.gitignore` les patterns suivants (déduplication si déjà présents) : `*.db`, `*.sqlite`, `*.exe`, `cov*.out`, `cover*.out`, `__pycache__/`. Vérifier que `.claude-flow/` est toujours ignoré (déjà ligne 39 selon AUDIT.md §16 Tier 1).
- **Critère succès** :
  - `git check-ignore tau.exe ruvector.db cov.out` retourne chacun avec exit 0.
  - `git status` ne ré-introduit aucun fichier purgé.
- **Dépendances** : `blockedBy: [T-001, T-002, T-003]`
- **Bloque** : T-041 (revue finale)
- **Effort** : 10 min
- **Risque** : faible

---

## Lot 1 — P0 critiques + ADRs préparatoires (parallèle entre tâches)

**Objectif** : fermer les 2 P0 bloquants (anti-patron #6 sans garde, trap péremption) et préparer les 4 ADR qui guideront le Lot 2. Les 4 ADR peuvent se rédiger en parallèle des tâches code.

### T-005 — ADR-0006 : Types valeur transverses (`internal/thresholds/`)
- **Source** : R1 (préparatoire), résout P1-01 / D1
- **Agent** : `ruflo-swarm:architect` (rédaction) + `ruflo-core:researcher` (vérification théorie)
- **Description** : produire `docs/adr/0006-types-valeur-transverses.md` selon le gabarit ADR-0005. Documenter :
  - Contexte : 3 `Thresholds` parallèles (`tau.TraceThresholds`, `orchestration.Thresholds`, `calibration.Thresholds`) ; `HysteresisGap` présent uniquement côté calibration.
  - Décision : extraire `internal/thresholds/` (package transverse, sans dépendance domaine). Type unique `thresholds.T` (ou `Thresholds`) avec tous les champs. Alias re-exportés dans chaque package consommateur pendant la transition (Hypothèse — à valider en T-014).
  - Conséquences : `arch_test.go` autorise import depuis `tau`, `orchestration`, `calibration` ; règle ajoutée empêchant `internal/thresholds` d'importer ces 3 couches (étanchéité descendante).
  - Alternatives rejetées : (a) interface partagée — trop indirecte ; (b) duplication assumée — drift démontré P1-01.
  - Statut : Accepté.
- **Critère succès** :
  - Fichier `docs/adr/0006-types-valeur-transverses.md` présent, lint markdown vert.
  - Renvois `PRD.md §8` et `arch_test.go` cités.
- **Dépendances** : aucune (parallèle T-006..T-013)
- **Bloque** : T-014
- **Effort** : 1 h
- **Risque** : faible

### T-006 — ADR-0007 : Hystérèse V1 simplifiée
- **Source** : R7, résout P1-07
- **Agent** : `ruflo-swarm:architect`
- **Description** : `docs/adr/0007-hysteresis-v1-simplifiee.md`. Documenter le choix V1 (simplification à `Deterministe` dans la bande, sans mémoire `LastRegime`). Référencer `Profile.Thresholds.HysteresisGap` non lu (Confirmé AUDIT.md §7 étape 7). Préciser que l'implémentation complète avec `LastRegime` (map concurrente x.ID→Regime + TTL) est déférée en V0.2. Renvois PRD §10 étape 7 (à amender en T-040).
- **Critère succès** : ADR rédigée avec statut "Accepté — V1 simplifiée déclarée", alternatives explicites.
- **Dépendances** : aucune
- **Bloque** : T-040 (mise à jour PRD §10.1)
- **Effort** : 1 h
- **Risque** : faible

### T-007 — ADR-0008 : Trace ventilée D-SENS / D-AUTORITÉ / D-INVARIANT
- **Source** : R3, résout P1-04 / P2-03 / P2-06 / V-A4
- **Agent** : `ruflo-swarm:architect` + `ruflo-core:researcher`
- **Description** : `docs/adr/0008-trace-ventilee-scores-dimensions.md`. Documenter l'enrichissement additif de `tau.Trace` :
  - Nouveaux champs : `DSens`, `DAuthority`, `DInvariant` (type `dimensions.Score` — préciser localisation finale du type pour éviter import cycle entre `tau` et `tau/dimensions` — *Hypothèse* à confirmer T-015).
  - Conséquences sur `EvaluateI3` (lecture directe `DAuthority` au lieu du proxy `TauScore`), `EvaluateI4` (détection bypass silencieux possible), `calibration.simulate` (lit la Trace au lieu de dupliquer `Decide` — résout P2-06).
  - Compatibilité : champs additifs ; tag `v0.1.0` reste sémantiquement valide (Confirmé AUDIT.md §19).
- **Critère succès** : ADR présente, contient un mini-snippet Go du nouveau `Trace` avec marqueurs `Probable`.
- **Dépendances** : aucune
- **Bloque** : T-015, T-016, T-017
- **Effort** : 1.5 h
- **Risque** : moyen (décision affecte 4 packages)

### T-008 — ADR-0009 : Types d'erreurs typées (`internal/errors` peuplé)
- **Source** : R2, résout P1-02
- **Agent** : `ruflo-swarm:architect`
- **Description** : `docs/adr/0009-types-erreurs-typees.md`. Spécifier les 3 types issus de PRD §14.2 : `DispatchError`, `RefusError`, `CalibrationError`. Champs minimaux : `Stage` (étape dispatcher 1-8), `Cause error`, `ExchangeID string`, `Detail string`. Implémentation `Unwrap() error` et `Error() string` standardisée. Statuer sur l'ergonomie : `errors.Is`/`errors.As` doivent fonctionner.
- **Critère succès** : ADR présente avec snippets ; le PR de T-018 référencera cette ADR.
- **Dépendances** : aucune
- **Bloque** : T-018
- **Effort** : 1 h
- **Risque** : faible

### T-009 — R0-1 : Implémenter `TestArchNoConcreteLLMInDomain`
- **Source** : R0-1 / P0-01 / V-A1 (bloquant absolu)
- **Agent** : `ruflo-core:coder`
- **Description** : ajouter dans `internal/arch_test.go` une nouvelle fonction `TestArchNoConcreteLLMInDomain` qui :
  - Walk AST sur `internal/tau/**/*.go` et `internal/orchestration/*.go` (exclure `_test.go` — calque la logique `TestBridgeNoTauImport` lignes 88-109).
  - Détecter tout import contenant l'une des substrings : `"anthropic"`, `"openai"`, `"mistralai"`, `"cohere"`, `"google.golang.org/genai"`, `"huggingface"`, `"ollama"`, `"replicate"`.
  - `t.Errorf` à chaque détection avec format `"LLM concret interdit dans le domaine: %s importe %s"`.
- **Critère succès** :
  - `go test -run TestArchNoConcreteLLMInDomain ./internal/...` PASSE (≥ 1 ligne PASS, 0 FAIL).
  - Forcer le test à échouer en ajoutant temporairement `import _ "github.com/anthropics/anthropic-sdk-go"` dans `internal/orchestration/dispatcher.go` : le test DOIT échouer (validation négative, à retirer aussitôt).
  - Anti-patron #6 désormais gardé en CI.
- **Dépendances** : aucune (parallèle T-005..T-008)
- **Bloque** : T-041
- **Effort** : 3 h
- **Risque** : faible (calque structurel disponible)
- **Anti-patron / invariant** : Anti-patron PRD §7.2 #6

### T-010 — R0-2 : Durcir `app.NewDispatcher()` contre la trap péremption
- **Source** : R0-2 / P0-02 (bloquant absolu)
- **Agent** : `ruflo-core:coder`
- **Description** : option (a) recommandée par AUDIT.md §8 — rendre `orchestration.NewDispatcher` package-private (renommer en `newDispatcherWithoutProfile`, usage interne tests uniquement) et exposer uniquement `NewDispatcherWithProfile`. Adapter `internal/app/app.go` pour fournir un profil par défaut M3 (réutiliser `calibration.DefaultProfile()` si présent, sinon le créer dans `app` avec marqueur `Hypothèse` documenté). Veiller à ce que la CLI `tau decide` instancie toujours via le chemin avec profil.
- **Critère succès** :
  - Tests existants `TestExpiredProfileRefuses` (tag `e2e`) doivent passer sans modification.
  - Ajouter un test `TestApp_NewDispatcher_ChargeProfilParDefaut` : appel CLI `tau decide` avec date système simulée > date révision profil par défaut → décision = `Refus` avec diagnostic mentionnant `profil périmé`.
  - `grep -r "orchestration.NewDispatcher(" cmd/ internal/app/` retourne 0 (sauf si renvoi vers le constructeur avec profil).
  - `go test -tags=e2e ./test/e2e/...` vert.
- **Dépendances** : aucune
- **Bloque** : T-017 (injection Profile.Weights réutilise même chemin)
- **Effort** : 1 j
- **Risque** : moyen (touche CLI ; potentiellement modifie 2-3 tests E2E)
- **Anti-patron / invariant** : Anti-patron #3 (profil périmé toléré)

### T-011 — R10 : Règle arch `calibration → tau/*` interdite
- **Source** : R10, résout V-A2 (drift silencieux potentiel)
- **Agent** : `ruflo-core:coder`
- **Description** : ajouter dans `internal/arch_test.go` la règle interdisant aux fichiers de `internal/calibration/` d'importer `internal/tau/*`, `internal/orchestration/`, ou `internal/bridge/*`. Confirmer par grep préalable que `internal/calibration/*.go` n'importe effectivement aucun de ces packages (Confirmé AUDIT.md §3.1 ligne `calibration` → aucun import taugo).
- **Critère succès** : `go test -run TestArchitectureLayering ./internal/...` vert ; ajout d'un sous-test passant pour la nouvelle règle.
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible

### T-012 — R6 : Gate CI couverture per-package ≥ 90 % `tau/*`
- **Source** : R6, résout P1-03
- **Agent** : `ruflo-core:coder`
- **Description** : modifier `.github/workflows/coverage.yml` lignes 22-27 (commentaire actuel *"90% per-package gate activates in M1+"*). Ajouter step après collecte coverage :
  ```yaml
  - name: Per-package coverage gate (tau/* >= 90%)
    run: |
      go tool cover -func=coverage.txt | grep -E "internal/tau(/|/dimensions|/invariants)/" \
        | awk '{gsub("%","",$3); if ($3+0 < 90.0) { print "FAIL: " $0; exit 1 }}'
  - name: Global coverage gate (>= 80%)
    run: |
      go tool cover -func=coverage.txt | grep total: \
        | awk '{gsub("%","",$3); if ($3+0 < 80.0) { print "FAIL global: " $3; exit 1 }}'
  ```
- **Critère succès** :
  - Workflow CI vert sur PR de cette tâche.
  - Test négatif (optionnel local) : simuler un fichier `internal/tau/foo.go` non testé pour vérifier que le gate fail (à reverter).
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible

### T-013 — R8 : Convertir `I3PerimptionLimite` en getter `func`
- **Source** : R8, résout P1-08
- **Agent** : `ruflo-core:coder`
- **Description** : dans `internal/tau/invariants/i3_authority_asymmetry.go:13`, remplacer `var I3PerimptionLimite time.Time` par `func I3PerimptionLimite() time.Time { return time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC) }` (Probable — vérifier la valeur exacte par lecture du fichier ; date `2027-01-01` mentionnée AUDIT.md §4). Mettre à jour tous les sites d'appel (recherche `grep -rn "I3PerimptionLimite" internal/ cmd/ test/`) — passer de référence variable à appel `I3PerimptionLimite()`. Retirer le `//nolint:gochecknoglobals` associé.
- **Critère succès** :
  - `go build ./...` vert.
  - `go test ./internal/tau/invariants/... -count=1` vert.
  - `grep -n "//nolint:gochecknoglobals" internal/tau/invariants/i3_authority_asymmetry.go` retourne vide.
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible
- **Anti-patron / invariant** : Anti-patron #7 (globaux mutables — renforcement)

---

## Lot 2 — P1 architectural (parallèle entre tâches, dépend de Lot 1)

**Objectif** : appliquer R1, R3, R4, R5 — extraction `thresholds`, Trace ventilée, Profile.Weights runtime, méthode `FrontierCheck`.

### T-014 — R1 : Extraire `internal/thresholds/` (package transverse)
- **Source** : R1, résout P1-01 / D1
- **Agent** : `ruflo-core:coder`
- **Description** :
  1. Créer `internal/thresholds/thresholds.go` avec struct unique :
     ```go
     type Thresholds struct {
         TauMin           float64 // seuil refus
         TauMax           float64 // seuil acceptation
         HysteresisGap    float64 // bande hystérèse (de calibration.Thresholds)
     }
     ```
  2. Remplacer les 3 définitions parallèles par alias type ou usage direct :
     - `internal/tau/operator.go:71-77` `TraceThresholds` → alias ou suppression si non exporté hors `tau`.
     - `internal/orchestration/thresholds.go:5-11` `Thresholds` → alias `= thresholds.Thresholds`.
     - `internal/calibration/profile.go:19-26` `Thresholds` → alias `= thresholds.Thresholds`.
  3. Mettre à jour `internal/arch_test.go` pour autoriser `tau`, `orchestration`, `calibration` → `internal/thresholds` (règle additive).
  4. Conserver la rétro-compatibilité : ne supprimer aucun nom exporté pour cette release ; les alias suffisent.
- **Critère succès** :
  - `go build ./...` vert.
  - `go test ./... -count=1` vert (100 % des tests existants passent sans modification).
  - `grep -rn "type Thresholds struct" internal/` retourne **1 seule** occurrence : `internal/thresholds/thresholds.go`.
  - Couverture `internal/thresholds/` ≥ 90 % (ajouter un mini test de constructor/validation si besoin).
- **Dépendances** : `blockedBy: [T-005]`
- **Bloque** : T-017
- **Effort** : 1.5 j
- **Risque** : moyen (touche 3 packages, mais alias minimisent le blast radius)
- **Anti-patron / invariant** : non applicable (qualité structurelle)

### T-015 — R3 partie 1 : Enrichir `tau.Trace` avec scores ventilés
- **Source** : R3 (Trace) / ADR-0008
- **Agent** : `ruflo-core:coder`
- **Description** : ajouter à `internal/tau/operator.go` (struct `Trace`) les champs additifs `DSens`, `DAuthority`, `DInvariant` de type `dimensions.Score` (avec tag JSON `omitempty`). Si import cycle `tau ↔ tau/dimensions` : promouvoir `Score` en `tau.Score` (type minimal `type Score float64`) et `tau/dimensions` aliase — décider en T-007 (ADR-0008) puis appliquer ici.
- **Critère succès** :
  - `go build ./...` vert.
  - JSON marshalling vérifié par test : marshalling d'une `Decision` avec ces 3 champs nuls → `omitempty` retire les clés ; non-nuls → présents.
  - `go test ./internal/tau/... -count=1` vert (≥ 100 % couverture maintenue).
- **Dépendances** : `blockedBy: [T-007]`
- **Bloque** : T-016, T-022
- **Effort** : 1 j
- **Risque** : moyen (potentiel import cycle, mitigé par ADR-0008)

### T-016 — R3 partie 2 : Peupler scores ventilés aux étapes 2/4 du dispatcher
- **Source** : R3 (peuplement runtime)
- **Agent** : `ruflo-core:coder`
- **Description** : dans `internal/orchestration/dispatcher.go` :
  - Étape 2 (l. 104-110, garde I3 ontologique) : calculer `dauthority.Score(x)` et l'assigner à `decision.Trace.DAuthority` avant le test de refus.
  - Étape 4 (l. 121-129, scores) : assigner `decision.Trace.DSens` et `decision.Trace.DInvariant` après leur calcul.
  - Mettre à jour `internal/tau/invariants/i3_authority_asymmetry.go:73-76` (`EvaluateI3`) pour lire `trace.DAuthority` directement (au lieu du proxy `TauScore`).
  - Mettre à jour `internal/tau/invariants/i4_coherence.go` `EvaluateI4` pour vérifier la cohérence ventilée (détection bypass silencieux — Confirmé AUDIT.md §9 P1-04).
- **Critère succès** :
  - Nouveau test `TestDispatcher_TraceContientScoresVentiles` : pour un Exchange déterministe, `Decide` produit `Decision.Trace.DSens > 0 && DAuthority > 0 && DInvariant > 0`.
  - Nouveau test `TestEvaluateI3_LitDAuthorityVentile` : un Exchange forgé avec `Trace.DAuthority = 0.0` doit déclencher Refus ontologique.
  - `make test` vert (`go test -race -cover ./...`).
  - Couverture `internal/orchestration` reste ≥ 91 %.
- **Dépendances** : `blockedBy: [T-015]`
- **Bloque** : T-022
- **Effort** : 1 j
- **Risque** : moyen
- **Anti-patron / invariant** : I3 (sondes I3 lisent désormais le champ ventilé), I4 (bypass détectable)

### T-017 — R4 : Injecter `Profile.Weights` dans `Dispatcher`
- **Source** : R4, résout P1-09 / P2-03
- **Agent** : `ruflo-core:coder`
- **Description** : dans `internal/orchestration/dispatcher.go:136-139` (étape 6, composite τ_score), remplacer `defaultDimensionWeights` par les poids issus du profil injecté. Signature `NewDispatcherWithProfile(profile *calibration.Profile, ...)` doit déjà exister après T-010 ; lire `profile.Weights.{SensProbes, AuthorityProbes, InvariantProbes}`. Fallback `defaultDimensionWeights` uniquement si appelé par le constructeur sans profil (chemin tests internes).
- **Critère succès** :
  - Nouveau test `TestDispatcher_AppliqueProfileWeights` : injection d'un profil custom avec poids `[0.5, 0.3, 0.2]` produit un `TauScore` différent de celui obtenu avec les poids par défaut, pour le même Exchange.
  - `make test` vert.
- **Dépendances** : `blockedBy: [T-010, T-014]`
- **Bloque** : T-022
- **Effort** : 1 j
- **Risque** : moyen

### T-018 — R2 : Peupler `internal/errors` (types d'erreurs typées)
- **Source** : R2 / ADR-0009, résout P1-02
- **Agent** : `ruflo-core:coder`
- **Description** : remplacer `internal/errors/doc.go` (4 LOC) par `internal/errors/errors.go` contenant :
  - `type DispatchError struct { Stage int; Cause error; ExchangeID string; Detail string }`
  - `type RefusError struct { Stage int; Diagnostic string; ExchangeID string }`
  - `type CalibrationError struct { ProfileVersion string; Cause error }`
  - Méthodes `Error() string` et `Unwrap() error` sur chacun.
  - Sentinels : `ErrFrontiereFranchie`, `ErrPeremptionProfile`, `ErrIncoherenceI4`.
- **Critère succès** :
  - `go test ./internal/errors/... -count=1` vert (tests sur `errors.Is`/`errors.As`).
  - Couverture package ≥ 90 %.
  - Adoption progressive : au moins 1 site dans `internal/orchestration/dispatcher.go` retourne désormais un `*errors.RefusError` au lieu d'une `errors.New` literal.
- **Dépendances** : `blockedBy: [T-008]`
- **Bloque** : T-041
- **Effort** : 1.5 j
- **Risque** : faible

### T-019 — Peupler `internal/testutil` (helper `BuildExchange`)
- **Source** : R2 (corollaire), résout D7 et partie de P1-02
- **Agent** : `ruflo-core:coder`
- **Description** : remplacer `internal/testutil/doc.go` par `internal/testutil/builders.go` :
  - `BuildExchange(opts ...Option) tau.Exchange` — construction fluide pour tests.
  - Options : `WithRegime(tau.Regime)`, `WithTrace(tau.Trace)`, `WithFrontiere(...)`, etc.
- **Critère succès** :
  - ≥ 3 tests existants migrés pour utiliser `testutil.BuildExchange` (PoC — pas migration totale).
  - Build + tests verts.
- **Dépendances** : `blockedBy: [T-002]` (conserver `doc.go` → remplacement)
- **Bloque** : T-041
- **Effort** : 1 j
- **Risque** : faible

### T-020 — R5 : Promouvoir `frontierFromExchange` → méthode `Exchange.FrontierCheck()`
- **Source** : R5, résout P1-05 / D2
- **Agent** : `ruflo-core:coder`
- **Description** : extraire la logique de `internal/orchestration/dispatcher.go:176-184` `frontierFromExchange` en méthode `(x Exchange) FrontierCheck() FrontierCheck` sur le type `tau.Exchange` (`internal/tau/operator.go`). Mettre à jour les 2 sites appelants :
  - `internal/orchestration/dispatcher.go:98-101` (étape 1)
  - `internal/tau/invariants/i2_irreductibility.go:48-68` `Recablage`
- **Critère succès** :
  - `grep -rn "frontierFromExchange" internal/` retourne 0.
  - Build + tests verts.
  - Couverture `tau` reste 100 %.
- **Dépendances** : aucune (peut s'exécuter en parallèle de T-014..T-018)
- **Bloque** : T-022
- **Effort** : 0.5 j
- **Risque** : faible

### T-021 — R9 : Drainer `errs` dans `StreamAsTauExchanges`
- **Source** : R9, résout P1-06 (goroutine leak risque)
- **Agent** : `ruflo-core:coder`
- **Description** : dans `internal/app/agentmesh.go:75-93`, ajouter le draining explicite du canal `errs` sur `ctx.Done()`.
- **Critère succès** :
  - Nouveau test `TestStreamAsTauExchanges_DrainsErrsOnContextCancel` avec `goleak.VerifyNone(t)` à `t.Cleanup`.
  - Build + tests verts.
- **Dépendances** : aucune (parallèle)
- **Bloque** : T-041
- **Effort** : 0.5 j
- **Risque** : faible

### T-022 — Test d'intégration Lot 2 : recalcul end-to-end
- **Source** : agrégation R1+R3+R4+R5
- **Agent** : `ruflo-core:coder`
- **Description** : ajouter un test E2E `TestE2E_TauDecide_TraceVentileeEtPoidsProfil` (tag `e2e`) qui :
  1. Construit un Exchange déterministe via `testutil.BuildExchange`.
  2. Charge un profil custom avec poids non-uniformes via `tau calibrate`.
  3. Exécute `tau decide` (chemin CLI complet).
  4. Vérifie que `Decision.Trace.DSens/DAuthority/DInvariant` sont peuplés et que `TauScore` reflète les poids custom (différent du score avec poids par défaut).
- **Critère succès** : test E2E vert sous `make e2e`.
- **Dépendances** : `blockedBy: [T-015, T-016, T-017, T-020]`
- **Bloque** : T-041
- **Effort** : 0.5 j
- **Risque** : faible

---

## Lot 3 — P1 quality + CI (fusionné dans Lot 1/2)

(Vide — R6/R7/R8/R9 absorbés par Lot 1 et Lot 2 pour maximiser le parallélisme.)

---

## Lot 4 — P2 cosmétique (parallèle, dépend de Lot 2)

**Objectif** : appliquer R11..R15 — méthodes `String()/MarshalJSON`, diagnostics typés, erreur typée `selectLLM`, typer `Exchange.Context`.

### T-023 — R11 : Méthodes `Regime.String()` + `MarshalJSON`
- **Source** : R11, résout D3 / P2-02 / P3-07
- **Agent** : `ruflo-core:coder`
- **Description** : ajouter à `internal/tau/operator.go` (type `Regime int`) :
  - `func (r Regime) String() string`
  - `func (r Regime) MarshalJSON() ([]byte, error)`
  - `func (r *Regime) UnmarshalJSON(b []byte) error`
  Migrer `cmd/generate-corpus/generator.go:31-39` (PascalCase) et `internal/bridge/agentmeshkafka/empirical_i4_test.go:159-169` (lowercase) pour utiliser `Regime.String()`. Décider du casing canonique (PascalCase recommandé — alignement Go standard).
- **Critère succès** :
  - Build + tests verts.
  - JSON `Decision` marshallé contient désormais `"regime": "Deterministe"` au lieu de `"regime": 2`.
  - `grep -rn "regimeString" cmd/ internal/` retourne 0.
- **Dépendances** : `blockedBy: [T-015]`
- **Bloque** : T-041
- **Effort** : 1 h
- **Risque** : faible

### T-024 — R12 : Méthodes `DiscoveryMode.String()` + `MarshalJSON`
- **Source** : R12, résout P3-06
- **Agent** : `ruflo-core:coder`
- **Description** : symétrique à T-023, pour `DiscoveryMode` (`internal/tau/operator.go:42-51`). Casing PascalCase.
- **Critère succès** : build + tests verts ; JSON contient désormais les strings.
- **Dépendances** : aucune (parallèle T-023)
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible

### T-025 — R13 : Diagnostics en constantes `tau.Diag*`
- **Source** : R13, résout P3-10
- **Agent** : `ruflo-core:coder`
- **Description** : audit `grep -rn 'Diagnostic:\s*"' internal/` pour lister tous les littéraux. Promouvoir en constantes exportées dans `internal/tau/diagnostics.go` : `DiagFrontiereFranchie`, `DiagPeremptionProfile`, `DiagVerrouOntologique`, `DiagIncoherenceI4`, `DiagObservationNonModelisee`. Remplacer les 5-10 sites d'usage.
- **Critère succès** : `grep -rn 'Diagnostic:\s*"' internal/orchestration/ internal/tau/` retourne 0.
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 1 h
- **Risque** : faible

### T-026 — R14 : Typer `Exchange.Context` (struct + bag)
- **Source** : R14, résout P2-04 (magic strings)
- **Agent** : `ruflo-core:coder`
- **Description** : remplacer `Exchange.Context map[string]any` par struct `ExchangeContext` avec champs typés selon usage actuel : `EventRegistry`, `Lineage`, `ChannelType`, etc. (audit préalable de `internal/tau/dimensions/dinvariant.go:66,78,91,111`). Conserver un champ `Bag map[string]any` pour extensions ad-hoc.
- **Critère succès** :
  - `grep -rn '"event_registry"' internal/` retourne 0.
  - Build + tests verts.
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 1 j
- **Risque** : moyen (touche `tau.Exchange` — champ public, mais ADR-0005 dit DTO bridge déjà neutre, donc pas d'impact bridge)

### T-027 — R15 : `app.selectLLM` → erreur typée
- **Source** : R15, résout P2-11
- **Agent** : `ruflo-core:coder`
- **Description** : `internal/app/app.go:26-31` — remplacer le `panic` par retour `(llm.Client, error)` avec `*errors.DispatchError{Stage: 0, Cause: errors.New("LLM provider inconnu"), Detail: providerName}`. Mettre à jour les sites appelants.
- **Critère succès** : build + tests verts ; nouveau test `TestSelectLLM_ProviderInconnuRetourneErreur` (sans panic).
- **Dépendances** : `blockedBy: [T-018]` (types erreurs disponibles)
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible

### T-028 — P2-05 : Valider `CorpusEntry.ExpectedRegime` (4 valeurs valides)
- **Source** : P2-05 (corollaire R11)
- **Agent** : `ruflo-core:coder`
- **Description** : dans `internal/calibration/calibrate.go:21-23`, après T-018 (UnmarshalJSON via Regime), ajouter validation au chargement corpus : si `entry.ExpectedRegime ∉ {Refus, Deterministe, Heuristique, Indecidable}` → erreur typée `*errors.CalibrationError`.
- **Critère succès** : nouveau test `TestCorpus_ValidateExpectedRegime_RejetteValeurInvalide` vert.
- **Dépendances** : `blockedBy: [T-023, T-018]`
- **Bloque** : T-033 (renommage `ExpectedRegime`)
- **Effort** : 30 min
- **Risque** : faible

### T-029 — P2-08 : Détection clock-jump dans `durationNs` (déférable)
- **Source** : P2-08
- **Agent** : `ruflo-core:coder`
- **Description** : `internal/orchestration/dispatcher.go:63-68` — actuellement clamp 1ns sans détection. Ajouter détection clock skew (delta négatif > 1s) → log + Trace.AnomalyClockSkew bool. Discutable si scope V0.1.1 — *Hypothèse* sur la priorité ; envisager déférer V0.2 si charge trop forte.
- **Critère succès** : si retenu, nouveau test `TestDurationNs_DetecteClockJump`. Si déféré, créer issue tracker et marquer T-029 `won't-fix-v0.1.1`.
- **Dépendances** : aucune
- **Bloque** : T-041 (si retenu)
- **Effort** : 30 min - 2 h
- **Risque** : faible

### T-030 — P2-07 : Optimiser `Aggregate` (1 passe pile I5)
- **Source** : P2-07
- **Agent** : `ruflo-core:coder`
- **Description** : `internal/tau/invariants/i5_composition.go:26-39` — `Aggregate` parcourt 2 fois la pile. Refactorer en 1 passe (accumulateur max + somme dans même boucle).
- **Critère succès** : benchmark `BenchmarkI5_Aggregate` doit montrer amélioration ≥ 10 % ; corpus fuzz `FuzzI5_CompositionConjonctive` (701 K exec/s actuel) doit rester ≥ 700 K exec/s (Confirmé AUDIT.md §4).
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible (test fuzz garde le comportement)

---

## Lot 5 — P3 + docs (parallèle final)

**Objectif** : appliquer R16..R19 et P3 cosmétiques.

### T-031 — R16 : Refactor `cmd/tau/main.go` → `runMain(args, in, out, stderr) int`
- **Source** : R16, résout P2-10
- **Agent** : `ruflo-core:coder`
- **Description** : extraire le corps de `main()` dans `func runMain(args []string, in io.Reader, out, stderr io.Writer) int`. `main()` devient :
  ```go
  func main() { os.Exit(runMain(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
  ```
  Cette structure rend `runMain` testable unitairement sans rebuild binaire.
- **Critère succès** :
  - Nouveau test `TestRunMain_DecideAvecExchangeStdin` vert.
  - Couverture `cmd/tau` ≥ 90 % (up from 76.1 %).
- **Dépendances** : aucune
- **Bloque** : T-032, T-041
- **Effort** : 1 h
- **Risque** : faible

### T-032 — R17 : `TestMain` réutilise binaire CLI buildé
- **Source** : R17, résout P2-12
- **Agent** : `ruflo-core:coder`
- **Description** : dans `cmd/tau/main_test.go`, ajouter `TestMain(m *testing.M)` qui build le binaire une seule fois en `TempDir`, puis tous les `TestEndToEnd_*` réutilisent ce binaire au lieu de rebuilder.
- **Critère succès** : `go test ./cmd/tau/... -count=1 -v` exécution ≥ 30 % plus rapide qu'avant.
- **Dépendances** : `blockedBy: [T-031]`
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible

### T-033 — R18 : Renommer `ExpectedRegime` → `LabeledRegime` (anti-patron #1 lexical)
- **Source** : R18, résout P3-12
- **Agent** : `ruflo-core:coder`
- **Description** : `internal/calibration/calibrate.go:21-23` — renommer le champ pour éviter la proximité lexicale avec anti-patron #1. Migration : maintenir une compat alias `ExpectedRegime` deprecated (JSON tag) pour ne pas casser les corpus existants ; ajouter `LabeledRegime` comme champ canonique. Plan de retrait du tag deprecated en V0.2.
- **Critère succès** :
  - `grep -rn "ExpectedRegime" internal/ cmd/` retourne uniquement le commentaire `// Deprecated:`.
  - Build + tests verts.
  - JSON corpus existant continue de désérialiser correctement.
- **Dépendances** : `blockedBy: [T-028]`
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible

### T-034 — R19 : Aligner CLAUDE.md / PRD.md sur `TestFrontierCheck_Inside_*`
- **Source** : R19, résout P3-11
- **Agent** : `ruflo-core:researcher`
- **Description** : `CLAUDE.md` §Anti-patrons (table ligne 89) et `PRD.md` §4.3 référencent encore `TestRefusHorsFrontiere` (nom obsolète). Remplacer par `TestFrontierCheck_Inside_*` (Confirmé AUDIT.md §2 anti-patron #2). Conserver un renvoi historique (footnote) pour traçabilité.
- **Critère succès** :
  - `grep -n "TestRefusHorsFrontiere" CLAUDE.md PRD.md` retourne 0 (sauf footnote historique).
  - `markdownlint` vert.
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 15 min
- **Risque** : faible

### T-035 — P3-02 : Ajouter `var _ Kernel = (*Dispatcher)(nil)`
- **Source** : P3-02
- **Agent** : `ruflo-core:coder`
- **Description** : `internal/tau/operator.go:100-102` — `tau.Kernel` est purement documentaire. Ajouter dans `internal/orchestration/dispatcher.go` la ligne d'assertion compile-time :
  ```go
  var _ tau.Kernel = (*Dispatcher)(nil)
  ```
- **Critère succès** : `go build ./...` vert ; si `Dispatcher` divergeait de `Kernel`, compile FAIL (validation négative manuelle).
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 15 min
- **Risque** : faible

### T-036 — P3-08 : `EmpiricalI4Stats.Sensitivity` → `*float64 omitempty`
- **Source** : P3-08
- **Agent** : `ruflo-core:coder`
- **Description** : `internal/bridge/agentmeshkafka/classifier.go:103-149` — remplacer la sentinelle `-1` par pointeur nil + `omitempty`.
- **Critère succès** : JSON marshallé ne contient plus `"sensitivity": -1` quand non calculé.
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible

### T-037 — P3-09 : Convertir `fuzzRefTime` global en helper
- **Source** : P3-09
- **Agent** : `ruflo-core:coder`
- **Description** : `internal/tau/invariants/fuzz_targets_test.go:106` — convertir le `var fuzzRefTime` en fonction `fuzzRefTime() time.Time`.
- **Critère succès** : `grep -n "var fuzzRefTime" internal/` retourne 0 ; fuzz test passe.
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 15 min
- **Risque** : faible

### T-038 — P3-01 : Pass typographie FR sur commentaires structurants `.go`
- **Source** : P3-01
- **Agent** : `ruflo-core:researcher`
- **Description** : audit typographique sur commentaires non-godoc (FR-CA). Vérifier guillemets `«` `»`, espaces insécables avant `:`, `;`, `?`, `!`. Godoc anglais préservé. Cibler `internal/orchestration/*.go`, `internal/calibration/*.go`.
- **Critère succès** : revue manuelle ; pas de test automatisé. PR contient diff typographique seul.
- **Dépendances** : aucune
- **Bloque** : T-041
- **Effort** : 1 h
- **Risque** : faible

### T-039 — V-A4 : Peupler `Decision.ProfileVersion` et `DateRevision` dynamiquement
- **Source** : V-A4 (AUDIT.md §3.2), corollaire R3
- **Agent** : `ruflo-core:coder`
- **Description** : `internal/orchestration/dispatcher.go:147-157` — actuellement `ProfileVersion: "M3-default"` hardcodé, `DateRevision` zéro. Lire depuis `d.profile.Version` et `d.profile.DateRevision` (disponibles après T-010). Si profil nil (chemin tests internes seulement), garder "M3-default" mais documenter le cas.
- **Critère succès** : test `TestDispatcher_DecisionContientVersionProfileEtDateRevision` vert ; Trace conforme PRD §9.1.
- **Dépendances** : `blockedBy: [T-010, T-017]`
- **Bloque** : T-041
- **Effort** : 1 h
- **Risque** : faible

### T-040 — Mise à jour PRD §10.1 pour refléter hystérèse V1 simplifiée
- **Source** : T-006 (ADR-0007) → spec
- **Agent** : `ruflo-core:researcher`
- **Description** : amender `PRD.md` §10 étape 7 pour expliciter la simplification V1 (Deterministe dans la bande, sans mémoire `LastRegime`) avec renvoi `*(cf. ADR-0007)*`. Marquer la version complète comme cible V0.2.
- **Critère succès** : `grep -n "ADR-0007" PRD.md` retourne ≥ 1 ; section §10.1 contient marqueur `Probable` sur la décision V1.
- **Dépendances** : `blockedBy: [T-006]`
- **Bloque** : T-041
- **Effort** : 30 min
- **Risque** : faible

---

## Lot 6 — Revue + tests CI + release v0.1.1 (sériel terminal)

### T-041 — Revue finale `ruflo-core:reviewer`
- **Source** : gate avant merge (CLAUDE.md §Agent teams ligne 47)
- **Agent** : `ruflo-core:reviewer`
- **Description** : revue exhaustive du diff cumulé `git diff v0.1.0..HEAD` couvrant :
  - Anti-patrons §7.2 #1-#7 — toutes les gardes opérationnelles.
  - Étanchéité Clean Arch — `arch_test.go` exécuté.
  - Couverture per-package — vérifier `tau/* ≥ 90 %`, global `≥ 80 %`.
  - Marqueurs d'incertitude (`Confirmé`/`Probable`/`Hypothèse`) sur toute affirmation datée des 4 nouveaux ADR.
  - Conventions code PRD §14 respectées.
- **Critère succès** :
  - Rapport reviewer accepté (1 PR approved par sous-agent).
  - Pas de finding sévère "Major" non résolu.
- **Dépendances** : `blockedBy: [tous T-001 à T-040]`
- **Bloque** : T-042
- **Effort** : 2-3 h
- **Risque** : moyen (le reviewer peut demander itération)

### T-042 — Release v0.1.1 (CI verts + tag + CHANGELOG)
- **Source** : critère acceptation final
- **Agent** : thread principal (intégration)
- **Description** :
  1. Exécuter localement la batterie complète (*cf.* Annexe C).
  2. Pousser sur `main`, attendre CI GitHub Actions vert (3 OS matrix, tag `ci.yml` + `coverage.yml`).
  3. Mettre à jour `CHANGELOG.md` : section `## v0.1.1 — 2026-XX-XX` listant R0-1, R0-2, R1..R19, ADRs 0006-0009, purge §16.
  4. Créer tag annoté : `git tag -a v0.1.1 -m "TauGo v0.1.1 — refactor consolidation post-audit"`.
- **Critère succès** :
  - Tag `v0.1.1` présent sur `main`.
  - CI verte sur le tag.
  - CHANGELOG à jour.
- **Dépendances** : `blockedBy: [T-041]`
- **Effort** : 1 h
- **Risque** : faible

---

## Annexe A — ADRs à créer (4 + 1 déferré)

| ID | Titre | Lot | Statut cible | Tâche |
|---|---|---|---|---|
| ADR-0006 | Types valeur transverses (`internal/thresholds`) | Lot 1 | Accepté | T-005 |
| ADR-0007 | Hystérèse V1 simplifiée | Lot 1 | Accepté — V1 déclaré | T-006 |
| ADR-0008 | Trace ventilée D-SENS / D-AUTORITÉ / D-INVARIANT | Lot 1 | Accepté | T-007 |
| ADR-0009 | Types d'erreurs typées (`internal/errors`) | Lot 1 | Accepté | T-008 |
| ADR-0010 | Bridge TauGo ↔ cia-runtime (V0.2) | — | **Déferré V0.2** | hors v0.1.1 |

Chaque ADR doit suivre le gabarit `docs/adr/0005-agentmeshkafka-dto.md` : sections **Contexte / Décision / Conséquences / Alternatives rejetées / Renvois / Statut**. Marqueurs `Confirmé`/`Probable`/`Hypothèse` sur toute affirmation datée. Renvois explicites vers PRD, CLAUDE.md, AUDIT.md.

---

## Annexe B — Tests à ajouter (par tâche)

| Test | Fichier | Propriété testée | Tâche |
|---|---|---|---|
| `TestArchNoConcreteLLMInDomain` | `internal/arch_test.go` | Anti-patron #6 — pas d'import SDK LLM dans `tau/*` ni `orchestration/*` | T-009 |
| `TestApp_NewDispatcher_ChargeProfilParDefaut` | `internal/app/app_test.go` | Anti-patron #3 durci — péremption détectée même via CLI par défaut | T-010 |
| sous-test `calibration → tau/*` | `internal/arch_test.go` | Règle V-A2 explicitée | T-011 |
| `TestThresholds_StructureUnique` | `internal/thresholds/thresholds_test.go` | Existence + champs Triplet `TauMin/TauMax/HysteresisGap` | T-014 |
| `TestDispatcher_TraceContientScoresVentiles` | `internal/orchestration/dispatcher_test.go` | Trace.DSens/DAuthority/DInvariant peuplés | T-016 |
| `TestEvaluateI3_LitDAuthorityVentile` | `internal/tau/invariants/i3_authority_asymmetry_test.go` | I3 utilise champ ventilé au lieu du proxy `TauScore` | T-016 |
| `TestDispatcher_AppliqueProfileWeights` | `internal/orchestration/dispatcher_test.go` | Poids issus du profil utilisés à l'étape 6 | T-017 |
| `TestErrors_IsAsCompatible` | `internal/errors/errors_test.go` | `errors.Is`/`errors.As` fonctionnent sur les 3 types | T-018 |
| `TestStreamAsTauExchanges_DrainsErrsOnContextCancel` | `internal/app/agentmesh_test.go` | Pas de goroutine leak via `goleak` | T-021 |
| `TestE2E_TauDecide_TraceVentileeEtPoidsProfil` (tag `e2e`) | `test/e2e/e2e_trace_test.go` | Recalcul end-to-end Trace ventilée + poids profil | T-022 |
| `TestRegime_MarshalJSON_String` | `internal/tau/operator_test.go` | Regime sérialisé en string PascalCase | T-023 |
| `TestDiscoveryMode_MarshalJSON_String` | `internal/tau/operator_test.go` | DiscoveryMode sérialisé en string | T-024 |
| `TestSelectLLM_ProviderInconnuRetourneErreur` | `internal/app/app_test.go` | Erreur typée, pas de panic | T-027 |
| `TestCorpus_ValidateExpectedRegime_RejetteValeurInvalide` | `internal/calibration/calibrate_test.go` | 4 valeurs valides imposées | T-028 |
| `TestRunMain_DecideAvecExchangeStdin` | `cmd/tau/main_test.go` | Couverture `runMain` testable directement | T-031 |
| `TestDispatcher_DecisionContientVersionProfileEtDateRevision` | `internal/orchestration/dispatcher_test.go` | V-A4 résolu | T-039 |

---

## Annexe C — Commandes de vérification finale

À exécuter avant T-042 (release) :

```bash
# 1. Propreté workspace
git status --short                          # doit être vide
git diff --stat v0.1.0..HEAD                # vérification volume diff

# 2. Build + tests unitaires + race
go build ./...
make test                                   # = go test -v -race -cover ./...

# 3. Lint
make lint                                   # = golangci-lint run ./...

# 4. Architecture
go test -run 'TestArch|TestBridgeNoTauImport' ./internal/... -count=1

# 5. Anti-patrons
go test -run 'TestNoPredictiveAPI|TestArchNoConcreteLLMInDomain|TestExpiredProfileRefuses|TestUnmodeledObservationsReported|TestRefusOntologiqueDAUTORITE|TestFrontierCheck_Inside' ./... -tags=e2e -count=1

# 6. Fuzz I1-I5 (30 s par cible)
make fuzz                                   # = go test -fuzz=. -fuzztime=30s ./internal/tau/invariants/

# 7. Couverture (gate >= 90% tau/*, >= 80% global)
make coverage
go tool cover -func=coverage.txt | grep "internal/tau/" | awk '{gsub("%","",$3); if ($3+0 < 90.0) print "FAIL: " $0}'
go tool cover -func=coverage.txt | grep total: | awk '{gsub("%","",$3); print "Global: " $3 "%"; if ($3+0 < 80.0) exit 1}'

# 8. E2E (incluant nouveaux tests Lot 2)
make e2e
make e2e-calibration

# 9. Empirique I4 (smoke)
make empirical-i4

# 10. Tag
git tag -a v0.1.1 -m "TauGo v0.1.1 — refactor consolidation post-audit (AUDIT.md 2026-05-24)"
git push --tags
```

---

## Annexe D — Critères d'acceptation release v0.1.1

- [ ] Tous les tests verts (`make test`) sur 3 OS (linux, macOS, windows)
- [ ] Lint vert (`make lint` — golangci-lint v1.64.8, 24 linters)
- [ ] Fuzz 30 s vert sur I1-I5 (`make fuzz`) — débits ≥ 700 K exec/s pour I5, ≥ 8 M pour I1-I4 (calque AUDIT.md §4)
- [ ] Couverture ≥ 80 % global, ≥ 90 % par package `internal/tau/*` (`make coverage` + gate CI T-012)
- [ ] Gate CI per-package actif sur `.github/workflows/coverage.yml` (T-012)
- [ ] Aucun anti-patron PRD §7.2 violé :
  - [ ] #1 `TestNoPredictiveAPI` vert
  - [ ] #2 `TestFrontierCheck_Inside_*` vert
  - [ ] #3 `TestExpiredProfileRefuses` + nouveau `TestApp_NewDispatcher_ChargeProfilParDefaut` verts (T-010)
  - [ ] #4 `TestUnmodeledObservationsReported` vert
  - [ ] #5 audit docs sans fabrication
  - [ ] #6 **nouveau** `TestArchNoConcreteLLMInDomain` vert (T-009)
  - [ ] #7 `gochecknoglobals` vert ; `I3PerimptionLimite` n'est plus une `var` (T-013)
- [ ] 4 ADRs créées (0006, 0007, 0008, 0009) selon gabarit ADR-0005
- [ ] CHANGELOG.md mis à jour avec section `v0.1.1`
- [ ] Tag `v0.1.1` créé et poussé
- [ ] Purge effective : `git ls-files | grep -E '\.(exe|db|sqlite)$|cov.*\.out$'` retourne vide

---

## Annexe E — Récapitulatif effort & risque

| Lot | Tâches | Effort cumulé | Risque |
|---|---|---|---|
| Lot 0 — Purge | T-001..T-004 | ~1.5 h | faible |
| Lot 1 — P0 + ADRs | T-005..T-013 | ~2 j | moyen (P0-02 touche CLI) |
| Lot 2 — P1 architectural | T-014..T-022 | ~5.5 j | moyen (Trace ventilée multi-package) |
| Lot 3 — fusionné dans Lot 1/2 | — | — | — |
| Lot 4 — P2 cosmétique | T-023..T-030 | ~3 j | faible-moyen (R14 Exchange.Context) |
| Lot 5 — P3 + docs | T-031..T-040 | ~3 j | faible |
| Lot 6 — Revue + release | T-041..T-042 | ~0.5 j | moyen |
| **Total** | **42 tâches** | **~14-15 j-ing** | **moyen** |

Parallélisable sur 4-5 `coder` simultanés → ~3-4 j calendaires.

---

*AUDITPlan.md — produit 2026-05-24 par `Plan` (architecte logiciel), intégré par le thread principal selon CLAUDE.md §Agent teams. Réviser à chaque clôture de lot. Source canonique : `AUDIT.md` 2026-05-24 commit `5a68c12`.*
