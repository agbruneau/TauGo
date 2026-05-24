# M2 — Décisions de référence (10 échantillons)

> Généré le 2026-05-23 avec `tau decide` + dispatcher M2 (`v0.0.3-alpha`).
> Profil : M2-default (seuils PRD §11.1 initiaux, poids PRD §5.1-5.3).
> LLM backend : `stub:v0` (FNV-1a déterministe).
> Statut : Hypothèse — scores non calibrés (pré-M4).

---

## Seuils actifs (DefaultThresholds)

| Seuil | Valeur | Rôle |
|---|---|---|
| `Deterministe` | 0.35 | τ_score < 0.35 → Deterministe |
| `Probabiliste` | 0.65 | τ_score >= 0.65 → Probabiliste |
| `AuthBlock` | 0.85 | D-AUTH >= 0.85 ∧ Attestation == nil → Refus I3 |
| `InvCoherence` | 0.50 | D-INV >= 0.50 ∧ D-SENS < SensCoherence → Refus I4 |
| `SensCoherence` | 0.50 | Seuil D-SENS dans la garde I4 |

Zone hysterèse (M2) : [0.35, 0.65) → Deterministe par défaut (historique de régime différé à M5).

---

## Tableau synthèse

| # | ID | D-SENS | D-AUTH | D-INV | τ_score | Régime | Garde déclenchée | Marqueur |
|---|---|---|---|---|---|---|---|---|
| 1 | f01 | — | — | — | — | **Refus** | Hors frontière τ | Confirmé |
| 2 | f02 | 0.239 | 0.563 | 0.250 | 0.339 | **Deterministe** | — | Confirmé |
| 3 | f03 | 0.975 | 1.000 | 1.000 | 0.990 | **Probabiliste** | — | Confirmé |
| 4 | f04 | 0.867 | 1.000 | 0.450 | — | **Refus** | I3 D-AUTH | Confirmé |
| 5 | f05 | 0.239 | 0.563 | 0.800 | — | **Refus** | I4 incohérence | Confirmé |
| 6 | f06 | 0.883 | 1.000 | 0.450 | 0.788 | **Probabiliste** | — (attestation) | Confirmé |
| 7 | f07 | 0.629 | 0.563 | 0.250 | 0.495 | **Deterministe** | — (zone hysterèse) | Probable |
| 8 | f08 | 0.914 | 1.000 | 1.000 | 0.966 | **Probabiliste** | — | Confirmé |
| 9 | f09 | 0.940 | 0.563 | 0.450 | 0.680 | **Probabiliste** | — | Probable |
| 10 | f10 | 0.881 | 0.563 | 1.000 | 0.821 | **Probabiliste** | — (I4 cohérent) | Confirmé |

*D-AUTH = D-AUTORITÉ. τ_score = 0.4·D-SENS + 0.3·D-AUTH + 0.3·D-INV. « — » dans τ_score : décision prise avant le calcul composite (garde préemptive).*

---

## Décision f01 — Refus hors frontière τ

**Objectif** : exercer le refus de frontière (étape 1 du dispatcher). Échange entièrement statique et ancré humain — aucune des quatre conditions d'agentic boundary n'est violée ; τ ne s'applique pas.

**Fixture** :
```json
{
  "id": "f01",
  "intent_description": "",
  "initiator": {"id": "user-1", "human_in_loop": true, "organization": "org-a", "delegation_depth": 0},
  "target": {"id": "svc-a", "discovery_mode": 0, "contract_uri": "https://api.internal/v1"}
}
```

**FrontierCheck** : `UniversOuvert=false, CompositionVariable=false, PairProbabiliste=false, CoutNonBorne=false` → `Inside()=false`.

**Décision CLI** :
```json
{"regime":3,"diagnostic":"hors frontière τ","profile_version":"","trace":{"exchange_id":"f01","tau_score":0,"frontier":{"univers_ouvert":false,"composition_variable":false,"pair_probabiliste":false,"cout_non_borne":false},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Confirmé — frontière déterminée par `Inside()` qui est une règle booléenne, pas une pondération.

---

## Décision f02 — Déterministe (frontière franchie, scores bas)

**Objectif** : exercer le régime Deterministe dans un échange à faible τ_score. Capacité découverte dynamiquement (DynamicMCP, frontière franchie) mais sans contrat, sans intention, sans contexte enrichi. D-AUTH modéré mais sous AuthBlock.

**Fixture** :
```json
{
  "id": "f02",
  "intent_description": "",
  "initiator": {"id": "agent-1", "human_in_loop": false, "organization": "org-a", "delegation_depth": 1},
  "target": {"id": "svc-b", "discovery_mode": 1, "contract_uri": "https://api.internal/v1"},
  "context": {}
}
```

**FrontierCheck** : `UniversOuvert=true, CompositionVariable=true, PairProbabiliste=true, CoutNonBorne=true` → `Inside()=true`.

**Sondes D-SENS** :

| Sonde | Valeur | Explication |
|---|---|---|
| `S_contract` | 0.000 | ContractURI présent → pôle 0 |
| `S_runtime_resolve` | 0.000 | IntentDescription vide |
| `S_capability_discovery` | 1.000 | DynamicMCP |
| `S_reasoner_intent` | 0.261 | FNV-1a("") = 2166136261 % 1000 / 1000 |
| **D-SENS** | **0.239** | 0.35·0 + 0.30·0 + 0.20·1 + 0.15·0.261 |

**Sondes D-AUTORITÉ** :

| Sonde | Valeur | Explication |
|---|---|---|
| `A_chain_depth` | 0.250 | DelegationDepth=1 → 1/4 |
| `A_cross_org` | 0.000 | org-a, depth<=1 → intra-org |
| `A_human_anchor` | 1.000 | HumanInLoop=false → inversé |
| `A_dynamic_resolution` | 1.000 | DynamicMCP |
| **D-AUTH** | **0.563** | 0.25·(0.25+0+1+1) |

**Sondes D-INVARIANT** :

| Sonde | Valeur | Explication |
|---|---|---|
| `I_event_registry` | 0.000 | Clé absente du contexte |
| `I_idempotency_derived` | 0.000 | Clé absente |
| `I_capability_mediation` | 1.000 | DynamicMCP → médiation dynamique |
| `I_enumerated_plan` | 0.000 | ContractURI présent → plan figé |
| **D-INV** | **0.250** | 0.30·0 + 0.25·0 + 0.25·1 + 0.20·0 |

τ_score = 0.4·0.239 + 0.3·0.563 + 0.3·0.250 = 0.096 + 0.169 + 0.075 = **0.339** < 0.35 → **Deterministe**.

**Décision CLI** :
```json
{"regime":1,"profile_version":"M2-default","trace":{"exchange_id":"f02","tau_score":0.33941,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Confirmé — τ_score = 0.339, clairement sous le seuil Deterministe de 0.35.

---

## Décision f03 — Probabiliste pur (tous les scores maximaux)

**Objectif** : exercer le régime Probabiliste à score maximal. Échange entièrement dynamique, inter-org, chaîne profonde, sans contrat, contexte pleinement enrichi, attestation présente.

**Fixture** :
```json
{
  "id": "f03",
  "intent_description": "discover and invoke optimal integration tool",
  "initiator": {"id": "sub-agent-5", "human_in_loop": false, "organization": "", "delegation_depth": 5},
  "target": {"id": "", "discovery_mode": 1, "contract_uri": ""},
  "context": {"event_registry": true, "idempotency_key_mode": "derived", "capability_mediation": true},
  "attestation_institutionnelle": {"emetteur": "IETF", "reference": "draft-delegation-00", "marqueur": "Hypothèse", "asserted_at": "2026-05-23T00:00:00Z"}
}
```

**Sondes D-SENS** : S_contract=1.000, S_runtime=1.000, S_discov=1.000, S_reasoner=0.833 → **D-SENS=0.975**.

**Sondes D-AUTORITÉ** : A_chain=1.000 (depth=5>=4), A_cross=1.000 (org=""), A_human=1.000, A_dynamic=1.000 → **D-AUTH=1.000**. Attestation présente → garde I3 ne se déclenche pas.

**Sondes D-INVARIANT** : I_event=1.000, I_idem=1.000, I_med=1.000, I_enum=1.000 (pas de contrat, pas de plan) → **D-INV=1.000**.

τ_score = 0.4·0.975 + 0.3·1.000 + 0.3·1.000 = **0.990** >= 0.65 → **Probabiliste**.

**Décision CLI** :
```json
{"regime":2,"profile_version":"M2-default","trace":{"exchange_id":"f03","tau_score":0.9899800000000001,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Confirmé — tous les indicateurs TauGo au pôle 1, score composite très élevé.

---

## Décision f04 — Refus I3 (verrou ontologique D-AUTORITÉ, sans attestation)

**Objectif** : exercer la garde ontologique D-AUTORITÉ (étape 2 du dispatcher). Chaîne longue (depth=5), inter-org (org=""), sans humain, DynamicA2A → D-AUTH=1.000 >= AuthBlock=0.85, et aucune attestation.

**Fixture** :
```json
{
  "id": "f04",
  "intent_description": "invoke external financial system without oversight",
  "initiator": {"id": "sub-agent-3", "human_in_loop": false, "organization": "", "delegation_depth": 5},
  "target": {"id": "external-fin", "discovery_mode": 2, "contract_uri": ""}
}
```

**Sondes D-AUTORITÉ** : A_chain=1.000, A_cross=1.000, A_human=1.000, A_dynamic=1.000 → **D-AUTH=1.000 >= 0.85**.

`Attestation == nil` → garde I3 déclenche **Refus(« I3 — verrou ontologique D-AUTORITÉ »)** avant le calcul de D-SENS et D-INVARIANT.

**Décision CLI** :
```json
{"regime":3,"diagnostic":"I3 — verrou ontologique D-AUTORITÉ","profile_version":"","trace":{"exchange_id":"f04","tau_score":0,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Confirmé — D-AUTH=1.000, absence d'attestation, garde I3 déterministe par construction.

---

## Décision f05 — Refus I4 (incohérence D-INVARIANT élevé / D-SENS bas)

**Objectif** : exercer la garde I4 (étape 5 du dispatcher). D-AUTH modéré (0.563, sous AuthBlock) mais D-INV élevé (0.800 >= InvCoherence=0.50) et D-SENS bas (0.239 < SensCoherence=0.50) : combinaison incohérente (invariants tracés à l'exécution mais sens figé à la conception).

**Fixture** :
```json
{
  "id": "f05",
  "intent_description": "",
  "initiator": {"id": "agent-batch", "human_in_loop": false, "organization": "org-a", "delegation_depth": 1},
  "target": {"id": "batch-proc", "discovery_mode": 1, "contract_uri": "https://api.internal/batch"},
  "context": {"event_registry": true, "idempotency_key_mode": "derived"}
}
```

**D-SENS = 0.239** (ContractURI présent, pas d'intention) < 0.50.
**D-INV = 0.800** (event_registry=1, idempotency=1, mediation=1 via DynamicMCP, enumerated=0 car ContractURI présent) >= 0.50.

Garde I4 : D-INV(0.800) >= InvCoherence(0.50) ∧ D-SENS(0.239) < SensCoherence(0.50) → **Refus(« I4 — combinaison incohérente détectée »)**.

**Décision CLI** :
```json
{"regime":3,"diagnostic":"I4 — combinaison incohérente détectée","profile_version":"","trace":{"exchange_id":"f05","tau_score":0,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Confirmé — D-SENS=0.239 clairement sous 0.50, D-INV=0.800 clairement sur 0.50.

---

## Décision f06 — Attestation présente, I3 levé → Probabiliste

**Objectif** : vérifier que la garde I3 ne se déclenche pas lorsqu'une attestation institutionnelle est fournie, même avec D-AUTH=1.000. Même échange que f04 avec attestation IETF ajoutée.

**Fixture** :
```json
{
  "id": "f06",
  "intent_description": "invoke external financial system with proper attestation",
  "initiator": {"id": "sub-agent-3", "human_in_loop": false, "organization": "", "delegation_depth": 5},
  "target": {"id": "external-fin", "discovery_mode": 2, "contract_uri": ""},
  "attestation_institutionnelle": {"emetteur": "IETF", "reference": "draft-delegation-00", "marqueur": "Hypothèse", "asserted_at": "2026-05-23T00:00:00Z"}
}
```

**D-AUTH=1.000**, mais `Attestation != nil` → garde I3 ne se déclenche pas.

D-SENS=0.883 (no contract=1, intent=1, dynamic=1, S_reasoner=0.220), D-INV=0.450 (mediation=1, enum=1, event=0, idem=0).

Garde I4 : D-INV(0.450) < InvCoherence(0.50) → garde I4 ne se déclenche pas.

τ_score = 0.4·0.883 + 0.3·1.000 + 0.3·0.450 = **0.788** >= 0.65 → **Probabiliste**.

**Décision CLI** :
```json
{"regime":2,"profile_version":"M2-default","trace":{"exchange_id":"f06","tau_score":0.7882,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Confirmé — l'attestation lève le verrou I3 par construction (condition booléenne).

---

## Décision f07 — Zone hysterèse → Deterministe

**Objectif** : exercer la zone hysterèse [0.35, 0.65). τ_score = 0.495, entre les deux seuils. En M2, le régime par défaut dans cette zone est Deterministe (historique de régime différé à M5).

**Fixture** :
```json
{
  "id": "f07",
  "intent_description": "call payment service",
  "initiator": {"id": "agent-mid", "human_in_loop": false, "organization": "org-a", "delegation_depth": 1},
  "target": {"id": "pay-svc", "discovery_mode": 1, "contract_uri": "https://pay.api/v2"},
  "context": {}
}
```

**D-SENS=0.629** : S_contract=0 (ContractURI présent), S_runtime=1 (intent non vide), S_discov=1 (DynamicMCP), S_reasoner=0.860 (FNV-1a de « call payment service »).

**D-AUTH=0.563** : A_chain=0.25 (depth=1), A_cross=0 (org-a, depth<=1), A_human=1 (no human), A_dynamic=1 (DynamicMCP).

**D-INV=0.250** : I_event=0, I_idem=0, I_med=1 (DynamicMCP), I_enum=0 (ContractURI présent).

τ_score = 0.4·0.629 + 0.3·0.563 + 0.3·0.250 = **0.495** ∈ [0.35, 0.65) → **Deterministe** (hysterèse M2).

**Décision CLI** :
```json
{"regime":1,"profile_version":"M2-default","trace":{"exchange_id":"f07","tau_score":0.49535,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Probable — la valeur dans la zone hysterèse est sensible aux pondérations initiales ; un régime différent est plausible selon le calibrage M4.

---

## Décision f08 — Probabiliste extrême (DynamicAGNTCY, depth=4, contexte maximal, attestation)

**Objectif** : exercer le mode DynamicAGNTCY (discovery_mode=3) et la saturation de A_chain_depth à 1.0 pour depth=4. Cas candidat pour la certification d'un registry externe (AGNTCY).

**Fixture** :
```json
{
  "id": "f08",
  "intent_description": "discover and bind autonomous financial agent via AGNTCY registry",
  "initiator": {"id": "orchestrator-7", "human_in_loop": false, "organization": "", "delegation_depth": 4},
  "target": {"id": "", "discovery_mode": 3, "contract_uri": ""},
  "context": {"event_registry": true, "idempotency_key_mode": "derived", "capability_mediation": true},
  "attestation_institutionnelle": {"emetteur": "AGNTCY", "reference": "agt-registry-v1", "marqueur": "Hypothèse", "asserted_at": "2026-05-23T00:00:00Z"}
}
```

**D-SENS=0.914** : S_contract=1, S_runtime=1, S_discov=1, S_reasoner=0.429.

**D-AUTH=1.000** : A_chain=1.000 (depth=4 → saturation), A_cross=1.000 (org=""), A_human=1.000, A_dynamic=1.000. Attestation AGNTCY présente → I3 ne se déclenche pas.

**D-INV=1.000** : I_event=1, I_idem=1, I_med=1 (DynamicAGNTCY), I_enum=1.

τ_score = 0.4·0.914 + 0.3·1.000 + 0.3·1.000 = **0.966** >= 0.65 → **Probabiliste**.

**Décision CLI** :
```json
{"regime":2,"profile_version":"M2-default","trace":{"exchange_id":"f08","tau_score":0.96574,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Confirmé — la saturation à depth=4 est un invariant de l'implémentation M2.

---

## Décision f09 — Probabiliste borderline (D-AUTH marginal, sans attestation)

**Objectif** : exercer un échange où D-AUTH (0.563) est bien sous AuthBlock (0.85) sans attestation. D-SENS élevé (0.940) dépasse le seuil Probabiliste. Cas marginal sur la garde I4 : D-INV=0.450 < InvCoherence=0.50, donc I4 ne se déclenche pas.

**Fixture** :
```json
{
  "id": "f09",
  "intent_description": "list available tools and invoke best match",
  "initiator": {"id": "agent-mcp", "human_in_loop": false, "organization": "org-a", "delegation_depth": 1},
  "target": {"id": "", "discovery_mode": 1, "contract_uri": ""},
  "context": {}
}
```

**D-SENS=0.940** : S_contract=1 (pas de ContractURI), S_runtime=1 (intent non vide), S_discov=1 (DynamicMCP), S_reasoner=0.599.

**D-AUTH=0.563** : A_chain=0.25, A_cross=0 (org-a, depth<=1), A_human=1, A_dynamic=1.

**D-INV=0.450** : I_event=0, I_idem=0, I_med=1 (DynamicMCP), I_enum=1 (pas de ContractURI, pas de plan explicite).

Garde I4 : D-INV(0.450) < InvCoherence(0.50) → garde I4 ne se déclenche pas.

τ_score = 0.4·0.940 + 0.3·0.563 + 0.3·0.450 = **0.680** >= 0.65 → **Probabiliste**.

**Décision CLI** :
```json
{"regime":2,"profile_version":"M2-default","trace":{"exchange_id":"f09","tau_score":0.6796899999999999,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Probable — τ_score = 0.680, peu au-dessus du seuil Probabiliste ; une révision des poids D-SENS pourrait le faire basculer.

---

## Décision f10 — I4 cohérent (D-INV élevé ET D-SENS élevé) → Probabiliste

**Objectif** : exercer le cas I4 cohérent — D-INV élevé (1.000) mais D-SENS également élevé (0.881 >= SensCoherence=0.50). La garde I4 ne se déclenche pas car la contrainte de cohérence dirigée est satisfaite : le support d'invariant est bien tracé à l'exécution et le sens est aussi résolu à l'exécution.

**Fixture** :
```json
{
  "id": "f10",
  "intent_description": "dynamically discover and orchestrate tools via MCP",
  "initiator": {"id": "agent-coherent", "human_in_loop": false, "organization": "org-a", "delegation_depth": 1},
  "target": {"id": "", "discovery_mode": 1, "contract_uri": ""},
  "context": {"event_registry": true, "idempotency_key_mode": "derived"}
}
```

**D-SENS=0.881** : S_contract=1 (pas de ContractURI), S_runtime=1 (intent non vide), S_discov=1 (DynamicMCP), S_reasoner=0.209 (FNV-1a de « dynamically discover and orchestrate tools via MCP »).

**D-AUTH=0.563** : A_chain=0.25, A_cross=0 (org-a, depth<=1), A_human=1, A_dynamic=1. Sous AuthBlock=0.85 → garde I3 ne se déclenche pas.

**D-INV=1.000** : I_event=1 (event_registry=true), I_idem=1 (idempotency_key_mode=derived), I_med=1 (DynamicMCP), I_enum=1 (pas de ContractURI).

Garde I4 : D-INV(1.000) >= InvCoherence(0.50) ∧ D-SENS(0.881) >= SensCoherence(0.50) → **I4 cohérent, garde ne se déclenche pas**.

τ_score = 0.4·0.881 + 0.3·0.563 + 0.3·1.000 = **0.821** >= 0.65 → **Probabiliste**.

**Décision CLI** :
```json
{"regime":2,"profile_version":"M2-default","trace":{"exchange_id":"f10","tau_score":0.8212899999999999,"frontier":{"univers_ouvert":true,"composition_variable":true,"pair_probabiliste":true,"cout_non_borne":true},"thresholds":{"deterministe":0.35,"probabiliste":0.65,"auth_block":0.85,"sens_coherence":0.5,"inv_coherence":0.5},"duration_ns":1}}
```

**Marqueur** : Confirmé — D-SENS=0.881 clairement au-dessus de 0.50, I4 ne peut pas se déclencher.

---

## Observations et questions ouvertes

1. **Surprise f02 borderline** : τ_score = 0.339, juste sous le seuil Deterministe (0.35). Une variation infime des pondérations (p. ex. S_reasoner_intent = 0.27 au lieu de 0.261) pourrait le faire passer dans la zone hysterèse. Marquer comme cas de test de régression prioritaire. (Hypothèse)

2. **Zone hysterèse f07** : Dans la zone [0.35, 0.65), le régime M2 est Deterministe par défaut. L'historique de régime (qui change la réponse dans la zone) est différé à M5. La valeur 0.495 est bien centrée dans la zone — un bon candidat pour tester l'hysterèse M5. (Probable)

3. **S_reasoner_intent stub** : Le stub FNV-1a produit des scores entre 0.000 et 0.999 selon l'intent. Les valeurs observées (0.115 à 0.860) montrent une dispersion satisfaisante. Une vérification sur des intents réels en M4 est nécessaire pour calibrer le poids 0.15 de S_reasoner. (Hypothèse)

4. **Coupure D-AUTH à depth=4** : La saturation `DelegationDepth >= 4 → A_chain_depth = 1.0` est une heuristique non calibrée. f08 (depth=4) et f03/f04/f06 (depth=5) donnent le même A_chain_depth=1.0. (Hypothèse)

5. **Garde I3 préemptive** : La garde I3 s'exécute avant le calcul de D-SENS et D-INVARIANT (étape 2 avant étape 4). Le `tau_score` est donc 0 dans la sortie JSON pour f04 — ce n'est pas un bug mais un artefact de l'ordre d'exécution du dispatcher. (Confirmé)

---

*Renvoi PRD : §5 (dimensions), §4.4 (asymétrie ontologique), §6.1 (I4), §10 (pseudo-algo), §11 (seuils). Plan M2 : `docs/superpowers/plans/2026-05-23-M2-dimensions-gardes.md`. Theorie : `docs/theory/04-dimensions.md`.*

**Version binaire** : `v0.0.3-alpha` (build go1.24+, stub:v0).
**Daté** : 2026-05-23.
