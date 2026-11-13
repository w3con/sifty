#!/usr/bin/env python3
"""Q.6 analysis of the run recorded in a Sifty store (read-only).

Usage: q6-analyse.py <store.sqlite> <window-start-utc, e.g. 2026-11-11T02:22:00Z> [--seed N] [--unclear N] [--min-fits X]
--min-fits keeps only the "unclear" lines of vacancies whose stored Fits you is at least X (0..1).

Prints: per-vacancy timing (median, p90, mean), the projected backfill, the share of requirements
per verdict, and a reproducible random sample of "unclear" lines for the hand check.

Timing method: one worker handles one vacancy at a time (extraction, then matching), so a
vacancy's time is the gap between the previous vacancy's last saved row and its own last saved row.
Gaps longer than 600 s (restart, pause, backoff) are dropped as non-work.
"""
import json
import random
import sqlite3
import statistics
import sys
import urllib.request
from datetime import datetime


def ts(s):
    return datetime.strptime(s, "%Y-%m-%dT%H:%M:%S.%fZ").timestamp()


def pct(xs, p):
    xs = sorted(xs)
    k = (len(xs) - 1) * p
    lo, hi = int(k), min(int(k) + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (k - lo)


def main():
    args = sys.argv[1:]
    db, start = args[0], args[1]
    seed = int(args[args.index("--seed") + 1]) if "--seed" in args else 6
    nunclear = int(args[args.index("--unclear") + 1]) if "--unclear" in args else 10
    minfits = float(args[args.index("--min-fits") + 1]) if "--min-fits" in args else None
    con = sqlite3.connect("file:%s?mode=ro" % db, uri=True, timeout=30)

    ext = {r[0]: (r[1], json.loads(r[2])) for r in con.execute(
        "select job_id, created_at, record from match_vacancy_extractions where created_at >= ? order by created_at", (start,))}
    mat = {}
    for jid, created, rec in con.execute(
            "select job_id, created_at, record from match_requirement_matches where created_at >= ? order by created_at", (start,)):
        mat[jid] = (created, json.loads(rec))
    jobs = {r[0]: r[1:] for r in con.execute("select id, company, title, length(description) from jobs")}

    print("# Q.6 run analysis\n")
    print("Window start: %s. Vacancies extracted in the window: %d; matched: %d.\n" % (start, len(ext), len(mat)))

    # --- timing
    rows = []
    for jid, (te, _rec) in ext.items():
        end = max(ts(te), ts(mat[jid][0])) if jid in mat else ts(te)
        rows.append((end, jid, ts(te)))
    rows.sort()
    times, ext_t, mat_t, per = [], [], [], []
    prev_end = None
    for end, jid, te in rows:
        if prev_end is not None and end - prev_end <= 600:
            t = end - prev_end
            times.append(t)
            per.append((jid, t))
            if jid in mat:
                ext_t.append(te - prev_end)
                mat_t.append(end - te)
        prev_end = end
    print("## Timing\n")
    print("Vacancies timed: %d (first row after a start and gaps over 600 s dropped)." % len(times))
    if times:
        print("- Per vacancy, extraction plus matching: median %.1f s, p90 %.1f s, mean %.1f s, max %.1f s." % (
            statistics.median(times), pct(times, 0.9), statistics.mean(times), max(times)))
    if ext_t:
        print("- Extraction pass alone: median %.1f s, p90 %.1f s. Matching pass alone: median %.1f s, p90 %.1f s." % (
            statistics.median(ext_t), pct(ext_t, 0.9), statistics.median(mat_t), pct(mat_t, 0.9)))
    pending = None
    try:
        with urllib.request.urlopen("http://localhost:8094/api/quick-match/analysis", timeout=5) as r:
            st = json.load(r)
            pending = st.get("pendingExtraction")
    except Exception as exc:  # the server may be stopped
        print("(status endpoint unavailable: %s)" % exc)
    if pending is not None and times:
        for label, v in (("median", statistics.median(times)), ("mean", statistics.mean(times)), ("p90", pct(times, 0.9))):
            print("- Projected backfill of %d vacancies still pending, at the %s: %.1f hours." % (pending, label, pending * v / 3600))
    print("\nSpecial vacancies in the window:")
    for jid in (84, 22):
        t = dict(per).get(jid)
        n = len(ext[jid][1]["requirements"]) if jid in ext else None
        c = jobs.get(jid, ("?", "?", 0))
        print("- %d %s / %s: time %s, requirements %s, description %s characters" % (
            jid, c[0], c[1], "%.1f s" % t if t else "not timed", n, c[2]))

    # --- verdict shares
    print("\n## Verdicts\n")
    counts = {"match": 0, "partial": 0, "no": 0, "unclear": 0}
    per_job_unclear = []
    for jid, (_c, rec) in mat.items():
        vs = rec["verdicts"]
        for v in vs:
            counts[v["verdict"]] = counts.get(v["verdict"], 0) + 1
        per_job_unclear.append((jid, sum(1 for v in vs if v["verdict"] == "unclear"), len(vs)))
    total = sum(counts.values())
    print("Requirements judged: %d across %d vacancies." % (total, len(mat)))
    for k, v in counts.items():
        print("- %s: %d (%.1f %%)" % (k, v, 100.0 * v / total if total else 0))
    nreq = [len(r["requirements"]) for _t, r in ext.values()]
    if nreq:
        print("Requirements per vacancy: median %.0f, max %d; vacancies with none: %d." % (statistics.median(nreq), max(nreq), sum(1 for n in nreq if n == 0)))

    # --- unclear sample
    pool = []
    fits = {r[0]: r[1] for r in con.execute("select job_id, fits_raw from match_fit_results")}
    for jid, (_c, rec) in mat.items():
        if minfits is not None and (fits.get(jid) is None or fits[jid] < minfits):
            continue
        reqs = ext.get(jid, (None, {"requirements": []}))[1]["requirements"]
        for v in rec["verdicts"]:
            if v["verdict"] == "unclear" and v["index"] < len(reqs):
                pool.append((jid, v["index"], reqs[v["index"]], v.get("reason", "")))
    random.Random(seed).shuffle(pool)
    print("\n## Sample of %d 'unclear' lines (seed %d, pool %d%s)\n" % (min(nunclear, len(pool)), seed, len(pool), ", Fits you >= %.2f" % minfits if minfits is not None else ""))
    for i, (jid, idx, rq, reason) in enumerate(pool[:nunclear], 1):
        c = jobs.get(jid, ("?", "?", 0))
        print("%d. job %d %s / %s, requirement %d (%s, %s): %s\n   quote: %s\n   model reason: %s" % (
            i, jid, c[0], c[1], idx, rq.get("kind"), rq.get("level"), rq.get("text"), rq.get("quote", "")[:200], reason))


if __name__ == "__main__":
    main()
