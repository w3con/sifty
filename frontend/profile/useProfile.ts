import {useCallback, useEffect, useRef, useState} from 'react';
import {ApiError} from '../api';
import {
  acceptProfileFact,
  analyzeDocument,
  cancelDocumentAnalysis,
  documentFileUrl,
  getDocumentText,
  getProfile,
  getProfileCapabilities,
  listProfileDocuments,
  listProfileFacts,
  putProfile,
  rejectProfileFact,
  resolveProfileFact,
  retryDocument,
  uploadProfileDocument,
  uploadProfileText,
} from './api';
import {
  cloneData,
  DOCUMENT_ACTIVE_STATES,
  mergeSection,
  sectionValues,
  type Capabilities,
  type ConflictPayload,
  type Drafts,
  type Fact,
  type FactConflict,
  type Profile,
  type ProfileData,
  type ProfileDocument,
  type SectionDraft,
  type SectionKey,
  type SectionValues,
} from './contract';

// Poll only while a document is visibly extracting or analyzing. This is an update interval for
// in-flight server work, not a configurable product limit.
const ACTIVE_POLL_MS = 2500;

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function conflictPayload(error: unknown): ConflictPayload | null {
  if (error instanceof ApiError && error.status === 409) return (error.data ?? {}) as ConflictPayload;
  return null;
}

function setDraftAt(drafts: Drafts, key: SectionKey, value: SectionDraft<any>) {
  (drafts as Record<SectionKey, SectionDraft<any>>)[key] = value;
}

export function useProfileEditor() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [drafts, setDrafts] = useState<Drafts>({});
  const [revisionConflict, setRevisionConflict] = useState<number | null>(null);
  const [latestProfile, setLatestProfile] = useState<Profile | null>(null);
  const profileRef = useRef<Profile | null>(null);
  const latestProfileRef = useRef<Profile | null>(null);

  useEffect(() => {
    profileRef.current = profile;
  }, [profile]);

  useEffect(() => {
    latestProfileRef.current = latestProfile;
  }, [latestProfile]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [nextProfile, nextCapabilities] = await Promise.all([getProfile(), getProfileCapabilities()]);
      setProfile(nextProfile);
      profileRef.current = nextProfile;
      setLatestProfile(nextProfile);
      latestProfileRef.current = nextProfile;
      setCapabilities(nextCapabilities);
    } catch (cause) {
      setError(messageOf(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  function patchDraft<K extends SectionKey>(key: K, patch: Partial<SectionDraft<SectionValues[K]>>) {
    setDrafts((current) => {
      const existing = current[key];
      if (!existing) return current;
      return {...current, [key]: {...existing, ...patch}} as Drafts;
    });
  }

  const beginEdit = useCallback((key: SectionKey) => {
    const current = profileRef.current;
    if (!current) return;
    setDrafts((drafts) => {
      if (drafts[key]) return drafts;
      return {
        ...drafts,
        [key]: {
          value: sectionValues(current.data, key),
          baseRevision: current.revision,
          dirty: false,
          stale: false,
          saving: false,
          error: '',
        },
      } as Drafts;
    });
  }, []);

  const cancelEdit = useCallback((key: SectionKey) => {
    setDrafts((drafts) => {
      const next = {...drafts};
      delete next[key];
      return next;
    });
  }, []);

  const changeDraft = useCallback(<K extends SectionKey>(key: K, value: SectionValues[K]) => {
    setDrafts((drafts) => {
      const existing = drafts[key];
      if (!existing) return drafts;
      return {...drafts, [key]: {...existing, value, dirty: true, error: ''}} as Drafts;
    });
  }, []);

  // A save sends the revision captured when the section draft began. The server is the authority:
  // a stale revision returns 409 and the draft is kept, never replaced by another editor's values.
  const saveSection = useCallback(async (key: SectionKey): Promise<boolean> => {
    const current = profileRef.current;
    const draft = drafts[key];
    if (!current || !draft) return false;
    if (draft.stale) {
      patchDraft(key, {error: 'This draft is based on an older profile revision. Reload latest before saving.'});
      return false;
    }
    patchDraft(key, {saving: true, error: ''});
    setNotice('');
    try {
      const merged: ProfileData = mergeSection(current.data, key, draft.value as never);
      const saved = await putProfile(draft.baseRevision, merged);
      setProfile(saved);
      profileRef.current = saved;
      setLatestProfile(saved);
      latestProfileRef.current = saved;
      setRevisionConflict(null);
      setNotice(`Saved ${key} at profile revision ${saved.revision}.`);
      setDrafts((existing) => {
        const next: Drafts = {...existing};
        delete next[key];
        for (const other of Object.keys(next) as SectionKey[]) {
          const otherDraft = next[other];
          if (otherDraft && otherDraft.baseRevision !== saved.revision) {
            setDraftAt(next, other, {...otherDraft, stale: true, saving: false});
          }
        }
        return next;
      });
      return true;
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 409) {
        const data = cause.data as ConflictPayload;
        setRevisionConflict(data.currentRevision ?? null);
        if (data.current) {
          // Keep the newest stored profile available for side-by-side section comparison.
          setProfile(data.current);
          profileRef.current = data.current;
          setLatestProfile(data.current);
          latestProfileRef.current = data.current;
        }
        patchDraft(key, {
          saving: false,
          stale: true,
          error: `Profile changed to revision ${data.currentRevision ?? 'unknown'}. Your draft is kept; compare this section and choose rebase or discard.`,
        });
        setDrafts((existing) => {
          const next: Drafts = {...existing};
          for (const other of Object.keys(next) as SectionKey[]) {
            const otherDraft = next[other];
            if (otherDraft && otherDraft.baseRevision !== (data.currentRevision ?? -1)) {
              setDraftAt(next, other, {...otherDraft, stale: true});
            }
          }
          return next;
        });
      } else {
        patchDraft(key, {saving: false, error: messageOf(cause)});
      }
      return false;
    }
  }, [drafts]);

  // Fetching the latest profile never rebases a draft by itself. Each stale section must be
  // compared and then explicitly rebased or discarded, so a same-section external edit cannot be
  // overwritten by an ambiguous global refresh.
  const reloadLatest = useCallback(async () => {
    setError('');
    setNotice('');
    try {
      const next = await getProfile();
      setProfile(next);
      profileRef.current = next;
      setLatestProfile(next);
      latestProfileRef.current = next;
      setRevisionConflict(null);
      setDrafts((existing) => {
        const updated: Drafts = {};
        for (const key of Object.keys(existing) as SectionKey[]) {
          const draft = existing[key];
          if (draft) {
            setDraftAt(updated, key, draft.baseRevision === next.revision
              ? draft
              : {...draft, stale: true});
          }
        }
        return updated;
      });
      setNotice(`Loaded profile revision ${next.revision}. Section drafts were kept; compare each stale section before saving.`);
    } catch (cause) {
      setError(messageOf(cause));
    }
  }, []);

  // Explicit per-section choice after comparison. Rebasing keeps the draft values; the next Save
  // deliberately replaces this section on the newest revision.
  const rebaseDraft = useCallback((key: SectionKey) => {
    const latest = latestProfileRef.current ?? profileRef.current;
    if (!latest) return;
    setDrafts((existing) => {
      const draft = existing[key];
      if (!draft) return existing;
      const next: Drafts = {...existing};
      setDraftAt(next, key, {...draft, baseRevision: latest.revision, stale: false, error: ''});
      return next;
    });
  }, []);

  // The other explicit choice: drop the draft and continue from the latest stored section.
  const discardDraft = useCallback((key: SectionKey) => {
    const latest = latestProfileRef.current;
    if (latest && latest.revision !== profileRef.current?.revision) {
      setProfile(latest);
      profileRef.current = latest;
    }
    setDrafts((existing) => {
      const next = {...existing};
      delete next[key];
      return next;
    });
  }, []);

  // A profile-changing explicit action outside a section editor (for example accepting a fact)
  // makes every open section draft stale until the user reloads the latest revision.
  const applyExternalProfile = useCallback((next: Profile) => {
    setProfile(next);
    profileRef.current = next;
    setLatestProfile(next);
    latestProfileRef.current = next;
    setDrafts((existing) => {
      const updated: Drafts = {};
      for (const key of Object.keys(existing) as SectionKey[]) {
        const draft = existing[key];
        if (draft) {
          setDraftAt(updated, key, draft.baseRevision === next.revision
            ? draft
            : {...draft, stale: true, error: 'The profile changed after this draft began. Compare this section and choose rebase or discard.'});
        }
      }
      return updated;
    });
  }, []);

  return {
    profile, latestProfile, capabilities, loading, error, notice, drafts, revisionConflict,
    load, reloadLatest, beginEdit, cancelEdit, changeDraft, saveSection, applyExternalProfile,
    rebaseDraft, discardDraft, setError, setNotice,
  };
}

export type ProfileEditor = ReturnType<typeof useProfileEditor>;

function readBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(new Error('Could not read the selected file.'));
    reader.onload = () => {
      const result = String(reader.result ?? '');
      const comma = result.indexOf(',');
      resolve(comma >= 0 ? result.slice(comma + 1) : result);
    };
    reader.readAsDataURL(file);
  });
}

function fileExtension(name: string): string {
  const index = name.lastIndexOf('.');
  return index >= 0 ? name.slice(index + 1).toLowerCase() : '';
}

export function useProfileDocuments(capabilities: Capabilities | null) {
  const [documents, setDocuments] = useState<ProfileDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState('');
  const [texts, setTexts] = useState<Record<string, {loading: boolean; text: string; error: string}>>({});

  const refresh = useCallback(async () => {
    try {
      const list = await listProfileDocuments();
      setDocuments(list);
      setError('');
    } catch (cause) {
      setError(messageOf(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const active = documents.some((document) => (DOCUMENT_ACTIVE_STATES as readonly string[]).includes(document.processingState));
  useEffect(() => {
    if (!active) return;
    const timer = setInterval(() => void refresh(), ACTIVE_POLL_MS);
    return () => clearInterval(timer);
  }, [active, refresh]);

  function validate(name: string, bytes: number): string {
    if (!capabilities) return 'Document capabilities have not loaded yet.';
    const extension = fileExtension(name);
    if (!extension || !capabilities.documentTypes.includes(extension)) {
      return `This file type is not accepted. Configured types: ${capabilities.documentTypes.join(', ')}.`;
    }
    if (bytes > capabilities.maxDocumentBytes) {
      return `This document is too large. The configured limit is ${capabilities.maxDocumentBytes} bytes.`;
    }
    return '';
  }

  const uploadFile = useCallback(async (file: File): Promise<boolean> => {
    setBusy('upload');
    setError('');
    setNotice('');
    try {
      const invalid = validate(file.name, file.size);
      if (invalid) {
        setError(invalid);
        return false;
      }
      const encoded = await readBase64(file);
      const uploaded = await uploadProfileDocument(file.name, encoded);
      setDocuments((existing) => [uploaded, ...existing.filter((document) => document.documentId !== uploaded.documentId)]);
      setNotice(`Uploaded ${uploaded.originalName}. Review its status; analysis is explicit.`);
      return true;
    } catch (cause) {
      setError(messageOf(cause));
      return false;
    } finally {
      setBusy('');
    }
  }, [capabilities]);

  const uploadText = useCallback(async (filename: string, text: string): Promise<boolean> => {
    setBusy('upload');
    setError('');
    setNotice('');
    try {
      const name = filename.trim() || 'profile.txt';
      const withExtension = fileExtension(name) ? name : `${name}.txt`;
      const invalid = validate(withExtension, new Blob([text]).size);
      if (invalid) {
        setError(invalid);
        return false;
      }
      if (!text.trim()) {
        setError('Text is empty.');
        return false;
      }
      const uploaded = await uploadProfileText(withExtension, text);
      setDocuments((existing) => [uploaded, ...existing.filter((document) => document.documentId !== uploaded.documentId)]);
      setNotice(`Uploaded ${uploaded.originalName}. Review its status; analysis is explicit.`);
      return true;
    } catch (cause) {
      setError(messageOf(cause));
      return false;
    } finally {
      setBusy('');
    }
  }, [capabilities]);

  const replaceDocument = (updated: ProfileDocument) => {
    setDocuments((existing) => existing.map((document) => document.documentId === updated.documentId ? updated : document));
  };

  async function act(action: 'analyze' | 'cancel' | 'retry', id: string) {
    setBusy(id);
    setError('');
    setNotice('');
    try {
      const updated = action === 'analyze'
        ? await analyzeDocument(id)
        : action === 'cancel'
          ? await cancelDocumentAnalysis(id)
          : await retryDocument(id);
      replaceDocument(updated);
      setNotice(action === 'analyze'
        ? `Analysis started for ${updated.originalName}.`
        : action === 'cancel'
          ? `Analysis cancelled for ${updated.originalName}.`
          : `Retry started for ${updated.originalName}.`);
      return true;
    } catch (cause) {
      setError(messageOf(cause));
      return false;
    } finally {
      setBusy('');
    }
  }

  const loadText = useCallback(async (id: string) => {
    setTexts((current) => ({...current, [id]: {loading: true, text: current[id]?.text ?? '', error: ''}}));
    try {
      const result = await getDocumentText(id);
      setTexts((current) => ({...current, [id]: {loading: false, text: result.text, error: ''}}));
    } catch (cause) {
      setTexts((current) => ({...current, [id]: {loading: false, text: '', error: messageOf(cause)}}));
    }
  }, []);

  return {
    documents, loading, error, notice, busy, active, texts,
    refresh, uploadFile, uploadText, analyze: (id: string) => act('analyze', id),
    cancel: (id: string) => act('cancel', id), retry: (id: string) => act('retry', id), loadText,
    fileUrl: documentFileUrl, setError, setNotice,
  };
}

export type ProfileDocuments = ReturnType<typeof useProfileDocuments>;

export function useProfileFacts(profile: Profile | null, documentsActive: boolean, onProfileChanged: (next: Profile) => void) {
  const [facts, setFacts] = useState<Fact[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [actionError, setActionError] = useState<Record<string, string>>({});
  const [conflicts, setConflicts] = useState<Record<string, FactConflict>>({});
  const [busy, setBusy] = useState<Record<string, string>>({});
  const [revisionConflict, setRevisionConflict] = useState<number | null>(null);
  const wasActive = useRef(false);

  const refresh = useCallback(async () => {
    try {
      setFacts(await listProfileFacts());
      setError('');
    } catch (cause) {
      setError(messageOf(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh, profile?.revision]);

  useEffect(() => {
    if (wasActive.current && !documentsActive) void refresh();
    wasActive.current = documentsActive;
  }, [documentsActive, refresh]);

  useEffect(() => {
    if (!documentsActive) return;
    const timer = setInterval(() => void refresh(), ACTIVE_POLL_MS);
    return () => clearInterval(timer);
  }, [documentsActive, refresh]);

  function replaceFact(updated: Fact) {
    setFacts((existing) => existing.map((fact) => fact.factId === updated.factId ? updated : fact));
  }

  function setAction(id: string, value: string) {
    setBusy((current) => {
      const next = {...current};
      if (value) next[id] = value; else delete next[id];
      return next;
    });
  }

  function handleError(id: string, cause: unknown) {
    const payload = conflictPayload(cause);
    if (payload?.conflict) setConflicts((current) => ({...current, [id]: payload.conflict as FactConflict}));
    if (payload?.currentRevision !== undefined) setRevisionConflict(payload.currentRevision);
    setActionError((current) => ({...current, [id]: messageOf(cause)}));
  }

  async function accept(fact: Fact): Promise<boolean> {
    if (!profile) return false;
    setAction(fact.factId, 'accept');
    setActionError((current) => ({...current, [fact.factId]: ''}));
    try {
      const decision = await acceptProfileFact(fact.factId, profile.revision);
      replaceFact(decision.fact);
      if (decision.profile) onProfileChanged(decision.profile);
      if (decision.conflict) setConflicts((current) => ({...current, [fact.factId]: decision.conflict as FactConflict}));
      return decision.profileChanged;
    } catch (cause) {
      handleError(fact.factId, cause);
      return false;
    } finally {
      setAction(fact.factId, '');
    }
  }

  async function reject(fact: Fact): Promise<boolean> {
    setAction(fact.factId, 'reject');
    setActionError((current) => ({...current, [fact.factId]: ''}));
    try {
      const decision = await rejectProfileFact(fact.factId);
      replaceFact(decision.fact);
      return true;
    } catch (cause) {
      handleError(fact.factId, cause);
      return false;
    } finally {
      setAction(fact.factId, '');
    }
  }

  async function resolve(fact: Fact, choice: 'current' | 'proposed'): Promise<boolean> {
    if (!profile) return false;
    setAction(fact.factId, choice);
    setActionError((current) => ({...current, [fact.factId]: ''}));
    try {
      const decision = await resolveProfileFact(fact.factId, profile.revision, choice);
      replaceFact(decision.fact);
      if (decision.profile) onProfileChanged(decision.profile);
      setConflicts((current) => {
        const next = {...current};
        delete next[fact.factId];
        return next;
      });
      return true;
    } catch (cause) {
      handleError(fact.factId, cause);
      return false;
    } finally {
      setAction(fact.factId, '');
    }
  }

  return {facts, loading, error, actionError, conflicts, busy, revisionConflict, refresh, accept, reject, resolve, setError};
}

export type ProfileFacts = ReturnType<typeof useProfileFacts>;
