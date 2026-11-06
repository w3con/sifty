// automation.test.mjs — step-by-step checks for the shared browser-automation discipline.
// Each test drives a local fixture server so no real external site is contacted.
// Run with: node --test automation.test.mjs
// Run one step: node --test --test-name-pattern '1\.1' automation.test.mjs

import { test } from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';
import { chromium } from 'playwright';
import {
  BROWSER_UA,
  launchOptions,
  contextOptions,
  addFingerprintScripts,
  HostPacer,
  Semaphore,
  CooldownRegistry,
  createDiscipline,
} from './automation.mjs';
import { inspectFormWith } from './index.mjs';

// listen starts the server on a random loopback port and resolves with that port number.
function listen(server) {
  return new Promise((resolve) => server.listen(0, '127.0.0.1', () => resolve(server.address().port)));
}

// stop closes the server gracefully.
function stop(server) {
  return new Promise((resolve) => server.close(resolve));
}

// --- Step 1.1: realistic fingerprint ---

test('1.1 realistic fingerprint: UA and navigator.webdriver', async () => {
  let receivedUA = '';
  const server = http.createServer((req, res) => {
    receivedUA = req.headers['user-agent'] ?? '';
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<html><body>UA echo</body></html>');
  });
  const port = await listen(server);

  const browser = await chromium.launch(launchOptions());
  try {
    const context = await browser.newContext(contextOptions());
    await addFingerprintScripts(context);
    const page = await context.newPage();
    await page.goto(`http://127.0.0.1:${port}/`);

    // The fixture server must have received the configured User-Agent.
    assert.equal(receivedUA, BROWSER_UA, 'server received wrong User-Agent');
    assert.ok(!receivedUA.includes('HeadlessChrome'), 'UA must not expose HeadlessChrome');

    // navigator.webdriver must read false in the page (init script suppresses it).
    const webdriver = await page.evaluate(() => navigator.webdriver);
    assert.equal(webdriver, false, 'navigator.webdriver must be false');

    await context.close();
  } finally {
    await browser.close();
    await stop(server);
  }
});

// --- Step 1.2: host pacing ---

test('1.2 host pacing: sequential navigations are spaced by the configured interval', async () => {
  const intervalMs = 300;
  const reqTimestamps = [];
  const server = http.createServer((req, res) => {
    reqTimestamps.push(Date.now());
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<html><body></body></html>');
  });
  const port = await listen(server);

  // Use HostPacer directly with lightweight fetch requests so timer precision is not masked
  // by browser-launch variance. inspectFormWith calls discipline.pacing.wait() identically.
  const discipline = createDiscipline({ intervalMs, jitterMs: 0 });
  for (let i = 0; i < 2; i++) {
    if (discipline.pacing) await discipline.pacing.wait('127.0.0.1');
    await fetch(`http://127.0.0.1:${port}/`);
  }

  await stop(server);
  assert.ok(reqTimestamps.length >= 2, 'fixture must receive at least 2 requests');
  const gap = reqTimestamps[1] - reqTimestamps[0];
  assert.ok(
    gap >= intervalMs - 20,
    `requests must be at least ${intervalMs}ms apart (got ${gap}ms)`,
  );
});

// --- Step 1.3: bounded concurrency ---

test('1.3 concurrency: at most the configured number of navigations run at once', async () => {
  const concurrencyLimit = 2;
  let concurrent = 0;
  let peak = 0;

  // Fixture holds each response for 150ms so overlapping requests pile up visibly.
  const server = http.createServer((req, res) => {
    concurrent++;
    peak = Math.max(peak, concurrent);
    setTimeout(() => {
      concurrent--;
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end('<html><body></body></html>');
    }, 150);
  });
  const port = await listen(server);

  // Use the Semaphore directly with lightweight fetches: inspectFormWith calls
  // discipline.semaphore.acquire/release around each browser navigation identically.
  const discipline = createDiscipline({ intervalMs: 0, jitterMs: 0, concurrency: concurrencyLimit });
  const navigate = async () => {
    await discipline.semaphore.acquire();
    try {
      await fetch(`http://127.0.0.1:${port}/`);
    } finally {
      discipline.semaphore.release();
    }
  };

  // Drive 4 concurrent navigations; semaphore allows only 2 at a time.
  await Promise.all([navigate(), navigate(), navigate(), navigate()]);
  await stop(server);

  assert.ok(peak <= concurrencyLimit, `peak concurrency ${peak} exceeds limit ${concurrencyLimit}`);
  assert.ok(peak > 0, 'at least one navigation must have run');
});

// --- Step 2.1: block detection ---

test('2.1 block detection: 429 response ends in backed-off state with no retry', async () => {
  let requestCount = 0;
  const server = http.createServer((req, res) => {
    requestCount++;
    res.writeHead(429, { 'Content-Type': 'text/html' });
    res.end('<html><body>Too Many Requests</body></html>');
  });
  const port = await listen(server);

  // A very short cooldown so the test does not take long.
  const discipline = createDiscipline({ intervalMs: 0, jitterMs: 0, concurrency: 1, cooldownMs: 50 });
  const result = await inspectFormWith({ url: `http://127.0.0.1:${port}/` }, discipline);

  await stop(server);

  assert.equal(result.status, 'backed-off', `expected status backed-off, got ${result.status}`);
  assert.ok(result.gaps.length > 0, 'result must include a descriptive gap message');
  assert.equal(requestCount, 1, `only one request must be made (no retry); got ${requestCount}`);
});

// --- Step 2.2: cooldown deferral ---

test('2.2 cooldown: second inspect to blocked host is deferred; proceeds after cooldown', async () => {
  let requestCount = 0;
  const server = http.createServer((req, res) => {
    requestCount++;
    // First request returns 429 (block); subsequent ones return 200.
    const status = requestCount === 1 ? 429 : 200;
    res.writeHead(status, { 'Content-Type': 'text/html' });
    res.end('<html><body>ok</body></html>');
  });
  const port = await listen(server);
  const url = `http://127.0.0.1:${port}/`;

  // 60ms cooldown so the test can wait it out.
  const cooldownMs = 60;
  const discipline = createDiscipline({ intervalMs: 0, jitterMs: 0, concurrency: 1, cooldownMs });

  // First inspect: hits 429, returns backed-off, registers cooldown.
  const r1 = await inspectFormWith({ url }, discipline);
  assert.equal(r1.status, 'backed-off', 'first inspect must be backed-off');
  const countAfterFirst = requestCount;

  // Second inspect within cooldown: must be deferred immediately (no browser navigation).
  const r2 = await inspectFormWith({ url }, discipline);
  assert.equal(r2.status, 'backed-off', 'second inspect within cooldown must be backed-off');
  assert.equal(requestCount, countAfterFirst, 'second inspect must not make a new request during cooldown');

  // Wait for cooldown to lapse, then inspect again: should proceed normally.
  await new Promise((resolve) => setTimeout(resolve, cooldownMs + 20));
  const r3 = await inspectFormWith({ url }, discipline);
  // After cooldown the server returns 200; the page has no form fields but is not blocked.
  assert.ok(r3.status !== 'backed-off', `third inspect after cooldown must not be backed-off (got ${r3.status})`);

  await stop(server);
});

