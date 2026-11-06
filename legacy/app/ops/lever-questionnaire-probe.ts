// Explicit public read-only probe. Never uses operational storage or owner login.
import '../test/isolate.js';
import {mkdirSync,writeFileSync,readFileSync} from 'node:fs';import assert from 'node:assert/strict';
import {browserSession} from '../src/network.js';import {inspectApplication,inspectionProfile} from '../src/application-browser.js';import {localPath,rootPath,now} from '../src/core.js';
const mode=process.argv[2];assert.ok(['dom','verify'].includes(mode),'Use dom | verify');
const url='https://jobs.eu.lever.co/aavelabs/992c5de1-1cee-4002-bcc5-8f2131630d0b/apply',dir=localPath('verification');mkdirSync(dir,{recursive:true});
const captured:any[]=[];
const snapshot=await inspectApplication(url,{context:async()=>{
 const context=await browserSession(inspectionProfile),close=context.close.bind(context);
 context.close=async()=>{
  try{for(const page of context.pages())if(!page.isClosed()&&page.url()===url){
   writeFileSync(`${dir}/lever-${mode}-rendered.html`,await page.content(),{mode:0o600});
   const read=()=>Array.from(document.querySelectorAll('input,textarea,select,[role=combobox],[role=checkbox],[role=radio]')).map(el=>({tag:el.tagName,type:(el as HTMLInputElement).type,name:(el as HTMLInputElement).name,id:el.id,placeholder:el.getAttribute('placeholder'),visible:el.getClientRects().length>0,hasAnswer:['radio','checkbox'].includes((el as HTMLInputElement).type)?(el as HTMLInputElement).checked:(el as HTMLInputElement).type==='file'?!!(el as HTMLInputElement).files?.length:(el as HTMLInputElement).type==='hidden'?false:!!(el as HTMLInputElement).value,labels:Array.from((el as HTMLInputElement).labels??[]).map(label=>label.outerHTML),ancestors:[el.parentElement,el.parentElement?.parentElement,el.parentElement?.parentElement?.parentElement].filter(Boolean).map(parent=>({tag:parent!.tagName,class:parent!.className,html:parent!.outerHTML.slice(0,22000),text:(parent as HTMLElement).innerText}))}));
   captured.push({url:page.url(),controls:await page.evaluate(`(()=>{const __name=(fn)=>fn;return (${read.toString()})()})()`)});
   await page.screenshot({path:`${dir}/lever-${mode}.png`,fullPage:true});
  }}finally{await close();}
 };return context;
}});
writeFileSync(`${dir}/lever-${mode}.json`,JSON.stringify({checked:now(),url,snapshot,captured},null,2),{mode:0o600});
if(mode==='verify'){
 assert.equal(snapshot.status,'partial');assert.equal(snapshot.url,url);assert.equal(snapshot.fields.length,20);assert.equal(captured.length,1);assert.ok(captured[0].controls.every((c:any)=>!c.hasAnswer),'Public controls must remain empty/unselected; no form interaction allowed');
 const questions=[['316da9cc-e22e-4bfc-838d-5da12f2547b5',[
  'Where are you based/Where will you work from?','What is your Right To Work status in the location you plan to work from?','What are your salary expectations?','Please share more about your experience in DeFi:'
 ]],['60488034-6b4a-49f2-9924-89e599009012',[
  'What is the most interesting contract or project you have designed or contributed to? How does it work?','What is the most challenging design decision you have faced in a smart contract project?','(Optional) Please share the on chain address for a contract you have written/contributed to:'
 ]]] as const;
 for(const [card,labels] of questions)for(const [i,label] of labels.entries())assert.equal(snapshot.fields.find(f=>f.controlName===`cards[${card}][field${i}]`)?.label,label);
 assert.equal(snapshot.fields.find(f=>f.controlName==='opportunityLocationId')?.label,'Which location are you applying for?');assert.equal(snapshot.fields.find(f=>f.controlName==='resume')?.label,'Resume/CV');assert.equal(snapshot.fields.find(f=>f.controlName==='location')?.label,'Current location');assert.ok(snapshot.fields.every(f=>f.labelStatus==='resolved'&&!/Type your response|cards\[|Loading|auto-read|Analyzing|Success!/.test(f.label)));
 const baseline=process.argv[3];assert.ok(baseline&&(baseline.startsWith('.local/tests/')||baseline.startsWith('.local/research/')),'verify requires an isolated baseline JSON path');const old=JSON.parse(readFileSync(rootPath(baseline),'utf8')).snapshot;
 assert.deepEqual(snapshot.fields.map(({id,controlName,type})=>({id,controlName,type})),old.fields.map(({id,controlName,type}:any)=>({id,controlName,type})),'Existing field IDs/control names/types must be unchanged');
 writeFileSync(`${dir}/lever-verification.json`,JSON.stringify({passed:true,checked:now(),url,controls:20,readableCustomQuestions:7,stableIds:true,emptyControls:true,providerInvocations:0,formFillUploadSubmitActions:0,baseline,evidence:`${dir}/lever-verify.json`},null,2),{mode:0o600});
}
console.log(JSON.stringify({state:localPath(),evidence:`${dir}/lever-${mode}.json`,status:snapshot.status,fields:snapshot.fields.map(({controlName,label,type,options,required})=>({controlName,label,type,options,required})),gaps:snapshot.gaps},null,2));
