import './isolate.js';
import assert from 'node:assert/strict';
import {mkdirSync,writeFileSync} from 'node:fs';
import {chromium} from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import {Store} from '../src/store.js';
import {serve} from '../src/server.js';
import {localPath,now} from '../src/core.js';
import {assessmentEnabled,queueStatus} from '../src/queue.js';

const port=4320,base=`http://127.0.0.1:${port}`,path=localPath('jobs.sqlite'),dir=localPath('verification');mkdirSync(dir,{recursive:true});
let store=new Store(path),probes=0;
store.upsert({title:'Engineer (isolated assessment switch fixture)',company:'Fixture',originalUrl:'https://fixture.test/job',source:'fixture',description:'Build TypeScript services and maintain reliable production software. '.repeat(4),descriptionStatus:'available fixture'});
const options={localEvidence:false,availability:async()=>{probes++;return {available:false,reason:'Isolated test: no inference'};},diagnostics:async()=>[]};
let server=await serve(store,port,options);const browser=await chromium.launch({headless:true}),context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage(),checks:any[]=[];
function deferred(){let resolve!:()=>void;const promise=new Promise<void>(r=>resolve=r);return {promise,resolve};}
try{
 await page.goto(base);const toggle=page.getByRole('switch',{name:'Assess vacancies',exact:true});await toggle.waitFor();assert.equal(await toggle.getAttribute('aria-checked'),'true');assert.equal(assessmentEnabled(store),true);
 for(const enabled of [null,'false',0,{}]){const response=await page.request.put(`${base}/api/queue/assessment`,{data:{enabled}});assert.equal(response.status(),400);}
 assert.equal((await page.request.put(`${base}/api/queue/assessment`,{data:{}})).status(),400);
 assert.equal((await page.request.put(`${base}/api/queue/assessment`,{data:{enabled:false},headers:{Origin:'https://evil.example'}})).status(),403);
 assert.equal((await page.request.put(`${base}/api/queue/assessment`,{data:'false',headers:{'content-type':'text/plain'}})).status(),415);assert.equal(assessmentEnabled(store),true);
 const off=page.waitForResponse(r=>r.url().endsWith('/api/queue/assessment')&&r.request().method()==='PUT');await toggle.click();assert.equal((await off).status(),200);await page.waitForFunction(()=>document.querySelector('[role=switch]')?.getAttribute('aria-checked')==='false');assert.equal(assessmentEnabled(store),false);
 await page.reload();await toggle.waitFor();assert.equal(await toggle.getAttribute('aria-checked'),'false');
 await new Promise<void>(r=>server.close(()=>r()));store.close();store=new Store(path);server=await serve(store,port,options);await page.reload();await toggle.waitFor();assert.equal(await toggle.getAttribute('aria-checked'),'false');
 const beforeProbes=probes,id=store.list()[0].id;
 const priority=await page.request.post(`${base}/api/jobs/${id}/assess`,{data:{}});assert.equal(priority.status(),202);await page.waitForTimeout(1200);assert.equal(queueStatus(store).counts.pending,1);assert.equal(queueStatus(store).counts.running,0);assert.equal(probes,beforeProbes);
 checks.push({test:'default on, strict boolean/Origin/JSON validation, UI off survives page reload and Store/server restart; priority stays pending with no availability/inference',passed:true});
 await page.route('**/api/queue/assessment',r=>r.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:'Fixture write unavailable'})}));await toggle.click();await page.getByRole('alert').filter({hasText:'Fixture write unavailable'}).waitFor();assert.equal(await toggle.getAttribute('aria-checked'),'false');assert.equal(await toggle.isEnabled(),true);assert.equal(assessmentEnabled(store),false);await page.waitForTimeout(2600);assert.equal(await toggle.getAttribute('aria-checked'),'false');await page.unroute('**/api/queue/assessment');
 await toggle.press('Space');await page.waitForFunction(()=>document.querySelector('[role=switch]')?.getAttribute('aria-checked')==='true');assert.equal(assessmentEnabled(store),true);assert.equal(await page.locator('.assessment-switch-error').count(),0);
 // Hold an old GET snapshot and a successful PUT response independently.
 const gotSnapshot=deferred(),releaseSnapshot=deferred(),gotWrite=deferred(),releaseWrite=deferred();let held=false;
 await page.route('**/api/queue',async r=>{if(!held){held=true;const response=await r.fetch();gotSnapshot.resolve();await releaseSnapshot.promise;await r.fulfill({response});}else await r.continue();});
 await page.route('**/api/queue/assessment',async r=>{const response=await r.fetch();gotWrite.resolve();await releaseWrite.promise;await r.fulfill({response});});
 await gotSnapshot.promise;await toggle.click();await gotWrite.promise;assert.equal(await toggle.isDisabled(),true);assert.equal(await toggle.getAttribute('aria-checked'),'true');await page.getByRole('status').filter({hasText:'Saving…'}).waitFor();
 releaseWrite.resolve();await page.waitForFunction(()=>document.querySelector('[role=switch]')?.getAttribute('aria-checked')==='false');releaseSnapshot.resolve();await page.waitForTimeout(200);assert.equal(await toggle.getAttribute('aria-checked'),'false');assert.equal(await toggle.isEnabled(),true);await page.unroute('**/api/queue');await page.unroute('**/api/queue/assessment');
 checks.push({test:'failed write remains authoritative/off with visible retryable error; Space toggles; only saving disables; stale polling cannot revert successful choice',passed:true});
 const axe=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();assert.deepEqual(axe.violations.map(v=>v.id),[]);await page.screenshot({path:`${dir}/assessment-switch-desktop.png`});
 for(const width of [390,320]){
  await page.setViewportSize({width,height:844});const box=await toggle.boundingBox();assert.ok(box&&box.x>=0&&box.x+box.width<=width&&box.y+box.height<=844);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);assert.equal(await toggle.isVisible(),true);
  await page.locator('.queue-status').evaluate(e=>e.scrollLeft=e.scrollWidth);const pinned=await toggle.boundingBox();assert.deepEqual(pinned,box);
  const result=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();assert.deepEqual(result.violations.map(v=>v.id),[]);await page.screenshot({path:`${dir}/assessment-switch-${width}.png`});
 }
 checks.push({test:'desktop and 390/320px switch visible independently of status scrolling; three axe states zero violations and no page overflow',passed:true});writeFileSync(`${dir}/assessment-switch.json`,JSON.stringify({checked:now(),checks},null,2));console.log(JSON.stringify({state:localPath(),checks},null,2));
}finally{await browser.close();await new Promise<void>(r=>server.close(()=>r()));store.close();}
