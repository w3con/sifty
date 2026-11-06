"""Task, status and resume navigation tests.

Views are derived from canonical card metadata; a worktree's state is shown
when it differs from the base checkout.
"""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import CardRepoTest  # noqa: E402


class NavigationTest(CardRepoTest):
    def test_list_is_derived_and_filterable(self):
        self.make_card("TEST-1", slug="bug", status="backlog", kind="bug")
        self.make_card("TEST-2", slug="active", status="in progress")
        self.make_card("TEST-3", slug="idea", status="review", kind="idea")
        done = self.make_card("TEST-4", slug="done", status="founded")
        self.add_file(done, "done.md", "# TEST-4 — done\n")
        self.commit_all()

        listing = self.work("work-list")
        self.assertEqual(listing.returncode, 0, listing.stdout)
        for fragment in ("TEST-1", "TEST-2", "TEST-3", "TEST-4",
                         "backlog", "in progress", "review", "done"):
            self.assertIn(fragment, listing.stdout)
        self.assertIn("work/ROADMAP.md", listing.stdout)

        backlog = self.work("work-list", "--state", "backlog")
        self.assertEqual(backlog.returncode, 0, backlog.stdout)
        self.assertIn("TEST-1", backlog.stdout)
        self.assertNotIn("TEST-2", backlog.stdout)

        bugs = self.work("work-list", "--kind", "bug")
        self.assertEqual(bugs.returncode, 0, bugs.stdout)
        self.assertIn("TEST-1", bugs.stdout)
        self.assertNotIn("TEST-3", bugs.stdout)

    def test_status_shows_exact_next_step_and_blocker(self):
        card = self.make_card(
            status="review",
            plan_steps=("- [x] 1.1 — done",
                        "- [ ] 1.2 — the exact next step"))
        description = (card / "description.md").read_text(encoding="utf-8")
        (card / "description.md").write_text(
            description.replace("**Status:** review\n",
                                "**Status:** review\n**Blocker:** owner decision\n"),
            encoding="utf-8")
        self.commit_all()

        result = self.work("work-status", "TEST-1")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("State: review", result.stdout)
        self.assertIn("Next open step: 1.2 — the exact next step", result.stdout)
        self.assertIn("Blocker: **Blocker:** owner decision", result.stdout)

    def test_archived_outcomes_are_visible(self):
        rejected = self.make_card("TEST-1", slug="rejected", status="founded")
        self.add_file(rejected, "rejected.md", "# TEST-1 — rejected\n")
        cancelled = self.make_card("TEST-2", slug="cancelled", status="founded")
        self.add_file(cancelled, "cancelled.md", "# TEST-2 — cancelled\n")
        self.commit_all()

        rejected_view = self.work("work-list", "--state", "rejected")
        self.assertIn("TEST-1", rejected_view.stdout)
        self.assertIn("archived", rejected_view.stdout)
        cancelled_view = self.work("work-list", "--state", "cancelled")
        self.assertIn("TEST-2", cancelled_view.stdout)
        self.assertIn("archived", cancelled_view.stdout)

    def test_worktree_state_is_shown_when_it_differs_from_base(self):
        self.make_card(status="backlog")
        self.commit_all()

        self.git("checkout", "-q", "-b", "work/TEST-1")
        description = self.root / "work/tasks/test/TEST-1_demo/description.md"
        text = description.read_text(encoding="utf-8")
        description.write_text(text.replace("**Status:** backlog",
                                            "**Status:** in progress"),
                               encoding="utf-8")
        self.commit_all("chore(test): start")
        self.git("checkout", "-q", "main")
        self.git("worktree", "add", "-q", str(self.root / ".worktrees" / "TEST-1"),
                 "work/TEST-1")

        status = self.work("work-status", "TEST-1")
        self.assertEqual(status.returncode, 0, status.stdout)
        self.assertIn("State: backlog", status.stdout)
        self.assertIn("worktree state in progress (base backlog)", status.stdout)
        self.assertIn("differs from base main", status.stdout)

        listing = self.work("work-list")
        self.assertIn("worktree state in progress (base backlog)",
                      listing.stdout)

        resumed = self.work("work-resume", "TEST-1",
                            env={"WORK_SESSION": "1"})
        self.assertEqual(resumed.returncode, 97, resumed.stdout)
        self.assertIn("State: in progress", resumed.stdout)
        self.assertIn("Next open step: 1.1 — do the thing", resumed.stdout)
        self.assertIn("Base checkout still records: backlog", resumed.stdout)

    def test_generated_views_agree_after_a_transition(self):
        self.make_card(status="backlog")
        self.commit_all()
        changed = self.work("work-status", "TEST-1", "review")
        self.assertEqual(changed.returncode, 0, changed.stdout)

        status = self.work("work-status", "TEST-1")
        listing = self.work("work-list", "--state", "review")
        self.assertIn("State: review", status.stdout)
        self.assertIn("TEST-1", listing.stdout)


if __name__ == "__main__":
    unittest.main()
