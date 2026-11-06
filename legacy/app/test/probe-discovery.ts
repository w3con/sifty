import './isolate.js';
import { request } from 'undici';
import { readFileSync,writeFileSync } from 'node:fs';
import { localPath } from '../src/core.js';
import { fetchPage } from '../src/network.js';
for(const task of process.argv.slice(2))try{
 if(task==='getro')for(const page of [0,1]){
  const url='https://api.getro.com/api/v2/collections/1640/search/jobs';
  const r=await request(url,{method:'POST',headers:{'content-type':'application/json','accept':'application/json','origin':'https://jobs.electriccapital.com','referer':'https://jobs.electriccapital.com/jobs'},body:JSON.stringify({hits_per_page:100,page,filters:'',query:''}),signal:AbortSignal.timeout(20000)});const raw=await r.body.text();console.log(JSON.stringify({task,page,http:r.statusCode,location:r.headers.location,rawPreview:raw.slice(0,300)}));if(!raw)continue;const b=JSON.parse(raw);writeFileSync(localPath(`observations/getro-api-${page}.json`),JSON.stringify(b));
  console.log(JSON.stringify({task,page,status:r.statusCode,count:b.results?.count,returned:b.results?.jobs?.length,first:b.results?.jobs?.[0],keys:Object.keys(b)}).slice(0,7000));
 }else if(task==='algolia'){
  const js=readFileSync(localPath('observations/cryptocurrency-search.js'),'utf8'),match=js.match(/algoliasearch\("([^"]+)","([^"]+)"\)/)!;
  const url=`https://${match[1].toLowerCase()}-dsn.algolia.net/1/indexes/*/queries`;
  const r=await request(url,{method:'POST',headers:{'content-type':'application/json','referer':'https://cryptocurrencyjobs.co/','origin':'https://cryptocurrencyjobs.co','x-algolia-application-id':match[1],'x-algolia-api-key':match[2]},body:JSON.stringify({requests:[{indexName:'jobs',params:'filters=searchFilter%3Aengineering&hitsPerPage=100&page=0&query='},{indexName:'jobs',params:'filters=searchFilter%3AIsHome&hitsPerPage=100&page=0&query='}]}),signal:AbortSignal.timeout(20000)});const b=await r.body.json() as any;writeFileSync(localPath('observations/cryptocurrency-api.json'),JSON.stringify(b));console.log(JSON.stringify({task,status:r.statusCode,results:b.results?.map((v:any)=>({total:v.nbHits,pages:v.nbPages,hits:v.hits?.length,first:v.hits?.[0]}))}).slice(0,6000));
 }else{
  const url=task==='braintrust-detail'?'https://app.usebraintrust.com/api/jobs/18096/':task==='wellfound-engineering'?'https://wellfound.com/role/l/software-engineer/united-states':task==='wwr-all'?'https://weworkremotely.com/remote-jobs.rss':task==='malt-fetch'?'https://www.malt.fr/':'https://www.welcometothejungle.com/fr/jobs';
  const p=await fetchPage(url);writeFileSync(localPath(`observations/${task}.txt`),p.text);console.log(JSON.stringify({task,url:p.url,size:p.text.length,text:p.text.slice(0,1500)}));
 }
}catch(e){console.log(JSON.stringify({task,error:String(e)}));}
