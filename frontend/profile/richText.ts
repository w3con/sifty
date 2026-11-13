// Client-side helpers for the supported rich-text subset. The server remains authoritative: it
// validates the document and derives the plain text used by ranking, assessment and preparation.
// These helpers only keep the editor from emitting nodes or marks the server refuses (for example
// ProseMirror's ordered-list `start`/`type` attributes) and provide a plain-text fallback for
// values stored before rich text existed.

export interface RichTextValue {
  text: string;
  document?: unknown;
}

const SUPPORTED_NODES = new Set([
  'doc',
  'paragraph',
  'heading',
  'bulletList',
  'orderedList',
  'listItem',
  'text',
  'hardBreak',
]);

const SUPPORTED_MARKS = new Set(['bold', 'italic']);

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

export function emptyDocument(): Record<string, unknown> {
  return {type: 'doc', content: [{type: 'paragraph'}]};
}

// documentFromPlainText preserves every line exactly by using hard breaks inside one paragraph.
// It is used only when a stored value has plain text but no supported document, so opening the
// editor does not silently lose or reflow the retained text.
export function documentFromPlainText(text: string): Record<string, unknown> {
  const source = typeof text === 'string' ? text.replace(/\r\n?/g, '\n') : '';
  if (!source) return emptyDocument();
  const content: Record<string, unknown>[] = [];
  source.split('\n').forEach((line, index) => {
    if (index > 0) content.push({type: 'hardBreak'});
    if (line) content.push({type: 'text', text: line});
  });
  return {type: 'doc', content: [{type: 'paragraph', content}]};
}

function sanitizeNode(input: unknown): Record<string, unknown> | null {
  if (!isObject(input)) return null;
  const type = typeof input.type === 'string' ? input.type : '';
  if (!SUPPORTED_NODES.has(type)) return null;
  if (type === 'text') {
    const text = typeof input.text === 'string' ? input.text : '';
    if (!text) return null;
    const marks = Array.isArray(input.marks)
      ? input.marks
        .filter((mark): mark is Record<string, unknown> => isObject(mark) && typeof mark.type === 'string' && SUPPORTED_MARKS.has(mark.type))
        .map((mark) => ({type: mark.type}))
      : [];
    return marks.length ? {type: 'text', text, marks} : {type: 'text', text};
  }
  if (type === 'hardBreak') return {type: 'hardBreak'};
  const out: Record<string, unknown> = {type};
  if (type === 'heading') {
    const rawLevel = isObject(input.attrs) ? Number(input.attrs.level) : NaN;
    const level = Number.isInteger(rawLevel) && rawLevel >= 1 && rawLevel <= 6 ? rawLevel : 1;
    out.attrs = {level};
  }
  if (Array.isArray(input.content)) {
    out.content = input.content.map(sanitizeNode).filter((node): node is Record<string, unknown> => node !== null);
  }
  return out;
}

// toSupportedDocument returns a document the backend validator accepts, or null when the value is
// not a supported document at all. It deliberately drops all attributes except a heading level.
export function toSupportedDocument(value: unknown): Record<string, unknown> | null {
  if (!isObject(value) || value.type !== 'doc') return null;
  const content = Array.isArray(value.content)
    ? value.content.map(sanitizeNode).filter((node): node is Record<string, unknown> => node !== null)
    : [];
  return {type: 'doc', content};
}

export function editorContent(value: RichTextValue): Record<string, unknown> {
  const document = toSupportedDocument(value.document);
  return document ?? documentFromPlainText(value.text ?? '');
}

// plainTextFromDocument mirrors the server's authoritative projection closely enough for the
// draft text and for tests. The server still recomputes and stores this value on save.
export function plainTextFromDocument(value: unknown): string {
  const document = toSupportedDocument(value);
  if (!document) return '';
  const out: string[] = [];
  writeNode(document, out, 0);
  return out.join('').trim();
}

function inlineText(node: Record<string, unknown>): string {
  const content = Array.isArray(node.content) ? node.content : [];
  let out = '';
  for (const child of content) {
    if (!isObject(child)) continue;
    if (child.type === 'text') out += typeof child.text === 'string' ? child.text : '';
    else if (child.type === 'hardBreak') out += '\n';
    else out += inlineText(child);
  }
  return out;
}

function writeNode(node: Record<string, unknown>, out: string[], depth: number): void {
  const content = Array.isArray(node.content) ? node.content : [];
  switch (node.type) {
    case 'doc':
      for (const child of content) if (isObject(child)) writeNode(child, out, depth);
      break;
    case 'paragraph':
    case 'heading':
      out.push(inlineText(node), '\n\n');
      break;
    case 'bulletList':
    case 'orderedList':
      for (const child of content) {
        if (isObject(child) && child.type === 'listItem') writeListItem(child, out, depth);
      }
      out.push('\n');
      break;
    case 'listItem':
      writeListItem(node, out, depth);
      break;
    case 'text':
      out.push(typeof node.text === 'string' ? node.text : '');
      break;
    case 'hardBreak':
      out.push('\n');
      break;
  }
}

function writeListItem(node: Record<string, unknown>, out: string[], depth: number): void {
  const indent = '  '.repeat(depth);
  const content = Array.isArray(node.content) ? node.content : [];
  let wrote = false;
  for (const child of content) {
    if (!isObject(child)) continue;
    switch (child.type) {
      case 'paragraph':
      case 'heading': {
        const text = inlineText(child);
        if (!wrote) {
          out.push(indent, '- ', text, '\n');
          wrote = true;
        } else {
          out.push(indent, '  ', text, '\n');
        }
        break;
      }
      case 'bulletList':
      case 'orderedList':
        writeNode(child, out, depth + 1);
        wrote = true;
        break;
      case 'hardBreak':
        out.push('\n');
        break;
      case 'text':
        if (!wrote) {
          out.push(indent, '- ');
          wrote = true;
        }
        out.push(typeof child.text === 'string' ? child.text : '', '\n');
        break;
    }
  }
  if (!wrote) out.push(indent, '-\n');
}
