#!/usr/bin/env python3
"""leak_census — reads the census at scripts/leakcheck/census.md and scans a block of text (a
commit message, or the added lines of a diff, or a whole file) for the forms it names.

Shared by commit-msg-check, commit-diff-check, range-check and tree-check, so the lists the
census carries by hand — the public-repository list, the literal-string list and the watched-
name list — are read in exactly one place, and a line added to any of them changes what every
one of those commands rejects without a line of this module's own code changing.

The pattern-shaped forms (a cloud key, a personal access token, a JSON Web Token, an absolute
path, and so on) are NOT read from the census: a regular expression is not a list a census
entry can grow, so each one is a small, named function below instead.
"""
import os
import re
import subprocess
from collections import namedtuple

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_CENSUS_PATH = os.path.join(SCRIPT_DIR, "census.md")

Finding = namedtuple("Finding", ["form", "cls", "line_no", "line_text", "hint"])

_CARD_DIR_RE = re.compile(r"^([A-Z]+)-\d+(?:_.*)?$")


def discover_card_prefixes(repo_root=None):
    """Every card-identifier prefix this repository's own ai/tasks/ tree actually uses, read
    from the card folder names themselves rather than hard-coded, so a new epic's prefix is
    caught the day its first card folder exists with no code change here. Returns an empty set
    when ai/tasks does not exist — as in a standalone public clone that carries no cards."""
    if repo_root is None:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            capture_output=True, text=True)
        if result.returncode != 0:
            return set()
        repo_root = result.stdout.strip()
    tasks_root = os.path.join(repo_root, "ai", "tasks")
    if not os.path.isdir(tasks_root):
        return set()
    prefixes = set()
    for epic_name in os.listdir(tasks_root):
        epic_dir = os.path.join(tasks_root, epic_name)
        if not os.path.isdir(epic_dir):
            continue
        for card_name in os.listdir(epic_dir):
            m = _CARD_DIR_RE.match(card_name)
            if m:
                prefixes.add(m.group(1))
    return prefixes


def _read_fenced_block(text, heading):
    """The lines inside the first fenced code block (```...```) that follows a `## heading`
    line, one per output line, blank lines dropped. None when the heading or the block after
    it is not found."""
    heading_re = re.compile(r"^## " + re.escape(heading) + r"\s*$", re.MULTILINE)
    m = heading_re.search(text)
    if not m:
        return None
    rest = text[m.end():]
    fence_m = re.search(r"```\n(.*?)```", rest, re.DOTALL)
    if not fence_m:
        return None
    return [line for line in fence_m.group(1).splitlines() if line.strip()]


def parse_census(census_path=DEFAULT_CENSUS_PATH):
    """Reads the census document once and returns the lists each check needs: the public
    repositories (matched against a repository's own `origin`), the public subtrees (paths
    held to the full public rules even inside a private repository), the literal strings not
    recognised by any pattern, and the names watched only when "secret" or "key" sits on the
    same line."""
    with open(census_path, encoding="utf-8") as fh:
        text = fh.read()
    public_repos = _read_fenced_block(text, "Public repositories") or []
    public_subtrees = _read_fenced_block(text, "Public subtrees") or []
    private_owners = _read_fenced_block(text, "Private repository owners") or []
    literal_strings = _read_fenced_block(text, "Literal strings not recognized by pattern") or []
    watched_names = _read_fenced_block(
        text, "Names watched only when a secret or a key is mentioned nearby") or []
    return {
        "public_repos": public_repos,
        "public_subtrees": public_subtrees,
        "private_owners": private_owners,
        "literal_strings": literal_strings,
        "watched_names": watched_names,
    }


def is_public_subtree(file_path, census):
    """True when file_path is under a subtree the census marks as public. The subtree list
    entries may carry a trailing slash or not; both forms match correctly."""
    for subtree in census.get("public_subtrees", []):
        prefix = subtree if subtree.endswith("/") else subtree + "/"
        if file_path.startswith(prefix) or file_path == subtree.rstrip("/"):
            return True
    return False


def is_public_repo(origin_url, census):
    """True when a repository's own `origin` remote is one the census's public-repository list
    names. A repository absent from that list is treated as private, so only an irreversible
    form is refused there."""
    if not origin_url:
        return False
    return origin_url.strip() in census["public_repos"]


# ── Irreversible, pattern-shaped forms ──────────────────────────────────────────────────────
# A secret of any of these shapes cannot be recalled once it reaches a server, so each is
# refused in every repository, public or private.

_PEM_KEY_RE = re.compile(r"-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----")
# Cloud and provider keys with a distinctive, self-identifying prefix.
_AWS_KEY_RE = re.compile(r"\b(?:AKIA|ASIA)[0-9A-Z]{16}\b")
_GITHUB_PAT_RE = re.compile(
    r"\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[0-9A-Za-z_]{22,})\b")
_GOOGLE_KEY_RE = re.compile(r"\bAIza[0-9A-Za-z_\-]{35}\b")
_SLACK_TOKEN_RE = re.compile(r"\bxox[baprs]-[0-9A-Za-z-]{10,}\b")
_STRIPE_KEY_RE = re.compile(r"\bsk_(?:live|test)_[0-9A-Za-z]{16,}\b")
_OPENAI_KEY_RE = re.compile(r"\bsk-(?:proj-)?[A-Za-z0-9_-]{20,}\b")
_ANTHROPIC_KEY_RE = re.compile(r"\bsk-ant-[A-Za-z0-9_-]{20,}\b")
# A JSON Web Token: three base64url segments, the first the base64 of a JSON object ("eyJ").
_JWT_RE = re.compile(r"\beyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b")

# A generic "name a secret and assign it a real value" line. Tightened against false positives:
# the value must be long and secret-shaped, and must not be an obvious placeholder.
_SECRET_ASSIGN_RE = re.compile(
    r"""(?ix)
    \b(?:api[_-]?key|secret(?:[_-]?key)?|access[_-]?key|auth[_-]?token|
       client[_-]?secret|private[_-]?key|password|passwd|passphrase)\b
    \s*[:=]\s*
    (?P<q>["']?)(?P<val>[^\s"']{12,})(?P=q)
    """)
_PLACEHOLDER_RE = re.compile(
    r"(?i)(example|your[_-]?|xxx+|<[^>]+>|\$\{?[A-Za-z_]|changeme|placeholder|redacted|"
    r"dummy|sample|todo|fixme|\*\*\*|\.\.\.|here)")

# End-anchored on purpose: a `.env` line is the whole line. A quoted value may hold spaces, so
# both quoting styles are accepted as the whole value.
_ENV_LINE_RE = re.compile(r"""^[A-Z][A-Z0-9_]{2,}=(?:"[^"]*"|'[^']*'|\S*)$""")

# A non-public server named by an SSH connection string.
_SSH_IP_RE = re.compile(r"\b[a-z][a-z0-9_-]{1,31}@\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b")
_SSH_ALIAS_RE = re.compile(
    r"(?<![/\w.-])[a-z][a-z0-9_-]{1,31}@(?!v\d)[a-z][a-z0-9-]{1,31}\b(?!\.[a-zA-Z])")


def _looks_like_placeholder(value):
    return bool(_PLACEHOLDER_RE.search(value))


def scan_line_irreversible(line):
    """Every irreversible-class finding this line carries. Returns a list of (form, hint)."""
    hits = []
    if _PEM_KEY_RE.search(line):
        hits.append(("private key (PEM block)",
                      "a key never enters a commit; generate it and hand it to whoever needs "
                      "it outside any repository"))
    for name, rx in (("cloud access key", _AWS_KEY_RE),
                     ("personal access token", _GITHUB_PAT_RE),
                     ("cloud API key", _GOOGLE_KEY_RE),
                     ("Slack token", _SLACK_TOKEN_RE),
                     ("payment-provider secret key", _STRIPE_KEY_RE),
                     ("model-provider API key", _OPENAI_KEY_RE),
                     ("model-provider API key", _ANTHROPIC_KEY_RE),
                     ("JSON Web Token", _JWT_RE)):
        if rx.search(line):
            hits.append((name,
                          "store the credential outside the repository and read it from the "
                          "environment or a local file the .gitignore excludes"))
    m = _SECRET_ASSIGN_RE.search(line)
    if m and not _looks_like_placeholder(m.group("val")):
        hits.append(("secret assigned a real value",
                      "name the variable without its value; read the real one from the "
                      "environment or a local file the .gitignore excludes"))
    if _ENV_LINE_RE.match(line.strip()):
        hits.append((".env file content",
                      "name the variable in .env.example without its real value"))
    if _SSH_IP_RE.search(line) or _SSH_ALIAS_RE.search(line):
        hits.append(("address or hostname of a non-public server",
                      "refer to the host by its role, not its address"))
    return hits


# ── Untidy, pattern-shaped forms ────────────────────────────────────────────────────────────
# Internal references that are harmless in the private repository but do not belong in the
# public one. Refused only where the census marks a repository public.

_RUSSIAN_RE = re.compile(r"[Ѐ-ӿ]")
# A string or regex literal: a backtick, double-quoted or single-quoted span. Cyrillic inside one
# is load-bearing matching logic (the code recognising Russian user content), not internal prose,
# so it is exempt; Cyrillic in a comment or in prose, outside every literal, is still flagged.
_LITERAL_SPAN_RE = re.compile(r"`[^`]*`|\"(?:[^\"\\]|\\.)*\"|'(?:[^'\\]|\\.)*'")
_WORK_COMMAND_RE = re.compile(r"\bbin/work-[a-z][a-z-]*\b")
_TRAILER_RE = re.compile(r"^(Co-Authored-By|Claude-Session):", re.MULTILINE)
_GENERATED_WITH_RE = re.compile(r"Generated with \[Claude|🤖 Generated with")
_ABS_PATH_RE = re.compile(r"(?:^|[\s(`\"'])(/(?:Users|Volumes)/\S+)")
# A github reference to a repository under the owner's org; flagged unless the census lists the
# full clone address as public.
_ORG_REPO_REF_RE = re.compile(
    r"(?:git@github\.com:|github\.com/)([\w-]+)/([\w-]+?)(?:\.git)?(?=[\s\)\]\.,;:'\"]|$)")


def _cyrillic_outside_literals(line):
    """True when the line carries Cyrillic outside every string or regex literal. Cyrillic inside
    a literal is the code matching Russian user content and is allowed; Cyrillic in a comment or
    in prose is internal text and is flagged."""
    masked = _LITERAL_SPAN_RE.sub("", line)
    return bool(_RUSSIAN_RE.search(masked))


def _card_identifier_hints(line, prefixes):
    """Return a hint string when `line` names a card identifier built from one of `prefixes`,
    or None when it does not. Case-insensitive, so identifiers written in lowercase in a
    commit message subject line are caught as well as the uppercase form in prose."""
    if not prefixes:
        return None
    card_id_re = re.compile(
        r"\b(?:" + "|".join(re.escape(p) for p in prefixes) + r")-\d+(?:\.[A-Za-z0-9]+)*\b",
        re.IGNORECASE)
    m = card_id_re.search(line)
    if m:
        return f"remove the identifier {m.group(0)!r}; keep the sentence it was attached to"
    return None


def scan_line_untidy(line, prefixes, census):
    """Every untidy-class finding this line carries. Returns a list of (form, hint)."""
    hits = []
    card_hint = _card_identifier_hints(line, prefixes)
    if card_hint:
        hits.append(("card identifier", card_hint))
    if _cyrillic_outside_literals(line):
        hits.append(("Russian-language text", "write the same sentence in English"))
    if _WORK_COMMAND_RE.search(line):
        hits.append(("reference to the private work-management harness",
                      "drop it; the public repository does not carry the harness"))
    if _TRAILER_RE.search(line) or _GENERATED_WITH_RE.search(line):
        hits.append(("tool-authorship trailer",
                      "remove the trailer; authorship is the committer's own git identity"))
    if _ABS_PATH_RE.search(line):
        hits.append(("absolute filesystem path",
                      "say what the path pointed at, without the path itself"))
    for m in _ORG_REPO_REF_RE.finditer(line):
        owner, name = m.group(1), m.group(2)
        full = f"git@github.com:{owner}/{name}.git"
        alt = f"github.com/{owner}/{name}"
        is_public = full in census["public_repos"] or alt in census["public_repos"]
        if owner in census["private_owners"] and not is_public:
            hits.append(("reference to a private repository",
                          "name the kind of information without naming its address"))
            break
    for literal in census["literal_strings"]:
        if literal and literal in line:
            hits.append((f"internal literal {literal!r}",
                          "see this form's own entry in the census for what replaces it"))
    lowered = line.lower()
    if "secret" in lowered or re.search(r"\bkey\b", lowered):
        for name in census["watched_names"]:
            if name and re.search(r"\b" + re.escape(name) + r"\b", line):
                hits.append((f"person's name ({name}) next to the mention of a secret",
                              "drop the name; keep the rest of the sentence"))
    return hits


def scan_text(text, census, prefixes, is_public, allow_env_shape=False):
    """Scans every line of `text` and returns a list of Finding, in line order. Irreversible
    forms are always checked; untidy forms are checked only when `is_public` is true.
    `allow_env_shape` skips the .env-content form for a file meant to carry KEY=VALUE shapes
    without real values (.env.example)."""
    findings = []
    for line_no, line in enumerate(text.splitlines(), start=1):
        for form, hint in scan_line_irreversible(line):
            if form == ".env file content" and allow_env_shape:
                continue
            findings.append(Finding(form, "irreversible", line_no, line, hint))
        if is_public:
            for form, hint in scan_line_untidy(line, prefixes, census):
                findings.append(Finding(form, "untidy", line_no, line, hint))
    return findings


def format_findings(findings, source_label):
    """One block of text naming every finding: the form, the file/message it came from, the
    line, and what replaces it — never a silent refusal."""
    lines = []
    for f in findings:
        lines.append(f"{source_label}:{f.line_no}: [{f.cls}] {f.form}")
        lines.append(f"    {f.line_text.strip()}")
        lines.append(f"    replace with: {f.hint}")
    return "\n".join(lines)
