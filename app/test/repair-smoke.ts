import './isolate.js';
// Historical one-time repair; default execution is now isolated and cannot touch operational jobs.
import assert from 'node:assert/strict';
import { Store } from '../src/store.js';
const s=new Store();
try{
  const j=s.list().find(j=>j.originalUrl==='https://news.ycombinator.com/item?id=49925643');
  if(j){
    assert.match(j.title,/popular request of including a 4DWW tag/);assert.equal(j.status,'new');assert.equal(j.assessment,null);
    s.db.exec('BEGIN IMMEDIATE');
    try{for(const table of ['sightings','snapshots','assessments'])s.db.prepare(`DELETE FROM ${table} WHERE job_id=?`).run(j.id);s.db.prepare('DELETE FROM jobs WHERE id=?').run(j.id);s.db.exec('COMMIT');console.log(`Removed executor-created non-vacancy smoke record ${j.id}; no owner-saved record touched.`);}catch(e){s.db.exec('ROLLBACK');throw e;}
  }else console.log('No executor meta-comment remains.');
}finally{s.close();}
