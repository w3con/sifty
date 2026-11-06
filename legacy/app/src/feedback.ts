import { mkdirSync, writeFileSync, readdirSync, readFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import { hash, now, rootPath, localPath, ROOT } from './core.js';
import { relative } from 'node:path';
import type { Store } from './store.js';
import { contextVersion, type CandidateContext, type EvidenceSource } from './context.js';
export type Fact={subject:string;value:string;quote:string};
// Conservative optional extraction. Every other statement remains durable attributed raw evidence.
export function explicitFacts(text:string):Fact[]{
 const facts:Fact[]=[];
 for(const m of text.matchAll(/\b(Kubernetes|Docker)\s*(?:[:=—-]\s*)?(not known|no experience|experienced|known)\b/gi))facts.push({subject:m[1].toLowerCase(),value:/not known|no experience/i.test(m[2])?'explicit absence':'explicit experience',quote:m[0]});
 for(const clause of text.matchAll(/\brelocation\s*[:=—-]?\s*([^.\n;]+)/gi))for(const m of clause[1].matchAll(/\b(Canada|USA)\s*(?:[:=—-]\s*)?(yes|no)\b/gi))facts.push({subject:`relocation ${m[1].toLowerCase()}`,value:m[2].toLowerCase(),quote:clause[0]});
 return facts;
}
function feedbackDirectory(){return process.env.FCTO_STATE_DIR&&process.env.FCTO_STATE_DIR!=='.local'?relative(ROOT,localPath('candidate-feedback')):'kb/Applications/candidate-feedback';}
export function feedbackRecords(){const dir=rootPath(feedbackDirectory());try{return readdirSync(dir).filter(p=>p.endsWith('.md')).map(p=>{const text=readFileSync(`${dir}/${p}`,'utf8'),match=text.match(/<!-- FCTO_RECORD (.+) -->/);return match?JSON.parse(Buffer.from(match[1],'base64').toString('utf8')):null;}).filter(Boolean);}catch(e:any){if(e.code==='ENOENT')return [];throw e;}}
export function feedbackSources():EvidenceSource[]{const records=feedbackRecords();return records.map(r=>{const revisions=records.filter(next=>next.supersedes===r.id).map(next=>next.id);const text=`Owner-confirmed candidate feedback. Record ID: ${r.id}. Date: ${r.created}. Scope: ${r.scope}. Source job: ${r.jobUrl}. ${revisions.length?`Historical statement explicitly revised by ${revisions.join(', ')}; do not treat the old wording as a current fact.`:''}\nOriginal user statement (verbatim):\n${r.text}\n\nGrounded explicit facts: ${JSON.stringify(r.facts)}\nConflicts/review: ${JSON.stringify(r.conflicts)}. Supersedes: ${r.supersedes??'none'}. Earlier records are retained. Do not silently resolve conflicting facts or generalise role-scoped statements.`;return {id:r.path,reference:r.path,retrievedAt:r.created,hash:hash(text),status:'available',category:'owner-confirmed-feedback',scope:r.scope==='job'?`job:${r.jobId}`:'general candidate statement; conflicts unresolved',text};});}
export function bindFeedback(c:CandidateContext):CandidateContext {const fresh=feedbackSources(),old=c.sources.filter(s=>s.category==='owner-confirmed-feedback');if(JSON.stringify(old.map(s=>s.hash))===JSON.stringify(fresh.map(s=>s.hash)))return c;const payload={...c,sources:[...c.sources.filter(s=>s.category!=='owner-confirmed-feedback'),...fresh]};return {...payload,version:contextVersion(payload)};}
export function contextForRole(c:CandidateContext,id:number,resolveId=(n:number)=>n):CandidateContext {const sources=c.sources.filter(s=>!s.scope?.startsWith('job:')||resolveId(Number(s.scope.slice(4)))===id);if(sources.length===c.sources.length&&!c.sources.some(s=>s.category==='owner-confirmed-feedback'))return c;const result={...c,sources};return {...result,version:contextVersion(result)};}
export function saveFeedback(store:Store,jobId:number,text:string,scope='general',supersedes?:string){
 if(!text.trim()||text.length>20000)throw new Error('Enter 1–20,000 characters of candidate feedback');
 if(!['general','job'].includes(scope))throw new Error('Invalid feedback scope');
 const job=store.get(jobId),prior=feedbackRecords();
 if(supersedes&&!prior.some(r=>r.id===supersedes&&store.resolveId(r.jobId)===job.id))throw new Error('Unknown feedback revision');
 const id=randomUUID(),created=now(),facts=explicitFacts(text);
 const conflicts=prior.filter(r=>r.id!==supersedes&&(r.scope==='general'||store.resolveId(r.jobId)===job.id)).flatMap(r=>(r.facts??[]).filter((f:Fact)=>facts.some(n=>n.subject===f.subject&&n.value!==f.value)).map((f:Fact)=>({record:r.path,subject:f.subject,previous:f.value,resolution:'unresolved; ask owner, do not overwrite'})));
 const path=`${feedbackDirectory()}/${created.slice(0,10)}-${id}.md`;
 const record={id,created,path,jobId:job.id,jobUrl:job.originalUrl,jobTitle:job.title,source:'Owner clicked Save and reassess in local catalogue',scope,text,facts,conflicts,supersedes:supersedes??null};
 mkdirSync(rootPath(feedbackDirectory()),{recursive:true});
 const fence='`'.repeat(Math.max(3,...(text.match(/`+/g)??[]).map(s=>s.length+1)));
 // The file is the first durable write; it survives missing inference, DB failures and restart.
 writeFileSync(rootPath(path),`---\ntitle: "Candidate feedback — ${created.slice(0,10)}"\ntype: reference\nstatus: working\ntags: [candidate-feedback]\n---\n\n# Owner-confirmed candidate feedback\n\nRecorded ${created}. Source: owner confirmation in the local job catalogue for [${job.title.replace(/[\[\]]/g,'')}](${job.originalUrl}). Scope: ${scope==='general'?'general candidate knowledge':'this role only'}.\n\n## Original user statement (unchanged)\n\n${fence}text\n${text}\n${fence}\n\n## Grounded extraction and review\n\n${facts.length?facts.map(f=>`- ${f.subject}: ${f.value}. Original evidence: ${JSON.stringify(f.quote)}.`).join('\n'):'No structured facts extracted. The original statement remains available as attributed evidence; no model is required to save it.'}\n\n${conflicts.length?'Conflicting statements are unresolved: '+JSON.stringify(conflicts):'Earlier evidence remains intact. Missing mentions are unknown, not confirmed absence. Any disagreement with CV/KB evidence requires review.'}\n\n${supersedes?`This is an explicit revision of record ${supersedes}; the original is retained.`:'This record is additive; it does not overwrite earlier facts.'}\n\n## For AI / source map\n\n<!-- FCTO_RECORD ${Buffer.from(JSON.stringify(record)).toString('base64')} -->\n`,{flag:'wx',mode:0o600});
 store.db.prepare('INSERT INTO feedback VALUES(?,?,?,?,?)').run(id,job.id,created,path,JSON.stringify(record));
 const c=store.context();if(c)store.saveContext(bindFeedback(c));
 return record;
}
