// jobspy worker: the Go core's sidecar for the jobspy scraper, over the protocol in
// docs/worker-protocol.md. In production this worker invokes the Python jobspy library and maps
// its output to the posting shape the core stores. Here the mapping is exercised directly: the
// worker receives jobspy-shaped records and returns normalised postings, so the Go core's
// integration is tested without the scraper's network access.
import process from 'node:process';

function toPosting(r) {
  const description = typeof r.description === 'string' ? r.description : '';
  return {
    title: r.title ?? '',
    company: r.company ?? 'Unknown',
    originalUrl: r.job_url ?? '',
    applyUrl: null,
    description,
    descriptionStatus: description ? 'jobspy description' : 'missing: no jobspy description',
    location: r.location ?? '',
    publishedAt: r.date_posted ?? undefined,
    source: r.site ?? 'jobspy',
  };
}

let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => { input += c; });
process.stdin.on('end', () => {
  try {
    const req = JSON.parse(input);
    if (req.method !== 'collect') throw new Error(`unknown method ${req.method}`);
    const records = Array.isArray(req.params?.records) ? req.params.records : [];
    const result = records
      .filter((r) => r && typeof r.title === 'string' && r.title && r.job_url)
      .map(toPosting);
    process.stdout.write(JSON.stringify({ ok: true, result }));
  } catch (e) {
    process.stdout.write(JSON.stringify({ ok: false, error: String(e && e.message ? e.message : e) }));
  }
});
