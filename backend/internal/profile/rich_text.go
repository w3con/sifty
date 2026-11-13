package profile

import (
	"encoding/json"
	"fmt"
	"strings"
)

// This file defines the supported rich-text document subset and its plain-text derivative.
// The document shape is the Tiptap/ProseMirror JSON shape the editor will emit. Only the nodes
// and marks the product uses are accepted: paragraphs, headings, lists, bold and italic. Embeds,
// arbitrary HTML and unknown nodes are refused rather than stored and re-rendered later.

type richMark struct {
	Type  string                     `json:"type"`
	Attrs map[string]json.RawMessage `json:"attrs,omitempty"`
}

type richNode struct {
	Type    string                     `json:"type"`
	Text    string                     `json:"text,omitempty"`
	Marks   []richMark                 `json:"marks,omitempty"`
	Attrs   map[string]json.RawMessage `json:"attrs,omitempty"`
	Content []richNode                 `json:"content,omitempty"`
}

var supportedRichNodes = map[string]bool{
	"doc": true, "paragraph": true, "heading": true,
	"bulletList": true, "orderedList": true, "listItem": true,
	"text": true, "hardBreak": true,
}

var supportedRichMarks = map[string]bool{"bold": true, "italic": true}

// hasRichTextDocument distinguishes an absent document from an explicit JSON null used to
// clear one during a section save.
func hasRichTextDocument(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

// validateRichTextDocument checks that raw is a supported document. An absent document is valid
// and means no rich text; the existing plain-text value remains authoritative in that case.
func validateRichTextDocument(raw json.RawMessage) error {
	if !hasRichTextDocument(raw) {
		return nil
	}
	var root richNode
	if err := json.Unmarshal(raw, &root); err != nil {
		return fmt.Errorf("must be a JSON document: %w", err)
	}
	if root.Type != "doc" {
		return fmt.Errorf("document root must be type doc")
	}
	if len(root.Attrs) > 0 {
		return fmt.Errorf("document root does not accept attributes")
	}
	return validateRichNode(root, true)
}

func validateRichNode(n richNode, root bool) error {
	if !root && !supportedRichNodes[n.Type] {
		return fmt.Errorf("unsupported rich-text node %q", n.Type)
	}
	if n.Type != "text" && strings.TrimSpace(n.Text) != "" {
		return fmt.Errorf("%s cannot contain text directly", n.Type)
	}
	for _, mark := range n.Marks {
		if !supportedRichMarks[mark.Type] {
			return fmt.Errorf("unsupported rich-text mark %q", mark.Type)
		}
		if len(mark.Attrs) > 0 {
			return fmt.Errorf("rich-text mark %q does not accept attributes", mark.Type)
		}
	}
	switch n.Type {
	case "doc", "paragraph", "bulletList", "orderedList", "listItem":
		if len(n.Attrs) > 0 {
			return fmt.Errorf("%s does not accept attributes", n.Type)
		}
	case "heading":
		if len(n.Attrs) != 1 {
			return fmt.Errorf("heading requires only a level attribute")
		}
		var level int
		if err := json.Unmarshal(n.Attrs["level"], &level); err != nil || level < 1 || level > 6 {
			return fmt.Errorf("heading level must be 1 to 6")
		}
	case "text":
		if len(n.Content) > 0 {
			return fmt.Errorf("text cannot contain child nodes")
		}
		if len(n.Attrs) > 0 {
			return fmt.Errorf("text does not accept attributes")
		}
	case "hardBreak":
		if len(n.Content) > 0 || len(n.Text) > 0 {
			return fmt.Errorf("hardBreak cannot contain content")
		}
		if len(n.Attrs) > 0 {
			return fmt.Errorf("hardBreak does not accept attributes")
		}
	}
	for _, child := range n.Content {
		if err := validateRichNode(child, false); err != nil {
			return err
		}
	}
	return nil
}

// RichTextPlainText derives the search/model plain text from a supported document. It preserves
// paragraph boundaries and marks list items, but never renders HTML or formatting instructions to
// the model. An invalid document is refused instead of producing misleading text.
func RichTextPlainText(raw json.RawMessage) (string, error) {
	if !hasRichTextDocument(raw) {
		return "", nil
	}
	if err := validateRichTextDocument(raw); err != nil {
		return "", err
	}
	var root richNode
	if err := json.Unmarshal(raw, &root); err != nil {
		return "", err
	}
	var out strings.Builder
	writeRichText(&out, root, 0)
	return strings.TrimSpace(out.String()), nil
}

func writeRichText(out *strings.Builder, n richNode, depth int) {
	switch n.Type {
	case "doc":
		for _, child := range n.Content {
			writeRichText(out, child, depth)
		}
	case "paragraph", "heading":
		out.WriteString(inlineText(n))
		out.WriteString("\n\n")
	case "bulletList", "orderedList":
		for _, child := range n.Content {
			if child.Type == "listItem" {
				writeListItem(out, child, depth)
			}
		}
		out.WriteString("\n")
	case "listItem":
		writeListItem(out, n, depth)
	case "text":
		out.WriteString(n.Text)
	case "hardBreak":
		out.WriteString("\n")
	}
}

func writeListItem(out *strings.Builder, n richNode, depth int) {
	indent := strings.Repeat("  ", depth)
	wrote := false
	for _, child := range n.Content {
		switch child.Type {
		case "paragraph", "heading":
			text := inlineText(child)
			if !wrote {
				out.WriteString(indent)
				out.WriteString("- ")
				out.WriteString(text)
				wrote = true
			} else {
				out.WriteString(indent)
				out.WriteString("  ")
				out.WriteString(text)
			}
			out.WriteString("\n")
		case "bulletList", "orderedList":
			writeRichText(out, child, depth+1)
			wrote = true
		case "hardBreak":
			out.WriteString("\n")
		case "text":
			if !wrote {
				out.WriteString(indent)
				out.WriteString("- ")
				wrote = true
			}
			out.WriteString(child.Text)
			out.WriteString("\n")
		}
	}
	if !wrote {
		out.WriteString(indent)
		out.WriteString("-\n")
	}
}

// inlineText preserves inline text and hard line breaks. It never concatenates separate block
// children; block boundaries are written by writeRichText.
func inlineText(n richNode) string {
	var out strings.Builder
	for _, child := range n.Content {
		switch child.Type {
		case "text":
			out.WriteString(child.Text)
		case "hardBreak":
			out.WriteString("\n")
		default:
			out.WriteString(inlineText(child))
		}
	}
	return out.String()
}
