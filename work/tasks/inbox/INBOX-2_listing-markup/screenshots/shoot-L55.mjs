import { chromium } from 'playwright';
const BASE = 'http://127.0.0.1:9199';
const CARD = process.env.CARD_DIR;

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
const page = await ctx.newPage();

async function shot(name) {
  await page.screenshot({ path: `${CARD}/screenshots/${name}.png`, fullPage: false });
  console.log(`Screenshot: ${name}.png`);
}

// Load job 8 (has Requirements class sections)
await page.goto(`${BASE}/#${8}`);
await page.waitForTimeout(1500);
await shot('L55-job8-original');

// Click "By section" button
const bySectionBtn = page.locator('.text-mode button', { hasText: 'By section' });
const isDisabled = await bySectionBtn.getAttribute('disabled');
console.log(`By section button disabled: ${isDisabled !== null}`);
if (isDisabled === null) {
  await bySectionBtn.click();
  await page.waitForTimeout(500);
  await shot('L55-job8-by-section');

  // Check that section-view is present
  const sectionView = await page.locator('.section-view').count();
  console.log(`Section view blocks: ${await page.locator('.section-block').count()}`);
  
  // Check first section is open (details[open])
  const firstOpen = await page.locator('.section-block[open]').count();
  console.log(`First section open: ${firstOpen > 0}`);

  // Click on a collapsed section to expand it
  const closedSection = page.locator('.section-block:not([open]) .section-label').first();
  if (await closedSection.count() > 0) {
    await closedSection.click();
    await page.waitForTimeout(300);
    await shot('L55-job8-section-expanded');
    console.log('Expanded a collapsed section');
  }

  // Check no raw tags visible in the section content
  const rawTags = await page.locator('.section-content').evaluate(el => {
    return el && el.textContent && (
      el.textContent.includes('<p>') || el.textContent.includes('<li>') || el.textContent.includes('<strong>')
    );
  });
  console.log(`Raw tags visible in section content: ${rawTags}`);
}

// Switch back to Original
const origBtn = page.locator('.text-mode button', { hasText: 'Original' });
await origBtn.click();
await page.waitForTimeout(300);
await shot('L55-job8-back-to-original');

// Job 222 — no description_html, By section should be disabled
await page.goto(`${BASE}/#${222}`);
await page.waitForTimeout(1200);
const btn222 = page.locator('.text-mode button', { hasText: 'By section' });
const disabled222 = await btn222.getAttribute('disabled');
console.log(`Job 222 (no description) By section disabled: ${disabled222 !== null}`);
await shot('L55-job222-disabled-button');

await browser.close();
console.log('Done.');
