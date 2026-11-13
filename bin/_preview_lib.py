#!/usr/bin/env python3
"""Continuous preview supervisor for one active card worktree.

The supervisor watches frontend, backend and worker sources (including
untracked/uncommitted files), builds them into immutable generation
directories, and serves a reverse proxy that injects a small status banner.
The production server is never modified: the banner and status endpoint exist
only in this local proxy.

One supervisor owns a main repository at a time through an OS file lock. A
second start is refused; stop only terminates a process whose recorded owner
token, pid and command line match.
"""
from __future__ import annotations

import argparse
import fcntl
import hashlib
import http.client
import json
import os
import shutil
import signal
import socket
import subprocess
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import _deploy_lib as d

SCHEMA = 1
CLIENT_PATH = Path(__file__).with_name("preview-client.js")
EXCLUDE_FRONTEND = {"node_modules", "dist", "dist-staging", "dist-preview",
                    ".vite", "coverage", ".cache", "__pycache__"}
EXCLUDE_BACKEND = {"node_modules", "bin", "dist", "build", ".cache",
                   "__pycache__", "workers-prod", "workers-staging", "workers"}
EXCLUDE_WORKERS = {"node_modules", ".cache", "dist", "__pycache__"}


class PreviewError(RuntimeError):
    pass


class PreviewBuildError(PreviewError):
    def __init__(self, stage: str, message: str):
        super().__init__("%s: %s" % (stage, message))
        self.stage = stage
        self.message = message


def _files_under(root: Path, excludes: set[str]) -> list[Path]:
    if not root.is_dir():
        return []
    found: list[Path] = []
    for dirpath, dirnames, filenames in os.walk(root, followlinks=False):
        dirnames[:] = sorted(dn for dn in dirnames if dn not in excludes)
        for name in sorted(filenames):
            if name in excludes:
                continue
            path = Path(dirpath) / name
            if path.is_file() and not path.is_symlink():
                found.append(path)
    return found


def _signature(files: list[Path], base: Path) -> dict:
    h = hashlib.sha256()
    for path in sorted(files, key=lambda p: p.relative_to(base).as_posix()):
        rel = path.relative_to(base).as_posix()
        stat = path.stat()
        h.update(rel.encode("utf-8"))
        h.update(b"\0")
        h.update(str(stat.st_size).encode("ascii"))
        h.update(b"\0")
        h.update(str(stat.st_mtime_ns).encode("ascii"))
        h.update(b"\n")
    return {"hash": h.hexdigest(), "files": len(files)}


def scan_sources(worktree: Path, workers_source: Path) -> dict:
    frontend = _signature(_files_under(worktree / "frontend", EXCLUDE_FRONTEND),
                          worktree / "frontend")
    backend = _signature(_files_under(worktree / "backend", EXCLUDE_BACKEND),
                         worktree / "backend")
    workers = _signature(_files_under(workers_source, EXCLUDE_WORKERS),
                         workers_source)
    all_hash = hashlib.sha256(
        ("%s:%s:%s" % (frontend["hash"], backend["hash"], workers["hash"])).encode("ascii")
    ).hexdigest()
    return {"frontend": frontend, "backend": backend, "workers": workers,
            "all": all_hash}


def git_info(worktree: Path) -> dict:
    try:
        commit = d.git(worktree, "rev-parse", "HEAD")
        status = subprocess.run(
            ["git", "-C", str(worktree), "status", "--porcelain"],
            text=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL).stdout
        return {"commit": commit, "dirty": bool(status.strip())}
    except Exception:
        return {"commit": "", "dirty": True}


def state_status_path(state: Path) -> Path:
    return state / "status.json"


def owner_path(state: Path) -> Path:
    return state / "owner.json"


def lock_path(state: Path) -> Path:
    return state / "lock"


def current_link(state: Path) -> Path:
    return state / "current"


def read_current(state: Path) -> Path | None:
    link = current_link(state)
    if not link.is_symlink():
        return None
    target = (state / os.readlink(link)).resolve()
    return target if target.is_dir() else None


def read_current_meta(state: Path) -> dict | None:
    current = read_current(state)
    if current is None:
        return None
    return d.read_json(current / "meta.json")


def swap_current(state: Path, generation: str) -> None:
    link = current_link(state)
    tmp = state / (".current.tmp.%d" % os.getpid())
    try:
        tmp.unlink()
    except FileNotFoundError:
        pass
    os.symlink(os.path.join("generations", generation), tmp)
    os.replace(tmp, link)


def write_status(state: Path, status: dict) -> None:
    d.write_json_atomic(state_status_path(state), status)


def read_status(state: Path) -> dict | None:
    return d.read_json(state_status_path(state))


def _component_path(name: str) -> str:
    return {"frontend": "dist", "backend": "bin", "workers": "workers"}[name]


def build_generation(worktree: Path, state: Path, generation: str,
                     signature: dict, git: dict, workers_source: Path,
                     build_log: Path, workers_cache: Path,
                     frontend_cache: Path) -> dict:
    previous = read_current_meta(state)
    previous_dir = read_current(state)
    gen_dir = state / "generations" / generation
    if gen_dir.exists():
        shutil.rmtree(gen_dir)
    gen_dir.mkdir(parents=True)
    changed: dict[str, bool] = {}
    component_generations: dict[str, str] = {}
    try:
        for component in ("frontend", "backend", "workers"):
            target = gen_dir / _component_path(component)
            prev_path = (previous_dir / _component_path(component)) if previous_dir else None
            previous_signature = ({} if previous is None else
                                  ((previous.get("signatures") or {})
                                   .get(component) or {}))
            is_changed = (previous is None or prev_path is None or
                          not prev_path.exists() or
                          previous_signature.get("hash") !=
                          signature[component]["hash"])
            changed[component] = is_changed
            if not is_changed:
                os.symlink(os.path.relpath(prev_path, gen_dir), target)
                component_generations[component] = (
                    (previous.get("components") or {}).get(component) or
                    previous.get("generation") or "")
                continue
            try:
                if component == "frontend":
                    d.build_frontend(worktree / "frontend", target, build_log,
                                     generation, frontend_cache)
                elif component == "backend":
                    d.build_binary(worktree / "backend", target / "sifty",
                                   generation, build_log)
                else:
                    d.build_workers(workers_source, target, workers_cache,
                                    build_log, link=True)
            except d.DeployError as exc:
                raise PreviewBuildError(component, str(exc)) from exc
            component_generations[component] = generation
        meta = {
            "schema": SCHEMA, "generation": generation,
            "createdAt": d.now_iso(), "commit": git["commit"],
            "dirty": git["dirty"], "signatures": signature,
            "changed": changed, "components": component_generations,
        }
        d.write_json_atomic(gen_dir / "meta.json", meta)
        # A build that raced an editor must not become the served generation.
        after = scan_sources(worktree, workers_source)
        if after["all"] != signature["all"]:
            shutil.rmtree(gen_dir, ignore_errors=True)
            return {"unstable": True, "signature": after}
        return meta
    except Exception:
        shutil.rmtree(gen_dir, ignore_errors=True)
        raise


class PreviewRuntime:
    def __init__(self, cfg: dict):
        self.cfg = cfg
        self.state = Path(cfg["state"])
        self.state.mkdir(parents=True, exist_ok=True)
        self.token = cfg["token"]
        self.lock = threading.Lock()
        self.stop_event = threading.Event()
        self.backend_pid: int | None = None
        self.backend_port: int | None = None
        self.proxy: ThreadingHTTPServer | None = None
        self.proxy_thread: threading.Thread | None = None
        self.status = self._base_status()

    def _base_status(self) -> dict:
        return {
            "schema": SCHEMA,
            "state": "starting",
            "worktree": self.cfg["worktree"],
            "label": self.cfg.get("label") or "",
            "commit": "",
            "dirty": True,
            "requested": None,
            "served": None,
            "build": {"count": 0, "running": False, "lastStartedAt": None,
                      "lastFinishedAt": None, "lastError": None,
                      "lastErrorAt": None, "lastStage": None},
            "ports": {"public": self.cfg["port"], "backend": None},
            "paths": {"state": str(self.state), "data": self.cfg["data"]},
            "health": {"ok": False, "checkedAt": None, "error": "not checked"},
            "supervisor": {"pid": os.getpid(), "token": self.token,
                           "startedAt": d.now_iso(),
                           "label": self.cfg.get("label") or ""},
        }

    def update(self, **fields) -> dict:
        with self.lock:
            self.status.update(fields)
            d.write_json_atomic(state_status_path(self.state), self.status)
            return dict(self.status)

    def update_nested(self, key: str, **fields) -> dict:
        with self.lock:
            value = dict(self.status.get(key) or {})
            value.update(fields)
            self.status[key] = value
            d.write_json_atomic(state_status_path(self.state), self.status)
            return dict(self.status)

    def status_snapshot(self) -> dict:
        with self.lock:
            value = json.loads(json.dumps(self.status))
        if self.backend_pid:
            value["health"] = dict(value.get("health") or {})
            value["health"]["processAlive"] = d.pid_alive(self.backend_pid)
        return value


def choose_backend_port() -> int:
    return d.free_port()


def start_backend(runtime: PreviewRuntime, expected_version: str | None = None,
                  timeout: int = 25) -> None:
    cfg = runtime.cfg
    current = read_current(runtime.state)
    if current is None:
        raise PreviewError("no published generation to serve")
    # Serve through the current symlink so a frontend-only or workers-only
    # generation swap reaches the running process without a restart. The
    # binary is resolved at exec time; a backend change restarts it.
    link = current_link(runtime.state)
    binary = link / "bin" / "sifty"
    dist = link / "dist"
    workers = link / "workers"
    if not binary.is_file():
        raise PreviewError("generation has no backend binary: %s" % binary)
    port = choose_backend_port()
    public = int(cfg["port"])
    log = runtime.state / "backend.log"
    command = [str(binary), "serve", "--data-dir", cfg["data"],
               "--dist", str(dist), "--workers-dir", str(workers),
               "--port", str(port), "--public-port", str(public)]
    log.parent.mkdir(parents=True, exist_ok=True)
    log_file = open(log, "a", encoding="utf-8")
    log_file.write("\n%s start: %s\n" % (d.now_iso(), " ".join(command)))
    log_file.flush()
    child = subprocess.Popen(command, stdout=log_file, stderr=subprocess.STDOUT,
                             start_new_session=True, cwd=str(cfg["worktree"]))
    runtime.backend_pid = child.pid
    runtime.backend_port = port
    d.write_json_atomic(runtime.state / "backend.pid", {
        "pid": child.pid, "started": d.process_start(child.pid),
        "startedAt": d.now_iso(), "command": " ".join(command),
    })
    runtime.update(ports={"public": public, "backend": port})
    deadline = time.time() + timeout
    last = "timeout"
    public_host = "127.0.0.1:%d" % public
    while time.time() < deadline:
        if not d.pid_alive(child.pid):
            last = "server exited: " + d.tail(log, 2000)
            break
        code, body = d.http_json("http://127.0.0.1:%d/api/version" % port, 1.0,
                                 host=public_host)
        if code == 200 and isinstance(body, dict):
            if expected_version is None or body.get("version") == expected_version:
                hcode, _ = d.http_json("http://127.0.0.1:%d/api/health" % port, 1.0,
                                       host=public_host)
                if hcode == 200:
                    runtime.update_nested("health", ok=True, checkedAt=d.now_iso(),
                                          error=None)
                    return
                last = "health returned %s" % hcode
            else:
                last = "version %r != %r" % (body.get("version"), expected_version)
        time.sleep(0.15)
    stop_backend(runtime)
    raise PreviewError("backend did not become healthy: %s; log: %s" %
                       (last, d.tail(log, 2000)))


def stop_backend(runtime: PreviewRuntime) -> bool:
    info = d.read_json(runtime.state / "backend.pid")
    if not info:
        runtime.backend_pid = None
        runtime.backend_port = None
        return False
    pid = int(info.get("pid", 0))
    if pid > 0 and d.pid_alive(pid) and (not info.get("started") or
                                         d.process_start(pid) == info.get("started")):
        try:
            os.kill(pid, signal.SIGTERM)
        except OSError:
            pass
        deadline = time.time() + 10
        while time.time() < deadline and d.pid_alive(pid):
            time.sleep(0.1)
        if d.pid_alive(pid) and d.process_start(pid) == info.get("started"):
            try:
                os.kill(pid, signal.SIGKILL)
            except OSError:
                pass
    try:
        (runtime.state / "backend.pid").unlink()
    except FileNotFoundError:
        pass
    runtime.backend_pid = None
    runtime.backend_port = None
    runtime.update(ports={"public": runtime.cfg["port"], "backend": None})
    return True


def check_health(runtime: PreviewRuntime) -> None:
    if not runtime.backend_pid or not runtime.backend_port:
        return
    if not d.pid_alive(runtime.backend_pid):
        runtime.update_nested("health", ok=False, checkedAt=d.now_iso(),
                              error="backend process is not running")
        return
    host = "127.0.0.1:%d" % runtime.cfg["port"]
    code, _ = d.http_json("http://127.0.0.1:%d/api/health" % runtime.backend_port,
                          1.0, host=host)
    runtime.update_nested("health", ok=(code == 200), checkedAt=d.now_iso(),
                          error=None if code == 200 else "health returned %s" % code)


def _inject(html: bytes) -> bytes:
    marker = b'<script src="/__preview/client.js" defer></script>'
    if marker in html:
        return html
    lowered = html.lower()
    idx = lowered.rfind(b"</body>")
    if idx >= 0:
        return html[:idx] + marker + html[idx:]
    return html + marker


class PreviewHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        if getattr(self.server, "log_path", None):
            try:
                with open(self.server.log_path, "a", encoding="utf-8") as fh:
                    fh.write("%s %s\n" % (d.now_iso(), fmt % args))
            except OSError:
                pass

    def do_GET(self):
        self._handle()

    def do_HEAD(self):
        self._handle()

    def do_POST(self):
        self._handle()

    def do_PUT(self):
        self._handle()

    def do_PATCH(self):
        self._handle()

    def do_DELETE(self):
        self._handle()

    def _read_body(self) -> bytes:
        length = self.headers.get("Content-Length")
        if length:
            try:
                return self.rfile.read(int(length))
            except ValueError:
                return b""
        if self.headers.get("Transfer-Encoding", "").lower() == "chunked":
            chunks = []
            while True:
                line = self.rfile.readline().strip()
                size = int(line.split(b";", 1)[0], 16)
                if size == 0:
                    self.rfile.readline()
                    break
                chunks.append(self.rfile.read(size))
                self.rfile.read(2)
            return b"".join(chunks)
        return b""

    def _send(self, status: int, body: bytes, content_type: str,
              extra: dict | None = None) -> None:
        self.send_response_only(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        for key, value in (extra or {}).items():
            self.send_header(key, value)
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(body)

    def _handle(self):
        runtime: PreviewRuntime = self.server.runtime
        path = self.path.split("?", 1)[0]
        if path == "/__preview/status":
            body = json.dumps(runtime.status_snapshot(), indent=2).encode("utf-8") + b"\n"
            self._send(200, body, "application/json; charset=utf-8")
            return
        if path == "/__preview/client.js":
            try:
                body = CLIENT_PATH.read_bytes()
            except OSError:
                body = b"console.warn('sifty preview: client script missing');\n"
            self._send(200, body, "application/javascript; charset=utf-8")
            return
        if path.startswith("/__preview/"):
            self._send(404, b"not found\n", "text/plain; charset=utf-8")
            return
        if not runtime.backend_port:
            screen = ("<html><body><h1>Sifty preview is building or unavailable</h1>"
                      "<p>No healthy backend generation is being served yet. "
                      "Check /__preview/status for the build error.</p></body></html>").encode()
            self._send(503, _inject(screen), "text/html; charset=utf-8")
            return
        body = self._read_body()
        headers = {}
        for key, value in self.headers.items():
            if key.lower() in ("connection", "proxy-connection", "keep-alive",
                               "transfer-encoding", "content-length"):
                continue
            headers[key] = value
        if body:
            headers["Content-Length"] = str(len(body))
        try:
            conn = http.client.HTTPConnection("127.0.0.1", runtime.backend_port,
                                             timeout=30)
            conn.request(self.command, self.path, body=body if body else None,
                         headers=headers)
            response = conn.getresponse()
            data = response.read()
            response_headers = {}
            for key, value in response.getheaders():
                if key.lower() in ("connection", "transfer-encoding",
                                   "content-length", "server", "date"):
                    continue
                response_headers[key] = value
            conn.close()
            content_type = response_headers.pop("Content-Type",
                                                "application/octet-stream")
            if (self.command != "HEAD" and "text/html" in content_type.lower()
                    and not response_headers.get("Content-Encoding")):
                data = _inject(data)
            self._send(response.status, data, content_type, response_headers)
        except Exception as exc:
            message = ("<html><body><h1>Sifty preview backend unavailable</h1>"
                       "<p>%s</p><p>Check /__preview/status for the current "
                       "generation and last build error.</p></body></html>" %
                       (str(exc).replace("<", "&lt;").replace(">", "&gt;"))).encode()
            self._send(502, _inject(message), "text/html; charset=utf-8")


class PreviewServer(ThreadingHTTPServer):
    daemon_threads = True
    allow_reuse_address = True


def start_proxy(runtime: PreviewRuntime) -> None:
    try:
        server = PreviewServer(("127.0.0.1", int(runtime.cfg["port"])), PreviewHandler)
    except OSError as exc:
        raise PreviewError("preview port %s is unavailable: %s" %
                           (runtime.cfg["port"], exc)) from exc
    server.runtime = runtime
    server.log_path = runtime.state / "proxy.log"
    runtime.proxy = server
    runtime.proxy_thread = threading.Thread(target=server.serve_forever,
                                            kwargs={"poll_interval": 0.2},
                                            daemon=True)
    runtime.proxy_thread.start()


def stop_proxy(runtime: PreviewRuntime) -> None:
    if runtime.proxy:
        runtime.proxy.shutdown()
        runtime.proxy.server_close()
        runtime.proxy = None


def _build_once(runtime: PreviewRuntime, signature: dict, git: dict,
                start_server: bool = True) -> bool:
    state = runtime.state
    count = int(runtime.status.get("build", {}).get("count", 0)) + 1
    generation = "g%04d-%s" % (count, signature["all"][:10])
    runtime.update_nested("build", count=count, running=True,
                          lastStartedAt=d.now_iso(), lastStage="build")
    runtime.update(state="building")
    try:
        meta = build_generation(
            Path(runtime.cfg["worktree"]), state, generation, signature, git,
            Path(runtime.cfg["workers_source"]), state / "build.log",
            Path(runtime.cfg["workers_cache"]), Path(runtime.cfg["frontend_cache"]))
        if meta.get("unstable"):
            next_sig = meta["signature"]
            runtime.update_nested("build", running=False,
                                  lastFinishedAt=d.now_iso(),
                                  lastError="source changed during build; retrying",
                                  lastErrorAt=d.now_iso(), lastStage="unstable")
            runtime.update(requested={
                "commit": git["commit"], "dirty": git["dirty"],
                "fingerprint": next_sig["all"], "components": next_sig,
                "at": d.now_iso()})
            return False
        previous = read_current(state)
        needs_restart = (previous is None or
                         meta["changed"].get("backend", False) or
                         runtime.backend_pid is None)
        if needs_restart and not start_server:
            swap_current(state, generation)
            runtime.update(state="built", served={
                "commit": meta["commit"], "dirty": meta["dirty"],
                "fingerprint": meta["signatures"]["all"],
                "generation": generation,
                "components": meta["components"],
                "servedAt": d.now_iso(), "backendPid": None,
            }, commit=meta["commit"], dirty=meta["dirty"])
            runtime.update_nested("build", running=False,
                                  lastFinishedAt=d.now_iso(), lastError=None,
                                  lastErrorAt=None, lastStage="built")
            return True
        if needs_restart:
            old_link = None
            if current_link(state).is_symlink():
                old_link = os.readlink(current_link(state))
            stop_backend(runtime)
            swap_current(state, generation)
            try:
                start_backend(runtime, expected_version=generation)
            except Exception as exc:
                # Restore the previous generation and its server.
                if old_link is None:
                    try:
                        current_link(state).unlink()
                    except FileNotFoundError:
                        pass
                else:
                    tmp = state / (".current.tmp.%d" % os.getpid())
                    try:
                        tmp.unlink()
                    except FileNotFoundError:
                        pass
                    os.symlink(old_link, tmp)
                    os.replace(tmp, current_link(state))
                    try:
                        start_backend(runtime, expected_version=None)
                    except Exception:
                        pass
                raise PreviewBuildError("start", str(exc)) from exc
        else:
            swap_current(state, generation)
        served = {
            "commit": meta["commit"], "dirty": meta["dirty"],
            "fingerprint": meta["signatures"]["all"],
            "generation": generation,
            "components": meta["components"],
            "servedAt": d.now_iso(), "backendPid": runtime.backend_pid,
        }
        runtime.update(state="running", served=served, commit=meta["commit"],
                       dirty=meta["dirty"])
        runtime.update_nested("build", running=False, lastFinishedAt=d.now_iso(),
                              lastError=None, lastErrorAt=None, lastStage="done")
        runtime.update_nested("health", ok=True, checkedAt=d.now_iso(), error=None)
        return True
    except PreviewBuildError as exc:
        runtime.update(state="failed")
        runtime.update_nested("build", running=False, lastFinishedAt=d.now_iso(),
                              lastError=str(exc), lastErrorAt=d.now_iso(),
                              lastStage=exc.stage)
        return False
    except Exception as exc:
        runtime.update(state="failed")
        runtime.update_nested("build", running=False, lastFinishedAt=d.now_iso(),
                              lastError=str(exc), lastErrorAt=d.now_iso(),
                              lastStage="unknown")
        return False


def supervisor_run(cfg: dict, lock_fd: int) -> int:
    state = Path(cfg["state"])
    state.mkdir(parents=True, exist_ok=True)
    lock_file = os.fdopen(lock_fd, "a+")
    runtime = PreviewRuntime(cfg)
    d.write_json_atomic(owner_path(state), {
        "schema": SCHEMA, "pid": os.getpid(), "token": cfg["token"],
        "startedAt": d.now_iso(), "processStart": d.process_start(os.getpid()),
        "worktree": cfg["worktree"], "port": cfg["port"],
        "label": cfg.get("label") or "",
    })
    try:
        start_proxy(runtime)
    except Exception as exc:
        runtime.update(state="failed")
        runtime.update_nested("build", lastError=str(exc), lastErrorAt=d.now_iso(),
                              lastStage="proxy")
        return 1

    def _stop(signum, frame):
        runtime.stop_event.set()

    signal.signal(signal.SIGTERM, _stop)
    signal.signal(signal.SIGINT, _stop)
    write_status(state, runtime.status)
    last_seen = None
    change_at = None
    last_attempted = None
    last_health = 0.0
    try:
        while not runtime.stop_event.is_set():
            signature = scan_sources(Path(cfg["worktree"]),
                                     Path(cfg["workers_source"]))
            git = git_info(Path(cfg["worktree"]))
            now = time.time()
            if signature["all"] != last_seen:
                last_seen = signature["all"]
                change_at = now
                runtime.update(requested={
                    "commit": git["commit"], "dirty": git["dirty"],
                    "fingerprint": signature["all"],
                    "components": signature, "at": d.now_iso()},
                    commit=git["commit"], dirty=git["dirty"])
            if (change_at is not None and now - change_at >= float(cfg["debounce"])
                    and signature["all"] != last_attempted):
                _build_once(runtime, signature, git)
                last_attempted = signature["all"]
                after = scan_sources(Path(cfg["worktree"]),
                                     Path(cfg["workers_source"]))
                if after["all"] != signature["all"]:
                    last_seen = after["all"]
                    change_at = time.time()
                    runtime.update(requested={
                        "commit": git_info(Path(cfg["worktree"]))["commit"],
                        "dirty": True, "fingerprint": after["all"],
                        "components": after, "at": d.now_iso()})
                else:
                    change_at = None
            if now - last_health >= 2.0:
                check_health(runtime)
                last_health = now
            runtime.stop_event.wait(float(cfg["poll"]))
    finally:
        stop_backend(runtime)
        stop_proxy(runtime)
        runtime.update(state="stopped")
        try:
            lock_file.close()
        except Exception:
            pass
        try:
            owner_path(state).unlink()
        except FileNotFoundError:
            pass
        return 0


def _load_status(state: Path) -> dict | None:
    return read_status(state)


def status_text(status: dict, running: bool) -> str:
    lines = []
    lines.append("Preview: %s" % ("running" if running else "not running"))
    lines.append("Worktree: %s" % status.get("worktree", "?"))
    supervisor = status.get("supervisor") or {}
    lines.append("Owner: pid %s token %s label %s" %
                 (supervisor.get("pid", "?"), supervisor.get("token", "?"),
                  supervisor.get("label", "") or "-"))
    requested = status.get("requested") or {}
    served = status.get("served") or {}
    lines.append("Requested: %s%s fp=%s" %
                 ((requested.get("commit") or "?")[:12],
                  " dirty" if requested.get("dirty") else "",
                  (requested.get("fingerprint") or "?")[:12]))
    if served:
        lines.append("Served:    %s%s fp=%s generation=%s" %
                     ((served.get("commit") or "?")[:12],
                      " dirty" if served.get("dirty") else "",
                      (served.get("fingerprint") or "?")[:12],
                      served.get("generation", "?")))
        lines.append("Components: %s" %
                     ", ".join("%s=%s" % (k, v)
                               for k, v in sorted((served.get("components") or {}).items())))
    else:
        lines.append("Served:    none")
    build = status.get("build") or {}
    lines.append("Builds: %s; running=%s; state=%s" %
                 (build.get("count", 0), build.get("running", False),
                  status.get("state", "?")))
    if build.get("lastError"):
        lines.append("Last error (%s): %s" %
                     (build.get("lastStage", "?"), build.get("lastError")))
    health = status.get("health") or {}
    lines.append("Health: %s%s" %
                 ("ok" if health.get("ok") else "not ok",
                  "" if not health.get("error") else " (%s)" % health["error"]))
    ports = status.get("ports") or {}
    lines.append("Ports: public=%s backend=%s" %
                 (ports.get("public", "?"), ports.get("backend", "?")))
    return "\n".join(lines)


def _owner_alive(owner: dict | None) -> bool:
    if not owner:
        return False
    pid = int(owner.get("pid", 0) or 0)
    if pid <= 0 or not d.pid_alive(pid):
        return False
    if owner.get("processStart") and d.process_start(pid) != owner["processStart"]:
        return False
    command = d.process_command(pid)
    return "sift-preview" in command or "_preview_lib" in command


def _resolve_preview_cfg(script_path: str, args) -> dict:
    app_home, main_repo = d.repo_roots(script_path)
    cfg = d.load_config(app_home)
    staging = d.environment_config("staging", app_home, main_repo, cfg)
    worktree = getattr(args, "worktree", None)
    if worktree:
        worktree = Path(worktree).resolve()
    else:
        cwd = Path.cwd()
        worktree = Path(d.git(cwd, "rev-parse", "--show-toplevel")).resolve()
    if not (worktree / "frontend").is_dir() or not (worktree / "backend").is_dir():
        raise PreviewError("%s does not look like a sifty checkout" % worktree)
    port = getattr(args, "port", None)
    if port is None:
        port = int(cfg.get("SIFTY_PREVIEW_PORT", staging["port"]))
    if not 1 <= int(port) <= 65535:
        raise PreviewError("preview port must be between 1 and 65535")
    data = getattr(args, "data_dir", None) or cfg.get("SIFTY_PREVIEW_DATA_DIR") or str(staging["data"])
    state = getattr(args, "state_dir", None) or cfg.get("SIFTY_PREVIEW_STATE_DIR") or str(main_repo / ".local" / "preview")
    workers_source = d.resolve_path(
        cfg.get("SIFTY_WORKERS_SOURCE", str(app_home / "backend" / "workers")), app_home)
    return {
        "worktree": str(worktree), "port": int(port),
        "data": str(d.resolve_path(data, main_repo)),
        "state": str(d.resolve_path(state, main_repo)),
        "poll": float(getattr(args, "poll", None) or
                      cfg.get("SIFTY_PREVIEW_POLL_SECONDS", "1.0")),
        "debounce": float(getattr(args, "debounce", None) or
                          cfg.get("SIFTY_PREVIEW_DEBOUNCE_SECONDS", "1.0")),
        "workers_source": str(workers_source),
        "workers_cache": str(d.resolve_path(
            cfg.get("SIFTY_WORKERS_CACHE_DIR", str(main_repo / ".local" / "cache" / "workers")),
            main_repo)),
        "frontend_cache": str(d.resolve_path(
            cfg.get("SIFTY_FRONTEND_CACHE_DIR", str(main_repo / ".local" / "cache" / "frontend")),
            main_repo)),
        "label": getattr(args, "label", None) or "",
        "token": hashlib.sha256(("%s-%s-%s" % (os.getpid(), time.time(), worktree)).encode()).hexdigest()[:32],
    }


def _state_dir_from_args(script_path: str, args) -> Path:
    app_home, main_repo = d.repo_roots(script_path)
    cfg = d.load_config(app_home)
    value = getattr(args, "state_dir", None) or cfg.get("SIFTY_PREVIEW_STATE_DIR")
    if not value:
        value = str(main_repo / ".local" / "preview")
    return d.resolve_path(value, main_repo)


def start_main(script_path: str, argv: list[str]) -> int:
    parser = argparse.ArgumentParser(prog="sift-preview start")
    parser.add_argument("--worktree", default=None)
    parser.add_argument("--label", default=None)
    parser.add_argument("--port", type=int, default=None)
    parser.add_argument("--data-dir", default=None)
    parser.add_argument("--state-dir", default=None)
    parser.add_argument("--poll", type=float, default=None)
    parser.add_argument("--debounce", type=float, default=None)
    parser.add_argument("--foreground", action="store_true")
    parser.add_argument("--wait", type=float, default=120.0)
    args = parser.parse_args(argv)
    cfg = _resolve_preview_cfg(script_path, args)
    state = Path(cfg["state"])
    state.mkdir(parents=True, exist_ok=True)
    fd = os.open(lock_path(state), os.O_CREAT | os.O_RDWR, 0o600)
    try:
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        owner = d.read_json(owner_path(state))
        existing = status_text(read_status(state) or {}, _owner_alive(owner))
        print("sift-preview: a preview owner is already active; refusing to replace it.",
              file=sys.stderr)
        print(existing, file=sys.stderr)
        os.close(fd)
        return 3
    if args.foreground:
        return supervisor_run(cfg, fd)
    log_path = state / "supervisor.log"
    log_file = open(log_path, "a", encoding="utf-8")
    command = [sys.executable, str(Path(script_path).resolve()), "_run",
               "--lock-fd", str(fd), "--worktree", cfg["worktree"],
               "--port", str(cfg["port"]), "--data-dir", cfg["data"],
               "--state-dir", cfg["state"], "--poll", str(cfg["poll"]),
               "--debounce", str(cfg["debounce"]),
               "--workers-source", cfg["workers_source"],
               "--workers-cache", cfg["workers_cache"],
               "--frontend-cache", cfg["frontend_cache"],
               "--label", cfg["label"], "--token", cfg["token"]]
    child = subprocess.Popen(command, stdout=log_file, stderr=subprocess.STDOUT,
                             start_new_session=True, pass_fds=(fd,),
                             cwd=cfg["worktree"])
    log_file.close()
    os.close(fd)
    deadline = time.time() + max(1.0, args.wait)
    last = None
    while time.time() < deadline:
        if child.poll() is not None:
            print("sift-preview: supervisor exited (%s); %s" %
                  (child.returncode, d.tail(log_path, 3000)), file=sys.stderr)
            return 1
        status = read_status(state)
        if status and (status.get("supervisor") or {}).get("pid") == child.pid:
            state_name = status.get("state")
            count = int((status.get("build") or {}).get("count", 0))
            if state_name == "running":
                print("sift-preview: %s preview of %s on http://localhost:%s" %
                      ("running", cfg["worktree"], cfg["port"]))
                print(status_text(status, True))
                return 0
            if state_name == "failed" and count:
                print("sift-preview: preview is running but the first build failed.",
                      file=sys.stderr)
                print(status_text(status, True), file=sys.stderr)
                return 1
        last = status
        time.sleep(0.1)
    print("sift-preview: preview did not become ready; %s" %
          status_text(last or {}, _owner_alive(d.read_json(owner_path(state)))),
          file=sys.stderr)
    return 1


def status_main(script_path: str, argv: list[str]) -> int:
    parser = argparse.ArgumentParser(prog="sift-preview status")
    parser.add_argument("--state-dir", default=None)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)
    state = _state_dir_from_args(script_path, args)
    status = read_status(state)
    owner = d.read_json(owner_path(state))
    running = _owner_alive(owner)
    if args.json:
        print(json.dumps({"running": running, "status": status}, indent=2, sort_keys=True))
        return 0 if running else 3
    if not status:
        print("Preview: not running (no status at %s)" % state_status_path(state))
        return 3
    print(status_text(status, running))
    return 0 if running else 3


def stop_main(script_path: str, argv: list[str]) -> int:
    parser = argparse.ArgumentParser(prog="sift-preview stop")
    parser.add_argument("--state-dir", default=None)
    args = parser.parse_args(argv)
    state = _state_dir_from_args(script_path, args)
    owner = d.read_json(owner_path(state))
    if not owner:
        status = read_status(state) or {}
        if status.get("state") == "running":
            print("sift-preview: no owner record; refusing to guess which process to stop.",
                  file=sys.stderr)
            return 1
        print("Preview is not running.")
        return 0
    if not _owner_alive(owner):
        print("sift-preview: recorded owner pid %s is not a live preview supervisor; "
              "refusing to kill it." % owner.get("pid"), file=sys.stderr)
        return 1
    pid = int(owner["pid"])
    try:
        os.kill(pid, signal.SIGTERM)
    except OSError as exc:
        print("sift-preview: could not stop pid %s: %s" % (pid, exc), file=sys.stderr)
        return 1
    deadline = time.time() + 25
    while time.time() < deadline and d.pid_alive(pid):
        time.sleep(0.1)
    if d.pid_alive(pid):
        if d.process_command(pid) and "sift-preview" in d.process_command(pid):
            try:
                os.kill(pid, signal.SIGKILL)
            except OSError:
                pass
        time.sleep(0.2)
    status = read_status(state) or {}
    status["state"] = "stopped"
    status["stoppedAt"] = d.now_iso()
    write_status(state, status)
    print("Preview stopped.")
    return 0


def once_main(script_path: str, argv: list[str]) -> int:
    parser = argparse.ArgumentParser(prog="sift-preview once")
    parser.add_argument("--worktree", default=None)
    parser.add_argument("--label", default=None)
    parser.add_argument("--port", type=int, default=None)
    parser.add_argument("--data-dir", default=None)
    parser.add_argument("--state-dir", default=None)
    parser.add_argument("--poll", type=float, default=None)
    parser.add_argument("--debounce", type=float, default=None)
    parser.add_argument("--no-serve", action="store_true")
    args = parser.parse_args(argv)
    cfg = _resolve_preview_cfg(script_path, args)
    state = Path(cfg["state"])
    state.mkdir(parents=True, exist_ok=True)
    fd = os.open(lock_path(state), os.O_CREAT | os.O_RDWR, 0o600)
    try:
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        print("sift-preview: a preview owner is active; stop it before running once.",
              file=sys.stderr)
        os.close(fd)
        return 3
    runtime = PreviewRuntime(cfg)
    try:
        signature = scan_sources(Path(cfg["worktree"]), Path(cfg["workers_source"]))
        git = git_info(Path(cfg["worktree"]))
        runtime.update(state="building", requested={
            "commit": git["commit"], "dirty": git["dirty"],
            "fingerprint": signature["all"], "components": signature,
            "at": d.now_iso()})
        ok = _build_once(runtime, signature, git,
                         start_server=not args.no_serve)
        if ok and not args.no_serve:
            stop_backend(runtime)
            runtime.update(state="built")
        print(status_text(read_status(state) or {}, False))
        return 0 if ok else 1
    finally:
        try:
            stop_backend(runtime)
        except Exception:
            pass
        try:
            owner_path(state).unlink()
        except FileNotFoundError:
            pass
        os.close(fd)


def run_main(script_path: str, argv: list[str]) -> int:
    parser = argparse.ArgumentParser(prog="sift-preview _run")
    parser.add_argument("--lock-fd", type=int, required=True)
    parser.add_argument("--worktree", required=True)
    parser.add_argument("--port", type=int, required=True)
    parser.add_argument("--data-dir", required=True)
    parser.add_argument("--state-dir", required=True)
    parser.add_argument("--poll", type=float, default=1.0)
    parser.add_argument("--debounce", type=float, default=1.0)
    parser.add_argument("--workers-source", required=True)
    parser.add_argument("--workers-cache", required=True)
    parser.add_argument("--frontend-cache", required=True)
    parser.add_argument("--label", default="")
    parser.add_argument("--token", required=True)
    args = parser.parse_args(argv)
    cfg = {
        "worktree": args.worktree, "port": args.port, "data": args.data_dir,
        "state": args.state_dir, "poll": args.poll, "debounce": args.debounce,
        "workers_source": args.workers_source,
        "workers_cache": args.workers_cache,
        "frontend_cache": args.frontend_cache,
        "label": args.label, "token": args.token,
    }
    return supervisor_run(cfg, args.lock_fd)
