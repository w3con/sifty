# GO-3 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-04 — 1.1

Check: `python3 -c "
import sys, tempfile, os
sys.path.insert(0, 'sifty/scripts/leakcheck')
import leak_census
prefixes = leak_census.discover_card_prefixes('.')
expected = {'GO', 'APP', 'SET'}
missing = expected - prefixes
assert not missing, f'Missing prefixes {missing}; got {prefixes}'
print('OK: prefixes from this repo =', sorted(prefixes))
with tempfile.TemporaryDirectory() as d:
    p = leak_census.discover_card_prefixes(d)
    assert p == set(), f'Expected empty set, got {p}'
    print('OK: empty set for scratch dir without ai/tasks')
"`
Exit: 0

```
OK: prefixes from this repo = ['APP', 'GO', 'SET']
OK: empty set for scratch dir without ai/tasks
```


## 2026-11-04 — 1.2

Check: `
SCRATCH=$(mktemp -d /private/tmp/go3-step12-XXXXXX)
WORKTREE=/Volumes/Documents/Work/FCTO/.worktrees/GO-3
mkdir -p $SCRATCH/repo/sifty/scripts/leakcheck $SCRATCH/repo/ai/tasks/go-migration/GO-1_test $SCRATCH/repo/ai $SCRATCH/repo/sifty
cp $WORKTREE/sifty/scripts/leakcheck/*.py $WORKTREE/sifty/scripts/leakcheck/commit-* $SCRATCH/repo/sifty/scripts/leakcheck/
cd $SCRATCH/repo
git init -q && git config user.email t@t.com && git config user.name T && git remote add origin git@github.com:w3con/fcto-private.git
git add sifty/ ai/ && git commit -q -m 'chore: initial'
echo 'see GO-1 for details' > sifty/test.go && git add sifty/test.go
python3 sifty/scripts/leakcheck/commit-diff-check --staged --repo . && echo FAIL_sifty_should_refuse || echo OK_sifty_refused
git restore --staged sifty/test.go && rm sifty/test.go
echo 'see GO-1 for details' > ai/note.md && git add ai/note.md
python3 sifty/scripts/leakcheck/commit-diff-check --staged --repo . && echo OK_ai_passed || echo FAIL_ai_should_pass
rm -rf $SCRATCH
`
Exit: 0

```
Traceback (most recent call last):
  File "/private/tmp/go3-step12-qbHvkb/repo/sifty/scripts/leakcheck/commit-diff-check", line 199, in <module>
    sys.exit(main())
             ^^^^^^
  File "/private/tmp/go3-step12-qbHvkb/repo/sifty/scripts/leakcheck/commit-diff-check", line 171, in main
    census = leak_census.parse_census()
             ^^^^^^^^^^^^^^^^^^^^^^^^^^
  File "/private/tmp/go3-step12-qbHvkb/repo/sifty/scripts/leakcheck/leak_census.py", line 75, in parse_census
    with open(census_path, encoding="utf-8") as fh:
         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
FileNotFoundError: [Errno 2] No such file or directory: '/private/tmp/go3-step12-qbHvkb/repo/sifty/scripts/leakcheck/census.md'
OK_sifty_refused
Traceback (most recent call last):
  File "/private/tmp/go3-step12-qbHvkb/repo/sifty/scripts/leakcheck/commit-diff-check", line 199, in <module>
    sys.exit(main())
             ^^^^^^
  File "/private/tmp/go3-step12-qbHvkb/repo/sifty/scripts/leakcheck/commit-diff-check", line 171, in main
    census = leak_census.parse_census()
             ^^^^^^^^^^^^^^^^^^^^^^^^^^
  File "/private/tmp/go3-step12-qbHvkb/repo/sifty/scripts/leakcheck/leak_census.py", line 75, in parse_census
    with open(census_path, encoding="utf-8") as fh:
         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
FileNotFoundError: [Errno 2] No such file or directory: '/private/tmp/go3-step12-qbHvkb/repo/sifty/scripts/leakcheck/census.md'
FAIL_ai_should_pass
```


## 2026-11-04 — 1.3

Check: `
SCRATCH=$(mktemp -d)
WORKTREE=/Volumes/Documents/Work/FCTO/.worktrees/GO-3
LCHECK_SRC=$WORKTREE/sifty/scripts/leakcheck
mkdir -p $SCRATCH/repo/sifty/scripts/leakcheck $SCRATCH/repo/ai/tasks/go-migration/GO-1_test $SCRATCH/repo/ai $SCRATCH/repo/sifty
cp $LCHECK_SRC/census.md $LCHECK_SRC/*.py $LCHECK_SRC/commit-msg-check $SCRATCH/repo/sifty/scripts/leakcheck/
cd $SCRATCH/repo
git init -q && git config user.email t@t.com && git config user.name T && git remote add origin git@github.com:w3con/fcto-private.git
echo private > ai/readme.md && echo pkg > sifty/main.go && git add sifty/ ai/ && git commit -q -m 'chore: initial'
LCHECK=$SCRATCH/repo/sifty/scripts/leakcheck
echo 'package sifty' > sifty/pkg.go && git add sifty/pkg.go
MSG=$(mktemp); printf 'chore(sifty): see GO-1 for details' > $MSG
python3 $LCHECK/commit-msg-check $MSG --repo . && echo FAIL_1 || echo OK_1_refused
git restore --staged sifty/pkg.go && rm sifty/pkg.go $MSG
echo 'private note' > ai/note.md && git add ai/note.md
MSG=$(mktemp); printf 'chore: see GO-1 for details' > $MSG
python3 $LCHECK/commit-msg-check $MSG --repo . && echo OK_3_passed || echo FAIL_3
rm -rf $SCRATCH
`
Exit: 0

```
commit-msg-check: refused — /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.57mopDQkfH carries 1 form this repository's commit messages must not:
/var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.57mopDQkfH:1: [untidy] card identifier
    chore(sifty): see GO-1 for details
    replace with: remove the identifier 'GO-1'; keep the sentence it was attached to
OK_1_refused
OK_3_passed
```


## 2026-11-04 — 1.4

Check: `
SCRATCH=$(mktemp -d)
LCHECK_SRC=/Volumes/Documents/Work/FCTO/.worktrees/GO-3/sifty/scripts/leakcheck
mkdir -p $SCRATCH/repo/sifty/scripts/leakcheck $SCRATCH/repo/ai/tasks/go-migration/GO-1_test $SCRATCH/repo/ai $SCRATCH/repo/sifty
cp $LCHECK_SRC/census.md $LCHECK_SRC/*.py $LCHECK_SRC/commit-msg-check $LCHECK_SRC/commit-diff-check $LCHECK_SRC/range-check $SCRATCH/repo/sifty/scripts/leakcheck/
cd $SCRATCH/repo
git init -q && git config user.email t@t.com && git config user.name T && git remote add origin git@github.com:w3con/fcto-private.git
echo init > README.md && git add README.md && git commit -q -m 'chore: initial'
BASE=$(git rev-parse HEAD)
echo private > ai/note.md && git add ai/note.md && git commit -q -m 'chore: see GO-1 for details'
echo pkg > sifty/pkg.go && git add sifty/pkg.go && git commit -q -m 'chore(sifty): see GO-1 for details'
LCHECK=$SCRATCH/repo/sifty/scripts/leakcheck
python3 $LCHECK/range-check ${BASE}..HEAD --repo . && echo FAIL_1_sifty_should_refuse || echo OK_1_refused
git reset --hard HEAD~ -q
python3 $LCHECK/range-check ${BASE}..HEAD --repo . && echo OK_2_private_passed || echo FAIL_2_private_should_pass
rm -rf $SCRATCH
`
Exit: 0

```
range-check: refused — 1 of 2 commit(s) carry a form this repository must not push:
commit e9f52656c0e4:
commit-msg-check: refused — /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp7ghwzvpk.msg carries 1 form this repository's commit messages must not:
/var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp7ghwzvpk.msg:1: [untidy] card identifier
    chore(sifty): see GO-1 for details
    replace with: remove the identifier 'GO-1'; keep the sentence it was attached to

OK_1_refused
range-check: 1 commit(s) checked, clean
OK_2_private_passed
```


## 2026-11-04 — 2.1

Check: `
SCRATCH=$(mktemp -d)
WORKTREE=/Volumes/Documents/Work/FCTO/.worktrees/GO-3
mkdir -p $SCRATCH/repo/sifty/scripts/leakcheck $SCRATCH/repo/ai/tasks/go-migration/GO-1_test $SCRATCH/repo/ai $SCRATCH/repo/sifty $SCRATCH/repo/.githooks/pre-commit.d
touch $SCRATCH/repo/ai/tasks/go-migration/GO-1_test/.keep
cp $WORKTREE/sifty/scripts/leakcheck/census.md $WORKTREE/sifty/scripts/leakcheck/*.py $WORKTREE/sifty/scripts/leakcheck/commit-msg-check $WORKTREE/sifty/scripts/leakcheck/commit-diff-check $SCRATCH/repo/sifty/scripts/leakcheck/
cp $WORKTREE/.githooks/pre-commit $WORKTREE/.githooks/commit-msg $SCRATCH/repo/.githooks/
cp $WORKTREE/.githooks/pre-commit.d/30-leak-check $SCRATCH/repo/.githooks/pre-commit.d/
chmod +x $SCRATCH/repo/.githooks/commit-msg $SCRATCH/repo/.githooks/pre-commit.d/30-leak-check
cd $SCRATCH/repo
git init -q && git config user.email t@t.com && git config user.name T && git config core.hooksPath .githooks && git remote add origin git@github.com:w3con/fcto-private.git
git add sifty/ ai/ .githooks/ && git commit -q --no-verify -m 'chore: initial'
printf 'see GO-1 for details\n' > sifty/bad.txt && git add sifty/bad.txt
git commit -m 'chore(sifty): add file' 2>&1 && echo FAIL_1 || echo OK_1_refused
git restore --staged sifty/bad.txt && rm sifty/bad.txt
printf 'package main\n' > sifty/clean.go && git add sifty/clean.go
git commit -m 'chore(sifty): add clean go file' && echo OK_3_passed || echo FAIL_3
git worktree add $SCRATCH/wt -b test-wt -q
cd $SCRATCH/wt && git config user.email t@t.com && git config user.name T
printf 'see GO-1 for details\n' > sifty/bad.txt && git add sifty/bad.txt
git commit -m 'chore(sifty): worktree test' 2>&1 && echo FAIL_4 || echo OK_4_wt_refused
git restore --staged sifty/bad.txt && rm sifty/bad.txt
printf 'package wt\n' > sifty/wt_clean.go && git add sifty/wt_clean.go
git commit -m 'chore(sifty): add clean worktree file' && echo OK_5_wt_passed || echo FAIL_5
rm -rf $SCRATCH
`
Exit: 0

```
commit-diff-check: refused — the staged changes in /private/var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.F0H8Ql8IV3/repo carries 1 form this repository's content must not:
sifty/bad.txt:1: [untidy] card identifier
    see GO-1 for details
    replace with: remove the identifier 'GO-1'; keep the sentence it was attached to
OK_1_refused
[main 842bc1f] chore(sifty): add clean go file
 1 file changed, 1 insertion(+)
 create mode 100644 sifty/clean.go
OK_3_passed
commit-diff-check: refused — the staged changes in /private/var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.F0H8Ql8IV3/wt carries 1 form this repository's content must not:
sifty/bad.txt:1: [untidy] card identifier
    see GO-1 for details
    replace with: remove the identifier 'GO-1'; keep the sentence it was attached to
OK_4_wt_refused
[test-wt c6d7c7d] chore(sifty): add clean worktree file
 1 file changed, 1 insertion(+)
 create mode 100644 sifty/wt_clean.go
OK_5_wt_passed
```


## 2026-11-04 — 3.1

Check: `python3 sifty/scripts/leakcheck/tree-check . --subtree sifty/`
Exit: 0

```
tree-check: /Volumes/Documents/Work/FCTO/.worktrees/GO-3 (subtree sifty/) is clean (private repository, origin git@github.com:w3con/fcto.git).
```


## 2026-11-04 — 3.2

Check: `python3 sifty/scripts/leakcheck/leakcheck-selftest && bin/work-test`
Exit: 0

```
ok  flag  [cloud access key]
ok  flag  [personal access token]
ok  flag  [cloud API key]
ok  flag  [JSON Web Token]
ok  flag  [private key (PEM block)]
ok  flag  [secret assigned a real value]
ok  flag  [.env file content]
ok  pass  'fitScore := assess(candidate, posting)'
ok  pass  'api_key = ${SIFTY_API_KEY}'
ok  pass  'export API_KEY from the environment'
ok  pass  'token: your-token-here'
ok  pass  'password = <changeme>'
ok  pass  '// read the key from a local file the .gitignore excludes'

PASS: 7 seeded secrets flagged, 6 clean lines passed
ok   compile _work_lib.py
ok   compile work-new
ok   compile work-plan
ok   compile work-handover
ok   compile work-note
ok   compile work-commit
ok   compile work-resume
ok   compile work-accept
ok   compile work-finish
ok   template convention.md
ok   template description.md
ok   template short-description.md
ok   template plan.md
ok   template log.md
ok   template comments.md
ok   template attention.md
ok   template executor-brief.md
ok   AGENTS.md present
ok   AGENTS.md names the commands
ok   AGENTS.md has no @-import of the mechanism
ok   --help work-new
ok   --help work-plan
ok   --help work-handover
ok   --help work-note
ok   --help work-commit
ok   --help work-resume
ok   --help work-accept
ok   --help work-finish

PASS: all harness checks
```

