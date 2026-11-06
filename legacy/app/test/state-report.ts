import './isolate.js';
import { existsSync, readFileSync, readdirSync, writeFileSync, mkdirSync } from 'node:fs';
import { Store } from '../src/store.js';
import { hash,localPath,rootPath,now } from '../src/core.js';
import { providerDiagnostics } from '../src/providers.js';
import { sourceHealth } from '../src/services.js';
import { DatabaseSync } from 'node:sqlite';
const s=new Store();
try{
  const jobs=s.list(),c=s.context();
  mkdirSync(localPath('verification'),{recursive:true});
  const backups=existsSync(localPath('backups'))?readdirSync(localPath('backups')).filter(n=>n.endsWith('.sqlite')).sort():[];
  let backupIntegrity:unknown=null;
  if(backups.length){const b=new DatabaseSync(localPath(`backups/${backups.at(-1)}`),{readOnly:true});try{backupIntegrity=b.prepare('PRAGMA integrity_check').get();}finally{b.close();}}
  const report={checked:now(),jobs:jobs.length,assessed:jobs.filter(j=>j.assessment).map(j=>({id:j.id,company:j.company,title:j.title,score:j.assessment.score,error:j.assessmentError,stale:j.assessment.stale,contextVersion:j.assessment.contextVersion})),statusCounts:Object.fromEntries(['new','saved','rejected'].map(status=>[status,jobs.filter(j=>j.status===status).length])),integrity:s.db.prepare('PRAGMA integrity_check').get(),backupIntegrity,activeLocks:s.db.prepare('SELECT * FROM locks').all(),candidate:c?.sources.map((v:any)=>({id:v.id,status:v.status,characters:v.text.length,category:v.category,unchangedLocalInput:v.reference&&!v.reference.startsWith('http')&&existsSync(rootPath(v.reference))?hash(readFileSync(rootPath(v.reference)))===v.hash:undefined})),providers:await providerDiagnostics(),sourceHealth:sourceHealth(s)};
  writeFileSync(localPath('verification/state-report.json'),JSON.stringify(report,null,2));console.log(JSON.stringify(report,null,2));
}finally{s.close();}
