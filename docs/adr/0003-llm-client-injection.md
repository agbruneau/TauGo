# ADR-0003 — LLM client injecté via interface étroite

*Statut : Accepté · Daté 2026-05-24 (rétroactif M1) · Auteurs : thread principal + ruflo-swarm:architect*

## Contexte

La sonde `S_reasoner_intent` (dimension D-SENS, chap. III.8.3) nécessite d'interpréter
l'intention sémantique d'un échange agentique. Cette opération est fondamentalement
probabiliste : elle s'appuie sur un modèle de langage (LLM) pour produire un score
de sens `[0, 1]`.

Or, l'anti-patron PRD §12.2 / §7.2.6 l'interdit explicitement :

> Aucun import LLM concret (`anthropic`, `openai`, …) dans `internal/tau/*`
> ou `internal/orchestration/*`.

Cette interdiction est motivée par trois risques simultanés :

1. **Risque PRD §18 #5** — si le kernel τ importe un SDK LLM concret, il hérite
   de sa politique de retry, de ses timeouts et de ses conventions d'erreur.
   La sonde devient non-déterministe même en test unitaire.

2. **Testabilité CI** — la CI ne doit jamais appeler un service LLM externe. Un
   import concret dans `tau/*` rend impossible l'exécution de `make test` sans
   clé API ou mock de réseau.

3. **Calibration reproductible** — PRD §12.2 prescrit que le stub déterministe
   `internal/bridge/llm/stub.go` doit produire exactement les mêmes scores pour
   une entrée donnée, à des fins de golden tests et de benchmarks.

La tension est donc : le kernel a besoin d'une capacité LLM, mais ne doit pas
connaître l'implémentation concrète.

## Décision

TauGo adopte une **interface étroite `llm.Client`** définie dans
`internal/bridge/llm/`, injectée par `internal/app/` :

```go
// Package llm fournit l'abstraction du client LLM injecté dans le kernel τ.
// Toute implémentation concrète réside dans internal/app/ ou internal/bridge/llm/.
package llm

// Client est l'interface étroite exposée au kernel τ via la couche app.
// Deux méthodes maximum : Fingerprint et Interpret.
type Client interface {
    // Fingerprint retourne une empreinte déterministe de l'exchange
    // (utilisée pour le cache de calibration).
    Fingerprint(ctx context.Context, input string) (string, error)

    // Interpret retourne un score de sens [0, 1] pour l'input donné.
    Interpret(ctx context.Context, input string) (float64, error)
}
```

Règles d'application :

1. **Injection en `internal/app/`** — `app/` est la seule couche autorisée à
   instancier un client LLM concret (Anthropic, OpenAI, ou autre).
2. **Stub obligatoire** — `internal/bridge/llm/stub.go` implémente `Client` avec
   une table de correspondance `intent → score` checked-in. Tout test sans
   `TAUGO_LLM_BACKEND=real` utilise le stub.
3. **Garde `arch_test.go`** — aucun import `anthropic`, `openai` ou tout autre
   SDK LLM externe n'est autorisé dans `internal/tau/*` ou
   `internal/orchestration/*`. Violation détectée à `go test ./internal/...`.
4. **Interface stable jusqu'à V1** — toute modification de signature nécessite
   une revue car elle est un point de rupture API pour les implémentations concrètes.

## Conséquences

**Positives :**
- Le kernel τ est indépendant de tout fournisseur LLM. Remplacement d'Anthropic
  par un modèle local (ollama, llama.cpp) sans modification de `internal/tau/*`.
- `make test` et `make fuzz` s'exécutent sans clé API ni accès réseau — contrainte
  CI satisfaite sur les 3 OS.
- La calibration est reproductible : le stub déterministe garantit que deux exécutions
  de `make benchmark` sur le même corpus produisent des scores identiques.
- L'interface à 2 méthodes (ISP — PRD §11.3) minimise la surface de contrat.

**Négatives :**
- Une couche d'indirection supplémentaire : `tau` → interface → `app` → SDK concret.
  Coût = ~1 allocation supplémentaire par appel LLM dans le chemin chaud.
- Le stub doit être maintenu en synchronisation avec le comportement du LLM réel
  pour que les golden tests restent significatifs. Dérive possible si le modèle
  change de comportement.
- L'injection par `app/` impose que les tests d'intégration (`test/e2e/`) soient
  sous build tag `integration` et fournissent leur propre implémentation concrète.

**Neutres :**
- Le stub fait < 100 LOC — charge de maintenance négligeable.

## Alternatives rejetées

1. **Import direct de l'Anthropic SDK dans `internal/tau/`** — solution la plus
   simple en apparence, mais viole anti-patron PRD §7.2.6 et §3.3 (anti-objectifs).
   Couplage non testable en CI. Rejet immédiat M1.

2. **Plugin dynamique (DLL / `.so`)** — aurait isolé le code concret du kernel,
   mais introduit une complexité de chargement dynamique sans bénéfice pour V1.
   Le build cross-platform (Linux/macOS/Windows) devient fragile. Rejet : complexité
   gratuite pour un projet n'ayant pas encore de cas d'usage de hot-swap en production.

3. **Service LLM externe via gRPC** — isole totalement le LLM dans un processus
   séparé. Overkill pour V1 ; ajoute latence et infrastructure réseau en test.
   Non écarté définitivement pour V2+ si le kernel devient distribué.

## Renvois

- PRD §12.2 (interface llm.Client et injection)
- PRD §7.2.6 (anti-patron #6 — import LLM concret interdit dans tau/*)
- PRD §18 risque #5 (LLM fuite l'abstraction probabiliste)
- PRD §11.3 (ISP — interfaces étroites ≤ 5 méthodes)
- `internal/bridge/llm/` (interface + stub)
- `internal/app/` (injection et implémentation concrète)
- `internal/arch_test.go` (garde statique imports LLM)
- ADR-0001 (Clean Architecture — étanchéité que cette ADR renforce)
- ADR-0005 (DTO — pattern analogue pour AgentMeshKafka)

*Statut : Confirmé — stub actif depuis M1, aucune fuite LLM détectée par arch_test.*
