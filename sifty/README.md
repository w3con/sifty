# Sifty

Local-first job triage. Sifty collects public vacancies, preserves their descriptions and provenance, assesses fit against your own candidate evidence, and prepares editable application drafts. It does not fill, upload or submit employer forms.

The backend is a single Go binary. Browser automation (form inspection and dynamic-site collection) runs in a thin TypeScript worker on the official Playwright, which the Go core invokes; the job-board scraper runs the same way, as a worker the core calls. Candidate data, remembered answers and feedback live in a local data directory you configure; nothing is written outside it.

## Status

Functional parity with Stages 1 (collection and fit) and 2 (application-answer preparation). The weighted scorer, model separation and the liveness detector are tracked as later work.

## Build

```
go build ./cmd/sifty
```

produces a runnable `sifty` binary.

## Run

Point Sifty at a data directory and run:

```
SIFTY_DATA_DIR=/path/to/data ./sifty
```

The data directory holds your candidate evidence, remembered answers and feedback. On first run Sifty writes the answer and feedback files from templates. See `docs/` for the data-directory layout.

## Workers

The TypeScript workers live under `workers/`. They are invoked by the Go core over the line-delimited JSON protocol documented in `docs/worker-protocol.md`; there is no browser-automation code in Go.

## License

MIT. See `LICENSE`.
