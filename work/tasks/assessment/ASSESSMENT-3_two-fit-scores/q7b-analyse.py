#!/usr/bin/env python3
"""Q.7b comparison of the matcher on two models (read-only).

Usage: q7b-analyse.py --before q7-before.json --db9 <store with the 9B verdicts and fits>
                      --run1 <store after the 35B run 1> --run2 <store after the 35B run 2>
                      --v9 <9B matcher version digest> --v35 <35B matcher version digest> [--seed 7]

Compares, for the 43 vacancies of the Q.6 set (list `first43` in the before file): verdict shares of the
old Q.6 matcher, the 9B Q.7 matcher and the 35B matcher (two runs); You fit of the unrelated roles and
the vacancies named in the brief; how many requirements got the same verdict in both 35B runs; time
per vacancy; and the same random sample of "unclear" and "no" lines the Q.7 hand check used, with the
verdicts of both models side by side.
"""
import json
import random
import sqlite3
import statistics
import sys
from collections import Counter
from datetime import datetime

NAMED = {84: "Parity", 22: "Ethena Labs"}
UNRELATED = ("accountant", "people partner", "growth lead", "attorney", "copywriter", "trader",
             "marketing", "recruiter", "sales", "designer", "support", "business development")


def ts(s):
    return datetime.strptime(s, "%Y-%m-%dT%H:%M:%S.%fZ").timestamp()


def pct(xs, p):
    xs = sorted(xs)
    k = (len(xs) - 1) * p
    lo, hi = int(k), min(int(k) + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (k - lo)


def opts():
    a = sys.argv[1:]
    get = lambda n, d=None: a[a.index(n) + 1] if n in a else d
    return {k: get("--" + k) for k in ("before", "db9", "run1", "run2", "v9", "v35", "seed")}


def con(path):
    return sqlite3.connect("file:%s?mode=ro" % path, uri=True, timeout=30)


def load(path, version):
    c = con(path)
    matches = {}
    for jid, created, v, rec in c.execute("select job_id, created_at, matcher_version, record from match_requirement_matches order by created_at"):
        if version in v:
            matches[jid] = (created, json.loads(rec))
    fits = {r[0]: json.loads(r[1]) for r in c.execute("select job_id, record from match_fit_results")}
    jobs = {r[0]: (r[1], r[2]) for r in c.execute("select id, company, title from jobs")}
    exts = {r[0]: json.loads(r[1]) for r in c.execute("select job_id, record from match_vacancy_extractions")}
    return matches, fits, jobs, exts


def you(fit):
    y = (fit or {}).get("youFit") or {}
    return y.get("score") if y.get("status") == "scored" else None


def shares(vs):
    c = Counter(v["verdict"] for v in vs)
    n = sum(c.values()) or 1
    return "%d req: " % n + ", ".join("%s %d (%.1f %%)" % (k, c.get(k, 0), 100.0 * c.get(k, 0) / n) for k in ("match", "partial", "no", "unclear"))


def main():
    o = opts()
    before = json.load(open(o["before"]))
    first43 = [int(j) for j in before["first43"]]
    m9, f9, jobs, exts = load(o["db9"], o["v9"])
    m1, f1, _, _ = load(o["run1"], o["v35"])
    m2, f2, _, _ = load(o["run2"], o["v35"]) if o["run2"] else ({}, {}, None, None)
    both = [j for j in first43 if j in m9 and j in m1]
    print("# Q.7b: the matcher on qwen3.5:35b-a3b against qwen3.5:9b\n")
    print("Vacancies of the Q.6 set matched by both models: %d of %d; 35B run 1 covers %d vacancies in all, run 2 %d.\n" % (
        len(both), len(first43), len(m1), len(m2)))

    print("## Verdict shares (Q.6 set)\n")
    b = [v for j in both if str(j) in before["matches"] for v in before["matches"][str(j)]["record"]["verdicts"]]
    print("- Q.6 matcher (9B, before the Q.7 changes): " + shares(b))
    print("- Q.7 matcher, 9B: " + shares([v for j in both for v in m9[j][1]["verdicts"]]))
    print("- Q.7 matcher, 35B run 1: " + shares([v for j in both for v in m1[j][1]["verdicts"]]))
    if m2:
        print("- Q.7 matcher, 35B run 2: " + shares([v for j in both if j in m2 for v in m2[j][1]["verdicts"]]))

    print("\n## Stability: same verdict for the same requirement in both 35B runs\n")
    if m2:
        same = total = vac_same = vac_total = 0
        for j in both:
            if j not in m2:
                continue
            a, c2 = m1[j][1]["verdicts"], m2[j][1]["verdicts"]
            if m1[j][1]["requirementsHash"] != m2[j][1]["requirementsHash"] or len(a) != len(c2):
                continue
            s = sum(1 for x, y in zip(a, c2) if x["verdict"] == y["verdict"])
            same += s
            total += len(a)
            vac_total += 1
            vac_same += s == len(a)
        print("- requirements with the same verdict: %d of %d (%.1f %%)" % (same, total, 100.0 * same / max(total, 1)))
        print("- vacancies where every verdict is the same: %d of %d" % (vac_same, vac_total))
        fit_same = sum(1 for j in both if j in m2 and you(f1.get(j)) is not None and you(f1.get(j)) == you(f2.get(j)))
        print("- vacancies with the same You fit: %d of %d; largest difference %s points" % (
            fit_same, len(both), max((abs(you(f1[j]) - you(f2[j])) for j in both if j in m2 and you(f1.get(j)) is not None and you(f2.get(j)) is not None), default="n/a")))

    print("\n## Time per vacancy (matching only)\n")
    for label, mm in (("35B run 1", m1), ("35B run 2", m2), ("9B (Q.7 run)", m9)):
        if not mm:
            continue
        rows = sorted((ts(c), j) for j, (c, _r) in mm.items())
        d = [rows[i][0] - rows[i - 1][0] for i in range(1, len(rows)) if 0 < rows[i][0] - rows[i - 1][0] <= 600]
        if d:
            print("- %s: %d timed, median %.1f s, p90 %.1f s, mean %.1f s, max %.1f s; whole run %.0f s" % (
                label, len(d), statistics.median(d), pct(d, 0.9), statistics.mean(d), max(d), rows[-1][0] - rows[0][0]))

    print("\n## You fit, 9B and 35B\n")
    print("| job | company / title | 9B | 35B run 1 | 35B run 2 |\n|---|---|---|---|---|")
    table = []
    for j in both:
        company, title = jobs.get(j, ("?", "?"))
        table.append((j, company, title, you(f9.get(j)), you(f1.get(j)), you(f2.get(j)) if m2 else None))
    for j, company, title, a, b1, b2 in sorted(table, key=lambda r: (r[4] is None, r[4] if r[4] is not None else 0)):
        mark = " **(unrelated role)**" if any(k in title.lower() for k in UNRELATED) else ""
        mark = " **(%s, named in the brief)**" % NAMED[j] if j in NAMED else mark
        print("| %d | %s / %s%s | %s | %s | %s |" % (j, company, title, mark, a, b1, b2))
    for label, idx in (("9B", 3), ("35B run 1", 4), ("35B run 2", 5)):
        xs = [r[idx] for r in table if r[idx] is not None]
        if xs:
            print("\n- %s: min %d, median %.0f, max %d; below 50: %d of %d" % (label, min(xs), statistics.median(xs), max(xs), sum(1 for x in xs if x < 50), len(xs)))

    # the Q.7 hand-check sample, reproduced from the 9B verdicts with the same seed
    rnd = random.Random(int(o["seed"] or 7))
    lines = {"unclear": [], "no": []}
    for j in both:
        reqs = exts[j]["requirements"]
        for v in m9[j][1]["verdicts"]:
            if v["verdict"] in lines:
                lines[v["verdict"]].append((j, v["index"], reqs[v["index"]], v))
    for verdict, n in (("unclear", 20), ("no", 10)):
        pool = lines[verdict]
        pick = rnd.sample(pool, min(n, len(pool)))
        print("\n## The Q.7 sample of %d '%s' lines of the 9B run, with the 35B verdicts (pool %d)\n" % (len(pick), verdict, len(pool)))
        for i, (j, idx, r, v) in enumerate(pick, 1):
            company, title = jobs.get(j, ("?", "?"))
            v1 = next((x for x in m1[j][1]["verdicts"] if x["index"] == idx), {})
            v2 = next((x for x in m2[j][1]["verdicts"] if x["index"] == idx), {}) if j in m2 else {}
            print("%d. job %d %s / %s, %s %s: **%s**" % (i, j, company, title, r["kind"], r["level"], r["text"]))
            if r.get("alternatives"):
                print("   any of: %s" % " | ".join(r["alternatives"]))
            print("   quote: %s" % r["quote"][:240])
            print("   9B: %s; 35B run 1: %s%s; 35B run 2: %s" % (
                v["verdict"], v1.get("verdict"), " (%s: %s)" % (v1["evidence"]["field"], v1["evidence"]["value"][:140]) if v1.get("evidence") else "",
                v2.get("verdict")))


if __name__ == "__main__":
    main()
