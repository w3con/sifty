import React, {useState} from 'react';
import {EditorContent, useEditor, useEditorState} from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import {editorContent, plainTextFromDocument, toSupportedDocument, type RichTextValue} from './richText';

// The server accepts only paragraphs, headings, bullet/ordered lists, bold, italic and hard
// breaks. Starter Kit also ships nodes and marks the server refuses, so they are disabled here
// rather than hidden in the toolbar: the editor can never produce a document that fails on save.
const extensions = [
  StarterKit.configure({
    blockquote: false,
    codeBlock: false,
    code: false,
    horizontalRule: false,
    strike: false,
    link: false,
    underline: false,
    trailingNode: false,
    heading: {levels: [1, 2, 3]},
  }),
];

export type RichTextEditorProps = {
  label: string;
  value: RichTextValue;
  onChange: (value: RichTextValue) => void;
  hint?: string;
};

// A toolbar button must not take focus on mousedown. Otherwise the editor loses its selection
// before the command runs, and the next keystroke goes to the button instead of the document.
function ToolbarButton({label, active, onClick, children}: {
  label: string; active: boolean; onClick: () => void; children: React.ReactNode;
}) {
  return <button type="button" aria-label={label} title={label} aria-pressed={active}
    onMouseDown={(event) => event.preventDefault()} onClick={onClick}>{children}</button>;
}

export function RichTextEditor({label, value, onChange, hint}: RichTextEditorProps) {
  // Initial content is fixed for this mounted row. The parent keeps the resulting value in the
  // section draft; an unrelated rerender must not reset the caret or the document.
  const [initial] = useState(() => editorContent(value));
  const editor = useEditor({
    extensions,
    content: initial,
    editorProps: {
      attributes: {
        class: 'profile-rich-text',
        'aria-label': label,
        role: 'textbox',
        'aria-multiline': 'true',
      },
    },
    onUpdate: ({editor: current}) => {
      const document = toSupportedDocument(current.getJSON());
      if (!document) return;
      onChange({document, text: plainTextFromDocument(document)});
    },
  });

  const active = useEditorState({
    editor,
    selector: ({editor: current}) => current ? {
      bold: current.isActive('bold'),
      italic: current.isActive('italic'),
      bulletList: current.isActive('bulletList'),
      orderedList: current.isActive('orderedList'),
      heading1: current.isActive('heading', {level: 1}),
      heading2: current.isActive('heading', {level: 2}),
      heading3: current.isActive('heading', {level: 3}),
    } : null,
  });

  return <div className="profile-field-block">
    <span>{label}</span>
    <div className="profile-rich-editor">
      <div className="profile-rich-toolbar" role="toolbar" aria-label={`${label} formatting`}>
        <ToolbarButton label="Bold" active={active?.bold ?? false} onClick={() => editor.chain().focus().toggleBold().run()}><strong>B</strong></ToolbarButton>
        <ToolbarButton label="Italic" active={active?.italic ?? false} onClick={() => editor.chain().focus().toggleItalic().run()}><em>I</em></ToolbarButton>
        <ToolbarButton label="Heading 1" active={active?.heading1 ?? false} onClick={() => editor.chain().focus().toggleHeading({level: 1}).run()}>H1</ToolbarButton>
        <ToolbarButton label="Heading 2" active={active?.heading2 ?? false} onClick={() => editor.chain().focus().toggleHeading({level: 2}).run()}>H2</ToolbarButton>
        <ToolbarButton label="Heading 3" active={active?.heading3 ?? false} onClick={() => editor.chain().focus().toggleHeading({level: 3}).run()}>H3</ToolbarButton>
        <ToolbarButton label="Bulleted list" active={active?.bulletList ?? false} onClick={() => editor.chain().focus().toggleBulletList().run()}>• List</ToolbarButton>
        <ToolbarButton label="Numbered list" active={active?.orderedList ?? false} onClick={() => editor.chain().focus().toggleOrderedList().run()}>1. List</ToolbarButton>
      </div>
      <EditorContent editor={editor}/>
    </div>
    <small>{hint || 'Bold, italic, headings and lists are supported; other formatting is not stored.'}</small>
  </div>;
}
