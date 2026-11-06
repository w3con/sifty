#!/usr/bin/env -S npx tsx
import * as fs from 'node:fs';
import path from 'node:path';
import net from 'node:net';
import { fileURLToPath } from 'node:url';
import YAML from 'yaml';
import { ROOT, providerSchema } from '../../src/core.js';
import { infer } from '../../src/providers.js';
// @ts-ignore Node-only module shared with the standalone restore.
import { databaseSummary, FORMAT, digest } from './transfer-lib.mjs';

export function loopbackEndpoint(value: string) {
  const u = new URL(value);
  if (u.protocol !== 'http:' || !['127.0.0.1', '[::1]'].includes(u.hostname) || u.username || u.password || u.pathname !== '/' || u.search || u.hash) throw Error('Use a literal loopback HTTP endpoint, e.g. http://127.0.0.1:11434');
  return u.origin;
}
export async function installedModels(endpoint: string): Promise<string[]> {
  endpoint = loopbackEndpoint(endpoint);
  const r = await fetch(`${endpoint}/api/tags`, { signal: AbortSignal.timeout(3000), redirect: 'error' });
  if (!r.ok) throw Error(`Ollama discovery HTTP ${r.status}`);
  let text = '';
  if (!r.body) throw Error('Missing Ollama response');
  const reader = r.body.getReader();
  try {
    for (;;) { const { done, value } = await reader.read(); if (done) break; text += Buffer.from(value).toString('utf8'); if (text.length > 1_000_000) throw Error('Ollama tags response too large'); }
  } finally { await reader.cancel(); }
  const body = JSON.parse(text);
  if (!Array.isArray(body.models) || body.models.some((m: any) => typeof m.name !== 'string' || !m.name || m.name.length > 500)) throw Error('Malformed Ollama model list');
  return [...new Set<string>(body.models.map((m: any) => m.name))];
}
async function reservePort(port: number) {
  const server = net.createServer();
  await new Promise<void>((resolve, reject) => { server.once('error', () => reject(Error(`Stop the catalogue/backend first (port ${port} is in use)`))); server.listen({ port, host: '127.0.0.1', exclusive: true }, resolve); });
  return () => new Promise<void>((resolve, reject) => server.close(e => e ? reject(e) : resolve()));
}
export async function configureOllama(root: string, { model, endpoint = 'http://127.0.0.1:11434' }: { model?: string; endpoint?: string } = {}) {
  endpoint = loopbackEndpoint(endpoint);
  const receipt = path.join(root, '.local/transfer-receipt.json');
  if (!fs.existsSync(receipt) || JSON.parse(fs.readFileSync(receipt, 'utf8')).format !== FORMAT) throw Error('Run this helper only in a restored target checkout (restore receipt required)');
  const models = await installedModels(endpoint);
  if (!model) return { models };
  if (!models.includes(model)) throw Error(`Model is not installed: ${model}. Use --list and select an exact installed ID; nothing was changed`);
  const configPath = path.join(root, 'config/providers.yaml');
  if (fs.lstatSync(configPath).isSymbolicLink()) throw Error('Provider config symlink refused');
  const original = fs.readFileSync(configPath), document = YAML.parseDocument(original.toString());
  if (document.errors.length) throw Error('Invalid provider YAML');
  const p = providerSchema.parse(document.toJS().providers?.ollama);
  if (p.type !== 'ollama') throw Error('Existing ollama provider must use the local adapter');
  const runtime = YAML.parse(fs.readFileSync(path.join(root, 'config/runtime.yaml'), 'utf8'));
  if (!Number.isInteger(runtime.port) || runtime.port < 1024 || runtime.port > 65535) throw Error('Invalid catalogue port');
  const release = await reservePort(runtime.port);
  const lock = path.join(root, '.local/ollama-setup.lock');
  let ownsLock = false, temp: string | undefined;
  try {
    const fd = fs.openSync(lock, 'wx', 0o600); fs.closeSync(fd); ownsLock = true;
    const summaries = () => ['jobs', 'network'].map(kind => databaseSummary(path.join(root, `.local/${kind}.sqlite`), kind));
    const before = JSON.stringify(summaries());
    const schema = { type: 'object', additionalProperties: false, required: ['ok', 'value'], properties: { ok: { type: 'boolean', const: true }, value: { type: 'integer', const: 7 } } };
    const result: any = await infer({ ...p, endpoint, model, timeout_ms: Math.min(p.timeout_ms, 120000) }, 'Synthetic connectivity test only. Return exactly {"ok":true,"value":7}. No candidate data. No tools.', schema);
    if (!result || typeof result !== 'object' || Array.isArray(result) || result.ok !== true || result.value !== 7 || Object.keys(result).sort().join(',') !== 'ok,value') throw Error('Ollama structured-output smoke failed; configuration unchanged');
    if (before !== JSON.stringify(summaries()) || !fs.readFileSync(configPath).equals(original)) throw Error('Target state/config changed during smoke; configuration unchanged');
    document.setIn(['providers', 'ollama', 'model'], model);
    document.setIn(['providers', 'ollama', 'endpoint'], endpoint);
    document.setIn(['stages', 'assessment'], 'ollama');
    document.setIn(['stages', 'extraction'], 'ollama');
    const mode = fs.statSync(configPath).mode & 0o777;
    const backup = `${configPath}.before-ollama-${Date.now()}-${digest(original).slice(0, 8)}.bak`;
    fs.writeFileSync(backup, original, { flag: 'wx', mode: 0o600 });
    temp = `${configPath}.ollama-${process.pid}.tmp`;
    fs.writeFileSync(temp, document.toString(), { flag: 'wx', mode });
    fs.renameSync(temp, configPath); temp = undefined;
    return { models, model, endpoint, backup };
  } finally {
    if (temp) fs.rmSync(temp, { force: true });
    if (ownsLock) fs.rmSync(lock, { force: true });
    await release();
  }
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const args = process.argv.slice(2); let model: string | undefined, endpoint: string | undefined;
    for (let i = 0; i < args.length; i++) {
      if (args[i] === '--list') continue;
      if (args[i] === '--model' && args[i + 1]) model = args[++i];
      else if (args[i] === '--endpoint' && args[i + 1]) endpoint = args[++i];
      else throw Error('Usage: npx tsx ops/transfer/ollama.ts --list | --model EXACT_ID [--endpoint http://127.0.0.1:PORT]');
    }
    const result = await configureOllama(ROOT, { model, endpoint });
    console.log(JSON.stringify(result, null, 2));
    if (model) console.log('Assessment/extraction now use Ollama; Assessment remains Off. Strong default drafting still needs separate OpenCode CLI login, or explicitly choose Ollama in Application > More actions. No automatic fallback.');
  } catch (e: any) { console.error(`Ollama setup refused: ${e.message}. Start the installed loopback runtime and use --list; no model is downloaded.`); process.exitCode = 1; }
}
