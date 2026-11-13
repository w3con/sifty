# Optional shared tooling

`tools/shared` is the pinned Git submodule of the private shared tooling
(`github.com:w3con/ai`). It is **optional** for a public clone: the application,
the `bin/work-*` lifecycle commands, the leak scanner and the project Git hooks
do not require it.

- `tools/bin`, `tools/harness` and `tools/templates` are relative symlinks into
  `tools/shared`. They are intentionally dangling when the submodule is not
  initialized; that is not a broken checkout.
- With access to the private upstream: `git submodule update --init tools/shared`.
- Without access: skip it. `bin/work-setup` still verifies and installs the core
  harness, and `bin/work-test` passes without the submodule.
- The knowledge-base hooks look for `tools/bin/*` only when a local `kb/`
  directory exists. A missing tool is reported explicitly rather than silently
  skipped.

This relocation preserves the pinned gitlink (`aebbcda835ab708be1b465cfcb566b1994f09b19`)
and the upstream contents; it does not fork or modify the shared repository.
The retired transfer tooling under `legacy/app/ops/transfer/` still describes the
old `ai/shared` layout and is retained as historical code.
