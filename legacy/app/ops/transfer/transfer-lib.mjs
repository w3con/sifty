// Node-only shared archive verification and restore. Never construct the application's Store.
import { DatabaseSync } from 'node:sqlite';
import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import * as fs from 'node:fs';
import path from 'node:path';

export const FORMAT = 'sift-development-transfer-v1';
export const digest = data => createHash('sha256').update(data).digest('hex');
const lexists = p => { try { fs.lstatSync(p); return true; } catch (e) { if (e.code === 'ENOENT') return false; throw e; } };
export function command(cwd, name, args) {
  return execFileSync(name, args, { cwd, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024, env: { ...process.env, COPYFILE_DISABLE: '1', GIT_CONFIG_NOSYSTEM: '1', GIT_CONFIG_GLOBAL: '/dev/null', GIT_TERMINAL_PROMPT: '0' } }).trim();
}
export const git = (cwd, ...args) => command(cwd, 'git', ['-c', 'core.hooksPath=/dev/null', ...args]);
function optionalGit(cwd, ...args) {
  try { return git(cwd, ...args); } catch (e) { if (e.status === 1) return null; throw e; }
}
function branchName(cwd, name) {
  if (typeof name !== 'string' || !name || name.startsWith('-') || /[\x00-\x20\x7f]/.test(name)) throw Error('Unsafe Git branch metadata');
  safeRelative(name);
  try { git(cwd, 'check-ref-format', `refs/heads/${name}`); git(cwd, 'check-ref-format', '--branch', name); } catch { throw Error(`Unsafe Git branch metadata: ${name}`); }
  if (name === 'HEAD') throw Error('Unsafe Git branch metadata: HEAD');
  return name;
}
function upstreamNames(cwd, u) {
  if (!u || typeof u.remote !== 'string' || !/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(u.remote) || u.remote.endsWith('.lock') || u.remote.includes('..')) throw Error('Unsafe Git upstream remote metadata');
  if (typeof u.merge !== 'string' || !u.merge.startsWith('refs/heads/')) throw Error('Unsafe Git upstream merge metadata');
  branchName(cwd, u.merge.slice('refs/heads/'.length));
}
export function validateBranchMetadata(cwd, repo) {
  if (repo.branch === null) { if (repo.upstream !== null) throw Error('Detached source cannot have branch upstream metadata'); return; }
  branchName(cwd, repo.branch);
  if (repo.upstream === null) return;
  const u = repo.upstream;
  upstreamNames(cwd, u); originSafe(u.origin);
  if (u.remote === 'origin' && u.origin !== repo.origin) throw Error('Upstream origin mismatch');
  if (u.revision !== null && !/^[0-9a-f]{40}$/.test(u.revision)) throw Error('Unsafe Git upstream revision metadata');
}
export function branchMetadata(cwd) {
  const symbolic = optionalGit(cwd, 'symbolic-ref', '--quiet', 'HEAD');
  if (symbolic === null) return { branch: null, upstream: null };
  if (!symbolic.startsWith('refs/heads/')) throw Error('Source HEAD must refer to a local branch or be detached');
  const branch = symbolic.slice('refs/heads/'.length);
  branchName(cwd, branch);
  const remote = optionalGit(cwd, 'config', '--get', `branch.${branch}.remote`);
  const merge = optionalGit(cwd, 'config', '--get', `branch.${branch}.merge`);
  if (remote === null && merge === null) return { branch, upstream: null };
  const result = { branch, upstream: { remote, merge, origin: null, revision: null } };
  // Validate names before using them as Git options, config keys or ref paths.
  upstreamNames(cwd, result.upstream);
  result.upstream.origin = originSafe(git(cwd, 'remote', 'get-url', remote));
  const ref = `refs/remotes/${remote}/${merge.slice('refs/heads/'.length)}`;
  result.upstream.revision = optionalGit(cwd, 'rev-parse', '--verify', '--quiet', `${ref}^{commit}`);
  return result;
}
export function safeRelative(p) {
  if (typeof p !== 'string' || !p || p.includes('\\') || p.includes('\0') || p.split('/').some(s => !s || s === '.' || s === '..') || path.isAbsolute(p)) throw Error(`Unsafe relative path: ${p}`);
  return p;
}
export function safeLink(p, target) {
  safeRelative(p);
  if (!target || path.isAbsolute(target) || target.includes('\\') || target.includes('\0')) throw Error(`Unsafe symlink: ${p}`);
  const resolved = path.posix.normalize(path.posix.join(path.posix.dirname(p), target));
  safeRelative(resolved);
  if (resolved.split('/').includes('.git')) throw Error(`Symlink into Git metadata: ${p}`);
}
export function noSymlinkParents(root, p) {
  safeRelative(p);
  let current = root;
  if (fs.lstatSync(root).isSymbolicLink()) throw Error(`Symlink root: ${root}`);
  for (const part of p.split('/').slice(0, -1)) {
    current = path.join(current, part);
    if (fs.existsSync(current) && (!fs.lstatSync(current).isDirectory() || fs.lstatSync(current).isSymbolicLink())) throw Error(`Unsafe parent: ${p}`);
  }
}
export function entry(root, p) {
  noSymlinkParents(root, p);
  const file = path.join(root, p), s = fs.lstatSync(file);
  const mode = s.mode & 0o777;
  if (s.isSymbolicLink()) {
    const target = fs.readlinkSync(file); safeLink(p, target);
    return { path: p, type: 'symlink', target, mode, sha256: digest(target) };
  }
  if (!s.isFile()) throw Error(`Not a regular file: ${p}`);
  return { path: p, type: 'file', mode, size: s.size, sha256: digest(fs.readFileSync(file)) };
}
export function walk(root, prefix = '') {
  const result = [];
  for (const name of fs.readdirSync(path.join(root, prefix)).sort()) {
    const p = prefix ? `${prefix}/${name}` : name; safeRelative(p);
    const s = fs.lstatSync(path.join(root, p));
    if (s.isDirectory()) result.push(...walk(root, p)); else result.push(p);
  }
  return result;
}
export function copyEntry(source, p, target, dest = p) {
  const stamp = fs.lstatSync(path.join(source, p), { bigint: true });
  const before = entry(source, p); noSymlinkParents(target, dest);
  fs.mkdirSync(path.dirname(path.join(target, dest)), { recursive: true, mode: 0o700 });
  if (before.type === 'symlink') fs.symlinkSync(before.target, path.join(target, dest));
  else { fs.copyFileSync(path.join(source, p), path.join(target, dest), fs.constants.COPYFILE_EXCL); fs.chmodSync(path.join(target, dest), before.mode); }
  const after = fs.lstatSync(path.join(source, p), { bigint: true });
  if (['ino', 'size', 'mtimeNs', 'ctimeNs', 'mode'].some(k => stamp[k] !== after[k]) || JSON.stringify(before) !== JSON.stringify(entry(source, p)) || entry(target, dest).sha256 !== before.sha256) throw Error(`Source changed while capturing: ${p}`);
  return before;
}
const quoted = name => `"${name.replaceAll('"', '""')}"`;
const canonical = value => JSON.stringify(value, (_, v) => typeof v === 'bigint' ? { sqliteInteger: String(v) } : v instanceof Uint8Array ? { sqliteBlob: Buffer.from(v).toString('hex') } : v);
export function openReadonly(file) {
  if (!fs.lstatSync(file).isFile() || fs.lstatSync(file).isSymbolicLink()) throw Error(`Database is not a regular file: ${file}`);
  return new DatabaseSync(file, { readOnly: true });
}
export function databaseSummary(file, kind) {
  const db = openReadonly(file);
  try {
    if (db.prepare('PRAGMA integrity_check').all().some(r => Object.values(r)[0] !== 'ok')) throw Error(`Database integrity failed: ${kind}`);
    if (db.prepare('PRAGMA foreign_key_check').all().length) throw Error(`Foreign key violations: ${kind}`);
    const tables = {};
    for (const { name, sql } of db.prepare("SELECT name,sql FROM sqlite_master WHERE type='table' ORDER BY name").all()) {
      const stmt = db.prepare(`SELECT * FROM ${quoted(name)}`); stmt.setReadBigInts(true);
      const rows = stmt.all().map(canonical).sort();
      tables[name] = { count: rows.length, sha256: digest(canonical(rows)), schema: sql };
    }
    const required = kind === 'jobs' ? ['jobs', 'queue_state', 'assessment_queue', 'preparation_operations', 'application_drafts', 'application_revisions', 'contexts'] : ['domains', 'pages'];
    for (const t of required) if (!tables[t]) throw Error(`Missing ${kind} table: ${t}`);
    const values = t => tables[t] ? db.prepare(`SELECT * FROM ${quoted(t)}`).all() : [];
    if (kind === 'jobs') {
      const off = db.prepare("SELECT data FROM queue_state WHERE key='assessment_enabled'").get();
      if (!off || off.data !== 'false') throw Error('Assessment must be explicitly Off before transfer; no preference was changed');
      if (values('preparation_operations').some(r => ['queued', 'running'].includes(r.status) || r.pid && alive(r.pid))) throw Error('Active preparation operation');
      if (values('assessment_queue').some(r => r.status === 'running' || r.owner && alive(r.owner))) throw Error('Active assessment');
      if (['runs', 'collection_runs'].some(t => values(t).some(r => ['running', 'queued'].includes(r.status) || !r.finished))) throw Error('Active collection/run');
      if (values('locks').some(r => alive(r.pid))) throw Error('Live slot lock');
    } else if (values('traffic_leases').some(r => alive(r.pid))) throw Error('Live network lease');
    return { userVersion: Object.values(db.prepare('PRAGMA user_version').get())[0], tables,
      ...(kind === 'jobs' ? { assessmentEnabled: false, queueState: values('queue_state'), contexts: values('contexts').map(r => ({ version: r.version, created: r.created })), applicationCounts: Object.fromEntries(['jobs', 'application_drafts', 'application_revisions', 'questionnaire_snapshots', 'preparation_operations', 'workflow_history'].map(t => [t, tables[t]?.count ?? 0])), drafts: values('application_drafts').map(r => { const d = JSON.parse(r.data); return { jobId: r.job_id, revision: d.revision, fields: d.fields.length, nonemptyAnswers: d.fields.filter(f => f.type !== 'file' && (Array.isArray(f.value) ? f.value.length : f.value?.trim())).length }; }) } : { domains: values('domains') }) };
  } finally { db.close(); }
}
export function alive(pid) {
  if (!Number.isInteger(Number(pid)) || Number(pid) <= 0) return false;
  try { process.kill(Number(pid), 0); return true; } catch (e) { return e.code !== 'ESRCH'; }
}
export function snapshot(source, dest) {
  const db = openReadonly(source);
  try { db.exec(`VACUUM INTO '${dest.replaceAll("'", "''")}'`); } finally { db.close(); }
  fs.chmodSync(dest, 0o600);
}
export function originSafe(origin) {
  if (typeof origin !== 'string' || !/^(git@github\.com:[\w.-]+\/[\w.-]+\.git|https:\/\/github\.com\/[\w.-]+\/[\w.-]+\.git)$/.test(origin)) throw Error('Expected credential-free GitHub origin');
  return origin;
}
export function validateArchive(root) {
  const mfile = path.join(root, 'manifest.json');
  noSymlinkParents(root, 'manifest.json');
  if (!fs.lstatSync(mfile).isFile() || fs.lstatSync(mfile).isSymbolicLink()) throw Error('Unsafe manifest');
  const m = JSON.parse(fs.readFileSync(mfile, 'utf8'));
  if (m.format !== FORMAT || !Array.isArray(m.payload) || !Array.isArray(m.workingTree)) throw Error('Unsupported manifest');
  const expected = new Set(['manifest.json']);
  for (const e of m.payload) {
    safeRelative(e.path);
    if (expected.has(e.path)) throw Error(`Duplicate payload: ${e.path}`);
    expected.add(e.path);
    if (JSON.stringify(entry(root, e.path)) !== JSON.stringify(e)) throw Error(`Payload checksum/mode/link mismatch: ${e.path}`);
  }
  for (const p of walk(root)) if (!expected.has(p)) throw Error(`Unexpected archive file: ${p}`);
  const tree = new Set();
  for (const e of m.workingTree) {
    safeRelative(e.path);
    if (tree.has(e.path) || e.path.split('/').includes('.git') || e.path.startsWith('.local/')) throw Error(`Unsafe overlay: ${e.path}`);
    tree.add(e.path);
    const actual = entry(root, `payload/tree/${e.path}`); actual.path = e.path;
    if (JSON.stringify(actual) !== JSON.stringify(e)) throw Error(`Overlay mismatch: ${e.path}`);
  }
  for (const e of m.payload) if (e.path.startsWith('payload/tree/') && !tree.has(e.path.slice('payload/tree/'.length))) throw Error(`Unlisted overlay: ${e.path}`);
  for (const p of m.configuredInputs ?? []) {
    safeRelative(p);
    if (!tree.has(p) && !expected.has(`payload/state/files/${p}`)) throw Error(`Missing configured input: ${p}`);
  }
  originSafe(m.git.origin);
  validateBranchMetadata(root, m.git);
  for (const [kind, summary] of Object.entries(m.databases)) {
    if (!['jobs', 'network'].includes(kind) || JSON.stringify(databaseSummary(path.join(root, `payload/state/${kind}.sqlite`), kind)) !== JSON.stringify(summary)) throw Error(`Database manifest mismatch: ${kind}`);
  }
  if (!m.databases.jobs || !m.databases.network) throw Error('Both databases required');
  for (const repo of [m.git, m.submodule]) {
    if (!repo || !/^[0-9a-f]{40}$/.test(repo.revision)) throw Error('Missing Git revision/submodule');
    originSafe(repo.origin); safeRelative(repo.bundle);
    if (!expected.has(repo.bundle)) throw Error('Missing Git bundle');
  }
  if (m.submodule.path !== 'ai/shared') throw Error('Unsupported submodule path');
  // Reject runtime destinations outside the explicit allowlist, even in a rewritten manifest.
  for (const p of walk(path.join(root, 'payload/state'))) {
    if (!/^(jobs\.sqlite|network\.sqlite|files\/(\.local\/(application-inspections|application-preparation-errors|imports)\/[^\0]+))$/.test(p)) throw Error(`Unsafe runtime payload: ${p}`);
    if (entry(path.join(root, 'payload/state'), p).type !== 'file') throw Error(`Runtime symlink refused: ${p}`);
  }
  return m;
}

export function restoreArchive(root, target, { verifyOnly = false, install = false } = {}) {
  const m = validateArchive(root);
  if (verifyOnly) return m;
  target = path.resolve(target);
  // Every existing ancestor must be a real directory; never follow a target symlink.
  let ancestor = target;
  while (!fs.existsSync(ancestor)) ancestor = path.dirname(ancestor);
  for (let p = ancestor;; p = path.dirname(p)) {
    if (!fs.lstatSync(p).isDirectory() || fs.lstatSync(p).isSymbolicLink()) throw Error('Target ancestor must not be a symlink');
    if (p === path.dirname(p)) break;
  }
  if (lexists(target)) throw Error('Target already exists; choose a new, absent directory');
  if (!fs.existsSync(path.dirname(target))) throw Error('Create the target parent directory first');
  const staging = fs.mkdtempSync(path.join(path.dirname(target), '.sift-restore-incomplete-'));
  try {
    git(staging, 'init', '--quiet');
    function clone(repo, dest, project = false) {
      git(staging, 'bundle', 'verify', path.join(root, repo.bundle));
      git(staging, 'clone', '--no-checkout', path.join(root, repo.bundle), dest);
      git(dest, 'checkout', '--detach', repo.revision);
      if (git(dest, 'rev-parse', 'HEAD') !== repo.revision) throw Error('Restored revision mismatch');
      git(dest, 'remote', 'set-url', 'origin', originSafe(repo.origin));
      if (project && repo.branch !== null) {
        git(dest, 'checkout', '-B', repo.branch, repo.revision);
        if (repo.upstream) {
          const u = repo.upstream, ref = `refs/remotes/${u.remote}/${u.merge.slice('refs/heads/'.length)}`;
          if (u.remote !== 'origin') git(dest, 'remote', 'add', u.remote, u.origin);
          if (u.revision) { git(dest, 'cat-file', '-e', `${u.revision}^{commit}`); git(dest, 'update-ref', ref, u.revision); }
          else git(dest, 'update-ref', '-d', ref);
          git(dest, 'config', `branch.${repo.branch}.remote`, u.remote);
          git(dest, 'config', `branch.${repo.branch}.merge`, u.merge);
        } else {
          for (const key of ['remote', 'merge']) if (optionalGit(dest, 'config', '--get', `branch.${repo.branch}.${key}`) !== null) git(dest, 'config', '--unset-all', `branch.${repo.branch}.${key}`);
        }
      }
      if (project && JSON.stringify(branchMetadata(dest)) !== JSON.stringify({ branch: repo.branch, upstream: repo.upstream })) throw Error('Restored branch/upstream mismatch');
    }
    const checkout = path.join(staging, 'checkout');
    clone(m.git, checkout, true);
    // The overlay is a complete source snapshot, including deletions and uncommitted files.
    for (const name of fs.readdirSync(checkout)) if (name !== '.git') fs.rmSync(path.join(checkout, name), { recursive: true, force: true });
    fs.mkdirSync(path.join(checkout, 'ai'), { recursive: true });
    clone(m.submodule, path.join(checkout, 'ai/shared'));
    for (const name of fs.readdirSync(path.join(checkout, 'ai/shared'))) if (name !== '.git') fs.rmSync(path.join(checkout, 'ai/shared', name), { recursive: true, force: true });
    for (const e of m.workingTree) copyEntry(path.join(root, 'payload/tree'), e.path, checkout);
    git(checkout, 'submodule', 'init', '--', 'ai/shared');
    fs.mkdirSync(path.join(checkout, '.local'), { mode: 0o700 });
    for (const kind of ['jobs', 'network']) copyEntry(path.join(root, 'payload/state'), `${kind}.sqlite`, checkout, `.local/${kind}.sqlite`);
    const files = path.join(root, 'payload/state/files');
    if (fs.existsSync(files)) for (const p of walk(files)) copyEntry(files, p, checkout);
    for (const e of m.workingTree) if (JSON.stringify(entry(checkout, e.path)) !== JSON.stringify(e)) throw Error(`Restored overlay mismatch: ${e.path}`);
    for (const kind of ['jobs', 'network']) if (JSON.stringify(databaseSummary(path.join(checkout, `.local/${kind}.sqlite`), kind)) !== JSON.stringify(m.databases[kind])) throw Error(`Restored database mismatch: ${kind}`);
    if (git(checkout, 'ls-tree', 'HEAD', 'ai/shared').split(/\s+/)[2] !== m.submodule.revision) throw Error('Submodule pin mismatch');
    if (install) { command(path.join(checkout, 'app'), 'npm', ['ci']); command(path.join(checkout, 'app'), 'npm', ['run', 'build']); }
    fs.writeFileSync(path.join(checkout, '.local/transfer-receipt.json'), JSON.stringify({ format: FORMAT, created: m.created, revision: m.git.revision, verified: new Date().toISOString() }, null, 2), { mode: 0o600, flag: 'wx' });
    // mkdir is the exclusive claim: unlike an exists-then-rename it cannot replace
    // a target that appeared during verification, including a broken symlink.
    fs.mkdirSync(target, { mode: 0o700 });
    const claimed = fs.lstatSync(target);
    try { fs.renameSync(checkout, target); }
    catch (e) { const s = fs.lstatSync(target); if (s.ino === claimed.ino && s.dev === claimed.dev && fs.readdirSync(target).length === 0) fs.rmdirSync(target); throw e; }
    return m;
  } finally { fs.rmSync(staging, { recursive: true, force: true }); }
}
