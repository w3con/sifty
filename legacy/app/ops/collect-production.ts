// Explicit owner-approved production collection. No inference or application actions.
import assert from 'node:assert/strict';
import {DatabaseSync}from'node:sqlite';
import {writeFileSync,mkdirSync,existsSync,readFileSync}from'node:fs';
import {localPath,runtime,now,hash}from'../src/core.js';
if(process.env.FCTO_STATE_DIR&&process.env.FCTO_STATE_DIR!=='.local')throw new Error('Production collector expects operational .local state');
const db=new DatabaseSync(localPath('jobs.sqlite'),{readOnly:true});
const before=db.prepare('SELECT id,status,first_seen,assessment FROM jobs').all()as any[];
const assessmentCount=(db.prepare('SELECT count(*) n FROM assessments').get()as any).n;
const base=`http://127.0.0.1:${runtime().port}`,started=now();mkdirSync(localPath('verification'),{recursive:true});
writeFileSync(localPath('verification/correction-before.json'),JSON.stringify({started,jobs:before,assessmentCount},null,2));
writeFileSync(localPath(`verification/correction-before-${Date.now()}.json`),JSON.stringify({started,jobs:before,assessmentCount},null,2));
if(existsSync(localPath('verification/production-collection.json'))){const prior=JSON.parse(readFileSync(localPath('verification/production-collection.json'),'utf8'));writeFileSync(localPath(`verification/production-collection-${prior.run.id}.json`),JSON.stringify(prior,null,2));}
try{
 const r=await fetch(`${base}/api/search`,{method:'POST',headers:{'content-type':'application/json'},body:'{}'});const result=await r.json()as any;assert.equal(r.status,202,JSON.stringify(result));console.log(JSON.stringify({started,run:result}));
 const duplicate=await fetch(`${base}/api/search`,{method:'POST',headers:{'content-type':'application/json'},body:'{}'});assert.equal(duplicate.status,400);assert.match((await duplicate.json()as any).error,/already active/);console.log('Concurrent second start correctly refused.');
 let previous='';let run:any;
 for(let attempt=0;attempt<600;attempt++){
  run=await(await fetch(`${base}/api/collection`)).json();assert.equal(run.id,result.runId);
  const active=run.sources.find((s:any)=>s.status==='running');const phase=active?`${active.source}: ${active.coverage.phase}`:run.status;
  if(phase!==previous){console.log(JSON.stringify({time:now(),phase,sourceCount:run.sources.length}));previous=phase;}
  if(run.status!=='running')break;await new Promise(r=>setTimeout(r,3000));
 }
 assert.ok(run.status!=='running','Still running; durable progress remains in SQLite');
 const after=db.prepare('SELECT id,status,first_seen,assessment FROM jobs').all()as any[];
 const aliases=db.prepare('SELECT old_id,new_id FROM job_aliases').all()as any[];
 for(const old of before){const id=aliases.find(a=>a.old_id===old.id)?.new_id??old.id;const current=after.find(j=>j.id===id);assert.ok(current,`Existing record ${old.id} lost`);assert.ok(Date.parse(current.first_seen)<=Date.parse(old.first_seen));if(old.status!=='new')assert.equal(current.status,old.status);if(old.assessment)assert.equal(hash(current.assessment??''),hash(old.assessment),'Existing assessment overwritten');}
 assert.equal((db.prepare('SELECT count(*) n FROM assessments').get()as any).n,assessmentCount,'Collection unexpectedly invoked assessment');
 const health=await(await fetch(`${base}/api/health`)).json();const report={started,finished:now(),beforeCount:before.length,afterCount:after.length,existingRecordsPreserved:true,assessmentCount,assessmentCalls:0,run,health,integrity:db.prepare('PRAGMA integrity_check').get()};
 writeFileSync(localPath('verification/production-collection.json'),JSON.stringify(report,null,2));
 writeFileSync(localPath(`verification/production-collection-${run.id}.json`),JSON.stringify(report,null,2));
 console.log(JSON.stringify({before:before.length,after:after.length,runId:run.id,status:run.status,sources:run.sources.map((s:any)=>({source:s.source,status:s.status,reviewed:s.coverage.reviewed,matched:s.coverage.matched,added:s.coverage.added,updated:s.coverage.updated,duplicates:s.coverage.duplicates,details:s.coverage.detailFetched,pages:s.coverage.pages.length,stop:s.coverage.stopReason}))},null,2));
}finally{db.close();}
