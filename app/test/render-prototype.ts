import './isolate.js';
import {chromium} from 'playwright';
import {mkdirSync,writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';
import {resolve} from 'node:path';
import {localPath} from '../src/core.js';
if(!process.argv[2])throw new Error('Pass the handoff directory containing Jobs Workflow.dc.html and support.js');
const browser=await chromium.launch({headless:true}),page=await browser.newPage({viewport:{width:1440,height:1000}}),dir=localPath('verification');mkdirSync(dir,{recursive:true});
const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));
try{
  await page.goto(pathToFileURL(resolve(process.argv[2],'Jobs Workflow.dc.html')).href);
  await page.getByRole('heading',{name:'AI Native Developer',exact:true}).waitFor({timeout:45000});await page.evaluate(()=>document.fonts.ready);
  await page.screenshot({path:`${dir}/prototype-desktop.png`});await page.getByRole('button',{name:'Settings',exact:true}).click();await page.getByRole('heading',{name:'Configuration',exact:true}).waitFor();await page.screenshot({path:`${dir}/prototype-settings.png`});
  writeFileSync(`${dir}/prototype.json`,JSON.stringify({source:process.argv[2],errors},null,2));console.log(JSON.stringify({screenshots:dir,errors}));
}finally{await browser.close();}
