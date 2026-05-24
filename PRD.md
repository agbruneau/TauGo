# PRD — TauGo

**Projet** : TauGo — kernel exécutable Go de l'opérateur τ et validateur empirique des invariants I1-I5
**Auteur** : André-Guy Bruneau, M.Sc. **·** **Date** : 2026-05-23 **·** **Statut** : V0.2 (refactorisé)
**Référence canonique** : `agbruneau/InteroperabiliteAgentique` v2.4.3, chap. III.8 (monographie : opérateur τ, dimensions D-SENS/D-AUTORITÉ/D-INVARIANT, invariants I1-I5)
**Référence d'ingénierie** : `agbruneau/FibGo` (Clean Architecture, calibration adaptative, fuzz, déterminisme byte-identique)
**Référence empirique** : `agbruneau/AgentMeshKafka` (substrat de validation, traces réelles)

---

## Sommaire

| Bloc | Sections |
|---|---|
| **I — Cadrage stratégique** | [1. Finalité](#1-finalité) · [2. Thèse exécutable](#2-thèse-exécutable) · [3. Périmètre V1](#3-périmètre-v1) |
| **II — Théorie opérationnalisée** | [4. Opérateur τ](#4-opérateur-τ--formalisation-exécutable) · [5. Trois dimensions](#5-les-trois-dimensions) · [6. Cinq invariants](#6-les-cinq-invariants--reformulation-exécutable) · [7. Conditions de validité & anti-patrons](#7-conditions-de-validité--anti-patrons-dusage) |
| **III — Architecture & ingénierie** | [8. Architecture](#8-architecture-cible) · [9. Modèle de données](#9-modèle-de-données) · [10. Algorithme de dispatch](#10-algorithme-de-dispatch) · [11. Calibration](#11-calibration-adaptative) · [12. Bridges externes](#12-bridges-externes) |
| **IV — Discipline d'exécution** | [13. Stack](#13-stack-technique) · [14. Conventions](#14-conventions) · [15. Tests](#15-stratégie-de-test) |
| **V — Programme** | [16. Roadmap](#16-roadmap-v1) · [17. Critères de succès](#17-critères-de-succès-v1) · [18. Risques](#18-risques--mitigation) · [19. Glossaire](#19-glossaire-des-termes-contrôlés) · [20. Prochaines étapes](#20-documents-liés--prochaines-étapes) |

---

# Bloc I — Cadrage stratégique

## 1. Finalité

TauGo implémente le **kernel exécutable de l'opérateur τ** défini au chap. III.8 de la monographie. Il bâtit le pont entre :

- la **théorie** — τ comme opérateur de migration de l'instant de fixation, trois dimensions, cinq invariants réfutables *(chap. III.8.3-8.5)* ;
- l'**empirie** — `AgentMeshKafka` comme substrat de validation contre traces réelles ;
- l'**ingénierie** — `FibGo` comme calque opérationnel : dispatch multi-mode, calibration adaptative versionnée, fuzz, build reproductible byte-identique.

**Livrable empirique du Calcul d'Intégration Agentique (CIA)** — instrument d'épreuve du modèle, pas le modèle.

### 1.1 Posture épistémique et marqueurs d'incertitude

TauGo hérite intégralement de la posture du chap. III.8.2 : *le modèle τ vaut par ce qu'il rend pensable, non par ce qu'il garantit.* Validation visée = **structurelle**, pas prédictive. Toute prétention prédictive serait une dénaturation *(anti-patron §7.2)*.

Convention héritée de `InteroperabiliteAgentique/CLAUDE.md` §1.4, appliquée à tout `docs/`, ADR ou commentaire qui pose une affirmation factuelle datée :

| Marqueur | Sémantique | Application TauGo |
|---|---|---|
| **Confirmé** | Source primaire ou résultat déductif | I2 ; chaîne build FibGo (Go 1.25+, golangci-lint, race) ; horodatage gelé |
| **Probable** | Inférence solide, signaux convergents | I1, I3, I5 ; choix `errgroup` pour orchestration |
| **Hypothèse** | Plausible non corroboré | I4 (testable, non testée) ; pondérations initiales des sondes ; capacité d'AgentMeshKafka à servir de validateur M4 |
| **À vérifier** | Recherche complémentaire requise | Estimation 6-10 semaines ; pertinence de deux profils LLM (raisonnement / outillage) |

**Une fabrication détectée** (citation, chiffre, API, version, DOI inventés) **invalide le livrable concerné — sans appel.**

---

## 2. Thèse exécutable

τ est un opérateur de **dispatch** entre régime déterministe (garantie de message, protocole strict) et régime probabiliste (raisonneur LLM ouvert), à la **frontière de validité** strictement délimitée *(chap. III.8.3.2)*.

### 2.1 Frontière de validité — les 4 conditions classiques toutes violées

τ ne s'applique qu'aux échanges qui satisfont **simultanément** les quatre violations suivantes :

| Condition classique | Violation requise |
|---|---|
| Univers de capacités clos et énumérable | Univers ouvert, capacités découvertes à l'exécution |
| Composition fixe à la conception | Composition variable à l'exécution |
| Pair non probabiliste (déterministe sous contrat) | Pair réellement probabiliste |
| Coût d'erreur borné et réversible | Coût d'erreur non borné et/ou irréversible |

**Hors frontière → `Refus`** avec diagnostic. Pas de fallback silencieux. Évite la « sur-extension symétrique de la table rase » *(chap. III.8.6.2 C1)*.

### 2.2 Trois dimensions — vue synoptique

| Dimension *(III.8.4)* | Manifestation V1 | Métrique |
|---|---|---|
| **D-SENS** | Lieu de fixation du sens : avant / pendant | `[0,1]` — 0 = contrat câblé, 1 = sens négocié à l'exécution |
| **D-AUTORITÉ** | Portée de la chaîne de délégation | `[0,1]` — 0 = courte/intra-domaine/humain ancré, 1 = longue/inter-org/sans humain |
| **D-INVARIANT** | Support des invariants d'intégration | `[0,1]` — 0 = support figé, 1 = support tracé/négocié pendant |

Sondes et calibration : §5 et §11.

### 2.3 Sortie discrète et API publique

```
Regime ∈ { Deterministe, Probabiliste, Refus }
```

```go
// Decide est l'unique point de décision public. Renvoie Deterministe,
// Probabiliste ou Refus — jamais un comportement du pair appelé.
// La trace expose scores, invariants, seuils, profil de calibration.
func (k *Kernel) Decide(ctx context.Context, x Exchange) (Decision, error)
```

τ décide *où* le sens, l'autorité et le support se fixent, donc *avec quoi* appeler — jamais ce que le pair répondra.

---

## 3. Périmètre V1

### 3.1 Inclus

- Bibliothèque Go `internal/tau/` — dispatcher, frontière, opérateur τ formalisé
- Trois dimensions calculables, sondes nommées, métriques `[0,1]`
- Cinq invariants I1-I5 sous forme de cibles fuzz `FuzzI*`
- Calibration adaptative — pattern FibGo : hystérèse, `atomic.Int64`, profils versionnés, invalidation par drift
- Adaptateur `AgentMeshKafka` — validateur empirique end-to-end
- CLI minimale `cmd/tau/` — dispatch, dump de trace, rapport d'invariants, export de profil
- CI — `go test -race`, fuzz court, lint, build reproductible byte-identique, cross-compile
- `docs/theory/` aligné monographie (renvois explicites chap. III.8)

### 3.2 Exclus de V1 (reportés)

- **V2 — `cia-runtime`** : mécanisation Lean 4 des invariants *(renvoi HGL — `RechercheFondamentale.md`)*
- **V3 — `tau-stack`** : TUI Bubble Tea, replay de traces, calibration en charge, dashboard
- Couche RAG sur `ruvector.db` — étude séparée
- Service réseau (gRPC/HTTP) — V1 = lib + CLI uniquement
- Métrique de pile composée M(π) opérante *(chap. III.8.6.3)* — V1 expose l'API, V2 calcule

### 3.3 Anti-objectifs (anti-platform discipline)

TauGo **n'est pas** un framework agentique · **n'orchestre pas** d'agents · **n'embarque pas** de LLM · **ne fait pas de RAG** · **ne prédit aucun comportement** · **n'opère pas hors frontière** (refus explicite + diagnostic).

*Toute PR qui érode ces anti-objectifs est refusée, ou exige une mise à jour explicite de cette section.*

---

# Bloc II — Théorie opérationnalisée

## 4. L'opérateur τ — formalisation exécutable

*Renvoi canonique : chap. III.8.3.*

### 4.1 Définition

```
τ : t_fix(g) ≺ t_int  ↦  t_fix(g) ≈ t_int
```

où `g` est une grandeur d'interopérabilité (sens, autorité, support d'invariant), `t_fix(g)` l'instant où elle est fixée, `t_int` l'instant de l'interaction, `≺` la précédence stricte, `≈` la simultanéité opérationnelle.

### 4.2 Propriétés exploitables

| Propriété *(III.8.3.1)* | Conséquence exécutable |
|---|---|
| τ opère sur `t_fix`, jamais sur le contenu de `g` | TauGo ne réécrit pas les capacités ; il décide *quand* leur résolution s'effectue. Base I1. |
| τ non trivial seulement si `t_fix(g) ≺ t_int` peut être strictement violé sans détruire `g` | TauGo n'applique τ qu'aux échanges dont la migration est elle-même possible. Base I2. |
| L'application de τ à une grandeur n'entraîne pas mécaniquement son application à une autre | Les trois dimensions sont scoreées **indépendamment** ; seule contrainte : I4 (cohérence). Base de l'orthogonalité. |

### 4.3 Encodage exécutable de la frontière

```go
type FrontierCheck struct {
    UniversOuvert       bool  // capacités découvertes à l'exécution
    CompositionVariable bool  // composition à l'exécution
    PairProbabiliste    bool  // raisonneur LLM ou équivalent
    CoutNonBorne        bool  // erreur non bornée ou irréversible
}

func (f FrontierCheck) Inside() bool {
    return f.UniversOuvert && f.CompositionVariable &&
           f.PairProbabiliste && f.CoutNonBorne
}
```

**Garde V1** *(M0)* — `Inside() == false` → `Refus(diag: "hors frontière τ")`. Test : `TestRefusHorsFrontiere`.

### 4.4 Asymétrie ontologique de τ_AUTORITÉ

*Renvoi : chap. III.8.4.2.bis — Searle (1995), faits institutionnels vs protocolaires.*

τ n'est **pas symétrique** sur ses trois dimensions :

- `τ_SENS` et `τ_INVARIANT` — faits protocolaires, instaurables in-band par accord entre pairs. Coûteux mais applicables.
- `τ_AUTORITÉ` — fait institutionnel ; déplacement vers l'exécution = instaurer un fait sans institution. **Ontologiquement bloqué** sans institution émettrice externe.

**Encodage V1** *(M2)* :

```go
type Attestation struct {
    Emetteur   string  // RFC, juridiction, consortium nommé
    Reference  string  // URI ou identifiant opposable
    Marqueur   string  // "Confirmé" | "Probable" | "Hypothèse" | "À vérifier"
    AssertedAt time.Time
}
```

`D-AUTORITÉ ≥ θ_auth_block ∧ Attestation == nil` → `Refus(diag: "I3 — verrou ontologique D-AUTORITÉ")`. Test : `TestRefusOntologiqueDAUTORITE`.

---

## 5. Les trois dimensions

*Renvoi canonique : chap. III.8.4. Granularité = **par frontière d'interopérabilité**, pas par système ; un système réel mélange des frontières des deux pôles.*

### 5.1 D-SENS — lieu de fixation du sens

**Question opérante** *(III.8.4.1)* : *le pair qui consomme une capacité décide-t-il de l'invoquer à partir d'une interprétation produite à l'exécution, ou à partir d'un câblage produit à la conception ?*

| Sonde | Indicateur | Poids initial *(à calibrer M4)* |
|---|---|---|
| `S_contract` | Présence d'un contrat de forme publié, versionné, opposable | 0.35 |
| `S_runtime_resolve` | Résolution sémantique à l'exécution (embedding, NL parsing) | 0.30 |
| `S_capability_discovery` | Découverte dynamique (MCP `list_tools`, A2A équivalent) | 0.20 |
| `S_reasoner_intent` | Interprétation d'intention par raisonneur probabiliste | 0.15 |

`D_SENS = Σ wᵢ · Sᵢ(x)`. *Statut : hypothèse — pondérations initiales, à corroborer sur traces AgentMeshKafka.*

### 5.2 D-AUTORITÉ — portée de la chaîne de délégation

**Question opérante** *(III.8.4.2)* : *la chaîne est-elle longue, dynamique, inter-organisationnelle, sans humain ancré ?*

| Sonde | Indicateur | Poids initial |
|---|---|---|
| `A_chain_depth` | Profondeur de la chaîne de délégation | 0.25 |
| `A_cross_org` | Traverse une frontière organisationnelle | 0.25 |
| `A_human_anchor` | Humain dans la boucle (inversé) | 0.25 |
| `A_dynamic_resolution` | Autorité résolue à l'exécution vs pré-câblée | 0.25 |

**Garde V1** — `A_attestation_institutionnelle` (booléen, hors agrégation) déclenche refus ontologique §4.4.

### 5.3 D-INVARIANT — support des invariants d'intégration

**Question opérante** *(III.8.4.3)* : *le support de l'invariant repose-t-il sur un artefact figé avant l'interaction, ou tracé / négocié / observé pendant ?*

**Contrainte de cohérence** *(III.8.4.5, I4)* — D-INVARIANT est **contraint par D-SENS** : `i ≈ pendant ⟹ s ≈ pendant`. Direction dissymétrique : c'est D-SENS qui contraint D-INVARIANT, jamais l'inverse.

| Sonde | Indicateur | Poids initial |
|---|---|---|
| `I_event_registry` | Registre d'effets tracé à l'exécution | 0.30 |
| `I_idempotency_derived` | Clé d'idempotence dérivée de l'intention (vs imposée) | 0.25 |
| `I_capability_mediation` | Médiation de capacités négociée pendant l'échange | 0.25 |
| `I_enumerated_plan` | Plan d'étapes énuméré à la conception (inversé) | 0.20 |

### 5.4 Synoptique

| | D-SENS | D-AUTORITÉ | D-INVARIANT |
|---|---|---|---|
| **Pôle 0** *(avant)* | Contrat figé, publié, opposable | Chaîne courte, intra-domaine, humain ancré | Support énuméré à la conception |
| **Pôle 1** *(pendant)* | Capacité décrite, découverte, interprétée | Chaîne longue, inter-org, sans humain | Support tracé / négocié / observé |
| **Nature** | Fait protocolaire | **Fait institutionnel** (Searle 1995) | Fait protocolaire |
| **τ applicable** | Oui (coûteux) | **Conditionné à institution externe** | Oui (coûteux) |
| **Contrainte** | Contraint D-INVARIANT (I4) | Indépendant en valeur, asymétrique en maturité (I3) | Contraint par D-SENS (I4) |

---

## 6. Les cinq invariants — reformulation exécutable

*Renvoi canonique : chap. III.8.5. Verbatim disponible dans `InteroperabiliteAgentique/Monographie.md` lignes ~5723-5737.*

### 6.1 Tableau-maître

| # | Énoncé monographie | Statut | Reformulation exécutable | Cible fuzz |
|---|---|---|---|---|
| **I1** | τ déplace l'instant de fixation d'une grandeur **sans altérer la grandeur** | Probable | Pour tout échange `x` admissible, `Conserve(x, τ(x)) == true` *(égalité modulo équivalence métier déclarée)* | `FuzzI1_Conservation` |
| **I2** | Le résidu migrant est **non vide et non recâblable hors ligne** sans détruire l'agentivité | Confirmé par construction | Pour tout `x` dans la frontière, `Residu(x) := { g | t_fix(g) ≈ t_int } ≠ ∅` ; tout `Recablage(x)` qui vide le résidu doit faire perdre ≥ 1 condition de frontière | `FuzzI2_Irreductibilite` |
| **I3** | Trois dimensions **orthogonales en valeur, asymétriques en maturité** ; D-AUTORITÉ = fait institutionnel sans support à 2026-05-16 | Probable, daté 2026-05-16 | `D-AUTORITÉ(x) ≥ θ_auth_block ∧ Attestation == nil ⇒ Refus`. Clause de péremption : `date_revision ≤ 2027-01-01` dans le profil | `FuzzI3_AsymetrieAutorite` |
| **I4** | D-INVARIANT contraint par D-SENS : `i ≈ pendant ⟹ s ≈ pendant` ; combinaisons incohérentes **observables** | Hypothèse, empiriquement testable | `D-INVARIANT(x) ≥ θ_inv ∧ D-SENS(x) < θ_sens ⇒ Refus(diag: "I4")` | `FuzzI4_CoherenceContrainte` |
| **I5** | Pile hérite de la **conjonction** des angles morts ; pas de réconciliation transversale sauf hors pile | Probable | Pour pile `π = [C₁,…,Cₙ]`, `M(π) = |⋃Aᵢ|` satisfait `M(π) ≥ max(|Aᵢ|)` et `M(π) ≤ Σ|Aᵢ|`. V1 expose l'API d'agrégation ; V2 calcule | `FuzzI5_CompositionConjonctive` |

### 6.2 Conditions de réfutation observables

| # | Réfutation *(III.8.5)* | Test négatif TauGo |
|---|---|---|
| I1 | Exhibition d'une grandeur d'interopérabilité que l'agentivité *supprime* | `TestRefutationI1_GrandeurSupprimee` |
| I2 | Méthode d'ingénierie ramenant intégralement une frontière agentique au cas câblé hors ligne *tout en préservant* les 4 conditions de frontière | `TestRefutationI2_RecablageComplet` |
| I3 | Accession d'un standard d'identité agentique déléguée au statut de RFC. **Revérification au 2026-12-01.** | `TestI3_DateRevisionRespectee` |
| I4 | Système opérant durablement avec sens négocié à l'exécution mais support d'invariant figé | `TestRefutationI4_CombinaisonIncoherente` |
| I5 | Pile où une couche referme l'angle mort d'une autre *sans* mécanisme transversal ajouté | `TestRefutationI5_AngleMortReferme` |

### 6.3 Articulation et priorités V1

- **I1 + I2** fondent l'opérateur : conservation + non-trivialité. Garde combinée : `TestOperatorWellDefined`.
- **I3 + I4** caractérisent la structure : asymétrie de maturité + contrainte de cohérence. Garde : `TestSpaceNonFlat`.
- **I5** régit la composition. Garde V2 : `TestM_Monotonicity`.

**Priorité empirique #1** : **I4** (Hypothèse non encore testée) — campagne dédiée en M4, rapport `docs/empirical/I4-report.md`.

---

## 7. Conditions de validité & anti-patrons d'usage

*Renvoi : chap. III.8.6.2 (conditions) et III.8.7 (anti-patrons). Garde opérationnelle pour chaque ligne.*

### 7.1 Conditions de validité en environnement gouverné

| # | Condition *(III.8.6.2)* | Encodage TauGo |
|---|---|---|
| C1 | La frontière agentique est réelle (4 conditions classiques toutes violées) | `FrontierCheck.Inside()` §4.3 |
| C2 | D-AUTORITÉ = facteur limitant, non résolu (I3 = contrainte de conception) | `θ_auth_block` conservateur (≤ 0.85), refus ontologique §4.4 |
| C3 | Modèle daté et révisable (horizon 2027-2030) | `Profile.DateRevision` ; CI échoue si `today > date_revision` sans MAJ |

### 7.2 Quatre anti-patrons d'usage interdits

| # | Anti-patron | Pourquoi *(III.8.7)* | Garde TauGo |
|---|---|---|---|
| 1 | **Usage prédictif** — `Predict*`, `Expected*`, `Forecast*` exportés | Le modèle est structurant, pas prédictif. Le substrat probabiliste interdit toute prédiction de comportement. | `TestNoPredictiveAPI` (réflexion sur méthodes exportées) ; PR rejetée |
| 2 | **Usage hors frontière** — appliquer τ à une frontière non agentique | Sur-ingénierie injustifiée, signale au client un régime agentique alors qu'il est classique | `TestRefusHorsFrontiere` ; aucun drapeau « skip frontier check » toléré |
| 3 | **Usage atemporel** — I3 sans date ni revérification | Transforme un instrument de navigation daté en assertion intemporelle | `Trace.profile.date_revision` + `profile.version_monographie` ; CI échoue si périmé |
| 4 | **Usage clos** — tenir les 3 dimensions et 5 invariants pour exhaustifs | Hypothèse de complétude non acquise (chap. III.8.7) | `Decision.Trace.UnmodeledObservations []string` ; rapport mensuel `docs/empirical/unmodeled.md` |

### 7.3 Refus — décision de premier rang

| Cas | Diagnostic | Renvoi |
|---|---|---|
| Hors frontière | `hors frontière τ` | §4.3 |
| Verrou ontologique D-AUTORITÉ (I3) | `I3 — verrou ontologique` | §4.4 |
| Incohérence I4 (`s < θ_sens ∧ i ≥ θ_inv`) | `I4 — incohérence détectée` | §6.1 |
| Profil périmé | `profil périmé — veille requise` | §11.4 |
| Observation non modélisée à fort impact | `usage clos potentiel` | §7.2 #4 |

**Refus n'est pas un échec** : c'est une décision pleine, instrumentée, opposable. La trace expose le diagnostic, les scores qui l'ont produit, le profil en vigueur et le renvoi III.8.

---

# Bloc III — Architecture & ingénierie

## 8. Architecture cible

Clean Architecture, **quatre couches strictes**, calque structurel de FibGo. Dépendances **unidirectionnelles descendantes**, gardées par `internal/arch_test.go`.

```
cmd/
  tau/                           # CLI principale
  generate-golden/               # oracle indépendant (V1.1)
internal/
  app/                           # lifecycle, dispatch top-level, injection LLM
  tau/                           # CŒUR : opérateur τ
    {operator, frontier}.go      # τ formalisé (§4) ; FrontierCheck (§4.3)
    dimensions/{dsens, dauthority, dinvariant}.go + probes/
    invariants/{i1..i5}.go + fuzz_targets.go
  orchestration/                 # dispatcher (§10) ; Decision ; Trace immuable
  calibration/                   # Profile, drift, thresholds (atomic.Int64)
  bridge/
    agentmeshkafka/              # validateur empirique
    llm/                         # interface client LLM injecté + stub déterministe
  {config, errors, metrics, testutil}/
docs/
  theory/                        # renvois III.8.* (03-tau, 04-dimensions, 05-invariants, 06-conditions, 07-anti-patrons)
  algorithms/                    # dispatch, calibration, drift
  adr/                           # 0001-clean-arch, 0002-go-1.25, 0003-llm-injection, 0004-agentmeshkafka
  empirical/                     # I4-report, unmodeled, fuzz-summary
test/{e2e, golden}/
CLAUDE.md · README.md · LICENSE · Makefile · .golangci.yml · go.mod
```

### 8.1 Couches et étanchéité

| Couche | Packages | Importe | N'importe PAS |
|---|---|---|---|
| **1 Présentation** | `cmd/tau`, `internal/app` | `orchestration`, `config`, `errors` | `tau/*`, `bridge/*` directement |
| **2 Orchestration** | `orchestration` | `tau`, `calibration`, `errors`, `metrics` | `bridge/*` directement (passe par interfaces injectées en `app`) |
| **3 Domaine** | `tau`, `tau/dimensions`, `tau/invariants` | `errors`, `metrics` | `orchestration`, `bridge`, `app`, `cmd` |
| **4 Infrastructure** | `bridge/*`, `calibration` *(persistance)* | `errors`, `metrics` | `tau/*`, `orchestration` |

**Gardes architecturales** *(M0)* — `internal/arch_test.go` interdit :

- `tau/* → orchestration` · `tau/* → bridge` · `bridge → tau/*` direct
- `dimensions ↔ invariants` (orthogonalité encodée)
- Import LLM concret hors `app/` et `bridge/llm/`

### 8.2 Patterns réutilisés de FibGo

- Seuils dynamiques en `atomic.Int64` privés, accesseurs publics *(`calibration/thresholds.go`)*
- Étanchéité par `arch_test.go` *(règles propres TauGo)*
- Aucun global mutable dans `tau/*` ; seules les `Thresholds` mutables, via accesseurs atomiques
- Erreurs typées (`DispatchError`, `RefusError`, `CalibrationError`)
- `t.Parallel()` cible 100 % adoption M1
- Sentinel panic re-propagé (calque `bigfft/fermat.go`) pour invariants internes cassés

---

## 9. Modèle de données

### 9.1 Types canoniques

```go
package tau

// Exchange — l'échange d'interopérabilité agentique soumis à τ.
type Exchange struct {
    ID                          string
    Initiator                   Principal
    Target                      Capability
    IntentDescription           string
    DiscoveredAt                time.Time
    AttestationInstitutionnelle *Attestation       // nil si non fournie
    Context                     map[string]any
}

type Principal struct {
    ID              string
    HumanInLoop     bool
    Organization    string
    DelegationDepth int                            // 0 = humain direct
}

type Capability struct {
    ID            string
    DiscoveryMode DiscoveryMode                    // Static | DynamicMCP | DynamicA2A | DynamicAGNTCY
    ContractURI   string                           // vide = pas de contrat
}

// Score — un score normalisé [0,1] avec sa traçabilité.
type Score struct {
    Value      float64
    Probes     map[string]float64                  // valeurs des sondes individuelles
    Weights    map[string]float64                  // poids en vigueur
    ComputedAt time.Time
}

// Decision — sortie complète de Kernel.Decide.
type Decision struct {
    Regime         Regime                          // Deterministe | Probabiliste | Refus
    Trace          Trace                           // instrumentation complète, immuable
    Diagnostic     string                          // non vide ⟺ Regime == Refus
    ProfileVersion string
    DateRevision   time.Time
}

type Regime int
const (
    RegimeUnknown Regime = iota
    Deterministe
    Probabiliste
    Refus
)

// Trace — instrumentation immuable d'une décision.
type Trace struct {
    ExchangeID            string
    DSens, DAuthority,
    DInvariant            Score
    TauScore              float64                  // composite pondéré
    Frontier              FrontierCheck
    Invariants            InvariantStatuses
    Thresholds            Thresholds
    UnmodeledObservations []string                 // §7.2 #4
    DurationNs            int64
}

type InvariantStatus struct {
    ID         string                              // "I1" à "I5"
    Status     string                              // "ok" | "violated" | "n/a"
    Marqueur   string                              // marqueur épistémique §6
    Diagnostic string                              // non vide si violated
}

type InvariantStatuses struct { I1, I2, I3, I4, I5 InvariantStatus }
```

### 9.2 Profil de calibration

```go
package calibration

type Profile struct {
    ID                  string
    Version             string                     // SemVer
    CreatedAt           time.Time
    DateRevision        time.Time                  // péremption §7.1 C3
    VersionMonographie  string                     // tag monographie épinglé
    CPUFingerprint      string                     // invalidation matériel (calque FibGo)
    ModelLLMFingerprint string                     // invalidation modèle LLM
    CorpusFingerprint   string                     // invalidation corpus
    Thresholds          Thresholds
    Weights             Weights
}

type Thresholds struct {
    Deterministe   float64                         // τ_score < θ → déterministe
    Probabiliste   float64                         // τ_score ≥ θ → probabiliste
    AuthBlock      float64                         // refus ontologique D-AUTORITÉ
    SensCoherence,
    InvCoherence   float64                         // gardes I4
    HysteresisGap  float64
}

type Weights struct {
    DSens, DAuthority, DInvariant float64          // somme = 1.0
    SensProbes,
    AuthorityProbes,
    InvariantProbes map[string]float64             // somme par dimension = 1.0
}
```

### 9.3 Invariants des types — gardes de test

| Invariant | Garde |
|---|---|
| `Decision.Trace` toujours non nul | `TestDecisionAlwaysTraced` |
| `Decision.Regime == Refus ⟺ Decision.Diagnostic ≠ ""` | `TestRefusImpliesDiagnostic` |
| `Σ Weights.SensProbes == 1.0 ± ε` (idem 2 autres dimensions) | `TestProbeWeightsSumToOne` |
| `Profile.DateRevision > Profile.CreatedAt` | `TestProfileRevisionAfterCreation` |
| Tous les `Score.Value ∈ [0, 1]` | `TestScoreBounded` |
| `Trace` immuable post-construction | `TestTraceImmutable` |

---

## 10. Algorithme de dispatch

### 10.1 Pseudo-algorithme — V1

```
ENTRÉE  : x Exchange, π Profile (calibration), inv InvariantStatuses (état trace)
SORTIE  : d Decision (toujours instrumentée)

1. FRONTIÈRE              (§4.3, C1)
   ¬FrontierCheck(x).Inside() ⇒ return Refus(diag: "hors frontière τ")

2. GARDE ONTOLOGIQUE      (§4.4, I3)
   a := ScoreDAutorite(x, π)
   a.Value ≥ π.Thresholds.AuthBlock ∧ x.Attestation == nil
     ⇒ return Refus(diag: "I3 — verrou ontologique D-AUTORITÉ")

3. GARDE PÉREMPTION       (§7.1 C3, anti-patron #3)
   today > π.DateRevision ⇒ return Refus(diag: "profil périmé — veille requise")

4. SCORES                 (§5)
   s := ScoreDSens(x, π)
   i := ScoreDInvariant(x, π)

5. GARDE COHÉRENCE I4     (§6)
   i.Value ≥ π.Thresholds.InvCoherence ∧ s.Value < π.Thresholds.SensCoherence
     ⇒ return Refus(diag: "I4 — combinaison incohérente détectée")

6. COMPOSITE τ
   τ_score := π.Weights.DSens · s.Value
            + π.Weights.DAuthority · a.Value
            + π.Weights.DInvariant · i.Value

7. DÉCISION AVEC HYSTÉRÈSE (invariant : θ_d ≤ θ_p)
   τ_score < θ_d              ⇒ return Deterministe
   τ_score ≥ θ_p              ⇒ return Probabiliste
   sinon (zone hystérèse)     ⇒ return LastRegime(x.ID, default: Deterministe)

8. ÉVALUATION INVARIANTS
   inv := EvaluateInvariants(x, decision, π)
   inv.AnyViolated() ⇒ trace.UnmodeledObservations += inv.Summary()
```

**L'ordre des étapes 1-8 n'est pas arbitraire** : frontière → ontologie → péremption → scores → cohérence → composite → hystérèse → invariants. Réordonner = casser une garde.

### 10.2 Interface publique

```go
type Kernel interface {
    Decide(ctx context.Context, x Exchange) (Decision, error)
    Calibrate(ctx context.Context, corpus CalibrationCorpus) (Profile, error)
    CurrentProfile() Profile
}
```

### 10.3 Instrumentation

Toute décision produit une `Trace` non-mutable couvrant : scores avec sondes et poids · état de la frontière · `τ_score` · seuils · état des cinq invariants · profil (version, `date_revision`) · durée · observations non modélisées. Vérifié par `TestDecisionAlwaysTraced` + `TestTraceImmutable`.

---

## 11. Calibration adaptative

*Discipline héritée de `FibGo/internal/calibration/` : atomic, hystérèse, profils versionnés persistés, invalidation par drift de fingerprint.*

### 11.1 Paramètres calibrés

| Paramètre | Domaine | Init | Influence |
|---|---|---|---|
| `Thresholds.Deterministe` | `[0,1]` | 0.35 | Régime déterministe en deçà |
| `Thresholds.Probabiliste` | `[0,1]` | 0.65 | Régime probabiliste au-delà |
| `Thresholds.AuthBlock` | `[0,1]` | 0.85 | Refus ontologique D-AUTORITÉ |
| `Thresholds.SensCoherence` | `[0,1]` | 0.50 | Garde I4 |
| `Thresholds.InvCoherence` | `[0,1]` | 0.50 | Garde I4 |
| `Thresholds.HysteresisGap` | `[0, 0.2]` | 0.10 | Largeur de la bande |
| `Weights.D*` (composite) | `[0,1]`, somme = 1 | `(0.4, 0.3, 0.3)` | Pondération `τ_score` |
| `Weights.*Probes` | par sonde, somme par dimension = 1 | §5 | Pondération interne |

*Statut : hypothèse — initialisations à corroborer sur traces AgentMeshKafka M4.*

### 11.2 Pattern atomic (calque FibGo `bigfft/fft.go`)

```go
type Thresholds struct {
    deterministe atomic.Int64                      // milli-unités, lecture sans verrou
    // ...
}

func (t *Thresholds) Deterministe() float64 {
    return float64(t.deterministe.Load()) / 1000.0
}

func (t *Thresholds) SetDeterministe(v float64) {
    t.deterministe.Store(int64(v * 1000))
}
```

**Invariant** : `Thresholds.Deterministe() ≤ Thresholds.Probabiliste()` en tout temps. Violation = panic interne sentinel (calque FibGo `bigfft/fermat.go`).

### 11.3 Persistance des profils

Format JSON sous `~/.config/taugo/profiles/{ID}-{Version}.json`. Profil actif = symlink `current.json`.

```json
{
  "id": "default",
  "version": "0.1.0",
  "date_revision": "2026-11-23T00:00:00Z",
  "version_monographie": "v2.4.3",
  "cpu_fingerprint": "AMD-Ryzen-5900X-..",
  "model_llm_fingerprint": "claude-opus-4-7:8b3a..",
  "corpus_fingerprint": "agentmeshkafka-2026-05.sha256:..",
  "thresholds": { ... },
  "weights": { ... }
}
```

### 11.4 Invalidation par drift

| Drift | Détection | Action |
|---|---|---|
| `cpu_fingerprint` change | Hash `cpuid` au démarrage | Recalibration en arrière-plan, profil marqué `stale` |
| `model_llm_fingerprint` change | Empreinte client LLM injecté | Idem |
| `corpus_fingerprint` change | Hash du corpus de calibration | Idem |
| `today > date_revision` | Au démarrage + chaque `Decide` | `Refus(diag: "profil périmé")` — pas de fallback |
| Distribution des scores hors zone calibrée | Statistique fenêtre glissante *(M5)* | Marqueur `drift_warning` dans la trace |

### 11.5 Calibration déterministe

```bash
tau calibrate \
  --corpus path/to/agentmeshkafka-traces.jsonl \
  --output ~/.config/taugo/profiles/run-2026-05-23.json \
  --date-revision 2026-11-23 \
  --version-monographie v2.4.3
```

**Reproductible byte-identique à corpus fixé** : même corpus + même seed → mêmes seuils + mêmes poids. Vérifié par `TestCalibrationDeterministic`.

---

## 12. Bridges externes

### 12.1 `AgentMeshKafka` — validateur empirique

Le pont expose un DTO local `AgentMeshExchange` (miroir nominal de `tau.Exchange`, type délibérément distinct) afin de préserver l'étanchéité Clean Architecture : `arch_test.go` interdit `bridge/agentmeshkafka → tau` (lignes 32-34). La conversion vers `tau.Exchange` est hébergée en `internal/app/agentmesh.go` via `app.ToTauExchange` et `app.StreamAsTauExchanges`, seule couche autorisée à voir simultanément `bridge/*` et `tau/*`. *(ADR-0005)*

```go
package agentmeshkafka

// AgentMeshExchange est un DTO local — miroir nominal de tau.Exchange mais
// type délibérément distinct (ADR-0005, étanchéité Clean Architecture).
// La conversion vers tau.Exchange est hébergée en internal/app/agentmesh.go.
type Adapter interface {
    Stream(ctx context.Context, topics []string) (<-chan AgentMeshExchange, <-chan error)
    Close() error
}
```

*Statut : Confirmé par ADR-0005 (DTO neutre, M4). La signature initiale qui retournait `tau.Exchange` violait `arch_test.go` ligne 32 — corrigée en M4. Dépendance résiduelle : stabilité du schéma AgentMeshKafka. (Hypothèse — dépend de la stabilité d'AgentMeshKafka au-delà de M4.)*

### 12.2 `LLMClient` injecté

```go
package llm

// Client est l'interface étroite que TauGo consomme.
// Aucun LLM n'est embarqué ; l'implémentation est injectée par app.
type Client interface {
    // Fingerprint identifie modèle + version + paramètres figés.
    // Utilisé pour invalidation de profil (§11.4).
    Fingerprint() string

    // Interpret renvoie un score d'interprétation [0,1] pour une
    // description d'intention. Utilisé par la sonde S_reasoner_intent
    // de D-SENS (§5.1). Doit être déterministe sous mêmes paramètres
    // (température 0).
    Interpret(ctx context.Context, intent string) (float64, error)
}
```

**Stub déterministe obligatoire** — `internal/bridge/llm/stub.go` fournit un mapping `intent → score` checked-in. Évite la dépendance LLM externe en CI ; garantit calibration reproductible.

**Garde** — aucun import de package LLM concret (`anthropic`, `openai`, …) dans `internal/tau/*` ou `internal/orchestration/*`. Injection en `internal/app/`. Vérifié par `arch_test.go`.

---

# Bloc IV — Discipline d'exécution

## 13. Stack technique

| Composant | Choix | Statut |
|---|---|---|
| **Go** | 1.25.0+ (toolchain 1.26.x), aligné FibGo | Confirmé |
| **Module** | `github.com/agbruneau/taugo` | À vérifier (username GitHub à confirmer M0) |
| **Licence** | Apache-2.0 | Confirmé |
| **Dépendances** | `golang.org/x/sync/errgroup`, stdlib `log/slog`, `math/big` *(si scoring l'exige)* | Probable |
| **Aucun framework** | Pas de Bubble Tea V1, ni gRPC, ni cobra ; `flag` standard | Confirmé (§3.3) |
| **LLM** | Injecté via interface §12.2 ; aucune dépendance concrète en `tau/*` | Confirmé |
| **Lint** | `golangci-lint v1.64.8` épinglé, config calque FibGo (24 linters, `govet shadow`, complexité max 15/30) | Confirmé |
| **Build reproductible** | `-trimpath`, `-buildvcs=true`, timestamp gelé `1778889600` (calque InteroperabiliteAgentique) | Confirmé |
| **PGO** | Optionnel `make build-pgo`, profil checked-in après M3 | Probable |
| **Cross-compile** | linux/{amd64,arm64}, darwin/{amd64,arm64}, windows/amd64 | Confirmé |
| **Race detector** | `go test -race` sur 3 OS via CGO (Linux/macOS) | Confirmé |
| **Fuzz** | `FuzzI1`-`FuzzI5` ; 30 s CI courte, 24 h nocturne | Confirmé |

---

## 14. Conventions

### 14.1 Éditoriales (héritées `InteroperabiliteAgentique/CLAUDE.md` §1.1-§1.8)

| Convention | Application TauGo |
|---|---|
| **Langue** | FR-CA pour `PRD.md`, `CLAUDE.md`, `docs/`, commentaires structurants. **Godoc en anglais.** |
| **Typographie française** | Espaces insécables U+00A0 avant `: ; ? ! »` et après `«` ; guillemets `« … »`. **Cible M6** ; M0-M5 = dette éditoriale assumée. |
| **Marqueurs d'incertitude** | Obligatoires dans `docs/` sur affirmation datée : `Confirmé · Probable · Hypothèse · À vérifier · Je ne sais pas (avec piste)`. |
| **Citations** | Style auteur-date `(Nom, année)`. Pagination pour citation directe. |
| **Renvois croisés monographie** | Chaque décision théorique dans `docs/theory/` cite `*(chap. III.8.X.Y)*` en italique. |
| **Patrons de raisonnement** | Recommandation = (1) compromis principal · (2) ≥ 1 alternative · (3) conditions de retournement. |
| **Anonymisation** | Aucun cas Desjardins identifiable. Références publiques libres (MCP, A2A, AGNTCY, IBM, IETF, NVIDIA, RFC). |
| **Pas d'emoji** | Aucun emoji dans code, commits ou docs sauf demande explicite. |
| **Zéro fabrication** | Aucune citation, statistique, API, version, date, DOI inventée. Fabrication détectée = livrable invalidé. |

### 14.2 Code (calque FibGo)

- Packages par responsabilité, jamais par feature
- Interfaces étroites (ISP) ≤ 5 méthodes : `Kernel`, `Adapter`, `Client`, `TraceReporter`
- Erreurs structurées (`fmt.Errorf("%w", err)`, types typés). **Pas de panic** sauf invariants internes — sentinel re-propagé via classifier *(calque `bigfft/fermat.go`)*
- `t.Parallel()` systématique (cible 100 %)
- Complexité max : cyclomatique 15, cognitive 30 ; fonction ≤ 100 LOC / 50 statements
- `doc.go` par package public, obligatoire M0 pour `tau`, `orchestration`, `calibration`
- Commentaires : *pourquoi*, jamais *quoi*. Pas de référence au caller ni à la tâche
- **Pas de globaux mutables non synchronisés** dans `tau/*` — exception = ADR

### 14.3 Commits — Conventional Commits

`<type>(<scope>): <description>` avec types : `feat · fix · perf · refactor · test · docs · chore · theory`. `theory` = mise à jour `docs/theory/` motivée par révision monographie.

Co-signature obligatoire pour commits assistés par IA :

```
Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

### 14.4 Lignes interdites (PR-blocking)

| Construction | Garde |
|---|---|
| `Predict*` / `Expected*` / `Forecast*` dans l'API publique de `tau` | Anti-patron §7.2 #1 ; `TestNoPredictiveAPI` |
| Imports LLM concrets dans `internal/tau/*` ou `internal/orchestration/*` | §12.2 ; `arch_test.go` |
| Globaux mutables non synchronisés dans `tau/*` | §14.2 ; `gochecknoglobals` |
| Citation sans référence vérifiable dans `docs/` | §14.1 ; audit manuel |
| Affirmation datée sans marqueur d'incertitude dans `docs/` | §14.1 ; revue PR |
| Suppression de garde dans `arch_test.go` sans ADR | §8.1 ; revue PR |

---

## 15. Stratégie de test

### 15.1 Pyramide stratifiée (calque FibGo)

| Niveau | Cible | Outil | Couverture cible |
|---|---|---|---|
| **Unit** | Chaque fonction publique, sonde, score | `go test` standard | ≥ 80 % / package |
| **Property-based** | Pureté, monotonie, idempotence | `gopter` (calque FibGo) | Toutes les propriétés algébriques déclarées |
| **Fuzz** | I1-I5, bordures de frontière | `go test -fuzz` | 30 s / cible en CI courte ; 24 h hebdomadaire |
| **Golden** *(V1.1)* | Traces de référence, non-régression de décision | `internal/testdata/golden/` ; oracle `cmd/generate-golden/` | Immuable sans ADR |
| **Architecture** | Étanchéité des couches *(§8.1)* | `internal/arch_test.go` | 100 % des règles |
| **E2E** *(M4+)* | Via `AgentMeshKafka` | `test/e2e/` | ≥ 1 scénario par régime |
| **Empirique I4** *(M4+)* | Détection combinaisons incohérentes sur traces réelles | `docs/empirical/I4-report.md` | Campagne dédiée |

### 15.2 Cibles fuzz I1-I5

```go
// internal/tau/invariants/fuzz_targets.go
func FuzzI1_Conservation(f *testing.F)      // τ(x).grandeur ≡ x.grandeur
func FuzzI2_Irreductibilite(f *testing.F)   // tout recâblage hors ligne détruit ≥ 1 condition de frontière
func FuzzI3_AsymetrieAutorite(f *testing.F) // jamais Probabiliste avec D-AUTORITÉ ≥ θ_auth_block ∧ Attestation == nil
func FuzzI4_CoherenceContrainte(f *testing.F) // (s < θ_sens, ·, i ≥ θ_inv) ⇒ Refus(I4)
func FuzzI5_CompositionConjonctive(f *testing.F) // M(π) ≥ max(|Aᵢ|), M(π) ≤ Σ|Aᵢ|
```

Commande de référence : `go test -fuzz=FuzzI4_CoherenceContrainte -fuzztime=30s ./internal/tau/invariants/`.

### 15.3 Gates CI

| Gate | Seuil | Action |
|---|---|---|
| Couverture globale | ≥ 80 % | Bloque merge |
| Couverture `tau/*` | ≥ 90 % | Bloque merge |
| Race detector | 0 warning | Bloque merge |
| Lint | 0 warning | Bloque merge |
| Reproductibilité build | hash byte-identique entre 2 builds | Bloque merge |
| Fuzz court (30 s sur I1-I5) | 0 panique, 0 crash | Bloque merge |
| Profil ≥ 6 mois avant `date_revision` | — | Avertissement |

### 15.4 Stub LLM déterministe

`internal/bridge/llm/stub.go` implémente `Client` avec mapping `intent → score` checked-in. Permet CI sans dépendance LLM externe, calibration reproductible, tests d'invariants sans variance.

**Garde** — tout `go test ./...` sans `TAUGO_LLM_BACKEND=real` utilise le stub. `TestDefaultLLMIsStub`.

---

# Bloc V — Programme

## 16. Roadmap V1

| Milestone | Contenu | Critère d'acceptation |
|---|---|---|
| **M0** | Squelette repo, CI 3 OS, `CLAUDE.md`, `.golangci.yml`, `arch_test.go`, `FrontierCheck`, `cmd/tau` minimal | `git init` + premier commit vert ; tag `v0.0.1-alpha` ; `TestRefusHorsFrontiere` passe |
| **M1** | Dispatcher minimal, deux régimes, stub LLM | `tau decide --input fixture.json` rend une `Decision` instrumentée |
| **M2** | Trois dimensions + score τ composite + gardes ontologique D-AUTORITÉ et I4 | Rapport décision avec scores/sondes/poids ; `TestRefusOntologiqueDAUTORITE` + `TestI4_IncoherenceDetectee` passent |
| **M3** | Cinq invariants comme cibles fuzz | `go test -fuzz=. -fuzztime=30s ./internal/tau/invariants/` vert sur I1-I5 ; rapport `docs/empirical/fuzz-summary.md` |
| **M4** | Adaptateur `AgentMeshKafka` + campagne empirique I4 | Trace end-to-end ; rapport `docs/empirical/I4-report.md` avec ≥ 100 traces analysées |
| **M5** | Calibration adaptative + persistance versionnée + détection de drift | `tau calibrate` reproductible byte-identique sur corpus fixé ; `TestCalibrationDeterministic` passe |
| **M6** | Documentation alignée monographie + typographie française + release `v0.1.0` | Tag, `CHANGELOG.md`, `README.md`, `docs/theory/` complet avec renvois III.8 |

**Livrables M0 minimaux** : `go.mod` · `Makefile` · `.golangci.yml` · `.github/workflows/{ci,coverage}.yml` · `internal/tau/operator.go` *(panic `not implemented`)* · `internal/tau/frontier.go` + test · `internal/arch_test.go` · `cmd/tau/main.go` *(squelette)* · `docs/theory/03-operateur-tau.md` · `LICENSE` · `CHANGELOG.md`.

**Estimation indicative** : 6-10 semaines à temps partiel. *À vérifier selon disponibilité réelle.*

**Cadence de revue** :

- **Mensuelle** sur dérive de scope *(§3.3 anti-objectifs)*
- **Trimestrielle** sur péremption I3 *(`date_revision`)* et veille statut RFC d'identité agentique déléguée
- **Post-M3** : la reformulation exécutable des invariants tient-elle ? Réajustement éventuel après campagne fuzz

---

## 17. Critères de succès V1

*Checklist falsifiable — chaque item vérifiable par un test ou un artefact.*

| # | Critère | Vérification |
|---|---|---|
| 1 | Dispatch τ instrumenté sur cas BFSI réaliste anonymisé | `docs/empirical/case-study-bfsi.md` |
| 2 | Cinq invariants exécutables, fuzz ≥ 30 s sans panique | `go test -fuzz=FuzzI*_* -fuzztime=30s ./internal/tau/invariants/` vert |
| 3 | Trace empirique end-to-end via `AgentMeshKafka` | `test/e2e/agentmeshkafka_test.go` vert |
| 4 | Build reproductible byte-identique en CI | Deux builds successifs même commit → même SHA256 |
| 5 | Couverture ≥ 80 % global, ≥ 90 % sur `tau/*` | `coverage.yml` vert |
| 6 | Chaque décision design dans `docs/` renvoie chap. III.8 | Lint manuel + grep |
| 7 | Aucun emoji, aucune fabrication, aucune citation non sourçée | Audit textuel M6 |
| 8 | Trois OS supportés (Linux/macOS/Windows) | CI matrix verte |
| 9 | Quatre anti-patrons gardés par tests *(§7.2)* | `TestNoPredictiveAPI`, `TestRefusHorsFrontiere`, `TestI3_DateRevisionRespectee`, `TestUnmodeledObservationsReported` |
| 10 | Profil de calibration reproductible byte-identique | `TestCalibrationDeterministic` |

---

## 18. Risques & mitigation

| # | Risque | Probabilité | Impact | Mitigation | Marqueur |
|---|---|---|---|---|---|
| 1 | `AgentMeshKafka` pas prêt comme validateur M4 | Probable | Élevé | Stabiliser AgentMeshKafka avant M4 ; mock intermédiaire ; M4 reportable 2-3 sem sans bloquer M5 | Probable |
| 2 | Invariants I1-I5 trop abstraits pour fuzz direct | Probable | Moyen | Reformulation exécutable §6 ; revue ciblée M3 ; raffinement après M4 | Probable |
| 3 | Drift TauGo ↔ révisions monographie | Probable | Moyen | Tag version épinglé dans `CLAUDE.md` et chaque `Profile` ; revue à chaque release monographie | Probable |
| 4 | Scope creep vers framework agentique | Probable | Élevé | §3.3 anti-objectifs ; revue mensuelle stricte ; lignes interdites en CI *(§14.4)* | Probable |
| 5 | Interface LLM fuit l'abstraction probabiliste dans `tau/*` | À vérifier | Moyen | Interface étroite §12.2 ; stub déterministe ; `arch_test.go` interdit imports concrets | À vérifier |
| 6 | `ruvector.db` impose couplage RAG prématuré | Probable | Faible | Exclu V1 §3.2 ; étude séparée | Probable |
| 7 | Verrou D-AUTORITÉ mal calibré → faux refus en cascade | Hypothèse | Moyen | Calibration empirique M4 ; `θ_auth_block` initial conservateur (0.85) ; corpus cas-limites | Hypothèse |
| 8 | Calibration sensible au modèle LLM injecté → profils non-portables | Probable | Moyen | `model_llm_fingerprint` dans profil §11.3 ; matrice de profils par modèle | Probable |
| 9 | Échéance I3 (2026-12-01) non respectée → modèle silencieusement périmé | Probable | Élevé | Garde CI `TestI3_DateRevisionRespectee` ; alerte 30 j avant péremption | Probable |
| 10 | Couplage `AgentMeshKafka` rend TauGo non-portable | Hypothèse | Faible | `bridge/agentmeshkafka/` isole ; interface `Adapter` minimale §12.1 | Hypothèse |

---

## 19. Glossaire des termes contrôlés

*Convention héritée `InteroperabiliteAgentique/CLAUDE.md` §1.1 : un concept = un terme constant ; pas de synonymie flottante.*

| Terme | Définition opérante | Renvoi |
|---|---|---|
| **τ (opérateur)** | Migration de l'instant de fixation des grandeurs d'interopérabilité (sens, autorité, support) de l'avant-interaction vers l'interaction | §4, III.8.3 |
| **Échange** *(Exchange)* | Objet soumis à τ : initiateur, capacité, intention, attestation éventuelle | §9.1 |
| **Régime** *(Regime)* | Sortie discrète de τ : `Deterministe | Probabiliste | Refus`. Jamais un comportement | §2.3 |
| **Dimension** | Axe sur lequel τ se projette : D-SENS, D-AUTORITÉ, D-INVARIANT. Orthogonales en valeur sous contrainte I4 | §5, III.8.4 |
| **Invariant** | Proposition réfutable du modèle (I1-I5). Marqueur épistémique gradué | §6, III.8.5 |
| **Frontière de validité** | 4 conditions classiques toutes violées simultanément. Hors frontière → `Refus` | §2.1, III.8.3.2 |
| **Décision** *(Decision)* | Sortie de `Decide` : `Regime`, `Trace`, `Diagnostic`, `ProfileVersion`. Toujours instrumentée | §9.1, §10 |
| **Trace** | Instrumentation immuable d'une décision | §9.1, §10.3 |
| **Profil** *(Profile)* | Calibration versionnée et opposable : seuils, poids, empreintes, `date_revision` | §11 |
| **Drift** | Désynchronisation profil ↔ environnement (matériel, modèle, corpus) déclenchant recalibration | §11.4 |
| **Attestation institutionnelle** | Référence opposable (RFC, juridiction) peuplant le pôle « exécution » de D-AUTORITÉ | §4.4, III.8.4.2.bis |
| **Sonde** *(Probe)* | Composante atomique d'un score de dimension. `[0,1]` | §5 |
| **Métrique cardinale M(π)** | Taille de l'union des angles morts d'une pile composée. Borne pour I5 | §6.1, III.8.6.3 |
| **CIA** | Calcul d'Intégration Agentique — programme de recherche dont TauGo est le livrable empirique | §1 |
| **HGL** | Héritage des Garanties de Livraison — manuscrit-compagnon (`RechercheFondamentale.md`) qui mécanise formellement ce que TauGo éprouve | §3.2 |
| **Anti-patron** | Usage qui dénature le modèle : prédictif, hors frontière, atemporel, clos | §7.2 |

---

## 20. Documents liés & prochaines étapes

### 20.1 Documents liés

- `agbruneau/InteroperabiliteAgentique` v2.4.3 (2026-05-21) — **monographie source**, chap. III.8 canonique
- `agbruneau/InteroperabiliteAgentique/RechercheFondamentale.md` — manuscrit-compagnon HGL, mécanisation Lean en dépôt à créer
- `agbruneau/AgentMeshKafka` — substrat de validation empirique
- `agbruneau/FibGo` — **référence d'ingénierie** (commit épinglé à fixer M0)
- `agbruneau/FibRust` — référence ergonomie type-safe (pertinent si extension Rust V3+)

### 20.2 Prochaines étapes Claude Code (M0)

1. Lire ce `PRD.md` intégralement et confirmer la compréhension du périmètre V1
2. Confirmer la fidélité des invariants I1-I5 reformulés §6 contre `Monographie.md` lignes ~5723-5737
3. Rédiger / réviser `CLAUDE.md` héritant des conventions FibGo *(structure)* et InteroperabiliteAgentique *(rédaction)*
4. Générer le squelette Clean Architecture *(M0 livrables, §16)*
5. Configurer la CI : `golangci-lint`, `go test -race`, fuzz court, build reproductible byte-identique sur 3 OS
6. Premier commit signé sur `main`, premier tag `v0.0.1-alpha`
7. Ouvrir issue M1 : dispatcher minimal deux régimes avec stub LLM déterministe

### 20.3 Document vivant

*Toute déviation matérielle doit être justifiée par mise à jour de ce fichier — en premier, avant le code. Toute affirmation datée porte un marqueur explicite et est revérifiée à chaque révision substantielle.*

**Prochaine revue planifiée** : clôture de M2 (trois dimensions calculables), pour ajuster sondes et pondérations sur retour empirique précoce. Date cible : *à vérifier — fonction du démarrage effectif.*

---

*Fin du PRD V0.2 (refactorisé) — 2026-05-23. V0.1 = commit précédent ; V0 = commit `b771dd1`.*
