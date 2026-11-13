import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { homedir } from 'node:os';

const BASE = 'http://localhost:9199';
const HERE = dirname(fileURLToPath(import.meta.url));
const playwrightPkg = resolve(homedir(), '.npm/_npx/e41f203b7505f1fb/node_modules/playwright/index.mjs');
const { chromium } = await import(playwrightPkg);
const browser = await chromium.launch({ headless: true });

async function shot(name, fn, viewport = { width: 1440, height: 900 }) {
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

// Find a job with an open (undecided) non-skill ask row.
// Non-skill confirm buttons have title "Confirm you have this".
async function findJobWithExperienceRow(page) {
  await page.goto(`${BASE}/`);
  await page.waitForSelector('.job-row', { timeout: 30000 });
  const rows = page.locator('.job-row');
  const count = await rows.count();
  for (let i = 0; i < Math.min(count, 40); i++) {
    const row = rows.nth(i);
    const id = await row.getAttribute('data-job-id');
    if (!id) continue;
    await row.scrollIntoViewIfNeeded();
    await row.click();
    try { await page.waitForSelector('.fits-you-band', { timeout: 4000 }); } catch { continue; }
    await page.waitForTimeout(200);
    const confirmBtn = page.locator('.ask-actions button[title="Confirm you have this"]').first();
    const visible = await confirmBtn.isVisible().catch(() => false);
    if (visible) {
      console.log(`Found experience row in job ${id}`);
      return { id, page };
    }
  }
  return { id: null };
}

// Before: show the What they ask panel with the ✓ button on a non-skill row
await shot('L5r6-before-confirm', async (page) => {
  const { id } = await findJobWithExperienceRow(page);
  if (!id) throw new Error('No job with open experience require row found');
  // Scroll the first confirm-have button into view
  await page.evaluate(() => {
    const btn = document.querySelector('.ask-actions button[title="Confirm you have this"]');
    if (btn) btn.closest('li')?.scrollIntoView({ behavior: 'instant', block: 'center' });
  });
  await page.waitForTimeout(300);
});

// After: click ✓ and screenshot the notice + updated panel
// (do not wait for recheck — it polls for model completion which is disabled in scratchpad)
await shot('L5r6-after-confirm', async (page) => {
  const { id } = await findJobWithExperienceRow(page);
  if (!id) throw new Error('No job with open experience require row found');

  // Extract the requirement text before clicking
  const reqText = await page.evaluate(() => {
    const btn = document.querySelector('.ask-actions button[title="Confirm you have this"]');
    const li = btn?.closest('li');
    return li?.querySelector('.ask-text')?.textContent ?? '';
  });
  console.log(`Confirming: "${reqText}"`);

  // Scroll the row into view
  await page.evaluate(() => {
    const btn = document.querySelector('.ask-actions button[title="Confirm you have this"]');
    if (btn) btn.closest('li')?.scrollIntoView({ behavior: 'instant', block: 'center' });
  });
  await page.waitForTimeout(200);

  // Click ✓ — this fires the API call and notice
  await page.locator('.ask-actions button[title="Confirm you have this"]').first().click();

  // Wait just for the notice toast to appear (the API response), not the full recheck
  await page.waitForTimeout(1500);

  // Take the screenshot showing the notice banner
  // (the row itself won't update until recheck completes, but the notice confirms the action)
});

await browser.close();
console.log('Done.');
