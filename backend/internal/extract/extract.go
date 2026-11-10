// Package extract reads text from candidate input documents. PDF text goes through the system
// pdftotext binary (the same reader the candidate context already used), Markdown/plain text is
// read directly, and DOCX is delegated to a caller-supplied reader so this package carries no
// document-format implementation of its own.
package extract

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DocxReader extracts visible text from a .docx file. The profile document service supplies a
// reader backed by the existing Node worker runtime; tests may supply a stub.
type DocxReader func(ctx context.Context, path string) (string, error)

// Kind returns the lower-case file extension without its leading dot.
func Kind(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}

// PlainText reads PDF and plain-text/Markdown files. The caller owns the context timeout.
func PlainText(ctx context.Context, path string) (string, error) {
	switch Kind(path) {
	case "pdf":
		out, err := exec.CommandContext(ctx, "pdftotext", "-layout", path, "-").Output()
		if err != nil {
			return "", fmt.Errorf("pdftotext %s: %w", path, err)
		}
		return string(out), nil
	case "md", "markdown", "txt", "text":
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported document type %q", Kind(path))
	}
}

// Text reads any supported document. docx may be nil when DOCX input is not enabled.
func Text(ctx context.Context, path string, docx DocxReader) (string, error) {
	if Kind(path) == "docx" {
		if docx == nil {
			return "", fmt.Errorf("DOCX reader is not configured")
		}
		return docx(ctx, path)
	}
	return PlainText(ctx, path)
}
