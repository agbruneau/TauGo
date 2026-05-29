# 04 — Performance

> Audit Go multi-agents TauGo — axe « Performance & benchmarks ». HEAD `1948a7b`, v0.1.2-pre, 2026-05-29. Lecture seule, FR-CA. Sévérités : CRITIQUE 0 · MAJEUR 2 · MINEUR 2 · INFORMATIF 3.

**Conclusion (headline) :** [confirme] La performance du kernel est saine dans l absolu mais l outillage de mesure est lacunaire et le statut epistemique des debits annonces est survendu. Aucune anomalie algorithmique grave : le hot path Decide n alloue que quelques pointeurs Score, et BoundsHold (I5) est bien mono-passe comme annonce. DEUX constats MAJEUR cependant : (1) les debits fuzz annonces dans CLAUDE.md/PRD (~8.2-9.5 M exec/s pour I1-I4) sont ~6x au-dessus de la mesure solo reproductible (~1.4 M exec/s) et sont presentes sans marqueur d incertitude ni methodologie ; (2) il n existe AUCUN benchmark pour l API publique Decide, les 3 dimensions, l orchestration ni la calibration, ce qui rend la directive 5 (regression perf >5% bloquante sur tau/* et calibration/*) non verifiable. BoundsHold alloue ~2x la memoire d Aggregate (per-layer local map) sans baseline pour qualifier de regression.

**Outils exécutés :**
- go version -> go1.26.3 windows/amd64 ; git rev-parse HEAD -> 1948a7b ; git status --short -> propre
- grep 'func Benchmark' (Grep) -> seulement BenchmarkI5_Aggregate et BenchmarkI5_BoundsHold (i5_composition_test.go:126,135) dans tout le depot
- grep 'func Fuzz' (Grep) -> 5 cibles FuzzI1..I5 dans fuzz_targets_test.go
- CGO_ENABLED=0 go test -bench=. -benchmem -run=^$ -count=3 ./internal/tau/... -> I5_Aggregate ~15.2k ns/op 15560 B/op 12 allocs ; I5_BoundsHold ~14.5k ns/op 31576 B/op 41 allocs (3 runs stables)
- CGO_ENABLED=0 go test -bench=. -benchmem -cpuprofile -memprofile -run=^$ ./internal/tau/invariants/ -> profils generes (single package requis ; multi-package rejete)
- go tool pprof -top cpu.prof -> runtime.mapassign_faststr 54.46% cum (point chaud = insertions map)
- go tool pprof -top -sample_index=alloc_space mem.prof -> BoundsHold 2.44GB (67%) vs Aggregate 1.18GB (33%) cumulatif
- go tool pprof -list=BoundsHold -sample_index=alloc_objects mem.prof -> ligne 59 (local map par couche) = 2.16M/3.08M objets (70%)
- CGO_ENABLED=0 go test -fuzz=FuzzI1..I5 -fuzztime=10s ./internal/tau/invariants/ -> I1~1.4M/s, I2~1.4M/s, I3~1.4M/s, I4~1.6M/s, I5~1.1M/s exec/s ; 0 crash
- Read dispatcher.go (Decide hot path) -> aucune map ni slice par appel hors violation ; seulement 3 *tau.Score alloues
- Read fuzz-summary.md L113,136 + AUDIT-v0.1.0-to-v0.1.1.md L84-88 -> origine methodologique des debits annonces (5s smoke, >=8M entrees)
- rm audit/*.prof + git status --short -> arbre propre, aucun artefact laisse, aucun fichier suivi modifie

**Outils indisponibles / repli :**
- -race : indisponible (CGO_ENABLED=0, aucun compilateur C) -> non execute ; impossibilite de detecter data race par tooling confirmee [a verifier sur Linux/macOS avec CGO]
- make benchmark : make absent (ADR-0010 pure-local) -> repli go test -bench direct
- baseline perf v0.1.0 figee : absente du sandbox -> deltas v0.1.1 (-46% ns/op BoundsHold) non verifiables [a verifier] ; chiffres actuels rapportes comme reference reproductible

---

## Conclusion (pyramide inversee)

[confirme] Aucune anomalie algorithmique grave cote performance : le hot path `Decide` est econome (quelques pointeurs `*tau.Score`, aucune `map`/`slice` par appel hors violation d invariant), et `BoundsHold` (I5) est bien mono-passe comme le promet la v0.1.1. **Zero CRITIQUE.** Les deux problemes reels sont d ordre *epistemique* et *outillage*, classes MAJEUR : des debits fuzz survendus dans la doc canonique, et l absence totale de benchmark sur l API publique et la calibration — ce qui rend la directive « regression >5% bloquante » litteralement non verifiable. Sans baseline figee dans le sandbox, les chiffres ci-dessous valent comme **reference reproductible** (Intel Core Ultra 9 275HX, 24 vCPU, Windows 11, Go 1.26.3, `CGO_ENABLED=0`, run solo).

Note machine : poste partage (24 coeurs, Windows). Le run etant solo, les mesures sont fiables a l ordre de grandeur ; la variabilite inter-run observee est < 6 % sur les benchmarks I5 (3 repetitions).

---

## Constats

**[P4-01] MAJEUR — Debits fuzz annonces ~6x au-dessus de la mesure reproductible, sans marqueur ni methodologie** *(statut epistemique survendu)* — [confirme]
- Fichier:ligne : `CLAUDE.md:138-142` (table invariants, colonne « Debit ») ; `PRD.md:103` (« debits 1.1 M a 9.5 M exec/s »).
- Preuve (annonce) verbatim : `| I1 | … | ~8.6 M exec/s |`, `| I2 | … | ~8.6 M exec/s |`, `| I3 | … | ~8.2 M exec/s |`, `| I4 | … | ~9.5 M exec/s |`, `| I5 | … | ~1.1 M exec/s |`.
- Preuve (mesure solo, `go test -fuzz=… -fuzztime=10s ./internal/tau/invariants/`) verbatim :
  - `FuzzI1 : execs: 15050707 (… ~1.36-1.60M/sec)`
  - `FuzzI2 : execs: 14252720 (… ~1.27-1.46M/sec)`
  - `FuzzI3 : execs: 14236553 (… ~1.29-1.48M/sec)`
  - `FuzzI4 : execs: 16044029 (… ~1.49-1.65M/sec)`
  - `FuzzI5 : execs: 11830821 (… ~1.06-1.24M/sec)`
- Analyse : I5 (~1.1 M/s) **correspond** a l annonce ; I1-I4 sont **~6x sous** l annonce (8.2-9.5 M vs ~1.4 M mesure). L origine est documentee : `docs/empirical/fuzz-summary.md:113` indique que les ~8M sont mesures « sur leurs corpus respectifs (5 s, ≥ 8M entrees chacun) » — c est-a-dire *entrees-corpus / temps* d un smoke run sur un autre hote, **pas** le debit du moteur `go test -fuzz` (instrumentation de couverture + coordination 24 workers + mutation). L audit archive (`docs/archive/audits/2026-05-24-AUDIT-v0.1.0-to-v0.1.1.md:88`) chiffre d ailleurs I5 a `701 K exec/s`, ensuite arrondi a ~1.1 M dans `CLAUDE.md`. Le commentaire source `i5_composition.go:88` confirme la confusion : « roughly 700K executions/s vs 8M for scalar fuzz targets » — les ~8M referent au debit de la *fonction propriete scalaire*, pas au moteur fuzz.
- Impact : la table conflate deux metriques sous un seul label « exec/s ». Une affirmation chiffree, datee et evolutive est presentee **sans marqueur d incertitude** (viole conventions editoriales CLAUDE.md « Marqueurs d incertitude obligatoires »). Un lecteur architecte conclut a tort que I1-I4 explorent 6x plus vite que la realite mesurable, faussant le dimensionnement des fenetres `-fuzztime`.
- Recommandation : (compromis principal) annoter la colonne d un marqueur `[a verifier]` + preciser la methodologie (« debit fonction-propriete scalaire isolee, hote X » vs « debit moteur fuzz »). (Alternative) remplacer par le debit moteur reproductible (~1.1-1.6 M/s) avec hote et `GOMAXPROCS` notes. (Conditions de retournement) si une mesure scalaire dediee (boucle `b.N` directe sur `EvaluateI1`) confirme ~8M sur ce poste, conserver les chiffres mais reetiqueter explicitement la metrique.

**[P4-02] MAJEUR — Aucun benchmark sur Decide, dimensions, orchestration ni calibration -> directive perf non verifiable** — [confirme]
- Fichier:ligne : seuls `internal/tau/invariants/i5_composition_test.go:126` (`BenchmarkI5_Aggregate`) et `:135` (`BenchmarkI5_BoundsHold`) existent.
- Preuve verbatim : `grep -rn "func Benchmark" ./internal ./cmd ./test` -> exactement 2 resultats, tous deux I5. Aucun `Benchmark` pour `Dispatcher.Decide`, `dimensions.ScoreDSens/DAuthority/DInvariant`, `calibration.*`.
- Impact : `CLAUDE.md` directive 5 (« modifs dans tau/* ou calibration/* : make benchmark avant + apres. Regression > 5 % = blocage ») et directive 5 §Performance critique reposent sur une mesure qui **n existe pas** pour le chemin de sortie unique (`Decide`) ni pour la calibration. La garde anti-regression est de jure mais non de facto. Le seul code benchmarke (I5 `BoundsHold`/`Aggregate`) n est meme pas sur le chemin nominal (`EvaluateI5` retourne `Held` par construction en V1).
- Recommandation : (compromis principal) ajouter `BenchmarkDecide` couvrant les 3 issues (Deterministe/Probabiliste/Refus) avec stub LLM deterministe, plus `BenchmarkScoreD*` et un benchmark de (de)serialisation `calibration.Profile`. (Alternative minimale si scope serre) au moins `BenchmarkDecide` sur un `Exchange` nominal pour ancrer une baseline. (Mesurable) cible : capturer ns/op + allocs/op et les figer comme reference v0.1.2 dans `docs/empirical/`. (Conditions de retournement) si le projet acte que la perf n est pas un objectif V1, retirer la directive 5 plutot que la laisser non verifiable.

**[P4-03] MINEUR — BoundsHold alloue ~2x la memoire d Aggregate via une local map par couche** — [confirme] (regression : [a verifier], pas de baseline)
- Fichier:ligne : `internal/tau/invariants/i5_composition.go:59` (`local := make(map[string]struct{}, len(layer))`).
- Preuve (benchmark, 3 runs) verbatim : `BenchmarkI5_Aggregate-24  15560 B/op  12 allocs/op` vs `BenchmarkI5_BoundsHold-24  31576 B/op  41 allocs/op`. Profil alloc cumulatif : `BoundsHold 2.44GB (67.33%)` vs `Aggregate 1.18GB (32.49%)`. Attribution ligne (`pprof -list=BoundsHold -sample_index=alloc_objects`) : ligne 59 = `2156819 / 3075519` objets ≈ **70 %** des allocations de la fonction.
- Impact : sur une pile 10×50, `BoundsHold` cree 11 maps (10 `local` + 1 `global`) la ou `Aggregate` n en cree qu une. Le surcout est ~16 KB et 29 allocs/op supplementaires. C est l implementation *mono-passe annoncee* (confirmee : aucune 2e traversee, aucun appel a `Aggregate`), donc l optimisation v0.1.1 tient pour le *temps* (ns/op quasi a parite avec Aggregate) mais le cout *memoire* reste superieur a Aggregate par construction. **Non bloquant** : code hors chemin nominal (`EvaluateI5` = `Held`), exerce seulement par le fuzz.
- Recommandation : (compromis principal) si la perte memoire devient sensible en V2 (quand la pile sera reifiee dans `Trace`), remplacer la `local` map par un comptage de distincts via le `global` : memoriser `len(global)` avant/apres chaque couche pour deriver le nombre d entrees nouvelles, et calculer la cardinalite distincte de la couche par insertion conditionnelle — supprime N allocations de map. (Alternative) reutiliser une seule `local` map et la `clear()` entre couches (Go 1.21+) : 1 alloc au lieu de N. (Mesurable) attendre `BenchmarkI5_BoundsHold` ≈ `15560 B/op` apres. (Conditions de retournement) ne rien toucher tant que I5 n est pas sur le chemin nominal — gold-plating sinon (cf. Simplicity First).

**[P4-04] MINEUR — Commentaire de benchmark perime contredit la source** — [confirme]
- Fichier:ligne : `internal/tau/invariants/i5_composition_test.go:134`.
- Preuve verbatim : `// BenchmarkI5_BoundsHold measures BoundsHold (includes Aggregate) on the same pile.` — contredit par `i5_composition.go:49-50` : « No second traversal and no call to Aggregate — bounds and union are co-computed. »
- Impact : commentaire trompeur, vestige pre-v0.1.1 (avant l optimisation mono-passe). Induit en erreur sur ce que mesure le benchmark.
- Recommandation : corriger en « measures the single-pass BoundsHold (no call to Aggregate) ». Hors perimetre lecture-seule de cet audit ; signale a l orchestrateur.

**[P4-05] INFORMATIF — Hot path Decide propre ; unique point chaud localise a I5** — [confirme]
- Fichier:ligne : `internal/orchestration/dispatcher.go:127-225`.
- Preuve : revue ligne a ligne — aucune `map`, aucune `slice` allouee par appel hors branche violation (`UnmodeledObservations` append seulement si `AnyViolated()`). Seules allocations : 3 `&tau.Score{}` (lignes 150, 174, 175). Profil CPU global (package invariants) : `runtime.mapassign_faststr 54.46% cum` — entierement imputable a I5 `Aggregate`/`BoundsHold`, seul code benchmarke ; non representatif du chemin `Decide`.
- Impact : positif. Le chemin de decision n a pas de complexite cachee detectee.
- Recommandation : aucune action ; confirmer via le `BenchmarkDecide` recommande en P4-02.

**[P4-06] INFORMATIF — Delta perf v0.1.1 BoundsHold non verifiable (pas de baseline)** — [a verifier]
- Fichier:ligne : annonce `CLAUDE.md:142` (« -46 % ns/op ») et `i5_composition.go:48-50`.
- Preuve : aucune baseline v0.1.0 dans le sandbox (HEAD = 1948a7b unique). Chiffres ACTUELS reproductibles fournis comme reference : `BoundsHold ~14.2-14.9k ns/op, 31576 B/op, 41 allocs/op` ; `Aggregate ~15.2-15.9k ns/op, 15560 B/op, 12 allocs/op`. L implementation actuelle EST mono-passe (confirme, P4-03), coherente avec l intention « 1 passe au lieu de 2 ».
- Impact : le gain temps -46 % est plausible et l implementation est saine, mais le pourcentage exact ne peut etre confirme ici.
- Recommandation : si verification souhaitee, `git stash`/checkout du tag `v0.1.0` puis `benchstat` avant/apres ; sinon archiver les chiffres actuels comme nouvelle reference.

**[P4-07] INFORMATIF — Drift documentaire mineur sur le debit I5 (701K -> ~1.1M)** — [confirme]
- Fichier:ligne : `docs/archive/audits/2026-05-24-AUDIT-v0.1.0-to-v0.1.1.md:88` (`701 K exec/s`) vs `CLAUDE.md:142` (`~1.1 M exec/s`).
- Preuve : re-mesure solo I5 = `11830821 execs / 10s ≈ 1.06-1.24M/sec` — confirme l ordre ~1.1M, donc l arrondi CLAUDE.md est defendable, mais s ecarte du 701K source. Le commentaire `i5_composition.go:88` (« roughly 700K ») reste, lui, sur l ancienne valeur.
- Impact : incoherence interne benigne entre trois documents.
- Recommandation : aligner sur la valeur re-mesuree (~1.1 M/s) avec marqueur, ou conserver 701K avec la mention « smoke 5 s ».

---

## Note de propriete / hygiene sandbox

[confirme] Lecture seule respectee. Profils `cpu.prof`/`mem.prof` ecrits sous `audit/` puis supprimes. La campagne fuzz n a ajoute aucun fichier au corpus suivi : `git status --short` final = vide (exit 0). Aucun `git restore` requis. `-race` non execute (CGO indisponible) — la detection de data race par outillage reste **[a verifier]** sur Linux/macOS avec CGO.
