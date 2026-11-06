import './isolate.js';
import test from 'node:test';
import assert from 'node:assert/strict';
import {chromium} from 'playwright';
import {localPath} from '../src/core.js';
import {openApplicationLogin} from '../src/application-browser.js';

test('owner login reuses the startup page and page close releases the persistent profile with its session intact',async()=>{
 const profile=localPath('login-close-profile'),context=await chromium.launchPersistentContext(profile,{headless:true});
 await context.route('**/*',r=>r.fulfill({contentType:'text/html',body:'<html><title>Owner login fixture</title><body>Login handoff fixture</body></html>'}));
 await context.addCookies([{name:'fixture-session',value:'retained',domain:'fixture.test',path:'/',expires:Math.floor(Date.now()/1000)+3600}]);
 const first=context.pages()[0],login=openApplicationLogin('https://fixture.test/login',{context:async()=>context,validate:async u=>new URL(u)});
 try{
  await first.waitForURL('https://fixture.test/login');await first.waitForLoadState('domcontentloaded');
  assert.equal(context.pages().length,1);assert.equal(context.pages()[0],first);
  await first.close();let timeout:ReturnType<typeof setTimeout>|undefined;
  try{await Promise.race([login,new Promise((_,reject)=>{timeout=setTimeout(()=>reject(Error('Login did not finish after owner page closed')),3000);})]);}finally{clearTimeout(timeout);}
 }finally{await context.close();}
 const restored=await chromium.launchPersistentContext(profile,{headless:true});
 try{assert.ok((await restored.cookies('https://fixture.test')).some(c=>c.name==='fixture-session'&&c.value==='retained'));}finally{await restored.close();}
});
test('cancelling owner login closes only the owned browser and preserves its authenticated profile',async()=>{
 const profile=localPath('login-cancel-profile'),context=await chromium.launchPersistentContext(profile,{headless:true}),other=await chromium.launch();const unrelated=await other.newPage();
 await context.route('**/*',r=>r.fulfill({contentType:'text/html',body:'<html><title>Login cancellation fixture</title></html>'}));await context.addCookies([{name:'retained',value:'yes',domain:'fixture.test',path:'/',expires:Math.floor(Date.now()/1000)+3600}]);const signal=new AbortController(),first=context.pages()[0],login=openApplicationLogin('https://fixture.test/login',{context:async()=>context,validate:async u=>new URL(u),signal:signal.signal});const rejected=assert.rejects(login,/abort/i);
 try{await first.waitForURL('https://fixture.test/login');signal.abort();await rejected;assert.equal(context.pages().length,0);assert.equal(unrelated.isClosed(),false);}finally{await context.close();await other.close();}
 const restored=await chromium.launchPersistentContext(profile,{headless:true});try{assert.ok((await restored.cookies('https://fixture.test')).some(c=>c.name==='retained'));}finally{await restored.close();}
});
