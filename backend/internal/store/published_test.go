package store_test

import "testing"

// TestUpsertFillsAndPreservesPublishedAt catches silent loss of a genuine
// source date on re-collection and accidental rejuvenation of an established
// publication date. Discovery/workflow fields must survive both updates.
func TestUpsertFillsAndPreservesPublishedAt(t *testing.T) {
	s := openFresh(t)
	in := fixtureJobInput("publication-source")
	in.PublishedAt = ""

	id, err := s.Upsert(in, false)
	if err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	first, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get first: %v", err)
	}
	if first.PublishedAt != nil {
		t.Fatalf("new job with no source date has publishedAt %q", *first.PublishedAt)
	}

	in.PublishedAt = "2025-07-18"
	if got, err := s.Upsert(in, false); err != nil || got != id {
		t.Fatalf("fill Upsert id=%d err=%v, want id=%d", got, err, id)
	}
	filled, _ := s.Get(id)
	if filled.PublishedAt == nil || *filled.PublishedAt != "2025-07-18" {
		t.Fatalf("missing date was not filled: %v", filled.PublishedAt)
	}
	if filled.FirstSeen != first.FirstSeen {
		t.Errorf("firstSeen changed on re-collection: %q -> %q", first.FirstSeen, filled.FirstSeen)
	}

	in.PublishedAt = "2025-06-01"
	if _, err := s.Upsert(in, false); err != nil {
		t.Fatalf("preserve Upsert: %v", err)
	}
	kept, _ := s.Get(id)
	if kept.PublishedAt == nil || *kept.PublishedAt != "2025-07-18" {
		t.Errorf("established publication date was replaced: %v", kept.PublishedAt)
	}

	in.PublishedAt = "not-a-date"
	if _, err := s.Upsert(in, false); err != nil {
		t.Fatalf("invalid Upsert: %v", err)
	}
	invalid, _ := s.Get(id)
	if invalid.PublishedAt == nil || *invalid.PublishedAt != "2025-07-18" {
		t.Errorf("invalid incoming date cleared established publication: %v", invalid.PublishedAt)
	}
}

// TestUpsertNormalizesWeb3PublicationFormat catches a valid source date being
// discarded because its zone had no separating space.
func TestUpsertNormalizesWeb3PublicationFormat(t *testing.T) {
	s := openFresh(t)
	in := fixtureJobInput("web3-source")
	in.PublishedAt = "2026-10-28 05:10:29+07:00"
	id, err := s.Upsert(in, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, _ := s.Get(id)
	if got.PublishedAt == nil || *got.PublishedAt != "2026-10-27T22:10:29Z" {
		t.Fatalf("Web3 publication lost: %v", got.PublishedAt)
	}
}

func TestUpsertAssociatesPublicationBasisWithoutInventingHistory(t *testing.T) {
	s := openFresh(t)
	in := fixtureJobInput("basis-source")
	in.PublishedAt = ""
	in.PublishedBasis = ""
	id, err := s.Upsert(in, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	in.PublishedAt = "2025-07-18"
	if _, err := s.Upsert(in, false); err != nil {
		t.Fatalf("legacy date Upsert: %v", err)
	}
	legacy, _ := s.Get(id)
	if legacy.PublishedBasis != nil {
		t.Fatalf("legacy row invented a basis: %v", *legacy.PublishedBasis)
	}
	in.PublishedBasis = "datePosted"
	if _, err := s.Upsert(in, false); err != nil {
		t.Fatalf("same-date basis Upsert: %v", err)
	}
	known, _ := s.Get(id)
	if known.PublishedBasis == nil || *known.PublishedBasis != "datePosted" {
		t.Fatalf("matching known source did not retain its basis: %v", known.PublishedBasis)
	}
	in.PublishedAt = "2025-06-01"
	in.PublishedBasis = "pubDate"
	if _, err := s.Upsert(in, false); err != nil {
		t.Fatalf("different-date Upsert: %v", err)
	}
	kept, _ := s.Get(id)
	if kept.PublishedAt == nil || *kept.PublishedAt != "2025-07-18" || kept.PublishedBasis == nil || *kept.PublishedBasis != "datePosted" {
		t.Fatalf("different source stamped a new origin: date=%v basis=%v", kept.PublishedAt, kept.PublishedBasis)
	}
}
