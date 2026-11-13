// Message kinds for the striped message system (see the Sifty design system, "Messages").
// broken: will not pass on its own; something must be fixed (settings, a file, a login).
// retry:  did not work this time; trying again usually helps. Unknown errors default here.
export type MessageKind = 'broken' | 'retry';

const BROKEN = [
  /not logged in/i, /log ?in required/i, /unauthori[sz]ed/i, /forbidden/i, /permission denied/i,
  /not configured/i, /no (?:model|provider) (?:is )?(?:set|configured|selected)/i, /model .*not (?:found|available)/i,
  /could not be reached/i, /connection refused/i, /no such host/i,
  /original file is missing/i, /file (?:is )?missing/i, /nothing to read/i, /unsupported/i, /invalid (?:yaml|configuration|config)/i,
];

export const messageKind = (text: string | null | undefined): MessageKind =>
  text && BROKEN.some((rule) => rule.test(text)) ? 'broken' : 'retry';
