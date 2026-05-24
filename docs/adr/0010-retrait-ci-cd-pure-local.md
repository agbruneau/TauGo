# ADR-0010 — Retrait complet de l'outillage CI/CD : projet pure-local

**Statut** : Accepté
**Date** : 2026-05-24
**Version** : v0.1.2-pre
**Décideur** : André-Guy Bruneau

---

## Contexte

À l'issue du refactor v0.1.1-pre (commit `2cf560c`), le projet TauGo disposait d'un outillage CI/CD complet :

- **GitHub Actions** — deux workflows (`.github/workflows/ci.yml` et `coverage.yml`) couvrant matrice 3 OS (ubuntu-latest, windows-latest, macos-latest), `go test -race` via CGO sur Linux/macOS, lint `golangci-lint v1.64.8`, build reproductible, fuzz smoke 30 s sur I1-I5, gate per-package `internal/tau/*` ≥ 90 % et gate global ≥ 80 % (activé v0.1.1 T-012).
- **Cibles `Makefile` orientées CI** — `fuzz-long` (24 h nocturne), `e2e` (tag `integration`), `e2e-calibration` (tag `e2e`), `empirical-i4` (tag `empirical`), `build-reproducible` (timestamp gelé `1778889600`).
- **Références documentaires** — `README.md` (badges CI/coverage, sections gates, alerte 30 jours avant péremption I3), `CLAUDE.md` §10 « CI active », `PRD.md` §13/§15.3/§16/§17/§18 (critères de succès, risques mitigés par CI), `CHANGELOG.md`.

Le propriétaire a décidé de **retirer l'intégralité de cet outillage** pour positionner TauGo comme un projet *pure-local* — validation entièrement déléguée au poste développeur, sans gate automatisé bloquant.

---

## Décision

**Supprimer tout l'outillage CI/CD du repo** et aligner la documentation en conséquence.

### Périmètre supprimé

| Élément | Action |
|---|---|
| `.github/workflows/ci.yml` (143 lignes) | Suppression |
| `.github/workflows/coverage.yml` (86 lignes) | Suppression |
| Dossier `.github/` | Suppression complète |
| Cible `make fuzz-long` | Suppression (remplacement local : `go test -fuzz=. -fuzztime=24h ./internal/tau/invariants/`) |
| Cible `make e2e` | Suppression (remplacement : `go test -tags=integration ./test/e2e/...`) |
| Cible `make e2e-calibration` | Suppression (remplacement : `go test -tags=e2e ./test/e2e/... -run="TestCalibration\|..."`) |
| Cible `make empirical-i4` | Suppression (remplacement : `go test -tags=empirical ./internal/bridge/agentmeshkafka/... -run=^TestEmpiricalI4Campaign$$`) |
| Cible `make build-reproducible` | Suppression (timestamp gelé `1778889600` — était nécessaire pour byte-identité CI) |
| Badges `README.md` `[![CI]]` `[![Coverage]]` | Suppression |
| Section `README.md` § Stack technique « CI matrix » | Réécrite en « Validation locale uniquement » |
| Directive `CLAUDE.md` §10 « CI active » | Réécrite « Validation locale obligatoire » |
| `PRD.md` §15.3 « Gates CI » | Renommée « Gates locaux » — devient objectifs vérifiables, plus de blocage de merge automatisé |
| `PRD.md` §17 critères #4/#5/#8 | Annotés « vérification locale ; CI retirée v0.1.2 » |
| `PRD.md` §18 risque 9 (alerte 30 j péremption I3) | Bascule en cron externe / check manuel |

### Périmètre conservé

- **Tous les tests** (`make test`, `make test-short`, fuzz I1-I5 30 s, e2e, empirical) — le **code** des tests reste inchangé. Seule l'**orchestration automatisée** est retirée.
- `Makefile` cibles `build`, `build-pgo`, `build-all`, `test`, `test-short`, `coverage`, `benchmark`, `lint`, `fuzz`, `calibrate`, `clean`.
- Gate per-package ≥ 90 % `tau/*` et global ≥ 80 % : conservés comme **objectifs locaux vérifiables** via `make coverage` (rapport HTML).
- Étanchéité Clean Architecture (`internal/arch_test.go`, 7 règles) — toujours exécutée par `make test`.
- Anti-patrons §7.2 #1-7 — toujours gardés par tests (`TestNoPredictiveAPI`, `TestFrontierCheck_Inside_*`, `TestExpiredProfileRefuses`, `TestUnmodeledObservationsReported`, `TestArchNoConcreteLLMInDomain`, `gochecknoglobals`).

---

## Conséquences

### Positives

- **Simplicité** — un seul environnement (le poste développeur). Pas de drift CI ↔ local.
- **Vitesse de feedback** — pas d'attente runner GitHub Actions (5-10 minutes pour une matrice 3 OS).
- **Coût** — zéro consommation minutes GitHub Actions.
- **Autonomie** — aucune dépendance externe pour valider une modification.
- **Reproductibilité de l'audit** — les vérifications restent toutes exécutables localement avec les commandes documentées dans `PRD.md` §15.3 et `CLAUDE.md` §Commandes essentielles.

### Négatives (acceptées)

- **Plus de gate automatisé de merge** — la discipline `make test && make lint && make fuzz` avant commit devient une convention sociale, plus une garde automatisée. Si elle est oubliée, du code défaillant peut atteindre `main`.
- **Plus de matrice OS** — Linux/macOS/Windows validés simultanément dans le même run disparaît. Couverte par `go build` cross-compile (`make build-all`) au moment du release, mais pas testée en runtime sur chaque OS à chaque commit.
- **Plus d'alerte 30 jours avant péremption I3** — qui passait par CI à fréquence quotidienne. **Bascule en cron externe ou check manuel** — le risque PRD §18 #9 est désormais mitigé par : (a) garde runtime `TestExpiredProfileRefuses` qui bloque toute décision si profil périmé ; (b) `app.NewDispatcher()` qui charge un profil par défaut activant la garde sur chemin CLI (P0-02 fermé v0.1.1).
- **Plus de race detector sur 3 OS** — `go test -race` exige CGO ; sous Windows local sans gcc, retombe sur `go test -short ./...`. Sous Linux/macOS, le développeur exécute `make test` localement.
- **Plus de fuzz 24 h nocturne** — devient exécution manuelle sur demande.

### Réversibilité

**Réversible à coût quasi nul.** Pour réintroduire la CI :

1. Récréer `.github/workflows/ci.yml` et `coverage.yml` à partir de l'historique git (commit antérieur à v0.1.2).
2. Restaurer les cibles `Makefile` retirées.
3. Rouvrir une décision ADR explicite révoquant la présente.

Cette éventualité est tracée dans `PRD.md` §20.2 #9 (« Réintroduction CI minimale (option) — si le projet grandit »).

---

## Alternatives considérées

### A. Garder une CI minimale (lint + build + test short)

**Rejetée.** Bénéfice marginal (vérification locale déjà disciplinée) contre coût de maintien et drift potentiel. Si le projet grandit (multiples contributeurs, branches concurrentes), cette option redeviendra pertinente.

### B. Migrer vers un autre orchestrateur (Drone, CircleCI, GitLab CI…)

**Rejetée.** Le problème n'est pas l'orchestrateur ; c'est la pertinence d'une CI automatisée pour un projet à un mainteneur unique avec discipline locale stricte.

### C. Garder les workflows mais les désactiver

**Rejetée.** Fichiers morts dans le repo, ambiguïté sur l'état attendu. Préférer la suppression franche et la réintroduction explicite si besoin.

### D. Pre-commit hooks Git

**Considérée pour V0.2.** Pourrait remplacer une partie du rôle de garde (lint, test-short) sans pousser dans GitHub Actions. Hors scope v0.1.2 — à proposer en ADR dédiée si la discipline locale s'avère insuffisante.

---

## Critères de retournement

Cette décision **doit être réévaluée** si l'un des éléments suivants survient :

1. **≥ 2 contributeurs actifs** sur le repo (PR de tiers, équipe distribuée). La discipline « valider avant push » se dilue mécaniquement à plusieurs.
2. **Première occurrence d'un bug atteignant `main`** qui aurait été bloqué par une CI minimale (lint failure, test rouge, fuzz crash).
3. **Externalisation du projet** (dépendance d'un client/sponsor exigeant des artefacts CI signés ou un badge build vert).
4. **Adoption de pre-commit hooks** ne couvrant pas la matrice OS (Windows-only et Linux-only divergent silencieusement).

---

## Vérifications post-décision

- `make test` vert localement (Linux/macOS avec `-race` via CGO ; Windows via `-short`).
- `make lint` vert (golangci-lint v1.64.8, 24 linters).
- `make fuzz` vert (30 s sur I1-I5).
- `go build ./...` vert.
- `git log` montre suppression `.github/` + Makefile refactor + alignement doc dans **un seul commit conventionnel** `chore(ci): retrait outillage CI/CD complet (ADR-0010)`.
- `CHANGELOG.md` section v0.1.2-pre documentant le retrait.
- Aucune référence orpheline « gate CI », « workflow », « coverage.yml », « CI matrix verte » dans `README.md` / `CLAUDE.md` / `PRD.md` (audit grep manuel).

---

## Références

- [`.github/workflows/ci.yml`](https://github.com/agbruneau/taugo/blob/v0.1.1-pre/.github/workflows/ci.yml) (historique git, retiré v0.1.2)
- [`.github/workflows/coverage.yml`](https://github.com/agbruneau/taugo/blob/v0.1.1-pre/.github/workflows/coverage.yml) (historique git, retiré v0.1.2)
- [`CHANGELOG.md`](../../CHANGELOG.md) section v0.1.2-pre
- [`README.md`](../../README.md) § Stack technique (rangée « Validation »)
- [`CLAUDE.md`](../../CLAUDE.md) §10 Directives projet (« Validation locale obligatoire »)
- [`PRD.md`](../../PRD.md) §15.3 (Gates locaux), §17 critères #4/#5/#8, §18 risque #9, §20.2 #9 (réintroduction option V0.2+)
- ADR-0011 (à créer V0.2) — bridge TauGo ↔ `cia-runtime` (numérotation décalée par la présente)
