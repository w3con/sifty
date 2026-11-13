// Owner-facing text for the structured Profile fields. The server validates the same typed
// contract it did before, but its messages name internal JSON paths (for example
// `geography.workAuthorizations[1]: country is required`). A section editor shows the section,
// a 1-based row number and the action instead. Anything not listed here passes through
// unchanged: this is a small presentation mapping, not a new error framework.
const RULES: {pattern: RegExp; text: (match: RegExpMatchArray) => string}[] = [
  {
    pattern: /^(?:geography: )?geography\.workAuthorizations\[(\d+)\]: country is required$/,
    text: (match) => `Work authorization ${Number(match[1]) + 1}: choose a country.`,
  },
  {
    pattern: /^(?:geography: )?geography\.workAuthorizations\[(\d+)\]\.status .* must be eligible, need_visa or empty$/,
    text: (match) => `Work authorization ${Number(match[1]) + 1}: choose Eligible to work, Need visa sponsorship or Not set.`,
  },
  {
    pattern: /^(?:geography: )?geography\.preferredLocations\[(\d+)\]: country is required$/,
    text: (match) => `Preferred work location ${Number(match[1]) + 1}: choose a country.`,
  },
  {
    pattern: /^(?:geography: )?geography\.currentLocation: country is required$/,
    text: () => 'Current location: choose a country.',
  },
  {
    pattern: /^(?:geography: )?geography\.sponsorship ".*" must be required or not_required$/,
    text: () => 'Sponsorship: choose Required or Not required.',
  },
  {
    pattern: /^(?:work: )?engagement\.types\[(\d+)\]: .* must be employment or contract .*$/,
    text: () => 'Contract type: choose Employment or Contract.',
  },
  {
    pattern: /^(?:work: )?hours\.mode ".*" must be full_time or part_time$/,
    text: () => 'Working time: choose Full-time or Part-time.',
  },
  {
    pattern: /^(?:work: )?hours\.weeklyMax must be greater than or equal to weeklyMin$/,
    text: () => 'Hours per week: the maximum must be at least the minimum.',
  },
  {
    pattern: /^(?:work: )?workModes\.modes\[(\d+)\]: .* must be remote, hybrid or office$/,
    text: () => 'Work mode: choose Remote, Hybrid or Office.',
  },
  {
    pattern: /^(?:compensation: )?compensation\.currency is required when a cash amount is provided$/,
    text: () => 'Compensation: choose a currency for the amount.',
  },
  {
    pattern: /^(?:compensation: )?compensation\.dayRates\.currency is required when a day rate is provided$/,
    text: () => 'Day rate: choose a currency.',
  },
  {
    pattern: /^(?:compensation: )?compensation\.cashTarget must be greater than or equal to cashMin$/,
    text: () => 'Compensation: the desired amount must be at least the minimum.',
  },
  {
    pattern: /^(?:compensation: )?compensation\.dayRates\.target must be greater than or equal to min$/,
    text: () => 'Day rate: the desired amount must be at least the minimum.',
  },
  {
    pattern: /^(?:compensation: )?compensation\.grossNet ".*" must be gross or net$/,
    text: () => 'Compensation: choose Before taxes or After taxes.',
  },
  {
    pattern: /^(?:compensation: )?compensation\.period ".*" is not a supported period$/,
    text: () => 'Compensation: choose Monthly or Yearly.',
  },
  {
    pattern: /^(?:compensation: )?compensation\.crypto ".*" must be fiat_only, crypto_accepted or crypto_only$/,
    text: () => 'Compensation: choose a payment method.',
  },
  {
    pattern: /^(?:compensation: )?compensation\.dayRates\.period must be "day"$/,
    text: () => 'Day rate: choose the day rate period.',
  },
  {
    pattern: /^(?:languages: )?languages\[(\d+)\]: name is required$/,
    text: (match) => `Language ${Number(match[1]) + 1}: choose a language.`,
  },
  {
    pattern: /^(?:availability: )?availability\.noticeUnit is required when noticeValue is provided$/,
    text: () => 'Notice period: choose days, weeks or months.',
  },
  {
    pattern: /^(?:availability: )?availability\.noticeUnit ".*" must be day, week or month$/,
    text: () => 'Notice period: choose days, weeks or months.',
  },
  {
    pattern: /^(?:availability: )?availability\.noticeValue must be positive when provided$/,
    text: () => 'Notice period: enter a number greater than zero.',
  },
];

export function humanizeSectionError(section: string, message: string): string {
  // Validation errors arrive without a section prefix; merge/decode errors can carry one.
  const bare = message.startsWith(`${section}: `) ? message.slice(section.length + 2) : message;
  for (const rule of RULES) {
    const match = bare.match(rule.pattern);
    if (match) return rule.text(match);
  }
  return message;
}
