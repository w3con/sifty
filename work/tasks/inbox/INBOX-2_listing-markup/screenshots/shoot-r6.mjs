// Screenshots for Round 6: post-merge smoke check
// Usage: node shoot-r6.mjs <shots-dir>
// Server must be on port 9198. Import Playwright from the worktree's legacy/app.
import { createRequire } from 'module';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);
const playwrightPath = '/Volumes/SSD/Dev/tknff/sifty/legacy/app/node_modules/playwright';
const { chromium } = require(playwrightPath);

const SHOTS_DIR = process.argv[2] || __dirname;
const PORT = 9198;
const BASE = `http://localhost:${PORT}`;

async function shot(page, name) {
  const dest = path.join(SHOTS_DIR, name);
  await page.screenshot({ path: dest, fullPage: false });
  console.log('SHOT:', name);
  return dest;
}

async function navToJob(page, jobId) {
  const el = page.locator(`[data-job-id="${jobId}"]`);
  await el.first().click();
  await page.waitForTimeout(2000);
}

// Scroll the detail panel (main.detail) which has its own overflow scroll
async function scrollDetail(page, y) {
  await page.evaluate((scrollY) => {
    const detail = document.querySelector('main.detail');
    if (detail) detail.scrollTo(0, scrollY);
    else window.scrollTo(0, scrollY);
  }, y);
  await page.waitForTimeout(300);
}

const browser = await chromium.launch({ headless: true });
const context = await browser.newContext();
const page = await context.newPage();
await page.setViewportSize({ width: 1440, height: 900 });

await page.goto(BASE, { waitUntil: 'networkidle' });
await page.waitForTimeout(2000);

// ── 1. Job 950 — Fits you band tiles at 1440 ─────────────────────────────
console.log('\n=== Job 950 ===');
await navToJob(page, 950);
// Wait for band to render
await page.waitForSelector('.band-tile', { timeout: 10000 }).catch(() => {
  console.log('WARN: .band-tile not found, trying other selectors');
});
await scrollDetail(page, 0);
await shot(page, 'R6-950-band-1440.png');

// Check tiles content
const tiles950 = page.locator('.band-tile');
const tileCount950 = await tiles950.count();
console.log('Band tile count (950):', tileCount950);
for (let i = 0; i < tileCount950; i++) {
  const t = await tiles950.nth(i).textContent().catch(() => '?');
  console.log(`  tile ${i}: "${t.trim().replace(/\s+/g, ' ')}"`);
}

// What they ask card — scroll past the band to show the ask/offer cards area
await scrollDetail(page, 350);
await shot(page, 'R6-950-ask-card.png');

// Check undecided rows and + / − buttons — scroll to undecided rows area
const unknownRows = page.locator('li.ask-row.unknown');
const unknownCount = await unknownRows.count();
console.log('Undecided rows (950):', unknownCount);
if (unknownCount > 0) {
  // Scroll past first few rows to show the + and − buttons clearly
  const midRow = unknownRows.nth(Math.min(2, unknownCount - 1));
  await midRow.scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  // Verify button symbols
  const firstRow = unknownRows.first();
  const addBtn = firstRow.locator('button.add').first();
  const dismissBtn = firstRow.locator('button.dismiss').first();
  const addText = await addBtn.textContent().catch(() => 'NOT FOUND');
  const dismissText = await dismissBtn.textContent().catch(() => 'NOT FOUND');
  console.log('Add button text:', JSON.stringify(addText));
  console.log('Dismiss button text:', JSON.stringify(dismissText));
  if (addText.trim() !== '+') console.error('ERROR: add button should be +, got:', addText);
  if (dismissText.trim() !== '−') console.error('ERROR: dismiss button should be −, got:', dismissText);
}

// ── 2. Job 950 at 400px — responsive/overview spacing ───────────────────
console.log('\n=== Job 950 narrow ===');
await page.setViewportSize({ width: 400, height: 900 });
await scrollDetail(page, 0);
await shot(page, 'R6-950-band-400.png');

// ── 3. Job 1011 at 1440 — full view ──────────────────────────────────────
console.log('\n=== Job 1011 ===');
await page.setViewportSize({ width: 1440, height: 900 });
await navToJob(page, 1011);
await page.waitForSelector('.band-tile', { timeout: 10000 }).catch(() => {});
await scrollDetail(page, 0);
await shot(page, 'R6-1011-band-1440.png');

const tiles1011 = page.locator('.band-tile');
const tileCount1011 = await tiles1011.count();
console.log('Band tile count (1011):', tileCount1011);
for (let i = 0; i < tileCount1011; i++) {
  const t = await tiles1011.nth(i).textContent().catch(() => '?');
  console.log(`  tile ${i}: "${t.trim().replace(/\s+/g, ' ')}"`);
}

// What they offer card — scroll to show it after the band tiles
await scrollDetail(page, 250);
await shot(page, 'R6-1011-offer-card.png');

// ── 4. What they ask — + / − buttons on job 1011 ────────────────────────
console.log('\n=== What they ask buttons job 1011 ===');
const unknownRows1011 = page.locator('li.ask-row.unknown');
const unknownCount1011 = await unknownRows1011.count();
console.log('Undecided rows (1011):', unknownCount1011);
if (unknownCount1011 > 0) {
  // Scroll to show several undecided rows - much further down the ask card
  const midRow1011 = unknownRows1011.nth(Math.min(10, unknownCount1011 - 1));
  await midRow1011.scrollIntoViewIfNeeded();
  await page.waitForTimeout(500);
  await shot(page, 'R6-1011-ask-buttons.png');
  const addBtn = unknownRows1011.first().locator('button.add').first();
  const dismissBtn = unknownRows1011.first().locator('button.dismiss').first();
  const addText = await addBtn.textContent().catch(() => 'NOT FOUND');
  const dismissText = await dismissBtn.textContent().catch(() => 'NOT FOUND');
  console.log('Add button text:', JSON.stringify(addText));
  console.log('Dismiss button text:', JSON.stringify(dismissText));
}

// ── 5. Job 1294 — Remote (US & Canada) location tile ─────────────────────
console.log('\n=== Job 1294 — remote restriction ===');
await navToJob(page, 1294);
await page.waitForSelector('.band-tile', { timeout: 10000 }).catch(() => {});
await scrollDetail(page, 0);
await shot(page, 'R6-1294-remote-restriction.png');

const tiles1294 = page.locator('.band-tile');
const tileCount1294 = await tiles1294.count();
console.log('Band tile count (1294):', tileCount1294);
for (let i = 0; i < tileCount1294; i++) {
  const t = await tiles1294.nth(i).textContent().catch(() => '?');
  console.log(`  tile ${i}: "${t.trim().replace(/\s+/g, ' ')}"`);
}

// ── 6. Full listing section — scroll to full listing area ────────────────
console.log('\n=== Full listing scroll job 950 ===');
// Navigate back to job 950 and scroll to the full listing section
await navToJob(page, 950);
await page.waitForSelector('.band-tile', { timeout: 10000 }).catch(() => {});
// Scroll past the band + ask/offer cards (approx 1200-1500px) to reach the full listing
await page.evaluate(() => {
  const detail = document.querySelector('main.detail');
  if (detail) detail.scrollTo(0, detail.scrollHeight * 0.6);
  else window.scrollTo(0, document.body.scrollHeight * 0.6);
});
await page.waitForTimeout(500);
await shot(page, 'R6-950-full-listing.png');

// ── 7. Overview spacing — check CSS gap/padding on band ─────────────────
console.log('\n=== Overview spacing measurement ===');
await scrollDetail(page, 0);
// Try to measure the gap between band and next section
const gap = await page.evaluate(() => {
  // Find the band element
  const band = document.querySelector('[class*="band"], .band-tile')?.closest('[class*="Band"], [class*="band-container"]');
  if (!band) {
    // Try to find by looking at the structure
    const tiles = document.querySelectorAll('.band-tile');
    if (tiles.length === 0) return 'no band found';
    const bandEl = tiles[0].closest('section, [class*="Section"], [class*="Card"]');
    if (!bandEl) return 'no band container found';
    const style = window.getComputedStyle(bandEl);
    return {
      marginBottom: style.marginBottom,
      paddingBottom: style.paddingBottom,
      gap: style.gap,
    };
  }
  const style = window.getComputedStyle(band);
  return {
    marginBottom: style.marginBottom,
    paddingBottom: style.paddingBottom,
    gap: style.gap,
  };
});
console.log('Band spacing:', gap);

await browser.close();
console.log('\nDONE');
