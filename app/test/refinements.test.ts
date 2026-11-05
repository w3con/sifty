import './isolate.js';
import {test} from 'node:test';
import assert from 'node:assert/strict';
import {mkdtempSync,readFileSync,rmSync} from 'node:fs';
import {resolve} from 'node:path';
import { Store } from '../src/store.js';
import { localPath, now, searchConfig, type JobInput } from '../src/core.js';
import {AssessmentWorker,enqueueAll,enqueueJob,fingerprint,queueStatus} from '../src/queue.js';
import {DomainTraffic,RequestCache,httpError} from '../src/traffic.js';
import {discoverSource,emptyCoverage} from '../src/discovery.js';
import {saveFeedback,feedbackRecords,bindFeedback,contextForRole} from '../src/feedback.js';
import {rootPath} from '../src/core.js';
import {filterCatalogue,emptySelection,termMatch,assessmentStatus} from '../web/filters.js';
import {assess} from '../src/providers.js';
import {DatabaseSync} from 'node:sqlite';
const input=(n=1,description='Build reliable software and TypeScript systems. '.repeat(5)):JobInput=>({title:'Software Engineer',company:`Fixture ${n}`,originalUrl:`https://example.test/jobs/${n}`,description,descriptionStatus:'available public description',source:'Fixture'});
function context(s:Store){s.saveContext({version:'fixture',createdAt:now(),preferences:searchConfig().preferences,reviewNotes:[],sources:[{id:'fixture',reference:'fixture',retrievedAt:now(),hash:'fixture',text:'Docker experienced. Built TypeScript systems.\n\nLed distributed teams.',status:'available'}]});}
test('pre-refinement SQLite shape migrates without changing job IDs, statuses or assessment/history bytes',()=>{
 const dir=mkdtempSync(localPath('migration-')),path=resolve(dir,'jobs.sqlite');let s=new Store(path);const id=s.upsert(input());s.status(id,'saved');s.assessment(id,{score:65,summary:'Preserve old assessment',evidence:[],gaps:['Unknown skill']});const history=s.db.prepare('SELECT * FROM assessments').all();s.close();const legacy=new DatabaseSync(path);legacy.exec('DROP TABLE assessment_queue; DROP TABLE discovery_cache; DROP TABLE feedback; DROP TABLE queue_state; DROP TABLE deleted_listings; DROP TABLE id_sequence; ALTER TABLE jobs DROP COLUMN tags; DELETE FROM migrations WHERE version>5; PRAGMA user_version=5;');legacy.close();s=new Store(path);assert.equal(s.get(id).id,id);assert.equal(s.get(id).status,'saved');assert.deepEqual(s.db.prepare('SELECT * FROM assessments').all(),history);assert.deepEqual(s.get(id).assessment.confirmedMismatches,[]);assert.deepEqual(s.get(id).tags,[]);assert.equal((s.db.prepare('PRAGMA user_version').get()as any).user_version,10);s.close();rmSync(dir,{recursive:true,force:true});
});
test('queue persists, deduplicates, skips completed current work and invalidates actual changed inputs',async()=>{
 const dir=mkdtempSync(localPath('queue-')),path=resolve(dir,'jobs.sqlite');let s=new Store(path);context(s);const id=s.upsert(input());
 enqueueJob(s,id);enqueueJob(s,id);assert.equal(queueStatus(s).counts.pending,1);const key=fingerprint(s,id);s.close();s=new Store(path);
 let calls=0;const worker=new AssessmentWorker(s,async()=>{calls++;return {} as any;},async()=>({available:true,reason:'fixture'}));
 await worker.tick();assert.equal(calls,1);enqueueAll(s);await worker.tick();assert.equal(calls,1);assert.equal(queueStatus(s).counts.done,1);
 const changed='A shorter but fully available changed job description. '.repeat(3);s.upsert(input(1,changed));assert.equal(s.get(id).description,changed);enqueueAll(s);assert.notEqual(fingerprint(s,id),key);await worker.tick();assert.equal(calls,2);
 const c=s.context();c.version='new-candidate-evidence';s.saveContext(c);enqueueAll(s);assert.equal(queueStatus(s).counts.pending,1);
 worker.stop();s.close();rmSync(dir,{recursive:true,force:true});
});
test('interrupted inference recovers; provider pause and persistent retry do not loop or spend',async()=>{
 const s=new Store(':memory:');context(s);const id=s.upsert(input());enqueueJob(s,id);s.db.prepare("UPDATE assessment_queue SET status='running',owner=99999999").run();let clock=100000,calls=0,checks=0;
 const worker=new AssessmentWorker(s,async()=>{calls++;throw new Error('fixture rate limit');},async()=>{checks++;return {available:false,reason:'no installed local model'};},()=>clock);
 assert.equal(queueStatus(s).counts.pending,1);clock+=6000;await worker.tick();await worker.tick();assert.equal(checks,1);assert.equal(calls,0);assert.match(queueStatus(s).runtime?.data as string,/no installed local model/);worker.stop();
 clock+=61000;const failed=new AssessmentWorker(s,async()=>{calls++;throw new Error('fixture 429 Retry later');},async()=>({available:true,reason:'fixture'}),()=>clock);await failed.tick();assert.equal(calls,1);assert.equal(queueStatus(s).counts.failed,1);const row=queueStatus(s).rows[0];assert.ok(row.next_at>clock);enqueueJob(s,id,100,true);assert.equal(queueStatus(s).rows[0].next_at,row.next_at);await failed.tick();assert.equal(calls,1);assert.match(queueStatus(s).rows[0].error,/fixture 429/);failed.stop();s.close();
});
test('single inference, selected-role priority and eventual completion beyond ten jobs without collection lock',async()=>{
 const s=new Store(':memory:');context(s);const ids=Array.from({length:13},(_,i)=>s.upsert(input(i+1)));enqueueAll(s);enqueueJob(s,ids[12],100);s.db.prepare("INSERT INTO locks VALUES('daily',?,?)").run(process.pid,now());
 const seen:number[]=[];let active=0,max=0,release!:()=>void;const gate=new Promise<void>(r=>release=r);
 const worker=new AssessmentWorker(s,async(_s,id)=>{active++;max=Math.max(max,active);seen.push(id);if(seen.length===1)await gate;active--;return {}as any;},async()=>({available:true,reason:'fixture'}));
 const first=worker.tick();await new Promise(r=>setImmediate(r));await worker.tick();assert.equal(seen[0],ids[12]);assert.equal(seen.length,1);release();await first;for(let i=0;i<13;i++)await worker.tick();assert.equal(seen.length,13);assert.equal(max,1);assert.equal(queueStatus(s).counts.done,13);worker.stop();s.close();
});
test('missing descriptions are not inference pending; legacy gaps never become confirmed mismatches',()=>{
 const s=new Store(':memory:');context(s);const id=s.upsert({...input(),description:'TypeScript tags',descriptionStatus:'metadata; pending detail'});enqueueJob(s,id);assert.equal(queueStatus(s).counts['missing-description'],1);
 s.assessment(id,{score:50,summary:'Legacy',evidence:[],gaps:['Kubernetes not mentioned in CV']});assert.deepEqual(s.get(id).assessment.confirmedMismatches,[]);assert.deepEqual(s.get(id).assessment.candidateUnknowns,['Kubernetes not mentioned in CV']);s.close();
});
test('fresh assessment fingerprints skip inference and unsupported CV absence becomes a question, not a confirmed mismatch',async()=>{
 const s=new Store(':memory:');context(s);const id=s.upsert(input());
 const result=await assess(s,id,'openai',async()=>({score:40,summary:'Fixture',evidence:[],confirmedMismatches:[{reason:'Candidate lacks Kubernetes experience',kind:'explicit-absence',evidenceId:'E1',jobRequirement:'Build reliable software'}],candidateUnknowns:[],employerUnknowns:['Salary unspecified']}));
 assert.equal(result.confirmedMismatches.length,0);assert.match(result.candidateUnknowns.join(' '),/Kubernetes/);assert.deepEqual(result.employerUnknowns,['Salary unspecified']);assert.equal(result.inputFingerprint,fingerprint(s,id,'openai'));enqueueJob(s,id,0,false,'openai');assert.equal(queueStatus(s).counts.done,1);
 await assert.rejects(()=>assess(s,id,'openai',async()=>({score:40,summary:'Fixture',evidence:[],confirmedMismatches:[{reason:'Location conflict',kind:'documented-incompatibility',evidenceId:'E1',jobRequirement:'Invented mandatory country'}]})),/exact vacancy quote/);s.close();
});
test('domain requests serialize across aliases with 3–6s pacing, Retry-After, restart cooldown and independent hosts',async()=>{
 let clock=100000,active=0,max=0;const state=new Map<string,any>(),starts:number[]=[],sleep=async(ms:number)=>{clock+=ms;};
 const create=()=>new DomainTraffic(h=>state.get(h),(h,s)=>state.set(h,{...s}),()=>clock,sleep,()=>0.5);
 let traffic=create();await Promise.all([1,2,3].map(()=>traffic.run('https://api.getro.com/jobs',async()=>{active++;max=Math.max(max,active);starts.push(clock);await Promise.resolve();active--;return 'ok';})));
 assert.equal(max,1);assert.deepEqual(starts,[100000,104500,109000]);
 await assert.rejects(()=>traffic.run('https://api.getro.com/jobs',async()=>{throw httpError(429,'120');}));traffic=create();await assert.rejects(()=>traffic.run('https://api.getro.com/other',async()=>{assert.fail('cooldown bypass');}),/paused until/);
 assert.equal(await traffic.run('https://other.test/jobs',async()=>true),true);clock+=121000;await traffic.run('https://api.getro.com/jobs',async()=>true);
 await assert.rejects(()=>traffic.run('https://blocked.test/jobs',async()=>{throw httpError(403);}));assert.ok(state.get('blocked.test').cooldown-clock>=24*3600000);
});
test('three-hour fingerprint cache reuses unchanged queries, only fetches added keys and independently revalidates details at 24h',async()=>{
 let clock=1000,calls=0;const data=new Map<string,any>(),cache=new RequestCache(k=>data.get(k),(k,v)=>data.set(k,v),()=>clock);
 const load=async()=>{calls++;return {text:'inventory',status:200,url:'https://fixture.test'};};
 await cache.get('source/query=a',3*3600000,load);await cache.get('source/query=a',3*3600000,load);await cache.get('source/query=b',3*3600000,load);assert.equal(calls,2);
 await cache.get('detail/1',24*3600000,async()=>({text:'full body',status:200,etag:'v1',modified:'yesterday'}));clock+=4*3600000;await cache.get('source/query=a',3*3600000,load);assert.equal(calls,3);
 const old=await cache.get('detail/1',24*3600000,async()=>assert.fail('details still fresh'));assert.equal(old.text,'full body');clock+=21*3600000;
 const revalidated=await cache.get('detail/1',24*3600000,async headers=>{assert.equal(headers['if-none-match'],'v1');assert.equal(headers['if-modified-since'],'yesterday');return {status:304};},false,true);assert.equal(revalidated.text,'full body');
 await cache.get('source/query=b',3*3600000,load,true);assert.equal(calls,4);
});
test('cached finite inventory resumes description-only matches even with a non-technical title, and reports deferred/failed honestly',async()=>{
 const source={name:'Braintrust fixture',url:'https://www.usebraintrust.com/jobs',enabled:true};let inventory:any;let listings=0,details=0;
 const get=async(url:string)=>{if(url.endsWith('/api/jobs/')){listings++;return {url,status:200,text:JSON.stringify({count:3,next:null,results:[1,2,3].map(id=>({id,title:'Research colleague',main_skills:[],employer:{name:'Fixture'}}))})};}details++;return {url,status:200,text:JSON.stringify({description:'Work on Solidity and distributed systems. '.repeat(5)})};};
 const a=await discoverSource(source,{get,limits:{max_detail_requests:1},saveInventory:(jobs,coverage)=>inventory={jobs,coverage}});assert.equal(a.coverage.reviewed,3);assert.equal(a.coverage.descriptionUnreviewed,2);assert.equal(a.status,'bounded');assert.equal(a.jobs.length,1);
 const b=await discoverSource(source,{get,inventory,limits:{max_detail_requests:3},saveInventory:(jobs,coverage)=>inventory={jobs,coverage}});assert.equal(listings,1);assert.equal(details,3);assert.equal(b.jobs.length,3);assert.equal(b.coverage.descriptionUnreviewed,0);
 const c=await discoverSource(source,{get:async()=>{assert.fail('unchanged full inventory must not fetch');return {}as any;},inventory});assert.equal(c.jobs.length,3);assert.equal(c.coverage.networkRequests,0);
});
test('feedback is durable verbatim without a model, grounded, scoped, conflict-preserving and additive on revision',()=>{
 const s=new Store(':memory:');context(s);const id=s.upsert(input()),other=s.upsert(input(2));const raw='Kubernetes not known, Docker experienced, relocation Canada yes / USA no.\nНе приписывай мне опыт.';
 const saved=saveFeedback(s,id,raw,'general');assert.equal(saved.text,raw);assert.equal(saved.facts.length,4);assert.ok(readFileSync(rootPath(saved.path),'utf8').includes(raw));assert.ok(!readFileSync(rootPath(saved.path),'utf8').includes('\nai:'));
 const conflict=saveFeedback(s,id,'Kubernetes experienced','general');assert.equal(conflict.conflicts.length,1);assert.equal(feedbackRecords().find(r=>r.id===saved.id).text,raw);
 const before=fingerprint(s,other);const scoped=saveFeedback(s,id,'Only this role: relocation Canada no','job');assert.equal(fingerprint(s,other),before);const role=contextForRole(bindFeedback(s.context()),other);assert.ok(!role.sources.some(x=>x.id===scoped.path));
 const revised=saveFeedback(s,id,'Only this role: clarification','job',scoped.id);assert.equal(revised.supersedes,scoped.id);assert.ok(readFileSync(rootPath(scoped.path),'utf8').includes(scoped.text));s.close();
});
test('local filters use aliases, OR within AND across groups, tags and truthful assessed states',()=>{
 const s=new Store(':memory:');const a=s.upsert({...input(),title:'Chief Technology Officer',tags:['Kubernetes'],description:'Lead TypeScript services. '.repeat(6)}),b=s.upsert({...input(2),title:'Software Engineer',tags:['Docker']});const jobs=s.list();
 assert.ok(termMatch('Chief Technology Officer','CTO'));assert.ok(!termMatch('chair','AI'));
 assert.equal(filterCatalogue(jobs,emptySelection()).length,2);assert.deepEqual(filterCatalogue(jobs,{...emptySelection(),titles:['CTO','Architect'],keywords:['Kubernetes'],sources:['Fixture']}).map(j=>j.id),[a]);assert.equal(filterCatalogue(jobs,{...emptySelection(),titles:['CTO'],keywords:['Docker']}).length,0);
 assert.equal(assessmentStatus(s.get(b),[{job_id:b,status:'failed'}]),'failed');assert.equal(filterCatalogue(jobs,{...emptySelection(),assessment:['failed']},[{job_id:b,status:'failed'}]).length,1);s.close();
});
