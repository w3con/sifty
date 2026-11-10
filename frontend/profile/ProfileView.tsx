import React, {useRef, useState} from 'react';
import {
  ENGAGEMENT_LABELS,
  ENGAGEMENT_TYPES,
  GROSS_NET,
  GROSS_NET_LABELS,
  HOURS_LABELS,
  HOURS_MODES,
  PERIOD_LABELS,
  SPONSORSHIP_LABELS,
  WORK_MODE_LABELS,
  WORK_MODES,
  currentValue,
  draftFor,
  emptyExperience,
  emptySkill,
  factValueText,
  levelsWithRetained,
  listText,
  numberOrNull,
  numberText,
  optionNotes,
  optionValues,
  provenanceText,
  skillLevelOptions,
  textList,
  triStateText,
  triStateValue,
  type Availability,
  type Capabilities,
  type Compensation,
  type Demographics,
  type Fact,
  type Geography,
  type Goals,
  type OtherRequirements,
  type Profile,
  type WorkSection,
} from './contract';
import {
  AUTHORIZATION_LABELS,
  CRYPTO_LABELS,
  CRYPTO_ORDER,
  CURRENCY_LABELS,
  CURRENCY_ORDER,
  ETHNICITY_LABELS,
  ETHNICITY_ORDER,
  GENDER_LABELS,
  GENDER_ORDER,
  LANGUAGE_LEVEL_DESCRIPTIONS,
  LANGUAGE_LEVEL_ORDER,
  LANGUAGE_NAMES,
  NOTICE_UNIT_LABELS,
  NOTICE_UNIT_ORDER,
  RACE_LABELS,
  RACE_ORDER,
  SKILL_LEVEL_LABELS,
  countryName,
  labelFor,
  locationLabel,
  retainedLabel,
  timezoneLabel,
} from './referenceData';
import {
  AuthorizationEditor,
  LocationFields,
  LocationTags,
  LocationsEditor,
  RetainedCheckGroup,
  RetainedSelect,
  TimezonePicker,
} from './ReferenceFields';
import {DateInput, MonthYearInput} from './DateInputs';
import {documentSummary} from './documentPresentation';
import {RichTextEditor} from './RichTextEditor';
import {RichTextView} from './RichTextView';
import type {ProfileDocuments, ProfileEditor, ProfileFacts} from './useProfile';

export type ProfileViewProps = {
  editor: ProfileEditor;
  documents: ProfileDocuments;
  facts: ProfileFacts;
  onClose: () => void;
  onOpenSettings: () => void;
};

function NotSet() {
  return <span className="profile-not-set">Not set</span>;
}

// A number without currency is not a usable amount; show the missing currency explicitly
// instead of implying one.
function Money({amount, currency}: {amount: number | null; currency: string}) {
  if (amount === null) return <NotSet/>;
  return currency ? <>{currency} {amount}</> : <>{amount} · <NotSet/></>;
}

function Tags({items, labels = {}}: {items: string[]; labels?: Record<string, string>}) {
  const visible = items.filter((item) => item.trim() !== '');
  if (!visible.length) return <NotSet/>;
  return <div className="profile-tags">{visible.map((item) => <span key={item}>{labels[item] ?? item}</span>)}</div>;
}

function Provenance({value}: {value?: {kind?: string; reference?: string; quote?: string} | null}) {
  if (!value) return null;
  return (
    <span className="profile-provenance">
      {provenanceText(value)}{value.quote ? <q>{value.quote}</q> : null}
    </span>
  );
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return 'not configured';
  if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
  if (bytes >= 1024) return `${Math.round(bytes / 1024)} KiB`;
  return `${bytes} bytes`;
}

function replaceAt<T>(items: T[], index: number, value: T): T[] {
  return items.map((item, current) => current === index ? value : item);
}

function skillLevelText(level: string, capabilities: Capabilities | null): string {
  if (!level) return 'Not set';
  if (SKILL_LEVEL_LABELS[level]) return SKILL_LEVEL_LABELS[level];
  return retainedLabel(level, SKILL_LEVEL_LABELS, optionNotes(capabilities, 'skill_levels')[level]);
}

function languageLevelText(level: string): string {
  if (!level) return 'Not set';
  const description = LANGUAGE_LEVEL_DESCRIPTIONS[level];
  return description ? `${level} — ${description}` : `${level} (retained value)`;
}

function toggleRace(values: string[], option: string, checked: boolean): string[] {
  if (!checked) return values.filter((value) => value !== option);
  if (option === 'prefer_not_to_say') return [option];
  return [...values.filter((value) => value !== 'prefer_not_to_say'), option];
}

function SectionShell({
  title, description, editing, saving, error, onEdit, onCancel, onSave, children,
}: {
  title: string; description?: string; editing: boolean; saving: boolean; error: string;
  onEdit: () => void; onCancel: () => void; onSave: () => void; children: React.ReactNode;
}) {
  return (
    <section className="profile-card">
      <header className="profile-card-head">
        <div><h2>{title}</h2>{description ? <p className="muted">{description}</p> : null}</div>
        <div className="profile-card-actions">
          {editing ? <>
            <button type="button" onClick={onCancel} disabled={saving}>Cancel</button>
            <button type="button" className="primary" onClick={onSave} disabled={saving}>{saving ? 'Saving…' : 'Save'}</button>
          </> : <button type="button" onClick={onEdit}>Edit</button>}
        </div>
      </header>
      {error ? <div className="banner error" role="alert">{error}</div> : null}
      <div className="profile-card-body">{children}</div>
    </section>
  );
}

function Field({label, children, hint}: {label: string; children: React.ReactNode; hint?: string}) {
  return <label className="profile-field"><span>{label}</span>{children}{hint ? <small>{hint}</small> : null}</label>;
}

function TriState({value, onChange, yes = 'Yes', no = 'No', emptyLabel = 'Not set'}: {value: boolean | null; onChange: (value: boolean | null) => void; yes?: string; no?: string; emptyLabel?: string}) {
  return <select value={triStateText(value)} onChange={(event) => onChange(triStateValue(event.target.value))}>
    <option value="">{emptyLabel}</option>
    <option value="true">{yes}</option>
    <option value="false">{no}</option>
  </select>;
}

function EnumSelect({value, options, labels, emptyLabel = 'Not set', onChange}: {value: string; options: readonly string[]; labels: Record<string, string>; emptyLabel?: string; onChange: (value: string) => void}) {
  return <select value={value} onChange={(event) => onChange(event.target.value)}>
    <option value="">{emptyLabel}</option>
    {options.map((option) => <option key={option} value={option}>{labels[option] ?? option}</option>)}
  </select>;
}

function CheckGroup({values, options, labels, onChange}: {values: string[]; options: readonly string[]; labels: Record<string, string>; onChange: (values: string[]) => void}) {
  return <div className="profile-checks">{options.map((option) => <label key={option}><input type="checkbox" checked={values.includes(option)} onChange={(event) => {
    onChange(event.target.checked ? [...values.filter((item) => item !== option), option] : values.filter((item) => item !== option));
  }}/>{labels[option] ?? option}</label>)}</div>;
}

function MapTextField({label, value, hint = 'One value per line.', onChange}: {label: string; value: string; hint?: string; onChange: (value: string) => void}) {
  return <Field label={label} hint={hint}><textarea rows={3} value={value} onChange={(event) => onChange(event.target.value)}/></Field>;
}

export function ProfileView({editor, documents, facts, onClose, onOpenSettings}: ProfileViewProps) {
  const {profile, capabilities, loading, error, notice, drafts} = editor;
  return <section className="profile-screen" aria-label="Candidate profile">
    <div className="profile-inner">
      <header className="profile-head">
        <div>
          <p className="company">Profile</p>
          <h1>Candidate profile</h1>
          <p className="muted">Only confirmed data is shown. Empty values are not set; legacy configuration is retained on the server and is not edited here.</p>
        </div>
        <div className="profile-head-actions">
          <button type="button" onClick={onClose}>Close profile</button>
        </div>
      </header>

      {profile ? <div className={`profile-derived ${profile.derivedCurrent ? '' : 'warning'}`}>
        {profile.derivedCurrent
          ? <p className="profile-hint">Quick ranking reflects the saved profile.</p>
          : <>
            <p className="profile-hint">Quick ranking is not current for the saved profile. Your profile is saved; refresh scoring to update it.</p>
            {editor.derivedError ? <p className="profile-hint warning">Scoring refresh failed; the saved profile is unchanged. Try again.</p> : null}
            <div className="profile-document-actions">
              <button type="button" disabled={editor.derivedRefreshing} onClick={() => void editor.refreshDerived()}>
                {editor.derivedRefreshing ? 'Refreshing…' : 'Refresh scoring'}
              </button>
            </div>
            {(profile.contextError || profile.fastRankingError || editor.derivedError) ? <details>
              <summary>Technical details</summary>
              <code>{editor.derivedError || profile.contextError || profile.fastRankingError}</code>
            </details> : null}
          </>}
      </div> : null}
      <p className="profile-hint">Quick match uses target roles and skill names as search signals. A skill without a level is not production experience, and Other requirements are used only in the detailed LLM assessment.</p>

      {error ? <div className="banner error" role="alert">{error}</div> : null}
      {notice ? <div className="banner" role="status">{notice}</div> : null}
      {documents.error ? <div className="banner error" role="alert">
        Documents: {documents.error}{documents.errorAction ? ` ${documents.errorAction}` : ''}
        {documents.errorOpenSettings ? <button type="button" onClick={onOpenSettings}>Open Settings</button> : null}
        {documents.errorTechnical ? <details><summary>Technical details</summary><code>{documents.errorTechnical}</code></details> : null}
      </div> : null}
      {documents.notice ? <div className="banner" role="status">{documents.notice}</div> : null}
      {facts.error ? <div className="banner error" role="alert">Facts: {facts.error}</div> : null}

      {loading ? <p>Loading profile…</p> : !profile ? <p>Profile unavailable. Reload the page after the server is available.</p> : <div className="profile-columns">
        <div className="profile-sections">
          <IdentitySection profile={profile} capabilities={capabilities} editor={editor}/>
          <DemographicsSection profile={profile} capabilities={capabilities} editor={editor}/>
          <SkillsSection profile={profile} capabilities={capabilities} editor={editor}/>
          <ExperienceSection profile={profile} editor={editor}/>
          <GoalsSection profile={profile} editor={editor}/>
          <CompensationSection profile={profile} capabilities={capabilities} editor={editor}/>
          <WorkSectionView profile={profile} capabilities={capabilities} editor={editor}/>
          <GeographySection profile={profile} editor={editor}/>
          <LanguagesSection profile={profile} capabilities={capabilities} editor={editor}/>
          <AvailabilitySection profile={profile} capabilities={capabilities} editor={editor}/>
          <OtherRequirementsSection profile={profile} capabilities={capabilities} editor={editor}/>
        </div>
        <aside className="profile-side">
          <DocumentsPanel documents={documents} capabilities={capabilities} facts={facts} onOpenSettings={onOpenSettings}/>
          <FactsPanel facts={facts} profile={profile} onEditCompensation={() => editor.beginEdit('compensation')}/>
        </aside>
      </div>}
    </div>
  </section>;
}

function IdentitySection({profile, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'identity');
  return <SectionShell title="Identity" description="The name shown on your profile and used in application answers." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('identity')} onCancel={() => editor.cancelEdit('identity')} onSave={() => void editor.saveSection('identity')}>
    {draft ? <div className="profile-fields">
      <Field label="Display name"><input value={draft.value.displayName} onChange={(event) => editor.changeDraft('identity', {...draft.value, displayName: event.target.value})}/></Field>
      <Field label="Legal name" hint="Optional; used only where an application asks for your legal name."><input value={draft.value.legalName ?? ''} onChange={(event) => editor.changeDraft('identity', {...draft.value, legalName: event.target.value})}/></Field>
    </div> : <div className="profile-read">
      <p className="profile-lead">{profile.data.identity.displayName || <NotSet/>}</p>
      <p><strong>Legal name:</strong> {profile.data.identity.legalName || <NotSet/>}</p>
    </div>}
  </SectionShell>;
}

function DemographicsSection({profile, capabilities, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'demographics');
  const current: Demographics = profile.data.demographics ?? {gender: '', race: [], ethnicity: ''};
  const genders = optionValues(capabilities, 'genders', GENDER_ORDER);
  const races = optionValues(capabilities, 'races', RACE_ORDER);
  const ethnicities = optionValues(capabilities, 'ethnicities', ETHNICITY_ORDER);
  const genderOptions = levelsWithRetained(genders, current.gender ?? '');
  const ethnicityOptions = levelsWithRetained(ethnicities, current.ethnicity ?? '');
  const raceOptions = races;
  return <SectionShell title="Application form answers" description="Optional answers used only for application questionnaires. They are never used in matching, quick ranking, fit assessment, preparation or document extraction, and nothing is sent automatically." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('demographics')} onCancel={() => editor.cancelEdit('demographics')} onSave={() => void editor.saveSection('demographics')}>
    {draft ? <div className="profile-fields">
      <Field label="Gender" hint="Not set means no answer; Prefer not to say is a deliberate answer.">
        <select value={draft.value.gender ?? ''} onChange={(event) => editor.changeDraft('demographics', {...draft.value, gender: event.target.value})}>
          <option value="">Not set</option>
          {genderOptions.map((option) => <option key={option} value={option}>{labelFor(option, GENDER_LABELS)}</option>)}
        </select>
      </Field>
      {draft.value.gender === 'self_describe' ? <Field label="Self-described gender">
        <input value={draft.value.genderDescription ?? ''} onChange={(event) => editor.changeDraft('demographics', {...draft.value, genderDescription: event.target.value})}/>
      </Field> : null}
      <fieldset className="profile-fieldset">
        <legend>Race</legend>
        <p className="profile-hint">Select all that apply. Prefer not to say replaces the other choices.</p>
        <div className="profile-checks">
          {raceOptions.map((option) => <label key={option}>
            <input type="checkbox" checked={draft.value.race.includes(option)} onChange={(event) => {
              editor.changeDraft('demographics', {...draft.value, race: toggleRace(draft.value.race, option, event.target.checked)});
            }}/>{labelFor(option, RACE_LABELS)}
          </label>)}
        </div>
      </fieldset>
      {draft.value.race.includes('self_describe') ? <Field label="Self-described race">
        <input value={draft.value.raceDescription ?? ''} onChange={(event) => editor.changeDraft('demographics', {...draft.value, raceDescription: event.target.value})}/>
      </Field> : null}
      <Field label="Ethnicity">
        <select value={draft.value.ethnicity ?? ''} onChange={(event) => editor.changeDraft('demographics', {...draft.value, ethnicity: event.target.value})}>
          <option value="">Not set</option>
          {ethnicityOptions.map((option) => <option key={option} value={option}>{labelFor(option, ETHNICITY_LABELS)}</option>)}
        </select>
      </Field>
    </div> : <div className="profile-read">
      <p><strong>Gender:</strong> {current.gender ? labelFor(current.gender, GENDER_LABELS) : <NotSet/>}{current.genderDescription ? ` — ${current.genderDescription}` : ''}</p>
      <div className="profile-read-line"><strong>Race:</strong> {current.race.length ? <Tags items={current.race} labels={RACE_LABELS}/> : <NotSet/>}{current.raceDescription ? ` — ${current.raceDescription}` : ''}</div>
      <p><strong>Ethnicity:</strong> {current.ethnicity ? labelFor(current.ethnicity, ETHNICITY_LABELS) : <NotSet/>}</p>
    </div>}
  </SectionShell>;
}

function SkillsSection({profile, capabilities, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'skills');
  const currentLevels = skillLevelOptions(capabilities);
  return <SectionShell title="Skills" description="Skill names are quick-ranking search signals. The level is used only in the detailed assessment and preparation, not in the quick score." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('skills')} onCancel={() => editor.cancelEdit('skills')} onSave={() => void editor.saveSection('skills')}>
    {draft ? <div className="profile-rows">
      {draft.value.map((skill, index) => <div className="profile-row profile-row-skill" key={skill.id ?? `${skill.key}-${index}`}>
        <Field label="Name"><input value={skill.name} onChange={(event) => editor.changeDraft('skills', replaceAt(draft.value, index, {...skill, name: event.target.value}))}/></Field>
        <Field label="Level"><select value={skill.level} onChange={(event) => editor.changeDraft('skills', replaceAt(draft.value, index, {...skill, level: event.target.value}))}>
          <option value="">Not set</option>
          {levelsWithRetained(currentLevels, skill.level).map((level) => <option key={level} value={level}>{skillLevelText(level, capabilities)}</option>)}
        </select></Field>
        <button type="button" onClick={() => editor.changeDraft('skills', draft.value.filter((_, current) => current !== index))}>Remove</button>
      </div>)}
      <button type="button" onClick={() => editor.changeDraft('skills', [...draft.value, emptySkill()])}>Add skill</button>
    </div> : profile.data.skills.length ? <ul className="profile-list">{profile.data.skills.map((skill) => <li key={skill.id ?? skill.key}>
      <strong>{skill.name}</strong> <span className="profile-level">{skillLevelText(skill.level, capabilities)}</span>
      <Provenance value={skill.provenance}/>
    </li>)}</ul> : <NotSet/>}
  </SectionShell>;
}

function ExperienceSection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'experience');
  return <SectionShell title="Experience" description="Used by the detailed assessment and preparation; not part of the quick score. Dates are month/year and never invent a day." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('experience')} onCancel={() => editor.cancelEdit('experience')} onSave={() => void editor.saveSection('experience')}>
    {draft ? <div className="profile-rows">
      {draft.value.map((row, index) => <div className="profile-row profile-row-wide profile-row-experience" key={row.id ?? row.clientKey ?? index}>
        <Field label="Company"><input value={row.company} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, company: event.target.value}))}/></Field>
        <Field label="Role"><input value={row.title} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, title: event.target.value}))}/></Field>
        <MonthYearInput label="Start" value={row.start ?? ''} onChange={(value) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, start: value}))}/>
        <MonthYearInput label="End" value={row.end ?? ''} disabled={!!row.current} hint={row.current ? 'A current role has no end date.' : undefined} onChange={(value) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, end: value}))}/>
        <label className="profile-inline-check">
          <input type="checkbox" checked={!!row.current} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, current: event.target.checked}))}/>
          Currently working here
        </label>
        <RichTextEditor label="Summary" value={{text: row.summary ?? '', document: row.summaryDocument}}
          hint="Used in the detailed assessment and preparation. Existing plain text is retained."
          onChange={(value) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, summary: value.text, summaryDocument: value.document}))}/>
        <button type="button" onClick={() => editor.changeDraft('experience', draft.value.filter((_, current) => current !== index))}>Remove</button>
      </div>)}
      <button type="button" onClick={() => editor.changeDraft('experience', [...draft.value, emptyExperience()])}>Add experience</button>
    </div> : profile.data.experience.length ? <ul className="profile-list">{profile.data.experience.map((row, index) => <li key={row.id ?? row.clientKey ?? `${row.company}-${index}`}>
      <strong>{row.title || <NotSet/>}</strong> · {row.company || <NotSet/>}
      <p className="muted">{row.start || <NotSet/>} – {row.current ? 'Present' : row.end || <NotSet/>}</p>
      {row.summary || row.summaryDocument ? <RichTextView value={{text: row.summary ?? '', document: row.summaryDocument}}/> : null}
    </li>)}</ul> : <NotSet/>}
  </SectionShell>;
}

function GoalsSection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'goals');
  const current: Goals = {targetRoles: profile.data.targetRoles, targetIndustries: profile.data.targetIndustries};
  return <SectionShell title="Target roles and industries" editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('goals')} onCancel={() => editor.cancelEdit('goals')} onSave={() => void editor.saveSection('goals')}>
    {draft ? <div className="profile-fields">
      <MapTextField label="Target roles" hint="Free-form, one per line. Roles are quick-ranking search signals and are also used in the detailed assessment." value={listText(draft.value.targetRoles)} onChange={(value) => editor.changeDraft('goals', {...draft.value, targetRoles: textList(value)})}/>
      <MapTextField label="Target industries" hint="Free-form, one per line. Industries are used in the detailed assessment, not in the quick score." value={listText(draft.value.targetIndustries)} onChange={(value) => editor.changeDraft('goals', {...draft.value, targetIndustries: textList(value)})}/>
    </div> : <div className="profile-fields">
      <div><h3>Roles</h3><Tags items={current.targetRoles}/></div>
      <div><h3>Industries</h3><Tags items={current.targetIndustries}/></div>
    </div>}
  </SectionShell>;
}

function CompensationSection({profile, capabilities, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'compensation');
  const value: Compensation = profile.data.compensation;
  const currencies = optionValues(capabilities, 'currencies', CURRENCY_ORDER);
  const periods = optionValues(capabilities, 'compensation_periods', ['month', 'year']);
  const crypto = optionValues(capabilities, 'crypto_payments', CRYPTO_ORDER);
  // Crypto accepted is the effective no-restriction default. It is shown without being written
  // when the owner has not made an explicit payment choice.
  const cryptoValue = draft?.value.crypto || 'crypto_accepted';
  const grossNetText = value.grossNet ? (GROSS_NET_LABELS[value.grossNet] ?? value.grossNet) : '';
  const periodText = value.period ? (PERIOD_LABELS[value.period] ?? value.period) : '';
  return <SectionShell title="Compensation" description="Used by the detailed assessment, preparation and application answers; not the quick score. Amounts are not converted, and an unselected tax basis or period is never assumed." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('compensation')} onCancel={() => editor.cancelEdit('compensation')} onSave={() => void editor.saveSection('compensation')}>
    {draft ? <div className="profile-fields">
      <div className="profile-fields-grid">
        <Field label="Minimum compensation"><input type="number" min="0" value={numberText(draft.value.cashMin)} onChange={(event) => editor.changeDraft('compensation', {...draft.value, cashMin: numberOrNull(event.target.value)})}/></Field>
        <Field label="Desired compensation"><input type="number" min="0" value={numberText(draft.value.cashTarget)} onChange={(event) => editor.changeDraft('compensation', {...draft.value, cashTarget: numberOrNull(event.target.value)})}/></Field>
        <RetainedSelect label="Currency" value={draft.value.currency} options={currencies} labels={CURRENCY_LABELS} onChange={(next) => editor.changeDraft('compensation', {...draft.value, currency: next})}/>
        <RetainedSelect label="Taxes" value={draft.value.grossNet} options={GROSS_NET} labels={GROSS_NET_LABELS} onChange={(next) => editor.changeDraft('compensation', {...draft.value, grossNet: next})}/>
        <RetainedSelect label="Amount period" value={draft.value.period} options={periods} labels={PERIOD_LABELS} onChange={(next) => editor.changeDraft('compensation', {...draft.value, period: next})}/>
      </div>
      <p className="profile-hint">A stored currency or period that is not in the current choices stays visible as a retained value. It is never converted or silently replaced.</p>
      <div className="profile-fields-grid">
        <RetainedSelect label="Payment method" value={cryptoValue} options={crypto} labels={CRYPTO_LABELS} allowEmpty={false} onChange={(next) => editor.changeDraft('compensation', {...draft.value, crypto: next})}/>
      </div>
      <p className="profile-hint">Crypto accepted is the default and means no restriction on the payment method; it is not a requirement. The default is not written unless you choose an explicit payment method.</p>
      <label className="profile-inline-check"><input type="checkbox" checked={!!draft.value.equityOnTop} onChange={(event) => editor.changeDraft('compensation', {...draft.value, equityOnTop: event.target.checked})}/>Interested in equity/tokens in addition to base pay</label>
      <p className="profile-hint">Off means no additional wish, not a prohibition. Equity or tokens never replace the minimum guaranteed cash.</p>
      <details className="profile-legacy-block">
        <summary>Legacy day rates (retained)</summary>
        <p className="profile-hint">Kept from an older contract and still editable. They are separate from the monthly/yearly cash amount above.</p>
        <div className="profile-fields-grid">
          <Field label="Day-rate minimum"><input type="number" min="0" value={numberText(draft.value.dayRates.min)} onChange={(event) => editor.changeDraft('compensation', {...draft.value, dayRates: {...draft.value.dayRates, min: numberOrNull(event.target.value)}})}/></Field>
          <Field label="Day-rate target"><input type="number" min="0" value={numberText(draft.value.dayRates.target)} onChange={(event) => editor.changeDraft('compensation', {...draft.value, dayRates: {...draft.value.dayRates, target: numberOrNull(event.target.value)}})}/></Field>
          <RetainedSelect label="Day-rate currency" value={draft.value.dayRates.currency} options={currencies} labels={CURRENCY_LABELS} onChange={(next) => editor.changeDraft('compensation', {...draft.value, dayRates: {...draft.value.dayRates, currency: next}})}/>
          <RetainedSelect label="Day-rate period" value={draft.value.dayRates.period} options={['day']} labels={PERIOD_LABELS} onChange={(next) => editor.changeDraft('compensation', {...draft.value, dayRates: {...draft.value.dayRates, period: next}})}/>
        </div>
      </details>
    </div> : <div className="profile-read">
      <p><strong>Cash:</strong> {value.cashMin === null && value.cashTarget === null
        ? (value.currency ? <>No amount · currency retained: {value.currency}</> : <NotSet/>)
        : <><Money amount={value.cashMin} currency={value.currency}/> → <Money amount={value.cashTarget} currency={value.currency}/></>}</p>
      <p><strong>Taxes and period:</strong> {[grossNetText, periodText].filter(Boolean).join(' · ') || <NotSet/>}</p>
      <p><strong>Payment method:</strong> {value.crypto ? CRYPTO_LABELS[value.crypto] ?? value.crypto : <>Crypto accepted (default — no restriction)</>}</p>
      <p><strong>Equity/tokens:</strong> {value.equityOnTop ? 'Interested in addition to base pay' : 'No additional equity/token request'}</p>
      <p><strong>Legacy day rates:</strong> {value.dayRates.min === null && value.dayRates.target === null
        ? ((value.dayRates.currency || value.dayRates.period)
          ? <>No amount{value.dayRates.currency ? ` · currency retained: ${value.dayRates.currency}` : ''}{value.dayRates.period ? ` · ${PERIOD_LABELS[value.dayRates.period] ?? value.dayRates.period}` : ''}</>
          : <NotSet/>)
        : <><Money amount={value.dayRates.min} currency={value.dayRates.currency}/> → <Money amount={value.dayRates.target} currency={value.dayRates.currency}/>{value.dayRates.period ? ` · ${PERIOD_LABELS[value.dayRates.period] ?? value.dayRates.period}` : ''}</>}</p>
    </div>}
  </SectionShell>;
}

function WorkSectionView({profile, capabilities, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'work');
  const value: WorkSection = {engagement: profile.data.engagement, hours: profile.data.hours, workModes: profile.data.workModes};
  const engagementOptions = optionValues(capabilities, 'engagement_types', ENGAGEMENT_TYPES);
  const hoursOptions = optionValues(capabilities, 'hours_modes', HOURS_MODES);
  const workModeOptions = optionValues(capabilities, 'work_modes', WORK_MODES);
  const hours = draft?.value.hours ?? value.hours;
  const storedHours = hours.weekly !== null || hours.weeklyMin != null || hours.weeklyMax != null;
  const partTime = hours.mode === 'part_time';
  return <SectionShell title="Engagement, hours and work modes" description="Used by the detailed assessment and preparation; not the quick score. Employment and Contract may both be acceptable." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('work')} onCancel={() => editor.cancelEdit('work')} onSave={() => void editor.saveSection('work')}>
    {draft ? <div className="profile-fields">
      <div><h3>Engagement</h3>
        <RetainedCheckGroup values={draft.value.engagement.types} options={engagementOptions} labels={ENGAGEMENT_LABELS} hint="Consultant work belongs to Contract. Legacy Employee/Contractor selections stay visible and selected until you replace them." onChange={(values) => editor.changeDraft('work', {...draft.value, engagement: {types: values}})}/>
      </div>
      <div>
        <h3>Working time</h3>
        <Field label="Working time"><EnumSelect value={draft.value.hours.mode} options={hoursOptions} labels={HOURS_LABELS} onChange={(next) => editor.changeDraft('work', {...draft.value, hours: {...draft.value.hours, mode: next}})}/></Field>
        {partTime ? <div className="profile-fields-grid">
          <Field label="Desired hours per week" hint="The old weekly value is reused here, never hidden."><input type="number" min="0" value={numberText(draft.value.hours.weekly)} onChange={(event) => editor.changeDraft('work', {...draft.value, hours: {...draft.value.hours, weekly: numberOrNull(event.target.value)}})}/></Field>
          <Field label="Minimum hours per week (optional)"><input type="number" min="0" value={numberText(draft.value.hours.weeklyMin ?? null)} onChange={(event) => editor.changeDraft('work', {...draft.value, hours: {...draft.value.hours, weeklyMin: numberOrNull(event.target.value)}})}/></Field>
          <Field label="Maximum hours per week (optional)"><input type="number" min="0" value={numberText(draft.value.hours.weeklyMax ?? null)} onChange={(event) => editor.changeDraft('work', {...draft.value, hours: {...draft.value.hours, weeklyMax: numberOrNull(event.target.value)}})}/></Field>
        </div> : null}
        {draft.value.hours.mode === 'full_time' && storedHours ? <p className="profile-hint">Full-time selected. Stored part-time hours are retained but are not used as a full-time restriction.</p> : null}
        {!draft.value.hours.mode && storedHours ? <p className="profile-hint">Working time is not set. Stored part-time hours are retained and are not used as a restriction.</p> : null}
        {partTime && (draft.value.hours.weeklyMin != null || draft.value.hours.weeklyMax != null) ? <p className="profile-hint">The detailed assessment uses the desired weekly hours and the optional min/max range you set.</p> : null}
      </div>
      <div><h3>Work modes</h3><CheckGroup values={draft.value.workModes.modes} options={workModeOptions} labels={WORK_MODE_LABELS} onChange={(values) => editor.changeDraft('work', {...draft.value, workModes: {...draft.value.workModes, modes: values}})}/></div>
    </div> : <div className="profile-read">
      <div className="profile-read-line"><strong>Engagement:</strong> <Tags items={value.engagement.types} labels={ENGAGEMENT_LABELS}/></div>
      {value.engagement.types.some((type) => type === 'employee' || type === 'contractor') ? <p className="profile-hint">Legacy Employee/Contractor selections are retained; replace them with Employment/Contract.</p> : null}
      <p><strong>Working time:</strong> {value.hours.mode ? HOURS_LABELS[value.hours.mode] ?? value.hours.mode : <NotSet/>}{value.hours.mode === 'part_time' ? <>
        {value.hours.weekly !== null ? ` · ${value.hours.weekly} hours/week desired` : ''}
        {value.hours.weeklyMin != null || value.hours.weeklyMax != null ? ` · range ${value.hours.weeklyMin ?? '?'}–${value.hours.weeklyMax ?? '?'} hours/week` : ''}
      </> : null}</p>
      {value.hours.mode !== 'part_time' && (value.hours.weekly !== null || value.hours.weeklyMin != null || value.hours.weeklyMax != null) ? <p className="profile-hint">Stored part-time hours are retained; they are not used as {value.hours.mode === 'full_time' ? 'a full-time' : 'an active'} restriction.</p> : null}
      <div className="profile-read-line"><strong>Work modes:</strong> <Tags items={value.workModes.modes} labels={WORK_MODE_LABELS}/></div>
      {value.workModes.limits.length ? <p className="profile-hint">Previous work limits are retained and editable under Other requirements.</p> : null}
    </div>}
  </SectionShell>;
}

function authorizationLabel(status: string): string {
  return status ? AUTHORIZATION_LABELS[status] ?? `${status} (retained value)` : 'Not set';
}

function GeographySection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'geography');
  const value: Geography = profile.data.geography;
  const currentActive = Boolean(value.currentLocationSet) || value.currentLocation != null;
  const preferredActive = Boolean(value.preferredLocationsSet) || (value.preferredLocations?.length ?? 0) > 0;
  const authorizationActive = Boolean(value.workAuthorizationsSet) || (value.workAuthorizations?.length ?? 0) > 0;
  const timezonesActive = Boolean(value.timezonesSet) || (value.timezoneIds?.length ?? 0) > 0;
  const legacyCurrent = value.current.trim();
  const legacyDesired = value.desired.filter((item) => item.trim() !== '');
  const legacyEligibility = value.eligibility.filter((item) => item.trim() !== '');
  const legacyTimezones = value.timezones.filter((item) => item.trim() !== '');
  const availabilityTimezone = profile.data.availability.timezone.trim();
  return <SectionShell title="Geography and authorization" description="Used by the detailed assessment and preparation; not the quick score. No stated eligibility is not ineligibility, and current residence does not prove work authorization." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('geography')} onCancel={() => editor.cancelEdit('geography')} onSave={() => void editor.saveSection('geography')}>
    {draft ? <div className="profile-fields">
      <div>
        <h3>Current location</h3>
        <LocationFields value={draft.value.currentLocation ?? {country: '', city: ''}} onChange={(next) => editor.changeDraft('geography', {...draft.value, currentLocation: next})}/>
        <p className="profile-hint">Where you live now. It does not by itself prove work authorization in that country.</p>
        {draft.value.current ? <p className="profile-hint">Previous location text retained: {draft.value.current}. It is not used after you choose a country.</p> : null}
      </div>
      <div>
        <h3>Preferred work locations</h3>
        <LocationsEditor rows={draft.value.preferredLocations ?? []} onChange={(rows) => editor.changeDraft('geography', {...draft.value, preferredLocations: rows})}/>
        {legacyDesired.length ? <p className="profile-hint">Previous free-form desired locations retained: {legacyDesired.join('; ')}. They are not active after you add a structured location.</p> : null}
      </div>
      <div>
        <h3>Work authorization by country</h3>
        <AuthorizationEditor rows={draft.value.workAuthorizations ?? []} onChange={(rows) => editor.changeDraft('geography', {...draft.value, workAuthorizations: rows})}/>
        <p className="profile-hint">Add each country separately, including a country that is not a preferred location. Current residence is not inferred as eligibility.</p>
        {(legacyEligibility.length || draft.value.sponsorship) ? <p className="profile-hint">Previous authorization text retained: {[...legacyEligibility, draft.value.sponsorship ? SPONSORSHIP_LABELS[draft.value.sponsorship] ?? draft.value.sponsorship : ''].filter(Boolean).join('; ')}. It is not active after you add a country-specific answer.</p> : null}
      </div>
      <div>
        <h3>Relocation</h3>
        <div className="profile-fields-grid">
          <Field label="Willing to relocate"><TriState value={draft.value.willingToRelocate} onChange={(next) => editor.changeDraft('geography', {...draft.value, willingToRelocate: next})}/></Field>
        </div>
        <MapTextField label="Retained relocation notes (legacy)" value={listText(draft.value.relocation)} hint="Kept for compatibility; use the preferred locations and willingness above for current answers." onChange={(next) => editor.changeDraft('geography', {...draft.value, relocation: textList(next)})}/>
      </div>
      <div>
        <h3>Timezones and overlap</h3>
        <TimezonePicker values={draft.value.timezoneIds ?? []} hint="IANA timezone IDs with human region/city labels. There is one timezone master, here in Geography." onChange={(values) => editor.changeDraft('geography', {...draft.value, timezoneIds: values})}/>
        <Field label="Desired overlap (hours)" hint="Optional. Without a vacancy's working interval, overlap is not compared."><input type="number" min="0" value={numberText(draft.value.desiredOverlapHours ?? null)} onChange={(event) => editor.changeDraft('geography', {...draft.value, desiredOverlapHours: numberOrNull(event.target.value)})}/></Field>
        {legacyTimezones.length ? <p className="profile-hint">Previous free-form timezones retained: {legacyTimezones.join('; ')}. They are not active after you choose an IANA timezone.</p> : null}
        {availabilityTimezone ? <p className="profile-hint">Availability timezone retained as compatibility data: {availabilityTimezone}. It is not a second live timezone field.</p> : null}
      </div>
    </div> : <div className="profile-read">
      <p><strong>Current location:</strong> {currentActive
        ? (value.currentLocation ? locationLabel(value.currentLocation) : <NotSet/>)
        : (legacyCurrent || <NotSet/>)}</p>
      {!currentActive && legacyCurrent ? <p className="profile-hint">Legacy location text, used until a structured country/city is added. It is never parsed into a country.</p> : null}
      {currentActive && legacyCurrent ? <p className="profile-hint">Previous location text retained: {legacyCurrent}. It is not active.</p> : null}
      <div className="profile-read-line"><strong>Preferred work locations:</strong> {preferredActive ? <LocationTags locations={value.preferredLocations ?? []}/> : (legacyDesired.length ? <Tags items={legacyDesired}/> : <NotSet/>)}</div>
      {!preferredActive && legacyDesired.length ? <p className="profile-hint">Legacy desired-location text, used until structured locations are added.</p> : null}
      {preferredActive && legacyDesired.length ? <p className="profile-hint">Previous free-form desired locations retained: {legacyDesired.join('; ')}. They are not active.</p> : null}
      <div className="profile-read-line"><strong>Work authorization:</strong> {authorizationActive
        ? ((value.workAuthorizations?.length ?? 0) ? <ul className="profile-list">{(value.workAuthorizations ?? []).map((row, index) => <li key={`${row.country}-${index}`}><strong>{countryName(row.country)}</strong> <span className="profile-level">{authorizationLabel(row.status ?? '')}</span></li>)}</ul> : <NotSet/>)
        : (legacyEligibility.length || value.sponsorship ? <>{legacyEligibility.join('; ')}{value.sponsorship ? `${legacyEligibility.length ? ' · ' : ''}${SPONSORSHIP_LABELS[value.sponsorship] ?? value.sponsorship}` : ''}</> : <NotSet/>)}</div>
      {!authorizationActive && (legacyEligibility.length || value.sponsorship) ? <p className="profile-hint">Legacy authorization text, used until country-specific answers are added. Current residence is not inferred as eligibility.</p> : null}
      {authorizationActive && (legacyEligibility.length || value.sponsorship) ? <p className="profile-hint">Previous authorization text retained: {[...legacyEligibility, value.sponsorship ? SPONSORSHIP_LABELS[value.sponsorship] ?? value.sponsorship : ''].filter(Boolean).join('; ')}. It is not active.</p> : null}
      <p><strong>Willing to relocate:</strong> {value.willingToRelocate === null ? <NotSet/> : value.willingToRelocate ? 'Yes' : 'No'}</p>
      {value.relocation.length ? <div className="profile-read-line"><strong>Relocation notes (legacy):</strong> <Tags items={value.relocation}/></div> : null}
      <div className="profile-read-line"><strong>Timezones:</strong> {timezonesActive
        ? ((value.timezoneIds?.length ?? 0) ? <div className="profile-tags">{(value.timezoneIds ?? []).map((id) => <span key={id}>{timezoneLabel(id)}</span>)}</div> : <NotSet/>)
        : (legacyTimezones.length ? <Tags items={legacyTimezones}/> : (availabilityTimezone || <NotSet/>))}</div>
      {!timezonesActive && legacyTimezones.length ? <p className="profile-hint">Legacy free-form timezones, used until IANA timezones are selected.</p> : null}
      {!timezonesActive && availabilityTimezone ? <p className="profile-hint">Availability timezone retained as compatibility data: {availabilityTimezone}.</p> : null}
      {timezonesActive && legacyTimezones.length ? <p className="profile-hint">Previous free-form timezones retained: {legacyTimezones.join('; ')}. They are not active.</p> : null}
      {timezonesActive && availabilityTimezone ? <p className="profile-hint">Availability timezone retained as compatibility data: {availabilityTimezone}. It is not a second live timezone field.</p> : null}
      <p><strong>Desired overlap:</strong> {value.desiredOverlapHours != null ? `${value.desiredOverlapHours} hours` : <NotSet/>}</p>
      <p className="profile-hint">Overlap is not compared until a vacancy's working interval is available.</p>
    </div>}
  </SectionShell>;
}

function LanguagesSection({profile, capabilities, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'languages');
  const levelOptions = optionValues(capabilities, 'language_levels', LANGUAGE_LEVEL_ORDER);
  const listId = 'profile-language-names';
  return <SectionShell title="Languages" description="Used by the detailed assessment and preparation; not part of the quick score. Pre-A1 is an honest beginner label, not CEFR A0." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('languages')} onCancel={() => editor.cancelEdit('languages')} onSave={() => void editor.saveSection('languages')}>
    <datalist id={listId}>{LANGUAGE_NAMES.map((name) => <option key={name} value={name}/>)}</datalist>
    {draft ? <div className="profile-rows">
      {draft.value.map((row, index) => <div className="profile-row profile-row-language" key={index}>
        <Field label="Language" hint="Choose a name or type another."><input list={listId} value={row.name} onChange={(event) => editor.changeDraft('languages', replaceAt(draft.value, index, {...row, name: event.target.value}))}/></Field>
        <Field label="Level"><select value={row.level} onChange={(event) => editor.changeDraft('languages', replaceAt(draft.value, index, {...row, level: event.target.value}))}>
          <option value="">Not set</option>
          {levelsWithRetained(levelOptions, row.level).map((level) => <option key={level} value={level}>{languageLevelText(level)}</option>)}
        </select></Field>
        <button type="button" onClick={() => editor.changeDraft('languages', draft.value.filter((_, current) => current !== index))}>Remove</button>
      </div>)}
      <button type="button" onClick={() => editor.changeDraft('languages', [...draft.value, {name: '', level: ''}])}>Add language</button>
    </div> : profile.data.languages.length ? <ul className="profile-list">{profile.data.languages.map((row, index) => <li key={`${row.name}-${index}`}><strong>{row.name}</strong> <span className="profile-level">{languageLevelText(row.level)}</span></li>)}</ul> : <NotSet/>}
  </SectionShell>;
}

function noticeText(value: Availability): React.ReactNode {
  if (value.noticeValue !== null && value.noticeValue !== undefined && value.noticeUnit) {
    return `${value.noticeValue} ${value.noticeUnit}${value.noticeValue === 1 ? '' : 's'}`;
  }
  // Once the structured notice is deliberately cleared, the legacy wording is retained data,
  // not an acting notice. It must not reappear as the read-view notice period.
  if (value.noticeSet) return <NotSet/>;
  if (value.noticePeriod) return `${value.noticePeriod} (retained wording)`;
  return <NotSet/>;
}

function AvailabilitySection({profile, capabilities, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'availability');
  const value: Availability = profile.data.availability;
  const units = optionValues(capabilities, 'notice_units', NOTICE_UNIT_ORDER);
  return <SectionShell title="Availability" description="Used by the detailed assessment and preparation; not part of the quick score. Timezones live under Geography, not here." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('availability')} onCancel={() => editor.cancelEdit('availability')} onSave={() => void editor.saveSection('availability')}>
    {draft ? <div className="profile-fields">
      <div className="profile-fields-grid">
        <Field label="Notice value"><input type="number" min="0" value={numberText(draft.value.noticeValue ?? null)} placeholder="Not set" onChange={(event) => editor.changeDraft('availability', {...draft.value, noticeValue: numberOrNull(event.target.value)})}/></Field>
        <Field label="Notice unit"><select value={draft.value.noticeUnit ?? ''} onChange={(event) => editor.changeDraft('availability', {...draft.value, noticeUnit: event.target.value})}>
          <option value="">Not set</option>
          {levelsWithRetained(units, draft.value.noticeUnit ?? '').map((unit) => <option key={unit} value={unit}>{labelFor(unit, NOTICE_UNIT_LABELS)}</option>)}
        </select></Field>
      </div>
      {draft.value.noticePeriod ? <p className="profile-hint">Retained previous notice wording: {draft.value.noticePeriod}. Choose a number and unit to replace it.</p> : null}
      <DateInput label="Available from" value={draft.value.startDate} hint="DD/MM/YYYY. An older unknown value stays visible until you replace it." onChange={(next) => editor.changeDraft('availability', {...draft.value, startDate: next})}/>
    </div> : <div className="profile-read">
      <p><strong>Notice period:</strong> {noticeText(value)}</p>
      {value.noticeSet && !(value.noticeValue != null && value.noticeUnit) && value.noticePeriod ? <p className="profile-hint">Previous wording retained: {value.noticePeriod}. It is not the active notice period.</p> : null}
      <p><strong>Available from:</strong> {value.startDate || <NotSet/>}</p>
    </div>}
  </SectionShell>;
}

function OtherRequirementsSection({profile, capabilities, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'other-requirements');
  const current: OtherRequirements = profile.data.otherRequirements ?? {text: ''};
  const legacy = [...profile.data.dealbreakers, ...profile.data.workModes.limits].filter((item) => item.trim() !== '');
  const retained = current.selected || (current.text ?? '').trim() !== '' ? current.text : legacy.join('\n');
  const hint = 'Used by the detailed assessment and preparation; quick ranking does not interpret this text. Previously stored dealbreakers and work limits appear here until you replace them.';
  return <SectionShell title="Other requirements" description={hint} editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('other-requirements')} onCancel={() => editor.cancelEdit('other-requirements')} onSave={() => void editor.saveSection('other-requirements')}>
    {draft ? <RichTextEditor label="Other requirements" value={{text: draft.value.text, document: draft.value.document}}
      hint="Plain text is stored for the assessment; bold, italic, headings and lists are preserved in the profile."
      onChange={(next) => editor.changeDraft('other-requirements', {...draft.value, text: next.text, document: next.document})}/>
      : retained.trim() || current.document ? <RichTextView value={{text: retained, document: current.document}}/> : <NotSet/>}
  </SectionShell>;
}

function DocumentsPanel({documents, capabilities, facts, onOpenSettings}: {documents: ProfileDocuments; capabilities: Capabilities | null; facts: ProfileFacts; onOpenSettings: () => void}) {
  const fileInput = useRef<HTMLInputElement>(null);
  const [textName, setTextName] = useState('');
  const [text, setText] = useState('');
  const [openText, setOpenText] = useState<Record<string, boolean>>({});
  const [uploadError, setUploadError] = useState('');
  return <section className="profile-card profile-documents">
    <header className="profile-card-head"><div><h2>Documents and facts</h2><p className="muted">Uploads and analysis are explicit. Saving a section never starts analysis.</p></div>
      <button type="button" onClick={() => void documents.refresh()} disabled={documents.loading}>Refresh</button>
    </header>
    <div className="profile-card-body">
      <p className="profile-hint">{capabilities ? `Accepted types: ${capabilities.documentTypes.join(', ')}. Maximum: ${formatBytes(capabilities.maxDocumentBytes)}.` : 'Loading upload capabilities…'}</p>
      <div className="profile-upload">
        <input ref={fileInput} type="file" accept={capabilities?.documentTypes.map((type) => `.${type}`).join(',')} onChange={(event) => {
          const file = event.target.files?.[0];
          if (file) void documents.uploadFile(file).then((ok) => { setUploadError(ok ? '' : documents.error); });
          event.target.value = '';
        }} disabled={documents.busy === 'upload' || !capabilities}/>
        {documents.busy === 'upload' ? <span className="muted">Uploading…</span> : null}
      </div>
      <details className="profile-text-upload">
        <summary>Add a text document</summary>
        <Field label="Filename"><input value={textName} placeholder="profile.txt" onChange={(event) => setTextName(event.target.value)}/></Field>
        <Field label="Text"><textarea rows={5} value={text} onChange={(event) => setText(event.target.value)}/></Field>
        <button type="button" disabled={documents.busy === 'upload'} onClick={() => void documents.uploadText(textName, text).then((ok) => {
          if (ok) { setText(''); setTextName(''); setUploadError(''); } else setUploadError(documents.error);
        })}>Add text</button>
      </details>
      {uploadError ? <p className="error">{uploadError}</p> : null}
      {documents.loading ? <p>Loading documents…</p> : documents.documents.length ? <div className="profile-document-list">
        {documents.documents.map((document) => {
          const busy = documents.busy === document.documentId;
          const summary = documentSummary(document);
          const textState = documents.texts[document.documentId];
          return <article className="profile-document" key={document.documentId}>
            <div className="profile-document-head"><strong>{document.originalName}</strong><span className={`profile-state state-${document.processingState}`}>{summary.label}</span></div>
            <p className="muted">{document.kind} · {document.textChars.toLocaleString()} text characters{summary.factsText ? ` · ${summary.factsText}` : ''}</p>
            {document.sourceUrl ? <p className="muted">Source: <a href={document.sourceUrl} target="_blank" rel="noreferrer">{document.sourceUrl}</a></p> : null}
            {summary.error ? null : <p className="profile-document-detail">{summary.detail}</p>}
            {summary.error ? <div className="profile-document-error">
              <p><strong>{summary.error.cause}</strong> {summary.error.action}{summary.error.openSettings ? <button type="button" onClick={onOpenSettings}>Open Settings</button> : null}</p>
              {summary.technical ? <details><summary>Technical details</summary><code>{summary.technical}</code></details> : null}
            </div> : null}
            <div className="profile-document-actions">
              {summary.action === 'cancel' ? <button type="button" disabled={busy} onClick={() => void documents.cancel(document.documentId)}>Cancel analysis</button> : null}
              {summary.action === 'retry' ? <button type="button" disabled={busy} onClick={() => void documents.retry(document.documentId)}>Retry analysis</button> : null}
              {summary.action === 'analyze' ? <button type="button" disabled={busy} onClick={() => void documents.analyze(document.documentId)}>{summary.actionLabel}</button> : null}
              {summary.needsUpload ? <span className="muted">Upload a usable file above; retrying this source would have nothing to read.</span> : null}
              {document.hasText ? <button type="button" onClick={() => {
                const next = !openText[document.documentId];
                setOpenText((current) => ({...current, [document.documentId]: next}));
                if (next && !documents.texts[document.documentId]?.text && !documents.texts[document.documentId]?.loading) void documents.loadText(document.documentId);
              }}>{openText[document.documentId] ? 'Hide text' : 'View text'}</button> : null}
              {document.hasOriginal ? <a href={documents.fileUrl(document.documentId)} target="_blank" rel="noreferrer">Original</a> : null}
            </div>
            {openText[document.documentId] ? <div className="profile-document-text">
              {textState?.loading ? <p>Loading text…</p> : textState?.error ? <p className="error">{textState.error}</p> : <pre>{textState?.text || 'No text loaded.'}</pre>}
            </div> : null}
          </article>;
        })}
      </div> : <p className="muted">No documents yet. Upload a CV or add text; nothing is analyzed automatically.</p>}
      <p className="profile-hint">Facts are proposals with their source. Accepting one can update the profile; rejected proposals never change it. Manual correction belongs in the section editors above.</p>
      <button type="button" onClick={() => void facts.refresh()}>Refresh fact proposals</button>
    </div>
  </section>;
}

function FactsPanel({facts, profile, onEditCompensation}: {facts: ProfileFacts; profile: Profile; onEditCompensation: () => void}) {
  return <section className="profile-card profile-facts">
    <header className="profile-card-head"><div><h2>Fact proposals</h2><p className="muted">Each value keeps its exact source quote. Accept only the original proposal; to correct it, edit the profile section manually.</p></div></header>
    <div className="profile-card-body">
      {facts.loading ? <p>Loading facts…</p> : facts.facts.length ? <div className="profile-fact-list">{facts.facts.map((fact) => <FactRow key={fact.factId} fact={fact} facts={facts} profile={profile} onEditCompensation={onEditCompensation}/>)}</div> : <p className="muted">No fact proposals. Use Analyze on a document to extract them; this screen never starts that automatically.</p>}
    </div>
  </section>;
}

function FactRow({fact, facts, profile, onEditCompensation}: {fact: Fact; facts: ProfileFacts; profile: Profile; onEditCompensation: () => void}) {
  const serverConflict = facts.conflicts[fact.factId];
  const conflict = serverConflict ?? (fact.state === 'conflict' ? {
    factId: fact.factId, fieldKey: fact.fieldKey, current: currentValue(profile.data, fact.fieldKey),
    proposed: fact.value, currentRevision: profile.revision, message: 'Unresolved conflict with the stored profile.',
  } : null);
  const busy = facts.busy[fact.factId] ?? '';
  const error = facts.actionError[fact.factId];
  const needsCompensation = error && fact.fieldKey.startsWith('compensation.') && /currency|period|money/i.test(error);
  return <article className="profile-fact">
    <div className="profile-fact-head"><code>{fact.fieldKey}</code><span className={`profile-state state-${fact.state}`}>{fact.state}</span></div>
    {conflict ? <div className="profile-conflict">
      <p><strong>Current:</strong> {factValueText(conflict.current)}</p>
      <p><strong>Proposed:</strong> {factValueText(conflict.proposed)}</p>
      <div className="profile-document-actions">
        <button type="button" disabled={!!busy} onClick={() => void facts.resolve(fact, 'current')}>Keep current</button>
        <button type="button" className="primary" disabled={!!busy} onClick={() => void facts.resolve(fact, 'proposed')}>Use proposed</button>
      </div>
    </div> : <>
      <p className="profile-fact-value">{factValueText(fact.value)}</p>
      {fact.state === 'proposed' || fact.state === 'conflict' ? <div className="profile-document-actions">
        <button type="button" className="primary" disabled={!!busy} onClick={() => void facts.accept(fact)}>Accept proposal</button>
        <button type="button" disabled={!!busy} onClick={() => void facts.reject(fact)}>Reject</button>
      </div> : null}
    </>}
    {fact.quote ? <blockquote>{fact.quote}</blockquote> : null}
    <p className="profile-fact-source">{fact.documentName || fact.origin}{fact.sourceReference ? ` · ${fact.sourceReference}` : ''}</p>
    {error ? <div className="banner error" role="alert">{error}{needsCompensation ? <button type="button" onClick={onEditCompensation}>Edit compensation</button> : null}</div> : null}
    {needsCompensation ? <p className="profile-hint">This amount needs its currency and period to be valid. Set a complete cash compensation in the Compensation section first, or correct it manually there.</p> : null}
  </article>;
}

