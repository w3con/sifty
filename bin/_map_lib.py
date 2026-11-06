"""Shared standard-library checks for the active navigation documents.

The map is the entry point for requirement-to-code navigation. This module
validates what a reader can click: local files, explicit/heading anchors, the
requirement identifiers used by the map, and the canonical card index. It is
deliberately limited to the active navigation documents; archived prose keeps
its historical paths and is not rewritten.
"""
import re
import sys
from pathlib import Path
from urllib.parse import unquote, urlsplit

REPO_ROOT = Path(__file__).resolve().parents[1]

# Active navigation is the current map plus the documents a session reads on
# resume. Archived product notes and historical card logs are outside this list.
ACTIVE_DOCS = (
    "docs/MAP.md",
    "docs/README.md",
    "work/WORKFLOW.md",
    "work/ROADMAP.md",
)

MAP_REL = "docs/MAP.md"
TASK_INDEX_HEADING = "Canonical task index"
LEGACY_HEADING = "Legacy single-file cards"

_LINK_RE = re.compile(r"(?<!!)\[[^\]]*\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
_ANCHOR_RE = re.compile(r"<a\s+(?:id|name)=\"([^\"]+)\"", re.IGNORECASE)
_HEADING_RE = re.compile(r"^#{1,6}\s+(.+?)\s*$", re.MULTILINE)
_REQUIREMENT_RE = re.compile(r"^###\s+(R-[A-Z0-9-]+)\b", re.MULTILINE)
_CARD_DIR_RE = re.compile(r"^([A-Z][A-Z0-9]*-\d+)_([a-z0-9-]+)$")
_REQUIREMENT_TOKEN_RE = re.compile(r"\bR-[A-Z][A-Z0-9-]*\b")


def _github_slug(text):
    """Approximate the heading anchor GitHub generates."""
    text = re.sub(r"<[^>]+>", "", text)
    text = text.strip().lower()
    text = re.sub(r"[^\w\s-]", "", text)
    text = re.sub(r"\s+", "-", text)
    return text.strip("-")


def anchors(path):
    """Return the explicit and heading anchors in a Markdown file."""
    text = path.read_text(encoding="utf-8")
    found = set(_ANCHOR_RE.findall(text))
    counts = {}
    for heading in _HEADING_RE.findall(text):
        slug = _github_slug(heading)
        if not slug:
            continue
        # GitHub disambiguates repeated headings with -1, -2, ...
        n = counts.get(slug, 0)
        counts[slug] = n + 1
        found.add(slug if n == 0 else "%s-%d" % (slug, n))
    return found


def local_links(text):
    """Yield inline Markdown link targets from text."""
    for target in _LINK_RE.findall(text):
        if target.startswith("<") and target.endswith(">"):
            target = target[1:-1]
        yield target


def _is_external(target):
    parsed = urlsplit(target)
    return bool(parsed.scheme) and parsed.scheme not in ("", "file")


def _resolve(doc, target, root):
    """Resolve a local link target against its document."""
    path_text, _, fragment = target.partition("#")
    path_text = unquote(path_text)
    fragment = unquote(fragment)
    if not path_text:
        resolved = doc
    elif path_text.startswith("/"):
        resolved = root / path_text.lstrip("/")
    else:
        resolved = doc.parent / path_text
    return resolved.resolve(), fragment


def validate_documents(root, docs=None):
    """Validate local links and Markdown fragments in the active documents."""
    root = Path(root).resolve()
    docs = [Path(p) for p in (docs if docs is not None else ACTIVE_DOCS)]
    errors = []
    anchor_cache = {}
    for rel in docs:
        doc = root / rel
        if not doc.is_file():
            errors.append("%s: missing document" % rel)
            continue
        for target in local_links(doc.read_text(encoding="utf-8")):
            if not target or _is_external(target):
                continue
            resolved, fragment = _resolve(doc, target, root)
            try:
                shown = resolved.relative_to(root)
            except ValueError:
                shown = resolved
            if not resolved.exists():
                errors.append("%s: missing local target %s" % (rel, target))
                continue
            if fragment and resolved.suffix.lower() == ".md":
                if resolved not in anchor_cache:
                    anchor_cache[resolved] = anchors(resolved)
                if fragment not in anchor_cache[resolved]:
                    errors.append("%s: missing fragment #%s in %s"
                                  % (rel, fragment, shown))
    return errors


def discover_cards(root):
    """Return canonical card folders, duplicate IDs, malformed folders and legacy files."""
    root = Path(root).resolve()
    tasks = root / "work" / "tasks"
    cards = {}
    duplicates = []
    malformed = []
    legacy = set()
    if not tasks.is_dir():
        return cards, duplicates, malformed, legacy

    card_files = {
        "description.md", "plan.md", "log.md", "comments.md", "attention.md",
        "done.md", "rejected.md", "cancelled.md",
    }
    for epic in sorted(p for p in tasks.iterdir() if p.is_dir()):
        for card in sorted(p for p in epic.iterdir() if p.is_dir()):
            if not any((card / name).is_file() for name in card_files):
                continue  # a folder without any card file is not a card
            match = _CARD_DIR_RE.match(card.name)
            if not match:
                malformed.append(str(card.relative_to(root)))
                continue
            card_id = match.group(1)
            if card_id in cards:
                duplicates.append(card_id)
            else:
                cards[card_id] = card

    for path in sorted(tasks.iterdir()):
        if path.is_file() and path.suffix.lower() == ".md":
            legacy.add(str(path.relative_to(root)))
    return cards, duplicates, malformed, legacy


def _section(text, heading):
    """Return a level-2 Markdown section body, or None when absent."""
    match = re.search(r"^##\s+%s\s*$" % re.escape(heading), text, re.MULTILINE)
    if not match:
        return None
    body = text[match.end():]
    next_match = re.search(r"^##\s+", body, re.MULTILINE)
    return body[:next_match.start()] if next_match else body


def _table_rows(section):
    """Yield cell lists for the first Markdown table rows in a section."""
    in_table = False
    for line in section.splitlines():
        stripped = line.strip()
        if not stripped.startswith("|"):
            if in_table:
                break
            continue
        in_table = True
        cells = [cell.strip() for cell in stripped.strip("|").split("|")]
        if all(re.fullmatch(r":?-{3,}:?", cell) for cell in cells):
            continue
        if cells and cells[0].lower() in ("id", "card", ""):
            continue
        yield cells


def _card_id_from_link(doc, target, root, cards):
    resolved, _ = _resolve(doc, target, root)
    for card_id, card in cards.items():
        if resolved == card or card in resolved.parents:
            return card_id
    return None


def validate_map(root, docs=None):
    """Validate the current map, active documents and canonical task index."""
    root = Path(root).resolve()
    errors = validate_documents(root, docs)
    map_path = root / MAP_REL
    if not map_path.is_file():
        errors.append("%s: missing map" % MAP_REL)
        return errors

    text = map_path.read_text(encoding="utf-8")
    requirements = set(_REQUIREMENT_RE.findall(text))
    if not requirements:
        errors.append("%s: no requirement headings found" % MAP_REL)
    for token in sorted(set(_REQUIREMENT_TOKEN_RE.findall(text))):
        if token not in requirements:
            errors.append("%s: unknown requirement %s" % (MAP_REL, token))

    cards, duplicates, malformed, legacy = discover_cards(root)
    for card_id in sorted(set(duplicates)):
        errors.append("duplicate canonical card ID in repository: %s" % card_id)
    for path in malformed:
        errors.append("card folder does not carry a canonical ID: %s" % path)

    index = _section(text, TASK_INDEX_HEADING)
    if index is None:
        errors.append("%s: missing section '%s'" % (MAP_REL, TASK_INDEX_HEADING))
    else:
        indexed = []
        for cells in _table_rows(index):
            card_id = None
            for target in local_links(" | ".join(cells)):
                if _is_external(target):
                    continue
                card_id = _card_id_from_link(map_path, target, root, cards)
                if card_id:
                    break
            if card_id is None:
                errors.append("%s: task-index row does not link a canonical card: %s"
                              % (MAP_REL, " | ".join(cells)))
                continue
            if card_id in indexed:
                errors.append("%s: duplicate canonical ID in task index: %s"
                              % (MAP_REL, card_id))
            indexed.append(card_id)
        for card_id in sorted(set(cards) - set(indexed)):
            errors.append("%s: canonical ID missing from task index: %s"
                          % (MAP_REL, card_id))
        for card_id in sorted(set(indexed) - set(cards)):
            errors.append("%s: task index names unknown canonical ID: %s"
                          % (MAP_REL, card_id))

    legacy_section = _section(text, LEGACY_HEADING)
    if legacy_section is None:
        errors.append("%s: missing section '%s'" % (MAP_REL, LEGACY_HEADING))
    else:
        listed = set()
        for target in local_links(legacy_section):
            if _is_external(target):
                continue
            resolved, _ = _resolve(map_path, target, root)
            try:
                listed.add(str(resolved.relative_to(root)))
            except ValueError:
                errors.append("%s: legacy card link escapes the repository: %s"
                              % (MAP_REL, target))
        for rel in sorted(legacy - listed):
            errors.append("%s: legacy single-file card missing from map: %s"
                          % (MAP_REL, rel))
        for rel in sorted(listed - legacy):
            errors.append("%s: legacy table names a non-card file: %s"
                          % (MAP_REL, rel))

    return errors


def main(argv=None):
    import argparse
    parser = argparse.ArgumentParser(
        prog="map-check",
        description="Validate active navigation links, fragments, requirements "
                    "and the canonical task index.")
    parser.add_argument("--root", default=str(REPO_ROOT),
                        help="repository root (default: this checkout)")
    args = parser.parse_args(argv)
    root = Path(args.root).resolve()
    errors = validate_map(root)
    if errors:
        for error in errors:
            print("FAIL %s" % error, file=sys.stderr)
        print("FAILED: %d map check(s)" % len(errors), file=sys.stderr)
        return 1
    cards, _, _, legacy = discover_cards(root)
    print("OK: %d active document(s), %d requirement(s), %d canonical card(s), "
          "%d legacy card(s)" % (len(ACTIVE_DOCS),
                                 len(set(_REQUIREMENT_RE.findall(
                                     (root / MAP_REL).read_text(encoding="utf-8")))),
                                 len(cards), len(legacy)))
    return 0


if __name__ == "__main__":
    sys.exit(main())
