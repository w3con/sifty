import {z} from 'zod';import {now,providers} from './core.js';import type {Store} from './store.js';import {acquireLock} from './services.js';
import {localPath} from './core.js';import {mkdirSync,writeFileSync} from 'node:fs';import {concisePreparationError} from './preparation-error.js';
import {externalApplyUrl} from './workflow.js';import {applicationDraft,inspectIntoDraft} from './application-store.js';import {inspectApplication,openApplicationLogin} from './application-browser.js';import {generateApplication} from './application-generation.js';
const requestSchema=z.object({inspect:z.boolean().default(true),generate:z.boolean().default(true),provider:z.string().optional(),fieldId:z.string().optional(),login:z.boolean().default(false)}).strict();
// Only the worker that owns a running operation can stop its browser/child process.
const owners=new WeakMap<Store,Map<number,AbortController>>();
export function preparationState(store:Store,id:number){id=store.get(id).id;return {draft:applicationDraft(store,id),operations:store.db.prepare('SELECT * FROM preparation_operations WHERE job_id=? ORDER BY id DESC LIMIT 10').all(id)};}
export function queuePreparation(store:Store,id:number,input:unknown){
 const job=store.get(id),request=requestSchema.parse(input);if(job.stage!=='preselected')throw Error('Pre-select this role before preparing an application');if(request.provider!==undefined&&!providers().providers[request.provider])throw Error('Unknown preparation provider');
 const active=store.db.prepare("SELECT * FROM preparation_operations WHERE job_id=? AND status IN ('queued','running')").get(job.id);if(active)return active;
 const result=store.db.prepare("INSERT INTO preparation_operations(job_id,status,created,updated,request,progress) VALUES(?,'queued',?,?,?,'Waiting for preparation worker')").run(job.id,now(),now(),JSON.stringify(request));return store.db.prepare('SELECT * FROM preparation_operations WHERE id=?').get(result.lastInsertRowid);
}
export function cancelPreparation(store:Store,id:number){
 id=store.get(id).id;for(const op of store.db.prepare("SELECT id,status FROM preparation_operations WHERE job_id=? AND status IN ('queued','running')").all(id)){
  const owner=owners.get(store)?.get(Number(op.id));if(op.status==='running'&&!owner)throw Error('Preparation is owned by another process; cancel from its server or wait for recovery');
  store.db.prepare("UPDATE preparation_operations SET status='cancelled',progress='Cancelled; saved answers preserved',error=NULL,updated=? WHERE id=?").run(now(),op.id!);owner?.abort();
 }return preparationState(store,id);
}
export class PreparationWorker{
 private busy=false;private stopped=false;private timer?:NodeJS.Timeout;private owned=new Map<number,AbortController>();
 constructor(private store:Store,private inspect=inspectApplication,private generate=generateApplication,private login=openApplicationLogin){
  if(!owners.has(store))owners.set(store,new Map());
  for(const row of store.db.prepare("SELECT id,pid FROM preparation_operations WHERE status='running' OR (status='cancelled' AND pid IS NOT NULL)").all()){let alive=false;try{if(row.pid){process.kill(Number(row.pid),0);alive=true;}}catch{}if(!alive)store.db.prepare("UPDATE preparation_operations SET status=CASE WHEN status='cancelled' THEN 'cancelled' ELSE 'interrupted' END,pid=NULL,updated=?,error=CASE WHEN status='cancelled' THEN NULL ELSE 'Preparation interrupted; existing draft preserved. Retry explicitly.' END,progress=CASE WHEN status='cancelled' THEN 'Cancelled; saved answers preserved' ELSE 'Interrupted' END WHERE id=?").run(now(),row.id!);}
 }
 start(){this.timer=setInterval(()=>void this.tick(),1000);this.timer.unref();void this.tick();return this;}
 stop(){this.stopped=true;if(this.timer)clearInterval(this.timer);for(const controller of this.owned.values())controller.abort(Error('Server stopped; preparation interrupted'));}
 async shutdown(){this.stop();while(this.busy)await new Promise(r=>setTimeout(r,10));}
 async tick(){if(this.busy||this.stopped)return;this.busy=true;let operation:any,controller:AbortController|undefined;
  const progress=(message:string)=>this.store.db.prepare("UPDATE preparation_operations SET progress=?,updated=? WHERE id=? AND status='running'").run(message,now(),operation.id);
  const check=()=>{controller!.signal.throwIfAborted();if(this.store.db.prepare('SELECT status FROM preparation_operations WHERE id=?').get(operation.id)?.status!=='running')throw Error('Operation no longer running');};
  try{
   this.store.db.exec('BEGIN IMMEDIATE');try{
    if(this.store.db.prepare("SELECT id FROM preparation_operations WHERE status='running' OR (status='cancelled' AND pid IS NOT NULL)").get()){this.store.db.exec('COMMIT');return;}
    operation=this.store.db.prepare("SELECT * FROM preparation_operations WHERE status='queued' ORDER BY id LIMIT 1").get();if(operation)this.store.db.prepare("UPDATE preparation_operations SET status='running',pid=?,updated=? WHERE id=?").run(process.pid,now(),operation.id);this.store.db.exec('COMMIT');
   }catch(e){this.store.db.exec('ROLLBACK');throw e;}
   if(!operation)return;controller=new AbortController();this.owned.set(operation.id,controller);owners.get(this.store)!.set(operation.id,controller);const signal=controller.signal;
   const request=requestSchema.parse(JSON.parse(operation.request)),job=this.store.get(operation.job_id),url=externalApplyUrl(job);if(job.stage!=='preselected')throw Error('Preparation stopped: this role is no longer Pre-selected. Saved answers are preserved.');if(!url)throw Error('No valid application URL; add manual questions instead');
   if(request.login){const destination=applicationDraft(this.store,job.id).questionnaire?.url??url;progress('Waiting for sign-in. Close the sign-in window when finished; preparation will continue automatically.');await this.login(destination,{signal});check();request.inspect=true;request.generate=true;request.login=false;this.store.db.prepare('UPDATE preparation_operations SET request=? WHERE id=?').run(JSON.stringify(request),operation.id);}
   if(request.inspect){progress('Reading questionnaire; nothing is filled or submitted');const snapshot=await this.inspect(url,{signal});check();inspectIntoDraft(this.store,job.id,snapshot);if(!snapshot.fields.length){throw Error(snapshot.status==='login-required'?'Sign in to continue, then close the sign-in window. Saved answers are preserved.':'No readable questions found. Retry inspection or add questions manually.');}request.inspect=false;this.store.db.prepare('UPDATE preparation_operations SET request=? WHERE id=?').run(JSON.stringify(request),operation.id);}
   if(request.generate){
    const resolvedProvider=request.provider??providers().stages.preparation;
    if(!resolvedProvider)throw Error('No preparation model configured. Choose one in Configuration before preparing answers.');
    if(!providers().providers[resolvedProvider])throw Error(`Preparation provider "${resolvedProvider}" is not configured. Update Configuration and retry.`);
    let release:()=>void;try{release=acquireLock(this.store,'inference');}catch{check();this.store.db.prepare("UPDATE preparation_operations SET status='queued',progress='Waiting for shared inference slot',pid=NULL,updated=?,request=? WHERE id=?").run(now(),JSON.stringify(request),operation.id);return;}
    try{check();progress(`Drafting answers with ${resolvedProvider}; saved edits are protected`);await this.generate(this.store,job.id,resolvedProvider,request.fieldId,undefined,signal);check();}finally{release();}
   }
   check();this.store.db.prepare("UPDATE preparation_operations SET status='done',progress='Questionnaire ready for review; nothing submitted',updated=?,pid=NULL WHERE id=? AND status='running'").run(now(),operation.id);
  }catch(e){if(operation){
   const cancelled=controller?.signal.aborted;const status=cancelled?(this.stopped?'interrupted':'cancelled'):'failed';
   if(!cancelled){mkdirSync(localPath('application-preparation-errors'),{recursive:true});writeFileSync(localPath(`application-preparation-errors/${operation.id}.json`),JSON.stringify({operationId:operation.id,jobId:operation.job_id,created:now(),error:String(e),stack:e instanceof Error?e.stack:null},null,2),{mode:0o600});}
   this.store.db.prepare('UPDATE preparation_operations SET status=?,error=?,progress=?,updated=?,pid=NULL WHERE id=?').run(status,cancelled?null:concisePreparationError(e),`${status==='cancelled'?'Cancelled':status==='interrupted'?'Interrupted':'Preparation failed'}; saved answers preserved`,now(),operation.id);
  }else console.error('Preparation worker:',String(e));}
  finally{if(operation){this.owned.delete(operation.id);owners.get(this.store)?.delete(operation.id);this.store.db.prepare("UPDATE preparation_operations SET pid=NULL WHERE id=? AND status='cancelled'").run(operation.id);}this.busy=false;}
 }
}
