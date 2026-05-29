# CONVENTIONS — Audit Go multi-agents TauGo

> Contrat partagé par les 6 sous-agents d'audit (briefing autoportant injecté dans chaque dispatch).

## Langue & style
- **FR-CA** ; noms d'outils/API en anglais. **Pyramide inversée** (conclusion d'abord). Premiers principes, concision. Pas d'emoji.
- **Marqueurs épistémiques obligatoires** sur chaque constat : `[confirmé]` (preuve d'exécution/code) · `[probable]` · `[hypothèse]` · `[à vérifier]`.

## Sévérité
| Niveau | Définition |
|---|---|
| CRITIQUE | Décision non conforme III.8 · invariant violé · data race/deadlock · fallback silencieux hors frontière · non-déterminisme calibration · étanchéité rompue. |
| MAJEUR | Régression perf > 5 % · étanchéité contournée · API risquée · **statut épistémique survendu**. |
| MINEUR | Style, idiome Go, nommage. |
| INFORMATIF | Observation, dette tracée. |

## Format de constat
`**[ID] SÉVÉRITÉ — Titre** ⟨marqueur⟩` + sous-puces : Fichier:ligne / Preuve (verbatim) / Impact / Recommandation. IDs préfixés par axe (`C1-`, `I2-`, `R3-`, `P4-`, `Q5-`, `A6-`).

## Exécution (Windows / PowerShell / Go 1.26.3)
- **Lecture seule** sur code, `testdata/`, golden corpus. Aucune écriture hors `audit/`.
- Si un test régénère un artefact suivi → `git restore` et le signaler.
- Pas de `make` (absent) → `go` / `golangci-lint` directs. Pas de `-race` (CGO off) → le signaler `[à vérifier]`.
- Sorties temporaires confinées à `audit/`.

## Sortie
Objet structuré : `axis`, `headline`, `severity_counts`, `tools_run`, `tools_unavailable`, `top_findings`, `body_markdown`. L'orchestrateur écrit les fichiers `0N_*.md` et `RAPPORT_FINAL.md`.
