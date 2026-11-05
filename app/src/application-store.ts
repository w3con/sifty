import {z} from 'zod';
import {hash,now} from './core.js';
import type {Store} from './store.js';
export const questionSchema=z.object({id:z.string(),label:z.string().min(1).max(8000),instructions:z.string().max(12000).optional(),labelStatus:z.enum(['resolved','uncertain']).optional(),type:z.enum(['text','textarea','select','radio','checkbox','file']),options:z.array(z.string()).default([]),required:z.boolean().default(false),maxLength:z.number().nullable().default(null),accept:z.string().default(''),sourceUrl:z.string(),origin:z.enum(['site','manual']),language:z.string().default('und'),signature:z.string().default(''),controlName:z.string().default('')});
export type Question=z.infer<typeof questionSchema>;
export type AnswerField=Question&{value:string|string[];revision:number;edited:boolean;unknowns:string[];evidence:{sourceId:string;quote:string}[];changed?:boolean;removed?:boolean};
export type Questionnaire={url:string;capturedAt:string;language:string;status:'partial'|'blocked'|'login-required'|'unsupported';gaps:string[];fields:Question[];family:string;technicalLog?:string};
export type ApplicationDraft={jobId:number;revision:number;fields:AnswerField[];selectedFile:'cv.pdf'|'profile.pdf';questionnaire:Questionnaire|null;provenance:any;notes:string[]};
export function initApplications(store:Store){store.db.exec(`
 CREATE TABLE IF NOT EXISTS application_drafts(job_id INTEGER PRIMARY KEY,data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS application_revisions(id INTEGER PRIMARY KEY,job_id INTEGER NOT NULL,created TEXT NOT NULL,reason TEXT NOT NULL,data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS questionnaire_snapshots(id INTEGER PRIMARY KEY,job_id INTEGER NOT NULL,created TEXT NOT NULL,data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS preparation_operations(id INTEGER PRIMARY KEY,job_id INTEGER NOT NULL,status TEXT NOT NULL,created TEXT NOT NULL,updated TEXT NOT NULL,pid INTEGER,request TEXT NOT NULL,error TEXT,progress TEXT NOT NULL);
 INSERT OR IGNORE INTO migrations VALUES(9); PRAGMA user_version=9;`);}
export function mergeApplicationJobs(store:Store,from:number,to:number){
 const row=store.db.prepare('SELECT data FROM application_drafts WHERE job_id=?').get(from);
 if(row){const incoming=JSON.parse(String(row.data)) as ApplicationDraft,target=store.db.prepare('SELECT data FROM application_drafts WHERE job_id=?').get(to);
  if(target){const current=JSON.parse(String(target.data)) as ApplicationDraft;current.fields.push(...incoming.fields.filter(f=>!current.fields.some(c=>c.id===f.id)));current.notes.push(`Merged role ${from}; earlier draft retained in revision history. Review overlapping answers.`);writeDraft(store,current,'Job alias merge; both draft histories retained');}
  else store.db.prepare('INSERT INTO application_drafts VALUES(?,?)').run(to,JSON.stringify({...incoming,jobId:to}));
  store.db.prepare('DELETE FROM application_drafts WHERE job_id=?').run(from);
 }
 for(const table of ['application_revisions','questionnaire_snapshots','preparation_operations'])store.db.prepare(`UPDATE ${table} SET job_id=? WHERE job_id=?`).run(to,from);
}
export function applicationDraft(store:Store,id:number):ApplicationDraft {
 id=store.get(id).id;const row=store.db.prepare('SELECT data FROM application_drafts WHERE job_id=?').get(id);
 return row?JSON.parse(String(row.data)):{jobId:id,revision:0,fields:[],selectedFile:'profile.pdf',questionnaire:null,provenance:null,notes:[]};
}
export function writeDraft(store:Store,draft:ApplicationDraft,reason:string){
  store.get(draft.jobId); // Deleted roles cannot be recreated by a late async caller.
 draft={...draft,revision:draft.revision+1};const data=JSON.stringify(draft);
 store.db.prepare('INSERT INTO application_drafts VALUES(?,?) ON CONFLICT(job_id) DO UPDATE SET data=excluded.data').run(draft.jobId,data);
 store.db.prepare('INSERT INTO application_revisions(job_id,created,reason,data) VALUES(?,?,?,?)').run(draft.jobId,now(),reason,data);return draft;
}
function transaction<T>(store:Store,fn:()=>T):T{store.db.exec('BEGIN IMMEDIATE');try{const result=fn();store.db.exec('COMMIT');return result;}catch(e){store.db.exec('ROLLBACK');throw e;}}
export function inspectIntoDraft(store:Store,id:number,snapshot:Questionnaire){return transaction(store,()=>{
 const draft=applicationDraft(store,id);const current=new Map(draft.fields.map(f=>[f.id,f]));
 if(!snapshot.fields.length&&['blocked','login-required','unsupported'].includes(snapshot.status)){
  store.db.prepare('INSERT INTO questionnaire_snapshots(job_id,created,data) VALUES(?,?,?)').run(id,now(),JSON.stringify(snapshot));
  return writeDraft(store,{...draft,questionnaire:snapshot,notes:[...draft.notes,'Inspection could not read the questionnaire; existing fields and answers retained.']},'Inspection access failure; previous draft retained');
 }
 const fields:AnswerField[]=snapshot.fields.map(raw=>{const q=questionSchema.parse(raw),old=current.get(q.id);return old?{...old,...q,changed:old.signature!==q.signature||old.changed,removed:false}:{...q,value:q.type==='checkbox'?[]:'',revision:0,edited:false,unknowns:[],evidence:[]};});
 for(const old of draft.fields)if(!snapshot.fields.some(q=>q.id===old.id))fields.push(old.origin==='manual'?old:{...old,removed:true});
 store.db.prepare('INSERT INTO questionnaire_snapshots(job_id,created,data) VALUES(?,?,?)').run(id,now(),JSON.stringify(snapshot));
 return writeDraft(store,{...draft,questionnaire:snapshot,fields},'Questionnaire inspected; existing answers preserved');
});}
export function addManualQuestion(store:Store,id:number,label:string,type:Question['type']='textarea',options:string[]=[]){
 const q=questionSchema.parse({id:`manual-${hash(`${label}|${now()}`)}`,label,type,options,origin:'manual',sourceUrl:store.get(id).originalUrl});q.signature=hash(JSON.stringify(q));
 return transaction(store,()=>{const draft=applicationDraft(store,id);return writeDraft(store,{...draft,fields:[...draft.fields,{...q,value:type==='checkbox'?[]:'',revision:0,edited:false,unknowns:[],evidence:[]}]},'Owner added manual question');});
}
export function updateApplication(store:Store,id:number,change:any){return transaction(store,()=>{
 const draft=applicationDraft(store,id);
 if(change.selectedFile!==undefined){if(!['cv.pdf','profile.pdf'].includes(change.selectedFile))throw Error('Choose existing cv.pdf or profile.pdf');if(change.revision!==draft.revision)throw Error('Draft changed; reload before changing the selected file');draft.selectedFile=change.selectedFile;}
 if(change.fieldId!==undefined){const field=draft.fields.find(f=>f.id===change.fieldId);if(!field)throw Error('Unknown field');if(field.revision!==change.revision)throw Error('Field changed concurrently; reload before saving');
  const value=change.value;if(typeof value!=='string'&&!(Array.isArray(value)&&value.every(v=>typeof v==='string')))throw Error('Invalid answer value');
  if(JSON.stringify(value).length>40000)throw Error('Answer exceeds storage limit');
  if(field.type==='checkbox'&&!Array.isArray(value))throw Error('Checkbox answers must be an array');if(field.type!=='checkbox'&&typeof value!=='string')throw Error('This field requires a text value');
  if(['select','radio','checkbox'].includes(field.type)){const values=Array.isArray(value)?value:[value];if(values.some(v=>v&&!field.options.includes(v)))throw Error('Answer must use an available option');}
  if(field.maxLength&&typeof value==='string'&&value.length>field.maxLength)throw Error('Answer exceeds the field length limit');
  field.value=value;field.revision++;field.edited=true;field.changed=false;
 }
 return writeDraft(store,draft,'Owner edited draft only');
});}
export function mergeGenerated(store:Store,id:number,base:ApplicationDraft,answers:any[],provenance:any,selectedFile?:string){return transaction(store,()=>{
 const draft=applicationDraft(store,id),skipped:string[]=[];
 for(const answer of answers){const field=draft.fields.find(f=>f.id===answer.id),old=base.fields.find(f=>f.id===answer.id);if(!field||!old||field.revision!==old.revision||field.signature!==old.signature){skipped.push(answer.id);continue;}
  field.value=answer.value;field.unknowns=answer.unknowns;field.evidence=answer.evidence;field.revision++;field.edited=false;
 }
 if(draft.revision===base.revision&&['cv.pdf','profile.pdf'].includes(selectedFile??''))draft.selectedFile=selectedFile as ApplicationDraft['selectedFile'];
 draft.provenance=provenance;draft.notes=skipped.length?[`Concurrent changes preserved for ${skipped.join(', ')}`]:[];
 return writeDraft(store,draft,'Model draft prepared; concurrent changes preserved');
});}
