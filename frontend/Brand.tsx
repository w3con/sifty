import React,{useEffect,useState} from 'react';
import {CELLS} from './Logo';
import {api} from './api';

// Sifty brand: the cell mark plus the Tiny5 wordmark at its natural width, under one
// left-to-right gradient in the mark's colours. While a collection run is active a light wave
// passes over mark and word, with a pause between passes.
const CAP=18;                         // cap height = mark height
const MARK_W=CAP*97/47, GAP=10;
const SIZE=CAP*1024/640;              // Tiny5: cap 640 / 1024 units
const ADV=[640,256,640,512,512];      // S I F T Y advances (units)
const SWEEP=1.8, PAUSE=1.4;           // seconds

function useCollectionRunning(){
  const [running,setRunning]=useState(false);
  useEffect(()=>{let alive=true,timer:number|undefined;
    const poll=async()=>{if(document.visibilityState==='visible'){try{const r=await api('collection');if(alive)setRunning(r?.status==='running');}catch{/* header stays still */}}
      if(alive)timer=window.setTimeout(poll,running?2000:5000);};
    void poll();return()=>{alive=false;window.clearTimeout(timer);};},[running]);
  return running;
}

export function Brand(){
  const running=useCollectionRunning();
  let x=0;const letters='SIFTY'.split('').map((ch,i)=>{const at=x;x+=ADV[i]*SIZE/1024;return {ch,at};});
  const wordW=Math.ceil(x-128*SIZE/1024)+1;
  const W=Math.ceil(MARK_W+GAP+wordW), H=CAP+1, wx=MARK_W+GAP, end=W+70, dur=SWEEP+PAUSE;
  const cells=(white:boolean)=><g transform={`scale(${CAP/47})`}>{CELLS.map(([cx,cy,w,c])=><rect key={`${cx}-${cy}`} x={cx} y={cy} width={w} height={7} rx={0.6} fill={white?'#fff':c}/>)}</g>;
  const word=<g transform={`translate(${wx},0)`}>{letters.map(l=><text key={l.ch} x={0} y={CAP} fontFamily="Tiny5" fontSize={SIZE} fill="#fff" transform={`translate(${l.at},0)`}>{l.ch}</text>)}</g>;
  return <div className={`brand${running?' working':''}`} role="img" aria-label={running?'Sifty, collecting jobs':'Sifty'} title={running?'Collecting jobs…':undefined}>
    <svg width={W} height={H} viewBox={`0 0 ${W} ${H}`} aria-hidden="true" style={{display:'block',overflow:'visible'}}>
      <defs>
        <linearGradient id="brand-word" gradientUnits="userSpaceOnUse" x1={wx} x2={wx+wordW} y1="0" y2="0"><stop offset="0" stopColor="#b2b04f"/><stop offset=".3" stopColor="#3db328"/><stop offset=".55" stopColor="#22aa6e"/><stop offset=".78" stopColor="#258f99"/><stop offset="1" stopColor="#3a95d0"/></linearGradient>
        <linearGradient id="brand-sheen" gradientUnits="userSpaceOnUse" x1="-70" x2="0" y1="0" y2="0"><stop offset="0" stopColor="#fff" stopOpacity="0"/><stop offset=".5" stopColor="#fff" stopOpacity=".8"/><stop offset="1" stopColor="#fff" stopOpacity="0"/>
          {running&&<animateTransform attributeName="gradientTransform" type="translate" values={`0 0;${end} 0;${end} 0`} keyTimes={`0;${(SWEEP/dur).toFixed(3)};1`} dur={`${dur}s`} repeatCount="indefinite"/>}</linearGradient>
        <mask id="brand-letters" maskUnits="userSpaceOnUse" x="0" y="0" width={W} height={H}>{word}</mask>
        <mask id="brand-all" maskUnits="userSpaceOnUse" x="0" y="0" width={W} height={H}>{cells(true)}{word}</mask>
      </defs>
      {cells(false)}
      <rect x={wx} y="0" width={wordW} height={H} fill="url(#brand-word)" mask="url(#brand-letters)"/>
      {running&&<rect className="brand-sheen" x="0" y="0" width={W} height={H} fill="url(#brand-sheen)" mask="url(#brand-all)"/>}
    </svg>
  </div>;
}
