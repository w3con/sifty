// Stub worker: proves the Go-core to TypeScript-worker boundary round-trips a request. It reads
// one JSON request from standard input and writes one JSON response to standard output, per
// docs/worker-protocol.md. The real Playwright and jobspy workers follow the same contract.
import process from 'node:process';

let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (chunk) => { input += chunk; });
process.stdin.on('end', () => {
  try {
    const req = JSON.parse(input);
    if (typeof req.method !== 'string') throw new Error('request is missing a method');
    // The stub echoes its method and params so the core can verify the round-trip; a real
    // worker would act on the method here and return an operation-specific result.
    const result = { echoedMethod: req.method, echoedParams: req.params ?? null };
    process.stdout.write(JSON.stringify({ ok: true, result }));
  } catch (e) {
    process.stdout.write(JSON.stringify({ ok: false, error: String(e && e.message ? e.message : e) }));
  }
});
