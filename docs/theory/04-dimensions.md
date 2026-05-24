# 04 — Les trois dimensions — renvoi vers chap. III.8.4

*Document de renvoi croisé. Le verbatim canonique vit dans `InteroperabiliteAgentique/Monographie.md` v2.4.3, chap. III.8.4.*

*Statut global : Hypothèse — pondérations initiales, à corroborer sur traces AgentMeshKafka M4. Daté 2026-05-23.*

---

## Vue synoptique (III.8.4)

| Dimension | Pôle 0 *(avant)* | Pôle 1 *(pendant)* | Nature |
|---|---|---|---|
| **D-SENS** | Contrat figé, publié, opposable | Capacité découverte, interprétée à l'exécution | Fait protocolaire |
| **D-AUTORITÉ** | Chaîne courte, intra-domaine, humain ancré | Chaîne longue, inter-org, sans humain | Fait institutionnel (Searle 1995) |
| **D-INVARIANT** | Support énuméré à la conception | Support tracé / négocié / observé pendant | Fait protocolaire |

τ applicable : D-SENS et D-INVARIANT — oui, coûteux. D-AUTORITÉ — conditionné à institution externe (§4.4).

---

## D-SENS — lieu de fixation du sens (III.8.4.1)

**Question opérante** : *le pair décide-t-il d'invoquer à partir d'une interprétation produite à l'exécution, ou d'un câblage produit à la conception ?*

### Pôles

- **Pôle 0 (avant)** : contrat opposable publié (`ContractURI` non vide), invocation sans description d'intention, capacité statique connue à la conception.
- **Pôle 1 (pendant)** : capacité découverte à l'exécution, interprétation sémantique de l'intention par un raisonneur probabiliste, résolution du contrat au moment de l'échange.

### Sondes et poids initiaux (Hypothèse)

| Sonde | Indicateur TauGo | Poids | Encodage Go |
|---|---|---|---|
| `S_contract` | `Target.ContractURI == ""` → 1.0 | 0.35 | `probeContract()` dans `dsens.go` |
| `S_runtime_resolve` | `IntentDescription != ""` → 1.0 | 0.30 | `probeRuntimeResolve()` |
| `S_capability_discovery` | `Target.DiscoveryMode != Static` → 1.0 | 0.20 | `probeCapabilityDiscovery()` |
| `S_reasoner_intent` | Score LLM via `Client.Interpret()` | 0.15 | `probeReasonerIntent()` |

`D_SENS(x) = 0.35·S_contract + 0.30·S_runtime_resolve + 0.20·S_capability_discovery + 0.15·S_reasoner_intent`

**Fichier** : `internal/tau/dimensions/dsens.go`

---

## D-AUTORITÉ — portée de la chaîne de délégation (III.8.4.2)

**Question opérante** : *la chaîne est-elle longue, dynamique, inter-organisationnelle, sans humain ancré ?*

### Pôles

- **Pôle 0 (avant)** : humain directement à l'origine (`HumanInLoop=true`, `DelegationDepth=0`), intra-organisation, capacité cible connue et statique.
- **Pôle 1 (pendant)** : chaîne de délégation profonde (`DelegationDepth >= 4`), croisement de frontière organisationnelle, sans humain dans la boucle, capacité cible résolue dynamiquement.

### Asymétrie ontologique (III.8.4.2.bis)

D-AUTORITÉ est un **fait institutionnel** au sens de Searle (1995) : l'autorité d'un agent sur un autre ne découle pas d'un fait brut mais d'un acte de reconnaissance institutionnelle. Déplacer la fixation d'autorité de la conception vers l'exécution (`τ_AUTORITÉ`) exige l'existence d'une institution externe émettrice d'une reconnaissance opposable.

Sans `Attestation` opposable fournie dans l'échange, un score `D_AUTORITÉ(x) >= θ_auth_block` (défaut : 0.85) déclenche un **Refus ontologique**, non un régime `Probabiliste`. Cette asymétrie distingue `τ_AUTORITÉ` de `τ_SENS` et de `τ_INVARIANT` qui sont des faits protocolaires accessibles par coût.

Renvoi PRD : §4.4 (Asymétrie ontologique D-AUTORITÉ). Référence : Searle, J.R. (1995), *The Construction of Social Reality*.

### Sondes et poids initiaux (Hypothèse)

| Sonde | Indicateur TauGo | Poids | Encodage Go |
|---|---|---|---|
| `A_chain_depth` | Profondeur normalisée : `DelegationDepth / 4.0` (saturation à 1.0 pour `>= 4`) | 0.25 | `probeChainDepth()` |
| `A_cross_org` | `Organization == ""` ou `DelegationDepth > 1` → 1.0 | 0.25 | `probeCrossOrg()` |
| `A_human_anchor` | Inversé : `HumanInLoop == false` → 1.0 | 0.25 | `probeHumanAnchor()` |
| `A_dynamic_resolution` | `Target.DiscoveryMode != Static` → 1.0 | 0.25 | `probeDynamicResolution()` |

`D_AUTORITÉ(x) = 0.25·A_chain_depth + 0.25·A_cross_org + 0.25·A_human_anchor + 0.25·A_dynamic_resolution`

**Garde** (étape 2 du dispatcher) : `D_AUTORITÉ(x) >= 0.85 ∧ x.Attestation == nil ⇒ Refus("I3 — verrou ontologique D-AUTORITÉ")`.

**Fichier** : `internal/tau/dimensions/dauthority.go`

---

## D-INVARIANT — support des invariants d'intégration (III.8.4.3)

**Question opérante** : *le support repose-t-il sur un artefact figé avant l'interaction, ou tracé / négocié / observé pendant ?*

### Pôles

- **Pôle 0 (avant)** : plan d'étapes énuméré à la conception, clé d'idempotence imposée statiquement, contrat de médiation de capacité prédéfini.
- **Pôle 1 (pendant)** : registre d'effets tracé à l'exécution, clé d'idempotence dérivée de l'intention, médiation de capacité négociée pendant l'échange.

### Contrainte de cohérence I4 (III.8.4.5)

`i ≈ pendant ⟹ s ≈ pendant`. Direction dissymétrique : D-SENS contraint D-INVARIANT, pas l'inverse. Une combinaison D-INVARIANT élevé / D-SENS bas est ontologiquement incohérente — le support d'invariant ne peut pas être négocié à l'exécution si le sens lui-même est figé à la conception. Cette combinaison déclenche un refus (étape 5 du dispatcher) : `D_INVARIANT(x) >= θ_inv ∧ D_SENS(x) < θ_sens ⇒ Refus("I4 — combinaison incohérente détectée")`.

Renvoi PRD : §6.1 (I4, cohérence dirigée).

### Sondes et poids initiaux (Hypothèse)

| Sonde | Indicateur TauGo | Poids | Encodage Go |
|---|---|---|---|
| `I_event_registry` | `Context["event_registry"] == true` → 1.0 | 0.30 | `probeEventRegistry()` |
| `I_idempotency_derived` | `Context["idempotency_key_mode"] == "derived"` → 1.0 | 0.25 | `probeIdempotencyDerived()` |
| `I_capability_mediation` | `Context["capability_mediation"] == true` ou `DiscoveryMode != Static` | 0.25 | `probeCapabilityMediation()` |
| `I_enumerated_plan` | Inversé : `ContractURI == ""` et pas de plan explicite → 1.0 | 0.20 | `probeEnumeratedPlan()` |

`D_INVARIANT(x) = 0.30·I_event_registry + 0.25·I_idempotency_derived + 0.25·I_capability_mediation + 0.20·I_enumerated_plan`

**Fichier** : `internal/tau/dimensions/dinvariant.go`

---

## Score composite τ

```
τ_score = 0.4 · D_SENS(x) + 0.3 · D_AUTORITÉ(x) + 0.3 · D_INVARIANT(x)
```

Poids initiaux PRD §11.1 : `(0.4, 0.3, 0.3)`. Statut : Hypothèse — à corroborer sur traces AgentMeshKafka M4.

---

## Encodage Go des types d'entrée

Les scores dépendent des types `Principal` et `Capability` ajoutés à `Exchange` en M2.1 :

```go
type Principal struct {
    ID              string
    HumanInLoop     bool          // false => PairProbabiliste, A_human_anchor = 1
    Organization    string        // "" => A_cross_org = 1
    DelegationDepth int           // 0 = humain direct ; >= 4 => A_chain_depth = 1
}

type Capability struct {
    ID            string
    DiscoveryMode DiscoveryMode   // Static | DynamicMCP | DynamicA2A | DynamicAGNTCY
    ContractURI   string          // "" => S_contract = 1 (pas de contrat)
}
```

---

## Marqueurs d'incertitude

| Élément | Marqueur | Justification |
|---|---|---|
| Définition des trois dimensions (III.8.4) | Confirmé | Verbatim monographie v2.4.3 |
| Asymétrie ontologique D-AUTORITÉ (III.8.4.2.bis) | Confirmé | Verbatim monographie v2.4.3 ; Searle 1995 |
| Contrainte de cohérence I4 (III.8.4.5) | Confirmé | Verbatim monographie v2.4.3 |
| Poids D-SENS {0.35, 0.30, 0.20, 0.15} | Hypothèse | Valeurs initiales PRD §5.1 — non calibrées |
| Poids D-AUTORITÉ {0.25, 0.25, 0.25, 0.25} | Hypothèse | Valeurs initiales PRD §5.2 — symétrie postulée |
| Poids D-INVARIANT {0.30, 0.25, 0.25, 0.20} | Hypothèse | Valeurs initiales PRD §5.3 — non calibrées |
| Poids composites {0.4, 0.3, 0.3} | Hypothèse | Valeurs initiales PRD §11.1 — non calibrées |
| Seuil `θ_auth_block = 0.85` | Hypothèse | Valeur initiale PRD §11.1 — non calibrée |
| Seuils I4 `θ_inv = 0.50`, `θ_sens = 0.50` | Hypothèse | Valeurs initiales PRD §11.1 — non calibrées |
| Heuristique `DelegationDepth / 4.0` | Hypothèse | Placeholder M2 — révision attendue M4 |
| Encodage Go (M2) | Probable | Implémenté, compilé, testé — non empiriquement validé |

---

## Questions ouvertes (Hypothèse, 2026-05-23)

1. Les pondérations initiales {0.35, 0.30, 0.20, 0.15} pour D-SENS sont-elles robustes sur des traces AgentMeshKafka réelles ? *Réponse attendue : M4.*
2. L'heuristique `DelegationDepth >= 4 => A_chain_depth = 1.0` est-elle bien calibrée ? *À réviser avec données empiriques M4.*
3. Les clés de contexte (`event_registry`, `idempotency_key_mode`, `capability_mediation`) sont-elles portées par les messages AgentMeshKafka réels ? *À vérifier lors de l'intégration M4.*
4. La pondération égale (0.25 × 4) pour D-AUTORITÉ reflète-t-elle l'égale importance des quatre facteurs ? *Hypothèse de symétrie à tester M4.*

---

*Renvoi PRD : §5 (dimensions), §4.4 (asymétrie ontologique), §6.1 (I4). Plan M2 : `docs/archive/plans-m0-m6/2026-05-23-M2-dimensions-gardes.md`.*

**Aligné monographie** : v2.4.3 (2026-05-21).
**Daté** : 2026-05-23.
