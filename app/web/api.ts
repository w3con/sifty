import type {StopWord} from '../src/core';
export async function api(path:string,method='GET',body?:unknown):Promise<any> {
  const r=await fetch(`/api/${path}`,{method,headers:method==='GET'?{}:{'content-type':'application/json'},body:body===undefined?undefined:JSON.stringify(body)});
  if(!r.headers.get('content-type')?.includes('application/json'))throw new Error(`API /api/${path} returned HTTP ${r.status} without JSON. Refresh after restarting the catalogue server.`);
  const data=await r.json();if(!r.ok)throw new Error(data.error??`HTTP ${r.status}`);return data;
}
export const blockJob=(id:number,dimension:'company'|'source')=>api(`jobs/${id}/block`,'POST',{dimension});
export const listBlockRules=()=>api('block-rules');
export const listSources=():Promise<{name:string;roles:number;blocked:boolean}[]>=>api('sources');
export const addSourceBlock=(value:string)=>api('block-rules','POST',{kind:'source',value});
export const removeBlockRule=(id:number)=>api(`block-rules/${id}`,'DELETE');
export const listStopWords=():Promise<StopWord[]>=>api('stop-words');
export const addStopWord=(word:string):Promise<StopWord>=>api('stop-words','POST',{word});
export const removeStopWord=(id:number)=>api(`stop-words/${id}`,'DELETE');
export const previewStopWord=(word:string):Promise<{word:string;count:number}>=>api(`stop-words/preview?word=${encodeURIComponent(word)}`);
