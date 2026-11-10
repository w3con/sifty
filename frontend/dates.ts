// Source-date helpers shared by the catalogue listing, detail view and Newest
// ordering. Values are shown as unverified source dates: old rows carry no
// origin metadata, so the app must not assert them as verified publication.
const DATE_ONLY = /^(\d{4})-(\d{2})-(\d{2})$/;
const DATE_PREFIX = /^(\d{4})-(\d{2})-(\d{2})(?:[T ]|$)/;

function validCalendarParts(y: number, m: number, d: number): boolean {
  const probe = new Date(y, m - 1, d);
  return probe.getFullYear() === y && probe.getMonth() === m - 1 && probe.getDate() === d;
}

export function publicationDate(raw?: string | null): Date | null {
  const s = (raw ?? '').trim();
  if (!s) return null;
  const prefix = DATE_PREFIX.exec(s);
  if (prefix) {
    const y = Number(prefix[1]), m = Number(prefix[2]), d = Number(prefix[3]);
    // Reject impossible calendar dates such as 2026-02-31 instead of letting
    // Date.parse silently roll them into the next month.
    if (!validCalendarParts(y, m, d)) return null;
    if (DATE_ONLY.test(s)) {
      const only = new Date(y, m - 1, d);
      return only.getTime() > Date.now() ? null : only;
    }
  }
  let ms = Date.parse(s);
  if (Number.isNaN(ms)) ms = Date.parse(s.replace(/^(\d{4}-\d{2}-\d{2}) /, '$1T'));
  if (Number.isNaN(ms)) return null;
  const date = new Date(ms);
  if (Number.isNaN(date.getTime()) || date.getTime() > Date.now()) return null;
  return date;
}

export function publicationAge(raw?: string | null): { date: string; age: string } | null {
  const d = publicationDate(raw);
  if (!d) return null;
  const now = new Date();
  let years = now.getFullYear() - d.getFullYear();
  let months = now.getMonth() - d.getMonth();
  let days = now.getDate() - d.getDate();
  if (days < 0) {
    months -= 1;
    days += new Date(now.getFullYear(), now.getMonth(), 0).getDate();
  }
  if (months < 0) {
    years -= 1;
    months += 12;
  }
  // Completed calendar units only: 2022-04-25 on 2026-11-09 is 4y 6m, not 5y.
  let age: string;
  if (years > 0) {
    age = `${years} year${years === 1 ? '' : 's'}${months ? ` ${months} month${months === 1 ? '' : 's'}` : ''} old`;
  } else if (months > 0) {
    age = `${months} month${months === 1 ? '' : 's'} old`;
  } else if (days > 0) {
    age = `${days} day${days === 1 ? '' : 's'} old`;
  } else {
    age = 'today';
  }
  return { date: d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }), age };
}

export type PublicationInfo = { date: string; age: string; verified: boolean };

export function publicationInfo(publishedAt?: string | null, publishedBasis?: string | null): PublicationInfo | null {
  const p = publicationAge(publishedAt);
  if (!p) return null;
  return { ...p, verified: !!(publishedBasis ?? '').trim() };
}

export function publicationText(publishedAt?: string | null, publishedBasis?: string | null): string {
  const p = publicationInfo(publishedAt, publishedBasis);
  if (!p) return 'Publication date not reported';
  return p.verified ? `Published ${p.date} · ${p.age}` : `Source date ${p.date} · ${p.age} (unverified)`;
}
