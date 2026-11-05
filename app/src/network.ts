import { lookup } from 'node:dns/promises';
import { isIP } from 'node:net';
import { request, ProxyAgent } from 'undici';
import { chromium, type BrowserContext } from 'playwright';
import { z } from 'zod';
import { hash, localPath, runtime, yaml } from './core.js';
import { traffic, pageCache, httpError } from './traffic.js';

export const networkSchema = z.object({ version: z.literal(1), profiles: z.record(z.string(), z.object({ proxy_env: z.string().regex(/^[A-Z_][A-Z0-9_]*$/).optional(), browser: z.boolean().optional() }).strict()), sources: z.record(z.string(), z.string()) });
export function networkProfile(source: string,config: unknown = yaml('network'),env: NodeJS.ProcessEnv = process.env) {
  const c = networkSchema.parse(config);
  const name = c.sources[source] ?? 'direct', p = c.profiles[name];
  if (!p) throw new Error(`Missing network profile ${name}`);
  const proxy = p.proxy_env ? env[p.proxy_env] : undefined;
  if (p.proxy_env && !proxy) throw new Error(`Set proxy environment reference ${p.proxy_env}`);
  return { ...p, proxy };
}
function privateIP(ip: string): boolean {
  if (ip.includes(':')) return /^(::|fc|fd|fe[89ab]|::ffff:)/i.test(ip);
  const a = ip.split('.').map(Number);
  return a[0] === 0 || a[0] === 10 || a[0] === 127 || a[0] >= 224 || (a[0] === 169 && a[1] === 254) || (a[0] === 172 && a[1] >= 16 && a[1] <= 31) || (a[0] === 192 && a[1] === 168);
}
export async function publicUrl(value: string) {
  const u = new URL(value);
  if (!['http:','https:'].includes(u.protocol) || u.username || u.password) throw new Error('Public HTTP(S) URL required, without credentials');
  const host = u.hostname.replace(/^\[|\]$/g, '');
  const ips = isIP(host) ? [{ address: host }] : await lookup(host, { all: true });
  if (!ips.length || ips.some(i => privateIP(i.address))) throw new Error('Private network destinations are not permitted for remote sources');
  return u;
}
export type Page = { text: string; url: string; status: number; contentType?:string; navigationHeaders?:Record<string,string>; etag?:string; modified?:string; cached?:boolean };
export function isReadOnlySearch(url:string,body:unknown):boolean {
  const u=new URL(url),b=body as any;
  if(u.hostname==='api.getro.com'&&/^\/api\/v2\/collections\/\d+\/search\/jobs$/.test(u.pathname))return b&&Object.keys(b).every(k=>['hits_per_page','page','filters','query'].includes(k));
  if(/^[a-z0-9-]+\.algolia\.net$/.test(u.hostname)&&u.pathname==='/1/indexes/*/queries')return Array.isArray(b?.requests)&&b.requests.every((r:any)=>typeof r.indexName==='string'&&typeof r.params==='string'&&Object.keys(r).every(k=>['indexName','params'].includes(k)));
  return false;
}
// These observed public endpoints use POST for read-only search. No other POST is exposed.
export async function postSearch(url:string,body:unknown,source:string,headers:Record<string,string>={}):Promise<Page>{
  if(!isReadOnlySearch(url,body))throw new Error('Only verified public read-only search POST endpoints are allowed');
  await publicUrl(url);const p=networkProfile(source),dispatcher=p.proxy?new ProxyAgent(p.proxy):undefined;
   try{return await traffic.run(url,async()=>{
    const r=await request(url,{method:'POST',dispatcher,signal:AbortSignal.timeout(runtime().request_timeout_ms),headers:{accept:'application/json','content-type':'application/json',...headers},body:JSON.stringify(body)});
    let text='';for await(const b of r.body){text+=b.toString();if(text.length>8_000_000){r.body.destroy();throw new Error('Search response exceeds 8MB');}}
     if(r.statusCode>=400)throw httpError(r.statusCode,r.headers['retry-after']);return {text,url,status:r.statusCode};
   });
  }finally{await dispatcher?.close();}
}
export async function fetchPage(url: string, source = '', timeout = runtime().request_timeout_ms, p = networkProfile(source), conditional:Record<string,string>={},signal?:AbortSignal): Promise<Page> {
  signal?.throwIfAborted();
  if (p.browser) return browserPage(url, source);
  let current = url;
  const dispatcher = p.proxy ? new ProxyAgent(p.proxy) : undefined;
   try {
     for (let n = 0; n < 6; n++) {
        signal?.throwIfAborted();await publicUrl(current);
       const result=await traffic.run(current,async()=>{
        signal?.throwIfAborted();const r = await request(current, { method: 'GET', dispatcher, signal: signal?AbortSignal.any([signal,AbortSignal.timeout(timeout)]):AbortSignal.timeout(timeout), headersTimeout: timeout, bodyTimeout: timeout, headers: { 'user-agent': 'FCTO-Catalogue/1.0 (personal read-only job discovery)', accept: 'text/html,application/json,application/rss+xml,application/xml;q=0.9,*/*;q=0.5',...conditional } });
       if ([301,302,303,307,308].includes(r.statusCode) && r.headers.location) {await r.body.dump();return {redirect:new URL(String(r.headers.location),current).href};}
      let text = '';
      for await (const chunk of r.body) { text += chunk.toString(); if (Buffer.byteLength(text) > 8_000_000) { r.body.destroy(); throw new Error('Page exceeds 8MB limit'); } }
       if (r.statusCode >= 400) throw httpError(r.statusCode,r.headers['retry-after']);
      if (/verify you are human|cf-chl-|captcha-container|just a moment\.\.\./i.test(text.slice(0,40000))) throw new Error('Access challenge/CAPTCHA; owner action required');
       const navigationHeaders=Object.fromEntries(['content-type','content-security-policy','content-security-policy-report-only','x-frame-options','referrer-policy'].flatMap(key=>r.headers[key]===undefined?[]:[[key,String(r.headers[key])]]));
       return {page:{ text, url: current, status: r.statusCode,contentType:r.headers['content-type'] as string|undefined,navigationHeaders,etag:r.headers.etag as string|undefined,modified:r.headers['last-modified'] as string|undefined }};
       });if(result.redirect){current=result.redirect;continue;}return result.page!;
    }
    throw new Error('Too many redirects');
   } finally { await dispatcher?.close(); }
 }
export async function detailPage(url:string,source=''):Promise<Page>{
 const key=`detail:${hash(JSON.stringify({url,profile:networkProfile(source)}))}`;
 return pageCache.get(key,24*3600000,headers=>fetchPage(url,source,runtime().request_timeout_ms,networkProfile(source),headers),false,true);
}
export async function listingPage(url:string,source='',force=false):Promise<Page>{
 const key=`listing:${hash(JSON.stringify({url,profile:networkProfile(source)}))}`;
 return pageCache.get(key,3*3600000,()=>fetchPage(url,source),force);
}
export async function listingSearch(url:string,body:unknown,source:string,headers:Record<string,string>={},force=false):Promise<Page>{
 if(!isReadOnlySearch(url,body))throw new Error('Only verified public read-only search POST endpoints are allowed');
 const key=`query:${hash(JSON.stringify({url,body,headers,profile:networkProfile(source)}))}`;
 return pageCache.get(key,3*3600000,()=>postSearch(url,body,source,headers),force);
}
export function sessionDirectory(source: string) { return localPath(`sessions/${source.replace(/[^a-z0-9-]/gi, '_')}`); }
export async function browserSession(source: string, headed = false): Promise<BrowserContext> {
  const p = networkProfile(source);
  const proxy = p.proxy ? new URL(p.proxy) : null;
  return chromium.launchPersistentContext(sessionDirectory(source), { headless: !headed,
    proxy: proxy ? { server: `${proxy.protocol}//${proxy.host}`, username: decodeURIComponent(proxy.username), password: decodeURIComponent(proxy.password) } : undefined,
    acceptDownloads: false, serviceWorkers: 'block' });
}
export async function browserPage(url: string, source: string): Promise<Page> {
  await publicUrl(url);
  const context = await browserSession(source);
  try {
    await context.route('**/*', async route => {
       if (!['GET','HEAD'].includes(route.request().method())||!['document','xhr','fetch','script'].includes(route.request().resourceType())) return route.abort();
       try { const url=route.request().url();await publicUrl(url);await traffic.run(url,async()=>{const r=await route.fetch({maxRedirects:0,timeout:runtime().request_timeout_ms});if(r.status()>=400)throw httpError(r.status(),r.headers()['retry-after']);const body=await r.body();if(/cf-chl-|captcha-container|verify you are human/i.test(body.toString().slice(0,40000)))throw new Error('CAPTCHA: owner session required');await route.fulfill({response:r,body});}); } catch { await route.abort(); }
    });
    const page = await context.newPage();
    const r = await page.goto(url, { timeout: runtime().request_timeout_ms, waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1200);
    const text = await page.content();
    if (!r || r.status() >= 400 || /captcha|verify you are human|just a moment/i.test(await page.title())) {
      const gate=/cloudfront/i.test(text)?'CloudFront':/cloudflare|cdn-cgi/i.test(text)?'Cloudflare':'Site access check';
      throw new Error(`${gate}: browser blocked (${r?.status()}); manual session setup required`);
    }
    return { text, url: page.url(), status: r.status() };
  } finally { await context.close(); }
}
