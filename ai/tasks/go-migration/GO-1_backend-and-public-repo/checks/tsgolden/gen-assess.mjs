import { writeFileSync } from 'node:fs';
import { assess } from './src/providers.ts';
import { contextVersion } from './src/context.js';

const fixture = {
  job: { id: 1, title: 'Chief Technology Officer', company: 'Acme',
    description: 'We need a CTO with Go experience and team leadership. Kubernetes required.',
    descriptionStatus: 'available structured description', location: 'Remote' },
  sources: [
    { id: 'cv', reference: 'cv', retrievedAt: '2026-10-01T00:00:00.000Z', hash: 'h1', status: 'available',
      category: 'experience', scope: 'CV',
      text: 'Led backend teams building Go services at scale.\n\nShipped Rust tooling and mentored engineers.' },
  ],
  preferences: {},
  reviewNotes: [],
  rawModelOutput: {
    score: 72,
    summary: 'Strong Go and leadership background; Kubernetes not evidenced.',
    evidence: [ { claim: 'Has Go backend leadership experience', evidenceId: 'E1' } ],
    gaps: ['Kubernetes depth unclear'],
    confirmedMismatches: [ { reason: 'Kubernetes not documented in candidate evidence', evidenceId: 'E2', jobRequirement: 'Kubernetes required', kind: 'explicit-absence' } ],
    candidateUnknowns: [],
    employerUnknowns: ['Team size not stated'],
  },
};

const payload = { preferences: fixture.preferences, reviewNotes: fixture.reviewNotes, sources: fixture.sources };
let ctx = { ...payload, version: contextVersion(payload), createdAt: '2026-10-01T00:00:00.000Z' };
let saved = null;
const store = {
  get: () => fixture.job,
  context: () => ctx,
  saveContext: (c) => { ctx = c; },
  assessment: (_id, data) => { saved = data; },
  assessmentFailure: (_id, msg) => { throw new Error('assessmentFailure: ' + msg); },
  resolveId: (n) => n,
};
const injectedInfer = async () => fixture.rawModelOutput;
const result = await assess(store, 1, undefined, injectedInfer);
const schema = {
  score: result.score, summary: result.summary,
  evidence: result.evidence, gaps: result.gaps,
  confirmedMismatches: result.confirmedMismatches,
  candidateUnknowns: result.candidateUnknowns,
  employerUnknowns: result.employerUnknowns,
};
writeFileSync(process.argv[2] + '/assess-fixture.json', JSON.stringify(fixture, null, 2) + '\n');
writeFileSync(process.argv[2] + '/assess-golden.json', JSON.stringify(schema, null, 2) + '\n');
console.log(JSON.stringify(schema, null, 2));
