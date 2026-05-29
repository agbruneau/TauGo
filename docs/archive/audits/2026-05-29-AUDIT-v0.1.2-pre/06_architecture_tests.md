# 06 — Architecture & tests

> Audit Go multi-agents TauGo — axe « Architecture, étanchéité, tests & état CI ». HEAD `1948a7b`, v0.1.2-pre, 2026-05-29. Lecture seule, FR-CA. Sévérités : CRITIQUE 0 · MAJEUR 2 · MINEUR 3 · INFORMATIF 3.

**Conclusion (headline) :** Axe SA6 sain [confirme]. Les 7 regles d etancheite passent (TestArchitectureLayering, TestBridgeNoTauImport, TestArchNoConcreteLLMInDomain, TestNoPredictiveAPI verts), la suite complete est verte sur 12 packages (exit 0, sans -race faute de CGO), le gate per-package tau/* >= 90% est tenu (tau 100%, dimensions 98.7%, invariants 92.7%), go vet et golangci-lint v1.64.8 sont propres, go mod verify confirme l integrite. Aucun CRITIQUE. Deux MAJEUR de gouvernance documentaire : (1) le global ~92.1% revendique en PRD/CHANGELOG comme fait v0.1.2-pre Confirme ne se reproduit pas a la mesure (coverpkg=./... donne 89.2%) — statut epistemique survendu ; (2) divergences PRD/CLAUDE.md vs arborescence reelle (cmd/generate-golden inexistant, table couches §8.1 cite encore config/metrics, test/golden absent, golden corpus loge sous tests/ et non test/). Etat post-retrait-CI conforme ADR-0010 (.github absent, cibles CI Makefile retirees) avec mitigation runtime explicite de la veille I3 manuelle.

**Outils exécutés :**
- git rev-parse HEAD + git status --short : HEAD=1948a7b, arbre propre au depart ET apres tous les tests (aucune regeneration d artefact suivi)
- go version : go1.26.3 windows/amd64 [confirme]
- go test ./internal/ -run 'TestArch|TestBridge|TestNoPredictiveAPI|TestArchNoConcreteLLMInDomain' -v -count=1 : PASS (ok 0.570s) — 7 regles etancheite + anti-patron #1 + #6 verts
- go test -cover ./... -count=1 : per-package capture (tau 100%, dimensions 98.7%, invariants 92.7%, orchestration 89.1%, calibration 90.1%, app 92.5%, bridge/llm 100%, bridge/agentmeshkafka 92.2%, errors 100%, thresholds 0.0%)
- go test -coverpkg=./... -coverprofile=audit/cover.out ./... + go tool cover -func : total=89.2% (denominateur=tous packages)
- go test -tags=integration ./test/e2e/... -count=1 : ok 0.252s
- go test -tags=e2e ./test/e2e/... -run 'TestCalibration|TestCalibrate|TestExpiredProfileRefuses' -count=1 : ok 1.091s
- go vet ./... : exit 0, aucune sortie
- go mod verify : all modules verified
- golangci-lint version : v1.64.8 built with go1.26.3 [confirme pin]
- golangci-lint run ./internal/ ./internal/orchestration/ ./internal/calibration/ : exit 0, propre
- go test ./... -count=1 : FULL_EXIT=0, 12 packages ok
- Lecture internal/arch_test.go (8 regles archRules + 3 walks AST) et internal/anti_patterns_test.go (TestNoPredictiveAPI, TestI3_DateRevisionRespectee, TestUnmodeledObservationsReported)
- Survol des 10 ADR (statuts) + ADR-0010 head/perimetre/mitigation I3
- git ls-files / git check-ignore : ruvector.db et audit/cover.out ignores ; internal/config et internal/metrics non suivis (dirs vides)
- grep t.Parallel : 48/48 fichiers de test l utilisent (seul fuzz_targets_test.go s en abstient, correct)
- which make : absent (seul le fichier Makefile est present)

**Outils indisponibles / repli :**
- -race / go test -race : indisponible (CGO_ENABLED=0, aucun compilateur C sous Windows). Repli : suite executee sans -race. Detection data race/deadlock NON verifiee sur ce poste [a verifier sur Linux/macOS avec CGO]
- make : binaire absent du PATH. Repli : invocation directe go / golangci-lint (conforme consigne). Le fichier Makefile existe mais ses cibles CI-only ont ete retirees (ADR-0010, verifie par grep)

---

## Conclusion (pyramide inversee)

L axe SA6 est **sain et conforme** [confirme]. Aucun constat CRITIQUE : les 7 regles d etancheite passent, la suite complete est verte (12 packages, exit 0), le gate per-package `internal/tau/*` >= 90% est tenu, l outillage statique (`go vet`, `golangci-lint v1.64.8`, `go mod verify`) est propre, et l arbre git reste intact apres tous les tests (aucune regeneration d artefact suivi). Les deux MAJEUR sont d ordre **gouvernance documentaire** — pas de defaut de code : (1) un chiffre de couverture globale survendu et (2) des renvois PRD/CLAUDE.md desynchronises de l arborescence reelle. L etat post-retrait-CI (ADR-0010) est coherent, avec mitigation runtime explicite de la veille I3 devenue manuelle.

Reserve d execution : `-race` n a pas pu etre execute (CGO absent sous Windows) — la detection data race/deadlock reste **[a verifier]** sur un poste Linux/macOS avec CGO.

## 1. Etancheite (7 regles)

**[A6-ETANCHE] INFORMATIF — Les 7 regles d etancheite passent** *(confirme)*
- Fichier : `internal/arch_test.go:19-62` (8 entrees `archRules`), `internal/anti_patterns_test.go`
- Preuve (verbatim) : `go test ./internal/ -run "TestArch|TestBridge|TestNoPredictiveAPI|TestArchNoConcreteLLMInDomain" -v -count=1` →
  ```
  --- PASS: TestBridgeNoTauImport (0.22s)
  --- PASS: TestArchNoConcreteLLMInDomain (0.22s)
  --- PASS: TestNoPredictiveAPI (0.00s)
  --- PASS: TestArchitectureLayering (0.00s)
  ok  github.com/agbruneau/taugo/internal  0.570s
  ```
- Enumeration des regles gardees :
  - `tau → {orchestration, bridge, app}` interdit (`arch_test.go:20-24`)
  - `dimensions → invariants` interdit (`:25-27`) et `invariants → {dimensions, orchestration, bridge}` interdit (`:28-32`) — orthogonalite I1-I5 vs 3 dimensions encodee
  - `bridge/llm → tau` interdit (`:33-35`) ; `bridge/agentmeshkafka → {tau, orchestration, app}` interdit (`:36-40`)
  - **V-A2** `calibration → {tau, orchestration, bridge}` interdit (`:44-48`) — confirme present
  - **ADR-0006** `thresholds → aucun package taugo` (`:51-61`) — etancheite descendante
  - **P0-01 / anti-patron #6** `TestArchNoConcreteLLMInDomain` (`:140-202`) — walk AST sur 12 substrings LLM concrets dans `tau/*` + `orchestration/` : vert
  - anti-patron #1 `TestNoPredictiveAPI` (`anti_patterns_test.go:34-62`) sur 4 packages : vert
- Impact : etancheite Clean Architecture preservee, anti-patrons #1 et #6 gardes.
- Recommandation : RAS. Voir A6-04 pour une asymetrie mineure de couverture.

## 2. Couverture

**[A6-01] MAJEUR — Couverture globale 92.1% revendiquee Confirme mais non reproductible** *(confirme)*
- Fichier : `PRD.md:831`, `PRD.md:848`, `CHANGELOG.md:99`
- Preuve (verbatim) :
  - `PRD.md:848` : `État v0.1.2-pre : 10/10 atteints (Confirmé : ... couverture globale 92.1 %, build reproductible)`
  - Mesure reelle `go test -coverpkg=./... -coverprofile=audit/cover.out ./... && go tool cover -func` → `total: (statements) 89.2%`
- Impact : l affirmation est datee et porte le marqueur **Confirme** ; or la methode `coverpkg=./...` (denominateur = tous les packages, incluant `thresholds` 0%, `examples/quickstart` 0%) donne 89.2%. Le 92.1% provient d une moyenne ponderee per-package heritee de v0.1.1 (commit `2cf560c`) — methode differente, pas une fabrication, mais le chiffre est **survendu** comme un fait v0.1.2-pre verifie. Viole l esprit des conventions editoriales (marqueur d incertitude / zero survente).
- Recommandation : soit reetiqueter en `[Probable, methode per-package ponderee — coverpkg=./... global = 89.2%]`, soit recalculer et figer une seule methode de reference. Compromis : la moyenne per-package flatte le chiffre ; alternative honnete = publier 89.2% (coverpkg) ; retournement = si l on exclut explicitement les packages a 0% du denominateur, documenter ce choix.

**[A6-COV-TAU] INFORMATIF — Gate per-package tau/* >= 90% tenu** *(confirme)*
- Preuve (verbatim) : `go test -cover ./internal/tau/... -count=1` →
  ```
  internal/tau              100.0%
  internal/tau/dimensions    98.7%
  internal/tau/invariants    92.7%
  ```
- Impact : objectif CLAUDE.md (≥ 90% sur `tau/*`) atteint. Trous residuels mineurs : `dimensions` 98.7%, `invariants` 92.7% — non bloquants.
- Recommandation : RAS.

**[A6-COV-LEAF] MINEUR — thresholds et examples a 0% de couverture propre** *(confirme)*
- Preuve : `internal/thresholds 0.0%`, `examples/quickstart 0.0%` (sortie `go test -cover ./...`). `thresholds` est exerce indirectement (type valeur transverse), `examples/quickstart` est un exemple runnable non teste.
- Impact : tire le global `coverpkg` vers le bas (cf. A6-01). Non bloquant.
- Recommandation : ajouter un test unitaire minimal a `thresholds` (`Ordered`, `Clamp`) ou l exclure explicitement du calcul de couverture documente.

> Note BOM : `-coverpkg=./...` a fonctionne sans erreur — aucun BOM UTF-8 en milieu de payload detecte sur ce run. [confirme]

## 3. Tests tagues

**[A6-TAGS] INFORMATIF — Tests integration et e2e calibration verts** *(confirme)*
- Preuve (verbatim) :
  - `go test -tags=integration ./test/e2e/... -count=1` → `ok github.com/agbruneau/taugo/test/e2e 0.252s`
  - `go test -tags=e2e ./test/e2e/... -run "TestCalibration|TestCalibrate|TestExpiredProfileRefuses" -count=1` → `ok ... 1.091s`
- `git status --short` apres ces runs : vide — aucune regeneration de `testdata/empirical-i4-results.json` ni de sortie calibration. Aucune restauration necessaire. [confirme]
- Impact : chemins E2E (calibration determinisme, agentmeshkafka, trace) operationnels via `go test` direct post-retrait-Make.

## 4. Analyse (layout, ADR, go.mod)

**[A6-02] MAJEUR — Divergences PRD/CLAUDE.md vs arborescence reelle** *(confirme)*
- Fichier : `PRD.md:331-353` (arbre §8), `PRD.md:360-363` (table couches §8.1), `CLAUDE.md` §Architecture
- Preuves (verbatim) :
  - `PRD.md:334` : `generate-golden/  # oracle indépendant (V1.1)` — or `ls cmd/` → `generate-corpus/` (pas de `generate-golden/`)
  - `PRD.md:360-363` (table 8.1) liste encore `config` (couche Presentation) et `metrics` (couches 2-4) comme imports valides, alors que `PRD.md:346` dit `config et metrics supprimés v0.1.1` — **contradiction interne au meme document**, et `ls internal/config internal/metrics` → repertoires vides
  - `PRD.md:352` et `CLAUDE.md` : `test/{e2e, golden}/` — or seul `test/e2e/` existe ; le golden corpus reel est `tests/calibration/golden-corpus.jsonl` (repertoire `tests/`, distinct de `test/`)
- Impact : la spec canonique et le CLAUDE.md decrivent une arborescence qui ne correspond pas au depot. Risque de confusion pour un nouvel arrivant et de fausse confiance dans les gardes. Aligne A6-02 sur la directive CLAUDE.md « document vivant : deviation = MAJ PRD ET CLAUDE.md AVANT le code ».
- Recommandation : (1) corriger `cmd/generate-golden` → `generate-corpus` dans PRD §8 ; (2) purger les lignes `config`/`metrics` de la table §8.1 (deja declares supprimes l.346) ; (3) reconcilier `test/golden` vs `tests/calibration` (choisir un emplacement canonique, MAJ doc). Compromis : edition doc pure, zero risque code ; alternative = creer `test/golden/` reel ; retournement = si V1.1 reintroduit l oracle `generate-golden`.

**[A6-ADR] INFORMATIF — ADR 0001-0010 tous Acceptes, coherents avec le code** *(confirme)*
- Preuve : `grep Statut docs/adr/*.md` → les 10 ADR portent `Statut : Accepté` (aucun Superseded/Deprecated orphelin). Verifications croisees code :
  - ADR-0006 (thresholds transverse) ↔ regle `arch_test.go:51-61` presente
  - ADR-0008 (Trace ventilee) ↔ scores dimensionnels lus par dispatcher (confirme axe non-SA6)
  - ADR-0009 (erreurs typees) ↔ `internal/errors/` 100% couvert, sentinels `ErrPeremptionProfile` etc. presents
  - ADR-0010 (retrait CI/CD) ↔ `.github/` absent (`ls -la .github` → No such file or directory), cibles `make fuzz-long|e2e-calibration|build-reproducible|empirical-i4` absentes du `Makefile` (grep vide)
- Impact : tracabilite ADR ↔ code saine.
- Recommandation : RAS, hormis A6-02 (PRD §8 a aligner).

**[A6-MOD] INFORMATIF — go.mod/go.sum sains** *(confirme)*
- Preuve (verbatim) : `go mod verify` → `all modules verified` ; `go vet ./...` → exit 0 sans sortie ; `golangci-lint v1.64.8 ... built with go1.26.3` puis `run` exit 0.
- Impact : integrite des dependances et conformite lint (24 linters, pin exact) confirmees.

**[A6-04] MINEUR — Couverture d etancheite asymetrique (deny-only, pas de regle 'from' pour app/errors/testutil)** *(confirme)*
- Fichier : `internal/arch_test.go:19-62`
- Preuve : les `archRules` definissent des regles `from` pour `tau`, `tau/dimensions`, `tau/invariants`, `bridge/llm`, `bridge/agentmeshkafka`, `calibration`, `thresholds` — mais **aucune** regle `from` pour `internal/app`, `internal/errors`, `internal/testutil`. Ces couches ne sont gardees que passivement (en tant que cible `deny` d autres regles).
- Impact : une future fuite (ex. `errors` ou `testutil` important `orchestration` ou `bridge`) ne serait pas detectee par `TestArchitectureLayering`. Risque faible aujourd hui (`errors` 100% couvert, periph. stable).
- Recommandation : ajouter des regles `from` defensives pour `app` (ne doit pas dependre de `bridge/*` directement — passe par interfaces, PRD §8.1 couche 1) et `errors`/`testutil` (feuilles, ne doivent rien importer de taugo hormis types valeur). Surgical, additif.

**[A6-03] MINEUR — internal/config et internal/metrics : repertoires vides non suivis, references residuelles** *(confirme)*
- Preuve : `ls -la internal/config internal/metrics` → uniquement `./` et `../` (aucun fichier) ; `git ls-files internal/config/ internal/metrics/` → vide (non suivis) ; seule reference code = `arch_test.go:58-59` (deny-list `thresholds`) et la table `PRD.md:360-363`.
- Impact : repertoires fantomes herites de la suppression v0.1.1 (cf. ADR / PRD l.346). Bruit d arborescence, sans effet runtime.
- Recommandation : supprimer les deux repertoires vides (action proprietaire, hors lecture-seule de cet audit) et retirer `metrics`/`config` de la deny-list `thresholds` (`arch_test.go:58-59`) puisque les packages n existent plus. A defaut, laisser tel quel est inoffensif.

## 5. Etat CI (ADR-0010) et qualite des tests

**[A6-07] INFORMATIF — Etat post-retrait-CI conforme** *(confirme)*
- Preuve : `.github/` absent ; `Makefile` present comme fichier mais `make` binaire absent du PATH (`which make` → no make) ; cibles CI-only retirees (grep `fuzz-long|e2e-calibration|build-reproducible|empirical-i4|^ci:` → vide). ADR-0010 documente exhaustivement le perimetre supprime et les remplacements `go test` directs.
- Impact : projet pure-local operationnel ; tests tagues accessibles via `go test -tags=...` (verifie §3).

**[A6-06] INFORMATIF — Veille I3 devenue manuelle, mitigee par garde runtime** *(confirme)*
- Fichier : `docs/adr/0010-retrait-ci-cd-pure-local.md:69`, `internal/orchestration/dispatcher.go:161`, `internal/tau/invariants/i3_authority_asymmetry.go:14`
- Preuve (verbatim) :
  - ADR-0010 l.69 : `Plus d'alerte 30 jours avant péremption I3 ... Bascule en cron externe ou check manuel — le risque PRD §18 #9 est désormais mitigé par : (a) garde runtime TestExpiredProfileRefuses ... (b) app.NewDispatcher() qui charge un profil par défaut`
  - `dispatcher.go:161` : `if d.profile != nil && !d.profile.DateRevision.IsZero() && d.now().After(d.profile.DateRevision) {` → `return refusDecision(x, tau.DiagPeremptionProfile, ...)`
  - `i3_authority_asymmetry.go:14` : `return time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)` (I3PerimptionLimite)
- Impact : la disparition de l alerte automatisee 30 j est un risque operationnel REEL (oubli de revision avant `date_revision`) mais **borne** : toute decision sur profil perime declenche un Refus runtime (`DiagPeremptionProfile`), et `TestI3_DateRevisionRespectee` garde que `DefaultProfile().DateRevision` reste dans (now, 2027-01-01]. Severite degradee a INFORMATIF par cette mitigation.
- Recommandation (option V0.2+) : reintroduire une **CI minimale** non bloquante — un unique workflow `go test ./... && golangci-lint run` + un cron mensuel evaluant `DateRevision - now < 30j` (echec = alerte). Compromis : reouvre une surface de maintenance que l ADR-0010 a voulu fermer ; alternative = cron local / tache planifiee Windows documentee dans le README ; retournement = si une peremption silencieuse survient en pratique malgre la garde runtime.

**[A6-PARALLEL] INFORMATIF — Qualite table-driven et t.Parallel a 100%** *(confirme)*
- Preuve : `grep -rl "func Test" --include=*_test.go` = 48 fichiers ; `grep -rl "t.Parallel()"` = 48 fichiers. Seul `internal/tau/invariants/fuzz_targets_test.go` n appelle pas `t.Parallel()` — correct (les cibles fuzz ne se parallelisent pas). Les tests d etancheite et anti-patrons sont table-driven (`archRules`, `gardedPackages`, sous-tests `t.Run` parallelises, cf. `arch_test.go:64-85`, `anti_patterns_test.go:34-62`).
- Impact : adoption `t.Parallel()` cible 100% atteinte (CLAUDE.md) ; structure table-driven idiomatique.
- Recommandation : RAS.

**[A6-08] INFORMATIF — Artefact ruvector.db modifie en session mais git-ignore** *(confirme)*
- Preuve : `ls -la ruvector.db` → 1 589 248 octets, modifie `May 29 06:57` (pendant la session, probablement par l outillage ruflo-swarm) ; `git check-ignore ruvector.db` → match (regle `*.db` du `.gitignore`). `git status --short` final : vide.
- Impact : aucun risque de contamination du depot — l artefact n est pas suivi. Mentionne pour tracabilite (le fichier mute hors du perimetre code TauGo).
- Recommandation : RAS pour le code ; si non desire dans la racine, le deplacer hors arbre projet (action proprietaire).

**[A6-05] MINEUR — Detection data race non verifiable (CGO absent)** *(a verifier)*
- Preuve : `go version` → `windows/amd64` ; CGO_ENABLED=0, aucun compilateur C. `-race` non execute sur ce poste (consigne d audit + CLAUDE.md « -race exige CGO Linux/macOS »).
- Impact : la suite passe sans `-race` (exit 0), mais l absence de data race / deadlock dans le dispatcher concurrent et les seuils `atomic.Int64` de `calibration` n est **pas** confirmee sur cet environnement.
- Recommandation : executer `go test -race ./...` sur un poste Linux/macOS avec CGO avant tout tag de release, et documenter ce gate manuel dans le README (la CI le couvrait auparavant).
