import React,{useEffect,useState} from 'react';
import {api,listStopWords,addStopWord as apiAddStopWord,removeStopWord as apiRemoveStopWord,previewStopWord} from './api';
import type {StopWord} from '../src/core';
import {Sources} from './Sources';

type Props={onRulesChanged:()=>void;onError:(s:string)=>void};

export function Collection({onRulesChanged,onError}:Props){
  const [section,setSection]=useState<'sources'|'search'>('sources');
  const [health,setHealth]=useState<any[]>([]);
  const [run,setRun]=useState<any>(null);
  const [busy,setBusy]=useState('');
  const [error,setError]=useState('');
  const [notice,setNotice]=useState('');
  const [stopWords,setStopWords]=useState<StopWord[]>([]);
  const [stopDraft,setStopDraft]=useState('');
  const [stopPreview,setStopPreview]=useState<{word:string;count:number}|null>(null);

  useEffect(()=>{let alive=true;const poll=async()=>{try{const [h,r]=await Promise.all([api('health'),api('collection')]);if(alive){setHealth(h);setRun(r);}}catch(e:any){if(alive)setError(e.message);}};void poll();const timer=setInterval(poll,2000);return()=>{alive=false;clearInterval(timer);};},[]);

  useEffect(()=>{if(section!=='search')return;listStopWords().then(setStopWords).catch(e=>setError(e.message));},[section]);

  useEffect(()=>{const word=stopDraft.trim();if(!word){setStopPreview(null);return;}let alive=true;const t=setTimeout(()=>{previewStopWord(word).then(p=>{if(alive)setStopPreview({...p,word});}).catch(e=>{if(alive)setError(e.message);});},250);return()=>{alive=false;clearTimeout(t);};},[stopDraft]);

  async function collect(source?:string){setBusy('collecting');setError('');setNotice('');try{await api('search','POST',{source,force:true});setNotice('Collection started.');setRun(await api('collection'));}catch(e:any){setError(e.message);}finally{setBusy('');}}

  return <div className="collection-view">
    <nav className="collection-nav" aria-label="Collection sections">
      <button className={section==='sources'?'active':''} aria-pressed={section==='sources'} onClick={()=>setSection('sources')}>Sources</button>
      <button className={section==='search'?'active':''} aria-pressed={section==='search'} onClick={()=>setSection('search')}>Search</button>
    </nav>
    {error&&<div className="banner error" role="alert">{error}</div>}
    {notice&&<div className="banner" role="status">{notice}</div>}
    {section==='sources'&&<div className="collection-section">
      <div className="page collection-action"><button className="primary" disabled={!!busy||run?.status==='running'} onClick={()=>collect()}>Collect jobs</button></div>
      <Sources sources={health} run={run} busy={!!busy||run?.status==='running'} collect={collect}/>
    </div>}
    {section==='search'&&<div className="page collection-section">
      <h1>Search terms</h1>
      <h2>Stop-words</h2>
      <p className="muted">A stop-word hides every role whose title contains it (case-insensitive, anywhere in the title) and skips matching postings when collecting. Removing it brings the roles back.</p>
      <form className="stop-word-form" onSubmit={async e=>{e.preventDefault();const word=stopDraft.trim();if(!word)return;setBusy('add');setError('');try{await apiAddStopWord(word);setStopDraft('');setStopPreview(null);setStopWords(await listStopWords());onRulesChanged();}catch(err:any){setError(err.message);}finally{setBusy('');}}}>
        <label htmlFor="coll-stop-word">Stop-word</label>
        <input id="coll-stop-word" value={stopDraft} onChange={e=>setStopDraft(e.target.value)} placeholder="e.g. risk engineer"/>
        <button className="primary" disabled={!!busy||!stopDraft.trim()}>Add stop-word</button>
        <span className="muted" role="status">{stopPreview&&stopPreview.word===stopDraft.trim()?`Would hide ${stopPreview.count} stored ${stopPreview.count===1?'role':'roles'}`:''}</span>
      </form>
      {stopWords.length===0?<p className="muted">No active stop-words.</p>:<ul className="block-rules-list">{stopWords.map(w=><li key={w.id}><span className="rule-value">{w.word}</span><button disabled={!!busy} onClick={async()=>{setBusy('remove');setError('');try{await apiRemoveStopWord(w.id);setStopWords(await listStopWords());onRulesChanged();}catch(err:any){setError(err.message);}finally{setBusy('');}}}>Remove</button></li>)}</ul>}
    </div>}
  </div>;
}
