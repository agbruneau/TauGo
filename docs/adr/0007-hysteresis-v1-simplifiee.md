# ADR-0007 — Hystérèse V1 simplifiée (sans mémoire `LastRegime`)

*Statut : Accepté — V1 simplifiée déclarée · Daté 2026-05-24 · Auteurs : ruflo-swarm:architect*

## Contexte

PRD §10 étape 7 spécifie le comportement suivant pour la bande hystérèse :

> `LastRegime(x.ID, default: Deterministe)` — si le score composite τ tombe
> dans la bande `[TauMin, TauMin + HysteresisGap]`, le régime précédent
> associé à l'identifiant d'échange est reconduit au lieu de basculer.

L'implémentation v0.1.0 (`internal/orchestration/dispatcher.go:142-145`)
simplifie cette logique de deux façons (Confirmé AUDIT.md §7 étape 7) :

1. Le régime dans la bande est toujours retourné comme `Deterministe`, sans
   consulter de table `lastRegime`.
2. `Thresholds.HysteresisGap` (présent dans `calibration.Thresholds`,
   `profile.go:19-26`) n'est jamais lu par le dispatcher. Le gap effectif est
   donc nul — la bande n'existe pas en pratique.

Cette simplification n'est pas documentée ; elle est invisible pour
l'opérateur qui calibre son profil et définit un `HysteresisGap` non nul
(Probable — comportement inobservable sans instrumentation de la Trace).

L'implémentation complète de la mémoire par Exchange ID requiert :
- une map concurrente `x.ID → Regime` avec TTL de nettoyage,
- une décision sur la durée de vie des entrées (non spécifiée dans PRD §10),
- des tests de concurrence supplémentaires.

Cet effort est estimé à 1-2 jours-ingénieur avec risque de regression sur la
séquence des tests parallèles (`t.Parallel()` systématique).

## Décision

La V1 (v0.1.0 → v0.1.1) conserve la simplification : `Deterministe` par
défaut dans la bande, sans table `lastRegime`. Cette simplification est
déclarée explicitement plutôt que tacite.

Actions associées :

1. PRD §10 étape 7 est amendé *(cf. AUDITPlan.md T-040)* pour refléter
   la simplification V1 avec renvoi `*(cf. ADR-0007)*` et marqueur `Probable`
   sur la décision de reconduite.
2. Le champ `HysteresisGap` est conservé dans `calibration.Thresholds` /
   `thresholds.Thresholds` pour assurer la rétro-compatibilité des profils
   sérialisés en V0.2 (Confirmé : champ additif, pas de rupture JSON).
3. La cible V0.2 est définie pour implémenter la mémoire complète :
   map concurrente `sync.Map[string, Regime]` + TTL configurable
   + `//nolint:gochecknoglobals` ou injection via constructeur.

## Conséquences

**Positives :**

- Aucun changement comportemental en V0.1.1 : les tests existants passent sans
  modification.
- La simplification est désormais opposable (ADR, renvoi PRD) — un opérateur
  qui lit la spec voit la divergence.
- `HysteresisGap` dans les profils calibrés est rétrocompatible : ignoré en
  V1, utilisé en V0.2.

**Négatives :**

- Un profil avec `HysteresisGap > 0` ne produit aucun effet en V0.1.1.
  Comportement potentiellement trompeur si non documenté dans le profil par
  défaut.
- La note de dépréciation doit être explicite dans le commentaire du champ
  `HysteresisGap` jusqu'en V0.2.

## Alternatives rejetées

1. **Implémentation complète en V0.1.1** — effort 1-2 jours, risque de
   concurrence (`sync.Map` + TTL), décision sur durée de vie des entrées non
   spécifiée dans PRD §10. Rejeté : le rapport coût/risque ne justifie pas
   l'inclusion dans un refactor de consolidation.

2. **Suppression de `HysteresisGap`** — rupture API calibration : les profils
   sérialisés contenant ce champ deviendraient invalides. Rejeté : la
   rétrocompatibilité des profils est garantie par AUDIT.md §19.

3. **Retour de `Heuristique` dans la bande** — cohérent avec une lecture
   littérale du PRD §10, mais dépend de la mémoire `LastRegime` pour être
   déterministe. Sans la mémoire, `Heuristique` serait retourné
   systématiquement dans la bande, ce qui ne correspond pas à la spec.
   Rejeté.

## Renvois

- AUDIT.md §7 étape 7 (simplification hystérèse constatée)
- AUDIT.md §9 P1-07 (écart spec + recommandation option b)
- AUDIT.md §18 R7 (recommandation ADR-0007)
- PRD.md §10 étape 7 (à amender en T-040)
- CLAUDE.md §Architecture et §Refus — décision de premier rang
- ADR-0001 (`docs/adr/0001-clean-architecture-4-layers.md`) — fondation
- ADR-0005 (`docs/adr/0005-agentmeshkafka-dto.md`) — modèle de format
- ADR-0006 (`docs/adr/0006-types-valeur-transverses.md`) — `HysteresisGap`
  migré vers `thresholds.Thresholds`
- AUDITPlan.md T-006 (ADR) et T-040 (mise à jour PRD §10.1)

*Statut : Accepté — V1 simplifiée déclarée.*
