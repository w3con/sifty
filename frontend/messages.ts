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

// messageText returns the user-facing text for an error message string.
// All messages pass through unchanged; browser-level network errors are classified by
// their class (ServerUnreachableError) in the caller, not by message text.
export const messageText = (raw: string): string => raw;

// SERVER_DOWN_TEXT is the user-facing notice shown when Sifty's own server is unreachable.
export const SERVER_DOWN_TEXT = 'Sifty server is not responding. Retrying…';
