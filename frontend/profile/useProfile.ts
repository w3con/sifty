import {useCallback, useEffect, useRef, useState} from 'react';
import {ApiError} from '../api';
import {
  acceptProfileFact,
  analyzeDocument,
  cancelDocumentAnalysis,
  deleteProfileDocument,
  deleteProfileAvatar,
  documentFileUrl,
  getDocumentText,
  getProfile,
  getProfileAvatarMeta,
  getProfileCapabilities,
  listProfileDocuments,
  listProfileFacts,
  putAbsentSkills,
  putProfileSection,
  refreshProfileDerivedContext,
  rejectProfileFact,
  resolveProfileFact,
  retryDocument,
  profileAvatarUrl,
  uploadProfileAvatar,
  uploadProfileDocument,
  uploadProfileText,
  type ProfileAvatarMeta,
} from './api';
import {actionErrorHelp, type DocumentErrorHelp} from './documentPresentation';
import {
  DOCUMENT_ACTIVE_STATES,
  cleanSection,
  draftValues,
  sectionValues,
  type Capabilities,
  type ConflictPayload,
  type Drafts,
  type Fact,
  type FactConflict,
  type Profile,
  type ProfileDocument,
  type SectionDraft,
  type SectionKey,
  type SectionValues,
} from './contract';
import {humanizeSectionError} from './sectionErrors';

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

function sameSectionValue<T>(left: T, right: T): boolean {
  return JSON.stringify(left) === JSON.stringify(right);
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
  const [derivedRefreshing, setDerivedRefreshing] = useState(false);
  const [derivedError, setDerivedError] = useState('');
  const profileRef = useRef<Profile | null>(null);

  useEffect(() => {
    profileRef.current = profile;
  }, [profile]);

  // Responses can arrive out of order when two sections save at once. A lower revision is a
  // snapshot the client already superseded; applying it would roll back the latest known profile.
  const applyNewerProfile = useCallback((next: Profile) => {
    const current = profileRef.current;
    if (current && next.revision < current.revision) return;
    setProfile(next);
    profileRef.current = next;
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [nextProfile, nextCapabilities] = await Promise.all([getProfile(), getProfileCapabilities()]);
      applyNewerProfile(nextProfile);
      setCapabilities(nextCapabilities);
    } catch (cause) {
      setError(messageOf(cause));
    } finally {
      setLoading(false);
    }
  }, [applyNewerProfile]);

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
          value: draftValues(current.data, key),
          dirty: false,
          saving: false,
          error: '',
        },
      } as Drafts;
    });
  }, []);

  // Cancel closes only the chosen editor. No other section draft or stored value changes.
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

  // The server merges only this section into the current stored profile. Other open drafts stay
  // exactly as they are, and a later edit made while this request was in flight is kept.
  const saveSection = useCallback(async (key: SectionKey): Promise<boolean> => {
    const draft = drafts[key];
    if (!draft) return false;
    const submitted = draft.value;
    const currentProfile = profileRef.current;
    const current = currentProfile ? sectionValues(currentProfile.data, key) : undefined;
    const cleaned = cleanSection(key, submitted as never, current as never);
    const cleanedCurrent = current ? cleanSection(key, current as never, current as never) : undefined;
    // Saving an untouched editor (or a value changed and reverted) is a local no-op. Sending it
    // would still let the server's effective-source markers turn an unchanged empty legacy field
    // into a new revision, and a seeded Other requirements block into a migration.
    if (!draft.dirty || (cleanedCurrent && sameSectionValue(cleaned, cleanedCurrent))) {
      setDrafts((existing) => {
        const next: Drafts = {...existing};
        delete next[key];
        return next;
      });
      setNotice('No changes to save.');
      return true;
    }
    patchDraft(key, {saving: true, error: ''});
    setNotice('');
    try {
      const result = await putProfileSection(key, cleaned);
      applyNewerProfile(result.profile);
      setDrafts((existing) => {
        const next: Drafts = {...existing};
        const currentDraft = next[key];
        if (currentDraft && sameSectionValue(currentDraft.value, submitted)) {
          delete next[key];
        } else if (currentDraft) {
          setDraftAt(next, key, {...currentDraft, saving: false, error: ''});
        }
        return next;
      });
      setNotice(result.changed ? 'Saved.' : 'No changes to save.');
      return true;
    } catch (cause) {
      patchDraft(key, {saving: false, error: humanizeSectionError(key, messageOf(cause))});
      return false;
    }
  }, [drafts, applyNewerProfile]);

  // Takes one skill off the list of skills the owner has said they do not have. It saves at once:
  // the list is not an editor draft, it is filled from vacancies and only trimmed here.
  const removeAbsentSkill = useCallback(async (key: string) => {
    const current = profileRef.current;
    if (!current) return;
    const rest = (current.data.absentSkills ?? []).filter((skill) => skill.key !== key);
    try {
      const result = await putAbsentSkills(rest);
      applyNewerProfile(result.profile);
      setNotice('Saved.');
    } catch (cause) {
      setError(messageOf(cause));
    }
  }, [applyNewerProfile]);

  // A fact action can update the stored profile while section drafts are open. The drafts are
  // deliberately not marked stale: version comparison is not part of the editing workflow.
  const applyExternalProfile = useCallback((next: Profile) => {
    applyNewerProfile(next);
  }, [applyNewerProfile]);

  // Explicit retry of the local derived context/ranking projection. This is the same local-only
  // path used after a save; it starts no extraction, remote fetch, model call or assessment.
  const refreshDerived = useCallback(async () => {
    setDerivedRefreshing(true);
    setDerivedError('');
    try {
      const result = await refreshProfileDerivedContext();
      applyNewerProfile(result.profile);
      if (!result.refreshed) {
        setDerivedError(result.profile.contextError || result.profile.fastRankingError
          || 'Scoring could not be refreshed; the saved profile is unchanged.');
      }
    } catch (cause) {
      setDerivedError(messageOf(cause));
    } finally {
      setDerivedRefreshing(false);
    }
  }, [applyNewerProfile]);

  return {
    profile, capabilities, loading, error, notice, drafts, derivedRefreshing, derivedError,
    load, beginEdit, cancelEdit, changeDraft, saveSection, applyExternalProfile, refreshDerived, removeAbsentSkill,
    setError, setNotice,
  };
}

export type ProfileEditor = ReturnType<typeof useProfileEditor>;

export function readBase64(file: File): Promise<string> {
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

export function useProfileAvatar(capabilities: Capabilities | null) {
  const [meta, setMeta] = useState<ProfileAvatarMeta>({exists: false});
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      setMeta(await getProfileAvatarMeta());
      setError('');
    } catch (cause) {
      setError(messageOf(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const upload = useCallback(async (file: File): Promise<boolean> => {
    setBusy(true);
    setError('');
    try {
      if (capabilities) {
        const extension = fileExtension(file.name);
        const normalized = extension === 'jpeg' ? 'jpg' : extension;
        const allowed = capabilities.avatarTypes.map((type) => type.toLowerCase() === 'jpeg' ? 'jpg' : type.toLowerCase());
        if (!allowed.includes(normalized)) {
          setError(`This photo type is not accepted. Configured types: ${capabilities.avatarTypes.join(', ')}.`);
          return false;
        }
        if (file.size > capabilities.maxAvatarBytes) {
          setError(`This photo is too large. The configured limit is ${capabilities.maxAvatarBytes} bytes.`);
          return false;
        }
      }
      const encoded = await readBase64(file);
      const saved = await uploadProfileAvatar(file.name, encoded);
      setMeta(saved);
      return true;
    } catch (cause) {
      setError(messageOf(cause));
      return false;
    } finally {
      setBusy(false);
    }
  }, [capabilities]);

  const remove = useCallback(async (): Promise<boolean> => {
    setBusy(true);
    setError('');
    try {
      await deleteProfileAvatar();
      setMeta({exists: false});
      return true;
    } catch (cause) {
      setError(messageOf(cause));
      return false;
    } finally {
      setBusy(false);
    }
  }, []);

  return {
    meta, loading, busy, error, upload, remove,
    url: meta.exists ? profileAvatarUrl(meta.updatedAt) : '',
  };
}

export type ProfileAvatar = ReturnType<typeof useProfileAvatar>;

function describeActionError(cause: unknown): DocumentErrorHelp {
  if (cause instanceof ApiError) {
    return actionErrorHelp(String(cause.data?.code ?? ''), messageOf(cause), cause.status === 503);
  }
  return {cause: messageOf(cause), action: '', openSettings: false};
}

export function useProfileDocuments(capabilities: Capabilities | null) {
  const [documents, setDocuments] = useState<ProfileDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [errorAction, setErrorAction] = useState('');
  const [errorTechnical, setErrorTechnical] = useState('');
  const [errorOpenSettings, setErrorOpenSettings] = useState(false);
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState('');
  const [uploadResults, setUploadResults] = useState<{name: string; ok: boolean; message: string}[]>([]);
  const [texts, setTexts] = useState<Record<string, {loading: boolean; text: string; error: string}>>({});
  // Names the document whose explicit action produced the current transient notice, so a later
  // poll can replace "started" with the real outcome instead of leaving a stale success message.
  const pendingNotice = useRef<{id: string; name: string} | null>(null);

  function clearActionError() {
    setError('');
    setErrorAction('');
    setErrorTechnical('');
    setErrorOpenSettings(false);
  }

  function showActionError(cause: unknown) {
    const help = describeActionError(cause);
    setError(help.cause);
    setErrorAction(help.action);
    setErrorTechnical(help.technical ?? '');
    setErrorOpenSettings(help.openSettings);
  }

  const refresh = useCallback(async () => {
    try {
      const list = await listProfileDocuments();
      setDocuments(list);
      clearActionError();
      const pending = pendingNotice.current;
      if (pending) {
        const document = list.find((item) => item.documentId === pending.id);
        if (document && !(DOCUMENT_ACTIVE_STATES as readonly string[]).includes(document.processingState)) {
          pendingNotice.current = null;
          if (document.processingState === 'analyzed') {
            const facts = Object.values(document.factCounts ?? {}).reduce((sum, count) => sum + count, 0);
            setNotice(facts > 0
              ? `Analysis completed for ${pending.name}: ${facts} proposed ${facts === 1 ? 'fact' : 'facts'}.`
              : `Analysis completed for ${pending.name}: no facts were found.`);
          } else if (document.processingState === 'error') {
            setNotice(`Analysis failed for ${pending.name}. See the reason and next step below.`);
          } else if (document.processingState === 'cancelled') {
            setNotice(`Analysis cancelled for ${pending.name}.`);
          } else if (document.processingState === 'no_text') {
            setNotice(`No text could be read for ${pending.name}. Upload a searchable export.`);
          }
        }
      }
    } catch (cause) {
      showActionError(cause);
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
    clearActionError();
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
      showActionError(cause);
      return false;
    } finally {
      setBusy('');
    }
  }, [capabilities]);

  // Several files are uploaded one after another through the existing upload path. A failure
  // names that file and keeps every earlier success; no file is analyzed automatically.
  const uploadFiles = useCallback(async (files: File[]): Promise<{name: string; ok: boolean; message: string}[]> => {
    if (!files.length) return [];
    setBusy('upload');
    clearActionError();
    setNotice('');
    const results: {name: string; ok: boolean; message: string}[] = [];
    for (const file of files) {
      try {
        const invalid = validate(file.name, file.size);
        if (invalid) throw new Error(invalid);
        const encoded = await readBase64(file);
        const uploaded = await uploadProfileDocument(file.name, encoded);
        setDocuments((existing) => [uploaded, ...existing.filter((document) => document.documentId !== uploaded.documentId)]);
        results.push({name: file.name, ok: true, message: 'Uploaded; analysis is explicit.'});
      } catch (cause) {
        const help = describeActionError(cause);
        results.push({name: file.name, ok: false, message: [help.cause, help.action].filter(Boolean).join(' ')});
      }
    }
    setUploadResults(results);
    const succeeded = results.filter((result) => result.ok).length;
    if (succeeded) {
      setNotice(`Uploaded ${succeeded} ${succeeded === 1 ? 'document' : 'documents'}. Review status; analysis is explicit.`);
    }
    if (succeeded < results.length) {
      setError(`${results.length - succeeded} of ${results.length} uploads failed. Successful files are kept.`);
    }
    setBusy('');
    return results;
  }, [capabilities]);

  const uploadText = useCallback(async (filename: string, text: string): Promise<boolean> => {
    setBusy('upload');
    clearActionError();
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
      showActionError(cause);
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
    clearActionError();
    setNotice('');
    try {
      const updated = action === 'analyze'
        ? await analyzeDocument(id)
        : action === 'cancel'
          ? await cancelDocumentAnalysis(id)
          : await retryDocument(id);
      replaceDocument(updated);
      if (action === 'cancel') {
        pendingNotice.current = null;
        setNotice(`Analysis cancelled for ${updated.originalName}.`);
      } else {
        pendingNotice.current = {id, name: updated.originalName};
        setNotice(action === 'analyze'
          ? `Analysis started for ${updated.originalName}.`
          : `Retry started for ${updated.originalName}.`);
      }
      return true;
    } catch (cause) {
      showActionError(cause);
      return false;
    } finally {
      setBusy('');
    }
  }

  async function removeDocument(id: string, name: string) {
    setBusy(id);
    clearActionError();
    setNotice('');
    try {
      await deleteProfileDocument(id);
      setDocuments((current) => current.filter((document) => document.documentId !== id));
      setNotice(`${name} deleted.`);
      return true;
    } catch (cause) {
      // Until the server implements DELETE, say so plainly instead of a raw HTTP error.
      if (cause instanceof ApiError && [404, 405, 501].includes(cause.status)) {
        setError('Deleting files is not supported by the server yet.');
        setErrorAction('');
        setErrorTechnical(cause.message);
      } else {
        showActionError(cause);
      }
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
    documents, loading, error, errorAction, errorTechnical, errorOpenSettings, notice, busy, active, texts,
    uploadResults,
    refresh, uploadFile, uploadFiles, uploadText, analyze: (id: string) => act('analyze', id),
    cancel: (id: string) => act('cancel', id), retry: (id: string) => act('retry', id), remove: removeDocument, loadText,
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
    if (payload?.conflict) {
      setConflicts((current) => ({...current, [id]: payload.conflict as FactConflict}));
      setActionError((current) => ({...current, [id]: messageOf(cause)}));
      return;
    }
    if (payload?.currentRevision !== undefined) {
      if (payload.current) onProfileChanged(payload.current);
      setActionError((current) => ({
        ...current,
        [id]: 'The profile changed while this proposal was being applied. Review the current profile and try again.',
      }));
      return;
    }
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

  return {facts, loading, error, actionError, conflicts, busy, refresh, accept, reject, resolve, setError};
}

export type ProfileFacts = ReturnType<typeof useProfileFacts>;
