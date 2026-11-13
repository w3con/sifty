"""L.5r0 fill-rate measurement. Read-only on a copy of production (migrated by the branch binary so that job_sections exists).
Usage: python3 -I measure.py <store.sqlite> [samples-out.txt]
Prototype regexes below are NOT product code; precision is checked by printing samples."""
import sqlite3, json, re, sys, collections, random, html

DB = sys.argv[1]
SAMPLES = sys.argv[2] if len(sys.argv) > 2 else None
db = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
random.seed(7)

def strip(h):
    h = re.sub(r"<(br|/p|/li|/h\d|/div|/ul|/ol)\s*/?>", "\n", h or "")
    h = re.sub(r"<[^>]+>", "", h)
    return html.unescape(h)

# ---------- load ----------
jobs = {}
for r in db.execute("select id,title,company,status,location,original_url,description,description_status from jobs"):
    jobs[r[0]] = dict(id=r[0], title=r[1], company=r[2], status=r[3], location=(r[4] or "").strip(), url=r[5], desc=r[6] or "", dstatus=r[7])
cat = set()
for j in jobs.values():
    if 1376 <= j["id"] <= 1415 and "remote3.co/jobs/" in j["url"]:
        cat.add(j["id"])
    if j["company"] == "Unknown" and re.search(r"remotive\.com/remote-jobs/[^/]+/?$", j["url"]):
        cat.add(j["id"])
sections = {jid: json.loads(s) for jid, s in db.execute("select job_id, sections from job_sections")}
facts = {jid: json.loads(f) for jid, f in db.execute("select job_id, facts from match_job_facts")}
ext = {jid: json.loads(r) for jid, r in db.execute("select job_id, record from match_vacancy_extractions")}
fit = {jid: json.loads(r) for jid, r in db.execute("select job_id, record from match_fit_results")}

# ---------- prototype code readers ----------
CUR = r"(?:USD|EUR|GBP|CAD|AUD|CHF|SGD|US\$|\$|€|£)"
AMT = rf"{CUR}\s?\d[\d,.]*\s?(?:[kK]\b|\s?(?:thousand|million))?"
AMT2 = rf"\d[\d,.]*\s?[kK]?\s?(?:USD|EUR|GBP|CAD|AUD|CHF)\b"
SAL_WORD = re.compile(r"salary|compensation|base pay|base|pay range|pay|per year|annual|/yr|per hour|/hr|hourly|a year|yearly|OTE|total cash", re.I)
RANGE = re.compile(rf"(?:{AMT}|{AMT2})\s*(?:-|–|—|to|and)\s*(?:{AMT}|{CUR}?\s?\d[\d,.]*\s?[kK]?|{AMT2})", re.I)
SINGLE = re.compile(rf"(?:{AMT}|{AMT2})", re.I)

def read_money(text):
    """returns (found, comparable, snippet). found = a salary figure; comparable = currency + clear period or annual-size number."""
    m = RANGE.search(text)
    snip = None
    if m:
        snip = text[max(0, m.start()-30): m.end()+30].replace("\n", " ")
        # a range of money with currency is a salary range unless it is funding ("raised $10 million")
        ctx = text[max(0, m.start()-80): m.end()+80]
        if re.search(r"raised|funding|series [a-e]|valuation|revenue|ARR|investors", ctx, re.I) and not SAL_WORD.search(ctx):
            m = None
    if not m:
        for s in SINGLE.finditer(text):
            ctx = text[max(0, s.start()-80): s.end()+80]
            if SAL_WORD.search(ctx) and not re.search(r"raised|funding|series [a-e]|valuation|revenue|ARR|investors|billion", ctx, re.I):
                m = s
                snip = ctx.replace("\n", " ")
                break
    if not m:
        return (False, False, None)
    ctx = text[max(0, m.start()-60): m.end()+60]
    has_cur = bool(re.search(CUR, m.group(0)) or re.search(r"USD|EUR|GBP|CAD|AUD|CHF", ctx))
    has_period = bool(re.search(r"per year|annual|/yr|a year|yearly|per hour|/hr|hourly|per month|monthly|per annum|p\.a\.", ctx, re.I))
    nums = [float(x.replace(",", "")) for x in re.findall(r"\d[\d,]*\.?\d*", m.group(0)) if x.replace(",", "").replace(".", "").isdigit()]
    big = any(n >= 10000 for n in nums) or bool(re.search(r"\d\s?[kK]\b", m.group(0)))
    return (True, has_cur and (has_period or big), snip)

MODE_STRICT = {
    "remote": re.compile(r"fully remote|100% remote|remote[- ]first|remote[- ](?:position|role|job|work|opportunity|only)|this (?:is a )?remote|work(?:ing)? remotely|work from (?:home|anywhere)|\bWFH\b|\(remote\)|remote\s*[-·/|,]\s*(?:us|usa|uk|eu|worldwide|global|anywhere|united)|location\s*:\s*remote|\bremote\s*\|", re.I),
    "hybrid": re.compile(r"\bhybrid\b", re.I),
    "office": re.compile(r"on-?site\b|in[- ]office|in the office|office[- ]based|\bin[- ]person\b", re.I),
}
def read_mode_strict(text):
    return sorted(k for k, rx in MODE_STRICT.items() if rx.search(text))

MODE_RE = {
    "remote": re.compile(r"\bremote(?:ly)?\b|work from (?:home|anywhere)|\bWFH\b|fully distributed|distributed team", re.I),
    "hybrid": re.compile(r"\bhybrid\b", re.I),
    "office": re.compile(r"on-?site|in[- ]office|in the office|office[- ]based|onsite|on site", re.I),
}
def read_mode(text):
    return sorted(k for k, rx in MODE_RE.items() if rx.search(text))

CONTRACT_STRICT = re.compile(r"\bfull[- ]time\b(?!\s+(?:work\s+)?(?:experience|equivalent|study|student|MBA|program))|\bpart[- ]time\b|\bpermanent (?:role|position|contract|employment|basis)|\bfixed[- ]term (?:contract|role|position|basis)|\bfreelanc(?:e|er)\b|\bindependent contractor|\bcontract(?:or)? (?:role|position|basis|opportunity|engagement|to hire)|\bcontract-to-hire|\b(?:this is an? |12[- ]month |6[- ]month |temporary )contract\b|\binternship (?:position|role|program)|\bW-?2\b|\b1099\b|employment type|job type|type of employment", re.I)
def read_contract_strict(text):
    m = CONTRACT_STRICT.search(text)
    return text[max(0, m.start()-30): m.end()+30].replace("\n", " ") if m else None
def contract_kind(text):
    t = text
    k = set()
    if re.search(r"freelanc|independent contractor|contractor|contract (?:role|position|basis|opportunity|engagement|to hire)|contract-to-hire|1099|\bB2B contract", t, re.I): k.add("contract")
    if re.search(r"permanent|\bW-?2\b|employee benefits|employment contract", t, re.I): k.add("employment")
    if re.search(r"full[- ]time", t, re.I): k.add("full-time")
    if re.search(r"part[- ]time", t, re.I): k.add("part-time")
    if re.search(r"internship", t, re.I): k.add("internship")
    return k

CONTRACT_RE = re.compile(r"\bfull[- ]time\b|\bpart[- ]time\b|\bpermanent\b|\bfixed[- ]term\b|\bfreelanc\w*|\bcontractor\b|\bcontract (?:role|position|basis|to hire|employment)|\bcontract-to-hire\b|\bindependent contractor\b|\binternship\b|\bB2B\b|\bW-?2\b|\b1099\b|\bemployee\b", re.I)
def read_contract(text):
    m = CONTRACT_RE.search(text)
    return text[max(0, m.start()-30): m.end()+30].replace("\n", " ") if m else None

LOC_LINE = re.compile(r"(?:^|\n)\s*(?:location|office|based in|headquarters|work location|where)\s*[:\-–]\s*([^\n]{3,80})", re.I)
def read_location(text):
    m = LOC_LINE.search(text)
    return m.group(1).strip() if m else None

OFFER_VOCAB = {
    "salary": None,  # via read_money
    "equity": re.compile(r"\bequity\b|stock option|\bRSUs?\b|\bESOP\b|\btokens?\b(?! of)|share options|stock grant", re.I),
    "health": re.compile(r"health(?:care)?\b|medical|dental|vision|insurance", re.I),
    "time_off": re.compile(r"\bPTO\b|paid time off|vacation|annual leave|time off|days? (?:of )?(?:paid )?(?:leave|holiday)|holidays|unlimited (?:pto|vacation|time off)|parental leave|sick (?:leave|days)", re.I),
    "retirement": re.compile(r"401\(?k\)?|pension|retirement", re.I),
    "learning": re.compile(r"learning (?:budget|stipend|allowance)|education (?:budget|stipend)|conference|training budget|professional development", re.I),
    "equipment": re.compile(r"equipment|laptop|home office|hardware", re.I),
    "bonus": re.compile(r"\bbonus\b|commission|profit[- ]sharing", re.I),
    "flexible": re.compile(r"flexible (?:hours|schedule|working)|async|four[- ]day|4[- ]day", re.I),
}

OFFER_CLASSES = {"Compensation", "Benefits"}
LOC_CLASSES = {"Location"}

def class_text(secs, classes):
    return "\n".join((s.get("heading") or "") + "\n" + strip(s["content"]) for s in secs if s["class"] in classes)

# ---------- populations ----------
pop_all = [j for j in jobs if j not in cat]
pop_text = [j for j in pop_all if j in sections]                 # a sectioned (readable) text
pop_new = [j for j in pop_text if jobs[j]["status"] == "new"]
P = {"all non-category": pop_all, "with readable text": pop_text, "of those status new": pop_new}

def pct(a, b): return f"{100*a/b:5.1f} %" if b else "  n/a"
out = []
def p(*a): out.append(" ".join(str(x) for x in a))

p("## Populations")
p(f"all rows {len(jobs)}; excluded category pages {len(cat)}; non-category {len(pop_all)}; with readable text (sections) {len(pop_text)}; of those new {len(pop_new)}")
p("statuses in non-category:", dict(collections.Counter(jobs[j]['status'] for j in pop_all)))
p("model extraction present (non-category):", sum(1 for j in pop_all if j in ext), "; zero requirements:", sum(1 for j in pop_all if j in ext and not ext[j].get('requirements')))

# ---------- tile values per layer ----------
agree_rows = []
tile_layers = {t: {} for t in ("money", "location", "work_mode", "contract")}
samples = collections.defaultdict(list)

for jid in pop_all:
    j = jobs[jid]; f = facts.get(jid, {}); e = ext.get(jid, {}); secs = sections.get(jid, [])
    full = strip(j["desc"]) if not secs else "\n".join((s.get("heading") or "") + "\n" + strip(s["content"]) for s in secs)
    offer_txt = class_text(secs, OFFER_CLASSES)
    # MONEY
    cash = (f.get("compensation") or {}).get("cash") or {}
    s1 = bool(cash.get("min") or cash.get("max"))
    m_off = read_money(offer_txt) if offer_txt else (False, False, None)
    m_all = read_money(full)
    mc = (e.get("conditions") or {}).get("compensation") or {}
    s3 = bool(mc.get("value") and re.search(r"\d", mc["value"]))
    s3cmp = bool(s3 and re.search(CUR + r"|USD|EUR|GBP|CAD|AUD|CHF", mc["value"]) and re.search(r"year|annual|month|hour|\d\s?[kK]|\d{2,3},\d{3}", mc["value"], re.I))
    tile_layers["money"][jid] = dict(s1=s1, s2_sec=m_off[0], s2_all=m_all[0], s3=s3, cmp_s2=m_off[1] or (m_all[1] and m_all[0]), cmp_s2sec=m_off[1], cmp_s2all=m_all[1], cmp_s3=s3cmp)
    if m_all[0] and not m_off[0]: samples["money s2_all-only"].append((jid, m_all[2]))
    if m_off[0]: samples["money s2_sec"].append((jid, m_off[2]))
    if s3: samples["money s3 model"].append((jid, mc["value"]))
    # LOCATION
    hl = bool(((f.get("geography") or {}).get("hiringLocations")))
    s1loc = bool(j["location"]) or hl
    loc_sec = read_location(class_text(secs, LOC_CLASSES)) or (class_text(secs, LOC_CLASSES).strip() or None) if secs else None
    loc_all = read_location(full)
    lm = (e.get("conditions") or {}).get("location") or {}
    s3loc = bool(lm.get("value"))
    tile_layers["location"][jid] = dict(s1=s1loc, s2_sec=bool(loc_sec), s2_all=bool(loc_all), s3=s3loc)
    if loc_all and not s1loc and not loc_sec: samples["location s2_all-only"].append((jid, loc_all))
    # WORK MODE
    wm = ((f.get("conditions") or {}).get("workModes")) or []
    # source hint: location text such as TELECOMMUTE / Remote
    locmode = bool(re.search(r"telecommute|\bremote\b|\bhybrid\b|on-?site", j["location"], re.I))
    s1wm = bool(wm) or locmode
    first_txt = full[:1500]
    mode_off = read_mode_strict(class_text(secs, {"Location", "Benefits", "Role", "About"})) if secs else []
    mode_all = read_mode_strict(full)
    mode_all_loose = read_mode(full)
    mm = (e.get("conditions") or {}).get("workMode") or {}
    s3wm = bool(mm.get("value"))
    tile_layers["work_mode"][jid] = dict(s1=s1wm, s2_sec=bool(mode_off), s2_all=bool(mode_all), s3=s3wm, loose=bool(mode_all_loose))
    agree_rows.append(("work_mode", jid, mm.get("value"), mode_all, mode_all_loose))
    if mode_all and not (s1wm or s3wm): samples["work_mode strict s2_all-only"].append((jid, str(mode_all) + " | " + (re.search(r".{40}" + "|".join(rx.pattern for rx in MODE_STRICT.values()) + r".{40}", full, re.I|re.S) or [''])[0].replace("\n"," ") if False else str(mode_all)))
    # CONTRACT
    eng = (f.get("conditions") or {}).get("engagement") or {}
    s1c = bool(eng.get("type"))
    c_sec = read_contract_strict(class_text(secs, {"Compensation", "Benefits", "Role", "About", "Location"})) if secs else None
    c_all = read_contract_strict(full)
    c_all_loose = read_contract(full)
    em = (e.get("conditions") or {}).get("engagement") or {}
    s3c = bool(em.get("value"))
    tile_layers["contract"][jid] = dict(s1=s1c, s2_sec=bool(c_sec), s2_all=bool(c_all), s3=s3c, loose=bool(c_all_loose))
    agree_rows.append(("contract", jid, em.get("value"), contract_kind(full), bool(c_all_loose)))
    if c_all and not (s1c or s3c): samples["contract strict s2_all-only"].append((jid, c_all))

def report_tile(t, pop, name):
    L = tile_layers[t]
    n = len(pop)
    s1 = sum(L[j]["s1"] for j in pop)
    s12 = sum(L[j]["s1"] or L[j]["s2_sec"] for j in pop)
    s12all = sum(L[j]["s1"] or L[j]["s2_all"] for j in pop)
    s123 = sum(L[j]["s1"] or L[j]["s2_sec"] or L[j]["s3"] for j in pop)
    s123all = sum(L[j]["s1"] or L[j]["s2_all"] or L[j]["s3"] for j in pop)
    s3o = sum(L[j]["s3"] for j in pop)
    p(f"| {t:9} | {pct(s1,n)} | {pct(s12,n)} | {pct(s12all,n)} | {pct(s3o,n)} | {pct(s123,n)} | {pct(s123all,n)} | {n-s123all} ({pct(n-s123all,n)}) |")

for name, pop in P.items():
    p(f"\n## Tile fill, population: {name} ({len(pop)})")
    p("| tile      | structured | +code in classified sections | +code in whole text | model only | structured+sections+model | structured+whole text+model | still 'not stated' |")
    p("|---|---|---|---|---|---|---|---|")
    for t in tile_layers: report_tile(t, pop, name)

# money comparability
p("\n## Money: stated, and comparable with the goal (currency + period or annual-size number)")
for name, pop in P.items():
    n = len(pop); L = tile_layers["money"]
    any_val = sum(L[j]["s1"] or L[j]["s2_all"] or L[j]["s3"] for j in pop)
    cmp_any = sum(L[j]["s1"] or L[j]["cmp_s2all"] or L[j]["cmp_s3"] for j in pop)
    p(f"- {name} ({n}): a figure is stated for {any_val} ({pct(any_val,n)}); of them comparable by code {cmp_any} ({pct(cmp_any,n)} of all)")

# ---------- current baseline: what the existing fit already decides per goal criterion ----------
p("\n## Baseline today: per goal criterion, verdict in the stored Fits you result (non-category)")
crit = collections.defaultdict(collections.Counter); reasons = collections.defaultdict(collections.Counter)
for jid in pop_all:
    L = fit[jid]["lists"]
    seen = set()
    for key, label in (("matches", "match"), ("mismatches", "mismatch"), ("toClarify", "to clarify")):
        for it in L[key]:
            if it.get("origin") == "goal":
                crit[it["criterion"]][label] += 1; reasons[(it["criterion"], label)][it.get("reason")] += 1
p("| criterion | match | mismatch | to clarify |"); p("|---|---|---|---|")
for k in sorted(crit): p(f"| {k} | {crit[k]['match']} | {crit[k]['mismatch']} | {crit[k]['to clarify']} |")
for (k, l), c in sorted(reasons.items()):
    if l == "to clarify": p(f"  - {k} to clarify reasons: {dict(c.most_common(4))}")

# ---------- sections ----------
p("\n## Sections: recognised groups (jobs with readable text)")
def has(j, cl): return any(s["class"] in cl for s in sections[j])
n = len(pop_text)
for label, cl in (("Requirements", {"Requirements"}), ("Nice to have", {"Nice to have"}), ("Requirements or Nice to have", {"Requirements", "Nice to have"}),
                  ("Compensation", {"Compensation"}), ("Benefits", {"Benefits"}), ("Offer = Compensation or Benefits", {"Compensation", "Benefits"}),
                  ("Responsibilities", {"Responsibilities"}), ("Location", {"Location"})):
    c = sum(has(j, cl) for j in pop_text)
    p(f"- {label}: {c} ({pct(c,n)})")
both = sum(has(j, {"Requirements", "Nice to have"}) and has(j, {"Compensation", "Benefits"}) for j in pop_text)
neither = sum(not has(j, {"Requirements", "Nice to have"}) and not has(j, {"Compensation", "Benefits"}) for j in pop_text)
p(f"- both a requirements-type and an offer-type section: {both} ({pct(both,n)}); neither: {neither} ({pct(neither,n)})")

# offer vocabulary in text of offer sections vs whole text
p("\n## What they offer: vocabulary hits (jobs with readable text)")
p("| item | in Compensation/Benefits sections | in whole text |"); p("|---|---|---|")
for k, rx in OFFER_VOCAB.items():
    a = b = 0
    for j in pop_text:
        secs = sections[j]
        full = "\n".join(strip(s["content"]) + " " + (s.get("heading") or "") for s in secs)
        off = class_text(secs, OFFER_CLASSES)
        if k == "salary":
            a += bool(off and read_money(off)[0]); b += read_money(full)[0]
        else:
            a += bool(off and rx.search(off)); b += bool(rx.search(full))
    p(f"| {k} | {a} ({pct(a,n)}) | {b} ({pct(b,n)}) |")
anyoff = sum(1 for j in pop_text if any((OFFER_VOCAB[k].search(class_text(sections[j], OFFER_CLASSES)) if OFFER_VOCAB[k] else read_money(class_text(sections[j], OFFER_CLASSES))[0]) for k in OFFER_VOCAB))
p(f"- at least one offer item read from Compensation/Benefits sections: {anyoff} ({pct(anyoff,n)})")

# ---------- agreement of the code readers with the model (where the model gave a value) ----------
p("\n## Agreement of prototype code readers with the model's value (where the model stated one)")
wm_rows = [r for r in agree_rows if r[0] == "work_mode" and r[2]]
def agree_mode(rows, idx):
    ok = hit = exact = 0
    for _, jid, mv, strict, loose in rows:
        codes = strict if idx == 3 else loose
        if codes: hit += 1
        mvn = {"office": "office", "remote": "remote", "hybrid": "hybrid"}.get(mv)
        if mvn in codes: ok += 1
        if codes == [mvn]: exact += 1
    return hit, ok, exact
for lbl, idx in (("strict", 3), ("loose", 4)):
    hit, ok, ex = agree_mode(wm_rows, idx)
    p(f"- work mode, {lbl} reader, on {len(wm_rows)} vacancies with a model value: reader found something in {hit} ({pct(hit,len(wm_rows))}); contains the model's mode in {ok} ({pct(ok,len(wm_rows))}); equals it exactly in {ex} ({pct(ex,len(wm_rows))})")
ct_rows = [r for r in agree_rows if r[0] == "contract" and r[2]]
mapv = lambda v: "contract" if re.search(r"contract|freelanc", v, re.I) else ("employment" if re.search(r"employ|full|salar", v, re.I) else "other")
a_emp = [r for r in ct_rows if mapv(r[2]) == "employment"]; a_con = [r for r in ct_rows if mapv(r[2]) == "contract"]
p(f"- contract: model says employment in {len(a_emp)}, contract/freelance in {len(a_con)}")
p(f"  - model 'employment': code sees employment words (permanent/W-2/employee benefits/employment contract) in {sum('employment' in r[3] for r in a_emp)}, full-time only in {sum(('full-time' in r[3]) and 'employment' not in r[3] and 'contract' not in r[3] for r in a_emp)}, contract words in {sum('contract' in r[3] for r in a_emp)}, nothing in {sum(not r[3] for r in a_emp)}")
p(f"  - model 'contract': code sees contract words in {sum('contract' in r[3] for r in a_con)}, nothing in {sum(not r[3] for r in a_con)}")
n_ct = len(pop_text)
kinds = collections.Counter()
for r in agree_rows:
    if r[0] == "contract" and r[1] in set(pop_text):
        for k in r[3]: kinds[k] += 1
p("- contract words found in whole text (strict-ish kinds), jobs:", dict(kinds))

# money agreement and explicit contract type
p("\n## Money and contract: extra checks")
L = tile_layers["money"]
both = [j for j in pop_all if L[j]["s3"]]
p(f"- money: model stated a figure for {len(both)}; the code reader on the whole text also finds a figure in {sum(L[j]['s2_all'] for j in both)} ({pct(sum(L[j]['s2_all'] for j in both),len(both))})")
codeonly = [j for j in pop_all if L[j]["s2_all"] and not L[j]["s3"]]
p(f"- money: code finds a figure where the model gave none: {len(codeonly)}; (18 of them read by eye above: all were real pay figures)")
EXPLICIT = re.compile(r"\bfreelanc(?:e|er)\b|\bindependent contractor|\bcontract(?:or)? (?:role|position|basis|opportunity|engagement|to hire)|\bcontract-to-hire|\b(?:this is an? |12[- ]month |6[- ]month |temporary )contract\b|\bpermanent (?:role|position|contract|employment|basis)|\bfixed[- ]term (?:contract|role|position|basis)|\bW-?2\b|\b1099\b|\bemployment contract|\bcontract of employment|(?:job|employment) type\s*:?\s*(?:full|part|contract|permanent)", re.I)
SCHED = re.compile(r"\bfull[- ]time\b(?!\s+(?:work\s+)?(?:experience|equivalent|study|student|program))|\bpart[- ]time\b", re.I)
for name, pop in P.items():
    n = len(pop); expl = sched_only = 0; model_or_struct = 0; union = 0
    for j in pop:
        secs = sections.get(j, [])
        full = "\n".join(strip(x["content"]) for x in secs) if secs else strip(jobs[j]["desc"])
        e1 = bool(EXPLICIT.search(full)); sc = bool(SCHED.search(full))
        ms = tile_layers["contract"][j]["s1"] or tile_layers["contract"][j]["s3"]
        model_or_struct += ms; union += (ms or e1); expl += e1
        sched_only += (sc and not (ms or e1))
    p(f"- contract type stated ({name}, {n}): structured+model {model_or_struct} ({pct(model_or_struct,n)}); + explicit contract words in text {union} ({pct(union,n)}); only a schedule word (full-time/part-time) on top: {sched_only} ({pct(sched_only,n)})")

# ---------- requirements decided by code ----------
p("\n## Requirements: extraction and verdicts by code (stored fit results; matching pass is off)")
withreq = [j for j in pop_all if j in ext and ext[j].get("requirements")]
p(f"- vacancies with >=1 model-extracted requirement: {len(withreq)} of {len(pop_all)} ({pct(len(withreq),len(pop_all))})")
tot = collections.Counter(); alldec = 0; someund = 0; perjob = []
bykind = collections.defaultdict(collections.Counter)
for j in withreq:
    cn = collections.Counter()
    for it in fit[j]["lists"]["matches"] + fit[j]["lists"]["missing"] + fit[j]["lists"]["mismatches"]:
        if it.get("origin") == "requirement": cn[{"match": "match", "no": "no", "partial": "partial", "unclear": "unclear"}.get(it.get("verdict"), "unclear")] += 1
    tot.update(cn)
    dec = cn["match"] + cn["no"]
    n_req = sum(cn.values())
    perjob.append((dec, n_req))
    if n_req and dec == n_req: alldec += 1
    if n_req and dec < n_req: someund += 1
    for it in fit[j]["lists"]["matches"] + fit[j]["lists"]["missing"] + fit[j]["lists"]["mismatches"]:
        if it.get("origin") == "requirement":
            bykind[it.get("kind")][it.get("verdict")] += 1
total_req = sum(tot.values())
p(f"- requirements in total: {total_req}; by code: match {tot['match']} ({pct(tot['match'],total_req)}), no {tot['no']} ({pct(tot['no'],total_req)}), partial {tot['partial']} ({pct(tot['partial'],total_req)}), unclear {tot['unclear']} ({pct(tot['unclear'],total_req)})")
p(f"- decided ✓/✕ by code (match + no): {tot['match']+tot['no']} ({pct(tot['match']+tot['no'],total_req)})")
p(f"- vacancies where every requirement is decided by code: {alldec} ({pct(alldec,len(withreq))}); with at least one undecided (needs the owner's ✓/✕): {someund} ({pct(someund,len(withreq))})")
p("- by requirement kind (verdict counts):")
for k, c in sorted(bykind.items(), key=lambda kv: -sum(kv[1].values())): p(f"   - {k}: {dict(c)}")
lv = collections.Counter()
for j in withreq:
    for r in ext[j]["requirements"]: lv[r.get("level")] += 1
p("- levels in extraction:", dict(lv))

print("\n".join(out))
if SAMPLES:
    with open(SAMPLES, "w") as fh:
        for k, v in samples.items():
            fh.write(f"## {k} ({len(v)})\n")
            for jid, s in random.sample(v, min(18, len(v))):
                fh.write(f"  {jid}: {s}\n")
