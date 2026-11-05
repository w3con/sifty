// Explicit bounded read-only production diagnostic. No Store/migrations, collection or inference.
import { DatabaseSync } from 'node:sqlite';
import {readFileSync,mkdirSync,writeFileSync,existsSync} from 'node:fs';
import assert from 'node:assert/strict';
import {hash,localPath,rootPath,now} from '../src/core.js';
const baseline=process.argv[2];if(!baseline?.startsWith('.local/backups/')||!baseline.endsWith('.sqlite'))throw new Error('Pass the pre-refinement backup as a root-relative .local/backups/*.sqlite path');
const before=new DatabaseSync(rootPath(baseline),{readOnly:true}),current=new DatabaseSync(localPath('jobs.sqlite'),{readOnly:true});
try{
 const columns=(before.prepare('PRAGMA table_info(jobs)').all()as any[]).map(c=>c.name);assert.ok(columns.every(c=>/^[a-z_]+$/.test(c)));
 const rows=(db:DatabaseSync,table:string,projection='*')=>db.prepare(`SELECT ${projection} FROM ${table} ORDER BY rowid`).all();
 const oldJobs=rows(before,'jobs',columns.join(',')),jobs=rows(current,'jobs',columns.join(','));assert.deepEqual(jobs,oldJobs,'Existing job content, IDs, statuses and current assessments must remain unchanged');
 const history=['sightings','snapshots','assessments','job_aliases','collection_runs','source_runs','collection_events'];const preserved:Record<string,number>={};for(const table of history){const old=rows(before,table),live=rows(current,table);assert.deepEqual(live,old,`${table} changed during read-only verification`);preserved[table]=old.length;}
 const integrity=current.prepare('PRAGMA integrity_check').get(),backupIntegrity=before.prepare('PRAGMA integrity_check').get();assert.equal((integrity as any).integrity_check,'ok');assert.equal((backupIntegrity as any).integrity_check,'ok');
 const contextRow=before.prepare('SELECT data FROM context_refreshes ORDER BY id DESC LIMIT 1').get()as any;const localInputs=JSON.parse(contextRow.data).sources.filter((s:any)=>s.reference&&!/^https?:/.test(s.reference)&&existsSync(rootPath(s.reference))).map((s:any)=>({path:s.reference,unchanged:hash(readFileSync(rootPath(s.reference)))===s.hash}));assert.ok(localInputs.every((s:any)=>s.unchanged),'A baseline candidate file changed');
 const base='http://127.0.0.1:4317',get=async(path:string)=>{const r=await fetch(`${base}${path}`,{signal:AbortSignal.timeout(10000)});assert.ok(r.ok,`${path}: HTTP ${r.status}`);return r;};
 const html=await(await get('/')).text(),built=readFileSync(rootPath('app/dist/index.html'),'utf8');assert.equal(html,built,'Server must serve the latest built index');const asset=html.match(/src="([^"]+\.js)"/)?.[1];assert.ok(asset);const bundle=await(await get(asset)).text();assert.equal(hash(bundle),hash(readFileSync(rootPath(`app/dist${asset}`))));
 const queue=await(await get('/api/queue')).json()as any;assert.equal(Object.values(queue.counts).reduce((n:number,v)=>n+Number(v),0),jobs.length);const runtime=queue.runtime?JSON.parse(queue.runtime.data):null;assert.equal(runtime.provider,'ollama');assert.equal(runtime.paused,true);assert.match(runtime.reason,/unavailable|installed model/i);
 const health=await(await get('/api/health')).json()as any[];const braintrust=health.find(s=>/braintrust/i.test(s.name));
 const report={checked:now(),baseline,jobs:jobs.length,allBaselineJobColumnsUnchanged:true,statuses:current.prepare('SELECT status,count(*) count FROM jobs GROUP BY status').all(),historyPreserved:preserved,integrity,backupIntegrity,localInputs,latestFrontendAsset:asset,latestFrontendServed:true,queue:{counts:queue.counts,runtime},braintrust:braintrust?{status:braintrust.status,reviewed:braintrust.coverage?.reviewed,available:braintrust.coverage?.descriptionAvailable,deferred:braintrust.coverage?.descriptionDeferred,failed:braintrust.coverage?.descriptionFailed,unreviewed:braintrust.coverage?.descriptionUnreviewed}:null,noProductionCollectionOrInference:true};
 mkdirSync(localPath('verification'),{recursive:true});writeFileSync(localPath('verification/refinements-state.json'),JSON.stringify(report,null,2));console.log(JSON.stringify(report,null,2));
}finally{before.close();current.close();}
