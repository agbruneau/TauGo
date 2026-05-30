# Changelog

Conforme à [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et au [Versionnage Sémantique](https://semver.org/lang/fr/).

## [Non publié] — v0.1.2-pre · retrait complet CI/CD (projet pure-local)

**Décision structurelle** orchestrée selon [ADR-0010](docs/adr/0010-retrait-ci-cd-pure-local.md). Le projet devient *pure-local* : validation entièrement déléguée au poste développeur (`make test && make lint && make fuzz`), plus aucun gate automatisé bloquant.

### Corrigé — post-audit de régression 2026-05-29

Lot de correctifs issus de l'audit multi-agents 2026-05-29 ([`docs/archive/audits/2026-05-29-AUDIT-v0.1.2-pre/`](docs/archive/audits/2026-05-29-AUDIT-v0.1.2-pre/)).

- **C1-01 (golden corpus de calibration dégénéré) — RÉSOLU** *([ADR-0012](docs/adr/0012-golden-corpus-calibration-schema.md))*. `tests/calibration/golden-corpus.jsonl` était en schéma `Exchange` (et non `CorpusEntry`) → `tau calibrate` produisait un profil dégénéré (no-op depuis M5 ; runtime `Decide` jamais affecté car `DefaultProfile`). Corpus régénéré au schéma `CorpusEntry` via `cmd/generate-corpus --scored` (170 lignes ; scores ventilés réels ; `probabiliste` 90 / `deterministe` 50 / `refus_authority` 30 ; `refus_i4` 0 — limitation I4 connue). Profil non dégénéré (seuils Det 0,45 / Prob 0,65 / Auth 0,70 / Sens 0,30 / Inv 0,30). Hash re-épinglé `8e5dc2fc…40c1`. Validation CLI rétablie : `loadCorpus` applique migration + `Validate`, corpus invalide → exit 2 (`TestRunCalibrate_CorpusInvalidRegime_NonZero`, `TestRunCalibrate_CorpusLegacyExpectedRegime_Migre`).
- **Q5-01** : `RefusError.Is` compare le `Diagnostic` aux sentinels → `errors.Is(refus, ErrFrontiereFranchie)` matche (ADR-0009).
- **I2-04** : I1/I2 utilisent la constante `tau.DiagFrontiereFranchie` (anti-drift). **I2-03** : test I4 renommé `..._NoVentilatedScores_Held`. **C1-04** : message `Validate` corrigé en `LabeledRegime`. **P4-04** : commentaire `BoundsHold` corrigé (mono-passe).
- **P4-02** : `BenchmarkDecide` (Det/Prob/Refus) + `BenchmarkScoreD*` ajoutés (directive perf 5 vérifiable).
- **R3-01** : godoc `StreamAsTauExchanges` documente la sémantique best-effort/lossy de `errc`. **R3-02** : docstring `SetTuning` ne survend plus l'atomicité transactionnelle.
- **A6-03/04** : `arch_test.go` purge les références `config`/`metrics` (supprimés) + règles `from` défensives pour `internal/errors` et `internal/testutil`.
- **Documentaire** : survente couverture (92,1 % → 89,2 % `-coverpkg`) et débits fuzz (méthode fonction-propriété vs moteur `go test -fuzz`) corrigées ; arborescence resynchronisée (`generate-corpus`, retrait `config`/`metrics`, `tests/calibration`) ; anti-patrons `theory/07` 4 théoriques + 3 gardes d’ingénierie = 7 ; graphe de connaissance régénéré (`understand --full`, 303 nœuds) + dashboard `gh-pages` rafraîchi.
### Supprimé

- **GitHub Actions** — `.github/workflows/ci.yml` (143 l., matrice 3 OS × Go 1.25, `go test -race` via CGO, lint, build, cross-compile, fuzz smoke) et `.github/workflows/coverage.yml` (86 l., gate per-package `tau/*` ≥ 90 %, global ≥ 80 %). Dossier `.github/` retiré entièrement.
- **Cibles `Makefile` CI-only** : `fuzz-long` (24 h nocturne), `e2e` (tag `integration`), `e2e-calibration` (tag `e2e`), `empirical-i4` (tag `empirical`), `build-reproducible` (timestamp gelé `1778889600`). Le code de test reste inchangé — seules les cibles d'orchestration sont retirées. Commandes `go test -tags=...` documentées dans `README.md` §5.D, `CLAUDE.md` § Commandes essentielles, `PRD.md` §15.3 comme équivalents directs.
- **Badges README** `[![CI]]` et `[![Coverage]]`.
- **Section gates automatisés** — `PRD.md` §15.3 « Gates CI » renommée « Gates locaux », gates conservés comme **objectifs vérifiables** via `make coverage`, plus comme blocage de merge.
- **Alerte 30 jours avant péremption I3** (passait par CI à fréquence quotidienne) — bascule en cron externe ou check manuel. Risque PRD §18 #9 mitigé par : (a) garde runtime `TestExpiredProfileRefuses` qui bloque toute décision si profil périmé ; (b) `app.NewDispatcher()` qui charge un profil par défaut activant la garde sur chemin CLI (P0-02 fermé v0.1.1).

### Modifié (alignement documentaire)

- **`README.md`** — section 5.D (exemples fuzz/e2e via `go test` direct), §6 (refactor v0.1.2 documenté), §9 (stack technique : rangée « Validation » remplaçant « CI »), §10 (cadence revue I3 actualisée), section ADRs (ajout 0010).
- **`CLAUDE.md`** — §Projet (« Validation » remplace « CI »), § Commandes essentielles (retrait cibles CI-only + remplacements `go test`), directive §10 (« Validation locale obligatoire »), directive §8 (veille I3 sans CI), pattern d'exécution §Agent teams (`tests CI verts` → `make test && make lint && make fuzz verts en local`), §Références (ADR-0010 ajoutée, ADR-0011 réservée pour HGL), footer V0.5.
- **`PRD.md`** — §3.1 (livrable « Validation locale »), §3.2 (ADR-0010 réallouée → ADR-0011 pour HGL), §6.1 C3 / §7.2 anti-patron #3 / §15.2 fuzz / §15.3 Gates locaux / §16 M0+v0.1.2-pre row / Livrables M0 minimaux (workflows barrés) / §17 critères #4/#5/#8 + intro / §18 risques #4/#9 / §20.2 prochaines étapes (#2 tag v0.1.2, #3 ADR-0011, #9 réintroduction CI minimale option V0.2+).
- **`Makefile`** — cibles `.PHONY` réduites, retrait `fuzz-long`/`e2e`/`e2e-calibration`/`empirical-i4`/`build-reproducible`, suppression commentaires « CI » associés.
- **`cmd/tau/main.go`** : `version = "0.1.1-pre"` → `"0.1.2-pre"`.

### Ajouté

- **ADR-0010** [`docs/adr/0010-retrait-ci-cd-pure-local.md`](docs/adr/0010-retrait-ci-cd-pure-local.md) — décision, périmètre supprimé/conservé, conséquences positives/négatives acceptées, réversibilité, alternatives considérées (CI minimale / autre orchestrateur / pre-commit hooks V0.2), critères de retournement (≥ 2 contributeurs / bug atteignant main / externalisation), vérifications post-décision.

### Conservé (intentionnellement)

- Tous les **tests** (`make test`, `make test-short`, fuzz I1-I5 30 s, e2e, empirical) — code inchangé.
- **Gates de qualité** comme objectifs locaux vérifiables : `tau/*` ≥ 90 %, global ≥ 80 % via `make coverage` (rapport HTML).
- **Anti-patrons §7.2 #1-7** — toujours gardés par tests (`TestNoPredictiveAPI`, `TestFrontierCheck_Inside_*`, `TestExpiredProfileRefuses`, `TestUnmodeledObservationsReported`, `TestArchNoConcreteLLMInDomain`, `gochecknoglobals`).
- **Étanchéité Clean Architecture** — `internal/arch_test.go` (7 règles), exécutée par `make test`.
- **Build reproductible** sous toolchain pinnée (`-trimpath -buildvcs=true`) ; seule la cible `make build-reproducible` à timestamp gelé `1778889600` est retirée (était CI-spécifique).

### Vérification (état HEAD)

- `go build ./...` : à valider en local avant push
- `go test -short ./...` : à valider en local avant push
- `go vet ./...` : à valider en local avant push
- Aucune référence orpheline « gate CI », « workflow », « coverage.yml », « CI matrix verte » dans `README.md` / `CLAUDE.md` / `PRD.md` (audit `grep` manuel)

### Réversibilité

Le retrait est réversible à coût quasi nul — les workflows sont récupérables via l'historique git (`git show v0.1.1-pre:.github/workflows/ci.yml`). Réintroduction documentée comme option V0.2+ (`PRD.md` §20.2 #9), conditionnée à : ≥ 2 contributeurs actifs / premier bug atteignant `main` qui aurait été bloqué par CI / externalisation du projet.

---

## [Non publié] — v0.1.1-pre · refactor consolidation post-audit

**Refactor agressif complet** orchestré par Agent teams selon `AUDITPlan.md` (42 tâches T-001..T-040, 4 vagues parallèles). Source : `AUDIT.md` 2026-05-24, base commit `5a68c12`.

### Ajouté

- **4 ADRs** : `0006-types-valeur-transverses.md` (extraction `internal/thresholds/`), `0007-hysteresis-v1-simplifiee.md` (V1 simplifiée déclarée, cible V0.2 pour `LastRegime`), `0008-trace-ventilee-scores-dimensions.md` (champs `Trace.DSens/DAuthority/DInvariant`), `0009-types-erreurs-typees.md` (`DispatchError`, `RefusError`, `CalibrationError`, sentinels `errors.Is`-compatibles).
- **Package `internal/thresholds/`** : type valeur transverse partagé (couche descendante, étanchéité gardée par `arch_test.go`). Déduplication D1 (3 `Thresholds` → 1).
- **Package `internal/errors/`** peuplé : types typés + sentinels (`ErrFrontiereFranchie`, `ErrPeremptionProfile`, `ErrIncoherenceI4`, `ErrVerrouOntologique`). Couverture 100 %.
- **Package `internal/testutil/`** peuplé : `BuildExchange(opts ...Option)` (PoC 3 tests migrés).
- **`Trace` ventilée** (ADR-0008) : `Trace.DSens`, `Trace.DAuthority`, `Trace.DInvariant` peuplés aux étapes 2/4 du dispatcher ; lus directement par `EvaluateI3` et `EvaluateI4` (suppression du proxy `TauScore`).
- **`Profile.Weights` injecté** à l'étape 6 du dispatcher (résout AUDIT P1-09 : poids calibrés sans effet runtime).
- **`Exchange.FrontierCheck()`** méthode publique (résout duplication D2 entre `frontierFromExchange` et `Recablage`).
- **`Regime.String() + MarshalJSON/UnmarshalJSON`** + identique pour `DiscoveryMode`. JSON désormais en string PascalCase (rétro-compat int préservée pour corpus v0.1.0).
- **Constantes `tau.Diag*`** : `DiagFrontiereFranchie`, `DiagPeremptionProfile`, `DiagVerrouOntologique`, `DiagIncoherenceI4`. Élimine les littéraux dupliqués.
- **`var _ tau.Kernel = (*Dispatcher)(nil)`** assertion compile-time (P3-02).
- **`cmd/tau/runMain(args, in, out, stderr) int`** testable directement (couverture 76.1 % → 90.0 %). `TestMain` réutilise un binaire unique (P2-12).
- **`CorpusEntry.Validate()`** + migration `ExpectedRegime → LabeledRegime` (rétro-compat JSON corpus v0.1.0).

### Modifié

- **P0-01 fermé** : nouvelle garde `TestArchNoConcreteLLMInDomain` (`internal/arch_test.go`) — walk AST détecte 12 substrings de SDK LLM concrets dans `internal/tau/**` et `internal/orchestration/`. Anti-patron PRD §7.2 #6 **désormais gardé en CI** (était annoncé par ADR-0003 mais absent).
- **P0-02 fermé** : `app.NewDispatcher()` charge `calibration.DefaultProfile()` par défaut. La garde de péremption (anti-patron #3) est désormais active sur le chemin CLI standard. Godoc `orchestration.NewDispatcher` documente explicitement le risque sans-profil (test internes uniquement).
- **Gate CI per-package** activé dans `.github/workflows/coverage.yml` : `internal/tau/*` ≥ 90 %, global ≥ 80 %.
- **`I3PerimptionLimite`** : variable globale exportée → fonction getter pur `I3PerimptionLimite() time.Time` (anti-patron #7 renforcement).
- **`StreamAsTauExchanges`** : drain explicite de `errs` sur `ctx.Done()` (résout P1-06 goroutine leak).
- **Règle arch `calibration → tau/orch/bridge`** ajoutée à `arch_test.go` (résout V-A2).
- **`BoundsHold` (I5)** : 1 passe au lieu de 2 — bench -46 % ns/op, -50 % allocs ; fuzz I5 inchangé (1.13M exec/s).
- **`Decision.ProfileVersion` / `DateRevision`** lus dynamiquement du profil injecté (résout V-A4).
- **`EmpiricalI4Stats.Sensitivity` / `Specificity`** : `float64` (sentinel `-1`) → `*float64 omitempty`.
- **`app.selectLLM`** : `panic` → `error` typée `*errors.DispatchError`.
- **CLAUDE.md / PRD.md / PRDPlanning.md** : alignement sur `TestFrontierCheck_Inside_*` (anciennement `TestRefusHorsFrontiere`). PRD §10.1 amendé avec note V1 hystérèse simplifiée + renvoi ADR-0007.
- **Typographie FR** : 10 substitutions NBSP supplémentaires dans commentaires structurants Go.

### Supprimé / Archivé

- **Purge agressive** : 10 fichiers `cov*.out` (~350 KB), 2 binaires `*.exe` (~6.9 MB), `scripts/__pycache__/`, `.claude-flow/`, **`ruvector.db` désindexé** (anti-RAG PRD §3.2).
- **Packages morts** : `internal/config/doc.go`, `internal/metrics/doc.go` supprimés (jamais peuplés, jamais importés depuis M0).
- **Plans M0-M6 archivés** : `docs/superpowers/plans/` → `docs/archive/plans-m0-m6/` (6 plans, 9 824 LOC), avec `README.md` de redirection.
- **`.gitignore`** durci : `*.exe`, `*.db`, `*.sqlite`, `*.sqlite3`, `__pycache__/`.

### Vérification (état HEAD)

- `go build ./...` : vert
- `go test ./... -count=1 -short` : 14 packages OK
- `go test -tags=e2e ./test/e2e/...` : vert
- `go vet ./...` : vert
- `golangci-lint run ./...` : vert (24 linters)
- `go test -fuzz=FuzzI5 -fuzztime=5s` : 5.7M execs, 0 crash, 1.13M exec/s
- **Couverture globale** : 92.1 % (était 90.9 %)
- **Couverture `internal/tau/*`** : 100 % / 98.7 % / 92.7 % (gate ≥ 90 % respecté)
- **Anti-patrons §7.2** : 7/7 gardés (était 6/7 — #6 désormais couvert)
- **ADRs** : 9 total (4 nouvelles, statut Accepté)

### Tâches déférées V0.2

- **T-026** Typage `Exchange.Context` (struct + bag) — risque moyen, déféré pour stabilité v0.1.1.
- **T-029** Détection clock-jump dans `durationNs` — optionnel.
- **ADR-0010** Bridge TauGo ↔ cia-runtime (mécanisation Lean 4) — déferré V0.2.

### Revue intégrée finale

`ruflo-core:reviewer` : verdict **Accepté avec réserves** (3 findings mineurs sur cohérence ADR↔code — corrigés sauf 2 cosmétiques tracés). Aucun finding bloquant.

### Ajouté (vague de couverture pré-audit, déjà annoncée)

- **Couverture globale 76.3 % → 90.9 %** (PRD §17 critère #5 dépassé sous l'interprétation A, `cmd/*` inclus). Stratégie en 3 vagues P1/P2/P3 :
  - **P1 — `cmd/tau`** (0 % → 76.1 %, 16 tests ajoutés) : refacto `runDecide(in io.Reader, out io.Writer) int` + `runCalibrate(args []string) int` (suppression des `os.Exit` directs, retour d'exit codes). Tests directs `TestRunDecide_*`, `TestRunCalibrate_*`, `TestParseDateRev_*`, `TestLoadCorpus_*`. Le plafond 76.1 % est imposé par `main()` (13 statements wrapper) et les branches d'erreur encode/dispatcher inaccessibles sans mock — documenté.
  - **P2.1 — `cmd/generate-corpus`** (66.3 % → 89.2 %, 8 tests ajoutés) : refacto `run(args []string, stdout io.Writer) int`. Tests `TestRun_HappyPath_{Stdout,FileOutput}`, `TestRun_WithAnnotateFlag`, `TestRun_BadDistribution_Exit2`, `TestRun_CountZero_Exit2`, etc. Default `--output` changé de `testdata/synthetic-corpus.jsonl` à `"-"` (stdout) — comportement plus prévisible en composition pipeline.
  - **P2.2 — branches d'erreur internes** : `tau/dimensions/clamp01` 60 % → 100 % (`TestClamp01_BelowZero/AboveOne`), `calibration/store` 88.6 % → 91.8 % (`TestExportSHA256_FileNotFound`, `TestSave_DirNotExists_ReturnsError`, `TestSave_RefreshCurrentFails_ReturnsError`), `agentmeshkafka/classifier` (`TestEmpiricalI4Summary_UnmodeledCounted`).
  - **P3 — `internal/app/selectLLM`** (66.7 % → 100 %) : `TestSelectLLM_RealBackend_Panics` via `t.Setenv` + `recover` ; couvre la branche `TAUGO_LLM_BACKEND=real` documentée comme `panic` M5+ feature.

### Notes

- **Couverture finale par package** : `internal/bridge/llm` 100.0 %, `tau` 100.0 %, `tau/dimensions` 98.7 %, `tau/invariants` 97.1 %, `internal/app` 95.5 %, `calibration` 91.8 %, `orchestration` 91.1 %, `agentmeshkafka` 89.6 %, `cmd/generate-corpus` 89.2 %, `cmd/tau` 76.1 %. Total **90.9 %**.
- **Audit reviewer** : décision sur l'interprétation 80 % du critère PRD §17 #5 : **interprétation A** retenue (`go tool cover -func ./... total ≥ 80 %`, incluant `cmd/*`). Le coût (~70 LOC pour P1.1 seul) était inférieur au coût d'amender la spec.
- `.gitignore` : ajout `cov*.out`, `cov*.html`, `scripts/__pycache__/`.
- Pas de nouveau tag : changement non-breaking (tests + refacto comportementalement identique). Sera bundle dans la prochaine release v0.1.1 ou v0.2.0.

## [0.1.0] — 2026-05-24

**Première release publique V1.** Tous les critères de succès PRD §17 (10 items) sont opposables par test ou artefact. Six milestones M0-M6 livrés en cascade. Verdict de revue intégrée finale : APPROVE.

### Synthèse des six milestones

| # | Livrable | Tag | Statut |
|---|---|---|---|
| M0 | Squelette + CI 3 OS + arch_test 4 couches + FrontierCheck | `v0.0.1-alpha` | ✓ |
| M1 | Dispatcher minimal + stub LLM déterministe + sous-commande `tau decide` | `v0.0.2-alpha` | ✓ |
| M2 | Trois dimensions D-SENS/D-AUTORITÉ/D-INVARIANT + gardes ontologique et I4 | `v0.0.3-alpha` | ✓ |
| M3 | Cinq invariants I1-I5 + fuzz targets + étape 8 dispatcher | `v0.0.4-alpha` | ✓ |
| M4 | Bridge AgentMeshKafka (DTO neutre, ADR-0005) + campagne empirique I4 (Régime B) | `v0.0.5-alpha` | ✓ |
| M5 | Calibration adaptative byte-identique + drift + étape 3 dispatcher | `v0.0.6-alpha` | ✓ |
| M6 | Docs alignées monographie + typographie française + ADRs 0001-0004 + release | `v0.1.0` | ✓ |

### Critères de succès PRD §17 — 10/10

1. ✓ Dispatch τ instrumenté sur cas BFSI anonymisé — `docs/empirical/case-study-bfsi.md` (312 l.)
2. ✓ Cinq invariants fuzz ≥ 30 s sans panique — `make fuzz` (CI), smoke local 5 s vert
3. ✓ Trace E2E via AgentMeshKafka — `make e2e` (FileAdapter mock, Régime B, PRD §18 risque #1 réalisé)
4. ✓ Build reproductible byte-identique — `make build-reproducible` deux runs → sha256 identique (`5883002eaec677303d503fe4afd279f3eb6fd5db37af4140235a98f680f7ef82`)
5. ✓ Couverture ≥ 80 % global, ≥ 90 % `tau/*` — `tau/dimensions` 96.1 %, `tau/invariants` 97.1 %, packages internes ≥ 87 %
6. ✓ Renvois chap. III.8 dans tous les `docs/` — vérifié grep
7. ✓ Aucun emoji, fabrication, citation non sourcée — audit textuel M6.7
8. ✓ Trois OS supportés — matrice CI `ubuntu-latest × macos-latest × windows-latest`
9. ✓ Quatre anti-patrons gardés — `TestNoPredictiveAPI` (#1), `TestFrontierCheck_*` (#2), `TestI3_DateRevisionRespectee` + `TestExpiredProfileRefuses` (#3), `TestUnmodeledObservationsReported` + `TestStep8_*` (#4)
10. ✓ Profil calibration byte-identique — `TestCalibrationDeterministic` + `TestCalibrate_GoldenCorpus_FixedHash` (hash pinné `d753245b87933f97c6324f54df1572fab7cc68c52bc49baa1b891ab97abff6c7`)

### Ajouté en M6

- **Typographie française canonique** (CLAUDE.md §14.1) appliquée à tous les `.md` du repo : espaces insécables U+00A0 avant `:` `;` `?` `!` `»` et après `«`, guillemets `« … »` dans la prose. Script idempotent `scripts/typography-fr.py` checked-in. 24 fichiers, 983 lignes touchées.
- `docs/theory/06-conditions-validite.md` (124 l.) : renvoi chap. III.8.6 — conditions C1/C2/C3 conjonctives.
- `docs/theory/07-anti-patrons.md` (121 l.) : renvoi chap. III.8.7 — 4 anti-patrons et leurs gardes par test.
- `docs/algorithms/dispatch.md` (407 l.) : pseudo-algorithme PRD §10 documenté en 8 étapes (1 frontière, 2 ontologique, 3 péremption, 4 scores, 5 cohérence I4, 6 composite, 7 hystérèse, 8 invariants).
- `docs/adr/0001-clean-architecture-4-layers.md` (107 l.) : ADR rétroactive M0.
- `docs/adr/0002-go-1.25-toolchain.md` (96 l.) : ADR rétroactive M0.
- `docs/adr/0003-llm-client-injection.md` (119 l.) : ADR rétroactive M1, avec clarification M6.7 (autorisation `tau/dimensions → bridge/llm.Client` car interface).
- `docs/adr/0004-agentmeshkafka-empirical-bridge.md` (126 l.) : ADR rétroactive M4, bi-régime A/B.
- `docs/empirical/case-study-bfsi.md` (312 l.) : cas BFSI anonymisé démontrant la garde I3 préemptive (PRD §17 #1).
- `README.md` final (343 l.) : badges CI/coverage/go-ref/Apache-2.0, doctrine, anti-objectifs, quick start, schéma ASCII des 4 couches, exemples d'usage (`tau decide`, `tau calibrate`, `make fuzz`, `make e2e`), tableau M0-M6, statut I1-I5.
- `docs/archive/plans-m0-m6/2026-05-24-M6-release-v0.1.0.md` (1 130 l.) : sous-plan détaillé M6 (11 tâches M6.0-M6.10).
- `internal/tau/dimensions/*_test.go` : tests `TestDefault{Sens,Authority,Invariant}Weights_StructureAndSum` + branches probes étendues (couverture 83.1 % → 96.1 %).
- `.golangci.yml` : `gochecknoglobals` activé (CLAUDE.md anti-patron #7) ; `//nolint:gochecknoglobals` chirurgicaux sur les globaux read-only documentés (`I3PerimptionLimite`, `defaultDimensionWeights`, `defaultThresholds`, `intents`, `buildTimestamp`).

### Corrigé en M6

- PRD §12.1 : signature `Adapter.Stream` corrigée — retourne `<-chan AgentMeshExchange` (DTO neutre) au lieu de `<-chan tau.Exchange` (cf. ADR-0005). Synchronisation finale code ↔ PRD.
- M3 reviewer obs : `EvaluateI2` zero-residue couvert, godoc `i5_composition.go` justifie la vitesse FuzzI5, commentaire `TestUnmodeledObservationsReported` référence `TestStep8_*`.
- M4 reviewer obs (NB1-NB4) : commentaire arch dans `empirical_i4_test.go`, godoc `TestAdapter_StreamSignature` zero-value, godoc `OtherRefusal` ventilation M4-bis, godoc `StreamAsTauExchanges` comportement ctx vs errs.
- M5 reviewer obs (OBS-1/OBS-2) : commentaires asymétrie date-révision drift vs dispatcher, commentaire `simulate()` renvoyant à PRD §4.

### Notes finales

- **Posture épistémique** : I1 Probable, I2 Confirmé par construction, I3 Probable (daté 2026-05-16, revue 2026-12-01), I4 Hypothèse (campagne empirique synthétique M4 inconclusive — cf. `docs/empirical/I4-report.md`), I5 Probable (bornes algébriques `max(|Aᵢ|) ≤ M(π) ≤ Σ|Aⱼ|`).
- **Régime empirique B actif** : AgentMeshKafka indisponible (PRD §18 risque #1 réalisé). Mock + corpus synthétique reproductible. Bascule Régime A documentée dans `docs/empirical/I4-regime.md`.
- **Audit M6.7** : DÉVIANT_MINEUR sur architecture (A1 `cmd/generate-corpus → bridge` accepté comme outil interne ; A2 `tau/dimensions → bridge/llm` accepté car interface, clarifié ADR-0003). APPROVE_WITH_CONCERNS sur PRD §17, devenu APPROVE après fixes (OBS-1 couverture, OBS-3 gochecknoglobals).
- **Compteurs** : 86 commits sur `main` depuis `aabda39` ; 7 tags (`v0.0.1-alpha` → `v0.1.0`) ; 159 tests + 5 cibles fuzz ; 25 documents Markdown sous `docs/` ; 5 ADRs.
- **Pré-V2** : la mécanisation Lean 4 des invariants (HGL — `agbruneau/InteroperabiliteAgentique/RechercheFondamentale.md`) sera traitée dans un dépôt compagnon à créer. Le TUI Bubble Tea (`tau-stack`) et le calcul effectif de `M(π)` sur piles réelles sont V3.

## [0.0.6-alpha] — 2026-05-24

M5 : calibration adaptative reproductible byte-identique (PRD §17 critère #10), détection de drift sur les 5 critères PRD §11.4, persistance JSON canonique avec `current.json` (symlink + fallback Windows), CLI `tau calibrate`, étape 3 dispatcher (refus profil périmé — anti-patron #3). Revue intégrée : APPROVE.

### Ajouté

- `internal/calibration/calibrate.go` : `Calibrate(corpus, seed)` — grid search déterministe en milli-unités int64 (calque FibGo, évite IEEE-754) sur `Deterministe ∈ [0.10,0.90]` × `HysteresisGap ∈ [0.05,0.20]` × `AuthBlock ∈ [0.70,0.95]` × `SensCoherence ∈ [0.30,0.70]` (InvCoherence = SensCoherence en V1). Tie-break conservateur. Helper `MarshalCanonical/UnmarshalCanonical` (tri JSON récursif sur deux passes via `json.UseNumber()` + `sortedAny`).
- `internal/calibration/weights.go` : `CalibrateWeights` V1 = passthrough (les poids `DefaultProfile()` ne sont pas mutés faute de signal empirique M4 — cf. I4-report.md). Type `WeightHook` pour la stratégie V2 (gradient ou bayésien).
- `internal/calibration/drift.go` : `DriftCriterion` énuméré (5 critères PRD §11.4), `DriftReport`, `Env`, `CheckDrift(profile, now, env)`. `FingerprintCPU()` = tuple `GOOS-GOARCH-NumCPU` (simplification V1), `FingerprintCorpus(path)` = sha256 du fichier. `DriftScoreDistribution` = placeholder V1 documenté (V2 introduira la fenêtre glissante). Skip si fingerprint profile vide (pas de faux positif au premier démarrage).
- `internal/calibration/store.go` : `Store{Dir}` — `Save/Load/LoadCurrent` ; chemin `<Dir>/<ID>-<Version>.json` ; `current.json` symlink sur Linux/macOS, fallback **copie + sidecar `.source`** sur Windows (`os.Symlink` requiert privilège Developer Mode). Tests conditionnés par build tag `!windows`/`windows`.
- `internal/orchestration/dispatcher.go` : étape 3 PRD §10 (reportée de M4) — `if !profile.DateRevision.IsZero() && now().After(profile.DateRevision) → Refus("profil périmé — veille requise")`. Constructeurs `NewDispatcherWithProfile`, méthode `WithClock(c)` (calque `EvaluateI3WithClock` M3). Helper `refusDecision` extrait pour ramener `Decide` sous funlen=100.
- `cmd/tau/calibrate.go` + extension `main.go` : sous-commande `tau calibrate --corpus PATH --output PATH --date-revision YYYY-MM-DD --version-monographie STRING --seed INT --created-at TIMESTAMP`. Le drapeau `--created-at` permet byte-identité (écrase `time.Now()` de `DefaultProfile()`).
- `cmd/generate-corpus/main.go` : drapeau `--annotate-with-dispatcher` (bool, défaut false) — enrichit chaque ligne avec `expected_regime` via `app.ToTauExchange` + `Dispatcher.Decide`. Préserve la byte-identité du baseline M4 quand inactif.
- `tests/calibration/golden-corpus.jsonl` : 200 lignes annotées (seed=42, balanced, sha256 `beb6c8d87911ef58d189c6f1c3d4adf9b71777e6dce328ed781e394614ac3a1b`). Distribution `expected_regime` : Deterministe 90 / Probabiliste 50 / Refus 60.
- `test/e2e/calibration_determinism_test.go` (build tag `e2e`) : `TestCalibrationDeterministic` (PRD §17 #10, deux runs → même sha256), `TestExpiredProfileRefuses` (PRD §15.1, anti-patron #3), `TestCalibrate_GoldenCorpus_FixedHash` (hash pinné `d753245b87933f97c6324f54df1572fab7cc68c52bc49baa1b891ab97abff6c7`).
- `internal/orchestration/dispatcher_expiry_test.go` : 4 tests étape 3 (expired/not-expired/zero-date/nil-profile).
- `Makefile` : cible `e2e-calibration` (tag `e2e`).
- `docs/algorithms/calibration.md` (214 l.) : domaines balayés, encodage milli-unités, tie-break, passthrough Weights, marshaller canonique, contrat byte-identique, 5 tests gardiens.
- `docs/algorithms/drift.md` (161 l.) : 5 critères, skip empty-fingerprint, seul `DriftDateExpired` → Refus en V1, fingerprints V1 documentés.
- `docs/archive/plans-m0-m6/2026-05-24-M5-calibration-drift.md` : sous-plan détaillé (1080 l., 10 tâches M5.0-M5.9).

### Notes

- **PRD §17 critère #10 atteint** : `TestCalibrationDeterministic` + hash pinné garantissent la byte-identité d'un profile pour `(corpus, seed, created_at, date_revision, version_monographie)` fixé.
- **Anti-patron #3 fermé** : la garde de péremption (étape 3 dispatcher) est désormais E2E. Test `TestExpiredProfileRefuses` couvre PRD §15.1.
- **Windows symlink** : fallback transparent (copie + sidecar) sans privilège Developer Mode. Logging slog.
- **Asymétrie date-révision drift vs dispatcher** : `CheckDrift` utilise `!now.Before(dateRevision)` (alerte précoce, today==dateRev → drift), dispatcher utilise `now().After(dateRevision)` (blocage strict, today==dateRev → pas de refus). Délibérée mais à documenter en M6 (OBS-1 reviewer).
- **Revue intégrée M5** : APPROVE. Deux observations info (asymétrie ci-dessus + commentaire `simulate()` à clarifier) reportées à M6.
- Race detector indisponible Windows local — CI Linux/macOS couvre.

## [0.0.5-alpha] — 2026-05-24

M4 : pont théorie ↔ empirie. Branche contingence active (PRD §18 risque #1 réalisé — AgentMeshKafka inexistant local/GitHub). DTO neutre + adaptateur fichier + convertisseur en couche `app/` + générateur de corpus synthétique reproductible byte-identique + harness empirique I4 + 3 rapports.

### Ajouté

- `internal/bridge/agentmeshkafka/adapter.go` : DTO neutre `AgentMeshExchange` (champs miroir de `tau.Exchange` sans import croisé), interface `Adapter` étroite (`Stream`, `Close` — ISP 2 méthodes).
- `internal/bridge/agentmeshkafka/file_adapter.go` : `FileAdapter` JSONL — lit ligne par ligne, résilient sur lignes malformées, `Close()` idempotent (`sync.Once`), respecte `ctx.Done()`.
- `internal/bridge/agentmeshkafka/testdata/{golden-3,golden-3-malformed}.jsonl` : corpus initial 3 lignes (nominal/sans attestation/contexte riche + variante malformée).
- `internal/bridge/agentmeshkafka/classifier.go` + tests : `I4Class` (6 classes : `i4_coherent_accepted`, `i4_incoherent_refused`, `i4_false_positive`, `i4_false_negative`, `other_refusal`, `unmodeled`), `EmpiricalDecision` neutre, `ClassifyI4`, `EmpiricalI4Summary` (sensitivity/specificity).
- `internal/bridge/agentmeshkafka/empirical_i4_test.go` (build tag `empirical`) : `TestEmpiricalI4Campaign` ingère 120 traces, écrit `testdata/empirical-i4-results.json`. Package externe `agentmeshkafka_test` autorise import croisé tau+app.
- `internal/app/agentmesh.go` : pivot unique bridge ↔ tau — `ToTauExchange` (pure totale) + `StreamAsTauExchanges` (wrapper streaming avec propagation d'erreurs et fermeture propre).
- `cmd/generate-corpus/` : CLI reproductible byte-identique (seed RNG explicite, sha256 frozen `a91d60cd9815d8183df57bfcf16bbe77d36360c4ed36e33fced9f12f70fd68ee` pinné dans `TestGenerateCorpus_FrozenHash_Seed42_120_Balanced`). 3 profils : `balanced`, `i4-heavy`, `refus-heavy`. Corpus checked-in : `synthetic-corpus-120-seed42-balanced.jsonl`.
- `test/e2e/agentmeshkafka_test.go` (build tag `integration`) : `TestE2E_AgentMeshKafka_FullPipeline` + variante `_NoTopicFilter` + `_MalformedCorpus`. 3 tests, full pipeline FileAdapter → StreamAsTauExchanges → Dispatcher.Decide.
- `internal/arch_test.go` : règle `bridge/agentmeshkafka` étendue (deny `tau`, `orchestration`, `app`) ; `TestBridgeNoTauImport` AST-walk sur tous `bridge/*` (exclut `_test.go`).
- `Makefile` : cibles `e2e` (tag integration) et `empirical-i4` (tag empirical).
- `docs/adr/0005-agentmeshkafka-dto.md` : décision DTO neutre + révision PRD §12.1 marquée pour M6.
- `docs/empirical/I4-report.md` (151 l.) : rapport campagne — 120 décisions classifiées, statut I4 **Hypothèse inchangée** (le générateur synthétique n'injecte pas les clés `Context` qui pilotent D-INVARIANT au-dessus du seuil ; la garde I4 n'a jamais été sollicitée).
- `docs/empirical/unmodeled.md` (108 l.) : 3 observations initiales (OBS-001 Context absent, OBS-002 frontière agrégée, OBS-003 AgentMeshKafka indisponible — risque #1 PRD §18 réalisé).
- `docs/empirical/I4-regime.md` (53 l.) : note d'audit — Régime B (contingence) sélectionné, conditions de bascule vers A documentées.
- `docs/archive/plans-m0-m6/2026-05-24-M4-agentmeshkafka-bridge.md` : sous-plan détaillé M4 (2134 l., 11 tâches).

### Notes

- **Découverte architecturale M4.1** : la signature PRD §12.1 `Stream(...) (<-chan tau.Exchange, ...)` viole `arch_test.go` (deny `bridge → tau`). Décision ADR-0005 : DTO neutre `AgentMeshExchange` dans `bridge/` + converter `ToTauExchange` dans `app/`. PRD §12.1 marqué pour révision M6.
- **Régime B activé** : `agbruneau/AgentMeshKafka` n'existe ni local ni sur GitHub (audit M4.0). Campagne empirique sur corpus synthétique reproductible. Bascule vers Régime A reportée à un éventuel M4-bis.
- **I4 inconclusif** : 84 `i4_coherent_accepted`, 36 `other_refusal`, 0 TP, 0 FN, 0 FP. Sensitivity = `-1` (dénominateur nul), Specificity = `1.0`. Cause racine documentée OBS-001 : le générateur ne peuple pas `Context.event_registry` ni `Context.idempotency_key_mode` → D-INVARIANT plafonne à 0.25 < `θ_inv = 0.50`.
- **Revue intégrée M4** : APPROVE_WITH_CONCERNS. 4 observations non-bloquantes (NB1-NB4) reportées à M4-bis ou M5 ; aucun bloquant code.
- Race detector indisponible sur Windows local (couvert par CI Linux/macOS). Le harness empirique court ~5s sur 120 traces.

## [0.0.4-alpha] — 2026-05-24

M3 : cinq invariants I1-I5 encodés et fuzzés, étape 8 dispatcher (`EvaluateInvariants → Trace.UnmodeledObservations`), gardes anti-patrons #1/#3/#4 par test. Smoke fuzz 5 s vert sur 5 cibles. Couverture `tau/invariants` 91.2 %.

### Ajouté

- `internal/tau/invariants/` (nouveau package) : `evaluator.go` (types `Status`, `Statuses`, `EvaluateInvariants`), `i1_conservation.go` + `Conserve`, `i2_irreductibility.go` + `Residu`/`Recablage`, `i3_authority_asymmetry.go` + `IsProfileExpired` + `EvaluateI3WithClock` + constante `I3PerimptionLimite` (2027-01-01 UTC), `i4_coherence.go` + `IsIncoherent`, `i5_composition.go` + `AngleMort`/`Pile`/`Aggregate`/`M`/`BoundsHold` (API d'agrégation calculée en V1 — la mention « V2 calcule » du PRD §6.1 est levée).
- `internal/tau/invariants/fuzz_targets_test.go` : 5 cibles fuzz `FuzzI1_Conservation`, `FuzzI2_Irreductibilite`, `FuzzI3_AsymetrieAutorite`, `FuzzI4_CoherenceContrainte`, `FuzzI5_CompositionConjonctive`. Smoke 5 s : I1 8.6M, I2 8.6M, I3 8.2M, I4 9.5M, I5 701K exécutions, 0 crash.
- `internal/tau/invariants/testdata/fuzz/FuzzI*/seed-*` : corpus seeds checked-in (3 seeds/cible) + 1 cas de régression FuzzI5 (`bf9c5ac437b95a58`).
- `internal/orchestration/dispatcher.go` : étape 8 du pseudo-algo PRD §10 — `invariants.EvaluateInvariants(x, dec)` ; `AnyViolated() → append(Summary())` dans `Trace.UnmodeledObservations`. Régime et Diagnostic intouchés.
- `internal/orchestration/dispatcher_invariants_test.go` : 3 tests (no violation, violation détectée, régime préservé).
- `internal/anti_patterns_test.go` : `TestNoPredictiveAPI` (parse AST des 4 packages, regex `^(Predict|Expected|Forecast)`), `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported`.
- `internal/arch_test.go` : règle `tau/invariants` étendue (3 deny : `tau/dimensions`, `orchestration`, `bridge`).
- `internal/tau/invariants/evaluator_test.go` : `TestStatus_String` couvre les 4 valeurs.
- `docs/theory/05-invariants.md` : renvoi croisé chap. III.8.5 — verbatims I1-I5, reformulations exécutables, conditions de réfutation, helpers Go, marqueurs épistémiques.
- `docs/empirical/fuzz-summary.md` : rapport empirique M3 — méthodologie, résultats par cible, découvertes, limites V1, prochaines étapes.
- `docs/archive/plans-m0-m6/2026-05-24-M3-invariants-fuzz.md` : sous-plan détaillé M3 (2047 l., 11 tâches bite-sized).

### Corrigé

- `internal/tau/invariants/i5_composition.go` : `BoundsHold` utilisait `len(layer)` (longueur slice) au lieu de la cardinalité ensembliste. Détecté par `FuzzI5_CompositionConjonctive` sur une pile `[["z","z"]]` (commit `7b4739c`). Calque le pattern FibGo « fuzz-discovered fix avant feat ».

### Notes

- Race detector indisponible sur Windows local (CGO/gcc) ; CI Linux/macOS couvre. Smoke fuzz 5 s sur dev, 30 s sur CI via `make fuzz`, 24 h hebdo via `make fuzz-long`.
- Étape 3 du dispatcher (péremption `today > date_revision`) reportée à M5 ; la propriété est gardée au niveau `Profile` en M3 (`TestI3_DateRevisionRespectee`).
- `EvaluateI5` retourne `Held` par construction V1 (pile d'angles morts non attachée à `Decision` avant V2) ; les bornes algébriques `max(|Aᵢ|) ≤ M(π) ≤ Σ|Aⱼ|` sont exercées directement par `FuzzI5` via `BoundsHold`.
- Revue intégrée M3 : APPROVE_WITH_CONCERNS. Observations résiduelles (info, non-bloquantes) reportées : couverture branche `EvaluateI2` zero-residue, vitesse FuzzI5 décodage byte-slice, couplage indirect `TestUnmodeledObservationsReported` ↔ `TestStep8_*`.
- Anti-patron #2 (hors frontière) toujours gardé par `TestFrontierCheck_*` (M0).

## [0.0.3-alpha] — 2026-05-23

M2 : trois dimensions (D-SENS, D-AUTORITÉ, D-INVARIANT) calculables, gardes ontologique I3 et cohérence I4 actives, pseudo-algo PRD §10 complet (étapes 1-7), Profile et AtomicThresholds. Couverture globale 92.2%.

### Ajouté

- `internal/tau/operator.go` : types `Principal`, `Capability`, `DiscoveryMode` (Static / DynamicMCP / DynamicA2A / DynamicAGNTCY) ; champs `Exchange.Initiator`, `Exchange.Target` ; `TraceThresholds` étendu (`AuthBlock`, `SensCoherence`, `InvCoherence`).
- `internal/tau/dimensions/` (nouveau package) : `score.go` (type `Score` partagé + `clamp01`), `dsens.go` + tests (4 sondes PRD §5.1), `dauthority.go` + tests (4 sondes PRD §5.2 + asymétrie ontologique), `dinvariant.go` + tests (4 sondes PRD §5.3 + contrainte I4).
- `internal/orchestration/dispatcher.go` : refonte M2 — `frontierFromExchange` heuristique (remplace placeholder M1) ; étape 2 garde ontologique D-AUTORITÉ (Refus I3) ; étape 4 scores des 3 dimensions ; étape 5 garde I4 ; étape 6 composite pondéré ; étape 7 hystérèse. Pseudo-algo PRD §10 étapes 1-7 complet.
- `internal/orchestration/thresholds.go` : étendu (`AuthBlock`, `SensCoherence`, `InvCoherence`) + `DefaultThresholds()`.
- `internal/orchestration/guards_test.go` : `TestRefusOntologiqueDAUTORITE`, `TestI4_IncoherenceDetectee`, `TestOntologicalGuardPassesWithAttestation`, `TestI4_CoherentCombinationAccepted`.
- `internal/calibration/profile.go` : `Profile`, `Weights`, `Thresholds` (PRD §11.3) + `DefaultProfile()` (DateRevision 2026-12-01, version monographie v2.4.3).
- `internal/calibration/thresholds_atomic.go` : `AtomicThresholds` calque FibGo `bigfft/fft.go` — `atomic.Int64` privés en milli-unités, accesseurs lecture, `SetTuning` coordonné, panic sentinel sur ordering violation, `Snapshot()` immuable.
- `internal/app/app.go` : utilise `orchestration.DefaultThresholds()` au lieu de hard-coded.
- `cmd/tau/main_test.go` : E2E adapté pour exchanges M2 (Initiator + Target inclus).
- `.golangci.yml` : termes français ajoutés au misspell ignore (combinaison, incohérente, détectée, frontière, verrou, ontologique).
- `docs/theory/04-dimensions.md` (170 l.) : renvoi croisé chap. III.8.4 — 3 dimensions, sondes, encodage Go, asymétrie ontologique (Searle 1995), contrainte I4.
- `docs/empirical/M2-sample-decisions.md` (397 l.) : 10 décisions tracées via `tau decide`, ventilation des scores par dimension, couvre tous les chemins (frontier refus, I3, I4, deterministe, probabiliste, hystérèse).
- `docs/archive/plans-m0-m6/2026-05-23-M2-dimensions-gardes.md` (2416 l.) : sous-plan détaillé M2.

### Modifié

- Tests dispatcher et invariants Decision adaptés pour le frontier heuristique M2 (les fixtures M1 fournissaient des Exchange sans Initiator/Target, qui maintenant tombent hors frontière par défaut).
- `TestDefaultLLMIsStub` adapté — vérification par déterminisme TauScore plutôt que comparaison directe au score Stub (le composite M2 ne se réduit plus à la sonde LLM seule).

### Notes

- Couverture par package : `tau/dimensions` 41.8 %, `orchestration` ≥ 80 %, `calibration` 100 %, `tau` 0.7 % (le code tau est mostly types/interface, sans logique testable directement — testée indirectement via dimensions et orchestration).
- M2.10 `docs/empirical/M2-sample-decisions.md` constitue le premier corpus de référence pour la calibration M5.
- Atomic accessors prêts pour la concurrence M5 (test `TestAtomicThresholds_ConcurrentReadsSafe` valide 100 goroutines).
- Anti-patrons à venir en M3 : `TestNoPredictiveAPI`, `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported`.

## [0.0.2-alpha] — 2026-05-23

M1 : dispatcher minimal + stub LLM. `tau decide` rend une `Decision` instrumentée. Cinq tests d'invariants `Decision`. Couverture globale 83.9 % (> 80 % gate).

### Ajouté

- `internal/tau/operator.go` : types `Trace` et `TraceThresholds` immuables ; champ `Decision.Trace` ; tags JSON snake_case sur tous les types publics.
- `internal/tau/frontier.go` : tags JSON snake_case sur `FrontierCheck`.
- `internal/orchestration/thresholds.go` : `Thresholds{Deterministe, Probabiliste}` avec invariant `Ordered()`.
- `internal/orchestration/dispatcher.go` : `Dispatcher` implémentant le sous-ensemble M1 du pseudo-algo PRD §10 (étapes 1, 6, 7) ; `NewDispatcher` panic sur ordering invariant ; clampage `DurationNs ≥ 1` pour résolution timer Windows.
- `internal/orchestration/dispatcher_test.go` : 3 tests (Deterministe, Probabiliste, hystérèse).
- `internal/orchestration/decision_invariants_test.go` : `TestDecisionAlwaysTraced`, `TestRefusImpliesDiagnostic` (3 subtests), `TestTraceImmutable`.
- `internal/bridge/llm/client.go` : interface étroite `Client` (PRD §12.2) — `Fingerprint()`, `Interpret(ctx, intent) (float64, error)`.
- `internal/bridge/llm/stub.go` : `Stub` déterministe via FNV-1a 32-bit hash ; fingerprint `stub:v0` ; score ∈ [0, 1) ; mappage checked-in.
- `internal/bridge/llm/stub_test.go` : fingerprint + déterminisme + bornes sur 4 cas (vide, 1 char, multi-mot, phrase).
- `internal/app/app.go` : `NewDispatcher()` factory ; sélection LLM via env `TAUGO_LLM_BACKEND` (défaut `Stub` ; `real` panic en M5+).
- `internal/app/app_test.go` : `TestDefaultLLMIsStub` (vérification comportementale TauScore vs Stub.Interpret).
- `internal/arch_test.go` : règle `internal/bridge` parent (skip-always) remplacée par règles concrètes sur `internal/bridge/llm` et `internal/bridge/agentmeshkafka`.
- `cmd/tau/main.go` : sous-commande `decide` (JSON stdin → JSON stdout, exit codes 0/2/3/4) ; version bumped à `0.0.2-alpha`.
- `cmd/tau/main_test.go` : tests E2E `TestEndToEnd_DecideDeterministe` (« creative generation » → 0.262) et `TestEndToEnd_DecideProbabiliste` (« hello world » → 0.807).
- `docs/archive/plans-m0-m6/2026-05-23-M1-dispatcher-stub-llm.md` : sous-plan détaillé M1 (1017 l., 9 tâches bite-sized).

### Corrigé

- Tags JSON manquants sur `tau.Exchange` : caché en M1.5, exposé par décodage silencieux de `intent_description` en chaîne vide (TauScore=0.261 = hash empty string). Fix `dff5565` aligne snake_case I/O.

### Spec et planification

- `PRDPlanning.md` reste référence ; sous-plan M1 commité séparément dans `docs/superpowers/plans/`.

### Notes

- Tous les sub-tasks M1 ont commit séparé. Couverture par package : `internal/tau` 100 %, `internal/orchestration` ≥ 90 %, `internal/bridge/llm` ≥ 80 %, `internal/app` ≥ 80 %.
- `tau decide` accepte stdin JSON ; sortie JSON snake_case ; régimes `0=Unknown, 1=Deterministe, 2=Probabiliste, 3=Refus` (marshaled comme nombre — M2+ peut ajouter `MarshalJSON`).
- Frontière de validité encore en mode placeholder (Inside=true toujours) ; les sondes réelles arrivent en M2.

## [0.0.1-alpha] — 2026-05-23

Premier tag. Squelette M0 du PRD : pas de logique métier, étanchéité architecturale gardée, CI verte sur 3 OS.

### Ajouté

- Squelette du module Go (`go.mod` `github.com/agbruneau/taugo`, `go 1.25.0`, `toolchain go1.26.3`).
- Licence Apache-2.0 (`LICENSE`).
- `.gitignore` (binaires, fichiers de test, IDE, artefacts agent runtime).
- `.golangci.yml` calque FibGo : 24 linters, complexité max 15/30, longueur fonction ≤ 100 LOC / 50 statements, `misspell` US + termes domaine FR-CA (Probabiliste, Deterministe, Refus, agentmeshkafka), `go: "1.25"` pour compatibilité golangci-lint v1.64.8.
- `Makefile` avec cibles `all`, `build`, `test`, `test-short`, `coverage`, `benchmark`, `lint`, `fuzz`, `fuzz-long`, `calibrate`, `build-reproducible`, `build-pgo`, `build-all`, `clean`. Timestamp gelé `1778889600` pour build reproductible (calque InteroperabiliteAgentique).
- Squelette `internal/` (10 packages avec `doc.go` descriptifs) : `tau`, `orchestration`, `calibration`, `bridge/{llm,agentmeshkafka}`, `app`, `config`, `errors`, `metrics`, `testutil`.
- `internal/tau/frontier.go` — `FrontierCheck` encodant les 4 conditions classiques de la frontière de validité de τ (chap. III.8.3.2) ; garde anti-patron #2 (« hors frontière »).
- `internal/tau/frontier_test.go` — 5 sous-tests TDD (all-true, all-false, 4× one-false), 100 % de couverture, `t.Parallel()` partout, init par champs nommés (anti-régression sur ajout de champ).
- `internal/tau/operator.go` — types `Regime`, `Exchange`, `Attestation`, `Decision` ; interface `Kernel` avec signature `Decide(ctx, Exchange) (Decision, error)`. Types `Trace`, `Principal`, `Capability` reportés à M1/M2.
- `internal/arch_test.go` — 4 règles d'étanchéité Clean Architecture (PRD §8.1) : `tau/* → orchestration/bridge/app` interdit ; `dimensions ↔ invariants` interdit (orthogonalité) ; `bridge → tau/*` direct interdit.
- `cmd/tau/main.go` — CLI minimale (`--help`, `--version`) ; points d'injection ldflags `main.version` et `main.buildTimestamp`.
- `.github/workflows/ci.yml` — matrice 3 OS (Linux/macOS/Windows) × Go 1.25.x : test (race CGO), lint (golangci-lint v1.64.8), build, cross-compile (linux/arm64, darwin/{amd64,arm64}), fuzz-smoke placeholder pour M3+.
- `.github/workflows/coverage.yml` — gate 80 % couverture globale ; per-package 90 % sur `tau/*` actif en M1+.
- `README.md` — point d'entrée FR-CA : quick start, doctrine (TauGo est / n'est pas), architecture résumée, liens vers PRD/CLAUDE/PRDPlanning.
- `docs/theory/03-operateur-tau.md` — premier renvoi croisé chap. III.8.3 : définition formelle `τ : t_fix(g) ≺ t_int ↦ t_fix(g) ≈ t_int`, table d'encodage TauGo, propriétés exploitables (bases I1, I2, orthogonalité), frontière de validité (4 conditions), anti-patrons cités.

### Spec et planification

- `PRD.md` V0.2 (refactorisé — 911 l., 20 sections, glossaire 16 termes).
- `CLAUDE.md` V0.3 (refactorisé + section Agent Teams + directive #11).
- `PRDPlanning.md` initial (1113 l., M0 détaillé bite-sized, M1-M6 résumés haut niveau).

### Notes

- Race detector requis sous CGO ; absent sur Windows sans gcc local — couvert par CI Linux/macOS.
- `golangci-lint` `run.go: "1.25"` requis car `golangci-lint v1.64.8` est construit avec Go 1.25 alors que `go.mod` carries `toolchain go1.26.3`.
- `.claude-flow/` (agent runtime artifacts) est ignoré par `.gitignore`.
- Drapeaux PRD §18 risques #1 (`AgentMeshKafka` not ready in M4) et #4 (scope creep) à surveiller.
