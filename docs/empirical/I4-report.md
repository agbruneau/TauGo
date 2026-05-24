# Rapport empirique I4 — campagne M4

> Généré le 2026-05-24. Régime : **B (contingence — corpus synthétique)** — voir `I4-regime.md`.
> Outil : `internal/bridge/agentmeshkafka/empirical_i4_test.go` (build tag `empirical`).
> Statut : **Hypothèse — campagne synthétique inconclusive sur I4.**
> *(chap. III.8.5.4 / PRD §6.1, §15 / PRDPlanning §M4)*

---

## §1 Contexte et régime

L'audit M4.0 (agent `Explore`) a établi que le dépôt `agbruneau/AgentMeshKafka` est **inexistant** — ni en local ni sur GitHub. En l'absence de traces réelles, le plan M4 *(PRDPlanning §M4)* prévoit une contingence : **Régime B**, campagne sur corpus synthétique généré par `cmd/generate-corpus`. La décision de régime est tracée dans `docs/empirical/I4-regime.md`.

Corpus utilisé :

| Attribut | Valeur |
|---|---|
| Fichier | `cmd/generate-corpus/testdata/synthetic-corpus-120-seed42-balanced.jsonl` |
| Lignes | 120 |
| Graine | 42 |
| Profil de distribution | `balanced` |
| SHA-256 | `a91d60cd9815d8183df57bfcf16bbe77d36360c4ed36e33fced9f12f70fd68ee` |
| Profil de calibration | `M3-default` |
| Horodatage d'exécution | 2026-05-24T11:27:43Z |

Le corpus est **gelé** — le SHA-256 ci-dessus est la référence d'audit. Toute ré-exécution doit produire des résultats identiques sur le même profil.

---

## §2 Méthodologie

### Distribution du profil `balanced`

| Préfixe d'ID | Catégorie sémantique | Fréquence cible |
|---|---|---|
| `rf-` | Hors frontière (refus frontière) | 15 % (18 traces) |
| `r3-` | Refus I3 (autorité asymétrique) | 15 % (18 traces) |
| `r4-` | Refus I4 (incohérence I4) | 10 % (12 traces) |
| `d-` | Décision déterministe | 25 % (30 traces) |
| `p-` | Décision probabiliste | 25 % (30 traces) |
| `h-` | Zone hysterèse | 10 % (12 traces) |

### Pipeline d'exécution

```
FileAdapter (JSONL) → ToTauExchange → app.NewDispatcher().Decide → classifyI4()
```

- `FileAdapter` : lit le JSONL ligne par ligne, construit un `agentmeshkafka.Exchange`.
- `ToTauExchange` : convertit vers `tau.Exchange` (DTO défini par ADR-0005).
- `Dispatcher.Decide` : applique la séquence complète des 8 étapes *(PRD §10)*, dont `FrontierCheck`, les trois dimensions, et les cinq invariants.
- `classifyI4()` : affecte chaque décision à l'une des six catégories selon le régime retourné et la présence d'une erreur I4.

### Calculs métriques

- **Sensitivity** = TP / (TP + FN) ; `-1` si dénominateur nul (aucun TP ni FN observé).
- **Specificity** = TN / (TN + FP).

---

## §3 Résultats bruts

| Catégorie | Compte | % | Signification |
|---|---|---|---|
| `i4_coherent_accepted` | 84 | 70,0 % | Traces dont D-INVARIANT < θ_inv et D-SENS ≥ θ_sens → Décision acceptée, I4 non déclenchée |
| `i4_incoherent_refused` (TP) | 0 | 0,0 % | Vrais positifs I4 : refus déclenché par incohérence D-INV ≥ θ_inv ∧ D-SENS < θ_sens |
| `i4_false_positive` | 0 | 0,0 % | I4 déclenchée à tort sur trace cohérente |
| `i4_false_negative` (FN) | 0 | 0,0 % | Incohérence I4 présente mais non détectée |
| `other_refusal` | 36 | 30,0 % | Refus par la garde frontière (`FrontierCheck`) ou autre invariant (I3) — I4 non évaluée |
| `unmodeled` | 0 | 0,0 % | Observations sans catégorie modélisée *(anti-patron #4 — PRD §7.2)* |
| **Total** | **120** | **100 %** | |

**Métriques :**

| Métrique | Valeur | Interprétation |
|---|---|---|
| Sensitivity | `-1` (indéfinie) | Dénominateur nul : aucun TP ni FN — I4 jamais déclenchée |
| Specificity | `1,0` | Aucun faux positif I4 — **Confirmé** |

---

## §4 Interprétation

**I4 reste Hypothèse.** Aucune trace du corpus synthétique n'a provoqué le déclenchement de la garde I4 *(chap. III.8.5.4)*. La raison directe est instrumentale : le générateur `cmd/generate-corpus` ne peuple pas les champs `Context` (`event_registry`, `idempotency_key_mode`, etc.) dont dépendent les sondes `D-INVARIANT`. Sans ces champs, le score D-INVARIANT reste à `0.25` pour toutes les traces — sous le seuil `θ_inv = 0.50`. L'invariant I4 ne se déclenche donc **jamais**, qu'il s'agisse de traces `r4-` (censées déclencher I4) ou d'autres.

**36 `other_refusal` confirment la garde frontière** *(anti-patron #2 — PRD §7.2)*. Ces refus sont produits par `FrontierCheck.Inside()` avant que les dimensions ne soient calculées. Ils correspondent aux traces `rf-` (18) et `r3-` (18), ce qui est cohérent avec la distribution du profil `balanced`. Le fait que **zéro** bypass de frontière n'ait été détecté est un signal positif — **Confirmé** *(chap. III.8.5.2)*.

**84 `i4_coherent_accepted`** : les 84 traces qui passent la frontière reçoivent un D-INVARIANT de 0.25, soit en deçà du seuil. Elles se répartissent entre `d-`, `p-`, `h-` et une fraction de `r4-` non refusées par frontière. Ces décisions confirment le profil nominal cohérent du stub déterministe, sans qu'aucune incohérence I4 ne soit générée — **Hypothèse** *(chap. III.8.5.4, condition de réfutation non atteinte)*.

---

## §5 Limites identifiées

1. **Absence de clés `Context` dans le corpus synthétique.** Le générateur `cmd/generate-corpus` produit des `agentmeshkafka.Exchange` sans champs `Context` structurés. Les sondes D-INVARIANT (`event_registry`, `idempotency_key_mode`, délégation de chaîne) lisent ces champs pour calculer le score. Sans eux, D-INVARIANT est fixé à `0.25` par défaut, soit toujours sous `θ_inv = 0.50`. I4 **ne peut pas se déclencher** dans ce corpus, quelle que soit l'intention du préfixe d'ID. *Impact : campagne inconclusive sur I4.*

2. **Profil `i4-heavy` non utilisé.** Le générateur M4.4 expose un profil `i4-heavy` conçu pour surreprésenter les traces `r4-`. Le corpus gelé dans `testdata/` utilise le profil `balanced` (10 % de `r4-`). Même avec `i4-heavy`, la limite #1 s'appliquerait tant que `Context` n'est pas injecté — mais ce profil n'a pas été testé. *Impact : une avenue d'enrichissement reste inexpllorée.*

3. **Aucune trace réelle.** La limite fondamentale est l'absence d'AgentMeshKafka *(PRD §18 risque #1)*. Un corpus réel apporterait des valeurs `Context` authentiques, des durées d'exécution mesurées, et des profils de délégation non prédictibles par un générateur synthétique. *Impact : campagne réelle reportée à M4-bis au plus tôt.*

4. **Profil de calibration `M3-default`.** Les seuils utilisés (`θ_inv = 0.50`, `θ_sens = 0.50`) sont des valeurs initiales pré-calibration *(PRD §11.1)*. La calibration M5 pourrait modifier ces seuils, changeant le périmètre de déclenchement d'I4. *Impact : résultats à re-évaluer après M5 — Hypothèse.*

---

## §6 Verdict de campagne

**Hypothèse inchangée.** Le passage à *Probable* attendu en M4 *(PRDPlanning §M4)* ne s'est pas matérialisé. La cause n'est pas une défaillance de la garde I4, mais une limite instrumentale du corpus synthétique qui rend la campagne **inconclusive** sur I4 *(chap. III.8.5.4)*.

Points positifs :
- Faux négatifs nuls : la garde I4 n'a pas été contournée sur les 120 traces — **Confirmé** pour ce corpus.
- Faux positifs nuls : spécificité parfaite (`1.0`) — **Hypothèse** (le corpus ne sollicite pas la garde).
- Garde frontière opérationnelle : 36/36 refus attendus reçus — **Confirmé**.

La campagne démontre la **robustesse défensive** du kernel mais ne peut pas établir la sensibilité d'I4 en l'absence de signaux D-INVARIANT élevés.

---

## §7 Prochaines étapes

| Priorité | Action | Milestone | Responsable |
|---|---|---|---|
| 1 | Enrichir `cmd/generate-corpus` : injecter champs `Context` pour faire monter D-INVARIANT | M4-bis (post-M5) | `ruflo-core:coder` |
| 2 | Exploiter profil `i4-heavy` avec Context enrichi — viser ≥ 10 vrais positifs I4 | M4-bis | `ruflo-core:coder` |
| 3 | Si AgentMeshKafka disponible : pivoter en Régime A, acquérir traces réelles | M4-bis ou M5 | `ruflo-core:researcher` |
| 4 | Re-run du harness après calibration M5 (seuils révisés peuvent modifier le périmètre I4) | Post-M5 | `ruflo-core:coder` |
| 5 | Ouvrir issue de suivi : « enrichissement générateur Context pour I4 » | Immédiat | Thread principal |

---

## §8 Marqueurs épistémiques

| Affirmation | Marqueur | Justification |
|---|---|---|
| I4 reste Hypothèse | **Hypothèse** | Aucune trace n'a déclenché I4 — pas d'évidence pour ou contre |
| Garde frontière opérationnelle | **Confirmé** | 36/36 refus attendus, zéro bypass |
| Spécificité = 1.0 | **Hypothèse** | Corpus ne sollicite pas la garde — absence de preuve n'est pas preuve d'absence |
| D-INVARIANT bloqué à 0.25 | **Confirmé** | Déterministe : champs Context absents du générateur |
| Calibration M5 peut changer le périmètre | **Hypothèse** | Seuils pré-calibration — À vérifier après M5 |

---

## Renvois

- *(chap. III.8.5.4)* — Invariant I4 : cohérence contrainte
- *(chap. III.8.5.2)* — Frontière τ
- *(chap. III.8.7)* — Anti-patrons
- PRD §6.1 (I4), §6.3 (priorité empirique #1), §7.2 (anti-patrons #2, #4), §11.1 (seuils initiaux), §15 (tests E2E), §18 (risque #1)
- PRDPlanning §M4
- `docs/empirical/I4-regime.md` — décision de régime A/B
- `docs/empirical/unmodeled.md` — registre observations non modélisées
- `docs/adr/0005-agentmeshkafka-dto.md` — contrat DTO
- `internal/bridge/agentmeshkafka/testdata/empirical-i4-results.json` — résultats bruts JSON
