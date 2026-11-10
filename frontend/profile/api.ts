import {api} from '../api';
import type {Capabilities, Fact, FactDecision, Profile, ProfileData, ProfileDocument} from './contract';

// Typed wrappers over the existing api() transport. Nothing here starts model work; only
// analyzeDocument is an explicit user action that reaches the extraction provider.

export const getProfile = (): Promise<Profile> => api('profile');
export const putProfile = (expectedRevision: number, data: ProfileData): Promise<Profile> =>
  api('profile', 'PUT', {expectedRevision, data});
export const getProfileCapabilities = (): Promise<Capabilities> => api('profile/capabilities');

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
