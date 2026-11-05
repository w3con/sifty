import React,{useEffect,useRef,useState} from 'react';
import {createRoot} from 'react-dom/client';
import type {Job} from '../src/core';
import type {Stage} from '../src/workflow';
import {externalApplyUrl,archiveRetentionMs,validAppliedAt} from '../src/workflow';
import {api,blockJob} from './api';
import {Filters} from './CatalogueFilters';
import {emptySelection,filterCatalogue,migrateSelection,selectionCount,sortCatalogue,type CatalogueSelection} from './filters';
import {emptyQuick,nextUndecided,quickMatch,shortcutGuard,visibleInStage,type QuickFilters,type View} from './workflow';
import {Settings} from './Settings';
import {Collection} from './Collection';
import {RoleContent} from './RoleContent';
import {ApplicationConfirmation} from './ApplicationConfirmation';
import {DeleteRole} from './DeleteRole';
import {Icon} from './Icon';
import '@fontsource/ibm-plex-sans/latin-400.css';
import '@fontsource/ibm-plex-sans/latin-500.css';
import '@fontsource/ibm-plex-sans/latin-600.css';
import '@fontsource/ibm-plex-sans/latin-700.css';
import '@fontsource/ibm-plex-sans/cyrillic-400.css';
import '@fontsource/ibm-plex-mono/latin-400.css';
import '@fontsource/ibm-plex-mono/latin-500.css';
import '@fontsource/unbounded/latin-600.css';
import './style.css';
import './role.css';
import './application.css';
import './polish.css';
import './queue-control.css';
import './application-draft.css';

function preference<T>(key:string,fallback:T):T {try{return JSON.parse(localStorage.getItem(key)??'null')??fallback;}catch{return fallback;}}
export const dateLabel=(date:string|null)=>date?new Date(date).toLocaleDateString(undefined,{month:'short',day:'numeric',year:'numeric'}):'Date not recorded';
function App(){
  const [jobs,setJobs]=useState<Job[]>([]),[selected,setSelected]=useState<number|null>(null),[view,setView]=useState<View>('triage');
  const [session,setSession]=useState(new Set<number>()),[query,setQuery]=useState(()=>localStorage.getItem('fcto.catalogue.query')??'');
  const [selection,setSelection]=useState<CatalogueSelection>(()=>migrateSelection(preference('fcto.catalogue.filters',{}),preference('sift.filters.version',1)));
  const [unread,setUnread]=useState(()=>preference('sift.unread',false)),[sort,setSort]=useState<'fit'|'newest'>(()=>preference('sift.sort','fit')),[heldUnread,setHeldUnread]=useState<number|null>(null);
  const [quick,setQuick]=useState<QuickFilters>(()=>({...emptyQuick(),...preference('sift.quick',{})}));
  const [filtersOpen,setFiltersOpen]=useState(false),[options,setOptions]=useState<any>(null),[queue,setQueue]=useState<any>(null),[stageModels,setStageModels]=useState<any>(null);
  const [settings,setSettings]=useState<string|null>(null),[error,setError]=useState(''),[notice,setNotice]=useState(''),[importUrl,setImportUrl]=useState(''),[busy,setBusy]=useState(false);
  const [confirmation,setConfirmation]=useState<Job|null>(null);
  const [deleting,setDeleting]=useState<Job|null>(null);
  const [assessmentOn,setAssessmentOn]=useState<boolean|null>(null),[savingAssessment,setSavingAssessment]=useState(false),[assessmentError,setAssessmentError]=useState('');
  const assessmentRevision=useRef(0),assessmentSaving=useRef(false);
  const search=useRef<HTMLInputElement>(null),filterBox=useRef<HTMLDivElement>(null),writes=useRef(0),epoch=useRef(0),polling=useRef(false);
  const replaceJob=(j:Job)=>setJobs(old=>old.map(x=>x.id===j.id?j:x));
  // Reject snapshots started before a workflow write. Serialize polling and keep review state separate.
  async function refresh(){const revision=epoch.current;const data=await api('jobs');if(!writes.current&&revision===epoch.current)setJobs(data);}
  useEffect(()=>{api('filter-options').then(setOptions).catch(e=>setError(e.message));},[]);
  // Fire one unforced collection on app mount so opening or reloading the app triggers a
  // throttled run; the server-side cooldown prevents repeated runs within the configured interval.
  useEffect(()=>{api('search','POST',{force:false}).catch(()=>{});},[]);
  useEffect(()=>{let alive=true;async function poll(){if(polling.current)return;polling.current=true;const revision=assessmentRevision.current;try{const [q,sp]=await Promise.all([api('queue'),api('stage-providers').catch(()=>null)]);if(alive){setQueue(q);if(sp)setStageModels(sp);if(!assessmentSaving.current&&revision===assessmentRevision.current&&typeof q.enabled==='boolean')setAssessmentOn(q.enabled);await refresh();}}catch(e:any){if(alive)setError(e.message);}finally{polling.current=false;}}void poll();const timer=setInterval(poll,2500);return()=>{alive=false;clearInterval(timer);};},[]);
  async function toggleAssessment(){
    if(assessmentSaving.current||assessmentOn===null)return;
    assessmentSaving.current=true;assessmentRevision.current++;setSavingAssessment(true);setAssessmentError('');
    try{const q=await api('queue/assessment','PUT',{enabled:!assessmentOn});setAssessmentOn(q.enabled);setQueue(q);}
    catch(e:any){setAssessmentError(`Could not change assessment setting: ${e.message}`);}
    finally{assessmentSaving.current=false;assessmentRevision.current++;setSavingAssessment(false);}
  }
  useEffect(()=>{localStorage.setItem('fcto.catalogue.query',query);localStorage.setItem('fcto.catalogue.filters',JSON.stringify(selection));localStorage.setItem('sift.filters.version','2');localStorage.setItem('sift.quick',JSON.stringify(quick));localStorage.setItem('sift.unread',JSON.stringify(unread));localStorage.setItem('sift.sort',JSON.stringify(sort));},[query,selection,quick,unread,sort]);
  const resetFilters=()=>{setSelection(emptySelection());setQuick(emptyQuick());setQuery('');setUnread(false);setSort('fit');};
  // A stop-word hides a role from every view and every count; the server already omits stopped roles from the list, this keeps a stale client copy hidden too.
  const live=jobs.filter(j=>!j.stopped);
  const filtered=sortCatalogue(filterCatalogue(live,selection,queue?.rows??[]).filter(j=>visibleInStage(j,view,session)&&(!j.blocked||view!=='triage')&&quickMatch(j,quick,query)&&(!unread||!j.seen||j.id===heldUnread)),sort);
  const job=filtered.find(j=>j.id===selected);
  useEffect(()=>{
    if(!filtered.some(j=>j.id===selected))setSelected(filtered[0]?.id??null);
  },[filtered.map(j=>j.id).join(','),selected]);
  useEffect(()=>{
    if(!job||job.seen)return;setHeldUnread(job.id);writes.current++;epoch.current++;
    api(`jobs/${job.id}/seen`,'POST',{}).then((j:Job)=>setJobs(old=>old.map(x=>x.id===j.id?{...x,seen:j.seen,seenAt:j.seenAt}:x))).catch(e=>setError(e.message)).finally(()=>{writes.current--;epoch.current++;});
  },[job?.id,job?.seen]);
  async function transition(stage:Stage,confirmed=false,target=job){
    if(!target||busy)return false;const id=target.id,next=nextUndecided(filtered,id);setBusy(true);setError('');writes.current++;epoch.current++;
    try{const result=await api(`jobs/${id}`,'PATCH',{stage,confirmed});replaceJob(result);
      if(view==='triage'&&['preselected','no'].includes(stage)){setSession(s=>new Set([...s,id]));setSelected(next);}
      else if(stage==='triage')setSession(s=>{const n=new Set(s);n.delete(id);return n;});
      return true;
    }catch(e:any){setError(e.message);return false;}finally{writes.current--;epoch.current++;setBusy(false);}
  }
  function apply(){
    if(!job||view!=='preselected'||job.stage!=='preselected'||busy)return;
    const url=externalApplyUrl(job);if(!url){setError('No valid employer or original listing URL is available. The role remains in Preparation.');return;}
    // Open within the original click/key gesture. No await, form inspection or model call.
    let popup:Window|null=null;
    try{popup=window.open('about:blank','_blank');if(popup){popup.opener=null;popup.location.href=url;}}
    catch{popup?.close();popup=null;}
    if(!popup){setError('The application page was blocked. Allow popups and try Apply again. The role remains in Preparation.');return;}
    setError('');setConfirmation(job);
  }
  async function doBlockCompany(){
    if(!job||busy)return;setBusy(true);setError('');writes.current++;epoch.current++;
    try{await blockJob(job.id,'company');const data=await api('jobs');if(!writes.current)setJobs(data);}
    catch(e:any){setError(e.message);}finally{writes.current--;epoch.current++;setBusy(false);}
  }
  function switchView(next:View){setView(next);setSelected(null);setHeldUnread(null);setError('');setNotice('');}
  function move(delta:number){if(!filtered.length)return;const i=filtered.findIndex(j=>j.id===selected);setHeldUnread(null);setSelected(filtered[(i+delta+filtered.length)%filtered.length].id);}
  useEffect(()=>{function key(e:KeyboardEvent){
    if(settings||document.querySelector('dialog[open]')||e.metaKey||e.ctrlKey||e.altKey||e.isComposing)return;
    if(e.key==='Escape'){setFiltersOpen(false);(document.activeElement as HTMLElement)?.blur();return;}
    if(shortcutGuard(e.target))return;
    const key=e.key.toLowerCase();
    if(['j','k','arrowdown','arrowup','/','y','n','a'].includes(key))e.preventDefault();
    if(key==='j'||key==='arrowdown')move(1);if(key==='k'||key==='arrowup')move(-1);
    if(key==='/')search.current?.focus();
    if(view==='triage'&&job?.stage==='triage'){if(key==='y')void transition('preselected');if(key==='n')void transition('no');}
    if(key==='a')apply();
  }window.addEventListener('keydown',key);return()=>window.removeEventListener('keydown',key);});
  useEffect(()=>{function outside(e:PointerEvent){if(filterBox.current&&!filterBox.current.contains(e.target as Node))setFiltersOpen(false);}document.addEventListener('pointerdown',outside);return()=>document.removeEventListener('pointerdown',outside);},[]);
  const activeCount=Object.values(quick).filter(Boolean).length+selectionCount(selection)+(unread?1:0);
  const runtime=queue?.runtime?JSON.parse(queue.runtime.data):null;
  const stageModel=(s:string)=>{const pn=stageModels?.stages?.[s];if(!pn)return '— not set —';const p=(stageModels?.providers??[]).find((x:any)=>x.name===pn);return p?`${pn} · ${p.model}`:pn;};
  return <div className="sift-app">
    <header className="app-header">
      <div className="brand"><svg aria-hidden="true" width="20" height="20" viewBox="0 0 20 20"><circle cx="5" cy="5" r="3" fill="#58585e"/><circle cx="13" cy="5" r="3" fill="#58585e"/><circle cx="9" cy="14" r="4" fill="#8ab4ff"/></svg><strong>sift</strong></div>
      <nav className="stages" aria-label="Workflow stages"><button aria-pressed={view==='collection'} className={view==='collection'?'active':''} onClick={()=>switchView('collection')}>Collection</button><button aria-pressed={view==='triage'} className={view==='triage'?'active':''} onClick={()=>switchView('triage')}>Triage <span>{live.filter(j=>j.stage==='triage'&&!j.blocked).length}</span></button><button aria-pressed={view==='preselected'} className={view==='preselected'?'active':''} onClick={()=>switchView('preselected')}>Preparation <span>{live.filter(j=>j.stage==='preselected').length}</span></button><button aria-pressed={view==='applied'} className={view==='applied'?'active':''} onClick={()=>switchView('applied')}>Archive <span>{live.filter(j=>j.stage==='applied').length}</span></button></nav>
      <div className="header-spacer"/>
      <div className="search"><span aria-hidden="true"><Icon name="search" size={15}/></span><input ref={search} id="find" aria-label="Search catalogue" placeholder="Search company, role, tech" value={query} onChange={e=>setQuery(e.target.value)}/><kbd>/</kbd></div>
      <div className="header-spacer"/><button className="settings-button" onClick={()=>setSettings('settings')}><Icon name="settings" size={15}/> Settings</button>
    </header>
    {view==='collection'&&<Collection onRulesChanged={()=>{epoch.current++;void refresh().catch(e=>setError(e.message));}} onError={setError}/>}
    {view!=='collection'&&<div className="workspace">
      <aside className="catalogue" aria-label="Vacancies"><div className="list-toolbar"><button aria-pressed={unread} onClick={()=>setUnread(!unread)}>Unread</button><div className="filter-box" ref={filterBox}><button aria-expanded={filtersOpen} onClick={()=>setFiltersOpen(!filtersOpen)}><Icon name="filters" size={14}/> Filters {activeCount>0&&<span className="filter-count">{activeCount}</span>}</button>{filtersOpen&&<div className="filter-popover"><div className="quick-filters">{([['web3','Web3 / blockchain only'],['remote','Remote only'],['strong','Strong fit (75+)'],['hideExpired','Hide expired'],['management','Management title']] as const).map(([key,label])=><label key={key}><input type="checkbox" checked={quick[key]} onChange={e=>setQuick({...quick,[key]:e.target.checked})}/>{label}</label>)}</div><p className="muted">Management matches CTO, Head, Director, Manager or Lead in the title, excluding lead generation. Responsibilities are not inferred.</p><Filters jobs={live} options={options} selection={selection} setSelection={setSelection} rows={queue?.rows??[]} shown={filtered.length} reset={resetFilters}/></div>}</div><label className="sort-control"><span className="sr-only">Sort roles</span><select aria-label="Sort roles" value={sort} onChange={e=>setSort(e.target.value as 'fit'|'newest')}><option value="fit">Best fit</option><option value="newest">Newest</option></select></label></div>
        {unread&&<p className="unread-note">Opening marks a role read. It stays here until you choose another role.</p>}
        {view==='applied'&&<p className="archive-retention">Applications are permanently deleted 30 days after the confirmed application date. Entries without a valid date are kept.</p>}
        <div className="job-list">{filtered.map(j=><button key={j.id} data-job-id={j.id} onClick={()=>{if(j.id!==selected)setHeldUnread(null);setSelected(j.id);}} aria-pressed={selected===j.id} className={`job-row ${selected===j.id?'selected':''} ${j.seen?'seen':'unseen'} ${j.stage==='no'?'session-no':''}`}><div className="row-company">{j.isWeb3&&<span className="web3-mark" aria-label="Web3">⬡</span>}{j.company}</div><div className="row-icons">{j.expired&&<span title="Listing expired">◷</span>}{view==='triage'&&j.stage==='preselected'&&<span className="yes-check" title="Preparation">✓</span>}<span className={`score ${j.matchScore>=75?'strong':''}`}>{j.matchScore>0?j.matchScore:'—'}</span></div><strong>{j.title}</strong><div className="row-location">{j.location||'Location unstated'}{view==='applied'&&<span> · Applied {dateLabel(j.appliedAt)}</span>}</div></button>)}
          {!filtered.length&&<div className="empty">{query||activeCount?'No matching roles. Adjust your filters.':view==='triage'?'Inbox zero. Collect more jobs in Sources.':view==='preselected'?'Nothing in Preparation yet — press Y in Triage.':'No applications yet.'}{!!(query||activeCount)&&<button onClick={resetFilters}>Reset filters</button>}</div>}
          {view==='triage'&&filtered.some(j=>j.stage==='no')&&<p className="session-note">{filtered.filter(j=>j.stage==='no').length} dismissed this session — they disappear next time.</p>}
        </div>
        <form className="import" onSubmit={async e=>{e.preventDefault();setBusy(true);setError('');try{const r=await api('import','POST',{url:importUrl});epoch.current++;await refresh();setView('triage');setSelected(r.id);setImportUrl('');}catch(e:any){setError(e.message);}finally{setBusy(false);}}}><label htmlFor="import">Import a public job URL</label><div><input id="import" type="url" required value={importUrl} onChange={e=>setImportUrl(e.target.value)} placeholder="https://…"/><button disabled={busy}>Import</button></div></form>
      </aside>
      <main className="detail" aria-label="Selected vacancy"><div className="detail-inner">
        {error&&<div className="banner error" role="alert">{error}</div>}{notice&&<div className="banner" role="status">{notice}<button aria-label="Dismiss notification" onClick={()=>setNotice('')}>×</button></div>}
        {job?<><div className="detail-top">
          <div className="role-heading"><p className="company">{job.isWeb3&&<span className="web3-chip"><span className="web3-mark">⬡</span> Web3</span>}{job.company}</p><h1>{job.title}</h1><div className="role-meta"><span>{job.location||'Location not stated'}</span>{job.expired&&<span>◷ Listing expired</span>}<a href={job.originalUrl} target="_blank" rel="noreferrer">Original listing ↗</a>{!job.applyUrl&&<span className="warning">Employer apply link not resolved</span>}</div></div>
          <div className="review-actions">{view==='triage'&&job.stage==='triage'?<><button className="no-button" disabled={busy} onClick={()=>transition('no')}>✕ Dismiss <kbd>N</kbd></button><button disabled={busy} onClick={doBlockCompany}>Block company</button><button className="primary" disabled={busy} onClick={()=>transition('preselected')}>✓ Select <kbd>Y</kbd></button></>:view==='preselected'?<><button disabled={busy} onClick={()=>transition('triage')}>Back to triage</button><button className="primary" disabled={busy} onClick={apply}>Apply ↗ <kbd>A</kbd></button><button className="confirm-later" disabled={busy} onClick={()=>setConfirmation(job)}>Confirm application…</button></>:<><span className={job.stage==='no'?'danger':'muted'}>{job.stage==='no'?'Dismissed — hidden after this session':job.stage==='applied'?`Applied ${dateLabel(job.appliedAt)}`:'Moved to Preparation'}</span><button disabled={busy} onClick={()=>transition(job.stage==='applied'?'preselected':'triage')}>Undo</button></>}</div>
        </div>
          {view!=='triage'&&<div className="archive-actions">{job.stage==='applied'&&<p className="muted">{validAppliedAt(job.appliedAt)&&Date.parse(job.appliedAt)<=Date.now()?`Confirmed ${dateLabel(job.appliedAt)}. Permanent removal after ${new Date(Date.parse(job.appliedAt)+archiveRetentionMs).toLocaleString()}.`:'No valid past confirmed application date; automatic deletion is disabled.'}</p>}<button disabled={busy} onClick={()=>setDeleting(job)}>Delete role…</button></div>}
          <RoleContent key={job.id} job={job} queue={queue} onQueue={setQueue} refresh={refresh} onError={setError} onNotice={setNotice}/>
        </>:<div className="empty"><h1>{filtered.length?'Select a role':'No role selected'}</h1><p>{filtered.length?'Choose a role to review its evidence.':'This stage has no visible roles. Change stage or filters, or collect jobs in Sources.'}</p></div>}
      </div></main>
    </div>}
    <footer className="queue-bar" aria-label="Assessment queue">
      <div className="assessment-control">{assessmentOn===null?<span role="status">Loading assessment setting…</span>:<button type="button" role="switch" aria-label="Assess vacancies" aria-checked={assessmentOn} aria-describedby="assessment-switch-state" disabled={savingAssessment} onClick={toggleAssessment}><span className="switch-track" aria-hidden="true"><span/></span>Assess vacancies</button>}<span id="assessment-switch-state" role="status">{savingAssessment?'Saving…':assessmentOn===false?(queue?.counts.running?'Off · current inference finishing':'Off'):assessmentOn===true?'On':''}</span></div>
      <div className="queue-status"><button className="queue-action" onClick={()=>setSettings('queue')} title={runtime?.reason??'Inspect assessment queue and errors'}>{runtime?.paused?'Paused · details':'Queue'}</button><span className={queue?.counts.running?'running':''}>● {queue?.counts.running??0} running</span><span>{queue?.counts.pending??0} pending</span><button onClick={()=>setSettings('queue')}>{queue?.counts.failed??0} failed</button><span className="warning">{queue?.counts['missing-description']??0} missing desc.</span><span>{queue?.counts.done??0} current</span><span className="queue-provider"><Icon name="target" size={13}/> Assessment: {stageModel('assessment')} <span className="queue-sep">·</span> <Icon name="pencil" size={13}/> Application: {stageModel('preparation')}{queue?.rows.filter((r:any)=>r.status==='running').map((r:any)=>` · #${r.jobId??r.job_id} ${Math.floor((r.elapsedMs??((r.finished??Date.now())-r.started))/1000)}s`)}</span></div>
      {assessmentError&&<div className="assessment-switch-error" role="alert">{assessmentError}</div>}
    </footer>
    {settings&&<Settings panel={settings} setPanel={setSettings} queue={queue} onError={setError} onRulesChanged={()=>{epoch.current++;void refresh().catch(e=>setError(e.message));}}/>}
    {confirmation&&<ApplicationConfirmation job={confirmation} onClose={()=>setConfirmation(null)} onConfirm={()=>transition('applied',true,confirmation)}/>}
    {deleting&&<DeleteRole job={deleting} onClose={()=>setDeleting(null)} onDeleted={(id,cleanupIncomplete)=>{epoch.current++;setJobs(old=>old.filter(j=>j.id!==id));setDeleting(null);setNotice(cleanupIncomplete?'Role deleted, but a local technical log could not be removed. See server logs.':'Role and application history permanently deleted.');}}/>}
  </div>;
}
createRoot(document.getElementById('root')!).render(<App/>);
