package collect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/w3con/sifty/internal/model"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}

func readGolden(t *testing.T, name string) []model.JobInput {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	var out []model.JobInput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse golden %s: %v", name, err)
	}
	return out
}

func assertParity(t *testing.T, label string, got, want []model.JobInput) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		gj, _ := json.MarshalIndent(got, "", "  ")
		wj, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("%s: Go output differs from the TypeScript golden\n--- got ---\n%s\n--- want ---\n%s", label, gj, wj)
	}
}

func TestStructuredJobsParity(t *testing.T) {
	got := StructuredJobs(readFixture(t, "jsonld.html"), "https://acme.test/jobs/cto", "Test Board")
	assertParity(t, "structuredJobs", got, readGolden(t, "jsonld.json"))
}

func TestParseFeedParity(t *testing.T) {
	got := ParseFeed(readFixture(t, "feed.xml"), "https://acme.test/feed", "Test Feed")
	assertParity(t, "parseFeed", got, readGolden(t, "feed.json"))
}

func TestHTMLLinksGetroParity(t *testing.T) {
	got := HTMLLinks(readFixture(t, "links_getro.html"), "https://gamma.test/", "Gamma Board")
	assertParity(t, "htmlLinks getro", got, readGolden(t, "links_getro.json"))
}

func TestHTMLLinksAnchorsParity(t *testing.T) {
	got := HTMLLinks(readFixture(t, "links_anchors.html"), "https://delta.test/careers", "Delta Board")
	assertParity(t, "htmlLinks anchors", got, readGolden(t, "links_anchors.json"))
}
