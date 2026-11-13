import React, {useEffect, useState} from 'react';
import {DayPicker} from 'react-day-picker';
import 'react-day-picker/style.css';
import {DATE_PICKER_BOUNDS, datePickerEnd, datePickerStart} from './datePickerConfig';

// Date controls use the ready-made React DayPicker. Experience stores only a month/year, so its
// calendar commits MM/YYYY and never invents or displays a stored day. Availability stores a
// DD/MM/YYYY date. Both keep a manual text input so an older unknown string stays visible until
// the owner explicitly replaces it. The shared calendar bounds live in datePickerConfig.ts.

function pad2(value: number): string {
  return String(value).padStart(2, '0');
}

export function formatMonthYear(date: Date): string {
  return `${pad2(date.getMonth() + 1)}/${date.getFullYear()}`;
}

export function formatDayMonthYear(date: Date): string {
  return `${pad2(date.getDate())}/${pad2(date.getMonth() + 1)}/${date.getFullYear()}`;
}

export function parseMonthYear(text: string): Date | null {
  const value = text.trim();
  let match = value.match(/^(\d{1,2})\/(\d{4})$/);
  if (match) {
    const month = Number(match[1]);
    const year = Number(match[2]);
    if (month >= 1 && month <= 12 && year >= DATE_PICKER_BOUNDS.earliestYear) return new Date(year, month - 1, 1);
  }
  match = value.match(/^(\d{4})-(\d{1,2})$/);
  if (match) {
    const year = Number(match[1]);
    const month = Number(match[2]);
    if (month >= 1 && month <= 12 && year >= DATE_PICKER_BOUNDS.earliestYear) return new Date(year, month - 1, 1);
  }
  return null;
}

export function parseDayMonthYear(text: string): Date | null {
  const value = text.trim();
  let match = value.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/);
  if (match) {
    const day = Number(match[1]);
    const month = Number(match[2]);
    const year = Number(match[3]);
    const date = new Date(year, month - 1, day);
    if (date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day) return date;
  }
  match = value.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/);
  if (match) {
    const year = Number(match[1]);
    const month = Number(match[2]);
    const day = Number(match[3]);
    const date = new Date(year, month - 1, day);
    if (date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day) return date;
  }
  return null;
}

function normalizeMonthYear(text: string): string {
  const date = parseMonthYear(text);
  return date ? formatMonthYear(date) : text;
}

function normalizeDayMonthYear(text: string): string {
  const date = parseDayMonthYear(text);
  return date ? formatDayMonthYear(date) : text;
}

function monthStart(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

export type MonthYearInputProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  hint?: string;
};

export function MonthYearInput({label, value, onChange, disabled, hint}: MonthYearInputProps) {
  const [open, setOpen] = useState(false);
  const [viewMonth, setViewMonth] = useState(() => monthStart(parseMonthYear(value) ?? new Date()));

  // A field can become disabled while its calendar is open (Currently working here). Close and
  // stop rendering it immediately, so Use/Clear/navigation cannot edit a disabled end date.
  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);

  function toggle() {
    if (disabled) return;
    if (!open) setViewMonth(monthStart(parseMonthYear(value) ?? new Date()));
    setOpen(!open);
  }

  return <div className="profile-field-block">
    <span>{label}</span>
    <div className="profile-date">
      <input aria-label={label} value={value} placeholder="MM/YYYY" inputMode="numeric" disabled={disabled}
        onChange={(event) => { if (!disabled) onChange(event.target.value); }}
        onBlur={() => { if (disabled) return; const next = normalizeMonthYear(value); if (next !== value) onChange(next); }}/>
      <button type="button" disabled={disabled} aria-expanded={open && !disabled} onClick={toggle}>{open && !disabled ? 'Close calendar' : 'Calendar'}</button>
    </div>
    {open && !disabled ? <div className="profile-day-picker profile-month-year-picker">
      <DayPicker
        month={viewMonth}
        onMonthChange={setViewMonth}
        captionLayout="dropdown"
        startMonth={datePickerStart()}
        endMonth={datePickerEnd()}
      />
      <div className="profile-picker-actions">
        <span>{formatMonthYear(viewMonth)}</span>
        <button type="button" className="primary" onClick={() => { if (disabled) return; onChange(formatMonthYear(viewMonth)); setOpen(false); }}>Use this month</button>
        <button type="button" onClick={() => { if (disabled) return; onChange(''); setOpen(false); }}>Clear</button>
      </div>
    </div> : null}
    {hint ? <small>{hint}</small> : null}
  </div>;
}

export type DateInputProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  hint?: string;
};

export function DateInput({label, value, onChange, disabled, hint}: DateInputProps) {
  const [open, setOpen] = useState(false);
  const selected = parseDayMonthYear(value);
  const [viewMonth, setViewMonth] = useState(() => monthStart(selected ?? new Date()));

  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);

  function toggle() {
    if (disabled) return;
    if (!open) setViewMonth(monthStart(parseDayMonthYear(value) ?? new Date()));
    setOpen(!open);
  }

  return <div className="profile-field-block">
    <span>{label}</span>
    <div className="profile-date">
      <input aria-label={label} value={value} placeholder="DD/MM/YYYY" inputMode="numeric" disabled={disabled}
        onChange={(event) => { if (!disabled) onChange(event.target.value); }}
        onBlur={() => { if (disabled) return; const next = normalizeDayMonthYear(value); if (next !== value) onChange(next); }}/>
      <button type="button" disabled={disabled} aria-expanded={open && !disabled} onClick={toggle}>{open && !disabled ? 'Close calendar' : 'Calendar'}</button>
    </div>
    {open && !disabled ? <div className="profile-day-picker">
      <DayPicker
        mode="single"
        selected={selected ?? undefined}
        month={viewMonth}
        onMonthChange={setViewMonth}
        onSelect={(date) => { if (disabled) return; if (date) { onChange(formatDayMonthYear(date)); setOpen(false); } }}
        captionLayout="dropdown"
        startMonth={datePickerStart()}
        endMonth={datePickerEnd()}
      />
      <div className="profile-picker-actions">
        <button type="button" onClick={() => { if (disabled) return; onChange(''); setOpen(false); }}>Clear</button>
      </div>
    </div> : null}
    {hint ? <small>{hint}</small> : null}
  </div>;
}
