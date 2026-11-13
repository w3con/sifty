// Focused verification for Application tab state rendering: spinner (1.1), per-question
// answer states (1.2), non-question outcomes (1.3), and provider selection (1.4).
import './isolate.js';import assert from 'node:assert/strict';import {chromium} from 'playwright';
import {Store} from '../src/store.js';import {serve} from '../src/server.js';import {now,searchConfig} from '../src/core.js';
import {inspectIntoDraft,applicationDraft,questionSchema,writeDraft} from '../src/application-store.js';

const store=new Store();
store.saveContext({version:'tab-states',createdAt:now(),preferences:searchConfig().preferences,reviewNotes:[],sources:[{id:'cv.pdf',reference:'cv.pdf',retrievedAt:now(),hash:'fix',status:'available',text:'Fixture source.'}]});

// Job for step 1.1/1.2: Pre-selected with no prior draft; auto-prepare will fire.
const spinnerId=store.upsert({title:'Spinner fixture',company:'Fixture',originalUrl:'https://f.test/spinner',applyUrl:'https://f.test/spinner',description:'Spinner',descriptionStatus:'fixture',source:'fixture'});
store.status(spinnerId,'preselected');

// Step 1.3 jobs (all Pre-selected except archiveId):
// Sign-in required
const loginId=store.upsert({title:'Login fixture',company:'Fixture',originalUrl:'https://f.test/login',applyUrl:'https://f.test/login',description:'Login',descriptionStatus:'fixture',source:'fixture'});
store.status(loginId,'preselected');
inspectIntoDraft(store,loginId,{url:'https://f.test/login',capturedAt:now(),language:'en',status:'login-required',family:'Login fixture',gaps:['Sign in required'],fields:[]});
store.db.prepare("INSERT INTO preparation_operations(job_id,status,created,updated,request,progress) VALUES(?,'done',?,?,'{\"inspect\":false}','Done')").run(loginId,now(),now());

// No resolved application form
const noFormId=store.upsert({title:'No-form fixture',company:'Fixture',originalUrl:'https://f.test/listing',description:'No form',descriptionStatus:'fixture',source:'fixture'});
store.status(noFormId,'preselected');
store.db.prepare("INSERT INTO preparation_operations(job_id,status,created,updated,request,progress) VALUES(?,'done',?,?,'{\"inspect\":false}','no resolved application form; apply via the original listing')").run(noFormId,now(),now());

// Backed-off
const backedId=store.upsert({title:'Backed-off fixture',company:'Fixture',originalUrl:'https://f.test/backed',applyUrl:'https://f.test/backed',description:'Backed',descriptionStatus:'fixture',source:'fixture'});
store.status(backedId,'preselected');
writeDraft(store,{...applicationDraft(store,backedId),questionnaire:{url:'https://f.test/backed',capturedAt:now(),language:'en',status:'backed-off' as any,family:'Backed-off fixture',gaps:['The site asked to slow down; paused until 14:00 UTC'],fields:[]}},'Backed-off fixture');
store.db.prepare("INSERT INTO preparation_operations(job_id,status,created,updated,request,progress) VALUES(?,'done',?,?,'{\"inspect\":false}','backed-off')").run(backedId,now(),now());

// Failed
const failedId=store.upsert({title:'Failed fixture',company:'Fixture',originalUrl:'https://f.test/failed',applyUrl:'https://f.test/failed',description:'Failed',descriptionStatus:'fixture',source:'fixture'});
store.status(failedId,'preselected');
store.db.prepare("INSERT INTO preparation_operations(job_id,status,created,updated,request,progress,error) VALUES(?,'failed',?,?,'{\"inspect\":false}','Preparation failed; saved answers preserved','Connection timeout navigating to the application page')").run(failedId,now(),now());

// Not pre-selected: applied (Archive) job — must go through pre-selected first per workflow rules.
const archiveId=store.upsert({title:'Archive fixture',company:'Fixture',originalUrl:'https://f.test/archive',description:'Archive',descriptionStatus:'fixture',source:'fixture'});
store.status(archiveId,'preselected');store.status(archiveId,'applied',true);

// Step 1.4: a fresh Pre-selected job for the provider check (no existing operations)
const provCheckId=store.upsert({title:'Provider check fixture',company:'Fixture',originalUrl:'https://f.test/prov',applyUrl:'https://f.test/prov',description:'Provider check',descriptionStatus:'fixture',source:'fixture'});
store.status(provCheckId,'preselected');
// Step 1.4 no-stage check: fresh Pre-selected job with no draft and no operations.
// The mock stage-providers returns preparation='' causing the config prompt to appear.
const noStageId=store.upsert({title:'No-stage fixture',company:'Fixture',originalUrl:'https://f.test/nostage',applyUrl:'https://f.test/nostage',description:'No stage',descriptionStatus:'fixture',source:'fixture'});
store.status(noStageId,'preselected');
// Pre-load draft with questions so auto-prepare does not fire (revision=0 would trigger it).
// We inject a draft with revision to prevent auto-prepare.
const q1=questionSchema.parse({id:'qa',label:'Sample question A',type:'textarea',required:false,maxLength:300,sourceUrl:'https://f.test/prov',origin:'site',signature:'s1'});
const q2=questionSchema.parse({id:'qb',label:'Sample question B',type:'select',options:['Yes','No'],required:false,maxLength:null,sourceUrl:'https://f.test/prov',origin:'site',signature:'s2'});
writeDraft(store,{jobId:provCheckId,revision:0,fields:[{...q1,value:'',revision:0,edited:false,unknowns:[],evidence:[]},{...q2,value:'',revision:0,edited:false,unknowns:[],evidence:[]}],selectedFile:'cv.pdf',questionnaire:null,provenance:null,notes:[]},'Provider check fixture');
store.db.prepare("INSERT INTO preparation_operations(job_id,status,created,updated,request,progress) VALUES(?,'done',?,?,'{\"inspect\":false}','Questionnaire ready for review; nothing submitted')").run(provCheckId,now(),now());

let inspectionDelay=3000;
const inspectionFields=[{id:'q1',label:'Why this role?',type:'textarea' as const,required:true,maxLength:500,sourceUrl:'https://f.test/spinner',origin:'site' as const,signature:'s1',options:[],accept:'',language:'en',controlName:'',labelStatus:undefined as any,instructions:undefined as any},{id:'q2',label:'Salary expectation?',type:'textarea' as const,required:false,maxLength:200,sourceUrl:'https://f.test/spinner',origin:'site' as const,signature:'s2',options:[],accept:'',language:'en',controlName:'',labelStatus:undefined as any,instructions:undefined as any}];
const server=await serve(store,4319,{localEvidence:false,availability:async()=>({available:false,reason:'Fixture unavailable'}),preparation:async(_url,deps)=>{await new Promise<void>((resolve,reject)=>{const timer=setTimeout(resolve,inspectionDelay);deps?.signal?.addEventListener('abort',()=>{clearTimeout(timer);reject(Error('Cancelled'));},{once:true});});return {url:'https://f.test/spinner',capturedAt:now(),language:'en',status:'partial' as const,family:'Fixture',gaps:[],fields:inspectionFields};},preparationGenerate:()=>{throw Error('Generate not needed in this test');}});

const browser=await chromium.launch({headless:true}),ctx=await browser.newContext({viewport:{width:1440,height:900}}),page=await ctx.newPage();
const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));

async function openApp(jobId:number):Promise<void>{
 // Job row must be visible in the current view before calling.
 await page.locator(`.job-row[data-job-id="${jobId}"]`).click();
 // Application button may be enabled or disabled depending on stage.
 await page.getByRole('button',{name:'Application',exact:true}).click();
 await page.getByRole('region',{name:'Application draft',exact:true}).waitFor({timeout:6000});
}

try{
 await page.goto('http://127.0.0.1:4319');
 await page.getByRole('button',{name:/^Pre-selected/}).click();
 await page.locator('.job-row').first().waitFor({timeout:8000});

 // ── Step 1.1: indeterminate spinner while form is being read with no questions ──
 await page.locator(`.job-row[data-job-id="${spinnerId}"]`).click();
 await page.getByRole('button',{name:'Application',exact:true}).click();
 await page.getByRole('region',{name:'Application draft',exact:true}).waitFor({timeout:6000});
 // Spinner must appear — inspection takes 3 s so it stays up long enough to verify.
 await page.locator('[role="status"]').filter({hasText:'Reading the application form'}).waitFor({timeout:10000});
 // No raw "queued:" / "running:" status text must appear.
 assert.equal(await page.locator('[role="status"]').filter({hasText:/queued:|running:/}).count(),0,'no raw operation status text');
 // No percentage or fraction progress text.
 assert.equal(await page.locator('[role="status"]').filter({hasText:/\d+%|\d+\/\d+/}).count(),0,'no percentage text');
 // The old generic banner must not appear.
 assert.equal(await page.getByText('No prepared answers yet',{exact:false}).count(),0,'no generic banner during spinner');
 console.error('Step 1.1 passed');

 // Let inspection fail fast so the tab settles (generate throws for spinnerId).
 inspectionDelay=0;
 // Wait for the question fields to appear (inspection writes them before generate throws).
 await page.locator('.draft-field').first().waitFor({timeout:12000});

 // ── Step 1.2: per-question "No answer yet" when fields are known but empty ──
 const fields=page.locator('.draft-field');
 await page.waitForFunction(()=>document.querySelectorAll('.draft-field').length>=2,undefined,{timeout:6000});
 assert.equal(await fields.count(),2,'two question fields visible');
 // Every field must show its answer state.
 const answerStates=await page.locator('.answer-state').count();
 assert.equal(answerStates,2,'each field shows "No answer yet" state');
 // The old generic banner must not appear anywhere.
 assert.equal(await page.getByText('No prepared answers yet',{exact:false}).count(),0,'generic banner absent');
 console.error('Step 1.2 passed');

 // ── Step 1.3: distinct states for each non-question outcome ──────────────────
 // 1a. Sign-in required
 await page.locator(`.job-row[data-job-id="${loginId}"]`).waitFor({timeout:3000});
 await openApp(loginId);
 await page.locator('[role="status"]').filter({hasText:'Sign-in required'}).waitFor({timeout:5000});
 assert.equal(await page.getByText('No prepared answers yet',{exact:false}).count(),0,'no generic banner for sign-in');
 console.error('Step 1.3 sign-in-required passed');

 // 1b. No resolved application form
 await page.locator(`.job-row[data-job-id="${noFormId}"]`).waitFor({timeout:3000});
 await openApp(noFormId);
 await page.locator('[role="status"]').filter({hasText:'No application form was resolved'}).waitFor({timeout:5000});
 assert.ok(await page.locator('[role="status"] a',{hasText:'Apply via the original listing'}).isVisible(),'listing link visible');
 assert.equal(await page.getByText('No prepared answers yet',{exact:false}).count(),0,'no generic banner for no-form');
 console.error('Step 1.3 no-resolved-form passed');

 // 1c. Backed-off
 await page.locator(`.job-row[data-job-id="${backedId}"]`).waitFor({timeout:3000});
 await openApp(backedId);
 await page.locator('[role="status"]').filter({hasText:'slow down'}).waitFor({timeout:5000});
 assert.equal(await page.getByText('No prepared answers yet',{exact:false}).count(),0,'no generic banner for backed-off');
 console.error('Step 1.3 backed-off passed');

 // 1d. Failed with concise error and retry button
 await page.locator(`.job-row[data-job-id="${failedId}"]`).waitFor({timeout:3000});
 await openApp(failedId);
 await page.locator('[role="status"]').filter({hasText:/did not respond|Retry inspection/}).waitFor({timeout:5000});
 assert.ok(await page.getByRole('button',{name:'Retry preparation',exact:true}).isVisible(),'retry button visible');
 assert.equal(await page.getByText('No prepared answers yet',{exact:false}).count(),0,'no generic banner for failed');
 console.error('Step 1.3 failed-with-retry passed');

 // 1e. Not pre-selected (Archive / applied stage)
 await page.getByRole('button',{name:/^Archive/}).click();
 await page.locator(`.job-row[data-job-id="${archiveId}"]`).waitFor({timeout:5000});
 await openApp(archiveId);
 await page.getByText('Move this role to Pre-selected',{exact:false}).waitFor({timeout:5000});
 assert.equal(await page.getByText('No prepared answers yet',{exact:false}).count(),0,'no generic banner for not-preselected');
 console.error('Step 1.3 not-pre-selected passed');

 console.error('Step 1.3 passed: all five outcomes show distinct states, none shows generic banner');

 // ── Step 1.4: provider selector defaults to configured stage ─────────────────
 // Switch back to Pre-selected and use the provider-check job.
 await page.getByRole('button',{name:/^Pre-selected/}).click();
 await page.locator(`.job-row[data-job-id="${provCheckId}"]`).waitFor({timeout:5000});
 await openApp(provCheckId);
 // Questions from the pre-loaded draft should be visible (no auto-prepare fires since operation exists).
 await page.locator('.draft-field').first().waitFor({timeout:5000});

 // Open More actions to access provider selector.
 await page.getByText('More actions',{exact:true}).click();
 const sel=page.locator('#preparation-provider');
 const defaultText=await sel.locator('option[value=""]').innerText();
 assert.ok(defaultText.toLowerCase().includes('stage')||defaultText.toLowerCase().includes('luna'),'default option references configured stage or its name');

 // Check: clicking Reinspect without changing provider sends no "provider" field.
 const capturedBodies:any[]=[];
 await page.route(`**/api/jobs/${provCheckId}/application/prepare`,async route=>{capturedBodies.push(route.request().postDataJSON());await route.abort();});
 await page.getByRole('button',{name:'Reinspect questionnaire',exact:true}).click();
 await page.waitForTimeout(400);
 assert.ok(capturedBodies.length>0,'prepare request was sent');
 assert.equal(capturedBodies[0]?.provider,undefined,'no provider sent when using stage default');
 capturedBodies.length=0;

 // Check: selecting an explicit provider sends it.
 const opts=await sel.locator('option').all();
 const namedOpts=[];for(const o of opts){const v=await o.getAttribute('value');if(v)namedOpts.push(v);}
 if(namedOpts.length>0){
  const pName=namedOpts[0];
  await sel.selectOption(pName);
  await page.getByRole('button',{name:'Reinspect questionnaire',exact:true}).click();
  await page.waitForTimeout(400);
  assert.ok(capturedBodies.length>0,'prepare request with provider was sent');
  assert.equal(capturedBodies[0]?.provider,pName,`explicit provider "${pName}" sent in request`);
 }
 await page.unroute(`**/api/jobs/${provCheckId}/application/prepare`);

 // Check: when no stage is configured, the tab shows the configuration prompt and sends nothing.
 // Use a fresh Pre-selected job with no draft/operations. Mock stage-providers to return preparation=''.
 await page.route('**/api/stage-providers',async route=>{await route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({stages:{assessment:'luna',preparation:'',extraction:'luna'},providers:[{name:'luna',model:'openai/gpt-6-luna'},{name:'openai',model:'openai/gpt-6.1-sol'}]})});});
 // Navigate to the fresh no-stage job while the mock is active.
 await page.locator(`.job-row[data-job-id="${noStageId}"]`).waitFor({timeout:5000});
 await openApp(noStageId);
 // With preparation stage empty and no prior operations, the config prompt must appear.
 await page.locator('[role="status"]').filter({hasText:'Choose a preparation model in Configuration'}).waitFor({timeout:8000});
 // The prepare button must be disabled (stage not configured, no explicit provider).
 assert.ok(await page.getByRole('button',{name:'Prepare application',exact:true}).isDisabled(),'prepare button disabled when no stage');
 // No POST should have been sent for this job.
 const noStageRequests:any[]=[];
 await page.route(`**/api/jobs/${noStageId}/application/prepare`,async route=>{noStageRequests.push(1);await route.abort();});
 await page.getByRole('button',{name:'Prepare application',exact:true}).click({force:true,timeout:200}).catch(()=>{});
 await page.waitForTimeout(300);
 assert.equal(noStageRequests.length,0,'no prepare request when stage not configured');
 await page.unroute(`**/api/jobs/${noStageId}/application/prepare`);
 await page.unroute('**/api/stage-providers');
 console.error('Step 1.4 passed: stage default sends no provider, explicit sends provider, no-stage shows config prompt');

 assert.deepEqual(errors,[],'no browser exceptions');
 console.log(JSON.stringify({passed:true,steps:['1.1','1.2','1.3','1.4']}));
}finally{
 await ctx.close();await browser.close();
 await new Promise<void>(r=>server.close(()=>r()));
 store.close();
}
