import { load } from 'cheerio';
import { canonical, hash, matchQueries, normalize, now, runtime, searchConfig, type JobInput, type Source } from './core.js';
import { cleanHtml } from './context.js';
import { browserPage, fetchPage, detailPage, listingPage, listingSearch, postSearch, type Page } from './network.js';
import { detailFromHtml, htmlLinks, parseFeed, structuredJobs } from './collectors.js';

export type Fetcher=(url:string,source?:string)=>Promise<Page>;
export type SearchPoster=(url:string,body:unknown,source:string,headers?:Record<string,string>)=>Promise<Page>;
export type PageReceipt={url:string;page:number;returned:number;newRecords:number;fingerprint:string;first:string|null;last:string|null};
export type GroupReceipt={label:string;url:string;pages:number;reviewed:number;total:number|null;complete:boolean;stopReason:string};
export type Coverage={adapter:string;started:string;finished?:string;phase:string;reviewed:number;matched:number;imported:number;added:number;updated:number;duplicates:number;merged:number;listingDuplicates:number;inventoryTotal:number|null;networkRequests:number;browserNavigations:number;detailRequests:number;detailFetched:number;detailFailed:number;descriptionAvailable?:number;descriptionDeferred?:number;descriptionFailed?:number;descriptionUnreviewed:number;total:number|null;queries:string[];queryMatches:Record<string,number>;queryApplication:string;groups:GroupReceipt[];pages:PageReceipt[];stopReason:string;completeness:string;blocker?:string;action?:string;errors:string[]};
export type DiscoveryResult={jobs:JobInput[];status:string;message:string;coverage:Coverage};
export type DiscoveryOptions={get?:Fetcher;post?:SearchPoster;browser?:(url:string,source:string)=>Promise<Page>;progress?:(c:Coverage)=>void;limits?:Partial<ReturnType<typeof runtime>>;force?:boolean;inventory?:{jobs:JobInput[];coverage:Coverage};saveInventory?:(jobs:JobInput[],coverage:Coverage)=>void;onJob?:(job:JobInput)=>void};
export function emptyCoverage():Coverage{return {adapter:'identifying',started:now(),phase:'discovering',reviewed:0,matched:0,imported:0,added:0,updated:0,duplicates:0,merged:0,listingDuplicates:0,inventoryTotal:null,networkRequests:0,browserNavigations:0,detailRequests:0,detailFetched:0,detailFailed:0,descriptionUnreviewed:0,total:null,queries:[...searchConfig().search.titles,...searchConfig().search.keywords],queryMatches:{},queryApplication:'Not yet applied',groups:[],pages:[],stopReason:'',completeness:'running',errors:[]};}
const abs=(value:string,base:string)=>new URL(value,base).href;
const plain=(s:string)=>cleanHtml(`<body>${s}</body>`);
const hasDescription=(j:JobInput)=>j.description.length>=100&&!/metadata|pending|missing|unavailable/i.test(j.descriptionStatus);
const technical=(j:JobInput)=>/engineer|develop|architect|technical|\bcto\b|project|program|platform|software|blockchain|smart.contract|\bai\b|\bllm\b/i.test(j.title);
const queryText=(j:JobInput)=>`${j.title} ${(j.tags??[]).join(' ')} ${j.description}`;

export function parseGetroJobs(rows:any[],base:string,source:string):JobInput[]{
 return rows.filter(j=>j.title&&j.organization?.name&&j.slug&&j.organization.slug).map(j=>({title:j.title,company:'Unknown',boardCompany:j.organization.name,originalUrl:abs(`/companies/${j.organization.slug}/jobs/${j.slug}`,base),applyUrl:j.url??null,applyStatus:'source-provided destination; employer not yet verified',description:(j.skills??[]).join(', '),descriptionStatus:`indexed skills metadata; board association: ${j.organization.name}; employer not verified`,location:(j.locations??[]).join(' · '),publishedAt:j.created_at?new Date(j.created_at*1000).toISOString():undefined,source}));
}
export function parsePortfolioJobs(html:string,base:string,source:string):JobInput[]{
 const raw=html.match(/const portfolioJobs\s*=\s*(\[.*?\]);/s)?.[1];if(!raw)throw new Error('Adapter incomplete: portfolioJobs JSON no longer present');
 const companies=JSON.parse(raw);if(!Array.isArray(companies))throw new Error('Invalid portfolio job payload');
 return companies.flatMap((c:any)=>(c.jobs??[]).filter((j:any)=>j.title&&j.url&&j.companyName).map((j:any)=>({title:j.title,company:j.companyName,originalUrl:abs(j.url,base),applyUrl:abs(j.url,base),applyStatus:'portfolio employer link; reachability not checked',description:(j.skills??[]).join(', '),descriptionStatus:'indexed skills metadata; full description pending',location:(j.locations??[]).map((v:any)=>typeof v==='string'?v:v.name).filter(Boolean).join(' · '),publishedAt:j.createdAt,source})));
}
export function parseWellfound(html:string,base:string,source:string):{jobs:JobInput[];pages:number;total:number|null}{
 const $=load(html),data=JSON.parse($('#__NEXT_DATA__').text()).props?.pageProps?.apolloState?.data;
 const result=Object.values(data?.ROOT_QUERY?.talent??{}).find((v:any)=>v?.__typename==='Results')as any;
 if(!result)throw new Error('Adapter incomplete: Wellfound public role search payload missing');
 const jobs:JobInput[]=[];
 for(const ref of result.startups??[]){const c=data[ref.__ref];if(!c?.name)continue;
  for(const link of c.highlightedJobListings??[]){const j=data[link.__ref];if(!j?.id||!j.title||!j.slug)continue;
   jobs.push({title:j.title,company:c.name,originalUrl:abs(`/jobs/${j.id}-${j.slug}`,base),applyUrl:null,applyStatus:'not resolved: Wellfound application may require account',description:plain(j.description??''),descriptionStatus:j.description?'available public search description':'missing public description',location:[...(j.locationNames??[]),...(j.acceptedRemoteLocationNames??[])].join(' · '),publishedAt:j.liveStartAt?new Date(j.liveStartAt*1000).toISOString():undefined,source});
  }
 }
 return {jobs,pages:Number(result.pageCount)||1,total:typeof result.totalJobCount==='number'?result.totalJobCount:null};
}
export function parseWeb3(html:string,base:string,source:string):JobInput[]{
 const $=load(html);return $('tr[data-jobid]').map((_,el)=>{
  const row=$(el),a=row.find('a[href]').has('h2').first(),href=a.attr('href');if(!href)return null;
  return {title:a.find('h2').text().trim(),company:row.find('h3').first().text().trim()||'Unknown',originalUrl:abs(href,base),applyUrl:null,description:row.find('.cell-tags').text().replace(/\s+/g,' ').trim(),descriptionStatus:'listing tags metadata; full description pending',location:row.find('.job-location-mobile').text().replace(/📍/g,'').trim(),publishedAt:row.find('time').attr('datetime'),source};
 }).get().filter(Boolean)as JobInput[];
}
export function parsePublicHtml(html:string,base:string,source:string):JobInput[]{
 const $=load(html);
 if($('tr[data-jobid]').length)return parseWeb3(html,base,source);
 const jobs=structuredJobs(html,base,source);if(jobs.length)return jobs;
 // Paradigm's full public portfolio list: title and company are distinct accessible fields.
 if($('a.job-link').length)return $('a.job-link[href]').map((_,el)=>{
  const a=$(el),aria=a.attr('aria-label')??'',parts=aria.split(' – '),company=parts.length>1?parts.shift()!:(a.find('.company-name,.job-company').first().text().trim()||'Unknown');
  const title=parts.join(' – ')||a.find('.job-title,.job-title-text').first().text().trim();
  if(!title)return null;return {title,company,originalUrl:abs(a.attr('href')!,base),applyUrl:null,description:'',descriptionStatus:'pending employer description',source};
 }).get().filter(Boolean)as JobInput[];
 // Strict role-link fallback; never treat /jobs/search, directories or location pages as roles.
 return htmlLinks(html,base,source).filter(j=>/\/(?:jobs|remote-jobs)\/\d+[-/]|\/companies\/[^/]+\/jobs\/\d+|\/(engineering|product|operations|other|marketing|sales)\/[^/]+\/?$/.test(new URL(j.originalUrl).pathname)||/^(jobs\.ashbyhq\.com|jobs\.lever\.co|job-boards\.greenhouse\.io|boards\.greenhouse\.io)$/.test(new URL(j.originalUrl).hostname));
}
export async function hydrateCandidate(j:JobInput,get:Fetcher):Promise<JobInput>{
 let p:Page;
  try{
  if(/app\.usebraintrust\.com/.test(new URL(j.originalUrl).hostname)){
   const id=new URL(j.originalUrl).pathname.match(/\/jobs\/(\d+)/)?.[1];if(!id)throw new Error('Missing Braintrust job ID');
   p=await get(`https://app.usebraintrust.com/api/jobs/${id}/`,j.source);const b=JSON.parse(p.text);
   return {...j,title:b.title??j.title,company:b.employer?.name??j.company,description:plain([b.introduction,b.description,b.requirements].filter(Boolean).join('\n\n')),descriptionStatus:'available public API description',location:(b.locations??[]).map((l:any)=>l.location).join(' · '),applyUrl:null,applyStatus:'Braintrust application requires owner account; no employer form resolved'};
  }
  // Getro organization is sometimes another aggregator, not the hiring employer. Prefer the actual target.
  if(j.boardCompany&&j.applyUrl&&new URL(j.applyUrl).hostname!==new URL(j.originalUrl).hostname){
   try{
    p=await get(j.applyUrl,j.source);let d=detailFromHtml(p.text,p.url,j.source);
    for(let hop=0;hop<2&&d.applyUrl&&canonical(d.applyUrl)!==canonical(p.url)&&/linkedin\.com|jobs\.solana\.com|jobs\.(electriccapital|multicoin|framework)/.test(new URL(p.url).hostname);hop++){p=await get(d.applyUrl,j.source);d=detailFromHtml(p.text,p.url,j.source);}
    if(d.description.length>=100&&d.company!=='Unknown')return {...j,title:d.title,company:d.company,description:d.description,descriptionStatus:`${d.descriptionStatus}; employer from linked job page, board association ${j.boardCompany}`,location:d.location||j.location,expiresAt:d.expiresAt,applyUrl:/linkedin\.com|jobs\.solana\.com/.test(new URL(p.url).hostname)?d.applyUrl??null:p.url,applyStatus:'linked employer page read via GET; intermediary links excluded'};
   }catch{/* Try available board text, retaining unknown employer if not independently resolved. */}
  }
  p=await get(j.originalUrl,j.source);let d:JobInput;
  try{d=detailFromHtml(p.text,p.url,j.source);}catch(e){if(!j.applyUrl||canonical(j.applyUrl)===canonical(j.originalUrl))throw e;p=await get(j.applyUrl,j.source);d=detailFromHtml(p.text,p.url,j.source);}
  if(!d.description&&j.applyUrl&&canonical(j.applyUrl)!==canonical(p.url)){try{p=await get(j.applyUrl,j.source);d=detailFromHtml(p.text,p.url,j.source);}catch{/* retain available board description */}}
  // For known ATS pages, the fetched page itself is the real employer application destination.
  const ats=/^(jobs\.ashbyhq\.com|jobs\.lever\.co|job-boards\.greenhouse\.io|boards\.greenhouse\.io)$/.test(new URL(p.url).hostname);
  const verifiedDetail=d.description.length>=100;
  return {...j,title:verifiedDetail&&!/^(careers|jobs|join us|join our team)$/i.test(d.title.trim())?d.title:j.title,company:!j.boardCompany&&verifiedDetail&&d.company!=='Unknown'?d.company:j.company,description:d.description||j.description,descriptionStatus:(d.description?d.descriptionStatus:'missing full description; listing metadata retained')+(j.boardCompany?`; board association ${j.boardCompany}; actual employer unverified`:''),location:d.location||j.location,expiresAt:d.expiresAt,applyUrl:ats?p.url:d.applyUrl??j.applyUrl,applyStatus:ats?'reachable employer ATS (GET verified; no form interaction)':d.applyUrl?'source-provided application destination; reachability not checked':j.applyStatus??'not resolved'};
 }catch(e){return {...j,descriptionStatus:`detail unavailable: ${String(e)}`,expiresAt:/HTTP (404|410)/.test(String(e))?new Date(Date.now()-1).toISOString():j.expiresAt};}
}

export async function discoverSource(source:Source,options:DiscoveryOptions={}):Promise<DiscoveryResult>{
 const limits={...runtime(),...options.limits};
 const c=emptyCoverage(),records=new Map<string,JobInput>(),fingerprints=new Map<string,Set<string>>();
  const get:Fetcher=async(url,name)=>{const p=await (options.get?options.get(url,name):listingPage(url,name,options.force));if(!p.cached)c.networkRequests++;return p;};
  const post:SearchPoster=async(url,body,name,headers)=>{const p=await (options.post?options.post(url,body,name,headers):listingSearch(url,body,name,headers,options.force));if(!p.cached)c.networkRequests++;return p;};
 const browser=async(url:string,name:string)=>{c.browserNavigations++;return (options.browser??browserPage)(url,name);};
  const emit=()=>options.progress?.(structuredClone(c));
  const emitted=new Map<string,string>();const emitJob=(j:JobInput)=>{if(!matchQueries(queryText(j)).length)return;const key=canonical(j.originalUrl),version=hash(JSON.stringify(j));if(emitted.get(key)!==version){options.onJob?.(j);emitted.set(key,version);}};
 function group(label:string,url:string){const g:GroupReceipt={label,url,pages:0,reviewed:0,total:null,complete:false,stopReason:''};c.groups.push(g);return g;}
 function addPage(g:GroupReceipt,url:string,page:number,items:JobInput[]){
  const keys=items.map(j=>canonical(j.originalUrl)),fingerprint=hash(keys.join('\n')),seen=fingerprints.get(g.label)??new Set<string>();
  if(items.length&&seen.has(fingerprint))throw new Error(`Repeated page detected in ${g.label}; pagination refused`);
  seen.add(fingerprint);fingerprints.set(g.label,seen);let added=0;
   for(const j of items){if(/metadata/i.test(j.descriptionStatus)&&!j.tags)j.tags=[j.description];const key=canonical(j.originalUrl);if(records.has(key)){c.listingDuplicates++;continue;}if(records.size<limits.max_reviewed_per_source){records.set(key,j);emitJob(j);added++;}}
  c.pages.push({url,page,returned:items.length,newRecords:added,fingerprint,first:keys[0]??null,last:keys.at(-1)??null});g.pages++;g.reviewed+=added;c.reviewed=records.size;emit();
 }
 function pageBound(){return c.pages.length>=limits.max_pages||records.size>=limits.max_reviewed_per_source;}
 async function tryBrowser(url:string):Promise<Page>{try{return await get(url,source.name);}catch(e){c.errors.push(`HTTP: ${String(e)}`);return browser(url,source.name);}}
  const host=new URL(source.url).hostname.replace(/^www\./,'');
  emit();
  try{
   if(options.inventory){for(const j of options.inventory.jobs)records.set(canonical(j.originalUrl),j);Object.assign(c,{...options.inventory.coverage,started:now(),finished:undefined,networkRequests:0,detailRequests:0,detailFetched:0,detailFailed:0,errors:[],phase:'cached public inventory',completeness:'running'});c.queries=[...searchConfig().search.titles,...searchConfig().search.keywords];emit();}
   else {
  if(/jobs\.(electriccapital|multicoin|framework)/.test(host)){
   c.adapter='Getro public search API';const url=abs('/jobs',source.url),p=await get(url,source.name),$=load(p.text),data=JSON.parse($('#__NEXT_DATA__').text()),id=data.props?.pageProps?.network?.id;
   if(!/^\d+$/.test(String(id)))throw new Error('Adapter incomplete: Getro network ID absent');
   const endpoint=`https://api.getro.com/api/v2/collections/${id}/search/jobs`,g=group('Entire portfolio, all configured OR terms applied locally',endpoint);
   for(let page=0;!pageBound();page++){
    const r=JSON.parse((await post(endpoint,{hits_per_page:100,page,filters:'',query:''},source.name,{origin:new URL(url).origin,referer:url})).text).results;
    if(!Array.isArray(r?.jobs)||typeof r.count!=='number')throw new Error('Adapter incomplete: unexpected Getro search response');
    g.total=r.count;c.total=r.count;addPage(g,endpoint,page,parseGetroJobs(r.jobs,source.url,source.name));
    if((page+1)*100>=r.count||!r.jobs.length){g.complete=true;g.stopReason='Public portfolio exhausted';break;}
   }
  }else if(/usebraintrust\.com/.test(host)){
   c.adapter='Braintrust public jobs API';const g=group('All public roles (account-only roles not accessible)','https://app.usebraintrust.com/api/jobs/');let url=g.url;
   while(url&&!pageBound()){
    const b=JSON.parse((await get(url,source.name)).text);if(!Array.isArray(b.results))throw new Error('Adapter incomplete: Braintrust public results missing');g.total=b.count;c.total=b.count;
    const jobs=b.results.map((j:any):JobInput=>({title:j.title,company:j.employer?.name??'Unknown',originalUrl:`https://app.usebraintrust.com/jobs/${j.id}`,applyUrl:null,description:(j.main_skills??[]).map((s:any)=>s.name).join(', '),descriptionStatus:'indexed skills metadata; full description pending',location:(j.locations??[]).map((l:any)=>l.location).join(' · '),publishedAt:j.created,source:source.name}));
    addPage(g,url,g.pages+1,jobs);url=b.next?new URL(b.next).href.replace(/^http:/,'https:'):'';if(url&&new URL(url).hostname!=='app.usebraintrust.com')throw new Error('Unexpected Braintrust pagination host');
    if(!url){g.complete=true;g.stopReason='Public API exhausted';}
   }
  }else if(/a16zfrontier/.test(host)){
   c.adapter='a16z embedded portfolio dataset';const p=await get(source.url,source.name),items=parsePortfolioJobs(p.text,p.url,source.name),g=group('Full embedded portfolio dataset',p.url);g.total=new Set(items.map(j=>canonical(j.originalUrl))).size;c.total=g.total;c.inventoryTotal=items.length;addPage(g,p.url,1,items);g.complete=records.size===g.total;g.stopReason=g.complete?'Embedded public dataset exhausted':'Reviewed-record bound';
  }else if(/remotive/.test(host)){
   c.adapter='Remotive public API';const url='https://remotive.com/api/remote-jobs',b=JSON.parse((await get(url,source.name)).text);if(!Array.isArray(b.jobs))throw new Error('Unexpected Remotive response');const g=group('Entire delayed public API window; paid/private inventory excluded',url);g.total=b.jobs.length;c.total=b.jobs.length;
   addPage(g,url,1,b.jobs.map((j:any)=>({title:j.title,company:j.company_name,originalUrl:j.url,applyUrl:null,description:plain(j.description??''),descriptionStatus:'available public API description',location:j.candidate_required_location,publishedAt:j.publication_date,source:source.name})));g.complete=records.size===b.jobs.length;g.stopReason=g.complete?'Public API window exhausted':'Reviewed-record bound';
  }else if(/remote3|weworkremotely/.test(host)){
   c.adapter='Public RSS feed';const url=/remote3/.test(host)?'https://www.remote3.co/api/rss':'https://weworkremotely.com/remote-jobs.rss',p=await get(url,source.name),items=parseFeed(p.text,url,source.name),g=group(/remote3/.test(host)?'Entire published feed window':'All-category public feed, including management',url);
   if(!items.length)throw new Error('Adapter incomplete: no recognised feed items');g.total=items.length;c.total=items.length;addPage(g,url,1,items);g.complete=records.size===items.length;g.stopReason=g.complete?'Published feed window exhausted':'Reviewed-record bound';
  }else if(/news\.ycombinator/.test(host)){
   c.adapter='HN official monthly hiring API';const user=JSON.parse((await get('https://hacker-news.firebaseio.com/v0/user/whoishiring.json',source.name)).text);let thread:any;
   for(const id of (user.submitted??[]).slice(0,12)){const item=JSON.parse((await get(`https://hacker-news.firebaseio.com/v0/item/${id}.json`,source.name)).text);if(/^Ask HN: Who is hiring\?/.test(item.title??'')){thread=item;break;}}
   if(!thread)throw new Error('Current official hiring thread not found');const g=group(thread.title,`https://news.ycombinator.com/item?id=${thread.id}`);g.total=thread.kids.length;c.total=g.total;
   const children=thread.kids.slice(0,limits.max_reviewed_per_source);let examined=0;
   for(let start=0;start<children.length&&!pageBound();start+=50){const rows:JobInput[]=[];
    await pooled(children.slice(start,start+50),limits.detail_concurrency,async(id:number)=>{const item=JSON.parse((await get(`https://hacker-news.firebaseio.com/v0/item/${id}.json`,source.name)).text);examined++;
     if(!item.text||item.dead||item.deleted)return;const description=plain(item.text),first=description.split('\n')[0],$=load(item.text),apply=$('a[href]').map((_,e)=>$(e).attr('href')!).get().find(h=>/^https?:/.test(h)&&/jobs|careers|ashby|greenhouse|lever|apply/i.test(h));
     if(!apply&&!/\||\b(hiring|join|position|role|engineer|developer)\b/i.test(first))return;
     const parts=first.split('|').map((v:string)=>v.trim());rows.push({title:parts.slice(1).find((v:string)=>/engineer|develop|cto|manager|architect|lead/i.test(v))??first.slice(0,220),company:first.split(/\|| — | - | \(/)[0].trim(),originalUrl:`https://news.ycombinator.com/item?id=${id}`,applyUrl:apply??null,applyStatus:'hiring-comment destination; employer identity unverified',description,descriptionStatus:'available complete hiring comment (author-supplied)',publishedAt:new Date(item.time*1000).toISOString(),source:source.name});
    });addPage(g,`https://hacker-news.firebaseio.com/v0/item/${thread.id}.json#children-${start}-${start+49}`,g.pages+1,rows);
   }g.reviewed=examined;g.complete=examined===thread.kids.length;g.stopReason=g.complete?'All top-level hiring comments examined':'Comment/page bound';
  }else if(/cryptocurrencyjobs\.co/.test(host)){
   c.adapter='Cryptocurrency Jobs observed public search';const p=await get(source.url,source.name),$=load(p.text),src=$('script[src]').map((_,e)=>$(e).attr('src')!).get().find(s=>/\/search\.min/.test(s));if(!src)throw new Error('Adapter incomplete: public search configuration script absent');
   const js=(await get(abs(src,p.url),source.name)).text,credentials=js.match(/algoliasearch\("([^"]+)","([^"]+)"\)/);if(!credentials)throw new Error('Adapter incomplete: public search client configuration changed');
   const [_,app,key]=credentials,endpoint=`https://${app.toLowerCase()}-dsn.algolia.net/1/indexes/*/queries`,g=group('All current curated public roles, all configured OR terms locally',source.url);
   for(let page=0;!pageBound();page++){
    const params=new URLSearchParams({filters:'searchFilter:IsHome',hitsPerPage:'100',page:String(page),query:''});const r=JSON.parse((await post(endpoint,{requests:[{indexName:'jobs',params:params.toString()}]},source.name,{'x-algolia-application-id':app,'x-algolia-api-key':key,referer:'https://cryptocurrencyjobs.co/',origin:'https://cryptocurrencyjobs.co'})).text).results?.[0];
    if(!Array.isArray(r?.hits))throw new Error('Adapter incomplete: public search response missing');g.total=r.nbHits;c.total=r.nbHits;
    const items=r.hits.filter((j:any)=>!j.is_ad&&j.permalink&&j.company?.name).map((j:any):JobInput=>({title:j.title,company:j.company.name,originalUrl:abs(j.permalink,source.url),applyUrl:null,description:(j.keywords??[]).map((k:any)=>k.name).join(', '),descriptionStatus:'listing keywords metadata; full description pending',location:(j.locationFilter??[]).join(' · '),publishedAt:j.datePublished,source:source.name}));
    addPage(g,endpoint,page,items);if(page+1>=r.nbPages){g.complete=true;g.stopReason='Current public index window exhausted (promoted ads excluded)';break;}
   }
  }else if(/wellfound\.com/.test(host)){
   c.adapter='Wellfound public role search / Apollo';const home=await get(source.url,source.name),$=load(home.text);
   const paths=['/role/software-engineer','/role/product-manager','/role/operations-manager'].filter(path=>$(`a[href="${path}"]`).length);
   if(!paths.length)throw new Error('Adapter incomplete: global public role links missing');const groups=paths.map(path=>group(path.split('/').at(-1)!,abs(path,source.url)));
   for(let page=1;!pageBound()&&groups.some(g=>!g.complete);page++)for(const g of groups.filter(g=>!g.complete)){
    if(pageBound())break;const url=new URL(g.url);if(page>1)url.searchParams.set('page',String(page));
    try{const p=await get(url.href,source.name),r=parseWellfound(p.text,p.url,source.name);g.total=r.total;addPage(g,url.href,page,r.jobs);if(page>=r.pages){g.complete=true;g.stopReason='Public role pages exhausted';}}catch(e){g.stopReason=String(e);c.errors.push(`${g.label}: ${String(e)}`);g.complete=true;}
   }
  }else if(/web3\.careers?$/.test(host)){
   c.adapter='Web3 Career verified category search';const base='https://web3.career/',p=await get(base,source.name),$=load(p.text);
   const configured=normalize(c.queries.join(' ')),wanted=['engineer','dev','cto','ai','project-manager','rust','solidity','blockchain'].filter(w=>w==='dev'?/develop/.test(configured):configured.includes(w.replace('-',' ')));
   const paths=wanted.map(tag=>`/${tag}-jobs`).filter(path=>$(`a[href="${path}"]`).length).slice(0,limits.max_query_groups);
   if(!paths.length)throw new Error('Adapter incomplete: no configured role/skill categories discovered');
   const groups=paths.map(path=>group(path.slice(1,-5),abs(path,base))),next=new Map(groups.map(g=>[g.label,g.url]));
   while(!pageBound()&&groups.some(g=>next.get(g.label)))for(const g of groups){const url=next.get(g.label);if(!url||pageBound())continue;
    const p=await get(url,source.name),items=parseWeb3(p.text,p.url,source.name);if(!items.length)throw new Error(`Adapter incomplete: no Web3 job rows in ${g.label}`);addPage(g,url,g.pages+1,items);
    const $=load(p.text),href=$('a[rel="next"]').attr('href');next.set(g.label,href?abs(href,p.url):'');if(!href){g.complete=true;g.stopReason='Category exhausted';}
   }
  }else if(/malt\.|cryptojobslist|welcometothejungle/.test(host)){
   c.adapter='Public search / dedicated browser session';const urls=/welcometothejungle/.test(host)?['https://www.welcometothejungle.com/fr/jobs',source.url]:[source.url];
   let found:Page|undefined;
   for(const url of urls){const g=group('Observed public search entry',url);try{
     let p=await tryBrowser(url),items=parsePublicHtml(p.text,p.url,source.name);
     if(!items.length){p=await browser(url,source.name);items=parsePublicHtml(p.text,p.url,source.name);}
     if(items.length){addPage(g,p.url,1,items);g.stopReason='Public listing parsed; dynamic pagination not verified';found=p;break;}
     c.errors.push(`${url}: shell has no public job records`);
    }catch(e){c.errors.push(`${url}: ${String(e)}`);}}
   if(!found){
    const challenged=c.errors.some(e=>/403|challenge|blocked|CAPTCHA|CloudFront/i.test(e));
    const vendor=c.errors.some(e=>/CloudFront/.test(e))?'CloudFront':c.errors.some(e=>/Cloudflare/.test(e))?'Cloudflare':'Site access check';
    c.completeness=challenged?'blocked':'adapter-incomplete';c.blocker=challenged?`${vendor} blocks ${new URL(source.url).hostname}'s public search in the dedicated browser. No vacancy query results were accessible.`:'The accessible page contains no public vacancy records; authenticated search integration is not verified.';
    c.action=`Owner step: npm --prefix legacy/app run cli -- session "${source.name}". If still blocked, configure a per-source proxy and retry.`;
   }else c.completeness='adapter-incomplete';
  }else if(/paradigm\.xyz/.test(host)){
   c.adapter='Paradigm scoped public portfolio list';const p=await get(source.url,source.name),inventory=parsePublicHtml(p.text,p.url,source.name);if(!inventory.length)throw new Error('Adapter incomplete: no employer job links');
   // Apply role/query scoping across the ENTIRE embedded list, before the review bound; never take a front-page prefix.
    const items=inventory;
    c.inventoryTotal=inventory.length;const g=group(`All ${inventory.length} published roles; descriptions needed for keyword-only decisions`,p.url);g.total=new Set(items.map(j=>canonical(j.originalUrl))).size;c.total=g.total;addPage(g,p.url,1,items);g.complete=records.size===g.total;g.stopReason=g.complete?'Entire public inventory exhausted (not a claim of description completeness)':'Reviewed-record bound';
  }else{
   c.adapter='Generic public rel-next adapter';const g=group('Configured public listing',source.url);let url=source.url;
   while(url&&!pageBound()){const p=await get(url,source.name),items=parsePublicHtml(p.text,p.url,source.name);if(!items.length)throw new Error('Adapter incomplete: unknown listing markup; no vacancies claimed');addPage(g,url,g.pages+1,items);const href=load(p.text)('a[rel="next"]').attr('href');url=href?abs(href,p.url):'';if(url&&new URL(url).hostname!==new URL(source.url).hostname)throw new Error('Unexpected pagination host');if(!url){g.complete=true;g.stopReason='Public pages exhausted';}}
   }
   }
  }catch(e){c.errors.push(String(e));c.blocker=String(e);c.completeness=/401|login|sign.in|Authentication/.test(String(e))?'needs-login':/403|406|429|timeout|Timeout|fetch failed|ECONN|challenge|blocked/i.test(String(e))?'blocked':'adapter-incomplete';c.action=c.completeness==='blocked'?`Retry this source using a dedicated browser session or a configured per-source proxy. ${source.name} was not treated as an empty search.`:'Adapter response changed; inspect recorded request/page receipts before claiming coverage.';}
 const listingErrors=c.errors.length;
 c.phase='reading descriptions';emit();
 const candidates=[...records.values()];
  // Any title can contain a keyword-only match in its description. Do not gate by technical titles.
   for(const j of candidates)emitJob(j);
  const needDetail=candidates.filter(j=>!hasDescription(j)&&(j.detailRetryAt??0)<=Date.now());
 // Alternate keyword-only candidates and title matches, rather than spending the entire budget on title matches.
  const unknown=needDetail.filter(j=>!matchQueries(queryText(j)).length),matched=needDetail.filter(j=>matchQueries(queryText(j)).length);
 const ordered:JobInput[]=[];for(let i=0;i<Math.max(unknown.length,matched.length);i++){if(unknown[i])ordered.push(unknown[i]);if(matched[i])ordered.push(matched[i]);}
 await pooled(ordered.slice(0,limits.max_detail_requests),limits.detail_concurrency,async j=>{
   c.detailRequests++;const result=await hydrateCandidate(j,options.get?get:async(url,name)=>{const p=await detailPage(url,name);if(!p.cached)c.networkRequests++;return p;});if(hasDescription(result))c.detailFetched++;else{result.detailRetryAt=Date.now()+3600000;c.detailFailed++;if(c.errors.length<20)c.errors.push(`${j.originalUrl}: ${result.descriptionStatus}`);}records.set(canonical(j.originalUrl),result);emitJob(result);emit();
 });
   const all=[...records.values()],matches=all.filter(j=>matchQueries(queryText(j)).length>0);
  c.reviewed=all.length;c.matched=matches.length;c.descriptionUnreviewed=all.filter(j=>!hasDescription(j)).length;
  c.descriptionAvailable=all.length-c.descriptionUnreviewed;c.descriptionFailed=all.filter(j=>!hasDescription(j)&&!!j.detailRetryAt).length;c.descriptionDeferred=c.descriptionUnreviewed-c.descriptionFailed;
  c.queryMatches=Object.fromEntries(c.queries.map(q=>[q,matches.filter(j=>matchQueries(queryText(j)).includes(q)).length]));
 c.queryApplication=all.length?`All ${c.queries.length} configured OR terms applied to available titles, metadata and descriptions. ${all.length-c.descriptionUnreviewed}/${all.length} records have available description text; keywords in unavailable/unfetched descriptions remain untested.`:'No records available; configured OR queries were not executed against vacancy content.';
  if(c.completeness==='running')c.completeness=c.groups.every(g=>g.complete)&&listingErrors===0&&all.every(hasDescription)&&matches.length<=limits.max_jobs_per_source?'complete-public-window':'bounded';
 c.stopReason=c.blocker??(records.size>=limits.max_reviewed_per_source?'Reviewed-record limit reached':c.pages.length>=limits.max_pages&&c.groups.some(g=>!g.complete)?'Page limit reached':c.groups.map(g=>g.stopReason).filter(Boolean).join('; '));
 if(ordered.length>limits.max_detail_requests)c.stopReason+=`; detail-request limit ${limits.max_detail_requests} reached (${ordered.length-limits.max_detail_requests} candidates not fetched)`;
 if(matches.length>limits.max_jobs_per_source)c.stopReason+=`; import limit ${limits.max_jobs_per_source} reached (${matches.length-limits.max_jobs_per_source} matches not imported)`;
  c.phase='ready to persist';c.finished=now();emit();
  options.saveInventory?.(all,structuredClone(c));
 const status=['blocked','needs-login','adapter-incomplete'].includes(c.completeness)?c.completeness:c.completeness==='complete-public-window'?'healthy':'bounded';
 const selected=matches.slice(0,limits.max_jobs_per_source).map(j=>{
  if(j.applyUrl&&/linkedin\.com|jobs\.solana\.com|jobs\.(electriccapital|multicoin|framework)/.test(new URL(j.applyUrl).hostname))return {...j,applyUrl:null,applyStatus:`Unresolved intermediary destination: ${j.applyUrl}. Use original listing; no direct employer form claimed.`};
  return j;
 });
 return {jobs:selected,status,message:c.blocker??`${c.reviewed} reviewed; ${c.matched} match configured queries. ${c.stopReason}`,coverage:c};
}

export async function pooled<T>(items:T[],concurrency:number,fn:(item:T)=>Promise<void>){let next=0,failure:unknown;await Promise.all(Array.from({length:Math.min(concurrency,items.length)},async()=>{while(next<items.length&&!failure){const item=items[next++];try{await fn(item);}catch(e){failure=e;}}}));if(failure)throw failure;}
