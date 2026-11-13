import React from 'react';
import {toSupportedDocument, type RichTextValue} from './richText';

// Read-only rendering of the same server-supported document subset the editor produces. It is a
// pure React tree: no HTML string, no dangerouslySetInnerHTML, and unsupported nodes/marks are
// already removed by toSupportedDocument. A value with no supported document falls back to its
// retained plain text exactly as before.

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function renderInline(nodes: unknown[], prefix: string): React.ReactNode[] {
  return nodes.map((node, index) => {
    if (!isObject(node)) return null;
    const key = `${prefix}-${index}`;
    if (node.type === 'hardBreak') return <br key={key}/>;
    if (node.type !== 'text') return null;
    let content: React.ReactNode = typeof node.text === 'string' ? node.text : '';
    const marks = Array.isArray(node.marks) ? node.marks : [];
    for (const mark of marks) {
      if (!isObject(mark)) continue;
      if (mark.type === 'bold') content = <strong>{content}</strong>;
      else if (mark.type === 'italic') content = <em>{content}</em>;
    }
    return <React.Fragment key={key}>{content}</React.Fragment>;
  });
}

function renderBlocks(nodes: unknown[], prefix: string): React.ReactNode[] {
  return nodes.map((node, index) => renderBlock(node, `${prefix}-${index}`));
}

function renderBlock(node: unknown, key: string): React.ReactNode {
  if (!isObject(node)) return null;
  const content = Array.isArray(node.content) ? node.content : [];
  switch (node.type) {
    case 'paragraph':
      return <p key={key}>{renderInline(content, key)}</p>;
    case 'heading': {
      const raw = isObject(node.attrs) ? Number(node.attrs.level) : 1;
      const level = Number.isInteger(raw) && raw >= 1 && raw <= 6 ? raw : 1;
      const Tag = `h${level}` as keyof React.JSX.IntrinsicElements;
      return <Tag key={key}>{renderInline(content, key)}</Tag>;
    }
    case 'bulletList':
      return <ul key={key}>{renderBlocks(content, key)}</ul>;
    case 'orderedList':
      return <ol key={key}>{renderBlocks(content, key)}</ol>;
    case 'listItem':
      return <li key={key}>{renderBlocks(content, key)}</li>;
    case 'text':
      return <React.Fragment key={key}>{renderInline([node], key)}</React.Fragment>;
    case 'hardBreak':
      return <br key={key}/>;
    default:
      return null;
  }
}

export function RichTextView({value}: {value: RichTextValue}) {
  const document = toSupportedDocument(value.document);
  if (!document) {
    return <div className="profile-rich-read">{value.text}</div>;
  }
  const content = Array.isArray(document.content) ? document.content : [];
  return <div className="profile-rich-view">{renderBlocks(content, 'rich')}</div>;
}
