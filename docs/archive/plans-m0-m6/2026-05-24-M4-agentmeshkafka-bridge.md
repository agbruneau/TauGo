# M4 Sub-plan — Adaptateur AgentMeshKafka + campagne empirique I4

> Sous-plan détaillé du milestone M4 (cf. [`PRDPlanning.md` §M4](../../../PRDPlanning.md) lignes ~985-1009). Bite-sized, exécutable par sous-agents frais. Calque structurel de `docs/superpowers/plans/2026-05-24-M3-invariants-fuzz.md`.

**Objectif** : livrer le pont théorie ↔ empirie. Le package `internal/bridge/agentmeshkafka/` expose une interface `Adapter` étroite qui produit un flux d'`AgentMeshExchange` (DTO neutre). Le wrapper d'app convertit en `tau.Exchange` et alimente le dispatcher M3. Une campagne empirique de ≥ 100 traces (réelles si AgentMeshKafka disponible, **synthétiques sinon** — branche contingence PRD §18 risque #1) produit `docs/empirical/I4-report.md` et `docs/empirical/unmodeled.md` initial, faisant passer I4 de *Hypothèse* à *Probable* si la classification confirme l'asymétrie cohérence.

**Critère d'acceptation global** :

```powershell
go test -race ./...
go test -race -tags=integration ./test/e2e/agentmeshkafka_test.go
go test -race -tags=empirical    ./internal/bridge/agentmeshkafka/...
```

…vert (0 panique, 0 crash). Rapport `docs/empirical/I4-report.md` daté contient : nombre de traces ingérées (≥ 100), classification (cohérentes / faux positifs I4 / faux négatifs I4 / cas hors modèle), distribution `Decision.Regime`, marqueur statut I4 (Hypothèse → Probable visé). Rapport `docs/empirical/unmodeled.md` initial liste ≥ 1 observation non modélisée (anti-patron #4 §7.2).

**Tag visé** : `v0.0.5-alpha`

**Pré-requis** : M0/M1/M2/M3 commités, tags `v0.0.1-alpha`..`v0.0.4-alpha` sur `main`. Le squelette `internal/bridge/agentmeshkafka/doc.go` existe déjà (M0). `test/` est vide. La règle `arch_test.go` ligne 32-34 (`bridge/agentmeshkafka → tau` interdit) est **déjà en place** et doit être respectée — voir Note de conception ci-dessous.

---

## Note de conception — étanchéité et DTO neutre

### Contrainte structurelle découverte

`internal/arch_test.go` lignes 32-34 interdit `bridge/agentmeshkafka → tau` :

```go
{from: "github.com/agbruneau/taugo/internal/bridge/agentmeshkafka", deny: []string{
    "github.com/agbruneau/taugo/internal/tau",
}},
```

La signature du PRD §12.1 (`Stream(ctx, topics) (<-chan tau.Exchange, ...)`) **viole cette règle telle quelle**. M4 ne touche pas à `arch_test.go` (suppression de règle = ADR obligatoire, PRD §14.4).

### Solution retenue — DTO neutre + wrapper d'app

1. `bridge/agentmeshkafka/` expose un **DTO local** `AgentMeshExchange` (champs miroir des champs de `tau.Exchange` mais **type-distinct** ; aucun import croisé). Adapter renvoie `<-chan AgentMeshExchange`.
2. `internal/app/agentmesh.go` héberge `func ToTauExchange(AgentMeshExchange) tau.Exchange` (transformation pure, testée). `app/` est la seule couche autorisée à voir **les deux** (`bridge/*` et `tau/*`).
3. `internal/app/agentmesh.go` expose `StreamAsTauExchanges(ctx, adapter, topics) (<-chan tau.Exchange, <-chan error)` qui combine `Adapter.Stream` et `ToTauExchange`.
4. PRD §12.1 doit être révisé pour refléter cette signature corrigée. **M4.1 inclut la rédaction d'un ADR `docs/adr/0005-agentmeshkafka-dto.md`** consignant la décision et la mise à jour mineure du verbatim PRD §12.1.

### Interface étroite (ISP ≤ 5 méthodes)

```go
type Adapter interface {
    // Stream ouvre un flux d'échanges sur les topics demandés.
    // Retourne deux canaux : exchanges et erreurs non-fatales.
    // Le canal exchanges est fermé proprement à ctx.Done() ou Close().
    Stream(ctx context.Context, topics []string) (<-chan AgentMeshExchange, <-chan error)

    // Close libère les ressources (consumer Kafka, fichier, etc.).
    // Doit être idempotente. Bloque jusqu'à drain.
    Close() error
}
```

2 méthodes. ISP respecté.

### Choix d'implémentation mock — fixture JSONL maison

Trois options évaluées :

| Option | Avantage | Inconvénient | Statut |
|---|---|---|---|
| **TestContainers Kafka** | Fidélité maximale | Lourd (Docker requis en CI), CGO/runtime, lent (~30 s startup) | Rejeté — viole §3.3 anti-platform |
| **Sarama mock** | Léger | Importe `IBM/sarama` → dépendance Kafka concrète en `bridge/agentmeshkafka/`, qui pollue le repo TauGo (anti-platform) | Rejeté |
| **Mock fichier JSONL maison** | Aucune dépendance externe ; fidélité du contrat `Stream` ; déterministe ; testable par golden file | Ne couvre pas la sémantique Kafka (offsets, partitions) — acceptable V1 puisque TauGo ne gère pas ces aspects | **Retenu** |

`FileAdapter` lit un fichier `*.jsonl` (une `AgentMeshExchange` par ligne), publie chaque ligne sur `exchanges`, fermeture propre à EOF ou `Close()`. Statut : *Confirmé par construction* (mock minimal, contrat respecté).

### Branche contingence — AgentMeshKafka indisponible

PRD §18 risque #1 énonce : *AgentMeshKafka peut ne pas être prêt comme validateur M4*. Vérification par sous-agent `Explore` à M4.0. Deux régimes :

- **Régime A** — AgentMeshKafka local cloné et publie un flux (vérifié par `Explore`) : M4.4-M4.6 ingèrent des traces réelles. Statut I4 : *Hypothèse → Probable* possible si la campagne confirme.
- **Régime B (contingence)** — AgentMeshKafka absent ou WIP : M4.4-M4.6 ingèrent des traces **synthétiques** générées par `cmd/generate-corpus/` (M4.4-bis). Statut I4 : reste *Hypothèse*, marqué « campagne synthétique — campagne réelle reportée à M4-bis » dans le rapport.

Le plan ci-dessous spécifie les **deux variantes** à M4.4. Le sous-agent `Explore` invoqué en M4.0 décide.

---

## Tâche M4.0 — Audit AgentMeshKafka + arbitrage Régime A/B

**Files :** aucun (recherche pure)

**Agent :** `Explore`

### Briefing autoportant

> Tu es l'agent `Explore` pour TauGo. Mission : déterminer si le projet `agbruneau/AgentMeshKafka` est utilisable comme validateur empirique M4.
>
> 1. Vérifier la présence locale : `ls "C:\Users\agbru\OneDrive\Documents\GitHub\AgentMeshKafka"` et/ou tout dossier `*AgentMeshKafka*` sur le disque.
> 2. Si présent : lire son `README.md`, son `go.mod`, son entrée principale. Évaluer :
>    - Le projet expose-t-il une API (lib, CLI, broker Kafka) qui peut être consommée ?
>    - Format des traces : JSONL, Avro, Protobuf ?
>    - Est-il en état "Probable" (livrable) ou "Hypothèse" (WIP, panics, TODO) ?
> 3. Si absent : vérifier GitHub via `gh repo view agbruneau/AgentMeshKafka --json description,visibility,defaultBranchRef` (sans cloner).
> 4. Renvoyer un verdict : **Régime A** (disponible et utilisable) ou **Régime B** (indisponible → contingence synthétique). Joindre justification courte (< 200 mots), liens, fingerprint commit si applicable.
>
> Aucune modification de fichier. Lecture-seule.

- [ ] **Étape 1 — Récupérer le verdict de `Explore`**

- [ ] **Étape 2 — Consigner le choix dans `docs/empirical/I4-regime.md`** (créé en M4.5 avec le rapport)

**Aucun commit cette tâche** — entrée pour M4.4.

---

## Tâche M4.1 — DTO `AgentMeshExchange` + interface `Adapter` + ADR-0005

**Files :**
- Create: `internal/bridge/agentmeshkafka/adapter.go`
- Create: `internal/bridge/agentmeshkafka/adapter_test.go`
- Modify: `internal/bridge/agentmeshkafka/doc.go` (compléter)
- Create: `docs/adr/0005-agentmeshkafka-dto.md`

**Agent :** `ruflo-core:coder` (TDD) + `ruflo-core:researcher` (rédaction ADR)

- [ ] **Étape 1 — Rédiger ADR-0005**

`docs/adr/0005-agentmeshkafka-dto.md` — squelette :

```markdown
# ADR-0005 — AgentMeshKafka adapter retourne un DTO neutre

*Statut : Accepté · Daté 2026-05-24 · Auteurs : thread principal + ruflo-core:researcher*

## Contexte

PRD §12.1 spécifie initialement :
`Stream(ctx, topics) (<-chan tau.Exchange, <-chan error)`. Cette signature
viole `internal/arch_test.go` qui interdit `bridge/agentmeshkafka → tau`
(règle d'étanchéité PRD §8.1).

## Décision

L'adaptateur expose un **DTO local** `AgentMeshExchange` (champs miroir,
type distinct). La conversion `AgentMeshExchange → tau.Exchange` est hébergée
en `internal/app/agentmesh.go`, la seule couche autorisée à voir `bridge/*`
ET `tau/*`. PRD §12.1 est révisé pour refléter cette signature corrigée.

## Conséquences

Positives : étanchéité préservée ; ADR-0001 (Clean Arch) tient ;
`arch_test.go` non modifié.

Négatives : conversion explicite à maintenir (un test golden pour
verrouiller la bijection).

## Renvois

- PRD §8.1 (Clean Arch), §12.1 (signature révisée), §18 risque #10
- `internal/arch_test.go` lignes 32-34
- Plan : `docs/superpowers/plans/2026-05-24-M4-agentmeshkafka-bridge.md`

Statut : *Confirmé par construction*.
```

- [ ] **Étape 2 — Compléter `internal/bridge/agentmeshkafka/doc.go`**

```go
// Package agentmeshkafka adapts AgentMeshKafka traces into a neutral DTO
// (AgentMeshExchange) for downstream consumption by the app layer.
//
// Architecture rule (gated by internal/arch_test.go): this package must NOT
// import internal/tau/* or internal/orchestration/*. Conversion to
// tau.Exchange lives in internal/app/agentmesh.go (cf. ADR-0005).
//
// V1 ships a file-backed mock adapter (FileAdapter) that reads JSONL
// fixtures. A real Kafka adapter is deferred to M4-bis pending the
// stability of agbruneau/AgentMeshKafka (PRD §18 risque #1).
package agentmeshkafka
```

- [ ] **Étape 3 — Écrire le test rouge `adapter_test.go`**

```go
package agentmeshkafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
)

func TestAdapter_InterfaceShape(t *testing.T) {
	t.Parallel()
	// Compile-time guarantee: any *FileAdapter is an Adapter.
	// The cast forces the interface check at build time.
	var _ agentmeshkafka.Adapter = (*agentmeshkafka.FileAdapter)(nil)
}

func TestAgentMeshExchange_FieldsPresent(t *testing.T) {
	t.Parallel()
	// Smoke: zero-value DTO does not panic; key fields are zero-init.
	x := agentmeshkafka.AgentMeshExchange{
		ID:                "e-0",
		IntentDescription: "noop",
		DiscoveredAt:      time.Unix(0, 0).UTC(),
	}
	if x.ID == "" || x.IntentDescription == "" {
		t.Fatal("zero-value AgentMeshExchange dropped fields unexpectedly")
	}
}

func TestAdapter_StreamSignature(t *testing.T) {
	t.Parallel()
	// Compile-time: an Adapter.Stream returns the documented channels.
	var a agentmeshkafka.Adapter
	if a != nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var (
			_ <-chan agentmeshkafka.AgentMeshExchange
			_ <-chan error
		)
		_, _ = a.Stream(ctx, []string{"topic-1"})
	}
}
```

- [ ] **Étape 4 — Écrire `adapter.go`**

```go
package agentmeshkafka

import (
	"context"
	"time"
)

// AgentMeshExchange is the neutral DTO produced by the adapter. Field names
// mirror the canonical tau.Exchange; conversion lives in internal/app/agentmesh.go
// (cf. ADR-0005). The two types are intentionally distinct to preserve the
// arch_test.go étanchéité (bridge → tau direct: forbidden).
type AgentMeshExchange struct {
	ID                          string                          `json:"id"`
	IntentDescription           string                          `json:"intent_description"`
	DiscoveredAt                time.Time                       `json:"discovered_at"`
	Initiator                   AgentMeshPrincipal              `json:"initiator"`
	Target                      AgentMeshCapability             `json:"target"`
	AttestationInstitutionnelle *AgentMeshAttestation           `json:"attestation_institutionnelle,omitempty"`
	Context                     map[string]any                  `json:"context,omitempty"`
	// Sourcing metadata, neutral to the τ kernel:
	SourceTopic     string `json:"source_topic,omitempty"`
	SourceOffset    int64  `json:"source_offset,omitempty"`
	SourcePartition int32  `json:"source_partition,omitempty"`
}

// AgentMeshPrincipal mirrors tau.Principal.
type AgentMeshPrincipal struct {
	ID              string `json:"id"`
	HumanInLoop     bool   `json:"human_in_loop"`
	Organization    string `json:"organization"`
	DelegationDepth int    `json:"delegation_depth"`
}

// AgentMeshCapability mirrors tau.Capability. DiscoveryMode is a free-form
// string here (e.g. "static" | "dynamic_mcp" | "dynamic_a2a" | "dynamic_agntcy");
// the app-layer converter maps it to the typed tau.DiscoveryMode.
type AgentMeshCapability struct {
	ID            string `json:"id"`
	DiscoveryMode string `json:"discovery_mode"`
	ContractURI   string `json:"contract_uri,omitempty"`
}

// AgentMeshAttestation mirrors tau.Attestation.
type AgentMeshAttestation struct {
	Emetteur   string    `json:"emetteur"`
	Reference  string    `json:"reference"`
	Marqueur   string    `json:"marqueur"`
	AssertedAt time.Time `json:"asserted_at"`
}

// Adapter streams AgentMesh traces. Two-method ISP-conforming interface.
// V1 ships FileAdapter (JSONL); a real Kafka adapter lands in M4-bis.
type Adapter interface {
	// Stream opens a flow of exchanges over the given topics. Returns two
	// channels: exchanges and non-fatal errors. The exchanges channel is
	// closed cleanly on ctx.Done() or after Close().
	Stream(ctx context.Context, topics []string) (<-chan AgentMeshExchange, <-chan error)

	// Close releases resources. Idempotent. Blocks until in-flight drain.
	Close() error
}
```

- [ ] **Étape 5 — Vérifier**

```powershell
go build ./internal/bridge/agentmeshkafka/
go test -race ./internal/bridge/agentmeshkafka/
go test -race -run TestArchitectureLayering ./internal/
golangci-lint run ./...
```

Attendu : tests verts (la conformité d'interface `*FileAdapter` est un cast nil — l'agent doit créer un stub temporaire `type FileAdapter struct{}` avec méthodes vides en M4.1 pour décrocher la compilation, puis le remplir en M4.2). **Hypothèse — *À vérifier*** : Go autorise le cast d'un `(*FileAdapter)(nil)` même sans méthodes implémentées si l'interface est vide ; pour deux méthodes, les stubs sont nécessaires.

- [ ] **Étape 6 — Stubs temporaires `FileAdapter`** (supprimés en M4.2)

Ajouter en bas de `adapter.go` :

```go
// === STUB TEMPORAIRE — remplacé par sa vraie implémentation en M4.2 ===

// FileAdapter (M4.1 stub) — méthodes vides pour décrocher la compilation
// du package squelette. Implémentation réelle en M4.2.
type FileAdapter struct{}

func (f *FileAdapter) Stream(_ context.Context, _ []string) (<-chan AgentMeshExchange, <-chan error) {
	ex := make(chan AgentMeshExchange)
	errs := make(chan error)
	close(ex)
	close(errs)
	return ex, errs
}

func (f *FileAdapter) Close() error { return nil }
```

- [ ] **Étape 7 — Commit**

```powershell
git add internal/bridge/agentmeshkafka/ docs/adr/0005-agentmeshkafka-dto.md
git commit -m "feat(bridge/agentmeshkafka): scaffold Adapter + AgentMeshExchange DTO

M4.1: introduces internal/bridge/agentmeshkafka/ with the neutral DTO
AgentMeshExchange (and its three sub-types Principal, Capability,
Attestation) and the Adapter interface (Stream, Close — ISP ≤ 5).

ADR-0005 records the étanchéité-driven choice: the adapter must not import
internal/tau (arch_test.go ligne 32-34); conversion to tau.Exchange lives
in internal/app/agentmesh.go (M4.7).

FileAdapter stub (empty Stream / Close) unblocks compilation; the real
JSONL implementation lands in M4.2.

Status: Confirmé par construction.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.2 — `FileAdapter` JSONL + tests golden

**Files :**
- Modify: `internal/bridge/agentmeshkafka/adapter.go` (retirer le stub `FileAdapter`)
- Create: `internal/bridge/agentmeshkafka/file_adapter.go`
- Create: `internal/bridge/agentmeshkafka/file_adapter_test.go`
- Create: `internal/bridge/agentmeshkafka/testdata/sample.jsonl`
- Create: `internal/bridge/agentmeshkafka/testdata/sample-malformed.jsonl`

**Agent :** `ruflo-core:coder` (TDD)

### Contrat fonctionnel

`FileAdapter` lit un fichier JSONL ligne à ligne, parse chaque ligne en `AgentMeshExchange`, publie sur `<-chan AgentMeshExchange`. Erreurs de parsing non-fatales → canal `<-chan error` (canal-erreur reste ouvert ; le canal-exchange continue). EOF ou `ctx.Done()` ou `Close()` → fermeture propre des deux canaux. **Filtrage par `topics`** : si la liste n'est pas vide, ne publie que les lignes dont `SourceTopic` matche.

- [ ] **Étape 1 — Écrire `file_adapter_test.go`**

```go
package agentmeshkafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
)

func newFileAdapter(t *testing.T, path string) *agentmeshkafka.FileAdapter {
	t.Helper()
	a, err := agentmeshkafka.NewFileAdapter(path)
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

func TestFileAdapter_StreamsAllLines(t *testing.T) {
	t.Parallel()
	a := newFileAdapter(t, "testdata/sample.jsonl")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	ex, errs := a.Stream(ctx, nil) // nil = no topic filter
	var n int
	for x := range ex {
		_ = x
		n++
	}
	// Drain errs (non-fatal).
	for e := range errs {
		t.Logf("non-fatal: %v", e)
	}
	if n != 5 {
		t.Fatalf("got %d exchanges, want 5 from sample.jsonl", n)
	}
}

func TestFileAdapter_TopicFilter(t *testing.T) {
	t.Parallel()
	a := newFileAdapter(t, "testdata/sample.jsonl")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	ex, _ := a.Stream(ctx, []string{"agentic.bfsi"})
	var n int
	for range ex {
		n++
	}
	if n == 0 {
		t.Fatal("topic filter returned 0 exchanges; expected ≥ 1 from sample.jsonl")
	}
}

func TestFileAdapter_MalformedLineYieldsError(t *testing.T) {
	t.Parallel()
	a := newFileAdapter(t, "testdata/sample-malformed.jsonl")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	ex, errs := a.Stream(ctx, nil)
	var nEx, nErr int
	doneEx, doneErr := false, false
	for !doneEx || !doneErr {
		select {
		case _, ok := <-ex:
			if !ok {
				doneEx = true
				continue
			}
			nEx++
		case _, ok := <-errs:
			if !ok {
				doneErr = true
				continue
			}
			nErr++
		}
	}
	if nErr == 0 {
		t.Fatal("expected ≥ 1 non-fatal error on malformed line")
	}
	if nEx == 0 {
		t.Fatal("expected ≥ 1 well-formed exchange to pass through")
	}
}

func TestFileAdapter_CloseIdempotent(t *testing.T) {
	t.Parallel()
	a := newFileAdapter(t, "testdata/sample.jsonl")
	if err := a.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("second Close (should be idempotent): %v", err)
	}
}

func TestFileAdapter_CancelStopsStream(t *testing.T) {
	t.Parallel()
	a := newFileAdapter(t, "testdata/sample.jsonl")
	ctx, cancel := context.WithCancel(t.Context())
	ex, _ := a.Stream(ctx, nil)
	cancel()
	// Drain — must terminate.
	for range ex {
	}
}
```

- [ ] **Étape 2 — Écrire `file_adapter.go`**

```go
package agentmeshkafka

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
)

// FileAdapter is the V1 mock adapter. It reads JSONL from a local file:
// one AgentMeshExchange per line. Errors during parse are emitted on the
// error channel (non-fatal); well-formed lines continue to flow.
//
// Deterministic by construction: line order = stream order; no goroutine
// shuffling, no buffered randomness.
type FileAdapter struct {
	path     string
	closeMu  sync.Mutex
	closed   bool
	cancelFn context.CancelFunc
}

// NewFileAdapter constructs a FileAdapter bound to path. The file is opened
// lazily in Stream (allows Close before Stream).
func NewFileAdapter(path string) (*FileAdapter, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("agentmeshkafka: stat %s: %w", path, err)
	}
	return &FileAdapter{path: path}, nil
}

// Stream opens the JSONL file and publishes each parsed AgentMeshExchange
// onto the returned channel. Closes both channels on EOF, ctx.Done, or
// Close.
func (f *FileAdapter) Stream(ctx context.Context, topics []string) (<-chan AgentMeshExchange, <-chan error) {
	ex := make(chan AgentMeshExchange)
	errs := make(chan error, 8)
	subCtx, cancel := context.WithCancel(ctx)
	f.closeMu.Lock()
	f.cancelFn = cancel
	f.closeMu.Unlock()

	go func() {
		defer close(ex)
		defer close(errs)

		file, err := os.Open(f.path)
		if err != nil {
			select {
			case errs <- fmt.Errorf("open %s: %w", f.path, err):
			default:
			}
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		// Bump buffer to 1 MiB for fat traces.
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			if subCtx.Err() != nil {
				return
			}
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var x AgentMeshExchange
			if err := json.Unmarshal(line, &x); err != nil {
				select {
				case errs <- fmt.Errorf("parse: %w", err):
				default:
				}
				continue
			}
			if len(topics) > 0 && !slices.Contains(topics, x.SourceTopic) {
				continue
			}
			select {
			case ex <- x:
			case <-subCtx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			select {
			case errs <- fmt.Errorf("scan: %w", err):
			default:
			}
		}
	}()
	return ex, errs
}

// Close releases the cancel func. Idempotent.
func (f *FileAdapter) Close() error {
	f.closeMu.Lock()
	defer f.closeMu.Unlock()
	if f.closed {
		return nil
	}
	if f.cancelFn != nil {
		f.cancelFn()
	}
	f.closed = true
	return nil
}
```

Supprimer le stub `FileAdapter` à la fin de `adapter.go`.

- [ ] **Étape 3 — Créer les fixtures JSONL**

`testdata/sample.jsonl` — 5 lignes, deux topics (`agentic.bfsi`, `agentic.support`) :

```json
{"id":"e-001","intent_description":"approve loan","discovered_at":"2026-04-01T10:00:00Z","initiator":{"id":"agent-bfsi-1","human_in_loop":false,"organization":"bank-a","delegation_depth":2},"target":{"id":"tool-credit-score","discovery_mode":"dynamic_mcp","contract_uri":""},"source_topic":"agentic.bfsi","source_offset":1,"source_partition":0}
{"id":"e-002","intent_description":"refund request","discovered_at":"2026-04-01T10:01:00Z","initiator":{"id":"agent-support-1","human_in_loop":true,"organization":"merchant-a","delegation_depth":1},"target":{"id":"tool-refund","discovery_mode":"static","contract_uri":"https://api/v1/refund"},"source_topic":"agentic.support","source_offset":1,"source_partition":0}
{"id":"e-003","intent_description":"transfer 5000","discovered_at":"2026-04-01T10:02:00Z","initiator":{"id":"agent-bfsi-2","human_in_loop":false,"organization":"bank-a","delegation_depth":3},"target":{"id":"tool-transfer","discovery_mode":"dynamic_a2a","contract_uri":""},"attestation_institutionnelle":{"emetteur":"ietf","reference":"draft-agentic-id-01","marqueur":"Hypothèse","asserted_at":"2026-04-01T00:00:00Z"},"source_topic":"agentic.bfsi","source_offset":2,"source_partition":0}
{"id":"e-004","intent_description":"lookup balance","discovered_at":"2026-04-01T10:03:00Z","initiator":{"id":"agent-support-2","human_in_loop":true,"organization":"merchant-a","delegation_depth":0},"target":{"id":"tool-balance","discovery_mode":"static","contract_uri":"https://api/v1/balance"},"source_topic":"agentic.support","source_offset":2,"source_partition":0}
{"id":"e-005","intent_description":"schedule meeting","discovered_at":"2026-04-01T10:04:00Z","initiator":{"id":"agent-cal-1","human_in_loop":false,"organization":"saas-a","delegation_depth":1},"target":{"id":"tool-calendar","discovery_mode":"dynamic_agntcy","contract_uri":""},"source_topic":"agentic.support","source_offset":3,"source_partition":0}
```

`testdata/sample-malformed.jsonl` — 3 lignes (1 mal-formée milieu, 2 bonnes) :

```json
{"id":"e-100","intent_description":"good 1","discovered_at":"2026-04-01T10:00:00Z","initiator":{"id":"a-1","human_in_loop":false,"organization":"o","delegation_depth":1},"target":{"id":"t","discovery_mode":"dynamic_mcp"},"source_topic":"agentic.bfsi"}
{ malformed json here, no quotes around keys: 42 }
{"id":"e-101","intent_description":"good 2","discovered_at":"2026-04-01T10:01:00Z","initiator":{"id":"a-2","human_in_loop":true,"organization":"o","delegation_depth":0},"target":{"id":"t","discovery_mode":"static"},"source_topic":"agentic.bfsi"}
```

- [ ] **Étape 4 — Vérifier**

```powershell
go test -race -v ./internal/bridge/agentmeshkafka/
golangci-lint run ./...
```

- [ ] **Étape 5 — Commit**

```powershell
git add internal/bridge/agentmeshkafka/
git commit -m "feat(bridge/agentmeshkafka): FileAdapter (JSONL mock)

M4.2: FileAdapter reads a JSONL file and publishes each parsed
AgentMeshExchange. Non-fatal parse errors flow on the error channel; the
exchange channel keeps draining. Topic filter supported. Close is
idempotent. Cancellation via ctx propagates.

Testdata:
  - sample.jsonl: 5 well-formed exchanges across 2 topics
  - sample-malformed.jsonl: 1 invalid line surrounded by 2 valid lines

V1 mock by design: TestContainers and Sarama-mock options were rejected
(anti-platform §3.3, dependency-bloat). A real Kafka adapter is deferred
to M4-bis pending agbruneau/AgentMeshKafka stability (PRD §18 risque #1).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.3 — Convertisseur `app/agentmesh.go` + `StreamAsTauExchanges`

**Files :**
- Create: `internal/app/agentmesh.go`
- Create: `internal/app/agentmesh_test.go`

**Agent :** `ruflo-core:coder` (TDD)

### Note de conception

`app/agentmesh.go` est le **pivot d'étanchéité**. Il importe `bridge/agentmeshkafka` et `tau` (couche autorisée à voir les deux). Expose :

1. `func ToTauExchange(AgentMeshExchange) tau.Exchange` — pure, testable.
2. `func StreamAsTauExchanges(ctx, adapter, topics) (<-chan tau.Exchange, <-chan error)` — combinateur.

Mapping `DiscoveryMode string → tau.DiscoveryMode` :

| String AgentMesh | Typed tau |
|---|---|
| `"static"` ou vide | `tau.Static` |
| `"dynamic_mcp"` | `tau.DynamicMCP` |
| `"dynamic_a2a"` | `tau.DynamicA2A` |
| `"dynamic_agntcy"` | `tau.DynamicAGNTCY` |
| autre | `tau.DynamicMCP` (fallback conservateur — dynamic-side ; un audit logs l'inconnu) |

- [ ] **Étape 1 — Écrire `agentmesh_test.go`**

```go
package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
	"github.com/agbruneau/taugo/internal/tau"
)

func TestToTauExchange_BijectionCoreFields(t *testing.T) {
	t.Parallel()
	src := agentmeshkafka.AgentMeshExchange{
		ID:                "e-conv-1",
		IntentDescription: "compute",
		DiscoveredAt:      time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID:              "p-1",
			HumanInLoop:     false,
			Organization:    "org-a",
			DelegationDepth: 2,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID:            "cap-1",
			DiscoveryMode: "dynamic_mcp",
			ContractURI:   "",
		},
	}
	got := app.ToTauExchange(src)
	if got.ID != src.ID {
		t.Fatalf("ID drift: got=%s, want=%s", got.ID, src.ID)
	}
	if got.Target.DiscoveryMode != tau.DynamicMCP {
		t.Fatalf("DiscoveryMode drift: got=%v, want=DynamicMCP", got.Target.DiscoveryMode)
	}
	if got.Initiator.DelegationDepth != 2 {
		t.Fatalf("DelegationDepth drift: got=%d", got.Initiator.DelegationDepth)
	}
}

func TestToTauExchange_DiscoveryModeMapping(t *testing.T) {
	t.Parallel()
	cases := map[string]tau.DiscoveryMode{
		"static":         tau.Static,
		"":               tau.Static,
		"dynamic_mcp":    tau.DynamicMCP,
		"dynamic_a2a":    tau.DynamicA2A,
		"dynamic_agntcy": tau.DynamicAGNTCY,
		"unknown_xyz":    tau.DynamicMCP, // conservative fallback (dynamic-side)
	}
	for in, want := range cases {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			src := agentmeshkafka.AgentMeshExchange{
				ID: "x", Target: agentmeshkafka.AgentMeshCapability{DiscoveryMode: in},
			}
			got := app.ToTauExchange(src).Target.DiscoveryMode
			if got != want {
				t.Fatalf("ToTauExchange(%q).Target.DiscoveryMode = %v, want %v", in, got, want)
			}
		})
	}
}

func TestToTauExchange_AttestationPreserved(t *testing.T) {
	t.Parallel()
	src := agentmeshkafka.AgentMeshExchange{
		ID: "e-att",
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur:   "ietf",
			Reference:  "draft-x",
			Marqueur:   "Hypothèse",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	got := app.ToTauExchange(src)
	if got.AttestationInstitutionnelle == nil {
		t.Fatal("Attestation dropped during conversion")
	}
	if got.AttestationInstitutionnelle.Emetteur != "ietf" {
		t.Fatalf("Emetteur drift: %s", got.AttestationInstitutionnelle.Emetteur)
	}
}

func TestStreamAsTauExchanges_EndToEnd(t *testing.T) {
	t.Parallel()
	a, err := agentmeshkafka.NewFileAdapter("../bridge/agentmeshkafka/testdata/sample.jsonl")
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	tauEx, _ := app.StreamAsTauExchanges(ctx, a, nil)
	var n int
	for x := range tauEx {
		if x.ID == "" {
			t.Fatalf("empty ID on converted exchange #%d", n)
		}
		n++
	}
	if n != 5 {
		t.Fatalf("converted count = %d, want 5", n)
	}
}
```

- [ ] **Étape 2 — Écrire `agentmesh.go`**

```go
package app

import (
	"context"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
	"github.com/agbruneau/taugo/internal/tau"
)

// ToTauExchange converts a neutral AgentMeshExchange into a typed tau.Exchange.
// This conversion is hosted at the app layer (cf. ADR-0005) because the
// arch_test.go forbids bridge/agentmeshkafka → tau.
//
// DiscoveryMode mapping is conservative on unknown values: it falls back
// to DynamicMCP (dynamic-side) rather than Static, so that an unknown
// frontier is treated as "inside τ" — a τ-application is preferable to a
// silent bypass (anti-patron #2 #4).
func ToTauExchange(x agentmeshkafka.AgentMeshExchange) tau.Exchange {
	out := tau.Exchange{
		ID:                x.ID,
		IntentDescription: x.IntentDescription,
		DiscoveredAt:      x.DiscoveredAt,
		Initiator: tau.Principal{
			ID:              x.Initiator.ID,
			HumanInLoop:     x.Initiator.HumanInLoop,
			Organization:    x.Initiator.Organization,
			DelegationDepth: x.Initiator.DelegationDepth,
		},
		Target: tau.Capability{
			ID:            x.Target.ID,
			DiscoveryMode: discoveryModeFromString(x.Target.DiscoveryMode),
			ContractURI:   x.Target.ContractURI,
		},
		Context: x.Context,
	}
	if x.AttestationInstitutionnelle != nil {
		out.AttestationInstitutionnelle = &tau.Attestation{
			Emetteur:   x.AttestationInstitutionnelle.Emetteur,
			Reference:  x.AttestationInstitutionnelle.Reference,
			Marqueur:   x.AttestationInstitutionnelle.Marqueur,
			AssertedAt: x.AttestationInstitutionnelle.AssertedAt,
		}
	}
	return out
}

func discoveryModeFromString(s string) tau.DiscoveryMode {
	switch s {
	case "", "static":
		return tau.Static
	case "dynamic_mcp":
		return tau.DynamicMCP
	case "dynamic_a2a":
		return tau.DynamicA2A
	case "dynamic_agntcy":
		return tau.DynamicAGNTCY
	default:
		return tau.DynamicMCP
	}
}

// StreamAsTauExchanges adapts a bridge Adapter to the kernel's input
// shape. The output channel is closed when the source Adapter closes its
// exchanges channel; non-fatal errors are forwarded verbatim.
func StreamAsTauExchanges(
	ctx context.Context,
	adapter agentmeshkafka.Adapter,
	topics []string,
) (<-chan tau.Exchange, <-chan error) {
	src, errs := adapter.Stream(ctx, topics)
	out := make(chan tau.Exchange)
	go func() {
		defer close(out)
		for x := range src {
			select {
			case out <- ToTauExchange(x):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errs
}
```

- [ ] **Étape 3 — Vérifier**

```powershell
go test -race -v ./internal/app/
go test -race -run TestArchitectureLayering ./internal/
golangci-lint run ./...
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/app/
git commit -m "feat(app): AgentMesh → tau.Exchange converter + Stream combinator

M4.3: internal/app/agentmesh.go bridges the AgentMeshKafka DTO and the
typed tau.Exchange. ToTauExchange is pure and deterministic;
StreamAsTauExchanges adapts Adapter.Stream output. DiscoveryMode mapping
is conservative on unknown values (falls back to DynamicMCP rather than
Static, to avoid a silent frontier bypass — anti-patron #2/#4).

This hosting at the app layer is forced by ADR-0005 / arch_test.go
(bridge/agentmeshkafka → tau forbidden).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.4 — Générateur de traces synthétiques (`cmd/generate-corpus`)

**Files :**
- Create: `cmd/generate-corpus/main.go`
- Create: `cmd/generate-corpus/doc.go`
- Create: `cmd/generate-corpus/generator.go`
- Create: `cmd/generate-corpus/generator_test.go`
- Create: `internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl` (généré, mais checked-in)

**Agent :** `ruflo-core:coder` (TDD) + `ruflo-core:researcher` (validation distribution)

### Note de conception

Indispensable que **Régime A ou B**. En Régime A, le générateur sert de corpus de référence reproductible pour les tests E2E. En Régime B, il devient *la* source de la campagne empirique.

Le générateur produit ≥ 100 `AgentMeshExchange` couvrant **les six branches du dispatcher M3** :

| # | Branche | Cible % |
|---|---|---|
| 1 | Refus hors frontière (Static + HumanInLoop) | ≈ 15 % |
| 2 | Refus I3 (DynamicMCP + DelegationDepth > 0 + sans attestation) | ≈ 15 % |
| 3 | Refus I4 (combinaison incohérente — fixture taguée par poids de sondes) | ≈ 10 % |
| 4 | Deterministe (scores composites bas, attestation) | ≈ 25 % |
| 5 | Probabiliste (scores composites hauts, attestation) | ≈ 25 % |
| 6 | Bord — cas ambigus pour zone d'hystérèse | ≈ 10 % |

Générateur **déterministe** : seed `int64` paramétrable ; même seed → même fichier byte-identique (calque FibGo).

- [ ] **Étape 1 — Écrire `cmd/generate-corpus/doc.go`**

```go
// Package main implements `tau generate-corpus`, a deterministic generator
// of AgentMeshExchange fixtures distributed across the six dispatcher
// branches (Refus×3, Deterministe, Probabiliste, hysteresis). Output is
// JSONL ingestible by FileAdapter.
//
// Used by M4 (synthetic empirical campaign — Régime B contingency, PRD
// §18 risque #1).
package main
```

- [ ] **Étape 2 — Écrire `generator.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"time"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
)

// Generator produces a deterministic stream of AgentMeshExchange across
// the six dispatcher branches. The seed argument fixes the entire output.
type Generator struct {
	rng *rand.Rand
}

// NewGenerator constructs a Generator with the given seed. Same seed →
// byte-identical output (PRD §11 calque FibGo).
func NewGenerator(seed int64) *Generator {
	src := rand.NewPCG(uint64(seed), uint64(seed)^0xdeadbeef)
	return &Generator{rng: rand.New(src)}
}

// Distribution returns the target percentage per branch (sums to 100).
// Public for testing — verifies the fixture matches the documented target.
type Distribution struct {
	RefusFrontiere   int // 15
	RefusI3          int // 15
	RefusI4          int // 10
	Deterministe     int // 25
	Probabiliste     int // 25
	Hysteresis       int // 10
}

// DefaultDistribution is the M4 reference mixture.
func DefaultDistribution() Distribution {
	return Distribution{RefusFrontiere: 15, RefusI3: 15, RefusI4: 10, Deterministe: 25, Probabiliste: 25, Hysteresis: 10}
}

// Generate writes n exchanges to w as JSONL. Lines are deterministic under
// the seed. Distribution is honored ± 1 entry due to rounding.
func (g *Generator) Generate(w io.Writer, n int, d Distribution) error {
	if d.RefusFrontiere+d.RefusI3+d.RefusI4+d.Deterministe+d.Probabiliste+d.Hysteresis != 100 {
		return fmt.Errorf("distribution does not sum to 100")
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	type branch func(i int) agentmeshkafka.AgentMeshExchange
	plan := make([]branch, 0, n)
	add := func(count int, b branch) {
		for i := 0; i < count; i++ {
			plan = append(plan, b)
		}
	}
	add(n*d.RefusFrontiere/100, g.refusFrontiere)
	add(n*d.RefusI3/100, g.refusI3)
	add(n*d.RefusI4/100, g.refusI4)
	add(n*d.Deterministe/100, g.deterministe)
	add(n*d.Probabiliste/100, g.probabiliste)
	for len(plan) < n {
		plan = append(plan, g.hysteresis)
	}
	// Shuffle deterministically so the order does not correlate with branch.
	g.rng.Shuffle(len(plan), func(i, j int) { plan[i], plan[j] = plan[j], plan[i] })

	base := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	for i, b := range plan {
		x := b(i)
		x.DiscoveredAt = base.Add(time.Duration(i) * time.Minute)
		x.SourceTopic = "agentic.synth"
		x.SourceOffset = int64(i)
		if err := enc.Encode(&x); err != nil {
			return err
		}
	}
	return nil
}

// Each branch builder is a pure function of (i, g.rng). Branches encode
// the dispatcher condition that they target (cf. dispatcher.go).
//
// V1 status: Hypothèse — these heuristics target the M3 dispatcher branches
// based on its current frontierFromExchange and dimension-score behavior.
// Calibration M5 may shift the actual cut points; the I4 report dates the
// fixture and reports actual observed distribution post-dispatch.

func (g *Generator) refusFrontiere(i int) agentmeshkafka.AgentMeshExchange {
	// Static + HumanInLoop → Inside() == false on all 4 conditions.
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-rf-%04d", i),
		IntentDescription: "static call with human anchor",
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID: fmt.Sprintf("agent-%04d", i), HumanInLoop: true, Organization: "org-synth", DelegationDepth: 0,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID: "tool-static", DiscoveryMode: "static", ContractURI: "https://api/v1/op",
		},
	}
}

func (g *Generator) refusI3(i int) agentmeshkafka.AgentMeshExchange {
	// DynamicMCP + DelegationDepth >= 2 + no humanInLoop + no attestation
	// → high D-AUTORITE; without attestation → refus I3.
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-r3-%04d", i),
		IntentDescription: "cross-org dynamic call",
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID: fmt.Sprintf("agent-x-%04d", i), HumanInLoop: false, Organization: "org-x", DelegationDepth: 3,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID: "tool-dyn", DiscoveryMode: "dynamic_mcp",
		},
	}
}

func (g *Generator) refusI4(i int) agentmeshkafka.AgentMeshExchange {
	// Target: high D-INVARIANT + low D-SENS — captured by very short intent
	// description (low S_reasoner_intent under stub LLM) on a fully static
	// contract (high I_enumerated_plan).
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-r4-%04d", i),
		IntentDescription: "x",
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID: fmt.Sprintf("agent-%04d", i), HumanInLoop: false, Organization: "org-i4", DelegationDepth: 1,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID: "tool-i4", DiscoveryMode: "dynamic_mcp", ContractURI: "https://api/v1/strict",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur: "ietf", Reference: "draft-x", Marqueur: "Hypothèse",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func (g *Generator) deterministe(i int) agentmeshkafka.AgentMeshExchange {
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-d-%04d", i),
		IntentDescription: "standard call with contract and human anchor",
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID: fmt.Sprintf("agent-%04d", i), HumanInLoop: false, Organization: "org-d", DelegationDepth: 1,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID: "tool-d", DiscoveryMode: "dynamic_mcp", ContractURI: "https://api/v1/op",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur: "ietf", Reference: "draft-x", Marqueur: "Probable",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func (g *Generator) probabiliste(i int) agentmeshkafka.AgentMeshExchange {
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-p-%04d", i),
		IntentDescription: "high-cardinality multi-step intent " + fmt.Sprintf("%d", g.rng.Int64()),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID: fmt.Sprintf("agent-%04d", i), HumanInLoop: false, Organization: "org-p", DelegationDepth: 2,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID: "tool-p", DiscoveryMode: "dynamic_a2a",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur: "ietf", Reference: "draft-x", Marqueur: "Probable",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func (g *Generator) hysteresis(i int) agentmeshkafka.AgentMeshExchange {
	// Mid-range intents engineered to land near the (Deterministe, Probabiliste)
	// threshold. Used to seed the campaign with realistic ambiguity.
	return agentmeshkafka.AgentMeshExchange{
		ID:                fmt.Sprintf("synth-h-%04d", i),
		IntentDescription: "boundary " + fmt.Sprintf("%d", i),
		Initiator: agentmeshkafka.AgentMeshPrincipal{
			ID: fmt.Sprintf("agent-%04d", i), HumanInLoop: false, Organization: "org-h", DelegationDepth: 1,
		},
		Target: agentmeshkafka.AgentMeshCapability{
			ID: "tool-h", DiscoveryMode: "dynamic_mcp", ContractURI: "https://api/v1/op",
		},
		AttestationInstitutionnelle: &agentmeshkafka.AgentMeshAttestation{
			Emetteur: "ietf", Reference: "draft-x", Marqueur: "Probable",
			AssertedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}
```

- [ ] **Étape 3 — Écrire `main.go`**

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		seed   = flag.Int64("seed", 1, "deterministic seed for the generator")
		count  = flag.Int("count", 100, "number of exchanges to generate")
		output = flag.String("output", "-", "output path (- for stdout)")
	)
	flag.Parse()
	if *count < 1 {
		fmt.Fprintln(os.Stderr, "count must be >= 1")
		os.Exit(2)
	}
	w := os.Stdout
	if *output != "-" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create %s: %v\n", *output, err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	g := NewGenerator(*seed)
	if err := g.Generate(w, *count, DefaultDistribution()); err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Étape 4 — Écrire `generator_test.go`**

```go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestGenerator_Determinism(t *testing.T) {
	t.Parallel()
	var a, b bytes.Buffer
	if err := NewGenerator(42).Generate(&a, 100, DefaultDistribution()); err != nil {
		t.Fatal(err)
	}
	if err := NewGenerator(42).Generate(&b, 100, DefaultDistribution()); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Fatal("same seed produced different output (determinism broken)")
	}
}

func TestGenerator_DifferentSeedsDiffer(t *testing.T) {
	t.Parallel()
	var a, b bytes.Buffer
	_ = NewGenerator(1).Generate(&a, 100, DefaultDistribution())
	_ = NewGenerator(2).Generate(&b, 100, DefaultDistribution())
	if a.String() == b.String() {
		t.Fatal("seed=1 and seed=2 produced identical output")
	}
}

func TestGenerator_LineCount(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	_ = NewGenerator(7).Generate(&buf, 100, DefaultDistribution())
	got := strings.Count(buf.String(), "\n")
	if got != 100 {
		t.Fatalf("line count = %d, want 100", got)
	}
}

func TestGenerator_FrozenHash_Seed1(t *testing.T) {
	t.Parallel()
	// Frozen hash guards against accidental drift of the synthetic corpus
	// that ships in testdata/synthetic-100.jsonl (M4.4 step 5).
	var buf bytes.Buffer
	_ = NewGenerator(1).Generate(&buf, 100, DefaultDistribution())
	h := sha256.Sum256(buf.Bytes())
	got := hex.EncodeToString(h[:])
	const want = "TO_BE_FILLED_AT_FIRST_RUN"
	if want == "TO_BE_FILLED_AT_FIRST_RUN" {
		t.Skip("frozen hash not yet computed; fill on first green CI run")
	}
	if got != want {
		t.Fatalf("frozen hash drift: got=%s, want=%s", got, want)
	}
}
```

**Note d'agent** : exécuter `go run ./cmd/generate-corpus -seed 1 -count 100 -output internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl` une fois, calculer le SHA256, et remplir `want` dans le test ci-dessus.

- [ ] **Étape 5 — Générer le corpus checked-in**

```powershell
go run ./cmd/generate-corpus -seed 1 -count 100 -output internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl
Get-FileHash -Algorithm SHA256 internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl
```

Reporter le hash dans `TestGenerator_FrozenHash_Seed1`.

- [ ] **Étape 6 — Vérifier**

```powershell
go test -race -v ./cmd/generate-corpus/
golangci-lint run ./...
```

- [ ] **Étape 7 — Commit**

```powershell
git add cmd/generate-corpus/ internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl
git commit -m "feat(cmd/generate-corpus): deterministic synthetic AgentMesh corpus

M4.4: cmd/generate-corpus produces 100+ AgentMeshExchange across the
six dispatcher branches (Refus×3, Deterministe, Probabiliste,
hysteresis). Same seed → byte-identical output (calque FibGo §11).

Checked-in: testdata/synthetic-100.jsonl (seed=1, count=100). The
frozen-hash test guards against accidental drift.

Used by:
  - M4.5 (E2E integration test)
  - M4.6 (empirical campaign, Régime B contingency under PRD §18 risque #1)

Status: Hypothèse — synthetic distribution targets M3 dispatcher
behavior; calibration M5 may shift cut points.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.5 — E2E `test/e2e/agentmeshkafka_test.go` (build tag `integration`)

**Files :**
- Create: `test/e2e/agentmeshkafka_test.go`
- Create: `test/e2e/doc.go`

**Agent :** `ruflo-core:coder` (TDD)

### Note de conception

Build tag `integration` (PRD §15.1) : ne tourne pas en CI courte ; lancé explicitement par `make test-e2e` (à ajouter Makefile en M4.9) et par CI nocturne. Le test charge `synthetic-100.jsonl`, le rejoue via `app.StreamAsTauExchanges` dans `orchestration.Dispatcher`, agrège les régimes observés, vérifie qu'au moins une décision est tombée dans **chacun des trois régimes** `Deterministe`, `Probabiliste`, `Refus` (PRD §15.1 « ≥ 1 scénario par régime »).

- [ ] **Étape 1 — Écrire `test/e2e/doc.go`**

```go
// Package e2e holds end-to-end tests that exercise TauGo through its
// public app entry points. Tests are guarded by the `integration` build
// tag so they do not run in the CI short suite.
//
// Run: go test -race -tags=integration ./test/e2e/...
package e2e
```

- [ ] **Étape 2 — Écrire `agentmeshkafka_test.go`**

```go
//go:build integration

package e2e

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestE2E_SyntheticCorpus_AllRegimesObserved verifies PRD §15.1 ≥ 1 scenario
// per regime over the synthetic 100-trace corpus.
func TestE2E_SyntheticCorpus_AllRegimesObserved(t *testing.T) {
	t.Parallel()
	path, err := filepath.Abs("../../internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := agentmeshkafka.NewFileAdapter(path)
	if err != nil {
		t.Fatalf("NewFileAdapter: %v", err)
	}
	defer adapter.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	exchanges, _ := app.StreamAsTauExchanges(ctx, adapter, nil)

	d := app.NewDispatcher()
	counts := map[tau.Regime]int{}
	var n int
	for x := range exchanges {
		dec, err := d.Decide(ctx, x)
		if err != nil {
			t.Fatalf("decide on %s: %v", x.ID, err)
		}
		counts[dec.Regime]++
		n++
	}
	if n < 100 {
		t.Fatalf("ingested %d traces, want ≥ 100", n)
	}
	for _, r := range []tau.Regime{tau.Deterministe, tau.Probabiliste, tau.Refus} {
		if counts[r] == 0 {
			t.Errorf("regime %v not observed in synthetic corpus (PRD §15.1 ≥ 1 per regime)", r)
		}
	}
	t.Logf("E2E synthetic-100 distribution: Det=%d, Prob=%d, Refus=%d, Unknown=%d",
		counts[tau.Deterministe], counts[tau.Probabiliste], counts[tau.Refus], counts[tau.RegimeUnknown])
}

// TestE2E_RegimeA_AgentMeshKafka is the placeholder for the real-Kafka
// scenario. Skipped in Régime B; enabled in Régime A by setting the env
// var TAUGO_AGENTMESH_KAFKA_BROKERS.
func TestE2E_RegimeA_AgentMeshKafka(t *testing.T) {
	t.Parallel()
	t.Skip("Régime A (real AgentMeshKafka adapter) deferred to M4-bis; cf. ADR-0005 + PRD §18 risque #1")
}
```

- [ ] **Étape 3 — Vérifier**

```powershell
go build -tags=integration ./test/e2e/
go test -race -tags=integration -v ./test/e2e/
```

- [ ] **Étape 4 — Commit**

```powershell
git add test/e2e/
git commit -m "test(e2e): synthetic AgentMesh corpus exercises all 3 regimes

M4.5: introduces test/e2e/ guarded by the `integration` build tag.
Replays the 100-trace synthetic corpus through the production
Dispatcher and asserts ≥ 1 decision per regime (PRD §15.1).

The real-Kafka scenario (TestE2E_RegimeA_AgentMeshKafka) is a
documented placeholder skipped until M4-bis lands the real adapter
(ADR-0005 + PRD §18 risque #1).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.6 — Harness empirique I4 + classificateur

**Files :**
- Create: `internal/bridge/agentmeshkafka/empirical.go`
- Create: `internal/bridge/agentmeshkafka/empirical_test.go`

**Build tag :** `empirical` (sépare le harness des tests unitaires)

**Agent :** `ruflo-core:researcher` (classification) + `ruflo-core:coder` (harness)

### Note de conception

Le harness empirique ingère un fichier JSONL via `FileAdapter`, dispatche chaque trace, et **classifie** chaque résultat selon la grille PRD §6.2 :

| Catégorie | Définition |
|---|---|
| `Cohérent` | Dispatch produit la branche attendue d'après l'ID du fixture (`synth-rf-*` ⇒ Refus frontière, etc.) |
| `Faux positif I4` | `Decision.Regime == Refus` AND `Decision.Diagnostic == "I4 — combinaison incohérente détectée"` AND ID n'est pas `synth-r4-*` |
| `Faux négatif I4` | ID est `synth-r4-*` AND `Decision.Regime != Refus(I4)` |
| `Hors modèle` | aucune des catégories ci-dessus (à reporter dans `unmodeled.md`) |

Le harness est encapsulé dans une **commande** (`internal/bridge/agentmeshkafka/empirical.go` expose `RunCampaign(path string, w io.Writer) (Stats, error)` ; `cmd/tau/main.go` peut l'invoquer via `tau empirical I4 --corpus …` — déferré M5 le câblage CLI).

- [ ] **Étape 1 — Écrire `empirical.go`**

```go
//go:build empirical

package agentmeshkafka

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/tau"
)

// Stats aggregates the campaign verdicts.
type Stats struct {
	Total           int            `json:"total"`
	Coherent        int            `json:"coherent"`
	FauxPositifsI4  int            `json:"faux_positifs_i4"`
	FauxNegatifsI4  int            `json:"faux_negatifs_i4"`
	HorsModele      int            `json:"hors_modele"`
	RegimeDistribution map[string]int `json:"regime_distribution"`
	Unmodeled       []string       `json:"unmodeled,omitempty"`
	StartedAt       time.Time      `json:"started_at"`
	FinishedAt      time.Time      `json:"finished_at"`
}

// RunCampaign ingests path (JSONL), dispatches each trace, classifies, and
// writes a JSON report to w. Returns aggregated Stats.
//
// Status: Hypothèse — classification ID-prefix-based, valid only on the
// synthetic corpus produced by cmd/generate-corpus. On real AgentMesh
// traces, the prefix rule is replaced by an expectation oracle (M4-bis).
func RunCampaign(path string, w io.Writer) (Stats, error) {
	s := Stats{StartedAt: time.Now().UTC(), RegimeDistribution: map[string]int{}}
	adapter, err := NewFileAdapter(path)
	if err != nil {
		return s, fmt.Errorf("open corpus: %w", err)
	}
	defer adapter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	exchanges, _ := app.StreamAsTauExchanges(ctx, adapter, nil)
	d := app.NewDispatcher()

	for x := range exchanges {
		dec, err := d.Decide(ctx, x)
		if err != nil {
			return s, fmt.Errorf("decide %s: %w", x.ID, err)
		}
		s.Total++
		s.RegimeDistribution[regimeString(dec.Regime)]++
		switch classify(x, dec) {
		case clsCoherent:
			s.Coherent++
		case clsFauxPositifI4:
			s.FauxPositifsI4++
			s.Unmodeled = append(s.Unmodeled, fmt.Sprintf("FP-I4: %s — %s", x.ID, dec.Diagnostic))
		case clsFauxNegatifI4:
			s.FauxNegatifsI4++
			s.Unmodeled = append(s.Unmodeled, fmt.Sprintf("FN-I4: %s — got %s/%s", x.ID, regimeString(dec.Regime), dec.Diagnostic))
		case clsHorsModele:
			s.HorsModele++
			s.Unmodeled = append(s.Unmodeled, fmt.Sprintf("OOM: %s — %s/%s", x.ID, regimeString(dec.Regime), dec.Diagnostic))
		}
	}
	s.FinishedAt = time.Now().UTC()
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&s); err != nil {
		return s, err
	}
	return s, nil
}

type cls int

const (
	clsCoherent cls = iota
	clsFauxPositifI4
	clsFauxNegatifI4
	clsHorsModele
)

func classify(x tau.Exchange, dec tau.Decision) cls {
	expected := expectedBranch(x.ID)
	got := branch(dec)
	if expected != "" && expected == got {
		return clsCoherent
	}
	if got == "refus_i4" && expected != "refus_i4" {
		return clsFauxPositifI4
	}
	if expected == "refus_i4" && got != "refus_i4" {
		return clsFauxNegatifI4
	}
	return clsHorsModele
}

func expectedBranch(id string) string {
	switch {
	case strings.HasPrefix(id, "synth-rf-"):
		return "refus_frontiere"
	case strings.HasPrefix(id, "synth-r3-"):
		return "refus_i3"
	case strings.HasPrefix(id, "synth-r4-"):
		return "refus_i4"
	case strings.HasPrefix(id, "synth-d-"):
		return "deterministe"
	case strings.HasPrefix(id, "synth-p-"):
		return "probabiliste"
	case strings.HasPrefix(id, "synth-h-"):
		return "" // hysteresis: no fixed expectation
	}
	return ""
}

func branch(dec tau.Decision) string {
	switch dec.Regime {
	case tau.Deterministe:
		return "deterministe"
	case tau.Probabiliste:
		return "probabiliste"
	case tau.Refus:
		switch dec.Diagnostic {
		case "hors frontière τ":
			return "refus_frontiere"
		case "I3 — verrou ontologique D-AUTORITÉ":
			return "refus_i3"
		case "I4 — combinaison incohérente détectée":
			return "refus_i4"
		default:
			return "refus_other"
		}
	}
	return "unknown"
}

func regimeString(r tau.Regime) string {
	switch r {
	case tau.Deterministe:
		return "deterministe"
	case tau.Probabiliste:
		return "probabiliste"
	case tau.Refus:
		return "refus"
	default:
		return "unknown"
	}
}
```

- [ ] **Étape 2 — Écrire `empirical_test.go`**

```go
//go:build empirical

package agentmeshkafka_test

import (
	"bytes"
	"testing"

	"github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
)

func TestRunCampaign_Synthetic100_AllBranchesPresent(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	s, err := agentmeshkafka.RunCampaign("testdata/synthetic-100.jsonl", &buf)
	if err != nil {
		t.Fatalf("RunCampaign: %v", err)
	}
	if s.Total < 100 {
		t.Fatalf("total = %d, want ≥ 100", s.Total)
	}
	// PRD §15.1 ≥ 1 scenario per regime.
	for _, r := range []string{"deterministe", "probabiliste", "refus"} {
		if s.RegimeDistribution[r] == 0 {
			t.Errorf("regime %s not observed", r)
		}
	}
	// The campaign must not be 100% coherent (or the synthetic distribution
	// would be too easy and would mask I4 verification value).
	// Tolerate up to 95% coherent — any synthetic miss-classification is
	// recorded in Unmodeled (anti-patron #4 garde).
	if s.Coherent == s.Total {
		t.Log("100% coherent — synthetic distribution may be too sanitized; review M4.6 branch builders")
	}
}
```

- [ ] **Étape 3 — Vérifier**

```powershell
go build -tags=empirical ./internal/bridge/agentmeshkafka/
go test -race -tags=empirical -v ./internal/bridge/agentmeshkafka/
```

- [ ] **Étape 4 — Commit**

```powershell
git add internal/bridge/agentmeshkafka/empirical.go internal/bridge/agentmeshkafka/empirical_test.go
git commit -m "feat(bridge/agentmeshkafka): empirical I4 campaign harness

M4.6: empirical.go (build tag 'empirical') ingests a JSONL corpus,
dispatches each trace through the production app.Dispatcher, and
classifies each verdict into 4 categories:

  - Cohérent      → expected branch == observed branch
  - FP I4         → Refus(I4) but no I4 expectation
  - FN I4         → I4 expectation but no Refus(I4)
  - Hors modèle   → none of the above (appended to Stats.Unmodeled)

Stats serialized as JSON for downstream report rendering (M4.7).

The classification is ID-prefix-based — valid for the synthetic corpus
produced by cmd/generate-corpus. Real AgentMesh traces will require an
expectation oracle (M4-bis).

Status: Hypothèse.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.7 — Rapports `docs/empirical/I4-report.md` + `docs/empirical/unmodeled.md` + `docs/empirical/I4-regime.md`

**Files :**
- Create: `docs/empirical/I4-report.md`
- Create: `docs/empirical/unmodeled.md`
- Create: `docs/empirical/I4-regime.md`

**Agent :** `ruflo-core:researcher`

### Squelette `I4-regime.md`

```markdown
# Régime de campagne I4 — décision

*Daté 2026-05-24. Reportée à M4.0 par `Explore`.*

## Verdict

- [ ] Régime A — AgentMeshKafka disponible : campagne sur traces réelles
- [x] Régime B — Contingence : campagne sur corpus synthétique seed=1
      *(à cocher après audit M4.0)*

## Justification

*(rapport de l'agent `Explore` M4.0 — copier/coller)*

## Conséquences

- Statut I4 visé : *Hypothèse → Probable* uniquement en Régime A.
- En Régime B : statut I4 reste *Hypothèse* ; rapport tag « campagne synthétique » ;
  un sous-plan M4-bis sera planifié quand AgentMeshKafka aura atteint le statut Probable.

## Renvois

- PRD §18 risque #1
- Plan : `docs/superpowers/plans/2026-05-24-M4-agentmeshkafka-bridge.md`
- ADR : `docs/adr/0005-agentmeshkafka-dto.md`
```

### Squelette `I4-report.md`

```markdown
# Rapport empirique I4 — campagne M4

> Généré le 2026-05-24. Régime : voir `I4-regime.md`. Outil : `internal/bridge/agentmeshkafka/empirical.go` (build tag `empirical`).

## Méthode

- Source : `testdata/synthetic-100.jsonl` (seed=1, 100 traces, hash gelé `<à remplir>`)
  *(Régime B)* OU traces réelles AgentMeshKafka acquises le `<date>` *(Régime A)*
- Pipeline : `FileAdapter → app.StreamAsTauExchanges → app.NewDispatcher().Decide → classify()`
- Classification : prefix-based (ID synthétique) OU oracle externe (Régime A — M4-bis)

## Distribution des régimes observés

| Régime | Compte | % |
|---|---|---|
| Deterministe | `<n_det>` | `<pct>` |
| Probabiliste | `<n_prob>` | `<pct>` |
| Refus | `<n_refus>` | `<pct>` |

## Classification (priorité I4)

| Catégorie | Compte | % | Marqueur |
|---|---|---|---|
| Cohérent | `<n_coh>` | `<pct>` | — |
| Faux positifs I4 | `<n_fp_i4>` | `<pct>` | Hypothèse |
| Faux négatifs I4 | `<n_fn_i4>` | `<pct>` | Hypothèse |
| Hors modèle | `<n_oom>` | `<pct>` | Anti-patron #4 — voir `unmodeled.md` |

## Verdict I4

- **Effectif observé** : `<résumé>`
- **Statut V0 → V1** : *Hypothèse* → `<Probable | reste Hypothèse>`. Justification : `<résumé>`.
- **Veille** : ré-évaluation au prochain milestone calibration M5 + une fois par trimestre.

## Observations non modélisées

Voir `unmodeled.md`. Section initiale ci-dessous synthétisée :

`<lister 1-3 observations >`

## Renvois

- PRD §6.1 I4, §6.3 priorité empirique #1, §15.1 E2E, §18 risque #1
- Plan : `docs/superpowers/plans/2026-05-24-M4-agentmeshkafka-bridge.md`
- ADR : `docs/adr/0005-agentmeshkafka-dto.md`
- Régime : `docs/empirical/I4-regime.md`
```

### Squelette `unmodeled.md`

```markdown
# Observations non modélisées — registre vivant

*Anti-patron #4 PRD §7.2. Document vivant. Daté 2026-05-24.*

## Convention

Chaque entrée porte : ID trace · diagnostic · classification proposée · piste de modélisation · statut (Ouvert / Acté / Rejeté).

## Entrées initiales (M4 — campagne `<régime>`)

| Date | ID | Diagnostic | Classification | Piste | Statut |
|---|---|---|---|---|---|
| 2026-05-24 | `<à remplir post-campagne>` | `<…>` | Hors modèle | `<piste M5/M6>` | Ouvert |

## Indicateurs à modéliser (V2+)

- Hystérèse réelle observée vs zone théorique
- Variance D-INVARIANT sous variation `IntentDescription` (sonde S_reasoner_intent stub)
- Cas où `DelegationDepth == 0` mais `HumanInLoop == false` (chaîne courte, sans humain) — actuellement *Refus frontière* ?

## Renvois

- PRD §6.1 I4, §7.2 anti-patron #4
- `docs/empirical/I4-report.md`
```

- [ ] **Étape 1 — Compléter `I4-regime.md` avec le verdict M4.0**

- [ ] **Étape 2 — Exécuter le harness empirique** (Régime A ou B selon M4.0)

```powershell
# Régime B (synthétique) — la commande de référence M4
go test -tags=empirical -run TestRunCampaign -v ./internal/bridge/agentmeshkafka/
# Récupérer la sortie JSON, l'utiliser pour remplir I4-report.md
```

Pour le mode commande directe (intermédiaire) :

```powershell
# Hypothèse — À vérifier : on peut écrire un petit programme one-shot
# go run ./cmd/empirical-i4 -corpus internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl
# qui appelle agentmeshkafka.RunCampaign et imprime le JSON. À ajouter
# si nécessaire ; sinon le test ci-dessus suffit (rediriger logs).
```

- [ ] **Étape 3 — Remplir les trois rapports**

L'agent `ruflo-core:researcher` :

1. Récupère la sortie JSON de l'étape 2.
2. Remplit les comptes / pourcentages dans `I4-report.md`.
3. Liste 1-3 observations dans `unmodeled.md` (au minimum : « campagne synthétique seed=1 — toute observation est dérivée du générateur ; valeur empirique externe = nulle »).
4. Sélectionne le marqueur final I4 (en Régime B : reste *Hypothèse*).
5. Pas d'emoji, FR-CA, marqueurs d'incertitude.

- [ ] **Étape 4 — Commit**

```powershell
git add docs/empirical/
git commit -m "docs(empirical): I4 campaign report + unmodeled register + régime decision

M4.7: writes the three campaign artifacts:
  - I4-regime.md  — Régime A/B verdict from M4.0 audit
  - I4-report.md  — Method, regime distribution, 4-category classification
  - unmodeled.md  — anti-patron #4 register, initial entries

Régime B (contingency) keeps I4 at status Hypothèse; a Régime A
campaign on real AgentMesh traces is reserved for M4-bis.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.8 — `TestArchitectureLayering` étendu + `TestBridgeNoTauImport`

**Files :**
- Modify: `internal/arch_test.go` (optionnel — ajout d'une règle plus fine)
- Create: `internal/app/arch_extra_test.go` (test ciblé)

**Agent :** `ruflo-core:coder`

### Note

La règle `bridge/agentmeshkafka → tau interdit` existe déjà dans `arch_test.go` (ligne 32-34). M4.8 ajoute un **test contractuel positif** côté `app/` : *`app/agentmesh.go` est la seule occurrence qui importe à la fois `bridge/agentmeshkafka` et `tau`*. Cela transforme la convention en garde testée — anti-patron silencieux.

- [ ] **Étape 1 — Écrire `internal/app/arch_extra_test.go`**

```go
package app_test

import (
	"go/build"
	"strings"
	"testing"
)

// TestAppIsSoleBridgeTauPivot verifies that internal/app is the only
// package importing both internal/bridge/agentmeshkafka and internal/tau.
// Any other co-import would create a second étanchéité hole.
func TestAppIsSoleBridgeTauPivot(t *testing.T) {
	t.Parallel()
	const (
		bridgePkg = "github.com/agbruneau/taugo/internal/bridge/agentmeshkafka"
		tauPkg    = "github.com/agbruneau/taugo/internal/tau"
	)
	roots := []string{
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/calibration",
		"github.com/agbruneau/taugo/internal/tau/dimensions",
		"github.com/agbruneau/taugo/internal/tau/invariants",
	}
	for _, root := range roots {
		root := root
		t.Run(strings.ReplaceAll(root, "/", "_"), func(t *testing.T) {
			t.Parallel()
			pkg, err := build.Default.Import(root, ".", build.ImportComment)
			if err != nil {
				t.Skipf("not built: %v", err)
			}
			imports := append([]string{}, pkg.Imports...)
			imports = append(imports, pkg.TestImports...)
			seesBridge, seesTau := false, false
			for _, imp := range imports {
				if imp == bridgePkg {
					seesBridge = true
				}
				if imp == tauPkg || strings.HasPrefix(imp, tauPkg+"/") {
					seesTau = true
				}
			}
			if seesBridge && seesTau && root != "github.com/agbruneau/taugo/internal/app" {
				t.Fatalf("%s imports both bridge/agentmeshkafka and tau — pivot must remain in app/", root)
			}
		})
	}
}
```

- [ ] **Étape 2 — Vérifier**

```powershell
go test -race -v -run TestAppIsSoleBridgeTauPivot ./internal/app/
go test -race -run TestArchitectureLayering ./internal/
```

- [ ] **Étape 3 — Commit**

```powershell
git add internal/app/arch_extra_test.go
git commit -m "test(app): app is the sole bridge/agentmeshkafka ↔ tau pivot

M4.8: TestAppIsSoleBridgeTauPivot scans the candidate roots (orchestration,
calibration, dimensions, invariants) and fails if any of them co-imports
bridge/agentmeshkafka and tau. The étanchéité pivot must remain in
internal/app/ per ADR-0005.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.9 — Makefile : `make e2e` + `make empirical-i4`

**Files :**
- Modify: `Makefile`

**Agent :** `ruflo-core:coder`

- [ ] **Étape 1 — Ajouter les cibles au Makefile**

```makefile
.PHONY: e2e empirical-i4 generate-corpus

# E2E integration tests (build tag 'integration').
e2e:
	go test -race -tags=integration -v ./test/e2e/...

# Empirical I4 campaign harness (build tag 'empirical').
empirical-i4:
	go test -race -tags=empirical -v -run TestRunCampaign ./internal/bridge/agentmeshkafka/

# Regenerate the synthetic-100 corpus (deterministic; same seed → same bytes).
generate-corpus:
	go run ./cmd/generate-corpus -seed 1 -count 100 \
	    -output internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl
```

- [ ] **Étape 2 — Vérifier**

```powershell
make e2e
make empirical-i4
make generate-corpus
git diff internal/bridge/agentmeshkafka/testdata/synthetic-100.jsonl
# Doit être vide — déterminisme garanti par le test du frozen hash M4.4.
```

**Hypothèse — *À vérifier*** : `make` est disponible sur la machine Windows du dev. PRD §13 le mentionne ; si absent, `mingw32-make` ou les commandes `go ...` directes documentées en équivalent.

- [ ] **Étape 3 — Commit**

```powershell
git add Makefile
git commit -m "chore(makefile): e2e, empirical-i4, generate-corpus targets

M4.9: three new make targets to ease local execution of the M4
artefacts:
  - make e2e            → integration build tag, test/e2e/
  - make empirical-i4   → empirical build tag, RunCampaign on synth-100
  - make generate-corpus → regenerate testdata/synthetic-100.jsonl

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Tâche M4.10 — Revue intégrée + tag `v0.0.5-alpha` + CHANGELOG

**Agent :** thread principal (intégration) + `ruflo-core:reviewer`

- [ ] **Étape 1 — Vérifier la suite locale complète**

```powershell
go build ./...
go test -race -v ./...
go vet ./...
golangci-lint run ./...
go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/... ./...
go tool cover -func=coverage.out | Select-Object -Last 1
# Sécurité : ré-exécuter les builds tagués
go test -race -tags=integration ./test/e2e/...
go test -race -tags=empirical ./internal/bridge/agentmeshkafka/...
```

Couverture cible : ≥ 80 % global ; ≥ 90 % sur `internal/tau`. Le package `bridge/agentmeshkafka` peut tomber à ≈ 70 % si `RunCampaign` n'est pas couvert par défaut (gating par build tag) — accepté V1, à élever en M5.

- [ ] **Étape 2 — Briefing reviewer**

> Revue intégrée de M4 (commit range `v0.0.4-alpha..HEAD`). Vérifier :
>
> 1. `internal/bridge/agentmeshkafka/Adapter` : 2 méthodes (ISP ≤ 5 OK). DTO `AgentMeshExchange` neutre — pas d'import `tau`.
> 2. `FileAdapter` JSONL : Close idempotent ; cancellation propagée ; erreurs non-fatales sur canal séparé.
> 3. ADR-0005 présent. PRD §12.1 référencé (révision indiquée).
> 4. `internal/app/agentmesh.go` : conversion totale (`ToTauExchange`) ; mapping DiscoveryMode documenté ; fallback conservateur.
> 5. `TestArchitectureLayering` : règle `bridge/agentmeshkafka → tau` toujours active. `TestAppIsSoleBridgeTauPivot` : seul `app/` co-importe bridge+tau.
> 6. `cmd/generate-corpus` : déterminisme byte-identique (test `FrozenHash`).
> 7. `test/e2e/agentmeshkafka_test.go` : tag `integration` ; ≥ 1 décision par régime sur synthetic-100.
> 8. `internal/bridge/agentmeshkafka/empirical.go` : tag `empirical` ; classifie 4 catégories ; `Unmodeled` non vide en cas de hors-modèle.
> 9. `docs/empirical/I4-report.md` : marqueur d'incertitude présent ; régime A/B explicite ; renvoi `docs/empirical/unmodeled.md`.
> 10. `docs/empirical/unmodeled.md` : ≥ 1 entrée, format tableau respecté.
> 11. Aucun emoji, FR-CA pour docs/commentaires structurants, godoc anglais.
> 12. Anti-patrons : pas de `Predict*`, pas d'import LLM concret dans `bridge/agentmeshkafka` (interface `Adapter` étroite).
> 13. Aucun import du SDK Sarama, segmentio/kafka-go, IBM/sarama, ou TestContainers — discipline anti-platform tenue.

- [ ] **Étape 3 — Tag `v0.0.5-alpha`**

```powershell
git tag -a v0.0.5-alpha -m "M4: AgentMeshKafka adapter (JSONL mock) + empirical I4 campaign

M4.1  - Adapter interface + AgentMeshExchange DTO + ADR-0005
M4.2  - FileAdapter (JSONL mock); testdata sample
M4.3  - app/agentmesh.go: ToTauExchange + StreamAsTauExchanges
M4.4  - cmd/generate-corpus: deterministic synthetic 100-trace corpus
M4.5  - test/e2e/agentmeshkafka_test.go (build tag 'integration')
M4.6  - empirical.go: RunCampaign + 4-category classifier (build tag 'empirical')
M4.7  - docs/empirical/{I4-report.md, unmodeled.md, I4-regime.md}
M4.8  - TestAppIsSoleBridgeTauPivot
M4.9  - Makefile: e2e, empirical-i4, generate-corpus targets
M4.10 - integrated review + tag

Régime : voir docs/empirical/I4-regime.md.
Régime B (contingency, PRD §18 risque #1) reserves a real-Kafka campaign
to M4-bis.

Spec: PRD.md §6.1 I4, §12.1, §15.1, §18 risque #1.
Plan: docs/superpowers/plans/2026-05-24-M4-agentmeshkafka-bridge.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin main
git push origin v0.0.5-alpha
```

- [ ] **Étape 4 — Mettre à jour `CHANGELOG.md`**

```markdown
## [0.0.5-alpha] — 2026-05-24

### Ajouté

- `internal/bridge/agentmeshkafka/` : interface `Adapter` (Stream, Close) + DTO neutre `AgentMeshExchange`. Mock `FileAdapter` JSONL (déterministe ; tests golden).
- `internal/app/agentmesh.go` : `ToTauExchange` et `StreamAsTauExchanges` — pivot d'étanchéité forcé par ADR-0005.
- `cmd/generate-corpus/` : générateur déterministe (100 traces synthétiques couvrant les six branches du dispatcher).
- `test/e2e/agentmeshkafka_test.go` (build tag `integration`) : ≥ 1 décision par régime sur synthetic-100.
- `internal/bridge/agentmeshkafka/empirical.go` (build tag `empirical`) : `RunCampaign` + classification 4 catégories (Cohérent / FP I4 / FN I4 / Hors modèle).
- `internal/app/arch_extra_test.go` : `TestAppIsSoleBridgeTauPivot`.
- `docs/adr/0005-agentmeshkafka-dto.md` : DTO neutre + pivot app/.
- `docs/empirical/I4-report.md`, `unmodeled.md`, `I4-regime.md`.
- Makefile : `make e2e`, `make empirical-i4`, `make generate-corpus`.

### Statut empirique

- I4 : *Hypothèse* (Régime B — campagne synthétique). Une vraie campagne sur traces AgentMeshKafka reste à planifier en M4-bis si le projet `agbruneau/AgentMeshKafka` atteint le statut Probable.
```

```powershell
git add CHANGELOG.md
git commit -m "docs(changelog): M4 release notes (v0.0.5-alpha)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
git push origin main
```

---

## Branche contingence — variante Régime A (AgentMeshKafka disponible)

Si M4.0 verdict = Régime A, les tâches M4.4 / M4.6 / M4.7 sont **adaptées**, pas remplacées. Le synthétique reste utile (gold standard reproductible) ; on **ajoute** :

- **M4.4-A (parallèle à M4.4)** : `cmd/dump-agentmesh/main.go` — clone AgentMeshKafka, lit ≥ 100 traces réelles, les sérialise en JSONL `testdata/real-100.jsonl`. *Pas checked-in s'il contient PII* — sinon checked-in avec hash gelé. Statut : *À vérifier* sur PII.
- **M4.6-A** : `RunCampaign("testdata/real-100.jsonl", w)`. La classification basée sur préfixe ID n'est plus valable ; l'agent `ruflo-core:researcher` revoit manuellement chaque entrée puis acte un oracle externe (CSV side-channel) pour autonomiser le test. **Hypothèse — *À vérifier*** : faisable en une session d'agent ; sinon scinder en M4.6-A.1 (acquisition) et M4.6-A.2 (oracle).
- **M4.7-A** : `I4-report.md` peut promouvoir I4 de *Hypothèse* à *Probable* si la classification confirme :
  - Pas de faux positif I4 sur ≥ 50 traces hors-I4
  - ≥ 1 faux négatif documenté avec piste (entrée dans `unmodeled.md`)
  - Distribution des régimes plausible (au moins 20 % Refus, au moins 20 % Probabiliste)

Si l'une des conditions casse → statut reste *Hypothèse* et `unmodeled.md` enrichi.

**Workflow opérationnel** : si le verdict M4.0 bascule en cours de milestone, **redémarrer à M4.4** avec la variante A — pas de revert du synthétique (M4.5 reste vrai).

---

## Annexe — Risques M4 spécifiques

| # | Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|---|
| R1 | **AgentMeshKafka indisponible** (PRD §18 risque #1) | Probable | Élevé | Branche contingence Régime B documentée ; M4.0 décide ; M4-bis reporte la campagne réelle si nécessaire |
| R2 | **Signature PRD §12.1 incompatible avec `arch_test.go`** | Confirmé (constaté) | Moyen | ADR-0005 acté en M4.1 ; signature révisée (DTO + pivot `app/`) |
| R3 | **Mock JSONL non représentatif** : un mock fichier ne valide pas la dimension Kafka (offsets, partitions, backpressure) | Confirmé | Faible (V1 ne gère pas ces aspects) | Documenté ADR-0005 ; M4-bis prévoit le vrai adaptateur Kafka |
| R4 | **Distribution synthétique sanitisée** : 100 % Cohérent → aucune valeur empirique | Probable | Élevé | M4.6 teste `s.Coherent == s.Total` ; M4.4 inclut hysteresis et bord ambigus ; reviewer M4.10 valide la distribution observée |
| R5 | **Frozen hash dérive avec la version Go** : `math/rand/v2` peut changer son output entre minor releases | À vérifier | Moyen | Test marqué skip si hash vide ; calibrer une fois sur `go1.25.x`, regelé en cas de bump Go |
| R6 | **Dépendance accidentelle Sarama / TestContainers** : un agent pourrait re-introduire un client Kafka | Probable | Élevé | Build tag `integration` n'autorise pas les imports externes nouveaux ; revue M4.10 explicite cet item |
| R7 | **Classification ID-prefix biaisée** : si le générateur biaise la prédiction d'expectedBranch, FP I4 / FN I4 deviennent triviaux | Probable | Moyen | M4.4 étape 5 frozen-hash protège ; M4.6-A (Régime A) remplace la classification par un oracle externe |
| R8 | **PII dans traces réelles** (Régime A) | Probable | Élevé | Pas de checked-in sans audit PRD §14.1 ; anonymisation côté `cmd/dump-agentmesh` ; consigne explicite dans M4.4-A |
| R9 | **Couverture `bridge/agentmeshkafka < 80 %`** car `empirical.go` gated | Probable | Faible | Acceptée V1 ; M5 ajoute `cmd/empirical-i4` testé par défaut |
| R10 | **`test/e2e/` n'a pas de doc.go → erreur build** | Faible | Faible | M4.5 étape 1 crée `doc.go` explicitement |
| R11 | **`go.mod` ne contient pas le nouveau cmd** | Faible | Faible | `cmd/generate-corpus/main.go` réside dans le même module ; pas de require nouvelle |
| R12 | **AgentMesh DiscoveryMode mapping ambigu** : un mode inconnu force-t-il un bypass de frontière ? | Probable | Élevé | M4.3 mapping documenté : fallback `DynamicMCP` (dynamic-side) plutôt que `Static` → la frontière reste à l'intérieur de τ ; anti-patron #2/#4 gardé |

---

## Annexe — Self-review (à exécuter avant commit du plan)

- [x] **Couverture M4 high-level** : M4.0 (audit) — M4.1 (DTO + ADR) — M4.2 (FileAdapter) — M4.3 (pivot app/) — M4.4 (générateur synthétique) — M4.5 (E2E integration) — M4.6 (empirical harness) — M4.7 (rapports docs/empirical/) — M4.8 (garde pivot) — M4.9 (Makefile) — M4.10 (revue + tag).
- [x] **Granularité bite-sized** : chaque tâche < 200 LOC. M4.4 est la plus volumineuse (générateur + corpus + test frozen hash) — peut être splittée en M4.4a (generator.go + tests) / M4.4b (main.go + corpus checked-in) si l'agent estime nécessaire.
- [x] **Étanchéité** : `bridge/agentmeshkafka → tau` interdit préservé. ADR-0005 acté. `TestAppIsSoleBridgeTauPivot` ajouté (M4.8).
- [x] **Anti-patrons** : #1 (godoc anglais — no `Predict*`), #2 (mapping conservateur DiscoveryMode), #3 (pas de profil ici — M5), #4 (`Unmodeled` peuplé par `RunCampaign`), #5 (régime A/B explicité dans rapport, jamais inventé), #6 (aucun import LLM concret), #7 (pas de globaux mutables).
- [x] **Discipline anti-platform PRD §3.3** : aucune dépendance Kafka concrète (Sarama, segmentio, IBM, TestContainers). Mock fichier maison. R6 explicité.
- [x] **Contingence Régime A/B** : M4.0 décide ; M4.4-A / M4.6-A / M4.7-A documentés en variante.
- [x] **Marqueurs d'incertitude** : I4 statut *Hypothèse → Probable (Régime A) | reste Hypothèse (Régime B)* explicite. Hash gelé frozen *Hypothèse*. Distribution générateur *Hypothèse*.
- [x] **Pas d'emoji**, godoc anglais, `t.Parallel()` sur 100 % des nouveaux tests, FR-CA pour les commentaires structurants et le sous-plan lui-même.
- [x] **Cohérence des types** : `AgentMeshExchange`, `AgentMeshPrincipal`, `AgentMeshCapability`, `AgentMeshAttestation` distincts des `tau.*`. `Stats`, `Distribution`, `Generator` introduits dans le bon package.

---

*Sous-plan V1 — 2026-05-24. Référence : `PRDPlanning.md` §M4 + `PRD.md` §6.1 I4, §12.1, §15.1, §18 risque #1. Coordinateur : Claude Code thread principal. Exécutants : agent teams (`Explore`, `ruflo-core:coder`, `ruflo-core:researcher`, `ruflo-core:reviewer`).*
```

### Critical Files for Implementation

Les fichiers les plus critiques pour exécuter ce sous-plan M4 (toujours en chemins absolus) :

- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\bridge\agentmeshkafka\adapter.go` (à créer — DTO + interface, M4.1)
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\bridge\agentmeshkafka\file_adapter.go` (à créer — mock JSONL, M4.2)
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\app\agentmesh.go` (à créer — pivot d'étanchéité, M4.3)
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\cmd\generate-corpus\generator.go` (à créer — générateur synthétique déterministe, M4.4)
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\bridge\agentmeshkafka\empirical.go` (à créer — harness campagne, M4.6)

Fichiers de référence à lire avant chaque tâche :

- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\PRD.md` §6.1 I4, §12.1, §15.1, §18 risque #1
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\arch_test.go` lignes 32-34 (règle inviolable)
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\tau\operator.go` (types canoniques à mirrorer)
- `C:\Users\agbru\OneDrive\Documents\GitHub\TauGo\internal\orchestration\dispatcher.go` (consommateur du flux)
