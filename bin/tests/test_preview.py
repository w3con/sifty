"""Continuous preview supervisor tests.

These tests use a throwaway Git repository and fake go/npm commands. They
exercise the real supervisor, proxy, generation build and ownership logic on a
free scratch port; no real checkout, data directory or server is touched.
"""
import json
import os
import socket
import subprocess
import sys
import tempfile
import time
import unittest
import urllib.request
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from support import BIN, run_cmd  # noqa: E402


def free_port():
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]
    finally:
        sock.close()


FAKE_GO = r'''#!/usr/bin/env bash
set -euo pipefail
out=""; while [ $# -gt 0 ]; do case "$1" in -o) out="$2"; shift 2;; *) shift;; esac; done
[ -n "$out" ] || { echo "no -o" >&2; exit 1; }
marker="$(cat marker.txt 2>/dev/null || echo missing)"
version="${SIFTY_BUILD_ID:-dev}"
fail=0; [ -f fail-serve ] && fail=1
python3 - "$out" "$version" "$marker" "$fail" <<'PY'
import os, sys
out, version, marker, fail = sys.argv[1:5]
source = """#!/usr/bin/env python3
import http.server, json, os, sys
VERSION = %r
MARKER = %r
FAIL = int(%r)
port = 0
dist = ''
args = sys.argv[1:]
i = 0
while i < len(args):
    token = args[i]
    if token in ('--port', '--public-port', '--data-dir', '--dist', '--workers-dir'):
        value = args[i + 1] if i + 1 < len(args) else ''
        if token == '--port': port = int(value)
        elif token == '--dist': dist = value
        i += 2
    else:
        i += 1
if FAIL:
    print('fake backend fails', file=sys.stderr); sys.exit(1)
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        if self.path == '/api/version':
            body = json.dumps({'version': VERSION, 'marker': MARKER}).encode()
            self.send_response(200); self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(body))); self.end_headers(); self.wfile.write(body); return
        if self.path == '/api/health':
            body = b'[]'; self.send_response(200); self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', '2'); self.end_headers(); self.wfile.write(body); return
        if self.path in ('/', '/index.html') and dist:
            path = os.path.join(dist, 'index.html')
            if os.path.exists(path):
                body = open(path, 'rb').read(); self.send_response(200)
                self.send_header('Content-Type', 'text/html; charset=utf-8')
                self.send_header('Content-Length', str(len(body))); self.end_headers(); self.wfile.write(body); return
        self.send_response(404); self.end_headers()
http.server.ThreadingHTTPServer(('127.0.0.1', port), H).serve_forever()
""" % (version, marker, fail)
open(out, 'w').write(source)
os.chmod(out, 0o755)
PY
'''

FAKE_NPM = r'''#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "ci" ]; then mkdir -p node_modules; echo ok > node_modules/.fake; exit 0; fi
if [ "${1:-}" = "run" ] && [ "${2:-}" = "build" ]; then
  [ -f fail-build ] && { echo "fake frontend build failed" >&2; exit 1; }
  marker="$(cat marker.txt 2>/dev/null || echo missing)"
  mkdir -p "$BUILD_OUTDIR"
  printf '<!doctype html><html><body><div id="root">%s</div></body></html>\n' "$marker" > "$BUILD_OUTDIR/index.html"
  exit 0
fi
echo "fake npm unsupported: $*" >&2; exit 1
'''


class PreviewTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.base = Path(self._tmp.name)
        self.repo = self.base / "repo"
        self.fakebin = self.base / "fakebin"
        self.state = self.base / "state"
        self.data = self.base / "data"
        self.cache = self.base / "cache"
        for path in (self.repo / "backend" / "cmd" / "sifty",
                     self.repo / "backend" / "workers" / "stub",
                     self.repo / "frontend", self.fakebin,
                     self.state, self.data, self.cache):
            path.mkdir(parents=True)
        (self.repo / "backend" / "marker.txt").write_text("backend-one\n", encoding="utf-8")
        (self.repo / "backend" / "go.mod").write_text("module test\n", encoding="utf-8")
        (self.repo / "backend" / "workers" / "stub" / "index.mjs").write_text(
            "worker-one\n", encoding="utf-8")
        (self.repo / "backend" / "workers" / "stub" / "marker.txt").write_text(
            "worker-one\n", encoding="utf-8")
        (self.repo / "frontend" / "marker.txt").write_text("frontend-one\n", encoding="utf-8")
        (self.repo / "frontend" / "package.json").write_text('{"name":"fake"}\n', encoding="utf-8")
        (self.repo / "frontend" / "package-lock.json").write_text("{}\n", encoding="utf-8")
        self.write_fake("go", FAKE_GO)
        self.write_fake("npm", FAKE_NPM)
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.email", "preview-test@example.com")
        self.git("config", "user.name", "Preview Test")
        self.git("add", "-A")
        self.git("commit", "-qm", "base")
        self.port = free_port()
        self.started = False

    def tearDown(self):
        if self.started:
            run_cmd([sys.executable, str(BIN / "sift-preview"), "stop",
                     "--state-dir", str(self.state)], cwd=self.base,
                    env=self.env(), timeout=30)
        self._tmp.cleanup()

    def write_fake(self, name, text):
        path = self.fakebin / name
        path.write_text(text, encoding="utf-8")
        path.chmod(0o755)

    def git(self, *args):
        result = run_cmd(["git", *args], cwd=self.repo)
        self.assertEqual(result.returncode, 0, result.stdout)
        return result.stdout.strip()

    def env(self):
        return {
            "PATH": str(self.fakebin) + os.pathsep + os.environ.get("PATH", ""),
            "SIFTY_WORKERS_SOURCE": str(self.repo / "backend" / "workers"),
            "SIFTY_WORKERS_CACHE_DIR": str(self.cache / "workers"),
            "SIFTY_FRONTEND_CACHE_DIR": str(self.cache / "frontend"),
        }

    def start(self):
        result = run_cmd(
            [sys.executable, str(BIN / "sift-preview"), "start",
             "--worktree", str(self.repo), "--state-dir", str(self.state),
             "--data-dir", str(self.data), "--port", str(self.port),
             "--poll", "0.05", "--debounce", "0.1", "--wait", "30"],
            cwd=self.repo, env=self.env(), timeout=45)
        self.started = result.returncode == 0
        self.assertEqual(result.returncode, 0, result.stdout)
        return result

    def status(self):
        with urllib.request.urlopen(
                "http://127.0.0.1:%d/__preview/status" % self.port, timeout=3) as response:
            return json.loads(response.read().decode("utf-8"))

    def fetch(self, path="/"):
        with urllib.request.urlopen(
                "http://127.0.0.1:%d%s" % (self.port, path), timeout=3) as response:
            return response.read().decode("utf-8")

    def wait_for(self, predicate, timeout=25, message="condition"):
        deadline = time.time() + timeout
        last = None
        while time.time() < deadline:
            try:
                last = self.status()
                if predicate(last):
                    return last
            except Exception:
                pass
            time.sleep(0.1)
        self.fail("timed out waiting for %s; last status=%s" %
                  (message, json.dumps(last, indent=2)[:4000]))

    def raised(self, old_fingerprint):
        return lambda status: (
            status.get("served") is not None and
            status["served"]["fingerprint"] != old_fingerprint and
            status.get("requested") is not None and
            status["served"]["fingerprint"] == status["requested"]["fingerprint"])

    def test_refresh_each_component_and_untracked_change(self):
        self.start()
        status = self.wait_for(lambda s: s.get("served") is not None, message="initial build")
        self.assertEqual(status["state"], "running")
        self.assertEqual(status["requested"]["fingerprint"],
                         status["served"]["fingerprint"])
        self.assertIn("frontend-one", self.fetch("/"))
        self.assertIn("/__preview/client.js", self.fetch("/"))
        first_builds = status["build"]["count"]
        first_backend_pid = status["served"]["backendPid"]

        # Untracked frontend change.
        (self.repo / "frontend" / "untracked.txt").write_text("new\n", encoding="utf-8")
        status = self.wait_for(self.raised(status["served"]["fingerprint"]),
                               message="untracked frontend build")
        self.assertGreater(status["build"]["count"], first_builds)
        self.assertEqual(status["served"]["backendPid"], first_backend_pid)

        # Frontend-only change: new bundle, same backend process.
        old = status["served"]["fingerprint"]
        (self.repo / "frontend" / "marker.txt").write_text("frontend-two\n", encoding="utf-8")
        status = self.wait_for(self.raised(old), message="frontend rebuild")
        self.assertIn("frontend-two", self.fetch("/"))
        self.assertEqual(status["served"]["backendPid"], first_backend_pid)

        # Backend change: new process and served revision.
        old = status["served"]["fingerprint"]
        (self.repo / "backend" / "marker.txt").write_text("backend-two\n", encoding="utf-8")
        status = self.wait_for(self.raised(old), message="backend rebuild")
        self.assertNotEqual(status["served"]["backendPid"], first_backend_pid)

        # Worker change: published workers update without a backend restart.
        old = status["served"]["fingerprint"]
        pid = status["served"]["backendPid"]
        (self.repo / "backend" / "workers" / "stub" / "index.mjs").write_text(
            "worker-two\n", encoding="utf-8")
        status = self.wait_for(self.raised(old), message="worker rebuild")
        worker = self.state / "current" / "workers" / "stub" / "index.mjs"
        self.assertEqual(worker.read_text(encoding="utf-8"), "worker-two\n")
        self.assertEqual(status["served"]["backendPid"], pid)

        # Stable sources must not rebuild.
        builds = status["build"]["count"]
        time.sleep(1.2)
        self.assertEqual(self.status()["build"]["count"], builds)

    def test_failed_build_keeps_serving_last_healthy_generation_and_recovers(self):
        self.start()
        status = self.wait_for(lambda s: s.get("served") is not None, message="initial build")
        old_generation = status["served"]["generation"]
        old_fingerprint = status["served"]["fingerprint"]
        self.assertIn("frontend-one", self.fetch("/"))

        (self.repo / "frontend" / "fail-build").write_text("fail\n", encoding="utf-8")
        (self.repo / "frontend" / "marker.txt").write_text("broken\n", encoding="utf-8")
        status = self.wait_for(lambda s: s["state"] == "failed" and s["build"]["lastError"],
                               message="visible build failure")
        self.assertEqual(status["served"]["generation"], old_generation)
        self.assertEqual(status["served"]["fingerprint"], old_fingerprint)
        self.assertNotEqual(status["requested"]["fingerprint"], old_fingerprint)
        self.assertIn("frontend", status["build"]["lastStage"])
        self.assertIn("frontend-one", self.fetch("/"))
        body = self.fetch("/__preview/status")
        self.assertIn("failed", body)

        (self.repo / "frontend" / "fail-build").unlink()
        (self.repo / "frontend" / "marker.txt").write_text("recovered\n", encoding="utf-8")
        status = self.wait_for(self.raised(old_fingerprint), message="recovery build")
        self.assertEqual(status["state"], "running")
        self.assertIn("recovered", self.fetch("/"))
        self.assertNotEqual(status["served"]["generation"], old_generation)

    def test_once_builds_checks_and_stops(self):
        result = run_cmd(
            [sys.executable, str(BIN / "sift-preview"), "once",
             "--worktree", str(self.repo), "--state-dir", str(self.state),
             "--data-dir", str(self.data), "--port", str(self.port)],
            cwd=self.repo, env=self.env(), timeout=45)
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("state=built", result.stdout)
        self.assertIn("Served:", result.stdout)
        status = run_cmd(
            [sys.executable, str(BIN / "sift-preview"), "status", "--json",
             "--state-dir", str(self.state)],
            cwd=self.base, env=self.env(), timeout=10)
        data = json.loads(status.stdout)
        self.assertFalse(data["running"])
        self.assertIsNotNone(data["status"]["served"])
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
            self.assertNotEqual(sock.connect_ex(("127.0.0.1", self.port)), 0)

    def test_single_owner_and_stop_ownership(self):
        self.start()
        first = self.wait_for(lambda s: s.get("served") is not None, message="initial build")
        owner_token = first["supervisor"]["token"]

        second = run_cmd(
            [sys.executable, str(BIN / "sift-preview"), "start",
             "--worktree", str(self.repo), "--state-dir", str(self.state),
             "--data-dir", str(self.data), "--port", str(self.port),
             "--label", "second", "--wait", "3"],
            cwd=self.repo, env=self.env(), timeout=10)
        self.assertEqual(second.returncode, 3, second.stdout)
        self.assertIn("already active", second.stdout)
        self.assertEqual(self.status()["supervisor"]["token"], owner_token)

        # A state directory whose owner record points at an unrelated process
        # must never be stopped.
        foreign = self.base / "foreign-state"
        foreign.mkdir()
        sleeper = subprocess.Popen(["sleep", "30"], start_new_session=True)
        try:
            (foreign / "owner.json").write_text(json.dumps({
                "schema": 1, "pid": sleeper.pid, "token": "foreign",
                "startedAt": "2026-11-06T00:00:00Z",
                "processStart": "unknown", "worktree": str(self.repo),
                "port": self.port, "label": "foreign",
            }), encoding="utf-8")
            refused = run_cmd(
                [sys.executable, str(BIN / "sift-preview"), "stop",
                 "--state-dir", str(foreign)], cwd=self.base, env=self.env(), timeout=10)
            self.assertEqual(refused.returncode, 1, refused.stdout)
            self.assertIsNone(sleeper.poll())
        finally:
            sleeper.terminate()
            try:
                sleeper.wait(timeout=5)
            except subprocess.TimeoutExpired:
                sleeper.kill()

        stopped = run_cmd(
            [sys.executable, str(BIN / "sift-preview"), "stop",
             "--state-dir", str(self.state)], cwd=self.base, env=self.env(), timeout=30)
        self.assertEqual(stopped.returncode, 0, stopped.stdout)
        self.started = False
        deadline = time.time() + 5
        while time.time() < deadline:
            try:
                with socket.create_connection(("127.0.0.1", self.port), timeout=0.2):
                    time.sleep(0.1)
            except OSError:
                break
        else:
            self.fail("preview port still accepts connections after stop")


if __name__ == "__main__":
    unittest.main()
