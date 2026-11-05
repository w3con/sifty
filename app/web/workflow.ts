import type { Job } from '../src/core';
import type { Stage } from '../src/workflow';
import {managementTitle} from './filters';
export type View=Exclude<Stage,'no'>;
export type QuickFilters={web3:boolean;remote:boolean;strong:boolean;hideExpired:boolean;management:boolean};
export const emptyQuick=():QuickFilters=>({web3:false,remote:false,strong:false,hideExpired:false,management:false});
export function visibleInStage(j:Job,view:View,session:Set<number>) {return j.stage===view||(view==='triage'&&session.has(j.id)&&['preselected','no'].includes(j.stage));}
export function quickMatch(j:Job,q:QuickFilters,query:string) {
  return (!q.management||managementTitle(j.title))&&(!q.web3||j.isWeb3)&&(!q.remote||/remote|anywhere|telecommute/i.test(j.location??''))&&(!q.strong||j.matchScore>=75)&&(!q.hideExpired||!j.expired)&&`${j.title} ${j.company} ${(j.tags??[]).join(' ')} ${j.description}`.toLowerCase().includes(query.toLowerCase());
}
export function nextUndecided(jobs:Job[],id:number) {const i=jobs.findIndex(j=>j.id===id);return [...jobs.slice(i+1),...jobs.slice(0,i)].find(j=>j.stage==='triage')?.id??id;}
export function shortcutGuard(target:EventTarget|null) {return target instanceof Element&&!!target.closest('input,textarea,select,[contenteditable]:not([contenteditable="false"]),[role="textbox"],[role="switch"]');}
