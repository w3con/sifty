import type {StopWord} from './model/core';
// ApiError preserves the HTTP status and JSON body so callers can keep a draft on 409 and show
// the server's current revision, instead of losing the payload in a generic message.
export class ApiError extends Error {
  status:number; data:any;
  constructor(message:string,status:number,data:any){super(message);this.name='ApiError';this.status=status;this.data=data;}
}
// ServerUnreachableError is thrown when the browser cannot reach Sifty's own server at all
// (a network-level failure, not an HTTP error). Every browser raises this as a TypeError, but
// the message wording differs (Chrome: "Failed to fetch", Safari: "Load failed", Firefox:
// "NetworkError…"). The discriminant field `kind` is used for classification rather than
// `instanceof` because class-extends-Error is unreliable after some bundler transforms.
export class ServerUnreachableError extends Error {
  readonly kind = 'server-unreachable' as const;
  constructor(){super('server unreachable');this.name='ServerUnreachableError';}
}
// isServerUnreachable classifies any thrown value by the kind field, which survives
// bundler transforms that break instanceof for Error subclasses.
export function isServerUnreachable(e: unknown): e is ServerUnreachableError {
  return e != null && typeof e === 'object' && (e as any).kind === 'server-unreachable';
}
export async function api(path:string,method='GET',body?:unknown,timeoutMs?:number):Promise<any> {
  let r:Response;
  try{r=await fetch(`/api/${path}`,{method,signal:timeoutMs!=null?AbortSignal.timeout(timeoutMs):undefined,headers:method==='GET'?{}:{'content-type':'application/json'},body:body===undefined?undefined:JSON.stringify(body)});}
  catch(e){if(e instanceof TypeError||(e instanceof DOMException&&e.name==='TimeoutError'))throw new ServerUnreachableError();throw e;}
  if(!r.headers.get('content-type')?.includes('application/json'))throw new ApiError(`API /api/${path} returned HTTP ${r.status} without JSON. Refresh after restarting the catalogue server.`,r.status,undefined);
  const data=await r.json();
  // A fresh database has no non-done queue rows. Normalise that empty state at the API boundary
  // so every consumer receives an array instead of relying on optional chaining that stops at
  // `queue` and then crashes on `rows.find`.
  if(data&&typeof data==='object'&&'rows' in data&&data.rows==null)data.rows=[];
  if(!r.ok)throw new ApiError(data.error??`HTTP ${r.status}`,r.status,data);return data;
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
