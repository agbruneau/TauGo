# 03 — L'opérateur τ — renvoi vers chap. III.8.3

*Document de renvoi croisé. Le verbatim canonique vit dans `agbruneau/InteroperabiliteAgentique` v2.4.3, `Monographie.md` chap. III.8.3 (lignes ~5588-5606).*

## Définition (chap. III.8.3.1)

τ déplace l'instant de fixation des grandeurs d'interopérabilité (sens, autorité, support d'invariant) de l'avant-interaction vers l'interaction :

`τ : t_fix(g) ≺ t_int ↦ t_fix(g) ≈ t_int`

où :
- `g` est une grandeur d'interopérabilité (sens, autorité, support d'invariant) ;
- `t_fix(g)` est l'instant où la grandeur est fixée ;
- `t_int` est l'instant de l'interaction ;
- `≺` dénote la précédence temporelle stricte ;
- `≈` dénote la simultanéité opérationnelle (la grandeur est tranchée *au cours* de l'échange, non avant).

L'agentivité, formellement, *est* l'application de τ à l'interopérabilité d'entreprise.

## Encodage TauGo

| Concept monographie | Encodage Go | Renvoi PRD |
|---|---|---|
| Grandeur `g` | Trois dimensions : D-SENS, D-AUTORITÉ, D-INVARIANT (`internal/tau/dimensions/`, M2+) | §5 |
| Instant `t_fix` | Position sur l'axe `[0, 1]` de chaque dimension | §2.2 |
| Frontière de validité | `FrontierCheck.Inside()` (`internal/tau/frontier.go`, M0) | §4.3 |
| Asymétrie ontologique D-AUTORITÉ | `Attestation` requise pour D-AUTORITÉ ≥ θ_auth_block | §4.4 |
| Sortie discrète | `Regime ∈ {Deterministe, Probabiliste, Refus}` (`internal/tau/operator.go`, M0) | §2.3 |

## Trois propriétés exploitables (chap. III.8.3.1)

1. **τ opère sur `t_fix`, jamais sur le contenu de `g`** → base de I1 (conservation). TauGo ne réécrit pas les capacités ; il décide *quand* leur résolution s'effectue.
2. **τ non trivial seulement si `t_fix(g) ≺ t_int` peut être strictement violé sans détruire `g`** → base de I2 (irréductibilité). TauGo n'applique τ qu'aux échanges dont la migration est elle-même possible.
3. **L'application de τ à une grandeur n'entraîne pas mécaniquement son application à une autre** → base de l'orthogonalité des trois dimensions. Les scores des trois dimensions sont calculés **indépendamment** ; seule contrainte explicite : I4 (cohérence dirigée `i ≈ pendant ⟹ s ≈ pendant`).

## Domaine de non-application — frontière de validité

τ ne s'applique qu'à la frontière où **les quatre conditions classiques sont toutes violées simultanément** (chap. III.8.3.2) :

| Condition classique | Violation requise |
|---|---|
| Univers de capacités clos et énumérable | Univers ouvert, capacités découvertes à l'exécution |
| Composition fixe à la conception | Composition variable à l'exécution |
| Pair non probabiliste (déterministe sous contrat) | Pair réellement probabiliste |
| Coût d'erreur borné et réversible | Coût d'erreur non borné et/ou irréversible |

Encodage exécutable : `internal/tau/frontier.go` (`FrontierCheck`, méthode `Inside()`). Garde anti-patron #2 (« hors frontière τ ») — voir [`PRD.md` §7.2](../../PRD.md).

## Ce que τ ne fait pas

- τ **ne décrit aucun comportement** du pair appelé (anti-patron #1, « usage prédictif »).
- τ **ne s'applique pas aux frontières classiques** où le pair est déterministe sous contrat (anti-patron #2, « hors frontière »).
- τ **n'est pas symétrique** sur ses trois dimensions : `τ_AUTORITÉ` rencontre une barrière ontologique (Searle 1995) que `τ_SENS` et `τ_INVARIANT` ne rencontrent pas — voir [`PRD.md` §4.4](../../PRD.md).

## Statut

*Confirmé pour la définition et les trois propriétés (chap. III.8.3 verbatim).*
*Probable pour l'encodage Go en M0 — sera éprouvé empiriquement à partir de M4 (campagne AgentMeshKafka).*

**Aligné monographie** : v2.4.3 (2026-05-21).
**Daté** : 2026-05-23.
