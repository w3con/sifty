// Screenshots for Round 5: + and − action buttons replacing ✓ and ✕
// Usage: node shoot-r5-buttons.mjs <shots-dir>
// Server must be on port 9198. Import Playwright from the worktree's legacy/app.
import { createRequire } from 'module';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
// Resolve playwright from the worktree's legacy/app node_modules
const require = createRequire(import.meta.url);
// legacy/app node_modules live in the main checkout, not duplicated per worktree
const playwrightPath = '/Volumes/SSD/Dev/tknff/sifty/legacy/app/node_modules/playwright';
const { chromium } = require(playwrightPath);

const SHOTS_DIR = process.argv[2] || __dirname;
const PORT = 9198;
const BASE = `http://localhost:${PORT}`;

async function shot(page, name) {
  const dest = path.join(SHOTS_DIR, name);
  await page.screenshot({ path: dest, fullPage: false });
  console.log('SHOT:', name);
}

async function navToJob(page, jobId) {
  const el = page.locator(`[data-job-id="${jobId}"]`);
  await el.first().click();
  await page.waitForTimeout(1500);
}

async function findAskRow(page, text) {
  const rows = page.locator('li.ask-row');
  const count = await rows.count();
  for (let i = 0; i < count; i++) {
    const t = await rows.nth(i).textContent();
    if (t && t.includes(text)) return rows.nth(i);
  }
  return null;
}

async function findAskRowByKind(page, kind) {
  // Find any undecided row of the given kind
  const rows = page.locator(`li.ask-row.unknown`);
  const count = await rows.count();
  for (let i = 0; i < count; i++) {
    const r = rows.nth(i);
    // We can't easily filter by kind from the DOM, so just return the first
    return r;
  }
  return null;
}

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage();
await page.setViewportSize({ width: 1440, height: 900 });

await page.goto(BASE, { waitUntil: 'networkidle' });
await page.waitForTimeout(1500);

// ── 1. Skill row (undecided): + and − buttons ─────────────────────────────
// Job 1083 has "C++ or C" (undecided alternatives), but first find a plain skill row
// Job 950 (Shakepay): should have skill rows
await navToJob(page, 950);
await page.waitForSelector('.ask-row', { timeout: 10000 }).catch(() => {});
await page.waitForTimeout(500);

// Find first undecided skill row (class 'unknown')
const unknownRows950 = page.locator('li.ask-row.unknown');
const count950 = await unknownRows950.count();
console.log('Job 950 undecided rows:', count950);

if (count950 > 0) {
  const firstRow = unknownRows950.first();
  await firstRow.scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  await shot(page, 'R5-skill-undecided.png');

  // Check buttons are + and −
  const addBtn = firstRow.locator('button.add').first();
  const dismissBtn = firstRow.locator('button.dismiss').first();
  const addText = await addBtn.textContent().catch(() => '?');
  const dismissText = await dismissBtn.textContent().catch(() => '?');
  console.log('Add button text:', addText, '| Dismiss button text:', dismissText);
}

// ── 2. Alternatives row: + Name buttons ──────────────────────────────────
await navToJob(page, 1279);
await page.waitForSelector('.ask-row', { timeout: 10000 }).catch(() => {});
await page.waitForTimeout(500);

const ddRow = await findAskRow(page, 'Datadog / OpenSearch');
if (ddRow) {
  await ddRow.scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  await shot(page, 'R5-slash-before.png');

  const addBtn = ddRow.locator('button.add').first();
  if (await addBtn.count() > 0) {
    await addBtn.click();
    await page.waitForTimeout(1500);
    await shot(page, 'R5-slash-alternatives.png');

    // Check alternative buttons show "+ Name"
    const altBtns = ddRow.locator('button.add');
    const count = await altBtns.count();
    for (let i = 0; i < count; i++) {
      const t = await altBtns.nth(i).textContent().catch(() => '?');
      console.log(`Alt button ${i}: "${t}"`);
    }
  }
}

// ── 3. List row (comma): + and − buttons ────────────────────────────────
await navToJob(page, 1011);
await page.waitForSelector('.ask-row', { timeout: 10000 }).catch(() => {});
await page.waitForTimeout(500);

const awsRow = await findAskRow(page, 'AWS, Kubernetes, Terraform');
if (awsRow) {
  await awsRow.scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  await shot(page, 'R5-list-row.png');
  const addBtn = awsRow.locator('button.add').first();
  const dismissBtn = awsRow.locator('button.dismiss').first();
  const addText = await addBtn.textContent().catch(() => '?');
  const dismissText = await dismissBtn.textContent().catch(() => '?');
  console.log('List add btn:', addText, '| List dismiss btn:', dismissText);
} else {
  console.log('WARN: AWS list row not found in job 1011');
}

// ── 4. Non-skill row (experience): + and − buttons ───────────────────────
// Find any undecided non-skill row in the first few jobs
await navToJob(page, 1083);
await page.waitForSelector('.ask-row', { timeout: 10000 }).catch(() => {});
await page.waitForTimeout(500);

const unknownRows1083 = page.locator('li.ask-row.unknown');
const count1083 = await unknownRows1083.count();
console.log('Job 1083 undecided rows:', count1083);
if (count1083 > 0) {
  const r = unknownRows1083.first();
  await r.scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  await shot(page, 'R5-nonsk-undecided.png');
}

// ── 5. Timing: three + clicks in a row on one vacancy ───────────────────
await navToJob(page, 950);
await page.waitForSelector('.ask-row', { timeout: 10000 }).catch(() => {});
await page.waitForTimeout(500);

// Intercept /api/jobs to count list fetches
let listFetches = 0;
page.on('request', req => {
  if (req.url().includes('/api/jobs') && !req.url().includes('/api/jobs/')) {
    listFetches++;
  }
});

const rows = page.locator('li.ask-row.unknown');
const rowCount = await rows.count();
console.log('Rows available for timing:', rowCount);

const timings = [];
for (let i = 0; i < Math.min(3, rowCount); i++) {
  const row = rows.nth(i);
  await row.scrollIntoViewIfNeeded();
  const addBtn = row.locator('button.add').first();
  if (await addBtn.count() === 0) continue;

  const t0 = Date.now();
  await addBtn.click();
  // Wait for the row to change state (not unknown) or timeout
  try {
    await row.locator('.ask-row').first().waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
    await page.waitForTimeout(500);
  } catch {}
  const elapsed = Date.now() - t0;
  timings.push(elapsed);
  console.log(`Click ${i+1} elapsed: ${elapsed} ms`);
  await page.waitForTimeout(200);

  // Re-find rows since the list may shift after a row changes
  await page.waitForTimeout(300);
}

console.log('List fetches after 3 clicks:', listFetches);
console.log('Timings:', timings);

await browser.close();
console.log('DONE');
