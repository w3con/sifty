import React, {useRef, useState} from 'react';
import {
  ENGAGEMENT_LABELS,
  ENGAGEMENT_TYPES,
  GROSS_NET,
  GROSS_NET_LABELS,
  HOURS_LABELS,
  HOURS_MODES,
  PERIOD_LABELS,
  PERIODS,
  SPONSORSHIP,
  SPONSORSHIP_LABELS,
  WORK_MODE_LABELS,
  WORK_MODES,
  currentValue,
  draftFor,
  emptyExperience,
  emptySkill,
  factValueText,
  hasAny,
  listText,
  numberOrNull,
  numberText,
  provenanceText,
  textList,
  triStateText,
  triStateValue,
  type Availability,
  type Capabilities,
  type Compensation,
  type Experience,
  type Fact,
  type Geography,
  type Goals,
  type Identity,
  type Language,
  type Profile,
  type SectionKey,
  type Skill,
  type WorkSection,
} from './contract';
import {documentSummary} from './documentPresentation';
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

function MapTextField({label, value, onChange}: {label: string; value: string; onChange: (value: string) => void}) {
  return <Field label={label} hint="One value per line."><textarea rows={3} value={value} onChange={(event) => onChange(event.target.value)}/></Field>;
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
      <p className="profile-hint">Quick match uses target roles and skill names as search signals. A skill without a level is not production experience, and free-text dealbreakers are checked only in the detailed LLM assessment.</p>

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
          <SkillsSection profile={profile} capabilities={capabilities} editor={editor}/>
          <ExperienceSection profile={profile} editor={editor}/>
          <GoalsSection profile={profile} editor={editor}/>
          <CompensationSection profile={profile} editor={editor}/>
          <WorkSectionView profile={profile} editor={editor}/>
          <GeographySection profile={profile} editor={editor}/>
          <LanguagesSection profile={profile} editor={editor}/>
          <AvailabilitySection profile={profile} editor={editor}/>
          <DealbreakersSection profile={profile} editor={editor}/>
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
  return <SectionShell title="Identity" editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('identity')} onCancel={() => editor.cancelEdit('identity')} onSave={() => void editor.saveSection('identity')}>
    {draft ? <div className="profile-fields">
      <Field label="Display name"><input value={draft.value.displayName} onChange={(event) => editor.changeDraft('identity', {...draft.value, displayName: event.target.value})}/></Field>
      <Field label="Headline"><input value={draft.value.headline} onChange={(event) => editor.changeDraft('identity', {...draft.value, headline: event.target.value})}/></Field>
    </div> : <div className="profile-read">
      <p className="profile-lead">{profile.data.identity.displayName || <NotSet/>}</p>
      <p>{profile.data.identity.headline || <NotSet/>}</p>
    </div>}
  </SectionShell>;
}

function SkillsSection({profile, capabilities, editor}: {profile: Profile; capabilities: Capabilities | null; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'skills');
  const levels = capabilities?.skillLevels?.length ? capabilities.skillLevels : [];
  return <SectionShell title="Skills" description="A mentioned skill is not production experience." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('skills')} onCancel={() => editor.cancelEdit('skills')} onSave={() => void editor.saveSection('skills')}>
    {draft ? <div className="profile-rows">
      {draft.value.map((skill, index) => <div className="profile-row" key={`${skill.key}-${index}`}>
        <Field label="Name"><input value={skill.name} onChange={(event) => editor.changeDraft('skills', replaceAt(draft.value, index, {...skill, name: event.target.value}))}/></Field>
        <Field label="Level"><select value={skill.level} onChange={(event) => editor.changeDraft('skills', replaceAt(draft.value, index, {...skill, level: event.target.value}))}>
          <option value="">Not set</option>
          {levels.map((level) => <option key={level} value={level}>{level}</option>)}
        </select></Field>
        <Field label="Context"><input value={skill.context ?? ''} onChange={(event) => editor.changeDraft('skills', replaceAt(draft.value, index, {...skill, context: event.target.value}))}/></Field>
        <button type="button" onClick={() => editor.changeDraft('skills', draft.value.filter((_, current) => current !== index))}>Remove</button>
      </div>)}
      <button type="button" onClick={() => editor.changeDraft('skills', [...draft.value, emptySkill()])}>Add skill</button>
    </div> : profile.data.skills.length ? <ul className="profile-list">{profile.data.skills.map((skill) => <li key={skill.key}>
      <strong>{skill.name}</strong> <span className="profile-level">{skill.level || <NotSet/>}</span>
      {skill.context ? <p>{skill.context}</p> : null}<Provenance value={skill.provenance}/>
    </li>)}</ul> : <NotSet/>}
  </SectionShell>;
}

function ExperienceSection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'experience');
  return <SectionShell title="Experience" editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('experience')} onCancel={() => editor.cancelEdit('experience')} onSave={() => void editor.saveSection('experience')}>
    {draft ? <div className="profile-rows">
      {draft.value.map((row, index) => <div className="profile-row profile-row-wide" key={index}>
        <Field label="Company"><input value={row.company} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, company: event.target.value}))}/></Field>
        <Field label="Title"><input value={row.title} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, title: event.target.value}))}/></Field>
        <Field label="Start"><input value={row.start ?? ''} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, start: event.target.value}))}/></Field>
        <Field label="End"><input value={row.end ?? ''} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, end: event.target.value}))}/></Field>
        <Field label="Summary"><textarea rows={2} value={row.summary ?? ''} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, summary: event.target.value}))}/></Field>
        <Field label="Skills" hint="One value per line."><textarea rows={2} value={listText(row.skills)} onChange={(event) => editor.changeDraft('experience', replaceAt(draft.value, index, {...row, skills: textList(event.target.value)}))}/></Field>
        <button type="button" onClick={() => editor.changeDraft('experience', draft.value.filter((_, current) => current !== index))}>Remove</button>
      </div>)}
      <button type="button" onClick={() => editor.changeDraft('experience', [...draft.value, emptyExperience()])}>Add experience</button>
    </div> : profile.data.experience.length ? <ul className="profile-list">{profile.data.experience.map((row, index) => <li key={`${row.company}-${index}`}>
      <strong>{row.title || <NotSet/>}</strong> · {row.company || <NotSet/>}
      <p className="muted">{row.start || <NotSet/>} – {row.end || <NotSet/>}</p>
      {row.summary ? <p>{row.summary}</p> : null}<Tags items={row.skills}/>
    </li>)}</ul> : <NotSet/>}
  </SectionShell>;
}

function GoalsSection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'goals');
  const current: Goals = {targetRoles: profile.data.targetRoles, targetIndustries: profile.data.targetIndustries};
  return <SectionShell title="Target roles and industries" editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('goals')} onCancel={() => editor.cancelEdit('goals')} onSave={() => void editor.saveSection('goals')}>
    {draft ? <div className="profile-fields">
      <MapTextField label="Target roles" value={listText(draft.value.targetRoles)} onChange={(value) => editor.changeDraft('goals', {...draft.value, targetRoles: textList(value)})}/>
      <MapTextField label="Target industries" value={listText(draft.value.targetIndustries)} onChange={(value) => editor.changeDraft('goals', {...draft.value, targetIndustries: textList(value)})}/>
    </div> : <div className="profile-fields">
      <div><h3>Roles</h3><Tags items={current.targetRoles}/></div>
      <div><h3>Industries</h3><Tags items={current.targetIndustries}/></div>
    </div>}
  </SectionShell>;
}

function CompensationSection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'compensation');
  const value: Compensation = profile.data.compensation;
  return <SectionShell title="Compensation" description="A cash or day amount needs its currency. Gross/net, period and equity are never assumed." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('compensation')} onCancel={() => editor.cancelEdit('compensation')} onSave={() => void editor.saveSection('compensation')}>
    {draft ? <div className="profile-fields profile-fields-grid">
      <Field label="Cash minimum"><input type="number" min="0" value={numberText(draft.value.cashMin)} onChange={(event) => editor.changeDraft('compensation', {...draft.value, cashMin: numberOrNull(event.target.value)})}/></Field>
      <Field label="Cash target"><input type="number" min="0" value={numberText(draft.value.cashTarget)} onChange={(event) => editor.changeDraft('compensation', {...draft.value, cashTarget: numberOrNull(event.target.value)})}/></Field>
      <Field label="Currency"><input value={draft.value.currency} placeholder="Not set" onChange={(event) => editor.changeDraft('compensation', {...draft.value, currency: event.target.value})}/></Field>
      <Field label="Gross / net"><EnumSelect value={draft.value.grossNet} options={GROSS_NET} labels={GROSS_NET_LABELS} onChange={(value) => editor.changeDraft('compensation', {...draft.value, grossNet: value})}/></Field>
      <Field label="Cash period"><EnumSelect value={draft.value.period} options={PERIODS} labels={PERIOD_LABELS} onChange={(value) => editor.changeDraft('compensation', {...draft.value, period: value})}/></Field>
      <Field label="Day-rate minimum"><input type="number" min="0" value={numberText(draft.value.dayRates.min)} onChange={(event) => editor.changeDraft('compensation', {...draft.value, dayRates: {...draft.value.dayRates, min: numberOrNull(event.target.value)}})}/></Field>
      <Field label="Day-rate target"><input type="number" min="0" value={numberText(draft.value.dayRates.target)} onChange={(event) => editor.changeDraft('compensation', {...draft.value, dayRates: {...draft.value.dayRates, target: numberOrNull(event.target.value)}})}/></Field>
      <Field label="Day-rate currency"><input value={draft.value.dayRates.currency} placeholder="Not set" onChange={(event) => editor.changeDraft('compensation', {...draft.value, dayRates: {...draft.value.dayRates, currency: event.target.value}})}/></Field>
      <Field label="Day-rate period"><select value={draft.value.dayRates.period} onChange={(event) => editor.changeDraft('compensation', {...draft.value, dayRates: {...draft.value.dayRates, period: event.target.value}})}><option value="">Not set</option><option value="day">Per day</option></select></Field>
      <Field label="Equity / tokens on top"><TriState value={draft.value.equityOnTop} onChange={(value) => editor.changeDraft('compensation', {...draft.value, equityOnTop: value})}/></Field>
    </div> : <div className="profile-read">
      <p><strong>Cash:</strong> {value.cashMin === null && value.cashTarget === null ? <NotSet/> : <><Money amount={value.cashMin} currency={value.currency}/> → <Money amount={value.cashTarget} currency={value.currency}/></>}</p>
      <p><strong>Qualifiers:</strong> {[value.grossNet ? GROSS_NET_LABELS[value.grossNet] : '', value.period ? PERIOD_LABELS[value.period] : ''].filter(Boolean).join(' · ') || <NotSet/>}</p>
      <p><strong>Day rates:</strong> {value.dayRates.min === null && value.dayRates.target === null ? <NotSet/> : <><Money amount={value.dayRates.min} currency={value.dayRates.currency}/> → <Money amount={value.dayRates.target} currency={value.dayRates.currency}/></>}</p>
      <p><strong>Equity/tokens on top:</strong> {value.equityOnTop === null ? <NotSet/> : value.equityOnTop ? 'Yes' : 'No'}</p>
    </div>}
  </SectionShell>;
}

function WorkSectionView({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'work');
  const value: WorkSection = {engagement: profile.data.engagement, hours: profile.data.hours, workModes: profile.data.workModes};
  return <SectionShell title="Engagement, hours and work modes" editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('work')} onCancel={() => editor.cancelEdit('work')} onSave={() => void editor.saveSection('work')}>
    {draft ? <div className="profile-fields">
      <div><h3>Engagement</h3><CheckGroup values={draft.value.engagement.types} options={ENGAGEMENT_TYPES} labels={ENGAGEMENT_LABELS} onChange={(values) => editor.changeDraft('work', {...draft.value, engagement: {types: values}})}/></div>
      <div className="profile-fields-grid">
        <Field label="Hours"><EnumSelect value={draft.value.hours.mode} options={HOURS_MODES} labels={HOURS_LABELS} onChange={(value) => editor.changeDraft('work', {...draft.value, hours: {...draft.value.hours, mode: value}})}/></Field>
        <Field label="Weekly hours"><input type="number" min="0" value={numberText(draft.value.hours.weekly)} onChange={(event) => editor.changeDraft('work', {...draft.value, hours: {...draft.value.hours, weekly: numberOrNull(event.target.value)}})}/></Field>
      </div>
      <div><h3>Work modes</h3><CheckGroup values={draft.value.workModes.modes} options={WORK_MODES} labels={WORK_MODE_LABELS} onChange={(values) => editor.changeDraft('work', {...draft.value, workModes: {...draft.value.workModes, modes: values}})}/></div>
      <MapTextField label="Work limits" value={listText(draft.value.workModes.limits)} onChange={(value) => editor.changeDraft('work', {...draft.value, workModes: {...draft.value.workModes, limits: textList(value)}})}/>
    </div> : <div className="profile-read">
      <p><strong>Engagement:</strong> <Tags items={value.engagement.types} labels={ENGAGEMENT_LABELS}/></p>
      <p><strong>Hours:</strong> {value.hours.mode ? HOURS_LABELS[value.hours.mode] ?? value.hours.mode : <NotSet/>}{value.hours.weekly !== null ? ` · ${value.hours.weekly} per week` : ''}</p>
      <p><strong>Work modes:</strong> <Tags items={value.workModes.modes} labels={WORK_MODE_LABELS}/></p>
      <p><strong>Limits:</strong> <Tags items={value.workModes.limits}/></p>
    </div>}
  </SectionShell>;
}

function GeographySection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'geography');
  const value: Geography = profile.data.geography;
  return <SectionShell title="Geography and authorization" description="No stated eligibility is not ineligibility." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('geography')} onCancel={() => editor.cancelEdit('geography')} onSave={() => void editor.saveSection('geography')}>
    {draft ? <div className="profile-fields">
      <Field label="Current location"><input value={draft.value.current} placeholder="Not set" onChange={(event) => editor.changeDraft('geography', {...draft.value, current: event.target.value})}/></Field>
      <MapTextField label="Desired locations" value={listText(draft.value.desired)} onChange={(value) => editor.changeDraft('geography', {...draft.value, desired: textList(value)})}/>
      <MapTextField label="Work eligibility" value={listText(draft.value.eligibility)} onChange={(value) => editor.changeDraft('geography', {...draft.value, eligibility: textList(value)})}/>
      <MapTextField label="Would relocate to" value={listText(draft.value.relocation)} onChange={(value) => editor.changeDraft('geography', {...draft.value, relocation: textList(value)})}/>
      <MapTextField label="Timezones / overlap" value={listText(draft.value.timezones)} onChange={(value) => editor.changeDraft('geography', {...draft.value, timezones: textList(value)})}/>
      <div className="profile-fields-grid">
        <Field label="Willing to relocate"><TriState value={draft.value.willingToRelocate} onChange={(value) => editor.changeDraft('geography', {...draft.value, willingToRelocate: value})}/></Field>
        <Field label="Sponsorship"><EnumSelect value={draft.value.sponsorship} options={SPONSORSHIP} labels={SPONSORSHIP_LABELS} onChange={(value) => editor.changeDraft('geography', {...draft.value, sponsorship: value})}/></Field>
      </div>
    </div> : <div className="profile-read">
      <p><strong>Current:</strong> {value.current || <NotSet/>}</p>
      <p><strong>Desired:</strong> <Tags items={value.desired}/></p>
      <p><strong>Eligibility:</strong> <Tags items={value.eligibility}/></p>
      <p><strong>Relocation:</strong> <Tags items={value.relocation}/></p>
      <p><strong>Willing to relocate:</strong> {value.willingToRelocate === null ? <NotSet/> : value.willingToRelocate ? 'Yes' : 'No'}</p>
      <p><strong>Sponsorship:</strong> {value.sponsorship ? SPONSORSHIP_LABELS[value.sponsorship] ?? value.sponsorship : <NotSet/>}</p>
      <p><strong>Timezones:</strong> <Tags items={value.timezones}/></p>
    </div>}
  </SectionShell>;
}

function LanguagesSection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'languages');
  return <SectionShell title="Languages" editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('languages')} onCancel={() => editor.cancelEdit('languages')} onSave={() => void editor.saveSection('languages')}>
    {draft ? <div className="profile-rows">
      {draft.value.map((row, index) => <div className="profile-row" key={index}>
        <Field label="Language"><input value={row.name} onChange={(event) => editor.changeDraft('languages', replaceAt(draft.value, index, {...row, name: event.target.value}))}/></Field>
        <Field label="Level"><input value={row.level} placeholder="Not set" onChange={(event) => editor.changeDraft('languages', replaceAt(draft.value, index, {...row, level: event.target.value}))}/></Field>
        <button type="button" onClick={() => editor.changeDraft('languages', draft.value.filter((_, current) => current !== index))}>Remove</button>
      </div>)}
      <button type="button" onClick={() => editor.changeDraft('languages', [...draft.value, {name: '', level: ''}])}>Add language</button>
    </div> : profile.data.languages.length ? <ul className="profile-list">{profile.data.languages.map((row, index) => <li key={`${row.name}-${index}`}><strong>{row.name}</strong> <span className="profile-level">{row.level || <NotSet/>}</span></li>)}</ul> : <NotSet/>}
  </SectionShell>;
}

function AvailabilitySection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'availability');
  const value: Availability = profile.data.availability;
  return <SectionShell title="Availability" editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('availability')} onCancel={() => editor.cancelEdit('availability')} onSave={() => void editor.saveSection('availability')}>
    {draft ? <div className="profile-fields">
      <Field label="Notice period"><input value={draft.value.noticePeriod} placeholder="Not set" onChange={(event) => editor.changeDraft('availability', {...draft.value, noticePeriod: event.target.value})}/></Field>
      <Field label="Start date"><input value={draft.value.startDate} placeholder="Not set" onChange={(event) => editor.changeDraft('availability', {...draft.value, startDate: event.target.value})}/></Field>
      <Field label="Timezone"><input value={draft.value.timezone} placeholder="Not set" onChange={(event) => editor.changeDraft('availability', {...draft.value, timezone: event.target.value})}/></Field>
    </div> : <div className="profile-read">
      <p><strong>Notice period:</strong> {value.noticePeriod || <NotSet/>}</p>
      <p><strong>Start date:</strong> {value.startDate || <NotSet/>}</p>
      <p><strong>Timezone:</strong> {value.timezone || <NotSet/>}</p>
    </div>}
  </SectionShell>;
}

function DealbreakersSection({profile, editor}: {profile: Profile; editor: ProfileEditor}) {
  const draft = draftFor(editor.drafts, 'dealbreakers');
  return <SectionShell title="Dealbreakers" description="Free-form conditions. This screen stores them; it does not claim the fast ranking already interprets them." editing={!!draft} saving={draft?.saving ?? false} error={draft?.error ?? ''}
    onEdit={() => editor.beginEdit('dealbreakers')} onCancel={() => editor.cancelEdit('dealbreakers')} onSave={() => void editor.saveSection('dealbreakers')}>
    {draft ? <Field label="Dealbreakers" hint="One condition per line."><textarea rows={5} value={listText(draft.value)} onChange={(event) => editor.changeDraft('dealbreakers', textList(event.target.value))}/></Field>
      : hasAny(profile.data.dealbreakers) ? <Tags items={profile.data.dealbreakers}/> : <NotSet/>}
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

