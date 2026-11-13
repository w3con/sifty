// Playwright script for L.4.2 — L.4 browser checks.
//
// How to run (from the worktree root):
//   node work/tasks/inbox/INBOX-2_listing-markup/screenshots/shoot-L4.mjs
//
// Prerequisites:
//   - Playwright 1.65.0 in the npx cache (~/.npm/_npx/<hash>/node_modules/playwright); the hash in the import path below is machine specific
//   - Chromium cached in ~/Library/Caches/ms-playwright
//   - Throwaway server already running on port 9199 against scratchpad data
//     (the script does NOT start or stop the server)
//
// DB rows prepared before this run:
//   84  Parity  — Tech Lead, company=Parity; has description_html
//   8   Amber Group — Full-Stack SW Engineer; LinkedIn, has <ul>/<p>
//   13  Discernis — Senior SW Engineer; has <ol>
//   27  CoinList — Senior Smart Contract Engineer; web3.career, needs_refetch=1
//   1   Lemon.io — description_html set to NULL (fallback case)
//   2   Lemon.io — description_html set to hostile string (security case)

import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { homedir } from 'node:os';

const __dir = dirname(fileURLToPath(import.meta.url));

// Playwright 1.65.0 lives in the npx cache (confirmed by: cd frontend && npx --no-install playwright --version).
// It is not installed in frontend/node_modules; it is in the npx cache.
// index.mjs exports { chromium, firefox, webkit, ... }
const playwrightPkg = resolve(homedir(), '.npm/_npx/e41f203b7505f1fb/node_modules/playwright/index.mjs');
const { chromium } = await import(playwrightPkg);

const BASE = 'http://localhost:9199';
const OUT = __dir; // screenshots land next to this script

async function shot(page, name) {
  const path = resolve(OUT, `${name}.png`);
  await page.screenshot({ path });
  console.log(`  saved ${name}.png`);
}

// Open the app, navigate to the right view, click the job row, wait for listing.
// viewName: 'triage' (Inbox) | 'applied' (Archive)
async function openJob(page, id, viewName = 'triage') {
  await page.goto(`${BASE}/`);
  // Wait for the job list to load — look for the first job row appearing
  await page.waitForSelector('.job-row', { timeout: 30000 });

  if (viewName === 'applied') {
    // Click "Archive" in the stages nav
    await page.locator('nav.stages button', { hasText: 'Archive' }).click();
    await page.waitForTimeout(600);
    // The Archive view has "Applied" / "Dismissed" subtabs — click Applied
    const appliedTab = page.locator('.archive-tabs button', { hasText: 'Applied' });
    if ((await appliedTab.count()) > 0) await appliedTab.click();
    await page.waitForTimeout(400);
  }

  // Find the job row
  const row = page.locator(`[data-job-id="${id}"]`);
  await row.waitFor({ state: 'visible', timeout: 10000 });
  await row.first().scrollIntoViewIfNeeded();
  await row.first().click();
  // Wait for the listing panel to appear
  await page.waitForSelector('.listing-text, .listing-missing', { timeout: 10000 });
  // Scroll listing into view for screenshot
  await page.locator('.listing-text, .listing-missing').first().scrollIntoViewIfNeeded();
}

async function main() {
  const browser = await chromium.launch({ headless: true });
  let failures = 0;

  // ── 1. Parity (id=84) — desktop 1440×900 ─────────────────────────────────
  console.log('\n[1] Parity (id=84) desktop');
  {
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
    await openJob(page, 84, 'triage');
    await shot(page, 'L4-parity');
    // Quick sanity: listing text exists
    const hasListing = (await page.locator('.listing-text').count()) > 0;
    console.log(`  .listing-text present: ${hasListing}`);
    await page.close();
  }

  // ── 2. LinkedIn with HTML (id=8) ─────────────────────────────────────────
  console.log('\n[2] LinkedIn with HTML (id=8)');
  {
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
    await openJob(page, 8, 'triage');
    const ulCount = await page.locator('.listing-text ul').count();
    const pCount  = await page.locator('.listing-text p').count();
    console.log(`  ul count: ${ulCount}, p count: ${pCount}`);
    await shot(page, 'L4-linkedin-html');
    await page.close();
  }

  // ── 3. Numbered list (id=13, has <ol>) ───────────────────────────────────
  console.log('\n[3] Numbered list (id=13)');
  {
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
    await openJob(page, 13, 'triage');
    const olCount = await page.locator('.listing-text ol').count();
    console.log(`  .listing-text ol count: ${olCount}`);
    if (olCount === 0) { console.warn('  WARN: no <ol> visible'); }
    await shot(page, 'L4-numbered-list');
    await page.close();
  }

  // ── 4. Web3 Careers needs_refetch (id=27) ────────────────────────────────
  // Job is in Archive / Applied view (status=applied in DB).
  // The open-time effect fires refreshSource() once — one polite HTTP request
  // to web3.career; afterwards needs_refetch should be cleared.
  console.log('\n[4] Web3 Careers needs_refetch (id=27)');
  {
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
    const externalReqs = [];
    page.on('request', req => {
      if (req.url().includes('web3.career')) externalReqs.push(req.url());
    });

    const before = await fetch(`${BASE}/api/jobs/27`).then(r => r.json());
    console.log(`  needsRefetch before open: ${before.needsRefetch}`);

    await openJob(page, 27, 'applied');
    // Wait up to 15 s for the refetch to complete
    await page.waitForTimeout(12000);

    const after = await fetch(`${BASE}/api/jobs/27`).then(r => r.json());
    console.log(`  needsRefetch after open+wait: ${after.needsRefetch}`);
    console.log(`  requests to web3.career: ${externalReqs.length} ${JSON.stringify(externalReqs.slice(0,3))}`);
    if (externalReqs.length > 1) console.warn('  WARN: more than one request to web3.career');

    await page.locator('.listing-text, .listing-missing').first().scrollIntoViewIfNeeded();
    await shot(page, 'L4-web3careers-refetch');
    await page.close();
  }

  // ── 5. Fallback — description_html=NULL (id=1) ───────────────────────────
  console.log('\n[5] Fallback (id=1, description_html=NULL)');
  {
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
    await openJob(page, 1, 'triage');
    const listingText    = await page.locator('.listing-text').count();
    const listingMissing = await page.locator('.listing-missing').count();
    console.log(`  .listing-text: ${listingText}, .listing-missing: ${listingMissing}`);
    // Confirm no raw tags appear in text content
    const rawTagsInText = await page.locator('.listing-text').evaluate(
      el => /<[a-zA-Z]/.test(el.textContent || '')
    ).catch(() => null);
    console.log(`  raw tags in textContent: ${rawTagsInText}`);
    await shot(page, 'L4-fallback');
    await page.close();
  }

  // ── 6. Hostile markup (id=2) ─────────────────────────────────────────────
  // description_html = '<p onclick="window.__pwn=1">x</p><script>window.__pwn=1</script>
  //                     <img src=x onerror="window.__pwn=1"><h3 style="color:red">Title</h3><p>text</p>'
  console.log('\n[6] Hostile case (id=2)');
  {
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
    await openJob(page, 2, 'triage');

    // 1. window.__pwn must be undefined (script never ran)
    const pwn = await page.evaluate(() => (window).__pwn);
    if (pwn !== undefined) {
      console.error(`  FAIL: window.__pwn = ${pwn} (expected undefined)`);
      failures++;
    } else {
      console.log('  PASS: window.__pwn is undefined');
    }

    // 2. No script/img/style-attr/on*-attr in .listing-text innerHTML
    const checks = await page.locator('.listing-text').evaluate(el => {
      const h = el.innerHTML;
      return {
        hasScript:   /<script/i.test(h),
        hasImg:      /<img/i.test(h),
        hasStyleAttr:/\bstyle\s*=/i.test(h),
        hasOnAttr:   /\bon\w+\s*=/i.test(h),
        h3texts: Array.from(el.querySelectorAll('h3')).map(n => n.textContent?.trim()),
        ptexts:  Array.from(el.querySelectorAll('p')).map(n => n.textContent?.trim()),
      };
    });
    console.log(`  innerHTML checks: ${JSON.stringify(checks)}`);

    if (checks.hasScript)    { console.error('  FAIL: <script> present'); failures++; }
    else                     console.log('  PASS: no <script>');
    if (checks.hasImg)       { console.error('  FAIL: <img> present');    failures++; }
    else                     console.log('  PASS: no <img>');
    if (checks.hasStyleAttr) { console.error('  FAIL: style= present');   failures++; }
    else                     console.log('  PASS: no style=');
    if (checks.hasOnAttr)    { console.error('  FAIL: on* attr present');  failures++; }
    else                     console.log('  PASS: no on* attr');
    if (!checks.h3texts.includes('Title')) console.warn('  WARN: "Title" not in h3; h3 texts: ' + JSON.stringify(checks.h3texts));
    else                                   console.log('  PASS: "Title" renders as h3');

    await shot(page, 'L4-hostile');
    await page.close();
  }

  // ── 7. Phone-width of Parity (id=84, 390×844) ────────────────────────────
  console.log('\n[7] Phone-width Parity (id=84, 390×844)');
  {
    const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
    await openJob(page, 84, 'triage');
    await shot(page, 'L4-phone');
    await page.close();
  }

  await browser.close();
  if (failures > 0) {
    console.error(`\n${failures} assertion(s) FAILED`);
    process.exit(1);
  }
  console.log('\nAll cases done.');
}

main().catch(err => { console.error(err); process.exit(1); });
