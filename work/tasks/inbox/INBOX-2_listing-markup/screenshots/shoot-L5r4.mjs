// Playwright screenshots for L.5r4 defect corrections acceptance check.
//
// How to run (from the worktree root):
//   node work/tasks/inbox/INBOX-2_listing-markup/screenshots/shoot-L5r4.mjs
//
// Prerequisites:
//   - Throwaway server already running on port 9199 against a copy of the production DB

import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { homedir } from 'node:os';

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
  await page.waitForSelector('.job-row', { timeout: 30000 });
  const row = page.locator(`[data-job-id="${id}"]`);
  await row.waitFor({ state: 'visible', timeout: 10000 });
  await row.first().scrollIntoViewIfNeeded();
  await row.first().click();
  await page.waitForSelector('.fits-you-band', { timeout: 8000 });
  await page.waitForTimeout(400);
}

// Job 950 at 1440px — salary tile and What they ask/offer
await shot('L5r4-950-1440', async (page) => {
  await openJob(page, 950);
  await page.evaluate(() => document.querySelector('.what-cards')?.scrollIntoView({ behavior: 'instant' }));
  await page.waitForTimeout(300);
}, { width: 1440, height: 900 });

// Job 950 at 1280px
await shot('L5r4-950-1280', async (page) => {
  await openJob(page, 950);
  await page.evaluate(() => document.querySelector('.what-cards')?.scrollIntoView({ behavior: 'instant' }));
  await page.waitForTimeout(300);
}, { width: 1280, height: 900 });

// Job 950 at 400px narrow
await shot('L5r4-950-400', async (page) => {
  await openJob(page, 950);
  await page.evaluate(() => document.querySelector('.what-cards')?.scrollIntoView({ behavior: 'instant' }));
  await page.waitForTimeout(300);
}, { width: 400, height: 900 });

// Job 1011 — no stored facts, salary from model extraction
await shot('L5r4-1011-1280', async (page) => {
  await openJob(page, 1011);
  await page.evaluate(() => document.querySelector('.what-cards')?.scrollIntoView({ behavior: 'instant' }));
  await page.waitForTimeout(300);
}, { width: 1280, height: 900 });

// Job 100 — no model extraction
await shot('L5r4-100-1280', async (page) => {
  await openJob(page, 100);
  await page.evaluate(() => document.querySelector('.what-cards')?.scrollIntoView({ behavior: 'instant' }));
  await page.waitForTimeout(300);
}, { width: 1280, height: 900 });

await browser.close();
console.log('Done.');
