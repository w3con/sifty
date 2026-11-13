#!/usr/bin/env python3
"""Shared implementation for the sifty deploy, data-refresh and preview commands.

This module is deliberately project-local: the generic work-* commands never
import it. It resolves per-environment destinations in the main checkout, so
artifacts built from a linked worktree survive that worktree's removal.

The important invariants enforced here:

* production and staging never share a binary, frontend bundle, workers
  directory or data directory;
* every build goes into a fresh candidate directory and is only published
  after it succeeds;
* a deploy stops only a server recorded in its own pid file (never a process
  found merely by port);
* normal deploys never refresh staging data; the explicit refresh operation
  uses a consistent SQLite backup and preserves the previous data tree;
* rollback restores the binary, frontend bundle, worker tree and a consistent
  SQLite database (removing stale WAL/SHM files first).
"""
from __future__ import annotations

import atexit
import datetime as _dt
import fcntl
import hashlib
import json
import os
import shlex
import shutil
import signal
import socket
import sqlite3
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

SCHEMA = 1
DEFAULT_PORTS = {"prod": 8080, "staging": 8081}
ENV_PREFIX = {"prod": "SIFTY_PROD", "staging": "SIFTY_STAGING"}
RUNTIME_FILE_NAMES = {"store.sqlite", "deploy.log", "prod-server.log", "staging-server.log", "server.pid"}
DATA_DIRS = ("config", "evidence", "answers", "feedback")
DATA_FILES = ("cv.pdf", "profile.pdf")
IGNORE_NAMES = {"node_modules", ".git", "dist", "dist-staging", "dist-preview", "build",
                ".vite", "coverage", ".cache", "__pycache__", "workers-prod", "workers-staging"}


class DeployError(RuntimeError):
    """A recoverable command failure with a user-facing message."""


def now_stamp() -> str:
    return _dt.datetime.now(_dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")


def unique_stamp() -> str:
    return _dt.datetime.now(_dt.timezone.utc).strftime("%Y%m%dT%H%M%S.%fZ")


def now_iso() -> str:
    return _dt.datetime.now(_dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def git(app_home: Path, *args: str) -> str:
    result = subprocess.run(["git", "-C", str(app_home), *args], text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if result.returncode != 0:
        raise DeployError("git %s failed: %s" % (" ".join(args), result.stderr.strip()))
    return result.stdout.strip()


def repo_roots(script_path: str | os.PathLike) -> tuple[Path, Path]:
    """Return (source checkout, main checkout) for a command in bin/."""
    script = Path(script_path).resolve()
    app_home = Path(git(script.parent.parent, "rev-parse", "--show-toplevel"))
    common = git(script.parent.parent, "rev-parse", "--git-common-dir")
    if common == ".git":
        main_repo = app_home
    else:
        main_repo = (app_home / common).resolve().parent
    return app_home, main_repo


def load_config(app_home: Path) -> dict[str, str]:
    """Load bin/work.config; an environment variable of the same name wins."""
    values: dict[str, str] = {}
    path = app_home / "bin" / "work.config"
    if path.is_file():
        for raw in path.read_text(encoding="utf-8").splitlines():
            line = raw.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            key = key.strip()
            if key:
                values[key] = value.strip()
    for key in list(values):
        if key in os.environ:
            values[key] = os.environ[key]
    # Also include environment-only keys used by this module.
    for key in ("SIFTY_PROD_PORT", "SIFTY_STAGING_PORT", "SIFTY_PROD_DATA_DIR",
                "SIFTY_STAGING_DATA_DIR", "SIFTY_PROD_BINARY", "SIFTY_STAGING_BINARY",
                "SIFTY_PROD_DIST", "SIFTY_STAGING_DIST", "SIFTY_PROD_WORKERS_DIR",
                "SIFTY_STAGING_WORKERS_DIR", "SIFTY_PROD_LOG", "SIFTY_STAGING_LOG",
                "SIFTY_WORKERS_SOURCE", "SIFTY_DEPLOY_STATE_DIR",
                "SIFTY_WORKERS_CACHE_DIR", "SIFTY_FRONTEND_CACHE_DIR",
                "SIFTY_PREVIEW_PORT", "SIFTY_PREVIEW_DATA_DIR", "SIFTY_PREVIEW_STATE_DIR",
                "SIFTY_PREVIEW_POLL_SECONDS", "SIFTY_PREVIEW_DEBOUNCE_SECONDS"):
        if key in os.environ:
            values[key] = os.environ[key]
    return values


def setting(cfg: dict[str, str], key: str, default: str | None = None) -> str | None:
    return cfg.get(key, default)


def resolve_path(value: str | os.PathLike, base: Path) -> Path:
    path = Path(os.path.expanduser(str(value)))
    if not path.is_absolute():
        path = base / path
    return path.resolve()


def _within(child: Path, parent: Path) -> bool:
    try:
        child.relative_to(parent)
        return True
    except ValueError:
        return False


def paths_overlap(a: Path, b: Path) -> bool:
    a, b = a.resolve(), b.resolve()
    return a == b or _within(a, b) or _within(b, a)


def environment_config(name: str, app_home: Path, main_repo: Path,
                       cfg: dict[str, str]) -> dict:
    if name not in DEFAULT_PORTS:
        raise DeployError("unknown environment %r" % name)
    prefix = ENV_PREFIX[name]
    is_prod = name == "prod"
    data_default = "data" if is_prod else "data-staging"
    binary_default = "bin/sifty" if is_prod else "bin/sifty-staging"
    dist_default = "frontend/dist" if is_prod else "frontend/dist-staging"
    workers_default = ("backend/workers-prod" if is_prod
                       else "backend/workers-staging")
    try:
        port = int(setting(cfg, prefix + "_PORT", str(DEFAULT_PORTS[name])))
    except ValueError as exc:
        raise DeployError("%s_PORT is not a number" % prefix) from exc
    if port < 1 or port > 65535:
        raise DeployError("%s_PORT is outside 1..65535" % prefix)
    data = resolve_path(setting(cfg, prefix + "_DATA_DIR", data_default), main_repo)
    binary = resolve_path(setting(cfg, prefix + "_BINARY", binary_default), main_repo)
    dist = resolve_path(setting(cfg, prefix + "_DIST", dist_default), main_repo)
    workers = resolve_path(setting(cfg, prefix + "_WORKERS_DIR", workers_default), main_repo)
    log_default = str(data / ("prod-server.log" if is_prod else "staging-server.log"))
    server_log = resolve_path(setting(cfg, prefix + "_LOG", log_default), main_repo)
    return {
        "name": name, "port": port, "data": data, "binary": binary,
        "dist": dist, "workers": workers, "server_log": server_log,
    }


def shared_config(app_home: Path, main_repo: Path, cfg: dict[str, str]) -> dict:
    if "SIFTY_WORKERS_DIR" in cfg or "SIFTY_WORKERS_DIR" in os.environ:
        raise DeployError(
            "SIFTY_WORKERS_DIR is no longer supported because it shared worker "
            "artifacts between environments; set SIFTY_PROD_WORKERS_DIR and "
            "SIFTY_STAGING_WORKERS_DIR instead")
    workers_source = resolve_path(
        setting(cfg, "SIFTY_WORKERS_SOURCE", str(app_home / "backend" / "workers")),
        app_home)
    state_root = resolve_path(
        setting(cfg, "SIFTY_DEPLOY_STATE_DIR", str(main_repo / ".local" / "deploy")),
        main_repo)
    cache_root = resolve_path(
        setting(cfg, "SIFTY_WORKERS_CACHE_DIR", str(main_repo / ".local" / "cache" / "workers")),
        main_repo)
    frontend_cache = resolve_path(
        setting(cfg, "SIFTY_FRONTEND_CACHE_DIR", str(main_repo / ".local" / "cache" / "frontend")),
        main_repo)
    return {
        "app_home": app_home, "main_repo": main_repo,
        "workers_source": workers_source, "state_root": state_root,
        "workers_cache": cache_root, "frontend_cache": frontend_cache,
    }


def validate_environment_paths(prod: dict, staging: dict, shared: dict) -> None:
    """Refuse overlapping destinations instead of trusting environment names."""
    prod_paths = {"data": prod["data"], "binary": prod["binary"],
                  "dist": prod["dist"], "workers": prod["workers"]}
    staging_paths = {"data": staging["data"], "binary": staging["binary"],
                     "dist": staging["dist"], "workers": staging["workers"]}
    for pk, pv in prod_paths.items():
        for sk, sv in staging_paths.items():
            if paths_overlap(pv, sv):
                raise DeployError(
                    "production %s (%s) overlaps staging %s (%s); environments "
                    "must use independent destinations" % (pk, pv, sk, sv))
    for env in (prod, staging):
        items = [("binary", env["binary"]), ("dist", env["dist"]),
                 ("workers", env["workers"]), ("data", env["data"])]
        for i, (ka, va) in enumerate(items):
            for kb, vb in items[i + 1:]:
                if paths_overlap(va, vb):
                    raise DeployError(
                        "%s %s (%s) overlaps %s (%s); each artifact needs its "
                        "own path" % (env["name"], ka, va, kb, vb))
        if paths_overlap(env["workers"], shared["workers_source"]):
            raise DeployError(
                "%s workers target (%s) is the shared source workers directory "
                "(%s); deploy must publish a private copy" %
                (env["name"], env["workers"], shared["workers_source"]))
        if paths_overlap(env["data"], shared["state_root"]):
            raise DeployError(
                "%s data directory overlaps deploy state (%s)" %
                (env["name"], shared["state_root"]))


def preview_state_path(main_repo: Path, cfg: dict[str, str]) -> Path:
    value = setting(cfg, "SIFTY_PREVIEW_STATE_DIR",
                    str(main_repo / ".local" / "preview"))
    return resolve_path(value, main_repo)


def acquire_environment_lock(state_root: Path, env: str) -> int:
    """One mutation at a time for an environment (deploy/build-only/refresh)."""
    directory = state_root / env
    directory.mkdir(parents=True, exist_ok=True)
    path = directory / "operation.lock"
    fd = os.open(path, os.O_CREAT | os.O_RDWR, 0o600)
    try:
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        os.close(fd)
        raise DeployError(
            "another deploy, build-only or data-refresh operation is already "
            "running for %s (%s)" % (env, path))
    # The CLI process is short-lived; release explicitly at process exit.
    atexit.register(lambda: os.close(fd))
    return fd


def preview_conflict(env: dict, main_repo: Path, cfg: dict[str, str]) -> None:
    """Refuse to touch an environment currently owned by a live preview."""
    state = preview_state_path(main_repo, cfg)
    owner = read_json(state / "owner.json")
    if not owner:
        return
    pid = int(owner.get("pid", 0) or 0)
    if pid <= 0 or not pid_alive(pid):
        return
    command = process_command(pid)
    if "sift-preview" not in command and "_preview_lib" not in command:
        return
    status = read_json(state / "status.json") or {}
    preview_port = int(owner.get("port") or
                       (status.get("ports") or {}).get("public") or 0)
    preview_data = (status.get("paths") or {}).get("data")
    conflicts = preview_port == env["port"]
    if preview_data and paths_overlap(Path(preview_data), env["data"]):
        conflicts = True
    if conflicts:
        raise DeployError(
            "a continuous preview (pid %d, %s) owns port %s / data %s; "
            "stop it with bin/sift-preview stop before deploying or refreshing "
            "this environment" %
            (pid, state, preview_port or "?", preview_data or "?"))


def validate_refresh_source(prod: dict, staging: dict, shared: dict) -> None:
    """Apply the same overlap rules after a --from data directory override."""
    for key in ("data", "binary", "dist", "workers"):
        if paths_overlap(prod["data"], staging[key]):
            raise DeployError(
                "refresh source %s overlaps staging %s (%s)" %
                (prod["data"], key, staging[key]))
    if paths_overlap(prod["data"], shared["state_root"]):
        raise DeployError(
            "refresh source %s overlaps deploy state %s" %
            (prod["data"], shared["state_root"]))


def all_configs(app_home: Path, main_repo: Path, cfg: dict[str, str]) -> tuple[dict, dict, dict]:
    shared = shared_config(app_home, main_repo, cfg)
    prod = environment_config("prod", app_home, main_repo, cfg)
    staging = environment_config("staging", app_home, main_repo, cfg)
    validate_environment_paths(prod, staging, shared)
    return prod, staging, shared


# ── generic helpers ─────────────────────────────────────────────────────────

def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def tree_digest(root: Path) -> str:
    """Path-and-content digest of a tree (used to compare published artifacts)."""
    root = root.resolve()
    h = hashlib.sha256()
    for dirpath, dirnames, filenames in os.walk(root, followlinks=False):
        dirnames.sort()
        for name in sorted(filenames):
            path = Path(dirpath) / name
            rel = path.relative_to(root).as_posix()
            h.update(rel.encode("utf-8"))
            h.update(b"\0")
            h.update(sha256_file(path).encode("ascii"))
            h.update(b"\n")
    return h.hexdigest()


def lock_digest(paths: list[Path]) -> str:
    h = hashlib.sha256()
    for path in paths:
        h.update(path.name.encode("utf-8"))
        h.update(b"\0")
        h.update(path.read_bytes())
        h.update(b"\0")
    return h.hexdigest()


def write_json_atomic(path: Path, value: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_name(path.name + ".tmp.%d" % os.getpid())
    tmp.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    os.replace(tmp, path)


def read_json(path: Path) -> dict | None:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return None


def tail(path: Path, limit: int = 4000) -> str:
    try:
        data = path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""
    return data[-limit:]


def run_logged(cmd: list[str], cwd: Path, log_path: Path, env: dict | None = None,
               timeout: int | None = None) -> None:
    log_path.parent.mkdir(parents=True, exist_ok=True)
    child_env = os.environ.copy()
    if env:
        child_env.update(env)
    with open(log_path, "a", encoding="utf-8") as fh:
        fh.write("$ %s\n(cwd %s)\n" % (" ".join(cmd), cwd))
        fh.flush()
        try:
            result = subprocess.run(cmd, cwd=str(cwd), env=child_env, text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                    timeout=timeout)
        except FileNotFoundError as exc:
            raise DeployError("required command not found: %s" % cmd[0]) from exc
        except subprocess.TimeoutExpired as exc:
            fh.write("TIMEOUT after %ss\n" % timeout)
            raise DeployError("command timed out: %s" % " ".join(cmd)) from exc
        fh.write(result.stdout or "")
        fh.write("\nexit=%d\n\n" % result.returncode)
    if result.returncode != 0:
        raise DeployError(
            "command failed (%d): %s; last output: %s" %
            (result.returncode, " ".join(cmd), (result.stdout or "")[-1200:]))


def copy_tree(src: Path, dst: Path, ignore=IGNORE_NAMES) -> None:
    if dst.exists():
        shutil.rmtree(dst)
    dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(src, dst, symlinks=True,
                    ignore=shutil.ignore_patterns(*ignore))


def package_has_dependencies(package_json: Path) -> bool:
    try:
        data = json.loads(package_json.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return False
    for key in ("dependencies", "optionalDependencies", "peerDependencies"):
        if data.get(key):
            return True
    return False


def _install_npm_tree(source_dir: Path, cache_dir: Path, log_path: Path,
                      label: str) -> Path:
    """Install a locked npm tree once per lock hash and return node_modules."""
    lock = source_dir / "package-lock.json"
    package = source_dir / "package.json"
    if not lock.is_file():
        raise DeployError("%s has no package-lock.json; locked install refused" % label)
    digest = lock_digest([package, lock])
    target = cache_dir / digest
    modules = target / "node_modules"
    if modules.is_dir():
        return modules
    cache_dir.mkdir(parents=True, exist_ok=True)
    staging = Path(tempfile.mkdtemp(prefix="deps-", dir=str(cache_dir)))
    try:
        shutil.copy2(package, staging / "package.json")
        shutil.copy2(lock, staging / "package-lock.json")
        run_logged(["npm", "ci", "--silent", "--no-audit", "--no-fund"],
                   staging, log_path, env={"CI": "1"})
        os.replace(staging, target)
    except Exception:
        shutil.rmtree(staging, ignore_errors=True)
        raise
    if not modules.is_dir():
        raise DeployError("npm ci for %s produced no node_modules" % label)
    return modules


def ensure_frontend_deps(frontend_dir: Path, log_path: Path) -> None:
    lock = frontend_dir / "package-lock.json"
    package = frontend_dir / "package.json"
    if not lock.is_file():
        raise DeployError("frontend/package-lock.json is required; npm install is not a fallback")
    digest = lock_digest([package, lock])
    marker = frontend_dir / "node_modules" / ".sifty-lock-sha256"
    try:
        if marker.read_text(encoding="utf-8").strip() == digest:
            return
    except OSError:
        pass
    run_logged(["npm", "ci", "--silent", "--no-audit", "--no-fund"],
               frontend_dir, log_path, env={"CI": "1"})
    marker.parent.mkdir(parents=True, exist_ok=True)
    marker.write_text(digest + "\n", encoding="utf-8")


def build_binary(source_backend: Path, dest: Path, build_id: str,
                 log_path: Path) -> None:
    dest.parent.mkdir(parents=True, exist_ok=True)
    if dest.exists():
        dest.unlink()
    ldflags = "-X github.com/w3con/sifty/internal/version.Version=%s" % build_id
    run_logged(["go", "build", "-trimpath", "-ldflags", ldflags,
                "-o", str(dest), "./cmd/sifty"],
               source_backend, log_path, env={"SIFTY_BUILD_ID": build_id})
    if not dest.is_file():
        raise DeployError("go build produced no binary at %s" % dest)
    dest.chmod(0o755)


def build_frontend(source_frontend: Path, dest_dist: Path, log_path: Path,
                   build_id: str, cache_root: Path | None = None) -> None:
    # Cache installs by lockfile content. The source worktree's node_modules is
    # ignored build state, never a published artifact.
    ensure_frontend_deps(source_frontend, log_path)
    if dest_dist.exists():
        shutil.rmtree(dest_dist)
    dest_dist.parent.mkdir(parents=True, exist_ok=True)
    run_logged(["npm", "run", "build"], source_frontend, log_path,
               env={"BUILD_OUTDIR": str(dest_dist), "CI": "1",
                    "SIFTY_BUILD_ID": build_id})
    if not (dest_dist / "index.html").is_file():
        raise DeployError("frontend build produced no index.html at %s" % dest_dist)


def copy_worker_dependencies(source_workers: Path, dest_workers: Path,
                             cache_root: Path, log_path: Path,
                             link: bool = False) -> None:
    if not source_workers.is_dir():
        raise DeployError("workers source directory missing: %s" % source_workers)
    for worker in sorted(p for p in dest_workers.iterdir() if p.is_dir()):
        package = worker / "package.json"
        if not package.is_file():
            continue
        if not package_has_dependencies(package):
            continue
        if not (worker / "package-lock.json").is_file():
            raise DeployError(
                "worker %s declares dependencies without package-lock.json; "
                "locked install refused" % worker.name)
        modules = _install_npm_tree(worker, cache_root / worker.name, log_path,
                                    "worker %s" % worker.name)
        target = worker / "node_modules"
        if link:
            os.symlink(modules, target, target_is_directory=True)
        else:
            shutil.copytree(modules, target, symlinks=True)


def build_workers(source_workers: Path, dest_workers: Path, cache_root: Path,
                  log_path: Path, link: bool = False) -> None:
    copy_tree(source_workers, dest_workers)
    copy_worker_dependencies(source_workers, dest_workers, cache_root, log_path,
                             link=link)


# ── SQLite / process / HTTP helpers ─────────────────────────────────────────

def sqlite_backup(source: Path, destination: Path) -> bool:
    """Consistent SQLite backup, safe while the source process is running."""
    if not source.is_file():
        return False
    destination.parent.mkdir(parents=True, exist_ok=True)
    if destination.exists():
        destination.unlink()
    src = sqlite3.connect(str(source), timeout=10)
    try:
        dst = sqlite3.connect(str(destination))
        try:
            src.backup(dst)
        finally:
            dst.close()
    finally:
        src.close()
    if not destination.is_file():
        raise DeployError("SQLite backup did not create %s" % destination)
    return True


def remove_sqlite_sidecars(path: Path) -> None:
    for suffix in ("-wal", "-shm", "-journal"):
        sidecar = Path(str(path) + suffix)
        try:
            sidecar.unlink()
        except FileNotFoundError:
            pass


def restore_sqlite(source_backup: Path, target: Path, existed: bool) -> None:
    # Validate the required backup before touching the live database or its
    # sidecars. A missing snapshot must never be interpreted as "no database".
    if existed and not source_backup.is_file():
        raise DeployError(
            "required SQLite recovery backup missing at %s; existing database "
            "preserved" % source_backup)
    target.parent.mkdir(parents=True, exist_ok=True)
    remove_sqlite_sidecars(target)
    try:
        target.unlink()
    except FileNotFoundError:
        pass
    if existed:
        shutil.copy2(source_backup, target)


def port_free(port: int) -> bool:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        sock.bind(("127.0.0.1", port))
        return True
    except OSError:
        return False
    finally:
        sock.close()


def free_port() -> int:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])
    finally:
        sock.close()


def pid_alive(pid: int) -> bool:
    try:
        os.kill(pid, 0)
    except OSError:
        return False
    try:
        stat = subprocess.run(["ps", "-p", str(pid), "-o", "stat="],
                              text=True, stdout=subprocess.PIPE,
                              stderr=subprocess.DEVNULL).stdout.strip()
    except OSError:
        return True
    return bool(stat) and not stat.startswith("Z")


def process_command(pid: int) -> str:
    try:
        result = subprocess.run(["ps", "-p", str(pid), "-o", "command="],
                                text=True, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL)
        return result.stdout.strip()
    except OSError:
        return ""


def process_start(pid: int) -> str:
    try:
        result = subprocess.run(["ps", "-p", str(pid), "-o", "lstart="],
                                text=True, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL)
        return result.stdout.strip()
    except OSError:
        return ""


def pid_path(state_root: Path, env: str) -> Path:
    return state_root / env / "server.pid"


def read_pid(state_root: Path, env: str) -> dict | None:
    return read_json(pid_path(state_root, env))


def write_pid(state_root: Path, env: str, info: dict) -> None:
    write_json_atomic(pid_path(state_root, env), info)


def clear_pid(state_root: Path, env: str) -> None:
    try:
        pid_path(state_root, env).unlink()
    except FileNotFoundError:
        pass


def _pid_matches(info: dict, binary: Path | None = None) -> bool:
    pid = int(info.get("pid", 0))
    if pid <= 0 or not pid_alive(pid):
        return False
    started = info.get("started")
    if started and process_start(pid) != started:
        return False
    if binary is not None:
        command = process_command(pid)
        if str(binary) not in command and str(binary.resolve()) not in command:
            return False
    return True


def probe_port_owner(cfg: dict, state_root: Path) -> dict | None:
    """Return the recorded server if the environment's port is owned by it.

    A busy port with no matching pid file is an unrelated process and must be
    refused, never killed.
    """
    info = read_pid(state_root, cfg["name"])
    if info:
        if not _pid_matches(info, cfg["binary"]):
            raise DeployError(
                "recorded %s server pid %s does not match %s; refusing to kill it"
                % (cfg["name"], info.get("pid"), cfg["binary"]))
        return info
    if not port_free(cfg["port"]):
        raise DeployError(
            "%s port %d is in use by an unowned process; refusing to kill it" %
            (cfg["name"], cfg["port"]))
    return None


def stop_server(cfg: dict, state_root: Path, log_path: Path | None = None,
                timeout: int = 15) -> bool:
    info = read_pid(state_root, cfg["name"])
    if not info:
        if not port_free(cfg["port"]):
            raise DeployError(
                "%s port %d is in use by an unowned process; refusing to kill it" %
                (cfg["name"], cfg["port"]))
        return False
    pid = int(info.get("pid", 0))
    if pid <= 0 or not pid_alive(pid):
        clear_pid(state_root, cfg["name"])
        if not port_free(cfg["port"]):
            raise DeployError(
                "%s port %d is in use after its recorded server exited; "
                "refusing to touch an unowned process" % (cfg["name"], cfg["port"]))
        return False
    if not _pid_matches(info, cfg["binary"]):
        raise DeployError(
            "recorded %s server pid %s is not the expected process; refusing to stop it"
            % (cfg["name"], info.get("pid")))
    if log_path:
        with open(log_path, "a", encoding="utf-8") as fh:
            fh.write("%s stopping %s server pid %d\n" % (now_iso(), cfg["name"], pid))
    try:
        os.kill(pid, signal.SIGTERM)
    except OSError:
        pass
    deadline = time.time() + timeout
    while time.time() < deadline and pid_alive(pid):
        if process_start(pid) != info.get("started"):
            break
        time.sleep(0.1)
    if pid_alive(pid) and process_start(pid) == info.get("started"):
        try:
            os.kill(pid, signal.SIGKILL)
        except OSError:
            pass
        time.sleep(0.2)
    clear_pid(state_root, cfg["name"])
    if not port_free(cfg["port"]):
        raise DeployError(
            "%s port %d is still busy after stopping the recorded server" %
            (cfg["name"], cfg["port"]))
    return True


def http_json(url: str, timeout: float = 2.0,
              host: str | None = None) -> tuple[int, dict | None]:
    headers = {"Accept": "application/json"}
    if host:
        headers["Host"] = host
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as response:
            body = response.read()
            try:
                return response.status, json.loads(body.decode("utf-8"))
            except ValueError:
                return response.status, None
    except urllib.error.HTTPError as exc:
        try:
            body = json.loads(exc.read().decode("utf-8"))
        except Exception:
            body = None
        return exc.code, body
    except Exception:
        return 0, None


def start_server(cfg: dict, state_root: Path, log_path: Path, build_id: str,
                 expect_version: bool = True, timeout: int = 30) -> int:
    if not port_free(cfg["port"]):
        raise DeployError("%s port %d is already in use" % (cfg["name"], cfg["port"]))
    cfg["data"].mkdir(parents=True, exist_ok=True)
    log_path.parent.mkdir(parents=True, exist_ok=True)
    command = [str(cfg["binary"]), "serve", "--data-dir", str(cfg["data"]),
               "--dist", str(cfg["dist"]), "--workers-dir", str(cfg["workers"]),
               "--port", str(cfg["port"]), "--public-port", str(cfg["port"])]
    log_file = open(log_path, "a", encoding="utf-8")
    log_file.write("\n%s starting %s: %s\n" % (now_iso(), cfg["name"],
                                               " ".join(command)))
    log_file.flush()
    try:
        child = subprocess.Popen(command, stdout=log_file, stderr=subprocess.STDOUT,
                                 start_new_session=True, cwd=str(cfg["worker_cwd"]
                                 if "worker_cwd" in cfg else Path.cwd()))
    except FileNotFoundError as exc:
        log_file.close()
        raise DeployError("server binary not found: %s" % cfg["binary"]) from exc
    info = {"pid": child.pid, "started": process_start(child.pid),
            "startedAt": now_iso(), "binary": str(cfg["binary"]),
            "data": str(cfg["data"]), "dist": str(cfg["dist"]),
            "workers": str(cfg["workers"]), "port": cfg["port"],
            "buildId": build_id, "env": cfg["name"]}
    write_pid(state_root, cfg["name"], info)
    deadline = time.time() + timeout
    last_error = None
    while time.time() < deadline:
        if child.poll() is not None:
            clear_pid(state_root, cfg["name"])
            raise DeployError(
                "%s server exited before becoming healthy; log: %s" %
                (cfg["name"], tail(log_path)))
        code, body = http_json("http://127.0.0.1:%d/api/version" % cfg["port"], 1.5)
        if code == 200 and isinstance(body, dict):
            version = body.get("version", "")
            if expect_version and version != build_id:
                last_error = ("server reports version %r but %r was built" %
                              (version, build_id))
            else:
                health_code, _ = http_json(
                    "http://127.0.0.1:%d/api/health" % cfg["port"], 1.5)
                if health_code == 200:
                    return child.pid
                last_error = "health endpoint returned %s" % health_code
        elif code != 0:
            last_error = "version endpoint returned %s" % code
        time.sleep(0.2)
    try:
        stop_server(cfg, state_root, log_path, timeout=5)
    except DeployError:
        pass
    raise DeployError(
        "%s server did not become healthy: %s; log: %s" %
        (cfg["name"], last_error or "timeout", tail(log_path)))


def restart_legacy_server(legacy: dict, env: dict, state_root: Path,
                          log_path: Path, timeout: int = 20) -> int:
    """Restart the captured pre-isolation process with its original argv.

    The old binary predates the new runtime interface: it may reject
    --public-port and has no /api/version. Reuse the exact verified command
    line and accept /api/health or / as the compatible health check.
    """
    argv = shlex.split(legacy.get("command") or "")
    if not argv:
        raise DeployError("legacy recovery has no recorded command line")
    binary = Path(argv[0])
    if not binary.is_file():
        raise DeployError("legacy server binary missing after restore: %s" % binary)
    if not port_free(env["port"]):
        raise DeployError(
            "legacy %s port %d is still in use; refusing to start a second "
            "server" % (env["name"], env["port"]))
    cwd = binary.resolve().parent.parent
    log_path.parent.mkdir(parents=True, exist_ok=True)
    log_file = open(log_path, "a", encoding="utf-8")
    log_file.write("\n%s restarting legacy %s: %s\n"
                   % (now_iso(), env["name"], " ".join(argv)))
    log_file.flush()
    try:
        child = subprocess.Popen(argv, stdout=log_file, stderr=subprocess.STDOUT,
                                 start_new_session=True, cwd=str(cwd))
    except OSError as exc:
        log_file.close()
        raise DeployError("cannot start legacy server %s: %s"
                          % (binary, exc)) from exc
    write_pid(state_root, env["name"], {
        "pid": child.pid, "started": process_start(child.pid),
        "binary": legacy["binary"], "data": legacy["data"],
        "dist": legacy["dist"], "workers": legacy["workers"],
        "port": env["port"], "legacy": True,
    })
    deadline = time.time() + timeout
    last_error = None
    while time.time() < deadline:
        if child.poll() is not None:
            clear_pid(state_root, env["name"])
            raise DeployError(
                "legacy %s server exited before becoming healthy; log: %s"
                % (env["name"], tail(log_path)))
        code, _ = http_json("http://127.0.0.1:%d/api/health" % env["port"], 1.5)
        if code == 200:
            return child.pid
        code, _ = http_json("http://127.0.0.1:%d/" % env["port"], 1.5)
        if code == 200:
            return child.pid
        last_error = "health/root returned %s" % code
        time.sleep(0.2)
    try:
        stop_server(env, state_root, log_path, timeout=5)
    except DeployError:
        pass
    raise DeployError(
        "legacy %s server did not become healthy: %s; log: %s"
        % (env["name"], last_error or "timeout", tail(log_path)))


def deploy_state_path(state_root: Path, env: str) -> Path:
    return state_root / env / "state.json"


def write_deploy_state(state_root: Path, env: str, **fields) -> None:
    value = {"schema": SCHEMA, "env": env, "at": now_iso()}
    value.update(fields)
    write_json_atomic(deploy_state_path(state_root, env), value)


def release_receipt_path(state_root: Path) -> Path:
    return state_root / "prod" / "release-receipt.json"


def write_release_receipt(state_root: Path, source: str, squash: str,
                          approval: str, state: dict) -> dict:
    """Pin the successful release's recovery point to this finish transaction."""
    receipt = {
        "schema": 1,
        "at": now_iso(),
        "sourceSha": source,
        "squashSha": squash,
        "approval": approval,
        "buildId": state.get("buildId"),
        "recovery": state.get("recovery"),
    }
    write_json_atomic(release_receipt_path(state_root), receipt)
    return receipt


def verify_release_receipt(state_root: Path, prod: dict, expected_source: str,
                           expected_squash: str, expected_approval: str):
    """Return (recovery_point, None) only for this transaction's live release.

    A later or failed deploy overwrites deploy state; a different successful
    release overwrites the receipt. Either makes the old recovery request
    stale, and the mismatch is refused rather than rolling back that release.
    """
    receipt = read_json(release_receipt_path(state_root))
    if not receipt:
        return None, "missing release receipt"
    for key, expected in (("sourceSha", expected_source),
                          ("squashSha", expected_squash),
                          ("approval", expected_approval)):
        if not expected or receipt.get(key) != expected:
            return None, "receipt %s does not match this finish" % key
    state = read_json(deploy_state_path(state_root, prod["name"])) or {}
    if state.get("result") != "OK":
        return None, "latest deploy state is %r" % state.get("result")
    for key in ("sourceSha", "buildId", "recovery"):
        if state.get(key) != receipt.get(key):
            return None, "latest deploy %s no longer matches the receipt" % key
    point = receipt.get("recovery")
    if not point or not Path(point).is_dir():
        return None, "pinned recovery point is missing"
    code, body = http_json("http://127.0.0.1:%d/api/version" % prod["port"], 1.5)
    if code != 200 or not isinstance(body, dict) or             body.get("version") != receipt.get("buildId"):
        return None, ("current production build cannot be verified as %s"
                      % receipt.get("buildId"))
    return point, None


def append_deploy_log(cfg: dict, line: str) -> None:
    try:
        cfg["data"].mkdir(parents=True, exist_ok=True)
        with open(cfg["data"] / "deploy.log", "a", encoding="utf-8") as fh:
            fh.write(line.rstrip() + "\n")
    except OSError:
        pass


def build_identity(app_home: Path) -> tuple[str, str, bool]:
    commit = git(app_home, "rev-parse", "HEAD")
    status = subprocess.run(["git", "-C", str(app_home), "status", "--porcelain"],
                            text=True, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL).stdout
    dirty = bool(status.strip())
    build_id = "%s%s-%s" % (commit[:12], "-dirty" if dirty else "", unique_stamp())
    return build_id, commit, dirty


def source_identity(commit: str) -> tuple[str, str, bool]:
    """Build identity for an immutable pinned revision, never a moving HEAD."""
    return ("%s-%s" % (commit[:12], unique_stamp()), commit, False)


def materialize_source(main_repo: Path, sha: str, state_root: Path) -> Path:
    """Extract the exact committed tree of `sha` into a private build directory.

    `git archive` never consults the main working tree, so a dirty or moved
    checkout cannot leak into a release. The extracted tree is removed after
    the deploy attempt; candidate artifacts and recovery copies are retained.
    """
    commit = git(main_repo, "rev-parse", "--verify", sha + "^{commit}")
    root = state_root / "source"
    root.mkdir(parents=True, exist_ok=True)
    target = root / ("%s-%s" % (commit[:12], unique_stamp()))
    target.mkdir(parents=True, exist_ok=False)
    archive = target.parent / (target.name + ".tar")
    try:
        try:
            subprocess.run(
                ["git", "-C", str(main_repo), "archive", "--format=tar",
                 "--output", str(archive), commit],
                check=True, text=True, stdout=subprocess.PIPE,
                stderr=subprocess.PIPE)
            with tarfile.open(archive) as tf:
                try:
                    tf.extractall(target, filter="data")
                except TypeError:
                    tf.extractall(target)
        except (subprocess.CalledProcessError, tarfile.TarError) as exc:
            shutil.rmtree(target, ignore_errors=True)
            detail = getattr(exc, "stderr", "") or str(exc)
            raise DeployError("cannot materialize source %s: %s"
                              % (commit, str(detail).strip())) from exc
    finally:
        try:
            archive.unlink()
        except FileNotFoundError:
            pass
    return target


def listening_pids(port: int) -> list[int]:
    """PIDs listening on `port`, verified by lsof rather than guessed."""
    try:
        result = subprocess.run(
            ["lsof", "-nP", "-iTCP:%d" % port, "-sTCP:LISTEN", "-t"],
            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    except OSError as exc:
        raise DeployError("cannot inspect port %d: %s" % (port, exc)) from exc
    if result.returncode != 0:
        return []
    pids = []
    for line in result.stdout.splitlines():
        line = line.strip()
        if line.isdigit():
            pids.append(int(line))
    return sorted(set(pids))


def process_flags(command: str) -> tuple[str, dict[str, str]]:
    """Split a recorded command line into argv[0] and --flag values."""
    try:
        argv = shlex.split(command)
    except ValueError as exc:
        raise DeployError("cannot parse the running command line: %s" % exc) from exc
    if not argv:
        raise DeployError("the running process has an empty command line")
    values: dict[str, str] = {}
    for index, item in enumerate(argv[1:], start=1):
        if item.startswith("--") and "=" in item:
            key, value = item.split("=", 1)
            values[key] = value
        elif item.startswith("--") and index + 1 < len(argv):
            values[item] = argv[index + 1]
    return argv[0], values


def _legacy_from_process(env: dict, pid: int, command: str) -> dict:
    binary_arg, flags = process_flags(command)
    binary = Path(binary_arg)
    if not (binary.is_file() and os.access(binary, os.X_OK)):
        raise DeployError(
            "--adopt-legacy: port %d owner pid %d binary %r is not an "
            "executable file" % (env["port"], pid, binary_arg))
    if binary.resolve() != env["binary"].resolve():
        raise DeployError(
            "--adopt-legacy: port %d owner pid %d runs %s, not the configured "
            "%s binary; refusing to adopt it"
            % (env["port"], pid, binary.resolve(), env["binary"].resolve()))
    data_arg = flags.get("--data-dir")
    if data_arg and Path(data_arg).resolve() != env["data"].resolve():
        raise DeployError(
            "--adopt-legacy: port %d owner uses data %s, not configured %s"
            % (env["port"], data_arg, env["data"]))
    dist_arg = flags.get("--dist")
    workers_arg = flags.get("--workers-dir")
    if not dist_arg or not Path(dist_arg).is_dir():
        raise DeployError(
            "--adopt-legacy: port %d owner serves no existing --dist directory; "
            "refusing to guess the bundle" % env["port"])
    if not workers_arg or not Path(workers_arg).is_dir():
        raise DeployError(
            "--adopt-legacy: port %d owner uses no existing --workers-dir; "
            "refusing to guess the worker tree" % env["port"])
    port_arg = flags.get("--port")
    try:
        recorded_port = int(port_arg) if port_arg else env["port"]
    except ValueError as exc:
        raise DeployError(
            "--adopt-legacy: port %d owner has an invalid --port %r"
            % (env["port"], port_arg)) from exc
    if recorded_port != env["port"]:
        raise DeployError(
            "--adopt-legacy: port %d owner was started with --port %s"
            % (env["port"], port_arg))
    return {
        "pid": pid,
        "started": process_start(pid),
        "command": command,
        "binary": str(binary.resolve()),
        "dist": str(Path(dist_arg).resolve()),
        "workers": str(Path(workers_arg).resolve()),
        "data": str(env["data"]),
        "port": env["port"],
    }


def _capture_legacy_assets(legacy: dict, env: dict, recovery: Path) -> dict:
    recovery.mkdir(parents=True, exist_ok=True)
    legacy_root = recovery / "legacy"
    legacy_root.mkdir(parents=True, exist_ok=True)
    binary = Path(legacy["binary"])
    shutil.copy2(binary, legacy_root / "binary.prev")
    copy_tree(Path(legacy["dist"]), legacy_root / "dist.prev", ignore=())
    copy_tree(Path(legacy["workers"]), legacy_root / "workers.prev", ignore=())
    legacy = dict(legacy)
    legacy["dbExisted"] = sqlite_backup(env["data"] / "store.sqlite",
                                        legacy_root / "store.sqlite")
    write_json_atomic(legacy_root / "legacy.json", legacy)
    return legacy


def _verify_recorded_legacy(legacy: dict, env: dict, state_root: Path) -> None:
    pid = int(legacy.get("pid", 0))
    if pid <= 0 or not pid_alive(pid):
        raise DeployError(
            "--adopt-legacy: recorded legacy pid %s is no longer running"
            % legacy.get("pid"))
    if legacy.get("started") and process_start(pid) != legacy["started"]:
        raise DeployError(
            "--adopt-legacy: recorded legacy pid %s was replaced" % pid)
    current = _legacy_from_process(env, pid, process_command(pid))
    for key in ("binary", "dist", "workers", "data", "port"):
        if current[key] != legacy.get(key):
            raise DeployError(
                "--adopt-legacy: running process %s no longer matches the "
                "recorded legacy %s" % (key, key))


def adopt_legacy_process(env: dict, state_root: Path, recovery: Path) -> dict:
    """Adopt a pre-isolation production process explicitly and safely.

    The port owner must be a single process whose command line names the
    intended production binary and data directory. Its actual --dist and
    --workers-dir arguments are read from that process, so the recovery copy
    is exactly the bundle and worker tree it serves, never a configured guess.
    A retry after a failed rollback re-captures those same actual assets. Once
    a migration completes, a second --adopt-legacy is refused.
    """
    adoption_path = state_root / env["name"] / "adoption.json"
    existing = read_pid(state_root, env["name"])
    if existing:
        if not _pid_matches(existing, env["binary"]):
            raise DeployError(
                "--adopt-legacy: recorded %s pid %s does not match %s; refusing "
                "to adopt or stop it" % (env["name"], existing.get("pid"),
                                         env["binary"]))
        adoption = read_json(adoption_path)
        if not adoption:
            raise DeployError(
                "--adopt-legacy: a pid record already exists for %s but no "
                "adoption record does; inspect %s before replacing it"
                % (env["name"], pid_path(state_root, env["name"])))
        if adoption.get("completedAt"):
            raise DeployError(
                "--adopt-legacy: %s was already migrated at %s; deploy without "
                "--adopt-legacy" % (env["name"], adoption["completedAt"]))
        _verify_recorded_legacy(adoption, env, state_root)
        return _capture_legacy_assets(adoption, env, recovery)

    if port_free(env["port"]):
        raise DeployError(
            "--adopt-legacy: nothing is listening on %s port %d; start the "
            "existing server first or deploy without --adopt-legacy"
            % (env["name"], env["port"]))
    pids = listening_pids(env["port"])
    if len(pids) != 1:
        raise DeployError(
            "--adopt-legacy: expected one listener on %s port %d, found %s; "
            "refusing to adopt an ambiguous port occupant"
            % (env["name"], env["port"], pids or "none"))
    pid = pids[0]
    legacy = _legacy_from_process(env, pid, process_command(pid))
    legacy = _capture_legacy_assets(legacy, env, recovery)
    write_json_atomic(adoption_path, legacy)
    write_pid(state_root, env["name"], {
        "pid": pid, "started": process_start(pid), "binary": legacy["binary"],
        "data": legacy["data"], "dist": legacy["dist"],
        "workers": legacy["workers"], "port": env["port"], "adopted": True,
    })
    print("sift-deploy: adopted legacy %s pid %d serving %s / %s"
          % (env["name"], pid, legacy["dist"], legacy["workers"]))
    return legacy


def restore_legacy_artifacts(recovery: Path, env: dict, legacy: dict) -> None:
    """Restore the assets the adopted process actually served, and remove new ones."""
    legacy_root = recovery / "legacy"
    if not (legacy_root / "legacy.json").is_file():
        raise DeployError("legacy recovery record is missing at %s"
                          % (legacy_root / "legacy.json"))
    for key, flag in (("binary", "binary"), ("dist", "dist"),
                      ("workers", "workers")):
        target = Path(legacy[flag])
        if target.exists():
            if target.is_dir():
                shutil.rmtree(target)
            else:
                target.unlink()
        src = legacy_root / (key + ".prev")
        if not src.exists():
            raise DeployError("legacy recovery copy for %s is missing" % key)
        if src.is_dir():
            copy_tree(src, target, ignore=())
        else:
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, target)
            target.chmod(0o755)
    # New isolated destinations are restored by restore_artifacts(): their
    # pre-deploy state (present or absent) must not be altered by adoption.
    shutil.copy2(legacy_root / "binary.prev", env["binary"])
    env["binary"].chmod(0o755)


# ── artifact publishing and recovery ────────────────────────────────────────

def _copy_or_move(src: Path, dst: Path) -> None:
    if dst.exists():
        raise DeployError("refusing to overwrite %s during publish" % dst)
    dst.parent.mkdir(parents=True, exist_ok=True)
    try:
        os.replace(src, dst)
    except OSError:
        if src.is_dir():
            shutil.copytree(src, dst, symlinks=True)
        else:
            shutil.copy2(src, dst)
        if src.is_dir():
            shutil.rmtree(src)
        else:
            src.unlink()


def backup_artifacts(cfg: dict, recovery: Path) -> None:
    recovery.mkdir(parents=True, exist_ok=True)
    existed = {
        "binary": cfg["binary"].is_file(),
        "dist": cfg["dist"].is_dir(),
        "workers": cfg["workers"].is_dir(),
    }
    write_json_atomic(recovery / "artifacts.json", existed)
    if existed["binary"]:
        shutil.copy2(cfg["binary"], recovery / "binary.prev")
    if existed["dist"]:
        copy_tree(cfg["dist"], recovery / "dist.prev", ignore=())
    if existed["workers"]:
        copy_tree(cfg["workers"], recovery / "workers.prev", ignore=())


def publish_artifacts(candidate: Path, cfg: dict) -> None:
    for key, target in (("binary", cfg["binary"]), ("dist", cfg["dist"]),
                        ("workers", cfg["workers"])):
        src = candidate / ("sifty" if key == "binary" else key)
        if not src.exists():
            raise DeployError("candidate is missing %s at %s" % (key, src))
        if target.exists():
            if target.is_dir():
                shutil.rmtree(target)
            else:
                target.unlink()
        _copy_or_move(src, target)
        if key == "binary":
            target.chmod(0o755)


def restore_artifacts(recovery: Path, cfg: dict) -> None:
    existed = read_json(recovery / "artifacts.json") or {}
    for key, target in (("binary", cfg["binary"]), ("dist", cfg["dist"]),
                        ("workers", cfg["workers"])):
        if target.exists():
            if target.is_dir():
                shutil.rmtree(target)
            else:
                target.unlink()
        if existed.get(key):
            src = recovery / ("binary.prev" if key == "binary" else key + ".prev")
            if src.exists():
                if src.is_dir():
                    copy_tree(src, target, ignore=())
                else:
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(src, target)
                    target.chmod(0o755)
            else:
                raise DeployError("recovery copy for %s is missing" % key)


def artifact_hashes(cfg: dict) -> dict:
    return {
        "binary": sha256_file(cfg["binary"]) if cfg["binary"].is_file() else None,
        "dist": tree_digest(cfg["dist"]) if cfg["dist"].is_dir() else None,
        "workers": tree_digest(cfg["workers"]) if cfg["workers"].is_dir() else None,
    }


# ── staging data refresh ────────────────────────────────────────────────────

RUNTIME_DATA_NAMES = {"store.sqlite", "store.sqlite-wal", "store.sqlite-shm",
                      "server.pid", "deploy.state.json"}


def _payload_names(data: Path) -> list[str]:
    """Top-level user data entries, excluding SQLite/runtime operational files."""
    if not data.is_dir():
        return []
    names = []
    for name in sorted(os.listdir(data)):
        if name in RUNTIME_DATA_NAMES:
            continue
        if name.startswith("store.sqlite.backup."):
            continue
        if name.startswith(".refresh-backup") or name.startswith(".refresh-incoming-"):
            continue
        if name.startswith(".refresh-") or name.endswith(".log"):
            continue
        names.append(name)
    return names


def _copy_entry(src: Path, dst: Path) -> None:
    if src.is_dir():
        copy_tree(src, dst, ignore=())
    else:
        shutil.copy2(src, dst)


def _backup_staging_data(staging: dict, backup: Path) -> dict:
    backup.mkdir(parents=True, exist_ok=False)
    meta = {"db": False, "entries": []}
    db = staging["data"] / "store.sqlite"
    if sqlite_backup(db, backup / "store.sqlite"):
        meta["db"] = True
    for name in _payload_names(staging["data"]):
        _copy_entry(staging["data"] / name, backup / name)
        meta["entries"].append(name)
    write_json_atomic(backup / "meta.json", meta)
    return meta


def _restore_staging_data(staging: dict, backup: Path) -> None:
    meta = read_json(backup / "meta.json") or {}
    db = staging["data"] / "store.sqlite"
    if meta.get("db"):
        restore_sqlite(backup / "store.sqlite", db, True)
    else:
        remove_sqlite_sidecars(db)
        try:
            db.unlink()
        except FileNotFoundError:
            pass
    # Remove every payload currently present, then restore the snapshot. The
    # refresh backup directory itself is not a payload and is preserved.
    for name in _payload_names(staging["data"]):
        target = staging["data"] / name
        if target.is_dir():
            shutil.rmtree(target)
        else:
            target.unlink()
    for name in meta.get("entries", []):
        src = backup / name
        if src.exists():
            _copy_entry(src, staging["data"] / name)


def refresh_staging_data(prod: dict, staging: dict, shared: dict,
                         log_path: Path, state_root: Path,
                         restart: bool = True,
                         cfg: dict[str, str] | None = None,
                         main_repo: Path | None = None) -> Path:
    """Explicitly replace staging data with a consistent production snapshot."""
    if cfg is not None and main_repo is not None:
        preview_conflict(staging, main_repo, cfg)
    prod_db = prod["data"] / "store.sqlite"
    if not prod_db.is_file():
        raise DeployError("production database not found at %s" % prod_db)
    if not (prod["data"] / "config").is_dir():
        raise DeployError(
            "production config directory not found at %s" % (prod["data"] / "config"))

    was_running = False
    stopped = False
    backup: Path | None = None
    try:
        info = read_pid(state_root, staging["name"])
        if info:
            if not _pid_matches(info, staging["binary"]):
                raise DeployError(
                    "recorded staging server pid %s is not the expected process"
                    % info.get("pid"))
            was_running = True
        elif not port_free(staging["port"]):
            raise DeployError(
                "staging port %d is in use by an unowned process; refusing to continue"
                % staging["port"])

        if was_running:
            stop_server(staging, state_root, log_path)
            stopped = True

        staging["data"].mkdir(parents=True, exist_ok=True)
        backup = staging["data"] / ".refresh-backup" / unique_stamp()
        _backup_staging_data(staging, backup)

        incoming = staging["data"] / (".refresh-incoming-%s" % unique_stamp())
        if incoming.exists():
            shutil.rmtree(incoming)
        incoming.mkdir(parents=True)
        try:
            sqlite_backup(prod_db, incoming / "store.sqlite")
            for name in _payload_names(prod["data"]):
                _copy_entry(prod["data"] / name, incoming / name)

            # Replace consistently: SQLite first, after removing any WAL/SHM
            # files from the old generation. Then mirror the payload entries
            # (config, config-backups, kb/, evidence, answers, feedback and any
            # other root-relative candidate file or directory).
            target_db = staging["data"] / "store.sqlite"
            remove_sqlite_sidecars(target_db)
            try:
                target_db.unlink()
            except FileNotFoundError:
                pass
            os.replace(incoming / "store.sqlite", target_db)
            for name in _payload_names(staging["data"]):
                target = staging["data"] / name
                if target.is_dir():
                    shutil.rmtree(target)
                else:
                    target.unlink()
            for name in _payload_names(incoming):
                os.replace(incoming / name, staging["data"] / name)
        finally:
            shutil.rmtree(incoming, ignore_errors=True)

        write_deploy_state(state_root, "staging", result="DATA_REFRESHED",
                           backup=str(backup), prodData=str(prod["data"]),
                           stagingData=str(staging["data"]))
        append_deploy_log(staging, "%s env=staging data-refresh backup=%s result=OK"
                          % (now_stamp(), backup))
        if restart and was_running:
            start_server(staging, state_root, log_path, "", expect_version=False)
        return backup
    except Exception as exc:
        errors = [str(exc)]
        safe_to_restore = True
        if was_running and not stopped:
            safe_to_restore = False
            errors.append(
                "target could not be safely stopped; backup retained and no "
                "restore attempted")
        elif stopped and not port_free(staging["port"]):
            safe_to_restore = False
            errors.append(
                "staging port %d is occupied by an unverified process; backup "
                "retained and no restore attempted" % staging["port"])

        restore_ok = safe_to_restore
        if safe_to_restore and backup is not None and (backup / "meta.json").is_file():
            try:
                _restore_staging_data(staging, backup)
            except Exception as restore_exc:
                restore_ok = False
                errors.append("data restore failed: %s" % restore_exc)

        restart_ok = True
        if restore_ok and was_running and staging["binary"].is_file():
            try:
                start_server(staging, state_root, log_path, "", expect_version=False)
            except Exception as restart_exc:
                restart_ok = False
                errors.append("previous server restart failed: %s" % restart_exc)

        result = ("REFRESH_FAILED" if safe_to_restore and restore_ok and restart_ok
                  else "REFRESH_INCOMPLETE")
        write_deploy_state(state_root, "staging", result=result,
                           error="; ".join(errors),
                           backup=str(backup) if backup else None)
        raise DeployError("; ".join(errors)) from exc


# ── deploy / refresh command entry points ──────────────────────────────────

def deploy_main(script_path: str, argv: list[str] | None = None) -> int:
    import argparse
    parser = argparse.ArgumentParser(prog="sift-deploy")
    parser.add_argument("--env", choices=("staging", "prod"), required=True)
    parser.add_argument("--build-only", action="store_true",
                        help="build and publish artifacts without stopping or starting a server")
    parser.add_argument("--refresh-data", action="store_true",
                        help="staging only: explicitly replace staging data from production first")
    parser.add_argument("--source", default=None, metavar="REVISION",
                        help="build the exact committed revision instead of the checkout HEAD")
    parser.add_argument("--adopt-legacy", action="store_true",
                        help="explicitly adopt the current pre-isolation process "
                             "on the target port before deploying")
    parser.add_argument("--recover", default=None, metavar="DIR",
                        help="standalone: restore the captured recovery point "
                             "and restart the previous server")
    args = parser.parse_args(argv)

    app_home, main_repo = repo_roots(script_path)
    cfg = load_config(app_home)
    prod, staging, shared = all_configs(app_home, main_repo, cfg)
    env = prod if args.env == "prod" else staging
    if args.refresh_data and args.env != "staging":
        raise DeployError("--refresh-data is only valid for staging")
    if args.refresh_data and args.build_only:
        raise DeployError("--refresh-data and --build-only are separate operations")
    if args.adopt_legacy and args.build_only:
        raise DeployError("--adopt-legacy requires a full deploy, not --build-only")
    if args.recover and any((args.source, args.adopt_legacy, args.build_only,
                             args.refresh_data)):
        raise DeployError("--recover is a standalone operation")
    state_root = shared["state_root"]
    acquire_environment_lock(state_root, env["name"])
    state_dir = state_root / env["name"]
    if args.recover:
        recovery = resolve_path(args.recover, main_repo)
        if not (recovery / "artifacts.json").is_file():
            raise DeployError("recovery point %s has no artifacts.json" % recovery)
        meta = read_json(recovery / "meta.json") or {}
        legacy = None
        legacy_file = recovery / "legacy" / "legacy.json"
        if legacy_file.is_file():
            legacy = read_json(legacy_file)
        preview_conflict(env, main_repo, cfg)
        _rollback(env, None, state_root, recovery,
                  str(meta.get("buildId") or recovery.name),
                  "explicit release recovery: %s" % recovery,
                  env["server_log"], legacy=legacy)
        result = (read_json(deploy_state_path(state_root, env["name"])) or {}).get("result")
        print("sift-deploy: recovery result=%s recovery=%s"
              % (result, recovery))
        return 0 if result == "ROLLBACK" else 1
    source_dir = None
    if args.source:
        source_commit = git(main_repo, "rev-parse", "--verify",
                            args.source + "^{commit}")
        source_dir = materialize_source(main_repo, source_commit, state_root)
        build_id, commit, dirty = source_identity(source_commit)
        # A pinned source supplies its own workers; an explicit configured
        # source still wins for a deliberate external worker tree.
        if not setting(cfg, "SIFTY_WORKERS_SOURCE") and \
                "SIFTY_WORKERS_SOURCE" not in os.environ:
            shared["workers_source"] = source_dir / "backend" / "workers"
    else:
        source_commit = None
        build_id, commit, dirty = build_identity(app_home)
    build_root = source_dir or app_home
    candidate_id = "%s-%s-%s" % (env["name"], unique_stamp(), build_id[:8])
    candidate = state_dir / "candidates" / candidate_id
    build_log = candidate / "build.log"
    candidate.mkdir(parents=True, exist_ok=False)
    cfg_doc = {"buildId": build_id, "commit": commit, "dirty": dirty,
               "source": str(build_root), "sourceSha": commit,
               "env": env["name"]}
    mutation = False
    refresh_backup = None
    legacy = None
    recovery = state_dir / ("build-recovery" if args.build_only else "recovery") / candidate_id

    try:
        write_deploy_state(state_root, env["name"], result="BUILDING",
                           buildId=build_id, commit=commit, dirty=dirty,
                           sourceSha=commit, candidate=str(candidate))
        # A live preview owns its port and data; never replace either behind it.
        preview_conflict(env, main_repo, cfg)
        if args.adopt_legacy:
            legacy = adopt_legacy_process(env, state_root, recovery)
        if not args.build_only and not args.refresh_data:
            if not env["data"].is_dir():
                raise DeployError(
                    "%s data directory %s does not exist; for staging use "
                    "--refresh-data to create an explicit snapshot" %
                    (env["name"], env["data"]))
            probe_port_owner(env, state_root)

        build_binary(build_root / "backend", candidate / "sifty", build_id, build_log)
        build_frontend(build_root / "frontend", candidate / "dist", build_log,
                       build_id, shared["frontend_cache"])
        build_workers(shared["workers_source"], candidate / "workers",
                      shared["workers_cache"], build_log)
        write_json_atomic(candidate / "meta.json", cfg_doc)

        if args.build_only:
            # Publishing over a running environment is unsafe. The candidate
            # is retained so an operator can inspect it, but this command fails.
            if not port_free(env["port"]):
                raise DeployError(
                    "refusing build-only publish for %s: port %d is in use "
                    "(candidate kept at %s)" % (env["name"], env["port"], candidate))
            backup_ok = False
            try:
                backup_artifacts(env, recovery)
                backup_ok = True
                publish_artifacts(candidate, env)
            except Exception as exc:
                if backup_ok:
                    try:
                        restore_artifacts(recovery, env)
                    except Exception as restore_exc:
                        raise DeployError(
                            "%s; artifact restore failed: %s" % (exc, restore_exc)) from exc
                raise
            hashes = artifact_hashes(env)
            write_deploy_state(state_root, env["name"], result="BUILT",
                               buildId=build_id, commit=commit, dirty=dirty,
                               artifacts=hashes, recovery=str(recovery))
            append_deploy_log(env, "%s env=%s build=%s hash=%s result=BUILT"
                              % (now_stamp(), env["name"], build_id,
                                 (hashes["binary"] or "")[:16]))
            print("sift-deploy: built %s at %s (no server touched)" %
                  (env["name"], build_id))
            return 0

        # Artifact copies may precede the stop. The final SQLite snapshot must
        # follow a verified stop so writes before shutdown are not lost.
        backup_artifacts(env, recovery)
        db_existed = False
        db = env["data"] / "store.sqlite"

        if args.refresh_data:
            try:
                refresh_backup = refresh_staging_data(
                    prod, staging, shared, build_log, state_root,
                    restart=False, cfg=cfg, main_repo=main_repo)
                mutation = True
                write_json_atomic(recovery / "meta.json", {
                    "dbExisted": False,
                    "refreshBackup": str(refresh_backup) if refresh_backup else None,
                    "refreshRequested": True,
                    "legacyAdopted": bool(legacy),
                })
                publish_artifacts(candidate, env)
                hashes = artifact_hashes(env)
                append_deploy_log(env, "%s env=%s build=%s hash=%s result=DEPLOYING"
                                  % (now_stamp(), env["name"], build_id,
                                     (hashes["binary"] or "")[:16]))
                start_server(env, state_root, env["server_log"], build_id)
            except Exception as exc:
                if mutation:
                    _rollback(env, staging, state_root, recovery, build_id,
                              str(exc), build_log, legacy=legacy)
                raise
        else:
            stop_server(env, state_root, env["server_log"])
            mutation = True
            try:
                db_existed = sqlite_backup(db, recovery / "store.sqlite")
            except Exception as backup_exc:
                # Artifacts are untouched and no database snapshot exists.
                # Restart the old server; never restore a missing snapshot.
                restart_error = None
                try:
                    start_server(env, state_root, env["server_log"], "",
                                 expect_version=False)
                except Exception as restart_exc:
                    restart_error = str(restart_exc)
                message = "database backup after stop failed: %s" % backup_exc
                if restart_error:
                    message += "; previous server restart failed: %s" % restart_error
                    write_deploy_state(state_root, env["name"],
                                       result="ROLLBACK_INCOMPLETE",
                                       buildId=build_id, error=message,
                                       recovery=str(recovery))
                else:
                    write_deploy_state(state_root, env["name"], result="FAILED",
                                       buildId=build_id, error=message,
                                       recovery=str(recovery))
                append_deploy_log(env, "%s env=%s build=%s result=BACKUP_FAILED error=%s"
                                  % (now_stamp(), env["name"], build_id,
                                     message.replace("\n", " ")[:300]))
                raise DeployError(message) from backup_exc

            try:
                write_json_atomic(recovery / "meta.json", {
                    "dbExisted": db_existed,
                    "refreshBackup": None,
                    "refreshRequested": False,
                    "legacyAdopted": bool(legacy),
                })
                publish_artifacts(candidate, env)
                hashes = artifact_hashes(env)
                append_deploy_log(env, "%s env=%s build=%s hash=%s result=DEPLOYING"
                                  % (now_stamp(), env["name"], build_id,
                                     (hashes["binary"] or "")[:16]))
                start_server(env, state_root, env["server_log"], build_id)
            except Exception as exc:
                _rollback(env, None, state_root, recovery, build_id,
                          str(exc), build_log, legacy=legacy)
                raise

        write_deploy_state(state_root, env["name"], result="OK",
                           buildId=build_id, commit=commit, dirty=dirty,
                           sourceSha=commit, artifacts=hashes,
                           recovery=str(recovery),
                           paths={"binary": str(env["binary"]), "dist": str(env["dist"]),
                                  "workers": str(env["workers"]), "data": str(env["data"])})
        if legacy:
            adoption_path = state_root / env["name"] / "adoption.json"
            adoption = read_json(adoption_path) or dict(legacy)
            adoption.update({"completedAt": now_iso(),
                             "completedBuild": build_id,
                             "completedSource": commit})
            write_json_atomic(adoption_path, adoption)
        append_deploy_log(env, "%s env=%s build=%s hash=%s result=OK"
                          % (now_stamp(), env["name"], build_id,
                             (hashes["binary"] or "")[:16]))
        print("sift-deploy: %s deployed %s on port %d" %
              (env["name"], build_id, env["port"]))
        print("sift-deploy: source %s build %s" % (commit, build_id))
        print("sift-deploy: url http://localhost:%d" % env["port"])
        return 0
    except Exception as exc:
        if isinstance(exc, DeployError):
            existing = read_json(deploy_state_path(state_root, env["name"])) or {}
            if existing.get("result") not in ("ROLLBACK", "ROLLBACK_INCOMPLETE",
                                              "REFRESH_INCOMPLETE"):
                write_deploy_state(state_root, env["name"], result="FAILED",
                                   buildId=build_id, commit=commit, dirty=dirty,
                                   error=str(exc), candidate=str(candidate))
            append_deploy_log(env, "%s env=%s build=%s result=FAILED error=%s"
                              % (now_stamp(), env["name"], build_id,
                                 str(exc).replace("\n", " ")[:300]))
            print("sift-deploy: ERROR: %s" % exc, file=sys.stderr)
            return 1
        raise
    finally:
        if source_dir is not None:
            try:
                shutil.rmtree(source_dir)
            except OSError as cleanup_exc:
                print("sift-deploy: WARNING: could not remove pinned source "
                      "checkout %s: %s" % (source_dir, cleanup_exc),
                      file=sys.stderr)

def _rollback(env: dict, refresh_env: dict | None, state_root: Path,
              recovery: Path, build_id: str, error: str, log_path: Path,
              legacy: dict | None = None) -> None:
    errors = [error]
    result = "ROLLBACK"
    stop_ok = True
    try:
        stop_server(env, state_root, env["server_log"], timeout=10)
    except Exception as stop_exc:
        stop_ok = False
        results_error = "target could not be stopped; no restore attempted: %s" % stop_exc
        errors.append(results_error)
        result = "ROLLBACK_INCOMPLETE"

    if stop_ok:
        artifacts_ok = True
        try:
            restore_artifacts(recovery, env)
            if legacy:
                # The adopted process served different dist/worker paths than
                # the new defaults; restore exactly those captured paths.
                restore_legacy_artifacts(recovery, env, legacy)
        except Exception as restore_exc:
            artifacts_ok = False
            errors.append("artifact restore failed: %s" % restore_exc)
            result = "ROLLBACK_INCOMPLETE"

        data_ok = artifacts_ok
        if artifacts_ok:
            meta = read_json(recovery / "meta.json")
            db = env["data"] / "store.sqlite"
            try:
                if not isinstance(meta, dict) or "dbExisted" not in meta:
                    raise DeployError(
                        "recovery metadata missing or invalid at %s; existing "
                        "database preserved" % (recovery / "meta.json"))
                if meta.get("refreshRequested") and refresh_env is not None:
                    backup = Path(meta.get("refreshBackup") or "")
                    if not (backup / "meta.json").is_file():
                        raise DeployError("refresh backup incomplete at %s" % backup)
                    # The refresh backup holds the full pre-refresh staging data tree.
                    _restore_staging_data(env, backup)
                else:
                    # Only trusted metadata may assert that no database existed.
                    restore_sqlite(recovery / "store.sqlite", db,
                                   bool(meta["dbExisted"]))
            except Exception as restore_exc:
                data_ok = False
                errors.append("data restore failed: %s" % restore_exc)
                result = "ROLLBACK_INCOMPLETE"

        if artifacts_ok and data_ok and env["binary"].is_file():
            try:
                if legacy:
                    # The captured old binary does not speak the new runtime
                    # interface; restart its exact original argv instead.
                    restart_legacy_server(legacy, env, state_root,
                                          env["server_log"], timeout=20)
                else:
                    start_server(env, state_root, env["server_log"], "",
                                 expect_version=False, timeout=20)
            except Exception as restart_exc:
                errors.append("previous server restart failed: %s" % restart_exc)
                result = "ROLLBACK_INCOMPLETE"

    message = "; ".join(errors)
    write_deploy_state(state_root, env["name"], result=result,
                       buildId=build_id, error=message, recovery=str(recovery))
    append_deploy_log(env, "%s env=%s build=%s result=%s error=%s"
                      % (now_stamp(), env["name"], build_id, result,
                         message.replace("\n", " ")[:300]))
    print("sift-deploy: %s: %s" % (result, message), file=sys.stderr)



def refresh_main(script_path: str, argv: list[str] | None = None) -> int:
    import argparse
    parser = argparse.ArgumentParser(prog="sift-data-refresh")
    parser.add_argument("--env", choices=("staging",), default="staging")
    parser.add_argument("--no-restart", action="store_true",
                        help="leave staging stopped after the refresh")
    parser.add_argument("--from", dest="source", default=None,
                        help="production data directory to snapshot (default: configured prod data)")
    args = parser.parse_args(argv)
    app_home, main_repo = repo_roots(script_path)
    cfg = load_config(app_home)
    prod, staging, shared = all_configs(app_home, main_repo, cfg)
    if args.source:
        prod = dict(prod)
        prod["data"] = resolve_path(args.source, main_repo)
    validate_refresh_source(prod, staging, shared)
    preview_conflict(staging, main_repo, cfg)
    state_root = shared["state_root"]
    acquire_environment_lock(state_root, staging["name"])
    log_path = state_root / "staging" / "refresh.log"
    try:
        backup = refresh_staging_data(
            prod, staging, shared, log_path, state_root,
            restart=not args.no_restart, cfg=cfg, main_repo=main_repo)
        print("sift-data-refresh: staging data refreshed from %s" % prod["data"])
        print("sift-data-refresh: previous staging data preserved at %s" % backup)
        return 0
    except Exception as exc:
        existing = read_json(deploy_state_path(state_root, "staging")) or {}
        if existing.get("result") != "REFRESH_INCOMPLETE":
            write_deploy_state(state_root, "staging", result="REFRESH_FAILED",
                               error=str(exc))
        print("sift-data-refresh: ERROR: %s" % exc, file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit("_deploy_lib is imported by bin/sift-deploy and bin/sift-data-refresh")
