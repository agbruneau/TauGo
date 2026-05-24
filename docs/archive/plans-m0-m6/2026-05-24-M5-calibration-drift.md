# M5 Sub-plan — Calibration adaptative + drift + reproductibilité byte-identique

> Sous-plan détaillé du milestone M5 (cf. [`PRDPlanning.md` §M5](../../../PRDPlanning.md) lignes ~1011-1037). Bite-sized, exécutable par sous-agents frais. Calque structurel de `docs/superpowers/plans/2026-05-24-M4-agentmeshkafka-bridge.md`.

**Objectif** : livrer la calibration adaptative versionnée et son contrôle d'invalidation. `internal/calibration/` expose désormais (a) un algorithme déterministe `Calibrate(corpus, seed, distribution) → Profile` qui produit le même JSON byte-identique pour un même couple `(corpus, seed)` ; (b) une détection de drift sur les 5 critères PRD §11.4 ; (c) une persistance JSON triée par clé avec `current.json` actif ; (d) une commande CLI `tau calibrate` ; (e) l'**étape 3 du dispatcher** reportée de M4 (refus si `today > profile.date_revision`). Statut I4/I3 inchangé — M5 ne touche pas la sémantique τ, seulement son ancrage opposable.

**Critère d'acceptation global** :

```powershell
go test -race ./...
go run ./cmd/tau calibrate --corpus tests/calibration/golden-corpus.jsonl --output /tmp/p1.json `
                           --date-revision 2026-11-23 --version-monographie v2.4.3 --seed 1
go run ./cmd/tau calibrate --corpus tests/calibration/golden-corpus.jsonl --output /tmp/p2.json `
                           --date-revision 2026-11-23 --version-monographie v2.4.3 --seed 1
# Windows : Get-FileHash -Algorithm SHA256 ; bash : sha256sum
(Get-FileHash /tmp/p1.json -Algorithm SHA256).Hash -eq (Get-FileHash /tmp/p2.json -Algorithm SHA256).Hash  # → True
```

…vert (0 panique, 0 crash). `TestCalibrationDeterministic` et `TestExpiredProfileRefuses` verts. PRD §17 critère #10 satisfait.

**Tag visé** : `v0.0.6-alpha`

**Pré-requis** : M0…M4 commités, tags `v0.0.1-alpha`..`v0.0.5-alpha` sur `main`. Les fichiers `internal/calibration/{profile.go, thresholds_atomic.go}` existent (livrés M2). `cmd/generate-corpus/` existe (M4) — il sera réutilisé en M5.7 pour produire `tests/calibration/golden-corpus.jsonl`. Aucune règle `arch_test.go` ne contraint `calibration/*` au-delà des règles existantes (`tau/*` reste autonome). Statut « **À vérifier** » : le test E2E `Get-FileHash` PowerShell (commande explicite PRD §11.5 est `sha256sum` POSIX) — la commande équivalente Windows est `Get-FileHash -Algorithm SHA256`, à mentionner dans la doc.

---

## Note de conception — reproductibilité byte-identique

### Le piège `encoding/json` standard

`json.Marshal(map[string]float64{...})` en Go 1.12+ trie les clés alphabétiquement (cf. `encoding/json/encode.go`, `mapEncoder.encode`). **Bonne nouvelle** : la simple combinaison `json.MarshalIndent(profile, "", "  ")` est *déjà* déterministe pour les `map[string]K` (Profile contient trois maps : `SensProbes`, `AuthorityProbes`, `InvariantProbes`). Statut : **Confirmé** par lecture de la stdlib.

**Mais** : un struct dont les champs sont nommés en *PascalCase* avec tags JSON conserve l'**ordre de déclaration** des champs Go pour le rendu JSON — pas l'ordre alphabétique. C'est néanmoins déterministe tant que la struct ne mute pas. Sécurité : ne **jamais** réordonner les champs `Profile`, `Thresholds`, `Weights` sans bumper la version `Profile.Version`.

### Stratégie retenue

1. **Encoding canonique** : `json.MarshalIndent(profile, "", "  ")` + suffixe `"\n"` (UNIX line-ending, écrit en `os.WriteFile` mode binaire pour éviter la traduction Windows CRLF).
2. **Round-trip test** dans `TestCalibrationDeterministic` : générer deux profils sous même seed, comparer les `sha256.Sum256` des bytes. Pas de tolerance flottante : les `float64` issus de `Thresholds` passent par `millis()` (M2) avant écriture donc sont entiers en stockage interne — pas de bruit IEEE-754.
3. **Tri map fail-safe** : pour éliminer toute incertitude résiduelle (Go pourrait changer `mapEncoder` en théorie), `MarshalProfile` (helper local M5.4) **re-décode** la sortie via `json.Decoder` avec `UseNumber()`, **re-sérialise** clé par clé en triant explicitement avec `sort.Strings`. Coût négligeable (< 1 ms / profil) ; couverture totale contre une régression future de la stdlib. Statut : **Probable** que ce soit overkill aujourd'hui, mais cheap insurance pour PRD §17 critère #10.

### Algorithme de calibration V1 — grid search trivial

Pour les seuils : balayer `Deterministe ∈ [0.10, 0.50]` et `Probabiliste ∈ [0.50, 0.90]` par pas de 0.05, sous contrainte `Deterministe ≤ Probabiliste - 0.05`. Pour chaque combinaison `(d, p)`, parcourir le corpus, classer chaque entrée selon le dispatcher M3 avec ces seuils-là, comparer au `expected_regime` étiquetté dans la ligne corpus, retenir la combinaison qui **maximise** le nombre d'accords (ties → premier rencontré, ordre lexico de la grille = déterministe).

Pour les poids : V1 garde les poids `DefaultProfile()` (PRD §11.1). M5.2 expose une `Calibrate` qui prend les poids en argument et ne les bouge pas en V1 ; l'algorithme V2 (M6+ ou ADR séparé) implémentera un gradient. Documenter ce périmètre V1 explicitement dans la docstring + `docs/algorithms/calibration.md`.

**Justification du périmètre V1** : la grille seuils suffit à satisfaire PRD §17 critère #10 (reproductibilité, pas qualité). La calibration des poids est *Hypothèse* dans PRD §11.1 et peut rester telle quelle pour V1.

### `current.json` symlink Windows

PRD §11.3 : *Profil actif = symlink `current.json`.* Sur Windows, `os.Symlink` exige le privilège `SeCreateSymbolicLinkPrivilege` (Developer Mode, ou admin). Stratégie :

1. Tenter `os.Symlink(target, current)`.
2. Si échec (`ErrNotImplemented` ou `EPERM`) : fallback **copie** + écrire un sidecar `current.json.source` contenant le path absolu de la source. Le sidecar permet à la lecture (`LoadCurrent`) de tracer la source originelle pour le diagnostic drift.
3. Documenter la limite dans la docstring `Save` et dans `docs/algorithms/calibration.md`.

Statut : **Probable** que `os.Symlink` échoue sur la machine de dev Windows sans Developer Mode — le fallback est testé en CI sur Windows runner.

### Étape 3 du dispatcher — péremption

Reportée de M4. Logique PRD §10 ligne 514 + PRD §11.4 : au démarrage et à chaque `Decide`, si `time.Now().UTC().After(profile.DateRevision)` → renvoyer `Refus` avec `Diagnostic: "profil périmé — veille requise"`. **Pas de fallback** : ne pas tenter de recalibrer en arrière-plan ici (M5.3 traite le drift d'environnement, pas la péremption temporelle).

Cette étape s'insère entre M3 step 2 (auth-block) et step 4 (scores), exactement comme la pseudo-spec PRD §10. Le `Dispatcher` doit donc connaître le `Profile` actif ; M5.5 modifie `NewDispatcher` pour prendre un `*calibration.Profile` (ou un `ProfileSource` interface pour le hot-reload futur). Statut : **À vérifier** la signature exacte au moment de l'implémentation — option simple = injection directe d'un pointeur, recompiler suffit pour M5 ; un `ProfileSource` avec `Current() *Profile` peut attendre M6 si on en a besoin.

---

## Tâche M5.0 — Pré-flight : audit existant + arch_test pour `calibration/`

**Files :** aucun (lecture-seule) ; vérifier la nécessité éventuelle d'une règle arch_test.

**Agent :** `Explore`

### Briefing autoportant

> Tu es l'agent `Explore` pour TauGo M5. Mission :
>
> 1. Lire `internal/calibration/profile.go` (M2) et `internal/calibration/thresholds_atomic.go` (M2). Lister les types exportés. Vérifier qu'aucun ne dépend de `internal/tau` ou `internal/orchestration` (sinon, lever l'alerte — la calibration doit rester autonome).
> 2. Lire `internal/orchestration/dispatcher.go` lignes ~21-50 — confirmer le commentaire « *Step 3 (profile expiration) lands in M5* » et l'absence de logique périmée. Identifier exactement où insérer le check `today > DateRevision` (entre M3 step 2 et step 4).
> 3. Vérifier que `cmd/generate-corpus/` (M4) écrit un format JSONL et expose un drapeau `--seed`. Énumérer les drapeaux disponibles. Décider si on étend ce binaire pour produire `golden-corpus.jsonl` (option A) ou si on crée un binaire séparé `cmd/seed-corpus/` (option B).
> 4. Vérifier `internal/arch_test.go` : `calibration/*` n'a-t-il aucune règle ? Si oui, proposer si une règle nouvelle s'impose (ex. `calibration → orchestration` interdit pour préserver l'inversion de dépendance).
> 5. Renvoyer un rapport bref (< 250 mots) avec les 4 réponses + recommandation A/B sur le point 3.
>
> Aucune modification de fichier. Lecture-seule.

- [ ] **Étape 1 — Récupérer le rapport de `Explore`**
- [ ] **Étape 2 — Consigner la décision A/B dans une note de tâche M5.7**

**Aucun commit cette tâche.**

---

## Tâche M5.1 — `calibrate.go` : algorithme de calibration des seuils + helper de sérialisation canonique

**Files :**
- Create: `internal/calibration/calibrate.go`
- Create: `internal/calibration/calibrate_test.go`
- Create: `internal/calibration/canonical.go` *(helper `MarshalProfileCanonical`)*
- Create: `internal/calibration/canonical_test.go`

**Agent :** `ruflo-core:coder` (TDD)

### Contrat fonctionnel

```go
package calibration

// CorpusEntry is one labeled exchange used for grid-search calibration.
// Each line of the JSONL golden corpus deserializes to a CorpusEntry.
type CorpusEntry struct {
    ID             string  `json:"id"`
    SensScore      float64 `json:"sens_score"`       // pre-computed by generator
    AuthorityScore float64 `json:"authority_score"`  // pre-computed
    InvariantScore float64 `json:"invariant_score"`  // pre-computed
    HumanInLoop    bool    `json:"human_in_loop"`
    HasAttestation bool    `json:"has_attestation"`
    ExpectedRegime string  `json:"expected_regime"`  // "deterministe" | "probabiliste" | "refus_authority" | "refus_i4"
}

// Calibrate runs the V1 grid-search algorithm against the corpus.
// Returns a Profile whose Thresholds maximize agreement with the labels.
// Determinism: same (corpus, seed, weights) → same Profile.Thresholds.
// V1 scope: weights are NOT calibrated (kept as-is from in.Weights).
func Calibrate(corpus []CorpusEntry, seed int64, in Profile) Profile { ... }

// MarshalProfileCanonical serializes p as byte-identical JSON: indented 2
// spaces, trailing newline, all map keys explicitly sorted, no HTML escape.
// Cf. PRD §11.5 — TestCalibrationDeterministic depends on this helper.
func MarshalProfileCanonical(p Profile) ([]byte, error) { ... }
```

### Algorithme

```
for d in [0.10, 0.15, 0.20, ..., 0.50]:
  for p in [d+0.05, d+0.10, ..., 0.90]:
    score := 0
    for entry in corpus:
      predicted := simulate(entry, d, p, in.Thresholds.AuthBlock, ...)
      if predicted == entry.ExpectedRegime: score++
    if score > best || (score == best && (d, p) < (best_d, best_p) lex):
      best = (d, p, score)
return Profile{... best_d, best_p ...}
```

`simulate` est une projection allégée du dispatcher M3 (pas d'appel LLM — les scores sont pré-calculés dans le corpus). Statut : **Probable** que ce mini-simulateur diverge à terme du vrai dispatcher ; M5.6 verrouille la conformité par un test golden inter-paquet.

- [ ] **Étape 1 — Écrire `canonical_test.go`**

```go
package calibration_test

import (
    "crypto/sha256"
    "testing"

    "github.com/agbruneau/taugo/internal/calibration"
)

func TestMarshalProfileCanonical_ByteIdentical(t *testing.T) {
    t.Parallel()
    p := calibration.DefaultProfile()
    b1, err := calibration.MarshalProfileCanonical(p)
    if err != nil { t.Fatalf("first marshal: %v", err) }
    b2, err := calibration.MarshalProfileCanonical(p)
    if err != nil { t.Fatalf("second marshal: %v", err) }
    if sha256.Sum256(b1) != sha256.Sum256(b2) {
        t.Fatal("MarshalProfileCanonical not idempotent")
    }
}

func TestMarshalProfileCanonical_TrailingNewline(t *testing.T) {
    t.Parallel()
    p := calibration.DefaultProfile()
    b, _ := calibration.MarshalProfileCanonical(p)
    if len(b) == 0 || b[len(b)-1] != '\n' {
        t.Fatal("canonical encoding must end with '\\n'")
    }
}

func TestMarshalProfileCanonical_MapKeysSorted(t *testing.T) {
    t.Parallel()
    // Mutation: inject probe keys in reverse alphabetic order; canonical
    // output must still position them alphabetically.
    p := calibration.DefaultProfile()
    p.Weights.SensProbes = map[string]float64{"z": 0.5, "a": 0.5}
    b, _ := calibration.MarshalProfileCanonical(p)
    aIdx := bytes.Index(b, []byte(`"a"`))
    zIdx := bytes.Index(b, []byte(`"z"`))
    if !(aIdx > 0 && aIdx < zIdx) {
        t.Fatalf(`expected "a" before "z" in canonical output:\n%s`, b)
    }
}
```

- [ ] **Étape 2 — Écrire `canonical.go`**

Stratégie : `json.MarshalIndent(p, "", "  ")` → décoder dans `map[string]any` → réencoder avec `sort.Strings` sur les clés à chaque niveau. Code court (≈ 80 lignes). Ajouter terminal `\n`.

- [ ] **Étape 3 — Écrire `calibrate_test.go`**

```go
func TestCalibrate_DeterministicSameSeed(t *testing.T) {
    corpus := loadFixture(t, "testdata/mini-corpus.jsonl")  // 30 entries inline
    p1 := calibration.Calibrate(corpus, 1, calibration.DefaultProfile())
    p2 := calibration.Calibrate(corpus, 1, calibration.DefaultProfile())
    if p1.Thresholds != p2.Thresholds {
        t.Fatalf("Calibrate not deterministic: %+v vs %+v", p1.Thresholds, p2.Thresholds)
    }
}

func TestCalibrate_ImprovesAgreement(t *testing.T) {
    corpus := loadFixture(t, "testdata/mini-corpus.jsonl")
    baseline := simulateAgreement(corpus, calibration.DefaultProfile().Thresholds)
    calibrated := calibration.Calibrate(corpus, 1, calibration.DefaultProfile())
    after := simulateAgreement(corpus, calibrated.Thresholds)
    if after < baseline {
        t.Fatalf("Calibrate worsened agreement: %d → %d", baseline, after)
    }
}

func TestCalibrate_PreservesWeightsV1(t *testing.T) {
    corpus := loadFixture(t, "testdata/mini-corpus.jsonl")
    in := calibration.DefaultProfile()
    out := calibration.Calibrate(corpus, 1, in)
    // V1 scope: weights are NOT touched.
    if !reflect.DeepEqual(out.Weights, in.Weights) {
        t.Fatal("Calibrate V1 must not mutate Weights")
    }
}
```

- [ ] **Étape 4 — Écrire `calibrate.go`**

Implémenter la grille, `simulate(entry, d, p, authBlock, sensCoh, invCoh) string`, et la sélection lexico-déterministe. Mettre à jour `out.CreatedAt = time.Unix(0, 0).UTC()` *si* on veut reproductibilité totale entre exécutions — sinon, accepter que `CreatedAt` soit gelé par M5.5 (CLI) à un horodatage passé en flag, **pas** `time.Now()`. **Décision** : `Calibrate` ne touche **pas** à `CreatedAt` ; la CLI M5.5 le fixe explicitement (drapeau `--created-at` optionnel, défaut = epoch `1970-01-01T00:00:00Z` si non fourni — c'est ce qui rend `sha256` reproductible entre invocations sans seed externe sur l'heure).

- [ ] **Étape 5 — Créer `internal/calibration/testdata/mini-corpus.jsonl`** (30 entrées : 10 deterministe, 10 probabiliste, 5 refus_authority, 5 refus_i4 ; scores pré-calculés à la main pour rester déterministes).

- [ ] **Étape 6 — Vérifier**

```powershell
go test -race -v ./internal/calibration/
golangci-lint run ./...
```

- [ ] **Étape 7 — Commit**

```powershell
git add internal/calibration/
git commit -m "$(cat <<'EOF'
feat(calibration): Calibrate V1 grid search + canonical JSON encoder

M5.1: Calibrate(corpus, seed, in) runs a deterministic grid search over
(Deterministe, Probabiliste) thresholds on an annotated JSONL corpus,
returning a Profile whose Thresholds maximize agreement with expected
regimes. Tie-break = lexicographic (d, p) — reproducible byte-identical.

MarshalProfileCanonical(p) emits indented JSON with all map keys
explicitly sorted and a trailing newline. Insurance against any future
stdlib change to map encoder ordering — see PRD §17 critère #10.

V1 scope: weights are not calibrated (kept as-is from input Profile).
Status: Hypothèse — V2 weight calibration deferred to ADR.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tâche M5.2 — `weights.go` : squelette V1 (poids non-mutés) + hook V2

**Files :**
- Create: `internal/calibration/weights.go`
- Create: `internal/calibration/weights_test.go`

**Agent :** `ruflo-core:coder` (TDD)

### Contrat

```go
// CalibrateWeights is a V1 NO-OP that returns w unchanged. V2 will
// implement gradient descent against the labeled corpus.
// Justification: PRD §11.1 marks initial weights as "Hypothèse"; M5
// scope is bounded by PRD §17 critère #10 (reproducibility, not
// quality). A real algorithm requires an ADR (M6+).
func CalibrateWeights(corpus []CorpusEntry, seed int64, w Weights) Weights { return w }
```

- [ ] **Étape 1 — Test rouge : reproductibilité de l'identité**

```go
func TestCalibrateWeights_IsIdentityV1(t *testing.T) {
    corpus := loadFixture(t, "testdata/mini-corpus.jsonl")
    w0 := calibration.DefaultProfile().Weights
    w1 := calibration.CalibrateWeights(corpus, 1, w0)
    if !reflect.DeepEqual(w0, w1) {
        t.Fatal("V1 CalibrateWeights must be identity")
    }
}

func TestCalibrateWeights_DocumentsScope(t *testing.T) {
    // Smoke: ensure CalibrateWeights doc-comment mentions "V1 NO-OP".
    // Read source file with go/parser; assert in docstring.
    // (Implementation: parse weights.go; find FuncDecl; check Doc.Text())
}
```

- [ ] **Étape 2 — Écrire `weights.go`** (squelette identité + docstring détaillée).

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v ./internal/calibration/
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/calibration/weights.go internal/calibration/weights_test.go
git commit -m "$(cat <<'EOF'
feat(calibration): CalibrateWeights V1 no-op + V2 hook documented

M5.2: CalibrateWeights is intentionally an identity function in V1,
keeping PRD §11.1 initial weights ("Hypothèse"). A V2 gradient descent
implementation requires an ADR; deferred to M6+. The hook exists so
Calibrate's public surface is stable.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tâche M5.3 — `drift.go` : 5 critères PRD §11.4 + fingerprints CPU/corpus/LLM

**Files :**
- Create: `internal/calibration/drift.go`
- Create: `internal/calibration/drift_test.go`
- Create: `internal/calibration/testdata/drift-corpus-a.jsonl`
- Create: `internal/calibration/testdata/drift-corpus-b.jsonl`

**Agent :** `ruflo-core:coder` (TDD)

### Contrat

```go
// DriftCheck represents the 5 PRD §11.4 invalidation criteria.
type DriftCheck struct {
    CPUFingerprintMatches    bool
    ModelLLMFingerprintMatches bool
    CorpusFingerprintMatches bool
    NotExpired               bool   // !time.Now().UTC().After(p.DateRevision)
    DistributionInZone       bool   // V1 placeholder: returns true always; emits a metric "drift.distribution.placeholder"
}

func (dc DriftCheck) AnyDrifted() bool { return !(dc.CPUFingerprintMatches && dc.ModelLLMFingerprintMatches && dc.CorpusFingerprintMatches && dc.NotExpired && dc.DistributionInZone) }

func (dc DriftCheck) Summary() []string { ... }

// CheckDrift inspects p against the current environment.
// observedCPU / observedLLM / observedCorpus are provided by the caller
// (dispatcher / startup) — calibration package does not import bridge/llm.
func CheckDrift(p Profile, observedCPU, observedLLM, observedCorpus string, now time.Time) DriftCheck { ... }

// CPUFingerprint computes a stable hash of CPU model + GOARCH + GOOS.
// Pure stdlib: runtime.GOARCH + runtime.GOOS + the cpuid leaf when
// available via golang.org/x/sys/cpu, else a fallback "unknown-arch".
func CPUFingerprint() string { ... }

// CorpusFingerprint(path) returns sha256:HEX of the JSONL file at path.
func CorpusFingerprint(path string) (string, error) { ... }
```

### Notes

- `DistributionInZone` est un placeholder V1 : il renvoie toujours `true`. La vraie statistique fenêtre glissante (PRD §11.4 dernière ligne) est explicitement marquée *M5* dans le PRD ; on émet une métrique `drift.distribution.placeholder = 1` via `internal/metrics/` (incrément simple) pour tracer son existence dans la trace. Statut : **Hypothèse** que l'implémentation V2 reste compatible avec cette signature ; documenter dans `docs/algorithms/drift.md`.
- `CPUFingerprint()` : V1 = `runtime.GOOS + "/" + runtime.GOARCH + "/" + cpuModelOrUnknown()`. **À vérifier** : `golang.org/x/sys/cpu` expose les variables `cpu.X86.Name` sur amd64 ; sur arm64 c'est `cpu.ARM64.HasAES` etc. Pour V1, un simple `runtime.GOARCH + "/" + runtime.GOOS` suffit ; le cpuid détaillé attend M6 si nécessaire.
- `CorpusFingerprint` : `sha256` du fichier brut, encodé en `"sha256:" + hex`. Pas de fancy stuff.

- [ ] **Étape 1 — Test rouge `drift_test.go`**

```go
func TestCheckDrift_AllAlignedNoExpiry(t *testing.T) {
    p := calibration.DefaultProfile()
    p.DateRevision = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
    p.CPUFingerprint = "windows/amd64"
    p.CorpusFingerprint = "sha256:abcd"
    p.ModelLLMFingerprint = "stub:v0"
    dc := calibration.CheckDrift(p, "windows/amd64", "stub:v0", "sha256:abcd", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
    if dc.AnyDrifted() {
        t.Fatalf("expected no drift, got %+v", dc)
    }
}

func TestCheckDrift_ExpiredProfile(t *testing.T) {
    p := calibration.DefaultProfile()
    p.DateRevision = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)  // in the past
    dc := calibration.CheckDrift(p, p.CPUFingerprint, p.ModelLLMFingerprint, p.CorpusFingerprint, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
    if dc.NotExpired {
        t.Fatal("expected NotExpired=false on past-dated profile")
    }
}

func TestCheckDrift_CPUMismatchDetected(t *testing.T) { ... }
func TestCheckDrift_LLMMismatchDetected(t *testing.T) { ... }
func TestCheckDrift_CorpusMismatchDetected(t *testing.T) { ... }

func TestCorpusFingerprint_StableAcrossReads(t *testing.T) {
    h1, _ := calibration.CorpusFingerprint("testdata/drift-corpus-a.jsonl")
    h2, _ := calibration.CorpusFingerprint("testdata/drift-corpus-a.jsonl")
    if h1 != h2 || !strings.HasPrefix(h1, "sha256:") {
        t.Fatalf("unstable or malformed fingerprint: %s vs %s", h1, h2)
    }
}

func TestCorpusFingerprint_DifferentFilesDifferentHashes(t *testing.T) {
    ha, _ := calibration.CorpusFingerprint("testdata/drift-corpus-a.jsonl")
    hb, _ := calibration.CorpusFingerprint("testdata/drift-corpus-b.jsonl")
    if ha == hb {
        t.Fatal("two distinct corpora produced identical fingerprints")
    }
}
```

- [ ] **Étape 2 — Écrire `drift.go`** (≈ 120 lignes).

- [ ] **Étape 3 — Créer `testdata/drift-corpus-a.jsonl` et `drift-corpus-b.jsonl`** (3 lignes chacun ; B = A avec une ligne différente).

- [ ] **Étape 4 — Vérifier**

```powershell
go test -race -v ./internal/calibration/
golangci-lint run ./...
```

- [ ] **Étape 5 — Commit**

```powershell
git add internal/calibration/drift.go internal/calibration/drift_test.go internal/calibration/testdata/drift-corpus-*.jsonl
git commit -m "$(cat <<'EOF'
feat(calibration): drift detection for PRD §11.4's 5 criteria

M5.3: CheckDrift(p, observedCPU, observedLLM, observedCorpus, now)
returns a DriftCheck covering all 5 invalidation triggers:
  1. cpu_fingerprint changed
  2. model_llm_fingerprint changed
  3. corpus_fingerprint changed
  4. today > date_revision  (proxy for the dispatcher step 3)
  5. score distribution outside calibrated zone (V1 placeholder=true,
     metric drift.distribution.placeholder emitted)

CPUFingerprint() returns "GOOS/GOARCH" for V1; richer cpuid is deferred.
CorpusFingerprint(path) returns "sha256:" + hex of the file bytes.

Status: criterion #5 is Hypothèse — sliding-window stats deferred to V2
(see docs/algorithms/drift.md for the rationale and acceptance bar).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tâche M5.4 — `store.go` : persistance JSON + `current.json` (symlink + fallback Windows)

**Files :**
- Create: `internal/calibration/store.go`
- Create: `internal/calibration/store_test.go`
- Create: `internal/calibration/store_windows_test.go` *(build tag `windows`)*
- Create: `internal/calibration/store_unix_test.go` *(build tag `!windows`)*

**Agent :** `ruflo-core:coder` (TDD)

### Contrat

```go
// Save writes p to dir/{ID}-{Version}.json (canonical encoding) and
// updates dir/current.json to point at the new file. On Unix, current.json
// is a symlink; on Windows, falls back to a copy + .source sidecar.
func Save(dir string, p Profile) (storedPath string, err error) { ... }

// Load reads and validates a Profile from path. Validates the
// Thresholds.Ordered() invariant.
func Load(path string) (Profile, error) { ... }

// LoadCurrent reads dir/current.json (resolving symlink on Unix, reading
// directly on Windows). Returns the parsed Profile.
func LoadCurrent(dir string) (Profile, error) { ... }
```

### Implémentation `Save`

```go
func Save(dir string, p Profile) (string, error) {
    if err := os.MkdirAll(dir, 0o755); err != nil { return "", err }
    name := fmt.Sprintf("%s-%s.json", p.ID, p.Version)
    target := filepath.Join(dir, name)
    b, err := MarshalProfileCanonical(p)
    if err != nil { return "", err }
    // O_TRUNC|O_WRONLY|O_CREATE — atomic via WriteFile is good enough V1.
    if err := os.WriteFile(target, b, 0o644); err != nil { return "", err }

    current := filepath.Join(dir, "current.json")
    _ = os.Remove(current)  // ignore err: may not exist
    _ = os.Remove(current + ".source")
    if err := os.Symlink(name, current); err != nil {
        // Fallback: copy + sidecar.
        if copyErr := os.WriteFile(current, b, 0o644); copyErr != nil { return "", copyErr }
        if sErr := os.WriteFile(current+".source", []byte(name+"\n"), 0o644); sErr != nil { return "", sErr }
    }
    return target, nil
}
```

- [ ] **Étape 1 — Test commun `store_test.go`** : Save / Load roundtrip ; vérifier que le fichier nommé `{ID}-{Version}.json` existe ; vérifier que `LoadCurrent` revient à l'identique.

- [ ] **Étape 2 — Test `store_unix_test.go`** (`//go:build !windows`) :

```go
func TestSave_CreatesSymlinkOnUnix(t *testing.T) {
    dir := t.TempDir()
    p := calibration.DefaultProfile()
    _, err := calibration.Save(dir, p)
    if err != nil { t.Fatal(err) }
    info, err := os.Lstat(filepath.Join(dir, "current.json"))
    if err != nil { t.Fatal(err) }
    if info.Mode()&os.ModeSymlink == 0 {
        t.Fatal("current.json must be a symlink on Unix")
    }
}
```

- [ ] **Étape 3 — Test `store_windows_test.go`** (`//go:build windows`) :

```go
func TestSave_FallbackOnWindowsWhenSymlinkUnavailable(t *testing.T) {
    dir := t.TempDir()
    p := calibration.DefaultProfile()
    _, err := calibration.Save(dir, p)
    if err != nil { t.Fatal(err) }
    // On Windows without Developer Mode, current.json is a plain copy.
    // If Developer Mode IS enabled, it's a symlink — both are acceptable.
    info, err := os.Stat(filepath.Join(dir, "current.json"))
    if err != nil { t.Fatal(err) }
    if info.Size() == 0 {
        t.Fatal("current.json must be non-empty (copy or symlink-resolved)")
    }
    // Sidecar exists only in the fallback branch.
    sidecarPath := filepath.Join(dir, "current.json.source")
    if _, err := os.Stat(sidecarPath); err == nil {
        b, _ := os.ReadFile(sidecarPath)
        if !strings.Contains(string(b), p.ID) {
            t.Fatalf("sidecar must record the source filename, got %q", b)
        }
    }
}
```

- [ ] **Étape 4 — Écrire `store.go`** (≈ 110 lignes).

- [ ] **Étape 5 — Vérifier (sur Windows local, sur Linux en CI si applicable)**

```powershell
go test -race -v ./internal/calibration/
```

- [ ] **Étape 6 — Commit**

```powershell
git add internal/calibration/store*.go
git commit -m "$(cat <<'EOF'
feat(calibration): persistent store with canonical JSON + current.json

M5.4: Save(dir, p) writes {ID}-{Version}.json under dir using
MarshalProfileCanonical, then updates dir/current.json to point at it.
On Unix, current.json is a symlink (PRD §11.3). On Windows, when
os.Symlink fails (no SeCreateSymbolicLinkPrivilege), we fall back to a
copy plus a current.json.source sidecar that records the original
filename — preserving the traceability the symlink would have given.

Load and LoadCurrent enforce Thresholds.Ordered() on read.

Status: Confirmé on Unix; fallback path Confirmé on Windows by build-tagged test.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tâche M5.5 — CLI `tau calibrate` + injection du profil dans le dispatcher

**Files :**
- Modify: `cmd/tau/main.go` (ajouter `case "calibrate": runCalibrate()`)
- Create: `cmd/tau/calibrate.go`
- Create: `cmd/tau/calibrate_test.go`
- Modify: `internal/app/app.go` (`NewDispatcher` accepte un `*calibration.Profile`)
- Modify: `internal/orchestration/dispatcher.go` (champ `profile *calibration.Profile`, étape 3)

**Agent :** `ruflo-core:coder` (TDD)

### Sous-tâches

#### M5.5.a — Étendre le `Dispatcher` avec un profil + étape 3

- [ ] **Étape 1 — Test rouge `dispatcher_test.go`** (ajout) :

```go
func TestDispatcher_RefusesWhenProfileExpired(t *testing.T) {
    t.Parallel()
    p := calibration.DefaultProfile()
    p.DateRevision = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) // past
    d := orchestration.NewDispatcherWithProfile(llm.Stub{}, orchestration.DefaultThresholds(), &p)
    dec, err := d.Decide(t.Context(), validExchange())
    if err != nil { t.Fatal(err) }
    if dec.Regime != tau.Refus {
        t.Fatalf("expected Refus, got %v", dec.Regime)
    }
    if !strings.Contains(dec.Diagnostic, "profil périmé") {
        t.Fatalf("expected 'profil périmé' diagnostic, got %q", dec.Diagnostic)
    }
}

func TestDispatcher_NoProfileSkipsStep3(t *testing.T) {
    // Backward-compat: NewDispatcher (no profile arg) does not enforce step 3.
    d := orchestration.NewDispatcher(llm.Stub{}, orchestration.DefaultThresholds())
    _, err := d.Decide(t.Context(), validExchange())
    if err != nil { t.Fatal(err) }
}
```

- [ ] **Étape 2 — Modifier `dispatcher.go`** : ajouter `NewDispatcherWithProfile(client, t, *calibration.Profile)`, garder `NewDispatcher` qui pointe vers `nil` (step 3 désactivée si profil nil — opt-in). Insérer le check `if d.profile != nil && time.Now().UTC().After(d.profile.DateRevision) → Refus` **entre step 2 (auth-block) et step 4 (scores)**. Mettre à jour le commentaire L24 (`// Step 3 (profile expiration) lands in M5.` → `// Step 3 (profile expiration) implemented via profile injection.`).

- [ ] **Étape 3 — `internal/app/app.go`** : exporter `NewDispatcherWithProfile(profilePath string)` (qui charge `current.json` du répertoire `~/.config/taugo/profiles/` ou `$XDG_CONFIG_HOME/taugo/profiles/` ou la variable `TAUGO_PROFILE_DIR`). Si aucun profil n'existe encore, log d'avertissement et fallback `NewDispatcher` (compat M4).

#### M5.5.b — CLI `tau calibrate`

- [ ] **Étape 4 — Test rouge `calibrate_test.go`** : invoquer `runCalibrate` avec un argv synthétique pointant sur `testdata/mini-corpus.jsonl`, comparer deux runs avec même seed → `sha256` identique.

- [ ] **Étape 5 — Écrire `cmd/tau/calibrate.go`** :

```go
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/agbruneau/taugo/internal/calibration"
)

func runCalibrate(args []string) {
    fs := flag.NewFlagSet("calibrate", flag.ExitOnError)
    var (
        corpus      = fs.String("corpus", "", "path to JSONL corpus (required)")
        output      = fs.String("output", "", "output profile JSON path (required)")
        dateRev     = fs.String("date-revision", "", "profile DateRevision (RFC3339 or YYYY-MM-DD)")
        versionMono = fs.String("version-monographie", "v2.4.3", "pinned monograph version")
        seed        = fs.Int64("seed", 1, "deterministic seed for the calibrator")
        createdAt   = fs.String("created-at", "1970-01-01T00:00:00Z", "fixed CreatedAt for byte-identical reproducibility")
    )
    _ = fs.Parse(args)
    if *corpus == "" || *output == "" {
        fmt.Fprintln(os.Stderr, "tau calibrate: --corpus and --output are required")
        os.Exit(2)
    }
    in := calibration.DefaultProfile()
    if *dateRev != "" {
        t, err := parseDateRev(*dateRev)
        if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
        in.DateRevision = t
    }
    in.VersionMonographie = *versionMono
    ca, err := time.Parse(time.RFC3339, *createdAt)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
    in.CreatedAt = ca

    // Compute corpus fingerprint NOW so it's embedded in the profile.
    cf, err := calibration.CorpusFingerprint(*corpus)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    in.CorpusFingerprint = cf
    in.CPUFingerprint = calibration.CPUFingerprint()
    // ModelLLMFingerprint stays as "stub:v0" until a real client is injected (M6+).

    entries, err := loadCorpus(*corpus)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

    out := calibration.Calibrate(entries, *seed, in)
    out.Weights = calibration.CalibrateWeights(entries, *seed, in.Weights)

    bytes, err := calibration.MarshalProfileCanonical(out)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    if err := os.WriteFile(*output, bytes, 0o644); err != nil {
        fmt.Fprintln(os.Stderr, err); os.Exit(1)
    }
}

func parseDateRev(s string) (time.Time, error) {
    if t, err := time.Parse(time.RFC3339, s); err == nil { return t, nil }
    return time.Parse("2006-01-02", s)
}

func loadCorpus(path string) ([]calibration.CorpusEntry, error) {
    f, err := os.Open(path)
    if err != nil { return nil, err }
    defer f.Close()
    dec := json.NewDecoder(f)
    var out []calibration.CorpusEntry
    for dec.More() {
        var e calibration.CorpusEntry
        if err := dec.Decode(&e); err != nil { return nil, err }
        out = append(out, e)
    }
    return out, nil
}
```

- [ ] **Étape 6 — Modifier `cmd/tau/main.go`** : ajouter `case "calibrate": runCalibrate(os.Args[2:])`.

- [ ] **Étape 7 — Vérifier**

```powershell
go test -race -v ./...
go build ./cmd/tau
.\tau.exe calibrate --corpus internal/calibration/testdata/mini-corpus.jsonl --output $env:TEMP\p1.json --seed 1
.\tau.exe calibrate --corpus internal/calibration/testdata/mini-corpus.jsonl --output $env:TEMP\p2.json --seed 1
(Get-FileHash $env:TEMP\p1.json -Algorithm SHA256).Hash -eq (Get-FileHash $env:TEMP\p2.json -Algorithm SHA256).Hash
```

Attendu : `True`.

- [ ] **Étape 8 — Commit**

```powershell
git add cmd/tau/ internal/app/app.go internal/orchestration/dispatcher.go internal/orchestration/dispatcher_test.go
git commit -m "$(cat <<'EOF'
feat(cmd/tau,orchestration): `tau calibrate` + dispatcher step 3 (expiry)

M5.5: ships the `tau calibrate` subcommand wiring Calibrate +
CalibrateWeights + MarshalProfileCanonical end-to-end. Flags --corpus,
--output, --date-revision, --version-monographie, --seed, --created-at.
The latter defaults to epoch so two invocations with identical inputs
produce byte-identical JSON (PRD §17 critère #10).

Dispatcher step 3 (profile expiration, deferred from M4) now lands via
NewDispatcherWithProfile(client, thresholds, *Profile). When the profile
is past its DateRevision at Decide-time, returns Refus with the diagnostic
"profil périmé — veille requise" — no fallback (PRD §11.4).

NewDispatcher (no profile) keeps current M3 behavior for back-compat.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tâche M5.6 — `TestCalibrationDeterministic` (PRD §17 #10) + `TestExpiredProfileRefuses` (PRD §15.1) en tant que tests E2E

**Files :**
- Create: `test/e2e/calibration_determinism_test.go` *(tag `e2e`)*
- Create: `tests/calibration/golden-corpus.jsonl` *(généré par M5.7, mais le test attend ce path)*
- Modify: `Makefile` (cible `e2e-calibration` qui exécute `go test -race -tags=e2e ./test/e2e/calibration_determinism_test.go`)

**Agent :** `ruflo-core:coder`

### Note

Le test E2E est l'**ancrage public** du critère PRD §17 #10. Il invoque le binaire `tau` deux fois et compare les sha256. Il vit en `test/e2e/` (déjà institué par M4) sous tag `e2e` pour ne pas alourdir `go test ./...`.

- [ ] **Étape 1 — Écrire `test/e2e/calibration_determinism_test.go`**

```go
//go:build e2e

package e2e

import (
    "crypto/sha256"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestCalibrationDeterministic(t *testing.T) {
    bin := buildTauBinary(t)
    dir := t.TempDir()
    p1 := filepath.Join(dir, "p1.json")
    p2 := filepath.Join(dir, "p2.json")
    args := []string{
        "calibrate",
        "--corpus", "../../tests/calibration/golden-corpus.jsonl",
        "--date-revision", "2026-11-23",
        "--version-monographie", "v2.4.3",
        "--seed", "1",
    }
    for _, out := range []string{p1, p2} {
        cmd := exec.Command(bin, append(args, "--output", out)...)
        cmd.Stderr = os.Stderr
        if err := cmd.Run(); err != nil { t.Fatalf("calibrate run: %v", err) }
    }
    h1 := sha256sumFile(t, p1)
    h2 := sha256sumFile(t, p2)
    if h1 != h2 {
        t.Fatalf("PRD §17 #10 violated: sha256 differ:\n  %s = %x\n  %s = %x", p1, h1, p2, h2)
    }
}

func TestExpiredProfileRefuses(t *testing.T) {
    bin := buildTauBinary(t)
    dir := t.TempDir()
    profilePath := filepath.Join(dir, "expired.json")
    cmd := exec.Command(bin,
        "calibrate",
        "--corpus", "../../tests/calibration/golden-corpus.jsonl",
        "--output", profilePath,
        "--date-revision", "2020-01-01",  // past
        "--seed", "1",
    )
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil { t.Fatalf("calibrate: %v", err) }

    t.Setenv("TAUGO_PROFILE", profilePath)
    decideCmd := exec.Command(bin, "decide")
    decideCmd.Stdin = strings.NewReader(`{"id":"x","intent_description":"y","initiator":{"id":"a","human_in_loop":true,"organization":"o"},"target":{"id":"t","discovery_mode":"static"}}`)
    out, err := decideCmd.Output()
    if err != nil { t.Fatalf("decide: %v", err) }
    if !strings.Contains(string(out), "profil périmé") {
        t.Fatalf("expected expired-profile diagnostic, got: %s", out)
    }
}

func buildTauBinary(t *testing.T) string { ... }
func sha256sumFile(t *testing.T, path string) [32]byte { ... }
```

- [ ] **Étape 2 — Vérifier `tau decide` consomme `TAUGO_PROFILE`** : modification mineure de `runDecide` pour `os.Getenv("TAUGO_PROFILE")` → `calibration.Load(...)` → `app.NewDispatcherWithProfile(...)`. Si vide → comportement actuel (compat).

- [ ] **Étape 3 — Makefile** : ajouter `e2e-calibration: ; go test -race -tags=e2e ./test/e2e/calibration_determinism_test.go`.

- [ ] **Étape 4 — Vérifier (après M5.7 qui produit `golden-corpus.jsonl`)**

```powershell
go test -race -tags=e2e ./test/e2e/calibration_determinism_test.go
```

- [ ] **Étape 5 — Commit**

```powershell
git add test/e2e/calibration_determinism_test.go Makefile cmd/tau/main.go
git commit -m "$(cat <<'EOF'
test(e2e): TestCalibrationDeterministic + TestExpiredProfileRefuses

M5.6: pins PRD §17 critère #10 (byte-identical profile reproducibility)
and PRD §15.1 / §11.4 (expired profile refuses with the canonical
diagnostic). Both tests build the `tau` binary fresh and exercise the
real subcommand surface — no in-process shortcut.

Tagged `e2e` to keep the default `go test ./...` fast. Run via
`make e2e-calibration` or directly with -tags=e2e.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tâche M5.7 — Corpus or `tests/calibration/golden-corpus.jsonl` + intégration générateur M4

**Files :**
- Modify ou Create: selon décision M5.0 sur option A/B
  - **Option A** (recommandée) : modifier `cmd/generate-corpus/generator.go` pour produire en sortie le format `CorpusEntry` (scores pré-calculés + `expected_regime`) via un drapeau `--format=calibration`. Calculer scores via les fonctions `dimensions.Score*` *dans le générateur* — note d'architecture : `cmd/generate-corpus/` peut importer `internal/tau/dimensions/` (couche `cmd/` autorisée à tout).
  - **Option B** : créer `cmd/seed-calibration-corpus/main.go` séparé.
- Create: `tests/calibration/golden-corpus.jsonl` (checked-in, généré reproductiblement avec seed = 1)
- Create: `tests/calibration/README.md` (justifie l'origine, le seed, la commande de regen)

**Agent :** `ruflo-core:coder` + `ruflo-core:researcher` (pour décisions distribution)

### Distribution cible

Mêmes 6 branches que M4 ; 200 entrées totales (≥ 100 PRD §17 #9, marge confortable) :

| Branche | Cible % | Entrées |
|---|---|---|
| Deterministe | 40 % | 80 |
| Probabiliste | 35 % | 70 |
| Refus authority | 15 % | 30 |
| Refus I4 | 10 % | 20 |

Plus minoritaire que la fixture M4 — la calibration a besoin d'**équilibre par cible**, pas de couverture des refus frontière (qui shortcircuitent avant les scores).

### Cycle

- [ ] **Étape 1 — Étendre le générateur** (option A) avec drapeau `--format=calibration` qui appelle `dimensions.ScoreD*` sur chaque échange synthétisé, dérive le `expected_regime` via la formule M3 step 7 (simulation interne), et écrit en JSONL au format `CorpusEntry`.
- [ ] **Étape 2 — Générer le corpus** :
```powershell
go run ./cmd/generate-corpus --format=calibration --count=200 --seed=1 `
                              --output tests/calibration/golden-corpus.jsonl
```
- [ ] **Étape 3 — Vérifier reproductibilité** : regénérer, comparer sha256. Doit être stable.
- [ ] **Étape 4 — Rédiger `tests/calibration/README.md`** : commande de regen exacte, sha256 attendu (commit-pinned), date de génération, motivation.
- [ ] **Étape 5 — Commit**

```powershell
git add cmd/generate-corpus/ tests/calibration/
git commit -m "$(cat <<'EOF'
feat(generate-corpus,tests): golden calibration corpus (200 labeled entries)

M5.7: extends cmd/generate-corpus with --format=calibration which emits
CorpusEntry records (pre-computed dimension scores + expected_regime).
tests/calibration/golden-corpus.jsonl is the checked-in artifact used by
TestCalibrationDeterministic; README documents the exact regeneration
recipe and the pinned sha256.

Distribution: 40% Deterministe / 35% Probabiliste / 15% Refus authority
/ 10% Refus I4 — balanced for grid-search convergence rather than
dispatcher branch coverage.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tâche M5.8 — Documentation : `docs/algorithms/calibration.md` + `docs/algorithms/drift.md`

**Files :**
- Create: `docs/algorithms/calibration.md`
- Create: `docs/algorithms/drift.md`

**Agent :** `ruflo-core:researcher`

### Squelettes

**`calibration.md`** :
1. Vue d'ensemble (lien PRD §11)
2. Algorithme V1 : grid search + pseudo-code + complexité (O(81 × N))
3. Reproductibilité byte-identique : pourquoi `MarshalProfileCanonical`, contre-exemple sans
4. Périmètre V1 vs V2 : poids non-calibrés en V1, ADR à venir
5. Commande CLI : exemple + drapeaux
6. Fingerprints (renvoi à `drift.md`)
7. Limites connues : Windows symlink fallback, placeholder distribution drift
8. Marqueurs d'incertitude (Confirmé / Probable / Hypothèse / À vérifier)

**`drift.md`** :
1. Vue d'ensemble (lien PRD §11.4)
2. 5 critères en détail :
   - `cpu_fingerprint` (formule V1, plan V2)
   - `model_llm_fingerprint` (source : client LLM injecté)
   - `corpus_fingerprint` (sha256 fichier)
   - `today > date_revision` (lien étape 3 dispatcher, PRD §10)
   - `distribution scores hors zone` (placeholder V1, plan fenêtre glissante V2 — référence ADR à créer si V2)
3. Action sur drift : *stale* mark + recalibration arrière-plan (V2) ; en V1, juste un avertissement métrique
4. Action sur péremption : `Refus` sans appel — pas de fallback (PRD §11.4)
5. Diagnostic textuel canonique : `"profil périmé — veille requise"` (renvoi PRD §10 + §17 #10)

- [ ] **Étape 1 — Rédaction**
- [ ] **Étape 2 — Audit fact-check par `understand-anything:explain`** (lancement sur le couple `(docs/algorithms/*, internal/calibration/*)` pour vérifier aucune fabrication)
- [ ] **Étape 3 — Commit**

```powershell
git add docs/algorithms/
git commit -m "$(cat <<'EOF'
docs(algorithms): calibration + drift algorithms documented (PRD §11)

M5.8: docs/algorithms/calibration.md describes the V1 grid-search
algorithm, the canonical JSON encoding strategy, and the reproducibility
contract (PRD §17 critère #10). docs/algorithms/drift.md catalogs the
five PRD §11.4 invalidation criteria, their V1 implementations, and the
deferred V2 work (sliding-window distribution drift).

Every claim is sourced — PRD section, file path, or marked as Hypothèse
where appropriate.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tâche M5.9 — Revue + tag `v0.0.6-alpha` + CHANGELOG

**Files :**
- Modify: `CHANGELOG.md` (ajouter section `v0.0.6-alpha — 2026-05-24`)
- Modify: `cmd/tau/main.go` (bump `version = "0.0.6-alpha"`)

**Agent :** thread principal + `ruflo-core:reviewer`

- [ ] **Étape 1 — Revue par sous-agent `ruflo-core:reviewer`**
  - Briefing : *vérifier que (a) `TestCalibrationDeterministic` et `TestExpiredProfileRefuses` passent ; (b) `go test -race ./...` reste vert ; (c) `golangci-lint run ./...` reste vert ; (d) la sortie de `Get-FileHash` est strictement identique sur deux runs ; (e) aucune fabrication détectée dans `docs/algorithms/*` (croisement avec le code) ; (f) `arch_test.go` reste vert.*
- [ ] **Étape 2 — Bumper la version**
- [ ] **Étape 3 — Rédiger CHANGELOG**

```markdown
## v0.0.6-alpha — 2026-05-24

### Ajouté
- `internal/calibration/calibrate.go` : algo grid-search déterministe (M5.1)
- `internal/calibration/canonical.go` : encodeur JSON byte-identique (M5.1)
- `internal/calibration/weights.go` : hook V2 (V1 = identité) (M5.2)
- `internal/calibration/drift.go` : 5 critères PRD §11.4 (M5.3)
- `internal/calibration/store.go` : persistance + `current.json` (M5.4)
- `cmd/tau/calibrate.go` : commande CLI `tau calibrate` (M5.5)
- Étape 3 dispatcher (péremption profil) — PRD §10 + §11.4 (M5.5)
- `tests/calibration/golden-corpus.jsonl` : corpus or, 200 entrées (M5.7)
- `docs/algorithms/calibration.md`, `docs/algorithms/drift.md` (M5.8)
- Tests E2E : `TestCalibrationDeterministic`, `TestExpiredProfileRefuses` (M5.6)

### Limites connues
- Windows : `current.json` est une copie (+ sidecar `.source`) lorsque
  `os.Symlink` échoue sans Developer Mode (cf. `docs/algorithms/calibration.md`).
- Critère drift #5 (distribution hors zone) est un placeholder V1 ; la
  statistique fenêtre glissante attend une ADR + V2.
- Calibration des poids est l'identité en V1 (PRD §11.1 reste *Hypothèse*).

### Critères PRD §17 satisfaits
- #10 Profil de calibration reproductible byte-identique : **vert**
```

- [ ] **Étape 4 — Tag**

```powershell
git add CHANGELOG.md cmd/tau/main.go
git commit -m "$(cat <<'EOF'
chore(release): v0.0.6-alpha — M5 calibration + drift + step 3

Tag marks the close of M5. PRD §17 critère #10 satisfied. Step 3 of the
dispatcher (profile expiration) deferred from M4 is now in place.
Known limits documented in CHANGELOG and docs/algorithms/*.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git tag -a v0.0.6-alpha -m "M5 — Calibration adaptative + drift"
```

- [ ] **Étape 5 — Vérification finale**

```powershell
go test -race ./...
go test -race -tags=e2e ./test/e2e/...
golangci-lint run ./...
git log --oneline v0.0.5-alpha..v0.0.6-alpha
```

---

## Annexe — Risques M5

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| **R1 — `MarshalIndent` change comportement futur Go** | Faible | PRD §17 #10 invalidé | `MarshalProfileCanonical` re-trie explicitement les clés (M5.1) ; insurance même si stdlib change |
| **R2 — Windows symlink échoue sur runner CI** | Élevée *(probable sans Developer Mode)* | `Save` retourne erreur si pas de fallback | Fallback copie + sidecar (M5.4) ; tests build-taggés `windows` vs `!windows` |
| **R3 — Étape 3 dispatcher casse les tests M3 existants** | Moyenne | Suite verte cassée | `NewDispatcher` reste opt-out (profile nil ⇒ pas d'enforcement) ; tests existants restent verts ; tests M5 utilisent `NewDispatcherWithProfile` |
| **R4 — Float64 IEEE-754 introduit du bruit dans le profil JSON** | Faible | sha256 instable d'une machine à l'autre | Les `Thresholds` passent par `millis()` (M2 — int64 stockage) avant écriture ; les poids `DefaultProfile()` sont des fractions sans queue binaire (`0.4`, `0.3`, `0.25`…). **À vérifier** : 0.35 + 0.30 + 0.20 + 0.15 = 1.0 exactement ? En float64, oui par chance — mais documenter, et envisager `*1000` rounding helper en V2 |
| **R5 — `time.Now()` injecté dans `Profile.CreatedAt` casse la reproductibilité** | Élevée si non géré | PRD §17 #10 KO | CLI M5.5 expose `--created-at` (défaut = epoch). `Calibrate` ne touche pas à `CreatedAt`. Test E2E vérifie. |
| **R6 — Corpus or régénéré incorrectement** | Faible | Tests E2E flaky | `tests/calibration/README.md` documente la commande exacte et le sha256 pinned ; CI peut ajouter un check `sha256sum --check` pre-test |
| **R7 — Métriques `drift.distribution.placeholder` non émises** | Moyenne | Détection drift V2 plus dur à câbler | Émission triviale via `internal/metrics/` ; test unitaire couvre l'appel |
| **R8 — Dispatcher hot-reload de profil non couvert M5** | Acceptée | Profile change requires restart | Hors périmètre M5 ; documenter dans `docs/algorithms/calibration.md` §7 ; ADR pour M6 si demandé |

**Statut global M5** : *Probable que les 9 tâches tiennent sur 2-3 jours de travail concentré.* L'incertitude principale est **R2** (Windows symlink) — tester tôt (M5.4) pour ajuster le fallback si besoin avant que M5.6 dépende du store.

### Critical Files for Implementation
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\calibration\calibrate.go` *(à créer — algo grid search + canonical encoding)*
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\calibration\drift.go` *(à créer — 5 critères PRD §11.4)*
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\calibration\store.go` *(à créer — persistance + Windows fallback)*
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\orchestration\dispatcher.go` *(à modifier — étape 3 péremption + injection profil)*
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\cmd\tau\main.go` *(à modifier — sous-commande `calibrate`)*
