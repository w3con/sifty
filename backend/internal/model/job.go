// Package model holds the core data types shared across collection, assessment and preparation.
package model

import "github.com/w3con/sifty/internal/matchcontract"

// JobInput is a collected posting before it is stored. Its JSON field names match the current
// TypeScript application, so a posting is interchangeable between the two.
type JobInput struct {
	Title             string   `json:"title"`
	Company           string   `json:"company"`
	BoardCompany      string   `json:"boardCompany,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	OriginalURL       string   `json:"originalUrl"`
	ApplyURL          *string  `json:"applyUrl"`
	ApplyStatus       string   `json:"applyStatus,omitempty"`
	Description       string   `json:"description"`
	DescriptionStatus string   `json:"descriptionStatus"`
	Location          string   `json:"location,omitempty"`
	PublishedAt       string   `json:"publishedAt,omitempty"`
	PublishedBasis    string   `json:"publishedBasis,omitempty"`
	ExpiresAt         string   `json:"expiresAt,omitempty"`
	Source            string   `json:"source"`
}

// JobSource is one (source-name, URL, seen-timestamp) triple attached to a role.
type JobSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Seen string `json:"seen"`
}

// Job is the full role record as returned to the interface. Field names and JSON keys match
// the TypeScript Job type so the React frontend can consume both backends unchanged.
type Job struct {
	ID                   int64               `json:"id"`
	Title                string              `json:"title"`
	Company              string              `json:"company"`
	Tags                 []string            `json:"tags"`
	Stage                string              `json:"stage"`
	Status               string              `json:"status"`
	Seen                 bool                `json:"seen"`
	SeenAt               *string             `json:"seenAt"`
	AppliedAt            *string             `json:"appliedAt"`
	OutcomeReason        *string             `json:"outcomeReason"`
	IsWeb3               bool                `json:"isWeb3"`
	OriginalURL          string              `json:"originalUrl"`
	ApplyURL             *string             `json:"applyUrl"`
	ApplyStatus          string              `json:"applyStatus"`
	Description          string              `json:"description"`
	DescriptionStatus    string              `json:"descriptionStatus"`
	Location             *string             `json:"location"`
	PublishedAt          *string             `json:"publishedAt"`
	PublishedBasis       *string             `json:"publishedBasis,omitempty"`
	ExpiresAt            *string             `json:"expiresAt"`
	FirstSeen            string              `json:"firstSeen"`
	LastSeen             string              `json:"lastSeen"`
	Expired              bool                `json:"expired"`
	MatchScore           int                 `json:"matchScore"`
	MatchProfileRevision int                 `json:"matchProfileRevision"`
	MatchStale           bool                `json:"matchStale"`
	QuickMatch           *matchcontract.View `json:"quickMatch,omitempty"`
	QuickMatchError      string              `json:"quickMatchError,omitempty"`
	Assessment           any                 `json:"assessment"`
	AssessmentError      *string             `json:"assessmentError"`
	Sources              []JobSource         `json:"sources"`
	Blocked              bool                `json:"blocked"`
	BlockedBy            string              `json:"blockedBy"`
	Stopped              bool                `json:"stopped"`
	StoppedBy            string              `json:"stoppedBy"`
}
