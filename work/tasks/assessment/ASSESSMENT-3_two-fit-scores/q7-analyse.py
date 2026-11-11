#!/usr/bin/env python3
"""Q.7 comparison of the matcher before and after (read-only).

Usage: q7-analyse.py <store.sqlite> <before.json> [--seed N] [--unclear N] [--no N] [--start ISO-UTC]

<before.json> is the snapshot of the Q.6 matches taken before the new matcher ran (it holds, per job,
the verdict record and the stored fit result, and the list `first43` of vacancies Q.6 reported on).
Prints, for the vacancies of that list that now carry a verdict from the new matcher (version
vacancyai.match/3): verdict shares before and after, how many requirements code decided without the
model, time per vacancy, You fit before and after (unrelated roles, Parity, Ethena Labs), and a
reproducible random sample of "unclear" and "no" lines for the hand check.
"""
import json
import random
import sqlite3
import statistics
import sys
from collections import Counter
from datetime import datetime

CODE_REASONS = ("Named skill found in the profile", "Only some of the named skills are in the profile")
UNRELATED = ("accountant", "people partner", "growth lead", "attorney", "copywriter", "trader",
             "marketing", "recruiter", "sales", "designer", "support", "business development")


def ts(s):
    return datetime.strptime(s, "%Y-%m-%dT%H:%M:%S.%fZ").timestamp()


def pct(xs, p):
    xs = sorted(xs)
    k = (len(xs) - 1) * p
    lo, hi = int(k), min(int(k) + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (k - lo)


def shares(verdicts):
    c = Counter(v["verdict"] for v in verdicts)
    n = sum(c.values()) or 1
    return n, {k: (c.get(k, 0), 100.0 * c.get(k, 0) / n) for k in ("match", "partial", "no", "unclear")}


def line(label, verdicts):
    n, s = shares(verdicts)
    return "%s: %d requirements; " % (label, n) + ", ".join("%s %d (%.1f %%)" % (k, v[0], v[1]) for k, v in s.items())


def main():
    args = sys.argv[1:]
    db, before_path = args[0], args[1]
    opt = lambda name, default: type(default)(args[args.index(name) + 1]) if name in args else default
    seed, nunclear, nno = opt("--seed", 7), opt("--unclear", 20), opt("--no", 10)
    start = opt("--start", "")
    con = sqlite3.connect("file:%s?mode=ro" % db, uri=True, timeout=30)
    before = json.load(open(before_path))
    first43 = [int(j) for j in before["first43"]]

    jobs = {r[0]: (r[1], r[2]) for r in con.execute("select id, company, title from jobs")}
    new = {}
    for jid, created, version, rec in con.execute(
            "select job_id, created_at, matcher_version, record from match_requirement_matches order by created_at"):
        if version.startswith("vacancyai.match/3"):
            new[jid] = (created, version, json.loads(rec))
    fits_now = {r[0]: json.loads(r[1]) for r in con.execute("select job_id, record from match_fit_results")}
    exts = {r[0]: json.loads(r[1]) for r in con.execute("select job_id, record from match_vacancy_extractions")}

    both = [j for j in first43 if j in new]
    print("# Q.7 re-measurement\n")
    print("Matcher after: %s. Vacancies re-matched by the new matcher: %d (all), of them in the Q.6 set of %d: %d.\n" % (
        sorted({v[1] for v in new.values()}), len(new), len(first43), len(both)))
    q7_all = [j for j in new]

    # ---- verdict shares
    print("## Verdict shares\n")
    for label, ids in (("Q.6 set (%d vacancies)" % len(both), both), ("all re-matched (%d vacancies)" % len(q7_all), q7_all)):
        b = [v for j in ids if str(j) in before["matches"] for v in before["matches"][str(j)]["record"]["verdicts"]]
        a = [v for j in ids for v in new[j][2]["verdicts"]]
        print("- %s\n  - before: %s\n  - after:  %s" % (label, line("old matcher", b), line("new matcher", a)))
    allv = [v for j in q7_all for v in new[j][2]["verdicts"]]
    code = sum(1 for v in allv if v.get("reason") in CODE_REASONS)
    print("\nRequirements decided by code without the model: %d of %d (%.1f %%) over all re-matched vacancies." % (code, len(allv), 100.0 * code / max(len(allv), 1)))
    code43 = sum(1 for j in both for v in new[j][2]["verdicts"] if v.get("reason") in CODE_REASONS)
    n43 = sum(len(new[j][2]["verdicts"]) for j in both)
    print("In the Q.6 set: %d of %d (%.1f %%)." % (code43, n43, 100.0 * code43 / max(n43, 1)))
    nomodel = [j for j in q7_all if all(v.get("reason") in CODE_REASONS for v in new[j][2]["verdicts"])]
    print("Vacancies in which code decided every requirement (no model call at all): %d of %d." % (len(nomodel), len(q7_all)))

    # ---- timing
    print("\n## Time per vacancy (matching only)\n")
    rows = sorted((ts(c), j) for j, (c, _v, _r) in new.items())
    prev = ts(start) if start else None
    times, asked_times = [], []
    for t, j in rows:
        if prev is not None and 0 < t - prev <= 600:
            times.append(t - prev)
            if j not in nomodel:
                asked_times.append(t - prev)
        prev = t
    if times:
        print("- all %d timed vacancies: median %.1f s, p90 %.1f s, mean %.1f s, max %.1f s" % (
            len(times), statistics.median(times), pct(times, 0.9), statistics.mean(times), max(times)))
    if asked_times:
        print("- the %d that needed the model: median %.1f s, p90 %.1f s, mean %.1f s" % (
            len(asked_times), statistics.median(asked_times), pct(asked_times, 0.9), statistics.mean(asked_times)))
    print("- Q.6 for comparison (matching alone, old matcher): median 10.1 s, p90 16.4 s.")

    # ---- You fit
    print("\n## You fit before and after\n")

    def you(fit):
        y = (fit or {}).get("youFit") or {}
        return y.get("score") if y.get("status") == "scored" else None

    table = []
    for j in both:
        b = you(before["fit_results"].get(str(j)))
        a = you(fits_now.get(j))
        table.append((j, jobs.get(j, ("?", "?")), b, a))
    bs = [r[2] for r in table if r[2] is not None]
    as_ = [r[3] for r in table if r[3] is not None]
    if bs and as_:
        print("- before: min %d, median %.0f, max %d; below 50: %d of %d" % (min(bs), statistics.median(bs), max(bs), sum(1 for x in bs if x < 50), len(bs)))
        print("- after:  min %d, median %.0f, max %d; below 50: %d of %d" % (min(as_), statistics.median(as_), max(as_), sum(1 for x in as_ if x < 50), len(as_)))
    print("\n| job | company / title | You fit before | You fit after |\n|---|---|---|---|")
    for j, (company, title), b, a in sorted(table, key=lambda r: (r[3] is None, r[3] if r[3] is not None else 0)):
        mark = " **(unrelated role)**" if any(k in title.lower() for k in UNRELATED) else ""
        if j in (84, 22):
            mark = " **(named in the brief)**"
        print("| %d | %s / %s%s | %s | %s |" % (j, company, title, mark, b, a))

    # ---- hand check samples
    rnd = random.Random(seed)
    lines = {"unclear": [], "no": []}
    for j in both:
        reqs = exts[j]["requirements"]
        for v in new[j][2]["verdicts"]:
            if v["verdict"] in lines:
                lines[v["verdict"]].append((j, reqs[v["index"]], v))
    for verdict, n in (("unclear", nunclear), ("no", nno)):
        pool = lines[verdict]
        pick = rnd.sample(pool, min(n, len(pool)))
        print("\n## Sample of %d '%s' lines (seed %d, pool %d)\n" % (len(pick), verdict, seed, len(pool)))
        for i, (j, r, v) in enumerate(pick, 1):
            company, title = jobs.get(j, ("?", "?"))
            fit = you(fits_now.get(j))
            print("%d. job %d %s / %s (You fit %s), %s %s: **%s**" % (i, j, company, title, fit, r["kind"], r["level"], r["text"]))
            if r.get("alternatives"):
                print("   any of: %s" % " | ".join(r["alternatives"]))
            print("   quote: %s" % r["quote"][:260])
            print("   model reason: %s" % v.get("reason", ""))
            if v.get("evidence"):
                print("   evidence: %s: %s" % (v["evidence"]["field"], v["evidence"]["value"][:200]))


if __name__ == "__main__":
    main()
