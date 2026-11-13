// Defect 5 and Defect 6 delivery screenshots.
// Server must be running on port 9198 against a production data copy before this script is run.
// Usage: node shoot-defect5.mjs
// Output: six PNG files next to this script.
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { homedir } from 'node:os';

const BASE = 'http://localhost:9198';
const HERE = dirname(fileURLToPath(import.meta.url));
const playwrightPkg = resolve(homedir(), '.npm/_npx/e41f203b7505f1fb/node_modules/playwright/index.mjs');
const { chromium } = await import(playwrightPkg);
const browser = await chromium.launch({ headless: true });

async function shot(name, fn) {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  try {
    await fn(page);
    const dest = resolve(HERE, `${name}.png`);
    await page.screenshot({ path: dest, fullPage: false });
    console.log(`${name}.png saved`);
  } catch (e) {
    console.error(`${name} FAILED: ${e.message.slice(0, 200)}`);
  } finally {
    await page.close();
  }
}

async function openJob(page, id) {
  await page.goto(`${BASE}/?sort=newest`);
  await page.waitForSelector('.job-row', { timeout: 30000 });
  let row = page.locator(`[data-job-id="${id}"]`);
  if (await row.count() === 0) {
    for (let i = 0; i < 30; i++) {
      await page.evaluate(() => {
        const list = document.querySelector('[class*="catalogue"]') || document.querySelector('.job-list') || document.body;
        list.scrollTop += 300;
      });
      await page.waitForTimeout(150);
      if (await row.count() > 0) break;
    }
  }
  if (await row.count() === 0) throw new Error(`Job ${id} not found in list`);
  await row.first().scrollIntoViewIfNeeded();
  await row.first().click();
  await page.waitForSelector('.fits-you-band', { timeout: 15000 });
  await page.waitForTimeout(600);
}

async function findAskRow(page, textFragment) {
  const rows = page.locator('li.ask-row');
  const n = await rows.count();
  for (let i = 0; i < n; i++) {
    const t = await rows.nth(i).textContent();
    if (t && t.includes(textFragment)) return rows.nth(i);
  }
  return null;
}

// ── Defect 6: "Elixir or Erlang" (job 327) ─────────────────────────────────
await shot('D5-elixir-erlang-before', async (page) => {
  await openJob(page, 327);
  const row = await findAskRow(page, 'Elixir or Erlang');
  if (!row) throw new Error('Elixir or Erlang row not found');
  await row.scrollIntoViewIfNeeded();
  await page.waitForTimeout(200);
});

await shot('D5-elixir-erlang-alts', async (page) => {
  await openJob(page, 327);
  const row = await findAskRow(page, 'Elixir or Erlang');
  if (!row) throw new Error('Elixir or Erlang row not found');
  await row.scrollIntoViewIfNeeded();
  const addBtn = row.locator('button.add').first();
  await addBtn.click();
  await page.waitForTimeout(1500);
  await row.scrollIntoViewIfNeeded();
});

await shot('D5-elixir-erlang-match', async (page) => {
  await openJob(page, 327);
  const row = await findAskRow(page, 'Elixir or Erlang');
  if (!row) throw new Error('Elixir or Erlang row not found');
  await row.scrollIntoViewIfNeeded();
  const addBtn = row.locator('button.add').first();
  await addBtn.click();
  await page.waitForTimeout(1000);
  const erlangBtn = page.locator('button.add', { hasText: 'Erlang' }).first();
  if (await erlangBtn.count() === 0) throw new Error('Erlang button not found');
  await erlangBtn.click();
  await page.waitForFunction(() => {
    for (const r of document.querySelectorAll('li.ask-row')) {
      if (r.textContent?.includes('Elixir or Erlang')) return r.classList.contains('match');
    }
    return false;
  }, { timeout: 15000 });
  await row.scrollIntoViewIfNeeded();
});

// ── Defect 5: "AI coding tools" (job 898) → adds as skill ──────────────────
await shot('D5-ai-coding-tools-before', async (page) => {
  await openJob(page, 898);
  const row = await findAskRow(page, 'AI coding tools');
  if (!row) throw new Error('AI coding tools row not found');
  await row.scrollIntoViewIfNeeded();
});

await shot('D5-ai-coding-tools-after', async (page) => {
  await openJob(page, 898);
  const row = await findAskRow(page, 'AI coding tools');
  if (!row) throw new Error('AI coding tools row not found');
  await row.scrollIntoViewIfNeeded();
  const addBtn = row.locator('button.add').first();
  await addBtn.click();
  await page.waitForFunction(() => {
    for (const r of document.querySelectorAll('li.ask-row')) {
      if (r.textContent?.includes('AI coding tools')) return r.classList.contains('match');
    }
    return false;
  }, { timeout: 10000 });
  await row.scrollIntoViewIfNeeded();
});

// ── Defect 5: "Coding, debugging, reviewing" (job 1050) → confirmed fact ───
await shot('D5-coding-confirmed-after', async (page) => {
  await openJob(page, 1050);
  const row = await findAskRow(page, 'Coding, debugging, reviewing');
  if (!row) throw new Error('Coding row not found');
  await row.scrollIntoViewIfNeeded();
  const addBtn = row.locator('button.add').first();
  await addBtn.click();
  await page.waitForFunction(() => {
    for (const r of document.querySelectorAll('li.ask-row')) {
      if (r.textContent?.includes('Coding, debugging, reviewing')) return r.classList.contains('match');
    }
    return false;
  }, { timeout: 10000 });
  await row.scrollIntoViewIfNeeded();
});

await browser.close();
console.log('Done.');
