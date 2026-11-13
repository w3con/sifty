"""Task-state inference and transition tests.

Every test builds an isolated temporary Git repository. The real checkout, its
worktrees, runtime data and servers are never touched.
"""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import CardRepoTest  # noqa: E402


class StateInferenceTest(CardRepoTest):
    def test_done_outcome_wins_over_founded_status_and_stale_checkbox(self):
        card = self.make_card(short=True, status="founded",
                              steps=("- [ ] stale template residue",))
        self.add_file(card, "done.md", "# TEST-1 — done\n")
        self.commit_all()

        status = self.work("work-status", "TEST-1")
        self.assertEqual(status.returncode, 0, status.stdout)
        self.assertIn("State: done (successful outcome)", status.stdout)
        self.assertIn("Next open step: none (archived outcome)", status.stdout)
        self.assertNotIn("plan still has", status.stdout)

        listing = self.work("work-list", "--state", "done")
        self.assertEqual(listing.returncode, 0, listing.stdout)
        self.assertIn("TEST-1", listing.stdout)

    def test_rejected_and_cancelled_are_archived_not_done(self):
        rejected = self.make_card("TEST-1", slug="rejected")
        self.add_file(rejected, "rejected.md", "# TEST-1 — rejected\n")
        cancelled = self.make_card("TEST-2", slug="cancelled")
        self.add_file(cancelled, "cancelled.md", "# TEST-2 — cancelled\n")
        self.commit_all()

        one = self.work("work-status", "TEST-1")
        self.assertEqual(one.returncode, 0, one.stdout)
        self.assertIn("rejected (archived unsuccessful outcome)", one.stdout)
        self.assertNotIn("successful outcome", one.stdout.replace(
            "archived unsuccessful outcome", ""))
        two = self.work("work-status", "TEST-2")
        self.assertEqual(two.returncode, 0, two.stdout)
        self.assertIn("cancelled (archived unsuccessful outcome)", two.stdout)

    def test_question_blocks_without_becoming_a_terminal_outcome(self):
        card = self.make_card(status="in progress")
        self.add_file(card, "question.md", "# Question\n\nShould this wait?\n")
        self.commit_all()

        status = self.work("work-status", "TEST-1")
        self.assertEqual(status.returncode, 0, status.stdout)
        self.assertIn("State: in progress", status.stdout)
        self.assertIn("Blocker: question.md: Should this wait?", status.stdout)

        handover = self.work("work-handover", "TEST-1",
                             env={"WORK_SESSION": "1"})
        self.assert_refused(handover, "unresolved blocker")
        self.assertFalse((self.root / ".worktrees" / "TEST-1").exists())

    def test_explicit_blocker_blocks_handover(self):
        card = self.make_card(status="review")
        description = (card / "description.md").read_text(encoding="utf-8")
        (card / "description.md").write_text(
            description.replace("**Status:** review\n",
                                "**Status:** review\n**Blocker:** waiting on owner\n"),
            encoding="utf-8")
        self.commit_all()

        status = self.work("work-status", "TEST-1")
        self.assertEqual(status.returncode, 0, status.stdout)
        self.assertIn("Blocker: **Blocker:** waiting on owner", status.stdout)
        handover = self.work("work-handover", "TEST-1",
                             env={"WORK_SESSION": "1"})
        self.assert_refused(handover, "unresolved blocker")

    def test_legacy_approved_status_is_backlog(self):
        self.make_card(status="approved; ready for handover")
        self.commit_all()
        result = self.work("work-status", "TEST-1")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("State: backlog", result.stdout)

    def test_legacy_descoped_status_is_an_archived_cancellation(self):
        self.make_card(status="descoped (see Decision)")
        self.commit_all()
        result = self.work("work-status", "TEST-1")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("cancelled (archived unsuccessful outcome)", result.stdout)

    def test_status_done_requires_done_evidence(self):
        self.make_card(status="done")
        self.commit_all()
        result = self.work("work-status", "TEST-1")
        self.assert_refused(result, "requires done.md")

    def test_multiple_outcomes_are_contradictory(self):
        card = self.make_card(status="founded")
        self.add_file(card, "done.md", "# TEST-1 — done\n")
        self.add_file(card, "rejected.md", "# TEST-1 — rejected\n")
        self.commit_all()
        result = self.work("work-status", "TEST-1")
        self.assert_refused(result, "multiple terminal outcomes")

    def test_terminal_outcome_cannot_carry_a_blocker(self):
        card = self.make_card(status="founded")
        self.add_file(card, "done.md", "# TEST-1 — done\n")
        description = (card / "description.md").read_text(encoding="utf-8")
        (card / "description.md").write_text(
            description.replace("**Status:** founded\n",
                                "**Status:** founded\n**Blocker:** late\n"),
            encoding="utf-8")
        self.commit_all()
        result = self.work("work-status", "TEST-1")
        self.assert_refused(result, "cannot carry a blocker")

    def test_duplicate_ids_are_ambiguous_and_refused(self):
        self.make_card("TEST-1", slug="one")
        self.make_card("TEST-1", slug="two")
        self.commit_all()
        result = self.work("work-status", "TEST-1")
        self.assert_refused(result, "duplicate card ID")
        listing = self.work("work-list")
        self.assert_refused(listing, "duplicate card ID")

    def test_base_card_blocker_stops_acceptance_of_clean_worktree(self):
        card = self.make_card(steps=("- [x] 1.1 — done",))
        self.commit_all()
        self.git("checkout", "-q", "-b", "work/TEST-1")
        self.git("checkout", "-q", "main")
        self.git("worktree", "add", "-q",
                 str(self.root / ".worktrees" / "TEST-1"), "work/TEST-1")
        self.add_file(card, "question.md", "# Question\n\nWait for the owner.\n")
        self.commit_all("chore(test): record base blocker")

        result = self.work("work-accept", "TEST-1")
        self.assert_refused(result, "unresolved blocker on base card")
        self.assertTrue((self.root / ".worktrees" / "TEST-1").exists())

    def test_unknown_status_is_reported_not_guessed(self):
        self.make_card(status="somewhere")
        self.commit_all()
        result = self.work("work-status", "TEST-1")
        self.assert_refused(result, "unknown **Status:** value")


class StateTransitionTest(CardRepoTest):
    def test_normal_transitions_commit_only_the_card_description(self):
        self.make_card(status="backlog")
        self.commit_all()

        result = self.work("work-status", "TEST-1", "in-progress")
        self.assertEqual(result.returncode, 0, result.stdout)
        description = (self.root / "work/tasks/test/TEST-1_demo/description.md")
        self.assertIn("**Status:** in progress", description.read_text(
            encoding="utf-8"))
        self.assertEqual(self.git("diff-tree", "--no-commit-id", "--name-only",
                                  "-r", "HEAD"),
                         "work/tasks/test/TEST-1_demo/description.md")

        result = self.work("work-status", "TEST-1", "review")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("State: review", result.stdout)

    def test_done_is_refused_and_never_written_directly(self):
        self.make_card(status="backlog")
        self.commit_all()
        result = self.work("work-status", "TEST-1", "done")
        self.assert_refused(result, "work-finish")
        self.assertFalse(
            (self.root / "work/tasks/test/TEST-1_demo/done.md").exists())

    def test_block_and_unblock(self):
        card = self.make_card(status="review")
        self.commit_all()
        blocked = self.work("work-status", "TEST-1", "--block", "waiting")
        self.assertEqual(blocked.returncode, 0, blocked.stdout)
        self.assertIn("**Blocker:** waiting",
                      (card / "description.md").read_text(encoding="utf-8"))
        self.assertIn("Blocker: **Blocker:** waiting", blocked.stdout)

        unblocked = self.work("work-status", "TEST-1", "--unblock")
        self.assertEqual(unblocked.returncode, 0, unblocked.stdout)
        self.assertNotIn("**Blocker:**",
                         (card / "description.md").read_text(encoding="utf-8"))

    def test_archived_outcome_cannot_be_changed(self):
        card = self.make_card(status="founded")
        self.add_file(card, "rejected.md", "# TEST-1 — rejected\n")
        self.commit_all()
        result = self.work("work-status", "TEST-1", "review")
        self.assert_refused(result, "archived evidence is not reopened")

    def test_unrelated_staged_change_survives_a_status_change(self):
        self.make_card(status="backlog")
        self.commit_all()
        (self.root / "staged.txt").write_text("staged outside\n",
                                              encoding="utf-8")
        self.git("add", "staged.txt")
        before = self.git("diff", "--cached", "--name-only")

        result = self.work("work-status", "TEST-1", "review")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.git("diff", "--cached", "--name-only"), before)
        self.assertEqual(self.git("diff-tree", "--no-commit-id", "--name-only",
                                  "-r", "HEAD"),
                         "work/tasks/test/TEST-1_demo/description.md")

    def test_handover_marks_worktree_in_progress_without_touching_base(self):
        self.make_card(status="backlog")
        self.commit_all()
        result = self.work("work-handover", "TEST-1",
                           env={"WORK_SESSION": "1"})
        self.assertEqual(result.returncode, 97, result.stdout)
        self.assertIn("State: in progress", result.stdout)
        self.assertIn("- Card folder: `work/tasks/test/TEST-1_demo`",
                      result.stdout)
        self.assertIn("(work/tasks/test/TEST-1_demo/plan.md)",
                      result.stdout)
        self.assertNotIn(".worktrees/TEST-1/work/tasks", result.stdout)
        worktree_card = (self.root / ".worktrees" / "TEST-1" / "work" /
                         "tasks" / "test" / "TEST-1_demo" / "description.md")
        self.assertIn("**Status:** in progress",
                      worktree_card.read_text(encoding="utf-8"))
        base_card = (self.root / "work" / "tasks" / "test" /
                     "TEST-1_demo" / "description.md")
        self.assertIn("**Status:** backlog",
                      base_card.read_text(encoding="utf-8"))

    def test_cancel_writes_archived_outcome_and_preserves_work(self):
        self.make_card(status="backlog")
        self.commit_all()
        result = self.work("work-cancel", "TEST-1", "--reason",
                           "owner changed direction")
        self.assertEqual(result.returncode, 0, result.stdout)
        cancelled = self.root / "work/tasks/test/TEST-1_demo/cancelled.md"
        self.assertTrue(cancelled.exists())
        self.assertIn("owner changed direction",
                      cancelled.read_text(encoding="utf-8"))
        status = self.work("work-status", "TEST-1")
        self.assertIn("cancelled (archived unsuccessful outcome)", status.stdout)


if __name__ == "__main__":
    unittest.main()
