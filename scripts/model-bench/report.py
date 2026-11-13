#!/usr/bin/env python3
"""Compare language models on the two model passes of Sifty and print a Markdown report.

Two sub-commands, both read-only and standard-library only (Python 3.9+):

  report.py extraction --run "9B=a1.json,a2.json" --run "35B=b1.json" [--expected expected.json] [--out report.md]
  report.py matching   --run "9B=a1.sqlite,a2.sqlite" --run "35B=b1.sqlite" [--out report.md]

extraction  Compares runs of the extraction pass (requirements read from a vacancy text). The run files are
            written by run-extraction.sh. With --expected (hand-written expectations, see README.md) the report
            scores every model against them; without it the report still shows volume, speed, repeatability
            and how much the models agree with each other.
matching    Compares the matching pass (requirements against your profile). Each run is a copy of the store
            taken after the pass finished (run-matching.sh). Shows verdict shares, the "You fit" score, time per
            vacancy, repeatability between runs of one model and agreement between models. There is no ground
            truth here: a requirement for a skill the profile lacks is "no", one it shows is "match", and only
            you know your profile, so check a sample of disagreements by hand (the report prints one).

A label may name several files separated by commas: they are repeated runs of the same model and are used to
measure repeatability.
"""
import argparse
import json
import random
import re
import sqlite3
import statistics
import sys
from collections import Counter
from datetime import datetime

# ---------------------------------------------------------------------------------------------------------
# shared helpers


def parse_runs(specs):
    runs = []
    for spec in specs:
        if "=" not in spec:
            sys.exit("--run must look like LABEL=file1[,file2]: %r" % spec)
        label, files = spec.split("=", 1)
        runs.append((label.strip(), [f.strip() for f in files.split(",") if f.strip()]))
    return runs


def pct(xs, p):
    xs = sorted(xs)
    k = (len(xs) - 1) * p
    lo, hi = int(k), min(int(k) + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (k - lo)


def table(header, rows):
    out = ["| " + " | ".join(header) + " |", "|" + "---|" * len(header)]
    out += ["| " + " | ".join(str(c) for c in r) + " |" for r in rows]
    return "\n".join(out)


def words(text):
    return set(re.findall(r"[a-z0-9+#.]{2,}", (text or "").lower()))


def similar(a, b, threshold=0.5):
    wa, wb = words(a), words(b)
    if not wa or not wb:
        return False
    return len(wa & wb) / len(wa | wb) >= threshold


# ---------------------------------------------------------------------------------------------------------
# extraction


def load_extraction(path):
    data = json.load(open(path))
    jobs = {}
    for j in data["jobs"]:
        reqs = j.get("requirements")
        if isinstance(reqs, str):
            reqs = json.loads(reqs)
        conds = j.get("conditions")
        if isinstance(conds, str):
            conds = json.loads(conds)
        jobs[j["id"]] = {"requirements": reqs or [], "conditions": conds or {}, "seconds": j.get("seconds", 0.0),
                         "attempts": j.get("attempts", 1), "error": j.get("error"), "title": j.get("title", "")}
    return {"model": data.get("model", "?"), "extractor": data.get("extractorVersion", "?"), "jobs": jobs}


def find_expected(exp, reqs):
    for i, r in enumerate(reqs):
        blob = " | ".join([r.get("text", ""), *(r.get("alternatives") or []), r.get("quote", "")]).lower()
        if any(re.search(p, blob) for p in exp["match"]):
            return i
    return None


def condition_ok(expected, got):
    if expected == "either":
        return True
    if expected == "null":
        return got is None
    return got is not None and expected in str(got.get("value", "")).lower()


def score_run(expected, run):
    t = Counter()
    for job in expected["jobs"]:
        got = run["jobs"].get(job["id"])
        t["expected"] += len(job["requirements"])
        t["cond_total"] += sum(1 for v in job["conditions"].values() if v != "either")
        if got is None or got["error"]:
            t["failed"] += 1
            continue
        reqs = got["requirements"]
        for e in job["requirements"]:
            i = find_expected(e, reqs)
            if i is None:
                continue
            t["found"] += 1
            t["level"] += e["level"] in ("any", reqs[i].get("level"))
            if e.get("group"):
                t["group_total"] += 1
                t["group"] += len(reqs[i].get("alternatives") or []) >= 2
        for f in job.get("forbidden", []):
            t["forbidden"] += sum(1 for r in reqs if any(re.search(p, r.get("text", "").strip().lower()) for p in f["match"]))
        for name, want in job["conditions"].items():
            if want != "either":
                t["cond"] += condition_ok(want, got["conditions"].get(name))
    return t


def extraction_report(args):
    runs = [(label, [load_extraction(f) for f in files]) for label, files in parse_runs(args.run)]
    expected = json.load(open(args.expected)) if args.expected else None
    out = ["# Extraction pass: model comparison\n"]
    out.append("Runs: " + ", ".join("%s (%s, %d run%s)" % (label, rs[0]["model"], len(rs), "" if len(rs) == 1 else "s") for label, rs in runs))
    out.append("Extractor version: " + ", ".join(sorted({r["extractor"] for _l, rs in runs for r in rs})) + "\n")

    if expected:
        out.append("## Score against the hand-written expectations\n")
        n_req = sum(len(j["requirements"]) for j in expected["jobs"])
        n_cond = sum(1 for j in expected["jobs"] for v in j["conditions"].values() if v != "either")
        n_grp = sum(1 for j in expected["jobs"] for e in j["requirements"] if e.get("group"))
        rows = []
        for label, rs in runs:
            for k, r in enumerate(rs, 1):
                t = score_run(expected, r)
                rows.append((label, k, "%d / %d" % (t["found"], n_req), "%d / %d" % (t["level"], t["found"]),
                             "%d / %d" % (t["group"], n_grp), t["forbidden"], "%d / %d" % (t["cond"], n_cond), t["failed"]))
        out.append(table(["Model", "Run", "Expected requirements found", "Level correct", "Alternative groups kept",
                          "Forbidden items", "Conditions correct", "Failed vacancies"], rows))
        out.append("\nHigher is better except forbidden items and failed vacancies. Expectations cover %d vacancies; "
                   "a requirement counts as found when its pattern matches the model's text or quote.\n" % len(expected["jobs"]))

    out.append("## Volume and speed\n")
    rows = []
    for label, rs in runs:
        secs = [j["seconds"] for r in rs for j in r["jobs"].values() if not j["error"]]
        counts = [len(j["requirements"]) for r in rs for j in r["jobs"].values() if not j["error"]]
        fails = sum(1 for r in rs for j in r["jobs"].values() if j["error"])
        retries = sum(1 for r in rs for j in r["jobs"].values() if j["attempts"] > 1)
        rows.append((label, len(secs), "%.1f" % statistics.median(counts) if counts else "-",
                     "%.1f" % statistics.median(secs) if secs else "-", "%.1f" % pct(secs, 0.9) if secs else "-", fails, retries))
    out.append(table(["Model", "Vacancies read", "Requirements per vacancy (median)", "Seconds per vacancy (median)",
                      "Seconds (90th percentile)", "Errors", "Needed a retry"], rows))
    out.append("")

    out.append("## Repeatability: the same model asked twice\n")
    rows = []
    for label, rs in runs:
        if len(rs) < 2:
            continue
        same = total = 0
        for jid, a in rs[0]["jobs"].items():
            b = rs[1]["jobs"].get(jid)
            if not b or a["error"] or b["error"]:
                continue
            for r in a["requirements"]:
                total += 1
                same += any(similar(r.get("text"), x.get("text")) for x in b["requirements"])
        rows.append((label, "%d of %d (%.0f %%)" % (same, total, 100.0 * same / max(total, 1))))
    out.append(table(["Model", "Requirements of run 1 found again in run 2"], rows) if rows else "Only one run per model, so repeatability is not measured.")
    out.append("")

    out.append("## Agreement between models\n")
    out.append("For every pair of models, the share of one model's requirements that the other also reports "
               "(similar words, same vacancy). It is not a quality score: two models can agree and both be wrong. "
               "Where the numbers are low, read the requirements the models disagree on.\n")
    rows = []
    for la, ra in runs:
        row = [la]
        for lb, rb in runs:
            if la == lb:
                row.append("-")
                continue
            same = total = 0
            for jid, a in ra[0]["jobs"].items():
                b = rb[0]["jobs"].get(jid)
                if not b or a["error"] or b["error"]:
                    continue
                for r in a["requirements"]:
                    total += 1
                    same += any(similar(r.get("text"), x.get("text")) for x in b["requirements"])
            row.append("%.0f %%" % (100.0 * same / total) if total else "-")
        rows.append(row)
    out.append(table(["Row's requirements found by column →"] + [l for l, _ in runs], rows))
    out.append("")
    print("\n".join(out))


# ---------------------------------------------------------------------------------------------------------
# matching


def ts(s):
    return datetime.strptime(s, "%Y-%m-%dT%H:%M:%S.%fZ").timestamp()


def load_matching(path, version=None):
    spec = path.split(":", 1)
    path, version = spec[0], (spec[1] if len(spec) > 1 else version)
    con = sqlite3.connect("file:%s?mode=ro" % path, uri=True, timeout=30)
    matches = {}
    for jid, created, v, rec in con.execute(
            "select job_id, created_at, matcher_version, record from match_requirement_matches order by created_at"):
        if version is None or version in v:
            matches[jid] = (created, v, json.loads(rec))
    fits = {r[0]: json.loads(r[1]) for r in con.execute("select job_id, record from match_fit_results")}
    exts = {r[0]: json.loads(r[1]) for r in con.execute("select job_id, record from match_vacancy_extractions")}
    jobs = {r[0]: (r[1], r[2]) for r in con.execute("select id, company, title from jobs")}
    return {"matches": matches, "fits": fits, "exts": exts, "jobs": jobs}


def you_fit(fit):
    y = (fit or {}).get("youFit") or {}
    return y.get("score") if y.get("status") == "scored" else None


def matching_report(args):
    runs = [(label, [load_matching(f) for f in files]) for label, files in parse_runs(args.run)]
    out = ["# Matching pass: model comparison\n"]
    common = None
    for _label, rs in runs:
        for r in rs:
            ids = set(r["matches"])
            common = ids if common is None else common & ids
    common = sorted(common or [])
    out.append("Vacancies matched in every run: %d.\n" % len(common))

    out.append("## Verdicts\n")
    rows = []
    for label, rs in runs:
        c = Counter(v["verdict"] for j in common for v in rs[0]["matches"][j][2]["verdicts"])
        n = sum(c.values()) or 1
        rows.append([label, n] + ["%d (%.1f %%)" % (c.get(k, 0), 100.0 * c.get(k, 0) / n) for k in ("match", "partial", "no", "unclear")])
    out.append(table(["Model", "Requirements", "match", "partial", "no", "unclear"], rows))
    out.append("\n`unclear` means the model would not decide. Too many wastes the score; too few from a small model usually means it guesses.\n")

    out.append("## \"You fit\" score\n")
    rows = []
    for label, rs in runs:
        xs = [you_fit(rs[0]["fits"].get(j)) for j in common]
        xs = [x for x in xs if x is not None]
        if xs:
            rows.append((label, len(xs), min(xs), "%.0f" % statistics.median(xs), max(xs), sum(1 for x in xs if x < 50)))
    out.append(table(["Model", "Scored vacancies", "Lowest", "Median", "Highest", "Below 50"], rows))
    out.append("\nA useful matcher gives low scores to vacancies unrelated to the profile and high ones to close matches. "
               "Look at the per-vacancy table below for the vacancies you know well.\n")

    out.append("## Time per vacancy\n")
    rows = []
    for label, rs in runs:
        stamps = sorted(ts(c) for c, _v, _r in rs[0]["matches"].values())
        d = [stamps[i] - stamps[i - 1] for i in range(1, len(stamps)) if 0 < stamps[i] - stamps[i - 1] <= 600]
        if d:
            rows.append((label, len(d), "%.1f" % statistics.median(d), "%.1f" % pct(d, 0.9), "%.0f" % (stamps[-1] - stamps[0])))
    out.append(table(["Model", "Timed vacancies", "Median (s)", "90th percentile (s)", "Whole run (s)"], rows))
    out.append("")

    out.append("## Repeatability: the same model asked twice\n")
    rows = []
    for label, rs in runs:
        if len(rs) < 2:
            continue
        same = total = fit_same = fit_total = 0
        for j in common:
            if j not in rs[1]["matches"]:
                continue
            a, b = rs[0]["matches"][j][2], rs[1]["matches"][j][2]
            if a["requirementsHash"] != b["requirementsHash"]:
                continue
            for x, y in zip(a["verdicts"], b["verdicts"]):
                total += 1
                same += x["verdict"] == y["verdict"]
            fa, fb = you_fit(rs[0]["fits"].get(j)), you_fit(rs[1]["fits"].get(j))
            if fa is not None and fb is not None:
                fit_total += 1
                fit_same += abs(fa - fb) <= 5
        rows.append((label, "%d of %d (%.0f %%)" % (same, total, 100.0 * same / max(total, 1)),
                     "%d of %d" % (fit_same, fit_total)))
    out.append(table(["Model", "Same verdict in both runs", "\"You fit\" within 5 points"], rows) if rows else
               "Only one run per model, so repeatability is not measured.")
    out.append("")

    out.append("## Agreement between models (same requirement, same verdict)\n")
    rows = []
    for la, ra in runs:
        row = [la]
        for lb, rb in runs:
            if la == lb:
                row.append("-")
                continue
            same = total = 0
            for j in common:
                a, b = ra[0]["matches"][j][2], rb[0]["matches"][j][2]
                if a["requirementsHash"] != b["requirementsHash"]:
                    continue
                for x, y in zip(a["verdicts"], b["verdicts"]):
                    total += 1
                    same += x["verdict"] == y["verdict"]
            row.append("%.0f %%" % (100.0 * same / total) if total else "-")
        rows.append(row)
    out.append(table(["Model"] + [l for l, _ in runs], rows))
    out.append("")

    out.append("## \"You fit\" per vacancy\n")
    jobs = runs[0][1][0]["jobs"]
    rows = []
    for j in common:
        company, title = jobs.get(j, ("?", "?"))
        rows.append([j, "%s / %s" % (company, title)] + [you_fit(rs[0]["fits"].get(j)) for _l, rs in runs])
    out.append(table(["Job", "Company / title"] + [l for l, _ in runs], rows))
    out.append("")

    # a reproducible sample of disagreements for the hand check
    out.append("## Disagreements to check by hand\n")
    out.append("A random sample of requirements on which the first model and another one gave different verdicts "
               "(seed %d). For each, decide from your own profile who is right.\n" % args.seed)
    first_label, first = runs[0][0], runs[0][1][0]
    rnd = random.Random(args.seed)
    for label, rs in runs[1:]:
        pool = []
        for j in common:
            a, b = first["matches"][j][2], rs[0]["matches"][j][2]
            if a["requirementsHash"] != b["requirementsHash"]:
                continue
            reqs = first["exts"][j]["requirements"]
            for x, y in zip(a["verdicts"], b["verdicts"]):
                if x["verdict"] != y["verdict"]:
                    pool.append((j, reqs[x["index"]], x["verdict"], y["verdict"]))
        out.append("### %s against %s: %d disagreements\n" % (first_label, label, len(pool)))
        for j, r, va, vb in rnd.sample(pool, min(args.sample, len(pool))):
            company, title = jobs.get(j, ("?", "?"))
            out.append("- %s / %s, %s %s: **%s**. %s says `%s`, %s says `%s`." % (
                company, title, r["kind"], r["level"], r["text"], first_label, va, label, vb))
        out.append("")
    print("\n".join(out))


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    e = sub.add_parser("extraction")
    e.add_argument("--run", action="append", required=True, help='LABEL=file1.json[,file2.json]')
    e.add_argument("--expected", help="hand-written expectations (optional)")
    m = sub.add_parser("matching")
    m.add_argument("--run", action="append", required=True, help='LABEL=store1.sqlite[:matcher-version][,store2.sqlite]')
    m.add_argument("--seed", type=int, default=7)
    m.add_argument("--sample", type=int, default=10)
    for p in (e, m):
        p.add_argument("--out", help="write the report to this file instead of printing it")
    args = ap.parse_args()
    if args.out:
        sys.stdout = open(args.out, "w")
    {"extraction": extraction_report, "matching": matching_report}[args.cmd](args)


if __name__ == "__main__":
    main()
