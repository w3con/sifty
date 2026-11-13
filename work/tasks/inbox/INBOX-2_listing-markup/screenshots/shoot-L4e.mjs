// L.4e verification: poll without refetching the catalogue.
// Tracks all /api/ requests for 30s and verifies /api/jobs is not fetched repeatedly.
// Also checks that /api/jobs is fetched on initial load and that /api/queue is polled regularly.
import { chromium } from 'playwright';

const BASE = 'http://127.0.0.1:9199';

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
const page = await ctx.newPage();

// Intercept all API requests
const requests = [];
page.on('request', req => {
  const url = req.url();
  if (url.includes('/api/')) {
    requests.push({ ts: Date.now(), url: url.replace(/^.*\/api\//, '/api/'), method: req.method() });
  }
});

await page.goto(`${BASE}/`);
// Wait 30 seconds while observing network traffic
await page.waitForTimeout(30000);

// Analysis
const queueReqs = requests.filter(r => r.url === '/api/queue');
const jobsReqs = requests.filter(r => r.url === '/api/jobs');
const healthReqs = requests.filter(r => r.url === '/api/health');
const analysiReqs = requests.filter(r => r.url === '/api/quick-match/analysis');

console.log('=== Network activity over 30s ===');
console.log(`/api/queue requests:                ${queueReqs.length} (expect ~12 for 2.5s interval)`);
console.log(`/api/jobs requests:                 ${jobsReqs.length} (expect 1 on load, 0 during poll)`);
console.log(`/api/health requests:               ${healthReqs.length} (expect ~12)`);
console.log(`/api/quick-match/analysis requests: ${analysiReqs.length} (expect ~12)`);

const otherReqs = requests.filter(r => !['/api/queue','/api/jobs','/api/health','/api/quick-match/analysis'].includes(r.url));
console.log(`Other /api/ requests:               ${otherReqs.length}`);
if (otherReqs.length) {
  const counts = {};
  otherReqs.forEach(r => { counts[r.url] = (counts[r.url] || 0) + 1; });
  Object.entries(counts).forEach(([url, n]) => console.log(`  ${url}: ${n}`));
}

console.log('\n=== Verification ===');
const jobsFetchOk = jobsReqs.length === 1;
const pollRunning = queueReqs.length >= 10;
const healthPolled = healthReqs.length >= 10;

console.log(`No repeated jobs list (expect 1): ${jobsFetchOk ? 'PASS' : `FAIL (got ${jobsReqs.length})`}`);
console.log(`Queue polled regularly:            ${pollRunning ? 'PASS' : `FAIL (got ${queueReqs.length})`}`);
console.log(`Health polled regularly:           ${healthPolled ? 'PASS' : `FAIL (got ${healthReqs.length})`}`);

// Check the initial /api/jobs request time vs first poll time
if (jobsReqs.length >= 1 && queueReqs.length >= 1) {
  const jobsTime = jobsReqs[0].ts;
  const firstQueueTime = queueReqs[0].ts;
  console.log(`Jobs fetched ${jobsTime - firstQueueTime}ms after first queue poll`);
}

await browser.close();
console.log('\nDone.');
process.exit(jobsFetchOk && pollRunning && healthPolled ? 0 : 1);
