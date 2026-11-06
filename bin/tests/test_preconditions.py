"""Acceptance/finish precondition tests.

Each test runs against a fresh temporary Git repository. The real repository,
its worktrees, data directories and servers are never touched.
"""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import LifecycleTest  # noqa: E402


class AcceptancePreconditionsTest(LifecycleTest):
    def test_accept_refuses_wrong_target_branch(self):
        self.git("checkout", "-q", "-b", "feature")
        result = self.work("work-accept", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "target branch is 'feature', not 'main'")
        self.assertTrue(self.wt.exists())
        self.assertFalse((self.card_dir / "done.md").exists())

    def test_finish_refuses_wrong_target_branch(self):
        self.git("checkout", "-q", "-b", "feature")
        result = self.work("work-finish", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "target branch is 'feature', not 'main'")
        self.assert_preserved()

    def test_accept_refuses_open_steps(self):
        self.reopen_source_plan()
        result = self.work("work-accept", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "open step")
        self.assertFalse((self.card_dir / "done.md").exists())

    def test_finish_refuses_open_steps(self):
        self.reopen_source_plan()
        result = self.work("work-finish", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "open step")
        self.assert_preserved()

    def test_finish_refuses_open_attention(self):
        attention = self.wt / self.card_rel / "attention.md"
        attention.write_text(
            "# %s — attention\n\n- [ ] waiting for owner\n" % self.card_id,
            encoding="utf-8")
        self.commit_in_source("open attention")
        result = self.work("work-finish", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "unresolved attention")
        self.assert_preserved()

    def test_finish_refuses_dirty_source_tracked(self):
        (self.wt / "README").write_text("changed\n", encoding="utf-8")
        result = self.work("work-finish", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "source worktree is dirty")
        self.assert_preserved()

    def test_finish_refuses_dirty_source_untracked(self):
        (self.wt / "untracked-source.txt").write_text("x\n", encoding="utf-8")
        result = self.work("work-finish", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "source worktree is dirty")
        self.assert_preserved()

    def test_finish_refuses_dirty_target_tracked(self):
        (self.root / "README").write_text("changed\n", encoding="utf-8")
        result = self.work("work-finish", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "target checkout is dirty")
        self.assert_preserved()

    def test_finish_refuses_dirty_target_untracked(self):
        (self.root / "untracked-target.txt").write_text("x\n", encoding="utf-8")
        result = self.work("work-finish", self.card_id,
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "target checkout is dirty")
        self.assert_preserved()

    def test_force_does_not_bypass_open_steps(self):
        self.reopen_source_plan()
        result = self.work("work-finish", self.card_id, "--force",
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "open step")
        self.assert_preserved()

    def test_force_does_not_bypass_dirty_source(self):
        (self.wt / "untracked-source.txt").write_text("x\n", encoding="utf-8")
        result = self.work("work-finish", self.card_id, "--force",
                           env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "source worktree is dirty")
        self.assert_preserved()

    def test_force_skips_only_the_acceptance_command(self):
        result = self.work(
            "work-finish", self.card_id, "--force",
            env={"WORK_ACCEPT_CMD": self.make_accept_script(exit_code=1)})
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertTrue((self.card_dir / "done.md").exists())
        self.assertFalse(self.wt.exists())

    def reopen_source_plan(self):
        plan = self.wt / self.card_rel / "plan.md"
        plan.write_text(
            "# %s — plan\n\n## Steps\n\n- [ ] 1.1 — do the thing\n\n"
            "## Acceptance criteria\n\n- The thing works.\n" % self.card_id,
            encoding="utf-8")
        self.commit_in_source("reopen plan step")
