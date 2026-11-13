// One presentation resource for the calendar range used by the profile date controls. This is
// a UI affordance, not a product setting or a new settings framework: both month/year and
// day/month/year pickers share the same bounds so their navigation cannot drift apart.

export const DATE_PICKER_BOUNDS = {
  earliestYear: 1950,
  latestYearOffset: 5,
} as const;

export function datePickerStart(now: Date = new Date()): Date {
  return new Date(DATE_PICKER_BOUNDS.earliestYear, 0);
}

export function datePickerEnd(now: Date = new Date()): Date {
  return new Date(now.getFullYear() + DATE_PICKER_BOUNDS.latestYearOffset, 11);
}
