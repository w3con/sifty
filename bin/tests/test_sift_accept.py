"""Tests for bin/sift-accept's check selection, using fake go/npm binaries.

The real application is never built and no server or production data is
touched. A copy of sift-accept is placed in a temporary repository so its
APP_HOME is that repository.
"""
import os
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import BIN, run_cmd  # noqa: E402


class SiftAcceptTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        self.record = self.root.parent / (self.root.name + "-record.txt")
        self.record.write_text("", encoding="utf-8")
        self.fakebin = self.root.parent / (self.root.name + "-fakebin")
        self.fakebin.mkdir()
        self.write_fake("go", '#!/usr/bin/env bash\necho "go $*" >> "$RECORD"\n')
        self.write_fake("npm", (
            '#!/usr/bin/env bash\n'
            'set -euo pipefail\n'
            'echo "npm $*" >> "$RECORD"\n'
            'if [[ "${1:-}" == "ci" ]]; then\n'
            '  [[ "${FAKE_NPM_CI_EXIT:-0}" == "0" ]] || exit 1\n'
            'fi\n'
            'if [[ "${1:-}" == "run" && "${2:-}" == "build" ]]; then\n'
            '  echo "BUILD_OUTDIR=${BUILD_OUTDIR:-}" >> "$RECORD"\n'
            'fi\n'
        ))

        (self.root / "bin").mkdir()
        shutil.copy2(BIN / "sift-accept", self.root / "bin" / "sift-accept")
        (self.root / "bin" / "work-test").write_text(
            'import os\n'
            'with open(os.environ["RECORD"], "a", encoding="utf-8") as fh:\n'
            '    fh.write("work-test\\n")\n'
            'raise SystemExit(int(os.environ.get("FAKE_WORK_TEST_EXIT", "0")))\n',
            encoding="utf-8")
        (self.root / "sifty" / "web").mkdir(parents=True)
        (self.root / "docs").mkdir()
        (self.root / "docs" / "base.md").write_text("base\n", encoding="utf-8")
        (self.root / "Makefile").write_text("all:\n\t@true\n", encoding="utf-8")
        (self.root / "bin" / "sift-deploy").write_text(
            "#!/usr/bin/env bash\nexit 0\n", encoding="utf-8")
        (self.root / "sifty" / "web" / "app.ts").write_text(
            "export const x = 1;\n", encoding="utf-8")

        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Work Test")
        self.git("config", "user.email", "work-test@example.com")
        self.git("add", "-A")
        self.git("commit", "-qm", "base")

    def tearDown(self):
        self._tmp.cleanup()
        shutil.rmtree(self.fakebin, ignore_errors=True)
        self.record.unlink(missing_ok=True)

    def write_fake(self, name, text):
        path = self.fakebin / name
        path.write_text(text, encoding="utf-8")
        path.chmod(0o755)

    def git(self, *args):
        result = run_cmd(["git", *args], cwd=self.root)
        self.assertEqual(result.returncode, 0, result.stdout)
        return result.stdout.strip()

    def change(self, relative, text="changed\n"):
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
        self.git("add", "-A")
        self.git("commit", "-qm", "change %s" % relative)

    def run_accept(self, extra_env=None):
        env = {"RECORD": str(self.record),
               "PATH": str(self.fakebin) + os.pathsep + os.environ.get("PATH", "")}
        if extra_env:
            env.update(extra_env)
        return run_cmd(["bash", str(self.root / "bin" / "sift-accept")],
                       cwd=self.root, env=env)

    def recorded(self):
        return self.record.read_text(encoding="utf-8").splitlines()

    def test_harness_test_always_runs(self):
        self.git("checkout", "-q", "-b", "work/X")
        self.change("docs/change.md")
        result = self.run_accept()
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.recorded(), ["work-test"])

    def test_build_change_runs_application_checks(self):
        self.git("checkout", "-q", "-b", "work/X")
        self.change("bin/sift-deploy", "#!/usr/bin/env bash\n# changed\nexit 0\n")
        result = self.run_accept()
        self.assertEqual(result.returncode, 0, result.stdout)
        lines = self.recorded()
        self.assertEqual(lines[0], "work-test")
        self.assertTrue(any(line.startswith("go test -race") for line in lines))
        self.assertIn("npm ci --silent", lines)
        self.assertIn("npm run build", lines)
        self.assertTrue(any(line.startswith("BUILD_OUTDIR=") and
                            line != "BUILD_OUTDIR=" for line in lines))

    def test_app_change_runs_application_checks(self):
        self.git("checkout", "-q", "-b", "work/X")
        self.change("sifty/web/app.ts", "export const x = 2;\n")
        result = self.run_accept()
        self.assertEqual(result.returncode, 0, result.stdout)
        lines = self.recorded()
        self.assertEqual(lines[0], "work-test")
        self.assertTrue(any(line.startswith("go test -race") for line in lines))
        self.assertIn("npm ci --silent", lines)

    def test_npm_ci_failure_is_not_silently_replaced(self):
        self.git("checkout", "-q", "-b", "work/X")
        self.change("Makefile", "all:\n\t@true\n# changed\n")
        result = self.run_accept({"FAKE_NPM_CI_EXIT": "1"})
        self.assertNotEqual(result.returncode, 0, result.stdout)
        lines = self.recorded()
        self.assertIn("npm ci --silent", lines)
        self.assertNotIn("npm install", " ".join(lines))
        self.assertNotIn("npm run build", lines)

    def test_harness_failure_stops_before_application_checks(self):
        self.git("checkout", "-q", "-b", "work/X")
        self.change("Makefile", "all:\n\t@true\n# changed\n")
        result = self.run_accept({"FAKE_WORK_TEST_EXIT": "1"})
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.recorded(), ["work-test"])


if __name__ == "__main__":
    unittest.main()
