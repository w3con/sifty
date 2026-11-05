import './isolate.js';
import test from 'node:test';import assert from 'node:assert/strict';import {existsSync,readFileSync} from 'node:fs';
import {Store} from '../src/store.js';import {PreparationWorker,cancelPreparation,queuePreparation,preparationState} from '../src/applications.js';import {addManualQuestion,applicationDraft,updateApplication} from '../src/application-store.js';import {generateApplication} from '../src/application-generation.js';import {runProcess} from '../src/providers.js';import {localPath,now,searchConfig} from '../src/core.js';
const delay=(ms:number)=>new Promise(r=>setTimeout(r,ms));
function fixture(){const store=new Store(':memory:'),id=store.upsert({title:'Engineer',company:'Fixture',originalUrl:'https://fixture.test/job',description:'Engineering',descriptionStatus:'fixture',source:'fixture'});store.status(id,'saved');addManualQuestion(store,id,'Experience?');return {store,id};}
test('queued cancellation never starts work; cancelled in-flight inspection cannot write a late snapshot or release another inference lock',async()=>{
 const {store,id}=fixture();queuePreparation(store,id,{});cancelPreparation(store,id);let inspected=0,finish!:Function;const gate=new Promise<any>(r=>finish=r);
 const worker=new PreparationWorker(store,async()=>{inspected++;return gate;});await worker.tick();assert.equal(inspected,0);
 queuePreparation(store,id,{});const running=worker.tick();await delay(5);const before=applicationDraft(store,id);cancelPreparation(store,id);assert.equal(preparationState(store,id).operations[0].status,'cancelled');
 const next=queuePreparation(store,id,{});await new PreparationWorker(store).tick();assert.equal(preparationState(store,id).operations[0].id,next!.id);assert.equal(preparationState(store,id).operations[0].status,'queued');
 finish({url:'https://fixture.test',fields:[],status:'partial'});await running;assert.deepEqual(applicationDraft(store,id),before);assert.equal(store.db.prepare('SELECT count(*) n FROM questionnaire_snapshots').get()!.n,0);worker.stop();store.close();
});
test('cancelled inference rejects late answers while preserving owner edit and shared inference ownership',async()=>{
 const {store,id}=fixture();store.saveContext({version:'fixture',createdAt:now(),preferences:searchConfig().preferences,reviewNotes:[],sources:[{id:'cv',reference:'fixture',retrievedAt:now(),hash:'fixture',status:'available',text:'Built services.'}]});
 let finish!:Function,started!:Function;const gate=new Promise<any>(r=>finish=r),ready=new Promise<void>(r=>started=r);
 const worker=new PreparationWorker(store,undefined,(s,n,p,f,_inference,signal)=>generateApplication(s,n,p,f,async()=>{started();return gate;},signal));queuePreparation(store,id,{inspect:false});const running=worker.tick();await ready;
 const field=applicationDraft(store,id).fields[0];updateApplication(store,id,{fieldId:field.id,revision:field.revision,value:'Owner edit'});cancelPreparation(store,id);assert.ok(store.db.prepare("SELECT * FROM locks WHERE name='inference'").get());
 finish({language:'en',selectedFile:'cv.pdf',answers:[{id:field.id,value:'Late model',unknowns:[],evidenceIds:['E1']}]});await running;assert.equal(applicationDraft(store,id).fields[0].value,'Owner edit');assert.equal(applicationDraft(store,id).provenance,null);assert.equal(store.db.prepare("SELECT * FROM locks WHERE name='inference'").get(),undefined);assert.equal(preparationState(store,id).operations[0].status,'cancelled');store.close();
});
test('provider cancellation terminates owned child group and preserves unrelated processes/files',async()=>{
 const controller=new AbortController(),marker=localPath('child-started.json');
 const script=`const fs=require('fs'),{spawn}=require('child_process');const child=spawn(process.execPath,['-e','setInterval(()=>{},1000)']);fs.writeFileSync(process.argv[1],JSON.stringify({parent:process.pid,child:child.pid}));setInterval(()=>{},1000)`;
 const work=runProcess(process.execPath,['-e',script,marker],'',10000,process.env,localPath(),[0],controller.signal);const rejected=assert.rejects(work,/cancelled/);
 for(let n=0;n<100&&!existsSync(marker);n++)await delay(10);assert.ok(existsSync(marker));const pids=JSON.parse(readFileSync(marker,'utf8'));controller.abort();await rejected;
 for(let n=0;n<100;n++){let alive=false;try{process.kill(pids.child,0);alive=true;}catch{}if(!alive)break;await delay(10);}
 assert.throws(()=>process.kill(pids.parent,0));assert.throws(()=>process.kill(pids.child,0));assert.doesNotThrow(()=>process.kill(process.pid,0));assert.ok(existsSync(marker));
});
