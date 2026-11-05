import { writeFileSync, mkdirSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { localPath, providers, runtime, searchConfig } from './core.js';
import { browserSession } from './network.js';
import { providerSmoke } from './providers.js';
import { Store } from './store.js';
import { assess, daily, enrich, exclusive, importUrl, refreshContext, search, sourceHealth, validateAll } from './services.js';
import { serve } from './server.js';

export async function main(args = process.argv.slice(2)) {
  const [command,...rest] = args; const store = new Store();
  const output = (o: unknown) => console.log(JSON.stringify(o,null,2));
  try {
    switch(command) {
      case 'serve': await serve(store); return;
      case 'validate': output(validateAll()); break;
      case 'context': {
        const c=await exclusive(store,() => refreshContext(store,searchConfig().preferences,rest.includes('--refresh')));
        output(rest.includes('--summary')?{version:c.version,reviewNotes:c.reviewNotes,sources:c.sources.map(({text,...s})=>({...s,characters:text.length}))}:c);break;
      }
      case 'search': output(await exclusive(store,() => search(store,rest.join(' ') || undefined))); break;
      case 'list': output(store.list(rest[0] ?? 'all',rest.slice(1).join(' '))); break;
      case 'detail': output(store.get(Number(rest[0]))); break;
      case 'status': output(store.status(Number(rest[0]),rest[1],rest.includes('--confirmed'))); break;
      case 'import': output({id:await exclusive(store,() => importUrl(store,rest[0]))}); break;
      case 'read': output({id:await exclusive(store,() => enrich(store,Number(rest[0])))}); break;
      case 'assess': output(await exclusive(store,() => assess(store,Number(rest[0]),rest[1]))); break;
      case 'health': output(sourceHealth(store)); break;
      case 'daily': output(await daily(store,rest.includes('--due'))); break;
      case 'provider-check': output(await exclusive(store,()=>providerSmoke(rest[0] ?? providers().stages.assessment),'inference')); break;
      case 'session': {
        const name = rest.join(' '), source = searchConfig().sites.find(s => s.name === name);
        if (!source) throw new Error('Use an exact configured source name');
        // Owner-controlled login only; no credential collection or automated interaction.
        const context = await browserSession(name,true); const page = await context.newPage(); await page.goto(source.url);
        console.error('Dedicated session open. Log in manually if needed, then close the browser. No application interaction is automated.');
        await new Promise<void>(resolve => context.on('close',() => resolve())); break;
      }
      case 'launchd-template': {
        const cfg = runtime();
        output({template:'app/ops/eu.fcto.jobs.plist.template',hour:cfg.daily_hour,minute:cfg.daily_minute,note:'Substitute explicit paths per README. No scheduler has been installed.'}); break;
      }
      case 'backup': {
        const name = `backups/jobs-${Date.now()}.sqlite`; mkdirSync(localPath('backups'),{recursive:true,mode:0o700});
        // VACUUM INTO makes an internally consistent backup including committed WAL data.
        store.db.prepare('VACUUM INTO ?').run(localPath(name)); output({backup:`.local/${name}`}); break;
      }
      default: output({commands:['validate','context [--refresh]','search [exact source name]','list [all|new|saved|rejected] [query]','detail ID','status ID new|saved|rejected','import URL','read ID','assess ID [provider]','health','provider-check [provider]','daily [--due]','serve','session SOURCE','launchd-template','backup']});
    }
  } finally { if (command !== 'serve') store.close(); }
}
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main().catch(e => { console.error(e.message); process.exitCode=1; });
