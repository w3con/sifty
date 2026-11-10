import type { Job } from './model/core';
import type { Stage } from './model/workflow';
import {managementTitle} from './filters';
export type View=Exclude<Stage,'no'>|'collection';
// fit: score buckets of the instant match score (matchScore). fresh: collected within N days ('' = any).
// strong is the older single "Strong fit (75+)" switch; it is still honoured and migrated to fit.
export type FitBucket='strong'|'partial'|'weak'|'unscored';
export type QuickFilters={web3:boolean;remote:boolean;strong:boolean;hideExpired:boolean;management:boolean;fit:FitBucket[];fresh:''|'7'|'30'};
export const emptyQuick=():QuickFilters=>({web3:false,remote:false,strong:false,hideExpired:false,management:false,fit:[],fresh:''});
export const fitBucket=(score:number):FitBucket=>score<=0?'unscored':score>=75?'strong':score>=55?'partial':'weak';
export function normalizeQuick(saved:Partial<QuickFilters>):QuickFilters{
  const q={...emptyQuick(),...saved};
  if(!Array.isArray(q.fit))q.fit=[];
  if(q.strong&&!q.fit.includes('strong'))q.fit=[...q.fit,'strong'];
  q.strong=false;
  if(!['','7','30'].includes(q.fresh))q.fresh='';
  return q;
}
export const quickCount=(q:QuickFilters)=>[q.web3,q.remote,q.hideExpired,q.management,q.fit.length>0,q.fresh!==''].filter(Boolean).length;
export function visibleInStage(j:Job,view:View,session:Set<number>) {return view!=='collection'&&(j.stage===view||(view==='triage'&&session.has(j.id)&&['preselected','no'].includes(j.stage)));}
export function quickMatch(j:Job,q:QuickFilters,query:string) {
  return (!q.management||managementTitle(j.title))&&(!q.web3||j.isWeb3)&&(!q.remote||/remote|anywhere|telecommute/i.test(j.location??''))&&(!q.strong||j.matchScore>=75)&&(!q.fit?.length||q.fit.includes(fitBucket(j.matchScore)))&&(!q.fresh||Date.now()-Date.parse(j.firstSeen)<=Number(q.fresh)*86400000)&&(!q.hideExpired||!j.expired)&&`${j.title} ${j.company} ${(j.tags??[]).join(' ')} ${j.description}`.toLowerCase().includes(query.toLowerCase());
}
export function nextUndecided(jobs:Job[],id:number) {const i=jobs.findIndex(j=>j.id===id);return [...jobs.slice(i+1),...jobs.slice(0,i)].find(j=>j.stage==='triage')?.id??id;}
export function shortcutGuard(target:EventTarget|null) {return target instanceof Element&&!!target.closest('input,textarea,select,[contenteditable]:not([contenteditable="false"]),[role="textbox"],[role="switch"]');}
