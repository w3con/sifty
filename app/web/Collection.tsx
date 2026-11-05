import React,{useEffect,useState} from 'react';
import {api,listStopWords,addStopWord as apiAddStopWord,removeStopWord as apiRemoveStopWord,previewStopWord,getSearchExpressions,putSearchExpressions,getSearchSites,addSearchSite,removeSearchSite,toggleSearchSite} from './api';
import type {StopWord} from '../src/core';
import type {SearchExpression,SiteConfig} from './api';
import {Sources} from './Sources';

type ExprGroup={type:'or'|'and';terms:string[]};

function exprsToGroups(exprs:SearchExpression[]):ExprGroup[]{
  return exprs.map(e=>e.and?.length?{type:'and',terms:e.and}:{type:'or',terms:e.or??[]});
}
function groupsToExprs(groups:ExprGroup[]):SearchExpression[]{
  return groups.map(g=>g.type==='and'?{and:g.terms}:{or:g.terms});
}

type Props={onRulesChanged:()=>void;onError:(s:string)=>void};

export function Collection({onRulesChanged,onError}:Props){
  const [section,setSection]=useState<'sources'|'search'>('sources');
  const [health,setHealth]=useState<any[]>([]);
  const [run,setRun]=useState<any>(null);
  const [busy,setBusy]=useState('');
  const [error,setError]=useState('');
  const [notice,setNotice]=useState('');

  // Search expressions
  const [savedExprs,setSavedExprs]=useState<ExprGroup[]>([]);
  const [draftExprs,setDraftExprs]=useState<ExprGroup[]>([]);
  const [termInputs,setTermInputs]=useState<string[]>([]);

  // Exclusion words (stop-words)
  const [stopWords,setStopWords]=useState<StopWord[]>([]);
  const [stopDraft,setStopDraft]=useState('');
  const [stopPreview,setStopPreview]=useState<{word:string;count:number}|null>(null);

  // Source sites
  const [sites,setSites]=useState<SiteConfig[]>([]);
  const [siteName,setSiteName]=useState('');
  const [siteUrl,setSiteUrl]=useState('');

  const exprsDirty=JSON.stringify(draftExprs)!==JSON.stringify(savedExprs);

  // Poll health/run
  useEffect(()=>{let alive=true;const poll=async()=>{try{const [h,r]=await Promise.all([api('health'),api('collection')]);if(alive){setHealth(h);setRun(r);}}catch(e:any){if(alive)setError(e.message);}};void poll();const timer=setInterval(poll,2000);return()=>{alive=false;clearInterval(timer);};},[]);

  // Load search section data
  useEffect(()=>{if(section!=='search')return;let alive=true;async function load(){try{const [exprs,words]=await Promise.all([getSearchExpressions(),listStopWords()]);if(!alive)return;const groups=exprsToGroups(exprs);setSavedExprs(groups);setDraftExprs(groups);setTermInputs(groups.map(()=>''));setStopWords(words);}catch(e:any){if(alive)setError(e.message);}}void load();return()=>{alive=false;};},[section]);

  // Load sites when in sources section
  useEffect(()=>{if(section!=='sources')return;getSearchSites().then(setSites).catch(e=>setError(e.message));},[section]);

  // Stop-word preview debounce
  useEffect(()=>{const word=stopDraft.trim();if(!word){setStopPreview(null);return;}let alive=true;const t=setTimeout(()=>{previewStopWord(word).then(p=>{if(alive)setStopPreview({...p,word});}).catch(e=>{if(alive)setError(e.message);});},250);return()=>{alive=false;clearTimeout(t);};},[stopDraft]);

  async function collect(source?:string){setBusy('collecting');setError('');setNotice('');try{await api('search','POST',{source,force:true});setNotice('Collection started.');setRun(await api('collection'));}catch(e:any){setError(e.message);}finally{setBusy('');}}

  // Expression helpers
  function updateGroup(idx:number,patch:Partial<ExprGroup>){setDraftExprs(prev=>prev.map((g,i)=>i===idx?{...g,...patch}:g));}
  function removeGroup(idx:number){setDraftExprs(prev=>prev.filter((_,i)=>i!==idx));setTermInputs(prev=>prev.filter((_,i)=>i!==idx));}
  function addTerm(idx:number){const term=(termInputs[idx]??'').trim();if(!term)return;updateGroup(idx,{terms:[...(draftExprs[idx]?.terms??[]),term]});setTermInputs(prev=>prev.map((v,i)=>i===idx?'':v));}
  function removeTerm(gi:number,ti:number){setDraftExprs(prev=>prev.map((g,i)=>i===gi?{...g,terms:g.terms.filter((_,j)=>j!==ti)}:g));}
  function addGroup(type:'or'|'and'){setDraftExprs(prev=>[...prev,{type,terms:[]}]);setTermInputs(prev=>[...prev,'']);}

  async function saveExpressions(){setBusy('save-exprs');setError('');try{const exprs=groupsToExprs(draftExprs.filter(g=>g.terms.length>0));const result=await putSearchExpressions(exprs);const groups=exprsToGroups(result.expressions??exprs);setSavedExprs(groups);setDraftExprs(groups);setTermInputs(groups.map(()=>''));setNotice('Search expressions saved.');onRulesChanged();}catch(e:any){setError(e.message);}finally{setBusy('');}}

  return <div className="collection-view">
    <nav className="collection-nav" aria-label="Collection sections">
      <button className={section==='sources'?'active':''} aria-pressed={section==='sources'} onClick={()=>setSection('sources')}>Sources</button>
      <button className={section==='search'?'active':''} aria-pressed={section==='search'} onClick={()=>setSection('search')}>Search</button>
    </nav>
    {error&&<div className="banner error" role="alert">{error}</div>}
    {notice&&<div className="banner" role="status">{notice}</div>}

    {section==='sources'&&<div className="collection-section">
      <div className="page">
        <h1>Source sites</h1>
        <p className="muted">Sites are searched during collection. Disable a site to pause it without removing it from the list.</p>
        {sites.length===0
          ?<p className="muted">No sites configured yet.</p>
          :<ul className="block-rules-list">{sites.map(s=><li key={s.name}>
            <span className="rule-value">{s.name}</span>
            {s.url&&<a href={s.url} target="_blank" rel="noreferrer" className="muted" style={{fontSize:'12px',overflow:'hidden',textOverflow:'ellipsis',whiteSpace:'nowrap',maxWidth:'200px'}}>{s.url}</a>}
            <label style={{display:'flex',alignItems:'center',gap:6,cursor:'pointer',fontWeight:500,fontSize:'13px'}}>
              <input type="checkbox" checked={s.enabled!==false} onChange={async e=>{setBusy('toggle');setError('');try{await toggleSearchSite(s.name,e.target.checked);setSites(await getSearchSites());}catch(err:any){setError(err.message);}finally{setBusy('');}}}/>
              Enabled
            </label>
            <button disabled={!!busy} onClick={async()=>{setBusy('remove-site');setError('');try{await removeSearchSite(s.name);setSites(await getSearchSites());onRulesChanged();}catch(err:any){setError(err.message);}finally{setBusy('');}}} style={{marginLeft:'auto'}}>Remove</button>
          </li>)}</ul>}
        <form className="stop-word-form" style={{marginTop:16}} onSubmit={async e=>{e.preventDefault();const name=siteName.trim(),url=siteUrl.trim();if(!name)return;setBusy('add-site');setError('');try{await addSearchSite(name,url);setSites(await getSearchSites());setSiteName('');setSiteUrl('');onRulesChanged();}catch(err:any){setError(err.message);}finally{setBusy('');}}} >
          <label htmlFor="site-name">Site name</label>
          <input id="site-name" value={siteName} onChange={e=>setSiteName(e.target.value)} placeholder="e.g. LinkedIn"/>
          <label htmlFor="site-url">URL (optional)</label>
          <input id="site-url" value={siteUrl} onChange={e=>setSiteUrl(e.target.value)} placeholder="https://…"/>
          <button className="primary" disabled={!!busy||!siteName.trim()}>Add site</button>
        </form>
      </div>
      <div className="page collection-action" style={{paddingTop:0}}>
        <button className="primary" disabled={!!busy||run?.status==='running'} onClick={()=>collect()}>Collect jobs</button>
      </div>
      <Sources sources={health} run={run} busy={!!busy||run?.status==='running'} collect={collect}/>
    </div>}

    {section==='search'&&<div className="page collection-section">
      <div className="section-title">
        <h1>Search expressions</h1>
        {exprsDirty&&<span className="muted" role="status">Unsaved changes</span>}
      </div>
      <p className="muted">Each group matches roles where any OR term appears, or where all AND terms appear. Groups are combined by OR: a role matches when any group matches.</p>
      {draftExprs.map((g,gi)=><div key={gi} style={{background:'#1e1e21',borderRadius:10,padding:16,margin:'10px 0'}}>
        <div className="section-title" style={{marginBottom:10}}>
          <select value={g.type} disabled={!!busy} onChange={e=>updateGroup(gi,{type:e.target.value as 'or'|'and'})} style={{fontWeight:600}}>
            <option value="or">OR — any term matches</option>
            <option value="and">AND — all terms must appear</option>
          </select>
          <button disabled={!!busy} onClick={()=>removeGroup(gi)}>Remove group</button>
        </div>
        {g.terms.length===0&&<p className="muted" style={{margin:'0 0 8px'}}>No terms yet — add one below.</p>}
        {g.terms.length>0&&<ul className="block-rules-list" style={{marginBottom:8}}>{g.terms.map((t,ti)=><li key={ti}><span className="rule-value">{t}</span><button disabled={!!busy} onClick={()=>removeTerm(gi,ti)}>Remove</button></li>)}</ul>}
        <div style={{display:'flex',gap:8}}>
          <input value={termInputs[gi]??''} onChange={e=>setTermInputs(prev=>prev.map((v,i)=>i===gi?e.target.value:v))} placeholder="e.g. CTO" onKeyDown={e=>{if(e.key==='Enter'){e.preventDefault();addTerm(gi);}}}/>
          <button disabled={!!busy||!(termInputs[gi]??'').trim()} onClick={()=>addTerm(gi)}>Add term</button>
        </div>
      </div>)}
      <div style={{display:'flex',gap:8,marginTop:16,flexWrap:'wrap'}}>
        <button onClick={()=>addGroup('or')}>+ OR group</button>
        <button onClick={()=>addGroup('and')}>+ AND group</button>
        <button className="primary" disabled={!!busy||!exprsDirty} onClick={saveExpressions}>Save</button>
      </div>

      <h2 style={{marginTop:32}}>Exclusion words</h2>
      <p className="muted">An exclusion word hides every role whose title or company name contains it (case-insensitive). Removing it brings the roles back.</p>
      <form className="stop-word-form" onSubmit={async e=>{e.preventDefault();const word=stopDraft.trim();if(!word)return;setBusy('add');setError('');try{await apiAddStopWord(word);setStopDraft('');setStopPreview(null);setStopWords(await listStopWords());onRulesChanged();}catch(err:any){setError(err.message);}finally{setBusy('');}}}>
        <label htmlFor="coll-excl-word">Word or phrase</label>
        <input id="coll-excl-word" value={stopDraft} onChange={e=>setStopDraft(e.target.value)} placeholder="e.g. risk engineer"/>
        <button className="primary" disabled={!!busy||!stopDraft.trim()}>Add exclusion</button>
        <span className="muted" role="status">{stopPreview&&stopPreview.word===stopDraft.trim()?`Would hide ${stopPreview.count} stored ${stopPreview.count===1?'role':'roles'}`:''}</span>
      </form>
      {stopWords.length===0?<p className="muted">No active exclusion words.</p>:<ul className="block-rules-list">{stopWords.map(w=><li key={w.id}><span className="rule-value">{w.word}</span><button disabled={!!busy} onClick={async()=>{setBusy('remove');setError('');try{await apiRemoveStopWord(w.id);setStopWords(await listStopWords());onRulesChanged();}catch(err:any){setError(err.message);}finally{setBusy('');}}}>Remove</button></li>)}</ul>}
    </div>}
  </div>;
}
