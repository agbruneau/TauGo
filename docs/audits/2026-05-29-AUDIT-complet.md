# Rapport d'audit complet — TauGo (kernel de l'opérateur τ)

**Date** : 2026-05-29 · **HEAD audité** : `b94e93f` · **Branche d'origine** : `main` · **Branche de correctifs** : `audit/fixes-2026-05-29`

> Marqueurs d'incertitude (convention TauGo, `CLAUDE.md` §Conventions) : `Confirmé` · `Probable` · `Hypothèse` · `À vérifier`. Toute affirmation datée ou chiffrée non re-mesurée ce jour porte un marqueur. Renvois théoriques en italique `chap. III.8.X.Y`. Zéro fabrication : tout chiffre, numéro de ligne et nom de symbole provient d'une lecture du code ou d'un gate exécuté et capturé.

---

## Méthodologie

- **Workflow multi-agents** (préface). Cadrage parallèle (PRD, CLAUDE.md/arch_test/golangci, ADR 0001-0012, table de réconciliation de l'audit 2026-05-29, théorie chap. III.8) → fan-out de *finders* pondérés sur le cœur τ → boucle de complétude → **vérification adversariale « réfuté-par-défaut »** (1 à 3 sceptiques indépendants par finding selon la sévérité ; lentilles conformité-théorie / correctness / architecture-étanchéité) → synthèse. Un finding n'est retenu que s'il survit à la majorité des sceptiques sur le code/doc **actuel**.
- **Garde-fous d'orchestration.** 10 des 17 finders de la première vague ont échoué à émettre une sortie structurée (mode d'échec connu : un agent qui exécute beaucoup d'outils puis omet l'appel de sortie). Les dimensions affectées ont été **recouvertes** : (a) en partie par la boucle de complétude (calibration, concurrence, dérive, rétro-compat JSON, couverture), (b) par deux agents analytiques dédiés (étanchéité des imports ; qualité des tests-gardes + non-régression des correctifs récents), (c) par l'exécution directe des gates ci-dessous. La couverture des 10 dimensions de l'audit est donc complète, mais ce rapport signale explicitement ce qui a été établi par agent vs par mesure directe.
- **Baseline et gates — exécutés et capturés ce jour** (`CGO_ENABLED=0`, Go 1.26.3, Windows ; sorties capturées en fichier puis relues, conformément à la discipline de fiabilité de l'hôte) :
  - `go build -trimpath -buildvcs=true ./cmd/tau` → **exit 0** `Confirmé`.
  - `go vet ./...` → **exit 0**, 0 alerte `Confirmé`.
  - `go test -short ./...` → **14 paquets verts**, 0 FAIL `Confirmé`.
  - `go test -tags=e2e ./test/e2e/...` → **4/4 PASS** dont `TestCalibrate_GoldenCorpus_FixedHash` et `TestCalibrationDeterministic` `Confirmé`. `go test -tags=integration ./test/e2e/...` → vert `Confirmé`.
  - Couverture `go test -coverpkg=./...` → **global 88,2 %** `Confirmé (mesuré 2026-05-29)` ; per-package `tau` 100 % / `dimensions` 98,7 % / `invariants` 92,7 % → gate `tau/* ≥ 90 %` **tenu** `Confirmé`.
  - Fuzz I1-I5, `-fuzztime=15s` chacun → **0 crash, 0 nouvel intéressant** ; débit moteur observé ~1,0-1,6 M exec/s (I5 ~1,0-1,1 M/s) `Confirmé (mesuré 2026-05-29)`.
  - Benchmarks → `BenchmarkDecide` présents (Det ~737 ns/op·16 allocs, Prob ~726 ns/op·16 allocs, Refus ~25,7 ns/op·0 alloc), `BenchmarkScoreD*` ~193-198 ns/op, `BenchmarkI5_{Aggregate,BoundsHold}` présents `Confirmé`.
  - `golangci-lint run ./...` → exit 1, mais **100 % des alertes sont des faux positifs CRLF gofmt** (voir §5) : chaque blob LF committé est `gofmt`-propre.
- **`-race` NON exécuté** : `CGO_ENABLED=0` sous Windows sans gcc/clang. Tout finding de concurrence est **report-only** et listé en §4 avec la commande de validation Linux/macOS. `À vérifier — aucune data race runtime vérifiée localement.`

---

## 1. Résumé exécutif

### 1.1 Verdict global

`Confirmé` — **Le kernel τ reste sain : aucun finding critique.** Le chemin runtime de production (`app.NewDispatcher()` → `calibration.DefaultProfile()` → `Decide`) n'est affecté par **aucun** finding majeur : tous les majeurs portent sur des chemins non câblés en V1 (calibration → vrai dispatcher, validation des poids d'un `Profile` non-Default, `CheckDrift` non branché) ou sur une **garde d'architecture incomplète** sans fuite active. Les correctifs du lot post-audit 2026-05-29 (`7cb818a`, `e320e70`) **tiennent tous, sans régression** (vérifié au code, §5). Les défauts identifiés sont des **fragilités latentes** (activables hors configuration CLI standard), des **écarts entre la doctrine documentée et ce que les tests-gardes protègent réellement**, et des **incohérences documentaires**.

`Probable` — Le risque structurant n'est pas algorithmique mais **épistémique et architectural-de-garde** : (a) la chaîne calibration→décision optimise un partitionneur différent de celui qui consomme le profil (F-030/F-031) ; (b) plusieurs gardes (poids de `Profile`, dérive, sous-paquet `dimensions`→`bridge`) sont documentées comme protégées mais ne le sont pas effectivement.

### 1.2 Décompte par sévérité (findings retenus)

| Sévérité | Nombre | Identifiants |
|---|:--:|---|
| Critique | 0 | — |
| Majeur | 5 | F-026, F-030, F-031, F-033, **F-052** |
| Mineur | 34 | F-001, F-002, F-005, F-006, F-007, F-012, F-013, F-014, F-015, F-016, F-017, F-020, F-022, F-027, F-028, F-029, F-034, F-035, F-036, F-038, F-039, F-042, F-043, F-045, F-046, F-047, F-048, F-050, **F-053, F-054, F-055, F-056** |
| Informatif | 15 | F-003, F-008, F-009, F-010, F-011, F-018, F-019, F-021, F-032, F-037, F-040, F-041, F-044, F-049, F-051, **F-057** |

> Total : **54 findings retenus** (0 critique · 5 majeur · 34 mineur · 15 informatif). Les identifiants en gras (F-052..F-057) proviennent des agents analytiques de comblement (étanchéité, gardes, non-régression, couverture) ; les autres du workflow principal. `Confirmé par énumération.` *(Note : 15 identifiants apparaissent dans la ligne « informatif » car F-051 y figure ; le décompte de 15 inclut F-057.)*

### 1.3 Top risques

1. **Chaîne calibration → décision incohérente et non gardée** (F-030 majeur + F-031 majeur). `calibration.simulate()` classe `det/prob` sur `SensScore` seul, tandis que le dispatcher décide sur le composite pondéré `τ_score = 0,4·sens + 0,3·auth + 0,3·inv` puis compare aux **mêmes** seuils. Un profil produit par `tau calibrate` optimise donc un autre partitionneur que celui qui le consomme ; aucun test ne ferme la boucle jusqu'au vrai `Dispatcher.Decide`. Divergence nulle ssi `sens ≈ auth ≈ inv`. *(chap. III.8.4 ; PRD §10 étapes 6-7, §11.1)*
2. **Garde d'étanchéité incomplète sur le sous-paquet `dimensions`** (F-052 majeur, NEUF). La règle « `tau/*` → `bridge` interdit » n'est **pas** effectivement gardée pour `internal/tau/dimensions` : `archRules` ne dénie à ce sous-paquet que `tau/invariants`, et `build.Import(".../internal/tau")` ne charge jamais ses sous-paquets. Un import de bridge **concret** depuis `dimensions` passerait sous tous les tests. Pas de fuite active (l'unique import `dimensions → bridge/llm` est l'exception ADR-0003), mais la protection repose aujourd'hui sur la revue humaine, pas sur un test rouge.
3. **Invariant de cohérence des poids documenté mais non gardé** (F-026 majeur, + F-027/F-028/F-029). `Weights.Validate()` est inexistant ; les chemins de chargement disque et `NewDispatcherWithProfile` acceptent un `Profile` aux poids malformés (somme ≠ 1, poids négatif) → `τ_score` hors `[0,1]` faussant l'étape 7. Latent : aucun chemin CLI de production ne charge aujourd'hui un `Profile` disque dans le dispatcher.
4. **`CheckDrift` non branché au runtime** (F-033 majeur + F-034). Les 5 critères PRD §11.4 sont calculés mais sans appelant production ; seule la péremption par **date** atteint le runtime (réimplémentée en ligne à l'étape 3). La garde anti-patron #3 ne couvre donc que la date. `docs/algorithms/drift.md §5` survend un `slog.Warn` de santé qui n'existe pas (anti-patron #5 documentaire).
5. **Angle mort du détecteur I3 sur régime `Deterministe`** (F-006 mineur). `EvaluateI3WithClock` n'applique la détection de bypass ontologique que dans la branche `Probabiliste` ; un `Deterministe` portant `DAuthority.Value ≥ AuthBlock` sans attestation passerait `Held` si l'étape 2 du dispatcher était contournée. *(chap. III.8.4.2.bis / III.8.5.3)*

### 1.4 Comparaison avec l'audit 2026-05-29 (neuf / déjà connu / déjà résolu)

- **Findings NEUFS de ce lot** (sans antécédent dans l'audit 2026-05-29) : F-002, F-005, F-006, F-007, F-008, F-009, F-010, F-011, F-013, F-014, F-015, F-016, F-017, F-020, F-026, F-027, F-028, F-029, F-030, F-031, F-039, F-040, F-041, F-042, F-043, F-044, F-045, F-046, F-047, F-048, F-049, F-051, **F-052, F-053, F-054, F-055, F-056, F-057**. Les plus structurants : **F-030/F-031** (divergence calibration↔dispatcher) et **F-052** (garde d'étanchéité incomplète sur `dimensions`), jamais relevés auparavant.
- **Findings ÉTENDANT un item déjà connu** : F-001 (parent I2-06), F-004 (P4-01), F-012 (Q5-02), F-018 (A6-02), F-022 (R3-03), F-033/F-034 (placeholder critère drift), F-035/F-036 (distincts de C1-01/C1-02), F-050 (R3-01), F-007 (I2-05, volet code non réconcilié).
- **Items 2026-05-29 confirmés déjà-résolus** (vérifiés au code, **aucune régression** — voir §5) : **C1-01** (ADR-0012, délégation à `calibration.LoadCorpus` + exit≠0 sur corpus invalide), **C1-04** (message `LabeledRegime`), **I2-03** (test I4 renommé), **I2-04** (constante `tau.DiagFrontiereFranchie`), **Q5-01** (`RefusError.Is` + cas négatifs prouvés), **A6-03** (`internal/config`+`internal/metrics` supprimés), **A6-04** (règles `from` défensives `errors`/`testutil`), **R3-01** (godoc lossy `errOut`), **R3-02** (docstring `SetTuning` non survendue), **A6-01**/**P4-01** (survente couverture/débit corrigée).
- **Réserve héritée non re-examinée** : I2-02 (timestamp horloge-murale dans `testdata/empirical-i4-results.json`, sous tag `empirical`) reste signalé pour mémoire. `À vérifier — hors périmètre des findings retenus.`

---

## 2. Tableau sévérité × dimension

| Dimension d'audit | Critique | Majeur | Mineur | Informatif |
|---|:--:|:--:|:--:|:--:|
| 1. Conformité théorie ↔ code (τ, frontière, 5 Refus) | — | — | F-001, F-002, F-045 | F-003 |
| 1. Invariants I2 / I3 / I5 | — | — | F-005, F-006, F-007 | F-008 |
| 1. Dimensions D-SENS/D-AUTORITÉ/D-INVARIANT | — | — | F-009 | F-010, F-011 |
| 1. Orchestration / dispatcher (8 étapes) | — | F-030 | F-012, F-013, F-014, F-022, F-027, F-028 | F-032 |
| 2. Anti-patrons & gardes | — | — | F-055, F-056 | — |
| 3. Architecture & étanchéité | — | **F-052** | F-053 | F-054 |
| 4. Intégrité documentaire | — | — | F-004, F-015, F-016, F-017, F-034 | F-018, F-019 |
| 5. Calibration & profil / dérive | — | F-026, F-031, F-033 | F-029, F-035, F-038, F-039 | F-021, F-040, F-041, F-057 |
| 6. Correctness & robustesse (CLI, enums, erreurs) | — | — | F-036, F-042, F-043 | F-037, F-044 |
| 7. Concurrence & sécurité mémoire | — | — | F-020, F-050 | — |
| 8. Tests, couverture & fuzz | — | — | F-046, F-047, F-048 | F-049, F-051 |
| 9. Performance | — | — | — | *(cf. §Annexe gates — P4-02 partiellement comblé)* |
| 10. Idiomatique Go & lint | — | — | — | *(propre ; faux positifs CRLF, §5)* |
| **Total** | **0** | **5** | **34** | **15** |

---

## 3. Détail par finding

> Format : id · dimension · sévérité · fichier:ligne · preuve/repro · invariant/anti-patron · renvoi théorique · statut · éligibilité correctif. Les findings F-052..F-057 (agents analytiques) suivent à la fin de la section.

### Majeurs

#### F-026 — `Weights.Validate` inexistant sur le chemin de chargement de `Profile` *(majeur)*
- **Fichier** : `internal/calibration/profile.go:9-19`. **Preuve** : `profile.go:10-11` pose deux invariants (« DSens+DAuthority+DInvariant must sum to 1.0 » ; « each probe map must sum to 1.0 ») ; seul `CorpusEntry.Validate()` existe (`calibrate.go:52`) ; `store.go readProfile/UnmarshalCanonical` désérialise sans validation ; `DefaultProfile()` jamais validé. **Repro** : `Profile{Weights{DSens:2.0,…}}` (ou négatif) + `NewDispatcherWithProfile(...).Decide(...)` → aucune erreur. **Renvoi** : *PRD §11.1 ; chap. III.8.4*. **Statut** : retenu (3/3). **Éligibilité** : **report-only** (schéma de profil → ADR). **Correctif proposé** : `Weights.Validate()` (somme composite ≈ 1,0 à ε ; chaque poids ≥ 0 ; chaque map de sondes ≈ 1,0), appelé à tout point d'entrée chargeant un `Profile` non-Default.

#### F-030 — Variable de décision divergente : calibration sur `SensScore` vs dispatcher sur composite *(majeur)*
- **Fichiers** : `internal/calibration/calibrate.go:184-198` ; `internal/orchestration/dispatcher.go:183-191`. **Preuve** : `calibrate.go:191/194` décide `det/prob` sur `e.SensScore` seul ; `dispatcher.go:185` calcule le composite pondéré, `:189` le compare aux **mêmes** seuils ; `weights.go:17 defaultWeightHook` = identité (pass-through V1). **Repro** : un `CorpusEntry` à `sens=0,5 / auth=inv=0,9 / label probabiliste` → `simulate()` classe « probabiliste » mais `τ_score=0,74 ≥ 0,65` au dispatcher (concorde) ; inverser les pondérations donne un désaccord opposé. **Renvoi** : *III.8.4 ; PRD §10 étapes 6-7, §11.1*. **Statut** : retenu (3/3). **Éligibilité** : **report-only** (sémantique de décision → ADR). **Correctif proposé** : (a) calibrer `simulate()` sur le composite pondéré (lire `Profile.Weights`) ; ou (b) documenter explicitement que la calibration V1 optimise un proxy mono-dimensionnel D-SENS, cohérent seulement sous `sens ≈ auth ≈ inv` (marqueur `Hypothèse`).

#### F-031 — Aucun test ne ferme la boucle calibration → vrai dispatcher *(majeur)*
- **Fichiers** : `test/e2e/calibration_determinism_test.go:98-135` ; `internal/calibration/calibrate_test.go:39-50`. **Preuve** : les gardes existantes ne testent que le déterminisme byte-identique (`:114`) et le hash épinglé (`:131`), jamais la justesse de classification ; `calibrate_test.go` n'exerce que `simulate()` via un helper qui **diverge** de la production (omet la branche `else if SensScore >= Deterministe`), jamais `orchestration.Dispatcher.Decide`. **Renvoi** : *PRD §10, §11.1, §17 #10 ; ADR-0012*. **Statut** : retenu (3/3). **Éligibilité** : **report-only** (extension de portée). **Correctif proposé** : test e2e fermant la boucle (calibrer → reconstruire un `Exchange` aux scores ventilés → `Dispatcher.Decide` → comparer `Regime` aux labels + taux d'accord minimal) ; réaligner le helper sur la production.

#### F-033 — `CheckDrift` non branché sur aucun chemin de décision *(majeur)*
- **Fichiers** : `internal/calibration/drift.go:73` ; `internal/orchestration/dispatcher.go:156-163` ; `internal/app/app.go:23-30`. **Preuve** : `CheckDrift` — 0 appelant hors test ; `dispatcher.go:161` ne teste que `d.profile.DateRevision` ; `DefaultProfile()` sans collecte de fingerprint. Seul `DriftDateExpired` a un effet runtime, obtenu **sans** `CheckDrift`. **Repro** : `grep -rn 'CheckDrift' internal/ cmd/ | grep -v _test.go` → définition + commentaires + docs seulement. **Renvoi** : *PRD §11.4 ; chap. III.8.6.2 (C3)*. **Statut** : retenu (3/3). **Éligibilité** : **report-only** (module sensible → ADR V2). **Correctif proposé** : (a) câbler `CheckDrift` dans `app.NewDispatcher` (collecte fingerprints → `Env` → log/marquage stale) ; ou (b) rétrograder PRD §11.4 et `drift.md §5` au statut « capacité calculée, non câblée en V1 ».

#### F-052 — Règle « `tau/*` → `bridge` interdit » non gardée pour le sous-paquet `dimensions` *(majeur, NEUF)*
- **Fichiers** : `internal/arch_test.go:20-32` (`archRules`) vs `internal/tau/dimensions/dsens.go:7`. **Preuve** : `archRules` interdit `bridge` pour `from: ".../internal/tau"` et `.../tau/invariants`, mais la règle `from: .../tau/dimensions` ne dénie que `tau/invariants` — **pas** `bridge`. Or `build.Default.Import(".../internal/tau", …)` ne charge **que** le paquet `tau` (operator/frontier/diagnostics), jamais le sous-paquet `dimensions`. Conséquence : un import de bridge **concret** (ex. `internal/bridge/agentmeshkafka`) depuis n'importe quel fichier de `dimensions` ne serait capté **ni** par `TestArchitectureLayering`, **ni** par `TestArchNoConcreteLLMInDomain` (qui ne cherche que des SDK tiers). L'import actuel `dimensions → bridge/llm` (`dsens.go:7`) est **autorisé** (ADR-0003 §Clarification, interface `llm.Client` pure) — donc **aucune fuite active**, mais la doctrine d'étanchéité (anti-patron #6 / règle 2) repose sur la revue humaine pour ce sous-paquet, pas sur un test rouge. **Invariant/anti-patron** : étanchéité Clean Arch (règle 2) ; anti-patron #6 par extension. **Renvoi** : *ADR-0001, ADR-0003 ; PRD §8*. **Statut** : retenu (agent étanchéité, recoupé `go list -deps`). **Éligibilité** : **report-only** (modifie la logique d'une garde d'anti-patron → ADR/revue). **Correctif proposé** : ajouter une règle `from: .../tau/dimensions` déniant `.../bridge` **avec** tolérance explicite pour `.../bridge/llm`, et faire walker `TestArchNoConcreteLLMInDomain`/le layering sur les sous-paquets de `tau/` (récursion `filepath.WalkDir`, déjà employée par la garde LLM).

### Mineurs

#### F-001 — 5ᵉ cas de Refus (« observation non modélisée ») jamais émis comme Refus *(mineur)*
- **Fichiers** : `dispatcher.go:216-224` ; `diagnostics.go:6-11` ; `PRD.md:319` ; `docs/theory/07-anti-patrons.md:87`. **Preuve** : l'étape 8 n'ajoute que des lignes à `Trace.UnmodeledObservations` sans positionner `Regime=Refus` ; `diagnostics.go` ne déclare que 4 constantes (pas de `DiagUsageClos`) ; le PRD §7.3 liste pourtant un diagnostic `usage clos potentiel` (`Confirmé par relecture` : `diagnostics.go` = 4 constantes). **Invariant** : anti-patron #4 / Refus cas 5. **Renvoi** : *III.8.7.4*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (sémantique de Refus → ADR). **Correctif proposé** : marquer le 5ᵉ Refus comme différé/aspirationnel au PRD §7.3 et `theory/07:87`, ou l'implémenter derrière un seuil « fort impact » + constante `DiagUsageClos`.

#### F-002 — Diagnostics PRD §7.3 divergents des constantes canoniques (I3, I4) *(mineur)* — **CORRIGÉ**
- **Fichiers** : `PRD.md:316-317` ; `internal/tau/diagnostics.go:9-10`. **Preuve** : `Confirmé par relecture` — `diagnostics.go:9 = "I3 — verrou ontologique D-AUTORITÉ"` (PRD écrivait « I3 — verrou ontologique ») ; `diagnostics.go:10 = "I4 — combinaison incohérente détectée"` (PRD écrivait « I4 — incohérence détectée »). **Renvoi** : *III.8.4.2.bis / III.8.5.4*. **Statut** : retenu (2/2). **Éligibilité** : **mécanique-sûr**. **Correctif** : **appliqué** (commit `16207dd`, alignement verbatim).

#### F-004 — Godoc `EvaluateI5` : débit « 700K vs 8M » non sourcé + « CI window » périmée *(mineur)* — **CORRIGÉ**
- **Fichier** : `internal/tau/invariants/i5_composition.go:86-91`. **Preuve** : chiffre brut sans marqueur (contredit la méthodologie réconciliée) + « 30 s CI window » (CI retirée, ADR-0010). **Invariant** : I5 ; anti-patron #5 ; marqueur d'incertitude. **Renvoi** : *III.8.5.5 ; parent P4-01*. **Statut** : retenu (2/2). **Éligibilité** : **mécanique-sûr** (godoc). **Correctif** : **appliqué** (commit `5965442`) — débit moteur I5 ~1,1 M exec/s (mesuré ce jour), distinction métrique fonction-propriété scalaire I1-I4 ~8,2-9,5 M, « local fuzz window ».

#### F-005 — `EvaluateI2` retourne `Violated` pour résidu vide hors-frontière ; test mal nommé *(mineur)*
- **Fichier** : `internal/tau/invariants/i2_irreductibility.go:69, 77-79`. **Preuve** : `if len(r)==0 { return Violated }` sans garde `Inside()` ; godoc omet la condition de frontière préservée ; test `i2_irreductibility_test.go` nommé `OnInsideFrontier` mais `Inside()==false`. **Renvoi** : *III.8.5.2*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (arbitrage sémantique d'invariant). **Correctif proposé** : résidu-vide ∧ `Inside()==false` → `NotApplicable` ; ne garder `Violated` que pour résidu-vide ∧ `Inside()==true` ; renommer le test.

#### F-006 — `EvaluateI3` ne détecte pas le bypass D-AUTORITÉ sur `Deterministe` *(mineur)*
- **Fichier** : `internal/tau/invariants/i3_authority_asymmetry.go:87-90`. **Preuve** : la détection n'est appliquée que dans `case tau.Probabiliste` ; `case tau.Deterministe` retourne `Held` inconditionnellement. `EvaluateI4` détecte le bypass indépendamment du régime ; `FuzzI3` fixe toujours `Regime: Probabiliste` (l'angle mort n'est donc pas fuzzé). **Repro** : `dec{Regime:Deterministe, Trace{DAuthority:&Score{0.90}, Thresholds{AuthBlock:0.85}}}` sans attestation → `Held` ; même `Probabiliste` → `Violated`. **Renvoi** : *III.8.4.2.bis / III.8.5.3*. **Statut** : retenu (3/3). **Éligibilité** : **report-only** (module gardé par invariant → ADR).

#### F-007 — Incohérence de date dans le godoc de `I3PerimptionLimite()` *(mineur)* — **CORRIGÉ**
- **Fichier** : `internal/tau/invariants/i3_authority_asymmetry.go:12`. **Preuve** : `Confirmé` — godoc « Dated 2026-05-24; next review 2027-01-01 » vs `EvaluateI3:101` « dated 2026-05-16 », `docs/theory/05:93` « Revérification au 2026-12-01 », `CLAUDE.md` (anti-patron #3) « daté 2026-05-16 ». 2027-01-01 est le plafond dur de péremption de profil (valeur retournée), distinct de la revue I3 (asymétrie assumée, `theory/05:225`). **Renvoi** : *III.8.5.3 / III.8.6.2 (C2)*. **Statut** : retenu (2/2 ; lié I2-05). **Éligibilité** : **mécanique-sûr** (godoc, valeur retournée inchangée). **Correctif** : **appliqué** (commit `1372a33`).

#### F-012 — Le dispatcher laisse fuiter l'erreur brute des scoreurs (`DispatchError` jamais utilisé) *(mineur)*
- **Fichier** : `dispatcher.go:148-149, 167-169, 171-173`. **Preuve** : `return tau.Decision{}, err` brut aux étapes 2 et 4 ; aucun import de `internal/errors` dans `dispatcher.go` ; `errors.go:23-42` définit `DispatchError{Stage 1..8}`. **Repro** : `errors.As(err, new(*errors.DispatchError)) == false`. **Renvoi** : *ADR-0009 ; Q5-02*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (à coordonner avec l'API `Decide` + test).

#### F-013 — `NewDispatcher` ne valide que l'ordre ; `AuthBlock=0` → refus ontologique systématique *(mineur)*
- **Fichier** : `dispatcher.go:44-49, 151`. **Preuve** : `thresholds.go:29 Ordered()` ne vérifie que `Deterministe <= Probabiliste` ; `AuthBlock=0` passe ; `authScore.Value >= 0` toujours vrai → tout `Exchange` in-frontière sans attestation refusé à tort. **Repro** : `NewDispatcher(llm.Stub{}, Thresholds{Deterministe:0.35, Probabiliste:0.65})` (AuthBlock=0) → Refus `DiagVerrouOntologique`. **Renvoi** : *III.8.6.2 (C2 : θ_auth_block ≤ 0,85)*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (module sensible → ADR).

#### F-014 — Seuil `Deterministe` mort : non lu par le dispatcher ni discriminant dans `simulate()` *(mineur)*
- **Fichiers** : `dispatcher.go:187-191` ; `calibrate.go:194-197`. **Preuve** : `dispatcher.go:188-191` ne compare que `Probabiliste` ; `calibrate.go:194-197` : branche `>= Deterministe` et branche défaut renvoient toutes deux « probabiliste » (no-op). La grid-search itère `Deterministe` sans pouvoir discriminant. **Renvoi** : *PRD §10 étape 7 ; ADR-0007*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (sémantique de décision, ADR-0007).

#### F-015 — Fragment de hash golden incorrect (« …40c1 ») dans CLAUDE.md et CHANGELOG.md *(mineur)* — **CORRIGÉ**
- **Fichiers** : `CLAUDE.md:295` ; `CHANGELOG.md:13`. **Preuve** : `Confirmé` — hash réel `test/e2e/calibration_determinism_test.go:33` = `8e5dc2fc…caa4` ; PRD.md et ADR-0012 déjà corrects. **Invariant** : anti-patron #5 (chiffre erroné). **Statut** : retenu (2/2). **Éligibilité** : **mécanique-sûr**. **Correctif** : **appliqué** (commit `27c36ed`).

#### F-016 — CHANGELOG « 303 nœuds » vs graphe réel et README « 325 » *(mineur)* — **CORRIGÉ**
- **Fichier** : `CHANGELOG.md:19`. **Preuve** : `Confirmé` — README.md:5 « 325 nœuds, 1 339 arêtes » ; HEAD `b94e93f` a réaligné le README mais pas le CHANGELOG. **Invariant** : anti-patron #5. **Statut** : retenu (2/2). **Éligibilité** : **mécanique-sûr**. **Correctif** : **appliqué** (commit `e456a67`).

#### F-017 — ADR-0012 absent de l'index canonique (README §Références et CLAUDE.md) *(mineur)* — **CORRIGÉ**
- **Fichiers** : `README.md:362-371` ; `CLAUDE.md:313`. **Preuve** : `Confirmé` — index README s'arrêtait à 0010 ; CLAUDE.md énumérait « ADRs 0001-0010 » ; `docs/adr/0012-…` existe et est référencé ailleurs ; 0011 réservé (HGL/Lean), déjà documenté. **Statut** : retenu (2/2). **Éligibilité** : **mécanique-sûr**. **Correctif** : **appliqué** (commit `d57fb7c`).

#### F-020 — `FileAdapter.Close()` via `sync.Once` neutralise un `Stream()` ouvert après le 1ᵉʳ Close *(mineur)*
- **Fichier** : `internal/bridge/agentmeshkafka/file_adapter.go:17-22, 36-52, 100-112`. **Preuve** : `once.Do(func(){ … f.stop() })` + `f.stop = cancel` réécrit à chaque `Stream()` ; `once` épuisé → flux ultérieur non annulable ; deux `Stream()` concurrents écrasent `f.stop`. **Repro déterministe (sans -race)** : `a.Close()` ; `a.Stream(ctx,nil)` ; `a.Close()` (2ᵉ no-op) → ctx du Stream jamais annulé. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (concurrence ; §4).

#### F-021 — `AtomicThresholds.millis()` tronque au lieu d'arrondir (biais vers le bas) *(informatif)*
- **Fichier** : `internal/calibration/thresholds_atomic.go:11-12, 103`. **Preuve** : `int64(v*1000)` (troncature) vs godoc « Resolution: 0.001 » ; `0.4499999→449→0.449`. Impact actuel nul (code mort, R3-05). **Statut** : retenu (1/1). **Éligibilité** : **report-only** (envisager le retrait du code mort).

#### F-022 — `Decide` n'inspecte ni `ctx.Err()` ni `ctx.Done()` *(mineur)*
- **Fichier** : `dispatcher.go:127-163`. **Preuve** : aucune référence `ctx.Err()/Done()` avant l'étape 4 ; `dauthority.go:34` ignore `ctx`. Connu (R3-03 ; garde abandonnée pour gocyclo 16>15). **Repro** : `cancel()` puis `Decide(ctx, horsFrontiere)` → `err==nil`, `Refus(frontière)` malgré annulation. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (module sensible ; §4).

#### F-027 — `dimensionWeights()` ne teste que somme>0 *(mineur)*
- **Fichier** : `dispatcher.go:71-80`. **Preuve** : `if d.profile != nil && (DSens+DAuthority+DInvariant) > 0` puis poids verbatim ; aucune vérif somme=1 ni poids≥0 ; `{2.0,0,0}` accepté. **Renvoi** : *PRD §10 étape 6, §11.1*. **Statut** : retenu (3/3). **Éligibilité** : **report-only** (dépend de F-026).

#### F-028 — Poids malformés → `τ_score` hors `[0,1]` faussant l'étape 7 *(mineur)*
- **Fichier** : `dispatcher.go:183-191`. **Preuve** : `tauScore = w.DSens*sens + …` sans clamp ; comparaison unique `>= Probabiliste` ; `TauScore` exporté brut. **Repro** : `fakeLLM{0.50}`, Weights `{0.8,0.6,0.6}` → `τ_score≈1,0 ≥ 0,65` → `Probabiliste` ; avec `DefaultProfile` → `Deterministe`. **Renvoi** : *PRD §10 étapes 6-7 ; III.8.4*. **Statut** : retenu (3/3). **Éligibilité** : **report-only**.

#### F-029 — `profile_test.go` ne couvre que `DefaultProfile()` *(mineur)*
- **Fichier** : `profile_test.go:41-68`. **Preuve** : les seuls tests non-Default emploient `(0.60,0.20,0.20)` (somme 1,0) ; aucun cas malformé. **Renvoi** : *PRD §11.1*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (dépend de F-026).

#### F-034 — `drift.md §5` survend un `slog.Warn` de santé inexistant *(mineur)*
- **Fichier** : `docs/algorithms/drift.md:119-135`. **Preuve** : décrit `CheckDrift(...)` « étape 3 » + « émis via slog.Warn » ; aucun `slog.Warn` de drift dans le code ; `dispatcher.go:161` compare la date en ligne sans `CheckDrift`. **Invariant** : anti-patron #5 (survente capacité non implémentée). **Renvoi** : *PRD §11.4*. **Statut** : retenu (2/2 ; connexe F-033). **Éligibilité** : **report-only** (dépend de l'arbitrage F-033).

#### F-035 — Corpus vide produit un profil dégénéré silencieux (exit 0) *(mineur)*
- **Fichiers** : `calibrate.go:111-145, 76-96` ; `cmd/tau/calibrate.go:70-96`. **Preuve** : `LoadCorpus` renvoie `([]CorpusEntry{}, nil)` ; `Calibrate` `best=-1` puis `countAgreement([],t)=0 > -1` → bascule sur le 1ᵉʳ point de grille dégénéré, écrit le profil, exit 0, aucun diagnostic. Runtime `Decide` non affecté (`DefaultProfile`). Distinct de C1-01 (résolu). **Repro** : `printf '' > empty.jsonl ; tau calibrate --corpus empty.jsonl …` → exit 0, profil dégénéré. **Renvoi** : *PRD §11.1, §20.4 ; III.8.6.2 (C3)*. **Statut** : retenu (3/3). **Éligibilité** : **report-only** (CLI/schéma → décision).

#### F-036 — Exit 3 (decide error) inatteignable en config CLI par défaut *(mineur)*
- **Fichier** : `cmd/tau/main.go:62-75`. **Preuve** : les 5 Refus retournent `(decision, nil)` ; le seul chemin d'erreur est `ScoreDSens → llm.Interpret`, or `stub.go:27` retourne toujours `nil`. Exit 3 mort sauf `TAUGO_LLM_BACKEND=real` défaillant. **Renvoi** : *PRD §7.3 ; étend C1-02*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (docstring/test).

#### F-038 — Aucun test ne couvre le chemin corpus-vide de `tau calibrate` *(mineur)*
- **Fichier** : `cmd/tau/calibrate_test.go`. **Preuve** : la suite couvre happy path, flags, dates, corpus introuvable, JSON invalide, régime invalide, legacy ; manquent corpus VIDE (F-035), tout-refus/mono-entrée, `--seed` malformé. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (dépend de F-035).

#### F-039 — Round-trip canonique valide l'idempotence, pas la fidélité *(mineur)*
- **Fichiers** : `calibrate_test.go:115-151` ; `store_test.go:133-149`. **Preuve** : `bytes.Equal(b1,b2)` (idempotence) ; `assertProfileEqual` exclut `CreatedAt`, omet `DateRevision`/`Weights`/fingerprints. Valeurs round-trippent en pratique — lacune de couverture, pas un bug de données. **Renvoi** : *PRD §17, §11.4*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (module sensible `profile.go`).

#### F-042 — `Regime.UnmarshalJSON` : branche fallback int sans garde de plage *(mineur)*
- **Fichier** : `internal/tau/operator.go:68-74`. **Preuve** : branche int `*r = Regime(n)` ne valide rien ; `Unmarshal("99")`/`("-1")` → `err=nil`, valeur invalide ; `String()→"Unknown"` ; re-Marshal → `"Unknown"` non ré-unmarshalable (round-trip rompu). `CorpusEntry` n'utilise pas l'enum (corpus non exposé). **Invariant** : anti-patron #4 ; §JSON enums. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (`operator.go` API publique sensible → ADR + revue golden).

#### F-043 — `DiscoveryMode.UnmarshalJSON` : même absence de garde de plage *(mineur)*
- **Fichier** : `internal/tau/operator.go:156-162`. **Preuve** : symétrique à F-042 ; exposition réelle via decode direct d'un `tau.Exchange` (`cmd/tau/main.go:66`). **Invariant** : anti-patron #4 ; §JSON enums. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (même raison que F-042).

#### F-045 — `FrontierCheck()` couple `UniversOuvert` et `CompositionVariable` *(mineur)*
- **Fichier** : `internal/tau/operator.go:240-243`. **Preuve** : `dynamic := x.Target.DiscoveryMode != Static` pilote à la fois `UniversOuvert` et `CompositionVariable` → aucune valeur d'`Exchange` ne produit `UniversOuvert != CompositionVariable` ; `i2_irreductibility.go:52-54` remet les deux à false ensemble. Heuristique M2 placeholder (jusqu'à M5). **Repro** : impossible de construire `x` tel que les deux diffèrent. **Renvoi** : *III.8.3.2 ; docs/theory/03:40-45 ; PRD §4.3*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (API publique gardée → ADR).

#### F-046 — Suite `e2e_trace_test.go` orpheline : le `-run` documenté l'exclut *(mineur)*
- **Fichier** : `test/e2e/e2e_trace_test.go:1,25`. **Preuve** : tag `e2e` ; les commandes documentées ne la lancent pas (`-tags=integration` ne compile pas le fichier ; `-run="TestCalibration|TestCalibrate|TestExpiredProfileRefuses"` ne matche pas `TestE2E_TauDecide_TraceVentileeEtPoidsProfil`). `Confirmé` que le test **passe** quand on lance `-tags=e2e` sans filtre `-run` (gate exécuté ce jour). **Renvoi** : *ADR-0008 (garde T-022)*. **Statut** : retenu (3/3). **Éligibilité** : **report-only** (étendre le `-run` documenté).

#### F-047 — `test/e2e/doc.go` : godoc prétend que tous les tests e2e sont gardés par `integration` *(mineur)*
- **Fichier** : `test/e2e/doc.go:2,5`. **Preuve** : « guarded by the integration build tag » ; or 2/3 fichiers portent `//go:build e2e`. **Statut** : retenu (2/2). **Éligibilité** : **report-only**.

#### F-048 — Référence godoc périmée à un helper `buildCLI` inexistant *(mineur)*
- **Fichier** : `test/e2e/calibration_determinism_test.go:50`. **Preuve** : commentaire « …the integration-tag buildCLI helper in agentmeshkafka_test.go » ; aucun `buildCLI` n'existe (seule fonction = `buildTauE2E`). **Statut** : retenu (2/2). **Éligibilité** : **report-only** (réécriture non univoque).

#### F-050 — Chemin best-effort de perte d'erreurs (`errc` plein → drop) sans garde comportementale *(mineur)*
- **Fichier** : `internal/app/agentmesh.go:121-124 (et 103-106)`. **Preuve** : `select { case errOut <- e: default: }` (drop) ; `chan error, 8` ; tests existants n'assertent que `errs >= 1`, jamais la saturation. Connu R3-01 (contrat documenté, non testé). **Statut** : retenu (2/2). **Éligibilité** : **report-only** (concurrence ; §4).

#### F-053 — Imports inverses `bridge → {tau, orchestration, app}` dans un test externe non gardé *(mineur)*
- **Fichier** : `internal/bridge/agentmeshkafka/empirical_i4_test.go:18-22` (tag `//go:build empirical`, paquet externe `agentmeshkafka_test`). **Preuve** : importe `app`, `orchestration`, `tau`, `tau/dimensions` — inverse de la règle 3. Non capté : `TestBridgeNoTauImport` exclut les `*_test.go` ; `TestArchitectureLayering` agrège `Imports`+`TestImports` mais jamais `XTestImports` (paquet externe). Légitime pour une campagne I4 (câble tout le stack), mais **totalement non gardé**. **Statut** : retenu (agent étanchéité). **Éligibilité** : **report-only** (documenter l'exception assumée, ou couvrir `XTestImports`).

#### F-055 — `TestNoPredictiveAPI` : court-circuit sur les blocs `GenDecl` multi-specs *(mineur)*
- **Fichier** : `internal/anti_patterns_test.go:74-88`. **Preuve** : `exportedDeclName` retourne le **premier** nom exporté d'un bloc `var(...)`/`const(...)`/`type(...)` puis sort ; un identifiant prédictif placé **après** un premier spec non prédictif dans le même bloc groupé échapperait à la détection. Impact faible : le vecteur principal (méthode/fonction `Predict*`/`Forecast*`) est toujours capté (chaque `FuncDecl` est un decl séparé). **Invariant** : anti-patron #1 (qualité de garde). **Statut** : retenu (agent gardes). **Éligibilité** : **report-only** (touche un test-garde d'anti-patron). **Correctif proposé** : collecter **tous** les noms exportés du bloc au lieu du premier.

#### F-056 — `TestStep8_..._TraceEnriched` : nom trompeur (assert l'absence de violation) *(mineur)*
- **Fichier** : `internal/orchestration/dispatcher_invariants_test.go:55-89`. **Preuve** : malgré son nom, le test échoue si une violation est détectée (`len(UnmodeledObservations)!=0`), car le scénario de bypass n'arrive pas dans le flux normal (l'étape 2 l'intercepte) — le commentaire l'admet (l.49-54). La détection réelle est gardée par `TestEvaluateI4_DetecteByPassSilencieux` et `TestEvaluateI3_LitDAuthorityVentile` (`dispatcher_scores_test.go`). L'anti-patron #4 reste gardé, mais le test au nom le plus explicite ne prouve pas ce qu'il annonce. **Statut** : retenu (agent gardes). **Éligibilité** : **report-only** (renommage/clarification d'un test en module sensible).

### Informatifs

#### F-003 — Godoc `FrontierCheck()` : 3 règles pour 4 conditions ; `!HumanInLoop ⇒ PairProbabiliste` *(informatif)*
- **Fichier** : `internal/tau/operator.go:238-245`. **Preuve** : `PairProbabiliste: !x.Initiator.HumanInLoop` ; le code/théorie parlent du *pair* (`frontier.go:9`), l'heuristique M2 le dérive de l'*initiateur*. Distinct de F-045 (godoc vs couplage). **Renvoi** : *III.8.3.2*. **Statut** : retenu (1/1). **Éligibilité** : **report-only** (marqueur d'incertitude à ajouter).

#### F-008 — Fallback proxy `EvaluateI3` compare le composite `TauScore` au seuil d'autorité *(informatif)*
- **Fichier** : `i3_authority_asymmetry.go:76-79`. **Preuve** : `authValue := dec.Trace.TauScore // fallback proxy` puis `>= AuthBlock` ; depuis ADR-0008 le dispatcher peuple toujours `DAuthority` → impact limité aux traces forgées/anciennes. **Renvoi** : *III.8.4.2.bis*. **Statut** : retenu (1/1). **Éligibilité** : **report-only**.

#### F-009 — Probe `S_reasoner_intent` : sortie LLM non bornée/validée avant agrégation *(informatif→mineur selon backend)*
- **Fichier** : `internal/tau/dimensions/dsens.go:44-64, 102-107`. **Preuve** : `return c.Interpret(...)` sans clamp ; `Value: clamp01(value)` final seulement ; `Probes["S_reasoner_intent"]` stocke la valeur brute. Stub conforme → non déclenchable sans backend réel non conforme ; un `Interpret` renvoyant `NaN` propagerait `Score.Value==NaN` et forcerait silencieusement `Deterministe`. **Renvoi** : *III.8.4.1*. **Statut** : retenu (2/2). **Éligibilité** : **report-only** (robustesse domaine).

#### F-010 — `clamp01` par-dimension inatteignable pour les sondes non-LLM *(informatif)*
- **Fichier** : `dimensions/score.go:16-24`. Filet défensif documenté, aucune action requise. **Statut** : retenu (1/1).

#### F-011 — Scorers acceptent des poids arbitraires ; poids calibrés par-sonde morts au runtime *(informatif)*
- **Fichier** : `dimensions/dsens.go:35-47`. **Preuve** : somme `Σ wᵢ·probeᵢ` sans garde ; le dispatcher passe `Default*Weights()`, jamais `Profile.Weights.*Probes`. **Statut** : retenu (1/1). **Éligibilité** : **report-only**.

#### F-018 — PRD §8.4 référence un chemin golden inexistant `internal/testdata/golden/` *(informatif)*
- **Fichier** : `PRD.md:779` (tag *(V1.1)*, livrable futur) ; golden réel = `tests/calibration/golden-corpus.jsonl`. **Statut** : retenu (1/1 ; A6-02). **Éligibilité** : **report-only**.

#### F-019 — Compte « 24 linters » vs 25 entrées `enable:` *(informatif)*
- **Fichier** : `CLAUDE.md:14` (+ README, PRD, ADR-0002). **Preuve** : `.golangci.yml` = 25 entrées sous `enable:` (`typecheck` incluse) ; « 24 » défendable si l'on exclut `typecheck`, mais non littéralement reproductible. **Statut** : retenu (1/1). **Éligibilité** : **report-only** (clarifier la convention de comptage).

#### F-032 — Gardes `refus_i4`/`refus_authority` : `simulate()` ≡ dispatcher (pas de divergence) *(informatif)*
- **Fichiers** : `calibrate.go:185,188` ; `dispatcher.go:151,178`. Conjonctions commutatives identiques des deux côtés → confirmation que seule `det/prob` diverge (cf. F-030). **Renvoi** : *III.8.5.3/5.4*. **Statut** : retenu (1/1, aucune action).

#### F-037 — Exit 4 (encode error) pratiquement inatteignable et non testé *(informatif)*
- **Fichier** : `cmd/tau/main.go:76-79`. `Decision` entièrement sérialisable ; `clamp01` borne `[0,1]` ; seul échec réaliste = `io.Writer`. **Statut** : retenu (1/1).

#### F-040 — `UnmarshalCanonical` non inverse exact pour `time.Time` ; godoc « inverse » trop fort *(informatif)*
- **Fichiers** : `calibrate.go:232-240` ; `profile.go:31-32`. `FixedZone` se relit en `Local` (`.Equal()` vrai, texte RFC3339 byte-identique) ; horloge monotone perdue. Impact nul. **Statut** : retenu (1/1).

#### F-041 — `MarshalCanonical` échoue pour une date hors plage RFC3339 [0..9999] *(informatif)*
- **Fichiers** : `calibrate.go:206-211`. Contrainte stdlib propagée proprement (pas de panic). Probabilité réelle nulle. **Statut** : retenu (1/1, aucune action).

#### F-044 — Tests `UnmarshalJSON` des enums ne couvrent que des int in-range *(informatif)*
- **Fichier** : `internal/tau/operator_test.go:70-79, 153-162`. Branche int testée seulement avec `1`/`0` ; la couverture 100 % masque le trou (F-042/F-043). **Statut** : retenu (1/1). **Éligibilité** : **report-only** (lié au fix code, ne pas ajouter le test seul).

#### F-049 — Aucun golden lock de la bijection `AgentMeshExchange↔tau.Exchange` *(informatif)*
- **Fichier** : `internal/app/agentmesh.go:17`. Fidélité gardée par assertions champ-par-champ ; `Context` non asserté, pas de round-trip inverse (`BijectionCoreFields` unidirectionnel). **Renvoi** : *ADR-0005*. **Statut** : retenu (1/1). **Éligibilité** : **report-only**.

#### F-051 — `e2e_trace_test.go` n'assert pas l'immutabilité de la Trace après `Decide` *(informatif)*
- **Fichier** : `test/e2e/e2e_trace_test.go:58-60, 71`. Assertions non-nil/positif + divergence `TauScore` ; aucune assertion d'immutabilité. Test orphelin (F-046). **Renvoi** : *ADR-0008*. **Statut** : retenu (1/1).

#### F-054 — `TestArchNoConcreteLLMInDomain` ne couvre pas tous les périmètres interdits *(informatif)*
- **Fichier** : `internal/arch_test.go:181-184`. **Preuve** : `domainDirs = {tau, orchestration}` ; la doctrine (anti-patron #6) vise aussi `bridge/agentmeshkafka`, `cmd/*`, `calibration`, `testutil`, `errors`. Risque actuel **nul** (`go.mod` sans dépendance externe), mais garde plus étroite que la doctrine. **Statut** : retenu (agent étanchéité). **Éligibilité** : **report-only**.

#### F-057 — Couverture globale re-mesurée 88,2 % vs 89,2 % documenté *(informatif)*
- **Fichiers** : `PRD.md:849` ; `README.md:302` ; `CHANGELOG.md:19`. **Preuve** : `Confirmé (mesuré 2026-05-29)` — `go test -coverpkg=./...` = **88,2 %** au HEAD `b94e93f`, vs « 89,2 % » documenté (mesuré au HEAD `1948a7b`). Écart de ~1 pt dû aux ajouts de code post-audit (gestion exit-code `calibrate.go`, `generator.go --scored`). Affirmation honnêtement marquée avec méthode dans les docs ; gate global ≥ 80 % **tenu**. **Statut** : retenu (mesure directe). **Éligibilité** : **report-only** (rafraîchir le chiffre relève d'un arbitrage : la mesure dérive à chaque ajout ; recommandation : dater le chiffre « au commit X » ou tolérer la fourchette 88-89 %).

---

## 4. À valider en `-race` sur Linux/macOS

`-race` non exécuté (`CGO_ENABLED=0`, pas de gcc/clang sur l'hôte). Findings de concurrence non validables localement, avec la commande :

| Finding | Sév. | Commande |
|---|---|---|
| **F-020** (`FileAdapter.Close()`/`sync.Once` vs `Stream()`) | mineur | `CGO_ENABLED=1 go test -race -run 'TestFileAdapter' -count=1 ./internal/bridge/agentmeshkafka/...` |
| **F-022** (`Decide` n'inspecte pas `ctx.Err()`/`Done()`) | mineur | `CGO_ENABLED=1 go test -race -run 'TestDispatcher|TestDecide' -count=1 ./internal/orchestration/...` |
| **F-050** (`errc` plein → drop, contrat best-effort non testé) | mineur | `CGO_ENABLED=1 go test -race -run 'TestStreamAsTauExchanges' -count=1 ./internal/app/...` |

> Note transverse (A6-05) : le `-race` n'est exécutable que sur Linux/macOS (CGO requis). Sur Windows local, repli `go test -short ./...`. F-020 dispose en outre d'un repro **déterministe sans -race** (séquence `Close → Stream → Close`). `Le cœur Decide est concurremment sain par construction` (immuabilité, zéro goroutine) `[probable, statique]`.

---

## 5. Correctifs commités

**Branche** : `audit/fixes-2026-05-29` (depuis `main` à `b94e93f`). Politique : un commit conventionnel par fix, diffs chirurgicaux, **uniquement** des correctifs mécaniques-sûrs (documentaire/godoc, hors logique d'invariant/frontière/schéma/API publique), chacun gate-vérifié sur sortie capturée.

| Commit | Finding | Type | Portée |
|---|---|---|---|
| `16207dd` | F-002 | docs(prd) | PRD §7.3 : diagnostics I3/I4 alignés verbatim sur `diagnostics.go` |
| `27c36ed` | F-015 | docs | hash golden `…40c1` → `…caa4` (CLAUDE.md, CHANGELOG.md) |
| `e456a67` | F-016 | docs(changelog) | « 303 nœuds » → « 325 nœuds, 1 339 arêtes » |
| `d57fb7c` | F-017 | docs | indexation ADR-0012 (README §Références, CLAUDE.md) |
| `5965442` | F-004 | docs(invariants) | godoc `EvaluateI5` : débit sourcé + retrait « CI window » |
| `1372a33` | F-007 | docs(invariants) | godoc `I3PerimptionLimite` : dates réconciliées |

**Gates post-correctifs** (capturés, relus) : `go build` exit 0 ; `go vet ./...` exit 0 ; `go test -short ./...` 14 paquets verts ; blobs LF des deux `.go` modifiés `gofmt`-propres ; aucun linter de longueur de ligne (`lll`) actif. Les deux modifications `.go` sont des commentaires seuls → effet runtime nul (benchmark non requis). `git status` propre après chaque commit.

**Tous les autres findings sont report-only** : théorie/invariant/dimension/frontière (F-001, F-005, F-006, F-008, F-045, F-003), sémantique de décision/Refus (F-013, F-014, F-030, F-031, F-035), garde d'anti-patron/architecture (F-052, F-053, F-055, F-056, F-054), schéma de profil/golden (F-026, F-039, F-040), API publique `operator.go` (F-042, F-043), concurrence (F-020, F-022, F-050), couverture/tests (F-029, F-038, F-044, F-046, F-047, F-048, F-049, F-051), chiffre dérivant (F-057). Ils exigent soit le workflow strict ADR → MAJ PRD → MAJ `docs/theory`, soit une décision de conception.

---

## 6. Faux positifs écartés (vérification adversariale)

### 6.1 Bruit gofmt / CRLF — **écarté avec preuve dure**

`golangci-lint run ./...` retourne exit 1 avec 8 alertes `(gofmt)` (`arch_test.go`, `i1_conservation.go`, `i2_irreductibility.go`, `i4_coherence_test.go`, `i5_composition_test.go`, `calibrate.go`, `errors.go`, `errors_test.go`). **Toutes sont des faux positifs CRLF** `Confirmé (mesuré 2026-05-29)` : pour chaque fichier, le blob LF committé (`git show :<f>`) est `gofmt`-propre (`gofmt -l` vide), alors que l'arbre de travail OneDrive est en CRLF (ex. `internal/errors/errors.go` : blob 0 retour chariot, arbre 95). Le code committé est intégralement formaté. Aucun de ces signaux n'est un défaut.

### 6.2 Findings réfutés ou confirmés déjà-résolus

| Finding | Sév. initiale | Verdict | Raison |
|---|---|---|---|
| **F-023** — `SetTuning` : 6 `Store` séquentiels (RMW non transactionnel) | informatif | **Déjà-résolu** | Défaut structurel réel mais **code mort** (`AtomicThresholds` jamais câblé — `grep` confiné au fichier+test) ; volet épistémique (docstring survendue) **déjà corrigé** (R3-02, `e320e70` : « is NOT a single atomic transaction… not yet wired into the dispatcher »). Blob LF `gofmt`-propre. |
| **F-024** — `StreamAsTauExchanges` : perte best-effort des erreurs | informatif | **Déjà-résolu (doc)** | Comportement par conception, **contrat documenté honnêtement** par R3-01 (`e320e70`, godoc `agentmesh.go` « best-effort (lossy)… the exchange stream is never sacrificed »). Lacune de **test** couverte distinctement par F-050. |
| **F-025** — Anti-patron #7 « satisfait » | confirmatoire | **Réfuté (aucun défaut)** | `internal/tau/*` : aucun global mutable ; seuls `regimeStrings`/`discoveryModeStrings` (lookup maps `//nolint:gochecknoglobals` read-only), `I3PerimptionLimite` est un getter ; `func init()` → 0 occurrence dans `internal/` ; `gochecknoglobals` actif. |

### 6.3 Items de l'audit 2026-05-29 confirmés déjà-résolus (vérifiés au code, non re-listés comme findings)

C1-01 (ADR-0012), C1-04, I2-03, I2-04, Q5-01, A6-01, A6-03, A6-04, P4-01, P4-02 (partiel — benchmarks `Decide`/dimensions présents ; bench calibration de bout en bout toujours absent), P4-04, R3-01, R3-02. **Vérification VOLET B (agent gardes/non-régression)** : 6/7 items prouvés landés et corrects au code ; le test Q5-01 inclut des **cas négatifs** (un diagnostic ne matche pas un sentinel d'un autre type), excluant un faux-positif « tout matche ». A6-04 conforme à la réalité architecturale (pas de règle `from: app`, car `app` est la racine de composition). **Aucune régression détectée.**

---

## Annexe — Mesures de gates (2026-05-29, HEAD `b94e93f`)

| Gate | Résultat | Marqueur |
|---|---|---|
| `go build -trimpath -buildvcs=true ./cmd/tau` | exit 0 | `Confirmé` |
| `go vet ./...` | exit 0, 0 alerte | `Confirmé` |
| `go test -short ./...` | 14 paquets verts, 0 FAIL | `Confirmé` |
| `go test -tags=e2e ./test/e2e/...` | 4/4 PASS (dont hash golden + déterminisme) | `Confirmé` |
| `go test -tags=integration ./test/e2e/...` | vert | `Confirmé` |
| Couverture `-coverpkg=./...` (global) | **88,2 %** | `Confirmé (mesuré)` |
| Couverture `tau` / `dimensions` / `invariants` | 100 % / 98,7 % / 92,7 % (gate ≥ 90 % tenu) | `Confirmé (mesuré)` |
| Fuzz I1-I5 (`-fuzztime=15s` chacun) | 0 crash ; débit moteur ~1,0-1,6 M exec/s (I5 ~1,0-1,1 M/s) | `Confirmé (mesuré)` |
| `BenchmarkDecide` Det/Prob/Refus | ~737 / ~726 / ~25,7 ns/op (16/16/0 allocs) | `Confirmé (mesuré)` |
| `golangci-lint run ./...` | exit 1 — **100 % faux positifs CRLF gofmt** (blobs LF propres) | `Confirmé` |
| `-race ./...` | **non exécuté** (CGO absent) | `À vérifier` (gate manuel Linux/macOS, §4) |

---

*Rapport produit par workflow multi-agents (cadrage → finders → boucle de complétude → vérification adversariale → synthèse) + 2 agents analytiques de comblement + exécution directe des gates. Thread principal : coordination, intégration, application des correctifs mécaniques, validation (règle Agent teams, `CLAUDE.md` §11). Lecture seule sur le code source ; seuls les 6 correctifs documentaires/godoc de la branche `audit/fixes-2026-05-29` modifient le dépôt. Aucune fabrication.*
