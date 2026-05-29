# `docs/archive/` — documents historiques

Documents préservés pour traçabilité — n'évoluent plus. Source canonique vivante : voir [`PRD.md`](../../PRD.md), [`CLAUDE.md`](../../CLAUDE.md), [`README.md`](../../README.md), [`CHANGELOG.md`](../../CHANGELOG.md), [`docs/adr/`](../adr/).

## Plans d'exécution

- [`PRDPlanning-m0-m6.md`](PRDPlanning-m0-m6.md) — plan-cadre M0-M6 v0.1.0 (clos 2026-05-24).
- [`plans-m0-m6/`](plans-m0-m6/) — six sous-plans détaillés bite-sized (M1 à M6), un fichier par milestone.

## Audits

- [`audits/2026-05-24-AUDIT-v0.1.0-to-v0.1.1.md`](audits/2026-05-24-AUDIT-v0.1.0-to-v0.1.1.md) — audit consolidé v0.1.0 → v0.1.1 (2026-05-24, commit base `5a68c12`).
- [`audits/2026-05-24-AUDITPlan-v0.1.1.md`](audits/2026-05-24-AUDITPlan-v0.1.1.md) — plan d'exécution refactor v0.1.1 (42 tâches T-001..T-040).
- [`audits/2026-05-29-AUDIT-v0.1.2-pre/`](audits/2026-05-29-AUDIT-v0.1.2-pre/) — audit de régression v0.1.2-pre (2026-05-29 ; rapport principal [`RAPPORT_FINAL.md`](audits/2026-05-29-AUDIT-v0.1.2-pre/RAPPORT_FINAL.md), 6 axes `01`..`06` + `00_bootstrap` + `CONVENTIONS`). Verdict : 0 critique, 10 majeur, 16 mineur, 15 informatif ; kernel sain, fragilités épistémiques/documentaires corrigées dans le lot d'alignement post-audit.

Les recommandations issues de l'audit et le plan ont été intégralement exécutés dans le commit `2cf560c` *(2026-05-24, branche `main`)*. Détail : [`CHANGELOG.md`](../../CHANGELOG.md) §v0.1.1-pre.
