import './isolate.js';
import {readFileSync} from 'node:fs';import {load} from 'cheerio';import {localPath} from '../src/core.js';
for(const task of process.argv.slice(2)){
 const file=task==='web3'?'web3-fetch.html':task==='wellfound'?'wellfound-engineering.txt':task==='wttj'?'wttj-fr.txt':'a16z.html';
 const $=load(readFileSync(localPath(`observations/${file}`),'utf8'));
 console.log('Discovery',JSON.stringify({task,jobLinks:$('a[href*="/jobs/"]').map((_,e)=>({t:$(e).text().replace(/\s+/g,' ').trim().slice(0,120),h:$(e).attr('href')})).get().slice(0,8),pagination:$('a[href]').map((_,e)=>$(e).attr('href')!).get().filter(s=>/[?&]page=/.test(s)),scripts:$('script[src]').map((_,e)=>$(e).attr('src')).get().slice(-6)}));
 console.log(task,JSON.stringify({links:$('a[href]').map((_,e)=>({text:$(e).text().replace(/\s+/g,' ').trim().slice(0,100),href:$(e).attr('href'),rel:$(e).attr('rel')})).get().filter(a=>/page|next|jobs|search|engineer|cto|developer/i.test(JSON.stringify(a))).slice(-35),inputs:$('input,form').map((_,e)=>({tag:e.tagName,name:$(e).attr('name'),placeholder:$(e).attr('placeholder'),action:$(e).attr('action')})).get()}));
 console.log('Cards',$('tr,h2 a').slice(0,4).map((_,e)=>$.html(e).slice(0,2500)).get());
 const next=$('#__NEXT_DATA__').text();if(next){const d=JSON.parse(next);console.log('Next',JSON.stringify(d.props?.pageProps??{}).slice(0,10000));}
}
