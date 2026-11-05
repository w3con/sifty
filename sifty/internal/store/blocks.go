package store

import (
	"fmt"
	"net/url"

	"github.com/w3con/sifty/internal/model"
)

// BlockRule is one durable filter rule that hides roles matching its kind and value.
type BlockRule struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Value     string `json:"value"`
	CreatedAt string `json:"createdAt"`
}

// AddBlockRule inserts a block rule, returning the existing rule when (kind, value) already exists.
// The insert is idempotent: a duplicate (kind, value) pair is silently ignored and the existing
// row is returned, so callers never need to check for duplicates themselves.
func (s *Store) AddBlockRule(kind, value string) (*BlockRule, error) {
	_, err := s.DB.Exec(
		`INSERT OR IGNORE INTO block_rules(kind, value, created_at) VALUES(?, ?, ?)`,
		kind, value, now(),
	)
	if err != nil {
		return nil, fmt.Errorf("insert block rule: %w", err)
	}
	var r BlockRule
	if err := s.DB.QueryRow(
		`SELECT id, kind, value, created_at FROM block_rules WHERE kind=? AND value=?`,
		kind, value,
	).Scan(&r.ID, &r.Kind, &r.Value, &r.CreatedAt); err != nil {
		return nil, fmt.Errorf("get block rule: %w", err)
	}
	return &r, nil
}

// ListBlockRules returns all block rules ordered by creation time descending.
func (s *Store) ListBlockRules() ([]*BlockRule, error) {
	rows, err := s.DB.Query(
		`SELECT id, kind, value, created_at FROM block_rules ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list block rules: %w", err)
	}
	defer rows.Close()
	var out []*BlockRule
	for rows.Next() {
		var r BlockRule
		if err := rows.Scan(&r.ID, &r.Kind, &r.Value, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan block rule: %w", err)
		}
		out = append(out, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate block rules: %w", err)
	}
	if out == nil {
		out = []*BlockRule{}
	}
	return out, nil
}

// RemoveBlockRule deletes the rule with the given id. Returns an error when the id does not exist.
func (s *Store) RemoveBlockRule(id int64) error {
	res, err := s.DB.Exec(`DELETE FROM block_rules WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete block rule: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("block rule %d not found", id)
	}
	return nil
}

// BlockJob derives a block rule from a job and inserts it. dimension must be "company"
// (creates a company rule using normalizeText(company)). The "source" dimension is no longer
// supported: source enable/disable is now the search.yaml enabled flag.
// The domain rule kind exists in blockMatch for a future explicit block-domain action but is
// never derived here.
func (s *Store) BlockJob(jobID int64, dimension string) (*BlockRule, error) {
	job, err := s.Get(jobID)
	if err != nil {
		return nil, err
	}
	var kind, value string
	switch dimension {
	case "company":
		kind = "company"
		value = normalizeText(job.Company)
	case "source":
		return nil, fmt.Errorf("source block rules have been retired; disable the source in search.yaml instead")
	default:
		return nil, fmt.Errorf("dimension must be \"company\"")
	}
	return s.AddBlockRule(kind, value)
}

// blockMatch checks whether a job matches any of the given block rules, returning (blocked, reason).
// Domain and company rules match a single field. Source rules are ignored — source control is now
// the search.yaml enabled flag and no source rules survive migration v14. The reason string takes
// the form "kind:value" for the first matching rule.
func blockMatch(job *model.Job, rules []*BlockRule) (bool, string) {
	if len(rules) == 0 {
		return false, ""
	}

	// Extract domain from original_url for domain rule matching.
	var jobDomain string
	if u, err := url.Parse(job.OriginalURL); err == nil {
		jobDomain = u.Hostname()
	}
	jobCompany := normalizeText(job.Company)

	// Check domain and company rules (any match hides the role).
	for _, r := range rules {
		switch r.Kind {
		case "domain":
			if jobDomain != "" && r.Value == jobDomain {
				return true, "domain:" + r.Value
			}
		case "company":
			if r.Value == jobCompany {
				return true, "company:" + r.Value
			}
		}
	}

	return false, ""
}

// SourceCount is one distinct source name present in the stored roles, with the number of roles
// sighted on it and whether an active source block rule covers it.
type SourceCount struct {
	Name    string `json:"name"`
	Roles   int    `json:"roles"`
	Blocked bool   `json:"blocked"`
}

// ListSources returns the distinct source names present in the sightings table, each with the
// number of distinct roles seen on it and a blocked flag from the active source rules,
// ordered by name.
func (s *Store) ListSources() ([]*SourceCount, error) {
	rules, err := s.ListBlockRules()
	if err != nil {
		return nil, err
	}
	blocked := map[string]bool{}
	for _, r := range rules {
		if r.Kind == "source" {
			blocked[r.Value] = true
		}
	}
	rows, err := s.DB.Query(
		`SELECT source, COUNT(DISTINCT job_id) FROM sightings GROUP BY source ORDER BY source`)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()
	out := []*SourceCount{}
	for rows.Next() {
		var c SourceCount
		if err := rows.Scan(&c.Name, &c.Roles); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		c.Blocked = blocked[c.Name]
		out = append(out, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sources: %w", err)
	}
	return out, nil
}
