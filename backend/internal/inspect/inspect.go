// Package inspect is the Go core's client for Stage 2 form inspection. The core has no browser
// binding, so it invokes the TypeScript Playwright worker (workers/playwright) over the worker
// protocol and decodes the questionnaire the worker returns. There is no browser-automation code in
// this package or anywhere else in Go: every rendered-DOM read happens in the worker.
package inspect

import (
	"context"

	"github.com/w3con/sifty/internal/worker"
)

// Question is one inspected questionnaire control, matching the question shape the current
// application stores.
type Question struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Instructions string   `json:"instructions,omitempty"`
	LabelStatus  string   `json:"labelStatus,omitempty"`
	Type         string   `json:"type"`
	Options      []string `json:"options"`
	Required     bool     `json:"required"`
	MaxLength    *int     `json:"maxLength"`
	Accept       string   `json:"accept"`
	SourceURL    string   `json:"sourceUrl"`
	Origin       string   `json:"origin"`
	Language     string   `json:"language"`
	Signature    string   `json:"signature"`
	ControlName  string   `json:"controlName"`
}

// Questionnaire is the result of inspecting an application form.
type Questionnaire struct {
	URL        string     `json:"url"`
	CapturedAt string     `json:"capturedAt"`
	Language   string     `json:"language"`
	Status     string     `json:"status"`
	Gaps       []string   `json:"gaps"`
	Fields     []Question `json:"fields"`
	Family     string     `json:"family"`
}

// Params selects what the worker inspects: a live URL to navigate, or an HTML document to inspect
// directly. Exactly one of URL or HTML is set.
//
// ATSFamily may be set to "ashby", "greenhouse", or "lever" to override hostname-based ATS
// detection in the worker. It is intended for test fixtures served as inline HTML that have no
// real hostname.
type Params struct {
	URL       string `json:"url,omitempty"`
	HTML      string `json:"html,omitempty"`
	ATSFamily string `json:"atsFamily,omitempty"`
}

// InspectForm asks the Playwright worker to inspect an application form and returns the questionnaire
// it reads. All browser automation happens in the worker; this call only exchanges JSON with it.
func InspectForm(ctx context.Context, w worker.Worker, params Params) (Questionnaire, error) {
	var q Questionnaire
	if err := w.Call(ctx, "inspectForm", params, &q); err != nil {
		return Questionnaire{}, err
	}
	return q, nil
}
