import React,{useEffect,useRef,useState} from 'react';
import type {Job} from './model/core';
import {parseListingHtml,type HtmlNode} from './listingHtml';
import {api} from './api';
import {assessmentStatus} from './filters';
import {publicationInfo} from './dates';
import {summarizeLocation} from './location';
import {ApplicationDraftView} from './ApplicationDraft';
import {FitsYouBand} from './FitsYouBand';
import {WhatTheyAsk} from './WhatTheyAsk';
import {WhatTheyOffer} from './WhatTheyOffer';
import {fitsYouCell,youFitCell,youFitOff,type AnalysisStatus} from './quickmatch';

type Props={job:Job;analysis?:AnalysisStatus|null;queue:any;onQueue:(q:any)=>void;refresh:()=>Promise<void>;onError:(s:string)=>void;onNotice:(s:string)=>void;onOpenProfile:(section?:string)=>void};
export function RoleContent({job,analysis,queue,onQueue,refresh,onError,onNotice,onOpenProfile}:Props){
  const [tab,setTab]=useState<'overview'|'application'>('overview'),[applicationOpened,setApplicationOpened]=useState(false);
  const [options,setOptions]=useState<any>(null),[provider,setProvider]=useState('');
  const [feedback,setFeedback]=useState(''),[scope,setScope]=useState('general'),[history,setHistory]=useState<any[]>([]),[revision,setRevision]=useState<string>(),[saving,setSaving]=useState(false);
  // The job list payload omits band and whatTheyOffer; fetch the full single-job record
  // when a job is opened so those fields are available.
  const [fullJob,setFullJob]=useState<Partial<Job>>({});
  useEffect(()=>{
    let active=true;
    setFullJob({}); // clear previous job's data while loading
    api(`jobs/${job.id}`).then(r=>{if(active)setFullJob(r);}).catch(()=>{});
    return()=>{active=false;};
  },[job.id]);
  const band=fullJob.band??job.band;
  const whatTheyOffer=fullJob.whatTheyOffer??job.whatTheyOffer;
  useEffect(()=>{api('assessment-options').then(setOptions).catch(e=>onError(e.message));},[]);
  useEffect(()=>{let active=true;api(`jobs/${job.id}/feedback`).then(r=>{if(active)setHistory(r);}).catch(e=>onError(e.message));return()=>{active=false;};},[job.id]);
  const assessment=job.assessment,row=queue?.rows.find((r:any)=>r.job_id===job.id),publication=publicationInfo(job.publishedAt,job.publishedBasis);
  const fitsYou=fitsYouCell(job.fit),youFit=youFitCell(job.fit,analysis);
  const fitsYouWidth=/^\d+$/.test(fitsYou.number)?Number(fitsYou.number):0;
  // After any profile change tied to this job, fetch only this job's updated fit result.
  async function refreshJobFit(){
    try{
      const updated=await api(`jobs/${job.id}`);
      setFullJob(updated);
    }catch(_){}
  }
  async function addSkill(name:string){
    try{
      const r=await api('profile/skills','POST',{name,jobId:job.id});
      if(r.action==='offer_alternatives'&&r.alternatives?.length){
        // Caller (AskRow) handles the alternatives UI; return the response so it can read r.alternatives.
        return r;
      }
      if(r.action==='fact_confirmed'){
        if(r.changed===false){
          onNotice(`"${r.factConfirmed||name}" is already in your confirmed facts.`);
        }else{
          onNotice(`Recorded "You have: ${r.factConfirmed||name}" in your profile.`);
        }
      }else{
        const added:string[]=r.skillsAdded||[name];
        if(r.changed===false){
          onNotice(`${added.map((s:string)=>`"${s}"`).join(', ')} already in your skills.`);
        }else{
          onNotice(`Added ${added.map((s:string)=>`"${s}"`).join(', ')} to your skills (level Proficient; change in your profile).`);
        }
      }
    }catch(e:any){onError(e.message);throw e;}
    await refreshJobFit();
    return undefined;
  }
  async function dismissSkill(name:string){
    try{
      const r=await api('profile/absent-skills','POST',{name,jobId:job.id});
      if(r.action==='absent_skill_recorded'){
        onNotice(`Noted that you do not have "${name}". Listed in your profile under Skills.`);
      }else{
        onNotice(`Recorded "You do not have: ${name}" in your profile.`);
      }
    }catch(e:any){onError(e.message);throw e;}
    await refreshJobFit();
  }
  async function confirmHave(text:string){
    try{
      await api('profile/confirmed-facts','POST',{text,have:true,jobId:job.id});
      onNotice(`Recorded "You have: ${text}" in your profile.`);
    }catch(e:any){onError(e.message);throw e;}
    await refreshJobFit();
  }
  async function confirmDoNotHave(text:string){
    try{
      await api('profile/confirmed-facts','POST',{text,have:false,jobId:job.id});
      onNotice(`Recorded "You do not have: ${text}" in your profile.`);
    }catch(e:any){onError(e.message);throw e;}
    await refreshJobFit();
  }
  async function refreshSource(){try{await api(`jobs/${job.id}/read`,'POST',{});await refresh();}catch(e:any){onError(e.message);}}
  // Open-time refetch: fire once per job opening when needs_refetch is true.
  const refetchOnceRef=useRef<number>(0);
  useEffect(()=>{
    if(job.needsRefetch && refetchOnceRef.current !== job.id){
      refetchOnceRef.current = job.id;
      void refreshSource();
    }
  },[job.id, job.needsRefetch]);

  const stale=assessment&&(assessment.stale||(row&&row.fingerprint!==assessment.inputFingerprint));
  const [reassessing,setReassessing]=useState(false);
  async function reassess(){if(reassessing)return;setReassessing(true);try{await api(`jobs/${job.id}/assess`,'POST',{provider:provider||undefined});onQueue(await api('queue'));onNotice('Assessment queued with priority.');}catch(e:any){onError(e.message);}finally{setReassessing(false);}}

  return <>
    <div className="detail-tabs-row">
      <nav className="detail-tabs" aria-label="Vacancy view">
        <button aria-pressed={tab==='overview'} className={tab==='overview'?'active':''} onClick={()=>setTab('overview')}>Overview</button>
        <button disabled={job.stage==='triage'} title={job.stage==='triage'?'Shortlist this role to open Application':undefined} aria-describedby={job.stage==='triage'?'ai-analysis-note':undefined} aria-pressed={tab==='application'} className={tab==='application'?'active':''} onClick={()=>{setApplicationOpened(true);setTab('application');}}>Application</button>
      </nav>
    </div>
    {tab==='application'&&<><section className="assessment">
        <div className="summary-card"><div className="score-overview"><div className="large-score">{fitsYou.number}<small>/100</small></div><div className="score-track"><span style={{width:`${fitsYouWidth}%`}}/></div><strong>Fits you{fitsYou.label?` — ${fitsYou.label}`:''}{!youFitOff(job.fit)&&` · You fit ${youFit.number}`}</strong></div><div className="summary-copy"><p>{assessment?.summary??'Not assessed yet. A configured provider will compare this description with versioned candidate evidence. No role is hidden for low fit.'}</p><div className="assessment-toolbar"><span className="assessed-date">{assessment?.createdAt?`Assessed ${new Date(assessment.createdAt).toLocaleString()}`:assessmentStatus(job,queue?.rows??[])}</span><select aria-label="Assessment provider" value={provider} onChange={e=>setProvider(e.target.value)}><option value="">Configured default ({options?.selected??'loading'})</option>{options?.providers.map((p:any)=><option key={p.name} value={p.name}>{p.name} · {p.model}</option>)}</select><button disabled={reassessing} onClick={reassess}>{reassessing?'Queuing…':'Assess now'}</button></div></div></div>
        {stale&&<p className="warning assessment-state">Candidate evidence or vacancy changed; this assessment uses an older snapshot.</p>}
        {job.fit?.stale&&<p className="warning assessment-state">The fit scores are being updated after a change; they are not presented as current.</p>}
        {row&&<p className={`assessment-state ${row.error?'warning':'muted'}`}>{row.status} · {row.provider} · {Math.floor(row.elapsedMs/1000)}s · attempt {row.attempts}{row.error?` · ${row.error}`:''}{row.next_at?` · Retry after ${new Date(row.next_at).toLocaleString()}`:''}</p>}
        {job.assessmentError&&<p className="error assessment-state">Assessment pending: {job.assessmentError}</p>}
        <div className="fit-grid"><section><h3>Evidence of fit <span>· {assessment?.evidence.length??0}</span></h3>{assessment?.evidence.length?assessment.evidence.map((e:any,i:number)=><div className="evidence" key={i}><p><span className="dot"/>{e.claim}</p><blockquote>"{e.quote}"</blockquote><small>{e.sourceId}</small></div>):<p className="empty-card">No verified supporting evidence returned.</p>}</section><div className="fit-unknowns">{[['confirmedMismatches','Confirmed mismatches'],['employerUnknowns','Unknown employer details'],['candidateUnknowns','Candidate facts / questions']].map(([key,label])=><section key={key}><h3>{label}</h3>{assessment?.[key]?.length?<ol className={`unknown-list ${key}`}>{assessment[key].map((g:string,i:number)=><li key={i}>{g}</li>)}</ol>:<p className="empty-card">None recorded.</p>}</section>)}</div></div>
        {!!assessment?.mismatchEvidence?.length&&<details><summary>Confirmed mismatch evidence</summary>{assessment.mismatchEvidence.map((m:any,i:number)=><div className="evidence" key={i}><p>{m.reason} · {m.sourceId}</p><blockquote>{m.quote}</blockquote><p>Vacancy requirement: {m.jobRequirement}</p></div>)}</details>}
        {assessment&&<details><summary>Assessment provenance</summary><pre>{JSON.stringify({provider:assessment.provider,model:assessment.model,createdAt:assessment.createdAt,contextVersion:assessment.contextVersion,promptVersion:assessment.promptVersion,selection:assessment.selection},null,2)}</pre></details>}
      </section>
      <section className="candidate-feedback"><h3>Candidate facts and corrections</h3><p>State experience, explicit absences or preferences in your own words. Saving confirms this text as candidate evidence and queues reassessment; it never drafts an employer answer.</p><label htmlFor="candidate-feedback">Your clarification</label><textarea id="candidate-feedback" value={feedback} onChange={e=>setFeedback(e.target.value)} placeholder="Add experience, preferences or a correction…"/><label htmlFor="feedback-scope">Scope</label><select id="feedback-scope" value={scope} onChange={e=>setScope(e.target.value)}><option value="general">General candidate knowledge</option><option value="job">This role only</option></select><button disabled={saving||!feedback.trim()} onClick={async()=>{setSaving(true);try{const r=await api(`jobs/${job.id}/feedback`,'POST',{text:feedback,scope,supersedes:revision});onNotice(`Saved verbatim to ${r.record.path}. ${r.record.conflicts.length?'Conflicts require review; earlier records retained.':'Reassessment queued.'}`);setFeedback('');setRevision(undefined);setHistory(await api(`jobs/${job.id}/feedback`));onQueue(r.queue);await refresh();}catch(e:any){onError(e.message);}finally{setSaving(false);}}}>{saving?'Saving…':'Save and reassess'}</button>{history.map(r=><details key={r.id}><summary>Saved {new Date(r.created).toLocaleString()} · {r.scope}</summary><pre>{r.text}</pre><p>{r.path}</p><pre>{JSON.stringify({facts:r.facts,conflicts:r.conflicts,supersedes:r.supersedes},null,2)}</pre><button onClick={()=>{setFeedback(r.text);setScope(r.scope);setRevision(r.id);}}>Edit as new revision</button></details>)}</section>
    </>}
    {applicationOpened&&<div hidden={tab!=='application'}><ApplicationDraftView jobId={job.id} canPrepare={job.stage==='preselected'}/></div>}
    {tab==='overview'&&<div className="overview-stack">
      {/* L.5r1: Band spans full detail panel width */}
      <FitsYouBand job={{...job, band}} analysis={analysis} onRefreshSource={refreshSource} onOpenProfile={onOpenProfile}/>
      {/* L.5r2: What they ask and What they offer in a 2-col grid, full width (no rail alongside) */}
      <div className="what-cards">
        <WhatTheyAsk fit={fullJob.fit??job.fit} onAddSkill={addSkill} onDismissSkill={dismissSkill} onConfirmHave={confirmHave} onConfirmDoNotHave={confirmDoNotHave}/>
        <WhatTheyOffer items={whatTheyOffer}/>
      </div>
      {/* L.5r3: Full listing always open, no By section / Original switch */}
      <section className="listing-panel">
        <header className="panel-head">
          <h2>Listing text</h2>
          <span className="panel-sub">{job.descriptionStatus}</span>
        </header>
        {(job.descriptionHtml||job.description)
          ?<ListingText text={job.description??''} descriptionHtml={job.descriptionHtml}/>
          :<p className="listing-missing">A full description has not been retrieved. Open the original listing; missing text is not an empty job.</p>}
      </section>
      {/* AI analysis note below listing; referenced by aria-describedby on the Application tab */}
      <section className="rail-card ai-note" id="ai-analysis-note"><h3>AI analysis</h3><p>{job.stage==='triage'?<>Detailed model assessment is shown in <b>Application</b> for shortlisted roles.</>:assessment?<>Assessed{assessment.createdAt?` ${new Date(assessment.createdAt).toLocaleDateString()}`:''}. Open <button className="link-button" onClick={()=>{setApplicationOpened(true);setTab('application');}}>Application</button> to read it.</>:<>Status: {assessmentStatus(job,queue?.rows??[])}. Results appear in <button className="link-button" onClick={()=>{setApplicationOpened(true);setTab('application');}}>Application</button>.</>}</p></section>
    </div>}
    {tab==='application'&&<section className="source-record"><h3>Source record</h3>{job.sources.map(s=><p key={s.url}><a href={s.url} target="_blank" rel="noreferrer">{s.name} ↗</a> · {new Date(s.seen).toLocaleString()}</p>)}<p className="muted">Board and employer metadata may be inaccurate. Application links open externally for your manual review. This app does not fill or submit forms.</p></section>}
  </>;
}

const BULLET=/^\s*(?:[-*•·▪◦–—]|\d{1,2}[.)])\s+/;
// Renders the listing text. When descriptionHtml is present it is preferred;
// the plain-text line renderer is the fallback when descriptionHtml is absent.
function ListingText({text,descriptionHtml}:{text:string;descriptionHtml?:string}){
  if(descriptionHtml) return <HtmlListingText html={descriptionHtml}/>;
  // Fallback: plain-text line renderer (unchanged from before L.4).
  const blocks=text.split(/\n\s*\n/).map(b=>b.split('\n').map(l=>l.trim()).filter(Boolean)).filter(b=>b.length);
  const out:React.ReactNode[]=[];
  blocks.forEach((lines,bi)=>{let list:string[]=[];const flush=(k:string)=>{if(list.length){out.push(<ul key={k}>{list.map((l,i)=><li key={i}>{l.replace(BULLET,'')}</li>)}</ul>);list=[];}};
    lines.forEach((l,li)=>{if(BULLET.test(l))list.push(l);else{flush(`${bi}-${li}-u`);out.push(<p key={`${bi}-${li}`} className={/:$/.test(l)&&l.length<90?'lead-in':undefined}>{l}</p>);}});flush(`${bi}-end`);});
  return <div className="listing-text">{out}</div>;
}
// Maps the parsed HtmlNode tree to React elements. No attributes on any element.
function renderHtmlNodes(nodes:HtmlNode[],prefix:string):React.ReactNode[]{
  return nodes.flatMap((n,i)=>{
    const k=`${prefix}-${i}`;
    if(n.type==='text') return n.text;
    const ch=renderHtmlNodes(n.children,k);
    switch(n.tag){
      case 'br':     return <br key={k}/>;
      case 'strong': return <strong key={k}>{ch}</strong>;
      case 'em':     return <em key={k}>{ch}</em>;
      case 'p':      return <p key={k}>{ch}</p>;
      case 'h2':     return <h2 key={k}>{ch}</h2>;
      case 'h3':     return <h3 key={k}>{ch}</h3>;
      case 'h4':     return <h4 key={k}>{ch}</h4>;
      case 'ul':     return <ul key={k}>{ch}</ul>;
      case 'ol':     return <ol key={k}>{ch}</ol>;
      case 'li':     return <li key={k}>{ch}</li>;
    }
  });
}
function HtmlListingText({html}:{html:string}){
  const nodes=parseListingHtml(html);
  return <div className="listing-text">{renderHtmlNodes(nodes,'r')}</div>;
}

type FactIconName='money'|'mode'|'pin'|'case'|'level'|'clock';
const FACT_PATHS:Record<FactIconName,React.ReactNode>={
  money:<><circle cx="12" cy="12" r="8.5"/><path d="M14.6 9.2c-.5-.8-1.5-1.3-2.6-1.3-1.5 0-2.6.8-2.6 2s1.1 1.7 2.6 2 2.6.8 2.6 2-1.1 2-2.6 2c-1.1 0-2.1-.5-2.6-1.3M12 6.3v1.6M12 16.1v1.6"/></>,
  mode:<><path d="M4 11.5 12 5l8 6.5"/><path d="M6 10v9h12v-9"/><path d="M10 19v-5h4v5"/></>,
  pin:<><path d="M12 21s-6.5-5.6-6.5-11a6.5 6.5 0 0 1 13 0c0 5.4-6.5 11-6.5 11z"/><circle cx="12" cy="10" r="2.3"/></>,
  case:<><rect x="3.5" y="7.5" width="17" height="12" rx="2"/><path d="M9 7.5V6a1.5 1.5 0 0 1 1.5-1.5h3A1.5 1.5 0 0 1 15 6v1.5M3.5 12.5h17"/></>,
  level:<path d="M4 19h4v-4h4v-4h4V7h4"/>,
  clock:<><circle cx="12" cy="12" r="8.5"/><path d="M12 7.5v5l3 2"/></>,
};
function FactIcon({name}:{name:FactIconName}){return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">{FACT_PATHS[name]}</svg>;}
