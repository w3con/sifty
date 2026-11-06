import type {Store} from './store.js';
export function startArchiveRetention(store:Store,intervalMs=60000,clock=Date.now){
 const purge=()=>{try{const result=store.purgeArchive(clock());if(result.deleted.length||result.skipped.length)console.error('Archive retention:',JSON.stringify(result));}catch(e){console.error('Archive retention failed; retrying next interval:',String(e));}};
 purge();const timer=setInterval(purge,intervalMs);timer.unref();return ()=>clearInterval(timer);
}
