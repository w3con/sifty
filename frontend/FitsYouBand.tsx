import React, {useState} from 'react';
import type {Job, BandTile} from './model/core';
import type {AnalysisStatus} from './quickmatch';
import {fitsYouCell, youFitCell, youFitOff} from './quickmatch';
import {api} from './api';

type Props = {
  job: Job;
  analysis?: AnalysisStatus | null;
  onRefreshSource: () => Promise<void>;
  onOpenProfile: (section?: string) => void;
};

export function FitsYouBand({job, analysis, onRefreshSource, onOpenProfile}: Props) {
  const [refreshing, setRefreshing] = useState(false);
  const [reassessing, setReassessing] = useState(false);

  async function doRefresh() {
    if (refreshing) return;
    setRefreshing(true);
    try { await onRefreshSource(); } finally { setRefreshing(false); }
  }

  async function doAssess() {
    if (reassessing) return;
    setReassessing(true);
    try { await api(`jobs/${job.id}/assess`, 'POST', {}); } catch (_) {} finally { setReassessing(false); }
  }

  const fit = job.fit;
  const fitsYou = fitsYouCell(fit);
  const youFit = youFitCell(fit, analysis);
  const off = youFitOff(fit);

  const tiles = job.band?.tiles ?? [];

  return (
    <section className="fits-you-band" aria-label="Fits you band">
      <div className="band-score">
        <span className="band-score-label">Fits you</span>
        <div className="band-score-number">
          <span>{fitsYou.number}</span>
          {fitsYou.number !== '–' && <small>/100</small>}
        </div>
        {fitsYou.label && (
          <span className={`qm-bucket ${fitsYou.bucket ?? ''}`}>{fitsYou.label}</span>
        )}
        {!off && youFit.number !== '–' && (
          <span className="band-score-label" style={{marginTop: 2}}>
            You fit {youFit.number}
          </span>
        )}
        <div className="band-score-actions">
          <button disabled={refreshing} onClick={doRefresh} title="Re-fetch the listing text">
            {refreshing ? 'Refreshing…' : 'Refresh'}
          </button>
          <button disabled={reassessing} onClick={doAssess} title="Queue an AI assessment for this role">
            {reassessing ? 'Queueing…' : 'Assess'}
          </button>
        </div>
      </div>
      <div className="band-tiles">
        {tiles.length > 0
          ? tiles.map(t => <Tile key={t.label} tile={t} />)
          : PLACEHOLDER_LABELS.map(label => <PlaceholderTile key={label} label={label} />)
        }
      </div>
    </section>
  );
}

const PLACEHOLDER_LABELS = ['Money', 'Location', 'Work mode', 'Contract'];

// Tile renders one goal tile with the design anatomy:
//   top row: category icon (left) + verdict circle (right, omitted when verdict is "")
//   value row: large value text or "Not stated"
//   hint row: description line
function Tile({tile}: {tile: BandTile}) {
  const verdictCircle = tile.verdict !== '' ? (
    <span
      className="band-tile-vc"
      aria-label={circleAriaLabel(tile.verdict)}
    >
      {circleSymbol(tile.verdict)}
    </span>
  ) : null;

  return (
    <div className={`band-tile ${tile.verdict}`} aria-label={tile.label}>
      <div className="band-tile-head">
        <TileIcon label={tile.label} verdict={tile.verdict} />
        {verdictCircle}
      </div>
      {tile.value
        ? <span className="band-tile-value">{tile.value}</span>
        : <span className="band-tile-value">Not stated</span>
      }
      {(tile.hint || tile.approxValue) && (
        <span className="band-tile-hint">
          {tile.hint}{tile.hint && tile.approxValue ? ' · ' : ''}{tile.approxValue}
        </span>
      )}
    </div>
  );
}

function PlaceholderTile({label}: {label: string}) {
  return (
    <div className="band-tile not_stated" aria-label={label}>
      <div className="band-tile-head">
        <TileIcon label={label} verdict="not_stated" />
        <span className="band-tile-vc" aria-label="unknown">?</span>
      </div>
      <span className="band-tile-value">Not stated</span>
    </div>
  );
}

function circleSymbol(verdict: string): string {
  switch (verdict) {
    case 'ok':       return '✓';
    case 'mismatch': return '✕';
    default:         return '?';
  }
}

function circleAriaLabel(verdict: string): string {
  switch (verdict) {
    case 'ok':       return 'matches';
    case 'mismatch': return 'does not match';
    default:         return 'unknown';
  }
}

// Icon stroke color follows verdict: green for ok, amber for mismatch, grey otherwise.
function tileStroke(verdict: string): string {
  switch (verdict) {
    case 'ok':       return '#1f5a17';
    case 'mismatch': return '#7a5c00';
    default:         return '#55555b';
  }
}

function TileIcon({label, verdict}: {label: string; verdict: string}) {
  const stroke = tileStroke(verdict);
  const props = {
    width: 28, height: 28, viewBox: '0 0 24 24', fill: 'none',
    stroke, strokeWidth: 1.8, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const,
    'aria-hidden': true,
  };
  switch (label) {
    case 'Money':
      return (
        <svg {...props}>
          <circle cx="12" cy="12" r="9"/>
          <path d="M14.8 9.2c-.5-.8-1.5-1.3-2.7-1.3-1.5 0-2.7.8-2.7 2s1.1 1.7 2.7 2 2.7.8 2.7 2-1.2 2-2.7 2c-1.2 0-2.2-.5-2.7-1.3M12 6.3v1.6M12 16.1v1.6"/>
        </svg>
      );
    case 'Location':
      return (
        <svg {...props}>
          <path d="M12 21s-6.5-5.6-6.5-11a6.5 6.5 0 0 1 13 0c0 5.4-6.5 11-6.5 11z"/>
          <circle cx="12" cy="10" r="2.3"/>
        </svg>
      );
    case 'Work mode':
      return (
        <svg {...props}>
          <path d="M4 11.5 12 5l8 6.5"/>
          <path d="M6 10v9h12v-9"/>
          <path d="M10 19v-5h4v5"/>
        </svg>
      );
    case 'Contract':
      return (
        <svg {...props}>
          <rect x="3.5" y="7.5" width="17" height="12" rx="2"/>
          <path d="M9 7.5V6a1.5 1.5 0 0 1 1.5-1.5h3A1.5 1.5 0 0 1 15 6v1.5M3.5 12.5h17"/>
        </svg>
      );
    default:
      return null;
  }
}
