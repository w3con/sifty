import type { Job } from './model/core';
import { publicationDate } from './dates';
export const normalizeTerm=(s:string)=>s.toLowerCase().replace(/node\.?js/g,'nodejs').replace(/web3\.?js/g,'web3js').replace(/\blayer\s*2\b/g,'l2').replace(/\bchief technology officer\b/g,'cto').replace(/\btechnical lead\b/g,'tech lead').replace(/\bartificial intelligence\b/g,'ai').replace(/[^\p{L}\p{N}]+/gu,' ').trim();
export const termMatch=(text:string,q:string)=>` ${normalizeTerm(text)} `.includes(` ${normalizeTerm(q)} `);
export type CatalogueSelection={titles:string[]|null;keywords:string[]|null;sources:string[];assessment:string[]};
export const emptySelection=():CatalogueSelection=>({titles:null,keywords:null,sources:[],assessment:[]});
// Legacy empty title/keyword lists were unrestricted; v2 [] explicitly means none.
export function migrateSelection(saved:any,version:number):CatalogueSelection {const result={...emptySelection(),...saved};if(version<2)for(const key of ['titles','keywords'] as const)if(!result[key]?.length)result[key]=null;return result;}
export const managementTitle=(title:string)=>/\b(?:cto|chief\s+technology\s+officer|head|director|manager)\b/i.test(title)||/\blead\b(?!\s+(?:generation|gen)\b)/i.test(title);
export const selectionCount=(s:CatalogueSelection)=>Object.values(s).reduce((n,v)=>n+(v===null?0:Math.max(1,v.length)),0)-(!s.sources.length?1:0)-(!s.assessment.length?1:0);
// Newest uses the source publication date, not the discovery time. Unknown or
// invalid publication dates sort after every known date, stably by id.
export function sortCatalogue(jobs:Job[],sort:'fit'|'newest'){const pub=(j:Job)=>{const t=publicationDate(j.publishedAt)?.getTime()??null;return{rank:t===null?2:(j.publishedBasis?0:1),t:t??0};};return [...jobs].sort((a,b)=>{if(sort!=='newest')return b.matchScore-a.matchScore||Date.parse(b.firstSeen)-Date.parse(a.firstSeen)||a.id-b.id;const pa=pub(a),pb=pub(b);return pa.rank-pb.rank||(pa.rank<2?pb.t-pa.t:0)||a.id-b.id;});}
export function assessmentStatus(j:Job,rows:any[]=[]){const row=rows.find(r=>r.job_id===j.id);if(row&&['running','failed','missing-description','pending'].includes(row.status))return row.status;return j.assessment?(j.assessment.stale?'stale':'assessed'):'pending';}
export function filterCatalogue(jobs:Job[],s:CatalogueSelection,rows:any[]=[]){return jobs.filter(j=>(s.titles===null||s.titles.some(q=>termMatch(j.title,q)))&&(s.keywords===null||s.keywords.some(q=>termMatch(`${j.title} ${(j.tags??[]).join(' ')} ${j.description}`,q)))&&(!s.sources.length||j.sources.some(src=>s.sources.includes(src.name)))&&(!s.assessment.length||s.assessment.includes(assessmentStatus(j,rows))));}
