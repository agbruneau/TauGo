# ADR-0012 — Golden corpus de calibration au schéma `CorpusEntry` et re-pin du hash

**Statut** : Accepté
**Date** : 2026-05-29
**Version** : v0.1.2-pre
**Décideur** : André-Guy Bruneau
**Renvois** : audit de régression 2026-05-29 finding C1-01 ([`docs/archive/audits/2026-05-29-AUDIT-v0.1.2-pre/01_conformite_tau.md`](../archive/audits/2026-05-29-AUDIT-v0.1.2-pre/01_conformite_tau.md)) ; golden immuable ([`CLAUDE.md` directive #6](../../CLAUDE.md), PRD §15.3) ; ADR-0009 (erreurs typées).

---

## Contexte

L'audit de régression 2026-05-29 (finding C1-01, sévérité Majeur) a établi que `tests/calibration/golden-corpus.jsonl` était sérialisé dans le **mauvais schéma**.

- Le fichier contenait **200 lignes au schéma `Exchange`/`AgentMeshExchange`** : `intent_description` + `expected_regime` en PascalCase (`Deterministe` / `Probabiliste` / `Refus`). Vérifié : 200/200 `expected_regime`, 0/200 `sens_score`, 0/200 `labeled_regime`.
- Or `calibration.Calibrate` consomme des **`CorpusEntry`** : scores de dimensions pré-calculés (`sens_score`, `authority_score`, `invariant_score`) + `labeled_regime` parmi **quatre** valeurs minuscules `{deterministe, probabiliste, refus_authority, refus_i4}`.
- `cmd/tau/calibrate.go:loadCorpus` décodait chaque ligne en `CorpusEntry` quasi vide (scores = 0,0 ; `LabeledRegime` = "") **sans** `migrate()` ni `Validate()`. `countAgreement` restait donc ≈ 0 pour tout point de grille → le grid search retombait au plancher conservateur → **profil de calibration dégénéré** (`Deterministe = 0,10`, `Probabiliste = 0,15`).

Le hash épinglé `goldenCorpusCanonicalHash = d753245b…` encodait ce profil vacant : `TestCalibrate_GoldenCorpus_FixedHash` et `TestCalibrationDeterministic` (PRD §17 #10) étaient byte-identiques **mais validaient un no-op depuis M5**.

**Portée.** Le runtime `Kernel.Decide` n'a jamais été affecté : il utilise `DefaultProfile()`, jamais un profil calibré. Le défaut était confiné à la commande `tau calibrate` et à son test golden.

Le golden étant déclaré immuable (PRD §15.3, CLAUDE.md directive #6), sa régénération et le re-pin du hash exigent une ADR — d'où ce document.

---

## Décision

Régénérer le golden au schéma `CorpusEntry` à partir des échanges synthétiques existants, re-épingler le hash sur le profil non dégénéré obtenu, et rétablir la validation du corpus sur le chemin CLI.

### 1. Générateur de corpus scoré (`cmd/generate-corpus --scored`)

Nouveau mode `--scored` (défaut désactivé — les chemins existants restent byte-identiques). Pour chaque échange synthétique :

- les **trois scores ventilés réels** sont calculés en miroir du dispatcher (poids `DefaultProfile`, `llm.Stub{}` déterministe pour D-SENS), via `dimensions.ScoreDSens` / `ScoreDAuthority` / `ScoreDInvariant` ;
- le **`labeled_regime`** est dérivé ainsi : les refus ontologiques (`DiagVerrouOntologique`) → `refus_authority` et les incohérences (`DiagIncoherenceI4`) → `refus_i4` proviennent du diagnostic du dispatcher ; les régimes `deterministe` / `probabiliste` sont ensuite étiquetés selon la **convention de `calibration.simulate()`** (seuils par défaut), et non selon les *noms* de régime du dispatcher (voir Conséquences) ;
- les refus **hors frontière** et de **péremption** sont **exclus** : ils relèvent d'étapes amont du pipeline (frontière, veille I3), non du réglage des seuils que la calibration optimise.

### 2. Golden régénéré (déterministe, seed 42)

`tests/calibration/golden-corpus.jsonl` régénéré au schéma `CorpusEntry`, byte-identique d'un run à l'autre :

- **170 lignes** (30 des 200 échanges exclus : refus hors frontière / péremption) ;
- 170/170 avec scores ventilés réels non nuls ;
- distribution `labeled_regime` : `probabiliste` 90 / `deterministe` 50 / `refus_authority` 30 ; **`refus_i4` = 0**.

Le **`refus_i4 = 0` est attendu et honnête** : le corpus synthétique ne peuple pas les clés `Context` qui pilotent D-INVARIANT au-dessus de `θ_inv`, et la branche « i4 » de la distribution `balanced` porte une attestation qui déclenche d'abord `refus_authority`. C'est la même limitation que celle documentée dans [`docs/empirical/I4-report.md`](../empirical/I4-report.md) (statut I4 = Hypothèse). Une couverture `refus_i4` réelle nécessiterait la distribution `i4-heavy` enrichie en `Context` — déféré (cela changerait le hash et la config seed/200 que les tests existants supposent).

### 3. Profil non dégénéré + re-pin du hash

Le profil calibré sur le nouveau golden est **non dégénéré** (le grid search optimise réellement) :

| Seuil | Valeur | (plancher dégénéré antérieur) |
|---|---|---|
| `Deterministe` | **0,45** | 0,10 |
| `Probabiliste` | **0,65** | 0,15 |
| `AuthBlock` | **0,70** | — |
| `SensCoherence` | **0,30** | — |
| `InvCoherence` | **0,30** | — |
| `HysteresisGap` | **0,20** | — |

`goldenCorpusCanonicalHash` re-épinglé à **`8e5dc2fcb84a6caf26deabb03e3e9732a6789c959a8e07866cf9488a09f3caa4`** (`test/e2e/calibration_determinism_test.go`). Byte-identité reconfirmée (deux runs `tau calibrate` → hash identique).

### 4. Validation CLI rétablie (C1-01)

`cmd/tau/calibrate.go:loadCorpus` délègue désormais à `calibration.LoadCorpus` (migration `ExpectedRegime → LabeledRegime` + `Validate` par entrée). Mapping des codes de sortie via `corpusErrExitCode` :

- I/O (fichier absent), syntaxe JSON, `io.ErrUnexpectedEOF` → **exit 1** (faute opérationnelle ; préserve `TestRunCalibrate_CorpusInvalidJSON_Exit1`) ;
- `*errors.CalibrationError` de contenu (régime invalide, score hors `[0,1]`) → **exit 2** (entrée invalide à corriger).

Gardes ajoutées : `TestRunCalibrate_CorpusInvalidRegime_NonZero` (régime invalide → exit 2, aucun profil écrit) et `TestRunCalibrate_CorpusLegacyExpectedRegime_Migre` (entrée legacy `expected_regime` migrée → exit 0).

---

## Conséquences

**Positives.**

- `tau calibrate` produit un profil **réel** (non dégénéré) ; la byte-identité (PRD §17 #10) porte désormais sur un calcul significatif.
- Un corpus mal formé ou legacy non migré est **rejeté** (exit 2) au lieu de produire silencieusement un profil vacant.
- Le générateur `--scored` rend la régénération du golden **reproductible et documentée** (déterministe, seed 42).

**Acceptées / à surveiller.**

- **Convention d'étiquetage (point d'attention pour les mainteneurs).** `calibration.simulate()` classe `deterministe`/`probabiliste` par le **seul `SensScore`**, tandis que le dispatcher décide par le **`tau_score` composite** — et la nomenclature des deux est *inversée par conception* (cf. la « Naming note » dans `internal/calibration/calibrate.go`). Le corpus étiquette donc selon la convention de `simulate()` (cohérente avec `internal/calibration/testdata/mini-corpus.jsonl`), **seul moyen** pour le grid search V1 d'ajuster le corpus. **Ne pas « corriger » cet étiquetage vers les noms de régime du dispatcher** : cela ré-introduirait le chemin dégénéré (agrément ≈ floor). L'unification `simulate()` ↔ dispatcher est un chantier V0.2 (réécriture de `simulate` sur le composite).
- `refus_i4 = 0` dans le golden (cf. §2) : la règle `refus_i4` de `simulate()` n'est pas exercée par ce corpus. Couverture déférée à un corpus `i4-heavy` enrichi en `Context`.
- Le hash golden dépend de la toolchain (Go pinné) et du `llm.Stub` ; tout changement de l'un ou l'autre exigera un re-pin sous nouvelle révision de cette ADR.

---

## Alternatives considérées

1. **Différer (documenter seulement).** Rejetée : laisse `tau calibrate` produire un profil dégénéré et un test golden vacant — dette active sur une fonctionnalité livrée.
2. **Relâcher `validRegimes` pour accepter le PascalCase du golden Exchange.** Rejetée : le profil resterait dégénéré (scores absents) ; ne corrige que le symptôme de validation, pas la cause.
3. **Corpus neuf équilibré conçu pour exercer les 4 régimes (dont `refus_i4` réels).** Reportée V0.2 : plus discriminante mais s'écarte de la source synthétique historique et demande l'enrichissement `Context` du générateur.

---

## Vérifications post-décision

- `go build ./...` · `go vet ./...` · `go test ./... -count=1` : verts.
- `go test -tags=e2e ./test/e2e/... -run "TestCalibration|TestCalibrate|TestExpiredProfileRefuses"` : verts (nouveau hash `8e5dc2fc…`).
- `go test -tags=integration ./test/e2e/...` : vert.
- `golangci-lint run` (fichiers touchés, LF) : 0 issue.
- Golden régénéré deux fois → byte-identique ; `tau calibrate` deux fois → hash identique.
- Corpus à `labeled_regime` invalide → `tau calibrate` exit 2 (vérifié).
