import { readFileSync, writeFileSync } from 'node:fs';
import { structuredJobs, parseFeed, htmlLinks } from './src/collectors.ts';
const TD = process.argv[2], OUT = process.argv[3];
const read = f => readFileSync(`${TD}/${f}`,'utf8');
const golden = {
  jsonld: structuredJobs(read('jsonld.html'),'https://acme.test/jobs/cto','Test Board'),
  feed: parseFeed(read('feed.xml'),'https://acme.test/feed','Test Feed'),
  links_getro: htmlLinks(read('links_getro.html'),'https://gamma.test/','Gamma Board'),
  links_anchors: htmlLinks(read('links_anchors.html'),'https://delta.test/careers','Delta Board'),
};
for (const [k,v] of Object.entries(golden)) writeFileSync(`${OUT}/${k}.json`, JSON.stringify(v,null,2)+'\n');
console.log('wrote goldens:', Object.keys(golden).join(', '));
