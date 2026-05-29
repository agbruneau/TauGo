# 03 — Concurrence

> Audit Go multi-agents TauGo — axe « Concurrence & data races ». HEAD `1948a7b`, v0.1.2-pre, 2026-05-29. Lecture seule, FR-CA. Sévérités : CRITIQUE 0 · MAJEUR 2 · MINEUR 2 · INFORMATIF 2.

**Conclusion (headline) :** Aucun risque CRITIQUE de concurrence sur le chemin de decision: Decide et les 8 etapes du dispatcher sont purement par valeur, sans goroutine, sans etat mutable partage, sans atomique [confirme]. La surface concurrente reelle se limite a deux fichiers de streaming (internal/app/agentmesh.go, internal/bridge/agentmeshkafka/file_adapter.go) hors du chemin Decide, plus AtomicThresholds qui est code mort jamais cable au dispatcher. AVERTISSEMENT: -race est indisponible (CGO_ENABLED=0, aucun compilateur C) — la confiance sur les races runtime reste donc plafonnee a [probable]/[a verifier]; les constats reposent sur go vet (clean) et analyse statique de code [a verifier]. Deux constats MAJEURS: errOut a perte silencieuse d erreurs et fenetre RMW non atomique dans SetTuning (sans impact actuel car non utilise).

**Outils exécutés :**
- CGO_ENABLED=0 go vet ./... -> aucun avertissement (exit 0)
- CGO_ENABLED=0 go vet -copylocks -atomic ./... -> exit 0, aucun avertissement (couvre copie de sync.Mutex/sync.Once et mauvais usage atomic)
- CGO_ENABLED=0 go test ./internal/calibration/ -run TestAtomicThresholds_ConcurrentReadsSafe -count=1 -v -> PASS (0.00s)
- CGO_ENABLED=0 go test ./internal/app/... ./internal/bridge/agentmeshkafka/... ./internal/calibration/... -count=1 -> 3 packages ok
- git status --short -> seul ?? audit/ (aucun artefact suivi regenere par les tests)
- Grep go func|sync.|atomic.|chan dans internal/**/*.go -> 8 fichiers, dont 4 production: agentmesh.go, file_adapter.go, adapter.go (interface), thresholds_atomic.go
- Grep go func|FileAdapter|Stream dans cmd/**/*.go -> aucun match (cmd ne spawn aucune goroutine)

**Outils indisponibles / repli :**
- go test -race / go build -race: INDISPONIBLE (CGO_ENABLED=0, aucun compilateur C sous Windows). Repli: analyse statique rigoureuse + go vet (analyseurs atomic, copylocks, loopclosure) + lecture de code. Consequence: tout constat sur une data race RUNTIME reste [probable]/[a verifier], non [confirme].
- uber-go/goleak: absent du depot (Grep -> seul une mention dans docs/archive). Repli: verification manuelle des chemins de close de goroutine + lecture du test TestStreamAsTauExchanges_DrainsErrsOnContextCancel qui valide deja l absence de leak par timeout.

---

## AVERTISSEMENT methodologique (lire en premier)

Le detecteur de courses `-race` est **indisponible** dans cet environnement (`CGO_ENABLED=0`, aucun compilateur C sous Windows). Tout l audit de concurrence repose donc sur **analyse statique** : `go vet` (analyseurs `atomic`, `copylocks`, `loopclosure`), lecture de code, et execution des tests sans instrumentation de course. **Consequence epistemique** : aucun constat d absence-de-data-race ne peut etre eleve a `[confirme]` sur le plan runtime — la borne haute est `[probable]`/`[a verifier]`. Le repli (vet + lecture) detecte les mauvais patrons structurels (copie de mutex, usage atomique incorrect, captures de boucle) mais pas les entrelacements runtime.

## Synthese (premiers principes)

La surface concurrente du kernel est **volontairement minimale et bien delimitee**. Le chemin de decision — `Dispatcher.Decide` et ses 8 etapes (`internal/orchestration/dispatcher.go:127-225`) — est **purement fonctionnel par valeur** : aucune goroutine lancee, aucun canal, aucun `sync.*`, aucun `atomic.*`, aucun champ mutable du `Dispatcher` ecrit pendant `Decide`. `[confirme]` (lecture integrale + `go vet` clean). Le seul etat partage lu est `d.thresholds` (valeur figee a la construction) et `d.profile` (pointeur, lu seul). La concurrence reelle se concentre dans deux fichiers de streaming hors-chemin-Decide.

## Constats

**[R3-01] MAJEUR — Perte silencieuse d erreurs dans errOut (default-drop sur buffer plein)** `[confirme]` (sur le drop) / `[probable]` (sur l impact)
- Fichier:ligne : `internal/app/agentmesh.go:96-101` et `114-118`
- Preuve (verbatim) :
  ```go
  for e := range adapterErrs {
      select {
      case errOut <- e:
      default:
      }
  }
  ```
  et plus bas :
  ```go
  case e, ok := <-adapterErrs:
      ...
      select {
      case errOut <- e:
      default:
      }
  ```
  Le canal est cree avec un buffer de 8 (`errOut := make(chan error, 8)`, ligne 82). Le `default:` jette l erreur si le consommateur n a pas draine et que les 8 slots sont pleins.
- Impact : des erreurs non-fatales de l adaptateur (lignes JSONL malformees) peuvent etre **silencieusement perdues** sous charge si le consommateur de `errc` est lent. Ce n est pas une data race ni un leak, mais une **perte d observabilite** sur un chemin qui se veut « resilient » (cf. doc `file_adapter.go:14-16`). Croise l esprit de l anti-patron #4 (observation passee sous silence) bien que celui-ci vise les `Trace.UnmodeledObservations` et non ce canal.
- Recommandation : documenter explicitement la semantique « best-effort, lossy » du canal `errc` dans le godoc de `StreamAsTauExchanges`, OU augmenter/supprimer le buffer et faire un envoi bloquant gate par `ctx.Done()` (symetrique a l envoi de `out` ligne 104-110). Trancher selon le contrat voulu : observabilite garantie vs. non-blocage du producteur.

**[R3-02] MAJEUR — SetTuning : 6 Store independants, fenetre lecture-modification-ecriture non atomique** `[confirme]` (structure) / `[probable]` (impact reel nul aujourd hui)
- Fichier:ligne : `internal/calibration/thresholds_atomic.go:82-92` (SetTuning) et `69-78` (Snapshot)
- Preuve (verbatim) :
  ```go
  func (at *AtomicThresholds) SetTuning(t Thresholds) {
      if t.Deterministe > t.Probabiliste { panic(...) }
      at.deterministe.Store(millis(t.Deterministe))
      at.probabiliste.Store(millis(t.Probabiliste))
      at.authBlock.Store(millis(t.AuthBlock))
      ... // 6 Store separes, non transactionnels
  }
  ```
  `Snapshot()` (ligne 69) lit les 6 champs via 6 `Load()` separes. Un `Snapshot` concurrent a un `SetTuning` peut donc observer un **etat mixte** : par ex. `deterministe` deja mis a jour mais `probabiliste` pas encore — violant potentiellement et transitoirement l invariant d ordre `Deterministe <= Probabiliste` que le panic-garde pretend proteger.
- Impact AUJOURD HUI : **nul**, car `SetTuning`/`Snapshot`/`AtomicThresholds` ne sont references nulle part en production (cf. R3-05). La docstring annonce « atomically updates all thresholds in one coordinated call » (ligne 80) — affirmation **survendue** : les Store individuels sont atomiques, la transaction d ensemble ne l est pas. C est un cas de statut epistemique survendu (severite MAJEUR par la grille).
- Recommandation : si ce type est un jour cable a un hot-reload (le seul scenario qui le justifierait), remplacer les 6 champs par un unique `atomic.Pointer[Thresholds]` (publication atomique d un snapshot immuable) — patron deja suggere par le commentaire « Snapshot returns ... immutable ». Sinon, corriger la docstring pour ne pas promettre une atomicite transactionnelle inexistante, ou supprimer le type (cf. R3-05).

**[R3-03] MINEUR — Decide ne verifie jamais l annulation du contexte** `[confirme]`
- Fichier:ligne : `internal/orchestration/dispatcher.go:127-225` ; propagation ctx uniquement vers `dimensions.ScoreD*` (lignes 146, 166, 170) qui le transmettent a `llm.Interpret` (`dsens.go:106`).
- Preuve : aucun `ctx.Err()` ni `select { case <-ctx.Done() }` dans `Decide` ni dans les 8 etapes. `Grep ctx.Err()|ctx.Done()` ne remonte que `agentmesh.go` et `file_adapter.go`. Le `Stub.Interpret` (`stub.go:17`) ignore son `ctx` (`_ context.Context`).
- Impact : avec le Stub deterministe (defaut), toutes les etapes sont du calcul CPU non bloquant ; un `ctx` annule/expire n interrompt pas `Decide` — il s execute jusqu au bout. Acceptable tant que le seul point bloquant potentiel (LLM reel) honore le `ctx`, ce qui depend de l implementation future (« real » non implementee, `app.go:38-42`). Pas un risque de course/deadlock.
- Recommandation : ajouter un `if err := ctx.Err(); err != nil { return tau.Decision{}, err }` en tete de `Decide` (cout negligeable, semantique claire) pour garantir l annulation cooperative quand un backend reel bloquant sera branche. `[a verifier]` que cela ne casse pas un golden test (chemin Decide deterministe attendu).

**[R3-04] MINEUR — Profile partage via pointeur non synchronise** `[confirme]` (structure) / `[a verifier]` (risque futur)
- Fichier:ligne : `internal/orchestration/dispatcher.go:32` (`profile *calibration.Profile`), lu en `161-162`, `72-79`, `206-208` ; injecte en `app.go:28-29`.
- Preuve : `d.profile` est un pointeur lu par `Decide`/`dimensionWeights` sans verrou. Le `Profile` (`profile.go:28-39`) contient des `map[string]float64` (`Weights.SensProbes`, etc.).
- Impact : **sur aujourd hui** — le profil est cree une fois (`DefaultProfile()`) et jamais mute apres injection ; lectures concurrentes d une structure immuable de facto sont sures (sans `-race` pour le prouver formellement, donc `[probable]`). **Risque futur** : si un hot-reload de profil etait introduit (mutation du `*Profile` ou des maps pendant que des `Decide` tournent), ce serait une data race classique (lecture/ecriture concurrente de map -> panic runtime ou corruption).
- Recommandation : documenter le contrat d immuabilite du `*Profile` apres injection (« must not be mutated after NewDispatcherWithProfile »), ou prevoir une publication atomique (`atomic.Pointer[Profile]`) le jour ou un reload sera requis. Lien avec R3-02 (meme patron de publication atomique de snapshot immuable).

**[R3-05] INFORMATIF — AtomicThresholds est du code mort** `[confirme]`
- Fichier:ligne : `internal/calibration/thresholds_atomic.go` (tout le fichier).
- Preuve : `Grep AtomicThresholds|SetTuning|\.Snapshot\(\)` ne remonte que `thresholds_atomic.go` et `thresholds_atomic_test.go`. Le dispatcher utilise `Thresholds` par valeur (`dispatcher.go:31`), fige a la construction (`NewDispatcher`, ligne 48). Aucun cable vers `AtomicThresholds` dans `app/`, `orchestration/`, `cmd/`.
- Impact : le type existe « en prevision » d un tuning concurrent (calque FibGo annonce ligne 9) mais n est exerce que par ses propres tests. Tant qu il n est pas cable, R3-02 est inerte. C est aussi une tension avec la directive projet « Simplicity First / pas de flexibilite non demandee » — a signaler, pas a supprimer (lecture seule, et le CLAUDE.md interdit de retirer du code non-mort sur demande).
- Recommandation : `[hypothese]` decider explicitement (ADR ou note) si ce type sert un besoin de hot-reload planifie (M5+ calibration) ou doit etre retire. Aucune action de correction dans le cadre de cet audit (lecture seule).

**[R3-06] INFORMATIF — Resolution P1-06 (drain errs sur ctx.Done) confirmee et testee ; pas de leak detectable** `[confirme]` (presence + test) / `[probable]` (absence de leak runtime)
- Fichier:ligne : `internal/app/agentmesh.go:88-92` (drain sur `ctx.Done()`), `106-109` (drain sur send-bloque annule) ; test `internal/app/agentmesh_test.go:172-218` (`TestStreamAsTauExchanges_DrainsErrsOnContextCancel`).
- Preuve : la goroutine de `StreamAsTauExchanges` (ligne 83) ferme `out` et `errOut` via `defer` (84-85) et, sur `ctx.Done()`, draine `adapterErrs` (`for range adapterErrs {}`) avant de retourner, debloquant le producteur. Le test injecte un `blockingAdapter` a canal d erreur **non bufferise** (ligne 160) et verifie par timeout 1s que les deux canaux se ferment et qu un send d erreur ne bloque pas. Cote `FileAdapter`, `Stream` enveloppe `ctx` dans un `context.WithCancel`, `defer cancel()` dans la goroutine (`file_adapter.go:46`), `Close()` idempotent via `sync.Once` (102-111) ; la boucle `scan` honore `ctx.Err()` (66) et `ctx.Done()` (83). Test concurrent `TestAtomicThresholds_ConcurrentReadsSafe` (100 goroutines) PASS.
- Impact : aucune fuite de goroutine identifiable en analyse statique. Sans `goleak` ni `-race`, la confiance sur l absence totale de leak/course runtime reste `[probable]`.
- Recommandation : `[hypothese]` ajouter `uber-go/goleak` en `TestMain` des packages `app` et `agentmeshkafka` donnerait une garantie runtime sur les leaks meme sans CGO (`goleak` ne requiert pas `-race`). Optionnel, non bloquant.

## Verdict concurrence

Le noyau de decision est concurremment sain par construction (immuabilite + zero goroutine sur `Decide`) `[confirme]` au niveau statique. Les deux MAJEUR (R3-01 perte d erreurs, R3-02 RMW non atomique) sont des dettes de robustesse/honnetete-epistemique sans impact operationnel actuel, le second etant inerte car code mort (R3-05). Recommandation transverse : adopter la publication atomique d un snapshot immuable (`atomic.Pointer`) si et seulement si un hot-reload de seuils/profil est planifie, sinon corriger les docstrings survendues. Rappel final : `-race` etant absent, ne pas conclure a « zero data race » — conclure a « aucune data race detectable par analyse statique et go vet, sous reserve d une validation `-race` ulterieure sur Linux/macOS » `[a verifier]`.
