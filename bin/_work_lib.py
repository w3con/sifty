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

# A card has four normal states. Rejection and cancellation are archived
# unsuccessful outcomes, not successful completion. question.md is a blocker,
# not a terminal outcome.
NORMAL_STATES = ("backlog", "in progress", "review", "done")
OUTCOME_FILES = {
    "done.md": "done",
    "rejected.md": "rejected",
    "cancelled.md": "cancelled",
}
TERMINAL_FILES = tuple(OUTCOME_FILES)
BLOCKER_FILE = "question.md"
STATUS_LINE_RE = re.compile(r"^\s*\*\*Status:\*\*\s*(.*)$", re.IGNORECASE)
BLOCKER_LINE_RE = re.compile(r"^\s*\*\*Blocker:\*\*\s*(.*)$", re.IGNORECASE)
KIND_LINE_RE = re.compile(r"^\s*\*\*Kind:\*\*\s*([A-Za-z]+)\s*$",
                          re.IGNORECASE)
HEADER_LINE_RE = re.compile(r"^\s*\*\*[^*]+:\*\*")

# Legacy prose statuses map onto the four normal states, except `descoped`,
# which is an archived cancellation. Anything else is reported, never guessed.
STATE_ALIASES = {
    "backlog": "backlog",
    "founded": "backlog",
    "approved": "backlog",
    "new": "backlog",
    "idea": "backlog",
    "bug": "backlog",
    "in progress": "in progress",
    "in-progress": "in progress",
    "started": "in progress",
    "wip": "in progress",
    "review": "review",
    "in review": "review",
    "done": "done",
    "complete": "done",
    "completed": "done",
    "rejected": "rejected",
    "cancelled": "cancelled",
    "canceled": "cancelled",
    "descoped": "cancelled",
    "abandoned": "cancelled",
}


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
    return os.path.join(repo_root(), "work", "tasks")


def legacy_tasks_dir():
    """The pre-relocation card location, still read during the 3.2 migration.

    `work/tasks` is canonical. The fallback lets the same card remain
    resumable while its folder is being moved and keeps old links readable.
    """
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


def ignored_status_lines(cwd=None):
    """Ignored local material, one porcelain line each (for preservation)."""
    cp = run(["git", "status", "--porcelain", "--ignored=matching",
              "--untracked-files=normal"], cwd=cwd)
    if cp.returncode != 0:
        return ["<git status --ignored failed: %s>"
                % (cp.stderr or cp.stdout).strip()]
    return [line for line in cp.stdout.splitlines() if line.startswith("!! ")]


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
                "WORK_BASE_BRANCH": base_branch(),
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


CARD_FILES = ("description.md", "plan.md", "log.md", "comments.md",
              "attention.md", "done.md", "rejected.md", "cancelled.md",
              "question.md")


def is_card_dir(path):
    """True when a folder carries any card file (an empty leftover is not)."""
    return any(os.path.exists(os.path.join(path, name)) for name in CARD_FILES)


def find_card_dirs(card_id, root=None):
    """Return every canonical card folder for an ID.

    Canonical `work/tasks/` wins over the pre-relocation `ai/tasks/` fallback.
    Two folders with the same ID in the same location are ambiguous and are a
    refusal, never a silent first match. Folders without description.md are
    not cards (for example an empty leftover directory)."""
    root = root or repo_root()
    for tasks in (os.path.join(root, "work", "tasks"),
                  os.path.join(root, "ai", "tasks")):
        if not os.path.isdir(tasks):
            continue
        found = []
        for epic in sorted(os.listdir(tasks)):
            epic_path = os.path.join(tasks, epic)
            if not os.path.isdir(epic_path):
                continue
            for entry in sorted(os.listdir(epic_path)):
                m = CARD_DIR_RE.match(entry)
                if not m or m.group(1) != card_id:
                    continue
                path = os.path.join(epic_path, entry)
                if is_card_dir(path):
                    found.append(path)
        if found:
            if len(found) > 1:
                refuse([
                    "duplicate card ID %r: %s" % (
                        card_id,
                        ", ".join(os.path.relpath(p, root) for p in found)),
                ])
            return found
    return []


def find_card_dir(card_id, root=None):
    """Return the absolute card folder for an ID, or None.

    `root` defaults to the current repository. Pass a worktree root to read the
    card state that lives on the work branch rather than the base checkout."""
    found = find_card_dirs(card_id, root=root)
    return found[0] if found else None


def require_card_dir(card_id):
    d = find_card_dir(card_id)
    if not d:
        die("no card folder found for id %r under work/tasks/ "
            "(legacy ai/tasks/ is also checked)" % card_id)
    return d


def iter_cards(root=None):
    """Every canonical card under `work/tasks/`, or the legacy fallback."""
    root = root or repo_root()
    tasks = os.path.join(root, "work", "tasks")
    if not os.path.isdir(tasks):
        tasks = os.path.join(root, "ai", "tasks")
    cards = []
    if not os.path.isdir(tasks):
        return cards
    for epic in sorted(os.listdir(tasks)):
        epic_path = os.path.join(tasks, epic)
        if not os.path.isdir(epic_path):
            continue
        for entry in sorted(os.listdir(epic_path)):
            m = CARD_DIR_RE.match(entry)
            if not m:
                continue
            path = os.path.join(epic_path, entry)
            if not is_card_dir(path):
                continue
            cards.append({
                "id": m.group(1),
                "path": path,
                "epic": epic,
                "slug": entry.split("_", 1)[1] if "_" in entry else "",
            })
    return cards


def duplicate_card_ids(root=None):
    """Mapping of duplicated IDs to their card paths (empty when unique)."""
    by_id = {}
    for card in iter_cards(root=root):
        by_id.setdefault(card["id"], []).append(card["path"])
    return {card_id: paths for card_id, paths in by_id.items()
            if len(paths) > 1}


def card_sort_key(card):
    m = re.match(r"^([A-Z0-9]+)-(\d+)$", card["id"])
    return (m.group(1), int(m.group(2))) if m else (card["id"], 0)


def is_short_card(card_dir):
    return not os.path.exists(os.path.join(card_dir, "plan.md"))


def steps_file(card_dir):
    """The file that holds the card's steps: plan.md (full) or description.md
    (short)."""
    if is_short_card(card_dir):
        return os.path.join(card_dir, "description.md")
    return os.path.join(card_dir, "plan.md")


def allocate_id(prefix):
    """One past the highest existing number for this prefix under work/tasks/,
    also counting legacy ai/tasks/ so a migrated card never gets a reused ID."""
    highest = 0
    pat = re.compile(r"^%s-(\d+)_" % re.escape(prefix))
    for root in (tasks_dir(), legacy_tasks_dir()):
        if not os.path.isdir(root):
            continue
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
# A checkbox-looking line, used to catch malformed steps such as `- [ ]`.
STEP_MARK_RE = re.compile(r"^\s*-\s*\[[ xX]?\]")


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


def malformed_step_lines(card_dir):
    """Checkbox-looking lines that are not a valid open or closed step."""
    out = []
    for line in read_lines(steps_file(card_dir)):
        if not STEP_MARK_RE.match(line):
            continue
        mo = OPEN_STEP_RE.match(line)
        mc = CLOSED_STEP_RE.match(line)
        if mo and mo.group(1).strip():
            continue
        if mc and mc.group(1).strip():
            continue
        out.append(line.strip())
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


# ── Task state inference ────────────────────────────────────────────────────
def parse_state_token(value):
    """Map a legacy or canonical Status value to a state, or None if unknown."""
    raw = (value or "").strip()
    if not raw:
        return None
    # The leading words before a separator carry the state; the rest is legacy
    # prose such as "backlog; preparation not started" or "founded (deferred)".
    m = re.match(r"^([A-Za-z][A-Za-z -]*?)(?=\s*[;(—–]|\s*$)", raw)
    token = (m.group(1) if m else raw).strip().lower()
    token = re.sub(r"\s+", " ", token)
    return STATE_ALIASES.get(token)


def card_title(card_dir):
    for line in read_lines(os.path.join(card_dir, "description.md")):
        if line.startswith("# "):
            title = line[2:].strip()
            m = re.match(r"^[A-Z0-9]+-\d+\s*[—-]\s*(.*)$", title)
            return m.group(1).strip() if m else title
    return os.path.basename(card_dir)


def question_reason(card_dir):
    """The first substantive line of question.md, or None when absent."""
    path = os.path.join(card_dir, BLOCKER_FILE)
    if not os.path.exists(path):
        return None
    for line in read_lines(path):
        text = line.strip()
        if text and not text.startswith("#") and not text.startswith("- [x]"):
            return text
    return "unanswered question in question.md"


def read_state_metadata(card_dir):
    statuses, blockers, kinds = [], [], []
    for line in read_lines(os.path.join(card_dir, "description.md")):
        m = STATUS_LINE_RE.match(line)
        if m:
            statuses.append(m.group(1).strip())
        m = BLOCKER_LINE_RE.match(line)
        if m:
            blockers.append(m.group(1).strip())
        m = KIND_LINE_RE.match(line)
        if m:
            kinds.append(m.group(1).lower())
    return {"statuses": statuses, "blockers": blockers, "kinds": kinds}


def card_state(card_dir):
    """Infer one card's state, blockers and next step.

    Precedence, documented in bin/templates/convention.md:
      1. An outcome file is authoritative: done.md, rejected.md or cancelled.md.
      2. question.md, **Blocker:** and open attention items block a nonterminal
         card without changing its state.
      3. description.md's leading **Status:** value, with legacy aliases.
      4. No metadata means backlog.
    Contradictions (several outcomes, an outcome plus a blocker, a status
    naming an outcome without its evidence) are returned in `errors` and are
    refused by the commands rather than guessed around.
    """
    info = {
        "card_dir": card_dir,
        "errors": [],
        "warnings": [],
        "blockers": [],
        "attention": open_attention(card_dir),
        "kind": "task",
        "state": "backlog",
        "state_source": "legacy default (no outcome file or **Status:**)",
        "status_value": None,
        "next_step": None,
        "steps_open": 0,
        "steps_closed": 0,
        "terminal": False,
        "successful": False,
    }
    meta = read_state_metadata(card_dir)
    if len(meta["statuses"]) > 1:
        info["errors"].append("multiple **Status:** lines in description.md")
    if len(meta["blockers"]) > 1:
        info["errors"].append("multiple **Blocker:** lines in description.md")
    if len(meta["kinds"]) > 1:
        info["errors"].append("multiple **Kind:** lines in description.md")

    explicit = None
    if meta["statuses"]:
        info["status_value"] = meta["statuses"][0]
        explicit = parse_state_token(info["status_value"])
        if explicit is None:
            info["errors"].append(
                "unknown **Status:** value %r (expected backlog, in progress, "
                "review, done, rejected or cancelled)"
                % info["status_value"])

    if meta["kinds"]:
        kind = meta["kinds"][0]
        if kind in ("bug", "idea", "task"):
            info["kind"] = kind
        else:
            info["errors"].append(
                "unknown **Kind:** value %r (expected bug, idea or task)"
                % kind)

    outcomes = [name for name in TERMINAL_FILES
                if os.path.exists(os.path.join(card_dir, name))]
    if len(outcomes) > 1:
        info["errors"].append(
            "multiple terminal outcomes: %s" % ", ".join(outcomes))
    outcome = OUTCOME_FILES[outcomes[0]] if len(outcomes) == 1 else None
    if outcome:
        info["state"] = outcome
        info["state_source"] = outcomes[0]
        info["terminal"] = True
        info["successful"] = outcome == "done"
        if explicit in ("done", "rejected", "cancelled") and explicit != outcome:
            info["errors"].append(
                "**Status:** %r contradicts %s"
                % (info["status_value"], outcomes[0]))
    elif explicit:
        info["state"] = explicit
        info["state_source"] = "**Status:**"
        if explicit in ("done", "rejected", "cancelled"):
            legacy_descoped = bool(re.match(
                r"^\s*descoped\b", info["status_value"] or "", re.IGNORECASE))
            if legacy_descoped and explicit == "cancelled":
                # Older cards recorded only the prose "descoped" decision. The
                # archived outcome is inferred; a cancelled.md can be added later.
                info["state_source"] = "legacy **Status:** descoped"
                info["terminal"] = True
                info["successful"] = False
            else:
                expected = {
                    "done": "done.md",
                    "rejected": "rejected.md",
                    "cancelled": "cancelled.md",
                }[explicit]
                info["errors"].append(
                    "**Status:** %r requires %s; no terminal outcome file found"
                    % (info["status_value"], expected))

    for reason in meta["blockers"]:
        if reason:
            info["blockers"].append("**Blocker:** " + reason)
    unanswered = question_reason(card_dir)
    if unanswered:
        info["blockers"].append("question.md: " + unanswered)
    if info["terminal"] and info["blockers"]:
        info["errors"].append(
            "terminal outcome %s cannot carry a blocker: %s"
            % (info["state"], "; ".join(info["blockers"])))
    if info["terminal"] and info["attention"]:
        info["errors"].append(
            "terminal outcome %s still has open attention" % info["state"])

    if os.path.exists(steps_file(card_dir)):
        steps = all_steps(card_dir)
        info["steps_open"] = sum(1 for status, _ in steps if status == "open")
        info["steps_closed"] = sum(1 for status, _ in steps
                                   if status == "closed")
        if not info["terminal"]:
            opens = [text for status, text in steps if status == "open"]
            info["next_step"] = opens[0] if opens else None
    return info


def state_transition(card_dir, requested):
    """Return reasons a state transition is refused. Empty = allowed."""
    info = card_state(card_dir)
    problems = list(info["errors"])
    if requested == "done":
        problems.append("done is written only by work-finish after its checks; "
                        "run bin/work-finish <id>")
        return problems
    if requested in ("rejected", "cancelled"):
        problems.append("%s is an archived outcome; use bin/work-finish <id> "
                        "--reject <reason> or bin/work-cancel <id> --reason "
                        "<reason>" % requested)
        return problems
    if requested not in NORMAL_STATES:
        problems.append("unknown state %r" % requested)
    if info["terminal"]:
        problems.append("card already has terminal outcome %s; archived "
                        "evidence is not reopened" % info["state"])
    return problems


def rewrite_status_metadata(card_dir, state=None, block=None, unblock=False):
    """Update explicit status/blocker metadata in description.md.

    Only the metadata lines change; all other prose is preserved. Returns True
    when the file changed."""
    path = os.path.join(card_dir, "description.md")
    lines = read_lines(path)
    status_idxs = [i for i, line in enumerate(lines) if STATUS_LINE_RE.match(line)]
    blocker_idxs = [i for i, line in enumerate(lines) if BLOCKER_LINE_RE.match(line)]
    if len(status_idxs) > 1 or len(blocker_idxs) > 1:
        refuse(["cannot rewrite contradictory metadata in %s"
                % os.path.relpath(path, repo_root())])
    changed = False

    if state is not None:
        new_line = "**Status:** %s" % state
        if status_idxs:
            idx = status_idxs[0]
            if lines[idx] != new_line:
                lines[idx] = new_line
                changed = True
        else:
            header = [i for i, line in enumerate(lines[:20])
                      if HEADER_LINE_RE.match(line)]
            idx = header[-1] if header else 0
            lines.insert(idx + 1, new_line)
            changed = True

    if block is not None:
        new_line = "**Blocker:** %s" % block
        if blocker_idxs:
            idx = blocker_idxs[0]
            if lines[idx] != new_line:
                lines[idx] = new_line
                changed = True
        else:
            status_idxs = [i for i, line in enumerate(lines)
                           if STATUS_LINE_RE.match(line)]
            if status_idxs:
                idx = status_idxs[0]
            else:
                header = [i for i, line in enumerate(lines[:20])
                          if HEADER_LINE_RE.match(line)]
                idx = header[-1] if header else 0
            lines.insert(idx + 1, new_line)
            changed = True

    if unblock and blocker_idxs:
        for idx in sorted(blocker_idxs, reverse=True):
            del lines[idx]
        changed = True

    if changed:
        with open(path, "w", encoding="utf-8") as fh:
            fh.write("\n".join(lines).rstrip("\n") + "\n")
    return changed


def commit_paths(paths, message, cwd=None):
    """Commit only `paths`, preserving unrelated staged and unstaged work."""
    cwd = cwd or repo_root()
    rels = [os.path.relpath(path, cwd) for path in paths]
    add = run(["git", "add", "--"] + rels, cwd=cwd)
    if add.returncode != 0:
        die("git add failed: %s" % (add.stderr or add.stdout).strip())
    has_head = run(["git", "rev-parse", "--verify", "--quiet", "HEAD"],
                   cwd=cwd).returncode == 0
    if has_head:
        cp = run(["git", "commit", "-q", "--only", "-m", message, "--"] + rels,
                 cwd=cwd)
    else:
        # A repository without HEAD cannot take a partial commit; this is the
        # initial card in a fresh clone.
        cp = run(["git", "commit", "-q", "-m", message], cwd=cwd)
    if cp.returncode != 0:
        die("git commit failed: %s" % (cp.stderr or cp.stdout).strip())


def fenced_block(text):
    """Fence `text` without changing its bytes (choose a longer fence)."""
    runs = [len(m) for m in re.findall(r"`+", text)] or [0]
    fence = "`" * max(3, max(runs) + 1)
    return "%stext\n%s\n%s" % (fence, text, fence)


def words_in_card(card_dir, words):
    """True when the exact owner wording already appears in the card."""
    for name in ("description.md", "comments.md"):
        path = os.path.join(card_dir, name)
        if os.path.exists(path):
            with open(path, "r", encoding="utf-8") as fh:
                if words in fh.read():
                    return True
    return False


def matching_cards_for_words(words, root=None):
    return [card for card in iter_cards(root=root)
            if words_in_card(card["path"], words)]


# ── Worktree / base navigation ──────────────────────────────────────────────
def base_checkout_root():
    """The checkout that carries the base branch, or the current root."""
    root = repo_root()
    wt = find_worktree(base_branch(), cwd=root)
    return wt or root


def worktree_info(card_id, root=None):
    """Describe the active work/<id> worktree and how it differs from base."""
    root = root or repo_root()
    branch = "work/%s" % card_id
    wt = find_worktree(branch, cwd=root)
    if not wt or not os.path.isdir(wt):
        return None
    base_root = base_checkout_root()
    base = base_branch()
    tip = git_rev(branch, cwd=root)
    base_tip = git_rev(base, cwd=base_root)
    ahead = behind = 0
    if tip and base_tip:
        cp = run(["git", "rev-list", "--left-right", "--count",
                  "%s...%s" % (base_tip, tip)], cwd=root)
        if cp.returncode == 0:
            parts = cp.stdout.split()
            if len(parts) == 2:
                behind, ahead = int(parts[0]), int(parts[1])
    dirty = git_status_lines(wt)
    wt_card = find_card_dir(card_id, root=wt)
    base_card = find_card_dir(card_id, root=base_root)
    wt_state = card_state(wt_card) if wt_card else None
    base_state = card_state(base_card) if base_card else None
    state_differs = bool(wt_state and base_state
                         and wt_state["state"] != base_state["state"])
    differs = (tip != base_tip) or bool(dirty) or state_differs
    return {
        "path": wt,
        "display_path": os.path.relpath(wt, base_root),
        "branch": branch,
        "tip": tip,
        "base": base,
        "base_tip": base_tip,
        "ahead": ahead,
        "behind": behind,
        "dirty": dirty,
        "wt_state": wt_state,
        "base_state": base_state,
        "state_differs": state_differs,
        "differs": differs,
    }


def worktree_note(info):
    if not info:
        return None
    short_tip = (info["tip"] or "?")[:7]
    parts = ["worktree %s @ %s" % (info["display_path"], short_tip)]
    if info["ahead"] or info["behind"]:
        parts.append("%d ahead / %d behind %s"
                     % (info["ahead"], info["behind"], info["base"]))
    if info["dirty"]:
        parts.append("dirty working copy")
    if info["state_differs"]:
        parts.append("worktree state %s (base %s)"
                     % (info["wt_state"]["state"],
                        info["base_state"]["state"]))
    if info["differs"]:
        base_short = (info["base_tip"] or "?")[:7]
        parts.append("differs from base %s @ %s" % (info["base"], base_short))
    else:
        parts.append("at base")
    return "; ".join(parts)


# ── Plan fitness (shared by work-plan and work-handover) ─────────────────────
def plan_fitness_problems(card_dir):
    """Return a list of reasons the plan is not fit to hand over. Empty = fit."""
    problems = []
    sf = steps_file(card_dir)
    if not os.path.exists(sf):
        problems.append("missing steps file (plan.md for a full card, "
                        "description.md for a short card)")
        return problems
    text = "\n".join(read_lines(sf))
    low = text.lower()
    if not text.strip():
        problems.append("plan is empty")
    if not all_steps(card_dir):
        problems.append("no steps found (need at least one `- [ ]` line in %s)"
                        % os.path.basename(sf))
    malformed = malformed_step_lines(card_dir)
    if malformed:
        problems.append("malformed step line(s): %s"
                        % "; ".join(malformed[:3]))
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
            # A missing, empty, no-step or malformed plan is unfinished work
            # even when open_steps() returns nothing.
            problems.extend(plan_fitness_problems(card_dir))
            open_steps_found = open_steps(card_dir)
            if open_steps_found:
                problems.append("plan still has %d open step(s): %s"
                                % (len(open_steps_found),
                                   "; ".join(open_steps_found[:3])))
            attention = open_attention(card_dir)
            if attention:
                problems.append("unresolved attention (%d open item(s))"
                                % len(attention))
            state = card_state(card_dir)
            problems.extend(state["errors"])
            for blocker in state["blockers"]:
                problems.append("unresolved blocker: %s" % blocker)
            terminal = terminal_file_present(card_dir)
            target_card = find_card_dir(card_id, root=target_root)
            if target_card and target_card != card_dir:
                # Rejection/cancellation write the outcome on the base card, not
                # on the work branch, so both copies must be checked. A blocker
                # recorded on the base copy (for example question.md) also stops
                # acceptance and merging.
                target_state = card_state(target_card)
                problems.extend(target_state["errors"])
                for blocker in target_state["blockers"]:
                    problems.append("unresolved blocker on base card: %s"
                                    % blocker)
                if not terminal:
                    terminal = terminal_file_present(target_card)
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
        ignored = ignored_status_lines(wt)
        if any(not line.startswith("!! ") for line in ignored):
            problems.append(
                "could not inspect ignored material in %s; preserved "
                "worktree and branch" % wt)
            return problems
        if ignored:
            problems.append(
                "worktree %s contains ignored local material (%d entry/entries: "
                "%s); preserved worktree and branch"
                % (wt, len(ignored), ", ".join(ignored[:5])))
            return problems
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
