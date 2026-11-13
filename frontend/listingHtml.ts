// Pure module: parses a sanitised HTML string into an allowed-node tree.
// Uses the browser's inert DOMParser — never assigns innerHTML to a live element.
//
// Whitelist: h2, h3, h4, p, ul, ol, li, strong, em, br, text nodes.
// Disallowed elements are unwrapped to their children.
// script and style elements are dropped entirely (along with their content).
// No attributes are kept on any element.

export type AllowedTag = 'h2'|'h3'|'h4'|'p'|'ul'|'ol'|'li'|'strong'|'em'|'br';

export type HtmlNode =
  | { type: 'text'; text: string }
  | { type: 'element'; tag: AllowedTag; children: HtmlNode[] };

const ALLOWED = new Set<string>(['h2','h3','h4','p','ul','ol','li','strong','em','br']);
// These tags are dropped together with all their descendants.
const DROP_WITH_CONTENT = new Set<string>(['script','style','head']);

function walkNode(node: Node): HtmlNode[] {
  if (node.nodeType === Node.TEXT_NODE) {
    const t = node.textContent ?? '';
    return t.length ? [{ type: 'text', text: t }] : [];
  }
  if (node.nodeType !== Node.ELEMENT_NODE) return [];
  const el = node as Element;
  const tag = el.tagName.toLowerCase();
  if (DROP_WITH_CONTENT.has(tag)) return [];
  const children = Array.from(el.childNodes).flatMap(walkNode);
  if (ALLOWED.has(tag)) {
    return [{ type: 'element', tag: tag as AllowedTag, children }];
  }
  // Unwrap: disallowed element passes its children up.
  return children;
}

// Parses an HTML string and returns the allowed-node tree for the body content.
// Safe: uses DOMParser (inert parse), never touches innerHTML on a live element.
export function parseListingHtml(html: string): HtmlNode[] {
  const doc = new DOMParser().parseFromString(html, 'text/html');
  return Array.from(doc.body.childNodes).flatMap(walkNode);
}
