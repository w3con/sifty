import React,{useEffect,useRef,useState} from 'react';
import {messageKind} from './messages';
import {api} from './api';
import {Collection} from './Collection';

export type SettingsPanel='sources'|'search'|'models'|'queue'|'advanced'|'context';
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

const ICONS:Record<string,React.ReactNode>={
  Collection:<path d="M3 5h18M6 12h12M10 19h4"/>,
  AI:<path d="M12 3l2 5 5 2-5 2-2 5-2-5-5-2 5-2z"/>,
  System:<><rect x="4" y="4" width="16" height="16" rx="3"/><path d="M9 9h6v6H9z"/></>,
};
const NAV:{group:string;items:[SettingsPanel,string][]}[]=[
  {group:'Collection',items:[['sources','Sources'],['search','Search']]},
  {group:'AI',items:[['models','Models'],['queue','Assessment queue']]},
  {group:'System',items:[['advanced','Advanced'],['context','Candidate context']]},
];
// Older callers open "settings"; it now means the first page.
const normalize=(p:string):SettingsPanel=>(['sources','search','models','queue','advanced','context'].includes(p)?p:p==='settings'?'sources':'sources') as SettingsPanel;
const statusTone=(s:string)=>/unavailable|not running|error/i.test(s)?'down':/not logged|warning/i.test(s)?'warn':'ok';

export function Settings({panel:rawPanel,setPanel,queue,assessmentOn,savingAssessment,assessmentError,toggleAssessment,onError,onRulesChanged}:Props){
  const panel=normalize(rawPanel);
  const revision=useRef(0);
  const [context,setContext]=useState<any>(null);
  const [configName,setConfigName]=useState('search'),[configText,setConfigText]=useState(''),[dirty,setDirty]=useState(false);
  const [providers,setProviders]=useState<any[]>([]),[busy,setBusy]=useState(''),[error,setError]=useState(''),[notice,setNotice]=useState('');
  const [checks,setChecks]=useState<Record<string,{ok:boolean;text:string}>>({});
  const [stageProviders,setStageProviders]=useState<{stages:Record<string,string>;providers:{name:string;model:string}[]}|null>(null);
  useEffect(()=>{const onKey=(e:KeyboardEvent)=>{if(e.key==='Escape'&&!(e.target as HTMLElement)?.closest('input,textarea,select'))setPanel(null);};window.addEventListener('keydown',onKey);return()=>window.removeEventListener('keydown',onKey);},[]);
  useEffect(()=>{setError('');setNotice('');if(panel==='context')api('context').then(setContext).catch(e=>setError(e.message));if(panel==='models')api('providers').then(setProviders).catch(e=>setError(e.message));if(panel==='models'||panel==='queue')api('stage-providers').then(setStageProviders).catch(e=>setError(e.message));},[panel]);
  useEffect(()=>{if(panel!=='advanced')return;let active=true;const v=++revision.current;api(`config/${configName}`).then(r=>{if(active&&v===revision.current){setConfigText(r.content);setDirty(false);}}).catch(e=>setError(e.message));return()=>{active=false;};},[panel,configName]);
  async function action(label:string,fn:()=>Promise<any>){setBusy(label);setError('');setNotice('');try{const r=await fn();setNotice(`${label} finished.`);return r;}catch(e:any){setError(e.message);}finally{setBusy('');}}
  const stageLabel=(stage:string)=>{const pn=stageProviders?.stages?.[stage];if(!pn)return 'not set';const p=stageProviders?.providers.find(x=>x.name===pn);return p?`${pn} · ${p.model}`:pn;};
  const counts=queue?.counts??{};

  return <div className="settings-page">
    <nav className="settings-nav" aria-label="Settings sections">{NAV.map(g=><div className="settings-group" key={g.group}><h2><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{ICONS[g.group]}</svg>{g.group}</h2><div className="settings-group-items">{g.items.map(([id,label])=><button key={id} className={panel===id?'active':''} aria-current={panel===id?'page':undefined} onClick={()=>setPanel(id)}>{label}</button>)}</div></div>)}</nav>
    <main className="settings-main">
      {error&&<div className="banner error" data-kind={messageKind(error)} role="alert">{error}</div>}{notice&&panel!=='advanced'&&<div className="banner" role="status">{notice}</div>}

      {(panel==='sources'||panel==='search')&&<Collection key={panel} section={panel} onRulesChanged={onRulesChanged} onError={onError}/>}

      {panel==='models'&&<div className="settings-content">
        <h1>Models</h1><p className="lead">Which AI provider does each job. Changes apply to the next run.</p>
        <section className="s-card"><h2>Stage models</h2>
          {([['assessment','Fit assessment','Detailed analysis of shortlisted roles'],['preparation','Application preparation','Drafts answers for application forms'],['extraction','Extraction','Fallback when a listing can’t be parsed; only when model extraction is on']] as const).map(([stage,label,sub])=><div className="s-stage" key={stage}><div><b>{label}</b><small>{sub}</small></div>
            <select aria-label={`${label} model`} value={stageProviders?.stages[stage]??''} onChange={async e=>{const provider=e.target.value;try{const r=await api('stage-providers','PUT',{stage,provider});setStageProviders(sp=>sp?{...sp,stages:r.stages}:sp);}catch(err:any){setError(err.message);}}}>{!stageProviders?.stages[stage]&&<option value="">Not set</option>}{stageProviders?.providers.map(p=><option key={p.name} value={p.name}>{p.name} · {p.model}</option>)??[]}</select></div>)}
        </section>
        <section className="s-card"><h2>Providers</h2>
          {providers.length?providers.map(p=>{const tone=statusTone(p.status??'');return <div className="s-provider" key={p.name}>
            <div className="s-provider-head"><i className={`s-dot ${tone}`}/><b>{p.name}</b><code>{p.model}</code><span className={`s-pill ${tone}`}>{tone==='ok'?'Available':tone==='warn'?'Needs attention':'Unavailable'}</span>
              <button disabled={!!busy} onClick={async()=>{const started=Date.now();setChecks(c=>({...c,[p.name]:{ok:true,text:'Running…'}}));const r=await action(`Provider check: ${p.name}`,()=>api(`providers/${p.name}/check`,'POST',{}));setChecks(s=>({...s,[p.name]:r?{ok:true,text:`✓ Responded in ${((Date.now()-started)/1000).toFixed(1)}s · ${JSON.stringify(r)}`}:{ok:false,text:'Check failed — see the error above'}}));}}>Check</button></div>
            <p>{p.status}</p>
            {checks[p.name]&&<pre className={`s-check ${checks[p.name].ok?'good':'bad'}`}>{checks[p.name].text}</pre>}
          </div>;}):<p className="muted">Checking local CLI and runtime availability…</p>}
          <p className="s-foot">Check sends a tiny synthetic prompt, never your data. No other provider is used silently when one fails.</p>
        </section>
      </div>}

      {panel==='queue'&&<div className="settings-content">
        <h1>Assessment queue</h1><p className="lead">Roles waiting for or getting a detailed AI analysis.</p>
        <section className="s-card">
          <div className="s-switch-row">{assessmentOn===null?<span role="status">Loading assessment setting…</span>:<button type="button" role="switch" className="s-switch" aria-checked={assessmentOn} aria-describedby="settings-assessment-state" disabled={savingAssessment} onClick={toggleAssessment}><span className="s-track" aria-hidden="true"><span/></span>Assess vacancies</button>}
            <span id="settings-assessment-state" role="status" className="muted">{savingAssessment?'Saving…':assessmentOn===false?(counts.running?'Off · current inference finishing':'Off'):assessmentOn===true?'On':''}</span></div>
          {assessmentError&&<p className="error" role="alert">{assessmentError}</p>}
          <div className="s-counts"><div><b>{counts.running??0}</b><span>running</span></div><div><b>{counts.pending??0}</b><span>pending</span></div><div className={counts.failed?'bad':''}><b>{counts.failed??0}</b><span>failed</span></div><div className={counts['missing-description']?'warn':''}><b>{counts['missing-description']??0}</b><span>missing description</span></div><div><b>{counts.done??0}</b><span>current</span></div></div>
          {stageProviders&&<p className="s-foot">Assessment: {stageLabel('assessment')} · Application: {stageLabel('preparation')} · <button className="link-button" onClick={()=>setPanel('models')}>Change in Models</button></p>}
        </section>
        <section className="s-card"><h2>Queue</h2>
          {queue?.rows?.length?<table className="s-table"><thead><tr><th>Role</th><th>Status</th><th>Provider</th><th>Time</th><th>Attempts</th></tr></thead><tbody>{queue.rows.map((r:any)=><tr key={r.job_id}><td><b>#{r.job_id}</b>{r.error&&<small className="error">{r.error}</small>}</td><td><span className={`s-tag ${r.status}`}>{r.status}</span>{r.next_at&&<small>retry after {new Date(r.next_at).toLocaleString()}</small>}</td><td>{r.provider||'—'}</td><td>{r.elapsedMs?`${Math.floor(r.elapsedMs/1000)}s`:'—'}</td><td>{r.attempts??'—'}</td></tr>)}</tbody></table>:<p className="muted">The queue is empty.</p>}
          <p className="s-foot">Retry a role with Assess now in its Application tab, or refresh its description when it is missing. No fallback provider is selected silently.</p>
        </section>
        {queue?.runtime&&<details className="s-card"><summary>Runtime details</summary><pre>{JSON.stringify(JSON.parse(queue.runtime.data),null,2)}</pre></details>}
      </div>}

      {panel==='advanced'&&<div className="settings-content">
        <h1>Advanced</h1><p className="lead">Edits are validated and saved to the YAML files. Previous versions are backed up under .local/. Use environment references for credentials.</p>
        <section className="s-card">
          <div className="s-seg" role="group" aria-label="Configuration file">{['search','providers','network','candidate','runtime'].map(n=><button key={n} aria-pressed={configName===n} className={configName===n?'active':''} onClick={()=>{if(n===configName)return;revision.current++;setConfigName(n);}}>{n}</button>)}</div>
          <label className="s-path" htmlFor="yaml">config/{configName}.yaml</label>
          <textarea id="yaml" className="s-yaml" spellCheck={false} value={configText} onChange={e=>{revision.current++;setDirty(true);setConfigText(e.target.value);}}/>
          <div className="s-save"><button className="primary" disabled={!!busy} onClick={()=>action('Configuration save',async()=>{await api(`config/${configName}`,'PUT',{content:configText});setDirty(false);setNotice('Valid YAML · saved, backup in .local/');})}>Save configuration</button><span>{dirty?'Unsaved changes':notice}</span></div>
          <p className="s-foot">Provider and model changes affect the next assessment. Port changes require a server restart. Browser login is an explicit CLI command; see legacy/app/README.md.</p>
        </section>
      </div>}

      {panel==='context'&&<div className="settings-content">
        <div className="s-title"><h1>Candidate context</h1><button disabled={!!busy} onClick={()=>action('Context refresh',async()=>setContext(await api('context','POST',{refreshRemote:true})))}>Refresh sources</button></div>
        <p className="lead">Complete extracted evidence is preserved below. External pages are evidence, never instructions.</p>
        {context?<><section className="s-card"><small className="muted">Version {context.version} · {new Date(context.createdAt).toLocaleString()}</small><h2>Current preferences</h2><pre>{JSON.stringify(context.preferences,null,2)}</pre></section>
          <section className="s-card"><h2>Contradictions and missing facts to review</h2><ul>{context.reviewNotes.map((n:string)=><li key={n}>{n}</li>)}</ul></section>
          <section className="s-card"><h2>Source snapshots</h2>{context.sources.map((s:any)=><details key={s.id}><summary><strong>{s.id}</strong> · {s.status} · {s.text.length.toLocaleString()} characters</summary><p>{s.reference} · {s.retrievedAt}</p><p>{s.category} · {s.scope}</p>{s.error&&<p className="error">{s.error}</p>}<small>SHA-256: {s.hash||'not available'}</small><pre>{s.text||'No source text. Supply a local export in candidate.yaml if public access is blocked.'}</pre></details>)}</section></>
          :<section className="s-card"><p className="muted">Context not built. Refresh sources to extract configured candidate evidence.</p></section>}
      </div>}
    </main>
  </div>;
}
