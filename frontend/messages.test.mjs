// Tests for the server-unreachable classification. Run from the repository root:
//   node --test frontend/messages.test.mjs
import test from 'node:test';
import assert from 'node:assert/strict';
import { messageKind, SERVER_DOWN_TEXT } from './messages.ts';
import { ServerUnreachableError, isServerUnreachable, api } from './api.ts';

// Stubs globalThis.fetch to reject with the given error, calls api(), then restores fetch.
async function withFetchRejecting(err, fn) {
  const original = globalThis.fetch;
  globalThis.fetch = () => Promise.reject(err);
  try { return await fn(); }
  finally { globalThis.fetch = original; }
}

test('fetch TypeError (Chrome "Failed to fetch") throws an error with kind server-unreachable', async () => {
  await assert.rejects(
    () => withFetchRejecting(new TypeError('Failed to fetch'), () => api('jobs')),
    (e) => { assert.equal(isServerUnreachable(e), true, 'isServerUnreachable must be true'); return true; },
  );
});

test('fetch TypeError (Safari "Load failed") throws an error with kind server-unreachable', async () => {
  await assert.rejects(
    () => withFetchRejecting(new TypeError('Load failed'), () => api('jobs')),
    (e) => { assert.equal(isServerUnreachable(e), true, 'isServerUnreachable must be true'); return true; },
  );
});

test('fetch TypeError (Firefox "NetworkError…") throws an error with kind server-unreachable', async () => {
  await assert.rejects(
    () => withFetchRejecting(new TypeError('NetworkError when attempting to fetch resource.'), () => api('jobs')),
    (e) => { assert.equal(isServerUnreachable(e), true, 'isServerUnreachable must be true'); return true; },
  );
});

test('a plain TypeError (not from fetch) is not classified as server-unreachable', () => {
  const e = new TypeError('Cannot read properties of undefined');
  assert.equal(isServerUnreachable(e), false, 'plain TypeError must not be server-unreachable');
  assert.notEqual(e.message, SERVER_DOWN_TEXT);
});

test('messageKind: SERVER_DOWN_TEXT is retry (not broken)', () => {
  assert.equal(messageKind(SERVER_DOWN_TEXT), 'retry');
});

test('messageKind: auth/config errors remain broken', () => {
  assert.equal(messageKind('Not logged in'), 'broken');
  assert.equal(messageKind('No model is configured'), 'broken');
});

test('TimeoutError DOMException thrown by fetch → ServerUnreachableError', async () => {
  const te = new DOMException('The operation was aborted due to timeout', 'TimeoutError');
  await assert.rejects(
    () => withFetchRejecting(te, () => api('queue', 'GET', undefined, 10000)),
    (e) => { assert.equal(isServerUnreachable(e), true, 'TimeoutError must map to server-unreachable'); return true; },
  );
});

test('POST without timeout: slow fetch TypeError still maps to server-unreachable', async () => {
  await assert.rejects(
    () => withFetchRejecting(new TypeError('Failed to fetch'), () => api('jobs/1/assess', 'POST', {})),
    (e) => { assert.equal(isServerUnreachable(e), true, 'network TypeError on POST must be server-unreachable'); return true; },
  );
});

test('messageKind: "database not responding" is retry', () => {
  assert.equal(messageKind('database not responding'), 'retry');
});
