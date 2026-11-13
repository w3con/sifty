import { DatabaseSync } from 'node:sqlite';
import { localPath } from './core.js';

type State = { next: number; cooldown: number; failures: number; reason: string };
export class DomainTraffic {
  private tails = new Map<string, Promise<unknown>>();
  constructor(private read:(host:string)=>State|undefined,private write:(host:string,s:State)=>void,
    private clock=Date.now,private sleep=(ms:number)=>new Promise<void>(r=>setTimeout(r,ms)),private random=Math.random,
    private lease?:(host:string)=>Promise<()=>void>) {}
  async run<T>(url:string,work:()=>Promise<T>):Promise<T> {
    const host=new URL(url).hostname.replace(/^www\./,'').toLowerCase();
    const previous=this.tails.get(host)??Promise.resolve();
    const pending=previous.catch(()=>{}).then(async()=>{
      const release=await this.lease?.(host);
      try{
      const s=this.read(host)??{next:0,cooldown:0,failures:0,reason:''};
      if(s.cooldown>this.clock())throw new Error(`Domain ${host} paused until ${new Date(s.cooldown).toISOString()}: ${s.reason}`);
      await this.sleep(Math.max(0,s.next-this.clock()));
      try { const value=await work();s.failures=0;return value; }
      catch(e:any){
        s.failures++; const retry=e.retryAfter;
        const blocked=/403|CAPTCHA|access challenge|browser blocked/i.test(String(e));
        const ordinary=/HTTP 4\d\d/.test(String(e))&&!/HTTP (403|429)/.test(String(e));
        const duration=ordinary?0:blocked?24*3600000:retry??Math.min(3600000,30000*2**Math.min(s.failures-1,7));
        s.cooldown=this.clock()+duration;s.reason=String(e);throw e;
      } finally {s.next=this.clock()+3000+Math.floor(this.random()*3001);this.write(host,s);}
      }finally{release?.();}
    });
    this.tails.set(host,pending);try{return await pending;}finally{if(this.tails.get(host)===pending)this.tails.delete(host);}
  }
}
let db:DatabaseSync|undefined;
function database(){if(!db){db=new DatabaseSync(localPath('network.sqlite'));db.exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS domains(host TEXT PRIMARY KEY,data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS pages(key TEXT PRIMARY KEY,created INTEGER NOT NULL,data TEXT NOT NULL);`);}return db;}
async function domainLease(host:string){const db=database();db.exec('CREATE TABLE IF NOT EXISTS traffic_leases(host TEXT PRIMARY KEY,pid INTEGER NOT NULL)');for(;;){db.exec('BEGIN IMMEDIATE');try{const r=db.prepare('SELECT pid FROM traffic_leases WHERE host=?').get(host)as any;let alive=false;if(r)try{process.kill(r.pid,0);alive=true;}catch{}if(!alive){db.prepare('INSERT OR REPLACE INTO traffic_leases VALUES(?,?)').run(host,process.pid);db.exec('COMMIT');return ()=>{db.prepare('DELETE FROM traffic_leases WHERE host=? AND pid=?').run(host,process.pid);};}db.exec('COMMIT');}catch(e){db.exec('ROLLBACK');throw e;}await new Promise(r=>setTimeout(r,1000));}}
export const traffic=new DomainTraffic(host=>{const r=database().prepare('SELECT data FROM domains WHERE host=?').get(host)as any;return r?JSON.parse(r.data):undefined;},(host,s)=>{database().prepare('INSERT OR REPLACE INTO domains VALUES(?,?)').run(host,JSON.stringify(s));},Date.now,undefined,Math.random,domainLease);
export function cachedPage(key:string){const r=database().prepare('SELECT * FROM pages WHERE key=?').get(key)as any;return r?{created:r.created,...JSON.parse(r.data)}:null;}
export function savePage(key:string,data:unknown){database().prepare('INSERT OR REPLACE INTO pages VALUES(?,?,?)').run(key,Date.now(),JSON.stringify(data));}
export function httpError(status:number,retry?:string|string[]){const e=new Error(`HTTP ${status}${status===403?': access denied; owner session/proxy required':''}`)as Error&{retryAfter?:number};if(retry){const v=String(retry);e.retryAfter=/^\d+$/.test(v)?Number(v)*1000:Math.max(0,Date.parse(v)-Date.now());if(!Number.isFinite(e.retryAfter))delete e.retryAfter;}return e;}
export class RequestCache {
 private running=new Map<string,Promise<any>>();
 constructor(private read:(key:string)=>any,private write:(key:string,data:any)=>void,private clock=Date.now){}
 async get(key:string,ttl:number,load:(headers:Record<string,string>)=>Promise<any>,force=false,conditional=false):Promise<any>{
  const old=this.read(key);if(!force&&old&&this.clock()-old.created<ttl)return {...old.page,cached:true};
  if(this.running.has(key))return this.running.get(key);
  const request=(async()=>{const headers:Record<string,string>={};if(conditional&&old?.page.etag)headers['if-none-match']=old.page.etag;if(conditional&&old?.page.modified)headers['if-modified-since']=old.page.modified;const response=await load(headers);const page=response.status===304&&old?{...old.page,etag:response.etag??old.page.etag,modified:response.modified??old.page.modified}:response;if(response.status===304&&!old)throw new Error('304 response without cached body');this.write(key,{page,created:this.clock()});return page;})();
  this.running.set(key,request);try{return await request;}finally{this.running.delete(key);}
 }
}
export const pageCache=new RequestCache(cachedPage,savePage);
