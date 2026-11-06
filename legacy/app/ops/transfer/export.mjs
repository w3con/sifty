#!/usr/bin/env node
import * as fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomUUID } from 'node:crypto';
import YAML from 'yaml';
import { FORMAT, git, command, digest, safeRelative, entry, walk, copyEntry, databaseSummary, snapshot, originSafe, branchMetadata } from './transfer-lib.mjs';

// Deliberate denylist in addition to Git's ignore rules. Configured inputs are checked separately.
function secretPath(p) {
  return p.split('/').some(n => /^\.env(?:\.|$)/.test(n) || /^(credentials|secrets|oauth|tokens?)(\.|$)/i.test(n) || ['auth.json', '.npmrc', '.netrc', '.git-credentials'].includes(n) || /\.(pem|key|p12|pfx)$/i.test(n));
}
export function forbidden(p) {
  return secretPath(p) || p.split('/').some(n => ['.git', 'node_modules', 'dist', '.obsidian', '.DS_Store', '.ai', '.ssh', 'sessions', 'browser-profile'].includes(n)) || p.startsWith('.local/');
}
function sourcePaths(root) {
  return git(root, 'ls-files', '-z', '--cached', '--others', '--exclude-standard').split('\0').filter(Boolean).filter(p => {
    if (p === 'ai/shared') return false;
    try { fs.lstatSync(path.join(root, p)); return true; } catch (e) { if (e.code === 'ENOENT') return false; throw e; }
  }).sort();
}
const fileStamp = (root, p) => { const s = fs.lstatSync(path.join(root, p), { bigint: true }); return ['ino', 'size', 'mtimeNs', 'ctimeNs', 'mode'].map(k => String(s[k])).join(':'); };
function dbStamp(file) {
  return ['', '-wal'].map(suffix => {
    const p = file + suffix;
    if (!fs.existsSync(p)) return null;
    const s = fs.statSync(p);
    // SQLite may create an empty WAL when a read-only connection opens a checkpointed DB.
    if (suffix && s.size === 0) return null;
    return [s.size, s.mtimeMs, digest(fs.readFileSync(p))];
  });
}
export function exportArchive(root, { state = '.local', output = '.local/transfers' } = {}) {
  root = fs.realpathSync(root); safeRelative(state); safeRelative(output);
  if (!(state === '.local' || state.startsWith('.local/')) || !output.startsWith('.local/')) throw Error('State/output must be inside ignored .local');
  if (git(root, 'rev-parse', '--show-toplevel') !== root) throw Error('Export must use the project Git root');
  const out = path.join(root, output);
  // Do not follow symlink ancestors when creating private output.
  for (let p = out; p !== root; p = path.dirname(p)) if (fs.existsSync(p) && fs.lstatSync(p).isSymbolicLink()) throw Error('Symlink output refused');
  fs.mkdirSync(out, { recursive: true, mode: 0o700 }); fs.chmodSync(out, 0o700);
  const stage = fs.mkdtempSync(path.join(out, '.capture-'));
  let archive;
  try {
    const created = new Date().toISOString();
    const revision = git(root, 'rev-parse', 'HEAD'), subroot = path.join(root, 'ai/shared');
    const pin = git(root, 'ls-tree', 'HEAD', 'ai/shared').split(/\s+/)[2];
    if (!fs.existsSync(subroot) || git(subroot, 'rev-parse', '--show-toplevel') !== subroot || git(subroot, 'rev-parse', 'HEAD') !== pin) throw Error('Shared tooling checkout missing or not at pinned revision');
    const info = (dir, bundle) => ({ revision: git(dir, 'rev-parse', 'HEAD'), origin: originSafe(git(dir, 'remote', 'get-url', 'origin')), remotes: git(dir, 'remote', '-v').split('\n'), bundle });
    const repo = { ...info(root, 'payload/project.bundle'), ...branchMetadata(root) }, submodule = { ...info(subroot, 'payload/shared.bundle'), path: 'ai/shared' };
    // Validate every remote before putting metadata into the private manifest.
    for (const r of [repo, submodule]) for (const line of r.remotes) originSafe(line.split(/\s+/)[1]);
    const databases = {}, stamps = {};
    fs.mkdirSync(path.join(stage, 'payload/state'), { recursive: true });
    for (const kind of ['jobs', 'network']) {
      const file = path.join(root, state, `${kind}.sqlite`);
      entry(root, `${state}/${kind}.sqlite`);
      stamps[kind] = dbStamp(file);
      const before = databaseSummary(file, kind);
      snapshot(file, path.join(stage, `payload/state/${kind}.sqlite`));
      const after = databaseSummary(path.join(stage, `payload/state/${kind}.sqlite`), kind);
      if (JSON.stringify(before) !== JSON.stringify(after)) throw Error(`Database moved during snapshot: ${kind}`);
      databases[kind] = after;
    }
    fs.mkdirSync(path.join(stage, 'payload/tree'), { recursive: true });
    for (const dir of [root, subroot]) for (const p of git(dir, 'ls-files', '-z', '--cached').split('\0').filter(Boolean)) if (secretPath(p)) throw Error(`Tracked credential-like file would travel in Git history: ${p}; review it before transfer`);
    const paths = sourcePaths(root).filter(p => !forbidden(p));
    const subpaths = sourcePaths(subroot).filter(p => !forbidden(p));
    const tree = [...paths, ...subpaths.map(p => `ai/shared/${p}`)];
    // Owner-authored Markdown may be intentionally untracked/ignored.
    for (const dir of ['kb/Applications/answers', 'kb/Applications/candidate-feedback']) if (fs.existsSync(path.join(root, dir))) for (const p of walk(path.join(root, dir))) if (p.endsWith('.md')) tree.push(`${dir}/${p}`);
    const candidate = YAML.parse(fs.readFileSync(path.join(root, 'config/candidate.yaml'), 'utf8'));
    const inputs = [...(candidate.files ?? []), ...(candidate.profiles ?? []).flatMap(p => p.export ? [p.export] : [])];
    const runtime = new Set();
    for (const p of inputs) {
      safeRelative(p);
      if (secretPath(p)) throw Error(`Configured input excluded by credential policy: ${p}`);
      if (entry(root, p).type !== 'file') throw Error(`Configured input must be a regular file, not a symlink: ${p}; configure its included relative file path explicitly`);
      if (p.startsWith('.local/imports/')) runtime.add(p);
      else if (forbidden(p)) throw Error(`Configured input excluded by transfer policy: ${p}`);
      else tree.push(p);
    }
    // Preserve referenced technical records from current AND historical JSON, never profiles/log directories.
    collectReferences(root, state, runtime);
    const sourceStamps = new Map();
    const workingTree = [...new Set(tree)].sort().map(p => { sourceStamps.set(p, fileStamp(root, p)); return copyEntry(root, p, path.join(stage, 'payload/tree')); });
    const captured = [];
    fs.mkdirSync(path.join(stage, 'payload/state/files'), { recursive: true });
    for (const p of [...runtime].sort()) {
      if (!/^\.local\/(application-inspections|application-preparation-errors|imports)\//.test(p)) throw Error(`Unsupported configured/referenced runtime path: ${p}`);
      const source = p.startsWith('.local/imports/') ? p : state + p.slice('.local'.length);
      if (entry(root, source).type !== 'file') throw Error(`Runtime symlink refused: ${p}`);
      captured.push({ source, stamp: fileStamp(root, source), original: copyEntry(root, source, path.join(stage, 'payload/state/files'), p) });
    }
    git(root, 'bundle', 'create', path.join(stage, repo.bundle), '--all', 'HEAD');
    git(subroot, 'bundle', 'create', path.join(stage, submodule.bundle), '--all', 'HEAD');
    const tool = path.dirname(fileURLToPath(import.meta.url));
    for (const p of ['restore.mjs', 'restore.command', 'transfer-lib.mjs', 'TRANSFER.md']) copyEntry(tool, p, stage);
    fs.chmodSync(path.join(stage, 'restore.command'), 0o700);
    // Re-read all included source bytes/modes and DB/WAL bytes after the entire capture.
    for (const e of workingTree) if (fileStamp(root, e.path) !== sourceStamps.get(e.path) || JSON.stringify(entry(root, e.path)) !== JSON.stringify(e)) throw Error(`Source changed while capturing: ${e.path}`);
    for (const { source, original, stamp } of captured) if (fileStamp(root, source) !== stamp || JSON.stringify(entry(root, source)) !== JSON.stringify(original)) throw Error(`Input changed while capturing: ${source}`);
    for (const kind of ['jobs', 'network']) if (JSON.stringify(dbStamp(path.join(root, state, `${kind}.sqlite`))) !== JSON.stringify(stamps[kind]) || JSON.stringify(databaseSummary(path.join(root, state, `${kind}.sqlite`), kind)) !== JSON.stringify(databases[kind])) throw Error(`State moved while capturing: ${kind}`);
    if (git(root, 'rev-parse', 'HEAD') !== revision || JSON.stringify(branchMetadata(root)) !== JSON.stringify({ branch: repo.branch, upstream: repo.upstream }) || git(subroot, 'rev-parse', 'HEAD') !== pin || JSON.stringify(sourcePaths(root).filter(p => !forbidden(p))) !== JSON.stringify(paths) || JSON.stringify(sourcePaths(subroot).filter(p => !forbidden(p))) !== JSON.stringify(subpaths)) throw Error('Source Git/tree changed during capture');
    const manifest = { format: FORMAT, created, git: repo, submodule, workingTree, databases, configuredInputs: inputs, payload: walk(stage).map(p => entry(stage, p)) };
    fs.writeFileSync(path.join(stage, 'manifest.json'), JSON.stringify(manifest, null, 2), { mode: 0o600 });
    archive = path.join(out, `sift-transfer-v1-${created.replace(/[:.]/g, '-')}-${revision.slice(0, 8)}-${randomUUID().slice(0, 8)}.tar.gz`);
    command(stage, 'tar', ['-czf', archive, '.']); fs.chmodSync(archive, 0o600);
    const checksum = digest(fs.readFileSync(archive));
    fs.writeFileSync(`${archive}.sha256`, `${checksum}  ${path.basename(archive)}\n`, { mode: 0o600, flag: 'wx' });
    fs.writeFileSync(`${archive}.manifest.json`, JSON.stringify(manifest, null, 2), { mode: 0o600, flag: 'wx' });
    return { archive, checksum, manifest };
  } catch (e) {
    if (archive) for (const suffix of ['', '.sha256', '.manifest.json']) fs.rmSync(archive + suffix, { force: true });
    throw e;
  } finally { fs.rmSync(stage, { recursive: true, force: true }); }
}

import { openReadonly } from './transfer-lib.mjs';
function collectReferences(root, state, runtime) {
  const db = openReadonly(path.join(root, state, 'jobs.sqlite'));
  function visit(value, key = '') {
    if (typeof value === 'string') {
      if (key === 'technicalLog' || key === 'path' && value.startsWith('.local/')) { safeRelative(value); runtime.add(value); }
      else if (/^[\[{]/.test(value)) { try { visit(JSON.parse(value)); } catch (e) { if (!(e instanceof SyntaxError)) throw e; } }
    } else if (Array.isArray(value)) value.forEach(v => visit(v));
    else if (value && typeof value === 'object') for (const [k, v] of Object.entries(value)) visit(v, k);
  }
  try {
    for (const { name } of db.prepare("SELECT name FROM sqlite_master WHERE type='table'").all()) {
      const q = '"' + name.replaceAll('"', '""') + '"';
      const statement = db.prepare(`SELECT * FROM ${q}`); statement.setReadBigInts(true);
      for (const row of statement.all()) visit(row);
    }
    for (const row of db.prepare('SELECT id FROM preparation_operations').all()) {
      const p = `application-preparation-errors/${row.id}.json`;
      if (fs.existsSync(path.join(root, state, p))) runtime.add(`.local/${p}`);
    }
  } finally { db.close(); }
}
// CLI intentionally resolves from this source file rather than the caller's current directory.
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if (process.argv.slice(2).some(a => a !== '--frozen')) throw Error('Usage: node ops/transfer/export.mjs --frozen');
    if (!process.argv.includes('--frozen')) throw Error('Stop the old writer first, then pass --frozen');
    const root = git(path.dirname(fileURLToPath(import.meta.url)), 'rev-parse', '--show-toplevel');
    if (process.env.FCTO_STATE_DIR) throw Error('Unset FCTO_STATE_DIR for the final source export');
    const result = exportArchive(root); console.log(JSON.stringify({ archive: result.archive, sha256: result.checksum, counts: result.manifest.databases.jobs.applicationCounts }, null, 2));
  } catch (e) { console.error(`Export refused: ${e.message}`); process.exitCode = 1; }
}
