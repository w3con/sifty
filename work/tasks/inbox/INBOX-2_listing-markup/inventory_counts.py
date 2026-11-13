#!/usr/bin/env python3
"""INBOX-2 L.1: read-only inventory of how jobs.description is stored, per source.

Usage: python3 -I inventory_counts.py [path/to/store.sqlite]
Opens the database with mode=ro (never writes). Source of a vacancy = the earliest
row in `sightings` for it (65 vacancies have sightings from more than one source).
Kinds are mutually exclusive, tested in this order:
  empty    : description is ''
  html     : contains a real tag (p, li, ul, ol, br, div, h1-h6, strong, b, em, i, span, a)
  escaped  : contains &lt; or a tag name directly followed by &gt; (e.g. 'strong&gt;'): escaped/mangled markup
  metadata : no newline, shorter than 300 chars (skill/tag lists, 'pending', error placeholders)
  glued    : no newline, 300 characters or more
  newlines : plain text with at least one newline
"""
import re, sys, sqlite3, collections, statistics

DB = sys.argv[1] if len(sys.argv) > 1 else '/Volumes/SSD/Dev/tknff/sifty/data-staging/store.sqlite'
db = sqlite3.connect('file:%s?mode=ro' % DB, uri=True)
rows = db.execute("""select j.id, j.description, j.description_status,
  (select source from sightings s where s.job_id=j.id order by seen, rowid limit 1)
  from jobs j order by j.id""").fetchall()

TAG = re.compile(r'</?(p|li|ul|ol|br|div|h[1-6]|strong|b|em|i|span|a)\b[^>]*>', re.I)
ESC = re.compile(r'&lt;|(^|[\s>.;:])/?(p|li|ul|ol|br|strong|b|em|h[1-6]|div|span)&gt;', re.I)
ENT = re.compile(r'&(#\d+|#x[0-9a-fA-F]+|[a-zA-Z][a-zA-Z0-9]*);')
PLI = re.compile(r'</?(p|li|br)\b[^>]*>', re.I)
BUL = re.compile(r'^\s*([-*•●▪]|–)\s+\S', re.M)
NUM = re.compile(r'^\s*\d{1,2}[.)]\s+\S', re.M)
COL = re.compile(r'^[^\n]{2,60}:\s*$', re.M)
KINDS = ['html', 'escaped', 'newlines', 'glued', 'metadata', 'empty']

def kind(d):
    if d == '': return 'empty'
    if TAG.search(d): return 'html'
    if ESC.search(d): return 'escaped'
    if '\n' not in d: return 'glued' if len(d) >= 300 else 'metadata'
    return 'newlines'

def status_group(st):
    if st.startswith('available structured description'): return 'available structured description (JSON-LD / detail page)'
    if st.startswith('available cleaned html'): return 'available cleaned html (page body)'
    if st.startswith('available public search description'): return 'available public search description (Wellfound)'
    if st.startswith('available public API description'): return 'available public API description'
    if st.startswith('available complete hiring comment') or st.startswith('available hiring comment'): return 'hiring comment (Hacker News)'
    if st.startswith('feed description'): return 'feed description (RSS)'
    if 'metadata' in st or 'pending' in st or st.startswith('missing'): return 'metadata only / pending / missing'
    if st.startswith('detail unavailable') or st.startswith('blocked'): return 'detail unavailable / blocked'
    return st

data = [(i, d, st, src, kind(d)) for i, d, st, src in rows]
srcs = sorted({r[3] for r in data}, key=lambda s: -sum(1 for r in data if r[3] == s))

print('== 1. Totals (whole table, %d vacancies)' % len(data))
tot = collections.Counter()
for i, d, st, src, k in data:
    tot['kind:' + k] += 1
    tot['<p>/<li>/<br> tags'] += bool(PLI.search(d))
    tot['any real tag'] += bool(TAG.search(d))
    tot['&amp;'] += '&amp;' in d
    tot['any entity'] += bool(ENT.search(d))
    tot['&lt; or tag-name&gt; (escaped)'] += bool(ESC.search(d))
    tot['contains newline'] += '\n' in d
for k in sorted(tot): print('%-34s %5d' % (k, tot[k]))

print('\n== 2. Kind per source (primary source = earliest sighting); columns are counts')
print('%-30s %5s %s' % ('source', 'total', ' '.join('%9s' % k for k in KINDS)))
for s in srcs:
    rs = [r for r in data if r[3] == s]
    c = collections.Counter(r[4] for r in rs)
    print('%-30s %5d %s' % (s, len(rs), ' '.join('%9d' % c[k] for k in KINDS)))
c = collections.Counter(r[4] for r in data)
print('%-30s %5d %s' % ('ALL', len(data), ' '.join('%9d' % c[k] for k in KINDS)))

print('\n== 3. Entities per source (rows containing the entity; whole table)')
names = ['&amp;', '&gt;', '&lt;', '&nbsp;', 'other']
print('%-30s %5s %s %6s' % ('source', 'rows', ' '.join('%7s' % n for n in names), 'any'))
for s in srcs + ['ALL']:
    rs = [r for r in data if s == 'ALL' or r[3] == s]
    c = collections.Counter(); other = collections.Counter()
    for r in rs:
        found = set('&' + m + ';' for m in ENT.findall(r[1]))
        for n in names[:4]:
            if n in found: c[n] += 1
        o = found - set(names[:4])
        if o: c['other'] += 1; other.update(o)
        if found: c['any'] += 1
    print('%-30s %5d %s %6d  %s' % (s, len(rs), ' '.join('%7d' % c[n] for n in names), c['any'], dict(other) if other else ''))

print('\n== 4. Structure hints inside plain text (rows with newlines or flat text; whole table)')
print('rows with bullet lines (-,*,bullet), numbered lines (1.), short lines ending with colon')
print('%-30s %7s %7s %7s' % ('source', 'bullets', 'numbers', 'colon'))
for s in srcs + ['ALL']:
    rs = [r for r in data if (s == 'ALL' or r[3] == s) and r[4] in ('newlines', 'html', 'escaped', 'glued')]
    print('%-30s %7d %7d %7d' % (s, sum(bool(BUL.search(r[1])) for r in rs), sum(bool(NUM.search(r[1])) for r in rs), sum(bool(COL.search(r[1])) for r in rs)))

print('\n== 4b. Markdown markers and page chrome inside stored text (rows; whole table)')
MD = {'# heading line': re.compile(r'^#{1,4} \S', re.M), '**bold**': re.compile(r'\*\*[^*\n]+\*\*'), '[text](http link)': re.compile(r'\[[^\]]+\]\(https?://')}
CH = {'"All jobs" page chrome (WWR page body)': re.compile(r'All jobs'), 'category-page text "Stop applying" (Remote3)': re.compile(r'Stop applying')}
print('%-30s %s' % ('source', ' | '.join(list(MD) + list(CH))))
for s_ in srcs:
    rs = [r for r in data if r[3] == s_]
    vals = [sum(bool(rx.search(r[1])) for r in rs) for rx in list(MD.values()) + list(CH.values())]
    if any(vals): print('%-30s %s' % (s_, ' | '.join(str(v) for v in vals)))
print('%-30s %s' % ('ALL', ' | '.join(str(sum(bool(rx.search(r[1])) for r in data)) for rx in list(MD.values()) + list(CH.values()))))

print('\n== 5. Kind per source and description_status group')
c = collections.Counter((r[3], status_group(r[2]), r[4]) for r in data)
for key in sorted(c): print('%5d  %-30s %-62s %s' % (c[key], key[0], key[1], key[2]))

print('\n== 6. Median length (characters) per source and kind')
for s in srcs:
    parts = []
    for k in KINDS:
        L = [len(r[1]) for r in data if r[3] == s and r[4] == k]
        if L: parts.append('%s=%d' % (k, statistics.median(L)))
    print('%-30s %s' % (s, ' '.join(parts)))

print('\n== 7. Three stored samples per source (first 240 chars, repr; deterministic choice)')
for s in srcs:
    rs = [r for r in data if r[3] == s and r[4] != 'empty']
    byk = collections.defaultdict(list)
    for r in rs: byk[r[4]].append(r)
    order = sorted(byk, key=lambda k: -len(byk[k]))
    picks = []
    if order: picks.append(byk[order[0]][0])
    if len(order) > 1: picks.append(byk[order[1]][0])
    if len(order) > 2: picks.append(byk[order[2]][0])
    for r in byk[order[0]][::-1]:
        if len(picks) >= 3: break
        if r not in picks: picks.append(r)
    print('-- %s' % s)
    for r in picks:
        print('   id=%d kind=%s status=%r len=%d: %r' % (r[0], r[4], r[2][:48], len(r[1]), r[1][:240]))
