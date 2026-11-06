import './isolate.js';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, symlinkSync, chmodSync, rmSync } from 'node:fs';
import { resolve, relative } from 'node:path';
import { createServer } from 'node:http';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { assessmentSchema, canonical, hash, localPath, matchQueries, now, ROOT, rootPath, searchConfig, searchSchema, type JobInput, type Provider } from '../src/core.js';
import { Store } from '../src/store.js';
import { collectSource, detailFromHtml, enrich, parseFeed, sourceCapability } from '../src/collectors.js';
import { contextForJob, contextVersion, type CandidateContext } from '../src/context.js';
import { assess, infer, isolatedEnv, parseJson, runProcess, sourceQuote } from '../src/providers.js';
import { acquireLock, daily, sourceHealth, validateAll } from '../src/services.js';
import { networkSchema, networkProfile, publicUrl, sessionDirectory, fetchPage, browserSession } from '../src/network.js';
import { extractWithModel } from '../src/extraction.js';
import {discoverSource}from'../src/discovery.js';

const temp = () => mkdtempSync(localPath('test-'));
const input = (overrides: Partial<JobInput> = {}): JobInput => ({title:'Software engineer',company:'Example',originalUrl:'https://board.example/jobs/1',applyUrl:'https://employer.example/careers/123',description:'Build TypeScript services and Solidity contracts.',descriptionStatus:'available',source:'Board A',...overrides});
const context = (): CandidateContext => ({version:'fixture-context',createdAt:now(),preferences:searchConfig().preferences,reviewNotes:['Location does not establish work authorisation.'],sources:[{id:'cv.pdf',reference:'cv.pdf',retrievedAt:now(),hash:'fixture',status:'available',text:'Built Solidity staking contracts and TypeScript services.\n\nLed software engineering teams.'},{id:'linkedin',reference:'https://linkedin.com/in/fixture',retrievedAt:now(),hash:'',status:'missing',text:'',error:'Export required'}]});

test('config validation preserves existing 16 sources and OR aliases, unknown sources explicit',()=>{
  assert.equal(validateAll().valid,true);const c=searchConfig();assert.equal(c.sites.length,16);
  assert.ok(matchQueries('Build NODE.JS and Layer 2 systems',c).includes('Nodejs'));
  assert.equal(matchQueries('chair repair',c).length,0); // AI must not match inside chair.
  assert.ok(matchQueries('A full-time Software Engineer in an onsite team',c).length);
  assert.throws(()=>searchSchema.parse({...c,sites:[c.sites[0],c.sites[0]]}));
  assert.throws(()=>searchSchema.parse({...c,sites:[{name:'Invalid',url:'file:///tmp/a'}]}));
  assert.match(sourceCapability({name:'Unknown',url:'https://unknown.example/jobs',enabled:true}),/unknown source/);
  assert.throws(()=>rootPath('../outside'));assert.throws(()=>rootPath('/tmp'));
  assert.throws(()=>networkSchema.parse({version:1,profiles:{bad:{proxy:'secret'}},sources:{}}));
  assert.equal(networkProfile('No override').proxy,undefined);
  assert.notEqual(sessionDirectory('Wellfound'),sessionDirectory('Remote3'));
});
test('SQLite persistence, canonical apply dedup, identity dedup, late merge and status history',()=>{
  const dir=temp(),path=resolve(dir,'jobs.sqlite');let s=new Store(path);
  try {
    const id=s.upsert(input());s.status(id,'saved');
    assert.equal(s.upsert(input({source:'Board B',originalUrl:'https://other.example/jobs/2',title:'Different title',company:'Other',applyUrl:'https://www.employer.example/careers/123?utm_source=b#form'})),id);
    assert.equal(s.get(id).sources.length,2);assert.equal(s.get(id).status,'saved');
    s.upsert(input({title:'Unrelated job',company:'Other',originalUrl:'https://other.example/jobs/3',applyUrl:null}));
    const a=s.upsert(input({title:'New unknown role',company:'Unknown',originalUrl:'https://board.example/jobs/4',applyUrl:null}));
    const b=s.upsert(input({title:'Protocol Engineer',company:'Real Employer',originalUrl:'https://board.example/jobs/5',applyUrl:'https://real.example/jobs/1'}));
    s.status(b,'rejected');
    const merged=s.upsert(input({title:'Protocol Engineer',company:'Real Employer',originalUrl:'https://board.example/jobs/4',applyUrl:'https://real.example/jobs/1'}));
    assert.equal(merged,a);assert.equal(s.get(a).company,'Real Employer');assert.equal(s.get(a).sources.length,2);assert.equal(s.get(a).status,'rejected');
    assert.equal(s.get(b).id,a);s.status(b,'saved');assert.equal(s.get(a).status,'saved');s.status(a,'rejected');
    s.close();s=new Store(path);assert.equal(s.get(id).status,'saved');assert.equal(s.list('rejected').length,1);
    assert.ok((s.db.prepare('SELECT count(*) n FROM snapshots').get() as any).n>=6);
    s.upsert(input({applyUrl:null,applyStatus:'board intermediary/restricted destination, not a resolved employer form',location:'Explicit source location'}));assert.equal(s.get(id).applyUrl,null);assert.equal(s.get(id).location,'Explicit source location');
    assert.throws(()=>s.status(id,'submitted'));assert.equal(canonical('https://www.example.com/job/?utm_source=x#apply'),'https://example.com/job');
  } finally{s.close();rmSync(dir,{recursive:true,force:true});}
});
test('HTML/RSS descriptions, explicit links, bounded pagination and honest failures',async()=>{
  const schema={'@type':'JobPosting',title:'Software Engineer',hiringOrganization:{name:'Example'},description:'<p>Build Solidity.</p>',validThrough:'2020-01-01'};
  const html=`<h1>Software Engineer</h1><script type="application/ld+json">${JSON.stringify(schema)}</script><a href="https://jobs.ashbyhq.com/example/123">Apply now</a><form method="post" action="https://submit.example">never submit</form>`;
  const j=detailFromHtml(html,'https://board.example/1','Fixture');assert.match(j.description,/Build Solidity/);assert.equal(j.applyUrl,'https://jobs.ashbyhq.com/example/123');
  const missing=detailFromHtml('<h1>Role</h1><p>Marketing boilerplate</p>','https://board.example/2','Fixture');assert.equal(missing.description,'');assert.equal(missing.applyUrl,null);
  const feed=parseFeed('<rss><channel><item><title>Example: Software Engineer</title><link>https://board.example/1</link><description><![CDATA[<p>Full text</p>]]></description></item></channel></rss>','https://feed.example','Fixture');assert.equal(feed[0].company,'Example');assert.equal(feed[0].description,'Full text');
  const calls:string[]=[];
  const get=async(url:string)=>{calls.push(url);return {url,status:200,text:`<script type="application/ld+json">${JSON.stringify({...schema,url:`https://fixture.example/jobs/${calls.length}`,description:'Full TypeScript engineering description. '.repeat(10)})}</script><a rel="next" href="/?page=${calls.length+1}">Next</a>`};};
  const result=await discoverSource({name:'Fixture',url:'https://fixture.example/jobs',enabled:true},{get,limits:{max_pages:2}});assert.equal(calls.length,2);assert.equal(result.jobs.length,2);assert.equal(result.status,'bounded');
  const failure=await discoverSource({name:'Fixture',url:'https://fixture.example/jobs',enabled:true},{get:async url=>({url,status:200,text:'Loading…'})});assert.equal(failure.status,'adapter-incomplete');
  const store=new Store(':memory:');try{const id=store.upsert(input());await enrich(store,id,async()=>{throw new Error('HTTP 410');});assert.equal(store.get(id).expired,true);assert.match(store.get(id).descriptionStatus,/expired/);}finally{store.close();}
});
test('candidate context selection retains provenance/missing notices and rejects overflow',()=>{
  assert.equal(sourceQuote('Built  Solidity\n staking contracts.','Built Solidity staking contracts.'),'Built  Solidity\n staking contracts.');assert.equal(sourceQuote('Worked on a project.','Led the project.'),null);
  const c=context(),selected=contextForJob(c,'Solidity',5000);assert.equal(selected.sources[0].id,'cv.pdf');assert.ok(selected.overview.sourceStatus.some((s:any)=>s.status==='missing'));assert.equal(selected.sources[0].omittedChars,0);
  const later=structuredClone(c);later.sources[0].retrievedAt='2030-01-01T00:00:00Z';assert.equal(contextVersion(c),contextVersion(later));later.sources[0].text+=' New fact.';assert.notEqual(contextVersion(c),contextVersion(later));
  assert.throws(()=>contextForJob(c,'x',50),/budget/);
  c.sources[0].text='x'.repeat(10000);assert.throws(()=>contextForJob(c,'x',1500),/Unsegmentable/);
});
test('optional model extraction routes separately and rejects invented page facts',async()=>{
  const html='<h1>Protocol Engineer</h1><p>Real Employer</p><p>Build Solidity smart contracts and test reliable production systems.</p>';
  const good={title:'Protocol Engineer',company:'Real Employer',description:'Build Solidity smart contracts and test reliable production systems.',location:null};
  const j=await extractWithModel(html,'https://example.com/job','Fixture',async()=>good);assert.equal(j.applyUrl,null);assert.match(j.descriptionStatus,/quoted excerpt/);
  await assert.rejects(()=>extractWithModel(html,'https://example.com/job','Fixture',async()=>({...good,company:'Invented company'})),/not a source quote/);
});
test('assessment evidence is validated, versioned, stale-detected; failures preserve pending jobs',async()=>{
  const s=new Store(':memory:');try{
    s.saveContext(context());const id=s.upsert(input());
    const good={score:12,summary:'Low fit is retained.',evidence:[{claim:'Hands-on experience',evidenceId:'E1'}],gaps:['Unknown location eligibility']};
    const result=await assess(s,id,'openai',async()=>good);assert.equal(result.contextVersion,'fixture-context');assert.equal(s.list()[0].assessment.score,12);assert.equal(s.get(id).assessment.stale,false);
    await assert.rejects(()=>assess(s,id,'openai',async()=>({...good,evidence:[{claim:'Fake',sourceId:'cv.pdf',quote:'Invented CEO experience'}]})),/Unsupported evidence/);
    assert.match(s.get(id).assessmentError!,/Unsupported evidence/);assert.equal(s.get(id).assessment.score,12);
    const newId=s.upsert(input({originalUrl:'https://board.example/new',applyUrl:null,company:'Different'}));
    await assert.rejects(()=>assess(s,newId,'openai',async()=>({score:'great'})));assert.equal(s.get(newId).assessment,null);assert.ok(s.get(newId).assessmentError);
    s.upsert(input({description:'Changed description with additional experience requirements.'}));assert.equal(s.get(id).assessment.stale,true);
    assert.equal((s.db.prepare("SELECT count(*) n FROM assessments WHERE json_extract(data,'$.status')='failed'").get() as any).n,2);
    assert.equal((s.db.prepare("SELECT count(*) n FROM assessments WHERE json_extract(data,'$.status') IS NULL").get() as any).n,1);
    const stale=context();stale.version='old-preferences';stale.preferences={obsolete_exclusion:true};s.saveContext(stale);
    const updated=await assess(s,id,'openai',async(_p,prompt)=>{assert.ok(!prompt.includes('obsolete_exclusion'));return good;});assert.notEqual(updated.contextVersion,'old-preferences');assert.deepEqual(s.context().preferences,searchConfig().preferences);
    const large=context();large.version='many-paragraphs';large.sources[0].text=Array.from({length:1200},(_,i)=>`Paragraph ${i}: Built Solidity staking contracts, TypeScript services and reviewed production systems.`).join('\n\n');s.saveContext(large);
    const budgeted=await assess(s,id,'openai',async(p,prompt)=>{assert.ok(prompt.length<=p.context_chars);return {score:4,summary:'Budget fixture',evidence:[],gaps:[]};});assert.ok(budgeted.selection[0].omittedChars>0);assert.equal(s.context().sources[0].text.length,large.sources[0].text.length);
  }finally{s.close();}
});
test('both local HTTP adapters use structured output; malformed JSON, HTTP and timeout errors surface',async()=>{
  let mode='good',seen:any;const server=createServer((req,res)=>{
    let body='';req.on('data',b=>body+=b);req.on('end',()=>{seen=JSON.parse(body);if(mode==='timeout')return;if(mode==='error'){res.writeHead(503);res.end('{}');return;}
      const content=mode==='malformed'?'{bad':JSON.stringify({score:1,summary:'Fixture response',evidence:[],gaps:[]});res.setHeader('content-type','application/json');res.end(JSON.stringify(req.url==='/api/chat'?{message:{content}}:{choices:[{message:{content}}]}));});
  });await new Promise<void>(r=>server.listen(0,'127.0.0.1',r));const port=(server.address() as any).port;
  const p:Provider={type:'compatible',model:'fixture',endpoint:`http://127.0.0.1:${port}/v1`,timeout_ms:150,context_chars:2000};
  try{
    assert.equal(assessmentSchema.parse(await infer(p,'Minimal fixture')).score,1);assert.equal(seen.response_format.type,'json_schema');assert.equal(seen.tools,undefined);
    assert.equal(assessmentSchema.parse(await infer({...p,type:'ollama',endpoint:`http://127.0.0.1:${port}`},'Minimal fixture')).score,1);assert.equal(seen.format.type,'object');
    mode='malformed';await assert.rejects(()=>infer(p,'x'),/JSON/);mode='error';await assert.rejects(()=>infer(p,'x'),/503/);mode='timeout';await assert.rejects(()=>infer(p,'x'));
    await assert.rejects(()=>infer(p,'x'.repeat(2001)),/overflow/);await assert.rejects(()=>infer({...p,endpoint:'https://public.example'},'x'),/loopback/);
  }finally{server.closeAllConnections();await new Promise<void>(r=>server.close(()=>r()));}
});
test('CLI adapters parse real subprocess protocol with controlled fixture; timeout kills process group',async()=>{
  const dir=temp(),bin=resolve(dir,'bin');mkdirSync(bin);const fixture=rootPath('legacy/app/test/fake-cli.cjs');chmodSync(fixture,0o755);
  symlinkSync(fixture,resolve(bin,'opencode'));symlinkSync(fixture,resolve(bin,'claude'));
  const old=process.env.PATH;process.env.PATH=`${bin}:${old}`;
  try {
    for(const type of ['opencode','claude'] as const) {
      const p:Provider={type,model:'fixture',timeout_ms:2000,context_chars:2000};
      assert.equal(assessmentSchema.parse(await infer(p,'minimal fixture')).score,7);
      await assert.rejects(()=>infer(p,'FCTO_TEST_MALFORMED'),/JSON/);
      await assert.rejects(()=>infer({...p,timeout_ms:150},'FCTO_TEST_TIMEOUT'),/timeout/);
    }
    process.env.FCTO_TEST_SECRET='must not inherit';assert.equal(isolatedEnv().FCTO_TEST_SECRET,undefined);assert.equal(isolatedEnv().OPENCODE_CONFIG_CONTENT,undefined);delete process.env.FCTO_TEST_SECRET;
    await assert.rejects(()=>runProcess(process.execPath,['-e','setTimeout(()=>{},30000)'],'',50,isolatedEnv(),dir),/timeout/);
    assert.throws(()=>parseJson('not JSON'));
  }finally{process.env.PATH=old;rmSync(dir,{recursive:true,force:true});}
});
test('scheduler excludes overlaps, recovers dead owner and honours same-day catch-up',async()=>{
  const dir=temp(),file=resolve(dir,'state.sqlite'),a=new Store(file),b=new Store(file);
  try {
    const release=acquireLock(a);assert.throws(()=>acquireLock(b),/already active/);release();
    a.db.prepare("INSERT INTO locks VALUES('daily',99999999,?)").run(now());a.db.prepare("INSERT INTO runs(started,status) VALUES(?,'running')").run(now());
    const releaseRecovered=acquireLock(b);assert.equal((a.db.prepare('SELECT status FROM runs').get() as any).status,'interrupted');releaseRecovered();
    a.db.prepare("INSERT INTO runs(started,finished,status) VALUES(?,?,'complete')").run(now(),now());assert.ok('skipped' in await daily(a,true));
    assert.equal((a.db.prepare('SELECT count(*) n FROM locks').get() as any).n,0);
  }finally{a.close();b.close();rmSync(dir,{recursive:true,force:true});}
});
test('remote import blocks private destinations before requests',async()=>{
  const scheme='http',fixtureUser='user',fixturePassword='password';
  const credentialUrl=scheme+'://'+fixtureUser+':'+fixturePassword+'@example.com';
  for(const url of ['http://127.0.0.1:1','http://localhost','http://[::1]','http://169.254.169.254','file:///etc/passwd',credentialUrl]) await assert.rejects(()=>publicUrl(url));
});
test('per-source proxy environment reference routes a real HTTP request through a controlled CONNECT proxy',async()=>{
  let destination='',authorization='';
  const proxy=createServer();proxy.on('connect',(req,socket)=>{
    destination=req.url??'';authorization=String(req.headers['proxy-authorization']??'');
    socket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
    socket.once('data',()=>{const body='<h1>Controlled proxy fixture</h1>';socket.end(`HTTP/1.1 200 OK\r\nContent-Length: ${Buffer.byteLength(body)}\r\nContent-Type: text/html\r\nConnection: close\r\n\r\n${body}`);});
  });await new Promise<void>(r=>proxy.listen(0,'127.0.0.1',r));
  try {
    const port=(proxy.address() as any).port;
    const config={version:1,profiles:{direct:{},selected:{proxy_env:'FCTO_PROXY_FIXTURE'}},sources:{Fixture:'selected'}};
    const scheme='http',proxyUser='fixture',proxyPassword='dummy';
    const proxyUrl=scheme+'://'+proxyUser+':'+proxyPassword+'@127.0.0.1:'+port;
    const profile=networkProfile('Fixture',config,{FCTO_PROXY_FIXTURE:proxyUrl});
    const p=await fetchPage('http://93.184.216.34/jobs/fixture','Fixture',2000,profile);
    assert.match(p.text,/Controlled proxy/);assert.equal(destination,'93.184.216.34:80');assert.match(authorization,/^Basic /);
    assert.throws(()=>networkProfile('Fixture',config,{}),/environment reference/);
  }finally{proxy.closeAllConnections();await new Promise<void>(r=>proxy.close(()=>r()));}
});
test('dedicated Chromium source sessions persist separately without using owner accounts',async()=>{
  const a=`fixture-session-a-${process.pid}`,b=`fixture-session-b-${process.pid}`;
  let c:Awaited<ReturnType<typeof browserSession>>|undefined;
  try{
    c=await browserSession(a);await c.addCookies([{name:'synthetic_fixture',value:'test-only',domain:'example.com',path:'/',expires:Math.floor(Date.now()/1000)+3600}]);await c.close();
    c=await browserSession(a);assert.ok((await c.cookies()).some(v=>v.name==='synthetic_fixture'));await c.close();
    c=await browserSession(b);assert.ok(!(await c.cookies()).some(v=>v.name==='synthetic_fixture'));await c.close();c=undefined;
  }finally{await c?.close();rmSync(sessionDirectory(a),{recursive:true,force:true});rmSync(sessionDirectory(b),{recursive:true,force:true});}
});
test('actual MCP stdio shares persisted state, import/search errors and tool-free assessment',async()=>{
  const dir=temp(),state=relative(ROOT,dir),bin=resolve(dir,'bin');mkdirSync(bin);symlinkSync(rootPath('legacy/app/test/fake-cli.cjs'),resolve(bin,'opencode'));
  const s=new Store(resolve(dir,'jobs.sqlite'));s.saveContext(context());const id=s.upsert(input());s.status(id,'saved');
  const transport=new StdioClientTransport({command:process.execPath,args:['--import','tsx',rootPath('legacy/app/src/mcp.ts')],cwd:rootPath('legacy/app'),env:{...process.env as Record<string,string>,FCTO_STATE_DIR:state,PATH:`${bin}:${process.env.PATH}`},stderr:'pipe'});
  const client=new Client({name:'integration-test',version:'1.0.0'});let stderr='';
  try {
    await client.connect(transport);transport.stderr?.on('data',b=>stderr+=b);
    const list=await client.listTools();assert.equal(list.tools.length,15);assert.ok(list.tools.some(t=>t.name==='application_cancel'));assert.ok(!list.tools.some(t=>/submit|fill/i.test(t.name)));
    const r:any=await client.callTool({name:'jobs_list',arguments:{status:'saved'}});assert.equal(JSON.parse(r.content[0].text)[0].id,id);
     const detail:any=await client.callTool({name:'jobs_detail',arguments:{id}});assert.equal(JSON.parse(detail.content[0].text).status,'saved');
     s.seen(id);const applied=s.status(id,'applied',true);
     const canonicalList:any=await client.callTool({name:'jobs_list',arguments:{status:'applied'}});const shared=JSON.parse(canonicalList.content[0].text)[0];assert.equal(shared.id,id);assert.equal(shared.stage,'applied');assert.equal(shared.seen,true);assert.equal(shared.appliedAt,applied.appliedAt);
     const cli=JSON.parse(await runProcess(process.execPath,['--import','tsx',rootPath('legacy/app/src/cli.ts'),'list','applied'],'',5000,{...process.env as Record<string,string>,FCTO_STATE_DIR:state},rootPath('legacy/app')));assert.equal(cli[0].appliedAt,shared.appliedAt);assert.equal(cli[0].seenAt,shared.seenAt);
     s.status(id,'preselected');const aliasList:any=await client.callTool({name:'jobs_list',arguments:{status:'shortlist'}});assert.equal(JSON.parse(aliasList.content[0].text)[0].appliedAt,null);
     const question:any=await client.callTool({name:'application_question',arguments:{id,label:'Fixture MCP question'}});const draft=JSON.parse(question.content[0].text),field=draft.fields[0];
     const updated:any=await client.callTool({name:'application_update',arguments:{id,fieldId:field.id,revision:0,value:'Fixture draft only'}});assert.equal(JSON.parse(updated.content[0].text).fields[0].value,'Fixture draft only');
     const sharedDraft:any=await client.callTool({name:'application_draft',arguments:{id}});assert.equal(JSON.parse(sharedDraft.content[0].text).draft.fields[0].value,'Fixture draft only');
    const imported:any=await client.callTool({name:'jobs_import',arguments:{url:'http://127.0.0.1:1'}});assert.equal(imported.isError,true);
    const search:any=await client.callTool({name:'jobs_search',arguments:{source:'Unknown fixture source'}});assert.equal(search.isError,true);
    const a:any=await client.callTool({name:'jobs_assess',arguments:{id,provider:'openai'}});assert.ok(!a.isError,JSON.stringify(a));assert.equal(s.get(id).assessment.score,7);
  }finally{await client.close();await transport.close();s.close();rmSync(dir,{recursive:true,force:true});}
});
