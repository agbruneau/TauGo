# 05 — Cinq invariants I1-I5 — renvoi vers chap. III.8.5

*Document de renvoi croisé. Le verbatim canonique vit dans `InteroperabiliteAgentique/Monographie.md` v2.4.3, chap. III.8.5.*

*Statut global : gradué par invariant — voir tableau ci-dessous. Daté 2026-05-24.*

---

## Vue synoptique

| # | Énoncé court | Statut | Helper Go | Cible fuzz |
|---|---|---|---|---|
| **I1** | Conservation — τ déplace `t_fix`, pas la grandeur | Probable | `Conserve(x, τ(x))` | `FuzzI1_Conservation` |
| **I2** | Irréductibilité — résidu non vide, non recâblable hors ligne | Confirmé par construction | `Residu(x)`, `Recablage(x)` | `FuzzI2_Irreductibilite` |
| **I3** | Asymétrie — dimensions orthogonales en valeur, asymétriques en maturité ; D-AUTORITÉ = fait institutionnel | Probable, daté 2026-05-16 | `Attestation` (struct du profil) | `FuzzI3_AsymetrieAutorite` |
| **I4** | Cohérence contrainte — D-INVARIANT contraint par D-SENS | Hypothèse | `θ_sens`, `θ_inv` | `FuzzI4_CoherenceContrainte` |
| **I5** | Composition par conjonction — pile hérite de l'union des angles morts | Probable | `M(π)` = `|⋃Aᵢ|` | `FuzzI5_CompositionConjonctive` |

---

## I1 — Invariant de conservation *(chap. III.8.5.1)*

### Verbatim canonique

> **I1 — Invariant de conservation.** *τ déplace l'instant de fixation d'une grandeur sans altérer la grandeur.* Le sens reste du sens, l'autorité reste de l'autorité, l'invariant d'intégration reste l'invariant.

*(Monographie.md v2.4.3, ligne 5746)*

### Dérivation (III.8.5)

Appui formel : τ opère sur `t_fix(g)`, pas sur `g` (§III.8.3.1). Généralisation de la loi III.7 sur quatre patrons composites (Saga, Outbox, BFF, Strangler Fig) et deux styles architecturaux (API-First, EDA). La généralisation à l'ensemble des patrons est une conjecture corrélée, non démontrée exhaustivement — la validation sur patrons non examinés est renvoyée aux Parties IV et V.

### Condition de réfutation (III.8.5)

> l'exhibition d'une seule grandeur d'interopérabilité que l'agentivité *supprime* — un invariant métier qui cesse d'être désirable parce que le pair est probabiliste — réfute I1 ; l'exhibition d'un patron ou style non examiné dont l'invariant est supprimé par τ borne I1 au périmètre démontré sans le réfuter dans ce périmètre.

### Reformulation exécutable (PRD §6.1)

Pour tout échange `x` admissible, `Conserve(x, τ(x)) == true` *(égalité modulo équivalence métier déclarée)*.

**Test négatif** : `TestRefutationI1_GrandeurSupprimee`

### Statut épistémique

**Probable** — généralisation solide sur 4 patrons + 2 styles examinés ; extension à un patron non examiné = conjecture. *(chap. III.8.5)*

---

## I2 — Invariant d'irréductibilité *(chap. III.8.5.2)*

### Verbatim canonique

> **I2 — Invariant d'irréductibilité.** *Le résidu de sens, d'autorité et de support qui migre vers l'exécution est non vide et non recâblable hors ligne sans détruire l'agentivité.* Ce n'est pas tout le sens qui migre — la part câblable hors ligne le reste — mais le résidu migrant est strictement non vide à la frontière agentique.

*(Monographie.md v2.4.3, ligne 5748)*

### Dérivation (III.8.5)

Argument en trois pas du chapitre III.1, étendu aux trois dimensions. Appui formel : τ n'est non trivial que si `t_fix(g) ≺ t_int` peut être strictement violé sans détruire `g` (§III.8.3.1).

### Condition de réfutation (III.8.5)

> une méthode d'ingénierie qui ramènerait intégralement une frontière agentique au cas câblé hors ligne *tout en préservant* l'univers ouvert, la composition variable et la nature probabiliste du pair réfuterait I2. Le chapitre III.1 a montré que toute tentative de réduction détruit l'objet qu'elle simplifie ; I2 est, des cinq, le plus solidement déductif.

### Reformulation exécutable (PRD §6.1)

Pour tout `x` dans la frontière, `Residu(x) := { g | t_fix(g) ≈ t_int } ≠ ∅` ; tout `Recablage(x)` qui vide le résidu doit faire perdre ≥ 1 condition de frontière.

**Test négatif** : `TestRefutationI2_RecablageComplet`

### Statut épistémique

**Confirmé par construction** — le plus déductif des cinq. *(chap. III.8.5)*

---

## I3 — Invariant d'asymétrie de maturité et d'asymétrie ontologique *(chap. III.8.5.3)*

### Verbatim canonique

> **I3 — Invariant d'asymétrie de maturité et d'asymétrie ontologique.** *Les trois dimensions sont orthogonales en valeur mais asymétriques en maturité d'instrumentation : D-AUTORITÉ est, à la date de rédaction, la dimension dont le pôle « exécution » n'a aucun support normatif stable.* Conséquence : dans la transition 2027-2030, l'autorité déléguée — non le sens — est le facteur limitant de l'agentivité gouvernée.

*(Monographie.md v2.4.3, ligne 5750)*

### Clause additionnelle ontologique (III.8.4.2.bis)

L'asymétrie n'est pas seulement de degré (maturité d'instrumentation) mais de **nature** : D-AUTORITÉ relève d'un fait institutionnel au sens de Searle (1995), distinct de D-SENS et D-INVARIANT (faits protocolaires). Un fait institutionnel n'existe que par assignation collective de fonction de statut par une autorité reconnue ; il ne peut être instauré par accord in-band entre pairs probabilistes. Sur D-AUTORITÉ, τ est *ontologiquement bloqué* tant qu'aucune institution émettrice externe n'instaure le fait.

### Condition de réfutation (III.8.5)

> l'accession d'un standard d'identité agentique déléguée au statut de RFC couvrant la responsabilité inter-organisationnelle réfute I3 […] L'émergence d'une institution émettrice externe (un consortium IETF aboutissant à un RFC d'identité agentique déléguée couvrant la responsabilité inter-organisationnelle ; une juridiction reconnaissant la chaîne comme preuve opposable ; ou les deux conjointement) réfute la clause ontologique.

**Revérification** : au 2026-12-01 (CI alerte 30 j avant).

### Reformulation exécutable (PRD §6.1)

`D-AUTORITÉ(x) ≥ θ_auth_block ∧ Attestation == nil ⇒ Refus`. Clause de péremption : `date_revision ≤ 2027-01-01` dans le profil.

**Test négatif** : `TestI3_DateRevisionRespectee`

### Statut épistémique

**Probable, daté 2026-05-16** — à revérifier en veille continue. *(chap. III.8.5)*

---

## I4 — Invariant de cohérence contrainte *(chap. III.8.5.4)*

### Verbatim canonique

> **I4 — Invariant de cohérence contrainte.** *D-INVARIANT est contraint par D-SENS : un support d'invariant ne peut se fixer à l'exécution que si le sens auquel il s'applique y est lui-même fixé ; les combinaisons incohérentes sont observables.* Formellement : `i ≈ pendant ⟹ s ≈ pendant`, et la configuration `(s ≈ pendant, ·, i ≈ avant)` est réalisable mais instable (§III.8.4.5). Conséquence : un système à sens négocié mais support figé est dans une combinaison incohérente, et son incohérence se manifeste par la rupture silencieuse du chapitre III.7.

*(Monographie.md v2.4.3, ligne 5752)*

### Dérivation (III.8.5)

§III.8.4.3 et §III.8.4.5. Direction dissymétrique : c'est D-SENS qui contraint D-INVARIANT, jamais l'inverse.

### Condition de réfutation (III.8.5)

> un système opérant durablement et sans défaillance avec un sens négocié à l'exécution mais un support d'invariant figé à la conception réfuterait I4 — c'est une proposition empiriquement testable, et le programme de validation (§III.8.7) la désigne comme telle.

### Reformulation exécutable (PRD §6.1)

`D-INVARIANT(x) ≥ θ_inv ∧ D-SENS(x) < θ_sens ⇒ Refus(diag: "I4")`.

**Test négatif** : `TestRefutationI4_CombinaisonIncoherente`

**Priorité empirique** : campagne dédiée M4, rapport `docs/empirical/I4-report.md`.

### Statut épistémique

**Hypothèse** — empiriquement testable, non encore testée. *(chap. III.8.5)*

---

## I5 — Invariant de composition par conjonction *(chap. III.8.5.5)*

### Verbatim canonique

> **I5 — Invariant de composition par conjonction.** *Une pile de couches d'interopérabilité agentique hérite de la conjonction — non de la disjonction ni de la moyenne — des angles morts de ses couches, et ne dispose d'aucun mécanisme transversal de réconciliation sauf à en ajouter un hors pile.* Conséquence : empiler des couches maximise l'interopérabilité sans produire de garantie de bout en bout ; la sûreté de la pile est *au mieux* celle de sa couche la plus faible, et *au pire* la conjonction de toutes leurs faiblesses.

*(Monographie.md v2.4.3, ligne 5754)*

### Métrique cardinale M(π) (III.8.6.3)

Soit π = (C₁, C₂, …, Cₙ) une pile de n couches τ-migrées et Aᵢ l'ensemble des angles morts de Cᵢ :

```
M(π) = |A₁ ∪ A₂ ∪ … ∪ Aₙ|
```

Trois propriétés ensemblistes immédiates :
- *Croissance* : M(π ∪ {Cₙ₊₁}) ≥ M(π)
- *Borne inférieure* : M(π) ≥ max(|Aᵢ|)
- *Borne supérieure* : M(π) ≤ Σᵢ |Aᵢ| (égalité ssi les Aᵢ sont deux à deux disjoints)

### Condition de réfutation (III.8.5)

> une pile composée où une couche referme structurellement l'angle mort d'une autre *sans* mécanisme transversal ajouté réfuterait I5 ; le chapitre III.6 ne l'a pas observée, mais I5 reste falsifiable et le modèle ne le présente pas comme acquis définitif.

### Reformulation exécutable (PRD §6.1)

Pour pile `π = [C₁,…,Cₙ]`, `M(π) = |⋃Aᵢ|` satisfait `M(π) ≥ max(|Aᵢ|)` et `M(π) ≤ Σ|Aᵢ|`. V1 expose l'API d'agrégation ; V2 calcule.

**Test négatif** : `TestRefutationI5_AngleMortReferme`

### Statut épistémique

**Probable** — généralisation solide du résultat III.6, non exhaustivement démontrée. *(chap. III.8.5)*

---

## Articulation des cinq invariants (III.8.5 + PRD §6.3)

Extrait verbatim (Monographie.md v2.4.3, ligne 5756) :

> I1 et I2 fondent l'opérateur (l'un dit que τ conserve, l'autre que τ est non trivial — ensemble ils établissent que τ *existe et fait quelque chose*) ; I3 et I4 caractérisent la structure de l'espace (l'un dit que les dimensions sont asymétriques en maturité, l'autre qu'elles sont contraintes en cohérence — ensemble ils établissent que l'espace n'est ni plat ni libre) ; I5 régit la composition (il dit ce qui se passe quand on empile des couches τ-migrées — c'est le seul invariant qui porte sur des assemblages plutôt que sur une frontière).

### Articulation V1 (PRD §6.3)

- **I1 + I2** fondent l'opérateur : conservation + non-trivialité. Garde combinée : `TestOperatorWellDefined`.
- **I3 + I4** caractérisent la structure : asymétrie de maturité + contrainte de cohérence. Garde : `TestSpaceNonFlat`.
- **I5** régit la composition. Garde V2 : `TestM_Monotonicity`.

**Priorité empirique #1** : I4 (Hypothèse non encore testée) — campagne dédiée en M4, rapport `docs/empirical/I4-report.md`.

---

## Cibles fuzz (PRD §15.2)

```go
// internal/tau/invariants/fuzz_targets.go
func FuzzI1_Conservation(f *testing.F)           // τ(x).grandeur ≡ x.grandeur
func FuzzI2_Irreductibilite(f *testing.F)        // tout recâblage hors ligne détruit ≥ 1 condition de frontière
func FuzzI3_AsymetrieAutorite(f *testing.F)      // jamais Probabiliste avec D-AUTORITÉ ≥ θ_auth_block ∧ Attestation == nil
func FuzzI4_CoherenceContrainte(f *testing.F)    // (s < θ_sens, ·, i ≥ θ_inv) ⇒ Refus(I4)
func FuzzI5_CompositionConjonctive(f *testing.F) // M(π) ≥ max(|Aᵢ|), M(π) ≤ Σ|Aᵢ|
```

---

## Marqueurs d'incertitude

| Élément | Marqueur | Justification |
|---|---|---|
| I1 — conservation | Probable | Généralisation sur 4 patrons + 2 styles ; extension = conjecture *(chap. III.8.5)* |
| I2 — irréductibilité | Confirmé | Argument déductif III.1 ; le plus solide des cinq *(chap. III.8.5)* |
| I3 — asymétrie ontologique | Probable, daté 2026-05-16 | Aucun RFC d'identité agentique déléguée au 2026-05-16 ; à revérifier 2026-12-01 |
| I4 — cohérence contrainte | Hypothèse | Testable, non encore testée ; campagne M4 *(chap. III.8.5)* |
| I5 — composition | Probable | Généralisation III.6 ; non exhaustivement démontrée *(chap. III.8.5)* |
| M(π) — métrique cardinale | Probable | Définition opérante III.8.6.3 ; formalisation déductive renvoyée à HGL |
| Reformulations exécutables PRD §6.1 | Probable | Fidèles au verbatim — voir note d'alignement ci-dessous |

---

## Note d'alignement verbatim → PRD §6.1

Les reformulations exécutables du PRD §6.1 sont fidèles au verbatim de la Monographie sur les cinq invariants, avec les précisions suivantes.

**I1** : la reformulation PRD (« `Conserve(x, τ(x)) == true` ») est une contraction correcte. Le verbatim énonce la conservation sur trois grandeurs (sens, autorité, support d'invariant) ; PRD §6.1 abstrait cela en un seul helper `Conserve` « modulo équivalence métier déclarée ». Écart mineur : le verbatim précise que la conjecture sur les patrons non examinés n'est pas démontrée — cette nuance est portée par le statut « Probable » du PRD mais n'est pas explicitement traduite dans le helper.

**I2** : la reformulation PRD est fidèle. Le verbatim dit « résidu de sens, d'autorité et de support » ; le PRD condense en `Residu(x)` avec notation ensembliste correcte. Cohérent.

**I3** : la reformulation PRD capture la garde opérationnelle (`D-AUTORITÉ(x) ≥ θ_auth_block ∧ Attestation == nil ⇒ Refus`) et la clause de péremption. La clause ontologique (Searle 1995, nature du fait institutionnel) est documentée dans `docs/theory/04-dimensions.md` §D-AUTORITÉ et non redupliquée — acceptable par discipline de périmètre. La date de veille PRD (`date_revision ≤ 2027-01-01`) est plus conservative que la date de revérification PRD §6.2 (`2026-12-01`) : cohérence à vérifier lors de la revue de profil M3.

**I4** : la reformulation PRD inverse la présentation du verbatim pour la rendre exécutable : le verbatim énonce `i ≈ pendant ⟹ s ≈ pendant` (direction contrainte) ; PRD §6.1 traduit en `D-INVARIANT(x) ≥ θ_inv ∧ D-SENS(x) < θ_sens ⇒ Refus(I4)` (détection de la violation). L'inversion est correcte et intentionnelle.

**I5** : la reformulation PRD ajoute la métrique M(π) et ses deux bornes, qui proviennent de §III.8.6.3 et non strictement de §III.8.5. C'est une extension opérationnelle licite — la monographie renvoie explicitement M(π) comme opérationnalisation de I5 — mais il faut noter que la formalisation déductive complète de M(π) est renvoyée au manuscrit-compagnon HGL. V2 (calcul effectif) doit attendre.

**Écart notable à signaler au coordinateur** : le renvoi de ligne du PRD §6 (« lignes ~5723-5737 ») est décalé. Les invariants I1-I5 se trouvent aux lignes **5746-5756** de la Monographie, pas 5723-5737. La zone 5723-5741 correspond à la section III.8.4 (commentaire du Tableau III.8.2). Ce renvoi de ligne dans le PRD doit être corrigé.

---

*Renvoi PRD : §6 (invariants), §7.2 (anti-patrons), §15.2 (fuzz). Plan M3 : `PRDPlanning.md`.*

**Aligné monographie** : v2.4.3 (2026-05-21).
**Daté** : 2026-05-24.
