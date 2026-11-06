# Forms a commit to this public repository may never carry

This is the census the leak-check scripts read: `commit-msg-check`, `commit-diff-check`, `range-check` and `tree-check` all parse the fenced lists below. Adding one line to the public-repository list, the literal-string list or the watched-name list changes what every one of those checks rejects without a line of their own code changing. The pattern-shaped forms (cloud keys, personal access tokens, API keys, JSON Web Tokens, absolute paths, and so on) are matched by regular expressions inside `leak_census.py`, because a regular expression is not a list a census entry can grow.

The rules are split into two groups. A **credential/security** form is irreversible and is refused in every repository and every tracked text area, public or private: application source, tests, scanner sources, commit messages and historical commits. A **formatting** form is publication style and is checked only under a formatting-required subtree below. Work records, management notes and tooling may keep the owner's Russian wording; current application source and public technical documentation are held to English.

Task identifiers (`OPS-4`, `GO-11`) and `bin/work-*` command names are ordinary project content. They are not secrets, private-harness references or formatting findings, including in commit scopes and messages that mention a step.

The scanner's own directory, `scripts/leakcheck/`, is exempt from formatting forms only, because the census and the engine legitimately spell out the very forms they exist to catch. Credential forms are still refused there, so a real secret pasted into the scanner is caught like anywhere else. Core pattern definitions in the scanner are written so they do not match their own source; that is a narrow construction rule, not a security exemption for the directory.

## Why three hooks and a manual sweep

`git commit --no-verify` skips the `commit-msg` and `pre-commit` hooks on the one commit being made, and `git push --no-verify` skips `pre-push` on the whole range. A single hook is a speed bump, not a wall. So `commit-msg` and `pre-commit` catch an ordinary commit; `pre-push` re-scans every commit about to leave the machine, catching whatever was committed with `--no-verify` or before the hooks were installed; and `tree-check` is the manual sweep that reads every tracked file, catching a form that entered the tree before the hooks existed and is nobody's added line any more. `tree-check` is run by hand before the first push.

## Public repositories

Checked against a repository's own `origin` remote, read from the repository being committed to, never assumed from where the check runs. A repository absent from this list is treated as private, so only irreversible forms are refused there.

```
git@github.com:w3con/sifty.git
github.com/w3con/sifty
https://github.com/w3con/sifty.git
https://github.com/w3con/sifty
```

## Formatting-required subtrees

A subtree declared here is held to English publication formatting in every repository, regardless of that repository's own origin remote. Current application source and public technical documentation live here. The same files are checked for credential forms wherever they live. Work records under `ai/` and future `work/`, management files and tooling are deliberately outside this list, so the owner's Russian wording is allowed there. The scanner's own sources are exempt from formatting forms as described above.

```
sifty/
```

## Private repository owners

GitHub account or organisation names whose repositories are private by default. A `github.com/<owner>/<repo>` reference is flagged as a private-repository address when its owner is listed here and the full address is not on the public-repository list above, so the owner's private repositories are caught while a public third-party dependency under any other owner (a Go module, for example) passes. The public repository's own address (`w3con/sifty`) is on the list above and so is never flagged.

```
w3con
```

## Literal strings not recognized by pattern

Internal names from the private repository the public one must not carry. Replace each with a neutral term: the data directory and its layout are configured, not named after the private project.

```
FCTO-Catalogue
FCTO_STATE_DIR
FCTO-Catalogue/1.0
FCTO_RECORD
fcto-job-catalogue
```

## Names watched only when a secret or a key is mentioned nearby

A person's name is flagged only when the word "secret" or "key" sits on the same line, so an ordinary mention of the owner passes while a line handing someone a credential is caught.

```
Alex Tikonoff
```

## Irreversible forms, by shape

A private key in a PEM block; a cloud access key (`AKIA`/`ASIA` prefix); a personal access token (`ghp_`, `github_pat_` and the other GitHub prefixes); a cloud API key (Google `AIza` prefix); a Slack token; a payment-provider secret key (`sk_live_`/`sk_test_`); a model-provider API key (`sk-`, `sk-ant-`); a JSON Web Token (three base64url segments beginning `eyJ`); a real value assigned to a variable named for a secret; credentials embedded in a URL (a scheme followed by a user, a secret and a host); the address of a non-public server.

An environment assignment is not automatically a secret. Benign settings such as `WORK_SESSION=1`, `SIFTY_PROD_PORT=8080` and `WORK_PUSH=0` pass, and so do placeholders (`API_KEY=your-key-here`, `<token>`, `${VAR}`, `changeme`). A secret-named variable assigned a real value fails in every path, as does a URL that carries a real user:password pair.
