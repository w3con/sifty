/**
 * Step 2.1 live verification against the Go server on port 4320.
 * Four observations:
 *   O1 — a job whose form was read shows its questions with per-field answer state
 *   O2 — a job with no resolved form shows the listing link state
 *   O3 — a job not pre-selected shows the selection prompt; Application tab is disabled
 *   O4 — preparation started without touching the selector sends no provider field
 *
 * Requires: Go server running on port 4320 with four fixture jobs (see step 2.1 work-note).
 */
import assert from 'node:assert/strict';
import { chromium } from 'playwright';

const BASE = 'http://127.0.0.1:4320';
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });

try {
  await page.goto(BASE);

  // O3 — not pre-selected: Job 1 (triage/new) Application tab is disabled, shows selection prompt.
  // The initial view shows Triage jobs.
  await page.locator('.job-row').waitFor();
  await page.locator('.job-row').filter({ hasText: 'Software Engineer' }).click();
  assert.equal(
    await page.getByRole('button', { name: 'Application', exact: true }).isDisabled(),
    true,
    'O3: Application tab must be disabled for a triage job'
  );
  await page.getByText('Select this role first to prepare an application.', { exact: true }).waitFor();
  console.log('O3 passed: not-pre-selected shows Application tab disabled and selection prompt');

  // Navigate to Pre-selected view for O1, O2, O4.
  await page.getByRole('button', { name: /^Pre-selected \d/ }).click();
  await page.locator('.job-row').first().waitFor();

  // O2 — no resolved form: Job 2 shows listing link state.
  await page.locator('.job-row').filter({ hasText: 'CTO' }).click();
  await page.getByRole('button', { name: 'Application', exact: true }).click();
  const noFormBanner = page.getByRole('status').filter({ hasText: /apply via the original listing/i });
  await noFormBanner.waitFor({ timeout: 5000 });
  const listingLink = page.getByRole('link', { name: /Apply via the original listing/i });
  assert.ok(await listingLink.count() > 0, 'O2: listing link must be present');
  assert.equal(
    await page.getByText('No prepared answers yet.', { exact: false }).count(),
    0,
    'O2: generic banner must be absent'
  );
  console.log('O2 passed: no-resolved-form shows Apply via the original listing link');

  // O1 — questions shown: Job 3 shows its two questions with "No answer yet" state.
  await page.locator('.job-row').filter({ hasText: 'Product Engineer' }).click();
  await page.getByRole('button', { name: 'Application', exact: true }).click();
  const q1 = page.getByRole('textbox', { name: /Why do you want to join/i });
  await q1.waitFor({ timeout: 5000 });
  const q2 = page.getByRole('textbox', { name: /Describe your engineering/i });
  await q2.waitFor({ timeout: 5000 });
  const noAnswerYet = page.locator('.answer-state').filter({ hasText: 'No answer yet.' });
  assert.ok(await noAnswerYet.count() >= 2, 'O1: both fields must show "No answer yet"');
  assert.equal(
    await page.getByText('No prepared answers yet.', { exact: false }).count(),
    0,
    'O1: generic banner must be absent'
  );
  console.log('O1 passed: form with questions shows each question with "No answer yet" per-field state');

  // O4 — provider not sent when none selected: use Job 4 (fresh preselected, no prior preparation).
  await page.locator('.job-row').filter({ hasText: 'Staff Engineer' }).click();
  await page.getByRole('button', { name: 'Application', exact: true }).click();
  // Wait for the state to load; the primary button should show "Prepare application".
  await page.getByRole('button', { name: 'Prepare application', exact: true }).waitFor({ timeout: 5000 });

  let capturedBody: any = null;
  await page.route(`**/api/jobs/4/application/prepare`, async route => {
    const req = route.request();
    if (req.method() === 'POST') {
      capturedBody = JSON.parse(req.postData() ?? '{}');
    }
    // Abort the request so no real model call happens.
    await route.abort();
  });

  await page.getByRole('button', { name: 'Prepare application', exact: true }).click();
  await page.waitForTimeout(500);
  await page.unroute(`**/api/jobs/4/application/prepare`);

  assert.ok(capturedBody !== null, 'O4: prepare request must have been intercepted');
  assert.ok(
    !('provider' in (capturedBody ?? {})),
    'O4: request must not include provider field when selector is at default'
  );
  console.log('O4 passed: prepare request sends no provider field when selector is at default stage');

  console.log(JSON.stringify({ passed: true, observations: ['O1', 'O2', 'O3', 'O4'] }));
} finally {
  await browser.close();
}
