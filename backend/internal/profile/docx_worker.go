package profile

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/w3con/sifty/internal/extract"
	"github.com/w3con/sifty/internal/worker"
)

// NewNodeDocxReader returns a DOCX text reader backed by the existing Node worker runtime. The
// worker module (workers/documents) owns the Mammoth dependency and returns only extracted text;
// no document rendering or HTML conversion is used.
func NewNodeDocxReader(workersDir string) extract.DocxReader {
	script := filepath.Join(workersDir, "documents", "index.mjs")
	dir := filepath.Dir(workersDir)
	return func(ctx context.Context, path string) (string, error) {
		w := worker.Worker{Command: "node", Args: []string{script}, Dir: dir}
		var out struct {
			Text string `json:"text"`
		}
		if err := w.Call(ctx, "extractDocx", map[string]string{"path": path}, &out); err != nil {
			return "", fmt.Errorf("DOCX reader: %w", err)
		}
		return out.Text, nil
	}
}
