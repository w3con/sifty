import React, {useState} from 'react';
import type {FitView, FitLine} from './quickmatch';

type Props = {
  fit?: FitView | null;
  onAddSkill?: (name: string) => Promise<any>;
  onDismissSkill?: (name: string) => Promise<void>;
  onConfirmHave?: (text: string) => Promise<void>;
  onConfirmDoNotHave?: (text: string) => Promise<void>;
};

export function WhatTheyAsk({fit, onAddSkill, onDismissSkill, onConfirmHave, onConfirmDoNotHave}: Props) {
  const lines = allRequirementLines(fit);

  // Compute counts from the requirement lines themselves so they are non-zero even when
  // the matching pass has not run (youFit.counts is zero when matching is off).
  const lineCounts = countLineVerdicts(lines);
  const hasLines = lines.length > 0;

  const subtitle = hasLines
    ? `${lines.length} requirement${lines.length === 1 ? '' : 's'}`
    : 'No model extraction yet';

  return (
    <section className="what-ask">
      <div className="card-icon-head">
        <span className="card-icon-box" aria-hidden="true">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#f4f4f5" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="8.5"/>
            <circle cx="12" cy="12" r="4.5"/>
            <circle cx="12" cy="12" r="0.8" fill="#f4f4f5" stroke="none"/>
          </svg>
        </span>
        <div className="card-title-block">
          <span className="card-title">What they ask</span>
          <span className="card-subtitle">{subtitle}</span>
        </div>
        {hasLines && (lineCounts.match > 0 || lineCounts.no > 0 || lineCounts.unclear > 0) && (
          <div className="card-counts" aria-label="Requirement counts">
            {lineCounts.match > 0 && <span className="count-pill ok">✓ {lineCounts.match}</span>}
            {lineCounts.no > 0 && <span className="count-pill no">✕ {lineCounts.no}</span>}
            {lineCounts.unclear > 0 && <span className="count-pill ns">? {lineCounts.unclear}</span>}
          </div>
        )}
      </div>

      {lines.length === 0 ? (
        <p className="ask-not-found">
          {fit
            ? 'No requirements extracted yet — run an assessment to populate this card.'
            : 'Not assessed yet.'}
        </p>
      ) : (
        <>
          {renderGroup('Required', lines.filter(l => l.level === 'required'), onAddSkill, onDismissSkill, onConfirmHave, onConfirmDoNotHave)}
          {renderGroup('Nice to have', lines.filter(l => l.level !== 'required'), onAddSkill, onDismissSkill, onConfirmHave, onConfirmDoNotHave)}
        </>
      )}
    </section>
  );
}

function renderGroup(
  label: string,
  lines: FitLine[],
  onAddSkill?: (name: string) => Promise<any>,
  onDismissSkill?: (name: string) => Promise<void>,
  onConfirmHave?: (text: string) => Promise<void>,
  onConfirmDoNotHave?: (text: string) => Promise<void>,
) {
  if (lines.length === 0) return null;
  return (
    <div key={label}>
      <div className="ask-section-label">{label.toUpperCase()}</div>
      <ul className="ask-list" style={{marginTop: 8}}>
        {lines.map((l, i) => (
          <AskRow key={i} line={l} onAddSkill={onAddSkill} onDismissSkill={onDismissSkill}
            onConfirmHave={onConfirmHave} onConfirmDoNotHave={onConfirmDoNotHave}/>
        ))}
      </ul>
    </div>
  );
}

// countLineVerdicts tallies match/no/unclear verdicts from the requirement lines.
// Used to show pills without relying on youFit.counts (which is zero when matching is off).
function countLineVerdicts(lines: FitLine[]): {match: number; no: number; unclear: number} {
  let match = 0, no = 0, unclear = 0;
  for (const l of lines) {
    if (l.verdict === 'match' || l.verdict === 'partial') match++;
    else if (l.verdict === 'no') no++;
    else unclear++;
  }
  return {match, no, unclear};
}

// All requirement lines from all four result lists, required first then preferred.
function allRequirementLines(fit?: FitView | null): FitLine[] {
  if (!fit) return [];
  const {matches, mismatches, missing, toClarify} = fit.result.lists;
  const all = [...matches, ...mismatches, ...missing, ...toClarify].filter(l => l.origin === 'requirement');
  const rank = (l: FitLine) => (l.level === 'required' ? 0 : 1);
  return all.map((l, i) => ({l, i})).sort((a, b) => rank(a.l) - rank(b.l) || a.i - b.i).map(x => x.l);
}

function rowClass(line: FitLine): string {
  switch (line.verdict) {
    case 'match':
    case 'partial':
      return 'match';
    case 'no':
      return 'no';
    default:
      return 'unknown';
  }
}

function markSymbol(line: FitLine): string {
  switch (line.verdict) {
    case 'match':
    case 'partial':
      return '✓';
    case 'no':
      return '✕';
    default:
      return '?';
  }
}

function AskRow({
  line,
  onAddSkill,
  onDismissSkill,
  onConfirmHave,
  onConfirmDoNotHave,
}: {
  line: FitLine;
  onAddSkill?: (name: string) => Promise<any>;
  onDismissSkill?: (name: string) => Promise<void>;
  onConfirmHave?: (text: string) => Promise<void>;
  onConfirmDoNotHave?: (text: string) => Promise<void>;
}) {
  const [busy, setBusy] = useState<'add' | 'dismiss' | null>(null);
  const [showAlts, setShowAlts] = useState(false);
  // Server-detected alternatives (from the POST /api/profile/skills response).
  // The frontend no longer reads line.alternatives for showing buttons; it reads the server response.
  const [serverAlts, setServerAlts] = useState<string[] | null>(null);
  const cls = rowClass(line);
  const pendingCls = busy ? ' pending' : '';
  const isUnknown = cls === 'unknown';
  const subject = line.subject ?? '';
  const isSkill = line.kind === 'skill';

  // For skill rows: use the add/dismiss skill handlers (vocabulary-based).
  // For non-skill rows: use the confirmHave/confirmDoNotHave handlers (fact-based).
  // A button that cannot act is never shown; never show both sets for the same row.
  const canAdd = isUnknown && subject !== '' && (isSkill ? !!onAddSkill : !!onConfirmHave);
  const canDismiss = isUnknown && subject !== '' && (isSkill ? !!onDismissSkill : !!onConfirmDoNotHave);

  async function handleAdd() {
    if (busy || !subject) return;
    setBusy('add');
    try {
      if (isSkill && onAddSkill) {
        // The server detects "X or Y" / "X/Y" alternatives from the name itself and returns
        // action="offer_alternatives" with the parsed parts. The frontend no longer passes
        // line.alternatives to the server; all detection is server-side.
        const r = await onAddSkill(subject);
        if (r?.action === 'offer_alternatives' && r?.alternatives?.length) {
          setServerAlts(r.alternatives);
          setShowAlts(true);
          return;
        }
      } else if (!isSkill && onConfirmHave) {
        await onConfirmHave(subject);
      }
    } finally { setBusy(null); }
  }

  async function handleAddAlt(alt: string) {
    if (busy || !onAddSkill) return;
    setBusy('add');
    try { await onAddSkill(alt); } finally { setBusy(null); }
  }

  async function handleDismiss() {
    if (busy || !subject) return;
    setBusy('dismiss');
    try {
      if (isSkill && onDismissSkill) await onDismissSkill(subject);
      else if (!isSkill && onConfirmDoNotHave) await onConfirmDoNotHave(subject);
    } finally { setBusy(null); }
  }

  // Only the short profile value is shown inline; long quotes go in title on the row.
  const inlineEvidence = line.profileValue || undefined;
  const titleHint = !inlineEvidence && line.quote ? line.quote : undefined;

  return (
    <li className={`ask-row ${cls}${pendingCls}`} title={titleHint}>
      <span className="ask-mark" aria-hidden="true">{busy ? '…' : markSymbol(line)}</span>
      <span className="ask-text">{subject || line.criterion}</span>
      {inlineEvidence && <span className="ask-evidence">{inlineEvidence}</span>}
      {showAlts && serverAlts ? (
        <div className="ask-actions ask-alts">
          {serverAlts.map(alt => (
            <button
              key={alt}
              className="add"
              disabled={!!busy}
              onClick={() => handleAddAlt(alt)}
              title={`Add "${alt}" to your profile`}
              aria-label={`Add ${alt}`}
            >
              {busy === 'add' ? '…' : `+ ${alt}`}
            </button>
          ))}
        </div>
      ) : (canAdd || canDismiss) ? (
        <div className="ask-actions">
          {canAdd && (
            <button
              className="add"
              disabled={!!busy}
              onClick={handleAdd}
              title={isSkill ? 'Add this skill to your profile' : 'Confirm you have this'}
              aria-label="I have this"
            >
              {busy === 'add' ? '…' : '+'}
            </button>
          )}
          {canDismiss && (
            <button
              className="dismiss"
              disabled={!!busy}
              onClick={handleDismiss}
              title={isSkill ? 'Mark that you do not have this skill' : 'Confirm you do not have this'}
              aria-label="I do not have this"
            >
              {busy === 'dismiss' ? '…' : '−'}
            </button>
          )}
        </div>
      ) : null}
    </li>
  );
}
