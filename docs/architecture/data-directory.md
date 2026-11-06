# The data directory

Sifty is self-contained: everything it reads and writes at runtime lives under one local data directory. The directory is configured by the `--data-dir` flag, then the `SIFTY_DATA_DIR` environment variable, then a default of `./data`. Nothing is written outside it, and no path is resolved from a git root.

## Layout

```
<data-dir>/
  config/
    search.yaml       sources and query terms
    runtime.yaml      collection and assessment bounds
    providers.yaml    inference providers and the provider chosen per stage
    candidate.yaml    candidate evidence files and profiles
    network.yaml      per-source network profiles (optional)
  evidence/           candidate evidence files referenced by candidate.yaml (CV, profile exports)
  answers/            remembered application answers, one markdown file per answer
  feedback/           owner-confirmed candidate feedback, one markdown file per statement
  store.sqlite        the local store
```

The configuration file shapes match the current application's schemas, so an existing configuration is valid here. Candidate evidence paths in `candidate.yaml` are resolved under `evidence/`, never a git root; there are no `kb/` writes and no project-specific directory names.

## Answer and feedback files

Each remembered answer and each feedback statement is a markdown file that is both human-readable and machine-recoverable: the prose is the record, and a base64-encoded JSON copy of the same record sits in an HTML comment marker at the end (`SIFT_ANSWER` for answers, `SIFT_FEEDBACK` for feedback). The file is the first durable write and survives a missing model, a store failure or a restart.

The exact shapes are the templates `backend/internal/records/templates/answer.md.tmpl` and `backend/internal/records/templates/feedback.md.tmpl`, embedded in the binary. A first write produces a file that follows the matching template; this is proven by the tests in `backend/internal/records`.
