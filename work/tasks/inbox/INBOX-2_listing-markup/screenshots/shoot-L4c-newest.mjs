import { chromium } from '/Users/tknff/.npm/_npx/e41f203b7505f1fb/node_modules/playwright/index.mjs';
import { execFileSync } from 'node:child_process';
const DB='/private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/2f034d6c-1e62-4017-977f-b4e8510b04cf/scratchpad/l4c/data/store.sqlite';
const OUT='/private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/2f034d6c-1e62-4017-977f-b4e8510b04cf/scratchpad/l4c/';
const b = await chromium.launch(); const p = await b.newPage({viewport:{width:1440,height:900}});
await p.goto('http://localhost:9199/'); await p.waitForSelector('.job-row',{timeout:30000});
await p.locator('button', {hasText:'Newest'}).first().click(); await p.waitForTimeout(800);
const ids = await p.$$eval('.job-row', els => els.slice(0,4).map(e=>e.getAttribute('data-job-id')));
console.log('first four in Newest:', ids.join(','));
execFileSync('sqlite3',[DB,`delete from match_fit_results where job_id in (${ids.join(',')})`]);
await p.waitForTimeout(6000); // two polls
const rows = await p.$$eval('.job-row', els => els.slice(0,6).map(e=>({id:e.getAttribute('data-job-id'), cells:[...e.querySelectorAll('.score-pair .score')].map(s=>s.textContent.trim()), label:e.querySelector('.score-pair')?.getAttribute('aria-label')})));
console.log(JSON.stringify(rows));
await p.screenshot({path:OUT+'newest-after-fix.png'});
await b.close();
