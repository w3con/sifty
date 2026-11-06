import '../test/isolate.js';
import {mkdirSync,writeFileSync} from 'node:fs';import {chromium} from 'playwright';import {load} from 'cheerio';
import {publicUrl,fetchPage} from '../src/network.js';import {traffic} from '../src/traffic.js';import {localPath} from '../src/core.js';
const url='https://jobs.framework.ventures/companies/zzz-wlsn-probe-2/jobs/96713574-senior-smart-contract-engineer',dir=localPath('verification');mkdirSync(dir,{recursive:true});await publicUrl(url);
const browser=await chromium.launch(),context=await browser.newContext(),results:any[]=[];
try{
 const normal=await traffic.run(url,async()=>{const r=await fetch(url,{redirect:'manual'});return {status:r.status,type:r.headers.get('content-type'),body:await r.text()};});writeFileSync(`${dir}/framework-public.html`,normal.body);const $=load(normal.body);
 const data=JSON.parse($('#__NEXT_DATA__').text()),jobData:any[]=[];function walk(value:any,path=''){if(!value||typeof value!=='object')return;if(value.title&&/smart contract/i.test(String(value.title)))jobData.push({path,keys:Object.keys(value),url:value.url,applyUrl:value.applyUrl,source:value.source,organization:value.organization});for(const [key,child] of Object.entries(value))if(child&&typeof child==='object')walk(child,`${path}.${key}`);}walk(data);
 results.push({transport:'Node fetch default',status:normal.status,type:normal.type,title:$('title').text(),apply:$('a,button').map((_,a)=>({text:$(a).text().trim(),href:$(a).attr('href'),attrs:a.attribs})).get().filter(a=>/apply|postuler/i.test(a.text)),jobData,embeds:$('iframe').map((_,f)=>$(f).attr('src')).get()});
 const api=await traffic.run(url,async()=>{const r=await context.request.get(url,{maxRedirects:0});return {status:r.status(),type:r.headers()['content-type'],size:(await r.body()).length};});results.push({transport:'Playwright APIRequest default',...api});
 const ua=await context.newPage();const userAgent=await ua.evaluate(()=>navigator.userAgent);
 const nodeUA=await traffic.run(url,async()=>{const r=await fetch(url,{headers:{'user-agent':userAgent},redirect:'manual'});return {status:r.status,type:r.headers.get('content-type'),size:(await r.arrayBuffer()).byteLength};});results.push({transport:'Node fetch browser UA only',...nodeUA});
 try{const existing=await fetchPage(url);results.push({transport:'Existing validated fetchPage',status:existing.status,title:load(existing.text)('title').text()});}catch(e){results.push({transport:'Existing validated fetchPage',error:String(e)});}
 writeFileSync(`${dir}/framework-probe.json`,JSON.stringify(results,null,2));console.log(JSON.stringify({state:localPath(),results},null,2));
}finally{await browser.close();}
