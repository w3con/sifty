import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { dirname, resolve, relative, isAbsolute } from 'node:path';
import { mkdirSync, readFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import YAML from 'yaml';
import { z } from 'zod';

export const ROOT = execFileSync('git', ['rev-parse', '--show-toplevel'], {
  cwd: resolve(dirname(fileURLToPath(import.meta.url)), '../..'), encoding: 'utf8'
}).trim();
export const hash = (s: string | Buffer) => createHash('sha256').update(s).digest('hex');
export const now = () => new Date().toISOString();
export function rootPath(p: string) {
  const result = resolve(ROOT, p);
  if (isAbsolute(p) || relative(ROOT, result).startsWith('..')) throw new Error('Path must be relative to repository root');
  return result;
}
export function localPath(p = '') {
  const base = process.env.FCTO_STATE_DIR ?? '.local';
  if (base !== '.local' && !base.startsWith('.local/')) throw new Error('FCTO_STATE_DIR must be under .local/');
  const d = rootPath(base); mkdirSync(d, { recursive: true, mode: 0o700 });
  const result = resolve(d,p); if (relative(d,result).startsWith('..')) throw new Error('Runtime path escapes state directory');
  return result;
}
export function yaml(name: string): unknown { return YAML.parse(readFileSync(rootPath(`config/${name}.yaml`), 'utf8')); }
const httpUrl = z.string().url().refine(s => /^https?:\/\//.test(s), 'Use HTTP(S) URL');
export const searchSchema = z.object({
  version: z.literal(1),
  sites: z.array(z.object({ name: z.string().min(1), url: httpUrl, enabled: z.boolean().default(true) }).passthrough()).min(1),
  search: z.object({ titles: z.array(z.string().min(1)), keywords: z.array(z.string().min(1)) }),
  preferences: z.record(z.string(), z.unknown())
}).superRefine((c, ctx) => {
  if (new Set(c.sites.map(s => s.name)).size !== c.sites.length) ctx.addIssue({ code: 'custom', message: 'Source names must be unique' });
});
export type Source = z.infer<typeof searchSchema>['sites'][number];
export const runtimeSchema = z.object({ version: z.literal(1), port: z.number().int().min(1024).max(65535), max_pages: z.number().int().min(1).max(50), max_jobs_per_source: z.number().int().min(1).max(2000), max_reviewed_per_source:z.number().int().min(1).max(5000).default(1200),max_detail_requests:z.number().int().min(0).max(500).default(100),detail_concurrency:z.number().int().min(1).max(8).default(4),max_query_groups:z.number().int().min(1).max(16).default(8), request_timeout_ms: z.number().min(100).max(120000), max_assessments_per_run: z.number().int().min(0).max(100), model_extraction: z.boolean().default(false), daily_hour: z.number().int().min(0).max(23), daily_minute: z.number().int().min(0).max(59) });
export const providerSchema = z.object({ type: z.enum(['opencode', 'claude', 'ollama', 'compatible']), model: z.string().min(1), endpoint: httpUrl.optional(), api_key_env: z.string().regex(/^[A-Z_][A-Z0-9_]*$/).optional(), timeout_ms: z.number().min(50).max(600000), context_chars: z.number().int().min(1000).max(500000) });
export type Provider = z.infer<typeof providerSchema>;
export function providers() {
  const c = z.object({ version: z.literal(1), stages: z.object({ assessment: z.string(), extraction: z.string(), preparation: z.string().default('') }), providers: z.record(z.string(), providerSchema) }).parse(yaml('providers'));
  for (const [k,n] of Object.entries(c.stages)) if (n && !c.providers[n]) throw new Error(`Unknown provider: ${n} (stage ${k})`);
  return c;
}
export const searchConfig = () => searchSchema.parse(yaml('search'));
export const runtime = () => runtimeSchema.parse(yaml('runtime'));
export function normalize(s: string): string {
  return s.toLowerCase().replace(/node\.?js/g, 'nodejs').replace(/web3\.?js/g, 'web3js').replace(/\blayer\s*2\b/g, 'l2').replace(/\bchief technology officer\b/g, 'cto').replace(/\btechnical lead\b/g, 'tech lead').replace(/\bartificial intelligence\b/g, 'ai').replace(/[^\p{L}\p{N}]+/gu, ' ').trim();
}
export function matchQueries(text: string, c = searchConfig()): string[] {
  const hay = ` ${normalize(text)} `;
  return [...c.search.titles, ...c.search.keywords].filter(q => hay.includes(` ${normalize(q)} `));
}
export function canonical(url: string): string {
  const u = new URL(url); if (!/^https?:$/.test(u.protocol)) throw new Error('Not an HTTP URL');
  u.hash = ''; u.hostname = u.hostname.replace(/^www\./, '');
  for (const k of [...u.searchParams.keys()]) if (/^(utm_|ref$|source$|gh_src$)/i.test(k)) u.searchParams.delete(k);
  u.searchParams.sort(); u.pathname = u.pathname.replace(/\/$/, '') || '/'; return u.toString();
}
export type JobInput = { title: string; company: string; boardCompany?:string; tags?:string[]; originalUrl: string; applyUrl?: string | null; applyStatus?: string; description: string; descriptionStatus: string; detailRetryAt?:number; location?: string; publishedAt?: string; expiresAt?: string; source: string };
export type StopWord = { id: number; word: string; createdAt: string };
export type BlockRule = { id: number; kind: string; value: string; createdAt: string };
export type Job = JobInput & { id: number; status: string; stage: import('./workflow.js').Stage; seen: boolean; seenAt: string|null; appliedAt: string|null; isWeb3: boolean; firstSeen: string; lastSeen: string; expired: boolean; matchScore: number; matchProfileRevision: number; matchStale: boolean; assessment: any; assessmentError: string | null; sources: { name: string; url: string; seen: string }[]; blocked: boolean; blockedBy: string; stopped: boolean; stoppedBy: string };
export const assessmentSchema = z.object({
  score: z.number().min(0).max(100), summary: z.string().min(1).max(4000),
  evidence: z.array(z.object({ claim: z.string().max(1000), sourceId: z.string(), quote: z.string().min(5).max(800) })).max(15),
  gaps: z.array(z.string().max(1500)).max(20).default([]),
  confirmedMismatches: z.array(z.string().max(1500)).max(20).default([]),
  candidateUnknowns: z.array(z.string().max(1500)).max(20).default([]),
  employerUnknowns: z.array(z.string().max(1500)).max(20).default([])
}).strict();
