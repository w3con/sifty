# Release steps for INBOX-2

## Step R.1 — Remove 40 Remote3 category-page rows

### What it does

The script `remote3_cleanup.py` removes 40 rows whose `original_url` matches
`https://remote3.co/jobs/%`. These are category pages (e.g. "Rust Jobs"), not
vacancy pages, and were collected by an earlier collection defect. The real Remote3
vacancy at id=7 (`/remote-jobs/…`) is not selected by the pattern.

**Prerequisites before running:**

- The release has started the new binary and it is healthy.
- A backup of the data directory exists (the standard pre-release backup).
- No collection is in progress. The script refuses to run while one is active.

### Commands

**Production** (port 8080, data in `data/`):

```
python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py data/store.sqlite
python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py data/store.sqlite --apply --port 8080
```

**Staging** (port 8081, data in `data-staging/`):

```
python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py data-staging/store.sqlite
python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py data-staging/store.sqlite --apply --port 8081
```

If the server listens on an internal port but the Host header must carry a
different public port (e.g. a reverse proxy setup), add `--public-port <N>`.

### How to check the result

After the apply run prints "Done: 40 rows removed.", confirm:

```
python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py data/store.sqlite --expect 0
```

This should print "Rows selected (0):" and exit successfully. It also confirms
that the safety guard against `/remote-jobs/` rows never triggered.

To confirm the real Remote3 vacancy survived:

```
sqlite3 data/store.sqlite "SELECT id, title FROM jobs WHERE id=7;"
```

Expected: one row for id=7.

### Notes

- The script is idempotent: running it again on a database where the rows are
  already gone selects 0 rows and exits with an error unless `--expect 0` is passed.
- The two-step deletion (PATCH to rejected, then DELETE with confirmation) goes
  through the server's own API so tombstones are written and cleanup runs for
  all related tables (extractions, matches, fit results, assessments, aliases).
- The script stops on the first API error and says which row failed. Any rows
  already removed before the failure stay removed; rerun once the cause is fixed.
- Do not touch the live bases before the release. This card's work is in
  `work/INBOX-2`; the cleanup runs only during the release.
- The server refuses deletion while a collection or assessment holds its lock (message "Work is active") and the script refuses while the latest run is still marked running. As of 2026-11-12 `data` is clear (latest run complete, no lock rows), but `data-staging` is held by another server using that directory (a daily run started 08:00 and still marked running, lock row of a live process). Run the staging step only after that server has finished or stopped.
- Proven on a `.backup` copy of `data-staging`: the copy carries those two rows from the live base, so they were cleared in the copy only; then 40 rows removed (2796 to 2756 vacancies), id 7 kept, 40 removal records written, a second run finds 0 rows.
