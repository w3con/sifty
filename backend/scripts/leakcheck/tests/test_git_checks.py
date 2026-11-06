"""Integration tests for the leak scanner's Git-facing checks.

Each test runs against a throwaway Git repository. The real repository, its worktrees, data
directories and servers are never touched. Synthetic credentials are assembled at runtime so
this file carries no contiguous fake token, and every diagnostic assertion proves the matched
value did not reach stdout or stderr.
"""
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCANNER_DIR = Path(__file__).resolve().parents[1]
COMMIT_DIFF = SCANNER_DIR / "commit-diff-check"
COMMIT_MSG = SCANNER_DIR / "commit-msg-check"
RANGE_CHECK = SCANNER_DIR / "range-check"
TREE_CHECK = SCANNER_DIR / "tree-check"

TOKEN = "ghp_" + "a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q7R8"
ASSIGNED = "9f8d7a6b5c4e" + "3f2a1b0c9d8e"
PEM_BLOCK = ("-----BEGIN " + "RSA PRIVATE KEY" + "-----\n"
             + "MIIE" + "A" * 40 + "\n"
             + "-----END " + "RSA PRIVATE KEY" + "-----")


def clean_env():
    env = os.environ.copy()
    env["GIT_CONFIG_GLOBAL"] = os.devnull
    env["GIT_CONFIG_SYSTEM"] = os.devnull
    env["GIT_CONFIG_NOSYSTEM"] = "1"
    env["GIT_TERMINAL_PROMPT"] = "0"
    env.update({
        "GIT_AUTHOR_NAME": "Leak Test",
        "GIT_AUTHOR_EMAIL": "leak-test@example.com",
        "GIT_COMMITTER_NAME": "Leak Test",
        "GIT_COMMITTER_EMAIL": "leak-test@example.com",
    })
    return env


class GitScannerTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        self.env = clean_env()
        self.init_repo(self.root)

    def tearDown(self):
        self._tmp.cleanup()

    # ── helpers ──────────────────────────────────────────────────────────────
    def init_repo(self, root):
        self.git("init", "-q", "-b", "main", cwd=root)
        self.git("config", "user.name", "Leak Test", cwd=root)
        self.git("config", "user.email", "leak-test@example.com", cwd=root)

    def git(self, *args, cwd=None, check=True):
        result = subprocess.run(
            ["git", *args], cwd=str(cwd or self.root), env=self.env,
            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        if check and result.returncode != 0:
            self.fail("git %s failed with exit %d"
                      % (" ".join(args), result.returncode))
        return result

    def write(self, relative, text):
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")

    def base_commit(self):
        self.write("README.md", "base\n")
        self.git("add", "-A")
        self.git("commit", "-qm", "chore(test): base")

    def stage(self):
        self.git("add", "-A")

    def run_scanner(self, script, *args):
        return subprocess.run(
            [sys.executable, str(script), *args, "--repo", str(self.root)],
            cwd=str(self.root), env=self.env, text=True,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    def run_message_check(self, message, *args):
        message_path = self.root / ".commit-message"
        message_path.write_text(message, encoding="utf-8")
        self.addCleanup(lambda: message_path.unlink(missing_ok=True))
        return self.run_scanner(COMMIT_MSG, str(message_path), *args)

    def assert_refused(self, result, form):
        self.assertEqual(result.returncode, 1,
                         "scanner exit %d" % result.returncode)
        combined = result.stdout + result.stderr
        self.assertIn(form, combined)
        if TOKEN in combined:
            self.fail("diagnostic printed the synthetic token")

    def assert_clean(self, result):
        self.assertEqual(result.returncode, 0,
                         "scanner exit %d" % result.returncode)

    # ── credential checks in every area ─────────────────────────────────────
    def test_staged_source_secret_refused_and_redacted(self):
        self.base_commit()
        self.write("backend/app.ts", 'const token = "' + TOKEN + '";\n')
        self.stage()
        self.assert_refused(self.run_scanner(COMMIT_DIFF, "--staged"),
                            "personal access token")

    def test_staged_test_file_secret_refused(self):
        self.base_commit()
        self.write("tests/test_app.py", 'token = "' + TOKEN + '"\n')
        self.stage()
        self.assert_refused(self.run_scanner(COMMIT_DIFF, "--staged"),
                            "personal access token")

    def test_staged_scanner_source_secret_refused(self):
        self.base_commit()
        self.write("backend/scripts/leakcheck/example.py",
                   'token = "' + TOKEN + '"\n')
        self.stage()
        self.assert_refused(self.run_scanner(COMMIT_DIFF, "--staged"),
                            "personal access token")

    def test_commit_message_secret_refused(self):
        self.base_commit()
        self.write("backend/app.ts", "export const x = 1;\n")
        self.stage()
        self.assert_refused(
            self.run_message_check("feat(test): add config\n\n" + TOKEN + "\n"),
            "personal access token")

    def test_scanner_sources_do_not_match_themselves(self):
        self.base_commit()
        target = self.root / "backend" / "scripts" / "leakcheck"
        for source in SCANNER_DIR.rglob("*"):
            if not source.is_file() or "__pycache__" in source.parts:
                continue
            relative = source.relative_to(SCANNER_DIR)
            destination = target / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
        self.stage()
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--staged"))

    # ── renames, root commits and ranges ────────────────────────────────────
    def test_pure_rename_scans_the_new_blob(self):
        self.write("old.txt", 'token = "' + TOKEN + '"\n')
        self.git("add", "-A")
        self.git("commit", "--no-verify", "-qm",
                 "chore(test): base content")
        self.git("mv", "old.txt", "new.txt")
        self.assert_refused(self.run_scanner(COMMIT_DIFF, "--staged"),
                            "personal access token")

    def test_pure_rename_into_compatibility_path_is_clean(self):
        self.write("sifty/web/model/core.ts",
                   "const base = process.env.FCTO_STATE_DIR ?? '.local';\n")
        self.git("add", "-A")
        self.git("commit", "--no-verify", "-qm",
                 "chore(test): base content")
        (self.root / "frontend" / "model").mkdir(parents=True)
        self.git("mv", "sifty/web/model/core.ts", "frontend/model/core.ts")
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--staged"))

    def test_compatibility_literal_at_another_path_is_refused(self):
        self.base_commit()
        self.write("frontend/model/other.ts",
                   "const base = process.env.FCTO_STATE_DIR ?? '.local';\n")
        self.stage()
        self.assert_refused(self.run_scanner(COMMIT_DIFF, "--staged"),
                            "internal literal named by the census")

    def test_root_commit_is_scanned(self):
        other = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, other, True)
        self.init_repo(other)
        (other / "secret.txt").write_text('token = "' + TOKEN + '"\n',
                                          encoding="utf-8")
        self.git("add", "-A", cwd=other)
        self.git("commit", "--no-verify", "-qm",
                 "chore(test): root content", cwd=other)
        result = subprocess.run(
            [sys.executable, str(COMMIT_DIFF), "--repo", str(other),
             "--commit", "HEAD"],
            cwd=str(other), env=self.env, text=True,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.assert_refused(result, "personal access token")

    def test_range_scan_finds_a_secret_commit(self):
        self.base_commit()
        base = self.git("rev-parse", "HEAD").stdout.strip()
        self.write("secret.txt", 'token = "' + TOKEN + '"\n')
        self.stage()
        self.git("commit", "--no-verify", "-qm",
                 "chore(test): add config")
        result = self.run_scanner(RANGE_CHECK, base + "..HEAD")
        self.assert_refused(result, "personal access token")
        self.assertIn("commit ", result.stderr)

    # ── index accuracy, paths and symlinks ──────────────────────────────────
    def test_staged_content_is_read_from_the_index(self):
        self.base_commit()
        self.write("backend/app.ts", "export const x = 1;\n")
        self.stage()
        # An unstaged edit must not be treated as staged content.
        self.write("backend/app.ts", 'const token = "' + TOKEN + '";\n')
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--staged"))
        self.stage()
        self.assert_refused(self.run_scanner(COMMIT_DIFF, "--staged"),
                            "personal access token")

    def test_paths_with_spaces_and_non_ascii(self):
        self.base_commit()
        self.write("backend/dir with space/файл.ts",
                   'const token = "' + TOKEN + '";\n')
        self.stage()
        result = self.run_scanner(COMMIT_DIFF, "--staged")
        self.assert_refused(result, "personal access token")
        self.assertIn("backend/dir with space/файл.ts", result.stderr)

    def test_tracked_symlink_is_not_followed(self):
        self.base_commit()
        outside = self.root.parent / (self.root.name + "-outside.txt")
        outside.write_text('token = "' + TOKEN + '"\n', encoding="utf-8")
        self.addCleanup(outside.unlink, missing_ok=True)
        self.write("backend/app.ts", "export const x = 1;\n")
        os.symlink(str(outside), self.root / "backend" / "link.ts")
        self.stage()
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--staged"))

    def test_gitlink_is_not_recursed(self):
        self.base_commit()
        gitlink = "1" * 40
        self.git("update-index", "--add", "--cacheinfo",
                 "160000,%s,vendor/sub" % gitlink)
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--staged"))

    def test_tree_scan_reads_the_index_not_unstaged_edits(self):
        self.base_commit()
        self.write("backend/app.ts", "export const x = 1;\n")
        self.stage()
        self.write("backend/app.ts", 'const token = "' + TOKEN + '";\n')
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--tree"))

    def test_git_error_is_not_a_clean_result(self):
        result = self.run_scanner(COMMIT_DIFF, "--commit", "deadbeef")
        self.assertEqual(result.returncode, 2,
                         "scanner exit %d" % result.returncode)
        self.assertNotIn("checked, clean", result.stdout)

    # ── formatting areas and message shape ──────────────────────────────────
    def test_task_ids_and_work_commands_are_allowed_in_source(self):
        self.base_commit()
        self.write("backend/app.ts",
                   "// OPS-4 uses bin/work-test and bin/work-commit\n")
        self.stage()
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--staged"))

    def test_russian_work_record_is_allowed(self):
        self.base_commit()
        self.write("ai/tasks/demo/log.md", "Проверить и записать результат\n")
        self.stage()
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--staged"))

    def test_russian_application_source_is_refused(self):
        self.base_commit()
        self.write("backend/app.ts", "// Проверить и записать результат\n")
        self.stage()
        self.assert_refused(self.run_scanner(COMMIT_DIFF, "--staged"),
                            "Russian-language text")

    def test_benign_environment_assignments_pass(self):
        self.base_commit()
        self.write("backend/config.txt",
                   "WORK_SESSION=1\nSIFTY_PROD_PORT=8080\n"
                   "WORK_PUSH=0\nAPI_KEY=your-key-here\n")
        self.stage()
        self.assert_clean(self.run_scanner(COMMIT_DIFF, "--staged"))

    def test_real_credential_assignment_fails(self):
        self.base_commit()
        self.write("backend/config.txt", "api_key = " + ASSIGNED + "\n")
        self.stage()
        self.assert_refused(self.run_scanner(COMMIT_DIFF, "--staged"),
                            "secret assigned a real value")

    def test_conventional_scope_may_name_a_task(self):
        self.base_commit()
        self.write("backend/app.ts", "export const x = 1;\n")
        self.stage()
        self.assert_clean(self.run_message_check(
            "docs(OPS-4): record the checkpoint\n"))

    def test_russian_message_touching_source_is_refused(self):
        self.base_commit()
        self.write("backend/app.ts", "export const x = 1;\n")
        self.stage()
        self.assert_refused(self.run_message_check(
            "docs(test): Проверить результат\n"), "Russian-language text")

    # ── historical tree ─────────────────────────────────────────────────────
    def test_tree_check_redacts_origin_credentials(self):
        self.base_commit()
        origin = "https://user:" + TOKEN + "@example.com/repo.git"
        self.git("remote", "add", "origin", origin)
        result = subprocess.run(
            [sys.executable, str(TREE_CHECK), str(self.root)],
            cwd=str(self.root), env=self.env, text=True,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.assertEqual(result.returncode, 0,
                         "scanner exit %d" % result.returncode)
        if TOKEN in result.stdout + result.stderr:
            self.fail("diagnostic printed a credential from the origin URL")

    def test_tree_check_ref_reads_commit_blobs(self):
        self.base_commit()
        self.write("secret.txt", 'token = "' + TOKEN + '"\n')
        self.stage()
        self.git("commit", "--no-verify", "-qm",
                 "chore(test): add config")
        result = subprocess.run(
            [sys.executable, str(TREE_CHECK), str(self.root),
             "--ref", "HEAD"],
            cwd=str(self.root), env=self.env, text=True,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.assert_refused(result, "personal access token")


if __name__ == "__main__":
    unittest.main()
