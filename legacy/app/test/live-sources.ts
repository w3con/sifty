import './isolate.js';
// Explicit read-only smoke. ALWAYS isolated from operational jobs/health/sessions.
import { mkdirSync, writeFileSync, readFileSync, existsSync } from 'node:fs';
import { localPath, searchConfig, now } from '../src/core.js';
import { collectSource, enrich } from '../src/collectors.js';
import { fetchPage } from '../src/network.js';
import { Store } from '../src/store.js';
import { exclusive } from '../src/services.js';
import { discoverSource } from '../src/discovery.js';
const store = new Store();
const dir = localPath('verification'); mkdirSync(dir,{recursive:true,mode:0o700});
try {
  await exclusive(store,async () => {
    const selected=process.argv.slice(2);
    const results:any[] = selected.length && existsSync(`${dir}/live-sources.json`) ? JSON.parse(readFileSync(`${dir}/live-sources.json`,'utf8')).results.filter((r:any)=>!selected.includes(r.source)) : [];
    for (const source of searchConfig().sites.filter(s => s.enabled && (!selected.length || selected.includes(s.name)))) {
      try {
        const result = await discoverSource(source,{limits:{max_pages:2,max_detail_requests:6,max_reviewed_per_source:200,max_jobs_per_source:10},get:async (url,s) => {
          const p = await fetchPage(url,s);
          if (url === source.url || /\/jobs$/.test(url)) writeFileSync(`${dir}/${source.name.replace(/[^a-z0-9]/gi,'_')}.html`,p.text);
          return p;
        }});
        // Import at most three per source during smoke; no inference in this script.
        const ids = result.jobs.slice(0,3).map(j => store.upsert(j));
        // Discovery already hydrates candidates. Never expand a smoke into operational enrichment.
        const uniqueIds=[...new Set(ids.map(id=>store.get(id).id))];
        store.health(source.name,result.status,`${result.message} Smoke persisted first ${ids.length} candidates (${uniqueIds.length} deduplicated) only.`,uniqueIds.length);
        const entry = {source:source.name,status:result.status,message:result.message,matched:result.coverage.matched,coverage:result.coverage,persisted:uniqueIds.map(id => {const j=store.get(id);return {id,title:j.title,company:j.company,original:j.originalUrl,apply:j.applyUrl,applyStatus:j.applyStatus,descriptionChars:j.description.length,descriptionStatus:j.descriptionStatus};})};
        results.push(entry); console.log(JSON.stringify(entry));
      } catch(e) { const entry={source:source.name,status:'error',message:String(e)};results.push(entry);store.health(source.name,'error',String(e));console.log(JSON.stringify(entry)); }
      writeFileSync(`${dir}/live-sources.json`,JSON.stringify({checked:now(),results},null,2));
    }
  });
} finally {store.close();}
