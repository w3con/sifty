// Playwright check for L.4b: sticky "Failed to fetch" banner fix.
// Pre-condition: server already running on 9199; this script does not start it.
// Usage: node screenshots/shoot-L4b.mjs <server-pid>
// Server PID is used to stop/start the server during the test.
// Screenshots saved to screenshots/ relative to this script.
import { chromium } from '/Users/tknff/.npm/_npx/e41f203b7505f1fb/node_modules/playwright/index.mjs';
import { execSync, spawn } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { existsSync } from 'node:fs';

const DIR = dirname(fileURLToPath(import.meta.url));
const BASE = 'http://127.0.0.1:9199';
const SCREENSHOT = (name) => resolve(DIR, `${name}.png`);

// Server PID from argv
const serverPid = parseInt(process.argv[2] ?? '0');
if (!serverPid) { console.error('Usage: node shoot-L4b.mjs <server-pid>'); process.exit(1); }

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const page = await ctx.newPage();

// Step 1: Load the app normally and confirm no banner
await page.goto(BASE);
await page.waitForSelector('.job-row', { timeout: 10000 });
const bannerBefore = await page.$('.banner.error');
console.log('[1] Banner visible before server stop:', bannerBefore !== null);

// Step 2: Stop the server
console.log(`[2] Stopping server PID ${serverPid}…`);
try { process.kill(serverPid, 'SIGTERM'); } catch(e) { console.error('kill failed:', e.message); process.exit(1); }
// Wait up to 2s for the port to close
let closed = false;
for (let i = 0; i < 20; i++) {
  await new Promise(r => setTimeout(r, 100));
  try { execSync(`lsof -ti tcp:9199 -sTCP:LISTEN`, { stdio: 'pipe' }); } catch { closed = true; break; }
}
if (!closed) console.warn('[2] Warning: server may still be listening');
console.log('[2] Server stopped, waiting for poll to trigger error (up to 5s)…');

// Wait for the banner to appear (poll fires every 2500ms)
await page.waitForSelector('.banner.error', { timeout: 8000 });
const bannerText = await page.$eval('.banner.error', el => el.textContent?.trim() ?? '');
console.log('[2] Banner text after stop:', bannerText);
await page.screenshot({ path: SCREENSHOT('L4b-server-stopped') });
console.log('  saved L4b-server-stopped.png');

// Step 3: Restart the server
console.log('[3] Restarting server…');
// Read the binary and args from env / known path
const serverBin = process.env.SIFTY_BIN;
const dataDir = process.env.SIFTY_DATA;
const distDir = process.env.SIFTY_DIST;
const workersDir = process.env.SIFTY_WORKERS;
if (!serverBin || !dataDir || !distDir || !workersDir) {
  console.error('Set SIFTY_BIN, SIFTY_DATA, SIFTY_DIST, SIFTY_WORKERS env vars for restart');
  await browser.close();
  process.exit(1);
}
const child = spawn(serverBin, [
  'serve', '--data-dir', dataDir, '--dist', distDir, '--workers-dir', workersDir, '--port', '9199'
], { detached: true, stdio: 'ignore' });
child.unref();
console.log(`[3] Server restarted PID ${child.pid}, waiting for banner to clear (up to 10s)…`);

// Wait for banner to disappear (next successful poll clears it)
await page.waitForFunction(() => !document.querySelector('.banner.error'), { timeout: 12000 });
console.log('[3] Banner cleared by itself after server restart');
await page.screenshot({ path: SCREENSHOT('L4b-server-restarted') });
console.log('  saved L4b-server-restarted.png');

// Final: verify banner text was correct
const expectedText = 'Sifty server is not responding. Retrying…';
if (bannerText !== expectedText) {
  console.error(`FAIL: banner text was "${bannerText}", expected "${expectedText}"`);
  process.exit(1);
}
console.log('[OK] Banner text matches expected.');
console.log('[OK] Banner auto-cleared after server restart.');

await browser.close();

// Stop the restarted server
try {
  const pid = execSync('lsof -ti tcp:9199 -sTCP:LISTEN', { stdio: 'pipe' }).toString().trim();
  if (pid) { process.kill(parseInt(pid), 'SIGTERM'); console.log(`[cleanup] Stopped server PID ${pid}`); }
} catch {}

console.log('All L.4b checks done.');
