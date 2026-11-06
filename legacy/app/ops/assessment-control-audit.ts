// Explicit operational verification. Never toggle the owner's assessment preference here.
import assert from 'node:assert/strict';
import {DatabaseSync} from 'node:sqlite';
import {mkdirSync,readFileSync,writeFileSync} from 'node:fs';
import {chromium} from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import {localPath,rootPath,hash,now} from '../src/core.js';
const dir=localPath('verification'),base='http://127.0.0.1:4317';mkdirSync(dir,{recursive:true});
const db=new DatabaseSync(localPath('jobs.sqlite'),{readOnly:true});
const preference=()=>db.prepare("SELECT data FROM queue_state WHERE key='assessment_enabled'").get()?.data??null;
const protectedFiles=['search','providers','network','candidate','runtime'].map(n=>`config/${n}.yaml`);
const hashes=()=>Object.fromEntries(protectedFiles.map(p=>[p,hash(readFileSync(rootPath(p)))]));
try{
 if(process.argv[2]==='before'){
  const snapshot={checked:now(),preference:preference(),jobs:db.prepare('SELECT id,status FROM jobs ORDER BY id').all(),queueIds:db.prepare('SELECT job_id FROM assessment_queue ORDER BY job_id').all(),configHashes:hashes()};writeFileSync(`${dir}/assessment-control-before.json`,JSON.stringify(snapshot,null,2));console.log(JSON.stringify({checked:snapshot.checked,preference:snapshot.preference,jobs:snapshot.jobs.length,queue:snapshot.queueIds.length}));
 }else if(process.argv[2]==='verify'){
  const before=JSON.parse(readFileSync(`${dir}/assessment-control-before.json`,'utf8'));assert.equal(preference(),before.preference);assert.deepEqual(db.prepare('SELECT id,status FROM jobs ORDER BY id').all().map(r=>({...r})),before.jobs);assert.deepEqual(db.prepare('SELECT job_id FROM assessment_queue ORDER BY job_id').all().map(r=>({...r})),before.queueIds);assert.deepEqual(hashes(),before.configHashes);
  const browser=await chromium.launch({headless:true}),context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage(),writes:string[]=[],errors:string[]=[],checks:any[]=[];
  page.on('pageerror',e=>errors.push(e.message));await context.route('**/api/**',r=>{if(r.request().method()!=='GET'&&!r.request().url().endsWith('/seen')){writes.push(r.request().url());return r.abort();}return r.continue();});
  try{
   await page.goto(base);const toggle=page.getByRole('switch',{name:'Assess vacancies',exact:true});await toggle.waitFor();const queue=await (await page.request.get(`${base}/api/queue`)).json();assert.equal(typeof queue.enabled,'boolean');assert.equal(await toggle.getAttribute('aria-checked'),String(queue.enabled));assert.equal(queue.enabled,before.preference===null?true:JSON.parse(before.preference));
   for(const width of [1440,390,320]){
    await page.setViewportSize({width,height:width===1440?1000:844});await page.evaluate(()=>document.fonts.ready);const bounds=await toggle.boundingBox();assert.ok(bounds&&bounds.x>=0&&bounds.x+bounds.width<=width&&bounds.y+bounds.height<=(width===1440?1000:844));assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
    const result=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();assert.deepEqual(result.violations.map(v=>v.id),[]);await page.screenshot({path:`${dir}/assessment-control-live-${width}.png`});checks.push({width,axeViolations:0,switchVisible:true,pageOverflow:false});
   }
   await page.reload();await toggle.waitFor();assert.equal(await toggle.getAttribute('aria-checked'),String(queue.enabled));assert.equal(preference(),before.preference);assert.deepEqual(writes,[]);assert.deepEqual(errors,[]);
   const current=await (await page.request.get(`${base}/api/queue`)).json(),report={checked:now(),enabled:current.enabled,preferenceRawUnchanged:preference(),jobsPreserved:before.jobs.length,queueIdsPreserved:before.queueIds.length,configHashesUnchanged:true,checks,mutationRequests:writes,browserErrors:errors,queue:current.counts,runtime:current.runtime?JSON.parse(current.runtime.data):null,screenshots:dir};writeFileSync(`${dir}/assessment-control-live.json`,JSON.stringify(report,null,2));console.log(JSON.stringify(report,null,2));
  }finally{await browser.close();}
 }else throw new Error('Use before or verify');
}finally{db.close();}
