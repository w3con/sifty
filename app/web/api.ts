import type {StopWord} from '../src/core';
export async function api(path:string,method='GET',body?:unknown):Promise<any> {
  const r=await fetch(`/api/${path}`,{method,headers:method==='GET'?{}:{'content-type':'application/json'},body:body===undefined?undefined:JSON.stringify(body)});
  if(!r.headers.get('content-type')?.includes('application/json'))throw new Error(`API /api/${path} returned HTTP ${r.status} without JSON. Refresh after restarting the catalogue server.`);
  const data=await r.json();if(!r.ok)throw new Error(data.error??`HTTP ${r.status}`);return data;
}
export const blockJob=(id:number,dimension:'company'|'source')=>api(`jobs/${id}/block`,'POST',{dimension});
export const listStopWords=():Promise<StopWord[]>=>api('stop-words');
export const addStopWord=(word:string):Promise<StopWord>=>api('stop-words','POST',{word});
export const removeStopWord=(id:number)=>api(`stop-words/${id}`,'DELETE');
export const previewStopWord=(word:string):Promise<{word:string;count:number}>=>api(`stop-words/preview?word=${encodeURIComponent(word)}`);
export type SearchExpression={or?:string[];and?:string[]};
export type SiteConfig={name:string;url:string;enabled?:boolean};
export const getSearchExpressions=():Promise<SearchExpression[]>=>api('search/expressions');
export const putSearchExpressions=(expressions:SearchExpression[]):Promise<{expressions:SearchExpression[]}>=>api('search/expressions','PUT',{expressions});
export const getSearchSites=():Promise<SiteConfig[]>=>api('search/sites');
export const addSearchSite=(name:string,url:string,enabled=true):Promise<SiteConfig>=>api('search/sites','POST',{name,url,enabled});
export const removeSearchSite=(name:string):Promise<unknown>=>api(`search/sites/${encodeURIComponent(name)}`,'DELETE');
export const toggleSearchSite=(name:string,enabled:boolean):Promise<unknown>=>api(`search/sites/${encodeURIComponent(name)}`,'PATCH',{enabled});
