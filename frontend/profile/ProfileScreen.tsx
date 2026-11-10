import React from 'react';
import {ProfileView} from './ProfileView';
import {useProfileDocuments, useProfileEditor, useProfileFacts} from './useProfile';

// Container: owns hook wiring only. The view renders state and callbacks and performs no requests.
export function ProfileScreen({onClose, onOpenSettings}: {onClose: () => void; onOpenSettings: () => void}) {
  const editor = useProfileEditor();
  const documents = useProfileDocuments(editor.capabilities);
  const facts = useProfileFacts(editor.profile, documents.active, editor.applyExternalProfile);
  return <ProfileView editor={editor} documents={documents} facts={facts} onClose={onClose} onOpenSettings={onOpenSettings}/>;
}
