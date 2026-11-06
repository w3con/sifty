"""Isolated tests for the active navigation map validator.

The real map is checked separately by bin/map-check. These tests build tiny
temporary repositories so every negative case is exercised without touching the
real checkout or runtime data.
"""
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import _map_lib as m  # noqa: E402


MAP_TEMPLATE = """# Test map

<a id="req-col"></a>
### R-COL — Collect

See [the requirement](#req-col) and [the legend](#status-legend).

## Status legend

Text.

## Canonical task index

| Card | Requirement | State | Depends on |
|---|---|---|---|
{rows}

## Legacy single-file cards

| Card | Role |
|---|---|
| [legacy](../work/tasks/legacy.md) | historical |
"""


class MapCheckTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        self.map_path = self.root / "docs" / "MAP.md"
        self.map_path.parent.mkdir(parents=True)
        self.write_card("TEST-1", "test", "demo")
        self.write(self.root / "work" / "tasks" / "legacy.md", "# old\n")
        self.write_map()

    def tearDown(self):
        self._tmp.cleanup()

    def write(self, path, text):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")

    def write_card(self, card_id, epic, slug):
        path = (self.root / "work" / "tasks" / epic /
                ("%s_%s" % (card_id, slug)) / "description.md")
        self.write(path, "# %s\n" % card_id)
        return path

    def write_map(self, extra="", rows=None):
        if rows is None:
            rows = "| [TEST-1](../work/tasks/test/TEST-1_demo/description.md) | R-COL | done | — |"
        self.write(self.map_path, MAP_TEMPLATE.format(rows=rows) + extra)

    def validate(self):
        return m.validate_map(self.root, docs=[Path("docs/MAP.md")])

    def assert_error(self, fragment):
        errors = self.validate()
        self.assertTrue(any(fragment in error for error in errors),
                        "expected %r in %r" % (fragment, errors))

    def test_valid_map_passes(self):
        self.assertEqual(self.validate(), [])

    def test_missing_local_file_fails(self):
        self.write_map(extra="\n[bad](../work/tasks/missing.md)\n")
        self.assert_error("missing local target")

    def test_missing_fragment_fails(self):
        self.write_map(extra="\n[bad](#req-missing)\n")
        self.assert_error("missing fragment #req-missing")

    def test_unknown_requirement_fails(self):
        self.write_map(extra="\nThis mentions R-UNKNOWN.\n")
        self.assert_error("unknown requirement R-UNKNOWN")

    def test_missing_canonical_id_fails(self):
        self.write_card("TEST-2", "test", "other")
        self.assert_error("canonical ID missing from task index: TEST-2")

    def test_duplicate_canonical_id_in_index_fails(self):
        rows = ("| [TEST-1](../work/tasks/test/TEST-1_demo/description.md) | R-COL | done | — |\n"
                "| [TEST-1](../work/tasks/test/TEST-1_demo/description.md) | R-COL | done | — |")
        self.write_map(rows=rows)
        self.assert_error("duplicate canonical ID in task index: TEST-1")

    def test_duplicate_canonical_id_in_repository_fails(self):
        self.write_card("TEST-1", "other", "duplicate")
        self.assert_error("duplicate canonical card ID in repository: TEST-1")

    def test_legacy_card_missing_from_map_fails(self):
        self.write(self.root / "work" / "tasks" / "another.md", "# another\n")
        self.assert_error("legacy single-file card missing from map")

    def test_external_links_are_ignored(self):
        self.write_map(extra="\n[web](https://example.com) [mail](mailto:x@example.com)\n")
        self.assertEqual(self.validate(), [])

    def test_active_document_link_fails(self):
        readme = self.root / "docs" / "README.md"
        self.write(readme, "[missing](missing.md)\n")
        errors = m.validate_documents(self.root, docs=[Path("docs/README.md")])
        self.assertTrue(any("missing local target" in error for error in errors), errors)


if __name__ == "__main__":
    unittest.main()
