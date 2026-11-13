package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Assessment records a successful fit-assessment result for a role.
func (s *Store) Assessment(id int64, data any) error {
	job, err := s.getW(id)
	if err != nil {
		return err
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal assessment: %w", err)
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	t := now()
	if _, err := tx.Exec(
		"INSERT INTO assessments(job_id,created,data) VALUES(?,?,?)",
		job.ID, t, string(dataJSON)); err != nil {
		return fmt.Errorf("insert assessment: %w", err)
	}
	if _, err := tx.Exec(
		"UPDATE jobs SET assessment=?, assessment_error=NULL WHERE id=?",
		string(dataJSON), job.ID); err != nil {
		return fmt.Errorf("update job assessment: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil
	return nil
}

// AssessmentFailure records a failed assessment attempt.
func (s *Store) AssessmentFailure(id int64, message string, metadata map[string]any) error {
	job, err := s.getW(id)
	if err != nil {
		return err
	}
	payload := map[string]any{"status": "failed", "error": message}
	for k, v := range metadata {
		payload[k] = v
	}
	dataJSON, _ := json.Marshal(payload)
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.Exec(
		"INSERT INTO assessments(job_id,created,data) VALUES(?,?,?)",
		job.ID, now(), string(dataJSON)); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"UPDATE jobs SET assessment_error=? WHERE id=?", message, job.ID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil
	return nil
}

// Context returns the most recent candidate context snapshot, or nil if none.
func (s *Store) Context() (map[string]any, error) {
	var data string
	err := s.Read.QueryRow(
		"SELECT data FROM context_refreshes ORDER BY id DESC LIMIT 1").Scan(&data)
	if err == sql.ErrNoRows {
		err = s.Read.QueryRow(
			"SELECT data FROM contexts ORDER BY created DESC LIMIT 1").Scan(&data)
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query context: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil, fmt.Errorf("parse context: %w", err)
	}
	return out, nil
}

// SaveContext persists a new candidate context version.
func (s *Store) SaveContext(c map[string]any) error {
	version, _ := c["version"].(string)
	if version == "" {
		return fmt.Errorf("context must have a version field")
	}
	dataJSON, err := json.Marshal(c)
	if err != nil {
		return err
	}
	t := now()
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.Exec(
		"INSERT OR IGNORE INTO contexts VALUES(?,?,?)", version, t, string(dataJSON)); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"INSERT INTO context_refreshes(version,created,data) VALUES(?,?,?)",
		version, t, string(dataJSON)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil
	return nil
}

// QueueRow represents one row in the assessment_queue table.
type QueueRow struct {
	JobID            int64   `json:"jobId"`
	Fingerprint      string  `json:"fingerprint"`
	Provider         string  `json:"provider"`
	Status           string  `json:"status"`
	Priority         int     `json:"priority"`
	Attempts         int     `json:"attempts"`
	NextAt           int64   `json:"nextAt"`
	Started          *int64  `json:"started"`
	Finished         *int64  `json:"finished"`
	Error            *string `json:"error"`
	Owner            *int    `json:"owner"`
	OverrideProvider *string `json:"overrideProvider"`
	Routing          *string `json:"routing"`
}

// EnqueueJob inserts or updates an assessment queue row for the given role.
func (s *Store) EnqueueJob(jobID int64, fingerprint, provider string, priority int) (*QueueRow, error) {
	_, err := s.getW(jobID)
	if err != nil {
		return nil, err
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
	_, err = tx.Exec(`
		INSERT INTO assessment_queue(job_id,fingerprint,provider,status,priority,attempts,next_at)
		VALUES(?,?,?,'pending',?,0,0)
		ON CONFLICT(job_id) DO UPDATE SET
		  fingerprint=excluded.fingerprint, provider=excluded.provider,
		  status='pending', priority=max(priority,excluded.priority), next_at=0`,
		jobID, fingerprint, provider, priority)
	if err != nil {
		return nil, fmt.Errorf("upsert queue row: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return s.QueueRow(jobID)
}

// QueueRow returns the assessment queue row for a role.
func (s *Store) QueueRow(jobID int64) (*QueueRow, error) {
	r := &QueueRow{}
	var started, finished sql.NullInt64
	var owner sql.NullInt64
	var errStr, overrideProv, routing sql.NullString
	err := s.DB.QueryRow(`
		SELECT job_id,fingerprint,provider,status,priority,attempts,next_at,
		       started,finished,error,owner,override_provider,routing
		FROM assessment_queue WHERE job_id=?`, jobID).Scan(
		&r.JobID, &r.Fingerprint, &r.Provider, &r.Status, &r.Priority,
		&r.Attempts, &r.NextAt, &started, &finished, &errStr, &owner,
		&overrideProv, &routing)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("queue row for job %d: %w", jobID, err)
	}
	if started.Valid {
		r.Started = &started.Int64
	}
	if finished.Valid {
		r.Finished = &finished.Int64
	}
	if errStr.Valid {
		r.Error = &errStr.String
	}
	if owner.Valid {
		v := int(owner.Int64)
		r.Owner = &v
	}
	if overrideProv.Valid {
		r.OverrideProvider = &overrideProv.String
	}
	if routing.Valid {
		r.Routing = &routing.String
	}
	return r, nil
}

// QueueState returns the queue-wide enable flag and pause state from queue_state.
type QueueState struct {
	Enabled bool `json:"enabled"`
	Paused  bool `json:"paused"`
}

// GetQueueState reads the queue enabled/paused flags through the read pool.
func (s *Store) GetQueueState() (QueueState, error) {
	return queueStateOn(s.Read)
}

func queueStateOn(q queryer) (QueueState, error) {
	state := QueueState{Enabled: true}
	var data string
	if err := q.QueryRow("SELECT data FROM queue_state WHERE key='config'").Scan(&data); err == nil {
		var m map[string]any
		if err := json.Unmarshal([]byte(data), &m); err == nil {
			if v, ok := m["enabled"].(bool); ok {
				state.Enabled = v
			}
			if v, ok := m["paused"].(bool); ok {
				state.Paused = v
			}
		}
	}
	return state, nil
}

// GetQueueRuntime returns the queue_state 'runtime' row in the shape the frontend
// expects — {"data": "<raw json string>"} — or nil when the row is absent. The
// frontend parses runtime.data itself, matching the TypeScript queueStatus, so the
// stored string is returned verbatim and never decoded here.
func (s *Store) GetQueueRuntime() (map[string]any, error) {
	var data string
	err := s.Read.QueryRow("SELECT data FROM queue_state WHERE key='runtime'").Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": data}, nil
}

// SetQueueEnabled enables or disables background assessment.
func (s *Store) SetQueueEnabled(enabled bool) error {
	state, err := queueStateOn(s.DB) // the value written back is built from what the writer sees
	if err != nil {
		return err
	}
	state.Enabled = enabled
	data, _ := json.Marshal(state)
	_, err = s.DB.Exec(
		"INSERT INTO queue_state VALUES('config',?) ON CONFLICT(key) DO UPDATE SET data=excluded.data",
		string(data))
	return err
}

// QueueCounts returns the per-status counts for the assessment queue.
func (s *Store) QueueCounts() (map[string]int, error) {
	rows, err := s.Read.Query(
		"SELECT status, COUNT(*) FROM assessment_queue GROUP BY status")
	if err != nil {
		return nil, err
	}
	counts := map[string]int{
		"running": 0, "pending": 0, "failed": 0,
		"done": 0, "missing-description": 0,
	}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			rows.Close()
			return nil, err
		}
		counts[status] = n
	}
	rows.Close()
	return counts, nil
}

// AllQueueRows returns all non-done assessment queue rows.
func (s *Store) AllQueueRows() ([]*QueueRow, error) {
	rows, err := s.Read.Query(`
		SELECT job_id,fingerprint,provider,status,priority,attempts,next_at,
		       started,finished,error,owner,override_provider,routing
		FROM assessment_queue WHERE status != 'done' ORDER BY priority DESC, job_id`)
	if err != nil {
		return nil, err
	}
	var out []*QueueRow
	for rows.Next() {
		r := &QueueRow{}
		var started, finished sql.NullInt64
		var owner sql.NullInt64
		var errStr, overrideProv, routing sql.NullString
		if err := rows.Scan(
			&r.JobID, &r.Fingerprint, &r.Provider, &r.Status, &r.Priority,
			&r.Attempts, &r.NextAt, &started, &finished, &errStr, &owner,
			&overrideProv, &routing); err != nil {
			rows.Close()
			return nil, err
		}
		if started.Valid {
			r.Started = &started.Int64
		}
		if finished.Valid {
			r.Finished = &finished.Int64
		}
		if errStr.Valid {
			r.Error = &errStr.String
		}
		if owner.Valid {
			v := int(owner.Int64)
			r.Owner = &v
		}
		if overrideProv.Valid {
			r.OverrideProvider = &overrideProv.String
		}
		if routing.Valid {
			r.Routing = &routing.String
		}
		out = append(out, r)
	}
	rows.Close()
	return out, nil
}

// WorkflowHistory returns all workflow transitions for a role.
func (s *Store) WorkflowHistory(jobID int64) ([]map[string]any, error) {
	rows, err := s.Read.Query(`
		SELECT id, job_id, created, from_stage, to_stage, confirmed
		FROM workflow_history WHERE job_id=? ORDER BY id`, jobID)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		var id, jid int64
		var created, fromStage, toStage string
		var confirmed int
		if err := rows.Scan(&id, &jid, &created, &fromStage, &toStage, &confirmed); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "jobId": jid, "created": created,
			"fromStage": fromStage, "toStage": toStage,
			"confirmed": confirmed != 0,
		})
	}
	rows.Close()
	return out, nil
}

// Lock acquires a named cross-process lock. If the lock is held by a dead process,
// it is reclaimed. Returns an error if the lock is held by a live process.
func (s *Store) AcquireLock(name string, pid int) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var holderPID int
	err = tx.QueryRow("SELECT pid FROM locks WHERE name=?", name).Scan(&holderPID)
	if err == nil && isProcessAlive(holderPID) {
		_ = tx.Rollback()
		tx = nil
		return fmt.Errorf("lock %q is held by process %d", name, holderPID)
	}
	_, err = tx.Exec(
		"INSERT INTO locks VALUES(?,?,?) ON CONFLICT(name) DO UPDATE SET pid=excluded.pid,acquired=excluded.acquired",
		name, pid, now())
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil
	return nil
}

// ReleaseLock releases a named lock if held by pid.
func (s *Store) ReleaseLock(name string, pid int) error {
	_, err := s.DB.Exec("DELETE FROM locks WHERE name=? AND pid=?", name, pid)
	return err
}

// CollectionRun records the start of a collection run.
func (s *Store) StartCollectionRun(pid int, mode, configHash string) (int64, error) {
	result, err := s.DB.Exec(
		"INSERT INTO collection_runs(started,status,pid,mode,config_hash) VALUES(?,?,?,?,?)",
		now(), "running", pid, mode, configHash)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// FinishCollectionRun marks a collection run as finished.
func (s *Store) FinishCollectionRun(runID int64, status string) error {
	_, err := s.DB.Exec(
		"UPDATE collection_runs SET finished=?, status=? WHERE id=?",
		now(), status, runID)
	return err
}

// CollectionRunByID returns the current state of a collection run.
func (s *Store) CollectionRunByID(runID int64) (map[string]any, error) {
	var id int64
	var started string
	var finished sql.NullString
	var status, mode, configHash string
	var pid int
	err := s.Read.QueryRow(`
		SELECT id, started, finished, status, pid, mode, config_hash
		FROM collection_runs WHERE id=?`, runID).
		Scan(&id, &started, &finished, &status, &pid, &mode, &configHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m := map[string]any{
		"id": id, "started": started, "status": status,
		"pid": pid, "mode": mode, "configHash": configHash,
	}
	if finished.Valid {
		m["finished"] = finished.String
	}
	return m, nil
}

// LatestCollectionRun returns the most recent collection run across both on-demand/auto runs
// (collection_runs) and daily scheduler runs (runs), together with source runs and events where
// available. Daily runs carry mode="daily" and no source or event detail.
//
// It reads through the read pool. A caller that decides a write from the answer (the collection
// cooldown) uses LatestCollectionRunOnWriter.
func (s *Store) LatestCollectionRun() (map[string]any, error) {
	return latestCollectionRunOn(s.Read)
}

// LatestCollectionRunOnWriter is LatestCollectionRun read on the write connection, for the
// collection cooldown gate that follows taking the cross-process lock on that same connection.
func (s *Store) LatestCollectionRunOnWriter() (map[string]any, error) {
	return latestCollectionRunOn(s.DB)
}

func latestCollectionRunOn(q queryer) (map[string]any, error) {
	var origin string
	var id int64
	var started string
	var finished sql.NullString
	var status, mode, configHash string
	var pid int
	err := q.QueryRow(`
		SELECT origin, id, started, finished, status, pid, mode, config_hash FROM (
		  SELECT 'collection' AS origin, id, started, finished, status, pid, mode, config_hash
		  FROM collection_runs
		  UNION ALL
		  SELECT 'daily' AS origin, id, started, finished, status, 0, 'daily', ''
		  FROM runs
		) ORDER BY started DESC LIMIT 1`).
		Scan(&origin, &id, &started, &finished, &status, &pid, &mode, &configHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest run: %w", err)
	}
	run := map[string]any{
		"id": id, "started": started, "status": status,
		"pid": pid, "mode": mode, "configHash": configHash,
	}
	if finished.Valid {
		run["finished"] = finished.String
	}

	// Source runs and events are only available for collection_runs rows.
	// For daily runs (origin == "daily") we set empty slices so callers never see nil.
	if origin == "collection" {
		sRows, err := q.Query(
			"SELECT source, status, coverage FROM source_runs WHERE run_id=? ORDER BY source", id)
		if err == nil {
			var sources []map[string]any
			for sRows.Next() {
				var src, stat, cov string
				if err := sRows.Scan(&src, &stat, &cov); err == nil {
					var covObj any
					_ = json.Unmarshal([]byte(cov), &covObj)
					sources = append(sources, map[string]any{
						"source": src, "status": stat, "coverage": covObj,
					})
				}
			}
			sRows.Close()
			if sources == nil {
				sources = []map[string]any{}
			}
			run["sources"] = sources
		}

		eRows, err := q.Query(
			"SELECT created, source, message FROM collection_events WHERE run_id=? ORDER BY id", id)
		if err == nil {
			var events []map[string]any
			for eRows.Next() {
				var created, message string
				var source sql.NullString
				if err := eRows.Scan(&created, &source, &message); err == nil {
					e := map[string]any{"created": created, "message": message}
					if source.Valid {
						e["source"] = source.String
					}
					events = append(events, e)
				}
			}
			eRows.Close()
			if events == nil {
				events = []map[string]any{}
			}
			run["events"] = events
		}
	} else {
		// Daily runs have no source or event detail rows; always return empty slices.
		run["sources"] = []map[string]any{}
		run["events"] = []map[string]any{}
	}

	return run, nil
}

// NextPendingQueueRow returns the highest-priority pending or failed row whose next_at is ≤ nowMs,
// or nil if none is ready.
func (s *Store) NextPendingQueueRow(nowMs int64) (*QueueRow, error) {
	r := &QueueRow{}
	var started, finished sql.NullInt64
	var owner sql.NullInt64
	var errStr, overrideProv, routing sql.NullString
	err := s.DB.QueryRow(`
		SELECT job_id,fingerprint,provider,status,priority,attempts,next_at,
		       started,finished,error,owner,override_provider,routing
		FROM assessment_queue
		WHERE status IN ('pending','failed') AND next_at<=?
		ORDER BY priority DESC, job_id LIMIT 1`, nowMs).Scan(
		&r.JobID, &r.Fingerprint, &r.Provider, &r.Status, &r.Priority,
		&r.Attempts, &r.NextAt, &started, &finished, &errStr, &owner,
		&overrideProv, &routing)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if started.Valid {
		v := started.Int64
		r.Started = &v
	}
	if finished.Valid {
		v := finished.Int64
		r.Finished = &v
	}
	if errStr.Valid {
		r.Error = &errStr.String
	}
	if owner.Valid {
		v := int(owner.Int64)
		r.Owner = &v
	}
	if overrideProv.Valid {
		r.OverrideProvider = &overrideProv.String
	}
	if routing.Valid {
		r.Routing = &routing.String
	}
	return r, nil
}

// MarkQueueRunning sets a queue row to 'running' and records the owning pid.
func (s *Store) MarkQueueRunning(jobID int64, pid int, nowMs int64) error {
	_, err := s.DB.Exec(`
		UPDATE assessment_queue SET status='running',started=?,finished=NULL,owner=?,attempts=attempts+1
		WHERE job_id=?`, nowMs, pid, jobID)
	return err
}

// MarkQueueDone marks a queue row as successfully completed.
func (s *Store) MarkQueueDone(jobID int64, nowMs int64) error {
	_, err := s.DB.Exec(`
		UPDATE assessment_queue SET status='done',finished=?,error=NULL,owner=NULL
		WHERE job_id=?`, nowMs, jobID)
	return err
}

// MarkQueueFailed records a failed assessment and schedules an exponential-backoff retry.
func (s *Store) MarkQueueFailed(jobID int64, errMsg string, attempts int, nowMs int64) error {
	delay := int64(30000)
	for i := 0; i < attempts && i < 10; i++ {
		delay *= 2
	}
	const maxDelay = 6 * 60 * 60 * 1000 // 6 h
	if delay > maxDelay {
		delay = maxDelay
	}
	_, err := s.DB.Exec(`
		UPDATE assessment_queue SET status='failed',finished=?,error=?,next_at=?,owner=NULL
		WHERE job_id=?`, nowMs, errMsg, nowMs+delay, jobID)
	return err
}

// RecoverInterruptedJobs transitions 'running' queue rows whose owning process is dead back to
// 'pending' with a short delay, matching the TypeScript worker's startup recovery.
func (s *Store) RecoverInterruptedJobs(nowMs int64) error {
	rows, err := s.DB.Query(`
		SELECT job_id, owner FROM assessment_queue WHERE status='running'`)
	if err != nil {
		return err
	}
	type stuckRow struct {
		jobID int64
		pid   int
	}
	var stuck []stuckRow
	for rows.Next() {
		var jobID int64
		var owner sql.NullInt64
		if err := rows.Scan(&jobID, &owner); err != nil {
			rows.Close()
			return err
		}
		if !owner.Valid || !isProcessAlive(int(owner.Int64)) {
			stuck = append(stuck, stuckRow{jobID, int(owner.Int64)})
		}
	}
	rows.Close()
	for _, r := range stuck {
		_, err := s.DB.Exec(`
			UPDATE assessment_queue SET status='pending',owner=NULL,error='Recovered interrupted inference',next_at=?
			WHERE job_id=?`, nowMs+5000, r.jobID)
		if err != nil {
			return err
		}
	}
	return nil
}

// StartDailyRun inserts a new row in the `runs` table with status 'running' and returns its ID.
func (s *Store) StartDailyRun() (int64, error) {
	result, err := s.DB.Exec("INSERT INTO runs(started,status) VALUES(?,'running')", now())
	if err != nil {
		return 0, fmt.Errorf("start daily run: %w", err)
	}
	return result.LastInsertId()
}

// FinishDailyRun marks a daily run as complete or failed with an optional detail payload.
func (s *Store) FinishDailyRun(id int64, status string, detail any) error {
	var detailStr string
	if detail != nil {
		b, _ := json.Marshal(detail)
		detailStr = string(b)
	}
	_, err := s.DB.Exec("UPDATE runs SET status=?,finished=?,detail=? WHERE id=?", status, now(), detailStr, id)
	return err
}

// HasDailyRunToday returns true if a successful daily run (status 'complete' or 'partial') was
// finished at any point today (local date). The time-of-day gate is applied in tick() before this
// is called, so a run recorded at any hour on today's date is sufficient to prevent a re-run.
// hour and minute are accepted for signature compatibility but are not used.
func (s *Store) HasDailyRunToday(hour, minute int) bool {
	var finished string
	err := s.DB.QueryRow(
		"SELECT finished FROM runs WHERE status IN ('complete','partial') ORDER BY id DESC LIMIT 1").Scan(&finished)
	if err != nil {
		return false
	}
	t, err := time.Parse("2006-01-02T15:04:05.000Z", finished)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z", finished)
	}
	if err != nil {
		t, err = time.Parse(time.RFC3339, finished)
	}
	if err != nil {
		return false
	}
	t = t.Local()
	now := time.Now()
	return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
}

// EnqueueAll enqueues every job that does not have a current fingerprint in the queue.
func (s *Store) EnqueueAll() error {
	rows, err := s.DB.Query("SELECT id FROM jobs")
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		job, err := s.getW(id)
		if err != nil {
			continue
		}
		if job.Stopped {
			continue // hidden by an active stop-word: not worth spending tokens on
		}
		fp := jobVersionHash(job.Title, job.Company, job.Description, job.DescriptionStatus, job.Location)
		var existingFP string
		_ = s.DB.QueryRow("SELECT fingerprint FROM assessment_queue WHERE job_id=?", id).Scan(&existingFP)
		if existingFP == fp {
			continue // already queued with current fingerprint
		}
		_, _ = s.EnqueueJob(id, fp, "", 0)
	}
	return nil
}
