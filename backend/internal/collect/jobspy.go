package collect

import (
	"context"

	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/worker"
)

// JobspyParams is the request the core sends to the jobspy worker.
type JobspyParams struct {
	Records []map[string]any `json:"records,omitempty"`
	Sites   []string         `json:"sites,omitempty"`
	Query   string           `json:"query,omitempty"`
}

// CollectViaJobspy runs the jobspy scraping worker as a sidecar and returns the postings it
// yields, already normalised to the core's posting shape. Network scraping lives entirely in the
// worker; the core only invokes it and decodes the result.
func CollectViaJobspy(ctx context.Context, w worker.Worker, params JobspyParams) ([]model.JobInput, error) {
	var out []model.JobInput
	if err := w.Call(ctx, "collect", params, &out); err != nil {
		return nil, err
	}
	return out, nil
}
