import { load } from 'cheerio';
import { fetchPage } from '../src/network.js';
for (const suffix of ['', '?page=2','?skip=20']) {
  const p=await fetchPage(`https://jobs.electriccapital.com/jobs${suffix}`,'Electric Capital');
  const $=load(p.text),d=JSON.parse($('#__NEXT_DATA__').text());
  const j=d.props?.pageProps?.initialState?.jobs;
  console.log(JSON.stringify({url:p.url,firstId:j?.found?.[0]?.id,count:j?.found?.length,keys:Object.keys(j??{}),meta:Object.fromEntries(Object.entries(j??{}).filter(([k])=>k!=='found'))}));
}
