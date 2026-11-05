// Explicit operational read-only audit/consistent backup; never imported by tests.
import assert from 'node:assert/strict';
import {DatabaseSync} from 'node:sqlite';
import {mkdirSync,readFileSync,writeFileSync} from 'node:fs';
import {chromium} from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import {hash,localPath,rootPath,now} from '../src/core.js';
const base='http://127.0.0.1:4317',dir=localPath('verification');mkdirSync(dir,{recursive:true});
const protectedPaths=['config/search.yaml','config/providers.yaml','config/network.yaml','config/candidate.yaml','config/runtime.yaml','cv.pdf','profile.pdf','kb/faq.md','ai/HANDOVER.md','kb/Strategy/positioning.md','kb/Market/where-the-work-comes-from.md','.gitignore'];
const protectedHashes=()=>Object.fromEntries(protectedPaths.map(p=>[p,hash(readFileSync(rootPath(p)))]));
const db=new DatabaseSync(localPath('jobs.sqlite'),{readOnly:true});
try{
  if(process.argv[2]==='backup'){
    const backup=localPath(`backups/sift-before-${Date.now()}.sqlite`);db.prepare('VACUUM INTO ?').run(backup);
    const report={created:now(),backup,hashes:protectedHashes(),queue:db.prepare('SELECT status,count(*) n FROM assessment_queue GROUP BY status').all()};writeFileSync(`${dir}/sift-before.json`,JSON.stringify(report,null,2));console.log(JSON.stringify(report,null,2));
  }else if(process.argv[2]==='verify'){
    const before=JSON.parse(readFileSync(`${dir}/sift-before.json`,'utf8')),old=new DatabaseSync(before.backup,{readOnly:true});
    const jobs=old.prepare('SELECT * FROM jobs ORDER BY id').all() as any[];let assessmentUpdates=0;
    for(const job of jobs){const current=db.prepare('SELECT * FROM jobs WHERE id=?').get(job.id) as any;assert.ok(current);for(const [key,value] of Object.entries(job)){if(['assessment','assessment_error','seen_at','applied_at'].includes(key))continue;assert.deepEqual(current[key],value,`Changed job ${job.id} ${key}`);}if(job.assessment!==current.assessment)assessmentUpdates++;}
    const tables:Record<string,number>={};
    for(const table of ['sightings','snapshots','assessments','job_aliases','collection_runs','source_runs','collection_events','feedback']){
      const current=new Set(db.prepare(`SELECT * FROM ${table}`).all().map(r=>JSON.stringify(r)));const rows=old.prepare(`SELECT * FROM ${table}`).all();for(const row of rows)assert.ok(current.has(JSON.stringify(row)),`Missing/changed ${table} record`);tables[table]=rows.length;
    }
    const queueIds=new Set(db.prepare('SELECT job_id FROM assessment_queue').all().map(r=>r.job_id));for(const row of old.prepare('SELECT job_id FROM assessment_queue').all())assert.ok(queueIds.has(row.job_id),'Lost queue job');
    assert.deepEqual(protectedHashes(),before.hashes);assert.equal(Object.values(db.prepare('PRAGMA integrity_check').get()!)[0],'ok');old.close();
    const browser=await chromium.launch({headless:true}),context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage(),errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));
    const writes:string[]=[];await context.route('**/api/**',r=>{const request=r.request();if(request.method()!=='GET'&&!/\/seen$/.test(request.url())){writes.push(`${request.method()} ${request.url()}`);return r.abort();}return r.continue();});
    try{
      const response=await page.goto(base);assert.equal(response?.status(),200);await page.locator('.job-row').first().waitFor();await page.getByRole('heading',{name:/^Evidence of fit/}).waitFor();await page.evaluate(()=>document.fonts.ready);
      const apiJobs=await (await page.request.get(`${base}/api/jobs`)).json();assert.equal(apiJobs.length,jobs.length);assert.ok(apiJobs.every((j:any)=>j.stage&&typeof j.seen==='boolean'));
      const desktopAxe=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();assert.deepEqual(desktopAxe.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.html)})),[]);
      await page.screenshot({path:`${dir}/sift-live-desktop.png`});const selected=await page.locator('.job-row[aria-pressed=true]').getAttribute('data-job-id');
      await page.getByRole('button',{name:'Job description',exact:true}).click();assert.ok((await page.locator('.description').innerText()).length);await page.screenshot({path:`${dir}/sift-live-description.png`});
      await page.getByRole('button',{name:'Settings',exact:true}).click();await page.locator('#yaml').waitFor();await page.locator('.provider-card').first().waitFor();await page.screenshot({path:`${dir}/sift-live-settings.png`});
      await page.keyboard.press('Escape');assert.equal(await page.locator('.job-row[aria-pressed=true]').getAttribute('data-job-id'),selected);
      await page.setViewportSize({width:390,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);const mobileAxe=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa']).analyze();assert.deepEqual(mobileAxe.violations.map(v=>v.id),[]);await page.screenshot({path:`${dir}/sift-live-mobile.png`});
      const queue=await (await page.request.get(`${base}/api/queue`)).json();assert.deepEqual(writes,[]);assert.deepEqual(errors,[]);assert.deepEqual(protectedHashes(),before.hashes);
      const index=await (await page.request.get(base)).text();assert.equal(index,readFileSync(rootPath('app/dist/index.html'),'utf8'));assert.equal((await page.request.get(`${base}/api/unknown-sift-endpoint`)).status(),404);
      const report={checked:now(),backup:before.backup,jobsPreserved:jobs.length,workflowStatusesUnchanged:true,historyRowsPreserved:tables,queueRowsPreserved:queueIds.size,backgroundAssessmentUpdates:assessmentUpdates,integrity:'ok',protectedHashesUnchanged:true,desktopAxe:0,mobileAxe:0,horizontalOverflow:false,browserErrors:errors,operationalWorkflowWrites:writes,queue:queue.counts,runtime:queue.runtime?JSON.parse(queue.runtime.data):null,screenshots:dir};writeFileSync(`${dir}/sift-live.json`,JSON.stringify(report,null,2));console.log(JSON.stringify(report,null,2));
    }finally{await browser.close();}
  }else throw new Error('Use backup or verify explicitly');
}finally{db.close();}
