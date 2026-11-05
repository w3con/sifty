# Forms a commit to this public repository may never carry

This is the census the leak-check scripts read: `commit-msg-check`, `commit-diff-check`, `range-check` and `tree-check` all parse the three fenced lists below. Adding one line to the public-repository list, the literal-string list or the watched-name list changes what every one of those checks rejects without a line of their own code changing. The pattern-shaped forms (cloud keys, personal access tokens, API keys, JSON Web Tokens, absolute paths, and so on) are matched by regular expressions inside `leak_census.py`, because a regular expression is not a list a census entry can grow.

Every check draws the same line. An **irreversible** form is rejected in any repository, public or private, because once it reaches a server it cannot be recalled: a private key, a cloud or provider credential, a JSON Web Token, the literal contents of a `.env` file, the address of a non-public server. An **untidy** form is rejected only in a repository the public-repository list below names: an internal reference, a private-repository address, a tool-authorship trailer, an absolute local path, Russian-language prose. The same untidy content sitting in the private repository is where it belongs and passes there.

The scanner's own directory, `scripts/leakcheck/`, is exempt from the untidy forms, because the census and the engine legitimately spell out the very forms they exist to catch. Irreversible forms are still refused there, so a real secret pasted into the scanner is caught like anywhere else.

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

## Public subtrees

A subtree declared here is treated as public in every repository, regardless of that repository's own origin remote. Added lines under a public subtree, and the message of any commit that touches a public subtree, are held to the full public rules; a commit that touches only private paths is checked for irreversible forms only. This lets the private repository enforce the public rules on code destined for the public one before it is ever pushed there.

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

A private key in a PEM block; a cloud access key (`AKIA`/`ASIA` prefix); a personal access token (`ghp_`, `github_pat_` and the other GitHub prefixes); a cloud API key (Google `AIza` prefix); a Slack token; a payment-provider secret key (`sk_live_`/`sk_test_`); a model-provider API key (`sk-`, `sk-ant-`); a JSON Web Token (three base64url segments beginning `eyJ`); any line that assigns a real value to a variable named for a secret; the literal contents of a `.env` file; the address of a non-public server. A placeholder value (`API_KEY=your-key-here`, `<token>`, `${VAR}`, `changeme`) is not a secret and passes.
