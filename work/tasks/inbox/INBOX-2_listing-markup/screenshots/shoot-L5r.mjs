// Playwright screenshots for L.5r1-r3 acceptance check.
//
// How to run (from the worktree root):
//   node work/tasks/inbox/INBOX-2_listing-markup/screenshots/shoot-L5r.mjs
//
// Prerequisites:
//   - Playwright 1.65.0 in the npx cache; Chromium cached in ~/Library/Caches/ms-playwright
//   - Throwaway server already running on port 9199 against a copy of the production DB
//
// Screenshots produced:
//   L5r-job950-band.png        — job 950 (MTS Senior SE, KamiwazaAI): Money ✓ $150-250k USD
//   L5r-job950-what-cards.png  — same job, What they ask and What they offer cards
//   L5r-job100-no-extraction.png — job 100: no model extraction, all tiles not_stated
//   L5r-job500-no-offer.png    — job 500: no offer section, all tiles not_stated
//   L5r-job950-full-listing.png — full listing open, no switch shown
//   L5r-job950-narrow.png      — job 950, narrow 400 px viewport

import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { homedir } from 'node:os';
import { mkdirSync } from 'node:fs';

const BASE = 'http://localhost:9199';
const HERE = dirname(fileURLToPath(import.meta.url));

const playwrightPkg = resolve(homedir(), '.npm/_npx/e41f203b7505f1fb/node_modules/playwright/index.mjs');
const { chromium } = await import(playwrightPkg);

const browser = await chromium.launch({ headless: true });

async function shot(name, fn, viewport = { width: 1280, height: 900 }) {
  const page = await browser.newPage();
  await page.setViewportSize(viewport);
  try {
    await fn(page);
    const path = resolve(HERE, `${name}.png`);
    await page.screenshot({ path, fullPage: false });
    console.log(`${name}.png — saved`);
  } finally {
    await page.close();
  }
}

async function openJob(page, id) {
  await page.goto(`${BASE}/`);
  // Wait for the job list to load.
  await page.waitForSelector('.job-row', { timeout: 30000 });
  // Find and click the matching job row.
  const row = page.locator(`[data-job-id="${id}"]`);
  await row.waitFor({ state: 'visible', timeout: 10000 });
  await row.first().scrollIntoViewIfNeeded();
  await row.first().click();
  // Wait for the detail panel.
  await page.waitForSelector('.fits-you-band', { timeout: 8000 });
  await page.waitForTimeout(400);
}

// 1. Job 950 — Fits you band: Money ✓, $150–250k USD, hint "above your minimum"
await shot('L5r-job950-band', async (page) => {
  await openJob(page, 950);
  // Scroll to top of detail panel
  await page.evaluate(() => document.querySelector('.fits-you-band')?.scrollIntoView({ behavior: 'instant' }));
  await page.waitForTimeout(300);
});

// 2. Job 950 — What they ask and What they offer (scroll down)
await shot('L5r-job950-what-cards', async (page) => {
  await openJob(page, 950);
  await page.evaluate(() => document.querySelector('.what-cards')?.scrollIntoView({ behavior: 'instant' }));
  await page.waitForTimeout(300);
});

// 3. Job 100 — no model extraction; What they ask shows no requirements
await shot('L5r-job100-no-extraction', async (page) => {
  await openJob(page, 100);
});

// 4. Job 500 — no offer section; all tiles not_stated
await shot('L5r-job500-no-offer', async (page) => {
  await openJob(page, 500);
});

// 5. Job 950 — full listing open, no By section / Original switch
await shot('L5r-job950-full-listing', async (page) => {
  await openJob(page, 950);
  await page.evaluate(() => document.querySelector('.listing-panel')?.scrollIntoView({ behavior: 'instant' }));
  await page.waitForTimeout(300);
});

// 6. Job 950 — narrow 400 px (phone width)
await shot('L5r-job950-narrow', async (page) => {
  await openJob(page, 950);
}, { width: 400, height: 800 });

await browser.close();
console.log('Done.');
