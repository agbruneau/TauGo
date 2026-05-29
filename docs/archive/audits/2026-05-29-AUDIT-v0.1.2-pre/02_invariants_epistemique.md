# 02 — Invariants & epistemique

> Audit Go multi-agents TauGo — axe « Invariants I1-I5 & rigueur épistémique ». HEAD `1948a7b`, v0.1.2-pre, 2026-05-29. Lecture seule, FR-CA. Sévérités : CRITIQUE 0 · MAJEUR 2 · MINEUR 4 · INFORMATIF 2.

**Conclusion (headline) :** [confirme] Les cinq cibles fuzz I1-I5 passent sans aucun crash ni contre-exemple (43M/40M/42M/44M/32M executions sur 30 s chacune) et toute la suite internal/ est verte ; le coeur invariant est sain. La fragilite est epistemique, pas algorithmique : (1) docs/theory/07-anti-patrons.md ne documente que 4 anti-patrons et s'auto-declare "Confirme" alors que le projet en garde 7 (survente de statut) ; (2) l'artefact empirique I4 suivi en git embarque un timestamp time.Now() qui brise la reproductibilite byte-identique pourtant affirmee "gele" ; (3) un test V1 d'I4 porte un nom et un commentaire obsoletes affirmant une limitation deja levee par ADR-0008. Aucune decision incorrecte ni invariant viole detecte.

**Outils exécutés :**
- go test -fuzz=FuzzI1_Conservation -fuzztime=30s ./internal/tau/invariants/ -> PASS, 43 087 352 execs, 0 crash
- go test -fuzz=FuzzI2_Irreductibilite -fuzztime=30s -> PASS, 40 356 436 execs, new interesting 0, 0 crash
- go test -fuzz=FuzzI3_AsymetrieAutorite -fuzztime=30s -> PASS, 42 188 331 execs, 0 crash
- go test -fuzz=FuzzI4_CoherenceContrainte -fuzztime=30s -> PASS, 44 335 791 execs, 0 crash
- go test -fuzz=FuzzI5_CompositionConjonctive -fuzztime=30s -> PASS, 32 059 304 execs (debit ~1M/s, conforme au commentaire i5_composition.go), 0 crash
- go test -tags=empirical ./internal/bridge/agentmeshkafka/... -run=^TestEmpiricalI4Campaign$ -count=1 -v -> PASS ; a regenere testdata/empirical-i4-results.json (timestamp + champ sensitivity), restaure via git restore
- git diff puis git restore -- internal/bridge/agentmeshkafka/testdata/empirical-i4-results.json -> arbre propre confirme
- go test ./internal/... -run Test -count=1 -> tous les packages OK (invariants, app, orchestration, calibration, dimensions, bridge...)
- go vet ./internal/tau/invariants/... ./internal/orchestration/... -> aucun diagnostic

**Outils indisponibles / repli :**
- -race : indisponible (CGO_ENABLED=0, pas de compilateur C sous Windows) -> fuzz et tests executes sans detecteur de course ; absence de data race [a verifier] sur plateforme CGO Linux/macOS
- make : absent (ADR-0010) -> invocations go directes, conforme a la consigne

---

# Audit SA2 — Invariants I1-I5 et rigueur epistemique

## Conclusion

Le coeur algorithmique des invariants est sain : les cinq cibles fuzz passent sans crash ni contre-exemple sur 30 s chacune (I1 43M, I2 40M, I3 42M, I4 44M, I5 32M executions), toute la suite `internal/` est verte et `go vet` est propre. **Aucune decision incorrecte, aucun invariant viole, aucun non-determinisme de calibration detecte au niveau code.** Les faiblesses sont **epistemiques et documentaires**, pas fonctionnelles : un document theorique survend son completude (4 anti-patrons / "Confirme" pour un projet qui en garde 7), un artefact empirique suivi en git n'est pas reproductible malgre l'affirmation contraire, et un test porte un nom/commentaire trompeur. `-race` indisponible (pas de CGO) : l'absence de data race reste *[a verifier]* sur plateforme Linux/macOS.

## Resultats fuzz (preuve d'execution)

Tous executes sequentiellement, 30 s, une cible a la fois.

| Cible | Execs | new interesting | Verdict |
|---|---|---|---|
| `FuzzI1_Conservation` | 43 087 352 | 1 | PASS |
| `FuzzI2_Irreductibilite` | 40 356 436 | 0 | PASS |
| `FuzzI3_AsymetrieAutorite` | 42 188 331 | 1 | PASS |
| `FuzzI4_CoherenceContrainte` | 44 335 791 | 1 | PASS |
| `FuzzI5_CompositionConjonctive` | 32 059 304 | 0 | PASS |

`[confirme]` Zero crash, zero `t.Fatalf` declenche sur les cinq cibles. Le debit I5 (~1M exec/s) est conforme au commentaire `i5_composition.go:88` qui annonce un decodage `Pile` plus lourd. Note : le CLAUDE.md annonce des debits de ~8.6M/s pour I1-I3 mais les mesures observees ici sont ~1.4M/s — l'ecart s'explique probablement par la plateforme (Windows, sans CGO) et n'est pas un defaut ; les chiffres CLAUDE.md restent *[a verifier]* sur la plateforme de reference.

## Constats classes par severite

### MAJEUR

**[I2-01] MAJEUR — theory/07-anti-patrons.md survend son statut : 4 anti-patrons "Confirme" pour un projet qui en garde 7** `[confirme]`
- Fichier : `docs/theory/07-anti-patrons.md:5`, `:11`, `:105-112`
- Preuve (verbatim) : preambule ligne 5 « *Statut global : 4 anti-patrons gardes par test au tag v0.1.0. Confirme.* » ; ligne 11 « *Quatre usages de tau contredisent ses hypotheses fondatrices* » ; le tableau final (`:107-112`) ne liste que AP#1-AP#4. Or `CLAUDE.md` §Anti-patrons interdits enumere **7** anti-patrons gardes depuis v0.1.1, dont #6 (import LLM concret, garde `TestArchNoConcreteLLMInDomain` — verifiee presente a `internal/arch_test.go:140`) et #7 (globaux mutables). La derniere note de revision du meme fichier (`:120`, « Daté 2026-05-24 ») est posterieure a v0.1.1.
- Impact : pour un projet de recherche dont la discipline epistemique est un livrable, un document theorique qui se declare « Confirme » tout en omettant 3 gardes reellement actives est une **survente de statut** : un lecteur conclut a tort que le perimetre anti-patrons est entierement decrit et trace ici. Viole la regle CLAUDE.md « zero fabrication / marqueurs honnetes ».
- Recommandation : (1) compromis principal — mettre a jour theory/07 pour couvrir AP#5 (citation fabriquee), #6 (LLM concret) et #7 (globaux) avec leurs gardes (`arch_test.go`, `gochecknoglobals`) ; (2) alternative — ajouter un encart « ce document couvre 4 des 7 anti-patrons ; voir CLAUDE.md pour #5-#7 » si le decoupage est intentionnel ; (3) condition de retournement — si la monographie chap. III.8.7 ne definit effectivement que 4 anti-patrons et que #5-#7 sont des gardes d'ingenierie hors theorie, alors le titre du document doit le dire explicitement plutot que se declarer « Confirme » globalement.

**[I2-02] MAJEUR — artefact empirique I4 suivi en git non reproductible malgre l'affirmation "gele / re-execution identique"** `[confirme]`
- Fichier : `internal/bridge/agentmeshkafka/empirical_i4_test.go:124` ; `docs/empirical/I4-report.md:26-27`
- Preuve (verbatim) : le test ecrit `Timestamp: time.Now().UTC().Format(time.RFC3339)` dans `testdata/empirical-i4-results.json`, fichier **suivi en git**. La re-execution a produit un diff reel :
  ```
  -  "timestamp": "2026-05-24T11:27:43Z",
  +  "timestamp": "2026-05-29T11:01:03Z",
  ```
  Or `I4-report.md:26-27` affirme « *Le corpus est gele [...] Toute re-execution doit produire des resultats identiques sur le meme profil.* » De plus, le checked-in contenait `"sensitivity": -1` que la regeneration supprime — signe que l'artefact versionne a ete edite a la main apres generation (le code `derefOrNeg1` emet toujours -1, jamais un champ absent).
- Impact : la promesse de reproductibilite est fausse pour cet artefact. Chaque execution `-tags=empirical` salit l'arbre git (regression silencieuse possible si commitee par megarde). Pour une campagne empirique qui sert de **preuve datee** d'un statut « Hypothese », un artefact non deterministe affaiblit l'opposabilite. (Fichier restaure via `git restore` durant l'audit — arbre propre confirme.)
- Recommandation : (1) compromis principal — exclure le timestamp du fichier byte-stable (le deplacer en log `t.Logf` plutot qu'en payload versionne), ou figer l'horloge via une injection comme le fait `WithClock` ailleurs ; (2) alternative — sortir l'artefact du suivi git (`.gitignore`) si c'est un produit derive ; (3) reconcilier la divergence `sensitivity: -1` checked-in vs absent en regeneration (le contenu versionne ne correspond pas a ce que le code produit).

### MINEUR

**[I2-03] MINEUR — test I4 au nom mensonger et commentaire obsolete sur une capacite deja livree** `[confirme]`
- Fichier : `internal/tau/invariants/i4_coherence_test.go:72-101`
- Preuve (verbatim) : la fonction s'appelle `TestEvaluateI4_IncoherentNonRefused_Violated` mais asserte `got != invariants.Held` (attend **Held**, pas Violated). Le commentaire annonce « *M5 will update this to Violated when Trace.DSens / Trace.DInvariant are available* ». Or cette capacite a deja atterri (ADR-0008, v0.1.1) : `EvaluateI4` lit `dec.Trace.DSens/DInvariant` (`i4_coherence.go:41-45`) et la detection est couverte par `TestEvaluateI4_DetecteByPassSilencieux` (`internal/orchestration/dispatcher_scores_test.go:102`, attend `Violated`).
- Impact : nom + commentaire trompeurs ; un lecteur croit la detection de bypass non implementee alors qu'elle l'est et qu'elle est testee ailleurs. Dette documentaire dans le module I4, l'un des modules sensibles.
- Recommandation : renommer (ex. `..._NoVentilatedScores_Held`) et reecrire le commentaire pour pointer la limitation reelle (verdict Held *quand les scores ventiles sont absents*), en renvoyant a `TestEvaluateI4_DetecteByPassSilencieux` pour le chemin V2.

**[I2-04] MINEUR — EvaluateI1/EvaluateI2 comparent le diagnostic a un litteral code en dur au lieu de la constante anti-drift** `[confirme]`
- Fichier : `internal/tau/invariants/i1_conservation.go:21` ; `internal/tau/invariants/i2_irreductibility.go:73`
- Preuve (verbatim) : `if dec.Regime == tau.Refus && dec.Diagnostic == "hors frontière τ"`. La constante canonique existe : `tau.DiagFrontiereFranchie = "hors frontière τ"` (`internal/tau/diagnostics.go:7`), dont le commentaire dit explicitement « *to prevent string drift on Refus messages* ». I3 (`i3_authority_asymmetry.go:64`) et I4 (`i4_coherence.go:34`) utilisent bien les constantes ; I1 et I2 ne le font pas.
- Impact : faible aujourd'hui (les chaines coincident, fuzz vert), mais c'est exactement le risque de drift que la constante etait censee eliminer ; une modification de la chaine canonique sans toucher I1/I2 ferait diverger silencieusement le verdict `NotApplicable`.
- Recommandation : remplacer les deux litteraux par `tau.DiagFrontiereFranchie`. Modification chirurgicale, deux lignes.

**[I2-05] MINEUR — incoherences de dates dans l'instrumentation datee d'I3** `[confirme]`
- Fichier : `internal/tau/invariants/i3_authority_asymmetry.go:12` et `:101` ; `internal/calibration/profile.go:47` ; `docs/theory/05-invariants.md:93`,`:103`
- Preuve (verbatim) : `I3PerimptionLimite` godoc dit « *Status: Probable. Dated 2026-05-24* » (`i3:12`) mais `EvaluateI3` godoc dit « *Status: Probable, dated 2026-05-16* » (`i3:101`, identique a theory/05:103). Par ailleurs trois dates-cible coexistent : `DefaultProfile().DateRevision = 2026-12-01` (`profile.go:47`), `I3PerimptionLimite() = 2027-01-01` (`i3:14`), et revérification theorique « 2026-12-01 » (`theory/05:93`). La note d'alignement theory/05:225 signale elle-meme cette tension comme « coherence a verifier ».
- Impact : pour un invariant dont la datation est le mecanisme de peremption, des dates divergentes entre godoc, profil et theorie nuisent a la tracabilite de la veille trimestrielle. Aucun effet fonctionnel (la garde compare a `DateRevision` du profil, soit 2026-12-01, qui est bien anterieure a la limite 2027-01-01 et gardee par `TestI3_DateRevisionRespectee`).
- Recommandation : unifier la date de datation (2026-05-16 vs 2026-05-24) entre les deux godocs d'i3, et documenter explicitement l'asymetrie voulue entre `DateRevision` profil (2026-12-01) et `I3PerimptionLimite` (2027-01-01).

**[I2-06] MINEUR — EvaluateI5 retourne toujours Held ; I5 non exerce sur les decisions reelles** `[confirme]`
- Fichier : `internal/tau/invariants/i5_composition.go:94-98`
- Preuve (verbatim) : `func EvaluateI5(_ tau.Exchange, _ tau.Decision) Status { return Held }` ; commentaire « *V1: stack not reified in Trace. Held by construction; fuzz verifies BoundsHold on arbitrary generated stacks independently.* »
- Impact : dans le pipeline `Decide` -> `EvaluateInvariants`, I5 est un no-op qui ne peut jamais signaler de violation sur une decision reelle ; seule la propriete mathematique `BoundsHold` est fuzzee, hors chemin de decision. Le decouplage est assume et documente (eviter un couplage speculatif `Decision`<->pile avant V2), mais il faut etre conscient que la garde I5 sur les decisions est inerte en V1.
- Recommandation : conserver le decouplage (conforme a Simplicity First), mais s'assurer que le statut « Probable » d'I5 dans CLAUDE.md/theory soit lu comme « propriete mathematique fuzzee », non « invariant verifie sur chaque decision ». Aucune action code requise en V1.

### INFORMATIF

**[I2-07] INFORMATIF — statut I4 "Hypothese" honnete et correctement justifie** `[confirme]`
- Fichier : `docs/empirical/I4-report.md:84`,`:94`,`:106` ; `internal/tau/invariants/i4_coherence.go:14`
- Preuve (verbatim) : I4-report.md:84 « *La raison directe est instrumentale : le generateur cmd/generate-corpus ne peuple pas les champs Context [...] le score D-INVARIANT reste a 0.25 [...] sous le seuil theta_inv = 0.50. L'invariant I4 ne se declenche donc jamais* » ; verdict §6 « *Hypothese inchangee [...] campagne inconclusive sur I4* ». La campagne re-executee confirme : TP=0, FN=0, specificity=1, sensitivity indefinie.
- Analyse de la question-cle : le marqueur « Hypothese » est **honnete**. La detection ventilee v0.1.1 (ADR-0008) ne change PAS le statut empirique : elle ameliore la *capacite de detection du code* (testee, `TestEvaluateI4_DetecteByPassSilencieux`) mais le *corpus empirique* ne sollicite toujours pas la garde car D-INVARIANT reste sous le seuil. Pour promouvoir I4 Hypothese -> Probable, il manque precisement : (a) un corpus dont les cles `Context` (`event_registry`, `idempotency_key_mode`, profondeur de delegation) font monter D-INVARIANT >= theta_inv ; (b) au moins quelques vrais positifs I4 (refus declenche par incoherence reelle) — l'I4-report.md:121-122 cible « >= 10 vrais positifs » via le profil `i4-heavy` enrichi en Context ; (c) idealement, des traces reelles (AgentMeshKafka inexistant, §1). Tant que (a) n'est pas livre, la campagne reste structurellement inconclusive, quel que soit le profil. Bonne pratique : le rapport distingue explicitement « absence de preuve » de « preuve d'absence » (`I4-report.md:135`).
- Recommandation : aucune correction ; signaler comme exemple de rigueur. Suivre l'action M4-bis (enrichissement Context du generateur) pour debloquer la promotion de statut.

**[I2-08] INFORMATIF — ecart assume entre propriete monographique complete d'I2 et reformulation V1 testee** `[confirme]`
- Fichier : `internal/tau/invariants/i2_irreductibility.go:71` ; `docs/theory/05-invariants.md:14`,`:73`,`:223`
- Preuve (verbatim) : le code annonce « *Status: Confirmed by construction* » et CLAUDE.md « Confirmé ». La propriete reellement encodee et fuzzee est la propriete V1 reduite : « residu non-vide ET recablage complet fait perdre Inside() », operant sur 4 conditions de frontiere mappees 1:1 (`i2:5-19`). L'enonce monographique complet (theory/05:53) porte sur un residu de *sens + autorite + support* preservant *univers ouvert + composition variable + nature probabiliste du pair*.
- Impact : le marqueur « Confirme » est honnete *pour la propriete V1 encodee* (la plus deductive des cinq, fuzz 40M vert). L'ecart entre verbatim complet et reformulation testee n'est pas signale dans le code Go, mais il l'est correctement dans theory/05 (note d'alignement :223 « fidele » et :73 « Confirme par construction — le plus deductif »). Pas de survente, mais asymetrie de tracabilite entre code et theorie.
- Recommandation : aucune action bloquante ; eventuellement ajouter une ligne de renvoi dans le godoc `EvaluateI2` vers la note d'alignement theory/05 pour rendre l'ecart V1/complet visible au niveau code.

## Synthese des 7 anti-patrons (gardes verifiees)

| # | Anti-patron | Garde | Statut verifie |
|---|---|---|---|
| 1 | API predictive | `TestNoPredictiveAPI` (`anti_patterns_test.go:34`, AST sur 4 packages) | `[confirme]` present, vert |
| 2 | Bypass frontiere | `TestFrontierCheck_Inside_*` (`frontier_test.go:73,86`) | `[confirme]` present, vert |
| 3 | Profil perime tolere | etape 3 dispatcher (`dispatcher.go:161`) + `TestApp_NewDispatcher_*` (`app_test.go:22,70`) + `TestExpiredProfileRefuses` | `[confirme]` actif sur chemin CLI : `app.NewDispatcher` injecte `DefaultProfile()` (`app.go:28-29`) ; `NewDispatcher` sans profil (etape 3 desactivee) est documente « internal tests only » mais utilise par la campagne empirique `-tags=empirical` (`empirical_i4_test.go:78`) — acceptable car hors production |
| 4 | Usage clos / observation non modelisee | `TestUnmodeledObservationsReported` (`anti_patterns_test.go:119`) + dispatcher etape 8 (`dispatcher.go:217-223`) | `[confirme]` present, vert |
| 5 | Citation fabriquee | audit PR (pas de garde automatisable) | `[confirme]` hors perimetre test, par revue |
| 6 | LLM concret hors app/ et bridge/llm/ | `TestArchNoConcreteLLMInDomain` (`arch_test.go:140`, walk AST sur 12 substrings : anthropic, openai, mistralai, cohere, genai, ollama, groq...) sur `internal/tau/` + `internal/orchestration/` | `[confirme]` present, vert ; garde porte sur le domaine (correct) |
| 7 | Globaux mutables non synchronises | `gochecknoglobals` + revue ; seuls globaux dans tau/* sont des lookup tables immuables `//nolint` justifie (`operator.go:27-28`,`:114-115`) ; `I3PerimptionLimite` est un getter (`i3:13`) | `[confirme]` aucun global mutable non synchronise dans tau/* |

`[confirme]` Les 7 anti-patrons sont gardes ou justifies. Seule reserve : la documentation theorique (theory/07) ne reflete que 4 d'entre eux (cf. I2-01).

## Notes d'execution

- Restauration effectuee : `git restore -- internal/bridge/agentmeshkafka/testdata/empirical-i4-results.json` apres la campagne empirique (le test regenere ce fichier suivi). Arbre git propre confirme post-restauration (seul `audit/` non-suivi, emplacement autorise).
- `-race` non execute : `CGO_ENABLED=0`, pas de compilateur C. L'absence de data race/deadlock dans les invariants et le dispatcher reste *[a verifier]* sur plateforme CGO (Linux/macOS).
