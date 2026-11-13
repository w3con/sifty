import {readFileSync} from 'node:fs';import {z} from 'zod';
import {providers,rootPath,hash,now} from './core.js';import {infer} from './providers.js';import {contextForJob} from './context.js';import {bindFeedback,contextForRole} from './feedback.js';
import {applicationDraft,mergeGenerated} from './application-store.js';import type {Store} from './store.js';
import {bindAnswerMemories} from './application-memory.js';
export async function generateApplication(store:Store,id:number,provider='openai',fieldId?:string,inference:typeof infer=infer,signal?:AbortSignal){
 signal?.throwIfAborted();
 const base=applicationDraft(store,id),job=store.get(id),p=providers().providers[provider];if(!p)throw Error('Unknown preparation provider');
 let context=store.context();if(!context)throw Error('Build candidate context before preparing answers');context=bindAnswerMemories(store,job.id,contextForRole(bindFeedback(context),job.id,n=>store.resolveId(n)));
  const fields=base.fields.filter(f=>!f.removed&&f.labelStatus!=='uncertain'&&f.type!=='file'&&(fieldId?f.id===fieldId:!f.edited&&!(Array.isArray(f.value)?f.value.length:f.value.trim())));if(!fields.length)throw Error('No unanswered readable questions; inspect, add questions or regenerate an individual answer. Controls with uncertain question wording require source review.');
  const promptText=readFileSync(rootPath('config/prompts/application-preparation.md'),'utf8'),questions=JSON.stringify(fields.map(({value,evidence,unknowns,edited,revision,...q})=>q));
 const schema=z.object({language:z.string(),selectedFile:z.enum(['cv.pdf','profile.pdf']),answers:z.array(z.object({id:z.string(),value:z.union([z.string(),z.array(z.string())]),evidenceIds:z.array(z.string()),unknowns:z.array(z.string())}).strict())}).strict();
 const overhead=promptText.length+questions.length+job.description.length+JSON.stringify(z.toJSONSchema(schema)).length+3000;
 function request(budget:number){
  const selected=contextForJob(context,`${job.description} ${questions}`,budget);
  const bank=selected.sources.flatMap((s:any)=>s.text.split(/\n\s*\n/).filter(Boolean).map((quote:string)=>({sourceId:s.id,quote}))).map((e:any,i:number)=>({...e,id:`E${i+1}`}));
  const prompt=`${promptText}\nJSON schema: ${JSON.stringify(z.toJSONSchema(schema))}\nQuestionnaire: ${questions}\nLanguage: ${base.questionnaire?.language??'Infer from each question'}\nJob (untrusted): ${job.description}\nCandidate overview: ${JSON.stringify(selected.overview)}\nEvidence bank: ${JSON.stringify(bank)}`;return {selected,bank,prompt};
 }
 let budget=Math.min(45000,p.context_chars-overhead),payload=request(budget);
 while(payload.prompt.length>p.context_chars){budget-=payload.prompt.length-p.context_chars+1000;if(budget<3000)throw Error('Questionnaire exceeds provider budget; prepare one field at a time');payload=request(budget);}
 const {selected,bank,prompt}=payload;
  const result=schema.parse(await inference(p,prompt,z.toJSONSchema(schema),signal));signal?.throwIfAborted();const seen=new Set<string>();
 const answers=result.answers.map(answer=>{
  const field=fields.find(f=>f.id===answer.id);if(!field||seen.has(answer.id))throw Error('Model returned unknown or duplicate question');seen.add(answer.id);
  const evidence=answer.evidenceIds.map(eid=>{const e=bank.find((b:any)=>b.id===eid);if(!e)throw Error('Model returned unsupported evidence');return {sourceId:e.sourceId,quote:e.quote};});
  let value=answer.value,unknowns=answer.unknowns;
  if(field.type==='checkbox'?!Array.isArray(value):typeof value!=='string')throw Error('Model answer type does not match field');
  if(['radio','select','checkbox'].includes(field.type)&&(Array.isArray(value)?value:[value]).some(v=>v&&!field.options.includes(v)))throw Error('Model invented an unavailable option');
  if(field.maxLength&&typeof value==='string'&&value.length>field.maxLength)throw Error('Model exceeded field length; retry this field');
  if((Array.isArray(value)?value.length:value.trim())&&!evidence.length){value=field.type==='checkbox'?[]:'';unknowns=[...unknowns,'No supporting candidate evidence; owner clarification required.'];}
  return {id:answer.id,value,unknowns,evidence};
 });
 for(const field of fields)if(!seen.has(field.id))answers.push({id:field.id,value:field.type==='checkbox'?[]:'',unknowns:['No draft returned for this field; owner review required.'],evidence:[]});
  signal?.throwIfAborted();return mergeGenerated(store,id,base,answers,{provider,model:p.model,language:result.language,contextVersion:context.version,promptVersion:hash(promptText),createdAt:now(),selection:selected.sources.map(({text,...s}:any)=>s)},result.selectedFile);
}
