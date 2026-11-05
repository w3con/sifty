import './isolate.js';
import {test}from'node:test';import assert from'node:assert/strict';
import {Store}from'../src/store.js';
import {discoverSource,parseWellfound,parsePortfolioJobs}from'../src/discovery.js';
import {isReadOnlySearch}from'../src/network.js';
import {search,collectionProgress,sourceHealth,exclusive,startSearch}from'../src/services.js';
import {localPath,now}from'../src/core.js';
const page=(url:string,text:string)=>({url,text,status:200});
const long='Build Solidity contracts and TypeScript services with reliable production tooling. '.repeat(8);
test('Getro verified API paginates distinct pages, never uses guessed skip/page HTML parameters',async()=>{
 const calls:any[]=[];const r=await discoverSource({name:'Electric fixture',url:'https://jobs.electriccapital.com/jobs',enabled:true},{limits:{max_pages:3,max_detail_requests:0},get:async url=>page(url,'<script id="__NEXT_DATA__">{"props":{"pageProps":{"network":{"id":"1640"}}}}</script>'),post:async(url,body)=>{calls.push({url,body});return page(url,JSON.stringify({results:{count:2,jobs:[0,1].map(p=>({id:p+1,title:'Software Engineer',organization:{name:`Employer ${p}`,slug:`company-${p}`},slug:`${p+1}-engineer`,skills:['TypeScript'],url:`https://employer.example/job/${p+1}`}))}}));}});
 assert.equal(calls.length,1);assert.equal(r.coverage.pages[0].newRecords,2);assert.equal(r.coverage.total,2);assert.ok(calls[0].url.endsWith('/search/jobs'));assert.equal(calls[0].body.query,'');
 const multiple=await discoverSource({name:'Electric fixture',url:'https://jobs.electriccapital.com/jobs',enabled:true},{limits:{max_pages:2,max_detail_requests:0},get:async url=>page(url,'<script id="__NEXT_DATA__">{"props":{"pageProps":{"network":{"id":"1640"}}}}</script>'),post:async(url,body)=>{const p=(body as any).page;return page(url,JSON.stringify({results:{count:201,jobs:[{title:'Software Engineer',organization:{name:`Employer ${p}`,slug:`c-${p}`},slug:`${p}-role`,skills:['Rust']} ]}}));}});
 assert.equal(multiple.coverage.pages.length,2);assert.notEqual(multiple.coverage.pages[0].first,multiple.coverage.pages[1].first);assert.equal(multiple.status,'bounded');
});
test('repeated pagination is detected and preserved as incomplete, not counted twice',async()=>{
 const r=await discoverSource({name:'Repeated',url:'https://fixture.example/jobs',enabled:true},{limits:{max_pages:4,max_detail_requests:0},get:async url=>page(url,`<script type="application/ld+json">${JSON.stringify({'@type':'JobPosting',url:'https://fixture.example/jobs/1',title:'Software Engineer',hiringOrganization:{name:'Fixture'},description:long})}</script><a rel="next" href="?page=2">Next</a>`)});
 assert.equal(r.coverage.pages.length,1);assert.equal(r.coverage.reviewed,1);assert.equal(r.status,'adapter-incomplete');assert.match(r.coverage.blocker!,/Repeated page/);
});
test('keyword-only match is discovered in description before generic technical title is rejected',async()=>{
 const r=await discoverSource({name:'Keyword fixture',url:'https://fixture.example/jobs',enabled:true},{get:async url=>url.endsWith('/jobs')?page(url,'<a href="/jobs/123-platform-specialist">Platform specialist</a>'):page(url,`<h1>Platform specialist</h1><script type="application/ld+json">${JSON.stringify({'@type':'JobPosting',title:'Platform specialist',hiringOrganization:{name:'Actual employer'},description:long})}</script>`)});
 assert.equal(r.jobs.length,1);assert.equal(r.jobs[0].company,'Actual employer');assert.match(r.jobs[0].description,/Solidity/);assert.equal(r.coverage.detailFetched,1);
});
test('a16z embedded data is parsed without evaluating page JavaScript',()=>{
 const rows=[{company:'Example',jobs:[{title:'Protocol Engineer',companyName:'Example',url:'https://employer.example/job',skills:['Rust']}]}];assert.equal(parsePortfolioJobs(`<script>const portfolioJobs = ${JSON.stringify(rows)}; alert('never execute');</script>`,'https://a16zfrontier.com','a16z')[0].company,'Example');
});
test('Wellfound company associations and full descriptions are read from Apollo references',()=>{
 const data={ROOT_QUERY:{talent:{query:{__typename:'Results',totalJobCount:101,pageCount:3,startups:[{__ref:'StartupResult:1'}]}}},'StartupResult:1':{name:'Actual company',highlightedJobListings:[{__ref:'Job:1'}]},'Job:1':{id:'11',slug:'platform-engineer',title:'Platform engineer',description:long,locationNames:['Remote']}};
 const r=parseWellfound(`<script id="__NEXT_DATA__">${JSON.stringify({props:{pageProps:{apolloState:{data}}}})}</script>`,'https://wellfound.com','Wellfound');assert.equal(r.jobs[0].company,'Actual company');assert.equal(r.jobs[0].description.trim(),long.trim());assert.equal(r.pages,3);
});
test('all-category feeds are healthy finite windows, not adapter failures, and description text drives OR matching',async()=>{
 let requested='';const xml=`<rss><channel><item><title>Company: Interim CTO</title><link>https://fixture.example/jobs/1</link><description>${long}</description></item></channel></rss>`;
 const r=await discoverSource({name:'We Work Remotely',url:'https://weworkremotely.com',enabled:true},{get:async url=>{requested=url;return page(url,xml);}});assert.equal(requested,'https://weworkremotely.com/remote-jobs.rss');assert.equal(r.status,'healthy');assert.equal(r.coverage.reviewed,1);assert.equal(r.jobs.length,1);
});
test('observed search POST allowlist cannot submit employer forms or mutate jobs',()=>{
 assert.ok(isReadOnlySearch('https://api.getro.com/api/v2/collections/1640/search/jobs',{query:'Rust',page:0,hits_per_page:100,filters:''}));assert.ok(!isReadOnlySearch('https://api.getro.com/api/v2/collections/1640/jobs/1/job_applications',{email:'x'}));assert.ok(!isReadOnlySearch('https://employer.example/apply',{query:'x'}));assert.ok(!isReadOnlySearch('https://api.getro.com/api/v2/collections/1640/search/jobs',{delete:true}));
});
test('collection receipts/progress and counters persist; owner status survives reruns and duplicates',async()=>{
 const s=new Store(':memory:');try{
  const payload={jobs:[{title:'Software Engineer',company_name:'Example',url:'https://remotive.com/remote-jobs/test/fixture-123',description:long}]};const options={get:async(url:string)=>page(url,JSON.stringify(payload))};
  await exclusive(s,()=>search(s,'Remotive',options));const id=s.list()[0].id;s.status(id,'saved');
  await exclusive(s,()=>search(s,'Remotive',options));const r=collectionProgress(s);assert.equal(r.sources[0].coverage.updated,1);assert.equal(r.sources[0].coverage.added,0);assert.equal(r.status,'complete');assert.equal(s.get(id).status,'saved');assert.ok(r.events.length>=3);assert.equal(sourceHealth(s).find(v=>v.name==='Remotive')?.status,'healthy');
 }finally{s.close();}
});
test('real access challenges remain actionable blocked states with no invented totals',async()=>{
 const r=await discoverSource({name:'Malt',url:'https://www.malt.fr',enabled:true},{get:async()=>{throw new Error('HTTP 403');},browser:async()=>{throw new Error('Browser blocked (403); Cloudflare challenge');}});assert.equal(r.status,'blocked');assert.equal(r.coverage.total,null);assert.equal(r.coverage.reviewed,0);assert.match(r.coverage.action!,/session "Malt"/);assert.equal(r.coverage.queryApplication.startsWith('No records'),true);
});
test('board association cannot masquerade as employer and linked employer wins',async()=>{
 const {hydrateCandidate}=await import('../src/discovery.js');
 const source={title:'Software Engineer',company:'Unknown',boardCompany:'StreamingFast',source:'Multicoin',originalUrl:'https://jobs.multicoin.capital/companies/streamingfast/jobs/123-engineer',applyUrl:'https://employer.example/job/1',description:'Rust',descriptionStatus:'indexed skills metadata'};
 const j=await hydrateCandidate(source,async url=>page(url,`<script type="application/ld+json">${JSON.stringify({'@type':'JobPosting',title:'Software Engineer',hiringOrganization:{name:'Actual Employer'},description:long})}</script>`));assert.equal(j.company,'Actual Employer');assert.match(j.descriptionStatus,/board association StreamingFast/);
});
test('distinct same-company openings with different employer IDs are preserved',()=>{
 const s=new Store(':memory:');try{
  const j={title:'Software Engineer',company:'Example',source:'Board',description:long,descriptionStatus:'available',originalUrl:'https://board.example/jobs/1',applyUrl:'https://jobs.ashbyhq.com/example/id-1'};
  const a=s.upsert(j),b=s.upsert({...j,originalUrl:'https://board.example/jobs/2',applyUrl:'https://jobs.ashbyhq.com/example/id-2'});assert.notEqual(a,b);assert.equal(s.list().length,2);
 }finally{s.close();}
});
test('a careers directory cannot supply the first unrelated posting as a role description',async()=>{
 const {detailFromHtml}=await import('../src/collectors.js');const jobs=[1,2].map(id=>({'@type':'JobPosting',url:`https://employer.example/jobs/${id}`,title:`Engineer ${id}`,hiringOrganization:{name:'Example'},description:long}));
 assert.throws(()=>detailFromHtml(`<script type="application/ld+json">${JSON.stringify(jobs)}</script>`,'https://employer.example/careers','Fixture'),/Directory/);
 assert.equal(detailFromHtml(`<script type="application/ld+json">${JSON.stringify(jobs)}</script>`,'https://employer.example/jobs/2','Fixture').title,'Engineer 2');
});
test('late dedup merge retains the existing current assessment and all history',()=>{
 const s=new Store(':memory:');try{
  const a=s.upsert({title:'Unverified',company:'Unknown',source:'A',description:long,descriptionStatus:'available',originalUrl:'https://a.example/jobs/1'});
  const b=s.upsert({title:'Software Engineer',company:'Example',source:'B',description:long,descriptionStatus:'available',originalUrl:'https://b.example/jobs/2',applyUrl:'https://employer.example/job/2'});s.status(b,'saved');s.assessment(b,{score:13,summary:'Existing assessment'});
  s.upsert({title:'Software Engineer',company:'Example',source:'A',description:long,descriptionStatus:'available',originalUrl:'https://a.example/jobs/1',applyUrl:'https://employer.example/job/2'});
  assert.equal(s.get(a).assessment.score,13);assert.equal(s.get(b).id,a);assert.equal(s.get(a).status,'saved');assert.equal((s.db.prepare('SELECT count(*) n FROM assessments').get()as any).n,1);
 }finally{s.close();}
});
