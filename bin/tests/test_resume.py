"""work-resume card resolution across a path migration.

The base checkout may still hold the card at a pre-relocation path while the
work branch already holds it at the new one. Resume must read the worktree copy
by ID and print the new path, not silently report every step as closed.
"""
import os
import shutil
import sys
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import BIN, LifecycleTest  # noqa: E402


class ResumeMigrationTest(LifecycleTest):
    def setUp(self):
        super().setUp()
        # The temp base repo needs the harness templates that work-resume renders.
        shutil.copytree(BIN / "templates", self.root / "bin" / "templates")

    def test_resume_reads_card_from_worktree_during_path_migration(self):
        # Open a step on the work branch.
        plan = self.wt / self.card_rel / "plan.md"
        plan.write_text(
            "# %s — plan\n\n## Steps\n\n- [ ] 1.1 — do the thing\n\n"
            "## Acceptance criteria\n\n- The thing works.\n" % self.card_id,
            encoding="utf-8")
        self.commit_in_source("reopen a step on the work branch")

        # Simulate the pre-merge state: the base checkout still has the old
        # card path, while the work branch has the new one.
        legacy_rel = Path("ai", "tasks", "test", "%s_demo" % self.card_id)
        (self.root / legacy_rel).parent.mkdir(parents=True, exist_ok=True)
        self.git("mv", str(self.card_rel), str(legacy_rel))
        self.git("commit", "-qm", "simulate pre-migration base path")

        result = self.work("work-resume", self.card_id,
                           env={"WORK_SESSION": "1"})
        self.assertEqual(result.returncode, 97, result.stdout)
        self.assertIn("work/tasks/test/%s_demo" % self.card_id, result.stdout)
        self.assertIn("- [ ] 1.1 — do the thing", result.stdout)


if __name__ == "__main__":
    unittest.main()
