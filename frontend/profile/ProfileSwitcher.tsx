import React, {useEffect, useRef, useState} from 'react';
import {getProfile} from './api';
import {Icon} from '../Icon';
import {PROFILE_COLORS, profileName} from './ProfileView';

// Header-level profile switcher. A profile is an app-level context: its own files, skills, roles,
// and therefore its own vacancies and scores. Today there is one stored profile; "Add new profile"
// stays disabled until the backend stores several.
//
// Until profiles are stored entities, a custom name is kept in this browser only (localStorage);
// without one, the profile is named after its first target role.
const NAME_KEY = 'sifty.profile.name';

function readName(): string {
  try { return localStorage.getItem(NAME_KEY) ?? ''; } catch { return ''; }
}
function writeName(value: string) {
  try { if (value) localStorage.setItem(NAME_KEY, value); else localStorage.removeItem(NAME_KEY); } catch { /* storage unavailable */ }
}
// Title Case for display: first letter of each word upper-cased, the rest kept (so "CTO" stays).
export function titleCase(value: string): string {
  return value.replace(/(^|[\s\-/(])(\p{L})/gu, (_, sep: string, letter: string) => sep + letter.toUpperCase());
}

export function ProfileSwitcher({open, onOpenProfile}: {open: boolean; onOpenProfile: () => void}) {
  const [menu, setMenu] = useState(false);
  const [derived, setDerived] = useState('Profile 1');
  const [custom, setCustom] = useState(readName);
  const [renaming, setRenaming] = useState(false);
  const [draft, setDraft] = useState('');
  const box = useRef<HTMLDivElement>(null);
  const name = titleCase(custom || derived);
  useEffect(() => {
    if (open) return;
    let alive = true;
    getProfile().then((profile) => { if (alive) setDerived(profileName(profile.data.targetRoles)); }).catch(() => {});
    return () => { alive = false; };
  }, [open]);
  useEffect(() => {
    if (!menu) { setRenaming(false); return; }
    const outside = (event: PointerEvent) => { if (box.current && !box.current.contains(event.target as Node)) setMenu(false); };
    const escape = (event: KeyboardEvent) => { if (event.key === 'Escape') setMenu(false); };
    document.addEventListener('pointerdown', outside);
    document.addEventListener('keydown', escape);
    return () => { document.removeEventListener('pointerdown', outside); document.removeEventListener('keydown', escape); };
  }, [menu]);
  const save = () => { const next = draft.trim(); writeName(next); setCustom(next); setRenaming(false); };
  return <div className="profile-switcher" ref={box}>
    <button type="button" className={`settings-button profile-button${open ? ' active' : ''}`} aria-haspopup="menu" aria-expanded={menu} title="Profiles"
      onClick={() => setMenu(!menu)}>
      <span className="profile-user-icon" style={{color: PROFILE_COLORS[0]}}><Icon name="user" size={16}/></span>
      <span className="profile-switcher-name">{name}</span>
      <span className="profile-switcher-caret" aria-hidden="true">▾</span>
    </button>
    {menu ? <div className="profile-menu" role="menu" aria-label="Profiles">
      <p className="profile-menu-title">Profiles</p>
      {renaming ? <form className="profile-menu-rename" onSubmit={(event) => { event.preventDefault(); save(); }}>
        <i className="profile-dot" style={{background: PROFILE_COLORS[0]}} aria-hidden="true"/>
        <input autoFocus aria-label="Profile name" value={draft} placeholder={titleCase(derived)} onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => { if (event.key === 'Escape') { event.stopPropagation(); setRenaming(false); } }}/>
        <button type="submit" className="primary">Save</button>
      </form> : <div className="profile-menu-row">
        <button type="button" role="menuitemradio" aria-checked="true" className="profile-menu-item current" title="Open profile" onClick={() => { setMenu(false); onOpenProfile(); }}>
          <i className="profile-dot" style={{background: PROFILE_COLORS[0]}} aria-hidden="true"/><span>{name}</span><span className="profile-menu-check" aria-hidden="true">✓</span>
        </button>
        <button type="button" className="profile-menu-rename-btn" aria-label={`Rename ${name}`} title="Rename" onClick={() => { setDraft(custom || titleCase(derived)); setRenaming(true); }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M17 3a2.83 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5z"/></svg>
        </button>
      </div>}
      {renaming ? <p className="profile-menu-hint">Leave empty to use your first target role.</p> : null}
      <button type="button" role="menuitem" className="profile-menu-item add" disabled title="Several profiles need backend support">+ Add new profile</button>
    </div> : null}
  </div>;
}
