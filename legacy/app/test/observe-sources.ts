import './isolate.js';
import { chromium } from 'playwright';
import { load } from 'cheerio';
import { mkdirSync,writeFileSync } from 'node:fs';
import { localPath } from '../src/core.js';
const targets:Record<string,string>={web3:'https://web3.career',web3plural:'https://web3.careers',braintrust:'https://www.usebraintrust.com/jobs',a16z:'https://a16zfrontier.com/jobs/',wttj:'https://www.welcometothejungle.com/en/jobs',malt:'https://www.malt.fr',crypto:'https://cryptojobslist.com',getro:'https://jobs.electriccapital.com/jobs',cryptocurrency:'https://cryptocurrencyjobs.co',wellfound:'https://wellfound.com/jobs',wwr:'https://weworkremotely.com'};
const browser=await chromium.launch({headless:true});const dir=localPath('observations');mkdirSync(dir,{recursive:true});
targets['braintrust-app']='https://app.usebraintrust.com/jobs/';
targets['wttj-root']='https://www.welcometothejungle.com/en';
targets['wttj-fr']='https://www.welcometothejungle.com/fr/jobs';
targets['wellfound-role']='https://wellfound.com/role/l/software-engineer/united-states';
try{for(const name of process.argv.slice(2).length?process.argv.slice(2):Object.keys(targets)){
  const context=await browser.newContext({serviceWorkers:'block'}),page=await context.newPage();page.setDefaultTimeout(5000);
  const requests:any[]=[],responses:any[]=[];const pending:Promise<void>[]=[];
  await context.route('**/*',route=>{const r=route.request();if(!['GET','HEAD'].includes(r.method())){requests.push({url:r.url(),method:r.method(),body:r.postData()?.slice(0,3000),blocked:true});return route.abort();}return route.continue();});
  page.on('response',r=>{const req=r.request();if(['xhr','fetch'].includes(req.resourceType()))pending.push((async()=>{let body='';try{body=(await r.text()).slice(0,1500000);}catch{}responses.push({url:r.url(),status:r.status(),body});})());});
  let error='';try{await page.goto(targets[name],{timeout:30000,waitUntil:'domcontentloaded'});await page.waitForTimeout(5000);}catch(e){error=String(e);}
  // Only read-only discovery controls are exercised; no accounts or application controls.
  if(name==='getro')try{await page.getByTestId('load-more').click({timeout:5000});await page.waitForTimeout(2000);}catch(e){error+=String(e);}
  let html='';try{html=await page.content();}catch(e){error+=String(e);}const $=load(html);
  const links=$('a[href]').map((_,e)=>({text:$(e).text().trim().replace(/\s+/g,' ').slice(0,140),href:$(e).attr('href')})).get().filter(a=>/jobs|careers|talent|freelanc|mission|search|next|engineer|developer|emploi/i.test(`${a.href} ${a.text}`)).slice(0,100);
  const inputs=await page.locator('input,select,iframe,form').evaluateAll(es=>es.map(e=>({tag:e.tagName,placeholder:e.getAttribute('placeholder'),name:e.getAttribute('name'),src:e.getAttribute('src'),action:e.getAttribute('action')}))).catch(()=>[]);
  await Promise.race([Promise.allSettled(pending),new Promise(r=>setTimeout(r,2500))]);
  const summary={name,url:page.url(),title:await page.title().catch(()=>''),error,body:clean($('body').text()).slice(0,6000),links,inputs,requests,responses:responses.map(({body,...r})=>({...r,bodyPreview:body.slice(0,600)})),scripts:$('script[src]').map((_,e)=>$(e).attr('src')).get()};
  writeFileSync(`${dir}/${name}.html`,html);writeFileSync(`${dir}/${name}.json`,JSON.stringify({...summary,responses},null,2));console.log(JSON.stringify({...summary,body:summary.body.slice(0,1500),requests:requests.filter(r=>!/analytics|cdn-cgi|awswaf|collect/.test(r.url)).slice(0,8),links:links.slice(0,15)}));await context.close();
}}finally{await browser.close();}console.log(`Evidence: ${dir}`);
function clean(t:string){return t.replace(/\s+/g,' ').trim();}
