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

import datetime as _dt
import hashlib
import json
import os
import shutil
import signal
import socket
import sqlite3
import subprocess
import sys
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
    target.parent.mkdir(parents=True, exist_ok=True)
    remove_sqlite_sidecars(target)
    try:
        target.unlink()
    except FileNotFoundError:
        pass
    if existed and source_backup.is_file():
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
        return True
    except OSError:
        return False


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
        if not port_free(cfg["port"]):
            raise DeployError(
                "%s port %d is busy while its recorded server is alive; "
                "refusing to continue" % (cfg["name"], cfg["port"]))
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
    if not _pid_matches(info, cfg["binary"]):
        raise DeployError(
            "recorded %s server pid %s is not the expected process; refusing to stop it"
            % (cfg["name"], info.get("pid")))
    pid = int(info["pid"])
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
        if not pid_alive(child.pid):
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


def deploy_state_path(state_root: Path, env: str) -> Path:
    return state_root / env / "state.json"


def write_deploy_state(state_root: Path, env: str, **fields) -> None:
    value = {"schema": SCHEMA, "env": env, "at": now_iso()}
    value.update(fields)
    write_json_atomic(deploy_state_path(state_root, env), value)


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


def publish_artifacts(candidate: Path, cfg: dict, recovery: Path) -> None:
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

def _backup_staging_data(staging: dict, backup: Path) -> dict:
    backup.mkdir(parents=True, exist_ok=False)
    meta = {"db": False, "dirs": {}, "files": {}}
    db = staging["data"] / "store.sqlite"
    if sqlite_backup(db, backup / "store.sqlite"):
        meta["db"] = True
    for name in DATA_DIRS:
        src = staging["data"] / name
        if src.is_dir():
            copy_tree(src, backup / name, ignore=())
            meta["dirs"][name] = True
    for name in DATA_FILES:
        src = staging["data"] / name
        if src.is_file():
            shutil.copy2(src, backup / name)
            meta["files"][name] = True
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
    for name in DATA_DIRS:
        dst = staging["data"] / name
        if dst.exists():
            shutil.rmtree(dst)
        if meta.get("dirs", {}).get(name):
            copy_tree(backup / name, dst, ignore=())
    for name in DATA_FILES:
        dst = staging["data"] / name
        if dst.exists():
            dst.unlink()
        if meta.get("files", {}).get(name):
            shutil.copy2(backup / name, dst)


def refresh_staging_data(prod: dict, staging: dict, shared: dict,
                         log_path: Path, state_root: Path,
                         restart: bool = True) -> Path:
    """Explicitly replace staging data with a consistent production snapshot."""
    prod_db = prod["data"] / "store.sqlite"
    if not prod_db.is_file():
        raise DeployError("production database not found at %s" % prod_db)
    if not (prod["data"] / "config").is_dir():
        raise DeployError(
            "production config directory not found at %s" % (prod["data"] / "config"))

    was_running = False
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

        staging["data"].mkdir(parents=True, exist_ok=True)
        backup = staging["data"] / ".refresh-backup" / unique_stamp()
        _backup_staging_data(staging, backup)

        incoming = staging["data"] / (".refresh-incoming-%s" % unique_stamp())
        if incoming.exists():
            shutil.rmtree(incoming)
        incoming.mkdir(parents=True)
        try:
            sqlite_backup(prod_db, incoming / "store.sqlite")
            for name in DATA_DIRS:
                src = prod["data"] / name
                if src.is_dir():
                    copy_tree(src, incoming / name, ignore=())
            for name in DATA_FILES:
                src = prod["data"] / name
                if src.is_file():
                    shutil.copy2(src, incoming / name)

            # Replace consistently: SQLite first, after removing any WAL/SHM
            # files from the old generation.
            target_db = staging["data"] / "store.sqlite"
            remove_sqlite_sidecars(target_db)
            try:
                target_db.unlink()
            except FileNotFoundError:
                pass
            os.replace(incoming / "store.sqlite", target_db)
            for name in DATA_DIRS:
                dst = staging["data"] / name
                src = incoming / name
                if dst.exists():
                    shutil.rmtree(dst)
                if src.is_dir():
                    os.replace(src, dst)
            for name in DATA_FILES:
                dst = staging["data"] / name
                src = incoming / name
                if dst.exists():
                    dst.unlink()
                if src.is_file():
                    os.replace(src, dst)
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
    except Exception:
        # Put the old staging data back when the refresh failed midway, and
        # restart the previous server when it was running.
        if backup is not None and backup.is_dir():
            try:
                _restore_staging_data(staging, backup)
            except Exception:
                pass
        if was_running and staging["binary"].is_file():
            try:
                start_server(staging, state_root, log_path, "", expect_version=False)
            except Exception:
                pass
        raise


# ── deploy / refresh command entry points ──────────────────────────────────

def deploy_main(script_path: str, argv: list[str] | None = None) -> int:
    import argparse
    parser = argparse.ArgumentParser(prog="sift-deploy")
    parser.add_argument("--env", choices=("staging", "prod"), required=True)
    parser.add_argument("--build-only", action="store_true",
                        help="build and publish artifacts without stopping or starting a server")
    parser.add_argument("--refresh-data", action="store_true",
                        help="staging only: explicitly replace staging data from production first")
    args = parser.parse_args(argv)

    app_home, main_repo = repo_roots(script_path)
    cfg = load_config(app_home)
    prod, staging, shared = all_configs(app_home, main_repo, cfg)
    env = prod if args.env == "prod" else staging
    if args.refresh_data and args.env != "staging":
        raise DeployError("--refresh-data is only valid for staging")
    state_root = shared["state_root"]
    state_dir = state_root / env["name"]
    build_id, commit, dirty = build_identity(app_home)
    candidate_id = "%s-%s-%s" % (env["name"], unique_stamp(), build_id[:8])
    candidate = state_dir / "candidates" / candidate_id
    build_log = candidate / "build.log"
    candidate.mkdir(parents=True, exist_ok=False)
    cfg_doc = {"buildId": build_id, "commit": commit, "dirty": dirty,
               "source": str(app_home), "env": env["name"]}

    try:
        write_deploy_state(state_root, env["name"], result="BUILDING",
                           buildId=build_id, commit=commit, dirty=dirty,
                           candidate=str(candidate))
        if not args.build_only:
            owner = probe_port_owner(env, state_root)
            if owner and not args.refresh_data:
                pass  # stop_server will verify ownership immediately before swapping
        build_binary(app_home / "backend", candidate / "sifty", build_id, build_log)
        build_frontend(app_home / "frontend", candidate / "dist", build_log,
                       build_id, shared["frontend_cache"])
        build_workers(shared["workers_source"], candidate / "workers",
                      shared["workers_cache"], build_log)
        write_json_atomic(candidate / "meta.json", cfg_doc)

        if args.build_only:
            recovery = state_dir / "build-recovery" / candidate_id
            publish_artifacts(candidate, env, recovery)
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

        if not env["data"].is_dir():
            raise DeployError(
                "%s data directory %s does not exist; for staging use "
                "--refresh-data to create an explicit snapshot" %
                (env["name"], env["data"]))

        refresh_backup = None
        if args.refresh_data:
            refresh_backup = refresh_staging_data(prod, staging, shared,
                                                  build_log, state_root,
                                                  restart=False)
        else:
            stop_server(env, state_root, env["server_log"])

        recovery = state_dir / "recovery" / candidate_id
        recovery.mkdir(parents=True, exist_ok=True)
        db = env["data"] / "store.sqlite"
        db_existed = sqlite_backup(db, recovery / "store.sqlite")
        write_json_atomic(recovery / "meta.json", {
            "dbExisted": db_existed, "refreshBackup": str(refresh_backup) if refresh_backup else None,
            "refreshRequested": bool(args.refresh_data),
        })

        publish_artifacts(candidate, env, recovery)
        hashes = artifact_hashes(env)
        append_deploy_log(env, "%s env=%s build=%s hash=%s result=DEPLOYING"
                          % (now_stamp(), env["name"], build_id,
                             (hashes["binary"] or "")[:16]))
        try:
            start_server(env, state_root, env["server_log"], build_id)
        except Exception as exc:
            _rollback(env, staging if args.refresh_data else None, state_root,
                      recovery, build_id, str(exc), build_log)
            raise

        write_deploy_state(state_root, env["name"], result="OK",
                           buildId=build_id, commit=commit, dirty=dirty,
                           artifacts=hashes, recovery=str(recovery),
                           paths={"binary": str(env["binary"]), "dist": str(env["dist"]),
                                  "workers": str(env["workers"]), "data": str(env["data"])})
        append_deploy_log(env, "%s env=%s build=%s hash=%s result=OK"
                          % (now_stamp(), env["name"], build_id,
                             (hashes["binary"] or "")[:16]))
        print("sift-deploy: %s deployed %s on port %d" %
              (env["name"], build_id, env["port"]))
        print("sift-deploy: url http://localhost:%d" % env["port"])
        return 0
    except Exception as exc:
        if isinstance(exc, DeployError):
            write_deploy_state(state_root, env["name"], result="FAILED",
                               buildId=build_id, commit=commit, dirty=dirty,
                               error=str(exc))
            append_deploy_log(env, "%s env=%s build=%s result=FAILED error=%s"
                              % (now_stamp(), env["name"], build_id,
                                 str(exc).replace("\n", " ")[:300]))
            print("sift-deploy: ERROR: %s" % exc, file=sys.stderr)
            return 1
        raise


def _rollback(env: dict, refresh_env: dict | None, state_root: Path,
              recovery: Path, build_id: str, error: str, log_path: Path) -> None:
    try:
        stop_server(env, state_root, env["server_log"], timeout=10)
    except Exception:
        pass
    try:
        restore_artifacts(recovery, env)
    except Exception as restore_exc:
        error += "; artifact restore failed: %s" % restore_exc
    meta = read_json(recovery / "meta.json") or {}
    db = env["data"] / "store.sqlite"
    try:
        if meta.get("refreshRequested") and refresh_env is not None:
            backup = Path(meta.get("refreshBackup") or "")
            if backup.is_dir():
                # The refresh backup holds the full pre-refresh staging data tree.
                _restore_staging_data(env, backup)
        else:
            restore_sqlite(recovery / "store.sqlite", db,
                           bool(meta.get("dbExisted")))
    except Exception as restore_exc:
        error += "; data restore failed: %s" % restore_exc
    if env["binary"].is_file():
        try:
            start_server(env, state_root, env["server_log"], "",
                         expect_version=False, timeout=20)
        except Exception as restart_exc:
            error += "; previous server restart failed: %s" % restart_exc
    write_deploy_state(state_root, env["name"], result="ROLLBACK",
                       buildId=build_id, error=error, recovery=str(recovery))
    append_deploy_log(env, "%s env=%s build=%s result=ROLLBACK error=%s"
                      % (now_stamp(), env["name"], build_id,
                         error.replace("\n", " ")[:300]))


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
    state_root = shared["state_root"]
    log_path = state_root / "staging" / "refresh.log"
    try:
        backup = refresh_staging_data(prod, staging, shared, log_path, state_root,
                                      restart=not args.no_restart)
        print("sift-data-refresh: staging data refreshed from %s" % prod["data"])
        print("sift-data-refresh: previous staging data preserved at %s" % backup)
        return 0
    except Exception as exc:
        write_deploy_state(state_root, "staging", result="REFRESH_FAILED",
                           error=str(exc))
        print("sift-data-refresh: ERROR: %s" % exc, file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit("_deploy_lib is imported by bin/sift-deploy and bin/sift-data-refresh")
