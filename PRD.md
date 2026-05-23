# PRD — TauGo

**Projet** : TauGo
**Type** : kernel exécutable Go pour l'opérateur τ et la validation empirique des invariants I1–I5
**Auteur** : Andre-Guy Bruneau, M.Sc.
**Statut** : V0 — proposition d'initialisation, à valider avant `git init`
**Date** : 2026-05-23
**Référence canonique** : `agbruneau/InteroperabiliteAgentique`, chap. III.8 (opérateur τ, dimensions, invariants)

---

## 1. Finalité

TauGo implémente le **kernel exécutable de l'opérateur τ** défini dans la monographie *Interopérabilité Agentique en Écosystème d'Entreprise*. Il bâtit le pont entre :

- la **théorie** (monographie : τ, D-SENS, D-AUTORITÉ, D-INVARIANT, invariants I1–I5)
- l'**empirie** (validation contre `AgentMeshKafka`, traces réelles)
- l'**ingénierie** (discipline `FibGo` : dispatch multi-mode, calibration adaptative, fuzz d'identités/propriétés, déterminisme reproductible)

TauGo est le **livrable empirique du Calcul d'Intégration Agentique (CIA)** — celui qui transforme la formalisation en preuve mesurable.

## 2. Thèse exécutable

τ est un opérateur de **dispatch** entre régime déterministe (garantie de message, protocole strict) et régime probabiliste (agent LLM, raisonnement ouvert). TauGo opérationnalise cette décision en temps réel sur trois dimensions, sous cinq invariants.

| Dimension monographie | Manifestation exécutable V1 |
|---|---|
| D-SENS | Score sémantique [0,1] (similarité d'intention, ancrage contexte) |
| D-AUTORITÉ | Autorité de l'appelant × autorité requise par la cible |
| D-INVARIANT | État courant des invariants I1–I5 sur la trace en cours |

Décision τ : `regime ∈ {déterministe, probabiliste, refus}` fonction des trois scores et de seuils calibrés.

### Invariants — formulation exécutable (à compléter)

| Invariant | Énoncé chap. III.8 | Cible fuzz V1 |
|---|---|---|
| I1 | *à compléter* | `FuzzI1_*` |
| I2 | *à compléter* | `FuzzI2_*` |
| I3 | *à compléter* | `FuzzI3_*` |
| I4 | *à compléter* | `FuzzI4_*` |
| I5 | *à compléter* | `FuzzI5_*` |

> **Action requise avant `git init`** : remplir les cinq énoncés depuis chap. III.8. TauGo refuse de démarrer avec des invariants flous — c'est précisément la discipline FibGo (les identités `F(2k) = F(k)·(2F(k+1) − F(k))` ne sont pas optionnelles).

## 3. Périmètre V1 — mince et focalisé

### Inclus

- Bibliothèque Go (`internal/tau/`) — dispatcher déterministe ↔ probabiliste
- Trois dimensions calculables avec scores normalisés [0,1]
- Cinq invariants I1–I5 sous forme de cibles fuzz
- Calibration adaptative des seuils (pattern FibGo, profil versionné, invalidation par drift)
- Adaptateur `AgentMeshKafka` (validateur empirique)
- CLI minimale `cmd/tau/` : dispatch, dump de trace, rapport d'invariants
- CI : `go test -race`, fuzz court, build reproductible, golangci-lint
- Documentation alignée monographie (renvois explicites aux chapitres)

### Exclus de V1 (reportés)

- **V2 — `cia-runtime`** : mécanisation Lean 4 des invariants, génération de tests Go depuis preuves
- **V3 — `tau-stack`** : TUI Bubble Tea, replay de traces, calibration en charge
- Couche RAG sur `ruvector.db` (étude séparée requise — rôle du store vectoriel à clarifier)
- Service réseau (gRPC/HTTP) — V1 est lib + CLI

### Non-objectifs (anti-platform discipline)

- TauGo **n'est pas** un framework agentique générique
- TauGo **n'orchestre pas** d'agents — il décide *comment* les appeler
- TauGo **n'embarque pas** de LLM — il consomme un client LLM injecté via interface

## 4. Architecture cible

Clean Architecture, quatre couches, calque structurel de FibGo.

```
cmd/tau/                       # entry point CLI
internal/
  app/                         # lifecycle, dispatch, version
  tau/                         # opérateur τ
    dimensions/                # D-SENS, D-AUTORITÉ, D-INVARIANT
    invariants/                # I1–I5, propriétés fuzz
  orchestration/               # dispatch déterministe ↔ probabiliste
  calibration/                 # seuils adaptatifs, profils versionnés
  bridge/
    agentmeshkafka/            # validateur empirique
    llm/                       # interface client LLM (injectée)
  config/                      # flags, env, seuils
  errors/                      # erreurs typées, codes sortie
  metrics/                     # observabilité (compteurs, histogrammes)
docs/
  theory/                      # renvois monographie chap. III.8
  invariants/                  # spécification exécutable I1–I5
  algorithms/                  # dispatch, calibration, seuils
test/
  e2e/                         # bout-en-bout via AgentMeshKafka
CLAUDE.md                      # conventions de rédaction (alignées monographie)
README.md
LICENSE                        # Apache-2.0 (cohérent FibGo)
```

## 5. Compromis & alternatives

### Compromis principal

**Focalisation vs. ambition.** TauGo doit faire *une* chose — décider du régime d'appel agentique — très bien. La tentation sera d'ajouter orchestration, mémoire d'agent, service réseau. Chaque ajout dilue la thèse exécutable et déplace TauGo vers une « plateforme agentique de plus », qui n'a aucune valeur démonstrative pour CIA.

### Alternative 1 — Repo monolithique `cia-runtime`

Inclure Lean 4 dès V1 : preuves + tests générés. **Rejeté pour V1** : double risque (boucle Lean ↔ Go peu balisée + scope d'ingénierie large). Devient V2 si V1 stabilise les invariants exécutables.

### Alternative 2 — Repo `delegated-authority-empirical`

Focaliser sur chantier V1 de la monographie (autorité déléguée, six Internet-Drafts congestionnés, terrain z/OS Connect 3.0.98 MCP). **Rejeté si τ est priorité publication CIA** ; à reconsidérer si chantier V1 prend précédence éditoriale.

### Conditions qui renversent la recommandation

- Refonte majeure du chap. III.8 dans la monographie → geler TauGo jusqu'à stabilisation
- `AgentMeshKafka` non prêt à servir de validateur → stabiliser d'abord
- Découverte d'un dépôt académique existant couvrant τ exécutable → contribuer plutôt que reproduire

## 6. Stack technique

- **Go 1.25+** (cohérent FibGo)
- **Dépendances minimales** : `errgroup`, `sync/atomic`, `math/big` si calculs ; clients LLM injectés via interface stricte
- **Pas de framework** (ni Bubble Tea en V1, ni grpc, ni cobra — `flag` standard suffit)
- **Linting** : `golangci-lint` (config héritée FibGo)
- **Build reproductible** : Makefile, `-trimpath`, PGO optionnel, cross-compile linux/windows/darwin
- **Licence** : Apache-2.0

## 7. Conventions éditoriales (alignées monographie)

- **Langue** : français (Canada) pour `docs/` et commentaires structurants ; commentaires d'API publique en anglais (ergonomie godoc)
- **Incertitude calibrée** : marqueurs *Confirmé / Probable / Hypothèse / À vérifier* obligatoires dans `docs/`
- **Zéro fabrication** : aucune citation, chiffre ou API inventée ; chaque affirmation factuelle vérifiable
- **Reproductibilité** : builds déterministes byte-identiques en CI (timestamps gelés)
- **Renvois croisés** : chaque décision de design dans `docs/theory/` cite le chapitre de monographie qui la motive
- **Anonymisation** : aucun cas Desjardins identifiable ; références publiques (MCP, A2A, AGNTCY, IBM, IETF) libres

## 8. Roadmap V1

| Milestone | Contenu | Critère d'acceptation |
|---|---|---|
| M0 | Squelette repo, CI, `CLAUDE.md`, conventions, `.golangci.yml` | `git init` + premier commit vert |
| M1 | Dispatcher minimal, deux régimes, mocks LLM | Dispatch d'une demande de bout en bout via CLI |
| M2 | Trois dimensions calculables + score τ composite | Rapport de décision instrumenté avec scores |
| M3 | Cinq invariants comme cibles fuzz | `go test -fuzz=. -fuzztime=30s` vert sur I1–I5 |
| M4 | Adaptateur `AgentMeshKafka` | Une trace empirique end-to-end |
| M5 | Calibration adaptative + persistance versionnée | Profils invalidés sur drift de modèle |
| M6 | Documentation alignée monographie + release `v0.1.0` | Tag, `CHANGELOG.md`, `README.md` final |

**Estimation indicative** : 6–10 semaines à temps partiel. *À vérifier* selon votre disponibilité réelle.

## 9. Critères de succès V1

- ✅ Dispatch τ instrumenté sur un cas BFSI réaliste (banking ou IARD, anonymisé)
- ✅ Cinq invariants exprimés exécutablement et testés par fuzz (≥ 30 s par cible sans panique)
- ✅ Une trace de validation empirique end-to-end via `AgentMeshKafka`
- ✅ Build reproductible byte-identique en CI
- ✅ Couverture de tests ≥ 80 %
- ✅ Chaque décision de design dans `docs/` renvoie au chapitre de monographie qui la motive

## 10. Risques & mitigation

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| `AgentMeshKafka` pas prêt à servir de validateur | Probable | Élevé | Stabiliser AgentMeshKafka avant M4 ; mock intermédiaire en attendant |
| Invariants I1–I5 trop abstraits pour fuzz direct | Probable | Moyen | Reformulation exécutable au M3 ; revue ciblée chap. III.8 |
| Drift entre TauGo et révisions de la monographie | Probable | Moyen | Tag de version monographie épinglé dans `CLAUDE.md` |
| Scope creep vers framework agentique | Probable | Élevé | §3 « non-objectifs » fait foi ; revue mensuelle stricte |
| Interface LLM injectée fuit l'abstraction probabiliste | À vérifier | Moyen | Interface étroite ; tests systématiques avec stub déterministe |
| `ruvector.db` impose un couplage prématuré au RAG | Probable | Faible | Exclu de V1 ; étude séparée |

## 11. Prochaines étapes pour Claude Code

À la première session Claude Code dans ce repo, exécuter dans cet ordre :

1. **Lire** ce `PRD.md` intégralement et confirmer la compréhension du périmètre V1
2. **Confirmer** la formulation exécutable des invariants I1–I5 (référence chap. III.8 de `InteroperabiliteAgentique/Monographie.md`)
3. **Générer** le squelette Clean Architecture conforme à §4
4. **Rédiger** `CLAUDE.md` héritant des conventions de `InteroperabiliteAgentique` (langue fr-CA, marqueurs d'incertitude, reproductibilité byte-identique)
5. **Configurer** la CI : `golangci-lint`, `go test -race`, fuzz court, build reproductible
6. **Premier commit signé** sur `main`, premier tag `v0.0.1-alpha`
7. **Ouvrir issue M1** : implémenter dispatcher minimal deux régimes

## 12. Documents liés

- `agbruneau/InteroperabiliteAgentique` — monographie source, chap. III.8 canonique
- `agbruneau/AgentMeshKafka` — substrat de validation empirique
- `agbruneau/FibGo` — référence d'ingénierie (dispatch multi-algo, calibration, fuzz d'identités, déterminisme)
- `agbruneau/FibRust` — référence ergonomie type-safe (pertinente si extension Rust envisagée en V3+)

---

*Ce PRD est un document vivant. Toute déviation matérielle doit être justifiée par mise à jour de ce fichier — en premier, avant le code.*
