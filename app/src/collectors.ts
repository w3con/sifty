import { load } from 'cheerio';
import { canonical, matchQueries, runtime, type JobInput, type Source } from './core.js';
import { cleanHtml } from './context.js';
import { fetchPage, detailPage, type Page } from './network.js';
import type { Store } from './store.js';
import { extractWithModel } from './extraction.js';

type Fetcher = (url: string, source?: string) => Promise<Page>;
export type Collection = { jobs: JobInput[]; status: string; message: string };
const abs = (u: string | undefined, base: string): string | null => { try { if (!u) return null; const v = new URL(u,base); return /^https?:$/.test(v.protocol) ? v.href : null; } catch { return null; } };
const text = (s: string) => cleanHtml(`<body>${s}</body>`);
export function structuredJobs(html: string, url: string, source: string): JobInput[] {
  const $ = load(html), jobs: JobInput[] = [];
  function walk(o: any) {
    if (!o || typeof o !== 'object') return;
    if (Array.isArray(o)) { o.forEach(walk); return; }
    if ([o['@type']].flat().includes('JobPosting')) {
      const original = abs(typeof o.url === 'string' ? o.url : undefined,url) ?? url;
      const company = typeof o.hiringOrganization === 'string' ? o.hiringOrganization : o.hiringOrganization?.name;
      const locations=[o.jobLocationType,...[o.applicantLocationRequirements??[]].flat().map((v:any)=>v.name),...[o.jobLocation??[]].flat().flatMap((v:any)=>[v.address?.addressLocality,v.address?.addressRegion,typeof v.address?.addressCountry==='string'?v.address.addressCountry:v.address?.addressCountry?.name])].filter(Boolean);
      if (o.title && company) jobs.push({ title:text(o.title),company:text(company),originalUrl:original,applyUrl:null,description:text(o.description ?? ''),descriptionStatus:o.description ? 'available structured description' : 'missing',location:[...new Set(locations)].join(' · '),publishedAt:o.datePosted,expiresAt:o.validThrough,source });
    }
    for (const [k,v] of Object.entries(o)) if (k !== 'description') walk(v);
  }
  $('script[type="application/ld+json"]').each((_,el) => { try { walk(JSON.parse($(el).text())); } catch { /* malformed schema is not authoritative */ } });
  return jobs;
}
export function detailFromHtml(html: string, url: string, source: string): JobInput {
  const $ = load(html), structured = structuredJobs(html,url,source);
  const exact=structured.filter(j=>canonical(j.originalUrl)===canonical(url));
  if(structured.length>1&&exact.length!==1)throw new Error('Directory returned multiple postings without an exact role URL; no job description selected');
  let job = exact[0]??structured[0];
  if (!job) {
    const title = $('h1').first().text().trim();
    if (!title) throw new Error('No job title/JobPosting found; unsupported page or login required');
    const body = $('[itemprop="description"],.job-description,#job-description,.job_description,[data-testid="job-description"],.job__description').first();
    const company = $('[itemprop="hiringOrganization"] [itemprop="name"], [itemprop="hiringOrganization"],.company-name,[data-testid="company-name"]').first().text().trim() || 'Unknown';
    job = { title,company,originalUrl:url,applyUrl:null,description:body.length ? text(body.html() ?? '') : '',descriptionStatus:body.length ? 'available page description' : 'missing: no verified description container',source };
  }
  // Only inspect link destinations. Never click a form or POST to resolve Apply.
  const candidates: string[] = [];
  $('a[href]').each((_,el) => {
    const a = $(el), href = abs(a.attr('href'),url); if (!href) return;
    if (/apply|application|postuler/i.test(a.text()) && canonical(href) !== canonical(url)) candidates.push(href);
  });
  const ats = candidates.find(h => /(?:jobs\.ashbyhq\.com|boards\.greenhouse\.io|job-boards\.greenhouse\.io|jobs\.lever\.co|apply\.workable\.com|myworkdayjobs\.com)/i.test(new URL(h).hostname));
  const external = candidates.find(h => new URL(h).hostname !== new URL(url).hostname && !/linkedin|wellfound|weworkremotely|cryptojobslist|remotive|twitter|facebook/i.test(new URL(h).hostname));
  job.applyUrl = ats ?? external ?? null;
  return job;
}
export function parseFeed(xml: string, url: string, source: string): JobInput[] {
  const $ = load(xml,{ xmlMode:true }), result: JobInput[] = [];
  $('item,entry').each((_,el) => {
    const e = $(el), title = text(e.find('title').first().text());
    const link = abs(e.find('link').first().attr('href') ?? e.find('link').first().text(),url);
    if (!title || !link) return;
    const split = title.includes(':') ? title.split(':') : title.split(' at ');
    const company = split.length > 1 ? (title.includes(':') ? split[0] : split.slice(1).join(' at ')).trim() : 'Unknown';
    const role = split.length > 1 ? (title.includes(':') ? split.slice(1).join(':') : split[0]).trim() : title;
    const description = text(e.find('description,content,content\\:encoded,summary').map((_,v)=>$(v).text()).get().sort((a,b)=>b.length-a.length)[0]??'');
    result.push({ title:role,company,originalUrl:link,applyUrl:null,description,descriptionStatus:'feed description; may be excerpt',publishedAt:e.find('pubDate,published,updated').first().text() || undefined,source });
  });
  return result;
}
export function htmlLinks(html: string, base: string, source: string): JobInput[] {
  const $ = load(html), jobs: JobInput[] = [];
  // Getro's public hydration payload provides explicit employer URLs and skills without guessing endpoints.
  try {
    const data = JSON.parse($('#__NEXT_DATA__').text());
    const found = data.props?.pageProps?.initialState?.jobs?.found;
    if (Array.isArray(found) && found.length) return found.filter(j => j.title && j.organization?.name && j.slug).map(j => ({title:j.title,company:j.organization.name,originalUrl:new URL(`/companies/${j.organization.slug}/jobs/${j.slug}`,base).href,applyUrl:abs(j.url,base),description:(j.skills ?? []).join(', '),descriptionStatus:'skills metadata only; full description pending',location:(j.locations ?? []).join(' · '),publishedAt:j.createdAt ? new Date(j.createdAt*1000).toISOString() : undefined,source}));
  } catch { /* Other boards do not use this payload. */ }
  $('a[href]').each((_,el) => {
    const a = $(el), url = abs(a.attr('href'),base); if (!url) return;
    const u = new URL(url), label = a.text().replace(/\s+/g,' ').trim();
    if (!label || label.length > 400 || /^(apply|view|learn|see all|jobs|careers|next|previous)/i.test(label)) return;
    const relevantPath = /\/jobs?\/(?:[^/]+\/)?[^/?]+/.test(u.pathname) || /-jobs\/[^/]+/.test(u.pathname) || /jobs\.(ashbyhq|lever)\.(com|co)/.test(u.hostname) || /greenhouse\.io/.test(u.hostname) || (/cryptocurrencyjobs\.co$/.test(u.hostname) && /^\/(engineering|operations|product|design|sales|marketing|finance|customer-support)\/[^/]+\/?$/.test(u.pathname));
    if (!relevantPath || /\/jobs\/(?:search|categories|new|remote|rss|feed)$/.test(u.pathname)) return;
    const container = a.closest('[itemtype*="JobPosting"], article,li,.job-list-item,.job-listing,[data-testid="job-card"],[data-testid="job-list-item"],.job').first();
    const title = a.attr('aria-label')?.replace(/^.*? – /,'') || a.find('h2,h3,[itemprop="title"],.job-title').first().text().trim() || label;
    const company = container.find('[itemprop="hiringOrganization"],.company,.company-name,[data-testid="company-name"]').first().text().trim() || 'Unknown';
    jobs.push({ title,company,originalUrl:url,applyUrl:null,description:'',descriptionStatus:'pending detail fetch',source });
  });
  return [...new Map(jobs.map(j => [canonical(j.originalUrl),j])).values()];
}
export function sourceCapability(s: Source): string {
  const h = new URL(s.url).hostname;
  if (/jobs\.(electriccapital|multicoin|framework)/.test(h))return 'Paginated public portfolio search API';
  if (/usebraintrust/.test(h))return 'Paginated public Braintrust API and job descriptions';
  if (/a16zfrontier/.test(h))return 'Entire embedded public portfolio dataset';
  if (/wellfound/.test(h))return 'Global public role searches with full descriptions and pagination';
  if (/web3\.careers?/.test(h))return 'Verified singular web3.career role/skill categories and pagination';
  if (/malt\.|cryptojobslist|welcometothejungle/.test(h))return 'Public search and dedicated browser access probe; live blockers reported per run';
  if (/cryptocurrencyjobs/.test(h))return 'Observed public search API; entire current curated window';
  if (/remote3|weworkremotely|remotive|paradigm|news\.ycombinator/.test(h))return 'Public feed/API/dataset with measured window coverage';
  return 'unsupported: unknown source; public adapter requires verification';
}
export async function collectSource(source: Source, get?: Fetcher): Promise<Collection> {
  return (await import('./discovery.js')).discoverSource(source,{get});
}
export async function importUrl(store: Store, url: string, source = 'Manual import', get: Fetcher = detailPage): Promise<number> {
  const p = await get(url,source), j = await extractedDetail(p.text,p.url,source,store);
  j.originalUrl = url;
  if (/jobs\.(ashbyhq\.com|lever\.co)|greenhouse\.io/.test(new URL(p.url).hostname)) {j.applyUrl = p.url;j.applyStatus='reachable employer ATS (GET verified; no form interaction)';}
  else await resolveApply(j,get);
  return store.upsert(j,{restore:true});
}
async function extractedDetail(html:string,url:string,source:string,store?:Store) {
  let j: JobInput;
  try {j=detailFromHtml(html,url,source);} catch(e) {if(!runtime().model_extraction)throw e;return extractWithModel(html,url,source,undefined,store);}
  if(!j.description&&runtime().model_extraction) {const extracted=await extractWithModel(html,url,source,undefined,store);return {...extracted,applyUrl:j.applyUrl};}
  return j;
}
async function resolveApply(j: JobInput,get: Fetcher,depth=0) {
  if (!j.applyUrl) {j.applyStatus='not resolved: open original listing for manual application';return;}
  try {
    const target = await get(j.applyUrl,j.source);
    const $=load(target.text);
    const board=/linkedin\.com|wellfound\.com|weworkremotely\.com|cryptojobslist\.com|remotive\.com/.test(new URL(target.url).hostname) || !!$('#__NEXT_DATA__').text().match(/"network":/);
    if (board) {
      let next:string|null|undefined;try{next=detailFromHtml(target.text,target.url,j.source).applyUrl;}catch{/* No parseable employer destination. */}
      if(depth<2&&next&&canonical(next)!==canonical(target.url)){j.applyUrl=next;await resolveApply(j,get,depth+1);return;}
      j.applyUrl=null;j.applyStatus='board intermediary/restricted destination, not a resolved employer form';return;
    }
    j.applyUrl=target.url;j.applyStatus='reachable source-provided application destination (GET verified; employer identity unverified)';
  } catch(e) {j.applyStatus=`unverified application destination: ${String(e)}`;}
}
export async function enrich(store: Store, id: number, get: Fetcher = detailPage): Promise<number> {
  const job = store.get(id), source = job.sources[0]?.name ?? 'Manual import';
  id=job.id;
  const save=(j:JobInput)=>{store.get(id);const saved=store.upsert(j);if(!saved)throw Error('Role was deleted; automatic restoration is disabled');return saved;};
  if(/app\.usebraintrust\.com/.test(new URL(job.originalUrl).hostname)){const {hydrateCandidate}=await import('./discovery.js');return save(await hydrateCandidate({...job,source},get));}
  if (/news\.ycombinator/.test(new URL(job.originalUrl).hostname)) {job.source=source;await resolveApply(job,get);return save(job);}
  async function employerDetail() {
    if (!job.applyUrl || canonical(job.applyUrl) === canonical(job.originalUrl)) throw new Error('No distinct employer destination');
    const p=await get(job.applyUrl,source), d=await extractedDetail(p.text,p.url,source,store);
    if (!d.description) throw new Error('Employer page has no extractable description');
    const result={...d,company:d.company==='Unknown'?job.company:d.company,originalUrl:job.originalUrl,applyUrl:d.applyUrl??p.url,applyStatus:'reachable source-provided employer description (GET verified; no form interaction)'};
    await resolveApply(result,get);return save(result);
  }
  try {
    const p = await get(job.originalUrl,source), detail = await extractedDetail(p.text,p.url,source,store);
    const j: JobInput = { ...detail,title:detail.title || job.title,company:detail.company === 'Unknown' ? job.company : detail.company,originalUrl:job.originalUrl,description:detail.description || job.description,descriptionStatus:detail.description ? detail.descriptionStatus : job.descriptionStatus,applyUrl:detail.applyUrl ?? job.applyUrl };
    if (/jobs\.(ashbyhq\.com|lever\.co)|greenhouse\.io/.test(new URL(p.url).hostname)) {j.applyUrl=p.url;j.applyStatus='reachable employer ATS (GET verified; no form interaction)';}
    else await resolveApply(j,get);
    if (!detail.description && job.applyUrl) {try {return await employerDetail();}catch {/* Preserve available board/feed text and explicit incomplete status. */}}
    return save(j);
  } catch(e) {
    if (job.applyUrl) {try {return await employerDetail();}catch {/* Both sources unavailable; retain the original error below. */}}
    store.get(id);const msg = String(e), status = /HTTP (404|410)/.test(msg) ? 'expired/unavailable (HTTP 404/410)' : `detail unavailable: ${msg}`;
    store.db.prepare('UPDATE jobs SET description_status=?, expires_at=CASE WHEN ? THEN ? ELSE expires_at END WHERE id=?').run(status,Number(/HTTP (404|410)/.test(msg)),new Date(Date.now()-1).toISOString(),id);
    return id;
  }
}
