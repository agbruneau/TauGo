# ADR-0006 — Types valeur transverses (`internal/thresholds/`)

*Statut : Accepté · Daté 2026-05-24 · Auteurs : ruflo-swarm:architect + ruflo-core:researcher*

## Contexte

Trois définitions sémantiquement équivalentes de `Thresholds` coexistent dans
la base de code v0.1.0 (Confirmé AUDIT.md §9 P1-01 et §13 D1) :

- `tau.TraceThresholds` — `internal/tau/operator.go:71-77`
- `orchestration.Thresholds` — `internal/orchestration/thresholds.go:5-11`
- `calibration.Thresholds` — `internal/calibration/profile.go:19-26`

Les deux premières exposent `TauMin` et `TauMax`. La troisième ajoute
`HysteresisGap` (bande hystérèse), qui n'est présent que côté calibration.
Ce drift est démontré : le dispatcher (`dispatcher.go:142-145`) ne peut lire
`HysteresisGap` sans importer `calibration`, ce qu'ADR-0001 interdit à cette
couche. La triplication n'est pas un choix architectural documenté : elle
résulte d'une croissance organique entre M1 et M3.

L'absence d'un type canonique unique ouvre la porte à des divergences de
champs silencieuses lors de tout refactor futur. ADR-0001 pose le principe de
couches étanches, mais ne prévoyait pas de couche *transverse* pour les types
valeur partagés. La présente ADR comble ce vide.

## Décision

Créer le package `internal/thresholds/` comme couche transverse de types
valeur, sans dépendance vers aucun autre package `taugo`.

Le package expose un type unique :

```go
// Package thresholds provides the canonical Thresholds value type shared
// across tau, orchestration, and calibration layers.
package thresholds

// Thresholds holds the decision boundary configuration for the τ operator.
type Thresholds struct {
    TauMin        float64 // refus si TauScore < TauMin
    TauMax        float64 // acceptation déterministe si TauScore >= TauMax
    HysteresisGap float64 // bande hystérèse [TauMin, TauMin+HysteresisGap]
}
```

Les trois packages consommateurs déclarent des **alias de type** durant la
transition v0.1.1, de façon à ne briser aucun nom exporté existant :

```go
// internal/orchestration/thresholds.go
type Thresholds = thresholds.Thresholds

// internal/calibration/profile.go
type Thresholds = thresholds.Thresholds

// internal/tau/operator.go  (TraceThresholds → alias ou suppression si non
// exporté hors tau — à trancher en T-014)
type TraceThresholds = thresholds.Thresholds
```

L'étanchéité descendante est garantie par règle ajoutée dans
`internal/arch_test.go` : `internal/thresholds` n'importe aucun package
`taugo` (Confirmé : vérifiable à la compilation).

## Conséquences

**Positives :**

- Un seul `type Thresholds struct` dans la base de code : `grep -rn "type Thresholds struct" internal/` retourne exactement 1 occurrence.
- `HysteresisGap` accessible par le dispatcher via `thresholds.Thresholds` sans violer ADR-0001 (pas d'import `calibration → orchestration`).
- `internal/arch_test.go` étendu avec deux nouvelles règles additives :
  - `tau` → `internal/thresholds` : autorisé
  - `orchestration` → `internal/thresholds` : autorisé
  - `calibration` → `internal/thresholds` : autorisé
  - `internal/thresholds` → tout package taugo : interdit

**Négatives :**

- Introduction d'un package supplémentaire (faible : 1 fichier, 0 dépendance).
- Les alias de type ne permettent pas d'ajouter de méthodes sur le type depuis les packages consommateurs — acceptable pour un type valeur pur.
- La suppression éventuelle de `TraceThresholds` (si non exporté hors `tau`) fera l'objet d'une vérification en T-014 avant décision.

## Alternatives rejetées

1. **Interface partagée `ThresholdsProvider`** — trop indirecte pour un simple
   struct de configuration immuable. Ajoute de la complexité sans bénéfice.
   Rejeté.

2. **Duplication assumée** — le drift `HysteresisGap` est démontré (Confirmé
   AUDIT.md §13 D1). Toute modification ultérieure risque de diverger
   silencieusement entre les trois sites. Rejeté.

3. **Fusion dans `internal/tau/`** — violerait ADR-0001 : `calibration` et
   `orchestration` pourraient importer `tau`, mais cela introduit une
   dépendance supplémentaire du kernel, contraire à la doctrine
   CLAUDE.md §Doctrine (τ ne dépend de rien). Rejeté.

## Renvois

- AUDIT.md §9 P1-01 (triple `Thresholds` + drift `HysteresisGap`) et §13 D1
- AUDIT.md §18 R1 (recommandation extraire `internal/thresholds/`)
- PRD.md §8 (Clean Architecture 4 couches) et §8.1 (étanchéité)
- CLAUDE.md §Architecture (étanchéité gardée par `arch_test.go`)
- ADR-0001 (`docs/adr/0001-clean-architecture-4-layers.md`) — fondation
- ADR-0005 (`docs/adr/0005-agentmeshkafka-dto.md`) — modèle de format
- `internal/arch_test.go` — règles additives à ajouter (T-014)
- AUDITPlan.md T-005 (ADR) et T-014 (implémentation)

*Statut : Accepté.*
