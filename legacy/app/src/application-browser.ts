import type {Page,BrowserContext,Route} from 'playwright';
import {browserSession,publicUrl,fetchPage,networkProfile} from './network.js';
import {traffic} from './traffic.js';
import {hash,now,runtime,localPath,ROOT} from './core.js';
import {mkdirSync,writeFileSync} from 'node:fs';import {randomUUID} from 'node:crypto';import {relative} from 'node:path';
import {navigationKind,publicGetroPage,getroApplicationDestination,applicationLinks,concisePreparationError} from './application-navigation.js';
import {questionSchema,type Question,type Questionnaire} from './application-store.js';
import {readRenderedFields} from './rendered-fields.js';
export const inspectionProfile='application-inspection';
async function bounded<T>(promise:Promise<T>,ms=4000):Promise<T>{let timer:NodeJS.Timeout|undefined;try{return await Promise.race([promise,new Promise<never>((_,reject)=>{timer=setTimeout(()=>reject(Error('Accessible document did not stabilize in time')),ms);})]);}finally{if(timer)clearTimeout(timer);}}
export async function renderedQuestionnaire(page:Page):Promise<Questionnaire>{
 const fields:Question[]=[],gaps=['Visible accessible controls only; hidden conditional and gated later steps have not been inspected. No fields were filled or submitted.'];
 const language=await bounded(page.locator('html').getAttribute('lang'))??'und';
 if(await bounded(page.locator('input[type=password]').count()))return {url:page.url(),capturedAt:now(),language,status:'login-required',family:/linkedin\.com$/.test(new URL(page.url()).hostname)?'LinkedIn public job / account gate':'Application login gate',fields:[],gaps:[...gaps,'Authentication/registration controls detected. Establish login in the owner browser and retry; account fields are not application questions.']};
 const frames=page.frames().filter(frame=>frame.url()&&!/^https?:\/\/(?:accounts\.google\.com|(?:www\.)?google\.com\/recaptcha|li\.protechts\.net)/i.test(frame.url()));
 if(frames.length>12)gaps.push('Additional embedded frames exceeded the inspection bound; their questionnaire coverage is unverified.');
 for(const frame of frames.slice(0,12))try{
  // tsx's keepNames transform inserts __name calls into nested callbacks. Scope the
  // identity helper locally in the serialized reader, without changing site globals.
    const items=await bounded(frame.evaluate(`(()=>{const __name=(fn)=>fn;return (${readRenderedFields.toString()})()})()`)) as ReturnType<typeof readRenderedFields>;
   for(const {warnings,...item} of items){for(const warning of warnings)gaps.push(`${item.controlName||'Unidentified control'}: ${warning}`);const id=hash(`${frame.url()}|${item.controlName}|${item.type}`),existing=fields.find(f=>f.id===id);if(existing&&['radio','checkbox'].includes(item.type)){existing.options=[...new Set([...existing.options,...item.options])];existing.required ||=item.required;}else fields.push(questionSchema.parse({...item,id,sourceUrl:frame.url(),language,origin:'site'}));}
 }catch(e){gaps.push(`Frame unavailable: ${frame.url()}: ${String(e)}`);}
 for(const file of [...fields])if(file.type==='file'&&/cover.?letter|lettre.?motivation/i.test(`${file.controlName} ${file.label}`)){
  fields.push({...file,id:`${file.id}-text-draft`,type:'textarea',label:language.startsWith('fr')?'Lettre de motivation (brouillon)':`Cover letter (editable draft for attachment)`,controlName:`${file.controlName}:draft`,accept:''});
  gaps.push('Cover-letter attachment includes a separate editable text draft; no cover-letter file is generated or uploaded.');
 }
 for(const field of fields)field.signature=hash(JSON.stringify(field));
 const text=(await bounded(page.locator('body').innerText())).slice(0,60000),password=await bounded(page.locator('input[type=password]').count());
 const blocked=/verify you are human|just a moment|access denied|captcha/i.test(await bounded(page.title()));
 const status=blocked?'blocked':password?'login-required':fields.length?'partial':'unsupported';
 if(password)gaps.push('Authentication/registration gate detected. Establish login in the owner browser; login fields are not application questions.');
 if(!fields.length)gaps.push('No accessible questionnaire controls found. Establish the owner login, retry, or paste questions manually.');
 if(/next|continue|step \d|suivant|étape|weiter/i.test(text))gaps.push('Possible multi-step form: later gated steps are unverified.');
 if(fields.some(f=>f.type==='select'&&!f.options.length))gaps.push('Custom dropdown options are not exposed without interaction; supply choices manually.');
 const host=new URL(page.url()).hostname,family=/greenhouse|boards\.greenhouse/.test(host)?'Greenhouse rendered controls':/lever\.co/.test(host)?'Lever rendered controls':/ashbyhq/.test(host)?'Ashby accessible controls':'Generic rendered HTML/accessibility controls';
 return {url:page.url(),capturedAt:now(),language,status,gaps,fields:password||blocked?[]:fields,family};
}
type NavigationPayload={status:number;headers:Record<string,string>;body:Buffer};
export type InspectionDependencies={signal?:AbortSignal;context?:()=>Promise<BrowserContext>;validate?:typeof publicUrl;publicFetch?:typeof fetchPage;navigation?: (route:Route)=>Promise<NavigationPayload>;pace?:typeof traffic.run};
export async function inspectApplication(url:string,dependencies:InspectionDependencies={}):Promise<Questionnaire>{
  const signal=dependencies.signal;signal?.throwIfAborted();
 const validate=dependencies.validate??publicUrl,pace=dependencies.pace??traffic.run.bind(traffic),publicFetch=dependencies.publicFetch??fetchPage;
 const events:any[]=[],failures=new Map<string,{kind:string;status:number;contentType:string;message?:string}>(),redirects=new Map<string,string>();let blockedRequests=0,resourceRedirects=0,requestCount=0,mainNavigations=0,context:BrowserContext|undefined,snapshot:Questionnaire|undefined,closing=false;
 const abort=async(route:Route)=>{try{await route.abort('blockedbyclient');}catch(e){if(!closing)events.push({requestCancelled:String(e).split('\n')[0]});}};
 const technicalPath=localPath(`application-inspections/${Date.now()}-${randomUUID()}.json`),technicalLog=relative(ROOT,technicalPath);
 mkdirSync(localPath('application-inspections'),{recursive:true});
  const record=(event:any)=>{if(signal?.aborted)return;events.push(event);writeFileSync(technicalPath,JSON.stringify({requestedUrl:url,events,result:snapshot},null,2),{mode:0o600});};
 record({startedAt:now(),phase:'starting'});
 const unavailable=(destination:string,kind:string,status=0,contentType='',message?:string):Questionnaire=>({url:destination,capturedAt:now(),language:'und',status:kind==='login-required'?'login-required':kind==='blocked'?'blocked':'unsupported',family:'Application navigation',fields:[],technicalLog,gaps:[!status&&message?message:kind==='download'?`The destination returns a file (${contentType||'unknown type'}), not a readable application form.`:kind==='not-found'?`The application destination is unavailable (HTTP ${status}).`:kind==='blocked'||kind==='login-required'?`The application destination requires access or owner login (HTTP ${status}).`:message??'No readable application questionnaire was returned.','Open the source manually, establish any required login in the dedicated owner browser, then retry; or paste questions manually.']});
  const cancel=()=>{void context?.close().catch(()=>{});};signal?.addEventListener('abort',cancel,{once:true});
  try{
   await validate(url);signal?.throwIfAborted();context=await (dependencies.context?.()??browserSession(inspectionProfile));if(signal?.aborted){await context.close();signal.throwIfAborted();}
  await context.route('**/*',async route=>{
   const request=route.request(),destination=request.url();requestCount++;
   if(requestCount>1500){if(requestCount===1501){record({phase:'request-bound',reason:'Page repeatedly retries blocked requests; inspection stopped'});for(const page of context!.pages())void page.close().catch(()=>{});}return abort(route);}
   if(!['GET','HEAD'].includes(request.method())){blockedRequests++;return abort(route);}
   try{
     signal?.throwIfAborted();await validate(destination);signal?.throwIfAborted();
    if(!request.isNavigationRequest()){
     // Playwright may bypass route interception on a browser-followed redirect.
     // Do not let scripts/images use that path to reach an unvalidated address.
     const response=await route.fetch({maxRedirects:0,timeout:runtime().request_timeout_ms});
     if(response.status()>=300&&response.status()<400&&response.headers().location){await validate(new URL(response.headers().location,destination).href);resourceRedirects++;return abort(route);}
     return route.fulfill({response});
    }
    if(request.frame()===request.frame().page().mainFrame()&&++mainNavigations>6){failures.set(destination,{kind:'unsupported',status:0,contentType:'',message:'Application redirect/navigation limit reached. Open the destination manually.'});return abort(route);}
     let payload:NavigationPayload=await pace(destination,async()=>{
      signal?.throwIfAborted();
     if(dependencies.navigation)return dependencies.navigation(route);
     const response=await route.fetch({maxRedirects:0,timeout:runtime().request_timeout_ms});const body=await response.body();if(body.length>8_000_000)throw Error('Application document exceeds 8MB');return {status:response.status(),headers:response.headers(),body};
    });
     signal?.throwIfAborted();let kind=navigationKind({status:payload.status,contentType:payload.headers['content-type']??'',disposition:payload.headers['content-disposition'],url:destination});
    record({url:destination,transport:'session-browser',status:payload.status,contentType:payload.headers['content-type'],bytes:payload.body.length,kind});
    // Framework's public board rejects the headless UA, while its ordinary public
    // HTTP page is readable. Never forward session cookies/auth to this recovery.
    if(kind==='blocked'&&payload.status===403&&publicGetroPage(destination)){
      const page=await publicFetch(destination,inspectionProfile,runtime().request_timeout_ms,{...networkProfile(inspectionProfile),browser:false},{},signal);signal?.throwIfAborted();
     const recoveredKind=navigationKind({status:page.status,contentType:page.contentType??'',url:page.url});record({url:page.url,transport:'validated-public-http',status:page.status,contentType:page.contentType,kind:recoveredKind});
     const directApply=recoveredKind==='html'?getroApplicationDestination(page.text,page.url):null;
     if(directApply){await validate(directApply);record({url:page.url,applyDestination:directApply,source:'Public Getro currentJob.url backing Apply now button'});redirects.set(destination,directApply);return abort(route);}
     if(page.url!==destination){await validate(page.url);redirects.set(destination,page.url);record({url:destination,redirect:page.url,source:'validated-public-http'});return abort(route);}
     payload={status:page.status,headers:page.navigationHeaders??{'content-type':page.contentType??''},body:Buffer.from(page.text)};kind=recoveredKind;
    }
    if(!['html','redirect'].includes(kind)){failures.set(destination,{kind,status:payload.status,contentType:payload.headers['content-type']??''});return abort(route);}
    if(kind==='redirect'){
     const location=payload.headers.location;if(!location){failures.set(destination,{kind:'unsupported',status:0,contentType:'',message:'The application redirect did not provide a destination.'});return abort(route);}
     const target=new URL(location,destination).href;await validate(target);record({url:destination,redirect:target,status:payload.status});
     if(request.frame()===request.frame().page().mainFrame())redirects.set(destination,target);
     else failures.set(destination,{kind:'unsupported',status:0,contentType:'',message:`Embedded form redirects to ${target}; this frame's coverage is unverified.`});
     return abort(route);
    }
    const headers={...payload.headers};delete headers['content-length'];delete headers['content-encoding'];await route.fulfill({status:payload.status,headers,body:payload.body});
   }catch(e){record({url:destination,error:String(e)});failures.set(destination,{kind:'blocked',status:0,contentType:'',message:concisePreparationError(e)});await abort(route);}
  });
  let page=await context.newPage();const visited=new Set<string>();let destination=url;
  for(let hop=0;hop<6;hop++){
   await validate(destination);if(visited.has(destination))break;visited.add(destination);
   record({url:destination,phase:'navigate',hop});
   try{await page.goto(destination,{waitUntil:'domcontentloaded',timeout:runtime().request_timeout_ms*2+15000});await page.waitForTimeout(/(?:^|\.)linkedin\.com$/.test(new URL(page.url()).hostname)?100:1500);}catch(e){const redirect=redirects.get(destination);if(redirect&&!visited.has(redirect)){await page.close();page=await context.newPage();destination=redirect;continue;}record({url:destination,navigationError:String(e)});const failure=failures.get(destination)??[...failures.entries()].reverse().find(([u])=>page.url()===u)?.[1]??[...failures.values()].at(-1);snapshot=unavailable(destination,failure?.kind??'unsupported',failure?.status, failure?.contentType,requestCount>1500?'The application page repeatedly retried blocked requests. Open it manually or add questions.':failure?.message??concisePreparationError(e));break;}
   record({phase:'reading-navigation',url:page.url()});
   const html=await bounded(page.content()),boardTarget=getroApplicationDestination(html,page.url());
   if(boardTarget&&!visited.has(boardTarget)){record({url:page.url(),applyDestination:boardTarget,source:'Getro currentJob.url backing Apply now button'});destination=boardTarget;continue;}
   // Public LinkedIn pages can contain an employer Apply link alongside account
   // widgets. Follow its explicit destination before mistaking those login fields
   // for an application questionnaire; no button/form interaction is needed.
   const linkedIn=/(?:^|\.)linkedin\.com$/.test(new URL(page.url()).hostname),applyCandidates=applicationLinks(html,page.url());
   if(linkedIn)record({url:page.url(),applyCandidates,phase:'public-linkedin-destinations'});
   const earlyLink=linkedIn?applyCandidates.find(target=>!visited.has(target)&&!/(?:^|\.)linkedin\.com$/.test(new URL(target).hostname)):undefined;
   if(earlyLink){record({url:page.url(),applyDestination:earlyLink,source:'Public employer Apply link'});destination=earlyLink;continue;}
   record({phase:'reading-controls',url:page.url()});snapshot=await renderedQuestionnaire(page);
   if(snapshot.fields.length||snapshot.status==='blocked')break;
   const next=applicationLinks(html,page.url()).find(target=>!visited.has(target));if(next){record({url:page.url(),applyDestination:next,source:'Apply navigation link or explicit button destination'});destination=next;continue;}
   break;
  }
  snapshot??=unavailable(destination,'unsupported',0,'','Application navigation stopped at its redirect/step limit.');
  if(blockedRequests)snapshot.gaps.push(`${blockedRequests} non-GET/HEAD requests blocked; dynamic or gated form coverage may be incomplete.`);
  if(resourceRedirects)snapshot.gaps.push(`${resourceRedirects} redirected supporting resources were not loaded; dynamic questionnaire coverage may be incomplete.`);
  for(const [frameUrl,failure] of failures)if(frameUrl!==snapshot.url)snapshot.gaps.push(`Embedded or redirected destination unavailable (${failure.kind}${failure.status?` HTTP ${failure.status}`:''}): ${frameUrl}`);
   signal?.throwIfAborted();snapshot.technicalLog=technicalLog;return snapshot;
  }catch(e){signal?.throwIfAborted();record({error:String(e)});snapshot=unavailable(url,'blocked',0,'',concisePreparationError(e));return snapshot;}
  finally{signal?.removeEventListener('abort',cancel);closing=true;record({phase:'closing',at:now()});await context?.unrouteAll({behavior:'ignoreErrors'}).catch(()=>{});await context?.close();record({phase:'closed',at:now()});}
}
export async function openApplicationLogin(url:string,dependencies:Pick<InspectionDependencies,'context'|'validate'|'signal'>={}){
  const signal=dependencies.signal;signal?.throwIfAborted();await (dependencies.validate??publicUrl)(url);signal?.throwIfAborted();const context=await (dependencies.context?.()??browserSession(inspectionProfile,true));
  const cancel=()=>{void context.close().catch(()=>{});};signal?.addEventListener('abort',cancel,{once:true});if(signal?.aborted){await context.close();signal.throwIfAborted();}
 // Persistent Chromium already has a startup page. Reuse it rather than leaving
 // a blank window behind that keeps the context alive after the owner closes login.
 const page=context.pages()[0]??await context.newPage();let complete!:()=>void;
 const closed=new Promise<void>(resolve=>{complete=resolve;});
 page.once('close',complete);context.once('close',complete);
 try{
  await page.goto(url,{waitUntil:'domcontentloaded',timeout:runtime().request_timeout_ms});await page.bringToFront();
  // Only the owner operates login. Closing that page completes the handoff, even
  // when macOS leaves the browser process running with no visible windows.
   await closed;signal?.throwIfAborted();
 }finally{
   signal?.removeEventListener('abort',cancel);page.off('close',complete);context.off('close',complete);await context.close();
 }
}
