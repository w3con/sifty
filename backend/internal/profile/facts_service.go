package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/w3con/sifty/internal/store"
)

// ListFacts returns facts across the profile or for one document. Facts never become profile
// values without an explicit accept/resolve action.
func (s *Service) ListFacts(documentID string) ([]*Fact, error) {
	current, err := s.Get()
	if err != nil {
		return nil, err
	}
	if documentID != "" {
		doc, err := s.store.GetProfileDocument(documentID)
		if err != nil {
			return nil, err
		}
		if doc.ProfileID != current.ID {
			return nil, fmt.Errorf("document belongs to another profile")
		}
	}
	records, err := s.store.ListProfileFacts(current.ID, documentID)
	if err != nil {
		return nil, err
	}
	docs, err := s.store.ListProfileDocuments(current.ID)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, doc := range docs {
		names[doc.ID] = doc.OriginalName
	}
	out := make([]*Fact, 0, len(records))
	for _, rec := range records {
		name := ""
		if rec.DocumentID != nil {
			name = names[*rec.DocumentID]
		}
		out = append(out, factFromRecord(rec, name))
	}
	return out, nil
}

// AcceptFact applies a proposed fact when expectedRevision is current. A different known value
// is stored as a visible conflict and the profile is left untouched; the user resolves it with
// ResolveFact. The accepted fact state, new profile revision and history row are one transaction.
func (s *Service) AcceptFact(factID string, expected int) (*FactDecision, error) {
	return s.acceptFact(factID, expected, false)
}

// ResolveFact is the explicit choice after a conflict. "proposed" forces the document value,
// "current" keeps the stored profile and rejects the proposal.
func (s *Service) ResolveFact(factID string, expected int, choice string) (*FactDecision, error) {
	switch choice {
	case "proposed":
		return s.acceptFact(factID, expected, true)
	case "current":
		current, err := s.Get()
		if err != nil {
			return nil, err
		}
		if expected != current.Revision {
			return nil, &ConflictError{Expected: expected, Current: current}
		}
		fact, err := s.store.RejectProfileFact(factID)
		if err != nil {
			return nil, err
		}
		return &FactDecision{Fact: factFromRecord(fact, ""), Profile: current, ProfileChanged: false}, nil
	default:
		return nil, fmt.Errorf("choice must be \"proposed\" or \"current\"")
	}
}

func (s *Service) acceptFact(factID string, expected int, force bool) (*FactDecision, error) {
	current, err := s.Get()
	if err != nil {
		return nil, err
	}
	if expected != current.Revision {
		return nil, &ConflictError{Expected: expected, Current: current}
	}
	rec, err := s.store.GetProfileFact(factID)
	if err != nil {
		return nil, err
	}
	if rec.ProfileID != current.ID {
		return nil, fmt.Errorf("fact belongs to another profile")
	}
	if rec.State != FactProposed && rec.State != FactStateConflict {
		return nil, store.ErrProfileFactResolved
	}
	canonical, err := ValidateFactValue(rec.FieldKey, json.RawMessage(rec.ValueJSON))
	if err != nil {
		return nil, err
	}
	data := current.Data
	changed, conflict, err := ApplyFact(&data, rec.FieldKey, canonical, force)
	if err != nil {
		return nil, err
	}
	if conflict != nil {
		conflict.FactID = factID
		conflict.CurrentRevision = expected
		if _, err := s.store.SetProfileFactStateCAS(factID, expected, []string{FactProposed, FactStateConflict}, FactStateConflict); err != nil {
			return nil, mapFactCASError(err, expected)
		}
		return nil, &FactConflictError{Conflict: conflict}
	}
	if !changed {
		updated, err := s.store.SetProfileFactStateCAS(factID, expected, []string{FactProposed, FactStateConflict}, FactAccepted)
		if err != nil {
			return nil, mapFactCASError(err, expected)
		}
		return &FactDecision{Fact: factFromRecord(updated, ""), Profile: current, ProfileChanged: false}, nil
	}
	if rec.FieldKey == "skills" {
		setAcceptedSkillProvenance(&data, canonical, rec)
	}

	data.Normalize()
	if err := data.Validate(); err != nil {
		return nil, fmt.Errorf("fact would make the profile invalid: %w", err)
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal profile after fact: %w", err)
	}
	profileRec, factRec, err := s.store.AcceptProfileFact(
		factID, expected, string(encoded), "accepted profile fact", "fact", FactAccepted)
	if err != nil {
		var rev *store.RevisionConflictError
		if errors.As(err, &rev) {
			latest, decodeErr := decodeProfile(rev.Current)
			if decodeErr != nil {
				return nil, decodeErr
			}
			return nil, &ConflictError{Expected: expected, Current: latest}
		}
		return nil, err
	}
	saved, err := decodeProfile(profileRec)
	if err != nil {
		return nil, err
	}
	return &FactDecision{
		Fact: factFromRecord(factRec, ""), Profile: s.syncAfterSave(saved), ProfileChanged: true,
	}, nil
}

func mapFactCASError(err error, expected int) error {
	var rev *store.RevisionConflictError
	if errors.As(err, &rev) {
		latest, decodeErr := decodeProfile(rev.Current)
		if decodeErr != nil {
			return decodeErr
		}
		return &ConflictError{Expected: expected, Current: latest}
	}
	return err
}

func setAcceptedSkillProvenance(data *ProfileData, value json.RawMessage, rec *store.ProfileFactRecord) {
	var proposed Skill
	if json.Unmarshal(value, &proposed) != nil {
		return
	}
	prov := &Provenance{Kind: "document", Reference: rec.SourceReference, Quote: rec.Quote}
	if rec.DocumentID == nil {
		prov = &Provenance{Kind: "fact", Reference: rec.ID}
	}
	for i := range data.Skills {
		if data.Skills[i].Key == proposed.Key || strings.EqualFold(data.Skills[i].Name, proposed.Name) {
			data.Skills[i].Provenance = prov
			return
		}
	}
}

// RejectFact marks a proposal rejected without touching the profile.
func (s *Service) RejectFact(factID string) (*FactDecision, error) {
	rec, err := s.store.RejectProfileFact(factID)
	if err != nil {
		return nil, err
	}
	current, err := s.Get()
	if err != nil {
		return nil, err
	}
	return &FactDecision{Fact: factFromRecord(rec, ""), Profile: current, ProfileChanged: false}, nil
}
