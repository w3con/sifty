import './isolate.js';
import test from 'node:test';
import assert from 'node:assert/strict';
import {Store} from '../src/store.js';
import {localPath} from '../src/core.js';
import {AssessmentWorker,assessmentEnabled,setAssessmentEnabled,enqueueJob,queueStatus} from '../src/queue.js';

function fixture(store:Store,n=1){return store.upsert({title:'Engineer',company:`Fixture ${n}`,originalUrl:`https://fixture.test/${n}`,source:'fixture',description:'Build reliable TypeScript services and maintain production software. '.repeat(4),descriptionStatus:'available fixture'});}
function deferred(){let resolve!:()=>void;const promise=new Promise<void>(r=>resolve=r);return {promise,resolve};}
test('enabled defaults true; validated explicit choice persists across Store/worker reconstruction independently of runtime',async()=>{
 const path=localPath('assessment-preference.sqlite');let store=new Store(path);const id=fixture(store);let calls=0,probes=0;
 assert.equal(assessmentEnabled(store),true);assert.equal(queueStatus(store).enabled,true);
 for(const invalid of [undefined,null,'false',0,{},[]])assert.throws(()=>setAssessmentEnabled(store,invalid),/boolean/);
 assert.equal(store.db.prepare("SELECT * FROM queue_state WHERE key='assessment_enabled'").get(),undefined);
 setAssessmentEnabled(store,false);store.db.prepare("INSERT OR REPLACE INTO queue_state VALUES('runtime',?)").run(JSON.stringify({paused:true,reason:'Independent provider state'}));
 enqueueJob(store,id,100,true);const before=store.db.prepare('SELECT * FROM assessment_queue').all();store.close();store=new Store(path);
 const worker=new AssessmentWorker(store,async()=>{calls++;return {} as any;},async()=>{probes++;return {available:true,reason:'fixture'};});
 await worker.tick();assert.equal(assessmentEnabled(store),false);assert.equal(calls,0);assert.equal(probes,0);assert.deepEqual(store.db.prepare('SELECT * FROM assessment_queue').all(),before);
 assert.equal(JSON.parse(String(queueStatus(store).runtime!.data)).reason,'Independent provider state');setAssessmentEnabled(store,true);await worker.tick();assert.equal(calls,1);assert.equal(queueStatus(store).counts.done,1);store.close();
 const reopened=new Store(path);assert.equal(assessmentEnabled(reopened),true);reopened.close();
});
test('off lets already-running inference finish and preserves remaining priority work',async()=>{
 const store=new Store(':memory:'),a=fixture(store,1),b=fixture(store,2),entered=deferred(),finish=deferred();let calls=0;
 enqueueJob(store,a,200);enqueueJob(store,b,100);
 const worker=new AssessmentWorker(store,async()=>{calls++;entered.resolve();await finish.promise;return {} as any;},async()=>({available:true,reason:'fixture'}));
 const ticking=worker.tick();await entered.promise;setAssessmentEnabled(store,false);assert.equal(queueStatus(store).counts.running,1);finish.resolve();await ticking;
 assert.equal(queueStatus(store).counts.done,1);assert.equal(queueStatus(store).counts.pending,1);await worker.tick();assert.equal(calls,1);assert.equal(store.db.prepare('SELECT * FROM locks').all().length,0);store.close();
});
test('off during asynchronous provider availability prevents launch and releases the inference lease',async()=>{
 const path=localPath('assessment-probe.sqlite'),store=new Store(path),other=new Store(path),probe=deferred(),ready=deferred();fixture(store);let calls=0;
 const worker=new AssessmentWorker(store,async()=>{calls++;return {} as any;},async()=>{probe.resolve();await ready.promise;return {available:true,reason:'fixture'};});
 const tick=worker.tick();await probe.promise;setAssessmentEnabled(other,false);ready.resolve();await tick;
 assert.equal(calls,0);assert.equal(queueStatus(store).counts.pending,1);assert.equal(store.db.prepare('SELECT attempts FROM assessment_queue').get()!.attempts,0);assert.equal(store.db.prepare('SELECT * FROM locks').all().length,0);
 setAssessmentEnabled(other,true);await worker.tick();assert.equal(calls,1);other.close();store.close();
});
test('resume honors existing provider retry and per-job backoff, then starts on an eligible tick',async()=>{
 const store=new Store(':memory:');fixture(store);let clock=1000000,calls=0,probes=0;
 const worker=new AssessmentWorker(store,async()=>{calls++;if(calls===1)throw new Error('Fixture inference failure');return {} as any;},async()=>{probes++;return {available:true,reason:'fixture'};},()=>clock);
 await worker.tick();assert.equal(calls,1);const runtime=queueStatus(store).runtime!.data;
 setAssessmentEnabled(store,false);clock+=1000;await worker.tick();setAssessmentEnabled(store,true);await worker.tick();assert.equal(calls,1);assert.equal(probes,1);assert.equal(queueStatus(store).runtime!.data,runtime);
 clock=JSON.parse(String(runtime)).retryAt+1;await worker.tick();assert.equal(calls,2);assert.equal(queueStatus(store).counts.done,1);store.close();
});
