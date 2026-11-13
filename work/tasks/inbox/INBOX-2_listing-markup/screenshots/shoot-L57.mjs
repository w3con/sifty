// L.5.7 Playwright screenshots: section view, Key Responsibilities heading,
// Web3 Careers without gaps, disabled toggle, collapsed section.
// Run from the backend/workers/playwright/ directory (needs playwright package):
//   CARD_DIR=<path-to-card> node shoot-L57.mjs
// Server must be running on 9199 against a production data copy.
//
// Job notes (as of 2026-11-12 reshoot):
//   8    — Amber Group (preselected/Shortlist): has <h3>Key Responsibilities</h3>
//   950  — MTS (triage/Inbox): 14 sections, vocabulary labels visible
//   1029 — Coinbag (triage/Inbox): 6 fully-headed sections
//   27   — CoinList (triage/Inbox): 1 section, toggle disabled
import { chromium } from 'playwright';
const BASE = 'http://127.0.0.1:9199';
const CARD = process.env.CARD_DIR;
if (!CARD) { console.error('CARD_DIR not set'); process.exit(1); }

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
const page = await ctx.newPage();

async function shot(name) {
  await page.screenshot({ path: `${CARD}/screenshots/${name}.png`, fullPage: false });
  console.log(`Screenshot: ${name}.png`);
}

// Wait until the app renders at least one job row.
async function waitForList() {
  await page.waitForSelector('[data-job-id]', { timeout: 15000 });
}

// Click a job row by ID, then wait for the detail panel to settle.
async function selectJob(id) {
  const btn = page.locator(`[data-job-id="${id}"]`).first();
  await btn.scrollIntoViewIfNeeded();
  await btn.click();
  await page.waitForTimeout(600);
}

// Wait until the "By section" button becomes enabled (secondary job-detail fetch delivered
// sections). Falls through quietly on timeout so callers can check the disabled state.
async function waitForBySection(timeoutMs = 8000) {
  try {
    await page.waitForFunction(() => {
      const btn = [...document.querySelectorAll('.text-mode button')].find(b => b.textContent?.trim() === 'By section');
      return btn && !btn.hasAttribute('disabled');
    }, { timeout: timeoutMs });
  } catch { /* disabled is a valid expected state for some jobs */ }
}

async function bySectionEnabled() {
  const btn = page.locator('.text-mode button', { hasText: 'By section' });
  const disabled = await btn.getAttribute('disabled');
  return disabled === null;
}

async function clickBySection() {
  await page.locator('.text-mode button', { hasText: 'By section' }).click();
  await page.waitForTimeout(400);
}

async function clickOriginal() {
  await page.locator('.text-mode button', { hasText: 'Original' }).click();
  await page.waitForTimeout(300);
}

// Load the app (triage/Inbox view by default).
await page.goto(BASE);
await waitForList();

// --- Job 8: Amber Group "Full-Stack Software Engineer" ---
// In Shortlist (preselected). Has <h3>Key Responsibilities</h3> in descriptionHtml.
// 4 sections, 3 headed → By section enabled.
console.log('=== Job 8 — Amber Group: Key Responsibilities heading in Original ===');
await page.locator('nav.stages button', { hasText: 'Shortlist' }).click();
await page.waitForTimeout(600);
await waitForList();
await selectJob(8);
// Original view: show <h3>Key Responsibilities</h3>
await shot('L57-job8-original-key-responsibilities');
{
  const h3text = await page.locator('.listing-text h3').evaluateAll(els => els.map(e => e.textContent));
  console.log('h3 headings in Original:', h3text);
}

// By section view
await waitForBySection();
const j8enabled = await bySectionEnabled();
console.log('By section enabled:', j8enabled);
await clickBySection();
await shot('L57-job8-by-section');
{
  const labels = await page.locator('.section-label').evaluateAll(els => els.map(e => e.textContent?.trim()));
  const blockCount = await page.locator('.section-block').count();
  const openCount = await page.locator('.section-block[open]').count();
  console.log('Section labels:', labels);
  console.log('Total blocks:', blockCount, '/ Open blocks:', openCount);
}
// Click a collapsed section to expand it
{
  const closed = page.locator('.section-block:not([open]) .section-label').first();
  if (await closed.count() > 0) {
    await closed.click();
    await page.waitForTimeout(300);
    await shot('L57-job8-collapsed-section-expanded');
    console.log('Clicked a collapsed section to expand it');
  }
}
// Check no raw HTML tags in section view
{
  const rawTags = await page.locator('.section-view').evaluate(el =>
    el.textContent?.includes('<p>') || el.textContent?.includes('<li>') || el.textContent?.includes('<strong>')
  );
  console.log('Raw HTML tags visible in section view:', rawTags);
}

// --- Switch back to Inbox for triage-stage jobs ---
await page.locator('nav.stages button', { hasText: 'Inbox' }).click();
await page.waitForTimeout(600);
await waitForList();

// --- Job 950: MTS "Senior Software Engineer" ---
// Triage. 14 sections — good coverage of vocabulary class labels.
// After vocabulary corrections: Compensation, Benefits, About, Process now classified.
await selectJob(950);
console.log('\n=== Job 950 — MTS: By section view ===');
await waitForBySection();
const j950enabled = await bySectionEnabled();
console.log('By section enabled:', j950enabled);
await clickBySection();
await shot('L57-job950-by-section');
{
  const labels = await page.locator('.section-label').evaluateAll(els => els.map(e => e.textContent?.trim()));
  const blockCount = await page.locator('.section-block').count();
  console.log('Section labels:', labels);
  console.log('Total blocks:', blockCount);
}

// --- Job 1029: Coinbag "Full Stack Engineer" ---
// Triage. 6 fully-headed sections → good coverage of section types.
await selectJob(1029);
console.log('\n=== Job 1029 — Coinbag: 6-section view ===');
await waitForBySection();
const j1029enabled = await bySectionEnabled();
console.log('By section enabled:', j1029enabled);
await clickBySection();
await shot('L57-job1029-by-section');
{
  const labels = await page.locator('.section-label').evaluateAll(els => els.map(e => e.textContent?.trim()));
  const blockCount = await page.locator('.section-block').count();
  console.log('Section labels:', labels);
  console.log('Total blocks:', blockCount);
}

// --- Job 27: CoinList "Senior Smart Contract Engineer" ---
// Triage. 1 section, 0 headings → By section disabled.
await selectJob(27);
console.log('\n=== Job 27 — CoinList: By section toggle disabled ===');
await page.waitForTimeout(1500); // secondary fetch settles; toggle should stay disabled
const j27enabled = await bySectionEnabled();
console.log('By section enabled (expected false):', j27enabled);
{
  const btn = page.locator('.text-mode button', { hasText: 'By section' });
  const disabledAttr = await btn.getAttribute('disabled');
  const title = await btn.getAttribute('title');
  console.log('disabled attr present:', disabledAttr !== null);
  console.log('title:', title);
}
await shot('L57-job27-coinlist-toggle-disabled');

await browser.close();
console.log('\nDone — all screenshots saved to', `${CARD}/screenshots/`);
