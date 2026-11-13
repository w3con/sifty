"""Focused tests for work-note's log serialisation.

The old append left a blank line at EOF, so recording a check made
`git diff --check` fail on the log that had just been written.
"""
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import BIN, run_cmd  # noqa: E402


class WorkNoteTest(unittest.TestCase):
    card_id = "NOTE-1"

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        self.card_rel = Path("work", "tasks", "test", "%s_demo" % self.card_id)
        self.card_dir = self.root / self.card_rel
        self.card_dir.mkdir(parents=True)
        (self.card_dir / "plan.md").write_text(
            "# NOTE-1 — plan\n\n## Steps\n\n- [x] 1.1 — done\n\n"
            "## Acceptance criteria\n\n- Works.\n", encoding="utf-8")
        self.log = self.card_dir / "log.md"
        self.log.write_text("# NOTE-1 — log\n", encoding="utf-8")
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Work Test")
        self.git("config", "user.email", "work-test@example.com")
        self.git("add", "-A")
        self.git("commit", "-qm", "card")

    def tearDown(self):
        self._tmp.cleanup()

    def git(self, *args):
        result = run_cmd(["git", *args], cwd=self.root)
        self.assertEqual(result.returncode, 0, result.stdout)
        return result.stdout.strip()

    def note(self, *args):
        return run_cmd(
            [sys.executable, str(BIN / "work-note"), self.card_id, *args],
            cwd=self.root)

    def assert_single_trailing_newline(self, text):
        self.assertTrue(text.endswith("\n"))
        self.assertFalse(text.endswith("\n\n"))

    def test_repeated_notes_keep_single_newline_and_diff_check_clean(self):
        first = self.note("--step", "first check", "--check",
                          "printf 'alpha\\nbeta\\n'")
        self.assertEqual(first.returncode, 0, first.stdout)
        text = self.log.read_text(encoding="utf-8")
        self.assert_single_trailing_newline(text)
        self.assertIn("alpha\nbeta", text)
        self.assertEqual(
            run_cmd(["git", "diff", "--check"], cwd=self.root).returncode, 0)

        second = self.note("--message", "second note")
        self.assertEqual(second.returncode, 0, second.stdout)
        text = self.log.read_text(encoding="utf-8")
        self.assert_single_trailing_newline(text)
        self.assertEqual(text.count("## "), 2)
        self.assertIn("alpha\nbeta", text)
        self.assertIn("second note", text)
        self.assertEqual(
            run_cmd(["git", "diff", "--check"], cwd=self.root).returncode, 0)

    def test_check_output_is_preserved_verbatim(self):
        result = self.note("--check", "printf 'keep  two  spaces\\nsecond\\n'")
        self.assertEqual(result.returncode, 0, result.stdout)
        text = self.log.read_text(encoding="utf-8")
        self.assert_single_trailing_newline(text)
        self.assertIn("keep  two  spaces", text)
        self.assertIn("second", text)

    def test_failing_check_is_recorded_and_propagated(self):
        result = self.note("--step", "failing", "--check", "exit 7")
        self.assertEqual(result.returncode, 1, result.stdout)
        text = self.log.read_text(encoding="utf-8")
        self.assert_single_trailing_newline(text)
        self.assertIn("Exit: 7", text)


if __name__ == "__main__":
    unittest.main()
