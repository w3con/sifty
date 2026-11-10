import React, {useCallback, useEffect, useRef, useState} from 'react';

// Width of the vacancy list column, adjustable by dragging its right edge. Remembered per browser.
const KEY = 'sift.listWidth';
export const LIST_MIN = 260;
export const LIST_MAX = 720;
export const LIST_DEFAULT = 380;
const clamp = (value: number) => Math.round(Math.min(LIST_MAX, Math.max(LIST_MIN, value)));

export function useListWidth(): [number, (value: number) => void] {
  const [width, setWidth] = useState(() => {
    try { const stored = Number(localStorage.getItem(KEY)); return stored ? clamp(stored) : LIST_DEFAULT; } catch { return LIST_DEFAULT; }
  });
  const update = useCallback((value: number) => {
    const next = clamp(value);
    setWidth(next);
    try { localStorage.setItem(KEY, String(next)); } catch { /* storage unavailable */ }
  }, []);
  return [width, update];
}

// Drag handle on the list's right edge. Mouse/touch drag, arrow keys (Shift = larger steps),
// double-click resets to the default width.
export function ListResizer({width, onChange}: {width: number; onChange: (value: number) => void}) {
  const start = useRef<{x: number; width: number} | null>(null);
  const [dragging, setDragging] = useState(false);
  useEffect(() => {
    if (!dragging) return;
    const move = (event: PointerEvent) => { if (start.current) onChange(start.current.width + event.clientX - start.current.x); };
    const up = () => { start.current = null; setDragging(false); };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
    document.body.classList.add('resizing-list');
    return () => { window.removeEventListener('pointermove', move); window.removeEventListener('pointerup', up); document.body.classList.remove('resizing-list'); };
  }, [dragging, onChange]);
  return <div className={`list-resizer${dragging ? ' dragging' : ''}`} role="separator" aria-orientation="vertical" aria-label="Resize vacancy list"
    aria-valuemin={LIST_MIN} aria-valuemax={LIST_MAX} aria-valuenow={width} tabIndex={0} title="Drag to resize · double-click to reset"
    onPointerDown={(event) => { event.preventDefault(); start.current = {x: event.clientX, width}; setDragging(true); }}
    onDoubleClick={() => onChange(LIST_DEFAULT)}
    onKeyDown={(event) => {
      const step = event.shiftKey ? 48 : 16;
      if (event.key === 'ArrowLeft') { event.preventDefault(); onChange(width - step); }
      if (event.key === 'ArrowRight') { event.preventDefault(); onChange(width + step); }
    }}/>;
}
