import React,{useEffect,useRef,useState} from 'react';
import type {Job} from '../src/core';

export function ApplicationConfirmation({job,onClose,onConfirm}:{job:Job;onClose:()=>void;onConfirm:()=>Promise<boolean>}){
  const dialog=useRef<HTMLDialogElement>(null),[saving,setSaving]=useState(false),[failed,setFailed]=useState(false);
  useEffect(()=>{const previous=document.activeElement as HTMLElement|null,element=dialog.current;element?.showModal();return()=>{element?.close();if(previous?.isConnected)previous.focus();};},[]);
  return <dialog ref={dialog} className="application-confirmation" aria-labelledby="application-question" aria-describedby="application-explanation" onCancel={e=>{e.preventDefault();if(!saving)onClose();}} onClick={e=>{if(e.target===e.currentTarget&&!saving){const r=e.currentTarget.getBoundingClientRect();if(e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom)onClose();}}}>
    <h2 id="application-question">Did you submit your application?</h2><p>{job.title} · {job.company}</p><p id="application-explanation" className="muted">Opening the employer page does not submit an application. Confirm only after you have submitted it yourself. You can also confirm later from Pre-selected.</p>{failed&&<p className="error" role="alert">Confirmation could not be saved. The role has not been moved by this action. Close this dialog to review the error, or try again.</p>}<div className="confirmation-actions"><button autoFocus disabled={saving} onClick={onClose}>Not yet</button><button className="primary" disabled={saving} onClick={async()=>{setSaving(true);setFailed(false);try{if(await onConfirm())onClose();else setFailed(true);}finally{setSaving(false);}}}>{saving?'Saving…':'Yes, applied'}</button></div>
  </dialog>;
}
