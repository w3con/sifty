#!/usr/bin/env python3
"""INBOX-2 L.1: summarise fetched pages (what markup the source delivers).
Usage: python3 -I inventory_pages.py PAGES_DIR FETCH_LOG.tsv
PAGES_DIR holds <key>.body files; FETCH_LOG has: key, url, 'status size content-type final-url'.
For each page: HTTP status, kind (json / rss / html), JSON-LD JobPosting description (raw form and tag counts),
embedded JSON (__NEXT_DATA__, API payload) description fields, and the page's own structural tag counts.
Read-only, no network."""
import sys, re, json, html as H, collections, os
pdir, log = sys.argv[1], sys.argv[2]
TAGS = ['h1', 'h2', 'h3', 'h4', 'p', 'ul', 'ol', 'li', 'br', 'strong', 'b', 'em']

def tagcount(s):
    c = collections.Counter(m.lower() for m in re.findall(r'<(/?[a-zA-Z][a-zA-Z0-9]*)', s) if not m.startswith('/'))
    return {t: c[t] for t in TAGS if c[t]}

def form(s):
    if re.search(r'<(p|li|br|ul|ol|h[1-6]|strong|div)\b', s, re.I): return 'raw HTML'
    if re.search(r'&lt;(p|li|br|ul|ol|h[1-6]|strong|div)\b', s, re.I): return 'escaped HTML (&lt;p&gt;)'
    if re.search(r'(^|\n)#{1,4} |\*\*[^*\n]+\*\*', s): return 'markdown'
    if '\n' in s: return 'plain with newlines'
    return 'plain, no newlines'

def walk(o, path=''):
    if isinstance(o, dict):
        for k, v in o.items(): yield from walk(v, path + '/' + k)
    elif isinstance(o, list):
        for i, v in enumerate(o): yield from walk(v, path + '[]')
    elif isinstance(o, str) and len(o) > 300 and re.search(r'desc|content|text|body|intro|requirement', path, re.I):
        yield path, o

for line in open(log):
    key, url, res = line.rstrip('\n').split('\t')[:3]
    code, size, ctype, final = (res.split(' ', 3) + ['', '', '', ''])[:4]
    print('## %s  %s' % (key, url))
    print('   HTTP %s, %s bytes, %s' % (code, size, ctype))
    path = os.path.join(pdir, key + '.body')
    body = open(path, 'rb').read().decode('utf-8', 'replace') if os.path.exists(path) else ''
    if not code.startswith('2'):
        print('   NOT FETCHED (no retry): %s' % body[:120].replace('\n', ' ')); continue
    t = re.search(r'<title[^>]*>(.*?)</title>', body, re.S | re.I)
    if t: print('   title: %s' % re.sub(r'\s+', ' ', H.unescape(t.group(1)))[:100])
    if re.search(r'Just a moment|captcha|Access Denied|cf-chl|datadome', body[:6000], re.I): print('   BLOCK PAGE SIGNS present')
    s = body.lstrip()
    if s[:1] in '{[':
        try:
            j = json.loads(body)
            for p, v in list(walk(j))[:4]: print('   json field %s: %s, %d chars, tags %s' % (p, form(v), len(v), tagcount(v)))
            continue
        except Exception as e: print('   json parse failed', e)
    if '<rss' in body[:2000] or '<feed' in body[:2000]:
        items = re.findall(r'<description>(.*?)</description>', body, re.S)
        print('   RSS items %d' % body.count('<item>'))
        for d in items[:2]:
            raw = d.strip(); cd = re.match(r'<!\[CDATA\[(.*)\]\]>$', raw, re.S)
            v = cd.group(1) if cd else H.unescape(raw)
            print('   feed description: wrapper=%s decoded form=%s, %d chars, tags %s' % ('CDATA' if cd else 'entity-escaped', form(v), len(v), tagcount(v)))
        continue
    ld = re.findall(r'<script[^>]*application/ld\+json[^>]*>(.*?)</script>', body, re.S | re.I)
    found = False
    for blk in ld:
        try: data = json.loads(blk)
        except Exception: continue
        stack = [data]
        while stack:
            o = stack.pop()
            if isinstance(o, list): stack += o
            elif isinstance(o, dict):
                if o.get('@type') == 'JobPosting' or 'JobPosting' in (o.get('@type') if isinstance(o.get('@type'), list) else []):
                    d = o.get('description', '')
                    print('   JSON-LD JobPosting description: %s, %d chars, tags %s' % (form(d), len(d), tagcount(d))); found = True
                    print('      decoded once: tags %s' % tagcount(H.unescape(d)))
                stack += [v for v in o.values() if isinstance(v, (dict, list))]
    if not found: print('   no JSON-LD JobPosting (ld+json blocks: %d)' % len(ld))
    nd = re.search(r'<script id="__NEXT_DATA__"[^>]*>(.*?)</script>', body, re.S)
    if nd:
        try:
            j = json.loads(nd.group(1))
            fs = list(walk(j))
            print('   __NEXT_DATA__ present; long text fields: %d' % len(fs))
            for p, v in fs[:3]: print('      %s: %s, %d chars, tags %s' % (p, form(v), len(v), tagcount(v)))
        except Exception as e: print('   __NEXT_DATA__ unparsable', e)
    clean = re.sub(r'<(script|style|noscript|svg)\b.*?</\1>', '', body, flags=re.S | re.I)
    print('   page body tags (scripts removed): %s' % tagcount(clean))
