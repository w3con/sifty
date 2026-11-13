import React from 'react';
import type {OfferItem} from './model/core';

type Props = {
  items?: OfferItem[] | null;
};

export function WhatTheyOffer({items}: Props) {
  const hasSection = items !== undefined && items !== null;

  return (
    <section className="what-offer">
      <div className="card-icon-head">
        <span className="card-icon-box" aria-hidden="true">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#f4f4f5" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <rect x="3.5" y="9" width="17" height="11" rx="1.5"/>
            <path d="M2.5 9h19M12 9v11M12 9c-1.5-3.5-5.5-4-5.5-1.5S10 9 12 9zM12 9c1.5-3.5 5.5-4 5.5-1.5S14 9 12 9z"/>
          </svg>
        </span>
        <div className="card-title-block">
          <span className="card-title">What they offer</span>
          <span className="card-subtitle">From the Compensation and Benefits sections</span>
        </div>
      </div>

      {!hasSection ? (
        <p className="offer-not-found">No Compensation or Benefits section recognised in this listing.</p>
      ) : items.length === 0 ? (
        <p className="offer-not-found">Section found but no specific items matched.</p>
      ) : (
        <div className="offer-tiles">
          {items.map(item => <OfferTile key={item.key} item={item} />)}
        </div>
      )}
    </section>
  );
}

function OfferTile({item}: {item: OfferItem}) {
  // Salary tile: big = short range (value), small = currency/period label.
  // All other tiles: big = item name (label), small = stated value or not-stated text.
  const isSalary = item.key === 'salary';
  const bigText = isSalary ? (item.value ?? '—') : item.label;
  const smallText = isSalary ? item.label : (item.value ?? 'Amount not stated');
  return (
    <div className="offer-tile">
      <OfferIcon itemKey={item.key} />
      <span className="offer-tile-value">{bigText}</span>
      <span className="offer-tile-desc">{smallText}</span>
    </div>
  );
}

function OfferIcon({itemKey}: {itemKey: string}) {
  const props = {
    width: 26, height: 26, viewBox: '0 0 24 24', fill: 'none',
    stroke: '#18181b', strokeWidth: 1.8,
    strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const,
    'aria-hidden': true,
  };
  switch (itemKey) {
    case 'salary':
      // Dollar circle — same icon as the money band tile.
      return (
        <svg {...props}>
          <circle cx="12" cy="12" r="9"/>
          <path d="M14.8 9.2c-.5-.8-1.5-1.3-2.7-1.3-1.5 0-2.7.8-2.7 2s1.1 1.7 2.7 2 2.7.8 2.7 2-1.2 2-2.7 2c-1.2 0-2.2-.5-2.7-1.3M12 6.3v1.6M12 16.1v1.6"/>
        </svg>
      );
    case 'equity':
      return (
        <svg {...props}>
          <path d="M4 18l5-6 4 3 7-9"/>
          <path d="M15 6h5v5"/>
        </svg>
      );
    case 'health':
      return (
        <svg {...props}>
          <path d="M12 20s-7-4.4-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 10c0 5.6-7 10-7 10z"/>
        </svg>
      );
    case 'timeOff':
      return (
        <svg {...props}>
          <rect x="4" y="5" width="16" height="16" rx="2"/>
          <path d="M4 10h16M9 3v4M15 3v4"/>
        </svg>
      );
    case 'retirement':
      return (
        <svg {...props}>
          <path d="M3 20h18M5 20V8l7-5 7 5v12"/>
        </svg>
      );
    case 'learning':
      return (
        <svg {...props}>
          <path d="M2 7l10-5 10 5-10 5-10-5z"/>
          <path d="M4 8.4v7L12 20l8-4.6v-7"/>
        </svg>
      );
    case 'equipment':
      return (
        <svg {...props}>
          <rect x="2" y="4" width="20" height="14" rx="2"/>
          <path d="M8 22h8M12 18v4"/>
        </svg>
      );
    case 'bonus':
      return (
        <svg {...props}>
          <circle cx="12" cy="12" r="9"/>
          <path d="M14.8 9.2c-.5-.8-1.5-1.3-2.7-1.3-1.5 0-2.7.8-2.7 2s1.1 1.7 2.7 2 2.7.8 2.7 2-1.2 2-2.7 2c-1.2 0-2.2-.5-2.7-1.3M12 6.3v1.6M12 16.1v1.6"/>
        </svg>
      );
    case 'flexible':
      return (
        <svg {...props}>
          <circle cx="12" cy="12" r="8.5"/>
          <path d="M12 7.5v5l3 2"/>
        </svg>
      );
    default:
      // Generic circle for unknown keys.
      return (
        <svg {...props}>
          <circle cx="12" cy="12" r="9"/>
          <path d="M12 8v5M12 16h.01"/>
        </svg>
      );
  }
}
