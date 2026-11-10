// Typed mirror of backend/internal/profile/contract.go and the profile API. This file is the
// single place for domain enum constants the capabilities endpoint does not yet expose. Empty
// string, null and empty arrays mean unknown; no default or example candidate value is defined.

export interface Provenance {
  kind?: string;
  reference?: string;
  quote?: string;
}

export interface Identity {
  displayName: string;
  headline: string;
}

export interface Skill {
  key: string;
  name: string;
  level: string;
  context?: string;
  provenance?: Provenance | null;
}

export interface Experience {
  company: string;
  title: string;
  start?: string;
  end?: string;
  summary?: string;
  skills: string[];
}

export interface DayRates {
  min: number | null;
  target: number | null;
  currency: string;
  period: string;
}

export interface Compensation {
  cashMin: number | null;
  cashTarget: number | null;
  currency: string;
  grossNet: string;
  period: string;
  dayRates: DayRates;
  equityOnTop: boolean | null;
}

export interface Engagement {
  types: string[];
}

export interface Hours {
  mode: string;
  weekly: number | null;
}

export interface WorkModes {
  modes: string[];
  limits: string[];
}

export interface Geography {
  current: string;
  desired: string[];
  eligibility: string[];
  relocation: string[];
  willingToRelocate: boolean | null;
  sponsorship: string;
  timezones: string[];
}

export interface Language {
  name: string;
  level: string;
}

export interface Availability {
  noticePeriod: string;
  startDate: string;
  timezone: string;
}

export interface ProfileData {
  identity: Identity;
  skills: Skill[];
  experience: Experience[];
  targetRoles: string[];
  targetIndustries: string[];
  compensation: Compensation;
  engagement: Engagement;
  hours: Hours;
  workModes: WorkModes;
  geography: Geography;
  languages: Language[];
  availability: Availability;
  dealbreakers: string[];
  legacyPreferences?: Record<string, unknown>;
  legacyContextVersion?: string;
}

export interface Profile {
  profileId: string;
  revision: number;
  createdAt: string;
  updatedAt: string;
  data: ProfileData;
  contextVersion?: string;
  contextProfileRevision?: number;
  contextCurrent: boolean;
  contextError?: string;
  fastRankingRevision?: number;
  fastRankingCurrent: boolean;
  fastRankingError?: string;
  derivedCurrent: boolean;
}

export interface Capabilities {
  documentTypes: string[];
  maxDocumentBytes: number;
  maxUploadBodyBytes: number;
  skillLevels: string[];
  factFieldKeys: string[];
  documentStates: string[];
  factStates: string[];
}

export interface ProfileDocument {
  documentId: string;
  profileId: string;
  kind: string;
  originalName: string;
  sourceUrl?: string;
  contentHash?: string;
  processingState: string;
  hasText: boolean;
  textChars: number;
  hasOriginal: boolean;
  analysisMetadata?: Record<string, unknown>;
  error?: string;
  createdAt: string;
  updatedAt: string;
  factCounts: Record<string, number>;
}

export interface Fact {
  factId: string;
  profileId: string;
  documentId?: string;
  documentName?: string;
  fieldKey: string;
  value: unknown;
  quote: string;
  sourceReference?: string;
  origin: string;
  state: string;
  supersedesId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface FactConflict {
  fieldKey: string;
  current: unknown;
  proposed: unknown;
  factId: string;
  currentRevision: number;
  message: string;
}

export interface FactDecision {
  fact: Fact;
  profile?: Profile;
  profileChanged: boolean;
  conflict?: FactConflict;
}

export interface ConflictPayload {
  error?: string;
  expectedRevision?: number;
  currentRevision?: number;
  current?: Profile;
  conflict?: FactConflict;
  code?: string;
  documentState?: string;
}

// Domain enums that are not (yet) supplied by GET /api/profile/capabilities. Skill levels come
// from capabilities; everything here remains the contract, not mock candidate data.
export const ENGAGEMENT_TYPES = ['employee', 'contractor'] as const;
export const HOURS_MODES = ['full_time', 'part_time'] as const;
export const WORK_MODES = ['remote', 'hybrid', 'office'] as const;
export const GROSS_NET = ['gross', 'net'] as const;
export const PERIODS = ['year', 'month', 'day', 'hour'] as const;
export const SPONSORSHIP = ['required', 'not_required'] as const;
export const DOCUMENT_ACTIVE_STATES = ['extracting', 'analyzing'] as const;

export const ENGAGEMENT_LABELS: Record<string, string> = {
  employee: 'Employee',
  contractor: 'Contractor',
};
export const HOURS_LABELS: Record<string, string> = {
  full_time: 'Full-time',
  part_time: 'Part-time',
};
export const WORK_MODE_LABELS: Record<string, string> = {
  remote: 'Remote',
  hybrid: 'Hybrid',
  office: 'Office',
};
export const GROSS_NET_LABELS: Record<string, string> = {gross: 'Gross', net: 'Net'};
export const PERIOD_LABELS: Record<string, string> = {
  year: 'Per year',
  month: 'Per month',
  day: 'Per day',
  hour: 'Per hour',
};
export const SPONSORSHIP_LABELS: Record<string, string> = {
  required: 'Sponsorship required',
  not_required: 'Sponsorship not required',
};

export type SectionKey =
  | 'identity'
  | 'skills'
  | 'experience'
  | 'goals'
  | 'compensation'
  | 'work'
  | 'geography'
  | 'languages'
  | 'availability'
  | 'dealbreakers';

export interface Goals {
  targetRoles: string[];
  targetIndustries: string[];
}

export interface WorkSection {
  engagement: Engagement;
  hours: Hours;
  workModes: WorkModes;
}

export interface SectionValues {
  identity: Identity;
  skills: Skill[];
  experience: Experience[];
  goals: Goals;
  compensation: Compensation;
  work: WorkSection;
  geography: Geography;
  languages: Language[];
  availability: Availability;
  dealbreakers: string[];
}

export interface SectionDraft<T> {
  value: T;
  dirty: boolean;
  saving: boolean;
  error: string;
}

export type Drafts = {[K in SectionKey]?: SectionDraft<SectionValues[K]>};

export function draftFor<K extends SectionKey>(drafts: Drafts, key: K): SectionDraft<SectionValues[K]> | undefined {
  return drafts[key] as SectionDraft<SectionValues[K]> | undefined;
}

export function emptyProfileData(): ProfileData {
  return {
    identity: {displayName: '', headline: ''},
    skills: [],
    experience: [],
    targetRoles: [],
    targetIndustries: [],
    compensation: {
      cashMin: null,
      cashTarget: null,
      currency: '',
      grossNet: '',
      period: '',
      dayRates: {min: null, target: null, currency: '', period: ''},
      equityOnTop: null,
    },
    engagement: {types: []},
    hours: {mode: '', weekly: null},
    workModes: {modes: [], limits: []},
    geography: {
      current: '',
      desired: [],
      eligibility: [],
      relocation: [],
      willingToRelocate: null,
      sponsorship: '',
      timezones: [],
    },
    languages: [],
    availability: {noticePeriod: '', startDate: '', timezone: ''},
    dealbreakers: [],
  };
}

export function cloneData<T>(value: T): T {
  return structuredClone(value);
}

export function emptySkill(): Skill {
  return {key: '', name: '', level: '', context: ''};
}

export function emptyExperience(): Experience {
  return {company: '', title: '', start: '', end: '', summary: '', skills: []};
}

// Editor drafts may start with one blank row when the stored section has none. The row exists
// only in the draft: it is never persisted, and cleaning before Save removes it when untouched.
export function draftValues<K extends SectionKey>(data: ProfileData, key: K): SectionValues[K] {
  const value = sectionValues(data, key);
  if (key === 'skills' && (value as Skill[]).length === 0) return [emptySkill()] as SectionValues[K];
  if (key === 'experience' && (value as Experience[]).length === 0) return [emptyExperience()] as SectionValues[K];
  return value;
}

export function sectionValues<K extends SectionKey>(data: ProfileData, key: K): SectionValues[K] {
  switch (key) {
    case 'identity': return cloneData(data.identity) as SectionValues[K];
    case 'skills': return cloneData(data.skills) as SectionValues[K];
    case 'experience': return cloneData(data.experience) as SectionValues[K];
    case 'goals': return cloneData({targetRoles: data.targetRoles, targetIndustries: data.targetIndustries}) as SectionValues[K];
    case 'compensation': return cloneData(data.compensation) as SectionValues[K];
    case 'work': return cloneData({engagement: data.engagement, hours: data.hours, workModes: data.workModes}) as SectionValues[K];
    case 'geography': return cloneData(data.geography) as SectionValues[K];
    case 'languages': return cloneData(data.languages) as SectionValues[K];
    case 'availability': return cloneData(data.availability) as SectionValues[K];
    case 'dealbreakers': return cloneData(data.dealbreakers) as SectionValues[K];
  }
  throw new Error(`unknown profile section ${key}`);
}

// cleanSection prepares one section value for the server-side atomic merge. The server stays
// authoritative and applies the final normalization and validation; this keeps the editor from
// sending clearly empty rows or untrimmed user text.
export function cleanSection<K extends SectionKey>(key: K, value: SectionValues[K]): SectionValues[K] {
  switch (key) {
    case 'identity': {
      const identity = value as Identity;
      return {displayName: identity.displayName.trim(), headline: identity.headline.trim()} as SectionValues[K];
    }
    case 'skills': return cleanSkills(value as Skill[]) as SectionValues[K];
    case 'experience': return cleanExperience(value as Experience[]) as SectionValues[K];
    case 'goals': {
      const goals = value as Goals;
      return {targetRoles: cleanList(goals.targetRoles), targetIndustries: cleanList(goals.targetIndustries)} as SectionValues[K];
    }
    case 'compensation': return cleanCompensation(value as Compensation) as SectionValues[K];
    case 'work': {
      const work = value as WorkSection;
      return {
        engagement: {types: cleanList(work.engagement.types)},
        hours: {mode: work.hours.mode, weekly: work.hours.weekly},
        workModes: {modes: cleanList(work.workModes.modes), limits: cleanList(work.workModes.limits)},
      } as SectionValues[K];
    }
    case 'geography': return cleanGeography(value as Geography) as SectionValues[K];
    case 'languages': return cleanLanguages(value as Language[]) as SectionValues[K];
    case 'availability': return cleanAvailability(value as Availability) as SectionValues[K];
    case 'dealbreakers': return cleanList(value as string[]) as SectionValues[K];
  }
  throw new Error(`unknown profile section ${key}`);
}

function cleanList(items: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const item of items) {
    const clean = item.trim();
    const identity = clean.toLocaleLowerCase();
    if (clean && !seen.has(identity)) {
      seen.add(identity);
      out.push(clean);
    }
  }
  return out;
}

function cleanSkills(skills: Skill[]): Skill[] {
  return skills
    .map((skill) => ({
      ...skill,
      key: (skill.key || skill.name).trim().toLocaleLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''),
      name: skill.name.trim(),
      level: skill.level,
      context: skill.context?.trim() || undefined,
    }))
    // Keep any non-empty row. A context-only or level-only entry must reach the
    // server validator and stay in the draft rather than disappear on Save.
    .filter((skill) => skill.key !== '' || skill.name !== '' || skill.level !== '' || !!skill.context);
}

function cleanExperience(rows: Experience[]): Experience[] {
  return rows
    .map((row) => ({
      ...row,
      company: row.company.trim(),
      title: row.title.trim(),
      start: row.start?.trim() || undefined,
      end: row.end?.trim() || undefined,
      summary: row.summary?.trim() || undefined,
      skills: cleanList(row.skills),
    }))
    // Keep any partially filled row. An empty additive row may be omitted, but
    // a user's start date, summary or skills must not vanish on Save.
    .filter((row) => row.company !== '' || row.title !== '' || !!row.start || !!row.end || !!row.summary || row.skills.length > 0);
}

function cleanCompensation(value: Compensation): Compensation {
  return {
    cashMin: value.cashMin,
    cashTarget: value.cashTarget,
    currency: value.currency.trim(),
    grossNet: value.grossNet,
    period: value.period,
    dayRates: {
      min: value.dayRates.min,
      target: value.dayRates.target,
      currency: value.dayRates.currency.trim(),
      period: value.dayRates.period,
    },
    equityOnTop: value.equityOnTop,
  };
}

function cleanGeography(value: Geography): Geography {
  return {
    current: value.current.trim(),
    desired: cleanList(value.desired),
    eligibility: cleanList(value.eligibility),
    relocation: cleanList(value.relocation),
    willingToRelocate: value.willingToRelocate,
    sponsorship: value.sponsorship,
    timezones: cleanList(value.timezones),
  };
}

function cleanLanguages(rows: Language[]): Language[] {
  return rows
    .map((row) => ({name: row.name.trim(), level: row.level.trim()}))
    // A level-only language is kept so validation can report the missing name.
    .filter((row) => row.name !== '' || row.level !== '');
}

function cleanAvailability(value: Availability): Availability {
  return {
    noticePeriod: value.noticePeriod.trim(),
    startDate: value.startDate.trim(),
    timezone: value.timezone.trim(),
  };
}

export function listText(items: string[]): string {
  return items.join('\n');
}

export function textList(text: string): string[] {
  return text.split(/\r?\n/);
}

export function numberOrNull(text: string): number | null {
  if (text.trim() === '') return null;
  const value = Number(text);
  return Number.isFinite(value) ? value : null;
}

export function numberText(value: number | null): string {
  return value === null || value === undefined ? '' : String(value);
}

export function triStateText(value: boolean | null): string {
  return value === null || value === undefined ? '' : value ? 'true' : 'false';
}

export function triStateValue(text: string): boolean | null {
  return text === 'true' ? true : text === 'false' ? false : null;
}

export function hasAny(list: string[]): boolean {
  return list.some((item) => item.trim() !== '');
}

export function provenanceText(provenance?: Provenance | null): string {
  if (!provenance) return '';
  const source = provenance.kind === 'document'
    ? 'Document'
    : provenance.kind === 'manual'
      ? 'Manual'
      : provenance.kind === 'legacy-config'
        ? 'Legacy configuration'
        : provenance.kind === 'legacy-context'
          ? 'Legacy context'
          : provenance.kind || 'Source';
  return provenance.reference ? `${source} · ${provenance.reference}` : source;
}

export function factValueText(value: unknown): string {
  if (value === null || value === undefined) return 'Not set';
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  if (Array.isArray(value)) return value.map(factValueText).join(', ');
  if (typeof value === 'object') {
    const object = value as Record<string, unknown>;
    if (typeof object.name === 'string') {
      const level = typeof object.level === 'string' && object.level ? ` · ${object.level}` : '';
      const context = typeof object.context === 'string' && object.context ? ` · ${object.context}` : '';
      return `${object.name}${level}${context}`;
    }
    if (typeof object.company === 'string') return `${object.company} · ${object.title || ''}`.trim();
    return JSON.stringify(value);
  }
  return String(value);
}

// currentValue reads the stored profile value for a fact field path so an unresolved conflict can
// show both the current and proposed values without a second server shape.
export function currentValue(data: ProfileData, fieldKey: string): unknown {
  const parts = fieldKey.split('.');
  let current: unknown = data;
  for (const part of parts) {
    if (current === null || current === undefined) return undefined;
    current = (current as Record<string, unknown>)[part];
  }
  return current;
}
