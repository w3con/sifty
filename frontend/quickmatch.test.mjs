// Checks of the pure presentation functions for the two-score result. Run from the repository root:
//   node --test frontend/quickmatch.test.mjs
// (Node 22.18 or newer strips the TypeScript types itself; nothing is built or installed.)
import test from 'node:test';
import assert from 'node:assert/strict';
import {
  bucketLabel, clarifyLabel, compareByFit, countsText, fitBucketOf, fitsYouCell, lineText, missingSection,
  pendingNotice, quickAddSkillNames, requirementLines, requirementMark, youFitCell, youFitOff,
} from './quickmatch.ts';

const counts = (match = 0, partial = 0, no = 0, unclear = 0) => ({match, partial, no, unclear});
function view({fits = 80, you = 'pending', youScore = null, source = 'usable', stale = false, c = counts()} = {}) {
  const fitsOn = source === 'usable' && fits !== null;
  return {
    stale,
    result: {
      source,
      fitsYou: fitsOn ? {status: 'scored', score: fits, rawScore: fits / 100} : {status: source === 'unavailable' ? 'unavailable' : 'unscored', score: null, rawScore: null},
      youFit: {status: source === 'unavailable' ? 'unavailable' : you, score: youScore, rawScore: youScore === null ? null : youScore / 100, counts: c},
      lists: {matches: [], mismatches: [], missing: [], toClarify: []},
    },
  };
}

test('both numbers and their buckets', () => {
  const v = view({fits: 82, you: 'scored', youScore: 54, c: counts(2, 1, 0, 3)});
  assert.equal(fitsYouCell(v).number, '82');
  assert.equal(fitsYouCell(v).bucket, 'strong');
  assert.equal(youFitCell(v).number, '54');
  assert.equal(youFitCell(v).bucket, 'weak');
  assert.equal(bucketLabel('partial'), 'Partial');
  assert.equal(countsText(counts(2, 1, 0, 3)), '2 match · 1 partial · 0 no · 3 unclear');
});

test('a score without a number says why, in words', () => {
  assert.equal(youFitCell(view({you: 'no_requirements'})).note, 'No requirements stated');
  assert.equal(youFitCell(view({you: 'pending'})).note, 'Analysing requirements…');
  assert.equal(youFitCell(view({you: 'pending'}), {state: 'waiting', modelAvailable: false}).note, 'Not analysed yet');
  const gone = view({source: 'unavailable'});
  assert.equal(fitsYouCell(gone).number, '—');
  assert.equal(youFitCell(gone).number, '—');
  assert.equal(fitsYouCell(gone).note, 'Full job description is unavailable');
  assert.equal(fitsYouCell(null).note, 'Not compared yet');
});

test('a model that is not running is a retry message, not "analysing"', () => {
  assert.deepEqual(pendingNotice({state: 'waiting', modelAvailable: false}).kind, 'retry');
  assert.equal(pendingNotice({state: 'waiting', modelAvailable: false}).text, 'Requirements could not be analysed — the local model is not running.');
  assert.equal(pendingNotice({state: 'blocked', modelAvailable: false}).kind, 'broken');
  assert.equal(pendingNotice({state: 'working', modelAvailable: true}).kind, 'working');
  assert.equal(pendingNotice(null).kind, 'working');
  for (const state of ['paused', 'limit']) {
    const text = pendingNotice({state, modelAvailable: true}).text;
    assert.ok(!/limit|queue|paused|cap|attempt/i.test(text), `no mechanics in "${text}"`);
  }
});

test('catalogue order: Fits you, then You fit (pending counts as the midpoint), then id; stale last', () => {
  const rows = [
    {id: 1, fit: view({fits: 70, you: 'scored', youScore: 40})},
    {id: 2, fit: view({fits: 70, you: 'scored', youScore: 90})},
    {id: 3, fit: view({fits: 70, you: 'pending'})},      // midpoint 0.5
    {id: 4, fit: view({fits: 90, you: 'pending'})},
    {id: 5, fit: view({source: 'unavailable'})},
    {id: 6, fit: view({fits: 99, you: 'scored', youScore: 99, stale: true})},
    {id: 7, fit: null},
    {id: 8, fit: view({fits: 70, you: 'no_requirements'})}, // midpoint 0.5, after 3 by id
  ];
  const order = [...rows].sort((a, b) => compareByFit(a.fit, b.fit) || a.id - b.id).map(r => r.id);
  assert.deepEqual(order, [4, 2, 3, 8, 1, 5, 6, 7]);
});

test('Fit filter works on Fits you and never on a stale number', () => {
  assert.equal(fitBucketOf(view({fits: 75})), 'strong');
  assert.equal(fitBucketOf(view({fits: 74})), 'partial');
  assert.equal(fitBucketOf(view({fits: 54})), 'weak');
  assert.equal(fitBucketOf(view({fits: 80, stale: true})), 'unscored');
  assert.equal(fitBucketOf(view({source: 'unavailable'})), 'unscored');
});

test('list lines read as plain words and carry no parser internals', () => {
  const goal = {origin: 'goal', criterion: 'conditions.work_mode', jobValue: 'Office in London', profileValue: 'remote'};
  assert.deepEqual(lineText(goal), {main: 'Work mode: Office in London', note: 'you want remote', partial: false, preferred: false});
  const pay = {origin: 'goal', criterion: 'compensation.cash', jobValue: 'EUR 70k', profileValue: '100000 EUR/year'};
  assert.equal(lineText(pay).note, 'your minimum 100000 EUR/year');
  const req = {origin: 'requirement', kind: 'skill', level: 'preferred', verdict: 'partial', subject: 'Go', profileField: 'skills', profileValue: 'Go (expert)', reason: 'x'};
  assert.deepEqual(lineText(req), {main: 'Go', note: 'Go (expert)', partial: true, preferred: true});
  const no = {origin: 'requirement', kind: 'language', level: 'required', verdict: 'no', subject: 'Fluent German', profileValue: 'German (A2)', reason: 'Profile states A2.'};
  assert.equal(lineText(no).note, 'German (A2)', 'a contradicting fact is the note; the model\'s reason is never shown');
  const absent = {origin: 'requirement', kind: 'skill', level: 'required', verdict: 'no', subject: 'ML', reason: 'No ML sphere experience listed in facts.'};
  assert.equal(lineText(absent).note, undefined, 'the explanation of an absence adds nothing to the line');
  const long = {origin: 'requirement', kind: 'experience', level: 'required', verdict: 'match', subject: 'Leading teams', profileValue: '9 years in leadership roles, computed from the dates of: Founder & CTO at Validité; Co-Founder & Chief Architect at Pilier Network'};
  const shown = lineText(long);
  assert.ok(shown.note.length <= 73 && shown.note.endsWith('…'), shown.note);
  assert.equal(shown.fullNote, long.profileValue);
  assert.equal(lineText(req).fullNote, undefined);
  assert.equal(clarifyLabel({origin: 'goal', criterion: 'compensation.cash'}), 'Salary');
  assert.equal(clarifyLabel({origin: 'goal', criterion: 'conditions.engagement'}), 'Contract type');
  assert.equal(clarifyLabel({origin: 'goal', criterion: 'conditions.work_mode'}), 'Work mode');
});

test('a missing requirement points to the profile section that can answer it', () => {
  assert.deepEqual(missingSection({origin: 'requirement', kind: 'skill'}), {id: 'profile-skills', label: 'skills'});
  assert.deepEqual(missingSection({origin: 'requirement', kind: 'language'}), {id: 'profile-languages', label: 'languages'});
  assert.deepEqual(missingSection({origin: 'requirement', kind: 'experience'}), {id: 'profile-experience', label: 'experience'});
  assert.equal(missingSection({origin: 'requirement', kind: 'qualification'}), undefined);
});

test('a demand with alternatives names its members on the line; a single demand shows none', () => {
  const group = lineText({origin: 'requirement', subject: 'Backend language', alternatives: ['Kotlin', 'Python', 'Go'], verdict: 'match', profileValue: 'Go (expert)'});
  assert.equal(group.main, 'Backend language');
  assert.equal(group.alternatives, 'Kotlin · Python · Go');
  assert.equal(lineText({origin: 'requirement', subject: 'Terraform', verdict: 'unclear'}).alternatives, undefined);
  assert.equal(lineText({origin: 'requirement', subject: 'Terraform', alternatives: [], verdict: 'unclear'}).alternatives, undefined);
});

test('a requirement line offers each skill it names separately, never the whole line', () => {
  const line = (over) => ({origin: 'requirement', kind: 'skill', subject: 'Kubernetes', ...over});
  assert.deepEqual(quickAddSkillNames(line({})), ['Kubernetes']);
  assert.deepEqual(quickAddSkillNames(line({subject: 'Kubernetes expertise'})), ['Kubernetes']);
  assert.deepEqual(quickAddSkillNames(line({subject: 'Experience with Rust'})), ['Rust']);
  // Lines taken from real vacancies: several skills in one line are several offers.
  assert.deepEqual(quickAddSkillNames(line({subject: 'Kubernetes, IaC and observability'})), ['Kubernetes', 'IaC', 'observability']);
  assert.deepEqual(quickAddSkillNames(line({subject: 'AWS, Kubernetes, Terraform, ArgoCD'})), ['AWS', 'Kubernetes', 'Terraform', 'ArgoCD']);
  assert.deepEqual(quickAddSkillNames(line({subject: 'Python and Postgres'})), ['Python', 'Postgres']);
  assert.deepEqual(quickAddSkillNames(line({subject: 'Strong TypeScript or Node'})), ['TypeScript', 'Node']);
  // A slash stays inside a name, so CI/CD and Firebase/GCP are not torn apart.
  assert.deepEqual(quickAddSkillNames(line({subject: 'Firebase/GCP'})), ['Firebase/GCP']);
  // A group of alternatives offers its members, not the group's name.
  assert.deepEqual(quickAddSkillNames(line({subject: 'Event-driven architecture experience', alternatives: ['Kafka', 'NATS', 'RabbitMQ']})), ['Kafka', 'NATS', 'RabbitMQ']);
  assert.deepEqual(quickAddSkillNames(line({subject: 'Backend language', alternatives: ['Go', 'go', 'Python']})), ['Go', 'Python']);
  // Not skills, or not a requirement, or too many to be a skill list.
  assert.deepEqual(quickAddSkillNames(line({verdict: 'no', profileField: 'skillsAbsent'})), [], 'a skill the owner already said they lack is not offered again');
  assert.deepEqual(quickAddSkillNames(line({kind: 'language'})), []);
  assert.deepEqual(quickAddSkillNames(line({kind: 'experience'})), []);
  assert.deepEqual(quickAddSkillNames({origin: 'goal', criterion: 'role.title'}), []);
  assert.deepEqual(quickAddSkillNames(line({subject: '   '})), []);
  assert.deepEqual(quickAddSkillNames(line({subject: 'A, B, C, D, E, F, G'})), []);
});

test('with the model matching off there is no You fit, and the requirements are one list', () => {
  const line = (subject, verdict, level = 'required') => ({origin: 'requirement', kind: 'skill', level, verdict, subject});
  const goal = {origin: 'goal', criterion: 'role.title'};
  const v = view({fits: 70, you: 'off'});
  v.result.matchingOff = true;
  v.result.lists = {
    matches: [goal, line('Go', 'match', 'preferred')],
    mismatches: [line('Rust', 'no')],
    missing: [line('Kubernetes', 'unclear'), line('Terraform', 'unclear', 'preferred')],
    toClarify: [],
  };
  assert.equal(youFitOff(v), true);
  assert.equal(youFitOff(view({you: 'scored', youScore: 50})), false);
  assert.equal(youFitCell(v).number, '—');
  assert.equal(youFitCell(v).note, '');
  assert.deepEqual(requirementLines(v).map(l => l.subject), ['Rust', 'Kubernetes', 'Go', 'Terraform']);
  assert.deepEqual(requirementLines(v).map(requirementMark), ['✕', '·', '✓', '·']);
  assert.equal(fitsYouCell(v).number, '70');
  // a vacancy not analysed yet, or without requirements, has no You fit either
  const waiting = view({fits: 70, you: 'pending'});
  waiting.result.matchingOff = true;
  assert.equal(youFitOff(waiting), true);
});
