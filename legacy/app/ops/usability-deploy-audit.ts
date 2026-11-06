// Coordinator-only operational entry point. Tests import helpers under test/isolate.ts.
// Never construct Store here: even opening it migrates the database.
import {DatabaseSync} from 'node:sqlite';
import {mkdirSync,readFileSync,readdirSync,writeFileSync} from 'node:fs';
import {resolve} from 'node:path';import {pathToFileURL} from 'node:url';import assert from 'node:assert/strict';
import {chromium} from 'playwright';
import {canonical,hash,localPath,rootPath,now,runtime} from '../src/core.js';
import {archiveRetentionMs,validAppliedAt} from '../src/workflow.js';

type Row=Record<string,any>;
export const jobTables=['jobs','sightings','snapshots','assessments','assessment_queue','feedback','workflow_history','application_drafts','application_revisions','questionnaire_snapshots','preparation_operations','job_aliases'];
const independentTables=['health','runs','collection_runs','source_runs','collection_events','queue_state','locks'];
const contextTables=['contexts','context_refreshes'];
const extraTables=['discovery_cache','deleted_listings','id_sequence','migrations'];
export type Before={format:1;checked:string;cutoff:number;backup:string;protected:Record<string,string>;tables:Record<string,Row[]>;eligibleIds:number[];removedIds:number[];preference:string;contextObservation:Record<string,string>};
const json=(file:string,data:unknown)=>writeFileSync(file,JSON.stringify(data,null,2),{mode:0o600});
export function rows(db:DatabaseSync,table:string):Row[]{
 assert.ok([...jobTables,...independentTables,...extraTables,...contextTables].includes(table));
 if(!db.prepare("SELECT name FROM sqlite_master WHERE type='table' AND name=?").get(table))return [];
 const pk=db.prepare(`PRAGMA table_info("${table}")`).all().filter(c=>Number(c.pk)>0).sort((a,b)=>Number(a.pk)-Number(b.pk)).map(c=>`"${c.name}"`);
 return db.prepare(`SELECT * FROM "${table}" ORDER BY ${pk.length?pk.join(','):'rowid'}`).all().map(r=>({...r}));
}
function allRows(db:DatabaseSync){for(const table of [...jobTables,...independentTables,...contextTables,'discovery_cache','migrations'])assert.ok(db.prepare("SELECT name FROM sqlite_master WHERE type='table' AND name=?").get(table),`Required table missing: ${table}`);return Object.fromEntries([...jobTables,...independentTables,...contextTables,...extraTables].map(t=>[t,rows(db,t)]));}
export function protectedHashes(){
 const paths=['cv.pdf','profile.pdf','ai/HANDOVER.md'];
 function visit(relative:string){for(const item of readdirSync(rootPath(relative),{withFileTypes:true}).sort((a,b)=>a.name.localeCompare(b.name))){const path=`${relative}/${item.name}`;if(item.isDirectory())visit(path);else if(item.isFile()||item.isSymbolicLink())paths.push(path);}}
 visit('kb');for(const file of readdirSync(rootPath('config')).sort())if(/\.ya?ml$/.test(file))paths.push(`config/${file}`);
 return Object.fromEntries(paths.sort().map(p=>[p,hash(readFileSync(rootPath(p)))]));
}
const live=(pid:unknown)=>{if(!Number.isSafeInteger(Number(pid))||Number(pid)<=0)return false;try{process.kill(Number(pid),0);return true;}catch(e:any){return e.code!=='ESRCH';}};
export function assertIdle(db:DatabaseSync){
 assert.equal(db.prepare("SELECT count(*) n FROM assessment_queue WHERE status='running'").get()!.n,0,'Assessment is running; do not deploy');
 assert.equal(db.prepare("SELECT count(*) n FROM preparation_operations WHERE status IN ('queued','running') OR (status='cancelled' AND pid IS NOT NULL)").get()!.n,0,'Preparation is queued/running/cleaning up; do not deploy');
 for(const table of ['runs','collection_runs'])assert.equal(db.prepare(`SELECT count(*) n FROM ${table} WHERE status='running'`).get()!.n,0,`${table} contains unfinished work; resolve it before deploying`);
 for(const lock of rows(db,'locks'))assert.equal(live(lock.pid),false,`Live ${lock.name} lock; do not deploy`);
 assert.equal(db.prepare("SELECT data FROM queue_state WHERE key='assessment_enabled'").get()?.data,'false','Expected explicitly saved Assessment Off; audit never changes it');
}
function deletionIds(tables:Record<string,Row[]>,eligible:number[]){const removed=new Set(eligible);let changed=true;while(changed){changed=false;for(const a of tables.job_aliases)if(removed.has(a.new_id)&&!removed.has(a.old_id)){removed.add(a.old_id);changed=true;}}return [...removed].sort((a,b)=>a-b);}
const urlKeys=(j:Row)=>[hash(`url:${canonical(j.originalUrl)}`),...(j.applyUrl?[hash(`url:${canonical(j.applyUrl)}`)]:[])];
function deletedUrlHashes(before:Before){
 const removed=new Set(before.removedIds),keys=new Set<string>();
 for(const j of before.tables.jobs)if(removed.has(j.id))for(const key of urlKeys({originalUrl:j.original_url,applyUrl:j.apply_url}))keys.add(key);
 for(const s of before.tables.sightings)if(removed.has(s.job_id))keys.add(hash(`url:${canonical(s.url)}`));
 for(const s of before.tables.snapshots)if(removed.has(s.job_id)){try{for(const key of urlKeys(JSON.parse(s.data)))keys.add(key);}catch{}}
 return keys;
}
function expectedCache(before:Before,keys:Set<string>){
 if(!before.eligibleIds.length)return before.tables.discovery_cache;
 return before.tables.discovery_cache.flatMap(row=>{try{const value=JSON.parse(row.data);if(!Array.isArray(value.jobs))return [row];value.jobs=value.jobs.filter((j:Row)=>!urlKeys(j).some(key=>keys.has(key)));return [{...row,data:JSON.stringify(value)}];}catch{return [];}});
}
export function captureBefore(cutoff=Date.now()):Before{
 assert.ok(Number.isFinite(cutoff),'Valid cutoff time required');
 const db=new DatabaseSync(localPath('jobs.sqlite'),{readOnly:true});
 try{
  assertIdle(db);const protectedBefore=protectedHashes();mkdirSync(localPath('backups'),{recursive:true});mkdirSync(localPath('verification'),{recursive:true});
  const backup=localPath(`backups/usability-before-${Date.now()}.sqlite`);db.prepare('VACUUM INTO ?').run(backup);
  const copy=new DatabaseSync(backup,{readOnly:true});let tables:Record<string,Row[]>;let contextObservation:Record<string,string>;
  try{assertIdle(copy);tables=allRows(copy);contextObservation=Object.fromEntries(['contexts','context_refreshes'].map(t=>[t,hash(JSON.stringify(rows(copy,t)))]));}finally{copy.close();}
  // Refuse a moving baseline rather than combining rows from different instants.
  assertIdle(db);assert.deepEqual(allRows(db),tables,'Database changed while taking backup; repeat before');assert.deepEqual(protectedHashes(),protectedBefore,'Protected files changed during backup');
  const eligibleIds=tables.jobs.filter(j=>j.status==='applied'&&validAppliedAt(j.applied_at)&&cutoff-Date.parse(j.applied_at)>=archiveRetentionMs).map(j=>j.id).sort((a,b)=>a-b);
  const before:Before={format:1,checked:now(),cutoff,backup,protected:protectedBefore,tables,eligibleIds,removedIds:deletionIds(tables,eligibleIds),preference:'false',contextObservation};
  json(localPath('verification/usability-before.json'),before);return before;
 }finally{db.close();}
}
export function verifyDatabase(db:DatabaseSync,before:Before){
 assert.equal(before.format,1);assertIdle(db);assert.equal(before.preference,'false');assert.equal(db.prepare('PRAGMA user_version').get()!.user_version,10);
 assert.deepEqual(db.prepare('PRAGMA integrity_check').all().map(r=>Object.values(r)[0]),['ok']);assert.deepEqual(db.prepare('PRAGMA foreign_key_check').all(),[]);
 for(const table of ['deleted_listings','id_sequence'])assert.ok(db.prepare("SELECT name FROM sqlite_master WHERE type='table' AND name=?").get(table),`Required schema10 table missing: ${table}`);
 const current=allRows(db),removed=new Set(before.removedIds);
 const eligible=before.tables.jobs.filter(j=>j.status==='applied'&&validAppliedAt(j.applied_at)&&before.cutoff-Date.parse(j.applied_at)>=archiveRetentionMs).map(j=>j.id).sort((a,b)=>a-b);
 assert.deepEqual(before.eligibleIds,eligible,'Eligible IDs must match the original strict cutoff');assert.deepEqual(before.removedIds,deletionIds(before.tables,eligible));
 for(const table of jobTables){const expected=before.tables[table].filter(r=>table==='jobs'?!removed.has(r.id):table==='job_aliases'?!removed.has(r.old_id)&&!removed.has(r.new_id):!removed.has(r.job_id));assert.deepEqual(current[table],expected,`${table}: survivor/history rows changed beyond eligible deletion`);}
 for(const table of independentTables)assert.deepEqual(current[table],before.tables[table],`${table}: independent records changed`);
 for(const table of contextTables){const key=table==='contexts'?'version':'id';for(const old of before.tables[table])assert.deepEqual(current[table].find(r=>r[key]===old[key]),old,`${table}: historical candidate context changed`);if(table==='context_refreshes'){const max=Math.max(0,...before.tables[table].map(r=>r.id));assert.ok(current[table].filter(r=>!before.tables[table].some(old=>old.id===r.id)).every(r=>r.id>max),'New context receipts must be appended');}}
 const keys=deletedUrlHashes(before),tombstones=[...new Set([...before.tables.deleted_listings.map(r=>r.hash),...keys])].sort();
 assert.deepEqual(current.deleted_listings,tombstones.map(hash=>({hash})),'Unexpected or missing URL-only tombstones');assert.deepEqual(current.discovery_cache,expectedCache(before,keys),'Cached inventories changed beyond deleted URLs');
 const highWater=Math.max(0,...before.tables.jobs.map(r=>r.id),...before.tables.job_aliases.flatMap(r=>[r.old_id,r.new_id]),...before.tables.id_sequence.filter(r=>r.name==='jobs').map(r=>r.value));
 const sequences=before.tables.id_sequence.filter(r=>r.name!=='jobs').concat({name:'jobs',value:highWater}).sort((a,b)=>a.name.localeCompare(b.name));assert.deepEqual(current.id_sequence,sequences,'Job high-water mark changed unexpectedly');
 const migrations=[...new Set([...before.tables.migrations.map(r=>r.version),10])].sort((a,b)=>a-b).map(version=>({version}));assert.deepEqual(current.migrations,migrations);
 assert.deepEqual(protectedHashes(),before.protected,'Protected config/CV/KB files changed');
 return {survivors:current.jobs.length,deletedEligibleIds:before.eligibleIds,deletedAliasIds:before.removedIds.filter(id=>!before.eligibleIds.includes(id)),tombstones:current.deleted_listings.length,contextHistoryPreserved:true,newContextReceipts:current.context_refreshes.length-before.tables.context_refreshes.length};
}
async function browserAudit(base:string,before:Before){
 const url=new URL(base);assert.ok(['127.0.0.1','localhost'].includes(url.hostname)&&url.protocol==='http:'&&!url.username&&!url.password&&url.pathname==='/','Loopback HTTP base URL required');
 assert.ok(!before.tables.jobs.some(r=>r.id===-1)&&!before.tables.job_aliases.some(r=>r.old_id===-1),'Cancellation probe ID must not exist or alias a real role');
 const probe=await fetch(`${url.origin}/api/jobs/-1/application/cancel`,{method:'POST',headers:{'content-type':'application/json'},body:'{}'});assert.equal(probe.status,400,'New cancellation backend route is not loaded');assert.equal((await probe.json() as any).error,'Job -1 not found');
 const browser=await chromium.launch(),context=await browser.newContext({viewport:{width:1440,height:1000},serviceWorkers:'block'}),page=await context.newPage(),blocked:Row[]=[],errors:string[]=[];
 page.on('pageerror',e=>errors.push(e.message));
 await context.route('**/*',route=>{const request=route.request(),destination=new URL(request.url());if(destination.origin!==url.origin||!['GET','HEAD'].includes(request.method())){blocked.push({method:request.method(),url:request.url()});return route.abort();}return route.continue();});
 try{
  await page.goto(url.origin);await page.locator('.job-row').first().waitFor();assert.equal(await page.locator('.app-header .filter-box').count(),0);assert.equal(await page.locator('.list-meta').count(),0);
  await page.locator('.list-toolbar').getByRole('button',{name:'Unread',exact:true}).waitFor();await page.locator('.list-toolbar').getByRole('button',{name:/^Filters/}).waitFor();assert.deepEqual(await page.getByRole('combobox',{name:'Sort roles',exact:true}).locator('option').allTextContents(),['Best fit','Newest']);
  await page.getByRole('button',{name:/^Triage \d/}).click();await page.getByRole('button',{name:'Application',exact:true}).waitFor();assert.equal(await page.getByRole('button',{name:'Application',exact:true}).isDisabled(),true);await page.getByText('Select this role first to prepare an application.',{exact:true}).waitFor();
  await page.getByRole('button',{name:/^Archive \d/}).click();await page.locator('.archive-retention').waitFor();assert.match(await page.locator('.archive-retention').innerText(),/30 days/);
   const stored=before.tables.application_drafts.find(r=>r.job_id===758);assert.ok(stored,'Job758 draft is missing from baseline');const draft=JSON.parse(stored.data);
   await page.getByRole('button',{name:/^Pre-selected \d/}).click();await page.locator('.job-row[data-job-id="758"]').click();await page.getByRole('button',{name:'Application',exact:true}).click();const panel=page.getByRole('region',{name:'Application draft',exact:true});await panel.waitFor();
   if(!draft.fields.length){
    assert.equal(draft.questionnaire?.status,'login-required');assert.ok(before.tables.preparation_operations.some(r=>r.job_id===758&&r.status==='done'),'Expected historical login-only completion');
    await panel.getByText('No prepared answers yet. Inspect the questionnaire or sign in to continue.',{exact:true}).waitFor();await panel.getByRole('button',{name:'Sign in to continue',exact:true}).waitFor();
   }else{
    await panel.locator('.draft-field').first().waitFor();assert.equal(await panel.locator('.draft-field').count(),draft.fields.length,'Saved questionnaire fields missing from live UI');
    await panel.getByRole('button',{name:'Review saved answers',exact:true}).waitFor();
   }
   assert.equal(await panel.getByText('Prepared for review; nothing submitted',{exact:true}).count(),0);
  const screenshots:string[]=[];for(const [label,width,height] of [['desktop',1440,1000],['mobile',390,844]] as const){await page.setViewportSize({width,height});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`${label} overflow`);const file=localPath(`verification/usability-${label}.png`);await page.screenshot({path:file});screenshots.push(file);}
  assert.deepEqual(errors,[]);const unexpected=blocked.filter(r=>r.method!=='POST'||!new RegExp(`^${url.origin.replace(/[.*+?^${}()|[\]\\]/g,'\\$&')}/api/jobs/\\d+/seen$`).test(r.url));assert.deepEqual(unexpected,[],'Unexpected mutation/external request attempted by audit UI');
  return {newCancellationBackend:true,toolbar:true,noHeaderFilters:true,triageApplicationDisabled:true,archive:true,legacyJob758Truthful:true,blockedRequests:blocked,screenshots,browserErrors:errors};
 }catch(e){await page.screenshot({path:localPath('verification/usability-browser-failed.png')}).catch(()=>{});json(localPath('verification/usability-browser-failed.json'),{error:String(e),browserErrors:errors,blockedRequests:blocked});throw e;}finally{await browser.close();}
}
export async function verifyDeployment(base=`http://127.0.0.1:${runtime().port}`){
 const before=JSON.parse(readFileSync(localPath('verification/usability-before.json'),'utf8')) as Before,db=new DatabaseSync(localPath('jobs.sqlite'),{readOnly:true});
 try{const preservation=verifyDatabase(db,before),browser=await browserAudit(base,before);assert.deepEqual(verifyDatabase(db,before),preservation,'Audit modified persistent records');const report={checked:now(),schema:10,integrity:'ok',foreignKeyViolations:0,assessmentOff:true,backup:before.backup,...preservation,...browser};json(localPath('verification/usability-deploy.json'),report);return report;}finally{db.close();}
}
if(process.argv[1]&&pathToFileURL(resolve(process.argv[1])).href===import.meta.url){
 try{const mode=process.argv[2];if(mode==='before'){const b=captureBefore();console.log(JSON.stringify({backup:b.backup,roles:b.tables.jobs.length,eligibleIds:b.eligibleIds,snapshot:localPath('verification/usability-before.json')}));}else if(mode==='verify')console.log(JSON.stringify(await verifyDeployment(process.argv[3])));else throw Error('Use before | verify [http://127.0.0.1:PORT]');}
 catch(e){mkdirSync(localPath('verification'),{recursive:true});json(localPath('verification/usability-failed.json'),{checked:now(),mode:process.argv[2],error:String(e)});console.error(String(e));process.exitCode=1;}
}
