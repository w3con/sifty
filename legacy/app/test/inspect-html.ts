import { readFileSync } from 'node:fs';
import { load } from 'cheerio';
import { localPath } from '../src/core.js';
// Bounded DOM diagnostics for minified source pages (line-based tools cannot render them).
for (const name of process.argv.slice(2)) {
  const $ = load(readFileSync(localPath(`verification/${name}.html`),'utf8'));
  console.log(name);
  console.log(JSON.stringify($('a[href]').map((_,el) => {const a=$(el);return {href:a.attr('href'),text:a.text().replace(/\s+/g,' ').slice(0,160),class:a.attr('class'),parent:a.parent().attr('class')};}).get().filter(a => /engineer|developer|next|page|jobs|software/i.test(`${a.href} ${a.text}`)).slice(0,40),null,2));
  console.log('Pagination',JSON.stringify($('[aria-label],button').map((_,e) => ({text:$(e).text().slice(0,100),aria:$(e).attr('aria-label'),data:$(e).attr('data-testid')})).get().filter(e => /next|more|page/i.test(JSON.stringify(e))).slice(0,15)));
  console.log('Scripts',$('script[src]').map((_,e)=>$(e).attr('src')).get().slice(-5));
  for (const selector of ['a.job-link','h2 a','a[href*="/companies/"]']) {
    const e=$(selector).first();if(e.length) console.log('Sample ancestors',e.parent().parent().html()?.slice(0,3500));
  }
  const next=$('#__NEXT_DATA__').text(); if(next) {const d=JSON.parse(next);console.log('InitialState',JSON.stringify(d.props?.pageProps?.initialState??{}).slice(0,14000));}
}
