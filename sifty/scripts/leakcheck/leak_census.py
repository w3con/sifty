#!/usr/bin/env python3
"""leak_census — reads the census at scripts/leakcheck/census.md and scans a block of text (a
commit message, a changed file, or a whole file) for the forms it names.

Shared by commit-msg-check, commit-diff-check, range-check and tree-check, so the lists the
census carries by hand — the public-repository list, the formatting-required subtrees, the
literal-string list and the watched-name list — are read in exactly one place, and a line added
to any of them changes what every one of those commands rejects without a line of this module's
own code changing.

The pattern-shaped forms (a cloud key, a personal access token, a JSON Web Token, an absolute
path, and so on) are NOT read from the census: a regular expression is not a list a census
entry can grow, so each one is a small, named function below instead.

Two groups of rules are deliberately separate:

* **credential/security** forms are irreversible and are checked in every tracked text area —
  application source, tests, scanner sources, commit messages and historical commits;
* **formatting** forms are the repository's publication style. They apply only under the
  formatting-required subtrees the census names (currently the public application tree), not to
  work records, management notes or tooling, which may carry the owner's Russian wording.

The scanner's own directory is exempt from formatting rules only. A real credential pasted into
the scanner is still refused. Pattern definitions in the scanner are written so that they do not
match themselves, rather than by exempting the whole directory from the security rules.
"""
import os
import re
from collections import namedtuple

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_CENSUS_PATH = os.path.join(SCRIPT_DIR, "census.md")

Finding = namedtuple("Finding", ["form", "cls", "line_no", "line_text", "hint"])

# Formatting rules are not applied to the scanner's own sources: the census and the engine
# legitimately spell out the forms they exist to catch. Credential rules still apply there.
FORMATTING_EXEMPT_PREFIXES = ("scripts/leakcheck/", "sifty/scripts/leakcheck/")


def formatting_exempt(file_path):
    """True where formatting rules (Russian prose, private addresses, literals, trailers) should
    not apply. This is a formatting exemption only; a credential in the same file is still
    refused by the security rules."""
    return any(file_path.startswith(prefix) or ("/" + prefix) in file_path
               for prefix in FORMATTING_EXEMPT_PREFIXES)


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
    repositories (matched against a repository's own `origin`), the subtrees held to English
    formatting, the literal strings not recognised by any pattern, and the names watched only
    when "secret" or "key" sits on the same line."""
    with open(census_path, encoding="utf-8") as fh:
        text = fh.read()
    public_repos = _read_fenced_block(text, "Public repositories") or []
    formatting_subtrees = _read_fenced_block(text, "Formatting-required subtrees")
    if formatting_subtrees is None:
        # Older census revisions called this list "Public subtrees".
        formatting_subtrees = _read_fenced_block(text, "Public subtrees") or []
    private_owners = _read_fenced_block(text, "Private repository owners") or []
    literal_strings = _read_fenced_block(text, "Literal strings not recognized by pattern") or []
    watched_names = _read_fenced_block(
        text, "Names watched only when a secret or a key is mentioned nearby") or []
    return {
        "public_repos": public_repos,
        "formatting_subtrees": formatting_subtrees,
        "public_subtrees": formatting_subtrees,
        "private_owners": private_owners,
        "literal_strings": literal_strings,
        "watched_names": watched_names,
    }


def is_formatting_required(file_path, census):
    """True when file_path is under a subtree the census holds to English publication
    formatting. The subtree list entries may carry a trailing slash or not; both forms match."""
    for subtree in census.get("formatting_subtrees", []):
        prefix = subtree if subtree.endswith("/") else subtree + "/"
        if file_path.startswith(prefix) or file_path == subtree.rstrip("/"):
            return True
    return False


# Backwards-compatible name used by older callers and documentation.
def is_public_subtree(file_path, census):
    return is_formatting_required(file_path, census)


def is_public_repo(origin_url, census):
    """True when a repository's own `origin` remote is one the census's public-repository list
    names. A repository absent from that list is treated as private."""
    if not origin_url:
        return False
    return origin_url.strip() in census["public_repos"]


# ── Credential/security forms ────────────────────────────────────────────────────────────────
# A secret of any of these shapes cannot be recalled once it reaches a server, so each is
# refused in every repository and every tracked text area, public or private.

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
# the value must be a secret-shaped run (letters, digits and the usual base64 punctuation), not
# an arbitrary code expression or a whole minified JavaScript object.
_SECRET_ASSIGN_RE = re.compile(
    r"""(?ix)
    \b(?:api[_-]?key|secret(?:[_-]?key)?|access[_-]?key|auth[_-]?token|
       client[_-]?secret|private[_-]?key|password|passwd|passphrase)\b
    \s*[:=]\s*
    (?P<q>["']?)(?P<val>[A-Za-z0-9_+/=.-]{12,})(?P=q)
    """)
_PLACEHOLDER_RE = re.compile(
    r"(?i)(example|your[_-]?|xxx+|<[^>]+>|\$\{?[A-Za-z_]|changeme|placeholder|redacted|"
    r"dummy|sample|todo|fixme|\*\*\*|\.\.\.|here)")

# A credential embedded in a URL: a scheme, then a user and a secret before the host. The
# separator is assembled so this definition line does not look like a credential URL to itself.
_CREDENTIAL_URL_RE = re.compile(
    r"\b[a-z][a-z0-9+.-]*" + "://" + r"[^\s:/@]+:(?P<pw>[^\s/@]{4,})@")

# A non-public server named by an SSH connection string.
_SSH_IP_RE = re.compile(r"\b[a-z][a-z0-9_-]{1,31}@\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b")
_SSH_ALIAS_RE = re.compile(
    r"(?<![/\w.-])[a-z][a-z0-9_-]{1,31}@(?!v\d)[a-z][a-z0-9-]{1,31}(?![A-Za-z0-9_.\\-])")


def _looks_like_placeholder(value):
    return bool(_PLACEHOLDER_RE.search(value))


def _looks_like_code_expression(line, match):
    """Reject a match that is a function call or a dotted property access, not a value.
    This keeps source such as `password: decodeURIComponent(proxy.password)` from reading as an
    assigned credential while a real literal value is still refused."""
    if line[match.end("val"):match.end("val") + 1] == "(":
        return True
    return not match.group("q") and "." in match.group("val")


def scan_line_irreversible(line):
    """Every credential/security finding this line carries. Returns a list of (form, hint)."""
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
    if (m and not _looks_like_placeholder(m.group("val"))
            and not _looks_like_code_expression(line, m)):
        hits.append(("secret assigned a real value",
                      "name the variable without its value; read the real one from the "
                      "environment or a local file the .gitignore excludes"))
    m = _CREDENTIAL_URL_RE.search(line)
    if m and not _looks_like_placeholder(m.group("pw")):
        hits.append(("credentials in a URL",
                      "refer to the service by role and read its credentials from the "
                      "environment; never commit user:password in a URL"))
    if _SSH_IP_RE.search(line) or _SSH_ALIAS_RE.search(line):
        hits.append(("address or hostname of a non-public server",
                      "refer to the host by its role, not its address"))
    return hits


# ── Formatting forms ─────────────────────────────────────────────────────────────────────────
# Publication style, not a security rule. Applied only where the census requires English
# formatting; work records, management notes and tooling may keep the owner's Russian wording.

_RUSSIAN_RE = re.compile(r"[Ѐ-ӿ]")
# A string or regex literal: a backtick, double-quoted or single-quoted span. Cyrillic inside one
# is load-bearing matching logic (the code recognising Russian user content), not internal prose,
# so it is exempt; Cyrillic in a comment or in prose, outside every literal, is still flagged.
_LITERAL_SPAN_RE = re.compile(r"`[^`]*`|\"(?:[^\"\\]|\\.)*\"|'(?:[^'\\]|\\.)*'")
_TRAILER_RE = re.compile(r"^(Co-Authored-By|Claude-Session):", re.MULTILINE)
_GENERATED_WITH_RE = re.compile(r"Generated with \[Claude|🤖 Generated with")
_ABS_PATH_RE = re.compile(r"(?:^|[\s(`\"'])(/(?:Users|Volumes)/\S+)")
# A github reference to a repository under the owner's org; flagged unless the census lists the
# full clone address as public.
_ORG_REPO_REF_RE = re.compile(
    r"(?:git" + "@" + r"github\.com:|github\.com/)([\w-]+)/([\w-]+?)(?:\.git)?"
    r"(?=[\s\)\]\.,;:'\"]|$)")


def _cyrillic_outside_literals(line):
    """True when the line carries Cyrillic outside every string or regex literal. Cyrillic inside
    a literal is the code matching Russian user content and is allowed; Cyrillic in a comment or
    in prose is internal text and is flagged."""
    masked = _LITERAL_SPAN_RE.sub("", line)
    return bool(_RUSSIAN_RE.search(masked))


def scan_line_formatting(line, census):
    """Every formatting finding this line carries. Returns a list of (form, hint)."""
    hits = []
    if _cyrillic_outside_literals(line):
        hits.append(("Russian-language text", "write the same sentence in English"))
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
            hits.append(("internal literal named by the census",
                          "see this form's own entry in the census for what replaces it"))
    lowered = line.lower()
    if "secret" in lowered or re.search(r"\bkey\b", lowered):
        for name in census["watched_names"]:
            if name and re.search(r"\b" + re.escape(name) + r"\b", line):
                hits.append(("person's name next to the mention of a secret",
                              "drop the name; keep the rest of the sentence"))
    return hits


def scan_text(text, census, apply_formatting=False):
    """Scans every line of `text` and returns a list of Finding, in line order. Credential forms
    are always checked; formatting forms are checked only when `apply_formatting` is true."""
    findings = []
    for line_no, line in enumerate(text.splitlines(), start=1):
        for form, hint in scan_line_irreversible(line):
            findings.append(Finding(form, "irreversible", line_no, line, hint))
        if apply_formatting:
            for form, hint in scan_line_formatting(line, census):
                findings.append(Finding(form, "formatting", line_no, line, hint))
    return findings


def redact_sensitive(text):
    """Remove every credential-shaped match from text before it can reach a diagnostic. Used for
    paths and source labels as well as matched lines: a secret in a filename is still a secret."""
    text = _PEM_KEY_RE.sub("[redacted]", text)
    for rx in (_AWS_KEY_RE, _GITHUB_PAT_RE, _GOOGLE_KEY_RE, _SLACK_TOKEN_RE,
               _STRIPE_KEY_RE, _OPENAI_KEY_RE, _ANTHROPIC_KEY_RE, _JWT_RE,
               _SSH_IP_RE, _SSH_ALIAS_RE):
        text = rx.sub("[redacted]", text)
    text = _SECRET_ASSIGN_RE.sub(
        lambda m: m.group(0).replace(m.group("val"), "[redacted]"), text)
    text = _CREDENTIAL_URL_RE.sub(
        lambda m: m.group(0).replace(m.group("pw"), "[redacted]"), text)
    return text


def format_findings(findings, source_label):
    """One line per finding: the form, the safe file/message label and the line number. Matched
    values and whole offending lines are never printed, and the label is redacted too."""
    label = redact_sensitive(str(source_label))
    return "\n".join(f"{label}:{f.line_no}: [{f.cls}] {f.form}" for f in findings)
