import React,{useEffect} from 'react';
import {ProfileView} from './ProfileView';
import {useProfileAvatar, useProfileDocuments, useProfileEditor, useProfileFacts} from './useProfile';

// Container: owns hook wiring only. The view renders state and callbacks and performs no requests.
export function ProfileScreen({onClose, onOpenSettings, initialSection}: {onClose: () => void; onOpenSettings: () => void; initialSection?: string}) {
  const editor = useProfileEditor();
  const documents = useProfileDocuments(editor.capabilities);
  const facts = useProfileFacts(editor.profile, documents.active, editor.applyExternalProfile);
  const avatar = useProfileAvatar(editor.capabilities);
  // A Quick Match unknown can open the existing profile at the section that owns the missing
  // fact. Scrolling is presentation only; the profile editor and its save path are unchanged.
  useEffect(() => {
    if (!initialSection || editor.loading || !editor.profile) return;
    document.getElementById(initialSection)?.scrollIntoView({block: 'start'});
  }, [initialSection, editor.loading, editor.profile]);
  return <ProfileView editor={editor} documents={documents} facts={facts} avatar={avatar} onClose={onClose} onOpenSettings={onOpenSettings}/>;
}
