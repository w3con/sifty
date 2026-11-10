import React from 'react';
import type {Job} from './model/core';
import {type CatalogueSelection,assessmentStatus} from './filters';
export function Filters({jobs,options,selection,setSelection,rows,shown,reset}:{jobs:Job[];options:any;selection:CatalogueSelection;setSelection:(s:CatalogueSelection)=>void;rows:any[];shown:number;reset:()=>void}){
 const toggle=(key:keyof CatalogueSelection,value:string,values:string[])=>{
  const current=selection[key]??values,next=current.includes(value)?current.filter(x=>x!==value):[...current,value];
  setSelection({...selection,[key]:['titles','keywords'].includes(key)&&values.every(v=>next.includes(v))?null:next});
 };
 const groups:{key:keyof CatalogueSelection;label:string;values:string[];tags?:boolean}[]=[
  {key:'sources',label:'Sources',values:options?.sites.map((s:any)=>s.name)??[]},
  {key:'assessment',label:'AI assessment',values:['assessed','stale','pending','running','failed','missing-description']}
 ];
 const renderGroup=(g:typeof groups[number])=>{const values=[...new Set([...g.values,...(selection[g.key]??[])])];return <fieldset key={g.key}><legend>{g.label}</legend><div className={g.tags?'filter-tags':''}>{values.map(v=>g.tags?<button key={v} aria-pressed={selection[g.key]===null||selection[g.key]!.includes(v)} onClick={()=>toggle(g.key,v,values)}>{v}</button>:<label className="filter-toggle" key={v}><input type="checkbox" checked={selection[g.key]!.includes(v)} onChange={()=>toggle(g.key,v,values)}/>{v} <small>({g.key==='sources'?jobs.filter(j=>j.sources.some(s=>s.name===v)).length:jobs.filter(j=>assessmentStatus(j,rows)===v).length})</small></label>)}</div></fieldset>;};
 // Filters slice what is already collected. Search terms only decide what Collection brings in,
 // so they are not offered here.
 return <div className="catalogue-filters">
  {groups.map(renderGroup)}
  <div className="filter-summary"><span role="status">Showing {shown} / {jobs.length} roles</span><button onClick={reset}>Reset filters</button></div>
 </div>;
}
