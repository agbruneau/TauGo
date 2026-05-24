# ADR-0008 — Trace ventilée D-SENS / D-AUTORITÉ / D-INVARIANT

*Statut : Accepté · Daté 2026-05-24 · Auteurs : ruflo-swarm:architect + ruflo-core:researcher*

## Contexte

PRD §9.1 (lignes 429-430) spécifie explicitement que `tau.Trace` doit porter
trois scores dimensionnels :

```
Trace.DSens      dimensions.Score  // score D-SENS calculé étape 4
Trace.DAuthority dimensions.Score  // score D-AUTORITÉ calculé étape 2
Trace.DInvariant dimensions.Score  // score D-INVARIANT calculé étape 4
```

Ces champs sont absents de l'implémentation v0.1.0. Trois conséquences
opérationnelles en découlent (Confirmé AUDIT.md §9 P1-04) :

**Conséquence 1 — proxy imparfait dans I3.**
`EvaluateI3` (`i3_authority_asymmetry.go:73-76`) utilise `TauScore` comme
approximation de D-AUTORITÉ. Ce proxy fusionne les contributions des trois
dimensions ; il ne permet pas de distinguer un Exchange refusé pour
D-AUTORITÉ insuffisante d'un Exchange refusé pour combinaison défavorable.

**Conséquence 2 — bypass silencieux non détectable dans I4.**
`EvaluateI4` (`i4_coherence.go:15-41`) vérifie la cohérence `i ≈ pendant
⟹ s ≈ pendant` mais lit `Regime` et `Diagnostic` depuis la Trace, non les
scores dimensionnels bruts. Sans `DAuthority` et `DSens` ventilés, un
opérateur modifiant les poids de calibration peut contourner silencieusement
la condition de refus I4 (Confirmé AUDIT.md §4).

**Conséquence 3 — duplication dans `calibration.simulate`.**
`calibrate.go:112-126` reproduit partiellement la logique du dispatcher pour
recalculer les scores, faute de pouvoir lire `Trace.DSens/DInvariant` depuis
la `Decision` retournée. Cela constitue la duplication D4 (AUDIT.md §13).

L'enrichissement est par ailleurs requis par le chemin V0.3 (TUI replay) et
la migration V0.2 vers `cia-runtime` (AUDIT.md §17).

Un risque d'import cycle existe entre `internal/tau` (qui héberge `Trace`) et
`internal/tau/dimensions` (qui héberge `Score`) : `tau/dimensions` importe
déjà `tau` pour accéder à `Exchange`. Faire dépendre `tau` de `tau/dimensions`
crée un cycle interdit. Ce point doit être résolu avant implémentation
*(Hypothèse — à confirmer en T-015)*.

## Décision

Enrichir `tau.Trace` additivement avec trois champs de type `tau.Score` :

```go
// Score is a ventilated dimension score with optional provenance.
// It is promoted to the tau package to break the import cycle between
// tau and tau/dimensions.
//
// Implementation v0.1.1 (Confirmé — supérieur à la spec ADR initiale qui
// envisageait un simple type Score float64) :
type Score struct {
    Value      float64   // valeur normalisée [0,1]
    Probes     []string  // optionnel — noms des sondes ayant produit le score
    Weights    []float64 // optionnel — poids appliqués lors de l'agrégation
    ComputedAt time.Time // optionnel — instant de calcul (instrumentation)
}

// Trace carries instrumentation data for a single Decide call.
type Trace struct {
    // ... champs existants préservés sans modification ...

    // DSens is the D-SENS dimension score computed at dispatcher step 4.
    // nil indicates the score was not computed (e.g. stub LLM path).
    DSens *Score `json:"d_sens,omitempty"`

    // DAuthority is the D-AUTORITÉ dimension score computed at dispatcher step 2.
    DAuthority *Score `json:"d_authority,omitempty"`

    // DInvariant is the D-INVARIANT dimension score computed at dispatcher step 4.
    DInvariant *Score `json:"d_invariant,omitempty"`
}
```

**Résolution de l'import cycle** : `Score` est promu dans le package `tau`
comme struct riche (Value + provenance). Le package `tau/dimensions` déclare
un alias `type Score = tau.Score` afin de conserver la compatibilité de ses
signatures publiques. Les champs `Trace.DSens/DAuthority/DInvariant` sont
des **pointeurs** (`*Score`) afin que `omitempty` JSON fonctionne correctement
sur un struct non-vide. Cette promotion est minimaliste : aucune méthode ni
constante n'est ajoutée dans `tau` au-delà du type *(Hypothèse — à valider
que l'alias dans `tau/dimensions` ne crée pas de nouvelle violation
arch_test ; T-015 tranche)*.

**Peuplement runtime** (AUDITPlan.md T-016) :
- Étape 2 du dispatcher (`dispatcher.go:104-110`) : assigner `decision.Trace.DAuthority` avant le test de refus I3.
- Étape 4 (`dispatcher.go:121-129`) : assigner `decision.Trace.DSens` et `decision.Trace.DInvariant` après calcul.

**Impact invariants** :
- `EvaluateI3` lira `trace.DAuthority` directement au lieu du proxy `TauScore`.
- `EvaluateI4` pourra vérifier la cohérence sur les scores dimensionnels ventilés.

**Compatibilité JSON** : les champs sont taggés `omitempty`. Un objet
`Decision` sérialisé par v0.1.0 sans ces champs reste désérialisable par
v0.1.1 ; les champs absents sont zéros. Le tag `v0.1.0` reste sémantiquement
valide (Confirmé AUDIT.md §19 footer).

## Conséquences

**Positives :**

- `EvaluateI3` dispose du score exact D-AUTORITÉ ; le proxy `TauScore` est abandonné.
- `EvaluateI4` peut détecter un bypass silencieux via les scores ventilés.
- `calibration.simulate` peut lire `Decision.Trace.DSens/DInvariant` au lieu de dupliquer la logique dispatcher (D4 résolu).
- La Trace devient exploitable pour V0.3 (TUI replay) et V0.2 (mécanisation Lean) sans re-calcul.
- Champs `omitempty` : aucune rupture JSON entre v0.1.0 et v0.1.1.

**Négatives :**

- La promotion de `Score` dans `tau` élargit légèrement l'API publique du package kernel (mineur : 1 type sans méthode).
- Le package `tau/dimensions` doit mettre à jour ses signatures pour utiliser l'alias — audit préalable requis en T-015.
- Les tests existants qui comparent une `Decision` par égalité de struct doivent tolérer les nouveaux champs (valeur zéro par défaut — non impactant si les tests utilisent des comparaisons de champ).

## Alternatives rejetées

1. **Garder le proxy `TauScore`** — anti-patron empirique documenté
   (AUDIT.md §9 P1-04). Le proxy fusionne D-SENS, D-AUTORITÉ et D-INVARIANT
   de façon non-séparable ; l'observabilité du refus I3 est dégradée pour
   l'opérateur. Rejeté.

2. **Wrapper externe `DimensionScores`** — structure additionnelle portée par
   `Decision` plutôt que par `Trace`. Couplage plus faible sur `Trace`, mais
   l'explicabilité est dégradée : les scores ne sont plus colocalisés avec
   le régime et le diagnostic dans la même valeur immuable. Rejeté.

3. **Interface `DimensionScorer` injectée dans le dispatcher**  — trop
   indirect pour un enrichissement additif sur un type valeur. Complexité
   non justifiée pour V0.1.1. Rejeté.

4. **Package intermédiaire `internal/tau/scores`** — résout le cycle
   différemment, mais introduit un troisième sous-package dans `tau/` pour
   un seul type `float64`. Surdimensionné. Rejeté en faveur de la promotion
   directe dans `tau`.

## Renvois

- AUDIT.md §9 P1-04 (Trace sans scores ventilés — conséquences I3, I4, simulate)
- AUDIT.md §13 D4 (duplication `simulate` ↔ `Decide`)
- AUDIT.md §3.2 V-A4 (`ProfileVersion` hardcodé — corollaire instrumentation)
- AUDIT.md §18 R3 (recommandation Trace ventilée)
- PRD.md §9.1 (spécification `Trace.DSens/DAuthority/DInvariant`)
- PRD.md §6 (invariants I3 et I4 — conditions de refus)
- CLAUDE.md §Invariants & dimensions (résumé exécutable)
- ADR-0001 (`docs/adr/0001-clean-architecture-4-layers.md`) — fondation étanchéité
- ADR-0005 (`docs/adr/0005-agentmeshkafka-dto.md`) — modèle de format
- AUDITPlan.md T-007 (ADR), T-015 (enrichissement Trace), T-016 (peuplement dispatcher)

*Statut : Accepté.*
