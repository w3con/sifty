import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { z } from 'zod';
import { canonical, hash, localPath, now, providers, rootPath, runtime, searchConfig, searchSchema, yaml } from './core.js';
import { candidateSchema, refreshContext } from './context.js';
import { collectSource, enrich, importUrl, sourceCapability } from './collectors.js';
import { assess as performAssessment } from './providers.js';
import { networkSchema } from './network.js';
import { Store } from './store.js';
import YAML from 'yaml';
import { discoverSource, type Coverage, type DiscoveryOptions } from './discovery.js';
import { enqueueAll, enqueueJob } from './queue.js';

export function validateAll() { searchConfig(); runtime(); providers(); candidateSchema.parse(yaml('candidate')); networkSchema.parse(yaml('network')); return { valid:true }; }
export function sourceHealth(store: Store) {
  const recorded = store.healthList() as any[];
  return searchConfig().sites.map(s => {
    const r = recorded.find(r => r.source === s.name);
    const latest=store.db.prepare('SELECT s.*,r.started,r.finished,r.mode FROM source_runs s JOIN collection_runs r ON r.id=s.run_id WHERE s.source=? ORDER BY r.id DESC LIMIT 1').get(s.name)as any;
    const coverage=latest?JSON.parse(latest.coverage):null;
    if(coverage){coverage.descriptionAvailable??=Math.max(0,coverage.reviewed-coverage.descriptionUnreviewed);coverage.descriptionFailed??=Math.min(coverage.descriptionUnreviewed,coverage.detailFailed??0);coverage.descriptionDeferred??=Math.max(0,coverage.descriptionUnreviewed-coverage.descriptionFailed);}
    const status=latest?.status==='healthy'&&coverage?.descriptionUnreviewed>0?'bounded':latest?.status??'not-run';
    return { ...s,capability:sourceCapability(s),status:!s.enabled ? 'disabled' : status,message:coverage?.blocker??r?.message??'No production search recorded',checked:coverage?.finished??null,started:coverage?.started??null,runId:latest?.run_id??null,mode:latest?.mode??null,coverage,count:coverage?.imported??0,legacy:!latest,stale:!coverage?.finished || Date.now()-Date.parse(coverage.finished)>36*3600000 };
  });
}
export async function search(store: Store, sourceName?: string,options:DiscoveryOptions={},onStart?:(id:number)=>void) {
  const config = searchConfig(), sites = config.sites.filter(s => s.enabled && (!sourceName || s.name === sourceName));
  if (sourceName && !sites.length) throw new Error(`No enabled source named ${sourceName}`);
  if (!config.search.titles.length && !config.search.keywords.length) throw new Error('No OR queries configured; refusing to fetch every job');
  const mode=(process.env.FCTO_STATE_DIR??'.local')==='.local'?'production':'isolated-test';
  const runId=Number(store.db.prepare("INSERT INTO collection_runs(started,status,pid,mode,config_hash) VALUES(?,'running',?,?,?)").run(now(),process.pid,mode,hash(JSON.stringify(config))).lastInsertRowid);
  onStart?.(runId);const event=(source:string,message:string)=>store.db.prepare('INSERT INTO collection_events(run_id,created,source,message) VALUES(?,?,?,?)').run(runId,now(),source,message);
  const save=(source:string,status:string,coverage:Coverage)=>store.db.prepare('INSERT INTO source_runs VALUES(?,?,?,?) ON CONFLICT(run_id,source) DO UPDATE SET status=excluded.status,coverage=excluded.coverage').run(runId,source,status,JSON.stringify(coverage));
  const results = [];event('',`Started ${mode} collection for ${sites.length} sources; configuration fingerprint recorded.`);
  try{
  for (const source of sites) {
    try {
      event(source.name,'Searching public source');let phase='';
      const host=new URL(source.url).hostname;
      const queryScoped=/web3\.careers?/.test(host);
      const key=hash(JSON.stringify({source,adapter:2,queries:queryScoped?config.search:undefined,limits:runtime()}));
      const cached=store.db.prepare('SELECT * FROM discovery_cache WHERE key=?').get(key)as any;
      const inventory=!options.force&&cached&&Date.now()-cached.created<3*3600000?JSON.parse(cached.data):undefined;
      if(inventory)event(source.name,'Reusing fresh public inventory; locally re-filtering and resuming missing descriptions.');
      const beforeIds=new Set((store.db.prepare('SELECT id FROM jobs').all()as any[]).map(r=>r.id));
      const beforeUrls=new Set((store.db.prepare('SELECT url FROM sightings').all()as any[]).map(r=>r.url));
      const result = await discoverSource(source,{...options,inventory,onJob:j=>{const id=store.upsert(j);if(id)enqueueJob(store,id);},saveInventory:(jobs,coverage)=>{if(!coverage.blocker)store.db.prepare('INSERT OR REPLACE INTO discovery_cache VALUES(?,?,?)').run(key,inventory?cached.created:Date.now(),JSON.stringify({jobs,coverage}));},progress:c=>{save(source.name,'running',c);if(c.phase!==phase){event(source.name,`${c.phase}: ${c.reviewed} records reviewed`);phase=c.phase;}options.progress?.(c);}}),ids:number[]=[];
      for(const job of result.jobs){
         const id=store.upsert(job);if(!id)continue;ids.push(id);
         enqueueJob(store,id);
      }
      const existingCanonical=new Set([...beforeIds].map(id=>store.resolveId(id)));
      const finalIds=new Set(ids.map(id=>store.resolveId(id)));
      result.coverage.added=[...finalIds].filter(id=>!existingCanonical.has(id)).length;
      result.coverage.updated=result.jobs.filter(j=>beforeUrls.has(canonical(j.originalUrl))).length;
      result.coverage.duplicates=ids.length-result.coverage.added-result.coverage.updated;
      result.coverage.merged=beforeIds.size+result.coverage.added-Number((store.db.prepare('SELECT count(*) n FROM jobs').get()as any).n);
      result.coverage.imported=ids.length;result.coverage.phase='finished';result.coverage.finished=now();
      save(source.name,result.status,result.coverage);store.health(source.name,result.status,result.message,ids.length);
      event(source.name,`${result.status}: ${result.coverage.reviewed} reviewed, ${result.coverage.matched} matched; ${result.coverage.added} added, ${result.coverage.updated} updated, ${result.coverage.duplicates} duplicates. ${result.coverage.stopReason}`);
      results.push({source:source.name,status:result.status,count:ids.length,message:result.message,coverage:result.coverage});
    } catch(e) {store.health(source.name,'error',String(e));event(source.name,`Collection error: ${String(e)}`);results.push({source:source.name,status:'error',count:0,message:String(e)});}
  }
  const status=results.every(r=>r.status==='healthy')?'complete':'partial';
  store.db.prepare('UPDATE collection_runs SET status=?,finished=? WHERE id=?').run(status,now(),runId);event('',`Collection ${status}; no model inference performed.`);
  return results;
  }catch(e){store.db.prepare("UPDATE collection_runs SET status='failed',finished=? WHERE id=?").run(now(),runId);event('',`Run failed: ${String(e)}`);throw e;}
}
export function collectionProgress(store:Store){
 const run=store.db.prepare('SELECT * FROM collection_runs ORDER BY id DESC LIMIT 1').get()as any;if(!run)return null;
 if(run.status==='running'){let alive=true;try{process.kill(run.pid,0);}catch(e:any){alive=e.code!=='ESRCH';}if(!alive){store.db.prepare("UPDATE collection_runs SET status='interrupted',finished=? WHERE id=?").run(now(),run.id);store.db.prepare("UPDATE source_runs SET status='interrupted' WHERE run_id=? AND status='running'").run(run.id);run.status='interrupted';}}
 return {...run,sources:store.db.prepare('SELECT source,status,coverage FROM source_runs WHERE run_id=?').all(run.id).map((r:any)=>({...r,coverage:JSON.parse(r.coverage)})),events:store.db.prepare('SELECT created,source,message FROM collection_events WHERE run_id=? ORDER BY id').all(run.id)};
}
export function startSearch(store:Store,source?:string,force=false){
 if(source&&!searchConfig().sites.some(s=>s.name===source&&s.enabled))throw new Error('Unknown enabled source');
 const release=acquireLock(store);let runId:number|undefined;
  void search(store,source,{force},id=>runId=id).catch(e=>console.error('Collection failed:',e.message)).finally(release);
 if(!runId){release();throw new Error('Collection did not start');}return {runId,status:'running'};
}
export function acquireLock(store: Store, name = 'daily'): () => void {
  store.db.exec('BEGIN IMMEDIATE');
  try {
    const existing = store.db.prepare('SELECT pid FROM locks WHERE name=?').get(name) as any;
    if (existing) {
      let alive = true; try { process.kill(existing.pid,0); } catch(e: any) { alive = e.code !== 'ESRCH'; }
      if (alive) throw new Error(`Run already active (PID ${existing.pid})`);
      store.db.prepare('DELETE FROM locks WHERE name=?').run(name);
      store.db.prepare("UPDATE runs SET status='interrupted',finished=?,detail='Recovered abandoned lock after dead process' WHERE status='running'").run(now());
    }
    store.db.prepare('INSERT INTO locks VALUES(?,?,?)').run(name,process.pid,now());
    store.db.exec('COMMIT');
  } catch(e) { store.db.exec('ROLLBACK'); throw e; }
  return () => { store.db.prepare('DELETE FROM locks WHERE name=? AND pid=?').run(name,process.pid); };
}
export async function exclusive<T>(store: Store, work: () => Promise<T>,name='daily'): Promise<T> {
  const release = acquireLock(store,name); try { return await work(); } finally { release(); }
}
export async function assess(store:Store,id:number,provider?:string){return exclusive(store,()=>performAssessment(store,id,provider),'inference');}
export async function daily(store: Store, dueOnly = false) {
  return exclusive(store,async () => {
    const latest = store.db.prepare("SELECT finished FROM runs WHERE status IN ('complete','partial') ORDER BY id DESC LIMIT 1").get() as any;
    const d = new Date(), cfg = runtime(), due = new Date(d); due.setHours(cfg.daily_hour,cfg.daily_minute,0,0);
    if (dueOnly && ((latest && new Date(latest.finished).toDateString() === d.toDateString()) || d < due)) return { skipped:'Not due or already completed today' };
    const id = Number(store.db.prepare("INSERT INTO runs(started,status) VALUES(?,'running')").run(now()).lastInsertRowid);
    try {
      await refreshContext(store,searchConfig().preferences);
      const sources = await search(store);
      enqueueAll(store);const assessments:any[]=[{status:'queued',message:'Persistent server worker processes all eligible jobs; no per-run ceiling.'}];
      const detail = {sources,assessments};
      const status = sources.some(s => s.status !== 'healthy') || assessments.some(a => a.status === 'pending') ? 'partial' : 'complete';
      store.db.prepare('UPDATE runs SET status=?,finished=?,detail=? WHERE id=?').run(status,now(),JSON.stringify(detail),id);
      return {id,status,...detail};
    } catch(e) { store.db.prepare("UPDATE runs SET status='failed',finished=?,detail=? WHERE id=?").run(now(),String(e),id); throw e; }
  });
}
export function readConfig(name: string) {
  if (!['search','candidate','providers','network','runtime'].includes(name)) throw new Error('Unknown configuration');
  return readFileSync(rootPath(`config/${name}.yaml`),'utf8');
}
export function saveConfig(name: string, content: string) {
  const schemas: Record<string,z.ZodType> = {search:searchSchema,candidate:candidateSchema,network:networkSchema,runtime: z.object({version:z.literal(1)}).passthrough()};
  if (name === 'providers') {
    const v = YAML.parse(content);
    // Full validation uses same schema as stage routing without touching disk first.
    providerConfigSchema.parse(v);
  } else if (name === 'runtime') runtimeConfigSchema.parse(YAML.parse(content));
  else if (schemas[name]) schemas[name].parse(YAML.parse(content)); else throw new Error('Unknown configuration');
  if (/\b(?:sk-[A-Za-z0-9]{12}|Bearer\s+\S+|https?:\/\/[^\s/]+:[^\s/]+@)/i.test(content)) throw new Error('Use local environment references, not secret values');
  const path = rootPath(`config/${name}.yaml`), backup = localPath(`config-${name}-${Date.now()}.yaml`);
  if (existsSync(path)) writeFileSync(backup,readFileSync(path),{mode:0o600});
  writeFileSync(path,content); return {saved:name};
}
import { providerSchema, runtimeSchema as runtimeConfigSchema } from './core.js';
const providerConfigSchema = z.object({version:z.literal(1),stages:z.object({assessment:z.string(),extraction:z.string()}),providers:z.record(z.string(),providerSchema)}).superRefine((c,ctx) => { for (const n of Object.values(c.stages)) if (!c.providers[n]) ctx.addIssue({code:'custom',message:`Unknown provider ${n}`}); });
export { importUrl, enrich, refreshContext };
