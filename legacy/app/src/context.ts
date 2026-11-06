import { readFileSync, existsSync } from 'node:fs';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { load } from 'cheerio';
import { z } from 'zod';
import { hash, now, rootPath, yaml } from './core.js';
import { fetchPage } from './network.js';
import type { Store } from './store.js';
import { feedbackSources } from './feedback.js';
const exec = promisify(execFile);
export const candidateSchema = z.object({ version: z.literal(1), files: z.array(z.string()), profiles: z.array(z.object({ id: z.string(), url: z.string().url(), export: z.string().optional() })), cache_hours: z.number().positive(), source_roles: z.record(z.string(),z.object({category:z.string(),scope:z.string()})).default({}), review_notes: z.array(z.string()) });
export type EvidenceSource = { id: string; reference: string; retrievedAt: string; hash: string; status: string; text: string; error?: string; category?:string;scope?:string };
export type CandidateContext = { version: string; createdAt: string; preferences: unknown; reviewNotes: string[]; sources: EvidenceSource[] };
export function contextVersion(c:Pick<CandidateContext,'preferences'|'reviewNotes'|'sources'>) {
  return hash(JSON.stringify({preferences:c.preferences,reviewNotes:c.reviewNotes,sources:c.sources.map(({retrievedAt,error,text,...source})=>({...source,textHash:hash(text)}))}));
}
export function localContextFingerprint(preferences:unknown){const c=candidateSchema.parse(yaml('candidate'));return hash(JSON.stringify({config:c,preferences,files:[...c.files,...c.profiles.flatMap(p=>p.export?[p.export]:[])].map(p=>[p,existsSync(rootPath(p))?hash(readFileSync(rootPath(p))):'missing']),feedback:feedbackSources().map(s=>s.hash)}));}
export function cleanHtml(html: string): string {
  const $ = load(html); $('script,style,nav,footer,header,noscript,svg').remove();
  $('p,br,div,li,h1,h2,h3,section').prepend('\n');
  return $('body').text().replace(/[ \t]+/g,' ').replace(/\n\s*\n+/g,'\n\n').trim();
}
async function fileText(path: string) {
  const p = rootPath(path);
  if (path.endsWith('.pdf')) { const { stdout } = await exec('pdftotext', ['-layout',p,'-'], { timeout: 20000, maxBuffer: 8_000_000 }); return stdout; }
  return readFileSync(p,'utf8');
}
export async function refreshContext(store: Store, preferences: unknown, refreshRemote = false, localOnly=false): Promise<CandidateContext> {
  const c = candidateSchema.parse(yaml('candidate')), previous = store.context() as CandidateContext | null;
  const sources: EvidenceSource[] = [];
  for (const path of c.files) {
    try {
      const text = await fileText(path); if (text.trim().length < 30) throw new Error('No extractable text; provide text/OCR export');
      sources.push({ id: path, reference: path, retrievedAt: now(), hash: hash(readFileSync(rootPath(path))), status: 'available', text });
    } catch(e) { sources.push({ id: path, reference: path, retrievedAt: now(), hash: '', status: 'missing', text: '', error: String(e) }); }
  }
  for (const p of c.profiles) {
     const old = previous?.sources.find(s => s.id === p.id);
     if(localOnly&&!p.export){sources.push(old?(old.reference===p.url?old:{...old,reference:p.url,status:'stale',error:`Configured profile URL changed; retained text is from ${old.reference}. Refresh this profile explicitly.`}):{id:p.id,reference:p.url,retrievedAt:now(),hash:'',status:'missing',text:'',error:'Remote profile not collected; refresh explicitly'});continue;}
    if (!refreshRemote && !p.export && old?.text && Date.now() - Date.parse(old.retrievedAt) < c.cache_hours * 3600000) { sources.push(old); continue; }
    try {
      if (p.export && !existsSync(rootPath(p.export))) throw new Error(`Configured export missing: ${p.export}`);
      const text = p.export && existsSync(rootPath(p.export)) ? await fileText(p.export) : cleanHtml((await fetchPage(p.url, `profile:${p.id}`)).text);
      if (text.length < 100 || (p.id === 'linkedin' && /sign in|join linkedin/i.test(text) && !/experience/i.test(text))) throw new Error('Public profile incomplete; configure an owner-provided export');
      sources.push({ id: p.id, reference: p.export ?? p.url, retrievedAt: now(), hash: hash(text), status: 'available', text });
    } catch(e) {
      sources.push({ id: p.id, reference: p.url, retrievedAt: old?.retrievedAt ?? now(), hash: old?.hash ?? '', text: old?.text ?? '', status: old?.text ? 'stale' : 'missing', error: String(e) });
    }
  }
  const payload = { preferences, reviewNotes: c.review_notes, sources:[...sources.map(s=>({...s,...(c.source_roles[s.id]??{category:'public-profile',scope:'Public source snapshot; unverified statements remain attributed'})})),...feedbackSources()] };
  const context = { ...payload, version: contextVersion(payload), createdAt: now() };
  store.saveContext(context); return context;
}
// Keep the complete context in storage/preview. Select evidence explicitly within the model budget.
export function contextForJob(c: CandidateContext, jobText: string, budget: number) {
  const terms = new Set(jobText.toLowerCase().match(/[a-z][a-z0-9+.#-]{2,}/g) ?? []);
  const headers = JSON.stringify({ preferences: c.preferences, reviewNotes: c.reviewNotes, sourceStatus: c.sources.map(s => ({ id: s.id, status: s.status, error: s.error,category:s.category,scope:s.scope })) });
  const available = c.sources.filter(s => s.text);
  if (budget - headers.length < available.length * 400) throw new Error('Context budget too small for source overview; increase context_chars');
  let left = budget - headers.length;
  const selected: { id: string; text: string; totalChars: number; omittedChars: number }[] = [];
  for (let i = 0; i < available.length; i++) {
    const s = available[i], allowance = Math.floor(left / (available.length - i));
    const paragraphs = s.text.split(/\n\s*\n/).filter(Boolean);
    const ranked = paragraphs.map((text,index) => ({text,index, score: [...terms].filter(t => text.toLowerCase().includes(t)).length})).sort((a,b) => b.score-a.score);
    const picked: typeof ranked = []; let used = 0;
    for (const p of ranked) if (used + p.text.length + 2 <= allowance) { picked.push(p); used += p.text.length + 2; }
    if (!picked.length && s.text.length > allowance) throw new Error(`Unsegmentable source ${s.id}; budget too small, no silent truncation`);
    const text = picked.sort((a,b) => a.index-b.index).map(p => p.text).join('\n\n');
    selected.push({ id: s.id, text, totalChars: s.text.length, omittedChars: Math.max(0,s.text.length-text.length) }); left -= text.length;
  }
  return { overview: JSON.parse(headers), sources: selected };
}
