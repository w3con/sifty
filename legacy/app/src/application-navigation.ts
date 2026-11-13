import {load} from 'cheerio';
export {concisePreparationError} from './preparation-error.js';
export type NavigationResponse={status:number;contentType:string;disposition?:string;url:string};
export function navigationKind(response:NavigationResponse){
 if(response.status===401)return 'login-required';
 if(response.status===404||response.status===410)return 'not-found';
 if(response.status>=400)return 'blocked';
 if(response.status>=300&&response.status<400)return 'redirect';
 if(/attachment/i.test(response.disposition??'')||/application\/(?:pdf|octet-stream)|(?:image|audio|video)\//i.test(response.contentType))return 'download';
 if(/(?:text\/html|application\/xhtml\+xml)/i.test(response.contentType))return 'html';
 return 'unsupported';
}
export function publicGetroPage(url:string){const u=new URL(url);return u.hostname==='jobs.framework.ventures'&&/^\/companies\/[^/]+\/jobs\/\d+-[^/]+\/?$/.test(u.pathname)&&!u.search;}
export function getroApplicationDestination(html:string,url:string):string|null{
 if(!publicGetroPage(url))return null;const $=load(html);
 try{const job=JSON.parse($('#__NEXT_DATA__').text()).props?.pageProps?.initialState?.jobs?.currentJob;const id=new URL(url).pathname.match(/\/jobs\/(\d+)-/)?.[1];
  if(!job||String(job.id)!==id||typeof job.url!=='string')return null;
  const target=new URL(job.url,url);return ['http:','https:'].includes(target.protocol)&&target.href!==url?target.href:null;
 }catch{return null;}
}
export function applicationLinks(html:string,url:string):string[]{
 const $=load(html),links:string[]=[];
 $('a[href],button[type=button][data-href],button[type=button][data-url]').each((_,el)=>{
  const e=$(el),label=`${e.text()} ${e.attr('aria-label')??''}`.trim();if(!/\bapply\b|postuler|candidature|bewerben/i.test(label))return;
  const raw=e.attr('href')??e.attr('data-href')??e.attr('data-url');if(!raw)return;
  try{const target=new URL(raw,url);if(!['http:','https:'].includes(target.protocol)||target.username||target.password)return;const before=new URL(url);before.hash='';const after=new URL(target);after.hash='';if(before.href!==after.href)links.push(target.href);}catch{}
 });
 return [...new Set(links)];
}
