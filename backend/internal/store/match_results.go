package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

// ErrMatchResultStale means the result's provenance no longer matches the authoritative saved
// profile, source, facts or active rules/source policy. The write is refused and any previously
// stored result is preserved.
var ErrMatchResultStale = errors.New("quick match result is stale")

// StoredMatchResult is one persisted evaluator result together with the identities used to
// decide whether it is still current. Reading never mutates the row.
type StoredMatchResult struct {
	JobID                int64
	Result               matchcontract.Result
	RulesIdentity        string
	SourcePolicyIdentity string
	ComputedAt           string
}

// SaveMatchResult writes one result with a same-transaction compare-and-swap. It verifies the
// actual current profile revision, retained source hash, stored facts hash and fact production
// versions, and requires non-empty rules/source-policy identities. The store does not load
// configuration and does not itself prove those identities are active: the application service
// derives them from the active configuration and holds config.Holder.WithSnapshot across the
// check and this write. The numeric compatibility match_score mirrors the result only after the
// transaction checks pass.
func (s *Store) SaveMatchResult(jobID int64, r matchcontract.Result) error {
	if jobID <= 0 {
		return fmt.Errorf("quick match result: job id must be positive")
	}
	if err := r.Validate(); err != nil {
		return fmt.Errorf("quick match result: %w", err)
	}
	p := r.Provenance
	if strings.TrimSpace(p.ProfileID) == "" || strings.TrimSpace(p.FactsHash) == "" ||
		strings.TrimSpace(p.RawSourceHash) == "" || strings.TrimSpace(p.RulesVersion) == "" ||
		strings.TrimSpace(p.RulesIdentity) == "" || strings.TrimSpace(p.SourcePolicyIdentity) == "" {
		return fmt.Errorf("%w: result provenance is incomplete", ErrMatchResultStale)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin quick match result for job %d: %w", jobID, err)
	}
	defer tx.Rollback()

	var profileID string
	var profileRevision int
	err = tx.QueryRow("SELECT id, revision FROM user_profiles ORDER BY id LIMIT 1").Scan(&profileID, &profileRevision)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: no current profile", ErrMatchResultStale)
	}
	if err != nil {
		return fmt.Errorf("read current profile: %w", err)
	}
	if profileID != p.ProfileID || profileRevision != p.ProfileRevision {
		return fmt.Errorf("%w: profile moved from %s@%d to %s@%d",
			ErrMatchResultStale, p.ProfileID, p.ProfileRevision, profileID, profileRevision)
	}

	var factsHash, rawHash, schema, extractor, policyVersion string
	err = tx.QueryRow(`SELECT facts_hash, raw_source_hash, schema_version, extractor_version,
		source_policy_version FROM match_job_facts WHERE job_id=?`, jobID).
		Scan(&factsHash, &rawHash, &schema, &extractor, &policyVersion)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: retained facts are missing", ErrMatchResultStale)
	}
	if err != nil {
		return fmt.Errorf("read facts for job %d: %w", jobID, err)
	}
	if factsHash != p.FactsHash || rawHash != p.RawSourceHash ||
		schema != p.SchemaVersion || extractor != p.ExtractorVersion ||
		policyVersion != p.SourcePolicyVersion {
		return fmt.Errorf("%w: facts production changed", ErrMatchResultStale)
	}

	var inputHash string
	err = tx.QueryRow("SELECT raw_source_hash FROM match_job_inputs WHERE job_id=?", jobID).Scan(&inputHash)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: retained input is missing", ErrMatchResultStale)
	}
	if err != nil {
		return fmt.Errorf("read retained input for job %d: %w", jobID, err)
	}
	current, err := readRetainedSource(tx, jobID)
	if err != nil {
		return err
	}
	currentHash, err := matchcontract.HashRetainedSource(current)
	if err != nil {
		return err
	}
	if inputHash != p.RawSourceHash || currentHash != p.RawSourceHash {
		return fmt.Errorf("%w: saved source changed", ErrMatchResultStale)
	}

	blob, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode quick match result for job %d: %w", jobID, err)
	}
	score := any(nil)
	if r.Score != nil {
		score = *r.Score
	}
	rawScore := any(nil)
	if r.RawScore != nil {
		rawScore = *r.RawScore
	}
	if _, err := tx.Exec(`
		INSERT INTO match_job_results(
			job_id, profile_id, profile_revision, facts_hash, raw_source_hash, schema_version,
			extractor_version, source_policy_version, source_policy_identity, rules_version,
			rules_identity, status, score, raw_score, result, computed_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(job_id) DO UPDATE SET
			profile_id=excluded.profile_id,
			profile_revision=excluded.profile_revision,
			facts_hash=excluded.facts_hash,
			raw_source_hash=excluded.raw_source_hash,
			schema_version=excluded.schema_version,
			extractor_version=excluded.extractor_version,
			source_policy_version=excluded.source_policy_version,
			source_policy_identity=excluded.source_policy_identity,
			rules_version=excluded.rules_version,
			rules_identity=excluded.rules_identity,
			status=excluded.status,
			score=excluded.score,
			raw_score=excluded.raw_score,
			result=excluded.result,
			computed_at=excluded.computed_at`,
		jobID, p.ProfileID, p.ProfileRevision, p.FactsHash, p.RawSourceHash, p.SchemaVersion,
		p.ExtractorVersion, p.SourcePolicyVersion, p.SourcePolicyIdentity, p.RulesVersion,
		p.RulesIdentity, string(r.Status), score, rawScore, string(blob), now()); err != nil {
		return fmt.Errorf("store quick match result for job %d: %w", jobID, err)
	}

	compat := 0
	if r.Status == matchcontract.StatusScored && r.Score != nil {
		compat = *r.Score
	}
	res, err := tx.Exec(
		"UPDATE jobs SET match_score=?, match_profile_revision=? WHERE id=? AND (match_profile_revision IS NULL OR match_profile_revision <= ?)",
		compat, p.ProfileRevision, jobID, p.ProfileRevision)
	if err != nil {
		return fmt.Errorf("mirror quick match compatibility score for job %d: %w", jobID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("quick match compatibility rows for job %d: %w", jobID, err)
	}
	if affected != 1 {
		return fmt.Errorf("%w: compatibility score was not updated", ErrMatchResultStale)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit quick match result for job %d: %w", jobID, err)
	}
	s.BumpCatalogueSeq()
	return nil
}

// GetMatchResult returns a persisted result, or nil when none exists. A schema mismatch is
// refused rather than interpreted as a current result.
func (s *Store) GetMatchResult(jobID int64) (*StoredMatchResult, error) {
	var blob, rulesIdentity, policyIdentity, computedAt string
	err := s.DB.QueryRow(`SELECT result, rules_identity, source_policy_identity, computed_at
		FROM match_job_results WHERE job_id=?`, jobID).Scan(&blob, &rulesIdentity, &policyIdentity, &computedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read quick match result for job %d: %w", jobID, err)
	}
	var r matchcontract.Result
	if err := json.Unmarshal([]byte(blob), &r); err != nil {
		return nil, fmt.Errorf("decode quick match result for job %d: %w", jobID, err)
	}
	if err := matchcontract.CheckSchema(r.Provenance.SchemaVersion); err != nil {
		return nil, fmt.Errorf("job %d quick match result: %w", jobID, err)
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("job %d quick match result: %w", jobID, err)
	}
	return &StoredMatchResult{
		JobID: jobID, Result: r,
		RulesIdentity: rulesIdentity, SourcePolicyIdentity: policyIdentity, ComputedAt: computedAt,
	}, nil
}

// MatchResultCurrent reports whether a stored result still reflects the authoritative saved
// profile, retained source, facts and the active rules/source-policy identities. It reads a
// single consistent transaction; it never recomputes, mutates, fetches or calls a model.
func (s *Store) MatchResultCurrent(jobID int64, r matchcontract.Result, rulesIdentity, sourcePolicyVersion, sourcePolicyIdentity string) (bool, []string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return false, nil, fmt.Errorf("begin quick match current check for job %d: %w", jobID, err)
	}
	defer tx.Rollback()

	p := r.Provenance
	var reasons []string
	check := func(ok bool, reason string) {
		if !ok {
			reasons = append(reasons, reason)
		}
	}
	check(p.SchemaVersion == matchcontract.SchemaVersion, "schema_version_changed")
	check(p.ExtractorVersion == matchcontract.ExtractorVersion, "extractor_version_changed")
	check(strings.TrimSpace(rulesIdentity) != "" && p.RulesIdentity == rulesIdentity, "rules_changed")
	check(strings.TrimSpace(sourcePolicyIdentity) != "" && p.SourcePolicyIdentity == sourcePolicyIdentity, "source_policy_changed")
	check(strings.TrimSpace(sourcePolicyVersion) != "" && p.SourcePolicyVersion == sourcePolicyVersion, "source_policy_version_changed")

	var profileID string
	var profileRevision int
	err = tx.QueryRow("SELECT id, revision FROM user_profiles ORDER BY id LIMIT 1").Scan(&profileID, &profileRevision)
	if err == sql.ErrNoRows {
		check(false, "profile_missing")
	} else if err != nil {
		return false, nil, fmt.Errorf("read current profile: %w", err)
	} else {
		check(profileID == p.ProfileID && profileRevision == p.ProfileRevision, "profile_changed")
	}

	var factsHash, rawHash, schema, extractor, policyVersion string
	err = tx.QueryRow(`SELECT facts_hash, raw_source_hash, schema_version, extractor_version,
		source_policy_version FROM match_job_facts WHERE job_id=?`, jobID).
		Scan(&factsHash, &rawHash, &schema, &extractor, &policyVersion)
	if err == sql.ErrNoRows {
		check(false, "facts_missing")
	} else if err != nil {
		return false, nil, fmt.Errorf("read facts for job %d: %w", jobID, err)
	} else {
		check(factsHash == p.FactsHash, "facts_hash_changed")
		check(rawHash == p.RawSourceHash, "facts_source_changed")
		check(schema == p.SchemaVersion, "facts_schema_changed")
		check(extractor == p.ExtractorVersion, "facts_extractor_changed")
		check(policyVersion == p.SourcePolicyVersion, "facts_source_policy_changed")
	}

	var inputHash string
	err = tx.QueryRow("SELECT raw_source_hash FROM match_job_inputs WHERE job_id=?", jobID).Scan(&inputHash)
	if err == sql.ErrNoRows {
		check(false, "source_input_missing")
	} else if err != nil {
		return false, nil, fmt.Errorf("read retained input for job %d: %w", jobID, err)
	} else {
		current, cerr := readRetainedSource(tx, jobID)
		if cerr != nil {
			return false, nil, cerr
		}
		currentHash, herr := matchcontract.HashRetainedSource(current)
		if herr != nil {
			return false, nil, herr
		}
		check(inputHash == p.RawSourceHash && currentHash == p.RawSourceHash, "saved_source_changed")
	}
	if err := tx.Commit(); err != nil {
		return false, nil, fmt.Errorf("commit quick match current check for job %d: %w", jobID, err)
	}
	return len(reasons) == 0, reasons, nil
}
