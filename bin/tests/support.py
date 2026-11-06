"""Shared fixtures for the work-* lifecycle tests.

Every test builds a tiny throwaway Git repository and runs the real scripts
from this checkout with that repository as the working directory. Nothing here
touches the real main checkout, its worktrees, runtime data or servers.
"""
import os
import shlex
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
BIN = REPO_ROOT / "bin"

_SCRUB_ENV = (
    "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY",
    "GIT_COMMON_DIR", "GIT_PREFIX", "GIT_NAMESPACE",
)


def clean_env(extra=None):
    env = os.environ.copy()
    for key in list(env):
        if key.startswith("WORK_"):
            del env[key]
    for key in _SCRUB_ENV:
        env.pop(key, None)
    # Keep the fixture independent of the developer's global Git configuration.
    env["GIT_CONFIG_GLOBAL"] = os.devnull
    env["GIT_CONFIG_SYSTEM"] = os.devnull
    env["GIT_CONFIG_NOSYSTEM"] = "1"
    env["GIT_TERMINAL_PROMPT"] = "0"
    env.update({
        "GIT_AUTHOR_NAME": "Work Test",
        "GIT_AUTHOR_EMAIL": "work-test@example.com",
        "GIT_COMMITTER_NAME": "Work Test",
        "GIT_COMMITTER_EMAIL": "work-test@example.com",
    })
    if extra:
        env.update(extra)
    return env


def run_cmd(cmd, cwd, env=None, timeout=120):
    return subprocess.run(
        [str(part) for part in cmd],
        cwd=str(cwd),
        env=clean_env(env),
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        timeout=timeout,
    )


class LifecycleTest(unittest.TestCase):
    """A temp repo with a base card and a work/<id> branch + worktree."""

    card_id = "TST-1"

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self._aux = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        self.aux_dir = Path(self._aux.name)
        self.card_rel = Path("work", "tasks", "test", "%s_demo" % self.card_id)
        self.card_dir = self.root / self.card_rel
        self.card_dir.mkdir(parents=True)

        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Work Test")
        self.git("config", "user.email", "work-test@example.com")
        (self.root / ".gitignore").write_text(
            ".worktrees/\n.work-scratch/\nignored/\n", encoding="utf-8")
        (self.root / "README").write_text("base\n", encoding="utf-8")
        self.write_card(closed=False)
        self.git("add", "-A")
        self.git("commit", "-qm", "base")
        self.base_sha = self.head()

        self.git("checkout", "-q", "-b", "work/%s" % self.card_id)
        self.write_card(closed=True)
        self.git("add", "-A")
        self.git("commit", "-qm", "close plan steps")
        self.source_sha = self.head()
        self.git("checkout", "-q", "main")
        self.git("worktree", "add", "-q",
                 ".worktrees/%s" % self.card_id,
                 "work/%s" % self.card_id)
        self.wt = self.root / ".worktrees" / self.card_id

    def tearDown(self):
        self._tmp.cleanup()
        self._aux.cleanup()

    # ── helpers ──────────────────────────────────────────────────────────────
    def write_card(self, closed):
        step = "- [x] 1.1 — do the thing" if closed else "- [ ] 1.1 — do the thing"
        plan = ("# %s — plan\n\n## Steps\n\n%s\n\n## Acceptance criteria\n\n"
                "- The thing works.\n" % (self.card_id, step))
        (self.card_dir / "plan.md").write_text(plan, encoding="utf-8")
        (self.card_dir / "attention.md").write_text(
            "# %s — attention\n" % self.card_id, encoding="utf-8")
        (self.card_dir / "log.md").write_text(
            "# %s — log\n" % self.card_id, encoding="utf-8")

    def git(self, *args, cwd=None):
        result = run_cmd(["git", *args], cwd=cwd or self.root)
        if result.returncode != 0:
            raise AssertionError(
                "git %s failed (%d):\n%s"
                % (" ".join(str(a) for a in args), result.returncode,
                   result.stdout))
        return result.stdout.strip()

    def head(self, cwd=None):
        return self.git("rev-parse", "HEAD", cwd=cwd)

    def work(self, command, *args, env=None, cwd=None):
        return run_cmd(
            [sys.executable, str(BIN / command), *args],
            cwd=cwd or self.root, env=env)

    def commit_in_source(self, message):
        self.git("add", "-A", cwd=self.wt)
        self.git("commit", "-qm", message, cwd=self.wt)
        self.source_sha = self.head(cwd=self.wt)

    def make_accept_script(self, exit_code=0, record=None, move_source=False,
                           move_target=False, change_target_branch=False,
                           dirty_source=False, dirty_target=False):
        script = self.aux_dir / "accept.py"
        script.write_text(
            "import os, subprocess, sys\n"
            "SOURCE = os.environ.get('WORK_ACCEPT_SOURCE_WORKTREE')\n"
            "TARGET = os.environ.get('WORK_ACCEPT_TARGET_ROOT')\n"
            "def sh(*args):\n"
            "    subprocess.check_call(args, stdout=subprocess.DEVNULL, "
            "stderr=subprocess.DEVNULL)\n"
            "if %r:\n"
            "    with open(%r, 'w', encoding='utf-8') as fh:\n"
            "        fh.write(subprocess.check_output("
            "['git', 'rev-parse', 'HEAD'], text=True).strip() + '\\n')\n"
            "        fh.write(os.environ.get('WORK_ACCEPT_SOURCE_SHA', '') "
            "+ '\\n')\n"
            "if %r:\n"
            "    sh('git', '-C', SOURCE, 'commit', '--allow-empty', '-m', "
            "'move source')\n"
            "if %r:\n"
            "    sh('git', '-C', TARGET, 'commit', '--allow-empty', '-m', "
            "'move target')\n"
            "if %r:\n"
            "    sh('git', '-C', TARGET, 'checkout', '-q', '-b', 'other-target')\n"
            "if %r:\n"
            "    open(os.path.join(SOURCE, 'dirty-source.txt'), 'w', "
            "encoding='utf-8').write('dirty\\n')\n"
            "if %r:\n"
            "    open(os.path.join(TARGET, 'dirty-target.txt'), 'w', "
            "encoding='utf-8').write('dirty\\n')\n"
            "sys.exit(%d)\n"
            % (bool(record), str(record) if record else "", move_source,
               move_target, change_target_branch, dirty_source, dirty_target,
               exit_code),
            encoding="utf-8")
        return "%s %s" % (shlex.quote(sys.executable), shlex.quote(str(script)))

    def assert_refused(self, result, fragment):
        self.assertEqual(result.returncode, 2, result.stdout)
        self.assertIn(fragment, result.stdout)

    def assert_preserved(self):
        self.assertTrue(self.wt.exists(), "source worktree was removed")
        self.assertIn("work/%s" % self.card_id,
                      self.git("branch", "--list", "work/%s" % self.card_id))
        self.assertFalse((self.card_dir / "done.md").exists(),
                         "done.md was written on a failed finish")
