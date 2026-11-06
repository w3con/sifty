# Sifty backend — HTTP API contract

**Status:** reference, current as of 2026-11-02. The exact request and response surface the React interface depends on and the Go backend must reproduce. Derived from `app/src/server.ts` and the services behind it. This is prose derived from code; the port captures the same contract as recorded request/response fixtures (see the architecture document, section 11) so it is enforced by a failing test rather than by trust in this page. Where this page and the code disagree, the code wins.

All endpoints are under `/api`, bound to loopback only. Shared rules apply to every endpoint and are not repeated per row.

## Shared rules

- **Transport.** HTTP on `127.0.0.1` at the configured port. The `Host` header must be `127.0.0.1:<port>` or `localhost:<port>`; an `Origin`, if present, must be the matching `http://` origin; otherwise `403`.
- **Request body.** Any non-GET request must carry `Content-Type: application/json`; otherwise `415`. Bodies are JSON, capped at 1 MB. Responses are JSON unless noted (one endpoint returns a PDF).
- **Response headers.** `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer` on every response.
- **Errors.** A handler that throws returns `400` with `{ "error": "<message>" }`. An unknown `/api/...` path returns `404` with `{ "error": "Unknown API endpoint. Refresh the page to load the current catalogue version." }`.
- **Ids.** A role id in a path is an integer; a stale id that was merged resolves transparently to its current role.
- **Async actions.** Collection, assessment and preparation are started, not awaited: those endpoints return `202` and the work proceeds in the background, observed through the queue, collection-progress and application endpoints.

## Shared response shapes

These shapes recur across endpoints and are defined once here.

**Role** (the object returned by every job endpoint):

```
{
  id: number, title: string, company: string, tags: string[],
  stage: "triage"|"preselected"|"no"|"applied", status: string,
  seen: boolean, seenAt: string|null, appliedAt: string|null, isWeb3: boolean,
  originalUrl: string, applyUrl: string|null, applyStatus: string,
  description: string, descriptionStatus: string, location: string|null,
  publishedAt: string|null, expiresAt: string|null,
  firstSeen: string, lastSeen: string, expired: boolean,
  assessment: Assessment|null, assessmentError: string|null,
  sources: { name: string, url: string, seen: string }[]
}
```

**Assessment** (the `assessment` field of a role, when present):

```
{
  score: number (0..100), summary: string,
  evidence: { claim: string, sourceId: string, quote: string }[],
  gaps: string[], confirmedMismatches: string[],
  candidateUnknowns: string[], employerUnknowns: string[],
  mismatchEvidence: object[], inputFingerprint: string,
  provider: string, model: string, contextVersion: string,
  promptVersion: string, jobVersion: string, createdAt: string,
  selection: object[], stale: boolean
}
```

**ApplicationDraft** (Stage 2 draft):

```
{
  jobId: number, revision: number, selectedFile: "cv.pdf"|"profile.pdf",
  questionnaire: Questionnaire|null, provenance: any, notes: string[],
  fields: AnswerField[]
}
```

**Question** / **AnswerField**:

```
Question = {
  id: string, label: string, instructions?: string,
  labelStatus?: "resolved"|"uncertain",
  type: "text"|"textarea"|"select"|"radio"|"checkbox"|"file",
  options: string[], required: boolean, maxLength: number|null,
  accept: string, sourceUrl: string, origin: "site"|"manual",
  language: string, signature: string, controlName: string
}
AnswerField = Question & {
  value: string|string[], revision: number, edited: boolean,
  unknowns: string[], evidence: { sourceId: string, quote: string }[],
  changed?: boolean, removed?: boolean
}
Questionnaire = {
  url: string, capturedAt: string, language: string,
  status: "partial"|"blocked"|"login-required"|"unsupported",
  gaps: string[], fields: Question[], family: string, technicalLog?: string
}
```

**CandidateContext** (the assembled evidence snapshot):

```
{
  version: string, createdAt: string, preferences: any, reviewNotes: string[],
  sources: { id: string, reference: string, retrievedAt: string, hash: string,
             status: string, text: string, error?: string,
             category?: string, scope?: string }[]
}
```

**QueueStatus**:

```
{
  enabled: boolean,
  counts: { running: number, pending: number, failed: number,
            done: number, "missing-description": number },
  rows: object[] (queue rows with an added elapsedMs; done rows omitted),
  runtime: object|null
}
```

## Endpoints

### Roles

- **GET `/api/jobs?status=<all|new|saved|rejected>&q=<text>`** → `Role[]`, sorted by assessment score descending. `status` defaults to `all`, `q` to empty.
- **GET `/api/jobs/:id`** → `Role`. `404`-class error if the role does not exist (thrown as `400` with message).
- **PATCH `/api/jobs/:id`** — body `{ stage?: string, status?: string, confirmed?: boolean }` (either `stage` or `status`; moving to applied requires `confirmed: true` and a pre-selected role) → the updated `Role`.
- **DELETE `/api/jobs/:id`** — body `{ confirmed: true }` (required; refused in triage and while work is active) → `{ deleted: number, cleanupErrors: string[] }`.
- **POST `/api/jobs/:id/seen`** → the updated `Role`.
- **POST `/api/jobs/:id/read`** — re-fetch and enrich one role's detail → `{ id: number }`.
- **POST `/api/jobs/:id/assess`** — body `{ provider?: string }` → `202`, the enqueue result `{ jobId, status, fingerprint }` or the existing queue row.

### Collection and sources

- **POST `/api/search`** — body `{ source?: string, force?: boolean }` (omit source to run all enabled) → `202 { runId: number, status: "running" }`.
- **GET `/api/collection`** → the current run `{ ...run, sources: { source, status, coverage }[], events: { created, source, message }[] }`, or `null` if none has run.
- **GET `/api/health`** → per-source health objects: the configured site fields plus `{ capability, status, message, checked, started, runId, mode, coverage, count, legacy, stale }`.
- **POST `/api/import`** — body `{ url: string }` → `{ id: number }`.

### Assessment queue

- **GET `/api/queue`** → `QueueStatus`.
- **PUT `/api/queue/assessment`** — body `{ enabled: boolean }` → `QueueStatus`.
- **GET `/api/assessment-options`** → `{ selected: string, providers: { name, model, type }[] }`.

### Candidate context, providers, config

- **GET `/api/context`** → `CandidateContext|null`.
- **POST `/api/context`** — body `{ refreshRemote?: boolean }` → the refreshed `CandidateContext`.
- **GET `/api/filter-options`** → the search configuration `{ version, sites[], search: { titles[], keywords[] }, preferences }`.
- **GET `/api/providers`** → `{ name, type, model, status }[]` (diagnostics).
- **POST `/api/providers/:name/check`** → a smoke-check result matching the assessment schema shape.
- **GET `/api/config/:name`** (`name` ∈ search|candidate|providers|network|runtime) → `{ content: string }` (the YAML text).
- **PUT `/api/config/:name`** — body `{ content: string }` (schema-validated, secrets rejected) → `{ saved: string }`.

### Feedback

- **GET `/api/jobs/:id/feedback`** → the feedback records for the role.
- **POST `/api/jobs/:id/feedback`** — body `{ text: string, scope?: "general"|"job", supersedes?: string }` → `202 { record: object, queue: QueueStatus }`; triggers reassessment.

### Application preparation (Stage 2)

- **GET `/api/jobs/:id/application`** → `{ draft: ApplicationDraft, operations: object[] }`.
- **POST `/api/jobs/:id/application/prepare`** — body `{ inspect?: boolean, generate?: boolean, provider?: string, fieldId?: string, login?: boolean }` (defaults: inspect true, generate true, provider "openai", login false; role must be pre-selected) → `202` the queued operation row.
- **POST `/api/jobs/:id/application/cancel`** → `{ draft, operations }` (the preparation state after cancel).
- **PATCH `/api/jobs/:id/application`** — body either `{ selectedFile: "cv.pdf"|"profile.pdf", revision: number }` or `{ fieldId: string, value: string|string[], revision: number }` (optimistic concurrency: revision must match) → the updated `ApplicationDraft`.
- **POST `/api/jobs/:id/application/questions`** — body `{ label: string, type?: Question["type"], options?: string[] }` → the updated `ApplicationDraft`.
- **POST `/api/jobs/:id/application/remember`** — body `{ fieldId: string, revision: number, scope: "general"|"company"|"job", confirmed: true }` → the saved answer record.
- **GET `/api/jobs/:id/application/memories`** → the remembered-answer records applicable to the role.
- **GET `/api/application-files/:name`** (`name` ∈ cv.pdf|profile.pdf) → the PDF bytes, `Content-Type: application/pdf`, served inline.
