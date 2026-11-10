// Pure controls shared by the current profile fields. They render values and call the supplied
// callback; they do not fetch, persist or know about the profile workflow. Country and timezone
// vocabularies come from the single reference-data module.
import React from 'react';
import type {Location, WorkAuthorization} from './contract';
import {
  AUTHORIZATION_LABELS,
  AUTHORIZATION_STATUSES,
  countryOptionsWithRetained,
  locationLabel,
  optionsWithRetained,
  retainedOptionLabel,
  timezoneLabel,
  timezoneOptionsWithRetained,
} from './referenceData';

export function RetainedSelect({
  label, value, options, labels, emptyLabel = 'Not set', allowEmpty = true, hint, onChange,
}: {
  label: string;
  value: string;
  options: readonly string[];
  labels: Record<string, string>;
  emptyLabel?: string;
  allowEmpty?: boolean;
  hint?: string;
  onChange: (value: string) => void;
}) {
  const values = optionsWithRetained(options, value);
  return <label className="profile-field-block">
    <span>{label}</span>
    <select value={value} onChange={(event) => onChange(event.target.value)}>
      {allowEmpty ? <option value="">{emptyLabel}</option> : null}
      {values.map((option) => <option key={option} value={option}>{retainedOptionLabel(option, options, labels)}</option>)}
    </select>
    {hint ? <small>{hint}</small> : null}
  </label>;
}

export function RetainedCheckGroup({
  values, options, labels, hint, onChange,
}: {
  values: string[];
  options: readonly string[];
  labels: Record<string, string>;
  hint?: string;
  onChange: (values: string[]) => void;
}) {
  const visible = optionsWithRetained(options, ...values);
  return <div>
    <div className="profile-checks">{visible.map((option) => <label key={option}>
      <input type="checkbox" checked={values.includes(option)} onChange={(event) => {
        onChange(event.target.checked
          ? [...values.filter((item) => item !== option), option]
          : values.filter((item) => item !== option));
      }}/>{retainedOptionLabel(option, options, labels)}
    </label>)}</div>
    {hint ? <p className="profile-hint">{hint}</p> : null}
  </div>;
}

export function CountrySelect({
  label = 'Country', value, hint, onChange,
}: {
  label?: string;
  value: string;
  hint?: string;
  onChange: (value: string) => void;
}) {
  const code = (value ?? '').trim().toUpperCase();
  const options = countryOptionsWithRetained([code]);
  return <label className="profile-field-block">
    <span>{label}</span>
    <select value={code} onChange={(event) => onChange(event.target.value.toUpperCase())}>
      <option value="">Not set</option>
      {options.map((option) => <option key={option.code} value={option.code}>{option.name}</option>)}
    </select>
    {hint ? <small>{hint}</small> : null}
  </label>;
}

export function LocationFields({
  value, cityHint, onChange,
}: {
  value: Location;
  cityHint?: string;
  onChange: (value: Location) => void;
}) {
  const country = value.country ?? '';
  return <div className="profile-fields-grid">
    <CountrySelect value={country} onChange={(next) => onChange({country: next, city: next ? (value.city ?? '') : ''})}/>
    <label className="profile-field-block">
      <span>City (optional)</span>
      <input
        value={value.city ?? ''}
        disabled={!country}
        placeholder={country ? 'Type a city' : 'Choose a country first'}
        onChange={(event) => onChange({country, city: event.target.value})}
      />
      {cityHint ? <small>{cityHint}</small> : null}
    </label>
  </div>;
}

export function LocationTags({locations}: {locations: Location[]}) {
  const visible = locations.filter((location) => (location.country ?? '').trim() !== '');
  if (!visible.length) return <span className="profile-not-set">Not set</span>;
  return <div className="profile-tags">{visible.map((location, index) => <span key={`${location.country}-${location.city ?? ''}-${index}`}>{locationLabel(location)}</span>)}</div>;
}

export function LocationsEditor({
  rows, onChange,
}: {
  rows: Location[];
  onChange: (rows: Location[]) => void;
}) {
  return <div className="profile-rows">
    {rows.map((row, index) => <div className="profile-row profile-row-location" key={index}>
      <LocationFields value={row} onChange={(next) => onChange(rows.map((item, current) => current === index ? next : item))}/>
      <button type="button" onClick={() => onChange(rows.filter((_, current) => current !== index))}>Remove</button>
    </div>)}
    <button type="button" onClick={() => onChange([...rows, {country: '', city: ''}])}>Add location</button>
  </div>;
}

export function AuthorizationEditor({
  rows, onChange,
}: {
  rows: WorkAuthorization[];
  onChange: (rows: WorkAuthorization[]) => void;
}) {
  return <div className="profile-rows">
    {rows.map((row, index) => <div className="profile-row profile-row-authorization" key={index}>
      <CountrySelect value={row.country ?? ''} onChange={(country) => onChange(rows.map((item, current) => current === index ? {...item, country} : item))}/>
      <RetainedSelect
        label="Status"
        value={row.status ?? ''}
        options={AUTHORIZATION_STATUSES}
        labels={AUTHORIZATION_LABELS}
        emptyLabel="Not set"
        onChange={(status) => onChange(rows.map((item, current) => current === index ? {...item, status} : item))}
      />
      <button type="button" onClick={() => onChange(rows.filter((_, current) => current !== index))}>Remove</button>
    </div>)}
    <button type="button" onClick={() => onChange([...rows, {country: '', status: ''}])}>Add country</button>
  </div>;
}

export function TimezonePicker({
  values, hint, onChange,
}: {
  values: string[];
  hint?: string;
  onChange: (values: string[]) => void;
}) {
  const selected = values.filter((value) => value.trim() !== '');
  const options = timezoneOptionsWithRetained(selected);
  const available = options.filter((option) => !selected.includes(option));
  return <div className="profile-fields">
    {selected.length ? <div className="profile-rows">{selected.map((id) => <div className="profile-row profile-row-timezone" key={id}>
      <span>{timezoneLabel(id)}</span>
      <code>{id}</code>
      <button type="button" onClick={() => onChange(selected.filter((item) => item !== id))}>Remove</button>
    </div>)}</div> : <p className="profile-hint">No timezones selected.</p>}
    <label className="profile-field-block">
      <span>Add timezone</span>
      <select value="" onChange={(event) => {
        const id = event.target.value;
        if (id) onChange([...selected, id]);
      }}>
        <option value="">Choose a timezone…</option>
        {available.map((id) => <option key={id} value={id}>{timezoneLabel(id)} — {id}</option>)}
      </select>
    </label>
    {hint ? <p className="profile-hint">{hint}</p> : null}
  </div>;
}
