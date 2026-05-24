#!/usr/bin/env python3
"""typography-fr.py — Applique la typographie française canonique aux fichiers Markdown.

Règles appliquées (idempotentes) :
  1. U+00A0 avant : ; ? ! » (si précédé d'un caractère non-espace/non-NBSP)
  2. U+00A0 après « (si suivi d'un caractère non-espace/non-NBSP)
  3. Guillemets droits "…" → « … » dans la prose narrative uniquement

Exclusions (jamais touchées) :
  - Blocs triple-backtick (```…```)
  - Spans inline (`…`)
  - Liens Markdown et URLs (](...) et https?://…)
  - Fichiers non-Markdown (.go, .json, .yml, etc.)
  - docs/superpowers/plans/*.md (immuables)

Usage:
    python scripts/typography-fr.py [--dry-run] [racine]
"""

import re
import sys
import pathlib
import argparse

NBSP = " "

# Patterns pour les ponctuation haute
# Ajoute NBSP avant : ; ? ! » si précédé d'un non-espace (inclut NBSP déjà présent → idempotent)
RE_NBSP_BEFORE = re.compile(r"(?<=[^\s])([:;?!»])")

# Ajoute NBSP après « si suivi d'un non-espace
RE_NBSP_AFTER_GUILLEMET_OPEN = re.compile(r"«(?=[^\s])")

# Guillemets droits "…" → « … » — seulement si les deux guillemets sont sur la même ligne
# Capture : " + contenu sans guillemet ni backtick + "
RE_STRAIGHT_QUOTES = re.compile(r'"([^"`\n]{1,200}?)"')


def apply_typography_to_segment(text: str) -> str:
    """Applique les règles typographiques à un segment de texte sans code inline."""
    # Règle 1 : NBSP avant ponctuation haute
    # Remplace espace normal (ou espace déjà précédé de rien) avant ponct par NBSP
    # D'abord : espace ordinaire + ponct → NBSP + ponct
    text = re.sub(r" ([:;?!»])", NBSP + r"\1", text)
    # Ensuite : ponct directement après un non-espace (sans espace du tout) → ajouter NBSP
    # Ex: "mot:" → "mot :" avec NBSP. On évite de toucher :// (URLs) et :: (Go)
    # Ne touche pas les :: consécutifs, ni ://
    text = re.sub(r"(?<=[^\s :])(:)(?![:/ ])", NBSP + r"\1", text)
    text = re.sub(r"(?<=[^\s ])(;)(?![ ])", NBSP + r"\1", text)
    text = re.sub(r"(?<=[^\s ])(\?)(?![ ])", NBSP + r"\1", text)
    text = re.sub(r"(?<=[^\s ])(!)(?![ ])", NBSP + r"\1", text)
    text = re.sub(r"(?<=[^\s ])(»)(?![ ])", NBSP + r"\1", text)

    # Règle 2 : NBSP après «
    text = re.sub(r"«(?=[^\s ])", "«" + NBSP, text)

    # Règle 3 : guillemets droits → guillemets français
    # Seulement si ce n'est pas déjà des guillemets français
    def replace_quotes(m: re.Match) -> str:
        inner = m.group(1)
        # Éviter de transformer si l'inner contient déjà des guillemets français
        if "«" in inner or "»" in inner:
            return m.group(0)
        return f"«{NBSP}{inner}{NBSP}»"

    text = RE_STRAIGHT_QUOTES.sub(replace_quotes, text)

    return text


def process_line_outside_code(line: str) -> str:
    """Traite une ligne hors bloc de code, en préservant les spans inline et URLs."""
    # Découper la ligne en segments : spans inline (`…`) et reste
    # Stratégie : extraire les backtick-spans et URLs, les préserver, traiter le reste

    # Tokeniser : alterner prose et spans-à-préserver
    # Préserve : `…`, [text](url), https?://\S+, <url>
    PRESERVE_RE = re.compile(
        r"(`[^`]*`)"              # inline code
        r"|(\[(?:[^\[\]]|\[[^\[\]]*\])*\]\([^)]*\))"  # liens Markdown [t](u)
        r"|(https?://\S+)"        # URLs nues
        r"|(<https?://[^>]+>)"    # URLs chevrons
    )

    parts = []
    last = 0
    for m in PRESERVE_RE.finditer(line):
        start, end = m.span()
        if start > last:
            parts.append(("prose", line[last:start]))
        parts.append(("preserve", m.group(0)))
        last = end
    if last < len(line):
        parts.append(("prose", line[last:]))

    result = []
    for kind, text in parts:
        if kind == "prose":
            result.append(apply_typography_to_segment(text))
        else:
            result.append(text)
    return "".join(result)


def process_file(path: pathlib.Path, dry_run: bool = False) -> int:
    """Traite un fichier Markdown. Retourne le nombre de lignes modifiées."""
    original = path.read_text(encoding="utf-8")
    lines = original.splitlines(keepends=True)

    in_code_block = False
    new_lines = []
    changed = 0

    for line in lines:
        stripped = line.rstrip("\n\r")

        # Détection bascule triple-backtick (début ou fin de bloc)
        if stripped.startswith("```"):
            in_code_block = not in_code_block
            new_lines.append(line)
            continue

        if in_code_block:
            new_lines.append(line)
            continue

        # Traitement hors bloc de code
        eol = line[len(stripped):]
        new_stripped = process_line_outside_code(stripped)
        new_line = new_stripped + eol

        if new_line != line:
            changed += 1
        new_lines.append(new_line)

    new_content = "".join(new_lines)
    if new_content != original:
        if not dry_run:
            path.write_text(new_content, encoding="utf-8")
        return changed
    return 0


def collect_targets(root: pathlib.Path) -> list[pathlib.Path]:
    """Collecte tous les fichiers Markdown cibles."""
    EXCLUDE_DIRS = {"docs/superpowers/plans", "docs\\superpowers\\plans"}
    targets = []
    for p in root.rglob("*.md"):
        rel = p.relative_to(root)
        rel_str = str(rel)
        # Exclure docs/superpowers/plans/
        if "superpowers" in rel_str and "plans" in rel_str:
            continue
        targets.append(p)
    return sorted(targets)


def main() -> None:
    parser = argparse.ArgumentParser(description="Applique la typographie française aux Markdown.")
    parser.add_argument("root", nargs="?", default=".", help="Racine du projet (défaut: .)")
    parser.add_argument("--dry-run", action="store_true", help="Affiche sans modifier")
    args = parser.parse_args()

    root = pathlib.Path(args.root).resolve()
    targets = collect_targets(root)

    total_files = 0
    total_lines = 0

    for path in targets:
        changed = process_file(path, dry_run=args.dry_run)
        if changed:
            total_files += 1
            total_lines += changed
            prefix = "[DRY] " if args.dry_run else "      "
            print(f"{prefix}{path.relative_to(root)}  ({changed} lignes)")

    print(f"\n{'Dry-run' if args.dry_run else 'Applied'}: {total_files} fichiers, {total_lines} lignes modifiées.")


if __name__ == "__main__":
    main()
