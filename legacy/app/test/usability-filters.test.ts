import './isolate.js';import test from 'node:test';import assert from 'node:assert/strict';
import {Store} from '../src/store.js';import {emptySelection,filterCatalogue,managementTitle,migrateSelection,sortCatalogue} from '../../../frontend/filters.js';
test('configured role/skill groups distinguish all, none and OR subsets; migration preserves legacy choices',()=>{
 const store=new Store(':memory:');for(const [i,title,description] of [[1,'CTO','TypeScript'],[2,'Engineer','Rust'],[3,'Director','Rust']] as const)store.upsert({title,company:`Fixture ${i}`,description,descriptionStatus:'fixture',originalUrl:`https://fixture.test/${i}`,source:'fixture'});
 const jobs=store.list();assert.equal(filterCatalogue(jobs,emptySelection()).length,3);assert.equal(filterCatalogue(jobs,{...emptySelection(),titles:[]}).length,0);assert.equal(filterCatalogue(jobs,{...emptySelection(),keywords:[]}).length,0);
 assert.deepEqual(filterCatalogue(jobs,{...emptySelection(),titles:['CTO','Director'],keywords:['Rust']}).map(j=>j.title),['Director']);assert.deepEqual(migrateSelection({titles:[],keywords:['Rust'],sources:['fixture']},1),{titles:null,keywords:['Rust'],sources:['fixture'],assessment:[]});assert.deepEqual(migrateSelection({titles:[]},2).titles,[]);store.close();
});
test('management is bounded title matching and sorting uses first collection date with unassessed last',()=>{
 for(const title of ['CTO','Head of Engineering','Engineering Director','Product Manager','Technical Lead','Lead Engineer','Chief Technology Officer'])assert.ok(managementTitle(title),title);
 for(const title of ['Lead generation specialist','Lead gen managerless','Shareholder','Manchester developer','Dealer','Directorate analyst'])assert.equal(managementTitle(title),false,title);
 const jobs=[{id:1,firstSeen:'2026-10-20',matchScore:0},{id:2,firstSeen:'2026-10-21',matchScore:70},{id:3,firstSeen:'2026-10-22',matchScore:90}] as any;
 assert.deepEqual(sortCatalogue(jobs,'fit').map(j=>j.id),[3,2,1]);assert.deepEqual(sortCatalogue(jobs,'newest').map(j=>j.id),[3,2,1]);jobs[0].firstSeen='2026-10-23';assert.deepEqual(sortCatalogue(jobs,'newest').map(j=>j.id),[1,3,2]);assert.deepEqual(sortCatalogue(jobs,'fit').map(j=>j.id),[3,2,1]);
});
