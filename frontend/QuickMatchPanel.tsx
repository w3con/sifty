import React from 'react';
import type {MatchComparison, MatchConflict, MatchResult, MatchUnknown, QuickMatchView} from './quickmatch';
import {
  allGroups, applicabilityLabel, comparisonForUnknown, coveragePercent, criterionLabel,
  groupLabel, percent, profileAnchor, profileEditLabel, quickState, reasonText,
  reliabilityLabel, scorePresentation, sectionLabel, staleReasonText, unknownTitle,
} from './quickmatch';

type Props = {
  view?: QuickMatchView | null;
  error?: string;
  onOpenProfile: (section?: string) => void;
};

// Presentational only: receives the stored envelope and a named callback. It performs no request
// and computes no weight, utility, score or bucket of its own.
export function QuickMatchPanel({view, error, onOpenProfile}: Props) {
  const state = quickState(view, error);
  const score = scorePresentation(view, error);
  const coverage = coveragePercent(view);
  const result = view?.result;
  const contributions = new Map((result?.contributions ?? []).map(item => [item.group, item]));
  const unknowns = result?.unknowns ?? [];
  const conflicts = result?.conflicts ?? [];
  const comparisons = result?.comparisons ?? [];
  const reasons = result?.reasons ?? [];
  const stateLabel = state === 'scored' ? 'Current' : state === 'stale' ? 'Previous' : state === 'unscored' ? 'Not scored' : state === 'error' ? 'Unavailable' : 'Not calculated';
  return <section className="rail-card quick-match">
    <div className="qm-head"><h3>Quick match</h3><span className={`qm-status ${state}`}>{stateLabel}</span></div>
    <div className="qm-score"><b className={score.previous ? 'previous-score' : undefined}>{score.number}</b><small>/100</small><span className={`qm-bucket ${score.bucket}`}>{score.label}</span></div>
    <div className="qm-track"><span style={{width:`${state === 'scored' || state === 'unscored' ? Math.max(0, Math.min(100, view?.result.score ?? 0)) : score.previous ? Math.max(0, Math.min(100, view?.result.score ?? 0)) : 0}%`}}/></div>

    {state === 'error' && <p className="qm-error">Quick match could not be read: {error}. This is not the same as a zero or an unscored role.</p>}
    {state === 'missing' && <p>No Quick Match result has been calculated yet. The role is not hidden or ranked down for that.</p>}
    {state === 'stale' && <p className="qm-warn">Previous result, not current: {staleReasonText(view!)}. The number is kept only as history and is not presented as the current score.</p>}
    {state === 'unscored' && <p>No comparable evidence produced a score. This is not a zero; missing facts stay unknown instead of counting as mismatches.</p>}
    {state === 'scored' && <p>{score.label}: {reasons.length ? 'The reasons and unknowns below explain the number.' : 'No reason codes were returned for this result.'}</p>}

    {view && <>
      <p className={`qm-coverage ${state === 'stale' ? 'previous' : ''}`}>Evidence coverage <b>{coverage === null ? '—' : `${coverage}%`}</b>{state === 'stale' ? ' (previous)' : ''}</p>
      <div className="qm-track qm-coverage-track"><span style={{width:`${coverage ?? 0}%`}}/></div>
      {coverage !== null && coverage < 100 && <p className="qm-coverage-note">Partial evidence. Missing facts stay unknown and are not counted as a mismatch, so the score is less certain.</p>}
      {coverage === null && <p className="qm-coverage-note">Coverage was not reported by this result.</p>}

      <div className="qm-groups" aria-label="Fit groups">
        {allGroups.map(group => {
          const applicability = result?.applicability[group] ?? 'unknown';
          const groupCoverage = result?.coverage.groups[group];
          const contribution = contributions.get(group);
          return <div className={`qm-group ${applicability}`} key={group}>
            <div className="qm-group-head"><strong>{groupLabel(group)}</strong><span>{applicabilityLabel(applicability)}</span></div>
            {applicability === 'not_applicable'
              ? <p className="qm-group-note">No comparison is required for this vacancy and profile.</p>
              : applicability === 'unknown'
                ? <p className="qm-group-note">Not compared: applicability is unknown.</p>
                : <div className="qm-group-meta">
                    <span>Evidence {percent(groupCoverage)}</span>
                    {contribution
                      ? <><span>Weight {percent(contribution.weight)}</span><span>Utility {percent(contribution.utility)}</span><span>Weighted {percent(contribution.weighted)}</span></>
                      : <span>No scored contribution</span>}
                  </div>}
          </div>;
        })}
      </div>
      <p className="qm-note">Weight and weighted values come from the stored result. Weighted is utility × group weight before the final score divides by the effective evidence denominator, so these values do not necessarily sum to 100.</p>

      <details className="qm-details">
        <summary>Unknowns <span>{unknowns.length ?? 0}</span></summary>
        {unknowns.length ? <ul className="qm-unknowns">{unknowns.map((unknown, index) => <UnknownRow key={`${unknown.group}-${unknown.fact}-${unknown.kind}-${index}`} result={result!} unknown={unknown} onOpenProfile={onOpenProfile}/>)}</ul> : <p className="qm-note">{coverage !== null && coverage < 100 ? 'No individual unknown is listed, but evidence coverage is partial; the missing facts remain unknown.' : 'No unknown is listed for this result.'}</p>}
      </details>

      <details className="qm-details">
        <summary>Conflicts <span>{conflicts.length ?? 0}</span></summary>
        {conflicts.length ? <ul className="qm-conflicts">{conflicts.map((conflict, index) => <ConflictRow key={`${conflict.group}-${conflict.fact}-${index}`} conflict={conflict}/>)}</ul> : <p className="qm-note">No retained source conflict is listed.</p>}
      </details>

      <details className="qm-details">
        <summary>Criterion comparisons <span>{comparisons.length ?? 0}</span></summary>
        {comparisons.length ? allGroups.map(group => {
          const rows = comparisons.filter(item => item.group === group);
          if (!rows.length) return null;
          return <section className="qm-compare-group" key={group}><h4>{groupLabel(group)}</h4>{rows.map((row, index) => <ComparisonRow key={`${row.criterion}-${index}`} row={row}/>)}</section>;
        }) : <p className="qm-note">No criterion comparison was stored.</p>}
      </details>

      <details className="qm-details">
        <summary>Why this result <span>{reasons.length ?? 0}</span></summary>
        {reasons.length ? <ul className="qm-reasons">{reasons.map(code => <li key={code}>{reasonText(code)} <code>{code}</code></li>)}</ul> : <p className="qm-note">No reason code was stored.</p>}
      </details>

      <details className="qm-details">
        <summary>Result provenance</summary>
        <dl className="qm-provenance">
          <div><dt>Profile revision</dt><dd>{result?.provenance.profileRevision ?? '—'}</dd></div>
          <div><dt>Rules</dt><dd>{result?.provenance.rulesVersion || '—'}</dd></div>
          <div><dt>Facts schema</dt><dd>{result?.provenance.schemaVersion || '—'}</dd></div>
          <div><dt>Extractor</dt><dd>{result?.provenance.extractorVersion || '—'}</dd></div>
          <div><dt>Facts usable</dt><dd>{result?.reliability.usable ?? '—'}</dd></div>
          <div><dt>Needs corroboration</dt><dd>{result?.reliability.needsCorroboration ?? '—'}</dd></div>
          <div><dt>Unusable</dt><dd>{result?.reliability.unusable ?? '—'}</dd></div>
        </dl>
      </details>
    </>}
  </section>;
}

function UnknownRow({result, unknown, onOpenProfile}: {result: MatchResult; unknown: MatchUnknown; onOpenProfile: (section?: string) => void}) {
  const comparison = comparisonForUnknown(result, unknown);
  const section = profileAnchor(unknown.profileEdit);
  const label = profileEditLabel(unknown.profileEdit);
  return <li>
    <div className="qm-unknown-head"><strong>{unknownTitle(unknown)}</strong><span>{groupLabel(unknown.group)} · {reasonText(unknown.kind)}</span></div>
    {comparison?.jobValue && <p className="qm-unknown-value">Vacancy requirement: {comparison.jobValue}</p>}
    {(comparison?.evidenceQuote || unknown.quote) && <blockquote>{comparison?.evidenceQuote || unknown.quote}</blockquote>}
    <p className="qm-unknown-meta">{sectionLabel(unknown.section)}{unknown.profileEdit ? ` · profile field: ${unknown.profileEdit}` : ''}</p>
    {section && <button className="link-button qm-edit" onClick={() => onOpenProfile(section)}>Edit {label}</button>}
  </li>;
}

function ConflictRow({conflict}: {conflict: MatchConflict}) {
  return <li><div className="qm-unknown-head"><strong>{conflict.fact}</strong><span>{groupLabel(conflict.group)} · {reasonText(conflict.kind)}</span></div><p>{conflict.detail}</p></li>;
}

function ComparisonRow({row}: {row: MatchComparison}) {
  const profile = row.profileValue || (row.profileNumber !== undefined ? String(row.profileNumber) : '—');
  const job = row.jobValue || (row.jobNumber !== undefined ? String(row.jobNumber) : '—');
  const evidence = row.evidenceField || row.evidenceQuote;
  return <article className={`qm-compare ${row.comparable ? 'comparable' : 'unknown'}`}>
    <div className="qm-compare-head"><strong>{criterionLabel(row.criterion)}</strong><span>{applicabilityLabel(row.applicability)} · {reliabilityLabel(row.reliability)}</span></div>
    <dl className="qm-compare-values">
      <div><dt>Profile</dt><dd>{profile}</dd></div>
      <div><dt>Vacancy</dt><dd>{job}</dd></div>
    </dl>
    {evidence && <p className="qm-evidence">{row.evidenceField ? `Source field: ${row.evidenceField}` : 'Source quote'}{row.evidenceQuote ? ` · “${row.evidenceQuote}”` : ''}</p>}
    <p className="qm-compare-meta">{reasonText(row.reason)}{row.comparable && row.utility !== undefined ? ` · Utility ${percent(row.utility)}` : ' · Not comparable'}{(row.contribution ?? 0) > 0 ? ` · Share of group ${percent(row.contribution)}` : ''}</p>
  </article>;
}
