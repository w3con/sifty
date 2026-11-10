import {api} from '../api';
import type {Capabilities, Fact, FactDecision, Profile, ProfileDocument, SectionKey, SectionValues} from './contract';

// Typed wrappers over the existing api() transport. Nothing here starts model work; only
// analyzeDocument is an explicit user action that reaches the extraction provider.

export const getProfile = (): Promise<Profile> => api('profile');
export interface SectionSaveResult {
  profile: Profile;
  changed: boolean;
}
// A section save sends only the editor's section; the server atomically merges it into the
// current stored profile. The client never submits a full profile from a section editor.
export const putProfileSection = <K extends SectionKey>(section: K, value: SectionValues[K]): Promise<SectionSaveResult> =>
  api(`profile/sections/${encodeURIComponent(section)}`, 'PUT', {value});
export const getProfileCapabilities = (): Promise<Capabilities> => api('profile/capabilities');
// Recomputes the local derived context/ranking from the already-saved profile. It starts no
// extraction, remote fetch, model call or assessment.
export interface DerivedRefreshResult {
  profile: Profile;
  refreshed: boolean;
}
export const refreshProfileDerivedContext = (): Promise<DerivedRefreshResult> =>
  api('profile/derived/refresh', 'POST', {});

export const listProfileDocuments = (): Promise<ProfileDocument[]> => api('profile/documents');
export const uploadProfileDocument = (filename: string, contentBase64: string): Promise<ProfileDocument> =>
  api('profile/documents', 'POST', {filename, contentBase64});
export const uploadProfileText = (filename: string, text: string): Promise<ProfileDocument> =>
  api('profile/documents', 'POST', {filename, text});
export const getDocumentText = (id: string): Promise<{documentId: string; text: string}> =>
  api(`profile/documents/${encodeURIComponent(id)}/text`);
export const analyzeDocument = (id: string): Promise<ProfileDocument> =>
  api(`profile/documents/${encodeURIComponent(id)}/analyze`, 'POST', {});
export const cancelDocumentAnalysis = (id: string): Promise<ProfileDocument> =>
  api(`profile/documents/${encodeURIComponent(id)}/cancel`, 'POST', {});
export const retryDocument = (id: string): Promise<ProfileDocument> =>
  api(`profile/documents/${encodeURIComponent(id)}/retry`, 'POST', {});

// Deletes one uploaded file or link. Server rule (to implement): open proposals from this document
// are removed with it; facts already accepted into the profile stay and keep a "deleted file" source.
export const deleteProfileDocument = (id: string): Promise<{deleted: boolean}> =>
  api(`profile/documents/${encodeURIComponent(id)}`, 'DELETE');

export const documentFileUrl = (id: string): string =>
  `/api/profile/documents/${encodeURIComponent(id)}/file`;

export const listProfileFacts = (documentId?: string): Promise<Fact[]> =>
  api(documentId ? `profile/facts?documentId=${encodeURIComponent(documentId)}` : 'profile/facts');
export const acceptProfileFact = (factId: string, expectedRevision: number): Promise<FactDecision> =>
  api(`profile/facts/${encodeURIComponent(factId)}/accept`, 'POST', {expectedRevision});
export const rejectProfileFact = (factId: string): Promise<FactDecision> =>
  api(`profile/facts/${encodeURIComponent(factId)}/reject`, 'POST', {});
export const resolveProfileFact = (factId: string, expectedRevision: number, choice: 'current' | 'proposed'): Promise<FactDecision> =>
  api(`profile/facts/${encodeURIComponent(factId)}/resolve`, 'POST', {expectedRevision, choice});

// Profile photo. Metadata is separate from the bytes; the stored image is never sent to a model
// and is not part of the CV/document evidence list.
export interface ProfileAvatarMeta {
  exists: boolean;
  contentType?: string;
  byteSize?: number;
  updatedAt?: string;
}
export const getProfileAvatarMeta = (): Promise<ProfileAvatarMeta> => api('profile/avatar/meta');
export const uploadProfileAvatar = (filename: string, contentBase64: string): Promise<ProfileAvatarMeta> =>
  api('profile/avatar', 'POST', {filename, contentBase64});
export const deleteProfileAvatar = (): Promise<{exists: false; deleted: boolean}> =>
  api('profile/avatar', 'DELETE');
export const profileAvatarUrl = (updatedAt?: string): string =>
  `/api/profile/avatar${updatedAt ? `?v=${encodeURIComponent(updatedAt)}` : ''}`;
