# 05 — Idiomatique Go

> Audit Go multi-agents TauGo — axe « Idiomatique Go & qualité de code ». HEAD `1948a7b`, v0.1.2-pre, 2026-05-29. Lecture seule, FR-CA. Sévérités : CRITIQUE 0 · MAJEUR 1 · MINEUR 2 · INFORMATIF 3.

**Conclusion (headline) :** L hygiène statique du kernel est excellente [confirmé] : gofmt, go vet, staticcheck v0.7.0 et golangci-lint v1.64.8 (24 linters, config v1 chargée) passent tous sans la moindre alerte sur l ensemble du dépôt. Aucun constat CRITIQUE. Les seuls signaux sont (1) une incohérence errors.Is sur *RefusError — qui n a pas de méthode Unwrap/Is, rendant les 4 sentinels Diagnostic non matchables (MAJEUR, statut épistémique survendu par le godoc « utilisables via errors.Is ») et le constat que le package internal/errors n est quasiment pas adopté en production ; (2) 9 findings gosec (3 HIGH, 6 MEDIUM) tous confinés au tooling/I-O hors kernel tau/*, dont les G304 sont déjà documentés comme acceptés (modèle CLI mono-utilisateur, revue M5).

**Outils exécutés :**
- go version -> go1.26.3 windows/amd64 [confirmé]
- golangci-lint version -> v1.64.8 (pin exact respecte) [confirmé]
- staticcheck -version -> 2026.1 (v0.7.0) [confirmé]
- gosec --version -> dev (binaire sans tag de version) [a verifier]
- gofmt -l . -> sortie vide, 0 fichier non formate [confirmé]
- go vet ./... -> sortie vide, exit 0 [confirmé]
- staticcheck ./... -> aucune alerte, exit 0 [confirmé]
- golangci-lint run ./... -> aucune alerte, exit 0 [confirmé]
- golangci-lint config path -> .golangci.yml ; run -v -> 'Active 24 linters: [bodyclose copyloopvar errcheck funlen gochecknoglobals gocognit gocritic gocyclo gofmt gosec gosimple govet ineffassign misspell nakedret noctx nolintlint prealloc revive staticcheck unconvert unparam unused whitespace]' [confirmé]
- gosec -fmt=text ./... -> 9 issues (G115 x2 HIGH, G404 HIGH, G304 x5 MEDIUM, G301 MEDIUM), Files 46 / Lines 3736 / Nosec 0 [confirmé]
- CGO_ENABLED=0 go build ./... -> exit 0 [confirmé]
- git status --short (avant/apres) -> vide, arbre propre, aucune ecriture [confirmé]

**Outils indisponibles / repli :**
- go test -race : non execute volontairement (CGO_ENABLED=0, aucun compilateur C sous Windows) — repli : CGO_ENABLED=0 go build ./... pour validation de compilation [a verifier en CI Linux/macOS]
- gosec version exacte : le binaire rapporte 'dev' sans tag/date de build — repli : findings analyses au cas par cas, regles G* standard [a verifier]

---

## Synthèse

L axe SA5 est globalement **vert**. Aucun constat CRITIQUE. Le code du kernel respecte les idiomes Go et les conventions du dépôt : **gofmt -l** ne liste aucun fichier, **go vet**, **staticcheck v0.7.0** et **golangci-lint v1.64.8** (24 linters, schéma config v1, fichier `.golangci.yml` chargé) terminent tous en exit 0 sans alerte sur `./...`. La compilation `CGO_ENABLED=0 go build ./...` réussit. Le seul signal de gravité MAJEUR est une incohérence `errors.Is` doublée d un statut épistémique survendu dans le godoc. Les 9 findings gosec sont tous hors du cœur `tau/*` et majoritairement déjà documentés comme acceptés.

Note d exécution : `-race` n a pas été lancé (CGO désactivé, pas de compilateur C sous Windows) — à valider en environnement Linux/macOS [à vérifier]. Le binaire gosec rapporte la version « dev » sans tag [à vérifier].

## Constats

**[Q5-01] MAJEUR — *RefusError sans Unwrap/Is : sentinels Diagnostic non-matchables ; godoc survendu** *(statut épistémique survendu)* — [confirmé]
- Fichier : `internal/errors/errors.go:44-55` (déclaration `RefusError`, méthode `Error` uniquement) ; godoc `internal/errors/errors.go:11-13`.
- Preuve (verbatim) : recherche `func \(e \*RefusError\) (Is|Unwrap)` → « No matches found ». Le godoc affirme : « Les sentinels exportés (ErrFrontiereFranchie, etc.) correspondent exactement aux chaînes Diagnostic utilisées dans le Dispatcher. Ils sont utilisables via errors.Is / errors.As. » Or `RefusError` n a ni `Unwrap()` ni `Is()` ; son champ `Diagnostic` est une simple `string`, jamais reliée au sentinel. Le test `TestSentinels_IdentifieesViaErrorsIs` (`errors_test.go:92-114`) ne teste les sentinels qu enveloppés dans un `DispatchError{Cause: sentinel}` — jamais via `RefusError`. Aucun test ne vérifie `errors.Is(unRefusError, ErrFrontiereFranchie)`.
- Impact : un appelant qui ferait `errors.Is(err, errors.ErrFrontiereFranchie)` sur un `*RefusError` obtiendrait toujours `false`, contrairement à ce que promet le godoc et l ADR-0009. Le lien « sentinel ↔ Diagnostic » est purement textuel, non programmatique. Risque de faux sentiment de robustesse côté intégrateur (M5+).
- Recommandation : soit ajouter à `RefusError` une méthode `Is(target error) bool` qui compare `e.Diagnostic` à la chaîne du sentinel (et un `Unwrap` retournant le sentinel approprié), soit corriger le godoc pour ne plus affirmer la matchabilité `errors.Is` du `RefusError`. Trancher par ADR car ADR-0009 est concerné. Compromis principal : ajouter `Is` (couplage chaîne↔sentinel) vs. retirer la promesse (plus simple, conforme au fait que le refus est une *Decision*, pas une *error*). Alternative crédible : laisser tel quel et documenter explicitement que les sentinels ne servent qu au `DispatchError`. Condition de retournement : si M5+ expose une API d erreur publique nécessitant le matching sur refus.

**[Q5-02] INFORMATIF — internal/errors quasi non adopté en production** — [confirmé]
- Fichier : seule occurrence production = `internal/app/app.go:39,44` (`*DispatchError` pour provider LLM inconnu). `internal/orchestration/*` n importe pas `internal/errors`.
- Preuve (verbatim) : `grep internal/errors internal/orchestration/` → « No matches found ». Le dispatcher retourne le refus comme `Decision{Regime: tau.Refus}` (`dispatcher.go:96,112`), pas comme `*RefusError`. Donc `RefusError` + les 4 sentinels (`ErrFrontiereFranchie`, `ErrPeremptionProfile`, `ErrIncoherenceI4`, `ErrVerrouOntologique`) ne sont câblés nulle part en production.
- Impact : cohérent avec la doctrine (« Refus n est pas un échec : c est une décision pleine ») — le refus est une valeur, pas une erreur. Le package `internal/errors` est donc surtout du scaffolding « adoption progressive » (cf. CLAUDE.md, ADR-0009). Pas un défaut en soi, mais explique pourquoi Q5-01 n a pas d impact runtime actuel.
- Recommandation : laisser tel quel ; ce constat sert de contexte à Q5-01. À réévaluer quand l adoption progressive avancera.

**[Q5-03] MINEUR — gosec G304 x5 + G301 sur les I/O calibration : findings connus et acceptés** — [confirmé]
- Fichiers : `internal/calibration/store.go:115,129` (G304), `internal/calibration/store.go:48` (G301, MkdirAll 0o755), `internal/calibration/drift.go:133` (G304), `internal/calibration/calibrate.go:77` (G304), `cmd/tau/calibrate.go:106` (G304).
- Preuve (verbatim) : gosec → « G304 (CWE-22): Potential file inclusion via variable (Confidence: HIGH, Severity: MEDIUM) » ×5 et « G301 (CWE-276): Expect directory permissions to be 0750 or less ». `.golangci.yml` (bloc issues) : « TauGo V1 keeps G304 active to surface every file-open; decisions are made case-by-case in M5 when the full threat model is reviewed. » Les templates d exclusion G304 y sont laissés *commentés* volontairement.
- Impact : chemins issus de flags/env sous modèle de menace CLI mono-utilisateur ; pas d exposition réseau. Tous hors `internal/tau/*` (cœur). Risque réel faible en V1.
- Recommandation : conserver l état (findings volontairement non supprimés pour visibilité). Trancher en M5 via le scope `os.Root` (Go ≥ 1.24) suggéré par gosec, ou activer les exclusions documentées. Pas d action immédiate.

**[Q5-04] MINEUR — G115 x2 (HIGH) sur generator.go:75 non couverts par le //nolint de la ligne 76** — [confirmé]
- Fichier : `cmd/generate-corpus/generator.go:75-76`.
- Preuve (verbatim) : gosec → « generator.go:75 - G115 (CWE-190): integer overflow conversion int64 -> uint64 » (deux fois) ; le code : `src := rand.NewPCG(uint64(seed), uint64(seed)^0xdeadbeef_cafebabe)` (ligne 75) puis `return &Generator{rng: rand.New(src)} //nolint:gosec ...` (ligne 76). Le `//nolint:gosec` couvre G404 (ligne 76) mais **pas** les G115 de la ligne 75.
- Impact : nul à l exécution — `seed int64 → uint64` est une conversion bit-à-bit intentionnelle pour seeder PCG de façon déterministe ; aucun comportement incorrect. golangci-lint (qui embarque gosec parmi ses 24 linters) ne remonte pas ces G115, donc la config golangci les neutralise déjà au niveau pertinent ; ne se manifeste qu en invocation gosec autonome. Outil = dev tool de génération de corpus, hors kernel.
- Recommandation (optionnelle) : si l on veut un run gosec autonome propre, déplacer/ajouter `//nolint:gosec // G115: conversion bit-à-bit volontaire pour seeder PCG` sur la ligne 75. Sinon, no-op : golangci-lint reste vert.

**[Q5-05] INFORMATIF — Outillage statique 100 % vert** — [confirmé]
- Preuve (verbatim) : `gofmt -l .` → vide ; `go vet ./...` → vide, exit 0 ; `staticcheck ./...` → exit 0 sans sortie ; `golangci-lint run ./...` → exit 0 sans sortie ; verbose → « Active 24 linters: [bodyclose copyloopvar errcheck funlen gochecknoglobals gocognit gocritic gocyclo gofmt gosec gosimple govet ineffassign misspell nakedret noctx nolintlint prealloc revive staticcheck unconvert unparam unused whitespace] ».
- Impact : aucun problème de nommage, errcheck, ineffassign, unused, complexité (gocyclo 15 / gocognit 30 / funlen 100 LOC-50 stmt, seuils confirmés `.golangci.yml:63-71`) signalé. Hygiène de code de premier ordre.
- Recommandation : aucune. Maintenir le pin v1.64.8 et la config v1.

**[Q5-06] INFORMATIF — Conformité conventions : panics, godoc, package-doc** — [confirmé]
- Preuve (verbatim) : `grep panic\(` → 4 sites uniquement : `app/app.go:26` (switch provider exhaustif, erreur de programmation), `orchestration/dispatcher.go:46` et `calibration/thresholds_atomic.go:26,84` — tous gardés par un invariant d ordre des seuils (« thresholds out of order », « AtomicThresholds ordering violated »). Conforme à la règle « Pas de panic sauf invariant interne cassé ». Chaque package public porte un commentaire `// Package ...` : via `doc.go` (9 fichiers) ou inline (`errors.go:1`, `thresholds/thresholds.go:1`, `testutil/builders.go:1`) — conforme à « doc.go peut être fusionné dans le fichier principal ».
- Impact : conventions de code (calque FibGo) respectées sur le périmètre audité.
- Recommandation : aucune.

## Limites de l audit
- `-race` non exécuté (CGO off, pas de gcc sous Windows) ; détection data race/deadlock hors périmètre de ce run — à couvrir en CI Linux/macOS [à vérifier].
- Version gosec « dev » (sans tag) ; les règles G* sont standard mais la version exacte n est pas attestable [à vérifier].
- Audit en lecture seule ; `git status --short` vide avant et après — aucune écriture, aucun artefact suivi régénéré.
