#!/usr/bin/env python3
"""Remove the 40 Remote3 category-page rows from a Sifty database.

By default this script is a dry run: it reads the database in read-only mode,
prints the rows it would remove, and does nothing else.

With --apply --port N it removes each row through the running server's API:
  1. PATCH /api/jobs/{id}  {"stage":"no"}  — move to Rejected (tombstone-safe)
  2. DELETE /api/jobs/{id}  {"confirmed":true}  — permanent deletion with cleanup

The script refuses to proceed when:
  - The selected row count does not equal --expect (default 40).
  - Any selected row's URL contains "/remote-jobs/" (would indicate id=7 is selected).
  - The server reports an active collection (status "running").
  - Any API call fails (stops at the first failing row and names it).

Usage:
  python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py <store.sqlite>
  python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py <store.sqlite> --apply --port 8080
  python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py <store.sqlite> --apply --port 8080 --public-port 8080
  python3 -I work/tasks/inbox/INBOX-2_listing-markup/release-steps/remote3_cleanup.py <store.sqlite> --expect 40 --apply --port 8080
"""

import argparse
import json
import sqlite3
import sys
import urllib.request
import urllib.error

PATTERN = "https://remote3.co/jobs/%"
SAFE_GUARD = "/remote-jobs/"


def select_rows(db_path: str) -> list[dict]:
    uri = f"file:{db_path}?mode=ro"
    con = sqlite3.connect(uri, uri=True)
    con.row_factory = sqlite3.Row
    try:
        rows = con.execute(
            "SELECT id, title, original_url FROM jobs WHERE original_url LIKE ?",
            (PATTERN,),
        ).fetchall()
        return [dict(r) for r in rows]
    finally:
        con.close()


def check_collection(port: int, public_port: int) -> None:
    url = f"http://127.0.0.1:{port}/api/collection"
    req = urllib.request.Request(
        url,
        headers={
            "Host": f"127.0.0.1:{public_port}",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read())
    except urllib.error.URLError as e:
        print(f"ERROR: cannot reach server at port {port}: {e}", file=sys.stderr)
        sys.exit(1)
    if data is None:
        return
    status = data.get("status", "")
    if status == "running":
        print(
            "ERROR: server reports an active collection (status=running). "
            "Wait for it to finish before running this script.",
            file=sys.stderr,
        )
        sys.exit(1)


def api_patch_no(port: int, public_port: int, job_id: int) -> None:
    url = f"http://127.0.0.1:{port}/api/jobs/{job_id}"
    body = json.dumps({"stage": "no"}).encode()
    req = urllib.request.Request(
        url,
        data=body,
        method="PATCH",
        headers={
            "Host": f"127.0.0.1:{public_port}",
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        _ = resp.read()


def api_delete(port: int, public_port: int, job_id: int) -> None:
    url = f"http://127.0.0.1:{port}/api/jobs/{job_id}"
    body = json.dumps({"confirmed": True}).encode()
    req = urllib.request.Request(
        url,
        data=body,
        method="DELETE",
        headers={
            "Host": f"127.0.0.1:{public_port}",
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        _ = resp.read()


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Dry-run or apply removal of Remote3 category-page rows."
    )
    parser.add_argument("db", help="Path to store.sqlite")
    parser.add_argument(
        "--apply", action="store_true", help="Actually remove rows via the server API"
    )
    parser.add_argument(
        "--port",
        type=int,
        default=0,
        help="Server port (required with --apply)",
    )
    parser.add_argument(
        "--public-port",
        type=int,
        default=0,
        help="Public port used in Host/Origin headers (defaults to --port)",
    )
    parser.add_argument(
        "--expect",
        type=int,
        default=40,
        help="Expected number of rows to remove (default 40); script refuses if count differs",
    )
    args = parser.parse_args()

    if args.apply and not args.port:
        print("ERROR: --apply requires --port", file=sys.stderr)
        return 1

    public_port = args.public_port if args.public_port else args.port

    rows = select_rows(args.db)

    # Safety: none of the selected rows must have /remote-jobs/ in the URL.
    bad = [r for r in rows if SAFE_GUARD in r["original_url"]]
    if bad:
        print(
            f"ERROR: {len(bad)} selected row(s) contain '{SAFE_GUARD}' in the URL — "
            "the pattern is too broad. Aborting.",
            file=sys.stderr,
        )
        for r in bad:
            print(f"  id={r['id']}  {r['original_url']}", file=sys.stderr)
        return 1

    if len(rows) != args.expect:
        print(
            f"ERROR: selected {len(rows)} rows but --expect is {args.expect}. "
            "Verify the database and re-run with the correct --expect value.",
            file=sys.stderr,
        )
        return 1

    print(f"Rows selected ({len(rows)}):")
    for r in rows:
        print(f"  id={r['id']:5d}  {r['title'][:60]!r}  {r['original_url']}")

    if not args.apply:
        print("\nDry run complete. Use --apply --port N to remove these rows.")
        return 0

    # Live run: check server not collecting, then delete each row.
    check_collection(args.port, public_port)

    removed = 0
    for r in rows:
        job_id = r["id"]
        try:
            api_patch_no(args.port, public_port, job_id)
        except urllib.error.URLError as e:
            print(
                f"ERROR: PATCH /api/jobs/{job_id} failed for id={job_id} "
                f"({r['title']!r}): {e}",
                file=sys.stderr,
            )
            print(f"Stopped after {removed} successful removals.", file=sys.stderr)
            return 1
        except Exception as e:
            print(
                f"ERROR: unexpected error patching id={job_id} ({r['title']!r}): {e}",
                file=sys.stderr,
            )
            return 1

        try:
            api_delete(args.port, public_port, job_id)
        except urllib.error.URLError as e:
            print(
                f"ERROR: DELETE /api/jobs/{job_id} failed for id={job_id} "
                f"({r['title']!r}): {e}",
                file=sys.stderr,
            )
            print(f"Stopped after {removed} successful removals.", file=sys.stderr)
            return 1
        except Exception as e:
            print(
                f"ERROR: unexpected error deleting id={job_id} ({r['title']!r}): {e}",
                file=sys.stderr,
            )
            return 1

        removed += 1
        print(f"  removed id={job_id}")

    print(f"\nDone: {removed} rows removed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
