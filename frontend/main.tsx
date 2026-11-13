import React,{useEffect,useRef,useState} from 'react';
import {messageKind, messageText, SERVER_DOWN_TEXT} from './messages';
import {createRoot} from 'react-dom/client';
import type {Job} from './model/core';
import type {Stage} from './model/workflow';
import {externalApplyUrl,archiveRetentionMs,validAppliedAt,outcomeReasonLabel,type OutcomeReason} from './model/workflow';
import {api,blockJob,isServerUnreachable} from './api';
import {Filters} from './CatalogueFilters';
import {emptySelection,filterCatalogue,migrateSelection,selectionCount,sortCatalogue,type CatalogueSelection} from './filters';
import {publicationText} from './dates';
import {summarizeLocation} from './location';
import {normalizeQuick,quickCount,type FitBucket,emptyQuick,nextUndecided,quickMatch,shortcutGuard,visibleInStage,type QuickFilters,type View} from './workflow';
import {Settings} from './Settings';
import {ProfileScreen} from './profile/ProfileScreen';
import {ProfileSwitcher} from './profile/ProfileSwitcher';
import {RoleContent} from './RoleContent';
import {ScorePair} from './QuickMatchPanel';
import type {AnalysisStatus} from './quickmatch';
import {ApplicationConfirmation} from './ApplicationConfirmation';
import {DeleteRole} from './DeleteRole';
import {Icon} from './Icon';
import {Brand} from './Brand';
import {ListResizer,useListWidth} from './ListResizer';
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
import './theme.css';
import './profile/profile.css';

function preference<T>(key:string,fallback:T):T {try{return JSON.parse(localStorage.getItem(key)??'null')??fallback;}catch{return fallback;}}
export const dateLabel=(date:string|null)=>date?new Date(date).toLocaleDateString(undefined,{month:'short',day:'numeric',year:'numeric'}):'Date not recorded';
function App(){
  const [jobs,setJobs]=useState<Job[]>([]),[selected,setSelected]=useState<number|null>(null),[view,setView]=useState<View>('triage');
  const [session,setSession]=useState(new Set<number>()),[query,setQuery]=useState(()=>localStorage.getItem('fcto.catalogue.query')??'');
  const [selection,setSelection]=useState<CatalogueSelection>(()=>({...migrateSelection(preference('fcto.catalogue.filters',{}),preference('sift.filters.version',1)),titles:null,keywords:null}));
  const [unread,setUnread]=useState(()=>preference('sift.unread',false)),[sort,setSort]=useState<'fit'|'newest'>(()=>preference('sift.sort','fit')),[heldUnread,setHeldUnread]=useState<number|null>(null);
  // The toggle is also saved on the server, because the background analysis follows it; the server's value wins on load.
  const chooseSort=(next:'fit'|'newest')=>{if(next===sort)return;setSort(next);api('quick-match/order','PUT',{order:next}).catch(()=>{});};
  useEffect(()=>{api('quick-match/order').then(r=>{if(r?.order==='fit'||r?.order==='newest')setSort(r.order);}).catch(()=>{});},[]);
  const [quick,setQuick]=useState<QuickFilters>(()=>normalizeQuick(preference('sift.quick',{})));
  const [filtersOpen,setFiltersOpen]=useState(false),[options,setOptions]=useState<any>(null),[queue,setQueue]=useState<any>(null),[analysis,setAnalysis]=useState<AnalysisStatus|null>(null);
  const [settings,setSettings]=useState<string|null>(null),[profileOpen,setProfileOpen]=useState(false),[profileSection,setProfileSection]=useState<string|null>(null),[error,setError]=useState(''),[notice,setNotice]=useState(''),[importUrl,setImportUrl]=useState(''),[busy,setBusy]=useState(false);
  const [confirmation,setConfirmation]=useState<Job|null>(null);
  const [listWidth,setListWidth]=useListWidth();
  const [deleting,setDeleting]=useState<Job|null>(null);
  const [archiveView,setArchiveView]=useState<'applied'|'dismissed'>('applied');
  const [dismissedJobs,setDismissedJobs]=useState<Job[]>([]);
  const [assessmentOn,setAssessmentOn]=useState<boolean|null>(null),[savingAssessment,setSavingAssessment]=useState(false),[assessmentError,setAssessmentError]=useState('');
  const assessmentRevision=useRef(0),assessmentSaving=useRef(false);
  const search=useRef<HTMLInputElement>(null),filterBox=useRef<HTMLDivElement>(null),writes=useRef(0),epoch=useRef(0),polling=useRef(false),pollError=useRef(false),lastCatalogueSeq=useRef<number>(-1),lastPollRefresh=useRef<number>(0);
  const replaceJob=(j:Job)=>setJobs(old=>old.map(x=>x.id===j.id?j:x));
  // Reject snapshots started before a workflow write. Serialize polling and keep review state separate.
  async function refresh(){const revision=epoch.current;const data=await api('jobs','GET',undefined,30000);if(!writes.current&&revision===epoch.current)setJobs(data);}
  useEffect(()=>{api('filter-options','GET',undefined,10000).then(setOptions).catch(e=>{if(!isServerUnreachable(e))setError(e.message);});},[]);
  // Fire one unforced collection on app mount so opening or reloading the app triggers a
  // throttled run; the server-side cooldown prevents repeated runs within the configured interval.
  useEffect(()=>{api('search','POST',{force:false}).catch(()=>{});},[]);
  useEffect(()=>{let alive=true;async function poll(){if(polling.current)return;polling.current=true;const revision=assessmentRevision.current;try{const q=await api('queue','GET',undefined,10000);const a=await api('quick-match/analysis','GET',undefined,10000).catch(()=>null);if(alive){setQueue(q);setAnalysis(a);if(!assessmentSaving.current&&revision===assessmentRevision.current&&typeof q.enabled==='boolean')setAssessmentOn(q.enabled);}await api('health','GET',undefined,10000);if(alive){const seq:number=typeof q.catalogueSeq==='number'?q.catalogueSeq:-1;if(seq!==lastCatalogueSeq.current){const now=Date.now();if(now-lastPollRefresh.current>=10000){lastPollRefresh.current=now;await refresh();lastCatalogueSeq.current=seq;}}if(pollError.current){setError('');pollError.current=false;}}}catch(e:any){if(alive){pollError.current=true;setError(isServerUnreachable(e)?SERVER_DOWN_TEXT:messageText(e.message));}}finally{polling.current=false;}}void poll();const timer=setInterval(poll,2500);return()=>{alive=false;clearInterval(timer);};},[]);
  async function toggleAssessment(){
    if(assessmentSaving.current||assessmentOn===null)return;
    assessmentSaving.current=true;assessmentRevision.current++;setSavingAssessment(true);setAssessmentError('');
    try{const q=await api('queue/assessment','PUT',{enabled:!assessmentOn});setAssessmentOn(q.enabled);setQueue(q);}
    catch(e:any){setAssessmentError(`Could not change assessment setting: ${e.message}`);}
    finally{assessmentSaving.current=false;assessmentRevision.current++;setSavingAssessment(false);}
  }
  useEffect(()=>{if(view==='applied'&&archiveView==='dismissed'){api('jobs?status=no').then((js:Job[])=>setDismissedJobs(js)).catch(e=>setError(e.message));}},[view,archiveView]);
  useEffect(()=>{localStorage.setItem('fcto.catalogue.query',query);localStorage.setItem('fcto.catalogue.filters',JSON.stringify(selection));localStorage.setItem('sift.filters.version','2');localStorage.setItem('sift.quick',JSON.stringify(quick));localStorage.setItem('sift.unread',JSON.stringify(unread));localStorage.setItem('sift.sort',JSON.stringify(sort));},[query,selection,quick,unread,sort]);
  const resetFilters=()=>{setSelection(emptySelection());setQuick(emptyQuick());setQuery('');setUnread(false);chooseSort('fit');};
  // A stop-word hides a role from every view and every count; the server already omits stopped roles from the list, this keeps a stale client copy hidden too.
  const live=jobs.filter(j=>!j.stopped);
  const isArchive=view==='applied';
  const baseFiltered=filterCatalogue(live,selection,queue?.rows??[]).filter(j=>visibleInStage(j,view,session)&&(!j.blocked||view!=='triage')&&quickMatch(j,quick,query)&&(!unread||isArchive||!j.seen||j.id===heldUnread));
  const filtered=isArchive?[...baseFiltered].sort((a,b)=>{const da=a.appliedAt?Date.parse(a.appliedAt):-Infinity,db=b.appliedAt?Date.parse(b.appliedAt):-Infinity;return db-da;}):sortCatalogue(baseFiltered,sort);
  const inDismissed=view==='applied'&&archiveView==='dismissed';
  const job=inDismissed?dismissedJobs.find(j=>j.id===selected):filtered.find(j=>j.id===selected);
  useEffect(()=>{
    const list=inDismissed?dismissedJobs:filtered;
    if(!list.some(j=>j.id===selected))setSelected(list[0]?.id??null);
  },[filtered.map(j=>j.id).join(','),dismissedJobs.map(j=>j.id).join(','),selected,inDismissed]);
  useEffect(()=>{
    if(!job||job.seen)return;setHeldUnread(job.id);writes.current++;epoch.current++;
    api(`jobs/${job.id}/seen`,'POST',{}).then((j:Job)=>setJobs(old=>old.map(x=>x.id===j.id?{...x,seen:j.seen,seenAt:j.seenAt}:x))).catch(e=>setError(e.message)).finally(()=>{writes.current--;epoch.current++;});
  },[job?.id,job?.seen]);
  async function transition(stage:Stage,confirmed=false,target=job,reason?:OutcomeReason){
    if(!target||busy)return false;const id=target.id,next=nextUndecided(filtered,id);setBusy(true);setError('');writes.current++;epoch.current++;
    try{const result=await api(`jobs/${id}`,'PATCH',{stage,confirmed,...(reason?{reason}:{})});
      // PATCH returns the undecorated job. Refetch the decorated read so the visible Quick Match
      // envelope is not dropped by a workflow transition.
      try{replaceJob(await api(`jobs/${id}`));}catch{replaceJob({...result,quickMatch:target.quickMatch,quickMatchError:target.quickMatchError});}
      if(view==='triage'&&['preselected','no'].includes(stage)){setSession(s=>new Set([...s,id]));setSelected(next);}
      else if(stage==='triage')setSession(s=>{const n=new Set(s);n.delete(id);return n;});
      return true;
    }catch(e:any){setError(e.message);return false;}finally{writes.current--;epoch.current++;setBusy(false);}
  }
  function apply(){
    if(!job||view!=='preselected'||job.stage!=='preselected'||busy)return;
    const url=externalApplyUrl(job);if(!url){setError('No valid employer or original listing URL is available. The role stays in Shortlist.');return;}
    // Open within the original click/key gesture. No await, form inspection or model call.
    let popup:Window|null=null;
    try{popup=window.open('about:blank','_blank');if(popup){popup.opener=null;popup.location.href=url;}}
    catch{popup?.close();popup=null;}
    if(!popup){setError('The application page was blocked. Allow popups and try Apply again. The role stays in Shortlist.');return;}
    setError('');setConfirmation(job);
  }
  async function restoreDismissed(j:Job){if(busy)return;setBusy(true);setError('');writes.current++;epoch.current++;try{await api(`jobs/${j.id}`,'PATCH',{stage:'triage'});setDismissedJobs(prev=>prev.filter(d=>d.id!==j.id));setSelected(dismissedJobs.find(d=>d.id!==j.id)?.id??null);}catch(e:any){setError(e.message);}finally{writes.current--;epoch.current++;setBusy(false);}}
  async function doBlockCompany(){
    if(!job||busy)return;setBusy(true);setError('');writes.current++;epoch.current++;
    try{await blockJob(job.id,'company');const data=await api('jobs');if(!writes.current)setJobs(data);}
    catch(e:any){setError(e.message);}finally{writes.current--;epoch.current++;setBusy(false);}
  }
  function openProfile(section?:string){setProfileSection(section??null);setProfileOpen(true);}
  function switchView(next:View){setProfileOpen(false);setProfileSection(null);setSettings(null);setView(next);setSelected(null);setHeldUnread(null);setError('');setNotice('');if(next!=='applied')setArchiveView('applied');}
  function move(delta:number){if(!filtered.length)return;const i=filtered.findIndex(j=>j.id===selected);setHeldUnread(null);setSelected(filtered[(i+delta+filtered.length)%filtered.length].id);}
  useEffect(()=>{function key(e:KeyboardEvent){
    if(profileOpen||settings||document.querySelector('dialog[open]')||e.metaKey||e.ctrlKey||e.altKey||e.isComposing)return;
    if(e.key==='Escape'){setFiltersOpen(false);(document.activeElement as HTMLElement)?.blur();return;}
    if(shortcutGuard(e.target))return;
    const key=e.key.toLowerCase(),select=e.code==='KeyY';
    if(['j','k','arrowdown','arrowup','/','y','n','a'].includes(key)||select)e.preventDefault();
    if(key==='j'||key==='arrowdown')move(1);if(key==='k'||key==='arrowup')move(-1);
    if(key==='/')search.current?.focus();
    if(view==='triage'&&job?.stage==='triage'){if(select)void transition('preselected');if(key==='n')void transition('no');}
    if(key==='a')apply();
  }window.addEventListener('keydown',key);return()=>window.removeEventListener('keydown',key);});
  useEffect(()=>{function outside(e:PointerEvent){if(filterBox.current&&!filterBox.current.contains(e.target as Node))setFiltersOpen(false);}document.addEventListener('pointerdown',outside);return()=>document.removeEventListener('pointerdown',outside);},[]);
  const activeCount=quickCount(quick)+selectionCount(selection)+(unread?1:0);
  return <div className="sift-app">
    <header className="app-header">
      <nav className="stages" aria-label="Workflow stages"><button aria-pressed={view==='triage'} className={view==='triage'&&!profileOpen&&!settings?'active':''} onClick={()=>switchView('triage')}>Inbox <span>{live.filter(j=>j.stage==='triage'&&!j.blocked).length}</span></button><button aria-pressed={view==='preselected'} className={view==='preselected'&&!profileOpen&&!settings?'active':''} onClick={()=>switchView('preselected')}>Shortlist <span>{live.filter(j=>j.stage==='preselected').length}</span></button><button aria-pressed={view==='applied'} className={view==='applied'&&!profileOpen&&!settings?'active':''} onClick={()=>switchView('applied')}>Archive <span>{live.filter(j=>j.stage==='applied').length}</span></button></nav>
      <Brand/>
      <div className="header-actions"><ProfileSwitcher open={profileOpen} onOpenProfile={()=>openProfile()}/><button className={`settings-button${settings?' active':''}`} aria-pressed={!!settings} onClick={()=>{setProfileOpen(false);setSettings('sources');}}><Icon name="settings" size={15}/> Settings</button></div>
    </header>
    {profileOpen?<ProfileScreen initialSection={profileSection??undefined} onClose={()=>{setProfileOpen(false);setProfileSection(null);void refresh().catch(e=>setError(e.message));}} onOpenSettings={()=>{setProfileOpen(false);setSettings('sources');}}/>:settings?<Settings panel={settings} setPanel={setSettings} queue={queue} assessmentOn={assessmentOn} savingAssessment={savingAssessment} assessmentError={assessmentError} toggleAssessment={toggleAssessment} onError={setError} onRulesChanged={()=>{epoch.current++;void refresh().catch(e=>setError(e.message));}}/>:<>
    {<div className="workspace" style={{'--list-w':`${listWidth}px`} as React.CSSProperties}>
      <aside className="catalogue" aria-label="Vacancies"><ListResizer width={listWidth} onChange={setListWidth}/><div className="search list-search"><span aria-hidden="true"><Icon name="search" size={15}/></span><input ref={search} id="find" aria-label="Search catalogue" placeholder="Search company, role, tech" value={query} onChange={e=>setQuery(e.target.value)}/><kbd>/</kbd></div><div className="list-toolbar">{!isArchive&&<button aria-pressed={unread} onClick={()=>setUnread(!unread)}>Unread</button>}<div className="filter-box" ref={filterBox}><button aria-expanded={filtersOpen} onClick={()=>setFiltersOpen(!filtersOpen)}><Icon name="filters" size={14}/> Filters {activeCount>0&&<span className="filter-count">{activeCount}</span>}</button>{filtersOpen&&<div className="filter-popover"><div className="fp-groups">
<section className="fp-group"><h4>Fits you</h4><div className="fp-chips">{([['strong','Strong 75+'],['partial','Partial 55–74'],['weak','Weak'],['unscored','Not scored']] as const).map(([key,label])=><button key={key} type="button" className="fp-chip" aria-pressed={quick.fit.includes(key as FitBucket)} onClick={()=>setQuick({...quick,fit:quick.fit.includes(key as FitBucket)?quick.fit.filter(f=>f!==key):[...quick.fit,key as FitBucket]})}>{label}</button>)}</div></section>
<section className="fp-group"><h4>Collected</h4><div className="fp-chips"><div className="fp-seg" role="group" aria-label="Collected within">{([['','Any time'],['7','7 days'],['30','30 days']] as const).map(([key,label])=><button key={key||'any'} type="button" aria-pressed={quick.fresh===key} onClick={()=>setQuick({...quick,fresh:key})}>{label}</button>)}</div><button type="button" className="fp-chip" aria-pressed={quick.hideExpired} onClick={()=>setQuick({...quick,hideExpired:!quick.hideExpired})}>Hide expired</button></div></section>
<section className="fp-group"><h4>Role</h4><div className="fp-chips">{([['remote','Remote'],['web3','Web3 / blockchain'],['management','Leadership title']] as const).map(([key,label])=><button key={key} type="button" className="fp-chip" aria-pressed={quick[key]} title={key==='management'?'CTO, Head, Director, Manager or Lead in the title':undefined} onClick={()=>setQuick({...quick,[key]:!quick[key]})}>{label}</button>)}</div></section>
</div><Filters jobs={live} options={options} selection={selection} setSelection={setSelection} rows={queue?.rows??[]} shown={filtered.length} reset={resetFilters}/></div>}</div>{!isArchive&&<div className="sort-toggle" role="group" aria-label="Sort roles">{([['fit','Best fit'],['newest','Newest']] as const).map(([key,label])=><button key={key} type="button" aria-pressed={sort===key} onClick={()=>chooseSort(key)}>{label}</button>)}</div>}</div>
        {!isArchive&&unread&&<p className="unread-note">Opening marks a role read. It stays here until you choose another role.</p>}
        {view==='applied'&&<div className="archive-tabs"><button className={archiveView==='applied'?'active':''} aria-pressed={archiveView==='applied'} onClick={()=>{setArchiveView('applied');setSelected(null);}}>Applied</button><button className={archiveView==='dismissed'?'active':''} aria-pressed={archiveView==='dismissed'} onClick={()=>{setArchiveView('dismissed');setSelected(null);}}>Dismissed</button></div>}
        {view==='applied'&&archiveView==='applied'&&<p className="archive-retention">Applications are permanently deleted 30 days after the confirmed application date. Entries without a valid date are kept.</p>}
        <div className="job-list">{inDismissed
          ?<>{dismissedJobs.map(j=><button key={j.id} data-job-id={j.id} onClick={()=>setSelected(j.id)} aria-pressed={selected===j.id} className={`job-row ${selected===j.id?'selected':''} ${j.seen?'seen':'unseen'}`}><div className="row-company">{j.isWeb3&&<span className="web3-mark" aria-label="Web3">⬡</span>}{j.company}</div><div className="row-icons"><ScorePair fit={j.fit}/></div><strong>{j.title}</strong><div className="row-location" title={summarizeLocation(j.location).compressed?j.location:undefined}>{summarizeLocation(j.location).text||'Location unstated'}</div><div className="row-location">{publicationText(j.publishedAt,j.publishedBasis)}</div></button>)}{!dismissedJobs.length&&<div className="empty">No dismissed roles.</div>}</>
          :<>{filtered.map(j=><button key={j.id} data-job-id={j.id} onClick={()=>{if(j.id!==selected)setHeldUnread(null);setSelected(j.id);}} aria-pressed={selected===j.id} className={`job-row ${selected===j.id?'selected':''} ${j.seen?'seen':'unseen'} ${j.stage==='no'?'session-no':''}`}><div className="row-company">{j.isWeb3&&<span className="web3-mark" aria-label="Web3">⬡</span>}{j.company}</div><div className="row-icons">{j.expired&&<span title="Listing expired">◷</span>}{view==='triage'&&j.stage==='preselected'&&<span className="yes-check" title="Shortlisted">✓</span>}<ScorePair fit={j.fit}/></div><strong>{j.title}</strong><div className="row-location" title={summarizeLocation(j.location).compressed?j.location:undefined}>{summarizeLocation(j.location).text||'Location unstated'}{view==='applied'&&<span> · Applied {dateLabel(j.appliedAt)}</span>}</div><div className="row-location">{publicationText(j.publishedAt,j.publishedBasis)}</div></button>)}
          {!filtered.length&&<div className="empty">{query||activeCount?'No matching roles. Adjust your filters.':view==='triage'?'Inbox zero. Collect more jobs in Settings → Sources.':view==='preselected'?'Nothing shortlisted yet — press Y in Inbox.':'No applications yet.'}{!!(query||activeCount)&&<button onClick={resetFilters}>Reset filters</button>}</div>}
          {view==='triage'&&filtered.some(j=>j.stage==='no')&&<p className="session-note">{filtered.filter(j=>j.stage==='no').length} dismissed this session — they disappear next time.</p>}</>}
        </div>
        
      </aside>
      <main className="detail" aria-label="Selected vacancy"><div className="detail-inner">
        {error&&<div className="banner error" data-kind={messageKind(error)} role="alert">{error}</div>}{notice&&<div className="banner" role="status">{notice}<button aria-label="Dismiss notification" onClick={()=>setNotice('')}>×</button></div>}
        {job?<><div className="detail-top">
          <div className="role-heading"><p className="company">{job.isWeb3&&<span className="web3-chip"><span className="web3-mark">⬡</span> Web3</span>}{job.company}</p><h1>{job.title}</h1><div className="role-meta">{job.expired&&<span className="state-expired">Listing expired</span>}<span>{publicationText(job.publishedAt,job.publishedBasis)}</span>{job.sources?.map(x=><a key={x.url} className="source-link" href={x.url} target="_blank" rel="noreferrer" title={`Found on ${x.name}`}>{x.name}</a>)}<a href={job.originalUrl} target="_blank" rel="noreferrer">Original listing ↗</a>{!job.applyUrl&&<span className="warning">Employer apply link not resolved</span>}</div></div>
          <div className="review-actions">{inDismissed?<><span className="danger">Dismissed{job.outcomeReason?` — ${outcomeReasonLabel(job.outcomeReason)}`:''}</span><button disabled={busy} onClick={()=>restoreDismissed(job)}>Restore to Inbox</button></>:view==='triage'&&job.stage==='triage'?<><button disabled={busy} onClick={doBlockCompany}>Block company</button><button className="no-button" disabled={busy} onClick={()=>transition('no')}>✕ Dismiss <kbd>N</kbd></button><button className="primary" disabled={busy} onClick={()=>transition('preselected')}>✓ Select <kbd>Y</kbd></button></>:view==='preselected'?<><button disabled={busy} onClick={()=>transition('triage')}>Back to Inbox</button><button className="primary" disabled={busy} onClick={apply}>Apply ↗ <kbd>A</kbd></button><button className="confirm-later" disabled={busy} onClick={()=>setConfirmation(job)}>Confirm application…</button></>:<><span className={job.stage==='no'?'danger':'muted'}>{job.stage==='no'?'Dismissed — hidden after this session':job.stage==='applied'?`Applied ${dateLabel(job.appliedAt)}`:'Moved to Shortlist'}</span><button disabled={busy} onClick={()=>transition(job.stage==='applied'?'preselected':'triage')}>Undo</button></>}</div>
        </div>
          {(view!=='triage')&&<div className="archive-actions">{job.stage==='applied'&&<p className="muted">{validAppliedAt(job.appliedAt)&&Date.parse(job.appliedAt)<=Date.now()?`Confirmed ${dateLabel(job.appliedAt)}. Permanent removal after ${new Date(Date.parse(job.appliedAt)+archiveRetentionMs).toLocaleString()}.`:'No valid past confirmed application date; automatic deletion is disabled.'}</p>}<button disabled={busy} onClick={()=>setDeleting(job)}>Delete role…</button></div>}
          <RoleContent key={job.id} job={job} analysis={analysis} queue={queue} onQueue={setQueue} refresh={refresh} onError={setError} onNotice={setNotice} onOpenProfile={openProfile}/>
        </>:<div className="empty"><h1>{filtered.length?'Select a role':'No role selected'}</h1><p>{filtered.length?'Choose a role to review its evidence.':'This stage has no visible roles. Change stage or filters, or collect jobs in Sources.'}</p></div>}
      </div></main>
    </div>}</>}
    {confirmation&&<ApplicationConfirmation job={confirmation} onClose={()=>setConfirmation(null)} onConfirm={()=>transition('applied',true,confirmation)} onDismiss={(reason)=>transition('no',false,confirmation,reason)}/>}
    {deleting&&<DeleteRole job={deleting} onClose={()=>setDeleting(null)} onDeleted={(id,cleanupIncomplete)=>{epoch.current++;setJobs(old=>old.filter(j=>j.id!==id));setDismissedJobs(old=>old.filter(j=>j.id!==id));setDeleting(null);setNotice(cleanupIncomplete?'Role deleted, but a local technical log could not be removed. See server logs.':'Role and application history permanently deleted.');}}/>}
  </div>;
}
createRoot(document.getElementById('root')!).render(<App/>);
