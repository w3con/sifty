"""Shared helpers for the model-agnostic work-* harness.

Standard library only. No dependency on any Claude-Code-only mechanism.
Every refusal lives in the command that calls these helpers, not in an
external git hook.
"""

import os
import re
import subprocess
import sys
from datetime import date

# ── Exit codes ──────────────────────────────────────────────────────────────
EXIT_OK = 0
EXIT_REFUSED = 2          # a refusal fired (plan not fit, attention open, ...)
EXIT_USAGE = 64           # bad arguments
EXIT_HANDED_TO_SESSION = 97  # the prompt was printed for a live session to run

TERMINAL_FILES = ("done.md", "rejected.md", "cancelled.md", "question.md")


# ── Process / repo helpers ──────────────────────────────────────────────────
def run(args, cwd=None, check=False, capture=True):
    """Run a command, returning CompletedProcess. Text mode."""
    return subprocess.run(
        args,
        cwd=cwd,
        check=check,
        text=True,
        stdout=subprocess.PIPE if capture else None,
        stderr=subprocess.PIPE if capture else None,
    )


def repo_root():
    cp = run(["git", "rev-parse", "--show-toplevel"])
    if cp.returncode != 0:
        die("not inside a git repository")
    return cp.stdout.strip()


def tasks_dir():
    return os.path.join(repo_root(), "ai", "tasks")


def templates_dir():
    return os.path.join(repo_root(), "bin", "templates")


def today():
    return date.today().isoformat()


def git_current_branch(cwd=None):
    cp = run(["git", "rev-parse", "--abbrev-ref", "HEAD"], cwd=cwd)
    return cp.stdout.strip() if cp.returncode == 0 else ""


def find_worktree(branch):
    """Absolute path of the worktree checked out on `branch`, or None."""
    cp = run(["git", "worktree", "list", "--porcelain"])
    path = None
    for line in cp.stdout.splitlines():
        if line.startswith("worktree "):
            path = line[len("worktree "):]
        elif line.startswith("branch "):
            if line[len("branch "):] == "refs/heads/%s" % branch:
                return path
    return None


def run_accept(card_id):
    """Run the acceptance checks against work/<id> in a throwaway scratch copy.

    Returns (returncode, combined_output). The check command is read from
    WORK_ACCEPT_CMD (env or bin/work.config), defaulting to bin/work-test, and
    is run with the scratch copy as its working directory. The scratch worktree
    is always removed, pass or fail."""
    branch = "work/%s" % card_id
    if not find_worktree(branch):
        return (3, "no worktree for %s — hand it over first" % branch)
    scratch = os.path.join(repo_root(), ".work-scratch", card_id)
    if os.path.exists(scratch):
        run(["git", "worktree", "remove", "--force", scratch])
    os.makedirs(os.path.dirname(scratch), exist_ok=True)
    add = run(["git", "worktree", "add", "--detach", scratch, branch])
    if add.returncode != 0:
        return (3, "could not create scratch copy: %s" % add.stderr.strip())
    try:
        cmd = config_get("WORK_ACCEPT_CMD", os.path.join("bin", "work-test"))
        proc = subprocess.run(
            cmd, shell=True, text=True, cwd=scratch,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        return (proc.returncode, proc.stdout or "")
    finally:
        run(["git", "worktree", "remove", "--force", scratch])


# ── Output / refusal helpers ────────────────────────────────────────────────
def die(msg, code=EXIT_USAGE):
    sys.stderr.write("work: %s\n" % msg)
    sys.exit(code)


def refuse(reasons):
    """reasons: list[str]. Print each and exit with the refusal code."""
    if isinstance(reasons, str):
        reasons = [reasons]
    sys.stderr.write("REFUSED:\n")
    for r in reasons:
        sys.stderr.write("  - %s\n" % r)
    sys.exit(EXIT_REFUSED)


def info(msg):
    sys.stderr.write(msg + "\n")


# ── Templates ───────────────────────────────────────────────────────────────
def render_template(name, **vars):
    path = os.path.join(templates_dir(), name)
    with open(path, "r", encoding="utf-8") as fh:
        text = fh.read()
    for key, val in vars.items():
        text = text.replace("{{%s}}" % key, str(val))
    return text


# ── Config / dispatch seam ──────────────────────────────────────────────────
def config_path():
    return os.path.join(repo_root(), "bin", "work.config")


def read_config():
    """Simple KEY=VALUE config. Lines starting with # are comments.
    Environment variables of the same name override the file."""
    cfg = {}
    path = config_path()
    if os.path.exists(path):
        with open(path, "r", encoding="utf-8") as fh:
            for line in fh:
                line = line.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                k, v = line.split("=", 1)
                cfg[k.strip()] = v.strip()
    return cfg


def config_get(key, default=None):
    if key in os.environ and os.environ[key] != "":
        return os.environ[key]
    return read_config().get(key, default)


def is_live_session():
    """True when a live agent session (or an interactive shell) is driving.

    Model-agnostic: honours an explicit override, then known session markers,
    then a TTY. No Claude-Code-only assumption — opencode or a plain terminal
    all read as live, and WORK_HEADLESS=1 forces headless for scripted runs."""
    if os.environ.get("WORK_HEADLESS") == "1":
        return False
    if os.environ.get("WORK_SESSION") == "1":
        return True
    for marker in ("CLAUDECODE", "CLAUDE_CODE", "OPENCODE", "OPENCODE_SESSION"):
        if os.environ.get(marker):
            return True
    try:
        return sys.stdout.isatty()
    except Exception:
        return False


def dispatch(prompt):
    """The cross-tool seam.

    Live session: print the prompt and return EXIT_HANDED_TO_SESSION so the
    session raises the agent itself. Headless: run the configured dispatch
    command, feeding the prompt on stdin. The command is read from
    WORK_DISPATCH_CMD (env or bin/work.config); it is never hardcoded, so it
    can be `claude -p`, an opencode invocation, or anything else."""
    if is_live_session():
        sys.stdout.write(prompt)
        if not prompt.endswith("\n"):
            sys.stdout.write("\n")
        info("\n[work] prompt handed to the live session "
             "(exit %d); the session raises the agent." % EXIT_HANDED_TO_SESSION)
        sys.exit(EXIT_HANDED_TO_SESSION)
    cmd = config_get("WORK_DISPATCH_CMD")
    if not cmd:
        refuse([
            "headless dispatch requested but no dispatch command is configured.",
            "Set WORK_DISPATCH_CMD (env or bin/work.config) to e.g. "
            "'claude -p' or an opencode invocation.",
        ])
    proc = subprocess.run(cmd, shell=True, input=prompt, text=True)
    sys.exit(proc.returncode)


# ── Card model ──────────────────────────────────────────────────────────────
SLUG_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
EPIC_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
ID_RE = re.compile(r"^[A-Z0-9]+-\d+$")
CARD_DIR_RE = re.compile(r"^([A-Z0-9]+-\d+)_[a-z0-9-]+$")


def find_card_dir(card_id):
    """Return the absolute card folder for an ID, or None."""
    root = tasks_dir()
    if not os.path.isdir(root):
        return None
    for epic in sorted(os.listdir(root)):
        epic_path = os.path.join(root, epic)
        if not os.path.isdir(epic_path):
            continue
        for entry in sorted(os.listdir(epic_path)):
            m = CARD_DIR_RE.match(entry)
            if m and m.group(1) == card_id:
                return os.path.join(epic_path, entry)
    return None


def require_card_dir(card_id):
    d = find_card_dir(card_id)
    if not d:
        die("no card folder found for id %r under ai/tasks/" % card_id)
    return d


def is_short_card(card_dir):
    return not os.path.exists(os.path.join(card_dir, "plan.md"))


def steps_file(card_dir):
    """The file that holds the card's steps: plan.md (full) or description.md
    (short)."""
    if is_short_card(card_dir):
        return os.path.join(card_dir, "description.md")
    return os.path.join(card_dir, "plan.md")


def allocate_id(prefix):
    """One past the highest existing number for this prefix anywhere under
    ai/tasks/."""
    root = tasks_dir()
    highest = 0
    pat = re.compile(r"^%s-(\d+)_" % re.escape(prefix))
    if os.path.isdir(root):
        for epic in os.listdir(root):
            epic_path = os.path.join(root, epic)
            if not os.path.isdir(epic_path):
                continue
            for entry in os.listdir(epic_path):
                m = pat.match(entry)
                if m:
                    highest = max(highest, int(m.group(1)))
    return "%s-%d" % (prefix, highest + 1)


# ── Reading steps / attention / estimate ────────────────────────────────────
OPEN_STEP_RE = re.compile(r"^\s*-\s*\[\s\]\s+(.*)$")
CLOSED_STEP_RE = re.compile(r"^\s*-\s*\[[xX]\]\s+(.*)$")


def read_lines(path):
    if not os.path.exists(path):
        return []
    with open(path, "r", encoding="utf-8") as fh:
        return fh.read().splitlines()


def open_steps(card_dir):
    return [m.group(1) for line in read_lines(steps_file(card_dir))
            for m in [OPEN_STEP_RE.match(line)] if m]


def all_steps(card_dir):
    sf = steps_file(card_dir)
    out = []
    for line in read_lines(sf):
        mo = OPEN_STEP_RE.match(line)
        mc = CLOSED_STEP_RE.match(line)
        if mo:
            out.append(("open", mo.group(1)))
        elif mc:
            out.append(("closed", mc.group(1)))
    return out


def open_attention(card_dir):
    path = os.path.join(card_dir, "attention.md")
    return [m.group(1) for line in read_lines(path)
            for m in [OPEN_STEP_RE.match(line)] if m]


def terminal_file_present(card_dir):
    for name in TERMINAL_FILES:
        if os.path.exists(os.path.join(card_dir, name)):
            return name
    return None


def read_estimate(card_dir):
    """Read `estimate:` from the plan's YAML frontmatter. Returns int or None."""
    sf = steps_file(card_dir)
    lines = read_lines(sf)
    if not lines or lines[0].strip() != "---":
        return None
    for line in lines[1:]:
        if line.strip() == "---":
            break
        if line.strip().startswith("estimate:"):
            val = line.split(":", 1)[1].strip()
            if val.isdigit():
                return int(val)
    return None


# ── Plan fitness (shared by work-plan and work-handover) ─────────────────────
def plan_fitness_problems(card_dir):
    """Return a list of reasons the plan is not fit to hand over. Empty = fit."""
    problems = []
    sf = steps_file(card_dir)
    if not os.path.exists(sf):
        problems.append("missing %s" % os.path.basename(sf))
        return problems
    text = "\n".join(read_lines(sf))
    low = text.lower()
    if not all_steps(card_dir):
        problems.append("no steps found (need at least one `- [ ]` line in %s)"
                        % os.path.basename(sf))
    if "acceptance criteria" not in low:
        problems.append("no 'Acceptance criteria' section")
    # Unfilled template scaffolding left behind.
    if "<!-- owner:" in low or "read cold by a future session" in low:
        problems.append("template placeholders still present — fill the plan")
    if is_short_card(card_dir):
        if "## why this work exists" in low and low.count("<!--") > 0 and \
           "state plainly what outcome" in low:
            problems.append("short card 'Why this work exists' still a placeholder")
    return problems
