package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/w3con/sifty/internal/candidate"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/extract"
	"github.com/w3con/sifty/internal/store"
)

// Profile is the API-facing envelope for one stored profile revision.
type Profile struct {
	ID        string      `json:"profileId"`
	Revision  int         `json:"revision"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	Data      ProfileData `json:"data"`

	// Derived status. The confirmed profile is authoritative; the candidate context and fast
	// ranking are projections of one revision. These fields report whether those projections
	// are known to match the current revision, so a failed or raced refresh cannot be presented
	// as current.
	ContextVersion         string `json:"contextVersion,omitempty"`
	ContextProfileRevision int    `json:"contextProfileRevision,omitempty"`
	ContextCurrent         bool   `json:"contextCurrent"`
	ContextError           string `json:"contextError,omitempty"`

	FastRankingRevision int    `json:"fastRankingRevision,omitempty"`
	FastRankingCurrent  bool   `json:"fastRankingCurrent"`
	FastRankingError    string `json:"fastRankingError,omitempty"`
	DerivedCurrent      bool   `json:"derivedCurrent"`
}

// ConflictError reports a save whose expectedRevision no longer matches the stored profile.
// Current carries the stored profile so the caller can present it without a second request.
type ConflictError struct {
	Expected int
	Current  *Profile
}

func (e *ConflictError) Error() string {
	cur := 0
	if e.Current != nil {
		cur = e.Current.Revision
	}
	return fmt.Sprintf("profile revision conflict: expected %d, current %d", e.Expected, cur)
}

// Service reads and saves the single local profile. It owns the one-time non-destructive
// bootstrap from legacy search preferences and candidate context; it never mutates those
// sources and never schedules assessment work.
type Service struct {
	store *store.Store
	cfg   *config.Holder
	dirs  dirs.Dirs
	docx  extract.DocxReader

	startMu sync.Mutex
	runsMu  sync.Mutex
	runs    map[string]activeRun
}

// NewService builds the profile application service from its persistence and configuration
// dependencies. cfg may be nil for tests that do not exercise bootstrap. Document storage and
// DOCX extraction are supplied as options where needed.
func NewService(st *store.Store, cfg *config.Holder, opts ...Option) *Service {
	s := &Service{store: st, cfg: cfg}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Get returns the current profile, bootstrapping it once from legacy preferences/context when
// no profile exists. Bootstrap is lazy, additive and does not alter configuration or context.
func (s *Service) Get() (*Profile, error) {
	rec, err := s.store.CurrentProfile()
	if err != nil {
		return nil, err
	}
	if rec == nil {
		data, err := s.bootstrapData()
		if err != nil {
			return nil, err
		}
		data.Normalize()
		encoded, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshal bootstrap profile: %w", err)
		}
		rec, err = s.store.CreateProfile(LocalProfileID, string(encoded))
		if err != nil {
			return nil, err
		}
	}
	current, err := decodeProfile(rec)
	if err != nil {
		return nil, err
	}
	return s.decorate(current), nil
}

// Update writes a new revision when expected matches the stored revision. Legacy bootstrap
// provenance (legacyPreferences, legacyContextVersion) is server-owned: it is copied from the
// stored profile and never accepted from the request, so an editor that omits those fields
// cannot drop them and a client cannot forge them. A stale expected revision returns
// *ConflictError carrying the current profile; nothing is overwritten.
func (s *Service) Update(expected int, data ProfileData) (*Profile, error) {
	currentRec, err := s.store.CurrentProfile()
	if err != nil {
		return nil, err
	}
	if currentRec == nil {
		return nil, fmt.Errorf("profile not found")
	}
	current, err := decodeProfile(currentRec)
	if err != nil {
		return nil, err
	}
	data.LegacyPreferences = current.Data.LegacyPreferences
	data.LegacyContextVersion = current.Data.LegacyContextVersion

	data.Normalize()
	markManualSkillProvenance(&data, current.Data.Skills)
	if err := data.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal profile: %w", err)
	}
	rec, err := s.store.SaveProfile(LocalProfileID, expected, string(encoded), "profile update", "manual")
	if err != nil {
		var conflict *store.RevisionConflictError
		if errors.As(err, &conflict) {
			current, decodeErr := decodeProfile(conflict.Current)
			if decodeErr != nil {
				return nil, decodeErr
			}
			return nil, &ConflictError{Expected: expected, Current: s.decorate(current)}
		}
		return nil, err
	}
	saved, err := decodeProfile(rec)
	if err != nil {
		return nil, err
	}
	return s.syncAfterSave(saved), nil
}

// syncAfterSave refreshes the derived candidate context and the catalogue fast scores for the
// exact saved revision. The profile row is already committed, so a projection failure never
// reports the save as failed; it marks the derived status stale instead. No model call or
// network fetch happens here.
func (s *Service) syncAfterSave(p *Profile) *Profile {
	contextErr, rankingErr := s.refreshDerived(p)
	out := s.decorate(p)
	if contextErr != nil {
		out.ContextCurrent = false
		out.ContextError = contextErr.Error()
	}
	if rankingErr != nil {
		out.FastRankingCurrent = false
		out.FastRankingError = rankingErr.Error()
	}
	out.DerivedCurrent = out.ContextCurrent && out.FastRankingCurrent
	return out
}

// SyncDerivedContext makes the derived context and fast ranking reflect the current stored
// profile. Explicit assessment and preparation call it before queueing so they never silently
// use an older profile snapshot. It performs only local reads and database writes.
func (s *Service) SyncDerivedContext() (*Profile, error) {
	rec, err := s.store.CurrentProfile()
	if err != nil || rec == nil {
		return nil, err
	}
	current, err := decodeProfile(rec)
	if err != nil {
		return nil, err
	}
	contextErr, rankingErr := s.refreshDerived(current)
	out := s.decorate(current)
	if contextErr != nil {
		out.ContextCurrent = false
		out.ContextError = contextErr.Error()
	}
	if rankingErr != nil {
		out.FastRankingCurrent = false
		out.FastRankingError = rankingErr.Error()
	}
	out.DerivedCurrent = out.ContextCurrent && out.FastRankingCurrent
	if contextErr != nil {
		return out, contextErr
	}
	if rankingErr != nil {
		return out, rankingErr
	}
	return out, nil
}

// refreshDerived rebuilds the CandidateContext from configured local sources plus the confirmed
// profile, then recomputes every role's existing BM25 fast score and stamps the profile revision
// that produced it. localOnly=true prevents remote profile fetches during a save.
func (s *Service) refreshDerived(p *Profile) (contextErr, rankingErr error) {
	var cfg *config.Config
	if s.cfg != nil {
		cfg = s.cfg.Get()
	}
	_, err := candidate.RefreshContextForProfile(
		s.store, s.dirs, cfg, ContextPreferences(p), p.ID, p.Revision, false, true, ContextSource(p))
	if err != nil {
		return fmt.Errorf("refresh derived candidate context: %w", err), nil
	}
	if err := s.store.RescoreForProfile(FastRankingQuery(cfg, &p.Data), p.Revision); err != nil {
		return nil, fmt.Errorf("refresh catalogue fast ranking: %w", err)
	}
	return nil, nil
}

// decorate reports whether the latest derived context snapshot belongs to this profile revision.
func (s *Service) decorate(p *Profile) *Profile {
	if p == nil {
		return nil
	}
	raw, err := s.store.Context()
	if err != nil {
		p.ContextCurrent = false
		p.ContextError = err.Error()
		return p
	}
	if raw == nil {
		p.ContextCurrent = false
		p.ContextError = "candidate context has not been refreshed"
		return p
	}
	version, _ := raw["version"].(string)
	profileID, _ := raw["profileId"].(string)
	revision := intValue(raw["profileRevision"])
	p.ContextVersion = version
	p.ContextProfileRevision = revision
	p.ContextCurrent = profileID == p.ID && revision == p.Revision
	if p.ContextCurrent {
		p.ContextError = ""
	} else if profileID == "" {
		p.ContextError = "stored context predates the confirmed profile; refresh context"
	} else if revision != p.Revision {
		p.ContextError = fmt.Sprintf("stored context reflects profile revision %d, current is %d", revision, p.Revision)
	} else {
		p.ContextError = "stored context belongs to a different profile"
	}

	rankRevision, known, uniform, rankErr := s.store.FastRankingRevision()
	p.FastRankingRevision = rankRevision
	switch {
	case rankErr != nil:
		p.FastRankingCurrent = false
		p.FastRankingError = rankErr.Error()
	case !known:
		p.FastRankingCurrent = false
		p.FastRankingError = "no roles have been ranked yet"
	case !uniform:
		p.FastRankingCurrent = false
		p.FastRankingError = "roles carry mixed profile revisions; rescore the catalogue"
	case rankRevision != p.Revision:
		p.FastRankingCurrent = false
		p.FastRankingError = fmt.Sprintf("fast scores reflect profile revision %d, current is %d", rankRevision, p.Revision)
	default:
		p.FastRankingCurrent = true
		p.FastRankingError = ""
	}
	p.DerivedCurrent = p.ContextCurrent && p.FastRankingCurrent
	return p
}

func intValue(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

// markManualSkillProvenance keeps the server-owned provenance for an unchanged skill and marks a
// changed or new skill as manual. Client-supplied provenance is ignored so an old document quote
// cannot appear to prove a manually changed level.
func markManualSkillProvenance(data *ProfileData, current []Skill) {
	for i := range data.Skills {
		incoming := data.Skills[i]
		var provenance *Provenance
		for _, old := range current {
			if old.Key == incoming.Key || strings.EqualFold(old.Name, incoming.Name) {
				if skillEqual(old, incoming) {
					provenance = old.Provenance
				}
				break
			}
		}
		if provenance == nil {
			provenance = &Provenance{Kind: "manual"}
		}
		data.Skills[i].Provenance = provenance
	}
}

func decodeProfile(rec *store.ProfileRecord) (*Profile, error) {
	if rec == nil {
		return nil, fmt.Errorf("profile record is nil")
	}
	var data ProfileData
	if err := json.Unmarshal([]byte(rec.Data), &data); err != nil {
		return nil, fmt.Errorf("parse stored profile: %w", err)
	}
	data.Normalize()
	return &Profile{
		ID:        rec.ID,
		Revision:  rec.Revision,
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
		Data:      data,
	}, nil
}
