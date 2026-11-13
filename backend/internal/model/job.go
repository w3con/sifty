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
	DescriptionHTML   string   `json:"descriptionHtml,omitempty"`
	DescriptionStatus string   `json:"descriptionStatus"`
	Location          string   `json:"location,omitempty"`
	PublishedAt       string   `json:"publishedAt,omitempty"`
	PublishedBasis    string   `json:"publishedBasis,omitempty"`
	ExpiresAt         string   `json:"expiresAt,omitempty"`
	Source            string   `json:"source"`
	// Original is the raw page body or feed item as received. Never serialised to
	// JSON (json:"-"). Upsert writes it to job_originals in the same transaction as
	// the vacancy row. Leave nil when no body is available.
	Original     []byte `json:"-"`
	// OriginalKind identifies the format: "html/page" for a fetched page body,
	// "html/api" for an API-sourced content field, or "rss/item" for a feed item
	// description field.
	OriginalKind string `json:"-"`
	// Sections is the description split into labelled blocks. Never serialised to
	// JSON (json:"-"). Upsert writes it to job_sections in the same transaction as
	// the vacancy row. Leave nil when no description_html is available.
	Sections []Section `json:"-"`
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
	DescriptionHTML      *string             `json:"descriptionHtml,omitempty"`
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
	// Fit is the stored two-score result (Fits you, You fit and the four lists). Nil until the
	// vacancy has been scored.
	Fit             *matchcontract.FitView `json:"fit,omitempty"`
	Assessment      any                    `json:"assessment"`
	AssessmentError *string                `json:"assessmentError"`
	Sources         []JobSource            `json:"sources"`
	Blocked         bool                   `json:"blocked"`
	BlockedBy       string                 `json:"blockedBy"`
	Stopped         bool                   `json:"stopped"`
	StoppedBy       string                 `json:"stoppedBy"`
	// NeedsRefetch is true when the stored description is a glued single-paragraph
	// HTML fragment (collected before structured description support) and the full
	// page should be refetched. The frontend triggers one refetch per opening.
	NeedsRefetch    bool                   `json:"needsRefetch,omitempty"`
	// Sections is the description split into named blocks by heading. Present only
	// in the single-job response (GET /api/jobs/{id}) to keep the list response small.
	Sections        []Section              `json:"sections,omitempty"`
	// Band is the Fits you band with one tile per profile goal. Present only in the
	// single-job response.
	Band            *Band                  `json:"band,omitempty"`
	// WhatTheyOffer is the offer vocabulary match from the Compensation and Benefits
	// sections. Present only in the single-job response.
	WhatTheyOffer   []OfferItem            `json:"whatTheyOffer,omitempty"`
}

// Section is one block of a vacancy description after splitting by headings.
// Heading is empty for the content that precedes the first heading.
// Class is one of the vocabulary labels (About, Role, Responsibilities, …, other).
// Content is the sanitised HTML fragment of this block.
type Section struct {
	Heading string `json:"heading,omitempty"`
	Class   string `json:"class"`
	Content string `json:"content"`
}

// BandTile is one goal tile in the Fits you band.
type BandTile struct {
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
	// Hint is the short description line shown below the value (e.g. "above your minimum").
	Hint   string `json:"hint,omitempty"`
	Origin string `json:"origin,omitempty"`
	// Verdict controls the tile colour and verdict circle:
	//   "ok"         green tile, ✓ circle
	//   "mismatch"   amber tile, ✕ circle
	//   "unclear"    neutral tile, ? circle (money range straddles minimum)
	//   "not_stated" neutral tile, ? circle (goal exists but job does not state the fact)
	//   ""           neutral tile, no circle (Location: no verdict computed yet)
	Verdict string `json:"verdict"`
	// ApproxValue is the converted amount shown for the Money tile (e.g. "≈ €230,000").
	ApproxValue string `json:"approxValue,omitempty"`
}

// Band is the Fits you band returned in the single-vacancy response.
type Band struct {
	Tiles []BandTile `json:"tiles"`
}

// OfferItem is one offer tile in the What they offer card.
type OfferItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Value is the stated value extracted from the listing text, or empty when not stated.
	Value string `json:"value,omitempty"`
}
