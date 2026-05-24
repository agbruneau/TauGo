# 07 — Anti-patrons d'usage — renvoi vers chap. III.8.7

*Document de renvoi croisé. Le verbatim canonique vit dans `agbruneau/InteroperabiliteAgentique` v2.4.3, `Monographie.md` chap. III.8.7.*

*Statut global : 4 anti-patrons gardés par test au tag `v0.1.0`. Confirmé. Daté 2026-05-24.*

---

## Vue synoptique *(chap. III.8.7)*

Quatre usages de τ contredisent ses hypothèses fondatrices et doivent être rejetés à la conception, non corrigés à l'exécution. *(PRD §7.2 ; CLAUDE.md §Anti-patrons interdits)*

| # | Anti-patron | Raison courte | Garde nominale |
|---|---|---|---|
| **AP#1** | Usage prédictif | Modèle structurant, pas prédictif | `TestNoPredictiveAPI` (M3.9) |
| **AP#2** | Usage hors frontière | Sur-ingénierie, frontière classique mal diagnostiquée | `TestRefusHorsFrontiere` (M0.5) |
| **AP#3** | Usage atemporel | Instrument daté transformé en assertion intemporelle | `TestI3_DateRevisionRespectee` (M3.9) |
| **AP#4** | Usage clos | Hypothèse de complétude non acquise | `TestUnmodeledObservationsReported` (M3.9) |

---

## AP#1 — Usage prédictif *(chap. III.8.7.1)*

### Pourquoi interdit

τ est un modèle **structurant** : il décrit où s'effectue la fixation des grandeurs d'interopérabilité (`t_fix`), pas ce que ces grandeurs vaudront. Le substrat probabiliste du pair appelé est par définition non déterministe ; toute méthode qui prétend prédire son comportement futur contredit I1 (conservation) et I2 (irréductibilité). *(chap. III.8.7.1)*

Exporter une API `Predict*` / `Expected*` / `Forecast*` signale au client une capacité que τ n'a pas : cela induit un usage défectueux en production, non détectable à la compilation.

### Garde TauGo

- **`TestNoPredictiveAPI`** : `internal/anti_patterns_test.go` (M3.9). Inspection par réflexion Go (`reflect`) de toutes les méthodes exportées des types publics dans `internal/tau/*` et `internal/orchestration/*`. Regex de rejet : `^(Predict|Expected|Forecast).*`. Le test échoue si un seul symbole correspond.
- **Règle PR** : toute méthode exportée avec ce préfixe entraîne rejet sans exception.

*Confirmé. Garde en place depuis M3.9. *(chap. III.8.7.1, PRD §7.2 #1, CLAUDE.md §Anti-patrons interdits #1)*

---

## AP#2 — Usage hors frontière *(chap. III.8.7.2)*

### Pourquoi interdit

Appliquer τ à une interaction classique (pair déterministe sous contrat, univers clos, composition fixe, coût réversible) est de la **sur-ingénierie injustifiée**. Pire : cela signale au client un régime agentique alors qu'il est classique — fausse alarme coûteuse, décision `Probabiliste` là où `Deterministe` suffit. *(chap. III.8.7.2)*

La frontière n'est pas configurable : soit les quatre conditions classiques sont simultanément violées, soit elles ne le sont pas. Il n'existe pas de mode « frontière partielle ».

### Garde TauGo

- **`TestRefusHorsFrontiere`** : `internal/tau/frontier_test.go` (M0.5). Vérifie que toute combinaison avec au moins une condition classique tenue retourne `Refus`.
- **`FrontierCheck.Inside()`** : `internal/tau/frontier.go` (M0). Aucun paramètre de contournement (`skipCheck`, `bypassFrontier`, etc.) n'est admis — ajout = ADR obligatoire.

*Confirmé. Garde en place depuis M0.5. *(chap. III.8.7.2, PRD §7.2 #2, CLAUDE.md §Anti-patrons interdits #2)*

---

## AP#3 — Usage atemporel *(chap. III.8.7.3)*

### Pourquoi interdit

I3 (asymétrie ontologique D-AUTORITÉ) est daté : il encode l'état du support normatif de l'identité agentique déléguée à une date précise. En l'absence de RFC IETF ou de reconnaissance juridique inter-organisationnelle, le bloc ontologique tient. Cette condition est **provisoire et révisable** — la Monographie la désigne explicitement comme telle. *(chap. III.8.5.3)*

Utiliser τ sans date de révision, ou tolérer un profil périmé, transforme un instrument de navigation en assertion intemporelle. Les décisions produites après la date de péremption ne sont plus opposables.

### Garde TauGo

- **`TestI3_DateRevisionRespectee`** : `internal/tau/invariants/` (M3.9). Vérifie que `Profile.DateRevision` est présente et future.
- **`TestExpiredProfileRefuses`** : `internal/calibration/drift_test.go` + `test/e2e/calibration_determinism_test.go` (M5.6). Vérifie que l'étape 3 du dispatcher retourne `Refus("profil périmé — veille requise")` dès que `today > Profile.DateRevision`.
- **CI alerte** : 30 jours avant péremption *(PRD §11.4)*.
- **Trace** : `Decision.Trace` expose `profile.date_revision` et `profile.version_monographie` pour auditabilité.

*Confirmé. Gardes en place depuis M3.9 (TestI3) et M5.6 (TestExpired). *(chap. III.8.7.3, PRD §7.2 #3, CLAUDE.md §Anti-patrons interdits #3)*

---

## AP#4 — Usage clos *(chap. III.8.7.4)*

### Pourquoi interdit

Le modèle τ opère sur **trois dimensions** et **cinq invariants** — mais la Monographie ne prétend pas que cet espace est exhaustif. L'hypothèse de complétude n'est pas acquise *(chap. III.8.7)* : une observation empirique peut révéler une dimension non modélisée ou un invariant supplémentaire. Tenir le modèle pour clos, c'est supprimer le signal d'alerte quand une observation ne rentre pas dans les cases existantes.

Un `Kernel.Decide` silencieux face à des observations non modélisées est plus dangereux qu'un refus : il produit une décision sans instrumenter l'incertitude. *(PRD §7.2 #4)*

### Garde TauGo

- **`TestUnmodeledObservationsReported`** : `internal/anti_patterns_test.go` (M3.9). Vérifie que `Decision.Trace.UnmodeledObservations []string` est non nil et non vide quand l'échange contient des champs de contexte non reconnus par le dispatcher.
- **Rapport mensuel** : `docs/empirical/unmodeled.md` (M4) — catalogue des observations non modélisées collectées sur traces AgentMeshKafka.
- **Cas de refus** : observation non modélisée à fort impact → `Refus("usage clos potentiel")` *(PRD §7.3)*.

*Confirmé pour la garde. A vérifier pour le seuil « fort impact » (non calibré, M4). *(chap. III.8.7.4, PRD §7.2 #4, CLAUDE.md §Anti-patrons interdits #5)*

---

## Posture épistémique *(PRD §1.1)*

τ est un **modèle structurant**, non un prédicteur. Cette distinction fonde les quatre interdictions ci-dessus :

> Un modèle structurant dit *où* et *comment* une grandeur se fixe — il ne dit pas *quelle valeur* elle prendra. La valeur dépend du pair, qui est probabiliste par définition. Prétendre prédire cette valeur serait nier la raison d'être même de τ.

Corollaire : les quatre anti-patrons ne sont pas des bugs d'implémentation corrigeables. Ce sont des **erreurs de conception** qui signalent que l'utilisateur a mal compris le domaine d'application de τ. La garde n'est pas là pour les corriger en production — elle est là pour les détecter avant qu'une PR ne soit mergée.

*Aligné PRD §1.1 (doctrine), §3.3 (anti-objectifs). *(chap. III.8.7)*

---

## Statut des gardes au tag `v0.1.0`

| Anti-patron | Garde | Milestone | Statut |
|---|---|---|---|
| AP#1 prédictif | `TestNoPredictiveAPI` | M3.9 | Confirmé |
| AP#2 hors frontière | `TestRefusHorsFrontiere` | M0.5 | Confirmé |
| AP#3 atemporel | `TestI3_DateRevisionRespectee` + `TestExpiredProfileRefuses` | M3.9 + M5.6 | Confirmé |
| AP#4 clos | `TestUnmodeledObservationsReported` | M3.9 | Confirmé |

Tous les tests passent en `-race` sur Linux/macOS (CI). Windows sans CGO : tests courts sans `-race`. *(PRD §15.1)*

---

*Renvoi PRD : §7.2 (anti-patrons), §7.3 (refus de premier rang), §1.1 (doctrine), §3.3 (anti-objectifs). CLAUDE.md §Anti-patrons interdits. Calque structurel : `docs/theory/03-operateur-tau.md`.*

**Aligné monographie** : v2.4.3 (2026-05-21).
**Daté** : 2026-05-24.
