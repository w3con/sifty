import './isolate.js';
import { readFileSync } from 'node:fs';
import { localPath } from '../src/core.js';
const js=readFileSync(localPath('observations/getro-_app-a87a6b223201483e.js'),'utf8');
for(const pattern of [/static async fetch\(e,t\)/g,/static async (?:search|jobs|list)\(/g,/hits_per_page/g,/type:"jobs"/g,/\/jobs"/g]){
 const matches=[...js.matchAll(pattern)].slice(0,8);console.log(String(pattern));for(const m of matches)console.log(js.slice(Math.max(0,m.index!-500),m.index!+2000));
}
const d=JSON.parse(readFileSync(localPath('observations/braintrust-app.json'),'utf8'));
const r=d.responses.find((r:any)=>/api\/jobs\/$/.test(r.url));if(r){const b=JSON.parse(r.body);console.log('Braintrust',JSON.stringify({count:b.count,next:b.next,keys:Object.keys(b.results[0]),first:b.results[0]},null,2).slice(0,9000));}
