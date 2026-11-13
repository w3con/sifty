package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// ApplicationDraft is the in-memory representation of a role's Stage 2 draft.
// The JSON field names match the TypeScript ApplicationDraft type.
type ApplicationDraft struct {
	JobID         int64          `json:"jobId"`
	Revision      int            `json:"revision"`
	Fields        []any          `json:"fields"`
	SelectedFile  string         `json:"selectedFile"`
	Questionnaire any            `json:"questionnaire"`
	Provenance    any            `json:"provenance"`
	Notes         []string       `json:"notes"`
}

// GetDraft returns the current application draft for a role, or an empty draft if none exists.
func (s *Store) GetDraft(jobID int64) (*ApplicationDraft, error) {
	job, err := s.getW(jobID)
	if err != nil {
		return nil, err
	}
	var data string
	err = s.DB.QueryRow(
		"SELECT data FROM application_drafts WHERE job_id=?", job.ID).Scan(&data)
	if err == sql.ErrNoRows {
		return &ApplicationDraft{
			JobID:        job.ID,
			Revision:     0,
			Fields:       []any{},
			SelectedFile: "profile.pdf",
			Notes:        []string{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get draft for job %d: %w", job.ID, err)
	}
	var d ApplicationDraft
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return nil, fmt.Errorf("parse draft: %w", err)
	}
	return &d, nil
}

// WriteDraft saves a new revision of the draft and appends to application_revisions.
func (s *Store) WriteDraft(draft *ApplicationDraft, reason string) (*ApplicationDraft, error) {
	if _, err := s.getW(draft.JobID); err != nil {
		return nil, err
	}
	draft.Revision++
	if draft.Fields == nil {
		draft.Fields = []any{}
	}
	if draft.Notes == nil {
		draft.Notes = []string{}
	}
	data, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft: %w", err)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.Exec(
		"INSERT INTO application_drafts VALUES(?,?) ON CONFLICT(job_id) DO UPDATE SET data=excluded.data",
		draft.JobID, string(data)); err != nil {
		return nil, fmt.Errorf("write draft: %w", err)
	}
	if _, err := tx.Exec(
		"INSERT INTO application_revisions(job_id,created,reason,data) VALUES(?,?,?,?)",
		draft.JobID, now(), reason, string(data)); err != nil {
		return nil, fmt.Errorf("write revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return draft, nil
}

// AddQuestion appends a manual question to the draft and increments the revision.
func (s *Store) AddQuestion(jobID int64, label, qtype string, options []string) (*ApplicationDraft, error) {
	draft, err := s.GetDraft(jobID)
	if err != nil {
		return nil, err
	}
	if qtype == "" {
		qtype = "textarea"
	}
	if options == nil {
		options = []string{}
	}
	q := map[string]any{
		"id":          hashStr(fmt.Sprintf("manual:%d:%s", jobID, label)),
		"label":       label,
		"type":        qtype,
		"options":     options,
		"required":    false,
		"maxLength":   nil,
		"accept":      "",
		"sourceUrl":   "",
		"origin":      "manual",
		"language":    "und",
		"signature":   "",
		"controlName": "",
		// AnswerField additions
		"value":    "",
		"revision": 0,
		"edited":   false,
		"unknowns": []any{},
		"evidence": []any{},
	}
	draft.Fields = append(draft.Fields, q)
	return s.WriteDraft(draft, "Manual question added")
}

// PreparationOperations returns the preparation operation records for a role.
func (s *Store) PreparationOperations(jobID int64) ([]map[string]any, error) {
	rows, err := s.DB.Query(`
		SELECT id, job_id, status, created, updated, pid, request, error, progress
		FROM preparation_operations WHERE job_id=? ORDER BY id DESC LIMIT 20`, jobID)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		var id, jid int64
		var status, created, updated, request, progress string
		var pid sql.NullInt64
		var errStr sql.NullString
		if err := rows.Scan(&id, &jid, &status, &created, &updated, &pid, &request, &errStr, &progress); err != nil {
			rows.Close()
			return nil, err
		}
		m := map[string]any{
			"id": id, "jobId": jid, "status": status,
			"created": created, "updated": updated, "request": request,
		}
		var reqObj any
		_ = json.Unmarshal([]byte(request), &reqObj)
		m["request"] = reqObj
		m["progress"] = progress // plain string, not JSON-decoded
		if pid.Valid {
			m["pid"] = pid.Int64
		}
		if errStr.Valid {
			m["error"] = errStr.String
		}
		out = append(out, m)
	}
	rows.Close()
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

// QueuePreparation inserts a new preparation operation in 'queued' status. If an operation for
// this job is already queued or running, it is returned unchanged. The returned row is the
// operation as it exists in the database, with all columns, in the shape used by the API.
func (s *Store) QueuePreparation(jobID int64, request any) (map[string]any, error) {
	if _, err := s.getW(jobID); err != nil {
		return nil, err
	}
	// Return any already-active operation.
	existing := s.activePreparation(jobID)
	if existing != nil {
		return existing, nil
	}
	reqJSON, _ := json.Marshal(request)
	result, err := s.DB.Exec(`
		INSERT INTO preparation_operations(job_id,status,created,updated,request,progress)
		VALUES(?,'queued',?,?,?,?)`,
		jobID, now(), now(), string(reqJSON), "Waiting for preparation worker")
	if err != nil {
		return nil, fmt.Errorf("queue preparation: %w", err)
	}
	id, _ := result.LastInsertId()
	return s.preparationRow(id)
}

// activePreparation returns the first queued or running preparation operation for a job, or nil.
func (s *Store) activePreparation(jobID int64) map[string]any {
	var id int64
	err := s.DB.QueryRow(`SELECT id FROM preparation_operations WHERE job_id=? AND status IN ('queued','running') ORDER BY id LIMIT 1`, jobID).Scan(&id)
	if err != nil {
		return nil
	}
	row, _ := s.preparationRow(id)
	return row
}

// preparationRow fetches a single preparation_operations row by id and returns it as a map.
func (s *Store) preparationRow(id int64) (map[string]any, error) {
	var jid int64
	var status, created, updated, request, progress string
	var pid sql.NullInt64
	var errStr sql.NullString
	err := s.DB.QueryRow(`
		SELECT id, job_id, status, created, updated, pid, request, error, progress
		FROM preparation_operations WHERE id=?`, id).Scan(
		&id, &jid, &status, &created, &updated, &pid, &request, &errStr, &progress)
	if err != nil {
		return nil, fmt.Errorf("fetch preparation row %d: %w", id, err)
	}
	m := map[string]any{
		"id": id, "jobId": jid, "status": status,
		"created": created, "updated": updated,
	}
	var reqObj any
	_ = json.Unmarshal([]byte(request), &reqObj)
	m["request"] = reqObj
	m["progress"] = progress // plain string, not JSON-decoded
	if pid.Valid {
		m["pid"] = pid.Int64
	}
	if errStr.Valid && errStr.String != "" {
		m["error"] = errStr.String
	}
	return m, nil
}

// StartPreparation inserts a new preparation operation record in 'running' status.
// Deprecated: prefer QueuePreparation + PreparationWorker. Kept for any direct callers.
func (s *Store) StartPreparation(jobID int64, pid int, request any) (int64, error) {
	reqJSON, _ := json.Marshal(request)
	result, err := s.DB.Exec(`
		INSERT INTO preparation_operations(job_id,status,created,updated,pid,request,progress)
		VALUES(?,?,?,?,?,?,?)`,
		jobID, "running", now(), now(), pid, string(reqJSON), `{}`)
	if err != nil {
		return 0, fmt.Errorf("start preparation: %w", err)
	}
	return result.LastInsertId()
}

// FinishPreparation marks a preparation operation as completed or failed.
func (s *Store) FinishPreparation(opID int64, status, errMsg string, progress any) error {
	progJSON, _ := json.Marshal(progress)
	var errArg any
	if errMsg != "" {
		errArg = errMsg
	}
	_, err := s.DB.Exec(`
		UPDATE preparation_operations
		SET status=?, updated=?, error=?, progress=?, pid=NULL WHERE id=?`,
		status, now(), errArg, string(progJSON), opID)
	return err
}

// FeedbackRows returns feedback records indexed in the database for a role.
func (s *Store) FeedbackRows(jobID int64) ([]map[string]any, error) {
	rows, err := s.DB.Query(
		"SELECT id, job_id, created, path, data FROM feedback WHERE job_id=? ORDER BY created", jobID)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		var id, path, data string
		var jid int64
		var created string
		if err := rows.Scan(&id, &jid, &created, &path, &data); err != nil {
			rows.Close()
			return nil, err
		}
		var rec map[string]any
		_ = json.Unmarshal([]byte(data), &rec)
		if rec == nil {
			rec = map[string]any{}
		}
		rec["id"] = id
		rec["jobId"] = jid
		rec["created"] = created
		rec["path"] = path
		out = append(out, rec)
	}
	rows.Close()
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

// IndexFeedback inserts a feedback record into the database index.
// The caller writes the durable file first; this is the second write.
func (s *Store) IndexFeedback(id string, jobID int64, path string, data any) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(
		"INSERT INTO feedback VALUES(?,?,?,?,?)", id, jobID, now(), path, string(dataJSON))
	return err
}

// SaveQuestionnaireSnapshot saves an inspected questionnaire snapshot.
func (s *Store) SaveQuestionnaireSnapshot(jobID int64, data any) error {
	dataJSON, _ := json.Marshal(data)
	_, err := s.DB.Exec(
		"INSERT INTO questionnaire_snapshots(job_id,created,data) VALUES(?,?,?)",
		jobID, now(), string(dataJSON))
	return err
}
