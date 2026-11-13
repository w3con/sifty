// Package background — helpers for form inspection.
// See assessment.go for package-level documentation.
package background

import (
	"encoding/json"
	"net/url"

	"github.com/w3con/sifty/internal/inspect"
	"github.com/w3con/sifty/internal/store"
)

// inspectIntoDraft writes an inspected questionnaire into the job's application draft,
// merging new fields with any existing draft fields and handling blocked-access cases
// gracefully. This mirrors TypeScript inspectIntoDraft in application-store.ts and must
// stay faithful to it.
func inspectIntoDraft(s *store.Store, jobID int64, q inspect.Questionnaire) error {
	draft, err := s.GetDraft(jobID)
	if err != nil {
		return err
	}

	// Round-trip questionnaire through JSON to get a plain map[string]any for storage.
	qJSON, _ := json.Marshal(q)
	var qAny any
	_ = json.Unmarshal(qJSON, &qAny)

	// Blocked access: no fields and status signals the form was not accessible.
	if len(q.Fields) == 0 && isAccessBlocked(q.Status) {
		_ = s.SaveQuestionnaireSnapshot(jobID, qAny)
		draft.Questionnaire = qAny
		draft.Notes = append(draft.Notes, "Inspection could not read the questionnaire; existing fields and answers retained.")
		_, err = s.WriteDraft(draft, "Inspection access failure; previous draft retained")
		return err
	}

	// Build lookup of current draft fields by id.
	current := make(map[string]map[string]any)
	for _, f := range draft.Fields {
		if fm, ok := f.(map[string]any); ok {
			if id, ok := fm["id"].(string); ok {
				current[id] = fm
			}
		}
	}

	// Merge snapshot fields into the draft, preserving existing answers.
	var fields []any
	for _, rawQ := range q.Fields {
		qFieldJSON, _ := json.Marshal(rawQ)
		var qmap map[string]any
		_ = json.Unmarshal(qFieldJSON, &qmap)

		id, _ := qmap["id"].(string)
		if old, exists := current[id]; exists {
			// Keep existing answer fields; update question metadata from the new snapshot.
			merged := make(map[string]any, len(old)+len(qmap))
			for k, v := range old {
				merged[k] = v
			}
			for k, v := range qmap {
				merged[k] = v
			}
			oldSig, _ := old["signature"].(string)
			newSig, _ := qmap["signature"].(string)
			oldChanged, _ := old["changed"].(bool)
			merged["changed"] = (oldSig != newSig) || oldChanged
			merged["removed"] = false
			fields = append(fields, merged)
		} else {
			// New field: initialise empty answer.
			var value any = ""
			if t, _ := qmap["type"].(string); t == "checkbox" {
				value = []any{}
			}
			qmap["value"] = value
			qmap["revision"] = 0.0
			qmap["edited"] = false
			qmap["unknowns"] = []any{}
			qmap["evidence"] = []any{}
			fields = append(fields, qmap)
		}
	}

	// Retain old fields absent from the new snapshot: manual questions stay unchanged;
	// site questions are kept but marked removed so the UI can surface the gap.
	newIDs := make(map[string]bool, len(q.Fields))
	for _, f := range q.Fields {
		newIDs[f.ID] = true
	}
	for _, f := range draft.Fields {
		fm, ok := f.(map[string]any)
		if !ok {
			continue
		}
		id, _ := fm["id"].(string)
		if newIDs[id] {
			continue
		}
		if origin, _ := fm["origin"].(string); origin == "manual" {
			fields = append(fields, fm)
		} else {
			removed := make(map[string]any, len(fm)+1)
			for k, v := range fm {
				removed[k] = v
			}
			removed["removed"] = true
			fields = append(fields, removed)
		}
	}

	if fields == nil {
		fields = []any{}
	}

	_ = s.SaveQuestionnaireSnapshot(jobID, qAny)
	draft.Questionnaire = qAny
	draft.Fields = fields
	_, err = s.WriteDraft(draft, "Questionnaire inspected; existing answers preserved")
	return err
}

// isAccessBlocked reports whether the questionnaire status indicates the form was not
// accessible rather than successfully read with no answerable questions.
func isAccessBlocked(status string) bool {
	return status == "blocked" || status == "login-required" || status == "unsupported" || status == "backed-off"
}

// externalApplyURL returns the first valid HTTP(S) URL from applyURL (preferred) or
// originalURL (fallback), matching TypeScript's externalApplyUrl in workflow.ts.
func externalApplyURL(applyURL *string, originalURL string) string {
	candidates := []string{originalURL}
	if applyURL != nil {
		candidates = []string{*applyURL, originalURL}
	}
	for _, s := range candidates {
		if isHTTPURL(s) {
			return s
		}
	}
	return ""
}

// isHTTPURL reports whether s is an http(s) URL with no embedded credentials.
func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.User == nil
}
