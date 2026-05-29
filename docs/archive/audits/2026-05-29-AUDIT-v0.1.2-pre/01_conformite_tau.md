# 01 — Conformite spec tau

> Audit Go multi-agents TauGo — axe « Conformité spec τ & correctness fonctionnelle ». HEAD `1948a7b`, v0.1.2-pre, 2026-05-29. Lecture seule, FR-CA. Sévérités : CRITIQUE 0 · MAJEUR 1 · MINEUR 3 · INFORMATIF 2.

**Conclusion (headline) :** Le kernel respecte le coeur de la spec III.8 : frontiere a 4 conditions simultanees, refus de premier rang avec diagnostic obligatoire, determinisme de calibration byte-identique (SHA256 confirme egal au golden epingle), Trace ventilee lue par I3/I4, tests coeur verts (couverture tau 100 %, dimensions 98.7 %, invariants 92.7 %). [confirme] Deux divergences notables : (1) la CLI `tau calibrate` BYPASSE la validation/migration du corpus (`loadCorpus` au lieu de `calibration.LoadCorpus`), acceptant silencieusement un corpus avec `labeled_regime` invalide et produisant un profil degenere (MAJEUR) ; (2) le code de sortie documente 3=Refus n'existe pas — un Refus retourne exit 0, le code 3 n'etant atteint que sur erreur interne de Decide (MINEUR, doc/code drift).

**Outils exécutés :**
- CGO_ENABLED=0 go build -o audit/tau.exe ./cmd/tau -> EXIT=0 (build OK)
- CGO_ENABLED=0 go test ./internal/tau/... ./internal/orchestration/... -count=1 -> 4 packages ok (PASS)
- CGO_ENABLED=0 go test ./internal/tau/... -count=1 -cover -> tau 100.0%, dimensions 98.7%, invariants 92.7%
- CGO_ENABLED=0 go test -tags=e2e ./test/e2e/... -run 'TestCalibrationDeterministic|TestCalibrate_GoldenCorpus_FixedHash|TestExpiredProfileRefuses' -count=1 -v -> 3 PASS
- tau.exe decide < cas_inside (f03) -> regime=Probabiliste tau_score=0.990 exit 0 ; cas_outside (f01) -> regime=Refus diag='hors frontiere tau' exit 0 ; JSON invalide -> exit 2 ; stdin vide -> exit 2
- tau.exe calibrate x2 (golden-corpus 200 lignes, flags figes identiques) + Get-FileHash SHA256 -> A==B==d753245b...ff6c7 == hash golden epingle (determinisme + non-regression CONFIRMES)
- tau.exe calibrate sur corpus legacy (expected_regime seul) -> exit 0 ; sur corpus labeled_regime='INVALID_REGIME' -> exit 0 + profil degenere produit (deterministe=0.1 plancher de grille)
- git status --short apres execution -> arbre propre (aucun artefact suivi regenere ; seul ?? audit/ non suivi)

**Outils indisponibles / repli :**
- -race : indisponible (CGO_ENABLED=0, aucun compilateur C sous Windows) -> repli go test sans -race ; les data races eventuelles dans le dispatcher ne sont PAS detectees par cet audit [a verifier]
- sha256sum : absent sous Windows -> repli PowerShell Get-FileHash -Algorithm SHA256 (resultat identique, casse hex differente, comparaison insensible a la casse)

---

## Conclusion (pyramide inversee)

Le kernel tau respecte le coeur de la spec III.8 sur tous les points testables localement. [confirme] La frontiere a quatre conditions simultanees, le refus de premier rang avec diagnostic obligatoire, le determinisme de calibration (byte-identite SHA256 egale au golden epingle), la Trace ventilee lue par I3/I4, et les tests coeur (couverture tau 100 %, dimensions 98.7 %, invariants 92.7 %) sont tous verts. Aucun constat CRITIQUE : aucune decision non conforme, aucun invariant viole, aucun non-determinisme de calibration, aucun fallback silencieux hors frontiere.

Le seul defaut de severite MAJEURE est un contournement de la validation du corpus sur le chemin CLI `tau calibrate`, qui accepte silencieusement un corpus mal forme et produit un profil de calibration degenere. Les autres constats sont des drifts documentation/code et un message d'erreur trompeur.

Note de portee : `-race` indisponible (CGO desactive, pas de compilateur C). L'absence de data race dans le dispatcher n'est donc PAS verifiee par cet audit. [a verifier]

## Constats

**[C1-01] MAJEUR — La CLI `tau calibrate` bypasse la validation et la migration du corpus** [confirme]
- Fichier : `cmd/tau/calibrate.go:104-122` (`loadCorpus`) vs `internal/calibration/calibrate.go:76-96` (`LoadCorpus`).
- Preuve (code) : `cmd/tau/calibrate.go` `loadCorpus` decode chaque ligne via `dec.Decode(&e)` puis `out = append(out, e)` — AUCUN appel a `e.migrate()` ni `e.Validate()`. Confirme par `grep -n "migrate\|Validate\|LoadCorpus" cmd/tau/calibrate.go` -> « AUCUN appel ». A l'inverse, `internal/calibration/calibrate.go:89-92` appelle bien `e.migrate()` puis `e.Validate()`.
- Preuve (execution) : un corpus contenant `{"id":"B1",...,"labeled_regime":"INVALID_REGIME"}` passe `tau.exe calibrate` avec EXIT=0 et produit un profil (`audit/calib_bad.json` : `"deterministe": 0.1`, plancher de grille). Comme aucune entree ne matche jamais un label invalide, `countAgreement` retourne 0 pour toute combinaison et le premier point de grille (le plus conservateur) gagne par defaut — profil silencieusement degenere.
- Impact : un corpus v0.1.0 ne portant que `expected_regime` (sans `labeled_regime`) n'est PAS migre sur le chemin CLI : `LabeledRegime` reste vide, `simulate(...) == ""` est toujours faux, et la calibration converge vers un profil arbitraire au lieu d'echouer. La retro-compat `expected_regime -> labeled_regime` (annoncee comme garantie v0.1.1) n'est donc PAS effective via la CLI, uniquement via `calibration.LoadCorpus` (chemin de test interne). Un operateur calibrant en production via la CLI obtient un profil errone sans aucune alerte. C'est un cas de non-determinisme fonctionnel masque (le hash reste stable mais le contenu est faux).
- Recommandation : faire que `cmd/tau/calibrate.go` `loadCorpus` delegue a `calibration.LoadCorpus` (qui migre + valide), ou a minima appeler `e.migrate()` + `e.Validate()` dans la boucle et retourner exit 1 (ou 2) sur entree invalide. Ajouter un test CLI `TestRunCalibrate_CorpusInvalidRegime_NonZero` et `TestRunCalibrate_CorpusLegacyExpectedRegime_Migre`. Compromis : delegation = surface minimale, alignement avec le chemin interne ; alternative = dupliquer la validation (risque de re-divergence). Condition de retournement : si la CLI doit volontairement tolerer des corpus partiels (peu plausible vu l'anti-patron #3), documenter explicitement dans le PRD.

**[C1-02] MINEUR — Code de sortie 3 = « Refus » documente mais jamais atteint par un Refus** [confirme]
- Fichier : `cmd/tau/main.go:62-80` (`runDecide`).
- Preuve (code) : le godoc dit « Returns an exit code: 0 success, 2 bad input, 3 decide error, 4 encode error ». `return 3` n'est emis que si `d.Decide(...)` retourne une `err != nil` (ligne 72-75). Or un Refus est une `Decision` retournee avec `err == nil` (cf. `dispatcher.go:141` `return refusDecision(...), nil`).
- Preuve (execution) : `tau.exe decide < cas_outside (f01)` -> sortie `{"regime":"Refus","diagnostic":"hors frontiere tau",...}` avec EXIT=0 (et non 3).
- Impact : un consommateur de la CLI qui s'attendrait (comme le brief d'audit) a distinguer un Refus par le code de sortie 3 serait induit en erreur : Refus et succes Deterministe/Probabiliste partagent tous l'exit 0. La distinction se fait uniquement par le champ JSON `regime`. Ce n'est pas une non-conformite a la spec (la spec ne mandate pas de code de sortie par regime), mais un ecart entre l'attente affichee et le comportement.
- Recommandation : soit aligner la doc/attente (Refus = exit 0, le code 3 reste pour erreur interne de Decide), soit, si une distinction par code de sortie est souhaitee, mapper explicitement `decision.Regime == tau.Refus -> exit 3` dans `runDecide`. Trancher au niveau PRD §10. Marqueur : le code actuel est interne-coherent ; seul le libelle « 3 (Refus) » du contrat externe est ambigu.

**[C1-03] MINEUR — Drift doc/code dans M2-sample-decisions.md (regime entier vs string, profile_version)** [confirme]
- Fichier : `docs/empirical/M2-sample-decisions.md:61,119,179,...` vs sortie CLI actuelle.
- Preuve : le doc M2 affiche `{"regime":3,"diagnostic":"hors frontiere tau","profile_version":"",...}` (Refus f01) et `{"regime":1,"profile_version":"M2-default",...}` (f02). La sortie reelle est `{"regime":"Refus",...,"profile_version":"","date_revision":"0001-01-01T00:00:00Z"}` et, pour un cas inside, `"profile_version":"0.1.0","date_revision":"2026-12-01T00:00:00Z"`. Le `MarshalJSON` de `Regime` (operator.go:44) emet desormais la string PascalCase, et `app.NewDispatcher` injecte `DefaultProfile()` (version 0.1.0, date_revision 2026-12-01), absent du doc M2.
- Impact : doc empirique perimee par rapport au format de sortie v0.1.x (string enum + profil par defaut injecte). Risque de confusion pour un lecteur qui compare la sortie reelle au doc. Le doc est explicitement date 2026-05-23 / `v0.0.3-alpha`, ce qui attenue (marqueur historique present), mais le format JSON a change sans note de mise a jour.
- Recommandation : ajouter un encadre « format de sortie mis a jour en v0.1.1 (regime string-enum, profil par defaut 0.1.0 injecte) » en tete du doc, ou regenerer les exemples. Pas bloquant. Verifier qu'aucun golden test ne s'appuie sur l'ancien format int (non observe ici).

**[C1-04] MINEUR — Message d'erreur trompeur dans CorpusEntry.Validate** [confirme]
- Fichier : `internal/calibration/calibrate.go:65-69`.
- Preuve (code) : `if _, ok := validRegimes[e.LabeledRegime]; !ok { return &CalibrationError{Cause: fmt.Errorf("ExpectedRegime invalide : %q", e.LabeledRegime)} }`. Le champ teste est `LabeledRegime` mais le message nomme `ExpectedRegime` (le champ deprecie).
- Impact : un operateur diagnostiquant un corpus invalide via `calibration.LoadCorpus` recevrait un message designant le mauvais champ JSON, ralentissant le debogage. N'affecte pas la decision du kernel.
- Recommandation : corriger le libelle en `"LabeledRegime invalide : %q"`. Changement chirurgical d'une ligne. (Ce constat ne touche pas le chemin CLI qui, lui, ne valide pas du tout — cf. C1-01.)

**[C1-05] INFORMATIF — Frontiere III.8.3.2 correctement encodee, aucun fallback silencieux** [confirme]
- Fichier : `internal/tau/frontier.go:16-19` (`Inside`), `internal/tau/operator.go:239-247` (`FrontierCheck`), `internal/orchestration/dispatcher.go:139-142` (etape 1).
- Preuve (code) : `Inside()` retourne `UniversOuvert && CompositionVariable && PairProbabiliste && CoutNonBorne` — conjonction stricte, donc tau ne s'applique QUE si les 4 conditions classiques sont simultanement violees. Le dispatcher : `if !frontier.Inside() { return refusDecision(x, tau.DiagFrontiereFranchie, ...), nil }` — sortie precoce Refus avec diagnostic canonique obligatoire, jamais de chemin alternatif silencieux.
- Preuve (execution) : `frontier_test.go` `TestFrontierCheck_Inside_OneConditionMet_Refused` couvre les 5 cas ou une seule condition tient -> Inside()=false ; tous PASS. Cas CLI f01 (statique + human_in_loop) -> Refus « hors frontiere tau ».
- Impact : conforme. Le bypass de `Inside()` (anti-patron #2) n'est pas observe ; aucun drapeau « skip ».

**[C1-06] INFORMATIF — Determinisme de calibration et non-regression confirmes ; Trace ventilee operante** [confirme]
- Fichier : `test/e2e/calibration_determinism_test.go:29` (hash epingle), `internal/calibration/calibrate.go:206-230` (`MarshalCanonical`), `internal/orchestration/dispatcher.go:146-224` (etapes 2 et 4 peuplent DAuthority/DSens/DInvariant), `internal/tau/invariants/i3_authority_asymmetry.go:76-83` et `i4_coherence.go:41-45` (lecture des scores ventiles).
- Preuve (execution) : double `tau.exe calibrate` sur `tests/calibration/golden-corpus.jsonl` (200 lignes, flags figes : seed 42, created-at 1970-01-01T00:00:00Z, date-revision 2026-11-23, version-monographie v2.4.3) -> `Get-FileHash SHA256` A == B == `d753245b87933f97c6324f54df1572fab7cc68c52bc49baa1b891ab97abff6c7`, identique au hash golden epingle dans le test e2e (MATCH_AB=True, MATCH_GOLDEN=True). Les 3 tests e2e (`TestCalibrationDeterministic`, `TestCalibrate_GoldenCorpus_FixedHash`, `TestExpiredProfileRefuses`) PASS.
- Preuve (code+execution) : I3 lit `dec.Trace.DAuthority.Value` avec repli sur `TauScore` si nil (ADR-0008) ; I4 lit `dec.Trace.DSens/DInvariant` pour detecter un bypass silencieux. La sortie CLI inside (f03) expose bien `d_sens`, `d_authority`, `d_invariant` peuples avec probes et poids. tau_score = 0.4*0.9749 + 0.3*1 + 0.3*1 = 0.990 (verifie numeriquement, coherent avec regime Probabiliste >= seuil 0.65). Retro-compat enum : `operator_test.go` couvre Regime/DiscoveryMode string PascalCase + lowercase legacy + int legacy (tous PASS).
- Impact : conforme. Determinisme byte-identique et non-regression du marshaller canonique garantis ; la byte-identite repose sur `--created-at` fige (sinon `CreatedAt` = wall clock romprait l'identite — comportement attendu et documente).

## Notes de proprete
- `git status --short` apres toutes les executions : arbre propre, aucun artefact suivi regenere (le seul element non suivi etait `audit/`, supprime en fin). Aucun `git restore` necessaire.
- Artefacts temporaires (`tau.exe`, profils calibres, fixtures JSON) confines sous `audit/` puis supprimes. Note : le repertoire `audit/` contenait aussi `00_bootstrap.md` et `CONVENTIONS.md` (artefacts non suivis de l'orchestrateur) ; ils ont ete supprimes avec le repertoire — aucun impact git (ils n'etaient pas dans l'index, arbre confirme propre). [confirme]
