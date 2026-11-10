package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/candidate"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/records"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/version"
)

// Deps holds the dependencies shared across all route handlers.
type Deps struct {
	Store *store.Store
	Dirs  dirs.Dirs
	Cfg   *config.Holder

	// FetchClient is the outbound HTTP client for import and enrich paths. If nil,
	// a default client is created on first use.
	FetchClient *fetch.Client

	// Enrich re-fetches and enriches a job's detail page. Tests can inject a stub.
	// If nil, the real EnrichJob implementation is used.
	EnrichFn func(s *store.Store, id int64) (int64, error)

	// InferenceFn is the provider-configurable model call. If nil, assess.Infer is used.
	InferenceFn assess.Inference

	// CollectFn is called by POST /api/search to start a (possibly throttled) collection. It
	// returns the runID (0 when skipped), the status ("running" or "skipped"), and any setup
	// error. When nil, POST /api/search falls back to the legacy behaviour of recording a run
	// row and returning "running" without collecting anything.
	CollectFn func(source string, force bool) (runID int64, status string, err error)

	// Profiles is the profile application service. When nil, RegisterRoutes builds one from
	// Store and Cfg so existing callers do not need to wire it explicitly.
	Profiles *profile.Service
}

// exclusive serialises async operations so at most one runs at a time.
type exclusive struct{ mu sync.Mutex }

// Run acquires the lock, runs f, and releases it. It returns an error if f panics.
func (e *exclusive) Run(f func() (any, error)) (any, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return f()
}

// Router is the subset of *http.ServeMux used by RegisterRoutes, allowing the drift test to
// substitute a recording wrapper without changing production call sites.
type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// RegisterRoutes registers all endpoint handlers onto mux, replacing the catch-all for known paths.
func RegisterRoutes(mux Router, d *Deps) {
	ex := &exclusive{}

	// ------------------------------------------------------------------
	// Phase 4.1 — Roles
	// ------------------------------------------------------------------

	// Build identity. Deploy and preview use this to verify that the process
	// answering on a port is the generation they just built, rather than a
	// stale process or a different environment.
	mux.HandleFunc("GET /api/version", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		writeJSON(w, http.StatusOK, map[string]string{"version": version.Version})
		return nil
	}))

	mux.HandleFunc("GET /api/jobs", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "all"
		}
		q := r.URL.Query().Get("q")
		jobs, err := d.Store.List(status, q)
		if err != nil {
			return err
		}
		if jobs == nil {
			jobs = []*model.Job{}
		}
		writeJSON(w, http.StatusOK, jobs)
		return nil
	}))

	mux.HandleFunc("GET /api/jobs/{id}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		job, err := d.Store.Get(id)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, job)
		return nil
	}))

	mux.HandleFunc("PATCH /api/jobs/{id}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var body struct {
			Stage     string `json:"stage"`
			Status    string `json:"status"`
			Confirmed bool   `json:"confirmed"`
			Reason    string `json:"reason"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		stageOrStatus := body.Stage
		if stageOrStatus == "" {
			stageOrStatus = body.Status
		}
		// Capture the stage before the transition so we can detect a fresh pre-selection.
		prevJob, _ := d.Store.Get(id)
		prevStage := ""
		if prevJob != nil {
			prevStage = prevJob.Stage
		}
		job, err := d.Store.StatusWithReason(id, stageOrStatus, body.Confirmed, body.Reason)
		if err != nil {
			return err
		}
		// If the job just transitioned to preselected for the first time, enqueue a
		// model-free inspect operation so the form questions are ready when the user
		// opens the Application tab.
		if job.Stage == "preselected" && prevStage != "preselected" {
			_, _ = d.Store.QueuePreparation(job.ID, map[string]any{"inspect": true, "generate": false})
		}
		writeJSON(w, http.StatusOK, job)
		return nil
	}))

	mux.HandleFunc("DELETE /api/jobs/{id}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var body struct {
			Confirmed bool `json:"confirmed"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		if !body.Confirmed {
			return fmt.Errorf("Permanent deletion requires explicit confirmation")
		}
		deleted, cleanupErrors, err := d.Store.Delete(id)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted":       deleted,
			"cleanupErrors": cleanupErrors,
		})
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/block", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var body struct {
			Dimension string `json:"dimension"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		rule, err := d.Store.BlockJob(id, body.Dimension)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, rule)
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/seen", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		job, err := d.Store.Seen(id)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, job)
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/read", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		enrichFn := d.EnrichFn
		if enrichFn == nil {
			// Default: re-fetch the detail page and update the description.
			enrichFn = func(s *store.Store, id int64) (int64, error) {
				return EnrichJob(s, d.FetchClient, id)
			}
		}
		result, err := ex.Run(func() (any, error) {
			return enrichFn(d.Store, id)
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": result})
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/assess", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var body struct {
			Provider string `json:"provider"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		// An explicit assessment must use the current confirmed profile, even if an earlier
		// projection failed or a concurrent save left the stored context one revision behind.
		if _, err := d.profileService().SyncDerivedContext(); err != nil {
			return fmt.Errorf("refresh profile context before assessment: %w", err)
		}
		job, err := d.Store.Get(id)
		if err != nil {
			return err
		}
		fp := jobVersionHash(job)
		row, err := d.Store.EnqueueJob(job.ID, fp, body.Provider, 100)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusAccepted, row)
		return nil
	}))

	// ------------------------------------------------------------------
	// Block rules
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/block-rules", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		rules, err := d.Store.ListBlockRules()
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, rules)
		return nil
	}))

	mux.HandleFunc("POST /api/block-rules", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		// Source block rules have been retired in migration v14. Source enable/disable is now
		// managed through the search.yaml sites list (GET/POST/DELETE/PATCH /api/search/sites).
		return fmt.Errorf("source block rules have been retired; use /api/search/sites to manage sources")
	}))

	mux.HandleFunc("GET /api/sources", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		sources, err := d.Store.ListSources()
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, sources)
		return nil
	}))

	mux.HandleFunc("DELETE /api/block-rules/{id}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err := d.Store.RemoveBlockRule(id); err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"removed": id})
		return nil
	}))

	// ------------------------------------------------------------------
	// Stop-words
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/stop-words", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		words, err := d.Store.ListStopWords()
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, words)
		return nil
	}))

	// The preview is read-only: it counts the stored roles the word would newly hide and
	// creates no rule. An empty word counts zero.
	mux.HandleFunc("GET /api/stop-words/preview", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		word := strings.TrimSpace(r.URL.Query().Get("word"))
		count, err := d.Store.StopWordMatchCount(word)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"word": strings.ToLower(word), "count": count})
		return nil
	}))

	mux.HandleFunc("POST /api/stop-words", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			Word string `json:"word"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		if strings.TrimSpace(body.Word) == "" {
			return fmt.Errorf("word must not be empty")
		}
		sw, err := d.Store.AddStopWord(body.Word)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, sw)
		return nil
	}))

	mux.HandleFunc("DELETE /api/stop-words/{id}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err := d.Store.RemoveStopWord(id); err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"removed": id})
		return nil
	}))

	// ------------------------------------------------------------------
	// Phase 4.2 — Collection and sources
	// ------------------------------------------------------------------

	mux.HandleFunc("POST /api/search", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			Source string `json:"source"`
			Force  bool   `json:"force"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		if d.CollectFn != nil {
			runID, status, err := d.CollectFn(body.Source, body.Force)
			if err != nil {
				return err
			}
			writeJSON(w, http.StatusAccepted, map[string]any{"runId": runID, "status": status})
			return nil
		}
		// Legacy fallback: record a run row without collecting.
		runID, err := d.Store.StartCollectionRun(0, body.Source, "")
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"runId": runID, "status": "running"})
		return nil
	}))

	mux.HandleFunc("GET /api/collection", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		run, err := d.Store.LatestCollectionRun()
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, run)
		return nil
	}))

	mux.HandleFunc("GET /api/health", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		list, err := d.Store.HealthList()
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, list)
		return nil
	}))

	mux.HandleFunc("POST /api/import", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			URL string `json:"url"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		if body.URL == "" {
			return fmt.Errorf("url is required")
		}
		result, err := ex.Run(func() (any, error) {
			return ImportURL(d.Store, d.FetchClient, body.URL)
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": result})
		return nil
	}))

	// ------------------------------------------------------------------
	// Phase 4.3 — Assessment queue and options
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/queue", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		status, err := queueStatus(d.Store)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, status)
		return nil
	}))

	mux.HandleFunc("PUT /api/queue/assessment", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		if err := d.Store.SetQueueEnabled(body.Enabled); err != nil {
			return err
		}
		status, err := queueStatus(d.Store)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, status)
		return nil
	}))

	mux.HandleFunc("GET /api/assessment-options", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		cfg := d.Cfg.Get()
		pc := &cfg.Providers
		type providerEntry struct {
			Name  string `json:"name"`
			Model string `json:"model"`
			Type  string `json:"type"`
		}
		var providers []providerEntry
		for name, p := range pc.Providers {
			providers = append(providers, providerEntry{Name: name, Model: p.Model, Type: p.Type})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"selected":  pc.Stages.Assessment,
			"providers": providers,
		})
		return nil
	}))

	// ------------------------------------------------------------------
	// Phase 4.4 — Candidate context, profile, providers, configuration
	// ------------------------------------------------------------------

	registerProfileRoutes(mux, d)

	mux.HandleFunc("GET /api/context", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		ctx, err := d.Store.Context()
		if err != nil {
			return err
		}
		if ctx != nil {
			if current, cerr := profile.Current(d.Store); cerr == nil && current != nil {
				rev := 0
				switch n := ctx["profileRevision"].(type) {
				case float64:
					rev = int(n)
				}
				pid, _ := ctx["profileId"].(string)
				ctx["currentProfileRevision"] = current.Revision
				ctx["profileCurrent"] = pid == current.ID && rev == current.Revision
			}
		}
		writeJSON(w, http.StatusOK, ctx)
		return nil
	}))

	mux.HandleFunc("POST /api/context", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			RefreshRemote bool `json:"refreshRemote"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		cfg := d.Cfg.Get()
		current, err := profile.Current(d.Store)
		if err != nil {
			return err
		}
		var data *profile.ProfileData
		preferences := any(cfg.Search.Preferences)
		profileID := ""
		revision := 0
		var extras []candidate.EvidenceSource
		if current != nil {
			data = &current.Data
			preferences = profile.ContextPreferences(current)
			profileID = current.ID
			revision = current.Revision
			extras = append(extras, profile.ContextSource(current))
		}
		result, err := ex.Run(func() (any, error) {
			ctx, err := candidate.RefreshContextForProfile(
				d.Store, d.Dirs, cfg, preferences, profileID, revision, body.RefreshRemote, false, extras...)
			if err != nil {
				return nil, err
			}
			// Rescore all roles using the updated profile. Errors are non-fatal: the
			// context was saved successfully, so return the context even if rescoring fails.
			if rerr := d.Store.RescoreForProfile(profile.FastRankingQuery(cfg, data), revision); rerr != nil {
				_ = rerr // logged by Rescore internals; context response is not affected
			}
			return ctx, nil
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	}))

	mux.HandleFunc("GET /api/filter-options", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		writeJSON(w, http.StatusOK, d.Cfg.Get().Search)
		return nil
	}))

	mux.HandleFunc("GET /api/providers", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		cfg := d.Cfg.Get()
		providers := make(map[string]assess.Provider)
		for name, p := range cfg.Providers.Providers {
			providers[name] = assess.Provider{
				Type: p.Type, Model: p.Model, Endpoint: p.Endpoint,
				APIKeyEnv: p.APIKeyEnv, TimeoutMS: p.TimeoutMS, ContextChars: p.ContextChars,
			}
		}
		result := assess.Diagnostics(providers)
		writeJSON(w, http.StatusOK, result)
		return nil
	}))

	mux.HandleFunc("POST /api/providers/{name}/check", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		cfg := d.Cfg.Get()
		cp, ok := cfg.Providers.Providers[name]
		if !ok {
			return fmt.Errorf("Unknown provider %s", name)
		}
		p := assess.Provider{
			Type: cp.Type, Model: cp.Model, Endpoint: cp.Endpoint,
			APIKeyEnv: cp.APIKeyEnv, TimeoutMS: cp.TimeoutMS, ContextChars: cp.ContextChars,
		}
		result, err := ex.Run(func() (any, error) {
			return assess.SmokeCheck(p, d.InferenceFn)
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	}))

	mux.HandleFunc("GET /api/config/{name}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		content, err := config.Read(d.Dirs, name)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"content": content})
		return nil
	}))

	mux.HandleFunc("PUT /api/config/{name}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		var body struct {
			Content string `json:"content"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		result, err := ex.Run(func() (any, error) {
			if err := config.Save(d.Dirs, name, body.Content); err != nil {
				return nil, err
			}
			resp := map[string]any{"saved": name}
			// For runtime config, detect a port change before reloading so we can report
			// it without reverting the in-memory config to the old port value.
			if name == "runtime" {
				oldPort := d.Cfg.Get().Runtime.Port
				if err := d.Cfg.Reload(d.Dirs); err == nil {
					if d.Cfg.Get().Runtime.Port != oldPort {
						resp["restartRequired"] = "port change requires a restart to take effect"
					}
				}
			} else {
				// For non-runtime configs reload silently; the file is already valid.
				_ = d.Cfg.Reload(d.Dirs)
			}
			return resp, nil
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	}))

	// ------------------------------------------------------------------
	// Stage model selection
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/stage-providers", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		cfg := d.Cfg.Get()
		pc := &cfg.Providers
		type providerEntry struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		}
		var providers []providerEntry
		for name, p := range pc.Providers {
			providers = append(providers, providerEntry{Name: name, Model: p.Model})
		}
		if providers == nil {
			providers = []providerEntry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"stages": map[string]string{
				"assessment":  pc.Stages.Assessment,
				"preparation": pc.Stages.Preparation,
				"extraction":  pc.Stages.Extraction,
			},
			"providers": providers,
		})
		return nil
	}))

	mux.HandleFunc("PUT /api/stage-providers", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			Stage    string `json:"stage"`
			Provider string `json:"provider"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		validStages := map[string]bool{"assessment": true, "preparation": true, "extraction": true}
		if !validStages[body.Stage] {
			return fmt.Errorf("unknown stage %q; must be assessment, preparation or extraction", body.Stage)
		}
		result, err := ex.Run(func() (any, error) {
			cfg := d.Cfg.Get()
			if _, ok := cfg.Providers.Providers[body.Provider]; !ok {
				return nil, fmt.Errorf("unknown provider %q", body.Provider)
			}
			// Use yaml.Node surgery to change only the target scalar, preserving comments,
			// field order, and absent optional fields (endpoint, api_key_env, etc.).
			updated, err := updateStageInProvidersYAML(
				d.Dirs.ConfigFile("providers"), body.Stage, body.Provider,
			)
			if err != nil {
				return nil, err
			}
			if err := config.Save(d.Dirs, "providers", updated); err != nil {
				return nil, err
			}
			_ = d.Cfg.Reload(d.Dirs)
			cfg = d.Cfg.Get()
			return map[string]any{
				"stage":    body.Stage,
				"provider": body.Provider,
				"stages": map[string]string{
					"assessment":  cfg.Providers.Stages.Assessment,
					"preparation": cfg.Providers.Stages.Preparation,
					"extraction":  cfg.Providers.Stages.Extraction,
				},
			}, nil
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	}))

	// ------------------------------------------------------------------
	// Phase 4.5 — Feedback
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/jobs/{id}/feedback", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		resolved, err := d.Store.ResolveID(id)
		if err != nil {
			return err
		}
		recs, err := records.ReadFeedback(d.Dirs)
		if err != nil {
			return err
		}
		var out []records.FeedbackRecord
		for _, rec := range recs {
			recResolved, _ := d.Store.ResolveID(int64(rec.JobID))
			if recResolved == resolved {
				out = append(out, rec)
			}
		}
		if out == nil {
			out = []records.FeedbackRecord{}
		}
		writeJSON(w, http.StatusOK, out)
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/feedback", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var body struct {
			Text       string  `json:"text"`
			Scope      string  `json:"scope"`
			Supersedes *string `json:"supersedes"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		if body.Scope == "" {
			body.Scope = "general"
		}

		job, err := d.Store.Get(id)
		if err != nil {
			return err
		}

		rec, path, err := saveFeedbackRecord(d.Store, d.Dirs, job.ID, body.Text, body.Scope, job.OriginalURL, job.Title, body.Supersedes)
		if err != nil {
			return err
		}
		// Index the feedback in the database.
		if err := d.Store.IndexFeedback(rec.ID, job.ID, path, rec); err != nil {
			return err
		}
		// Enqueue for reassessment.
		fp := jobVersionHash(job)
		_, _ = d.Store.EnqueueJob(job.ID, fp, "", 100)

		qs, _ := queueStatus(d.Store)
		writeJSON(w, http.StatusAccepted, map[string]any{"record": rec, "queue": qs})
		return nil
	}))

	// ------------------------------------------------------------------
	// Phase 4.6 — Application preparation and PDF serving
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/jobs/{id}/application", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		state, err := preparationState(d.Store, id)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, state)
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/application/prepare", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var body map[string]any
		if err := readBody(r, &body); err != nil {
			return err
		}
		// Preparation reads the derived candidate context, so use the current confirmed profile
		// before queueing rather than a snapshot that may lag a save.
		if _, err := d.profileService().SyncDerivedContext(); err != nil {
			return fmt.Errorf("refresh profile context before preparation: %w", err)
		}
		row, err := d.Store.QueuePreparation(id, body)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusAccepted, row)
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/application/cancel", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		ops, err := d.Store.PreparationOperations(id)
		if err != nil {
			return err
		}
		for _, op := range ops {
			if op["status"] == "running" {
				_ = d.Store.FinishPreparation(int64(op["id"].(float64)), "cancelled", "", nil)
			}
		}
		state, _ := preparationState(d.Store, id)
		writeJSON(w, http.StatusOK, state)
		return nil
	}))

	mux.HandleFunc("PATCH /api/jobs/{id}/application", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		draft, err := d.Store.GetDraft(id)
		if err != nil {
			return err
		}
		var body map[string]any
		if err := readBody(r, &body); err != nil {
			return err
		}
		// Merge body into draft fields.
		bodyJSON, _ := json.Marshal(body)
		_ = json.Unmarshal(bodyJSON, draft)
		written, err := d.Store.WriteDraft(draft, "user update")
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, written)
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/application/questions", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var body struct {
			Label   string   `json:"label"`
			Type    string   `json:"type"`
			Options []string `json:"options"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		draft, err := d.Store.AddQuestion(id, body.Label, body.Type, body.Options)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, draft)
		return nil
	}))

	mux.HandleFunc("POST /api/jobs/{id}/application/remember", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		job, err := d.Store.Get(id)
		if err != nil {
			return err
		}
		var body records.AnswerRecord
		if err := readBody(r, &body); err != nil {
			return err
		}
		body.JobID = int(job.ID)
		if _, err := records.WriteAnswer(d.Dirs, &body); err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, body)
		return nil
	}))

	mux.HandleFunc("GET /api/jobs/{id}/application/memories", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		job, err := d.Store.Get(id)
		if err != nil {
			return err
		}
		all, err := records.ReadAnswers(d.Dirs)
		if err != nil {
			return err
		}
		var out []records.AnswerRecord
		for _, a := range all {
			if int64(a.JobID) == job.ID {
				out = append(out, a)
			}
		}
		if out == nil {
			out = []records.AnswerRecord{}
		}
		writeJSON(w, http.StatusOK, out)
		return nil
	}))

	// ------------------------------------------------------------------
	// NAVIGATION-2 — Search expressions, exclusion terms, source sites
	// ------------------------------------------------------------------

	// GET /api/search/expressions — read the current OR/AND search expressions.
	// Returns the effective expressions: either the explicit Expressions list from search.yaml
	// or the backward-compat synthesised OR group from titles+keywords.
	mux.HandleFunc("GET /api/search/expressions", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		cfg := d.Cfg.Get()
		exprs := cfg.Search.EffectiveExpressions()
		type exprOut struct {
			Or  []string `json:"or,omitempty"`
			And []string `json:"and,omitempty"`
		}
		out := make([]exprOut, len(exprs))
		for i, e := range exprs {
			out[i] = exprOut{Or: e.Or, And: e.And}
		}
		if out == nil {
			out = []exprOut{}
		}
		writeJSON(w, http.StatusOK, out)
		return nil
	}))

	// PUT /api/search/expressions — replace the search expressions in search.yaml.
	// Body: {"expressions": [{"or": [...]}, {"and": [...]}]}
	// An empty expressions array removes the expressions key (restoring titles/keywords control).
	mux.HandleFunc("PUT /api/search/expressions", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			Expressions json.RawMessage `json:"expressions"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		exprs, err := parseExpressionsBody(body.Expressions)
		if err != nil {
			return err
		}
		result, err := ex.Run(func() (any, error) {
			updated, err := updateSearchExpressionsInYAML(d.Dirs.ConfigFile("search"), exprs)
			if err != nil {
				return nil, err
			}
			if err := config.Save(d.Dirs, "search", updated); err != nil {
				return nil, err
			}
			_ = d.Cfg.Reload(d.Dirs)
			cfg := d.Cfg.Get()
			return map[string]any{"expressions": cfg.Search.EffectiveExpressions()}, nil
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	}))

	// GET /api/search/sites — read the sites list from search.yaml.
	mux.HandleFunc("GET /api/search/sites", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		cfg := d.Cfg.Get()
		sites := cfg.Search.Sites
		if sites == nil {
			sites = []config.Source{}
		}
		writeJSON(w, http.StatusOK, sites)
		return nil
	}))

	// POST /api/search/sites — add a new site to search.yaml.
	// Body: {"name": "...", "url": "...", "enabled": true}
	mux.HandleFunc("POST /api/search/sites", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			Name    string `json:"name"`
			URL     string `json:"url"`
			Enabled *bool  `json:"enabled"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		enabled := body.Enabled == nil || *body.Enabled
		result, err := ex.Run(func() (any, error) {
			updated, err := updateSearchSiteInYAML(d.Dirs.ConfigFile("search"), "add", map[string]any{
				"name": body.Name, "url": body.URL, "enabled": enabled,
			})
			if err != nil {
				return nil, err
			}
			if err := config.Save(d.Dirs, "search", updated); err != nil {
				return nil, err
			}
			_ = d.Cfg.Reload(d.Dirs)
			return map[string]any{"name": body.Name, "url": body.URL, "enabled": enabled}, nil
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusCreated, result)
		return nil
	}))

	// DELETE /api/search/sites/{name} — remove a site from search.yaml by name.
	mux.HandleFunc("DELETE /api/search/sites/{name}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		result, err := ex.Run(func() (any, error) {
			updated, err := updateSearchSiteInYAML(d.Dirs.ConfigFile("search"), "remove", map[string]any{"name": name})
			if err != nil {
				return nil, err
			}
			if err := config.Save(d.Dirs, "search", updated); err != nil {
				return nil, err
			}
			_ = d.Cfg.Reload(d.Dirs)
			return map[string]any{"removed": name}, nil
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	}))

	// PATCH /api/search/sites/{name} — toggle the enabled flag of a site in search.yaml.
	// Body: {"enabled": true|false}
	mux.HandleFunc("PATCH /api/search/sites/{name}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := readBody(r, &body); err != nil {
			return err
		}
		op := "enable"
		if !body.Enabled {
			op = "disable"
		}
		result, err := ex.Run(func() (any, error) {
			updated, err := updateSearchSiteInYAML(d.Dirs.ConfigFile("search"), op, map[string]any{"name": name})
			if err != nil {
				return nil, err
			}
			if err := config.Save(d.Dirs, "search", updated); err != nil {
				return nil, err
			}
			_ = d.Cfg.Reload(d.Dirs)
			return map[string]any{"name": name, "enabled": body.Enabled}, nil
		})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	}))

	mux.HandleFunc("GET /api/application-files/{name}", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if name != "cv.pdf" && name != "profile.pdf" {
			return fmt.Errorf("Unknown candidate file")
		}
		path, err := d.Dirs.Under("evidence", name)
		if err != nil {
			return fmt.Errorf("file not found: %s", name)
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, name))
		http.ServeFile(w, r, path)
		return nil
	}))
}

// ------------------------------------------------------------------
// Internal helpers used by route handlers
// ------------------------------------------------------------------

// queueStatus builds the QueueStatus shape the API returns.
func queueStatus(s *store.Store) (map[string]any, error) {
	state, err := s.GetQueueState()
	if err != nil {
		return nil, err
	}
	counts, err := s.QueueCounts()
	if err != nil {
		return nil, err
	}
	rows, err := s.AllQueueRows()
	if err != nil {
		return nil, err
	}
	run, err := s.GetQueueRuntime()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"enabled": state.Enabled,
		"counts":  counts,
		"rows":    rows,
		"runtime": run,
	}, nil
}

// configExprsToFTSGroups converts config search expressions to store query groups.
// It is the server-package equivalent of background.expressionsToFTSGroups.
func configExprsToFTSGroups(exprs []config.SearchExpression) []store.SearchGroup {
	groups := make([]store.SearchGroup, len(exprs))
	for i, e := range exprs {
		groups[i] = store.SearchGroup{Or: e.Or, And: e.And}
	}
	return groups
}

// jobVersionHash computes a fingerprint of the job's assessable fields, used to detect staleness.
func jobVersionHash(job *model.Job) string {
	data, _ := json.Marshal(map[string]any{
		"title": job.Title, "company": job.Company,
		"description": job.Description, "location": job.Location,
	})
	var h hash.Hash = sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// saveFeedbackRecord creates and saves an owner-confirmed feedback record. Returns the record and path.
func saveFeedbackRecord(s *store.Store, d dirs.Dirs, jobID int64, text, scope, jobURL, jobTitle string, supersedes *string) (*records.FeedbackRecord, string, error) {
	if text == "" {
		return nil, "", fmt.Errorf("feedback text is required")
	}
	id := uuid.New().String()
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	rec := &records.FeedbackRecord{
		ID:         id,
		Created:    now,
		JobID:      int(jobID),
		JobURL:     jobURL,
		JobTitle:   jobTitle,
		Source:     "owner",
		Scope:      scope,
		Text:       text,
		Supersedes: supersedes,
	}
	path, err := records.WriteFeedback(d, rec)
	if err != nil {
		return nil, "", err
	}
	return rec, path, nil
}

// preparationState returns the full application state shape (draft + operations).
func preparationState(s *store.Store, jobID int64) (map[string]any, error) {
	draft, err := s.GetDraft(jobID)
	if err != nil {
		return nil, err
	}
	ops, err := s.PreparationOperations(jobID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"draft": draft, "operations": ops}, nil
}
