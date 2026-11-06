import React,{useEffect,useState} from 'react';
import {api,listStopWords,addStopWord as apiAddStopWord,removeStopWord as apiRemoveStopWord,previewStopWord,getSearchExpressions,putSearchExpressions,getSearchSites,addSearchSite,removeSearchSite,toggleSearchSite} from './api';
import type {StopWord} from '../src/core';
import type {SearchExpression,SiteConfig} from './api';
import {Sources} from './Sources';

type Props={onRulesChanged:()=>void;onError:(s:string)=>void};

export function Collection({onRulesChanged,onError}:Props){
  const [section,setSection]=useState<'sources'|'search'>('sources');
  const [health,setHealth]=useState<any[]>([]);
  const [run,setRun]=useState<any>(null);
  const [busy,setBusy]=useState('');
  const [error,setError]=useState('');
  const [notice,setNotice]=useState('');

  // OR/AND search terms — live state, no draft
  const [orTerms,setOrTerms]=useState<string[]>([]);
  const [andTerms,setAndTerms]=useState<string[]>([]);
  const [orInput,setOrInput]=useState('');
  const [andInput,setAndInput]=useState('');

  // Exclusion stop-words
  const [stopWords,setStopWords]=useState<StopWord[]>([]);
  const [exclInput,setExclInput]=useState('');
  const [exclPreview,setExclPreview]=useState<{word:string;count:number}|null>(null);

  // Source sites
  const [sites,setSites]=useState<SiteConfig[]>([]);
  const [siteName,setSiteName]=useState('');
  const [siteUrl,setSiteUrl]=useState('');

  // Poll health/run
  useEffect(()=>{let alive=true;const poll=async()=>{try{const [h,r]=await Promise.all([api('health'),api('collection')]);if(alive){setHealth(h);setRun(r);}}catch(e:any){if(alive)setError(e.message);}};void poll();const timer=setInterval(poll,2000);return()=>{alive=false;clearInterval(timer);};},[]);

  // Load search section data
  useEffect(()=>{
    if(section!=='search')return;
    let alive=true;
    async function load(){
      try{
        const [exprs,words]=await Promise.all([getSearchExpressions(),listStopWords()]);
        if(!alive)return;
        // Flatten any number of groups into one OR column and one AND column
        const or:string[]=[],and:string[]=[];
        for(const e of exprs){if(e.and?.length)and.push(...e.and);else if(e.or?.length)or.push(...e.or);}
        setOrTerms(or);setAndTerms(and);setStopWords(words);
      }catch(e:any){if(alive)setError(e.message);}
    }
    void load();
    return()=>{alive=false;};
  },[section]);

  // Load sites when in sources section
  useEffect(()=>{if(section!=='sources')return;getSearchSites().then(setSites).catch(e=>setError(e.message));},[section]);

  // Exclusion preview debounce
  useEffect(()=>{const word=exclInput.trim();if(!word){setExclPreview(null);return;}let alive=true;const t=setTimeout(()=>{previewStopWord(word).then(p=>{if(alive)setExclPreview({...p,word});}).catch(e=>{if(alive)setError(e.message);});},250);return()=>{alive=false;clearTimeout(t);};},[exclInput]);

  async function collect(source?:string){setBusy('collecting');setError('');setNotice('');try{await api('search','POST',{source,force:true});setNotice('Collection started.');setRun(await api('collection'));}catch(e:any){setError(e.message);}finally{setBusy('');}}

  // Persist current OR and AND terms immediately
  async function persistExprs(or:string[],and:string[]){
    const exprs:SearchExpression[]=[];
    if(or.length)exprs.push({or});
    if(and.length)exprs.push({and});
    await putSearchExpressions(exprs);
    onRulesChanged();
  }

  async function addOrTerm(){
    const term=orInput.trim();if(!term)return;
    setOrInput('');
    try{const next=[...orTerms,term];await persistExprs(next,andTerms);setOrTerms(next);}
    catch(e:any){setError(e.message);}
  }
  async function removeOrTerm(i:number){
    try{const next=orTerms.filter((_,j)=>j!==i);await persistExprs(next,andTerms);setOrTerms(next);}
    catch(e:any){setError(e.message);}
  }
  async function addAndTerm(){
    const term=andInput.trim();if(!term)return;
    setAndInput('');
    try{const next=[...andTerms,term];await persistExprs(orTerms,next);setAndTerms(next);}
    catch(e:any){setError(e.message);}
  }
  async function removeAndTerm(i:number){
    try{const next=andTerms.filter((_,j)=>j!==i);await persistExprs(orTerms,next);setAndTerms(next);}
    catch(e:any){setError(e.message);}
  }
  async function addExcl(){
    const word=exclInput.trim();if(!word)return;
    setExclInput('');setExclPreview(null);
    try{await apiAddStopWord(word);setStopWords(await listStopWords());onRulesChanged();}
    catch(e:any){setError(e.message);}
  }
  async function removeExcl(id:number){
    try{await apiRemoveStopWord(id);setStopWords(await listStopWords());onRulesChanged();}
    catch(e:any){setError(e.message);}
  }

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
      <h1>Search expressions</h1>
      <p className="muted">OR terms: a role matches any. AND terms: role must contain all of them. Exclusions: hides roles whose title or company matches. All three columns combine with AND — a role must satisfy every non-empty column.</p>
      <div className="search-columns">
        <div className="search-col">
          <h3>OR terms</h3>
          <div className="col-input">
            <input value={orInput} onChange={e=>setOrInput(e.target.value)} placeholder="e.g. CTO" onKeyDown={e=>{if(e.key==='Enter'){e.preventDefault();void addOrTerm();}}}/>
            <button disabled={!!busy||!orInput.trim()} onClick={()=>void addOrTerm()}>+</button>
          </div>
          <div className="term-chips">{orTerms.map((t,i)=><span key={i} className="term-chip">{t}<button aria-label={`Remove ${t}`} disabled={!!busy} onClick={()=>void removeOrTerm(i)}>×</button></span>)}</div>
          {orTerms.length===0&&<p className="muted" style={{fontSize:'12px',margin:0}}>No terms yet.</p>}
        </div>
        <div className="search-col">
          <h3>AND terms</h3>
          <div className="col-input">
            <input value={andInput} onChange={e=>setAndInput(e.target.value)} placeholder="e.g. Web3" onKeyDown={e=>{if(e.key==='Enter'){e.preventDefault();void addAndTerm();}}}/>
            <button disabled={!!busy||!andInput.trim()} onClick={()=>void addAndTerm()}>+</button>
          </div>
          <div className="term-chips">{andTerms.map((t,i)=><span key={i} className="term-chip">{t}<button aria-label={`Remove ${t}`} disabled={!!busy} onClick={()=>void removeAndTerm(i)}>×</button></span>)}</div>
          {andTerms.length===0&&<p className="muted" style={{fontSize:'12px',margin:0}}>No terms yet.</p>}
        </div>
        <div className="search-col">
          <h3>Exclusions</h3>
          <div className="col-input">
            <input value={exclInput} onChange={e=>setExclInput(e.target.value)} placeholder="e.g. risk engineer" onKeyDown={e=>{if(e.key==='Enter'){e.preventDefault();void addExcl();}}}/>
            <button disabled={!!busy||!exclInput.trim()} onClick={()=>void addExcl()}>+</button>
          </div>
          {exclPreview&&exclPreview.word===exclInput.trim()&&<p className="muted" style={{fontSize:'12px',margin:'0 0 8px'}}>Would hide {exclPreview.count} {exclPreview.count===1?'role':'roles'}.</p>}
          <div className="term-chips">{stopWords.map(w=><span key={w.id} className="term-chip">{w.word}<button aria-label={`Remove ${w.word}`} disabled={!!busy} onClick={()=>void removeExcl(w.id)}>×</button></span>)}</div>
          {stopWords.length===0&&<p className="muted" style={{fontSize:'12px',margin:0}}>No active exclusions.</p>}
        </div>
      </div>
    </div>}
  </div>;
}
