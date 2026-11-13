#!/usr/bin/env python3
"""leak_git — strict Git plumbing for the leak scanner.

Every path listing is NUL-separated, so spaces, quotes and non-ASCII names are carried as one
argument rather than parsed out of human-readable diff headers. Blob contents are read from the
index or the commit being checked, never from the working tree: an unstaged edit must not look
like staged content, and a historical scan must read the history it names.

Any unexpected Git failure raises GitError. The commands let that reach their caller, which
prints the failure and exits 2, so a broken repository or a bad revision can never be reported as
a clean scan.
"""
import subprocess


class GitError(RuntimeError):
    pass


def decode_path(raw):
    return raw.decode("utf-8", "replace")


def run(repo, args, text=False, allow=(0,)):
    """Run one Git command. `allow` lists extra exit codes the caller understands (for example,
    `git config --get` exits 1 when the key is absent). Anything else is a hard error."""
    result = subprocess.run(["git", "-C", repo] + list(args),
                            capture_output=True, text=text)
    if result.returncode not in allow:
        error = result.stderr
        if isinstance(error, bytes):
            error = error.decode("utf-8", "replace")
        raise GitError("git %s failed: %s"
                       % (" ".join(args), (error or "").strip()))
    return result


def origin_url(repo):
    result = run(repo, ["config", "--get", "remote.origin.url"],
                 text=True, allow=(0, 1))
    return result.stdout.strip() if result.returncode == 0 else ""


def parse_name_status_z(raw):
    """Parse `git diff --name-status -z` output into (status, old_path, new_path) entries.
    For additions, modifications, type changes and unmerged entries old_path is None; for a
    rename or copy old_path is the source and new_path is the destination."""
    tokens = raw.split(b"\0")
    entries = []
    i = 0
    while i < len(tokens):
        if not tokens[i]:
            i += 1
            continue
        status = tokens[i].decode("ascii", "replace")
        i += 1
        if not status:
            continue
        if status[0] in "RC":
            if i + 1 >= len(tokens):
                raise GitError("truncated rename entry in Git name-status output")
            old_path = decode_path(tokens[i])
            new_path = decode_path(tokens[i + 1])
            i += 2
            entries.append((status, old_path, new_path))
        else:
            if i >= len(tokens):
                raise GitError("truncated entry in Git name-status output")
            path = decode_path(tokens[i])
            i += 1
            entries.append((status, None, path))
    return entries


def changed_paths_staged(repo):
    result = run(repo, ["diff", "--cached", "--find-renames",
                        "--name-status", "-z", "--"])
    return parse_name_status_z(result.stdout)


def changed_paths_commit(repo, sha):
    run(repo, ["rev-parse", "--verify", sha + "^{commit}"], text=True)
    result = run(repo, ["diff-tree", "--root", "-r", "-m", "--no-commit-id",
                        "--find-renames", "--name-status", "-z", sha])
    return parse_name_status_z(result.stdout)


def entry_paths(entry):
    """Both sides of a changed path. A rename out of a formatting-required area still touches
    that area for the purpose of checking the commit message."""
    _status, old_path, new_path = entry
    return [path for path in (old_path, new_path) if path]


def new_path(entry):
    return entry[2]


def index_blobs(repo):
    """Map every stage-0 index path to (mode, blob SHA). Symlinks and gitlinks keep their mode
    so the caller can skip them without following either one."""
    result = run(repo, ["ls-files", "-s", "-z"])
    blobs = {}
    for record in result.stdout.split(b"\0"):
        if not record:
            continue
        meta, path = record.split(b"\t", 1)
        parts = meta.split(b" ")
        if len(parts) != 3:
            continue
        mode, sha, stage = parts
        if stage != b"0" or sha == b"0" * 40:
            continue
        blobs[decode_path(path)] = (mode.decode("ascii", "replace"),
                                    sha.decode("ascii", "replace"))
    return blobs


def tree_blobs(repo, rev):
    """Map every blob path in a commit's tree to (mode, blob SHA). Directories, submodules and
    symlinks are not blobs (or are skipped by mode) and are never recursed into."""
    result = run(repo, ["ls-tree", "-r", "-z", rev])
    blobs = {}
    for record in result.stdout.split(b"\0"):
        if not record:
            continue
        meta, path = record.split(b"\t", 1)
        parts = meta.split(b" ")
        if len(parts) != 3:
            continue
        mode, kind, sha = parts
        if kind != b"blob":
            continue
        blobs[decode_path(path)] = (mode.decode("ascii", "replace"),
                                    sha.decode("ascii", "replace"))
    return blobs


def read_blob(repo, sha):
    """Read a blob exactly as Git stores it. Binary content is skipped by returning None."""
    result = run(repo, ["cat-file", "blob", sha])
    raw = result.stdout
    if b"\0" in raw[:8000]:
        return None
    try:
        return raw.decode("utf-8")
    except UnicodeDecodeError:
        return None


def is_symlink(mode):
    return mode == "120000"


def is_submodule(mode):
    return mode == "160000"
