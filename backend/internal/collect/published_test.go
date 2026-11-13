package collect

import "testing"

// TestParseFeedPublicationIgnoresUpdated catches the hidden failure where a
// source update timestamp is presented as a fresh publication date.
func TestParseFeedPublicationIgnoresUpdated(t *testing.T) {
	xml := `<rss><channel>
	  <item><title>Acme: RSS pubDate</title><link>https://acme.test/jobs/rss</link>
	    <pubDate>Fri, 18 Jul 2025 12:34:56 +0000</pubDate>
	    <updated>2026-11-01T00:00:00Z</updated></item>
	  <item><title>Acme: update only</title><link>https://acme.test/jobs/update</link>
	    <updated>2026-11-01T00:00:00Z</updated></item>
	  <item><title>Acme: Atom published</title><link>https://acme.test/jobs/atom</link>
	    <published>2025-06-01T00:00:00Z</published>
	    <updated>2026-11-01T00:00:00Z</updated></item>
	</channel></rss>`
	jobs := ParseFeed(xml, "https://acme.test/feed", "Feed")
	if len(jobs) != 3 {
		t.Fatalf("got %d jobs, want 3", len(jobs))
	}
	if jobs[0].PublishedAt != "2025-07-18T12:34:56Z" {
		t.Errorf("pubDate = %q", jobs[0].PublishedAt)
	}
	if jobs[1].PublishedAt != "" {
		t.Errorf("update-only item publishedAt = %q, want empty", jobs[1].PublishedAt)
	}
	if jobs[2].PublishedAt != "2025-06-01T00:00:00Z" {
		t.Errorf("Atom published = %q", jobs[2].PublishedAt)
	}
}
