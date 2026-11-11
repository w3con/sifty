#!/usr/bin/env python3
"""Compare extraction-regression output with expected.json and print a markdown table.

usage: compare.py expected.json run-a.json [run-b.json ...]
Each run file is the output of TestLiveExtractionRegression (one model run over the job set).
"""
import json, re, sys


def texts(req):
    return [req.get("text", ""), *(req.get("alternatives") or []), req.get("quote", "")]


def find(exp, received):
    for i, r in enumerate(received):
        blob = " | ".join(texts(r)).lower()
        if any(re.search(p, blob) for p in exp["match"]):
            return i
    return None


def cond_ok(expected, got):
    if expected == "either":
        return True
    if expected == "null":
        return got is None
    return got is not None and expected in got["value"].lower()


def main():
    expected = json.load(open(sys.argv[1]))
    runs = [(p, json.load(open(p))) for p in sys.argv[2:]]
    for path, run in runs:
        print(f"\n## Run `{path}` — extractor `{run['extractorVersion']}`, model `{run['model']}`\n")
        totals = dict(found=0, expected=0, level=0, group=0, forbidden=0, cond=0, cond_total=0)
        byid = {j["id"]: j for j in run["jobs"]}
        for job in expected["jobs"]:
            got = byid.get(job["id"])
            print(f"### {job['id']} — {job['label']}\n")
            if got is None or got.get("error"):
                print(f"**Run failed:** {got and got.get('error')}\n")
                totals["expected"] += len(job["requirements"])
                totals["cond_total"] += len(job["conditions"])
                continue
            reqs = json.loads(got["requirements"]) if isinstance(got["requirements"], str) else got["requirements"]
            reqs = reqs or []
            conds = got["conditions"] if isinstance(got["conditions"], dict) else json.loads(got["conditions"])
            print(f"{len(reqs)} requirements kept, {got['seconds']:.1f} s, {got['attempts']} attempt(s).\n")
            print("| Expected | Level | Received | Found | Level ok | Group ok |")
            print("|---|---|---|---|---|---|")
            used = set()
            for e in job["requirements"]:
                totals["expected"] += 1
                i = find(e, reqs)
                if i is None:
                    print(f"| {e['name']} | {e['level']} | — | ✗ | | |")
                    continue
                used.add(i)
                r = reqs[i]
                totals["found"] += 1
                level_ok = e["level"] in ("any", r["level"])
                totals["level"] += level_ok
                group_ok = ""
                if e.get("group"):
                    ok = len(r.get("alternatives") or []) >= 2
                    totals["group"] += ok
                    group_ok = "✔" if ok else "✗ (no alternatives)"
                print(f"| {e['name']} | {e['level']} | {r['text']} [{r['kind']}, {r['level']}] | ✔ | {'✔' if level_ok else '✗ ' + r['level']} | {group_ok} |")
            bad = []
            for f in job.get("forbidden", []):
                for r in reqs:
                    if any(re.search(p, r["text"].strip().lower()) for p in f["match"]):
                        bad.append(f"{r['text']} [{r['kind']}, {r['level']}] — {f['name']}")
            totals["forbidden"] += len(bad)
            extra = [f"{r['text']} [{r['kind']}, {r['level']}]" for i, r in enumerate(reqs) if i not in used]
            print(f"\nForbidden items received: {len(bad)}" + ("".join(f"\n- ✗ {b}" for b in bad)))
            print(f"\nOther requirements received (not in the expected list; judge by hand): {len(extra)}" + ("".join(f"\n- {x}" for x in extra)))
            print("\n| Condition | Expected | Received | OK |\n|---|---|---|---|")
            for name, want in job["conditions"].items():
                g = conds.get(name)
                totals["cond_total"] += 1
                ok = cond_ok(want, g)
                totals["cond"] += ok
                print(f"| {name} | {want} | {g['value'] if g else 'null'} | {'✔' if ok else '✗'} |")
            print()
        t = totals
        print(f"**Totals for this run:** expected requirements found {t['found']}/{t['expected']}; level correct {t['level']}/{t['found']}; "
              f"groups with alternatives {t['group']}/{sum(1 for j in expected['jobs'] for e in j['requirements'] if e.get('group'))}; "
              f"forbidden items received {t['forbidden']}; conditions correct {t['cond']}/{t['cond_total']}.\n")


main()
