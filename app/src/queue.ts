import { readFileSync } from 'node:fs';
import { hash, now, providers, rootPath, searchConfig } from './core.js';
import { assess, localAvailability } from './providers.js';
import type { Store } from './store.js';
import { bindFeedback, contextForRole } from './feedback.js';
import { contextVersion } from './context.js';

export const descriptionReady=(j:{description:string;descriptionStatus:string})=>j.description.length>=100&&!/metadata|pending|missing|unavailable/i.test(j.descriptionStatus);
function fingerprintInputs(store:Store){const config=providers(),preferences=searchConfig().preferences,prompt=readFileSync(rootPath('prompts/assessment.md'),'utf8');let c=store.context();if(c){c=bindFeedback(c);if(JSON.stringify(c.preferences)!==JSON.stringify(preferences)){c={...c,preferences};c.version=contextVersion(c);}}return {config,preferences,prompt,context:c};}
export function fingerprint(store:Store,id:number,provider=providers().stages.assessment,inputs=fingerprintInputs(store)){
 const j=store.get(id),c=inputs.context?contextForRole(inputs.context,j.id,n=>store.resolveId(n)):null,p=inputs.config.providers[provider];if(!p)throw new Error('Unknown assessment provider');
 return hash(JSON.stringify({job:{title:j.title,company:j.company,description:j.description,descriptionStatus:j.descriptionStatus,location:j.location},context:c?.version,preferences:inputs.preferences,prompt:inputs.prompt,provider,p}));
}
export function enqueueJob(store:Store,id:number,priority=0,force=false,override?:string,inputs=fingerprintInputs(store)){
 const j=store.get(id);
 const old=store.db.prepare('SELECT * FROM assessment_queue WHERE job_id=?').get(j.id)as any;
 const manual=override??(old?.routing===inputs.config.stages.assessment?old.override_provider:null),provider=manual??inputs.config.stages.assessment,key=fingerprint(store,j.id,provider,inputs);
 if(old?.fingerprint===key&&(!force||old.status!=='done')){if(priority>old.priority)store.db.prepare('UPDATE assessment_queue SET priority=? WHERE job_id=?').run(priority,j.id);return old;}
 if(old?.status==='running')return old; // next reconciliation detects changed input after the in-flight result
 const status=!force&&j.assessment?.inputFingerprint===key?'done':descriptionReady(j)?'pending':'missing-description';
 store.db.prepare(`INSERT INTO assessment_queue(job_id,fingerprint,provider,status,priority,override_provider,routing) VALUES(?,?,?,?,?,?,?) ON CONFLICT(job_id) DO UPDATE SET fingerprint=excluded.fingerprint,provider=excluded.provider,status=excluded.status,priority=excluded.priority,override_provider=excluded.override_provider,routing=excluded.routing,attempts=0,next_at=0,started=NULL,finished=NULL,error=NULL,owner=NULL`).run(j.id,key,provider,status,priority,manual??null,inputs.config.stages.assessment);
 return {jobId:j.id,status,fingerprint:key};
}
export function enqueueAll(store:Store){const inputs=fingerprintInputs(store);for(const r of store.db.prepare('SELECT id FROM jobs').all()as any[])enqueueJob(store,r.id,0,false,undefined,inputs);}
export function assessmentEnabled(store:Store):boolean {
 const row=store.db.prepare("SELECT data FROM queue_state WHERE key='assessment_enabled'").get();
 return row?JSON.parse(String(row.data))===true:true;
}
export function setAssessmentEnabled(store:Store,enabled:unknown){
 if(typeof enabled!=='boolean')throw new Error('enabled must be a boolean');
 store.db.prepare("INSERT OR REPLACE INTO queue_state(key,data) VALUES('assessment_enabled',?)").run(JSON.stringify(enabled));
 return queueStatus(store);
}
export function queueStatus(store:Store){
 const rows=store.db.prepare('SELECT * FROM assessment_queue ORDER BY priority DESC,job_id').all()as any[];
  return {enabled:assessmentEnabled(store),counts:Object.fromEntries(['running','pending','failed','done','missing-description'].map(s=>[s,rows.filter(r=>r.status===s).length])),rows:rows.filter(r=>r.status!=='done').map(r=>({...r,elapsedMs:r.started?(r.finished??Date.now())-r.started:0})),runtime:store.db.prepare("SELECT data FROM queue_state WHERE key='runtime'").get()??null};
}
export class AssessmentWorker {
 private busy=false;private stopped=false;private timer?:NodeJS.Timeout;private lastScan=0;
 constructor(private store:Store,private inference=assess,private availability=localAvailability,private clock=Date.now) {
  store.db.exec('CREATE TABLE IF NOT EXISTS queue_state(key TEXT PRIMARY KEY,data TEXT NOT NULL)');
  for(const row of store.db.prepare("SELECT job_id,owner FROM assessment_queue WHERE status='running'").all()as any[]){let alive=false;try{if(row.owner){process.kill(row.owner,0);alive=true;}}catch{}if(!alive)store.db.prepare("UPDATE assessment_queue SET status='pending',owner=NULL,error='Recovered interrupted inference',next_at=? WHERE job_id=?").run(clock()+5000,row.job_id);}
 }
 start(){this.timer=setInterval(()=>void this.tick(),1000);this.timer.unref();void this.tick();return this;}
 stop(){this.stopped=true;if(this.timer)clearInterval(this.timer);}
 async tick(){
   if(this.busy||this.stopped||!assessmentEnabled(this.store))return;this.busy=true;
  try{
   if(this.clock()-this.lastScan>30000){enqueueAll(this.store);this.lastScan=this.clock();}
   const state=this.store.db.prepare("SELECT data FROM queue_state WHERE key='runtime'").get()as any;
   const previous=state?JSON.parse(state.data):null;const configKey=hash(JSON.stringify(providers()));
    this.store.db.exec('BEGIN IMMEDIATE');let row:any;
    try{
     if(!assessmentEnabled(this.store)){this.store.db.exec('COMMIT');return;}
    const lock=this.store.db.prepare("SELECT pid FROM locks WHERE name='inference'").get()as any;
    if(lock){let alive=true;try{process.kill(lock.pid,0);}catch{alive=false;}if(alive){this.store.db.exec('COMMIT');return;}this.store.db.prepare("DELETE FROM locks WHERE name='inference'").run();}
    row=this.store.db.prepare("SELECT * FROM assessment_queue WHERE status IN ('pending','failed') AND next_at<=? ORDER BY priority DESC,job_id LIMIT 1").get(this.clock());
    if(!row){this.store.db.exec('COMMIT');return;}
    if(previous?.retryAt>this.clock()&&previous.configKey===configKey&&previous.provider===row.provider){this.store.db.exec('COMMIT');return;}
    this.store.db.prepare("INSERT INTO locks VALUES('inference',?,?)").run(process.pid,now());
    this.store.db.exec('COMMIT');
   }catch(e){this.store.db.exec('ROLLBACK');throw e;}
   try{
     const p=providers().providers[row.provider];const ready=await this.availability(p);
     // A user may switch off while the asynchronous provider probe is pending.
     // Serialize the final permission check with other processes' preference writes.
     this.store.db.exec('BEGIN IMMEDIATE');
     try{
       if(this.stopped||!assessmentEnabled(this.store)||!this.store.db.prepare('SELECT id FROM jobs WHERE id=?').get(row.job_id)){this.store.db.exec('COMMIT');return;}
      if(ready.available){
       this.store.db.prepare("UPDATE assessment_queue SET status='running',started=?,finished=NULL,owner=?,attempts=attempts+1 WHERE job_id=?").run(this.clock(),process.pid,row.job_id);
      }
      this.store.db.exec('COMMIT');
     }catch(e){this.store.db.exec('ROLLBACK');throw e;}
    if(!ready.available){this.store.db.prepare("INSERT OR REPLACE INTO queue_state VALUES('runtime',?)").run(JSON.stringify({paused:true,reason:ready.reason,retryAt:this.clock()+60000,configKey,provider:row.provider}));return;}
    this.store.db.prepare("INSERT OR REPLACE INTO queue_state VALUES('runtime',?)").run(JSON.stringify({paused:false,provider:row.provider,model:p.model,configKey}));
    try{
     await this.inference(this.store,row.job_id,row.provider);
     this.store.db.prepare("UPDATE assessment_queue SET status='done',finished=?,error=NULL,owner=NULL WHERE job_id=?").run(this.clock(),row.job_id);
     enqueueJob(this.store,row.job_id); // changed context/description while inference ran
    }catch(e){
     const delay=Math.min(6*3600000,30000*2**Math.min(row.attempts,10));
     this.store.db.prepare("UPDATE assessment_queue SET status='failed',finished=?,error=?,next_at=?,owner=NULL WHERE job_id=?").run(this.clock(),String(e),this.clock()+delay,row.job_id);
     this.store.db.prepare("INSERT OR REPLACE INTO queue_state VALUES('runtime',?)").run(JSON.stringify({paused:true,reason:String(e),retryAt:this.clock()+delay,configKey,provider:row.provider}));
    }
   }finally{this.store.db.prepare("DELETE FROM locks WHERE name='inference' AND pid=?").run(process.pid);}
  }catch(e){console.error('Assessment worker:',String(e));}finally{this.busy=false;}
 }
}
