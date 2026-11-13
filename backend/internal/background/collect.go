package background

import (
	"context"
	"log"
	"net/url"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/source"
	"github.com/w3con/sifty/internal/store"
)

// defaultCollect is the real daily-collection implementation. It iterates
// enabled sources from config, dispatches through the source registry, fetches
// through the guarded HTTP client, dedupes by canonical URL, upserts postings
// and enqueues them for assessment.
func defaultCollect(s *store.Store, d dirs.Dirs, cfg *config.Config) error {
	client, err := fetch.New(fetch.Options{})
	if err != nil {
		return err
	}

	reg := buildRegistry(client)
	seen := map[string]bool{}
	var total, skipped int
	stopWords := loadStopWords(s)
	vocab := ConfigSectionVocab(cfg.Sections)

	for _, site := range cfg.Search.Sites {
		if !site.IsEnabled() {
			continue
		}
		h := hostOf(site.URL)
		src := reg.For(h)

		refs, err := src.Discover(context.Background(), site.URL)
		if err != nil {
			log.Printf("collect: discover %q (%s): %v", site.Name, site.URL, err)
			continue
		}

		for _, ref := range refs {
			key, err := collect.Canonical(ref.URL)
			if err != nil {
				continue
			}
			if seen[key] {
				continue
			}
			seen[key] = true

			// A title already known from discovery lets a stop-word skip the posting before
			// its detail page is fetched; the resolved title is checked again below.
			if stoppedPosting(stopWords, ref.Partial.Title, ref.Partial.Company) {
				skipped++
				continue
			}

			j, err := src.Resolve(context.Background(), ref)
			if err != nil {
				log.Printf("collect: resolve %q: %v", ref.URL, err)
				continue
			}
			// Source name from config overrides any internal source label.
			j.Source = site.Name
			if j.DescriptionStatus == "" {
				j.DescriptionStatus = pendingStatus(j.Description)
			}

			if stoppedPosting(stopWords, j.Title, j.Company) {
				skipped++
				continue
			}

			if j.DescriptionHTML != "" {
				j.Sections = collect.SplitSections(j.DescriptionHTML, vocab)
			}

			if _, err := s.Upsert(j, false); err != nil {
				log.Printf("collect: upsert %q: %v", ref.URL, err)
				continue
			}
			total++
		}
	}

	log.Printf("collect: upserted %d postings, skipped %d by stop-word", total, skipped)
	return nil
}

// ConfigSectionVocab converts a config.SectionsDef into the []collect.SectionClass
// slice expected by the section splitter. The two packages are kept independent by
// doing this conversion in the layer that imports both. It is exported so serve.go
// can build the vocab once and thread it through the enrich path.
func ConfigSectionVocab(def config.SectionsDef) []collect.SectionClass {
	result := make([]collect.SectionClass, len(def.Sections))
	for i, e := range def.Sections {
		result[i] = collect.SectionClass{
			Label:    e.Class,
			Headings: e.Headings,
			Prefix:   e.Prefix,
			Contains: e.Contains,
		}
	}
	return result
}

// expressionsToFTSGroups converts config search expressions to store query groups.
func expressionsToFTSGroups(exprs []config.SearchExpression) []store.SearchGroup {
	groups := make([]store.SearchGroup, len(exprs))
	for i, e := range exprs {
		groups[i] = store.SearchGroup{Or: e.Or, And: e.And}
	}
	return groups
}

// buildRegistry creates a source registry wired with all adapters.
func buildRegistry(client *fetch.Client) *source.Registry {
	generic := source.NewGenericSource(client)
	reg := source.NewRegistry(generic)

	// ATS platform adapters — matched by host pattern before the generic fallback.
	reg.Register(`(?:boards|job-boards)\.greenhouse\.io`, source.NewGreenhouseAdapter(client))
	reg.Register(`jobs\.lever\.co`, source.NewLeverAdapter(client))
	reg.Register(`jobs\.ashbyhq\.com`, source.NewAshbyAdapter(client))

	// Source-specific adapters that override the generic waterfall for known
	// data-quality reasons.
	//
	// web3.career's JSON-LD description is damaged (stripped apostrophes,
	// double-encoded entities); reading the page body is required.
	reg.Register(source.Web3CareerPattern, source.NewWeb3CareersSource(client))
	// remote3.co listing page mixes category links (/jobs/<cat>) with vacancy
	// links (/remote-jobs/<slug>); the adapter filters to vacancy links only and
	// reads the page body (no JSON-LD).
	reg.Register(source.Remote3Pattern, source.NewRemote3Source(client))
	// weworkremotely.com job pages are blocked by Cloudflare; the adapter
	// discovers via the RSS feed, which carries entity-escaped HTML descriptions.
	reg.Register(source.WWRPattern, source.NewWWRSource(client))

	return reg
}

// hostOf extracts the hostname from a raw URL, returning the full URL on error.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Hostname()
}

// pendingStatus returns a description-status string for a posting whose
// description is absent; it distinguishes the missing state from the pending state.
func pendingStatus(description string) string {
	if description == "" {
		return "pending detail fetch"
	}
	return model.JobInput{}.DescriptionStatus // zero value — caller sets this
}

// collectSources is the testable core of defaultCollect. It accepts an
// already-built registry and store so tests can inject both. matchQuery is the
// FTS5 MATCH expression built from the candidate profile; it is passed to
// Rescore after all postings are upserted. vocab is the section heading
// vocabulary; pass nil to skip section computation (used in tests).
func collectSources(ctx context.Context, s *store.Store, sites []config.Source, reg *source.Registry, matchQuery string, vocab []collect.SectionClass) (int, error) {
	seen := map[string]bool{}
	var total, skipped int
	stopWords := loadStopWords(s)

	for _, site := range sites {
		if !site.IsEnabled() {
			continue
		}
		h := hostOf(site.URL)
		src := reg.For(h)

		refs, err := src.Discover(ctx, site.URL)
		if err != nil {
			log.Printf("collect: discover %q (%s): %v", site.Name, site.URL, err)
			continue
		}

		for _, ref := range refs {
			key, err := collect.Canonical(ref.URL)
			if err != nil {
				continue
			}
			if seen[key] {
				continue
			}
			seen[key] = true

			// A title already known from discovery lets a stop-word skip the posting before
			// its detail page is fetched; the resolved title is checked again below.
			if stoppedPosting(stopWords, ref.Partial.Title, ref.Partial.Company) {
				skipped++
				continue
			}

			j, err := src.Resolve(ctx, ref)
			if err != nil {
				log.Printf("collect: resolve %q: %v", ref.URL, err)
				continue
			}
			j.Source = site.Name
			if j.DescriptionStatus == "" {
				j.DescriptionStatus = "pending detail fetch"
			}

			if stoppedPosting(stopWords, j.Title, j.Company) {
				skipped++
				continue
			}

			if j.DescriptionHTML != "" && vocab != nil {
				j.Sections = collect.SplitSections(j.DescriptionHTML, vocab)
			}

			if _, err := s.Upsert(j, false); err != nil {
				log.Printf("collect: upsert %q: %v", ref.URL, err)
				continue
			}
			total++
		}
	}

	if skipped > 0 {
		log.Printf("collect: skipped %d postings by stop-word", skipped)
	}
	if err := s.Rescore(matchQuery); err != nil {
		log.Printf("collect: rescore: %v", err)
	}
	return total, nil
}

// loadStopWords reads the active stop-words once per collection run. A failure to read them is
// logged and collection proceeds unfiltered rather than aborting the run.
func loadStopWords(s *store.Store) []*store.StopWord {
	words, err := s.ListStopWords()
	if err != nil {
		log.Printf("collect: load stop-words: %v (collecting without the stop-word filter)", err)
		return nil
	}
	return words
}

// stoppedPosting reports whether a posting matches an active exclusion term. The check tests
// both title and company via RoleMatchesExclusions. An empty title and company never matches.
func stoppedPosting(words []*store.StopWord, title, company string) bool {
	if title == "" && company == "" {
		return false
	}
	hit, _ := store.RoleMatchesExclusions(title, company, words)
	return hit
}
