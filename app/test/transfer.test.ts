import './isolate.js';
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import { syncBuiltinESMExports } from 'node:module';
import path from 'node:path';
import { DatabaseSync } from 'node:sqlite';
import http from 'node:http';
import net from 'node:net';
import YAML from 'yaml';
import { configureOllama, installedModels, loopbackEndpoint } from '../ops/transfer/ollama.js';
import { Store } from '../src/store.js';
import { localPath, ROOT } from '../src/core.js';
import { addManualQuestion, applicationDraft, writeDraft } from '../src/application-store.js';
// @ts-ignore Shared standalone Node module.
import { exportArchive } from '../ops/transfer/export.mjs';
// @ts-ignore Shared standalone Node module.
import { git, command, databaseSummary, restoreArchive, validateArchive, entry, digest, copyEntry, FORMAT } from '../ops/transfer/transfer-lib.mjs';

function write(root: string, p: string, data: string) { fs.mkdirSync(path.dirname(path.join(root, p)), { recursive: true }); fs.writeFileSync(path.join(root, p), data); }
function commit(root: string) { git(root, 'add', '.'); git(root, '-c', 'user.name=Migration Fixture', '-c', 'user.email=fixture@example.invalid', '-c', 'commit.gpgsign=false', 'commit', '-m', 'Fixture snapshot'); }
export function transferFixture() {
  const parent = localPath('transfer-fixtures'); fs.mkdirSync(parent, { recursive: true });
  const root = fs.mkdtempSync(path.join(parent, 'source-'));
  git(root, 'init', '--quiet', '--initial-branch=main'); git(root, 'remote', 'add', 'origin', 'git@github.com:w3con/fcto.git');
  write(root, '.gitignore', '.local/\n.env\napp/node_modules/\napp/dist/\nkb/Applications/answers/\n');
  write(root, 'config/candidate.yaml', 'version: 1\nfiles: [kb/facts.md, .local/imports/profile.md]\nprofiles: []\n');
  write(root, 'kb/facts.md', 'Synthetic candidate fact, not owner data.');
  write(root, 'app/root.mjs', "import {execFileSync} from 'node:child_process'; console.log(execFileSync('git',['rev-parse','--show-toplevel'],{encoding:'utf8'}).trim());\n");
  for (const p of fs.readdirSync(path.join(ROOT, 'app/ops/transfer'))) if (fs.statSync(path.join(ROOT, 'app/ops/transfer', p)).isFile()) {
    fs.mkdirSync(path.join(root, 'app/ops/transfer'), { recursive: true }); fs.copyFileSync(path.join(ROOT, 'app/ops/transfer', p), path.join(root, 'app/ops/transfer', p));
  }
  write(root, 'delete-me.md', 'Deleted after commit'); write(root, 'executable.sh', '#!/bin/sh\nexit 0\n'); fs.chmodSync(path.join(root, 'executable.sh'), 0o755);
  const shared = path.join(root, 'ai/shared'); fs.mkdirSync(shared, { recursive: true }); git(shared, 'init', '--quiet'); git(shared, 'remote', 'add', 'origin', 'git@github.com:w3con/ai.git'); write(shared, 'bin/tool', 'synthetic shared tool'); commit(shared);
  write(root, '.gitmodules', '[submodule "ai/shared"]\n path = ai/shared\n url = git@github.com:w3con/ai.git\n');
  fs.symlinkSync('shared/bin', path.join(root, 'ai/bin')); commit(root);
  git(root, 'update-ref', 'refs/remotes/origin/main', 'HEAD'); git(root, 'branch', '--set-upstream-to=origin/main', 'main');
  write(root, 'uncommitted-tool.mjs', 'export const fixture = true;'); write(root, 'kb/facts.md', 'Uncommitted exact candidate fact.'); fs.unlinkSync(path.join(root, 'delete-me.md'));
  write(shared, 'bin/uncommitted', 'shared overlay');
  write(root, 'kb/Applications/answers/remembered.md', 'Remembered synthetic answer');
  write(root, '.env', 'TOKEN=excluded'); write(root, '.local/sessions/cookies', 'excluded session'); write(root, '.local/inference-config/auth.json', 'excluded credentials');
  write(root, '.local/imports/profile.md', 'Synthetic owner-provided export'); write(root, '.local/application-inspections/fixture.json', '{"synthetic":true}');
  const store = new Store(path.join(root, '.local/jobs.sqlite'));
  const id = store.upsert({ title: 'Fixture Engineer', company: 'Fixture', originalUrl: 'https://example.invalid/job', description: 'Synthetic description', descriptionStatus: 'fixture', source: 'Fixture' });
  store.status(id, 'saved'); store.db.exec("INSERT INTO queue_state VALUES('assessment_enabled','false'); INSERT INTO queue_state VALUES('runtime','{\"paused\":true,\"retryAt\":9999999999999}');");
  addManualQuestion(store, id, 'Synthetic question?');
  const draft = applicationDraft(store, id); draft.fields[0].value = 'Synthetic saved answer'; draft.fields[0].edited = true; draft.fields[0].revision = 4;
  draft.questionnaire = { technicalLog: '.local/application-inspections/fixture.json', url: 'https://example.invalid/apply', capturedAt: '2026-10-01', language: 'en', status: 'partial', gaps: [], fields: [], family: 'Fixture' };
  writeDraft(store, draft, 'Synthetic answer edit');
  store.db.exec(`INSERT INTO assessment_queue(job_id,fingerprint,provider,status,attempts,next_at) VALUES(${id},'fixture','ollama','pending',2,9999999999999); INSERT INTO job_aliases VALUES(999,${id}); INSERT INTO deleted_listings VALUES('synthetic-tombstone');`);
  store.db.prepare('INSERT INTO contexts VALUES(?,?,?)').run('fixture-context', '2026-10-01', '{"sources":[{"text":"Synthetic runtime-only fact"}]}');
  store.db.prepare('INSERT INTO context_refreshes(version,created,data) VALUES(?,?,?)').run('fixture-context', '2026-10-02', '{"receipt":true}');
  store.close();
  const network = new DatabaseSync(path.join(root, '.local/network.sqlite'));
  network.exec("PRAGMA journal_mode=WAL; CREATE TABLE domains(host TEXT PRIMARY KEY,data TEXT NOT NULL); CREATE TABLE pages(key TEXT PRIMARY KEY,created INTEGER NOT NULL,data TEXT NOT NULL); CREATE TABLE traffic_leases(host TEXT PRIMARY KEY,pid INTEGER NOT NULL);");
  network.prepare('INSERT INTO domains VALUES(?,?)').run('example.invalid', '{"cooldown":9999999999999,"failures":3,"next":9000,"reason":"HTTP 429 fixture"}');
  network.prepare('INSERT INTO pages VALUES(?,?,?)').run('fixture-key', 123, '{"page":{"text":"Synthetic cached HTML","etag":"fixture-etag"}}'); network.close();
  return root;
}
function unpack(root: string) {
  const result = exportArchive(root); const unpacked = path.join(root, '.local/unpacked'); fs.mkdirSync(unpacked); command(unpacked, 'tar', ['-xzf', result.archive]); return { ...result, unpacked };
}
function changeManifest(root: string, mutate: (m: any) => void) { const p = path.join(root, 'manifest.json'); const m = JSON.parse(fs.readFileSync(p, 'utf8')); mutate(m); fs.writeFileSync(p, JSON.stringify(m, null, 2)); }

test('portable roundtrip preserves Git history, exact uncommitted source, executable/relative links and complete idle runtime, without source writes', () => {
  const root = transferFixture();
  // Keep a real, nonempty WAL open during export. Its committed data must be snapshotted,
  // while neither the source main database nor WAL bytes may be modified by transfer.
  const writer = new DatabaseSync(path.join(root, '.local/jobs.sqlite'));
  writer.exec("CREATE TABLE fixture_budgets(name TEXT PRIMARY KEY,remaining INTEGER,payload BLOB); INSERT INTO fixture_budgets VALUES('fixture',9007199254740993,X'0001ff');");
  const before = databaseSummary(path.join(root, '.local/jobs.sqlite'), 'jobs'), network = databaseSummary(path.join(root, '.local/network.sqlite'), 'network');
  const physical = ['jobs', 'network'].map(k => digest(fs.readFileSync(path.join(root, `.local/${k}.sqlite`))));
  const walBefore = digest(fs.readFileSync(path.join(root, '.local/jobs.sqlite-wal')));
  const config = fs.readFileSync(path.join(root, '.git/config'));
  const { archive, checksum, manifest, unpacked } = unpack(root);
  assert.equal(digest(fs.readFileSync(archive)), checksum);
  assert.deepEqual(manifest.databases.jobs, before); assert.deepEqual(manifest.databases.network, network);
  assert.ok(manifest.workingTree.some((e: any) => e.path === 'app/ops/transfer/export.mjs'));
  const target = path.join(root, '.local/restored');
  // Exercise the actual pre-dependency standalone entry point, not merely the library.
  command(unpacked, process.execPath, ['restore.mjs', '--verify-only']);
  command(unpacked, process.execPath, ['restore.mjs', '--target', target, '--no-install']);
  assert.equal(command(path.join(target, 'app'), process.execPath, ['root.mjs']), target);
  assert.equal(git(target, 'rev-parse', 'HEAD'), manifest.git.revision); assert.equal(git(target, 'remote', 'get-url', 'origin'), 'git@github.com:w3con/fcto.git');
  assert.equal(manifest.git.branch, 'main'); assert.equal(manifest.git.upstream.merge, 'refs/heads/main');
  assert.equal(git(target, 'symbolic-ref', '--short', 'HEAD'), 'main'); assert.equal(git(target, 'rev-parse', '--abbrev-ref', '@{upstream}'), 'origin/main');
  assert.equal(git(target, 'rev-parse', '@{upstream}'), manifest.git.upstream.revision);
   assert.equal(git(path.join(target, 'ai/shared'), 'rev-parse', 'HEAD'), manifest.submodule.revision); assert.equal(git(path.join(target, 'ai/shared'), 'remote', 'get-url', 'origin'), 'git@github.com:w3con/ai.git');
   assert.match(git(target, 'submodule', 'status', '--', 'ai/shared'), /^[0-9a-f]{40} ai\/shared/);
  assert.ok(git(target, 'log', '--oneline').includes('Fixture snapshot'));
  assert.equal(fs.readFileSync(path.join(target, 'kb/facts.md'), 'utf8'), 'Uncommitted exact candidate fact.'); assert.equal(fs.existsSync(path.join(target, 'delete-me.md')), false);
  for (const p of ['uncommitted-tool.mjs', 'ai/shared/bin/uncommitted', 'kb/Applications/answers/remembered.md', '.local/imports/profile.md', '.local/application-inspections/fixture.json']) assert.ok(fs.existsSync(path.join(target, p)), p);
  assert.equal(fs.statSync(path.join(target, 'executable.sh')).mode & 0o777, 0o755); assert.equal(fs.readlinkSync(path.join(target, 'ai/bin')), 'shared/bin');
  for (const p of ['.env', '.local/sessions', '.local/inference-config', 'app/node_modules', 'app/dist']) assert.equal(fs.existsSync(path.join(target, p)), false, p);
  assert.deepEqual(databaseSummary(path.join(target, '.local/jobs.sqlite'), 'jobs'), before); assert.deepEqual(databaseSummary(path.join(target, '.local/network.sqlite'), 'network'), network);
  assert.deepEqual(databaseSummary(path.join(root, '.local/jobs.sqlite'), 'jobs'), before); assert.deepEqual(['jobs', 'network'].map(k => digest(fs.readFileSync(path.join(root, `.local/${k}.sqlite`)))), physical); assert.ok(fs.readFileSync(path.join(root, '.git/config')).equals(config));
  assert.equal(digest(fs.readFileSync(path.join(root, '.local/jobs.sqlite-wal'))), walBefore); writer.close();
  assert.throws(() => restoreArchive(unpacked, target), /exists/); assert.equal(fs.readFileSync(path.join(target, 'kb/facts.md'), 'utf8'), 'Uncommitted exact candidate fact.');
  const symlink = path.join(root, '.local/symlink-target'); fs.symlinkSync(target, symlink); assert.throws(() => restoreArchive(unpacked, symlink), /symlink|exists/i);
  const empty = path.join(root, '.local/empty-target'); fs.mkdirSync(empty); assert.throws(() => restoreArchive(unpacked, empty), /exists/); assert.deepEqual(fs.readdirSync(empty), []);
  const broken = path.join(root, '.local/broken-target'); fs.symlinkSync('absent-directory', broken); assert.throws(() => restoreArchive(unpacked, broken), /exists/); assert.equal(fs.readlinkSync(broken), 'absent-directory');
});

test('development restore supports genuinely detached HEAD, an attached branch without upstream, and a named GitHub upstream at its archived tip', () => {
  for (const mode of ['detached', 'no-upstream', 'named-upstream']) {
    const root = transferFixture();
    if (mode === 'detached') git(root, 'checkout', '--detach');
    else {
      git(root, 'checkout', '-b', 'feature/development-move');
      if (mode === 'named-upstream') {
        git(root, 'remote', 'add', 'team', 'git@github.com:w3con/fcto.git');
        git(root, 'update-ref', 'refs/remotes/team/development', 'HEAD');
        git(root, 'branch', '--set-upstream-to=team/development', 'feature/development-move');
        // The local branch is ahead of its upstream: do not rewrite upstream to HEAD.
        write(root, 'tracked-development.md', 'Synthetic next development commit'); commit(root);
      }
    }
    const sourceConfig = fs.readFileSync(path.join(root, '.git/config'));
    const { manifest, unpacked } = unpack(root), target = path.join(root, '.local/restored'); restoreArchive(unpacked, target);
    assert.equal(git(target, 'rev-parse', 'HEAD'), manifest.git.revision);
    assert.equal(git(target, 'rev-parse', '--abbrev-ref', 'HEAD'), mode === 'detached' ? 'HEAD' : 'feature/development-move');
    if (mode === 'named-upstream') {
      assert.equal(git(target, 'rev-parse', '--abbrev-ref', '@{upstream}'), 'team/development');
      assert.equal(git(target, 'remote', 'get-url', 'team'), manifest.git.upstream.origin);
      assert.equal(git(target, 'rev-parse', '@{upstream}'), manifest.git.upstream.revision);
      assert.notEqual(manifest.git.revision, manifest.git.upstream.revision);
    } else { assert.equal(manifest.git.upstream, null); assert.throws(() => git(target, 'rev-parse', '--verify', '--quiet', '@{upstream}')); }
    assert.ok(fs.readFileSync(path.join(root, '.git/config')).equals(sourceConfig));
  }
});

test('restore refuses tampering, missing/corrupt databases, traversal, unsafe links and bundle revision mismatch before creating a ready target', () => {
  const root = transferFixture(), { unpacked } = unpack(root), target = path.join(root, '.local/refused');
  const m = fs.readFileSync(path.join(unpacked, 'manifest.json'));
  for (const branch of ['--orphan=bad', '../escape', '/absolute', 'main\n[core]', 'bad.lock', 'HEAD', '@{-1}']) {
    changeManifest(unpacked, m => { m.git.branch = branch; }); assert.throws(() => restoreArchive(unpacked, target), /Unsafe/); fs.writeFileSync(path.join(unpacked, 'manifest.json'), m);
  }
  for (const upstream of [{ remote: '--upload-pack=bad' }, { remote: '../escape' }, { merge: 'refs/tags/main' }, { merge: 'refs/heads/../escape' }, { revision: '--bad' }, { origin: 'https://evil.invalid/repo.git' }]) {
    changeManifest(unpacked, m => { Object.assign(m.git.upstream, upstream); }); assert.throws(() => restoreArchive(unpacked, target), /Unsafe|GitHub/); fs.writeFileSync(path.join(unpacked, 'manifest.json'), m);
  }
  const facts = path.join(unpacked, 'payload/tree/kb/facts.md'), original = fs.readFileSync(facts);
  fs.appendFileSync(facts, 'tampered'); assert.throws(() => restoreArchive(unpacked, target), /mismatch/); fs.writeFileSync(facts, original);
  changeManifest(unpacked, m => { m.payload[0].path = '../escape'; }); assert.throws(() => restoreArchive(unpacked, target), /Unsafe/); fs.writeFileSync(path.join(unpacked, 'manifest.json'), m);
  const link = path.join(unpacked, 'payload/tree/ai/bin'); fs.unlinkSync(link); fs.symlinkSync('/etc', link); assert.throws(() => validateArchive(unpacked), /Unsafe symlink/); fs.unlinkSync(link); fs.symlinkSync('shared/bin', link);
  fs.unlinkSync(link); fs.symlinkSync('../../../outside', link); assert.throws(() => restoreArchive(unpacked, target), /Unsafe|mismatch/); fs.unlinkSync(link); fs.symlinkSync('shared/bin', link);
  const db = path.join(unpacked, 'payload/state/network.sqlite'), dbBytes = fs.readFileSync(db); fs.unlinkSync(db); assert.throws(() => restoreArchive(unpacked, target)); fs.writeFileSync(db, 'corrupt'); fs.chmodSync(db, 0o600);
  changeManifest(unpacked, m => { m.payload = m.payload.map((e: any) => e.path === 'payload/state/network.sqlite' ? entry(unpacked, e.path) : e); }); assert.throws(() => restoreArchive(unpacked, target), /database/i); fs.writeFileSync(db, dbBytes); fs.writeFileSync(path.join(unpacked, 'manifest.json'), m);
  changeManifest(unpacked, m => { m.git.revision = '0'.repeat(40); }); assert.throws(() => restoreArchive(unpacked, target)); assert.equal(fs.existsSync(target), false);
  assert.equal(fs.readdirSync(path.join(root, '.local')).some(p => p.startsWith('.sift-restore-incomplete-')), false);
});

test('export refuses enabled inference, active workers/slots/leases, broken FKs, excluded or missing configured inputs and missing shared tooling', () => {
  const root = transferFixture(), file = path.join(root, '.local/jobs.sqlite');
  function rejects(sql: string, undo: string, pattern: RegExp) { const db = new DatabaseSync(file); db.exec(sql); db.close(); assert.throws(() => exportArchive(root), pattern); const reset = new DatabaseSync(file); reset.exec(undo); reset.close(); }
  rejects("UPDATE queue_state SET data='true' WHERE key='assessment_enabled'", "UPDATE queue_state SET data='false' WHERE key='assessment_enabled'", /Off/);
  rejects("UPDATE assessment_queue SET status='running'", "UPDATE assessment_queue SET status='pending'", /assessment/);
  rejects(`INSERT INTO locks VALUES('inference',${process.pid},'fixture')`, 'DELETE FROM locks', /slot lock/);
  rejects("INSERT INTO preparation_operations(job_id,status,created,updated,request,progress) VALUES(1,'queued','fixture','fixture','{}','fixture')", 'DELETE FROM preparation_operations', /preparation/);
  rejects("INSERT INTO runs(started,status) VALUES('fixture','running')", 'DELETE FROM runs', /collection/);
  rejects("PRAGMA foreign_keys=OFF; INSERT INTO sightings VALUES(123456,'fixture','https://orphan.invalid','fixture')", "DELETE FROM sightings WHERE job_id=123456", /Foreign key/);
  const n = new DatabaseSync(path.join(root, '.local/network.sqlite')); n.exec(`INSERT INTO traffic_leases VALUES('fixture',${process.pid})`); n.close(); assert.throws(() => exportArchive(root), /network lease/); const nr = new DatabaseSync(path.join(root, '.local/network.sqlite')); nr.exec('DELETE FROM traffic_leases'); nr.close();
  const candidate = fs.readFileSync(path.join(root, 'config/candidate.yaml')); write(root, 'config/candidate.yaml', 'files: [.local/sessions/cookies]\n'); assert.throws(() => exportArchive(root), /excluded/); fs.writeFileSync(path.join(root, 'config/candidate.yaml'), candidate);
  fs.symlinkSync('facts.md', path.join(root, 'kb/linked-facts.md')); write(root, 'config/candidate.yaml', 'files: [kb/linked-facts.md]\n'); assert.throws(() => exportArchive(root), /regular file/); fs.writeFileSync(path.join(root, 'config/candidate.yaml'), candidate);
  fs.unlinkSync(path.join(root, '.local/imports/profile.md')); assert.throws(() => exportArchive(root), /ENOENT/); write(root, '.local/imports/profile.md', 'replacement');
  fs.renameSync(path.join(root, 'ai/shared'), path.join(root, '.local/shared-away')); assert.throws(() => exportArchive(root), /Shared tooling/);
});

test('capture detects a file changing during its copy and refuses absolute/escaping source links', () => {
  const root = transferFixture(), target = path.join(root, '.local/copy-target'); fs.mkdirSync(target);
  // Mutating during the actual copy deterministically exercises capture's before/after check.
  const original = fs.copyFileSync;
  fs.copyFileSync = ((src: any, dest: any, mode: any) => { original(src, dest, mode); fs.appendFileSync(src, 'changed during capture'); }) as typeof original;
  syncBuiltinESMExports();
  try { assert.throws(() => copyEntry(root, 'kb/facts.md', target), /changed while capturing/); } finally { fs.copyFileSync = original; syncBuiltinESMExports(); }
  let changed = false;
  fs.copyFileSync = ((src: any, dest: any, mode: any) => {
    original(src, dest, mode);
    if (!changed && String(src).endsWith('/kb/facts.md')) {
      changed = true; const db = new DatabaseSync(path.join(root, '.local/jobs.sqlite')); db.exec("UPDATE queue_state SET data='{\"paused\":false}' WHERE key='runtime'"); db.close();
    }
  }) as typeof original;
  syncBuiltinESMExports();
  try { assert.throws(() => exportArchive(root), /State moved/); assert.equal(changed, true); } finally { fs.copyFileSync = original; syncBuiltinESMExports(); }
  changed = false;
  fs.copyFileSync = ((src: any, dest: any, mode: any) => {
    original(src, dest, mode);
    if (!changed && String(src).endsWith('/kb/facts.md')) { changed = true; git(root, 'checkout', '-b', 'feature/moved-during-capture'); }
  }) as typeof original;
  syncBuiltinESMExports();
  try { assert.throws(() => exportArchive(root), /Source Git\/tree changed/); assert.equal(changed, true); } finally { fs.copyFileSync = original; syncBuiltinESMExports(); }
  fs.symlinkSync('../../outside', path.join(root, 'escape')); assert.throws(() => exportArchive(root), /Unsafe relative path/);
});

async function freePort() {
  const server = net.createServer(); await new Promise<void>(r => server.listen(0, '127.0.0.1', r));
  const port = (server.address() as net.AddressInfo).port; await new Promise<void>(r => server.close(() => r())); return port;
}
test('target Ollama discovers exact installed models, validates real adapter structured smoke, preserves routing alternatives/budgets/Off and never writes on failure', async t => {
  const deadlines: number[] = [], timeout = AbortSignal.timeout.bind(AbortSignal);
  t.mock.method(AbortSignal, 'timeout', (ms: number) => { deadlines.push(ms); return timeout(ms); });
  const root = transferFixture();
  fs.copyFileSync(path.join(ROOT, 'config/providers.yaml'), path.join(root, 'config/providers.yaml'));
  write(root, '.local/transfer-receipt.json', JSON.stringify({ format: FORMAT }));
  const port = await freePort(); write(root, 'config/runtime.yaml', `port: ${port}\n`);
  const configFile = path.join(root, 'config/providers.yaml'), original = fs.readFileSync(configFile), before = YAML.parse(original.toString());
  const dbBefore = databaseSummary(path.join(root, '.local/jobs.sqlite'), 'jobs');
  let mode = 'valid', chats = 0;
  const server = http.createServer(async (req, res) => {
    res.setHeader('content-type', 'application/json');
    if (mode === 'http-error') { res.statusCode = 503; res.end('{}'); return; }
    if (req.url === '/api/tags') { res.end(mode === 'bad-tags' ? '{"models":[{}]}' : '{"models":[{"name":"fixture-model:exact"}]}'); return; }
    assert.equal(req.url, '/api/chat'); chats++;
    let text = ''; for await (const chunk of req) text += chunk;
    const body = JSON.parse(text); assert.equal(body.model, 'fixture-model:exact'); assert.equal(body.stream, false); assert.equal(body.format.additionalProperties, false); assert.deepEqual(body.format.required, ['ok', 'value']); assert.match(body.messages[0].content, /Synthetic connectivity/); assert.ok(!text.includes('Synthetic saved answer'));
    if (mode === 'smoke-http') { res.statusCode = 500; res.end('{}'); return; }
    if (mode === 'bad-json') { res.end('{bad'); return; }
    if (mode === 'moving-config') fs.appendFileSync(configFile, '\n# concurrent fixture edit\n');
    res.end(JSON.stringify({ message: { content: mode === 'wrong-output' ? '{"ok":false,"value":7}' : '{"ok":true,"value":7}' } }));
  });
  await new Promise<void>(r => server.listen(0, '127.0.0.1', r));
  const endpoint = `http://127.0.0.1:${(server.address() as net.AddressInfo).port}`;
  const unchanged = () => { assert.ok(fs.readFileSync(configFile).equals(original)); assert.equal(fs.readdirSync(path.dirname(configFile)).some(p => /before-ollama|\.tmp$/.test(p)), false); };
  try {
    for (const bad of ['https://127.0.0.1', 'http://example.com', 'http://127.0.0.1@evil.invalid', 'http://127.0.0.1/api', 'http://localhost:11434', 'http://127.0.0.1/?token=x']) assert.throws(() => loopbackEndpoint(bad), /loopback/);
    assert.deepEqual(await installedModels(endpoint), ['fixture-model:exact']); assert.deepEqual(await configureOllama(root, { endpoint }), { models: ['fixture-model:exact'] }); unchanged(); assert.equal(chats, 0);
    await assert.rejects(configureOllama(root, { endpoint, model: 'guessed-model' }), /not installed/); unchanged();
    for (const failure of ['bad-tags', 'http-error', 'wrong-output', 'bad-json', 'smoke-http']) { mode = failure; await assert.rejects(configureOllama(root, { endpoint, model: 'fixture-model:exact' })); unchanged(); }
    mode = 'valid';
    const blocker = net.createServer(); await new Promise<void>(r => blocker.listen(port, '127.0.0.1', r));
    try { await assert.rejects(configureOllama(root, { endpoint, model: 'fixture-model:exact' }), /Stop the catalogue/); unchanged(); } finally { await new Promise<void>(r => blocker.close(() => r())); }
    const db = new DatabaseSync(path.join(root, '.local/jobs.sqlite')); db.exec(`INSERT INTO locks VALUES('inference',${process.pid},'fixture')`); db.close();
    await assert.rejects(configureOllama(root, { endpoint, model: 'fixture-model:exact' }), /Live slot/); unchanged(); const reset = new DatabaseSync(path.join(root, '.local/jobs.sqlite')); reset.exec('DELETE FROM locks'); reset.close();
    mode = 'moving-config'; await assert.rejects(configureOllama(root, { endpoint, model: 'fixture-model:exact' }), /changed during smoke/); assert.match(fs.readFileSync(configFile, 'utf8'), /concurrent fixture edit/); fs.writeFileSync(configFile, original); unchanged();
    mode = 'valid'; const configured = await configureOllama(root, { endpoint, model: 'fixture-model:exact' });
    assert.ok(fs.readFileSync(configured.backup!).equals(original));
    const after = YAML.parse(fs.readFileSync(configFile, 'utf8'));
    assert.deepEqual(after.stages, { assessment: 'ollama', extraction: 'ollama' });
    assert.deepEqual(after.providers, { ...before.providers, ollama: { ...before.providers.ollama, model: 'fixture-model:exact', endpoint } });
    assert.ok(deadlines.includes(120000), 'cold-load smoke receives the configured 120-second adapter deadline'); assert.ok(!deadlines.includes(20000));
    for (const configuredTimeout of [30000, 240000]) {
      after.providers.ollama.timeout_ms = configuredTimeout; fs.writeFileSync(configFile, YAML.stringify(after)); deadlines.length = 0;
      await configureOllama(root, { endpoint, model: 'fixture-model:exact' });
      assert.ok(deadlines.includes(configuredTimeout === 30000 ? 30000 : 120000));
      assert.equal(YAML.parse(fs.readFileSync(configFile, 'utf8')).providers.ollama.timeout_ms, configuredTimeout, 'smoke does not rewrite normal provider timeout');
    }
    assert.deepEqual(databaseSummary(path.join(root, '.local/jobs.sqlite'), 'jobs'), dbBefore); assert.equal(fs.existsSync(path.join(root, '.local/ollama-setup.lock')), false);
    fs.unlinkSync(path.join(root, '.local/transfer-receipt.json')); await assert.rejects(configureOllama(root, { endpoint }), /restored target/);
  } finally { await new Promise<void>(r => server.close(() => r())); }
  await assert.rejects(configureOllama(root, { endpoint }), /restored target/);
  write(root, '.local/transfer-receipt.json', JSON.stringify({ format: FORMAT }));
  const finalBytes = fs.readFileSync(configFile); await assert.rejects(configureOllama(root, { endpoint, model: 'fixture-model:exact' })); assert.ok(fs.readFileSync(configFile).equals(finalBytes));
});
