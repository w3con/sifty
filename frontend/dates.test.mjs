// Checks the Newest sort key against the fixture that the server's copy of the key also reads
// (backend/internal/background/testdata/newest-key.json). Run from the repository root:
//   node --test frontend/dates.test.mjs
process.env.TZ = 'UTC';
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import { newestKey } from './dates.ts';

const fixture = JSON.parse(fs.readFileSync(new URL('../backend/internal/background/testdata/newest-key.json', import.meta.url), 'utf8'));

test('the Newest key matches the shared fixture', () => {
  const realNow = Date.now;
  Date.now = () => Date.parse(fixture.now);
  try {
    for (const c of fixture.cases) {
      const k = newestKey(c.raw, c.basis ? 'api' : '');
      assert.equal(k.rank, c.rank, `rank of ${JSON.stringify(c.raw)}`);
      assert.equal(k.rank === 2 ? 0 : k.t, c.millis, `time of ${JSON.stringify(c.raw)}`);
    }
  } finally {
    Date.now = realNow;
  }
});
