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


def git_rev(ref, cwd=None):
    """Resolve a revision to a full SHA, or None when it does not exist."""
    cp = run(["git", "rev-parse", "--verify", "--quiet", ref], cwd=cwd)
    return cp.stdout.strip() if cp.returncode == 0 else None


def git_status_lines(cwd=None):
    """Non-ignored tracked/untracked changes, one porcelain line each."""
    cp = run(["git", "status", "--porcelain", "--untracked-files=normal"],
             cwd=cwd)
    if cp.returncode != 0:
        return ["<git status failed: %s>" % (cp.stderr or cp.stdout).strip()]
    return [line for line in cp.stdout.splitlines() if line.strip()]


def find_worktree(branch, cwd=None):
    """Absolute path of the worktree checked out on `branch`, or None."""
    cp = run(["git", "worktree", "list", "--porcelain"], cwd=cwd)
    path = None
    for line in cp.stdout.splitlines():
        if line.startswith("worktree "):
            path = line[len("worktree "):]
        elif line.startswith("branch "):
            if line[len("branch "):] == "refs/heads/%s" % branch:
                return path
    return None


def worktree_registered(path, cwd=None):
    """True when `path` is registered as a worktree, even if it is missing."""
    cp = run(["git", "worktree", "list", "--porcelain"], cwd=cwd)
    target = os.path.abspath(path)
    for line in cp.stdout.splitlines():
        if line.startswith("worktree "):
            if os.path.abspath(line[len("worktree "):]) == target:
                return True
    return False


def run_accept(card_id, expected_source_sha=None, expected_target_sha=None,
               expected_target_branch=None):
    """Run acceptance checks against the exact captured source revision.

    Returns (returncode, combined_output, problems). The scratch worktree is
    created detached at the source SHA, never at a moving branch ref, so the
    checks test the revision that was captured before the run. A pre-existing
    scratch path is reported and left untouched; the scratch this call creates
    is removed with non-force `git worktree remove`, and a failed removal is
    reported rather than forced."""
    branch = "work/%s" % card_id
    target_root = repo_root()
    problems = []

    wt = find_worktree(branch, cwd=target_root)
    if not wt or not os.path.isdir(wt):
        message = "no worktree for %s — hand it over first" % branch
        return (3, message, [message])

    source_sha = git_rev(branch, cwd=target_root)
    if not source_sha:
        message = "cannot resolve %s" % branch
        return (3, message, [message])
    if expected_source_sha and source_sha != expected_source_sha:
        problems.append("source revision changed: expected %s, found %s"
                        % (expected_source_sha, source_sha))
    wt_head = git_rev("HEAD", cwd=wt)
    if wt_head != source_sha:
        problems.append("source worktree HEAD %s is not the %s tip %s"
                        % (wt_head, branch, source_sha))

    target_branch = git_current_branch(target_root)
    target_sha = git_rev("HEAD", cwd=target_root)
    if expected_target_branch and target_branch != expected_target_branch:
        problems.append("target branch changed: expected %s, found %s"
                        % (expected_target_branch, target_branch))
    if expected_target_sha and target_sha != expected_target_sha:
        problems.append("target revision changed: expected %s, found %s"
                        % (expected_target_sha, target_sha))

    src_dirty = git_status_lines(wt)
    if src_dirty:
        problems.append("source worktree is dirty (%d path(s)): %s"
                        % (len(src_dirty), ", ".join(src_dirty[:5])))
    tgt_dirty = git_status_lines(target_root)
    if tgt_dirty:
        problems.append("target checkout is dirty (%d path(s)): %s"
                        % (len(tgt_dirty), ", ".join(tgt_dirty[:5])))
    if problems:
        return (3, "\n".join(problems), problems)

    scratch = os.path.join(target_root, ".work-scratch", card_id)
    if os.path.lexists(scratch) or worktree_registered(scratch, cwd=target_root):
        message = ("scratch path %s already exists; refusing to remove a "
                   "pre-existing scratch directory/worktree" % scratch)
        return (3, message, [message])
    os.makedirs(os.path.dirname(scratch), exist_ok=True)

    add = run(["git", "worktree", "add", "--detach", scratch, source_sha],
              cwd=target_root)
    if add.returncode != 0:
        message = "could not create scratch copy: %s" % (
            add.stderr or add.stdout).strip()
        return (3, message, [message])

    code = 0
    output = ""
    try:
        scratch_head = git_rev("HEAD", cwd=scratch)
        if scratch_head != source_sha:
            problems.append("scratch HEAD %s is not the captured source %s"
                            % (scratch_head, source_sha))
        else:
            cmd = config_get("WORK_ACCEPT_CMD", os.path.join("bin", "work-test"))
            env = os.environ.copy()
            env.update({
                "WORK_ACCEPT_SCRATCH": scratch,
                "WORK_ACCEPT_SOURCE_SHA": source_sha,
                "WORK_ACCEPT_SOURCE_BRANCH": branch,
                "WORK_ACCEPT_SOURCE_WORKTREE": wt,
                "WORK_ACCEPT_TARGET_ROOT": target_root,
            })
            proc = subprocess.run(
                cmd, shell=True, text=True, cwd=scratch,
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, env=env)
            code = proc.returncode
            output = proc.stdout or ""

        post_source = git_rev(branch, cwd=target_root)
        post_target_branch = git_current_branch(target_root)
        post_target_sha = git_rev("HEAD", cwd=target_root)
        if post_source != source_sha:
            problems.append("source revision moved during acceptance: %s -> %s"
                            % (source_sha, post_source))
        if post_target_branch != target_branch:
            problems.append("target branch moved during acceptance: %s -> %s"
                            % (target_branch, post_target_branch))
        if post_target_sha != target_sha:
            problems.append("target revision moved during acceptance: %s -> %s"
                            % (target_sha, post_target_sha))
        post_src_dirty = git_status_lines(wt)
        if post_src_dirty:
            problems.append("source worktree became dirty during acceptance "
                            "(%d path(s)): %s"
                            % (len(post_src_dirty), ", ".join(post_src_dirty[:5])))
        post_tgt_dirty = git_status_lines(target_root)
        if post_tgt_dirty:
            problems.append("target checkout became dirty during acceptance "
                            "(%d path(s)): %s"
                            % (len(post_tgt_dirty), ", ".join(post_tgt_dirty[:5])))
        post_scratch_head = git_rev("HEAD", cwd=scratch)
        if post_scratch_head != source_sha:
            problems.append("scratch HEAD moved during acceptance: %s -> %s"
                            % (source_sha, post_scratch_head))
    except Exception as exc:  # noqa: BLE001 - report, never lose the checkpoint
        problems.append("acceptance run failed: %s" % exc)
        if code == 0:
            code = 4
    finally:
        removed = run(["git", "worktree", "remove", scratch], cwd=target_root)
        if removed.returncode != 0:
            problems.append(
                "could not remove scratch worktree without force: %s; "
                "preserved at %s"
                % ((removed.stderr or removed.stdout).strip(), scratch))
            if code == 0:
                code = 4

    if problems:
        output = ((output.rstrip("\n") + "\n\n") if output else "") + \
                 "ACCEPTANCE PROBLEMS:\n" + \
                 "\n".join("- " + p for p in problems) + "\n"
        if code == 0:
            code = 4
    return (code, output, problems)


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


def base_branch():
    """The branch acceptance and finish target and merge into."""
    return config_get("WORK_BASE_BRANCH", "main")


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


def find_card_dir(card_id, root=None):
    """Return the absolute card folder for an ID, or None.

    `root` defaults to the current repository. Pass a worktree root to read the
    card state that lives on the work branch rather than the base checkout."""
    root = root or repo_root()
    tasks = os.path.join(root, "ai", "tasks")
    if not os.path.isdir(tasks):
        return None
    for epic in sorted(os.listdir(tasks)):
        epic_path = os.path.join(tasks, epic)
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


# ── Acceptance / finish preconditions ───────────────────────────────────────
def finish_snapshot(card_id, target_root=None):
    """Capture the source revision, worktree and target state for a finish.

    Returns a dict. `problems` holds anything that prevents trusting the
    snapshot (missing worktree, unresolvable branch, detached source HEAD)."""
    target_root = target_root or repo_root()
    branch = "work/%s" % card_id
    wt = find_worktree(branch, cwd=target_root)
    problems = []
    if not wt or not os.path.isdir(wt):
        problems.append("no worktree for %s — hand it over first" % branch)
        return {
            "branch": branch,
            "worktree": None,
            "source_sha": None,
            "target_root": target_root,
            "target_branch": git_current_branch(target_root),
            "target_sha": git_rev("HEAD", cwd=target_root),
            "problems": problems,
        }
    source_sha = git_rev(branch, cwd=target_root)
    if not source_sha:
        problems.append("cannot resolve %s" % branch)
    else:
        wt_head = git_rev("HEAD", cwd=wt)
        if wt_head != source_sha:
            problems.append("source worktree HEAD %s is not the %s tip %s"
                            % (wt_head, branch, source_sha))
    return {
        "branch": branch,
        "worktree": wt,
        "source_sha": source_sha,
        "target_root": target_root,
        "target_branch": git_current_branch(target_root),
        "target_sha": git_rev("HEAD", cwd=target_root),
        "problems": problems,
    }


def readiness_problems(card_id, target_root=None):
    """Reasons a card cannot be accepted or finished. Empty = ready.

    Checks the source worktree and the target checkout are clean (including
    untracked, non-ignored files), every step is closed, attention is resolved,
    no terminal outcome exists, the source worktree is on the branch tip, and
    the target is the configured base branch."""
    snapshot = finish_snapshot(card_id, target_root=target_root)
    problems = list(snapshot["problems"])
    target_root = snapshot["target_root"]

    wt = snapshot["worktree"]
    if wt:
        card_dir = find_card_dir(card_id, root=wt)
        if not card_dir:
            problems.append("card folder for %s is missing in the source worktree"
                            % card_id)
        else:
            open_steps_found = open_steps(card_dir)
            if open_steps_found:
                problems.append("plan still has %d open step(s): %s"
                                % (len(open_steps_found),
                                   "; ".join(open_steps_found[:3])))
            attention = open_attention(card_dir)
            if attention:
                problems.append("unresolved attention (%d open item(s))"
                                % len(attention))
            terminal = terminal_file_present(card_dir)
            if terminal:
                problems.append("terminal file present: %s" % terminal)
        dirty_source = git_status_lines(wt)
        if dirty_source:
            problems.append("source worktree is dirty (%d path(s)): %s"
                            % (len(dirty_source),
                               ", ".join(dirty_source[:5])))

    expected_base = base_branch()
    if snapshot["target_branch"] != expected_base:
        problems.append("target branch is %r, not %r"
                        % (snapshot["target_branch"], expected_base))
    dirty_target = git_status_lines(target_root)
    if dirty_target:
        problems.append("target checkout is dirty (%d path(s)): %s"
                        % (len(dirty_target), ", ".join(dirty_target[:5])))
    return problems


def cleanup_finished(card_id, source_sha):
    """Remove the finished worktree and branch without force.

    Returns a list of problems. The branch is deleted only after the worktree
    is gone and only when it still points at the merged source revision. A
    failed removal or moved branch leaves both in place for recovery."""
    branch = "work/%s" % card_id
    root = repo_root()
    problems = []
    wt = find_worktree(branch, cwd=root)
    if wt:
        removed = run(["git", "worktree", "remove", wt], cwd=root)
        if removed.returncode != 0:
            problems.append(
                "worktree %s preserved (non-force removal failed): %s"
                % (wt, (removed.stderr or removed.stdout).strip()))
            return problems
    tip = git_rev(branch, cwd=root)
    if tip is None:
        return problems
    if tip != source_sha:
        problems.append("branch %s moved to %s; preserved" % (branch, tip))
        return problems
    deleted = run(["git", "branch", "-d", branch], cwd=root)
    if deleted.returncode != 0:
        problems.append(
            "branch %s preserved (non-force delete failed): %s"
            % (branch, (deleted.stderr or deleted.stdout).strip()))
    return problems
