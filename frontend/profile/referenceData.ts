// Presentation vocabulary for the profile editors. Values that the server already exposes through
// GET /api/profile/capabilities (optionSets) are read from there; this module holds the human
// labels and the local reference lists that are not a server contract. It contains no candidate
// values and no account/profile data.
//
// Countries and the timezone fallback come from the MIT-licensed `countries-and-timezones` package
// (IANA timezone data, ISO 3166-1 country names). The runtime `Intl.supportedValuesOf('timeZone')`
// list is preferred when the browser provides it; the local data is the compatibility fallback.
// The inputs never call a network geocoder.
import {getAllCountries, getAllTimezones} from 'countries-and-timezones';

export const SKILL_LEVEL_LABELS: Record<string, string> = {
  Beginner: 'Beginner',
  Proficient: 'Proficient',
  Expert: 'Expert',
};

export const GENDER_LABELS: Record<string, string> = {
  woman: 'Woman',
  man: 'Man',
  non_binary: 'Non-binary',
  self_describe: 'Self-describe',
  prefer_not_to_say: 'Prefer not to say',
};

export const GENDER_ORDER = ['woman', 'man', 'non_binary', 'self_describe', 'prefer_not_to_say'];

export const RACE_LABELS: Record<string, string> = {
  american_indian_or_alaska_native: 'American Indian or Alaska Native',
  asian: 'Asian',
  black_or_african_american: 'Black or African American',
  native_hawaiian_or_other_pacific_islander: 'Native Hawaiian or Other Pacific Islander',
  white: 'White',
  self_describe: 'Self-describe',
  prefer_not_to_say: 'Prefer not to say',
};

export const RACE_ORDER = [
  'american_indian_or_alaska_native',
  'asian',
  'black_or_african_american',
  'native_hawaiian_or_other_pacific_islander',
  'white',
  'self_describe',
  'prefer_not_to_say',
];

export const ETHNICITY_LABELS: Record<string, string> = {
  hispanic_or_latino: 'Hispanic or Latino',
  not_hispanic_or_latino: 'Not Hispanic or Latino',
  prefer_not_to_say: 'Prefer not to say',
};

export const ETHNICITY_ORDER = ['hispanic_or_latino', 'not_hispanic_or_latino', 'prefer_not_to_say'];

export const LANGUAGE_LEVEL_DESCRIPTIONS: Record<string, string> = {
  'Pre-A1': 'Novice',
  A1: 'Beginner',
  A2: 'Elementary',
  B1: 'Intermediate',
  B2: 'Upper-intermediate',
  C1: 'Advanced',
  C2: 'Proficient',
  Native: 'Native',
};

export const LANGUAGE_LEVEL_ORDER = ['Pre-A1', 'A1', 'A2', 'B1', 'B2', 'C1', 'C2', 'Native'];

export const NOTICE_UNIT_LABELS: Record<string, string> = {
  day: 'Days',
  week: 'Weeks',
  month: 'Months',
};

export const NOTICE_UNIT_ORDER = ['day', 'week', 'month'];

// Selectable language names. This is a local reference list: an older stored name that is not in
// the list is retained and shown, and the input still accepts a manually typed name.
export const LANGUAGE_NAMES = [
  'Arabic',
  'Bengali',
  'Bulgarian',
  'Catalan',
  'Chinese (Cantonese)',
  'Chinese (Mandarin)',
  'Croatian',
  'Czech',
  'Danish',
  'Dutch',
  'English',
  'Estonian',
  'Finnish',
  'French',
  'German',
  'Greek',
  'Hebrew',
  'Hindi',
  'Hungarian',
  'Icelandic',
  'Indonesian',
  'Italian',
  'Japanese',
  'Korean',
  'Latvian',
  'Lithuanian',
  'Malay',
  'Norwegian',
  'Persian',
  'Polish',
  'Portuguese',
  'Romanian',
  'Russian',
  'Serbian',
  'Slovak',
  'Slovenian',
  'Spanish',
  'Swahili',
  'Swedish',
  'Tagalog',
  'Tamil',
  'Thai',
  'Turkish',
  'Ukrainian',
  'Urdu',
  'Vietnamese',
  'Welsh',
];

export function labelFor(value: string, labels: Record<string, string>): string {
  return labels[value] ?? value;
}

// A stored level that predates the current options is never rewritten. It is shown as a distinct
// retained choice until the owner explicitly selects a current one.
export function retainedLabel(value: string, labels: Record<string, string>, note?: string): string {
  if (!value) return '';
  const label = labels[value];
  if (label && note) return `${label} — ${note}`;
  if (label) return label;
  return note ? `${value} — ${note}` : `${value} (retained value)`;
}

// Select options keep a stored value that is no longer in the current vocabulary. The option is
// visible and selectable, with a retained label, so saving another field never rewrites it to a
// new default.
export function optionsWithRetained(options: readonly string[], ...stored: Array<string | null | undefined>): string[] {
  const out = [...options];
  for (const raw of stored) {
    const value = (raw ?? '').trim();
    if (value && !out.includes(value)) out.push(value);
  }
  return out;
}

export function retainedOptionLabel(value: string, options: readonly string[], labels: Record<string, string>, note?: string): string {
  if (labels[value]) return options.includes(value) ? labels[value] : `${labels[value]} — retained value`;
  return note ? `${value} — ${note}` : `${value} (retained value)`;
}

// Currency values are the current EUR/USD choices. A legacy amount in another currency is shown
// as a retained value and is never converted or silently replaced.
export const CURRENCY_ORDER = ['EUR', 'USD'];
export const CURRENCY_LABELS: Record<string, string> = {
  EUR: 'EUR — Euro',
  USD: 'USD — US dollar',
};

// Crypto is a payment-method preference, not a currency. Crypto accepted is the no-restriction
// default meaning, not a requirement.
export const CRYPTO_ORDER = ['fiat_only', 'crypto_accepted', 'crypto_only'];
export const CRYPTO_LABELS: Record<string, string> = {
  fiat_only: 'Fiat only',
  crypto_accepted: 'Crypto accepted',
  crypto_only: 'Crypto only',
};

// Country-specific work authorization. Empty status is Not set, never ineligible.
export const AUTHORIZATION_STATUSES = ['eligible', 'need_visa'];
export const AUTHORIZATION_LABELS: Record<string, string> = {
  eligible: 'Eligible to work',
  need_visa: 'Need visa sponsorship',
};

export interface CountryOption {
  code: string;
  name: string;
}

const countryRecords = getAllCountries();
const countryNameByCode = new Map<string, string>(
  Object.values(countryRecords).map((country) => [country.id, country.name]),
);
export const COUNTRY_OPTIONS: CountryOption[] = [...countryNameByCode.entries()]
  .map(([code, name]) => ({code, name}))
  .sort((left, right) => left.name.localeCompare(right.name));

export function countryName(code: string): string {
  const value = (code ?? '').trim().toUpperCase();
  return countryNameByCode.get(value) ?? value;
}

export function countryOptionsWithRetained(codes: Array<string | null | undefined>): CountryOption[] {
  const seen = new Set(COUNTRY_OPTIONS.map((option) => option.code));
  const out = [...COUNTRY_OPTIONS];
  for (const raw of codes) {
    const code = (raw ?? '').trim().toUpperCase();
    if (code && !seen.has(code)) {
      seen.add(code);
      out.push({code, name: `${code} (retained value)`});
    }
  }
  return out;
}

// A selected location tag is "Paris, France" or "Canada — any city". The country name is local
// reference data; the city is the manually typed value and is never looked up online.
export function locationLabel(location: {country: string; city?: string}): string {
  const country = countryName(location.country);
  const city = (location.city ?? '').trim();
  if (!country) return city;
  return city ? `${city}, ${country}` : `${country} — any city`;
}

// Locale-independent human label for an IANA id: Europe/Paris -> Europe / Paris. The id itself
// remains the stored value; this is presentation only.
export function timezoneLabel(id: string): string {
  return id.split('/').map((part) => part.replace(/_/g, ' ')).join(' / ');
}

let timezoneCache: string[] | null = null;

// Prefer the runtime IANA list. Browsers/Nodes without Intl.supportedValuesOf fall back to the
// local IANA data from the reference package. Either source is complete enough to avoid a tiny
// handcrafted list; no continent is treated as one timezone.
export function timezoneOptions(): string[] {
  if (timezoneCache) return timezoneCache;
  const supported = (Intl as unknown as {supportedValuesOf?: (key: string) => string[]}).supportedValuesOf;
  if (typeof supported === 'function') {
    try {
      const values = supported.call(Intl, 'timeZone');
      if (Array.isArray(values) && values.length) {
        timezoneCache = [...values].sort();
      }
    } catch {
      // Fall through to the local data.
    }
  }
  if (!timezoneCache) {
    timezoneCache = Object.keys(getAllTimezones()).sort();
  }
  return timezoneCache;
}

export function timezoneOptionsWithRetained(selected: Array<string | null | undefined>): string[] {
  return optionsWithRetained(timezoneOptions(), ...selected);
}

// Broad work regions for "where would you work". A fixed list so matching can rely on it.
export const REGION_ORDER = ['europe', 'asia', 'southeast_asia', 'japan_oceania', 'north_america', 'south_america', 'middle_east_africa'];
export const REGION_LABELS: Record<string, string> = {
  europe: 'Europe',
  asia: 'Asia',
  southeast_asia: 'Southeast Asia',
  japan_oceania: 'Japan & Oceania',
  north_america: 'North America',
  south_america: 'South America',
  middle_east_africa: 'Middle East & Africa',
};
