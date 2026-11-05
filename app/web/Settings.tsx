import React,{useEffect,useRef,useState} from 'react';
import {api} from './api';

type Props={
  panel:string;
  setPanel:(s:string|null)=>void;
  queue:any;
  assessmentOn:boolean|null;
  savingAssessment:boolean;
  assessmentError:string;
  toggleAssessment:()=>void;
  onError:(s:string)=>void;
  onRulesChanged:()=>void;
};

export function Settings({panel,setPanel,queue,assessmentOn,savingAssessment,assessmentError,toggleAssessment,onError,onRulesChanged}:Props){
  const dialog=useRef<HTMLDialogElement>(null),revision=useRef(0);
  const [context,setContext]=useState<any>(null);
  const [configName,setConfigName]=useState('search'),[configText,setConfigText]=useState(''),[dirty,setDirty]=useState(false);
  const [providers,setProviders]=useState<any[]>([]),[busy,setBusy]=useState(''),[error,setError]=useState(''),[notice,setNotice]=useState('');
  const [checks,setChecks]=useState<Record<string,string>>({});
  const [stageProviders,setStageProviders]=useState<{stages:Record<string,string>;providers:{name:string;model:string}[]}|null>(null);
  useEffect(()=>{const previous=document.activeElement as HTMLElement|null,element=dialog.current;element?.showModal();return()=>{element?.close();if(previous?.isConnected)previous.focus();};},[]);
  useEffect(()=>{if(panel==='context')api('context').then(setContext).catch(e=>setError(e.message));if(panel==='settings'){api('providers').then(setProviders).catch(e=>setError(e.message));}if(panel==='settings'||panel==='queue')api('stage-providers').then(setStageProviders).catch(e=>setError(e.message));},[panel]);
  useEffect(()=>{if(panel!=='settings')return;let active=true;const v=++revision.current;api(`config/${configName}`).then(r=>{if(active&&v===revision.current){setConfigText(r.content);setDirty(false);}}).catch(e=>setError(e.message));return()=>{active=false;};},[panel,configName]);
  async function action(label:string,fn:()=>Promise<any>){setBusy(label);setError('');setNotice('');try{const r=await fn();setNotice(`${label} finished.`);return r;}catch(e:any){setError(e.message);}finally{setBusy('');}}

  const stageLabel=(stage:string)=>{const pn=stageProviders?.stages?.[stage];if(!pn)return '— not set —';const p=stageProviders?.providers.find(x=>x.name===pn);return p?`${pn} · ${p.model}`:pn;};

  return <dialog ref={dialog} className="settings-drawer" aria-label="Settings" onCancel={e=>{e.preventDefault();setPanel(null);}} onClick={e=>{if(e.target===e.currentTarget){const r=e.currentTarget.getBoundingClientRect();if(e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom)setPanel(null);}}}>
    <nav aria-label="Settings sections">{[['context','Context'],['settings','Configuration'],['queue','Queue']].map(([id,label])=><button key={id} className={panel===id?'active':''} aria-pressed={panel===id} onClick={()=>setPanel(id)}>{label}</button>)}<button className="close-settings" onClick={()=>setPanel(null)}>Close settings <kbd>Esc</kbd></button></nav>
    {error&&<div className="banner error" role="alert">{error}</div>}{notice&&<div className="banner" role="status">{notice}</div>}
    {panel==='context'&&<div className="page"><div className="section-title"><h1>Candidate context</h1><button disabled={!!busy} onClick={()=>action('Context refresh',async()=>setContext(await api('context','POST',{refreshRemote:true})))}>Refresh sources</button></div><p>Complete extracted evidence is preserved below. External pages are evidence, never instructions.</p>{context?<><small>Version {context.version} · {new Date(context.createdAt).toLocaleString()}</small><h2>Current preferences</h2><pre>{JSON.stringify(context.preferences,null,2)}</pre><h2>Contradictions &amp; missing facts to review</h2><ul>{context.reviewNotes.map((n:string)=><li key={n}>{n}</li>)}</ul><h2>Source snapshots</h2>{context.sources.map((s:any)=><details key={s.id}><summary><strong>{s.id}</strong> · {s.status} · {s.text.length.toLocaleString()} characters</summary><p>{s.reference} · {s.retrievedAt}</p><p>{s.category} · {s.scope}</p>{s.error&&<p className="error">{s.error}</p>}<small>SHA-256: {s.hash||'not available'}</small><pre>{s.text||'No source text. Supply a local export in candidate.yaml if public access is blocked.'}</pre></details>)}</>:<p>Context not built. Refresh sources to extract configured candidate evidence.</p>}</div>}
    {panel==='settings'&&<div className="page settings"><h1>Configuration</h1><p>Edits are validated and saved to the existing root-relative YAML files. Previous versions are backed up under ignored .local/. Use environment references for credentials.</p><h2>Provider availability</h2><div className="provider-list">{providers.length?providers.map(p=><section className="provider-card" key={p.name}><div><strong>{p.name}</strong> <code>{p.model}</code><p className={/unavailable/i.test(p.status)?'danger':/not logged/i.test(p.status)?'warning':'muted'}>{p.status}</p></div><button disabled={!!busy} onClick={async()=>{const started=Date.now();setChecks({...checks,[p.name]:'Running…'});const r=await action(`Provider check: ${p.name}`,()=>api(`providers/${p.name}/check`,'POST',{}));setChecks(s=>({...s,[p.name]:r?`Responded in ${((Date.now()-started)/1000).toFixed(1)}s · ${JSON.stringify(r)}`:'Check failed — see error above'}));}}>Minimal smoke check</button>{checks[p.name]&&<pre className="smoke-result">{checks[p.name]}</pre>}</section>):<p>Checking local CLI and runtime availability…</p>}</div><h2>Stage models</h2><div className="stage-models">{([['assessment','Fit assessment'],['preparation','Application preparation'],['extraction','Extraction (fallback when model_extraction is enabled)']] as const).map(([stage,label])=><label key={stage}>{label}<select value={stageProviders?.stages[stage]??''} onChange={async e=>{const provider=e.target.value;try{const r=await api('stage-providers','PUT',{stage,provider});setStageProviders(sp=>sp?{...sp,stages:r.stages}:sp);}catch(err:any){setError(err.message);}}}>{!stageProviders?.stages[stage]&&<option value="">— not set —</option>}{stageProviders?.providers.map(p=><option key={p.name} value={p.name}>{p.name} — {p.model}</option>)??[]}</select></label>)}</div><label htmlFor="config">Configuration file</label><select id="config" value={configName} onChange={e=>{revision.current++;setConfigName(e.target.value);}}>{['search','providers','network','candidate','runtime'].map(n=><option key={n}>{n}</option>)}</select><label className="config-path" htmlFor="yaml">config/{configName}.yaml</label><textarea id="yaml" spellCheck={false} value={configText} onChange={e=>{revision.current++;setDirty(true);setConfigText(e.target.value);}}/><div className="config-save"><button className="primary" disabled={!!busy} onClick={()=>action('Configuration save',async()=>{await api(`config/${configName}`,'PUT',{content:configText});setDirty(false);setNotice('Valid YAML · saved, backup in .local/');})}>Save configuration</button><span>{dirty?'Unsaved changes':notice}</span></div><p className="muted">Provider/model changes affect the next assessment. Port changes require a server restart. Browser login is an explicit CLI command; see app/README.md. Smoke checks send a minimal synthetic prompt, not candidate data.</p></div>}
    {panel==='queue'&&<div className="page">
      <h1>Assessment queue</h1>
      <div className="assessment-control" style={{marginBottom:16}}>
        {assessmentOn===null
          ?<span role="status">Loading assessment setting…</span>
          :<button type="button" role="switch" aria-label="Assess vacancies" aria-checked={assessmentOn} aria-describedby="settings-assessment-state" disabled={savingAssessment} onClick={toggleAssessment}><span className="switch-track" aria-hidden="true"><span/></span>Assess vacancies</button>}
        <span id="settings-assessment-state" role="status" style={{marginLeft:8,fontSize:13,color:'#a3a3aa'}}>{savingAssessment?'Saving…':assessmentOn===false?(queue?.counts.running?'Off · current inference finishing':'Off'):assessmentOn===true?'On':''}</span>
        {assessmentError&&<div className="assessment-switch-error" role="alert">{assessmentError}</div>}
      </div>
      <div className="queue-status" style={{background:'#1e1e21',borderRadius:8,padding:'10px 14px',display:'flex',flexWrap:'wrap',gap:16,marginBottom:16,fontFamily:'var(--mono,monospace)',fontSize:12}}>
        <span className={queue?.counts.running?'running':''}>● {queue?.counts.running??0} running</span>
        <span>{queue?.counts.pending??0} pending</span>
        <span className={queue?.counts.failed?'danger':''}>{queue?.counts.failed??0} failed</span>
        <span className="warning">{queue?.counts['missing-description']??0} missing desc.</span>
        <span>{queue?.counts.done??0} current</span>
        {stageProviders&&<span style={{marginLeft:'auto',display:'flex',gap:8,flexWrap:'wrap'}}><span>Assessment: {stageLabel('assessment')}</span><span>·</span><span>Application: {stageLabel('preparation')}</span></span>}
      </div>
      {queue?.runtime&&<pre>{JSON.stringify(JSON.parse(queue.runtime.data),null,2)}</pre>}
      <p>Check provider availability in Configuration. Retry a selected role using Reassess, or refresh its description when missing. No fallback provider is selected silently.</p>
      <button onClick={()=>setPanel('settings')}>Open Configuration</button>
      {queue?.rows.map((r:any)=><details key={r.job_id}><summary>Role #{r.job_id} · {r.status} · {r.provider}</summary><pre>{JSON.stringify(r,null,2)}</pre></details>)}
    </div>}
  </dialog>;
}
