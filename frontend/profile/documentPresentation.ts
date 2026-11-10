import type {ProfileDocument} from './contract';

// Presentation model for one profile document. It turns the persisted processing state plus the
// stored error classification into a plain label, explanation and single next action. The view
// renders this model; it owns no request, persistence or workflow effect.

export type DocumentAction = 'analyze' | 'retry' | 'cancel';

export interface DocumentErrorHelp {
  cause: string;
  action: string;
  openSettings: boolean;
  technical?: string;
}

// Maps one failed explicit action (Analyze/Retry) to the same plain cause/action vocabulary as a
// persisted analysis failure. The provider's raw sentence is kept only as technical detail.
export function actionErrorHelp(code: string, message: string, unavailable: boolean): DocumentErrorHelp {
  if (unavailable || code === 'analyzer_unavailable') {
    return {
      cause: 'The configured extraction model is not available.',
      action: 'Open Settings and choose an available model.',
      openSettings: true,
      technical: message,
    };
  }
  return {cause: message, action: '', openSettings: false};
}

export interface DocumentSummary {
  label: string;
  detail: string;
  error?: DocumentErrorHelp;
  technical?: string;
  action: DocumentAction | null;
  actionLabel: string;
  needsUpload: boolean;
  factsText: string | null;
}

function errorKind(document: ProfileDocument): string {
  const value = document.analysisMetadata?.errorKind;
  if (typeof value === 'string' && value) return value;
  // Records written before errorKind existed still carry the provider message. Classify that
  // text so an old timeout is not presented as an unexplained provider failure.
  const text = (document.error ?? '').toLowerCase();
  if (text.includes('context deadline exceeded') || text.includes('timed out') || text.includes('timeout')) return 'timeout';
  if (text.includes('connection refused') || text.includes('no such host') || text.includes('dial tcp')) return 'unavailable';
  if (text.includes('not found') && text.includes('model')) return 'model_unavailable';
  if (text.includes('context overflow')) return 'context_overflow';
  return '';
}

function modelName(document: ProfileDocument): string {
  const value = document.analysisMetadata?.model;
  return typeof value === 'string' ? value : '';
}

function proposedFacts(document: ProfileDocument): number {
  return Object.values(document.factCounts ?? {}).reduce((sum, count) => sum + count, 0);
}

function errorHelp(kind: string): DocumentErrorHelp {
  switch (kind) {
    case 'timeout':
      return {
        cause: 'The selected model took too long to respond.',
        action: 'Try again. If it keeps timing out, check the selected model in Settings.',
        openSettings: true,
      };
    case 'unavailable':
      return {
        cause: 'The selected model could not be reached.',
        action: 'Check that the local model service is running, or choose another model in Settings.',
        openSettings: true,
      };
    case 'model_unavailable':
      return {
        cause: 'The selected model is not available.',
        action: 'Open Settings and choose an available model.',
        openSettings: true,
      };
    case 'context_overflow':
      return {
        cause: 'The document is longer than the selected model can read at once.',
        action: 'Shorten the text or choose a model with a larger context in Settings.',
        openSettings: true,
      };
    case 'invalid_output':
      return {
        cause: 'The model returned a response that could not be read.',
        action: 'Try again. If it repeats, choose another model in Settings.',
        openSettings: true,
      };
    case 'publication':
      return {
        cause: 'The analysis finished, but its result could not be stored.',
        action: 'Try again.',
        openSettings: false,
      };
    case 'source_changed':
      return {
        cause: 'The stored document changed while the analysis was running.',
        action: 'Try again with the current text.',
        openSettings: false,
      };
    default:
      return {
        cause: 'The configured extraction provider failed.',
        action: 'Try again. If it repeats, check the selected model in Settings.',
        openSettings: true,
      };
  }
}

export function documentSummary(document: ProfileDocument): DocumentSummary {
  const state = document.processingState;
  const hasText = document.hasText || document.textChars > 0;
  const active = state === 'extracting' || state === 'analyzing';
  const facts = proposedFacts(document);
  const model = modelName(document);

  let label = state;
  let detail = '';
  let action: DocumentAction | null = null;
  let actionLabel = '';
  let needsUpload = false;
  let factsText: string | null = null;
  let error: DocumentErrorHelp | undefined;

  switch (state) {
    case 'extracting':
      label = 'Reading text…';
      detail = 'Reading text from the file locally. No model is running.';
      break;
    case 'analyzing':
      label = 'Analyzing…';
      detail = model
        ? `The selected model (${model}) is analyzing the text. No facts are published until it finishes.`
        : 'The selected model is analyzing the text. No facts are published until it finishes.';
      action = 'cancel';
      actionLabel = 'Cancel analysis';
      break;
    case 'ready':
    case 'imported':
    case 'uploaded':
    case 'stale':
      label = hasText ? 'Text ready' : 'Not read yet';
      detail = hasText
        ? 'Text is ready. Analysis has not started.'
        : 'The text has not been read yet.';
      break;
    case 'analyzed':
      label = 'Analyzed';
      detail = facts > 0
        ? `${facts} proposed ${facts === 1 ? 'fact' : 'facts'} ready for review.`
        : 'Analysis completed and found no facts matching the profile fields.';
      factsText = facts > 0 ? `${facts} proposed ${facts === 1 ? 'fact' : 'facts'}` : 'No facts found';
      break;
    case 'cancelled':
      label = 'Cancelled';
      detail = 'Analysis was cancelled. No facts were published.';
      break;
    case 'no_text':
      label = 'No text';
      detail = 'No text could be read from this file. Upload a text or searchable PDF/DOCX export.';
      break;
    case 'missing':
      label = 'Original missing';
      detail = hasText
        ? 'The original file is missing, but its cached text is still available.'
        : 'The original file is missing and there is no cached text. Upload a file to analyze it.';
      break;
    case 'error': {
      label = 'Failed';
      const help = errorHelp(errorKind(document));
      error = help;
      detail = `${help.cause} ${help.action}`;
      break;
    }
    default:
      detail = 'The document status is not recognised; refresh the list.';
  }

  if (!active) {
    if (state === 'error') {
      if (hasText || document.hasOriginal) {
        action = 'retry';
        actionLabel = 'Retry analysis';
      } else {
        needsUpload = true;
      }
    } else if (state === 'cancelled' && hasText) {
      action = 'retry';
      actionLabel = 'Retry analysis';
    } else if (hasText) {
      action = 'analyze';
      actionLabel = state === 'analyzed' ? 'Analyze again' : 'Analyze';
    } else if (document.hasOriginal && state !== 'no_text' && state !== 'missing') {
      action = 'analyze';
      actionLabel = 'Analyze';
    } else {
      // No cached text and no readable original: the honest next action is to upload a usable
      // file, not to repeat an analysis that has nothing to read.
      needsUpload = true;
    }
  }

  return {
    label,
    detail,
    error,
    technical: error ? document.error : undefined,
    action,
    actionLabel,
    needsUpload,
    factsText,
  };
}
