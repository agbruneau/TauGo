# ADR-0009 — Types d'erreurs typées (`internal/errors` peuplé)

*Statut : Accepté · Daté 2026-05-24 · Auteurs : ruflo-swarm:architect*

## Contexte

PRD §14.2 garantit l'existence de trois types d'erreurs nommés :
`DispatchError`, `RefusError` et `CalibrationError`. Ces types permettent à
l'appelant d'inspecter la cause structurée d'un refus ou d'une défaillance
via `errors.Is`/`errors.As`, conformément aux conventions Go idiomatiques.

L'implémentation v0.1.0 ne respecte pas cette promesse (Confirmé AUDIT.md §9
P1-02) :

- `internal/errors/doc.go` contient 4 LOC de documentation — aucun type
  défini. Le package est mort depuis M0.
- Tous les sites d'usage (`internal/orchestration/dispatcher.go`,
  `internal/app/app.go`, `internal/calibration/calibrate.go`) retournent soit
  `fmt.Errorf("%w", err)`, soit `errors.New(...)` avec des chaînes littérales.
- `app.selectLLM` (`app.go:26-31`) panique au lieu de retourner une erreur
  (AUDIT.md §10 P2-11).

L'absence de types structurés empêche les tests d'utiliser `errors.As` pour
vérifier le contexte d'un refus (étape dispatcher, identifiant d'échange,
diagnostic). Elle empêche également les appelants CLI d'afficher un message
d'erreur différencié selon l'origine de l'échec.

## Décision

Peupler `internal/errors/errors.go` (remplaçant `internal/errors/doc.go`) avec
les trois types promis par PRD §14.2, leurs méthodes standard et des sentinels
exportés :

```go
// Package errors provides structured error types for the τ operator kernel.
// Use errors.Is and errors.As to inspect error causes.
package errors

import "fmt"

// DispatchError is returned when the dispatcher fails at a specific stage.
type DispatchError struct {
    Stage      int    // étape dispatcher 1-8 (cf. PRD §10)
    Cause      error
    ExchangeID string
    Detail     string
}

func (e *DispatchError) Error() string {
    return fmt.Sprintf("dispatch error at stage %d (exchange %s): %s: %v",
        e.Stage, e.ExchangeID, e.Detail, e.Cause)
}

func (e *DispatchError) Unwrap() error { return e.Cause }

// RefusError is returned when Decide produces a Refus decision.
// It carries the stage at which the refusal was triggered and the diagnostic.
type RefusError struct {
    Stage      int    // étape dispatcher émettant le refus (1-8)
    Diagnostic string // sentinel tau.Diag* (cf. PRD §9.1)
    ExchangeID string
}

func (e *RefusError) Error() string {
    return fmt.Sprintf("refus at stage %d (exchange %s): %s",
        e.Stage, e.ExchangeID, e.Diagnostic)
}

// RefusError has no Unwrap: a refusal is a terminal decision, not a wrapped cause.

// CalibrationError is returned when a calibration operation fails.
type CalibrationError struct {
    ProfileVersion string
    Cause          error
}

func (e *CalibrationError) Error() string {
    return fmt.Sprintf("calibration error (profile %s): %v",
        e.ProfileVersion, e.Cause)
}

func (e *CalibrationError) Unwrap() error { return e.Cause }

// Sentinel errors for use with errors.Is.
var (
    ErrFrontiereFranchie  = &RefusError{Diagnostic: "frontiere_franchie"}
    ErrPeremptionProfile  = &RefusError{Diagnostic: "profil_perime"}
    ErrIncoherenceI4      = &RefusError{Diagnostic: "incoherence_i4"}
)
```

**Adoption progressive** : l'implémentation complète de tous les sites
d'usage est déférée. V0.1.1 couvre :
- T-018 : au moins un site dans `dispatcher.go` retourne `*errors.RefusError`
  au lieu d'une chaîne littérale.
- T-027 : `app.selectLLM` retourne `*errors.DispatchError{Stage: 0}` au lieu
  de paniquer.
- La migration des sites restants suit dans les lots suivants
  (T-027 `selectLLM`, couverture `internal/errors` ≥ 90 %).

## Conséquences

**Positives :**

- `errors.Is`/`errors.As` fonctionnels sur les trois types.
- Les tests peuvent vérifier `Stage`, `Diagnostic` et `ExchangeID` sans
  inspecter des chaînes littérales.
- PRD §14.2 tenu.
- `app.selectLLM` cesse de paniquer en production — anti-patron `panic`
  hors invariant interne cassé résolu (CLAUDE.md §Conventions de code).
- Couverture `internal/errors` mesurable et gatable (cible ≥ 90 %).

**Négatives :**

- Migration progressive : pendant la transition, les sites non encore migrés
  coexistent avec les anciens `fmt.Errorf` — les tests unitaires de ces sites
  ne bénéficient pas encore de `errors.As`.
- Les sentinels `ErrFrontiereFranchie` etc. sont des pointeurs (`*RefusError`) :
  `errors.Is` compare par adresse, non par valeur. Les appelants doivent
  utiliser `errors.As` pour inspecter les champs, et `errors.Is` uniquement
  pour les sentinels reconnus.

## Alternatives rejetées

1. **Erreurs sentinelles globales pures (`var ErrFoo = errors.New("foo")`)** —
   perte de structure. L'appelant ne peut pas extraire `Stage`,
   `ExchangeID` ni `Diagnostic` sans parser la chaîne. Incompatible avec
   PRD §14.2. Rejeté.

2. **`github.com/pkg/errors`** — dépendance externe dépréciée depuis Go 1.13
   (wrapping natif `%w`). Ajoute un module tiers sans gain fonctionnel.
   Rejeté.

3. **Types d'erreurs dans chaque package (`tau.RefusError`,
   `orchestration.DispatchError`)** — fragmentation. Les appelants CLI
   devraient connaître les types internes de chaque package. La couche
   `internal/errors` est précisément prévue par PRD §8 comme dépendance
   commune (Confirmé CLAUDE.md §Architecture, colonne `errors`). Rejeté.

4. **Peupler `internal/errors` mais sans sentinels** — les tests ne pourraient
   pas utiliser `errors.Is` pour les cas fréquents (péremption, frontière).
   Moitié du bénéfice perdu. Rejeté.

## Renvois

- AUDIT.md §9 P1-02 (packages morts + absence types typés)
- AUDIT.md §10 P2-11 (`app.selectLLM` panic)
- AUDIT.md §18 R2 (recommandation peupler `internal/errors`)
- PRD.md §14.2 (types `DispatchError`, `RefusError`, `CalibrationError` promis)
- PRD.md §8 (Clean Architecture — `errors` comme dépendance commune)
- CLAUDE.md §Conventions de code (pas de panic sauf invariant interne cassé)
- CLAUDE.md §Architecture (étanchéité Clean Architecture)
- ADR-0001 (`docs/adr/0001-clean-architecture-4-layers.md`) — fondation
- ADR-0005 (`docs/adr/0005-agentmeshkafka-dto.md`) — modèle de format
- AUDITPlan.md T-008 (ADR), T-018 (implémentation), T-027 (migration `selectLLM`)

*Statut : Accepté.*
