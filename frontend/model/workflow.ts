export type Stage = 'triage' | 'preselected' | 'no' | 'applied';
export type OutcomeReason = 'stale' | 'closed' | 'not_suitable';
export function outcomeReasonLabel(reason: string | null | undefined): string {
  if (reason === 'stale') return 'Vacancy is stale';
  if (reason === 'closed') return 'Vacancy is closed';
  if (reason === 'not_suitable') return 'Not suitable';
  return '';
}
export const archiveRetentionMs=30*24*60*60*1000;
export function validAppliedAt(value:unknown):value is string {if(typeof value!=='string'||!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/.test(value))return false;const time=Date.parse(value);return Number.isFinite(time)&&new Date(time).toISOString()===value;}
const aliases: Record<string, Stage> = {new:'triage',saved:'preselected',rejected:'no',shortlist:'preselected',triage:'triage',preselected:'preselected',no:'no',applied:'applied'};
export function stageFor(status: string): Stage {
  const stage=aliases[status];
  if(!stage) throw new Error('Invalid workflow stage');
  return stage;
}
export const statusFor=(stage:Stage)=>({triage:'new',preselected:'saved',no:'rejected',applied:'applied'}[stage]);
export function isWeb3(job:{title:string;description:string;tags?:string[]}) {
  return /\b(web\s?3|blockchains?|crypto(?:currency|currencies)?|defi|solidity|ethereum|bitcoin|smart\s+contracts?|nfts?|evm|solana|polkadot|on[ -]chain|decentralized\s+finance)\b/i.test(`${job.title} ${(job.tags??[]).join(' ')} ${job.description}`);
}
export function externalApplyUrl(job:{applyUrl?:string|null;originalUrl:string}):string|null {
  for(const value of [job.applyUrl,job.originalUrl]) {
    try {const url=new URL(value??'');if(['https:','http:'].includes(url.protocol)&&!url.username&&!url.password)return url.href;}catch{}
  }
  return null;
}
