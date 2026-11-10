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
  legalName?: string;
  headline: string;
}

// Optional application-form answers. Deliberately separate from the fit/preparation context;
// never a document-extraction field and never sent automatically.
export interface Demographics {
  gender?: string;
  genderDescription?: string;
  race: string[];
  raceDescription?: string;
  ethnicity?: string;
}

export interface Skill {
  id?: string;
  key: string;
  name: string;
  level: string;
  context?: string;
  provenance?: Provenance | null;
}

export interface RichText {
  text: string;
  document?: unknown;
}

export interface Experience {
  id?: string;
  company: string;
  title: string;
  start?: string;
  end?: string;
  current?: boolean;
  summary?: string;
  summaryDocument?: unknown;
  skills: string[];
  // Client-only draft identity for a row that has not been saved yet. It keeps a stateful rich
  // editor attached to its own row when an earlier unsaved row is removed. cleanExperience
  // strips it before the section save, and the server never sees or stores it.
  clientKey?: string;
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
  crypto?: string;
}

export interface Engagement {
  types: string[];
}

export interface Hours {
  mode: string;
  weekly: number | null;
  weeklyMin?: number | null;
  weeklyMax?: number | null;
  weeklyRangeSet?: boolean;
}

export interface WorkModes {
  modes: string[];
  limits: string[];
}

export interface Location {
  country: string;
  city?: string;
}

export interface WorkAuthorization {
  country: string;
  status?: string;
}

export interface Geography {
  current: string;
  desired: string[];
  eligibility: string[];
  relocation: string[];
  willingToRelocate: boolean | null;
  sponsorship: string;
  timezones: string[];
  currentLocation?: Location | null;
  preferredLocations?: Location[];
  // Broad regions the owner would work in (multi-select, fixed values; see REGION_ORDER).
  preferredRegions?: string[];
  workAuthorizations?: WorkAuthorization[];
  timezoneIds?: string[];
  desiredOverlapHours?: number | null;
  currentLocationSet?: boolean;
  preferredLocationsSet?: boolean;
  workAuthorizationsSet?: boolean;
  timezonesSet?: boolean;
}

export interface Language {
  name: string;
  level: string;
}

export interface Availability {
  noticePeriod: string;
  startDate: string;
  timezone: string;
  noticeValue?: number | null;
  noticeUnit?: string;
  noticeSet?: boolean;
}

export interface OtherRequirements extends RichText {
  selected?: boolean;
}

export interface ProfileData {
  identity: Identity;
  demographics?: Demographics;
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
  otherRequirements?: OtherRequirements;
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

export interface FieldDescriptor {
  key: string;
  kind: string;
  uses: string[];
  options?: string;
  format?: string;
  legacy?: boolean;
  note?: string;
}

export interface OptionSet {
  name: string;
  values: string[];
  default?: string;
  notes?: Record<string, string>;
}

export interface Capabilities {
  documentTypes: string[];
  maxDocumentBytes: number;
  maxUploadBodyBytes: number;
  avatarTypes: string[];
  maxAvatarBytes: number;
  skillLevels: string[];
  skillLevelOptions?: string[];
  factFieldKeys: string[];
  fieldDescriptors?: FieldDescriptor[];
  optionSets?: OptionSet[];
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
export const ENGAGEMENT_TYPES = ['employment', 'contract'] as const;
export const HOURS_MODES = ['full_time', 'part_time'] as const;
export const WORK_MODES = ['remote', 'hybrid', 'office'] as const;
export const GROSS_NET = ['gross', 'net'] as const;
export const PERIODS = ['year', 'month', 'day', 'hour'] as const;
export const SPONSORSHIP = ['required', 'not_required'] as const;
export const DOCUMENT_ACTIVE_STATES = ['extracting', 'analyzing'] as const;

export const ENGAGEMENT_LABELS: Record<string, string> = {
  employment: 'Employment',
  contract: 'Contract',
  // Legacy stored values remain selectable and are never conflated with the current labels.
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
export const GROSS_NET_LABELS: Record<string, string> = {gross: 'Before taxes', net: 'After taxes'};
export const PERIOD_LABELS: Record<string, string> = {
  year: 'Yearly',
  month: 'Monthly',
  day: 'Per day',
  hour: 'Per hour',
};
export const SPONSORSHIP_LABELS: Record<string, string> = {
  required: 'Sponsorship required',
  not_required: 'Sponsorship not required',
};

export type SectionKey =
  | 'identity'
  | 'demographics'
  | 'skills'
  | 'experience'
  | 'goals'
  | 'compensation'
  | 'work'
  | 'geography'
  | 'languages'
  | 'availability'
  | 'dealbreakers'
  | 'other-requirements';

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
  demographics: Demographics;
  skills: Skill[];
  experience: Experience[];
  goals: Goals;
  compensation: Compensation;
  work: WorkSection;
  geography: Geography;
  languages: Language[];
  availability: Availability;
  dealbreakers: string[];
  'other-requirements': OtherRequirements;
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

export function emptyDemographics(): Demographics {
  return {gender: '', race: [], ethnicity: ''};
}

export function emptyProfileData(): ProfileData {
  return {
    identity: {displayName: '', legalName: '', headline: ''},
    demographics: emptyDemographics(),
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
    otherRequirements: {text: ''},
  };
}

export function cloneData<T>(value: T): T {
  return structuredClone(value);
}

export function emptySkill(): Skill {
  return {key: '', name: '', level: '', context: ''};
}

let clientExperienceSequence = 0;

function nextClientExperienceKey(): string {
  clientExperienceSequence += 1;
  return `draft-experience-${clientExperienceSequence}`;
}

export function emptyExperience(): Experience {
  return {company: '', title: '', start: '', end: '', current: false, summary: '', skills: [], clientKey: nextClientExperienceKey()};
}

// Editor drafts may start with one blank row when the stored section has none. The row exists
// only in the draft: it is never persisted, and cleaning before Save removes it when untouched.
export function draftValues<K extends SectionKey>(data: ProfileData, key: K): SectionValues[K] {
  const value = sectionValues(data, key);
  if (key === 'skills' && (value as Skill[]).length === 0) return [emptySkill()] as SectionValues[K];
  if (key === 'experience' && (value as Experience[]).length === 0) return [emptyExperience()] as SectionValues[K];
  if (key === 'other-requirements') {
    // The merged editor replaces the old Dealbreakers and Work limits blocks. Seed it with their
    // retained text so those conditions are visible and editable; the server keeps the original
    // arrays until this editor is deliberately saved, and then treats this block as the source.
    const other = cloneData(value as OtherRequirements);
    if (!other.selected && !(other.text ?? '').trim() && !other.document) {
      const seeded = [...data.dealbreakers, ...data.workModes.limits]
        .map((item) => item.trim())
        .filter((item, index, all) => item !== '' && all.indexOf(item) === index);
      if (seeded.length) other.text = seeded.join('\n');
    }
    return other as SectionValues[K];
  }
  return value;
}

export function sectionValues<K extends SectionKey>(data: ProfileData, key: K): SectionValues[K] {
  switch (key) {
    case 'identity': return cloneData(data.identity) as SectionValues[K];
    case 'demographics': return cloneData(data.demographics ?? emptyDemographics()) as SectionValues[K];
    case 'skills': return cloneData(data.skills) as SectionValues[K];
    case 'experience': return cloneData(data.experience) as SectionValues[K];
    case 'goals': return cloneData({targetRoles: data.targetRoles, targetIndustries: data.targetIndustries}) as SectionValues[K];
    case 'compensation': return cloneData(data.compensation) as SectionValues[K];
    case 'work': return cloneData({engagement: data.engagement, hours: data.hours, workModes: data.workModes}) as SectionValues[K];
    case 'geography': return cloneData(data.geography) as SectionValues[K];
    case 'languages': return cloneData(data.languages) as SectionValues[K];
    case 'availability': return cloneData(data.availability) as SectionValues[K];
    case 'dealbreakers': return cloneData(data.dealbreakers) as SectionValues[K];
    case 'other-requirements': return cloneData(data.otherRequirements ?? {text: ''}) as SectionValues[K];
  }
  throw new Error(`unknown profile section ${key}`);
}

// optionValues reads one declarative server vocabulary when capabilities have loaded and falls
// back to the mirrored contract while they are unavailable. The server remains the validator.
export function optionValues(capabilities: Capabilities | null, name: string, fallback: readonly string[]): string[] {
  const set = capabilities?.optionSets?.find((candidate) => candidate.name === name);
  return set?.values?.length ? [...set.values] : [...fallback];
}

export function optionNotes(capabilities: Capabilities | null, name: string): Record<string, string> {
  return capabilities?.optionSets?.find((candidate) => candidate.name === name)?.notes ?? {};
}

export function skillLevelOptions(capabilities: Capabilities | null): string[] {
  if (capabilities?.skillLevelOptions?.length) return [...capabilities.skillLevelOptions];
  return ['Beginner', 'Proficient', 'Expert'];
}

// levelsWithRetained keeps an older stored value selectable without mapping it to a new level.
export function levelsWithRetained(options: string[], stored: string): string[] {
  const value = (stored ?? '').trim();
  if (!value || options.includes(value)) return options;
  return [...options, value];
}

// cleanSection prepares one section value for the server-side atomic merge. The server stays
// authoritative and applies the final normalization and validation. Every cleaner spreads the
// incoming value, so fields the current editor does not show (server row ids, retained legacy
// values, rich documents, marker booleans) are never dropped. Marker-bearing keys are included
// only when the structured value actually changed: an unchanged empty value must not turn a
// no-op save into a marker migration.
export function cleanSection<K extends SectionKey>(
  key: K,
  value: SectionValues[K],
  current?: SectionValues[K],
): SectionValues[K] {
  switch (key) {
    case 'identity': {
      const identity = value as Identity;
      return {
        ...identity,
        displayName: identity.displayName.trim(),
        legalName: (identity.legalName ?? '').trim(),
        headline: (identity.headline ?? '').trim(),
      } as SectionValues[K];
    }
    case 'demographics': return cleanDemographics(value as Demographics) as SectionValues[K];
    case 'skills': return cleanSkills(value as Skill[]) as SectionValues[K];
    case 'experience': return cleanExperience(value as Experience[]) as SectionValues[K];
    case 'goals': {
      const goals = value as Goals;
      return {...goals, targetRoles: cleanList(goals.targetRoles), targetIndustries: cleanList(goals.targetIndustries)} as SectionValues[K];
    }
    case 'compensation': return cleanCompensation(value as Compensation, current as Compensation | undefined) as SectionValues[K];
    case 'work': return cleanWork(value as WorkSection, current as WorkSection | undefined) as SectionValues[K];
    case 'geography': return cleanGeography(value as Geography, current as Geography | undefined) as SectionValues[K];
    case 'languages': return cleanLanguages(value as Language[]) as SectionValues[K];
    case 'availability': return cleanAvailability(value as Availability, current as Availability | undefined) as SectionValues[K];
    case 'dealbreakers': return cleanList(value as string[]) as SectionValues[K];
    case 'other-requirements': return cleanOtherRequirements(value as OtherRequirements) as SectionValues[K];
  }
  throw new Error(`unknown profile section ${key}`);
}

function sameNumber(left: number | null | undefined, right: number | null | undefined): boolean {
  return (left ?? null) === (right ?? null);
}

function cleanDemographics(value: Demographics): Demographics {
  const race = [...new Set((value.race ?? []).filter((item) => item.trim() !== ''))];
  return {
    ...value,
    gender: value.gender ?? '',
    genderDescription: (value.genderDescription ?? '').trim(),
    // Prefer not to say is exclusive; the server refuses a mixed answer.
    race: race.includes('prefer_not_to_say') ? ['prefer_not_to_say'] : race,
    raceDescription: (value.raceDescription ?? '').trim(),
    ethnicity: value.ethnicity ?? '',
  };
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
    .map((row) => {
      const cleaned: Experience = {
        ...row,
        company: row.company.trim(),
        title: row.title.trim(),
        start: row.start?.trim() || undefined,
        end: row.end?.trim() || undefined,
        summary: row.summary?.trim() || undefined,
        skills: cleanList(row.skills),
      };
      // Draft identity is client-only; never send it as data or let it reach the server.
      delete cleaned.clientKey;
      return cleaned;
    })
    // Keep any partially filled row. An empty additive row may be omitted, but
    // a user's start date, summary or skills must not vanish on Save.
    .filter((row) => row.company !== '' || row.title !== '' || !!row.start || !!row.end || !!row.summary || row.skills.length > 0);
}

function sameTriState(left: boolean | null | undefined, right: boolean | null | undefined): boolean {
  return (left ?? null) === (right ?? null);
}

function cleanCompensation(value: Compensation, current?: Compensation): Compensation {
  const out: Compensation = {
    ...value,
    currency: value.currency.trim(),
    dayRates: {
      ...value.dayRates,
      currency: value.dayRates.currency.trim(),
    },
  };
  // Equity/tokens is an additive preference, not guaranteed cash. A checkbox starts off; when
  // the stored value is unknown, keep it unknown rather than rewriting it to an explicit false
  // merely because another compensation field was saved.
  if (current) {
    const unchanged = sameTriState(value.equityOnTop, current.equityOnTop)
      || ((current.equityOnTop === null || current.equityOnTop === undefined) && value.equityOnTop === false);
    out.equityOnTop = unchanged ? (current.equityOnTop ?? null) : value.equityOnTop;
  }
  return out;
}

function cleanWork(value: WorkSection, current?: WorkSection): WorkSection {
  const hours: Hours = {
    ...value.hours,
    mode: value.hours.mode,
    weekly: value.hours.weekly,
  };
  // Weekly is the desired part-time field exposed by the current editor; it is always echoed.
  // Send the optional range keys and their marker only when the range actually changed, so
  // saving another work field cannot silently set weeklyRangeSet.
  if (current) {
    if (sameNumber(value.hours.weeklyMin, current.hours.weeklyMin)) delete hours.weeklyMin;
    else hours.weeklyMin = value.hours.weeklyMin ?? null;
    if (sameNumber(value.hours.weeklyMax, current.hours.weeklyMax)) delete hours.weeklyMax;
    else hours.weeklyMax = value.hours.weeklyMax ?? null;
    if (Boolean(value.hours.weeklyRangeSet) === Boolean(current.hours.weeklyRangeSet)) delete hours.weeklyRangeSet;
  }
  return {
    ...value,
    engagement: {...value.engagement, types: cleanList(value.engagement.types)},
    hours,
    workModes: {
      ...value.workModes,
      modes: cleanList(value.workModes.modes),
      // Limits are retained legacy storage; the editor no longer owns this block. Keep the
      // exact stored rows so a save from another work field cannot rewrite them.
      limits: current ? current.workModes.limits : value.workModes.limits,
    },
  };
}

function cleanLocation(value: Location | null | undefined): Location | null {
  const country = (value?.country ?? '').trim().toUpperCase();
  const city = (value?.city ?? '').trim();
  // An explicitly empty current location is a clear. A partially filled row (city without a
  // country) is preserved so the server validator can name the missing country instead of the
  // cleaner silently discarding the user's input.
  if (!country && !city) return null;
  return city ? {country, city} : {country};
}

function cleanLocations(rows: Location[] | undefined): Location[] {
  return (rows ?? [])
    .map(cleanLocation)
    .filter((row): row is Location => row !== null);
}

function cleanAuthorizations(rows: WorkAuthorization[] | undefined): WorkAuthorization[] {
  return (rows ?? [])
    .map((row) => ({country: (row.country ?? '').trim().toUpperCase(), status: row.status ?? ''}))
    // An untouched blank row may be omitted. A status without a country is a partial answer:
    // keep it so the server validation error remains visible and the draft is not silently lost.
    .filter((row) => row.country !== '' || row.status !== '');
}

function cleanTimezoneIDs(rows: string[] | undefined): string[] {
  return [...new Set((rows ?? []).map((row) => row.trim()).filter((row) => row !== ''))];
}

function cleanGeography(value: Geography, current?: Geography): Geography {
  // Legacy strings stay in the draft and are spread unchanged. They are retained unstructured
  // values until the owner supplies structured fields; they are never parsed into countries.
  const out: Partial<Geography> = {...value};
  // Relocation notes are retained legacy data. Keep the stored rows exactly when unchanged so
  // saving another geography field cannot rewrite their whitespace.
  if (current) {
    const nextRelocation = cleanList(value.relocation ?? []);
    out.relocation = JSON.stringify(nextRelocation) === JSON.stringify(cleanList(current.relocation ?? []))
      ? current.relocation
      : nextRelocation;
  }
  if (!current) return out as Geography;

  // Structured fields and their effective-source markers are emitted only on a real change. An
  // unchanged empty value must not turn an ordinary no-op save into a marker migration; an
  // explicit cleared choice must reach the server so it cannot revive retained legacy text.
  const nextCurrent = cleanLocation(value.currentLocation);
  if (JSON.stringify(nextCurrent) === JSON.stringify(cleanLocation(current.currentLocation))) {
    delete out.currentLocation;
  } else {
    out.currentLocation = nextCurrent;
  }

  const nextPreferred = cleanLocations(value.preferredLocations);
  if (JSON.stringify(nextPreferred) === JSON.stringify(cleanLocations(current.preferredLocations))) {
    delete out.preferredLocations;
  } else {
    out.preferredLocations = nextPreferred;
  }

  const nextRegions = [...new Set((value.preferredRegions ?? []).filter(Boolean))];
  if (JSON.stringify(nextRegions) === JSON.stringify([...new Set((current.preferredRegions ?? []).filter(Boolean))])) {
    delete out.preferredRegions;
  } else {
    out.preferredRegions = nextRegions;
  }

  const nextAuthorizations = cleanAuthorizations(value.workAuthorizations);
  if (JSON.stringify(nextAuthorizations) === JSON.stringify(cleanAuthorizations(current.workAuthorizations))) {
    delete out.workAuthorizations;
  } else {
    out.workAuthorizations = nextAuthorizations;
  }

  const nextTimezoneIDs = cleanTimezoneIDs(value.timezoneIds);
  if (JSON.stringify(nextTimezoneIDs) === JSON.stringify(cleanTimezoneIDs(current.timezoneIds))) {
    delete out.timezoneIds;
  } else {
    out.timezoneIds = nextTimezoneIDs;
  }

  if (sameNumber(value.desiredOverlapHours, current.desiredOverlapHours)) delete out.desiredOverlapHours;
  else out.desiredOverlapHours = value.desiredOverlapHours ?? null;

  for (const marker of ['currentLocationSet', 'preferredLocationsSet', 'workAuthorizationsSet', 'timezonesSet'] as const) {
    if (Boolean(value[marker]) === Boolean(current[marker])) delete out[marker];
    else out[marker] = Boolean(value[marker]);
  }
  return out as Geography;
}

function cleanLanguages(rows: Language[]): Language[] {
  return rows
    .map((row) => ({...row, name: row.name.trim(), level: row.level.trim()}))
    // A level-only language is kept so validation can report the missing name.
    .filter((row) => row.name !== '' || row.level !== '');
}

function cleanAvailability(value: Availability, current?: Availability): Availability {
  const out: Availability = {
    ...value,
    noticePeriod: value.noticePeriod.trim(),
    startDate: value.startDate.trim(),
    timezone: value.timezone.trim(),
  };
  delete out.noticeSet;
  if (current) {
    const changed = !sameNumber(value.noticeValue, current.noticeValue)
      || (value.noticeUnit ?? '') !== (current.noticeUnit ?? '');
    if (changed) {
      out.noticeValue = value.noticeValue ?? null;
      out.noticeUnit = value.noticeUnit ?? '';
    } else {
      // Unchanged structured notice: omit the keys so the server does not set NoticeSet.
      delete out.noticeValue;
      delete out.noticeUnit;
    }
  }
  return out;
}

function cleanOtherRequirements(value: OtherRequirements): OtherRequirements {
  const out: OtherRequirements = {text: (value.text ?? '').trim()};
  if (value.document !== undefined && value.document !== null) out.document = value.document;
  // `selected` is server-owned: the section merge sets it when this editor sends a real value.
  return out;
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
