// Presentation adapter for the stored Quick Match results. This module performs no requests and
// carries no scoring rules: every number, status and list line below is copied from the backend's
// stored result (`job.fit`, the two-score result of spec section 14). It only gives those values
// names, labels and display buckets. Two parts remain from the single-score projection
// (`job.quickMatch`): its vacancy-side facts feed the fact strip, and its types name them.


// Single-score projection: kept for the vacancy-side facts strip only.
export type MatchGroup = 'role' | 'professional' | 'compensation' | 'geography' | 'conditions';
export type MatchStatus = 'scored' | 'unscored' | 'stale';
export type MatchApplicability = 'applicable' | 'not_applicable' | 'unknown';
export type MatchReliability = 'usable' | 'needs_corroboration' | 'unusable';
export type MatchSection = 'required' | 'preferred' | 'company_context' | 'unknown';
export type FitKey = 'strong' | 'partial' | 'weak' | 'unscored';
export type ScoreBucket = FitKey | 'previous' | 'unavailable';

export interface MatchProvenance {
  kind?: string;
  reference?: string;
}

export interface MatchComparison {
  group: MatchGroup;
  criterion: string;
  profileField?: string;
  profileValue?: string;
  profileNumber?: number;
  skillId?: string;
  profileProvenance?: MatchProvenance | null;
  jobFact?: string;
  jobValue?: string;
  jobNumber?: number;
  section?: MatchSection;
  evidenceField?: string;
  evidenceQuote?: string;
  reliability?: MatchReliability;
  applicability: MatchApplicability;
  comparable: boolean;
  utility?: number;
  weight?: number;
  contribution?: number;
  reason: string;
}

export interface MatchContribution {
  group: MatchGroup;
  weight: number;
  evidenceFraction: number;
  utility: number;
  weighted: number;
  status: MatchReliability;
}

export interface MatchUnknown {
  group: MatchGroup;
  fact: string;
  section?: MatchSection;
  kind: string;
  quote?: string;
  profileEdit?: string;
}

export interface MatchConflict {
  group: MatchGroup;
  fact: string;
  kind: string;
  detail: string;
}

export interface MatchResult {
  status: MatchStatus;
  score: number | null;
  rawScore: number | null;
  coverage: {weighted: number; groups: Partial<Record<MatchGroup, number>>};
  applicability: Partial<Record<MatchGroup, MatchApplicability>>;
  reliability: {usable: number; needsCorroboration: number; unusable: number};
  contributions: MatchContribution[];
  comparisons: MatchComparison[];
  unknowns: MatchUnknown[];
  conflicts: MatchConflict[];
  reasons: string[];
  provenance: {
    profileId: string;
    profileRevision: number;
    rulesVersion: string;
    rulesIdentity?: string;
    factsHash: string;
    rawSourceHash: string;
    extractorVersion: string;
    schemaVersion: string;
  };
}

export interface MatchSummary {
  status: MatchStatus;
  score: number | null;
  rawScore: number | null;
  coverage: number;
  unknowns: number;
  conflicts: number;
}

export interface FactsEvidence {reliability?: MatchReliability}
export interface VacancyFactsView {
  schemaVersion: string;
  extractorVersion: string;
  factsHash: string;
  rawSourceHash: string;
  role?: {title?: {value: string; evidence?: FactsEvidence}};
  compensation?: {cash?: {min?: number; max?: number; currency?: string; period?: string; taxBasis?: string; basis?: string; evidence?: FactsEvidence}};
  geography?: {hiringLocations?: {label: string; country?: string; city?: string; evidence?: FactsEvidence}[]};
  conditions?: {
    workModes?: {mode: string; evidence?: FactsEvidence}[];
    engagement?: {type: string; evidence?: FactsEvidence};
    hours?: {mode?: string; weekly?: number; weekMin?: number; weekMax?: number; evidence?: FactsEvidence};
  };
}

export interface QuickMatchView {
  result: MatchResult;
  summary: MatchSummary;
  stale: boolean;
  staleReasons?: string[];
  facts?: VacancyFactsView | null;
}

export type QuickState = 'missing' | 'error' | 'stale' | 'unscored' | 'scored';

export function quickState(view?: QuickMatchView | null, error?: string): QuickState {
  if (error) return 'error';
  if (!view) return 'missing';
  if (view.stale || view.result.status === 'stale') return 'stale';
  if (view.result.status === 'unscored' || view.result.score === null) return 'unscored';
  return 'scored';
}

// ---------------------------------------------------------------------------------------------
// Two-score result: Fits you (the job against the owner's goals) and You fit (the owner's
// confirmed skills and experience against the stated requirements).
// ---------------------------------------------------------------------------------------------

export type ScoreStatus = 'scored' | 'unscored' | 'no_requirements' | 'pending' | 'unavailable' | 'off';
export type Verdict = 'match' | 'partial' | 'no' | 'unclear';
export type RequirementKind = 'skill' | 'experience' | 'qualification' | 'language' | 'authorization' | 'other';

export interface FitCounts {match: number; partial: number; no: number; unclear: number}

// One factual line of a list. Goal lines carry a criterion and the compared values; requirement
// lines carry the requirement text (`subject`), its verdict and the profile fact it rests on.
export interface FitLine {
  origin: 'goal' | 'requirement';
  group?: string;
  criterion?: string;
  kind?: RequirementKind;
  level?: 'required' | 'preferred';
  verdict?: Verdict;
  subject?: string;
  alternatives?: string[];
  profileField?: string;
  profileValue?: string;
  jobValue?: string;
  reason?: string;
  quote?: string;
}

export interface FitResult {
  source: 'usable' | 'unavailable';
  fitsYou: {status: ScoreStatus; score: number | null; rawScore: number | null};
  youFit: {status: ScoreStatus; score: number | null; rawScore: number | null; counts: FitCounts};
  // True while the model's matching pass is off: no You fit is drawn, whatever its status.
  matchingOff?: boolean;
  lists: {matches: FitLine[]; mismatches: FitLine[]; missing: FitLine[]; toClarify: FitLine[]};
}

export interface FitView {
  result: FitResult;
  stale: boolean;
  staleReasons?: string[];
}

// What GET /api/quick-match/analysis says about the background analysis. The panel uses it only
// to say why a number is not there yet; it shows no counts, limits or queue.
export interface AnalysisStatus {
  state: 'idle' | 'working' | 'paused' | 'blocked' | 'waiting' | 'limit' | 'no_profile';
  modelAvailable: boolean;
}

export function fitKeyForScore(score: number): FitKey {
  if (score >= 75) return 'strong';
  if (score >= 55) return 'partial';
  return 'weak';
}

export function bucketLabel(bucket: FitKey): string {
  switch (bucket) {
    case 'strong': return 'Strong';
    case 'partial': return 'Partial';
    case 'weak': return 'Weak';
    case 'unscored': return 'Not scored';
  }
}

export function countsText(c: FitCounts): string {
  return `${c.match} match · ${c.partial} partial · ${c.no} no · ${c.unclear} unclear`;
}

// One score as shown in the panel header and in a catalogue row. `number` is a digit string, or a
// dash / ellipsis when there is none; `note` says in words why there is no number.
export interface ScoreCell {
  number: string;
  bucket: FitKey;
  label: string;
  note: string;
  title: string;
  previous: boolean;
}

const NO_NUMBER = {number: '—', bucket: 'unscored' as FitKey, label: '', previous: false};

export function fitsYouCell(fit?: FitView | null): ScoreCell {
  if (!fit) return {...NO_NUMBER, note: 'Not compared yet', title: 'Fits you: not compared yet'};
  const r = fit.result.fitsYou;
  if (fit.result.source === 'unavailable' || r.status === 'unavailable') {
    return {...NO_NUMBER, note: 'Full job description is unavailable', title: 'Fits you: full job description is unavailable'};
  }
  if (r.status !== 'scored' || r.score === null) {
    return {...NO_NUMBER, note: 'Your profile states no goals to compare', title: 'Fits you: your profile states no goals to compare'};
  }
  const bucket = fitKeyForScore(r.score);
  return {number: String(r.score), bucket, label: bucketLabel(bucket), note: '', previous: fit.stale, title: `Fits you ${r.score}: ${bucketLabel(bucket)}`};
}

// You fit is not shown while the model's matching pass is switched off (spec 14.14): the backend
// then gives it the status `off`, and neither the panel nor a catalogue row draws it.
export function youFitOff(fit?: FitView | null): boolean {
  return fit?.result.matchingOff === true || fit?.result.youFit.status === 'off';
}

// The requirement lines of a result, the ones a view shows under "Requirements" while You fit is
// off: required before preferred, each list keeping its order.
export function requirementLines(fit: FitView): FitLine[] {
  const {matches, mismatches, missing} = fit.result.lists;
  const all = [...matches, ...mismatches, ...missing].filter(l => l.origin === 'requirement');
  const rank = (l: FitLine) => (l.level === 'required' ? 0 : 1);
  return all.map((l, i) => ({l, i})).sort((a, b) => rank(a.l) - rank(b.l) || a.i - b.i).map(x => x.l);
}

// The mark of a requirement line while You fit is off: a state only where code decided one.
export function requirementMark(line: FitLine): string {
  if (line.verdict === 'match' || line.verdict === 'partial') return '✓';
  if (line.verdict === 'no') return '✕';
  return '·';
}

export function youFitCell(fit?: FitView | null, analysis?: AnalysisStatus | null): ScoreCell {
  if (!fit) return {...NO_NUMBER, note: 'Not compared yet', title: 'You fit: not compared yet'};
  const r = fit.result.youFit;
  switch (r.status) {
    case 'off':
      return {...NO_NUMBER, note: '', title: ''};
    case 'scored': {
      if (r.score === null) break;
      const bucket = fitKeyForScore(r.score);
      return {number: String(r.score), bucket, label: bucketLabel(bucket), note: '', previous: fit.stale, title: `You fit ${r.score}: ${countsText(r.counts)}`};
    }
    case 'no_requirements':
      return {...NO_NUMBER, note: 'No requirements stated', title: 'You fit: no requirements stated'};
    case 'pending': {
      // Say "analysing" only while that is true; a model that is not running is not analysing.
      const working = pendingNotice(analysis).kind === 'working';
      const note = working ? 'Analysing requirements…' : 'Not analysed yet';
      return {...NO_NUMBER, number: '…', note, title: `You fit: ${note.toLowerCase()}`};
    }
  }
  return {...NO_NUMBER, note: 'Full job description is unavailable', title: 'You fit: full job description is unavailable'};
}

// Catalogue order, as in spec section 14.3 and the backend's SortByFit: a current Fits you
// number first (higher first), then You fit (higher first; pending or no requirements counts as
// the midpoint), then the job id. A result that is stale or has no Fits you number follows every
// current one.
export interface FitSortKey {hasFits: boolean; fits: number; you: number}

export function fitSortKey(fit?: FitView | null): FitSortKey {
  const key: FitSortKey = {hasFits: false, fits: 0, you: 0.5};
  if (!fit || fit.stale) return key;
  const f = fit.result.fitsYou, y = fit.result.youFit;
  if (f.status === 'scored' && f.rawScore !== null) { key.hasFits = true; key.fits = f.rawScore; }
  if (y.status === 'scored' && y.rawScore !== null) key.you = y.rawScore;
  return key;
}

// Orders two results as the catalogue does; the caller breaks a tie by job id.
export function compareByFit(a?: FitView | null, b?: FitView | null): number {
  const x = fitSortKey(a), y = fitSortKey(b);
  if (x.hasFits !== y.hasFits) return x.hasFits ? -1 : 1;
  if (x.hasFits && x.fits !== y.fits) return y.fits - x.fits;
  return y.you - x.you;
}

// The Fit filter in the catalogue works on Fits you.
export function fitBucketOf(fit?: FitView | null): FitKey {
  const r = fit?.result.fitsYou;
  if (!fit || fit.stale || r?.status !== 'scored' || r.score === null) return 'unscored';
  return fitKeyForScore(r.score);
}

const CRITERION_NOUN: Record<string, string> = {
  'role.title': 'Role',
  'compensation.cash': 'Pay',
  'geography.location': 'Location',
  'geography.authorization': 'Work authorization',
  'conditions.work_mode': 'Work mode',
  'conditions.engagement': 'Contract type',
  'conditions.hours': 'Hours',
  'conditions.timezone': 'Time zone',
};

const CRITERION_CLARIFY: Record<string, string> = {
  'role.title': 'Role',
  'compensation.cash': 'Salary',
  'geography.location': 'Location',
  'geography.authorization': 'Work authorization',
  'conditions.work_mode': 'Work mode',
  'conditions.engagement': 'Contract type',
  'conditions.hours': 'Hours',
  'conditions.timezone': 'Time zone',
};

// What the profile says about a goal, worded for the line it sits on.
const GOAL_PROFILE_PREFIX: Record<string, string> = {
  'role.title': 'your target ',
  'compensation.cash': 'your minimum ',
  'geography.location': 'you want ',
  'conditions.work_mode': 'you want ',
  'conditions.engagement': 'you want ',
  'conditions.hours': 'you want ',
  'conditions.timezone': 'you want ',
};

export interface LineText {
  main: string;
  // The members of an "any one of these" demand, joined for the line: "Kotlin · Python · Go".
  alternatives?: string;
  note?: string;
  // The note in full, when it was shortened for the line.
  fullNote?: string;
  partial: boolean;
  preferred: boolean;
}

// A profile fact can be long (a computed experience fact lists the roles it was computed from).
// The line keeps it to about one line of text and puts the whole of it in the hover text.
const NOTE_LIMIT = 72;

function shortNote(text: string): {note: string; fullNote?: string} {
  if (text.length <= NOTE_LIMIT) return {note: text};
  const cut = text.slice(0, NOTE_LIMIT);
  const space = cut.lastIndexOf(' ');
  return {note: `${(space > NOTE_LIMIT / 2 ? cut.slice(0, space) : cut).replace(/[\s,;:]+$/, '')}…`, fullNote: text};
}

// The one place a list line is turned into words. A goal line reads "Work mode: Remote — you want
// remote"; a requirement line reads "Go — Go (expert)" (the requirement, then the profile fact
// that meets it). The model's own explanation of a verdict is never shown: for a requirement the
// profile lacks it only restates the line; a fact that contradicts a requirement is shown as the note.
export function lineText(line: FitLine): LineText {
  if (line.origin === 'goal') {
    const noun = CRITERION_NOUN[line.criterion ?? ''] ?? 'Goal';
    const job = (line.jobValue ?? '').trim();
    const profile = (line.profileValue ?? '').trim();
    return {
      main: job ? `${noun}: ${job}` : noun,
      ...(profile ? shortNote(`${GOAL_PROFILE_PREFIX[line.criterion ?? ''] ?? ''}${profile}`) : {}),
      partial: false,
      preferred: false,
    };
  }
  const note = (line.profileValue ?? '').trim();
  return {
    main: (line.subject ?? '').trim() || 'Requirement',
    ...(line.alternatives?.length ? {alternatives: line.alternatives.join(' · ')} : {}),
    ...(note ? shortNote(note) : {}),
    partial: line.verdict === 'partial',
    preferred: line.level === 'preferred',
  };
}

// "? Salary", "? Contract type": a condition the owner cares about that the vacancy does not state.
export function clarifyLabel(line: FitLine): string {
  return CRITERION_CLARIFY[line.criterion ?? ''] ?? 'Detail';
}

// Profile sections a missing requirement can be answered in. A requirement of another kind has no
// section to point to: the profile has no education or free-form section.
const MISSING_SECTION: Partial<Record<RequirementKind, {id: string; label: string}>> = {
  skill: {id: 'profile-skills', label: 'skills'},
  language: {id: 'profile-languages', label: 'languages'},
  experience: {id: 'profile-experience', label: 'experience'},
  authorization: {id: 'profile-geography', label: 'work authorization'},
};

// The skills a requirement line offers to add to the profile, one by one. A requirement often lists
// several in one line ("Kubernetes, IaC and observability") and a group of alternatives lists them
// as members; each is its own skill, so a click adds exactly one and never the whole line. Only a
// skill (not a language, an experience or a qualification) is offered. Names are split at commas,
// semicolons, "and", "or" and "&"; a slash stays inside a name (CI/CD, Firebase/GCP), and a leading
// word such as "strong" or "experience with" and a trailing one such as "expertise" or "skills" are dropped.
const SKILL_SEPARATORS = /\s*(?:,|;|&|\s\band\b\s|\sor\b\s)\s*/i;
const SKILL_SUFFIX = /\s+(?:expertise|experience|skills?|proficiency|knowledge|familiarity|background)$/i;
const SKILL_PREFIX = /^(?:(?:strong|deep|solid|proven|good|excellent|advanced|expert|hands-on)\s+)?(?:(?:experience|knowledge|proficiency|familiarity|expertise)\s+(?:with|in|of)\s+)?(?:(?:strong|deep|solid|proven|good|excellent|advanced|expert|hands-on)\s+)?/i;
const MAX_SKILL_OFFERS = 6;

export function quickAddSkillNames(line: FitLine): string[] {
  if (line.origin !== 'requirement' || line.kind !== 'skill') return [];
  if (line.profileField === 'skillsAbsent') return []; // the owner already said they do not have it
  const parts = line.alternatives && line.alternatives.length > 0 ? line.alternatives : (line.subject ?? '').split(SKILL_SEPARATORS);
  const seen = new Set<string>();
  const names: string[] = [];
  for (const part of parts) {
    const name = part.replace(/\s+/g, ' ').trim().replace(SKILL_PREFIX, '').replace(SKILL_SUFFIX, '').trim();
    const key = name.toLowerCase();
    if (name !== '' && name.length <= 80 && !seen.has(key)) {
      seen.add(key);
      names.push(name);
    }
  }
  return names.length > MAX_SKILL_OFFERS ? [] : names;
}

export function missingSection(line: FitLine): {id: string; label: string} | undefined {
  return line.kind ? MISSING_SECTION[line.kind] : undefined;
}

// Why a number is not there yet, from the worker's state. The wording names no queue, limit or
// pass: a model that is not running is a retry message, a provider that may not be used is a
// broken one, anything else is just not analysed yet.
export type PendingNotice =
  | {kind: 'working'; text: string}
  | {kind: 'retry' | 'broken' | 'info'; text: string}
  | {kind: 'profile'; text: string};

export function pendingNotice(analysis?: AnalysisStatus | null): PendingNotice {
  switch (analysis?.state) {
    case 'waiting':
      return {kind: 'retry', text: 'Requirements could not be analysed — the local model is not running.'};
    case 'blocked':
      return {kind: 'broken', text: 'Requirements are analysed only by a local model. The configured provider is not local.'};
    case 'no_profile':
      return {kind: 'profile', text: 'Requirements are compared with your profile once it is saved.'};
    case 'paused':
    case 'limit':
      return {kind: 'info', text: 'Requirements have not been analysed yet.'};
    default:
      return {kind: 'working', text: 'Analysing requirements…'};
  }
}


export interface FactItem {
  icon: 'pin' | 'money' | 'mode' | 'case' | 'clock';
  label: string;
  value?: string;
  note?: string;
}

// The strip shows vacancy-side facts from the validated facts production that belongs to this
// current result, independent of whether the profile has a preference to compare them with. It
// never invents a country, office permission, seniority or guaranteed salary.
export function buildFactStrip(view: QuickMatchView | null | undefined, rawLocation?: string | null): FactItem[] {
  const state = quickState(view);
  const comparisons = state === 'scored' || state === 'unscored' ? (view!.result.comparisons ?? []) : [];
  const comparableValue = (criterion: string): string | undefined => {
    const comparison = comparisons.find(item => item.criterion === criterion && item.reliability === 'usable' && (item.jobValue ?? '').trim() !== '');
    return comparison?.jobValue?.trim() || undefined;
  };
  const facts = view?.facts;
  const usable = (evidence?: FactsEvidence) => evidence?.reliability === 'usable';
  const money = facts?.compensation?.cash;
  const moneyValue = money && usable(money.evidence) ? cashText(money) : undefined;
  const modes = facts?.conditions?.workModes?.filter(item => usable(item.evidence)).map(item => workModeLabel(item.mode)).filter(Boolean);
  const engagement = facts?.conditions?.engagement;
  const hours = facts?.conditions?.hours;
  const locationFacts = facts?.geography?.hiringLocations?.filter(item => usable(item.evidence)).map(item => item.label.trim()).filter(Boolean) ?? [];
  const locationComparison = comparisons.find(item => item.criterion === 'geography.location');
  const ambiguousLocation = (view?.result.conflicts ?? []).some(item => item.fact === 'hiringLocations' || item.fact === 'location') ||
    (!!locationComparison?.reliability && locationComparison.reliability !== 'usable');
  const raw = rawLocation?.trim() || undefined;
  const location = locationFacts.length ? [...new Set(locationFacts)].join(' · ') : (comparableValue('geography.location') ?? raw);
  const locationNote = location
    ? ambiguousLocation
      ? 'source metadata is ambiguous; it is not treated as a work restriction or permission'
      : 'as stated in the listing; not proof of work authorization'
    : undefined;
  return [
    {icon: 'pin', label: 'Location', value: location, note: locationNote},
    {icon: 'money', label: 'Compensation', value: moneyValue ?? comparableValue('compensation.cash'), note: moneyValue ? 'declared vacancy figure; not a guaranteed offer' : undefined},
    {icon: 'mode', label: 'Work mode', value: modes?.length ? [...new Set(modes)].join(' · ') : comparableValue('conditions.work_mode')},
    {icon: 'case', label: 'Engagement', value: engagement && usable(engagement.evidence) ? engagementLabel(engagement.type) : comparableValue('conditions.engagement')},
    {icon: 'clock', label: 'Hours', value: hours && usable(hours.evidence) ? hoursText(hours) : comparableValue('conditions.hours')},
  ];
}

function cashText(cash: {min?: number; max?: number; currency?: string; period?: string; taxBasis?: string}): string | undefined {
  const amount = cash.min !== undefined && cash.max !== undefined
    ? `${numberText(cash.min)}–${numberText(cash.max)}`
    : cash.max !== undefined
      ? `up to ${numberText(cash.max)}`
      : cash.min !== undefined
        ? `from ${numberText(cash.min)}`
        : undefined;
  if (!amount) return undefined;
  const unit = [cash.currency, cash.period ? `per ${cash.period}` : '', cash.taxBasis].filter(Boolean).join(' ');
  return `${amount}${unit ? ` · ${unit}` : ' · currency, period or tax basis not stated'}`;
}

function numberText(value: number): string { return Number.isFinite(value) ? value.toLocaleString() : '—'; }
function workModeLabel(mode: string): string | undefined {
  switch (mode) { case 'remote': return 'Remote'; case 'hybrid': return 'Hybrid'; case 'office': return 'Office'; default: return mode || undefined; }
}
function engagementLabel(type: string): string | undefined {
  switch (type) { case 'employment': return 'Employment'; case 'contract': return 'Contract'; default: return type || undefined; }
}
function hoursText(hours: {mode?: string; weekly?: number; weekMin?: number; weekMax?: number}): string | undefined {
  const parts: string[] = [];
  if (hours.mode === 'full_time') parts.push('Full-time');
  else if (hours.mode === 'part_time') parts.push('Part-time');
  else if (hours.mode) parts.push(hours.mode);
  if (hours.weekMin !== undefined && hours.weekMax !== undefined) parts.push(`${numberText(hours.weekMin)}–${numberText(hours.weekMax)} h/week`);
  else if (hours.weekly !== undefined) parts.push(`${numberText(hours.weekly)} h/week`);
  return parts.length ? parts.join(' · ') : undefined;
}
