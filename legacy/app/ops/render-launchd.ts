// Generates a LOCAL template only. Never installs, loads or modifies a launchd service.
import { readFileSync,writeFileSync,mkdirSync } from 'node:fs';
import { homedir } from 'node:os';
import { localPath,ROOT,rootPath,runtime } from '../src/core.js';
const c=runtime(),escape=(s:string)=>s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
const values:Record<string,string>={ROOT,HOME:homedir(),NODE:process.execPath,HOUR:String(c.daily_hour),MINUTE:String(c.daily_minute)};
const text=readFileSync(rootPath('legacy/app/ops/eu.fcto.jobs.plist.template'),'utf8').replace(/__(ROOT|HOME|NODE|HOUR|MINUTE)__/g,(_,k)=>escape(values[k]));
mkdirSync(localPath('logs'),{recursive:true,mode:0o700});writeFileSync(localPath('eu.fcto.jobs.plist'),text,{mode:0o600});
console.log('Generated .local/eu.fcto.jobs.plist. Nothing installed. Review and follow legacy/app/README.md on the operational mini.');
