#!/usr/bin/env python3
"""typography-fr.py -- Applique la typographie francaise canonique aux fichiers Markdown.

Regles appliquees (idempotentes) :
  1. U+00A0 avant : ; ? ! >> (si precede d'un espace ordinaire)
  2. U+00A0 apres << (si suivi d'un caractere non-espace)
  3. Guillemets droits "..." -> << ... >> dans la prose narrative uniquement

Exclusions (jamais touchees) :
  - Blocs triple-backtick
  - Spans inline (`...`)
  - Liens Markdown et URLs (](...) et https?://...)
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

# Guillemets droits "..." -> guillemets francais
# Capture : " + contenu sans guillemet ni backtick + "
RE_STRAIGHT_QUOTES = re.compile(r'"([^"`\n]{1,200}?)"')


def apply_typography_to_segment(text: str) -> str:
    """Applique les regles typographiques a un segment de texte sans code inline."""
    # Regle 1 : remplace espace ordinaire avant : ; ? ! >> par NBSP.
    # On ne touche QUE les cas avec espace ordinaire (0x20) avant la ponctuation --
    # jamais les separateurs techniques word:word (pas d'espace avant).
    text = re.sub(" ([:;?!»])", NBSP + r"\1", text)

    # Regle 2 : NBSP apres << si suivi directement d'un non-espace
    text = re.sub("«(?=[^  ])", "«" + NBSP, text)

    # Regle 3 : guillemets droits -> guillemets francais (prose uniquement)
    def replace_quotes(m: re.Match) -> str:
        inner = m.group(1)
        # Ne pas transformer si contient deja des guillemets francais ou est vide
        if "«" in inner or "»" in inner:
            return m.group(0)
        return "«" + NBSP + inner + NBSP + "»"

    text = RE_STRAIGHT_QUOTES.sub(replace_quotes, text)

    return text


def process_line_outside_code(line: str) -> str:
    """Traite une ligne hors bloc de code, en preservant les spans inline et URLs."""
    # Tokeniser : alterner prose et segments-a-preserver
    # Preserve : `...`, [text](url), https?://\S+, <url>
    PRESERVE_RE = re.compile(
        r"(`[^`]*`)"                                     # inline code
        r"|(\[(?:[^\[\]]|\[[^\[\]]*\])*\]\([^)]*\))"    # liens Markdown [t](u)
        r"|(https?://\S+)"                               # URLs nues
        r"|(<https?://[^>]+>)"                           # URLs chevrons
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
    """Traite un fichier Markdown. Retourne le nombre de lignes modifiees."""
    original = path.read_text(encoding="utf-8")
    lines = original.splitlines(keepends=True)

    in_code_block = False
    new_lines = []
    changed = 0

    for line in lines:
        stripped = line.rstrip("\n\r")

        # Detection bascule triple-backtick (debut ou fin de bloc)
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
    targets = []
    for p in root.rglob("*.md"):
        rel_str = str(p.relative_to(root))
        # Exclure docs/superpowers/plans/
        if "superpowers" in rel_str and "plans" in rel_str:
            continue
        targets.append(p)
    return sorted(targets)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Applique la typographie francaise aux Markdown."
    )
    parser.add_argument("root", nargs="?", default=".", help="Racine du projet (defaut: .)")
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

    verb = "Dry-run" if args.dry_run else "Applied"
    print(f"\n{verb}: {total_files} fichiers, {total_lines} lignes modifiees.")


if __name__ == "__main__":
    main()
