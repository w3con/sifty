// Presentation adapter for the deterministic Quick Match API envelope. This module performs no
// requests and carries no scoring rules: every number, status, contribution and reason below is
// copied from the stored backend result. It only gives those values names, labels and display
// buckets. A stale result is always labelled as a previous result, never as current.

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

export const allGroups: MatchGroup[] = ['role', 'professional', 'compensation', 'geography', 'conditions'];

export function groupLabel(group: MatchGroup): string {
  switch (group) {
    case 'role': return 'Role';
    case 'professional': return 'Professional';
    case 'compensation': return 'Compensation';
    case 'geography': return 'Geography';
    case 'conditions': return 'Conditions';
  }
}

export type QuickState = 'missing' | 'error' | 'stale' | 'unscored' | 'scored';

export function quickState(view?: QuickMatchView | null, error?: string): QuickState {
  if (error) return 'error';
  if (!view) return 'missing';
  if (view.stale || view.result.status === 'stale') return 'stale';
  if (view.result.status === 'unscored' || view.result.score === null) return 'unscored';
  return 'scored';
}

export function fitKeyForScore(score: number): FitKey {
  if (score >= 75) return 'strong';
  if (score >= 55) return 'partial';
  return 'weak';
}

export function fitBucketOf(view?: QuickMatchView | null): FitKey {
  if (quickState(view) !== 'scored') return 'unscored';
  return fitKeyForScore(view!.result.score as number);
}

export function bucketLabel(bucket: FitKey): string {
  switch (bucket) {
    case 'strong': return 'Strong fit';
    case 'partial': return 'Partial fit';
    case 'weak': return 'Weak fit';
    case 'unscored': return 'Not scored';
  }
}

export interface ScorePresentation {
  number: string;
  bucket: ScoreBucket;
  label: string;
  previous: boolean;
  title: string;
}

export function scorePresentation(view?: QuickMatchView | null, error?: string): ScorePresentation {
  const state = quickState(view, error);
  if (state === 'error') {
    return {number: '—', bucket: 'unavailable', label: 'Quick match unavailable', previous: false, title: `Quick match unavailable: ${error}`};
  }
  if (state === 'missing') {
    return {number: '—', bucket: 'unscored', label: 'Not calculated', previous: false, title: 'No Quick Match result has been calculated for this role.'};
  }
  const score = view!.result.score;
  if (state === 'stale') {
    const reasons = staleReasonText(view!);
    return {
      number: score === null ? '—' : String(score),
      bucket: 'previous',
      label: score === null ? 'Previous result' : 'Previous score',
      previous: true,
      title: `Previous Quick Match result, not current: ${reasons}`,
    };
  }
  if (state === 'unscored') {
    return {number: '—', bucket: 'unscored', label: 'Not scored', previous: false, title: 'No comparable evidence; this is not a zero score.'};
  }
  const value = score as number;
  return {number: String(value), bucket: fitKeyForScore(value), label: bucketLabel(fitKeyForScore(value)), previous: false, title: `Current Quick Match score ${value}/100`};
}

// Mirrors the backend catalogue order: current scored results first, raw score descending, then
// evidence coverage descending. Stale and unscored results never pretend to be current.
export function sortableQuickScore(view?: QuickMatchView | null): {scored: boolean; raw: number; coverage: number} {
  if (!view || quickState(view) !== 'scored') {
    return {scored: false, raw: 0, coverage: view?.summary.coverage ?? 0};
  }
  const raw = view.summary.rawScore ?? (view.summary.score === null ? 0 : view.summary.score / 100);
  return {scored: true, raw, coverage: view.summary.coverage};
}

export function percent(value?: number | null): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '—';
  return `${Math.round(Math.max(0, Math.min(1, value)) * 100)}%`;
}

export function coveragePercent(view?: QuickMatchView | null): number | null {
  if (!view) return null;
  return Math.round(Math.max(0, Math.min(1, view.result.coverage.weighted)) * 100);
}

export function applicabilityLabel(value?: MatchApplicability): string {
  switch (value) {
    case 'applicable': return 'Compared';
    case 'not_applicable': return 'Not applicable';
    default: return 'Unknown';
  }
}

export function reliabilityLabel(value?: MatchReliability): string {
  switch (value) {
    case 'usable': return 'usable';
    case 'needs_corroboration': return 'needs corroboration';
    case 'unusable': return 'unusable';
    default: return 'not recorded';
  }
}

export function sectionLabel(value?: MatchSection): string {
  switch (value) {
    case 'required': return 'Required';
    case 'preferred': return 'Preferred';
    case 'company_context': return 'Company context';
    case 'unknown': return 'Section not identified';
    default: return 'Section not recorded';
  }
}

export function criterionLabel(criterion: string): string {
  switch (criterion) {
    case 'role.title': return 'Role title';
    case 'professional.requirement': return 'Professional requirement';
    case 'compensation.cash': return 'Compensation';
    case 'geography.location': return 'Hiring location';
    case 'geography.authorization': return 'Work authorization';
    case 'conditions.work_mode': return 'Work mode';
    case 'conditions.engagement': return 'Engagement';
    case 'conditions.hours': return 'Hours';
    case 'conditions.timezone': return 'Timezone';
    default: return humanizeCode(criterion);
  }
}

export function factLabel(fact: string): string {
  switch (fact) {
    case 'title': return 'Role title';
    case 'cash': return 'Compensation';
    case 'requirements': return 'Professional requirements';
    case 'hiringLocations': return 'Hiring location';
    case 'location': return 'Hiring location';
    case 'workAuthorization': return 'Work authorization';
    case 'workModes': return 'Work mode';
    case 'engagement': return 'Engagement';
    case 'hours': return 'Hours';
    case 'timezone': return 'Timezone';
    default: return fact;
  }
}

export function unknownTitle(unknown: MatchUnknown): string {
  if (unknown.kind.startsWith('skill_')) return `Requirement: ${unknown.fact}`;
  return factLabel(unknown.fact);
}

const REASONS: Record<string, string> = {
  no_comparable_evidence: 'No comparable profile and vacancy evidence yet.',
  source_ambiguity: 'Source evidence is incomplete or ambiguous.',
  role_title_aligned: 'A target role overlaps the vacancy title.',
  role_title_no_overlap: 'The vacancy title does not overlap any target role.',
  role_title_missing: 'The vacancy title is missing.',
  role_title_unusable: 'The vacancy title is not reliable enough to compare.',
  skill_evidenced: 'The profile evidences this requirement.',
  skill_not_evidenced: 'The profile does not evidence this requirement.',
  skill_requirement_compound: 'The requirement bundles several skills and is not split into invented parts.',
  skill_years_unsupported: 'The requirement states years; Quick Match does not compare inferred years.',
  skill_level_unsupported: 'The requirement states a level; Quick Match does not compare inferred levels.',
  skill_requirement_unusable: 'The requirement text is not reliable enough to compare.',
  skill_requirement_section_unknown: 'The listing section of this requirement is unknown.',
  skill_requirements_missing: 'No explicit professional requirements were extracted.',
  compensation_aligned: 'The declared vacancy ceiling meets the profile minimum.',
  compensation_below_minimum: 'The declared vacancy ceiling is below the profile minimum.',
  compensation_job_missing: 'The vacancy states no declared compensation ceiling.',
  compensation_incomparable: 'Compensation uses different currency, period or tax basis.',
  compensation_fact_unusable: 'The vacancy compensation fact is not reliable enough.',
  location_compatible: 'A stated hiring location is compatible with the preferred locations.',
  location_disjoint: 'The stated hiring locations are disjoint from the preferred locations.',
  location_mapping_unsupported: 'A location statement could not be mapped to a country.',
  location_missing: 'The vacancy states no usable hiring location.',
  location_unusable: 'The vacancy location statement is not reliable enough.',
  authorization_compatible: 'The stated work authorization is compatible with the profile.',
  authorization_disjoint: 'The stated work authorization is incompatible with the profile.',
  authorization_country_missing: 'The authorization statement does not identify a country.',
  authorization_profile_missing: 'The profile has no authorization answer for this country.',
  authorization_profile_status_missing: 'The profile authorization entry has no status.',
  authorization_profile_ambiguous: 'The profile has conflicting authorization answers.',
  authorization_job_missing: 'The vacancy states no work authorization requirement.',
  authorization_fact_unusable: 'The authorization statement is not reliable enough.',
  work_mode_compatible: 'The vacancy work mode is compatible with the profile.',
  work_mode_disjoint: 'The vacancy work mode is incompatible with the profile.',
  work_mode_missing: 'The vacancy states no usable work mode.',
  work_mode_unusable: 'The vacancy work mode is not reliable enough.',
  engagement_compatible: 'The vacancy engagement type is compatible with the profile.',
  engagement_disjoint: 'The vacancy engagement type is incompatible with the profile.',
  engagement_missing: 'The vacancy states no usable engagement type.',
  engagement_unusable: 'The vacancy engagement statement is not reliable enough.',
  hours_compatible: 'The vacancy hours are compatible with the profile.',
  hours_disjoint: 'The vacancy hours are incompatible with the profile.',
  hours_missing: 'The vacancy states no usable hours.',
  hours_partial_unknown: 'The hours comparison is only partly known.',
  hours_unusable: 'The vacancy hours statement is not reliable enough.',
  timezone_unsupported: 'The timezone comparison is not supported by the retained facts.',
  timezone_missing: 'The vacancy states no usable timezone.',
};

export function reasonText(code: string): string {
  return REASONS[code] ?? `${humanizeCode(code)}.`;
}

const STALE: Record<string, string> = {
  schema_version_changed: 'the result schema changed',
  extractor_version_changed: 'the extractor changed',
  rules_changed: 'the Quick Match rules or weights changed',
  source_policy_changed: 'the source-field policy changed',
  source_policy_version_changed: 'the source-field policy changed',
  profile_missing: 'the saved profile is unavailable',
  profile_changed: 'the saved profile changed',
  facts_missing: 'the stored vacancy facts are missing',
  facts_hash_changed: 'the extracted vacancy facts changed',
  facts_source_changed: 'the saved vacancy changed',
  saved_source_changed: 'the saved vacancy changed',
  facts_schema_changed: 'the stored facts use an older schema',
  facts_extractor_changed: 'the stored facts use an older extractor',
  facts_source_policy_changed: 'the stored facts used an older source policy',
  source_input_missing: 'the retained vacancy input is missing',
};

export function staleReasonText(view: QuickMatchView): string {
  const codes = view.staleReasons ?? [];
  if (!codes.length) return 'the profile or rules changed';
  return [...new Set(codes.map(code => STALE[code] ?? humanizeCode(code)))].join('; ');
}

function humanizeCode(code: string): string {
  const text = code.replace(/[._]+/g, ' ').trim();
  return text ? text[0].toUpperCase() + text.slice(1) : 'Unknown reason';
}

// Profile section ids match ProfileView's in-page anchors. Missing or future values simply do not
// offer an edit button rather than guessing an editor.
const PROFILE_SECTIONS: Record<string, {id: string; label: string}> = {
  targetRoles: {id: 'profile-goals', label: 'Target roles'},
  skills: {id: 'profile-skills', label: 'Skills'},
  'compensation.minimum': {id: 'profile-compensation', label: 'Compensation'},
  workModes: {id: 'profile-work', label: 'Work modes'},
  engagementTypes: {id: 'profile-work', label: 'Engagement'},
  hours: {id: 'profile-work', label: 'Hours'},
  timezones: {id: 'profile-geography', label: 'Timezones'},
  'geography.preferredLocations': {id: 'profile-geography', label: 'Geography'},
  'geography.workAuthorizations': {id: 'profile-geography', label: 'Work authorization'},
};

export function profileAnchor(profileEdit?: string): string | undefined {
  return profileEdit ? PROFILE_SECTIONS[profileEdit]?.id : undefined;
}

export function profileEditLabel(profileEdit?: string): string | undefined {
  return profileEdit ? PROFILE_SECTIONS[profileEdit]?.label : undefined;
}

export function comparisonForUnknown(result: MatchResult, unknown: MatchUnknown): MatchComparison | undefined {
  if (!unknown.kind.startsWith('skill_')) return undefined;
  return (result.comparisons ?? []).find(c =>
    c.group === unknown.group &&
    c.criterion === 'professional.requirement' &&
    c.reason === unknown.kind &&
    (c.jobFact === unknown.fact || (c.jobValue ?? '').includes(unknown.fact)));
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
