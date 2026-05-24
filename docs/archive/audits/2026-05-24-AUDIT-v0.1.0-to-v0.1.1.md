# AUDIT.md — TauGo v0.1.0

**Date** : 2026-05-24 · **Commit base** : `5a68c12` · **Branche** : `main`

Audit consolidé issu de 3 sous-agents en parallèle (Phase A, Agent teams CLAUDE.md §11) :
A.1 — inventaire purge agressive (Explore) ·
A.2 — audit code complet (general-purpose) ·
A.3 — audit architectural + alignement théorie↔code (ruflo-swarm:architect).

---

## 1. Synthèse exécutive

| Indicateur | Valeur |
|---|---|
| Conformité PRD §17 | 8.5 / 10 |
| Anti-patrons §7.2 effectivement violés | **0** |
| **Anti-patrons §7.2 non gardés (test manquant)** | **1** (anti-patron #6) |
| Observations P0 | **2** (re-priorisées par l'intégrateur) |
| Observations P1 | 9 |
| Observations P2 | 12 |
| Observations P3 | 10 |
| Couverture globale | 90.9 % (CHANGELOG `5a68c12`) |
| Couverture `internal/tau/*` | 100 % / 98.7 % / 97.1 % |
| Adoption `t.Parallel()` | 219/207 tests (100 %+ avec sous-tests) |
| Fichiers candidats purge | 28 (1.6 MB tracked + ~11 MB orphelins) |
| LOC à purger | ~10 K (dont 9 824 dans plans M0-M6 archivables) |
| Verdict refactor | **Recommandé** — architectural inclus, sans rupture API ≥ ADR |

**Doctrine** : TauGo livre la spec V0.1.0 sans violation active des invariants I1-I5 ni des anti-objectifs PRD §3.3. La dette est principalement structurelle (3 `Thresholds` parallèles, 4 packages vides, garde anti-patron #6 manquante) et instrumentale (Trace sans scores ventilés). Aucune refonte architecturale majeure n'est requise ; un refactor agressif ciblé suffit.

---

## 2. Anti-patrons PRD §7.2 — état des gardes

| # | Anti-patron | Garde | Fichier | Conformité |
|---|---|---|---|---|
| 1 | `Predict*`/`Expected*`/`Forecast*` exporté | `TestNoPredictiveAPI` | `internal/anti_patterns_test.go:34` | OK (scan limité à `tau/*` et `orchestration` ; `CorpusEntry.ExpectedRegime` dans `calibration/` non couvert mais champ d'étiquette, non d'API prédictive) |
| 2 | Bypass `FrontierCheck.Inside()` | `TestFrontierCheck_Inside_*` | `internal/tau/frontier_test.go` | OK — nom réel diverge de PRD/CLAUDE.md qui réfèrent encore `TestRefusHorsFrontiere` (cf. P3-12) |
| 3 | Profil périmé toléré | `TestI3_DateRevisionRespectee`, `TestExpiredProfileRefuses` | `i3_authority_asymmetry.go:51-58`, `dispatcher.go:117-119` | OK conditionnel — voir **P0-02** (NewDispatcher sans profil désactive silencieusement la garde) |
| 4 | Observation non modélisée passée sous silence | `TestUnmodeledObservationsReported` | `dispatcher.go:160-166` | OK |
| 5 | Fabrication dans `docs/` | Audit textuel + revue | `docs/` | OK |
| 6 | Import LLM concret dans `tau/*` ou `orchestration/*` | **`TestArchNoConcreteLLMInDomain`** | **MANQUANT** | **NON GARDÉ — P0-01** |
| 7 | Globaux mutables non synchronisés dans `tau/*` | `gochecknoglobals` + revue | `.golangci.yml` | OK (5 `//nolint:gochecknoglobals` chirurgicaux documentés) |

---

## 3. Étanchéité Clean Architecture — matrice + violations

### 3.1 Matrice de dépendances (production, sans `_test.go`)

| Package | Imports taugo |
|---|---|
| `cmd/tau` | `app`, `calibration`, `orchestration`, `tau` |
| `cmd/generate-corpus` | `calibration` |
| `internal/app` | `bridge/llm`, `orchestration`, `bridge/agentmeshkafka`, `tau` |
| `internal/orchestration` | `bridge/llm`, `calibration`, `tau`, `tau/dimensions`, `tau/invariants` |
| `internal/tau` | aucun |
| `internal/tau/dimensions/dsens` | `bridge/llm` (interface, ADR-0003), `tau` |
| `internal/tau/dimensions/{dauthority,dinvariant}` | `tau` |
| `internal/tau/invariants/i{1..5}`, `evaluator` | `tau` |
| `internal/calibration` | aucun |
| `internal/bridge/llm` | aucun |
| `internal/bridge/agentmeshkafka` | aucun |
| `internal/app/agentmesh` | `bridge/agentmeshkafka`, `tau` |

### 3.2 Violations détectées

| ID | Règle | Site | Sévérité | Statut |
|---|---|---|---|---|
| V-A1 | `TestArchNoConcreteLLMInDomain` annoncée par ADR-0003 et CLAUDE.md #6 mais absente | `internal/arch_test.go` | **P0** | Anti-patron #6 non gardé |
| V-A2 | Règle `calibration → tau/*` absente de `arch_test.go` (respectée en pratique) | `internal/arch_test.go` | P2 | Drift silencieux possible au prochain refactor |
| V-A3 | `bridge/agentmeshkafka/empirical_i4_test.go:20-21` importe `tau` et `tau/dimensions` (package `_test` sous tag `//go:build empirical`) | bridge test file | P3 | Couvert par build tag ; le walk `TestBridgeNoTauImport` exclut `_test.go` |
| V-A4 | `Decision` nominale (`dispatcher.go:147-157`) porte `ProfileVersion: "M3-default"` en dur, `DateRevision` zéro | dispatcher | P2 | Instrumentation PRD §9.1 incomplète |

Les 4 règles formelles de `arch_test.go` (`TestArchTauNotDependOnOrchestration`, `TestArchTauNotDependOnBridge`, `TestArchDimensionsNotDependOnInvariants`, `TestBridgeNoTauImport`) sont implémentées correctement.

---

## 4. Invariants I1-I5 — alignement chap. III.8.5

| # | Verbatim | Implémentation | Fuzz | Conformité |
|---|---|---|---|---|
| I1 | τ déplace `t_fix`, pas la grandeur | `i1_conservation.go:12-28` `Conserve(x,dec) = dec.Trace.ExchangeID == x.ID` | `FuzzI1_Conservation` 8.6 M exec/s | Conforme V1, limité (1 grandeur proxy). Extension V2 (sens/autorité/support) |
| I2 | Résidu non vide, non recâblable hors ligne | `i2_irreductibility.go:8-94` `Residu`, `Recablage`, `EvaluateI2` | `FuzzI2_Irreductibilite` 8.6 M exec/s | Conforme |
| I3 | D-AUTORITÉ asymétrique (Searle 1995) ; péremption | `i3_authority_asymmetry.go:51-88` ; `I3PerimptionLimite` 2027-01-01 | `FuzzI3_AsymetrieAutorite` 8.2 M exec/s | Conforme avec proxy `TauScore` ≈ D-AUTORITÉ (déferré M5 — voir P1-04) |
| I4 | `i ≈ pendant ⟹ s ≈ pendant` | `i4_coherence.go:15-41` `IsIncoherent(s,i,sT,iT)` ; `EvaluateI4` ne lit que `Regime/Diagnostic` | `FuzzI4_CoherenceContrainte` 9.5 M exec/s | Conforme V1 ; bypass silencieux non détectable sans Trace ventilée (P1-04) |
| I5 | `M(π) ≥ max(\|Aᵢ\|)`, `≤ Σ\|Aᵢ\|` | `i5_composition.go:26-101` calculatoire **dépassement de V2 promis** | `FuzzI5_CompositionConjonctive` 701 K exec/s | Conforme et supérieur à PRD §6.1 |

**Orthogonalité** : zéro import `dimensions ↔ invariants`. Gardée par `arch_test.go`.

---

## 5. Dimensions — alignement chap. III.8.4

| Dimension | Sondes / poids PRD | Implémentation | Conformité |
|---|---|---|---|
| D-SENS | 4 sondes (0.35 / 0.30 / 0.20 / 0.15), Σ=1 | `dimensions/dsens.go` ; délégation `S_reasoner_intent` au `llm.Client` (nil-safe → 0) | Conforme (Hypothèse à corroborer M4) |
| D-AUTORITÉ | 4 sondes équipondérées 0.25 ; saturation `depth≥4` | `dimensions/dauthority.go` | Conforme |
| D-INVARIANT | 4 sondes (0.30 / 0.25 / 0.25 / 0.20 inversé) | `dimensions/dinvariant.go` ; lecture `Exchange.Context[key]` (magic strings — P2-04) | Conforme V1 |

---

## 6. Refus de premier rang (PRD §7.3 — 5 cas)

| Cas | Étape dispatcher | Implémentation | Garde test |
|---|---|---|---|
| Hors frontière τ | 1 | `dispatcher.go:98-101` via `frontierFromExchange` | `TestFrontierCheck_Inside_*` |
| Verrou ontologique D-AUTORITÉ (I3) | 2 | `dispatcher.go:104-110` | `TestRefusOntologiqueDAUTORITE` |
| Profil périmé | 3 | `dispatcher.go:112-119` **conditionnel à `d.profile != nil`** | `TestExpiredProfileRefuses` (tag `e2e`) |
| Incohérence I4 | 5 | `dispatcher.go:131-133` | `TestStep5_*`, `dispatcher_invariants_test.go` |
| Observation non modélisée à fort impact | 8 | `dispatcher.go:160-166` → `Trace.UnmodeledObservations` (rapport, pas Refus — conforme PRD §7.2 #4) | `TestUnmodeledObservationsReported` |

**Trap opérationnelle** : `app.NewDispatcher()` instancie via `NewDispatcher` (sans profil) — la garde de péremption (cas 3) est **silencieusement désactivée**. Voir **P0-02**.

---

## 7. Pseudo-algorithme dispatcher (PRD §10) — 8 étapes

| Étape | PRD §10 | Implémentation | Statut |
|---|---|---|---|
| 1 | Frontière | `dispatcher.go:98-101` | OK |
| 2 | Garde I3 ontologique | `dispatcher.go:104-110` | OK |
| 3 | Garde péremption | `dispatcher.go:112-119` | OK **conditionnel** (P0-02) |
| 4 | Scores D-SENS, D-INVARIANT | `dispatcher.go:121-129` | OK |
| 5 | Garde I4 cohérence | `dispatcher.go:131-133` | OK |
| 6 | Composite τ_score pondéré | `dispatcher.go:136-139` poids hardcodés `defaultDimensionWeights` | **Écart fonctionnel** — `Profile.Weights` jamais lu (P1-09) |
| 7 | Hystérèse `LastRegime` | `dispatcher.go:142-145` simplifié à `Deterministe` dans la bande | **Écart spec** (P1-07) |
| 8 | Évaluation invariants | `dispatcher.go:160-166` | OK |

L'ordre 1-8 est respecté ; les early-exits ne peuvent pas être réordonnés (exigence principale PRD §10).

---

## 8. Observations P0 — bloquantes

### P0-01 — Anti-patron #6 sans garde (`TestArchNoConcreteLLMInDomain` manquant)
- **Fichier** : `internal/arch_test.go`
- **Origine** : ADR-0003 dernière ligne ("La garde statique `TestArchNoConcreteLLMInDomain`…") + CLAUDE.md §Anti-patrons #6 + PRD §7.2 #6
- **Constat** : aucune fonction de ce nom n'existe. L'absence d'import SDK LLM concret tient uniquement par convention humaine, non par CI.
- **Fix** : ajouter un walk AST sur `internal/tau/**/*.go` et `internal/orchestration/*.go` détectant tout import contenant les substrings `anthropic`, `openai`, `mistralai`, `cohere`, `google.golang.org/genai`, `huggingface`, etc. Modèle : calque `TestBridgeNoTauImport` (déjà dans `arch_test.go`).
- **Effort** : 1 demi-journée.
- **Lien autre** : V-A1.

### P0-02 — Trap opérationnelle : `app.NewDispatcher()` désactive silencieusement la garde de péremption (anti-patron #3)
- **Fichier** : `internal/app/app.go` + `internal/orchestration/dispatcher.go:112-119`
- **Constat** : `dispatcher.Decide` à l'étape 3 ne vérifie `today > DateRevision` que si `d.profile != nil`. `app.NewDispatcher()` (chemin par défaut de la CLI `tau decide`) ne fournit pas de profil. Conséquence : un opérateur qui utilise la CLI standard ne déclenche **jamais** la garde de péremption, alors qu'elle est listée parmi les cinq Refus de premier rang (PRD §7.3 #4) et que l'anti-patron #3 PRD §7.2 prétend la couvrir.
- **Fix** : soit (a) rendre `NewDispatcher` package-private et exposer uniquement `NewDispatcherWithProfile` (avec un profil par défaut M3 si l'appelant n'en fournit pas), soit (b) charger automatiquement un profil par défaut au boot CLI. Option (a) **recommandée** — plus alignée avec la doctrine « refus = décision de premier rang ».
- **Effort** : 1 jour (incluant tests E2E mis à jour).
- **Lien autre** : V-A4.

---

## 9. Observations P1 — majeures

### P1-01 — Triple `Thresholds` parallèle (D1)
3 définitions sémantiquement équivalentes : `tau.TraceThresholds` (`operator.go:71-77`) ↔ `orchestration.Thresholds` (`thresholds.go:5-11`) ↔ `calibration.Thresholds` (`profile.go:19-26`). Drift potentiel : `calibration.Thresholds` a `HysteresisGap`, les deux autres non.
**Fix** : extraire `internal/thresholds/` (couche transverse de types valeur). Mettre à jour `arch_test.go` pour l'autoriser. **ADR-0006 à créer**.
**Effort** : 1-2 jours.

### P1-02 — Quatre packages morts (squelettes `doc.go` seuls)
`internal/{config,errors,metrics,testutil}/doc.go` jamais peuplés depuis M0. PRD §14.2 promet pourtant `DispatchError`, `RefusError`, `CalibrationError` typés.
**Fix recommandé** : option (a) **peupler** `internal/errors` avec les types typés (aligné spec) ; supprimer les 3 autres `doc.go` si non peuplés.
**Effort** : 1-2 jours.

### P1-03 — Gate CI couverture per-package non implémenté
`.github/workflows/coverage.yml:22-27` annonce explicitement *"90% per-package gate activates in M1+"* — jamais activé. Couverture mesurée conforme mais aucune protection régression.
**Fix** : ajouter step `go tool cover -func=coverage.out | grep "internal/tau/"` + check ≥ 90.
**Effort** : 30 min.

### P1-04 — `tau.Trace` sans scores ventilés D-SENS/D-AUTORITÉ/D-INVARIANT
PRD §9.1 (lignes 429-430) spécifie `Trace.DSens, DAuthority, DInvariant dimensions.Score`. Absent. Conséquences :
- `EvaluateI3` utilise `TauScore` proxy imparfait (`i3_authority_asymmetry.go:73-76`)
- `EvaluateI4` ne détecte pas un bypass silencieux (`i4_coherence_test.go:77-101`)
- `calibration.simulate` doit dupliquer la logique du dispatcher
**Fix** : enrichir `Trace` avec les 3 scores ventilés ; peupler aux étapes 2/4 du dispatcher.
**Effort** : 1-2 jours.
**Lien autre** : V6.

### P1-05 — Duplication logique frontière (D2)
`dispatcher.go:176-184 frontierFromExchange` vs `i2_irreductibility.go:48-68 Recablage`.
**Fix** : promouvoir en méthode `tau.Exchange.FrontierCheck()`. Drift impossible.
**Effort** : 1/2 jour.

### P1-06 — Risque goroutine leak `StreamAsTauExchanges`
`internal/app/agentmesh.go:75-93` propage `errs` sans wrapper. `sendErr` (`file_adapter.go:36-52`) bufferise à 8 puis drop silencieux. Le contrat côté appelant n'est documenté qu'en godoc.
**Fix** : (a) wrapper draine `errs` sur `ctx.Done()`, ou (b) test `t.Cleanup` qui force la fermeture.
**Effort** : 1/2 jour.

### P1-07 — Hystérèse simplifiée (PRD §10 étape 7)
`Thresholds.HysteresisGap` existe côté `calibration` mais jamais lu par le dispatcher. Bande hystérèse hardcodée à `Deterministe` sans mémoire `LastRegime`.
**Fix** : soit (a) implémenter `LastRegime` (map concurrente x.ID→Regime + TTL), soit (b) amender PRD §10.1 et créer un ADR documentant la simplification V1.
**Effort** : 30 min (b), 1-2 jours (a). **Recommandation : (b) avec ADR-0007** ; (a) déferré à V0.2.

### P1-08 — Global exporté mutable `I3PerimptionLimite`
`i3_authority_asymmetry.go:13` : `var I3PerimptionLimite time.Time` — exporté, non-const, mutable runtime. Le `//nolint:gochecknoglobals` documente l'exception mais ne protège pas contre la mutation depuis un test externe.
**Fix** : remplacer par `func I3PerimptionLimite() time.Time` (getter pur).
**Effort** : 30 min.

### P1-09 — `Profile.Weights` jamais appliqués au runtime
Étape 6 du dispatcher (`dispatcher.go:136-139`) utilise `defaultDimensionWeights` package-level. Les `Profile.Weights.{SensProbes,AuthorityProbes,InvariantProbes}` issus de `tau calibrate` ne sont jamais lus.
**Fix** : injecter `Profile.Weights` via `NewDispatcherWithProfile`. Lier au runtime.
**Effort** : 1 jour.
**Lien autre** : P2-03.

---

## 10. Observations P2 — qualité

| ID | Description | Site |
|---|---|---|
| P2-01 | Globaux exportés en lecture seule sans pattern fonctionnel (`defaultDimensionWeights`, `defaultThresholds`, `intents`) | `dispatcher.go:16`, `app.go:11`, `generator.go:186` |
| P2-02 | `regimeString` dupliqué avec casses divergentes (PascalCase vs lowercase) — risque désync `tau calibrate` | `cmd/generate-corpus/generator.go:31-39` vs `bridge/agentmeshkafka/empirical_i4_test.go:159-169` |
| P2-03 | Voir P1-09 |
| P2-04 | `Exchange.Context` magic strings (`"event_registry"`, etc.) — typo silencieux possible | `dimensions/dinvariant.go:66,78,91,111` |
| P2-05 | `CorpusEntry.ExpectedRegime` 4 valeurs valides sans enum/validation | `calibrate.go:21-23` |
| P2-06 | `simulate` duplique `Decide` (logique calibration) | `calibrate.go:112-126` |
| P2-07 | `Aggregate` parcourt 2× la pile (perf I5) | `i5_composition.go:26-39` |
| P2-08 | `durationNs` clamp 1ns sans détection clock jump | `dispatcher.go:63-68` |
| P2-09 | Calcul couverture global n'inclut pas packages vides (cohérent mais à clarifier) | — |
| P2-10 | `cmd/tau/main.go:run` partiellement testable (76.1 %) | `cmd/tau/main.go:21-50` |
| P2-11 | `app.selectLLM` panic au lieu d'erreur typée | `app.go:26-31` |
| P2-12 | `TestEndToEnd_*` rebuild binaire à chaque test | `cmd/tau/main_test.go` |

---

## 11. Observations P3 — nice-to-have

| ID | Description | Site |
|---|---|---|
| P3-01 | Vérifier typographie FR sur commentaires structurants `.go` | divers |
| P3-02 | `tau.Kernel` purement documentaire, aucun `var _ Kernel = (*Dispatcher)(nil)` | `tau/operator.go:100-102` |
| P3-03 | `ruvector.db` tracké (`M` git status) — PRD §3.2 exclut RAG V1 | repo root |
| P3-04 | `tau.exe`, `generate-corpus.exe` au root | repo root |
| P3-05 | 8 `cov*.out` au root | repo root |
| P3-06 | `DiscoveryMode` int sans `String()`/`MarshalJSON` (JSON `1`/`2`/`3`) | `operator.go:42-51` |
| P3-07 | `Regime` int sans `MarshalJSON` (JSON `1`/`2`/`3`) | `operator.go:10-17` |
| P3-08 | `EmpiricalI4Stats.Sensitivity = -1` au lieu de `*float64 omitempty` | `classifier.go:103-149` |
| P3-09 | `fuzzRefTime` global (test-scope) — convertir en helper | `fuzz_targets_test.go:106` |
| P3-10 | Diagnostics en string littéraux — sentinels `tau.Diag*` à généraliser | divers |
| P3-11 | Cohérence renvois CLAUDE.md §Anti-patrons #2 / PRD.md §4.3 → `TestFrontierCheck_Inside_*` (au lieu de `TestRefusHorsFrontiere`) | `CLAUDE.md`, `PRD.md` |
| P3-12 | Renommer `ExpectedRegime` en `LabeledRegime`/`CorpusLabel` (lever proximité lexicale anti-patron #1) | `calibration/calibrate.go:21-23` |

---

## 12. Couverture par package

| Package | Mesuré | Cible PRD | Écart | Action |
|---|---|---|---|---|
| `internal/tau` | 100.0 % | ≥ 90 % | +10 | maintenir |
| `internal/tau/dimensions` | 98.7 % | ≥ 90 % | +8.7 | maintenir |
| `internal/tau/invariants` | 97.1 % | ≥ 90 % | +7.1 | maintenir |
| `internal/orchestration` | 91.1 % | ≥ 80 % | +11.1 | maintenir |
| `internal/calibration` | 91.8 % | ≥ 80 % | +11.8 | maintenir |
| `internal/bridge/llm` | 100.0 % | ≥ 80 % | +20 | maintenir |
| `internal/bridge/agentmeshkafka` | 89.6 % | ≥ 80 % | +9.6 | maintenir |
| `internal/app` | 95.5 % | ≥ 80 % | +15.5 | maintenir |
| `internal/{config,errors,metrics,testutil}` | n/a | — | — | **P1-02** |
| `cmd/tau` | 76.1 % | — | — | P2-10 (extract `runMain`) |
| `cmd/generate-corpus` | 89.2 % | — | — | maintenir |

---

## 13. Duplications consolidées

| # | Type | Sites | Action |
|---|---|---|---|
| D1 | Struct `Thresholds` triplée | `orchestration` ↔ `tau` ↔ `calibration` | R1 (P1-01) |
| D2 | `frontierFromExchange` ↔ `Recablage` | `dispatcher.go` ↔ `i2_irreductibility.go` | R5 (P1-05) |
| D3 | `regimeString` casses divergentes | `cmd/generate-corpus` ↔ `bridge/agentmeshkafka/empirical_i4_test.go` | méthode `Regime.String()` (P2-02) |
| D4 | `simulate` ↔ `Decide` | `calibrate.go` ↔ `dispatcher.go` | résolu par P1-04/R3 |
| D5 | `EmpiricalDecision` ↔ `tau.Decision` (DTO bridge) | `classifier.go` ↔ `tau/operator.go` | **Justifié ADR-0005** — NE PAS toucher |
| D6 | `AgentMeshExchange` ↔ `tau.Exchange` (DTO bridge) | `adapter.go` ↔ `tau/operator.go` | **Justifié ADR-0005** — NE PAS toucher |
| D7 | Construction Exchange dans tests | tests | peupler `internal/testutil` (P1-02) |

---

## 14. Anti-objectifs PRD §3.3 — conformité

| Anti-objectif | Statut |
|---|---|
| Pas de framework agentique | OK |
| Pas d'orchestrateur (d'agents externes) | OK — `Dispatcher` orchestre des étapes internes |
| Pas de wrapper LLM | OK — interface étroite, 0 SDK concret |
| Pas de RAG | OK — `ruvector.db` exclu V1 (P3-03 à nettoyer) |
| Pas de service réseau dans le cœur | OK |
| Pas de prédiction de comportement | OK avec nuance lexicale `ExpectedRegime` (P3-12) |

---

## 15. ADR — cohérence

| ADR | Conformité | Contradiction |
|---|---|---|
| ADR-0001 Clean Arch 4 couches | OK | Règle `calibration → tau/*` listée mais non gardée (V-A2) |
| ADR-0002 Go 1.25 toolchain | OK | aucune |
| ADR-0003 LLM client injecté | **Partielle** | `TestArchNoConcreteLLMInDomain` annoncé mais **absent** du code (P0-01) |
| ADR-0004 AgentMeshKafka bridge | OK | aucune |
| ADR-0005 AgentMeshKafka DTO neutre | OK | aucune |

**ADR à créer** (issus du refactor agressif) :
- **ADR-0006 — Types valeur transverses** (`internal/thresholds`, R1)
- **ADR-0007 — Hystérèse V1 simplifiée** (P1-07, option b)
- **ADR-0008 — Trace ventilée D-SENS/D-AUTORITÉ/D-INVARIANT** (P1-04)
- **ADR-0009 — Types d'erreurs typées (`internal/errors` peuplé)** (P1-02)
- **ADR-0010 — Bridge TauGo ↔ cia-runtime (V0.2)** (cf. §17, déferré V0.2)

---

## 16. Purge agressive — inventaire actionnable

### Tier 1 — orphelins immédiats (`rm -f`)

| Fichier | Taille | Justification |
|---|---|---|
| `cov-e2e.out`, `cov-empirical.out`, `cov-integration.out`, `cov-merged.out`, `cov-unit.out`, `cov.out`, `cov_after.out`, `cov_before.out`, `cover.out`, `cover_dims.out` | 350 KB | Artefacts coverage non-trackés, jamais commités |
| `generate-corpus.exe`, `tau.exe` | 6.9 MB | Binaires Windows régénérables `make build` |
| `scripts/__pycache__/` | 44 KB | Bytecode Python regénéré |
| `.claude-flow/` | n/a | Trace agent locale, déjà ignoré `.gitignore:39` |

### Tier 2 — fichiers trackés morts (`git rm`)

| Fichier | LOC | Justification |
|---|---|---|
| `internal/config/doc.go` | 4 | Package vide jamais importé (P1-02) — **dépend du choix peupler vs supprimer** |
| `internal/metrics/doc.go` | 4 | idem |
| `internal/testutil/doc.go` | 3 | idem (NB : à peupler avec `BuildExchange` helper — voir D7) |
| `ruvector.db` | 1.6 MB | Binary trackée accidentellement 2026-05-23 commit `387c787`, 0 grep matches, PRD §3.2 exclut RAG V1 |

NB : `internal/errors/doc.go` **conservé et peuplé** (option (a) du P1-02).

Ajout `.gitignore` : `*.db`, `*.sqlite`, `*.exe`, `cov*.out`, `cover*.out`.

### Tier 3 — archivage docs M0-M6 (9 824 LOC)

`docs/superpowers/plans/*.md` → `docs/archive/plans-m0-m6/`

| Plan | LOC | Statut |
|---|---|---|
| `2026-05-23-M1-dispatcher-stub-llm.md` | 1 017 | M1 clos `v0.0.2-alpha` |
| `2026-05-23-M2-dimensions-gardes.md` | 2 416 | M2 clos `v0.0.3-alpha` |
| `2026-05-24-M3-invariants-fuzz.md` | 2 047 | M3 clos `v0.0.4-alpha` |
| `2026-05-24-M4-agentmeshkafka-bridge.md` | 2 134 | M4 clos `v0.0.5-alpha` |
| `2026-05-24-M5-calibration-drift.md` | 1 080 | M5 clos `v0.0.6-alpha` |
| `2026-05-24-M6-release-v0.1.0.md` | 1 130 | M6 clos `v0.1.0` |

---

## 17. Bloqueurs migration V0.2 (cia-runtime, mécanisation Lean 4)

| Bloqueur | Nature | Préparation refactor |
|---|---|---|
| `tau.Exchange` + 5 invariants en Go pur | Frontière de langage | ADR-0010 préalable ; protocole sérialisation (JSON ou Protobuf) ; dépôt compagnon `cia-runtime` |
| `BoundsHold` (I5) et `IsIncoherent` (I4) — fonctions pures fuzzées | Opportunité immédiate | candidates 1res à mécanisation Lean — code prêt |
| `EvaluateI3WithClock` horloge injectable | Abstraction temps en Lean | modélisation POSIX int |
| Scores `[0,1]` `float64` Go vs Lean `Float`/`Rat` | Décision architecturale | ADR à inclure dans ADR-0010 |

V0.3 (TUI replay) — pas de bloqueur si P1-04 (Trace ventilée) est résolu en V0.1.x.

---

## 18. Recommandations refactor architectural — table de synthèse

| ID | Recommandation | Résout | ADR requis | Effort | Priorité |
|---|---|---|---|---|---|
| R0-1 | Ajouter `TestArchNoConcreteLLMInDomain` | P0-01, V-A1 | non | 1/2 j | **P0** |
| R0-2 | `app.NewDispatcher()` charge profil par défaut OU rendre `NewDispatcher` package-private | P0-02 | possible | 1 j | **P0** |
| R1 | Extraire `internal/thresholds/` couche transverse | P1-01, D1 | **ADR-0006** | 1-2 j | P1 |
| R2 | Peupler `internal/errors` (types typés) + supprimer 3 autres doc.go | P1-02 | **ADR-0009** | 1-2 j | P1 |
| R3 | Enrichir `tau.Trace` avec scores ventilés | P1-04, P2-03, P2-06, V6 | **ADR-0008** | 1-2 j | P1 |
| R4 | Injecter `Profile.Weights` dans `Dispatcher` | P1-09, P2-03 | non | 1 j | P1 |
| R5 | Promouvoir `frontierFromExchange` → `Exchange.FrontierCheck()` | P1-05, D2 | non | 1/2 j | P1 |
| R6 | Activer gate CI per-package ≥ 90 % `tau/*` | P1-03 | non | 30 min | P1 |
| R7 | Hystérèse — option (b) ADR + spec PRD §10.1 | P1-07 | **ADR-0007** | 1 h | P1 |
| R8 | `func I3PerimptionLimite() time.Time` getter | P1-08 | non | 30 min | P1 |
| R9 | Drainer `errs` dans `StreamAsTauExchanges` | P1-06 | non | 1/2 j | P1 |
| R10 | Ajouter règle `calibration → tau/*` dans `arch_test.go` | V-A2 | non | 15 min | P2 |
| R11 | Method `Regime.String()` + `MarshalJSON` | D3, P2-02, P3-07 | non | 1 h | P2 |
| R12 | Method `DiscoveryMode.String()` + `MarshalJSON` | P3-06 | non | 30 min | P2 |
| R13 | Diagnostics en constantes `tau.Diag*` | P3-10 | non | 1 h | P2 |
| R14 | Typer `Exchange.Context` (struct + bag) | P2-04 | mineur | 1 j | P2 |
| R15 | `app.selectLLM` → erreur typée | P2-11 | non | 30 min | P2 |
| R16 | Refactor `cmd/tau/main.go` → `runMain(args,in,out,stderr) int` | P2-10 | non | 1 h | P3 |
| R17 | `TestMain` réutilise binaire CLI buildé | P2-12 | non | 30 min | P3 |
| R18 | Renommer `ExpectedRegime` → `LabeledRegime` | P3-12 | non | 15 min | P3 |
| R19 | Aligner CLAUDE.md / PRD.md sur `TestFrontierCheck_Inside_*` | P3-11 | non | 15 min | P3 |
| R20 | Purge Tier 1 + Tier 2 + Tier 3 + maj `.gitignore` | §16 | non | 30 min | P1 |

**Effort cumulé** : ~12-15 jours-ingénieur. Aucune rupture API publique : `tau.Kernel`, `tau.Decide`, `tau.Decision`, `tau.Trace` restent compatibles (Trace enrichie additivement).

---

## 19. Verdict de l'intégrateur

TauGo v0.1.0 est **conforme à la spec à 85-90 %** avec dette technique limitée. Les **2 P0** réintroduits par l'intégrateur (P0-01 garde manquante, P0-02 trap opérationnelle péremption) sont **critiques de sécurité** au sens PRD §7.3 et doivent passer en R0-1/R0-2 du plan. Les 9 P1 sont fixables sans rupture API. Les 12 P2 et 10 P3 peuvent suivre en lots.

**Refactor architectural agressif tolérable** : aucun déplacement de package au-delà de la création de `internal/thresholds/` (R1) ; pas de changement de la doctrine `Decide(ctx, Exchange) → Decision`. Tag `v0.1.0` reste sémantiquement valide après les R0+R1+R3 ; un `v0.1.1` couvrira l'ensemble du plan AUDITPlan.md.

---

*AUDIT.md — consolidé 2026-05-24, branche `main`, commit base `5a68c12`. Réviser à chaque clôture de phase D du AUDITPlan.md.*
