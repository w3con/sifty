// Generate the Stage 2 application-answer preparation parity fixture and golden from the real
// TypeScript generateApplication, with a stub store and an injected model output. Run from the app
// root (it imports ./src/...), with the output directory as argv[2]:
//   FCTO_STATE_DIR=.local/gen-prepare node_modules/.bin/tsx <this> <sifty-testdata-dir>
import { readFileSync, writeFileSync } from 'node:fs';
import { generateApplication } from './src/application-generation.ts';
import { contextVersion } from './src/context.ts';

const OUT = process.argv[2];
const PROMPT = readFileSync(new URL('../prompts/application-preparation.md', import.meta.url), 'utf8');

const job = {
  id: 1, title: 'Chief Technology Officer', company: 'Acme',
  originalUrl: 'https://acme.test/apply',
  description: 'We need a CTO with Go experience and team leadership. Relocation support available. Kubernetes a plus.',
  descriptionStatus: 'available structured description', location: 'Remote',
};

const sources = [
  { id: 'cv', reference: 'cv', retrievedAt: '2026-10-01T00:00:00.000Z', hash: 'h-cv', status: 'available',
    category: 'experience', scope: 'CV',
    text: 'Led backend teams building Go services at scale.\n\nShipped Rust tooling and mentored engineers.' },
  { id: 'profile', reference: 'profile', retrievedAt: '2026-10-01T00:00:00.000Z', hash: 'h-profile', status: 'available',
    category: 'public-profile', scope: 'Profile',
    text: 'Open to relocation within Europe.\n\nFifteen years in engineering leadership.' },
];
const payload = { preferences: {}, reviewNotes: [], sources };
const ctx = { ...payload, version: contextVersion(payload), createdAt: '2026-10-01T00:00:00.000Z' };

const q = (o) => ({
  options: [], required: false, maxLength: null, accept: '', sourceUrl: job.originalUrl,
  origin: 'site', language: 'und', signature: `sig-${o.id}`, controlName: o.id,
  value: o.type === 'checkbox' ? [] : '', revision: 0, edited: false, unknowns: [], evidence: [], ...o,
});
const baseFields = [
  q({ id: 'q-cover', label: 'Cover letter', type: 'textarea' }),
  q({ id: 'q-relocate', label: 'Willing to relocate?', type: 'radio', options: ['Yes', 'No'] }),
  q({ id: 'q-skills', label: 'Select your skills', type: 'checkbox', options: ['Go', 'Rust', 'Kubernetes'] }),
  q({ id: 'q-salary', label: 'Salary expectation', type: 'text' }),
  q({ id: 'q-start', label: 'Earliest start date', type: 'text' }),
  q({ id: 'q-cv', label: 'Upload CV', type: 'file' }),
  q({ id: 'q-done', label: 'Already answered', type: 'textarea', value: 'done' }),
  q({ id: 'q-edited', label: 'Edited by owner', type: 'textarea', value: 'owner text', edited: true }),
  q({ id: 'q-uncertain', label: 'Unclear wording', type: 'textarea', labelStatus: 'uncertain' }),
];
const baseDraft = {
  jobId: 1, revision: 3, fields: baseFields, selectedFile: 'profile.pdf',
  questionnaire: { url: job.originalUrl, capturedAt: '2026-10-01T00:00:00.000Z', language: 'en',
    status: 'partial', gaps: [], fields: [], family: 'greenhouse' },
  provenance: null, notes: [],
};

const modelOutput = {
  language: 'en', selectedFile: 'cv.pdf',
  answers: [
    { id: 'q-cover', value: 'I have led Go backend teams and shipped Rust tooling.', evidenceIds: ['E1'], unknowns: [] },
    { id: 'q-relocate', value: 'Yes', evidenceIds: ['E2'], unknowns: [] },
    { id: 'q-skills', value: ['Go', 'Rust'], evidenceIds: ['E3'], unknowns: [] },
    { id: 'q-salary', value: '120000 EUR', evidenceIds: [], unknowns: [] },
  ],
};

const drafts = new Map([[1, JSON.stringify(baseDraft)]]);
const store = {
  get: () => job,
  context: () => ctx,
  resolveId: (n) => n,
  db: {
    exec: () => {},
    prepare: (sql) => ({
      get: (...a) => (/application_drafts/.test(sql) && drafts.has(a[0]) ? { data: drafts.get(a[0]) } : undefined),
      run: (...a) => { if (/INSERT INTO application_drafts/.test(sql)) drafts.set(a[0], a[1]); },
    }),
  },
};

const injectedInfer = async () => modelOutput;
const result = await generateApplication(store, 1, 'openai', undefined, injectedInfer);

// Reconstruct the selected-field answers from the written draft, mirroring the selection filter.
const selected = baseFields.filter(f => !f.removed && f.labelStatus !== 'uncertain' && f.type !== 'file'
  && (!f.edited && !(Array.isArray(f.value) ? f.value.length : f.value.trim())));
const byId = new Map(result.fields.map(f => [f.id, f]));
const answers = selected.map(sf => {
  const f = byId.get(sf.id);
  return { id: f.id, value: f.value, unknowns: f.unknowns, evidence: f.evidence };
});

const fixture = {
  prompt: PROMPT,
  questionnaireLanguage: baseDraft.questionnaire.language,
  provider: { model: 'openai/gpt-6.1-sol', contextChars: 50000 },
  job: { id: job.id, title: job.title, company: job.company, description: job.description, location: job.location },
  context: { preferences: payload.preferences, reviewNotes: payload.reviewNotes,
    sources: sources.map(s => ({ id: s.id, status: s.status, category: s.category, scope: s.scope, text: s.text })) },
  fields: baseFields,
  modelOutput,
};
const golden = { language: result.provenance.language, selectedFile: result.selectedFile, answers };

writeFileSync(`${OUT}/prepare-fixture.json`, JSON.stringify(fixture, null, 2) + '\n');
writeFileSync(`${OUT}/prepare-golden.json`, JSON.stringify(golden, null, 2) + '\n');
console.log(JSON.stringify(golden, null, 2));
