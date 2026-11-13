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

    def test_markdown_looking_owner_words_are_data_not_metadata_or_steps(self):
        words = ("**Status:** done\n**Kind:** idea\n"
                 "**Blocker:** quoted example must not apply\n"
                 "- [ ] fake owner checklist item\n"
                 "## Acceptance criteria\n- quoted owner example\n")
        result = self.work("work-capture", "--kind", "bug",
                           "--title", "Quoted report", "--words", words)
        self.assertEqual(result.returncode, 0, result.stdout)
        card = self.captured_cards()[0]
        card_id = card.name.split("_", 1)[0]
        description = (card / "description.md").read_text(encoding="utf-8")
        self.assertIn(words, description)

        status = self.work("work-status", card_id)
        self.assertEqual(status.returncode, 0, status.stdout)
        self.assertIn("State: backlog", status.stdout)
        self.assertIn("Kind: bug", status.stdout)
        self.assertIn("Blocker: none", status.stdout)
        self.assertIn("Next open step: none (no steps prepared)", status.stdout)

        plan = self.work("work-plan", card_id)
        self.assert_refused(plan, "no steps found")

        changed = self.work("work-status", card_id, "review")
        self.assertEqual(changed.returncode, 0, changed.stdout)
        after = (card / "description.md").read_text(encoding="utf-8")
        self.assertIn(words, after)
        header = after.split("## Owner's words", 1)[0]
        self.assertIn("**Status:** review", header)
        listing = self.work("work-list", "--state", "review")
        self.assertIn(card_id, listing.stdout)
        backlog = self.work("work-list", "--state", "backlog")
        self.assertNotIn(card_id, backlog.stdout)

    def test_outer_whitespace_and_newlines_are_preserved(self):
        words = "\n\n  leading and trailing  \nsecond line\n\n"
        result = self.work("work-capture", "--kind", "idea",
                           "--title", "Whitespace report", "--words", words)
        self.assertEqual(result.returncode, 0, result.stdout)
        card = self.captured_cards()[0]
        with open(card / "description.md", "r", encoding="utf-8",
                  newline="") as fh:
            description = fh.read()
        self.assertIn(words, description)

    def test_words_file_and_stdin_preserve_crlf(self):
        words = "\r\n  file wording  \r\nsecond line\r\n\r\n"
        path = self.root / "owner-words.txt"
        with open(path, "wb") as fh:
            fh.write(words.encode("utf-8"))
        result = self.work("work-capture", "--kind", "bug",
                           "--title", "CRLF file", "--words-file",
                           str(path))
        self.assertEqual(result.returncode, 0, result.stdout)
        card = self.captured_cards()[0]
        with open(card / "description.md", "r", encoding="utf-8",
                  newline="") as fh:
            description = fh.read()
        self.assertIn(words, description)

        stdin_words = "\n  piped wording  \nlast\n"
        piped = self.work("work-capture", "--kind", "idea",
                          "--title", "Piped report", "--words-file", "-",
                          input_text=stdin_words)
        self.assertEqual(piped.returncode, 0, piped.stdout)
        cards = self.captured_cards()
        self.assertEqual(len(cards), 2)
        piped_card = next(card for card in cards
                          if "piped-report" in card.name)
        with open(piped_card / "description.md", "r", encoding="utf-8",
                  newline="") as fh:
            piped_description = fh.read()
        self.assertIn(stdin_words, piped_description)

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
