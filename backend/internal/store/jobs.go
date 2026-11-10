package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/w3con/sifty/internal/model"
)

// ResolveID follows the job_aliases chain to return the current canonical id.
// Returns an error if a cycle is detected.
func (s *Store) ResolveID(id int64) (int64, error) {
	seen := make(map[int64]bool)
	for {
		if seen[id] {
			return 0, fmt.Errorf("job alias cycle detected for id %d", id)
		}
		seen[id] = true
		var newID int64
		err := s.DB.QueryRow("SELECT new_id FROM job_aliases WHERE old_id=?", id).Scan(&newID)
		if err == sql.ErrNoRows {
			return id, nil
		}
		if err != nil {
			return 0, fmt.Errorf("query job_aliases: %w", err)
		}
		id = newID
	}
}

// derivedState is the current authoritative profile revision and derived context version used to
// decide whether stored fast scores and detailed assessments are still current. A missing value
// is kept empty/zero rather than invented.
type derivedState struct {
	profileID       string
	profileRevision int
	contextVersion  string
}

func (s *Store) derivedState() derivedState {
	var d derivedState
	_ = s.DB.QueryRow(
		"SELECT id, revision FROM user_profiles ORDER BY id LIMIT 1").
		Scan(&d.profileID, &d.profileRevision)
	err := s.DB.QueryRow(
		"SELECT version FROM context_refreshes ORDER BY id DESC LIMIT 1").
		Scan(&d.contextVersion)
	if err == sql.ErrNoRows {
		_ = s.DB.QueryRow(
			"SELECT version FROM contexts ORDER BY created DESC LIMIT 1").
			Scan(&d.contextVersion)
	}
	return d
}

// Get returns the full role record for id, resolving aliases transparently. Returns
// a not-found error (contains "not found") when the role does not exist.
func (s *Store) Get(id int64) (*model.Job, error) {
	return s.get(id, s.derivedState())
}

func (s *Store) get(id int64, derived derivedState) (*model.Job, error) {
	id, err := s.ResolveID(id)
	if err != nil {
		return nil, err
	}

	row := s.DB.QueryRow(`
		SELECT id, identity, title, company, original_url, apply_url, description,
		       description_status, location, published_at, expires_at, status,
		       first_seen, last_seen, assessment, assessment_error,
		       apply_status, tags, seen_at, applied_at, match_score, match_profile_revision,
		       outcome_reason
		FROM jobs WHERE id=?`, id)

	var r struct {
		id, identity, title, company, originalURL string
		applyURL                                  sql.NullString
		description, descriptionStatus            string
		location, publishedAt, expiresAt          sql.NullString
		status, firstSeen, lastSeen               string
		assessment, assessmentError               sql.NullString
		applyStatus, tagsJSON, seenAt, appliedAt  sql.NullString
		outcomeReason                             sql.NullString
		idInt                                     int64
		matchScore                                int
		matchProfileRevision                      int
	}
	err = row.Scan(
		&r.idInt, &r.identity, &r.title, &r.company, &r.originalURL, &r.applyURL,
		&r.description, &r.descriptionStatus, &r.location, &r.publishedAt, &r.expiresAt,
		&r.status, &r.firstSeen, &r.lastSeen, &r.assessment, &r.assessmentError,
		&r.applyStatus, &r.tagsJSON, &r.seenAt, &r.appliedAt, &r.matchScore, &r.matchProfileRevision,
		&r.outcomeReason,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("scan job %d: %w", id, err)
	}

	var tags []string
	if r.tagsJSON.Valid && r.tagsJSON.String != "" && r.tagsJSON.String != "null" {
		_ = json.Unmarshal([]byte(r.tagsJSON.String), &tags)
	}
	if tags == nil {
		tags = []string{}
	}

	// Parse and enrich the assessment JSON.
	var assessmentObj any
	if r.assessment.Valid && r.assessment.String != "" {
		var m map[string]any
		if err := json.Unmarshal([]byte(r.assessment.String), &m); err == nil {
			if _, ok := m["confirmedMismatches"]; !ok {
				m["confirmedMismatches"] = []any{}
			}
			if _, ok := m["candidateUnknowns"]; !ok {
				if gaps, ok2 := m["gaps"]; ok2 {
					m["candidateUnknowns"] = gaps
				} else {
					m["candidateUnknowns"] = []any{}
				}
			}
			if _, ok := m["employerUnknowns"]; !ok {
				m["employerUnknowns"] = []any{}
			}
			// Compute stale flag. An assessment is current only when the vacancy, the derived
			// candidate context and the confirmed profile revision it recorded all still match.
			// Missing version fields are stale, never assumed current.
			var loc *string
			if r.location.Valid {
				v := r.location.String
				loc = &v
			}
			jobVer := jobVersionHash(r.title, r.company, r.description, r.descriptionStatus, loc)
			if jv, ok := m["jobVersion"].(string); ok {
				m["stale"] = jv != jobVer
			} else {
				m["stale"] = true
			}
			if cv, ok := m["contextVersion"].(string); !ok || cv != derived.contextVersion {
				m["stale"] = true
			}
			apID, _ := m["profileId"].(string)
			apRev := intValue(m["profileRevision"])
			if apID != derived.profileID || apRev != derived.profileRevision {
				m["stale"] = true
			}
			assessmentObj = m
		}
	}

	// Load sources. Close explicitly (not deferred) before returning the job,
	// so the single database connection is not held open for the caller.
	srows, err := s.DB.Query(
		"SELECT source, url, seen FROM sightings WHERE job_id=? ORDER BY seen", r.idInt)
	if err != nil {
		return nil, fmt.Errorf("query sightings for job %d: %w", r.idInt, err)
	}
	var sources []model.JobSource
	for srows.Next() {
		var src model.JobSource
		if err := srows.Scan(&src.Name, &src.URL, &src.Seen); err != nil {
			srows.Close()
			return nil, fmt.Errorf("scan sighting: %w", err)
		}
		sources = append(sources, src)
	}
	srows.Close()
	if sources == nil {
		sources = []model.JobSource{}
	}

	stage, _ := stageFor(r.status)

	var applyURLPtr *string
	if r.applyURL.Valid && r.applyURL.String != "" {
		v := r.applyURL.String
		applyURLPtr = &v
	}

	var loc *string
	if r.location.Valid && r.location.String != "" {
		v := r.location.String
		loc = &v
	}
	var pubAt *string
	if r.publishedAt.Valid && r.publishedAt.String != "" {
		v := r.publishedAt.String
		pubAt = &v
	}
	var expAt *string
	if r.expiresAt.Valid && r.expiresAt.String != "" {
		v := r.expiresAt.String
		expAt = &v
	}
	var seenAt *string
	if r.seenAt.Valid && r.seenAt.String != "" {
		v := r.seenAt.String
		seenAt = &v
	}
	var appliedAt *string
	if r.appliedAt.Valid && r.appliedAt.String != "" {
		v := r.appliedAt.String
		appliedAt = &v
	}
	var assessmentErr *string
	if r.assessmentError.Valid && r.assessmentError.String != "" {
		v := r.assessmentError.String
		assessmentErr = &v
	}
	var outcomeReason *string
	if r.outcomeReason.Valid && r.outcomeReason.String != "" {
		v := r.outcomeReason.String
		outcomeReason = &v
	}

	expired := false
	if expAt != nil {
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", *expAt); err == nil {
			expired = time.Now().After(t)
		} else if t, err := time.Parse(time.RFC3339, *expAt); err == nil {
			expired = time.Now().After(t)
		}
	}

	job := &model.Job{
		ID:                   r.idInt,
		Title:                r.title,
		Company:              r.company,
		Tags:                 tags,
		Stage:                stage,
		Status:               r.status,
		Seen:                 seenAt != nil,
		SeenAt:               seenAt,
		AppliedAt:            appliedAt,
		OutcomeReason:        outcomeReason,
		IsWeb3:               isWeb3(r.title, tags, r.description),
		OriginalURL:          r.originalURL,
		ApplyURL:             applyURLPtr,
		ApplyStatus:          r.applyStatus.String,
		Description:          r.description,
		DescriptionStatus:    r.descriptionStatus,
		Location:             loc,
		PublishedAt:          pubAt,
		ExpiresAt:            expAt,
		FirstSeen:            r.firstSeen,
		LastSeen:             r.lastSeen,
		Expired:              expired,
		MatchScore:           r.matchScore,
		MatchProfileRevision: r.matchProfileRevision,
		MatchStale:           r.matchProfileRevision != derived.profileRevision,
		Assessment:           assessmentObj,
		AssessmentError:      assessmentErr,
		Sources:              sources,
	}

	// Compute block status. Failure to load rules is non-fatal: the job is returned unblocked.
	if rules, err2 := s.ListBlockRules(); err2 == nil {
		job.Blocked, job.BlockedBy = blockMatch(job, rules)
	}

	// Compute stop-word status from the active stop-words. Derived on every read, so removing
	// a stop-word reverses it with no stored role mutated. Failure to load is non-fatal.
	if words, err2 := s.ListStopWords(); err2 == nil {
		job.Stopped, job.StoppedBy = stopMatch(job, words)
	}

	return job, nil
}

// List returns all roles, optionally filtered by status and query text, sorted by
// assessment score descending (unassessed roles sort last).
func (s *Store) List(status, query string) ([]*model.Job, error) {
	rows, err := s.DB.Query("SELECT id FROM jobs ORDER BY first_seen DESC")
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close() // explicit close before calling Get below (single-connection pool)

	var stageFilter string
	if status != "" && status != "all" {
		sf, err := stageFor(status)
		if err == nil {
			stageFilter = sf
		}
	}
	normQuery := normalizeText(query)
	derived := s.derivedState()

	var out []*model.Job
	for _, id := range ids {
		job, err := s.get(id, derived)
		if err != nil {
			continue // skip deleted/missing
		}
		if job.Stopped {
			continue // a stop-word hides the role from every listing, whatever its stage
		}
		if stageFilter != "" && job.Stage != stageFilter {
			continue
		}
		if normQuery != "" {
			haystack := normalizeText(job.Title + " " + job.Company + " " + job.Description)
			if !strings.Contains(haystack, normQuery) {
				continue
			}
		}
		out = append(out, job)
	}

	// Sort by assessment score descending; unassessed roles sort last (-1).
	sortJobs(out)
	return out, nil
}

// sortJobs sorts jobs by assessment score descending, unassessed last.
func sortJobs(jobs []*model.Job) {
	for i := 1; i < len(jobs); i++ {
		for j := i; j > 0; j-- {
			a, b := jobs[j-1], jobs[j]
			if scoreOf(a) < scoreOf(b) {
				jobs[j-1], jobs[j] = jobs[j], jobs[j-1]
			}
		}
	}
}

func scoreOf(j *model.Job) float64 {
	if m, ok := j.Assessment.(map[string]any); ok {
		if v, ok := m["score"]; ok {
			switch n := v.(type) {
			case float64:
				return n
			case json.Number:
				f, _ := n.Float64()
				return f
			}
		}
	}
	return -1
}

// Upsert inserts or deduplicates the incoming posting and returns its canonical role id.
// Returns 0 (not an error) when the posting matches a permanently deleted listing.
// When restore is true, tombstones for the listing's keys are cleared first.
func (s *Store) Upsert(j model.JobInput, restore bool) (int64, error) {
	originalCanon, err := canonicalURL(j.OriginalURL)
	if err != nil {
		return 0, fmt.Errorf("canonical original URL: %w", err)
	}
	var applyCanon string
	if j.ApplyURL != nil && *j.ApplyURL != "" {
		applyCanon, err = canonicalURL(*j.ApplyURL)
		if err != nil {
			applyCanon = "" // soft: bad apply URL is skipped
		}
	}

	identity := normalizeText(j.Company) + "|" + normalizeText(j.Title)

	keys, err := listingKeys(j.OriginalURL, j.ApplyURL)
	if err != nil {
		return 0, fmt.Errorf("listing keys: %w", err)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if restore {
		for _, k := range keys {
			if _, err := tx.Exec("DELETE FROM deleted_listings WHERE hash=?", k); err != nil {
				return 0, fmt.Errorf("clear tombstone: %w", err)
			}
		}
	} else {
		for _, k := range keys {
			var h string
			err := tx.QueryRow("SELECT hash FROM deleted_listings WHERE hash=?", k).Scan(&h)
			if err == nil {
				// Key is deleted: skip silently.
				_ = tx.Rollback()
				tx = nil
				return 0, nil
			}
		}
	}

	// Compute same-host prefix to guard against false identity matches within one site.
	var sameHostPrefix string
	if u, err2 := url.Parse(originalCanon); err2 == nil {
		sameHostPrefix = u.Scheme + "://" + u.Host + "/%"
	}

	applyArg := sql.NullString{Valid: applyCanon != "", String: applyCanon}
	rows, err := tx.Query(`
		SELECT id, status, seen_at, applied_at FROM jobs
		WHERE id IN (SELECT job_id FROM sightings WHERE url=?)
		   OR (apply_url IS NOT NULL AND apply_url=?)
		   OR (identity=? AND company != 'Unknown' AND original_url NOT LIKE ?
		       AND (? IS NULL OR apply_url IS NULL OR apply_url=?))
		ORDER BY id`,
		originalCanon, applyArg,
		identity, sameHostPrefix,
		applyArg, applyArg,
	)
	if err != nil {
		return 0, fmt.Errorf("find existing jobs: %w", err)
	}

	type existingRow struct {
		id        int64
		status    string
		seenAt    sql.NullString
		appliedAt sql.NullString
	}
	var found []existingRow
	for rows.Next() {
		var r existingRow
		if err := rows.Scan(&r.id, &r.status, &r.seenAt, &r.appliedAt); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan existing: %w", err)
		}
		found = append(found, r)
	}
	rows.Close()

	t := now()
	var id int64
	var dupIDs []int64 // ids of duplicate rows merged and deleted during this Upsert

	if len(found) == 0 {
		// New role: allocate next id from sequence.
		var nextID int64
		err := tx.QueryRow(`
			UPDATE id_sequence
			SET value = max(value, (SELECT COALESCE(max(old_id),0) FROM job_aliases),
			                       (SELECT COALESCE(max(id),0) FROM jobs)) + 1
			WHERE name='jobs' RETURNING value`).Scan(&nextID)
		if err != nil {
			return 0, fmt.Errorf("allocate id: %w", err)
		}

		tagsJSON, _ := json.Marshal(j.Tags)
		applyStatus := j.ApplyStatus
		if applyStatus == "" {
			applyStatus = "not resolved"
		}
		_, err = tx.Exec(`
			INSERT INTO jobs(id, identity, title, company, original_url, apply_url,
			                 description, description_status, location, published_at, expires_at,
			                 first_seen, last_seen, apply_status, tags)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			nextID, identity, j.Title, j.Company, originalCanon, nullStr(applyCanon),
			j.Description, j.DescriptionStatus, nullStr(j.Location),
			nullStr(j.PublishedAt), nullStr(j.ExpiresAt),
			t, t, applyStatus, string(tagsJSON),
		)
		if err != nil {
			return 0, fmt.Errorf("insert job: %w", err)
		}
		id = nextID
	} else {
		id = found[0].id
		// Merge any additional duplicate rows into found[0].
		for _, dup := range found[1:] {
			// Move application data.
			for _, tbl := range []string{
				"application_drafts", "application_revisions",
				"questionnaire_snapshots", "preparation_operations",
			} {
				if _, err := tx.Exec("UPDATE "+tbl+" SET job_id=? WHERE job_id=?", id, dup.id); err != nil {
					// Table may not have the row; ignore.
					_ = err
				}
			}
			// Move assessment/history/sighting/snapshot/feedback.
			for _, tbl := range []string{
				"sightings", "snapshots", "assessments",
				"feedback", "workflow_history",
			} {
				if _, err := tx.Exec("UPDATE "+tbl+" SET job_id=? WHERE job_id=?", id, dup.id); err != nil {
					_ = err
				}
			}
			if _, err := tx.Exec("DELETE FROM assessment_queue WHERE job_id=?", dup.id); err != nil {
				_ = err
			}
			if _, err := tx.Exec("UPDATE job_aliases SET new_id=? WHERE new_id=?", id, dup.id); err != nil {
				_ = err
			}
			if _, err := tx.Exec("INSERT OR REPLACE INTO job_aliases VALUES(?,?)", dup.id, id); err != nil {
				return 0, fmt.Errorf("insert alias: %w", err)
			}
			if _, err := tx.Exec("DELETE FROM jobs WHERE id=?", dup.id); err != nil {
				return 0, fmt.Errorf("delete dup: %w", err)
			}
			dupIDs = append(dupIDs, dup.id)
		}

		// Carry forward the strongest workflow status.
		bestStatus := "new"
		for _, r := range found {
			stage, _ := stageFor(r.status)
			switch stage {
			case "applied":
				bestStatus = "applied"
			case "preselected":
				if bestStatus != "applied" {
					bestStatus = "saved"
				}
			case "no":
				if bestStatus == "new" {
					bestStatus = "rejected"
				}
			}
		}

		// Carry forward seen_at and applied_at from the found rows.
		var bestSeenAt, bestAppliedAt string
		for _, r := range found {
			if r.seenAt.Valid && r.seenAt.String != "" {
				if bestSeenAt == "" || r.seenAt.String < bestSeenAt {
					bestSeenAt = r.seenAt.String
				}
			}
			if r.appliedAt.Valid && r.appliedAt.String != "" && bestStatus == "applied" {
				if bestAppliedAt == "" {
					bestAppliedAt = r.appliedAt.String
				}
			}
		}
		if _, err := tx.Exec("UPDATE jobs SET seen_at=?, applied_at=? WHERE id=?",
			nullStr(bestSeenAt), nullStr(bestAppliedAt), id); err != nil {
			return 0, fmt.Errorf("update seen/applied: %w", err)
		}

		if _, err := tx.Exec(
			`UPDATE jobs SET title=?,
			  company=CASE WHEN ? != 'Unknown' THEN ? ELSE company END,
			  identity=CASE WHEN ? != 'Unknown' THEN ? ELSE identity END,
			  status=? WHERE id=?`,
			j.Title, j.Company, j.Company, j.Company, identity, bestStatus, id); err != nil {
			return 0, fmt.Errorf("update job core fields: %w", err)
		}

		// Update description only when the new one is richer.
		freshDescription := len(j.Description) >= 100 &&
			!strings.ContainsAny(strings.ToLower(j.DescriptionStatus), "metadata|pending|missing|unavailable")

		var prevDesc, prevDescStatus string
		_ = tx.QueryRow("SELECT description, description_status FROM jobs WHERE id=?", id).
			Scan(&prevDesc, &prevDescStatus)
		previousReady := len(prevDesc) >= 100 &&
			!strings.ContainsAny(strings.ToLower(prevDescStatus), "metadata|pending|missing|unavailable")

		replace := freshDescription || (!previousReady && len(j.Description) >= len(prevDesc))

		if _, err := tx.Exec(`
			UPDATE jobs SET last_seen=?, apply_url=COALESCE(?,apply_url),
			  description=CASE WHEN ? THEN ? ELSE description END,
			  description_status=CASE WHEN ? THEN ? ELSE description_status END,
			  expires_at=CASE WHEN ?!='' THEN ? ELSE expires_at END WHERE id=?`,
			t, nullStr(applyCanon),
			boolInt(replace), j.Description, boolInt(replace), j.DescriptionStatus,
			j.ExpiresAt, nullStr(j.ExpiresAt), id,
		); err != nil {
			return 0, fmt.Errorf("update job: %w", err)
		}
	}

	// Upsert sighting.
	if _, err := tx.Exec(
		"INSERT INTO sightings VALUES(?,?,?,?) ON CONFLICT(url) DO UPDATE SET seen=excluded.seen",
		id, j.Source, originalCanon, t); err != nil {
		return 0, fmt.Errorf("upsert sighting: %w", err)
	}
	if j.Location != "" {
		if _, err := tx.Exec("UPDATE jobs SET location=? WHERE id=?", j.Location, id); err != nil {
			return 0, err
		}
	}
	if len(j.Tags) > 0 {
		tagsJSON, _ := json.Marshal(j.Tags)
		if _, err := tx.Exec("UPDATE jobs SET tags=? WHERE id=?", string(tagsJSON), id); err != nil {
			return 0, err
		}
	}
	// Handle apply_status board-intermediary rules.
	if strings.HasPrefix(j.ApplyStatus, "board intermediary") ||
		strings.HasPrefix(j.ApplyStatus, "restricted board") {
		if _, err := tx.Exec("UPDATE jobs SET apply_url=NULL WHERE id=?", id); err != nil {
			return 0, err
		}
	}
	if j.ApplyStatus != "" {
		if _, err := tx.Exec("UPDATE jobs SET apply_status=? WHERE id=?", j.ApplyStatus, id); err != nil {
			return 0, err
		}
	}

	// Record a snapshot.
	snapJSON, _ := json.Marshal(j)
	if _, err := tx.Exec(
		"INSERT INTO snapshots(job_id,captured,data) VALUES(?,?,?)",
		id, t, string(snapJSON)); err != nil {
		return 0, fmt.Errorf("insert snapshot: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit upsert: %w", err)
	}
	tx = nil

	// Refresh the FTS index outside the transaction. No SQL triggers maintain jobs_fts (see
	// fts5Stmt in migrate.go for why); this call is the sole sync point for insert and update.
	// Merged duplicate rows are removed from the FTS as well.
	for _, dup := range dupIDs {
		_, _ = s.DB.Exec("DELETE FROM jobs_fts WHERE rowid=?", dup)
	}
	s.refreshFTS(id)

	return id, nil
}

// refreshFTS reads the current indexed columns for the given job id from the jobs table and
// re-writes the corresponding jobs_fts row. It runs outside any transaction (auto-commit), so
// the FTS shadow tables see committed data, which avoids the pending-terms inconsistency that
// plagues AFTER UPDATE triggers. Errors are silently swallowed: the FTS index is a secondary
// structure and a stale entry will be corrected on the next Upsert or Rescore call.
func (s *Store) refreshFTS(id int64) {
	var title, description, company, tags string
	if err := s.DB.QueryRow(
		"SELECT title, description, company, COALESCE(tags,'[]') FROM jobs WHERE id=?", id,
	).Scan(&title, &description, &company, &tags); err != nil {
		return
	}
	_, _ = s.DB.Exec("DELETE FROM jobs_fts WHERE rowid=?", id)
	_, _ = s.DB.Exec(
		"INSERT INTO jobs_fts(rowid, title, description, company, tags) VALUES(?,?,?,?,?)",
		id, title, description, company, tags)
}

// Seen marks the role as seen (sets seen_at on first call; subsequent calls are no-ops).
func (s *Store) Seen(id int64) (*model.Job, error) {
	job, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(
		"UPDATE jobs SET seen_at=COALESCE(seen_at,?) WHERE id=?", now(), job.ID); err != nil {
		return nil, fmt.Errorf("mark seen: %w", err)
	}
	return s.Get(job.ID)
}

// Status moves the role to the given workflow status. Moving to applied requires the
// role to be pre-selected and confirmed=true. Moving to the same stage is a no-op.
func (s *Store) Status(id int64, status string, confirmed bool) (*model.Job, error) {
	return s.StatusWithReason(id, status, confirmed, "")
}

// OutcomeReasonStale, OutcomeReasonClosed and OutcomeReasonNotSuitable are the durable
// reasons for a deliberate negative application outcome. An empty reason keeps the
// existing plain-dismissal behavior valid.
const (
	OutcomeReasonStale       = "stale"
	OutcomeReasonClosed      = "closed"
	OutcomeReasonNotSuitable = "not_suitable"
)

func validOutcomeReason(reason string) bool {
	switch reason {
	case "", OutcomeReasonStale, OutcomeReasonClosed, OutcomeReasonNotSuitable:
		return true
	}
	return false
}

// StatusWithReason moves the role and, for a negative (dismissed) outcome, stores a
// distinct durable reason. The write is conditional on the stage observed at read time,
// so a stale negative confirmation cannot overwrite an application another actor
// confirmed in the meantime: no applied timestamp, history or current stage is erased.
// Reason validation and the atomic stage/reason/history write belong here, in the store.
func (s *Store) StatusWithReason(id int64, status string, confirmed bool, reason string) (*model.Job, error) {
	stage, err := stageFor(status)
	if err != nil {
		return nil, err
	}
	if !validOutcomeReason(reason) {
		return nil, fmt.Errorf("unknown outcome reason %q", reason)
	}
	if reason != "" && stage != "no" {
		return nil, fmt.Errorf("an outcome reason is valid only for a negative outcome")
	}
	job, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	id = job.ID
	if reason != "" && job.Stage != "preselected" {
		return nil, fmt.Errorf("the role is no longer awaiting an application decision; reload and try again")
	}
	if stage == "applied" && job.Stage != "applied" {
		if job.Stage != "preselected" || !confirmed {
			return nil, fmt.Errorf("Applied requires a pre-selected role and explicit submission confirmation")
		}
	}
	if stage == job.Stage {
		return job, nil
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

	var appliedAt any
	if stage == "applied" {
		appliedAt = now()
	}
	var reasonValue any
	if stage == "no" && reason != "" {
		reasonValue = reason
	}
	res, err := tx.Exec(
		"UPDATE jobs SET status=?, applied_at=?, outcome_reason=? WHERE id=? AND status=?",
		statusFor(stage), appliedAt, reasonValue, id, job.Status)
	if err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}
	if changed == 0 {
		return nil, fmt.Errorf("job %d changed while this outcome was being saved; reload and try again", id)
	}
	var reasonHistory any
	if reason != "" {
		reasonHistory = reason
	}
	if _, err := tx.Exec(
		"INSERT INTO workflow_history(job_id,created,from_stage,to_stage,confirmed,reason) VALUES(?,?,?,?,?,?)",
		id, now(), job.Stage, stage, boolInt(confirmed), reasonHistory); err != nil {
		return nil, fmt.Errorf("insert workflow history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil

	// Enqueue for LLM assessment when the role is moved to preselected. This is the only
	// point where automatic enqueueing happens; collection no longer calls EnqueueAll.
	if stage == "preselected" {
		updated, gerr := s.Get(id)
		if gerr == nil {
			fp := jobVersionHash(updated.Title, updated.Company, updated.Description, updated.DescriptionStatus, updated.Location)
			_, _ = s.EnqueueJob(id, fp, "", 0)
		}
	}

	return s.Get(id)
}

// Delete permanently deletes a role. Returns an error when the role is in triage
// or when collection/assessment/preparation is currently active.
// All reads are performed before the write transaction to avoid single-connection deadlocks.
func (s *Store) Delete(id int64) (int64, []string, error) {
	job, err := s.Get(id)
	if err != nil {
		return 0, nil, err
	}
	id = job.ID
	if job.Stage == "triage" {
		return 0, nil, fmt.Errorf("Dismiss roles in Triage; permanent Delete is available outside Triage")
	}

	// Phase 1: reads — no transaction held while querying.

	// Check that no collection or assessment lock is held by a live process.
	if err := s.checkNoActiveLocks(); err != nil {
		return 0, nil, err
	}

	// Check for active assessment.
	var activeAssessment int
	_ = s.DB.QueryRow(
		"SELECT COUNT(*) FROM assessment_queue WHERE job_id=? AND status='running'", id).
		Scan(&activeAssessment)
	if activeAssessment > 0 {
		return 0, nil, fmt.Errorf("Assessment is active. Wait for it to finish, then delete again.")
	}

	// Collect all alias ids.
	allIDs, err := s.collectAliasIDs(id)
	if err != nil {
		return 0, nil, err
	}

	// Compute listing keys for tombstones.
	keys := make(map[string]bool)
	for _, k := range must(listingKeys(job.OriginalURL, job.ApplyURL)) {
		keys[k] = true
	}
	for rID := range allIDs {
		rows, err := s.DB.Query("SELECT url FROM sightings WHERE job_id=?", rID)
		if err != nil {
			continue
		}
		for rows.Next() {
			var u string
			if err := rows.Scan(&u); err == nil {
				if c, err2 := canonicalURL(u); err2 == nil {
					keys[hashStr("url:"+c)] = true
				}
			}
		}
		rows.Close()
	}

	// Read the discovery cache entries before the transaction.
	type cacheEntry struct{ key, data string }
	var cacheEntries []cacheEntry
	{
		rows, err := s.DB.Query("SELECT key, data FROM discovery_cache")
		if err == nil {
			for rows.Next() {
				var e cacheEntry
				if err := rows.Scan(&e.key, &e.data); err == nil {
					cacheEntries = append(cacheEntries, e)
				}
			}
			rows.Close()
		}
	}

	// Phase 2: writes — single transaction.
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	// Write tombstones.
	for k := range keys {
		if _, err := tx.Exec("INSERT OR IGNORE INTO deleted_listings VALUES(?)", k); err != nil {
			return 0, nil, err
		}
	}

	// Delete the role and all related rows.
	for rID := range allIDs {
		for _, tbl := range []string{
			"sightings", "snapshots", "assessments", "assessment_queue",
			"feedback", "workflow_history", "application_drafts",
			"application_revisions", "questionnaire_snapshots", "preparation_operations",
		} {
			_, _ = tx.Exec("DELETE FROM "+tbl+" WHERE job_id=?", rID)
		}
		_, _ = tx.Exec("DELETE FROM job_aliases WHERE old_id=? OR new_id=?", rID, rID)
	}

	// Advance the id_sequence so the deleted id is never reused.
	var maxID int64
	for rID := range allIDs {
		if rID > maxID {
			maxID = rID
		}
	}
	if _, err := tx.Exec(
		"UPDATE id_sequence SET value=max(value,?) WHERE name='jobs'", maxID); err != nil {
		return 0, nil, err
	}
	if _, err := tx.Exec("DELETE FROM jobs WHERE id=?", id); err != nil {
		return 0, nil, err
	}

	// Remove the deleted role from discovery cache.
	for _, e := range cacheEntries {
		var cached map[string]any
		if err := json.Unmarshal([]byte(e.data), &cached); err != nil {
			_, _ = tx.Exec("DELETE FROM discovery_cache WHERE key=?", e.key)
			continue
		}
		if jobsList, ok := cached["jobs"].([]any); ok {
			var filtered []any
			for _, item := range jobsList {
				if jm, ok2 := item.(map[string]any); ok2 {
					origURL, _ := jm["originalUrl"].(string)
					applyURLStr, _ := jm["applyUrl"].(string)
					var applyURLPtr *string
					if applyURLStr != "" {
						applyURLPtr = &applyURLStr
					}
					lk, _ := listingKeys(origURL, applyURLPtr)
					removed := false
					for _, k := range lk {
						if keys[k] {
							removed = true
							break
						}
					}
					if !removed {
						filtered = append(filtered, item)
					}
				}
			}
			cached["jobs"] = filtered
			newData, _ := json.Marshal(cached)
			_, _ = tx.Exec("UPDATE discovery_cache SET data=? WHERE key=?", string(newData), e.key)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}
	tx = nil

	// Remove FTS entries for all deleted ids (no SQL trigger does this any more; see fts5Stmt).
	for rID := range allIDs {
		_, _ = s.DB.Exec("DELETE FROM jobs_fts WHERE rowid=?", rID)
	}

	return id, nil, nil
}

// checkNoActiveLocks verifies that no live process holds a named lock in the database.
func (s *Store) checkNoActiveLocks() error {
	rows, err := s.DB.Query("SELECT pid FROM locks WHERE name IN ('daily','inference')")
	if err != nil {
		return fmt.Errorf("query locks: %w", err)
	}
	var pids []int
	for rows.Next() {
		var pid int
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		pids = append(pids, pid)
	}
	rows.Close()
	for _, pid := range pids {
		if isProcessAlive(pid) {
			return fmt.Errorf("Work is active. Wait for collection or assessment to finish, then delete again.")
		}
	}
	return nil
}

// collectAliasIDs returns all ids that are aliases of id (including id itself).
func (s *Store) collectAliasIDs(id int64) (map[int64]bool, error) {
	allAliasRows, err := s.DB.Query("SELECT old_id, new_id FROM job_aliases")
	if err != nil {
		return nil, fmt.Errorf("query job_aliases: %w", err)
	}
	type pair struct{ old, new int64 }
	var pairs []pair
	for allAliasRows.Next() {
		var p pair
		if err := allAliasRows.Scan(&p.old, &p.new); err != nil {
			allAliasRows.Close()
			return nil, err
		}
		pairs = append(pairs, p)
	}
	allAliasRows.Close()

	ids := map[int64]bool{id: true}
	changed := true
	for changed {
		changed = false
		for _, p := range pairs {
			if ids[p.new] && !ids[p.old] {
				ids[p.old] = true
				changed = true
			}
		}
	}
	return ids, nil
}

// Health upserts a per-source health record.
func (s *Store) Health(source, status, message string, count int) error {
	_, err := s.DB.Exec(
		`INSERT INTO health VALUES(?,?,?,?,?)
		 ON CONFLICT(source) DO UPDATE SET status=excluded.status,
		   message=excluded.message,checked=excluded.checked,count=excluded.count`,
		source, status, message, now(), count)
	return err
}

// HealthList returns all source health records sorted by source name.
func (s *Store) HealthList() ([]map[string]any, error) {
	rows, err := s.DB.Query("SELECT source, status, message, checked, count FROM health ORDER BY source")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var source, status, message, checked string
		var count int
		if err := rows.Scan(&source, &status, &message, &checked, &count); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"source": source, "status": status, "message": message,
			"checked": checked, "count": count,
		})
	}
	return out, nil
}

// PurgeArchive deletes applied roles whose applied_at exceeds the retention window.
func (s *Store) PurgeArchive(clock time.Time) ([]int64, []map[string]any) {
	rows, err := s.DB.Query("SELECT id, applied_at FROM jobs WHERE status='applied'")
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	type candidate struct {
		id        int64
		appliedAt string
	}
	var candidates []candidate
	for rows.Next() {
		var id int64
		var appliedAt sql.NullString
		if err := rows.Scan(&id, &appliedAt); err != nil {
			continue
		}
		if appliedAt.Valid {
			candidates = append(candidates, candidate{id, appliedAt.String})
		}
	}

	var deleted []int64
	var skipped []map[string]any
	for _, c := range candidates {
		t, err := time.Parse("2006-01-02T15:04:05.000Z", c.appliedAt)
		if err != nil {
			continue
		}
		if clock.Sub(t).Milliseconds() < archiveRetentionMs {
			continue
		}
		deletedID, _, err := s.Delete(c.id)
		if err != nil {
			skipped = append(skipped, map[string]any{"id": c.id, "reason": err.Error()})
		} else {
			deleted = append(deleted, deletedID)
		}
	}
	return deleted, skipped
}

// --- helpers ---

func nullStr(s string) sql.NullString {
	return sql.NullString{Valid: s != "", String: s}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func must(keys []string, err error) []string {
	if err != nil {
		return nil
	}
	return keys
}
