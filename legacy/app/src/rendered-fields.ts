// Serialized into each inspected frame. Keep all browser-only helpers in this function.
export function readRenderedFields(){
 const controls='input,textarea,select,[role=combobox],[role=radio],[role=checkbox]';
 const ignored='script,style,template,[hidden],[aria-hidden="true"],[inert]';
 const noise='input,textarea,select,button,[role=status],[role=alert],.application-field,.resume-upload-failure,.resume-upload-working,.resume-upload-success,.dropdown-container,[data-qa="privacy-link"]';
 const normalize=(text:string)=>text.replace(/\s+/g,' ').trim();
 const visible=(el:Element)=>{for(let node:Element|null=el;node;node=node.parentElement){const css=getComputedStyle(node);if(node.matches(ignored)||css.display==='none'||['hidden','collapse'].includes(css.visibility))return false;}return true;};
 // Walk rendered text rather than textContent/innerText on hidden nodes (which can
 // return hidden status text). Exclude controls/options and known UI, not whole-form prose.
 function text(el:Element|null,exclude=noise):string{
  if(!el||!visible(el))return '';const pieces:string[]=[];let length=0,visited=0;
  function walk(node:Node){if(++visited>4000||length>16000)return;if(node.nodeType===Node.TEXT_NODE){pieces.push(node.textContent??'');length+=(node.textContent??'').length;return;}if(!(node instanceof Element)||!visible(node)||node.matches(exclude))return;for(const child of node.childNodes)walk(child);pieces.push(' ');}
  walk(el);return normalize(pieces.join(''));
 }
 const rootElement=(el:Element,id:string)=>(el.getRootNode() as Document|ShadowRoot).getElementById(id);
 const refs=(el:Element,attribute:string)=>(el.getAttribute(attribute)??'').split(/\s+/).filter(Boolean).map(id=>text(rootElement(el,id))).filter(Boolean).join(' ');
 function nativeLabel(el:Element){
  const aria=refs(el,'aria-labelledby')||normalize(el.getAttribute('aria-label')??'');if(aria)return aria;
  return Array.from((el as HTMLInputElement).labels??[]).map(label=>{
   const members=Array.from(label.querySelectorAll(controls)).filter(c=>(c as HTMLInputElement).type!=='hidden');
   // An over-broad wrapping label cannot name several independent questions.
   if(members.length>1)return '';
   const markers=Array.from(label.querySelectorAll('.application-label')).filter(m=>m.closest('label')===label);
   return markers.length===1?text(markers[0]):text(label);
  }).filter(Boolean).join(' ')||(['radio','checkbox'].includes(el.getAttribute('role')??'')?text(el):'');
 }
 const generic=(label:string)=>!label||/^(?:type (?:your )?(?:response|answer)(?: here)?|enter (?:your )?(?:response|answer|text)(?: here)?|your (?:response|answer)|answer here|select(?: an? option)?|choose(?: an? option)?)[\s.!…]*$/i.test(label);
 function localQuestion(el:Element){
  const question=el.closest('.application-question');if(!question)return {label:'',scope:null as Element|null};
  for(let parent=el.parentElement,depth=0;parent&&depth<6;parent=parent.parentElement,depth++){
   if(!question.contains(parent))break;
   const labels=Array.from(parent.querySelectorAll('.application-label')).filter(label=>label.closest('.application-question')===question&&visible(label)&&text(label));
   const members=Array.from(parent.querySelectorAll(controls)).filter(c=>(c as HTMLInputElement).type!=='hidden'&&c.closest('.application-question')===question);
   const grouped=members.length>0&&members.every(c=>['radio','checkbox'].includes((c as HTMLInputElement).type||c.getAttribute('role')||'')&&(c as HTMLInputElement).name===(el as HTMLInputElement).name);
   if(labels.length===1&&(members.length===1||grouped))return {label:text(labels[0]),scope:parent};
   if(parent===question)break;
  }return {label:'',scope:question};
 }
 const elements:Element[]=[];function visit(root:Document|ShadowRoot){elements.push(...root.querySelectorAll(controls));for(const el of root.querySelectorAll('*'))if(el.shadowRoot)visit(el.shadowRoot);}visit(document);
 return elements.filter(el=>!el.closest('header,nav,footer,[role="navigation"],[role="search"]')&&!['hidden','submit','button','reset','password','search'].includes((el as HTMLInputElement).type)&&!(!el.closest('form,[role="dialog"]')&&/^(?:search\b|i[’']m looking for|select language$)/i.test(nativeLabel(el)||el.getAttribute('placeholder')||''))&&((el as HTMLInputElement).type==='file'||(el.getClientRects().length>0&&visible(el)))).map(el=>{
  const input=el as HTMLInputElement,role=el.getAttribute('role'),type=el.tagName==='SELECT'?'select':el.tagName==='TEXTAREA'?'textarea':role==='combobox'?'select':role==='radio'?'radio':role==='checkbox'?'checkbox':['radio','checkbox','file'].includes(input.type)?input.type:'text';
  // Preserve the previous identity fallback as well as every named/id control. Label
  // improvements change signatures for review, never the persisted field identifier.
  const legacy=normalize(Array.from(input.labels??[]).map(label=>label.textContent).join(' ')||el.getAttribute('aria-label')||(el.getAttribute('aria-labelledby')??'').split(' ').map(id=>document.getElementById(id)?.textContent??'').join(' ')||el.getAttribute('placeholder')||el.getAttribute('name')||el.id);
  const controlName=input.name||el.id||legacy,standard=nativeLabel(el),local=localQuestion(el),grouped=['radio','checkbox'].includes(type);
  const fieldset=el.closest('fieldset'),legend=fieldset?Array.from(fieldset.children).find(c=>c.tagName==='LEGEND')??null:null;
  const group=grouped?(text(legend)||local.label):'';
  const placeholder=normalize(el.getAttribute('placeholder')??''),raw=group||(!generic(standard)?standard:'')||local.label||(!generic(placeholder)?placeholder:'');
  const labelStatus=!raw||raw.length>8000?'uncertain':'resolved';
  const label=labelStatus==='resolved'?raw.replace(/\s*[*✱✳]+\s*$/,''):'Question wording unavailable — review the source';
  const options=type==='select'?Array.from((el as HTMLSelectElement).options??[]).map(o=>normalize(o.text)).filter(Boolean):grouped&&!generic(standard)?[standard.replace(/\s*[*✱✳]+\s*$/,'')]:[];
  const helperNodes=local.scope?Array.from(local.scope.querySelectorAll('.application-description,.description')).filter(n=>n.closest('.application-question')===el.closest('.application-question')):[];
  const instructions=[...new Set([refs(el,'aria-describedby'),...helperNodes.map(n=>text(n))].filter(t=>t&&t!==label))].join('\n');
  const warnings:string[]=[];if(labelStatus==='uncertain')warnings.push('Question wording could not be resolved from a visible associated label or supported local question container. Review the source or add the question manually; no answer will be generated for this control.');if(grouped&&!options.length)warnings.push('Visible option wording could not be resolved without interaction. Supply the choices manually.');if(instructions.length>12000)warnings.push('Helper instructions exceed the read bound; review their full wording on the source.');
  return {label,labelStatus,instructions:instructions.slice(0,12000),type,options,required:input.required||el.getAttribute('aria-required')==='true'||/[*✱✳]\s*$/.test(raw),maxLength:input.maxLength>0?input.maxLength:null,accept:input.accept??'',controlName,warnings};
 });
}
