"""Lifecycle failure and preservation tests.

Each test runs against a fresh temporary Git repository. The real repository,
its worktrees, data directories and servers are never touched.
"""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import LifecycleTest  # noqa: E402


class AcceptanceExecutionTest(LifecycleTest):
    def test_accept_succeeds_without_merging(self):
        record = self.aux_dir / "record.txt"
        result = self.work(
            "work-accept", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script(record=record)})
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.head(), self.base_sha)
        self.assertFalse((self.card_dir / "done.md").exists())
        self.assertTrue(self.wt.exists())
        self.assertEqual(record.read_text(encoding="utf-8").splitlines(),
                         [self.source_sha, self.source_sha])
        self.assertFalse((self.root / ".work-scratch" / self.card_id).exists())

    def test_failed_checks_preserve_branch_and_worktree(self):
        result = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script(exit_code=1)})
        self.assert_refused(result, "acceptance checks failed")
        self.assert_preserved()
        self.assertEqual(self.head(), self.base_sha)

    def test_moved_source_revision_is_refused(self):
        result = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script(move_source=True)})
        self.assert_refused(result, "source revision moved during acceptance")
        self.assert_preserved()
        self.assertNotEqual(
            self.git("rev-parse", "work/%s" % self.card_id), self.source_sha)

    def test_moved_target_revision_is_refused(self):
        result = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script(move_target=True)})
        self.assert_refused(result, "target revision moved during acceptance")
        self.assert_preserved()
        self.assertNotEqual(self.head(), self.base_sha)

    def test_changed_target_branch_is_refused(self):
        result = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script(
                change_target_branch=True)})
        self.assert_refused(result, "target branch moved during acceptance")
        self.assert_preserved()
        self.assertEqual(self.git("rev-parse", "--abbrev-ref", "HEAD"),
                         "other-target")

    def test_source_dirtied_during_acceptance_is_refused(self):
        result = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script(
                dirty_source=True)})
        self.assert_refused(result, "source worktree became dirty")
        self.assert_preserved()
        self.assertTrue((self.wt / "dirty-source.txt").exists())

    def test_target_dirtied_during_acceptance_is_refused(self):
        result = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script(
                dirty_target=True)})
        self.assert_refused(result, "target checkout became dirty")
        self.assert_preserved()
        self.assertTrue((self.root / "dirty-target.txt").exists())

    def test_preexisting_scratch_worktree_is_preserved(self):
        scratch = self.root / ".work-scratch" / self.card_id
        self.git("worktree", "add", "-q", "--detach", str(scratch),
                 self.base_sha)
        (scratch / "sentinel").write_text("keep\n", encoding="utf-8")
        result = self.work(
            "work-accept", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(result, "pre-existing scratch")
        self.assertTrue((scratch / "sentinel").exists())
        self.assertIn(str(scratch), self.git("worktree", "list", "--porcelain"))
        self.assertTrue(self.wt.exists())


class FinishOutcomeTest(LifecycleTest):
    def test_rejection_records_reason_and_preserves_work(self):
        (self.wt / "dirty.txt").write_text("preserve me\n", encoding="utf-8")
        result = self.work("work-finish", self.card_id,
                           "--reject", "not wanted")
        self.assertEqual(result.returncode, 0, result.stdout)
        rejected = (self.card_dir / "rejected.md").read_text(encoding="utf-8")
        self.assertIn("not wanted", rejected)
        self.assertIn("dirty.txt", rejected)
        self.assertIn("preserved", rejected)
        self.assertTrue(self.wt.exists())
        self.assertEqual((self.wt / "dirty.txt").read_text(encoding="utf-8"),
                         "preserve me\n")
        self.assertIn("work/%s" % self.card_id,
                      self.git("branch", "--list", "work/%s" % self.card_id))
        self.assertFalse((self.card_dir / "done.md").exists())
        self.assertNotEqual(self.head(), self.base_sha)
        self.assertEqual(self.git("status", "--porcelain"), "")

        retry = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assert_refused(retry, "terminal file present: rejected.md")

    def test_rejection_commit_does_not_sweep_staged_target_changes(self):
        (self.root / "staged.txt").write_text("staged outside\n",
                                              encoding="utf-8")
        self.git("add", "staged.txt")
        plan = self.card_dir / "plan.md"
        plan.write_text(plan.read_text(encoding="utf-8") +
                        "\nstaged card change\n", encoding="utf-8")
        self.git("add", str(self.card_rel / "plan.md"))
        (self.root / "README").write_text("unstaged change\n",
                                          encoding="utf-8")
        (self.wt / "dirty.txt").write_text("preserve me\n",
                                           encoding="utf-8")
        before_cached = self.git("diff", "--cached", "--binary")
        before_unstaged = self.git("diff", "--binary")

        result = self.work("work-finish", self.card_id,
                           "--reject", "not wanted")
        self.assertEqual(result.returncode, 0, result.stdout)
        committed = self.git("diff-tree", "--no-commit-id", "--name-only",
                             "-r", "HEAD")
        self.assertEqual(committed, str(self.card_rel / "rejected.md"))
        self.assertEqual(self.git("diff", "--cached", "--binary"),
                         before_cached)
        self.assertEqual(self.git("diff", "--binary"), before_unstaged)
        self.assertTrue((self.wt / "dirty.txt").exists())
        self.assertIn("not wanted",
                      (self.card_dir / "rejected.md").read_text(
                          encoding="utf-8"))

    def test_successful_finish_preserves_ignored_material(self):
        ignored = self.wt / "ignored"
        ignored.mkdir()
        payload = b"SQLite format 3\x00personal\n"
        (ignored / "personal.sqlite").write_bytes(payload)
        result = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script()})
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertTrue((self.card_dir / "done.md").exists())
        self.assertTrue(self.wt.exists())
        self.assertEqual((ignored / "personal.sqlite").read_bytes(), payload)
        self.assertIn("work/%s" % self.card_id,
                      self.git("branch", "--list", "work/%s" % self.card_id))
        self.assertIn("ignored local material", result.stdout)
        self.assertIn("preserved", result.stdout)
        self.assertIn(str(self.wt),
                      self.git("worktree", "list", "--porcelain"))

    def test_successful_finish_merges_the_tested_revision(self):
        record = self.aux_dir / "record.txt"
        result = self.work(
            "work-finish", self.card_id,
            env={"WORK_ACCEPT_CMD": self.make_accept_script(record=record)})
        self.assertEqual(result.returncode, 0, result.stdout)
        done = (self.card_dir / "done.md").read_text(encoding="utf-8")
        self.assertIn("**Tested revision:** %s" % self.source_sha, done)
        self.assertEqual(record.read_text(encoding="utf-8").splitlines(),
                         [self.source_sha, self.source_sha])
        self.assertFalse(self.wt.exists())
        self.assertEqual(
            self.git("branch", "--list", "work/%s" % self.card_id), "")
        self.assertEqual(
            self.git("merge-base", "--is-ancestor", self.source_sha, "HEAD"),
            "")
        self.assertEqual(self.git("status", "--porcelain"), "")
        self.assertIn("finished %s" % self.card_id, result.stdout)


if __name__ == "__main__":
    unittest.main()
