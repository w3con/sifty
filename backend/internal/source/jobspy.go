package source

import (
	"context"
	"fmt"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/worker"
)

// JobspySource implements Source by delegating to the jobspy worker subprocess.
// Discover invokes the worker with the configured query and sites and returns
// each result as a complete Ref (the worker returns full postings, so Resolve
// returns the partial without a network fetch). listingURL is used only to
// derive the source label; jobspy does its own network access.
type JobspySource struct {
	w      worker.Worker
	params collect.JobspyParams
}

// NewJobspySource creates a JobspySource. params.Sites and params.Query configure
// which boards jobspy targets.
func NewJobspySource(w worker.Worker, params collect.JobspyParams) *JobspySource {
	return &JobspySource{w: w, params: params}
}

// Discover calls the jobspy worker and returns all postings as complete Refs.
// listingURL is ignored; jobspy uses its own configured sites.
func (j *JobspySource) Discover(ctx context.Context, _ string) ([]Ref, error) {
	jobs, err := collect.CollectViaJobspy(ctx, j.w, j.params)
	if err != nil {
		return nil, fmt.Errorf("jobspy discover: %w", err)
	}
	refs := make([]Ref, 0, len(jobs))
	for _, job := range jobs {
		refs = append(refs, Ref{
			URL:     job.OriginalURL,
			Partial: job,
		})
	}
	return refs, nil
}

// Resolve returns ref.Partial without a network fetch. The jobspy worker returns
// full postings, so the detail is already present in the Ref.
func (j *JobspySource) Resolve(_ context.Context, ref Ref) (model.JobInput, error) {
	return ref.Partial, nil
}
