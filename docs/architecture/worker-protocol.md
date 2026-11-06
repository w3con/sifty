# Go-core to TypeScript-worker protocol

Browser automation (form inspection, dynamic-site collection) and the job-board scraper run in thin TypeScript workers on the official Playwright and on jobspy, which Go has no first-party binding for. The Go core invokes a worker as a subprocess and exchanges one JSON request and one JSON response per call. There is no browser-automation or scraping code in Go.

## Invocation

The core spawns the worker with a controlled working directory and a minimal, isolated environment (it does not propagate the parent's session, model or billing credentials). The same contract serves every worker; a worker is distinguished only by the command the core is configured to run (for example `node backend/workers/playwright/index.mjs` or `node backend/workers/jobspy/index.mjs`).

## Request

The core writes exactly one line of compact JSON to the worker's standard input, then closes it:

```
{"method": "<name>", "params": { ... }}
```

`method` names the operation (`inspectForm`, `collect`, and so on). `params` is an operation-specific object. The worker must read standard input to end before producing output.

## Response

The worker writes exactly one JSON object to standard output and exits. Diagnostics go to standard error, never standard output.

```
{"ok": true, "result": { ... }}
{"ok": false, "error": "<message>"}
```

A handled failure is reported as `{"ok": false, "error": ...}` with exit code 0. A non-zero exit code means the worker crashed; the core treats that, a timeout, unparsable output or oversized output as a transport failure and reports it with the captured standard error.

## Guarantees the core enforces

A per-call timeout that kills the worker process group; a cap on response size; and an isolated environment. The worker is expected to make no network request the operation does not require, and the browser workers additionally block non-GET/HEAD requests, downloads and service workers, as the current application does.
