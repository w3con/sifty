package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"

	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

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
	// Documents
	// ------------------------------------------------------------------

	mux.HandleFunc("GET /api/profile/capabilities", safeHandler(func(w http.ResponseWriter, r *http.Request) error {
		cfg := d.Cfg.Get()
		writeJSON(w, http.StatusOK, map[string]any{
			"documentTypes":      cfg.Runtime.EffectiveProfileDocumentTypes(),
			"maxDocumentBytes":   cfg.Runtime.EffectiveProfileMaxDocumentBytes(),
			"maxUploadBodyBytes": cfg.Runtime.EffectiveProfileUploadBodyBytes(),
			"skillLevels":        profile.SkillLevels(),
			"factFieldKeys":      profile.AllowedFactFieldKeys(),
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
