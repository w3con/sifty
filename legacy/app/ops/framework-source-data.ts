import {readFileSync} from 'node:fs';import {load} from 'cheerio';
const $=load(readFileSync(process.argv[2],'utf8')),job=JSON.parse($('#__NEXT_DATA__').text()).props.pageProps.initialState.jobs.currentJob;
console.log(JSON.stringify({id:job.id,slug:job.slug,title:job.title,applicationMethod:job.applicationMethod,applicationPath:job.applicationPath,url:job.url},null,2));
