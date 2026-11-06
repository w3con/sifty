import './isolate.js';
import test from 'node:test';
import assert from 'node:assert/strict';
import { Store } from '../src/store.js';
import { localPath } from '../src/core.js';
import { externalApplyUrl, isWeb3 } from '../src/workflow.js';
import {nextUndecided,visibleInStage,quickMatch,emptyQuick} from '../web/workflow.js';

test('legacy migration preserves records, evidence/history; stages, seen and explicit confirmation survive restart',()=>{
  const path=localPath('workflow.sqlite'),s=new Store(path);
  const ids=['new','saved','rejected'].map((status,i)=>{
    const id=s.upsert({title:`Engineer ${i}`,company:`Fixture ${i}`,originalUrl:`https://fixture.test/${i}`,description:'Solidity engineer',descriptionStatus:'fixture',source:'fixture'});
    s.status(id,status);s.assessment(id,{summary:'unchanged',score:80,evidence:[]});return id;
  });
  // Recreate a pre-v8 database with genuine legacy statuses and existing related data.
  s.db.exec('DROP TABLE workflow_history; ALTER TABLE jobs DROP COLUMN seen_at; ALTER TABLE jobs DROP COLUMN applied_at; DELETE FROM migrations WHERE version=8;');s.close();
  const migrated=new Store(path);
  assert.deepEqual(ids.map(id=>migrated.get(id).stage),['triage','preselected','no']);
  assert.deepEqual(ids.map(id=>migrated.get(id).status),['new','saved','rejected']);
  assert.equal(migrated.db.prepare('SELECT count(*) n FROM assessments').get()!.n,3);
  const before=JSON.stringify(migrated.get(ids[0]).assessment);
  const seen=migrated.seen(ids[0]);assert.ok(seen.seenAt);assert.equal(JSON.stringify(seen.assessment),before);
  assert.equal(migrated.seen(ids[0]).seenAt,seen.seenAt);
  assert.throws(()=>migrated.status(ids[1],'applied'),/explicit/);
  assert.throws(()=>migrated.status(ids[0],'applied',true),/pre-selected/);
  const applied=migrated.status(ids[1],'applied',true);assert.ok(applied.appliedAt);
  migrated.close();const reopened=new Store(path);
  assert.equal(reopened.get(ids[1]).appliedAt,applied.appliedAt);
  assert.equal(reopened.list('applied')[0].id,ids[1]);
  reopened.upsert({title:'Engineer 1',company:'Fixture 1',originalUrl:'https://fixture.test/1',description:'Updated Solidity engineer',descriptionStatus:'fixture',source:'fixture'});
  assert.equal(reopened.get(ids[1]).stage,'applied');assert.equal(reopened.get(ids[1]).appliedAt,applied.appliedAt);
  assert.equal(reopened.status(ids[1],'saved').appliedAt,null);
  assert.equal(reopened.list('preselected')[0].id,ids[1]);
  assert.equal(reopened.get(ids[0]).seenAt,seen.seenAt);
  assert.equal(reopened.db.prepare('SELECT count(*) n FROM workflow_history').get()!.n,2);
  reopened.close();
});
test('Web3 uses word-aware real content and external URLs accept only credential-free HTTP(S)',()=>{
  assert.equal(isWeb3({title:'Cryptography engineer',description:'blockchainless service'}),false);
  assert.equal(isWeb3({title:'Engineer',description:'Build smart contracts'}),true);
  assert.equal(isWeb3({title:'Engineer',description:'',tags:['DeFi']}),true);
  assert.equal(externalApplyUrl({applyUrl:'javascript:alert(1)',originalUrl:'https://fixture.test/job'}),'https://fixture.test/job');
  const scheme='https',fixtureUser='user',fixturePassword='pass';
  const credentialUrl=scheme+'://'+fixtureUser+':'+fixturePassword+'@fixture.test';
  assert.equal(externalApplyUrl({applyUrl:credentialUrl,originalUrl:'file:///tmp/job'}),null);
});
test('wrapped auto-advance skips decided roles; session retention does not become durable filter state',()=>{
  const s=new Store(':memory:');const ids=[0,1,2].map(i=>s.upsert({title:`Engineer ${i}`,company:`Fixture ${i}`,originalUrl:`https://fixture.test/${i}`,description:'',descriptionStatus:'missing',source:'fixture'}));
  s.status(ids[1],'no');const jobs=ids.map(id=>s.get(id));
  assert.equal(nextUndecided(jobs,ids[2]),ids[0]);assert.equal(nextUndecided(jobs,ids[0]),ids[2]);
  assert.equal(visibleInStage(jobs[1],'triage',new Set([ids[1]])),true);assert.equal(visibleInStage(jobs[1],'triage',new Set()),false);
  assert.equal(quickMatch(jobs[0],{...emptyQuick(),strong:true},''),false);s.close();
});
test('late merge preserves confirmed application date, seen, aliases and transition history',()=>{
  const s=new Store(':memory:');const base={title:'Unknown role',company:'Unknown',description:'',descriptionStatus:'missing',source:'fixture'};
  const a=s.upsert({...base,originalUrl:'https://board.test/a'}),b=s.upsert({...base,title:'Engineer',company:'Employer',originalUrl:'https://board.test/b',applyUrl:'https://employer.test/apply'});
  s.status(b,'preselected');const applied=s.status(b,'applied',true);const seen=s.seen(b);
  s.upsert({...base,title:'Engineer',company:'Employer',originalUrl:'https://board.test/a',applyUrl:'https://employer.test/apply'});
  assert.equal(s.get(b).id,a);assert.equal(s.get(a).appliedAt,applied.appliedAt);assert.equal(s.get(a).seenAt,seen.seenAt);assert.equal(s.get(a).stage,'applied');assert.equal(s.get(a).sources.length,2);
  assert.equal(s.db.prepare('SELECT count(*) n FROM workflow_history WHERE job_id=?').get(a)!.n,2);s.close();
});
