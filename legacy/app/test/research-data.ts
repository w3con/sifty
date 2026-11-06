import './isolate.js';
import { readFileSync,writeFileSync } from 'node:fs';
import { load } from 'cheerio';
import { localPath } from '../src/core.js';
import { fetchPage } from '../src/network.js';
const tasks=process.argv.slice(2);
for(const task of tasks){
 try{
  if(task==='wttj-code'){
   const html=readFileSync(localPath('observations/wttj-fr.txt'),'utf8'),$=load(html);
   console.log('HTML config', [...html.matchAll(/.{0,80}(?:ALGOLIA|__ENV|env-config|PUBLIC_API|runtime-config).{0,180}/g)].slice(0,10).map(m=>m[0]));
   const sources=$('script[src]').map((_,e)=>$(e).attr('src')!).get();
   for(let offset=0;offset<sources.length;offset+=5)await Promise.all(sources.slice(offset,offset+5).map(async src=>{try{
    const js=(await fetchPage(new URL(src,'https://www.welcometothejungle.com').href)).text;
    writeFileSync(localPath(`observations/wttj-${src.split('/').at(-1)}`),js);
    if(/441788/.test(js)){const at=js.indexOf('441788');console.log('ENV helper candidate',src,js.slice(Math.max(0,at-300),at+1400));}
    const matches=[...js.matchAll(/.{0,100}(?:algolia|api\.welcometothejungle|algoliaApp|algoliaKey|searchClient:|APP_ID|apiKey:|appId:).{0,220}/gi)].slice(0,12).map(m=>m[0]);
    if(matches.length){writeFileSync(localPath(`observations/wttj-${src.split('/').at(-1)}`),js);console.log(JSON.stringify({src,matches:matches.map(s=>s.replace(/[a-f0-9]{24,}/g,'[public-key-redacted]'))}));}
   }catch(e){console.log(JSON.stringify({src,error:String(e)}));}}));
  }else if(task==='getro'){
   const p=await fetchPage('https://jobs.electriccapital.com/jobs'),$=load(p.text);writeFileSync(localPath('observations/getro-fetch.html'),p.text);
   for(const src of $('script[src]').map((_,e)=>$(e).attr('src')!).get().filter(s=>/chunks\/(pages\/(jobs|_app)|[0-9]+-)/.test(s))){
    const js=(await fetchPage(new URL(src,p.url).href)).text;writeFileSync(localPath(`observations/getro-${src.split('/').at(-1)}`),js);
    const snippets=[...js.matchAll(/.{0,100}(?:api\.getro|jobs\/search|\/search\/jobs|skip:|per_page|\/jobs\?|fetchJobs|searchJobs|api\.jobs|api\.getro|job_functions:|getJobs:|getJobs\().{0,240}/g)].slice(0,35).map(m=>m[0]);if(snippets.length)console.log(JSON.stringify({src,snippets}));
   }
  }else if(task==='a16z'){
   const p=await fetchPage('https://a16zfrontier.com/jobs/');const match=p.text.match(/const portfolioJobs\s*=\s*(\[.*?\]);/s);const data=match?JSON.parse(match[1]):null;
   console.log(JSON.stringify({task,embedded:!!data,companies:data?.length,jobs:data?.flatMap((v:any)=>v.jobs).length,first:data?.[0]?.jobs?.[0]}));
  }else if(task==='crypto-search'){
   const p=await fetchPage('https://cryptocurrencyjobs.co'),$=load(p.text),src=$('script[src]').map((_,e)=>$(e).attr('src')).get().find(s=>/\/search\.min/.test(s))!;
   const js=(await fetchPage(new URL(src,p.url).href)).text;writeFileSync(localPath('observations/cryptocurrency-search.js'),js);
   // Do not print public search keys; inspect mechanism/indices only.
   console.log(JSON.stringify({task,body:js.replace(/[a-f0-9]{20,}/g,'[public-key-redacted]').slice(0,8000)}));
  }else{
   const url=task==='braintrust-app'?'https://app.usebraintrust.com/jobs/':task==='wttj-root'?'https://www.welcometothejungle.com/':task==='wwr'?'https://weworkremotely.com/':task==='wellfound'?'https://wellfound.com/jobs':'https://web3.career/?search=engineer';
   const p=await fetchPage(url),$=load(p.text);writeFileSync(localPath(`observations/${task}-fetch.html`),p.text);
   console.log(JSON.stringify({task,url:p.url,title:$('title').text(),links:$('a[href]').map((_,e)=>({t:$(e).text().replace(/\s+/g,' ').slice(0,80),h:$(e).attr('href')})).get().filter(a=>/jobs|search|login|sign|rss|next|page=|engineer/i.test(`${a.h} ${a.t}`)).slice(0,70),scripts:$('script[src]').map((_,e)=>$(e).attr('src')).get().slice(-8),forms:$('form').map((_,e)=>({action:$(e).attr('action'),method:$(e).attr('method')})).get()}));
  }
 }catch(e){console.log(JSON.stringify({task,error:String(e)}));}
}
