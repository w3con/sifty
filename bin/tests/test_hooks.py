"""Hook installation and behavior tests.

Each test builds a throwaway "fresh public clone": the project hooks, harness commands and leak
scanner are copied into a temporary Git repository with no ai/shared submodule and no kb/
directory. The real checkout, its Git configuration and its worktrees are never touched.

The tests install the hooks through the real bin/work-setup, then make actual commits, so they
prove the effective relative hooksPath and the adapters rather than only that files exist.
"""
import os
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import REPO_ROOT, run_cmd  # noqa: E402

TOKEN = "ghp_" + "a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q7R8"


class HookInstallTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        self.copy_fixture()
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Hook Test")
        self.git("config", "user.email", "hook-test@example.com")
        self.write("README.md", "base\n")
        self.git("add", "-A")
        self.git("commit", "-qm", "chore(test): base")

    def tearDown(self):
        self._tmp.cleanup()

    # ── fixture ──────────────────────────────────────────────────────────────
    def copy_fixture(self):
        shutil.copytree(REPO_ROOT / ".githooks", self.root / ".githooks")
        shutil.copytree(
            REPO_ROOT / "sifty" / "scripts" / "leakcheck",
            self.root / "sifty" / "scripts" / "leakcheck",
            ignore=shutil.ignore_patterns("__pycache__", "tests"))
        (self.root / "bin").mkdir()
        for path in (REPO_ROOT / "bin").glob("work-*"):
            if path.is_file():
                shutil.copy2(path, self.root / "bin" / path.name)
        shutil.copy2(REPO_ROOT / "bin" / "_work_lib.py",
                     self.root / "bin" / "_work_lib.py")

    def write(self, relative, text, cwd=None):
        path = (cwd or self.root) / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")

    def git(self, *args, cwd=None, check=True):
        result = run_cmd(["git", *args], cwd=cwd or self.root)
        if check and result.returncode != 0:
            self.fail("git %s failed with exit %d"
                      % (" ".join(args), result.returncode))
        return result

    def install(self):
        result = run_cmd(["bash", str(self.root / "bin" / "work-setup")],
                         cwd=self.root)
        self.assertEqual(result.returncode, 0, "work-setup failed")
        return result

    def commit(self, message, cwd=None):
        return run_cmd(["git", "commit", "-qm", message], cwd=cwd or self.root)

    def hook(self, relative, *args):
        return run_cmd(["bash", str(self.root / relative), *args],
                       cwd=self.root)

    def assert_no_token(self, text):
        if TOKEN in text:
            self.fail("hook output printed the synthetic token")

    def assert_hook_refused(self, result, form):
        self.assertNotEqual(result.returncode, 0, "hook accepted the commit")
        self.assertIn(form, result.stdout)
        self.assert_no_token(result.stdout)

    # ── setup ────────────────────────────────────────────────────────────────
    def test_setup_verifies_relative_path_and_executables(self):
        for path in (list((self.root / ".githooks").glob("*"))
                     + list((self.root / ".githooks" / "pre-commit.d").glob("*"))
                     + list((self.root / "bin").glob("work-*"))
                     + list((self.root / "sifty" / "scripts" / "leakcheck").glob("*"))):
            if path.is_file():
                path.chmod(0o644)
        self.install()
        self.assertEqual(
            run_cmd(["git", "config", "--get", "core.hooksPath"],
                    cwd=self.root).stdout.strip(), ".githooks")
        scanner_files = [p for p in (
            self.root / "sifty" / "scripts" / "leakcheck").glob("*")
            if p.is_file() and p.suffix != ".md"]
        for path in (list((self.root / ".githooks").glob("*"))
                     + list((self.root / ".githooks" / "pre-commit.d").glob("*"))
                     + list((self.root / "bin").glob("work-*"))
                     + scanner_files):
            if path.is_file():
                self.assertTrue(os.access(path, os.X_OK), str(path))

    def test_missing_whole_scanner_directory_fails(self):
        self.install()
        shutil.rmtree(self.root / "sifty" / "scripts" / "leakcheck")
        result = run_cmd(["bash", str(self.root / "bin" / "work-setup")],
                         cwd=self.root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("scanner directory missing", result.stdout)

    def test_each_required_hook_must_exist(self):
        hooks = ("pre-commit", "commit-msg", "pre-push",
                 "post-checkout", "post-merge", "post-rewrite")
        for hook in hooks:
            with self.subTest(hook=hook):
                self.install()
                target = self.root / ".githooks" / hook
                target.unlink()
                result = run_cmd(
                    ["bash", str(self.root / "bin" / "work-setup")],
                    cwd=self.root)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(hook, result.stdout)
                shutil.copy2(REPO_ROOT / ".githooks" / hook, target)
        self.install()

    def test_missing_required_scanner_file_fails(self):
        self.install()
        (self.root / "sifty" / "scripts" / "leakcheck" /
         "commit-diff-check").unlink()
        result = run_cmd(["bash", str(self.root / "bin" / "work-setup")],
                         cwd=self.root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("commit-diff-check", result.stdout)

    # ── optional integration ────────────────────────────────────────────────
    def test_ordinary_conventional_work_record_passes_without_shared_submodule(self):
        self.assertFalse((self.root / "ai").exists())
        self.assertFalse((self.root / "kb").exists())
        self.install()
        self.write("ai/tasks/demo/log.md",
                   "Проверить и записать результат\n")
        self.git("add", "-A")
        result = self.commit("docs(work): record checkpoint")
        self.assertEqual(result.returncode, 0, "ordinary commit was refused")

    def test_no_kb_skips_relation_build(self):
        self.install()
        result = self.hook(".githooks/pre-commit.d/10-kb-relation-build")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertNotIn("ERROR", result.stdout)

    def test_kb_docs_require_missing_relation_build(self):
        self.install()
        self.write("kb/note.md", "# note\n")
        self.git("add", "-A")
        result = self.hook(".githooks/pre-commit.d/10-kb-relation-build")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("relation-build", result.stdout)
        self.assertIn("missing", result.stdout)

    def test_post_checkout_skips_without_kb(self):
        self.install()
        result = self.hook(".githooks/post-checkout", "old", "new", "1")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertNotIn("ERROR", result.stdout)

    def test_post_checkout_reports_missing_kb_tool(self):
        self.install()
        (self.root / "kb").mkdir()
        result = self.hook(".githooks/post-checkout", "old", "new", "1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("kb-index", result.stdout)
        self.assertIn("missing", result.stdout)

    # ── required scanner ────────────────────────────────────────────────────
    def test_missing_scanner_fails_explicitly(self):
        self.install()
        (self.root / "sifty" / "scripts" / "leakcheck" /
         "commit-diff-check").unlink()
        self.write("docs/note.md", "note\n")
        self.git("add", "-A")
        result = self.hook(".githooks/pre-commit.d/30-leak-check")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("leak scanner missing", result.stdout)

    # ── actual hook behavior ────────────────────────────────────────────────
    def test_installed_hooks_refuse_secret_and_redact(self):
        self.install()
        self.write("sifty/app.ts", 'const token = "' + TOKEN + '";\n')
        self.git("add", "-A")
        result = self.commit("feat(test): add config")
        self.assert_hook_refused(result, "personal access token")

    def test_installed_hooks_refuse_secret_message(self):
        self.install()
        self.write("docs/note.md", "note\n")
        self.git("add", "-A")
        result = self.commit("feat(test): add note\n\n" + TOKEN + "\n")
        self.assert_hook_refused(result, "personal access token")

    def test_linked_worktree_uses_relative_hooks(self):
        self.install()
        linked = self.root / ".worktrees" / "linked"
        self.git("worktree", "add", "-q", str(linked), "-b", "linked")
        self.write("docs/note.md", "note\n", cwd=linked)
        self.git("add", "-A", cwd=linked)
        result = self.commit("docs(test): add note", cwd=linked)
        self.assertEqual(result.returncode, 0, "linked worktree commit was refused")

        self.write("sifty/app.ts", 'const token = "' + TOKEN + '";\n',
                   cwd=linked)
        self.git("add", "-A", cwd=linked)
        result = self.commit("feat(test): add config", cwd=linked)
        self.assert_hook_refused(result, "personal access token")


if __name__ == "__main__":
    unittest.main()
