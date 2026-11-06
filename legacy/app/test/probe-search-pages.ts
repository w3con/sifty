import './isolate.js';import {load} from 'cheerio';import{readFileSync,writeFileSync}from'node:fs';import{request}from'undici';import{localPath}from'../src/core.js';import{fetchPage}from'../src/network.js';
for(const task of process.argv.slice(2))try{
 if(task==='wttj'){
  const html=readFileSync(localPath('observations/wttj-fr.txt'),'utf8').replace(/\\"/g,'"');const env:Record<string,string>={};for(const m of html.matchAll(/"(PUBLIC_ALGOLIA_[A-Z_]+)"\s*:\s*"([^"]+)"/g))env[m[1]]=m[2];
  console.log('WTTJ public configuration keys',Object.keys(env),'index',env.PUBLIC_ALGOLIA_JOBS_INDEX);
  const $=load(html);console.log('Environment bootstrap',$('script').map((_,e)=>$(e).text()).get().filter(s=>/window\.env|atob\(|environment|env=/.test(s)).map(s=>s.slice(0,1800)).slice(0,3));
  const app=env.PUBLIC_ALGOLIA_APPLICATION_ID,key=env.PUBLIC_ALGOLIA_API_KEY_CLIENT,index=env.PUBLIC_ALGOLIA_JOBS_INDEX;
  if(app&&key&&index){const url=`https://${app.toLowerCase()}-dsn.algolia.net/1/indexes/*/queries`;const r=await request(url,{method:'POST',headers:{accept:'application/json','content-type':'application/json',referer:'https://www.welcometothejungle.com/fr/jobs',origin:'https://www.welcometothejungle.com','x-algolia-application-id':app,'x-algolia-api-key':key},body:JSON.stringify({requests:[{indexName:index,params:'query=engineer&hitsPerPage=100&page=0'}]}),signal:AbortSignal.timeout(20000)});const b=await r.body.json()as any;writeFileSync(localPath('observations/wttj-api.json'),JSON.stringify(b));console.log(JSON.stringify({status:r.statusCode,results:b.results?.map((v:any)=>({total:v.nbHits,first:v.hits?.[0]})),error:b.message}).slice(0,8000));}
 }else if(task==='wellfound')for(const path of ['/role/software-engineer','/role/software-engineer?page=2','/role/product-manager']){
  const p=await fetchPage(`https://wellfound.com${path}`),$=load(p.text),data=JSON.parse($('#__NEXT_DATA__').text()).props.pageProps.apolloState?.data;
  const results=Object.values(data?.ROOT_QUERY?.talent??{}).find((v:any)=>v?.__typename==='Results')as any;
  const company=Object.values(data??{}).find((v:any)=>v?.__typename==='StartupResult')as any;
  console.log(JSON.stringify({path,url:p.url,total:results?.totalJobCount,pages:results?.pageCount,companies:results?.startups?.slice(0,2),company,firstJob:Object.values(data??{}).find((v:any)=>v?.__typename==='JobListingSearchResult')}).slice(0,6000));writeFileSync(localPath(`observations/wellfound-${path.includes('page=2')?'p2':path.split('/').at(-1)}.html`),p.text);
 }else{
 const p=await fetchPage(task),$=load(p.text);console.log(JSON.stringify({url:p.url,rows:$('tr[data-jobid]').length,first:$('tr[data-jobid]').first().attr('data-jobid'),next:$('a[rel="next"]').attr('href'),title:$('title').text()}));}
}catch(e){console.log(task,String(e));}
