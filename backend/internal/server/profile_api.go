package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"
	"strings"
	"sync"

	"github.com/w3con/sifty/internal/matchsvc"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/vacancyai"
)

// bgRescore serialises the background full-catalogue rescore that is queued after a quick
// single-job rescore. At most one full rescore runs at a time; a second request while one is
// running sets the pending flag so a second run follows immediately after the first completes.
var bgRescore struct {
	mu      sync.Mutex
	running bool
	pending bool
}

// scheduleBackgroundRescore queues a full catalogue rescore (candidate context + all jobs) to run
// in background. Calls are coalesced: if a rescore is already running, one more is guaranteed to
// follow it; if one is already pending, the extra call is a no-op.
func scheduleBackgroundRescore(d *Deps) {
	bgRescore.mu.Lock()
	if bgRescore.running {
		bgRescore.pending = true
		bgRescore.mu.Unlock()
		return
	}
	bgRescore.running = true
	bgRescore.mu.Unlock()

	go func() {
		for {
			if _, err := d.profileService().SyncDerivedContext(); err != nil {
				log.Printf("background rescore: %v", err)
			}

			bgRescore.mu.Lock()
			if !bgRescore.pending {
				bgRescore.running = false
				bgRescore.mu.Unlock()
				return
			}
			bgRescore.pending = false
			bgRescore.mu.Unlock()
		}
	}()
}

// rescoreJobAndScheduleBackground rescores a single job synchronously (fast) and queues a full
// background rescore. Errors from the single-job rescore are logged but do not block the response.
func rescoreJobAndScheduleBackground(d *Deps, jobID int64) {
	if err := matchsvc.New(d.Store, d.Cfg).RecalculateJob(context.Background(), jobID); err != nil && !errors.Is(err, matchsvc.ErrNoProfile) {
		log.Printf("quick rescore job %d: %v", jobID, err)
	}
	scheduleBackgroundRescore(d)
}

// registerProfileRoutes wires the typed local profile read/save contract plus the explicit
// document/fact actions. No normal GET or PUT calls a model provider; only POST analyze does.
func registerProfileRoutes(mux Router, d *Deps) {
	mux.HandleFunc("GET /api/profile", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		service := d.profileService()
		current, err := service.Get()
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, current)
		return nil
	}))

	// PUT is written without safeHandler because a stale expectedRevision must return 409 with
	// the current revision, not the uniform 400 error.
	mux.HandleFunc("PUT /api/profile", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ExpectedRevision *int                 `json:"expectedRevision"`
			Data             *profile.ProfileData `json:"data"`
		}
		if err := readBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.ExpectedRevision == nil {
			writeError(w, http.StatusBadRequest, "expectedRevision is required")
			return
		}
		if body.Data == nil {
			writeError(w, http.StatusBadRequest, "data is required")
			return
		}

		service := d.profileService()
		saved, err := service.Update(*body.ExpectedRevision, *body.Data)
		if err != nil {
			var conflict *profile.ConflictError
			if errors.As(err, &conflict) {
				writeJSON(w, http.StatusConflict, map[string]any{
					"error":            "profile revision conflict",
					"expectedRevision": conflict.Expected,
					"currentRevision":  conflict.Current.Revision,
					"current":          conflict.Current,
				})
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		scheduleBackgroundRescore(d)
		writeJSON(w, http.StatusOK, saved)
	})

	// Section saves send only the editor's own section. The server merges it into the current
	// stored profile inside one transaction, so two editors never replace each other's data and
	// the client never needs to submit or understand a profile revision.
	mux.HandleFunc("PUT /api/profile/sections/{section}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Value json.RawMessage `json:"value"`
		}
		if err := readBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if len(body.Value) == 0 {
			writeError(w, http.StatusBadRequest, "value is required")
			return
		}
		saved, changed, err := d.profileService().UpdateSection(r.PathValue("section"), body.Value)
		if err != nil {
			writeProfileError(w, err)
			return
		}
		if changed {
			scheduleBackgroundRescore(d)
		}
		writeJSON(w, http.StatusOK, map[string]any{"profile": saved, "changed": changed})
	})

	// Adds one skill (or a confirmed fact) from where a requirement is read on a vacancy. The
	// endpoint classifies the name: a known vocabulary technology or a short non-verb phrase (≤ 3
	// words without -ing forms) is added as a skill; anything else is recorded as a confirmed "have"
	// fact. A comma-separated list of qualifying names is split into separate skills.
	// When the requirement has alternatives (e.g. "Go or Rust") the endpoint returns
	// action="offer_alternatives" and the UI presents each alternative as its own button.
	// When jobId > 0, the single job is rescored synchronously before the response; a full
	// catalogue rescore runs in background.
	mux.HandleFunc("POST /api/profile/skills", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name         string   `json:"name"`
			Level        string   `json:"level"`
			JobID        int64    `json:"jobId"`
			Alternatives []string `json:"alternatives"`
		}
		if err := readBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		vocab := vacancyai.NewVocabulary(d.Cfg.Get().Skills)
		cls := vocab.ClassifySkillAdd(body.Name, body.Alternatives)

		if cls.HasAlternatives {
			// Prefer alternatives detected from the name (e.g. "Go or Rust" → ["Go","Rust"]).
			// Fall back to caller-supplied alternatives (from the job extraction result).
			alts := cls.Alternatives
			if len(alts) == 0 {
				alts = body.Alternatives
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"action":       "offer_alternatives",
				"alternatives": alts,
			})
			return
		}

		svc := d.profileService()
		if cls.ConfirmedFact != "" {
			// Phrase or activity: record as confirmed "have" fact.
			saved, changed, err := svc.AddConfirmedFact(cls.ConfirmedFact, true)
			if err != nil {
				writeProfileError(w, err)
				return
			}
			if body.JobID > 0 {
				rescoreJobAndScheduleBackground(d, body.JobID)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"profile":       saved,
				"changed":       changed,
				"action":        "fact_confirmed",
				"factConfirmed": cls.ConfirmedFact,
			})
			return
		}

		// One or more skill names: add each.
		var lastProfile *profile.Profile
		anyChanged := false
		for _, skillName := range cls.SkillsToAdd {
			saved, changed, err := svc.AddSkill(skillName, body.Level)
			if err != nil {
				writeProfileError(w, err)
				return
			}
			lastProfile = saved
			if changed {
				anyChanged = true
			}
		}
		if lastProfile == nil {
			cur, _ := svc.Get()
			lastProfile = cur
		}
		if body.JobID > 0 {
			rescoreJobAndScheduleBackground(d, body.JobID)
		} else {
			matchsvc.PrioritizeModelWork(body.JobID)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"profile":     lastProfile,
			"changed":     anyChanged,
			"action":      "skills_added",
			"skillsAdded": cls.SkillsToAdd,
		})
	})

	// Records that the owner does not have a skill or does not meet a non-vocabulary requirement.
	// Classification mirrors the ✓ path (ClassifySkillAdd):
	//   - "X or Y"/"X/Y" where every part is a valid alternative name → AddAbsentSkill for each part.
	//   - Comma-separated list of skill names → AddAbsentSkill for each part.
	//   - Single skill name → AddAbsentSkill.
	//   - Phrase or activity → AddConfirmedFact(false).
	// This ensures that pressing ✕ on "C++ or C" records "C++" and "C" as separate absent skills
	// rather than the full phrase, so codeMatch can resolve the row on the next rescore.
	// When jobId > 0, the single job is rescored synchronously before the response.
	mux.HandleFunc("POST /api/profile/absent-skills", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name  string `json:"name"`
			JobID int64  `json:"jobId"`
		}
		if err := readBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		vocab := vacancyai.NewVocabulary(d.Cfg.Get().Skills)
		svc := d.profileService()
		cls := vocab.ClassifySkillAdd(body.Name, nil)

		var (
			lastProfile *profile.Profile
			anyChanged  bool
			action      string
		)
		switch {
		case cls.ConfirmedFact != "":
			// Phrase or activity: confirmed "do not have" fact.
			p, ch, err := svc.AddConfirmedFact(cls.ConfirmedFact, false)
			if err != nil {
				writeProfileError(w, err)
				return
			}
			lastProfile, anyChanged = p, ch
			action = "fact_confirmed_absent"
		default:
			// Skill name(s) or detected alternatives: add each to the absent-skills list.
			// HasAlternatives → cls.Alternatives; SkillsToAdd → comma-parts or single name.
			names := cls.SkillsToAdd
			if cls.HasAlternatives {
				names = cls.Alternatives
			}
			storedCount := 0
			for _, name := range names {
				p, ch, err := svc.AddAbsentSkill(name)
				if err != nil {
					writeProfileError(w, err)
					return
				}
				lastProfile = p
				if ch {
					anyChanged = true
					storedCount++
				}
			}
			// When alternatives were detected and fewer were stored than expected, some
			// alternatives share a normalised store key (e.g. "C++" and "C" both map to "c").
			// Also store the full phrase so codeMatch can resolve the row via its vocabulary groups
			// (the full phrase "C++ or C" indexes both C++ and C absent-groups).
			if cls.HasAlternatives && storedCount < len(names) {
				p, ch, err := svc.AddAbsentSkill(body.Name)
				if err == nil {
					lastProfile = p
					if ch {
						anyChanged = true
					}
				}
			}
			action = "absent_skill_recorded"
		}
		if body.JobID > 0 {
			rescoreJobAndScheduleBackground(d, body.JobID)
		} else {
			matchsvc.PrioritizeModelWork(body.JobID)
		}
		writeJSON(w, http.StatusOK, map[string]any{"profile": lastProfile, "changed": anyChanged, "action": action})
	})

	// Records that the owner meets or does not meet a non-skill requirement (experience,
	// qualification, authorization, language, other). The same requirement (by normalized text)
	// in every other vacancy is then decided by code without asking the model.
	// When jobId > 0, the single job is rescored synchronously before the response.
	mux.HandleFunc("POST /api/profile/confirmed-facts", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text  string `json:"text"`
			Have  bool   `json:"have"`
			JobID int64  `json:"jobId"`
		}
		if err := readBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		saved, changed, err := d.profileService().AddConfirmedFact(body.Text, body.Have)
		if err != nil {
			writeProfileError(w, err)
			return
		}
		if body.JobID > 0 {
			rescoreJobAndScheduleBackground(d, body.JobID)
		} else {
			matchsvc.PrioritizeModelWork(body.JobID)
		}
		writeJSON(w, http.StatusOK, map[string]any{"profile": saved, "changed": changed, "action": "fact_confirmed"})
	})

	// Removes one confirmed fact by its normalized key. The response is returned immediately after
	// the fast profile save; a full catalogue rescore runs in background (coalesced).
	mux.HandleFunc("DELETE /api/profile/confirmed-facts/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		saved, changed, err := d.profileService().RemoveConfirmedFact(key)
		if err != nil {
			writeProfileError(w, err)
			return
		}
		if changed {
			scheduleBackgroundRescore(d)
		}
		writeJSON(w, http.StatusOK, map[string]any{"profile": saved, "changed": changed})
	})

	// Explicit local recomputation of the derived candidate context and fast ranking from the
	// current stored profile. It reuses the same local-only path as a save, never fetches remote
	// sources, never starts extraction and never calls a model provider.
	mux.HandleFunc("POST /api/profile/derived/refresh", func(w http.ResponseWriter, r *http.Request) {
		updated, err := d.profileService().SyncDerivedContext()
		if updated == nil {
			if err == nil {
				writeError(w, http.StatusNotFound, "profile not found")
				return
			}
			writeProfileError(w, err)
			return
		}
		// A projection failure still leaves the saved profile intact and returns the honest
		// derived status (contextCurrent/contextError) for the client to render.
		writeJSON(w, http.StatusOK, map[string]any{"profile": updated, "refreshed": err == nil})
	})

	// ------------------------------------------------------------------
	// Profile photo
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/profile/avatar/meta", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		avatar, err := d.profileService().GetAvatar()
		if err != nil {
			return err
		}
		if avatar == nil {
			writeJSON(w, http.StatusOK, map[string]any{"exists": false})
			return nil
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"exists":      true,
			"contentType": avatar.ContentType,
			"byteSize":    avatar.ByteSize,
			"updatedAt":   avatar.UpdatedAt,
		})
		return nil
	}))

	mux.HandleFunc("GET /api/profile/avatar", func(w http.ResponseWriter, r *http.Request) {
		path, contentType, err := d.profileService().AvatarFile()
		if err != nil {
			if errors.Is(err, profile.ErrAvatarNotFound) {
				writeError(w, http.StatusNotFound, "profile photo is not set")
				return
			}
			writeProfileError(w, err)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Disposition", "inline")
		http.ServeFile(w, r, path)
	})

	mux.HandleFunc("POST /api/profile/avatar", func(w http.ResponseWriter, r *http.Request) {
		if d.Cfg != nil {
			r.Body = http.MaxBytesReader(w, r.Body,
				int64(d.Cfg.Get().Runtime.EffectiveProfileAvatarUploadBodyBytes()))
		}
		var body struct {
			Filename      string `json:"filename"`
			ContentBase64 string `json:"contentBase64"`
		}
		if err := readBody(r, &body); err != nil {
			if strings.Contains(err.Error(), "request body too large") {
				writeError(w, http.StatusRequestEntityTooLarge, "photo exceeds the configured upload allowance")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.ContentBase64 == "" {
			writeError(w, http.StatusBadRequest, "contentBase64 is required")
			return
		}
		content, err := base64.StdEncoding.DecodeString(body.ContentBase64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "contentBase64 is not valid base64")
			return
		}
		avatar, err := d.profileService().SaveAvatar(body.Filename, content)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"exists":      true,
			"contentType": avatar.ContentType,
			"byteSize":    avatar.ByteSize,
			"updatedAt":   avatar.UpdatedAt,
		})
	})

	mux.HandleFunc("DELETE /api/profile/avatar", func(w http.ResponseWriter, r *http.Request) {
		deleted, err := d.profileService().DeleteAvatar()
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"exists": false, "deleted": deleted})
	})

	// ------------------------------------------------------------------
	// Documents
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/profile/capabilities", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		cfg := d.Cfg.Get()
		writeJSON(w, http.StatusOK, map[string]any{
			"documentTypes":      cfg.Runtime.EffectiveProfileDocumentTypes(),
			"maxDocumentBytes":   cfg.Runtime.EffectiveProfileMaxDocumentBytes(),
			"maxUploadBodyBytes": cfg.Runtime.EffectiveProfileUploadBodyBytes(),
			"avatarTypes":        cfg.Runtime.EffectiveProfileAvatarTypes(),
			"maxAvatarBytes":     cfg.Runtime.EffectiveProfileAvatarMaxBytes(),
			"skillLevels":        profile.SkillLevels(),
			"skillLevelOptions":  profile.SkillLevelOptions(),
			"factFieldKeys":      profile.AllowedFactFieldKeys(),
			"fieldDescriptors":   profile.FieldDescriptors(),
			"optionSets":         profile.OptionSets(),
			"documentStates":     profile.DocumentStates(),
			"factStates":         profile.FactStates(),
		})
		return nil
	}))

	mux.HandleFunc("GET /api/profile/documents", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		docs, err := d.profileService().ListDocuments()
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, docs)
		return nil
	}))

	mux.HandleFunc("POST /api/profile/documents", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Filename      string `json:"filename"`
			ContentBase64 string `json:"contentBase64"`
			Text          string `json:"text"`
		}
		if d.Cfg != nil {
			r.Body = http.MaxBytesReader(w, r.Body,
				int64(d.Cfg.Get().Runtime.EffectiveProfileUploadBodyBytes()))
		}
		if err := readBody(r, &body); err != nil {
			if strings.Contains(err.Error(), "request body too large") {
				writeError(w, http.StatusRequestEntityTooLarge, "upload exceeds the configured document body allowance")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		var content []byte
		if body.ContentBase64 != "" {
			decoded, err := base64.StdEncoding.DecodeString(body.ContentBase64)
			if err != nil {
				writeError(w, http.StatusBadRequest, "contentBase64 is not valid base64")
				return
			}
			content = decoded
		} else if body.Text != "" {
			content = []byte(body.Text)
		}
		if len(content) == 0 {
			writeError(w, http.StatusBadRequest, "contentBase64 or text is required")
			return
		}
		doc, err := d.profileService().CreateDocument(body.Filename, content)
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, doc)
	})

	mux.HandleFunc("GET /api/profile/documents/{id}", func(w http.ResponseWriter, r *http.Request) {
		doc, err := d.profileService().GetDocument(r.PathValue("id"))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, doc)
	})

	mux.HandleFunc("GET /api/profile/documents/{id}/text", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		text, err := d.profileService().DocumentText(id)
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"documentId": id, "text": text})
	})

	mux.HandleFunc("GET /api/profile/documents/{id}/file", func(w http.ResponseWriter, r *http.Request) {
		path, name, err := d.profileService().DocumentFilePath(r.PathValue("id"))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		disposition := "inline"
		if strings.EqualFold(fileExt(name), "docx") {
			disposition = "attachment"
		}
		if cd := mime.FormatMediaType(disposition, map[string]string{"filename": name}); cd != "" {
			w.Header().Set("Content-Disposition", cd)
		}
		http.ServeFile(w, r, path)
	})

	// Removes the file, its text and its unaccepted proposals; accepted facts stay and show that
	// their source was deleted. The answer says what was removed.
	mux.HandleFunc("DELETE /api/profile/documents/{id}", func(w http.ResponseWriter, r *http.Request) {
		deletion, err := d.profileService().DeleteDocument(r.PathValue("id"))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, deletion)
	})

	mux.HandleFunc("POST /api/profile/documents/{id}/analyze", func(w http.ResponseWriter, r *http.Request) {
		doc, err := d.profileService().AnalyzeDocument(r.PathValue("id"))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, doc)
	})

	mux.HandleFunc("POST /api/profile/documents/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		doc, err := d.profileService().CancelDocumentAnalysis(r.PathValue("id"))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, doc)
	})

	mux.HandleFunc("POST /api/profile/documents/{id}/retry", func(w http.ResponseWriter, r *http.Request) {
		doc, err := d.profileService().RetryDocument(r.PathValue("id"))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, doc)
	})

	// ------------------------------------------------------------------
	// Facts
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/profile/facts", func(w http.ResponseWriter, r *http.Request) {
		facts, err := d.profileService().ListFacts(strings.TrimSpace(r.URL.Query().Get("documentId")))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, facts)
	})

	mux.HandleFunc("GET /api/profile/documents/{id}/facts", func(w http.ResponseWriter, r *http.Request) {
		facts, err := d.profileService().ListFacts(r.PathValue("id"))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, facts)
	})

	mux.HandleFunc("POST /api/profile/facts/{id}/accept", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ExpectedRevision *int `json:"expectedRevision"`
		}
		if err := readBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.ExpectedRevision == nil {
			writeError(w, http.StatusBadRequest, "expectedRevision is required")
			return
		}
		decision, err := d.profileService().AcceptFact(r.PathValue("id"), *body.ExpectedRevision)
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, decision)
	})

	mux.HandleFunc("POST /api/profile/facts/{id}/reject", func(w http.ResponseWriter, r *http.Request) {
		decision, err := d.profileService().RejectFact(r.PathValue("id"))
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, decision)
	})

	mux.HandleFunc("POST /api/profile/facts/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ExpectedRevision *int   `json:"expectedRevision"`
			Choice           string `json:"choice"`
		}
		if err := readBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.ExpectedRevision == nil {
			writeError(w, http.StatusBadRequest, "expectedRevision is required")
			return
		}
		decision, err := d.profileService().ResolveFact(r.PathValue("id"), *body.ExpectedRevision, body.Choice)
		if err != nil {
			writeProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, decision)
	})
}

// writeProfileError maps domain/persistence errors to explicit HTTP outcomes.
func writeProfileError(w http.ResponseWriter, err error) {
	var conflict *profile.FactConflictError
	if errors.As(err, &conflict) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":           "profile fact conflict",
			"conflict":        conflict.Conflict,
			"currentRevision": conflict.Conflict.CurrentRevision,
		})
		return
	}
	var revision *profile.ConflictError
	if errors.As(err, &revision) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":            "profile revision conflict",
			"expectedRevision": revision.Expected,
			"currentRevision":  revision.Current.Revision,
			"current":          revision.Current,
		})
		return
	}
	var unavailable *profile.AnalyzerUnavailableError
	if errors.As(err, &unavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": unavailable.Reason, "code": "analyzer_unavailable",
		})
		return
	}
	var docErr *profile.DocumentError
	if errors.As(err, &docErr) {
		status := http.StatusBadRequest
		if docErr.State == profile.DocumentAnalyzing {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]any{
			"error": docErr.Message, "code": "document_state", "documentState": docErr.State,
		})
		return
	}
	switch {
	case errors.Is(err, store.ErrProfileDocumentNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrProfileFactNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrProfileFactResolved):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

// profileService returns the configured service, or builds the default one from the shared
// store and configuration holder. Existing tests construct Deps without a profile service.
func (d *Deps) profileService() *profile.Service {
	if d.Profiles != nil {
		return d.Profiles
	}
	return profile.NewService(d.Store, d.Cfg, profile.WithDirs(d.Dirs))
}

func fileExt(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return ""
}
