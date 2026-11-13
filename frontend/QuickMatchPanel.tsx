import React, {useState} from 'react';
import type {AnalysisStatus, FitLine, FitView, ScoreCell} from './quickmatch';
import {clarifyLabel, countsText, fitsYouCell, lineText, missingSection, pendingNotice, quickAddSkillNames, requirementLines, requirementMark, youFitCell, youFitOff} from './quickmatch';

type Props = {
  fit?: FitView | null;
  analysis?: AnalysisStatus | null;
  onOpenProfile: (section?: string) => void;
  // Fetches the full vacancy text again; resolves when the stored result has been refreshed.
  onRefreshSource: () => Promise<void>;
  // Adds a skill to the profile without leaving the page and re-checks this vacancy.
  onAddSkill?: (name: string) => Promise<void>;
  // Records that the owner does not have a skill (kept in the profile's skills section, so it is
  // never offered again) and re-checks this vacancy.
  onDismissSkill?: (name: string) => Promise<void>;
};

// Presentational only: receives the stored two-score result and named callbacks. It performs no
// request of its own and computes no weight, utility, score, bucket or verdict.
export function QuickMatchPanel({fit, analysis, onOpenProfile, onRefreshSource, onAddSkill, onDismissSkill}: Props) {
  const unavailable = !!fit && fit.result.source === 'unavailable';
  return <section className="rail-card quick-match" aria-label="How this role fits">
    {unavailable
      ? <SourceUnavailable onRefreshSource={onRefreshSource}/>
      : fit ? <Scores fit={fit} analysis={analysis} onOpenProfile={onOpenProfile} onAddSkill={onAddSkill} onDismissSkill={onDismissSkill}/> : <NotCompared analysis={analysis} onOpenProfile={onOpenProfile}/>}
  </section>;
}

function NotCompared({analysis, onOpenProfile}: {analysis?: AnalysisStatus | null; onOpenProfile: (section?: string) => void}) {
  if (analysis?.state === 'no_profile') {
    return <>
      <h3>Fit</h3>
      <p className="fit-text">Roles are compared with your profile once it is saved.</p>
      <button className="link-button" onClick={() => onOpenProfile()}>Open profile</button>
    </>;
  }
  return <>
    <h3>Fit</h3>
    <p className="fit-text">Not compared yet.</p>
  </>;
}

function SourceUnavailable({onRefreshSource}: {onRefreshSource: () => Promise<void>}) {
  const [busy, setBusy] = useState(false);
  return <>
    <h3>Fit</h3>
    <p className="fit-text"><b>Full job description is unavailable</b></p>
    <p className="fit-text">Scores appear once the text of the role has been retrieved.</p>
    <button disabled={busy} onClick={async () => { setBusy(true); try { await onRefreshSource(); } finally { setBusy(false); } }}>{busy ? 'Refreshing…' : 'Refresh'}</button>
  </>;
}

function Scores({fit, analysis, onOpenProfile, onAddSkill, onDismissSkill}: {fit: FitView; analysis?: AnalysisStatus | null; onOpenProfile: (section?: string) => void; onAddSkill?: (name: string) => Promise<void>; onDismissSkill?: (name: string) => Promise<void>}) {
  const fitsYou = fitsYouCell(fit), youFit = youFitCell(fit, analysis);
  const {youFit: you} = fit.result;
  const off = youFitOff(fit);
  // While You fit is off the four lists carry the goal lines only; the requirements are one list.
  const goals = (lines: FitLine[]) => (off ? lines.filter(l => l.origin === 'goal') : lines);
  const lists = {matches: goals(fit.result.lists.matches), mismatches: goals(fit.result.lists.mismatches), missing: goals(fit.result.lists.missing), toClarify: fit.result.lists.toClarify};
  const pending = !off && you.status === 'pending';
  const notice = pending ? pendingNotice(analysis) : null;
  return <>
    <div className="fit-scores">
      <ScoreBlock label="Fits you" cell={fitsYou}/>
      {!off && <ScoreBlock label="You fit" cell={youFit} counts={you.status === 'scored' ? countsText(you.counts) : undefined}/>}
    </div>
    {fit.stale && <p className="msg info" role="status">Updating after a change; these numbers are from before it.</p>}
    {fit.result.fitsYou.status === 'unscored' && <p className="fit-text"><button className="link-button" onClick={() => onOpenProfile('profile-goals')}>Add what you are looking for</button> to see this score.</p>}
    {notice && notice.kind !== 'working' && notice.kind !== 'profile' && <p className={`msg ${notice.kind}`} role="status">{notice.text}</p>}
    {notice?.kind === 'profile' && <p className="fit-text">{notice.text} <button className="link-button" onClick={() => onOpenProfile()}>Open profile</button></p>}

    <Section title="Matches" lines={lists.matches} mark="✓" onOpenProfile={onOpenProfile}/>
    <Section title="Mismatches" lines={lists.mismatches} mark="✕" onOpenProfile={onOpenProfile} onAddSkill={onAddSkill} onDismissSkill={onDismissSkill}/>
    <Section title="Missing from your profile" lines={lists.missing} mark="?" onOpenProfile={onOpenProfile} onAddSkill={onAddSkill} onDismissSkill={onDismissSkill} offerAdd/>
    <ToClarify lines={lists.toClarify}/>
    {off && (requirementLines(fit).length > 0
      ? <Section title="Requirements" lines={requirementLines(fit)} mark={requirementMark} onOpenProfile={onOpenProfile} onAddSkill={onAddSkill} onDismissSkill={onDismissSkill}/>
      : <section className="fit-section"><h4>Requirements</h4><p className="fit-text">{you.status === 'no_requirements' ? 'No requirements stated' : 'Requirements appear once the role has been analysed.'}</p></section>)}
  </>;
}

function ScoreBlock({label, cell, counts}: {label: string; cell: ScoreCell; counts?: string}) {
  const hasNumber = /^\d+$/.test(cell.number);
  return <div className={`fit-score${cell.previous ? ' previous' : ''}`} title={cell.title}>
    <span className="fit-score-label">{label}</span>
    {hasNumber
      ? <><span className="fit-score-number">{cell.number}<small>/100</small></span><span className={`qm-bucket ${cell.bucket}`}>{cell.label}</span></>
      : <span className="fit-score-none">{cell.note}</span>}
    {counts && <span className="fit-counts" aria-label={counts}>{counts.split(' · ').map((part, i) => <React.Fragment key={part}>{i > 0 && ' · '}<span className="fit-count">{part}</span></React.Fragment>)}</span>}
  </div>;
}

// A list longer than this shows its first lines and a control for the rest, so a dense vacancy
// does not push the other lists out of sight.
const SHORT_LIST = 6;

function Section({title, lines, mark, onOpenProfile, onAddSkill, onDismissSkill, offerAdd}: {title: string; lines: FitLine[]; mark: string | ((line: FitLine) => string); onOpenProfile: (section?: string) => void; onAddSkill?: (name: string) => Promise<void>; onDismissSkill?: (name: string) => Promise<void>; offerAdd?: boolean}) {
  const [expanded, setExpanded] = useState(false);
  const [working, setWorking] = useState<string | null>(null);
  const [done, setDone] = useState<Record<string, 'added' | 'dismissed'>>({});
  async function act(name: string, kind: 'added' | 'dismissed') {
    const run = kind === 'added' ? onAddSkill : onDismissSkill;
    if (!run) return;
    setWorking(name);
    try { await run(name); setDone(d => ({...d, [name]: kind})); } finally { setWorking(null); }
  }
  if (!lines.length) return null;
  // Goal lines (what you are looking for) come first, then the lines about the requirements,
  // separated by a thin rule so that the requirement counts are not read against the goal lines.
  const ordered = [...lines.filter(l => l.origin === 'goal'), ...lines.filter(l => l.origin !== 'goal')];
  const shown = expanded ? ordered : ordered.slice(0, SHORT_LIST);
  return <section className="fit-section">
    <h4>{title} <span>{lines.length}</span></h4>
    <ul className="fit-lines">
      {shown.map((line, index) => {
        const text = lineText(line);
        const quickNames = onAddSkill && onDismissSkill ? quickAddSkillNames(line) : [];
        const section = offerAdd && quickNames.length === 0 ? missingSection(line) : undefined;
        const rule = index > 0 && line.origin !== 'goal' && shown[index - 1].origin === 'goal';
        return <li key={`${line.criterion ?? line.subject}-${index}`} className={rule ? 'fit-rule' : undefined} title={line.quote || undefined}>
          <span className="fit-mark" aria-hidden="true">{typeof mark === 'function' ? mark(line) : mark}</span>
          <span className="fit-line-text">
            <span>{text.main}</span>
            {text.alternatives && <span className="fit-alternatives"> ({text.alternatives})</span>}
            {text.partial && <span className="fit-tag">partial</span>}
            {text.preferred && <span className="fit-tag">preferred</span>}
            {text.note && <span className="fit-evidence" title={text.fullNote}> — {text.note}</span>}
            {quickNames.length > 0 && <span className="fit-add-skills">{quickNames.map(name => <span key={name} className="fit-add-skill">
              {quickNames.length > 1 && <span className="fit-add-name">{name}</span>}
              {done[name]
                ? <span className="fit-tag">{done[name] === 'added' ? 'added to skills' : 'marked as not yours'}</span>
                : <>
                  <button className="fit-skill-action add" disabled={working !== null} title={`Add this skill to your profile`} data-tip={`I have this: add ${name} to my skills`} aria-label={`Add ${name} to my skills`} onClick={() => act(name, 'added')}>{working === name ? '…' : '+'}</button>
                  <button className="fit-skill-action dismiss" disabled={working !== null} title={`Mark that you do not have this skill`} data-tip={`I do not have this: remember it in my profile and stop offering ${name}`} aria-label={`I do not have ${name}`} onClick={() => act(name, 'dismissed')}>−</button>
                </>}
            </span>)}</span>}
            {section && <> <button className="link-button" onClick={() => onOpenProfile(section.id)}>Add to {section.label}</button></>}
          </span>
        </li>;
      })}
    </ul>
    {lines.length > SHORT_LIST && <button className="link-button fit-more" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}>{expanded ? 'Show fewer' : `Show ${lines.length - SHORT_LIST} more`}</button>}
  </section>;
}

function ToClarify({lines}: {lines: FitLine[]}) {
  if (!lines.length) return null;
  return <section className="fit-section">
    <h4>To clarify <span>{lines.length}</span></h4>
    <ul className="fit-lines">
      {lines.map((line, index) => <li key={`${line.criterion}-${index}`}>
        <span className="fit-mark" aria-hidden="true">?</span>
        <span className="fit-line-text">{clarifyLabel(line)}</span>
      </li>)}
    </ul>
  </section>;
}

// The two numbers of a catalogue row: Fits you, then You fit. The title spells both out in words.
export function ScorePair({fit}: {fit?: FitView | null}) {
  const a = fitsYouCell(fit), b = youFitCell(fit);
  // No result stored yet (new vacancy not yet rescored) or matching off: show Fits you only.
  if (!fit || youFitOff(fit)) {
    return <span className="score-pair" role="group" aria-label={a.title} title={a.title}>
      <span className={`score fits-you${a.bucket === 'strong' ? ' strong' : ''}${a.previous ? ' previous' : ''}`}>{a.number}</span>
    </span>;
  }
  const cls = (cell: ScoreCell, kind: string) => `score ${kind}${cell.bucket === 'strong' ? ' strong' : ''}${cell.previous ? ' previous' : ''}`;
  return <span className="score-pair" role="group" aria-label={`${a.title}; ${b.title}`} title={`${a.title} · ${b.title}`}>
    <span className={cls(a, 'fits-you')}>{a.number}</span>
    <span className={cls(b, 'you-fit')}>{b.number}</span>
  </span>;
}
