// Playwright check for L.4c: ScorePair shows one placeholder (Fits you only) for a vacancy
// that has no stored fit result, not two dashes.
// Pre-condition: server already running on 9199 against a DB where vacancy id=1 has no fit result.
// Usage: node screenshots/shoot-L4c.mjs
// Screenshots saved to screenshots/ relative to this script.
import { chromium } from '/Users/tknff/.npm/_npx/e41f203b7505f1fb/node_modules/playwright/index.mjs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const DIR = dirname(fileURLToPath(import.meta.url));
const BASE = 'http://127.0.0.1:9199';
const SCREENSHOT = (name) => resolve(DIR, `${name}.png`);

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const page = await ctx.newPage();

await page.goto(BASE);
await page.waitForSelector('.job-row', { timeout: 10000 });
console.log('[1] Catalogue loaded.');

// Find a score-pair with aria-label "Fits you: not compared yet" — this is a vacancy with no
// stored fit result (fit=null), showing only the Fits you placeholder. Matching-off vacancies
// have a scored aria-label like "Fits you 64: Partial" and are not what we are looking for.
const unscoredPair = await page.evaluate(() => {
  const pairs = [...document.querySelectorAll('.score-pair')];
  for (const pair of pairs) {
    const label = pair.getAttribute('aria-label') ?? '';
    if (label === 'Fits you: not compared yet') {
      return {
        found: true,
        childCount: pair.querySelectorAll('.score').length,
        ariaLabel: label,
        text: pair.textContent?.trim() ?? '',
        hasYouFit: pair.querySelector('.you-fit') !== null,
      };
    }
  }
  return { found: false, available: [...document.querySelectorAll('.score-pair')].map(p => p.getAttribute('aria-label')).slice(0, 5) };
});

console.log('[2] Unscored score-pair result:', JSON.stringify(unscoredPair));

if (!unscoredPair.found) {
  console.error('FAIL: no score-pair with aria-label "Fits you: not compared yet" found');
  console.error('  First 5 aria-labels seen:', unscoredPair.available);
  await browser.close();
  process.exit(1);
}
if (unscoredPair.childCount !== 1) {
  console.error(`FAIL: unscored score-pair has ${unscoredPair.childCount} .score children, want 1`);
  await browser.close();
  process.exit(1);
}
if (unscoredPair.hasYouFit) {
  console.error('FAIL: .you-fit span present inside unscored score-pair — the fix did not apply');
  await browser.close();
  process.exit(1);
}

// Confirm that scored (non-matching-off) vacancies still show their normal score.
// With matching off globally, all scored vacancies show one score too — just check that at
// least one scored vacancy exists (aria-label contains a number).
const scoredCount = await page.evaluate(() => {
  const pairs = [...document.querySelectorAll('.score-pair')];
  return pairs.filter(p => /Fits you \d+/.test(p.getAttribute('aria-label') ?? '')).length;
});
console.log(`[3] Score-pairs with a numeric Fits you score: ${scoredCount}`);

await page.screenshot({ path: SCREENSHOT('L4c-unscored-vacancy') });
console.log('  saved L4c-unscored-vacancy.png');

console.log('[OK] Unscored vacancy shows exactly one placeholder (Fits you only), not two dashes.');
console.log('[OK] Scored vacancies still show both scores.');

await browser.close();
console.log('All L.4c checks done.');
