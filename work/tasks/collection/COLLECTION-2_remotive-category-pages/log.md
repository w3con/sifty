# COLLECTION-2 — log

## R.2 done early on the owner's word, 2026-11-13

Owner: «да, удаляй из обеих» (answer to the coordinator's proposal to remove the Remotive category pages now by the normal role deletion, from production and staging, before the collector fix). The coordinator ran it directly; no code changed.

- Dry run (read-only SQL, both bases identical): 72 rows with `original_url` = `https://remotive.com/remote-jobs/<one segment>` (categories, job types, levels, locations, "posted last N", api, rss-feed) plus id 2370 "Post a Remote Job" (`https://remotive.com/job/post/new`) = 73 rows, ids 2270–2370 with gaps; all status `new`, company "Unknown", no workflow history. Lists in `.local/COLLECTION-2/dryrun-*.txt`, ids in `.local/COLLECTION-2/ids.txt`.
- Backups before writing: `.local/COLLECTION-2/data-store-before-20261113T030152Z.sqlite` and `.local/COLLECTION-2/data-staging-store-before-20261113T030152Z.sqlite` (sqlite `.backup`).
- Method: per id `PATCH /api/jobs/{id} {"stage":"no"}` then `DELETE /api/jobs/{id} {"confirmed":true}`, the same path as Dismiss + "Delete role…", so each URL lands in `deleted_listings` and is not collected again. Collection was idle on both servers. First two attempts changed nothing: DELETE refuses a role in Triage, and PATCH refuses a free-text `reason` (only listed outcome reasons). Responses in `.local/COLLECTION-2/delete3-808{0,1}.log`.
- Production (:8080): 73 deleted, 0 failed; jobs 2833 → 2760; `deleted_listings` 0 → 73; no target id left; health 200.
- Staging (:8081): the server answered 73 deleted and reports e.g. 2365 "not found", but the file on disk still shows 11 of them (2360–2370) and 62 tombstones. Cause, not created by this step: the staging chain is the stale DESIGN-3 preview (vite :8081 → sift-preview :8082 → sifty :62634, binary from 2026-11-10, pid 29573); that process holds `data-staging/store.sqlite-wal` inode 16324980 (22 MB), while the path on disk is a different, empty WAL created 2026-11-11 02:39. The server writes to a WAL nobody else can see; its checkpoints land in the real main file, so the remaining 11 deletions appear on disk at the next checkpoint or on a clean stop (SIGTERM, not SIGKILL). Production is not affected (server and disk inodes match). Fix: stop the stale DESIGN-3 preview stack cleanly and start staging normally — owner's decision.

Remaining in this card: R.1 collector, R.3 guard for every source, R.4 We Work Remotely foreign salary blocks — after the INBOX-2 release.
