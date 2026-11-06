"""Bug/idea capture tests.

Each test uses an isolated temporary Git repository. Capture must preserve the
owner's exact wording, create a minimal canonical card, and never dispatch,
cut a worktree or touch staging.
"""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import CardRepoTest  # noqa: E402


class CaptureTest(CardRepoTest):
    words = ('Owner said: "the staging server burns CPU"\n'
             'second line  keeps  spaces\n``` literal fence\nend')

    def captured_cards(self):
        base = self.root / "work" / "tasks" / "inbox"
        return sorted(base.glob("INBOX-*")) if base.exists() else []

    def test_capture_creates_minimal_card_with_verbatim_words(self):
        result = self.work("work-capture", "--kind", "bug",
                           "--title", "Staging CPU",
                           "--words", self.words,
                           env={"WORK_SESSION": "1"})
        self.assertEqual(result.returncode, 0, result.stdout)
        cards = self.captured_cards()
        self.assertEqual(len(cards), 1, result.stdout)
        card = cards[0]
        description = (card / "description.md").read_text(encoding="utf-8")
        self.assertIn(self.words, description)
        self.assertIn("**Status:** backlog", description)
        self.assertIn("**Kind:** bug", description)
        self.assertFalse((card / "plan.md").exists())
        self.assertFalse((card / "log.md").exists())
        self.assertFalse((card / "attention.md").exists())
        self.assertTrue((card / "comments.md").exists())
        self.assertFalse((self.root / ".worktrees" / "INBOX-1").exists())
        self.assertNotIn("Executor brief", result.stdout)
        committed = self.git("diff-tree", "--no-commit-id", "--name-only",
                             "-r", "HEAD").splitlines()
        self.assertEqual(sorted(committed), sorted([
            "work/tasks/inbox/INBOX-1_staging-cpu/comments.md",
            "work/tasks/inbox/INBOX-1_staging-cpu/description.md",
        ]))

    def test_exact_duplicate_words_are_refused(self):
        first = self.work("work-capture", "--kind", "bug",
                          "--words", self.words)
        self.assertEqual(first.returncode, 0, first.stdout)
        second = self.work("work-capture", "--kind", "idea",
                           "--words", self.words)
        self.assert_refused(second, "already captured")
        self.assertEqual(len(self.captured_cards()), 1)

    def test_update_appends_exact_evidence_to_matched_card(self):
        card = self.make_card("TEST-1", status="backlog", kind="bug")
        self.commit_all()
        evidence = "Additional detail:\nkeeps  spacing and \"quotes\"."
        result = self.work("work-capture", "--update", "TEST-1",
                           "--kind", "bug", "--words", evidence)
        self.assertEqual(result.returncode, 0, result.stdout)
        comments = (card / "comments.md").read_text(encoding="utf-8")
        self.assertIn(evidence, comments)
        self.assertEqual(self.captured_cards(), [])
        duplicate = self.work("work-capture", "--update", "TEST-1",
                              "--kind", "bug", "--words", evidence)
        self.assert_refused(duplicate, "already recorded")

    def test_staged_change_is_not_swept_into_capture_commit(self):
        (self.root / "staged.txt").write_text("staged outside\n",
                                              encoding="utf-8")
        self.git("add", "staged.txt")
        before = self.git("diff", "--cached", "--name-only")
        result = self.work("work-capture", "--kind", "idea",
                           "--words", "A separate idea")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.git("diff", "--cached", "--name-only"), before)
        committed = self.git("diff-tree", "--no-commit-id", "--name-only",
                             "-r", "HEAD").splitlines()
        self.assertTrue(all(path.startswith("work/tasks/inbox/")
                            for path in committed), committed)

    def test_capture_does_not_touch_staging_or_create_a_worktree(self):
        sentinel = self.write("data-staging/sentinel.txt", "keep me\n")
        result = self.work("work-capture", "--kind", "bug",
                           "--words", "crash on startup",
                           env={"WORK_SESSION": "1"})
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(sentinel.read_text(encoding="utf-8"), "keep me\n")
        self.assertEqual(self.git("worktree", "list", "--porcelain").count(
            "worktree "), 1)
        self.assertFalse((self.root / ".worktrees").exists())


if __name__ == "__main__":
    unittest.main()
