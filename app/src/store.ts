import { DatabaseSync } from 'node:sqlite';
import { canonical, hash, localPath, normalize, now, type Job, type JobInput } from './core.js';
import { isWeb3, stageFor, statusFor,archiveRetentionMs,validAppliedAt } from './workflow.js';
import {initApplications,mergeApplicationJobs} from './application-store.js';
import {existsSync,unlinkSync} from 'node:fs';import {resolve,sep} from 'node:path';import {ROOT} from './core.js';

export {archiveRetentionMs};
export function retentionDue(appliedAt:unknown,clock=Date.now()){return validAppliedAt(appliedAt)&&clock-Date.parse(appliedAt)>=archiveRetentionMs;}
const listingKeys=(j:{originalUrl:string;applyUrl?:string|null})=>[hash(`url:${canonical(j.originalUrl)}`),...(j.applyUrl?[hash(`url:${canonical(j.applyUrl)}`)]:[])];

export class Store {
  db: DatabaseSync;
  constructor(path = localPath('jobs.sqlite')) {
    this.db = new DatabaseSync(path);
    this.db.exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;
      CREATE TABLE IF NOT EXISTS migrations(version INTEGER PRIMARY KEY);
      CREATE TABLE IF NOT EXISTS jobs(id INTEGER PRIMARY KEY, identity TEXT NOT NULL, title TEXT NOT NULL, company TEXT NOT NULL, original_url TEXT NOT NULL, apply_url TEXT, description TEXT NOT NULL, description_status TEXT NOT NULL, location TEXT, published_at TEXT, expires_at TEXT, status TEXT NOT NULL DEFAULT 'new', first_seen TEXT NOT NULL, last_seen TEXT NOT NULL, assessment TEXT, assessment_error TEXT);
      CREATE INDEX IF NOT EXISTS jobs_identity ON jobs(identity);
      CREATE INDEX IF NOT EXISTS jobs_apply ON jobs(apply_url);
      CREATE TABLE IF NOT EXISTS sightings(job_id INTEGER NOT NULL REFERENCES jobs(id), source TEXT NOT NULL, url TEXT NOT NULL UNIQUE, seen TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS health(source TEXT PRIMARY KEY, status TEXT NOT NULL, message TEXT NOT NULL, checked TEXT NOT NULL, count INTEGER NOT NULL);
      CREATE TABLE IF NOT EXISTS snapshots(id INTEGER PRIMARY KEY, job_id INTEGER REFERENCES jobs(id), captured TEXT NOT NULL, data TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS contexts(version TEXT PRIMARY KEY, created TEXT NOT NULL, data TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS context_refreshes(id INTEGER PRIMARY KEY,version TEXT NOT NULL,created TEXT NOT NULL,data TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS assessments(id INTEGER PRIMARY KEY, job_id INTEGER REFERENCES jobs(id), created TEXT NOT NULL, data TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS runs(id INTEGER PRIMARY KEY, started TEXT NOT NULL, finished TEXT, status TEXT NOT NULL, detail TEXT);
      CREATE TABLE IF NOT EXISTS locks(name TEXT PRIMARY KEY, pid INTEGER NOT NULL, acquired TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS job_aliases(old_id INTEGER PRIMARY KEY,new_id INTEGER NOT NULL);
      CREATE TABLE IF NOT EXISTS collection_runs(id INTEGER PRIMARY KEY,started TEXT NOT NULL,finished TEXT,status TEXT NOT NULL,pid INTEGER NOT NULL,mode TEXT NOT NULL,config_hash TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS source_runs(run_id INTEGER NOT NULL,source TEXT NOT NULL,status TEXT NOT NULL,coverage TEXT NOT NULL,PRIMARY KEY(run_id,source));
      CREATE TABLE IF NOT EXISTS collection_events(id INTEGER PRIMARY KEY,run_id INTEGER NOT NULL,created TEXT NOT NULL,source TEXT,message TEXT NOT NULL);
      INSERT OR IGNORE INTO migrations VALUES(1); PRAGMA user_version=1;`);
    const columns = this.db.prepare('PRAGMA table_info(jobs)').all() as any[];
    if (!columns.some(c=>c.name==='apply_status')) this.db.exec("ALTER TABLE jobs ADD COLUMN apply_status TEXT NOT NULL DEFAULT 'not resolved'; INSERT OR IGNORE INTO migrations VALUES(2); PRAGMA user_version=2;");
    if (!columns.some(c=>c.name==='tags')) this.db.exec("ALTER TABLE jobs ADD COLUMN tags TEXT NOT NULL DEFAULT '[]'");
    this.db.exec('INSERT OR IGNORE INTO migrations VALUES(2); INSERT OR IGNORE INTO migrations VALUES(3); INSERT OR IGNORE INTO migrations VALUES(4); INSERT OR IGNORE INTO migrations VALUES(5); PRAGMA user_version=5;');
    this.db.exec(`CREATE TABLE IF NOT EXISTS discovery_cache(key TEXT PRIMARY KEY,created INTEGER NOT NULL,data TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS assessment_queue(job_id INTEGER PRIMARY KEY, fingerprint TEXT NOT NULL, provider TEXT NOT NULL, status TEXT NOT NULL, priority INTEGER NOT NULL DEFAULT 0, attempts INTEGER NOT NULL DEFAULT 0, next_at INTEGER NOT NULL DEFAULT 0, started INTEGER, finished INTEGER, error TEXT, owner INTEGER);
      CREATE TABLE IF NOT EXISTS feedback(id TEXT PRIMARY KEY,job_id INTEGER NOT NULL,created TEXT NOT NULL,path TEXT NOT NULL,data TEXT NOT NULL);
      CREATE TABLE IF NOT EXISTS queue_state(key TEXT PRIMARY KEY,data TEXT NOT NULL);
      INSERT OR IGNORE INTO migrations VALUES(6); PRAGMA user_version=6;`);
    const queueColumns=this.db.prepare('PRAGMA table_info(assessment_queue)').all()as any[];
    if(!queueColumns.some(c=>c.name==='override_provider'))this.db.exec('ALTER TABLE assessment_queue ADD COLUMN override_provider TEXT; ALTER TABLE assessment_queue ADD COLUMN routing TEXT;');
    this.db.exec('INSERT OR IGNORE INTO migrations VALUES(7); PRAGMA user_version=7;');
    if(!columns.some(c=>c.name==='seen_at'))this.db.exec('ALTER TABLE jobs ADD COLUMN seen_at TEXT;');
    if(!columns.some(c=>c.name==='applied_at'))this.db.exec('ALTER TABLE jobs ADD COLUMN applied_at TEXT;');
    this.db.exec(`CREATE TABLE IF NOT EXISTS workflow_history(id INTEGER PRIMARY KEY,job_id INTEGER NOT NULL,created TEXT NOT NULL,from_stage TEXT NOT NULL,to_stage TEXT NOT NULL,confirmed INTEGER NOT NULL DEFAULT 0);
      INSERT OR IGNORE INTO migrations VALUES(8); PRAGMA user_version=8;`);
    initApplications(this);
    this.db.exec(`CREATE TABLE IF NOT EXISTS deleted_listings(hash TEXT PRIMARY KEY);
      CREATE TABLE IF NOT EXISTS id_sequence(name TEXT PRIMARY KEY,value INTEGER NOT NULL);
      INSERT INTO id_sequence(name,value) SELECT 'jobs',max(n) FROM (SELECT COALESCE(max(id),0) n FROM jobs UNION ALL SELECT COALESCE(max(old_id),0) n FROM job_aliases UNION ALL SELECT COALESCE(max(new_id),0) n FROM job_aliases) WHERE true ON CONFLICT(name) DO UPDATE SET value=max(value,excluded.value);
      INSERT OR IGNORE INTO migrations VALUES(10); PRAGMA user_version=10;`);
  }
  upsert(j: JobInput,options:{restore?:boolean}={}): number {
    const original = canonical(j.originalUrl), apply = j.applyUrl ? canonical(j.applyUrl) : null;
    const identity = `${normalize(j.company)}|${normalize(j.title)}`;
    this.db.exec('BEGIN IMMEDIATE');
    try {
      const keys=listingKeys(j);if(options.restore){for(const key of keys)this.db.prepare('DELETE FROM deleted_listings WHERE hash=?').run(key);}else if(keys.some(key=>this.db.prepare('SELECT hash FROM deleted_listings WHERE hash=?').get(key))){this.db.exec('COMMIT');return 0;}
      const sameHostPrefix=`${new URL(original).origin}/%`;
      const found = this.db.prepare(`SELECT id,status,seen_at,applied_at FROM jobs WHERE id IN (SELECT job_id FROM sightings WHERE url=?) OR (apply_url IS NOT NULL AND apply_url=?) OR (identity=? AND company != 'Unknown' AND original_url NOT LIKE ? AND (? IS NULL OR apply_url IS NULL OR apply_url=?)) ORDER BY id`).all(original, apply, identity,sameHostPrefix,apply,apply) as any[];
      const t = now(); let id = found[0]?.id as number | undefined;
      if (!id) {
        const next=Number(this.db.prepare("UPDATE id_sequence SET value=max(value,(SELECT COALESCE(max(old_id),0) FROM job_aliases),(SELECT COALESCE(max(id),0) FROM jobs))+1 WHERE name='jobs' RETURNING value").get()!.value);
        id = Number(this.db.prepare(`INSERT INTO jobs(id,identity,title,company,original_url,apply_url,description,description_status,location,published_at,expires_at,first_seen,last_seen) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`).run(next,identity, j.title, j.company, original, apply, j.description, j.descriptionStatus, j.location ?? '', j.publishedAt ?? '', j.expiresAt ?? '', t, t).lastInsertRowid);
      } else {
        // A detail fetch can reveal that two previously separate board records are the same role.
        for (const duplicate of found.slice(1)) {
          mergeApplicationJobs(this,duplicate.id,id);
          this.db.prepare('UPDATE jobs SET assessment=COALESCE(assessment,(SELECT assessment FROM jobs WHERE id=?)),assessment_error=COALESCE(assessment_error,(SELECT assessment_error FROM jobs WHERE id=?)) WHERE id=?').run(duplicate.id,duplicate.id,id);
          for (const table of ['sightings','snapshots','assessments','feedback','workflow_history']) this.db.prepare(`UPDATE ${table} SET job_id=? WHERE job_id=?`).run(id,duplicate.id);
          this.db.prepare('DELETE FROM assessment_queue WHERE job_id=?').run(duplicate.id);
          this.db.prepare('UPDATE job_aliases SET new_id=? WHERE new_id=?').run(id,duplicate.id);
          this.db.prepare('INSERT OR REPLACE INTO job_aliases VALUES(?,?)').run(duplicate.id,id);
          this.db.prepare('DELETE FROM jobs WHERE id=?').run(duplicate.id);
        }
        const status = found.some(r=>stageFor(r.status)==='applied') ? 'applied' : found.some(r=>stageFor(r.status)==='preselected') ? 'saved' : found.some(r=>stageFor(r.status)==='no') ? 'rejected' : 'new';
        this.db.prepare('UPDATE jobs SET seen_at=?,applied_at=? WHERE id=?').run(found.map(r=>r.seen_at).filter(Boolean).sort()[0]??null,status==='applied'?(found.find(r=>r.applied_at)?.applied_at??null):null,id);
        this.db.prepare("UPDATE jobs SET title=?, company=CASE WHEN ? != 'Unknown' THEN ? ELSE company END, identity=CASE WHEN ? != 'Unknown' THEN ? ELSE identity END, status=? WHERE id=?").run(j.title,j.company,j.company,j.company,identity,status,id);
        const freshDescription=j.description.length>=100&&!/metadata|pending|missing|unavailable/i.test(j.descriptionStatus);
        const previous=this.db.prepare('SELECT description,description_status FROM jobs WHERE id=?').get(id)as any;
        const previousReady=previous.description.length>=100&&!/metadata|pending|missing|unavailable/i.test(previous.description_status);
        const replace=freshDescription||(!previousReady&&j.description.length>=previous.description.length);
        this.db.prepare(`UPDATE jobs SET last_seen=?, apply_url=COALESCE(?,apply_url), description=CASE WHEN ? THEN ? ELSE description END, description_status=CASE WHEN ? THEN ? ELSE description_status END, expires_at=CASE WHEN ?!='' THEN ? ELSE expires_at END WHERE id=?`).run(t, apply,Number(replace), j.description,Number(replace), j.descriptionStatus, j.expiresAt ?? '', j.expiresAt ?? '', id);
      }
      this.db.prepare('INSERT INTO sightings VALUES(?,?,?,?) ON CONFLICT(url) DO UPDATE SET seen=excluded.seen').run(id, j.source, original, t);
      if (j.location) this.db.prepare('UPDATE jobs SET location=? WHERE id=?').run(j.location,id);
      if (j.tags?.length) this.db.prepare('UPDATE jobs SET tags=? WHERE id=?').run(JSON.stringify(j.tags),id);
      if (j.applyStatus?.startsWith('board intermediary') || j.applyStatus?.startsWith('restricted board')) this.db.prepare('UPDATE jobs SET apply_url=NULL WHERE id=?').run(id);
      if(j.applyStatus?.startsWith('Unresolved intermediary')){
        const current=this.db.prepare('SELECT apply_url FROM jobs WHERE id=?').get(id)as any;
        if(current?.apply_url&&/linkedin\.com|jobs\.solana\.com|jobs\.(electriccapital|multicoin|framework)/.test(new URL(current.apply_url).hostname))this.db.prepare('UPDATE jobs SET apply_url=NULL WHERE id=?').run(id);
      }
      if (j.applyStatus) this.db.prepare('UPDATE jobs SET apply_status=? WHERE id=?').run(j.applyStatus,id);
      this.db.prepare('INSERT INTO snapshots(job_id,captured,data) VALUES(?,?,?)').run(id, t, JSON.stringify(j));
      this.db.exec('COMMIT'); return id;
    } catch (e) { this.db.exec('ROLLBACK'); throw e; }
  }
  get(id: number): Job {
    id = this.resolveId(id);
    const r = this.db.prepare('SELECT * FROM jobs WHERE id=?').get(id) as any;
    if (!r) throw new Error(`Job ${id} not found`);
    const assessment = r.assessment ? JSON.parse(r.assessment) : null;
    if (assessment) { assessment.confirmedMismatches ??= []; assessment.candidateUnknowns ??= assessment.gaps ?? []; assessment.employerUnknowns ??= []; }
    if (assessment) assessment.stale = assessment.jobVersion !== hash(JSON.stringify({title:r.title,company:r.company,description:r.description,descriptionStatus:r.description_status,location:r.location}));
    const tags=JSON.parse(r.tags??'[]');
    return { id: r.id, title: r.title, company: r.company, tags, stage:stageFor(r.status),seen:!!r.seen_at,seenAt:r.seen_at,appliedAt:r.applied_at,isWeb3:isWeb3({title:r.title,description:r.description,tags}), originalUrl: r.original_url, applyUrl: r.apply_url, applyStatus:r.apply_status, description: r.description, descriptionStatus: r.description_status, location: r.location, publishedAt: r.published_at, expiresAt: r.expires_at, source: '', status: r.status, firstSeen: r.first_seen, lastSeen: r.last_seen, expired: !!r.expires_at && Date.parse(r.expires_at) < Date.now(), matchScore: r.match_score ?? 0, assessment, assessmentError: r.assessment_error, sources: this.db.prepare('SELECT source AS name,url,seen FROM sightings WHERE job_id=?').all(id) as any, blocked: false, blockedBy: '', stopped: false, stoppedBy: '' };
  }
  list(status = 'all', query = ''): Job[] {
    const stage=status==='all'?null:stageFor(status);
    return (this.db.prepare('SELECT id FROM jobs ORDER BY first_seen DESC').all() as any[]).map(r => this.get(r.id)).filter(j => (!stage||j.stage===stage)&&normalize(`${j.title} ${j.company} ${j.description}`).includes(normalize(query))).sort((a,b) => (b.assessment?.score ?? -1) - (a.assessment?.score ?? -1));
  }
  resolveId(id: number): number {const seen=new Set<number>();while(!seen.has(id)){seen.add(id);const row=this.db.prepare('SELECT new_id FROM job_aliases WHERE old_id=?').get(id);if(!row)return id;id=Number(row.new_id);}throw Error('Job alias cycle detected');}
  deleteJob(id:number){
    this.db.exec('BEGIN IMMEDIATE');const files=new Set<string>();let job:Job;
    try{
      job=this.get(id);id=job.id;if(job.stage==='triage')throw Error('Dismiss roles in Triage; permanent Delete is available outside Triage');
      const ids=new Set([id]);let changed=true;const aliases=this.db.prepare('SELECT old_id,new_id FROM job_aliases').all();while(changed){changed=false;for(const a of aliases)if(ids.has(Number(a.new_id))&&!ids.has(Number(a.old_id))){ids.add(Number(a.old_id));changed=true;}}
      // Synchronous transaction serializes this check with worker claims. Global slots
      // cover asynchronous collection/enrichment and provider availability probes too.
      for(const row of this.db.prepare("SELECT pid FROM locks WHERE name IN ('daily','inference')").all()){let alive=true;try{process.kill(Number(row.pid),0);}catch(e:any){alive=e.code!=='ESRCH';}if(alive)throw Error('Work is active. Wait for collection or assessment to finish, then delete again.');}
      for(const n of ids){if(this.db.prepare("SELECT job_id FROM assessment_queue WHERE job_id=? AND status='running'").get(n)||this.db.prepare("SELECT id FROM preparation_operations WHERE job_id=? AND (status='running' OR (status='cancelled' AND pid IS NOT NULL))").get(n))throw Error('Preparation or assessment is active. Cancel preparation or wait for it to finish, then delete again.');}
      const keys=new Set(listingKeys(job));for(const n of ids){
       for(const row of this.db.prepare('SELECT url FROM sightings WHERE job_id=?').all(n))keys.add(hash(`url:${canonical(String(row.url))}`));
       for(const row of this.db.prepare('SELECT data FROM snapshots WHERE job_id=?').all(n)){try{for(const key of listingKeys(JSON.parse(String(row.data))))keys.add(key);}catch{}}
       for(const row of this.db.prepare('SELECT data FROM questionnaire_snapshots WHERE job_id=?').all(n)){try{const log=JSON.parse(String(row.data)).technicalLog;if(log)files.add(resolve(ROOT,log));}catch{}}
       for(const row of this.db.prepare('SELECT id FROM preparation_operations WHERE job_id=?').all(n))files.add(localPath(`application-preparation-errors/${row.id}.json`));
      }for(const key of keys)this.db.prepare('INSERT OR IGNORE INTO deleted_listings VALUES(?)').run(key);
      for(const n of ids){for(const table of ['sightings','snapshots','assessments','assessment_queue','feedback','workflow_history','application_drafts','application_revisions','questionnaire_snapshots','preparation_operations'])this.db.prepare(`DELETE FROM ${table} WHERE job_id=?`).run(n);this.db.prepare('DELETE FROM job_aliases WHERE old_id=? OR new_id=?').run(n,n);}
      this.db.prepare("UPDATE id_sequence SET value=max(value,?) WHERE name='jobs'").run(Math.max(...ids));this.db.prepare('DELETE FROM jobs WHERE id=?').run(id);
      // Cached inventories are transient discovery data, not reusable candidate knowledge.
      for(const row of this.db.prepare('SELECT key,data FROM discovery_cache').all()){try{const cached=JSON.parse(String(row.data));if(Array.isArray(cached.jobs)){cached.jobs=cached.jobs.filter((j:any)=>!listingKeys(j).some(k=>keys.has(k)));this.db.prepare('UPDATE discovery_cache SET data=? WHERE key=?').run(JSON.stringify(cached),row.key!);}}catch{this.db.prepare('DELETE FROM discovery_cache WHERE key=?').run(row.key!);}}
      this.db.exec('COMMIT');
    }catch(e){this.db.exec('ROLLBACK');throw e;}
    const cleanupErrors:string[]=[];for(const file of files){if(![localPath('application-inspections')+sep,localPath('application-preparation-errors')+sep].some(prefix=>file.startsWith(prefix)))continue;try{if(existsSync(file))unlinkSync(file);}catch(e){cleanupErrors.push(String(e));}}if(cleanupErrors.length)console.error('Deleted role technical-log cleanup:',id,cleanupErrors);
    return {deleted:id!,cleanupErrors};
  }
  purgeArchive(clock=Date.now()){
    const deleted:number[]=[],skipped:{id:number;reason:string}[]=[];
    for(const row of this.db.prepare("SELECT id,applied_at FROM jobs WHERE status='applied'").all())if(retentionDue(row.applied_at,clock)){try{deleted.push(this.deleteJob(Number(row.id)).deleted);}catch(e){skipped.push({id:Number(row.id),reason:String(e)});}}
    return {deleted,skipped};
  }
  seen(id:number) {id=this.get(id).id;this.db.prepare('UPDATE jobs SET seen_at=COALESCE(seen_at,?) WHERE id=?').run(now(),id);return this.get(id);}
  status(id: number, status: string, confirmed=false) {
    const stage=stageFor(status),job=this.get(id);id=job.id;
    if(stage==='applied'&&job.stage!=='applied'&&(job.stage!=='preselected'||!confirmed))throw new Error('Applied requires a pre-selected role and explicit submission confirmation');
    if(stage===job.stage)return job;
    this.db.exec('BEGIN IMMEDIATE');
    try {
      this.db.prepare('UPDATE jobs SET status=?,applied_at=? WHERE id=?').run(statusFor(stage),stage==='applied'?now():null,id);
      this.db.prepare('INSERT INTO workflow_history(job_id,created,from_stage,to_stage,confirmed) VALUES(?,?,?,?,?)').run(id,now(),job.stage,stage,Number(confirmed));
      this.db.exec('COMMIT');return this.get(id);
    }catch(e){this.db.exec('ROLLBACK');throw e;}
  }
  health(source: string, status: string, message: string, count = 0) { this.db.prepare('INSERT INTO health VALUES(?,?,?,?,?) ON CONFLICT(source) DO UPDATE SET status=excluded.status,message=excluded.message,checked=excluded.checked,count=excluded.count').run(source,status,message,now(),count); }
  healthList() { return this.db.prepare('SELECT * FROM health ORDER BY source').all(); }
  context() { const r = (this.db.prepare('SELECT data FROM context_refreshes ORDER BY id DESC LIMIT 1').get() ?? this.db.prepare('SELECT data FROM contexts ORDER BY created DESC LIMIT 1').get()) as any; return r ? JSON.parse(r.data) : null; }
  saveContext(c: any) {
    this.db.prepare('INSERT OR IGNORE INTO contexts VALUES(?,?,?)').run(c.version,now(),JSON.stringify(c));
    this.db.prepare('INSERT INTO context_refreshes(version,created,data) VALUES(?,?,?)').run(c.version,now(),JSON.stringify(c));
  }
  assessment(id: number, data: any) {
    id=this.get(id).id;
    this.db.prepare('INSERT INTO assessments(job_id,created,data) VALUES(?,?,?)').run(id,now(),JSON.stringify(data));
    this.db.prepare('UPDATE jobs SET assessment=?, assessment_error=NULL WHERE id=?').run(JSON.stringify(data),id);
  }
  assessmentFailure(id: number, message: string,metadata:Record<string,unknown>={}) {
    id=this.get(id).id;
    this.db.prepare('INSERT INTO assessments(job_id,created,data) VALUES(?,?,?)').run(id,now(),JSON.stringify({status:'failed',error:message,...metadata}));
    this.db.prepare('UPDATE jobs SET assessment_error=? WHERE id=?').run(message,id);
  }
  close() { this.db.close(); }
}
