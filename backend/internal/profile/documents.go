package profile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/w3con/sifty/internal/candidate"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/extract"
	"github.com/w3con/sifty/internal/store"
)

// Profile document processing states. A state names a visible outcome, not a hidden job.
const (
	DocumentUploaded   = "uploaded"
	DocumentExtracting = "extracting"
	DocumentReady      = "ready"
	DocumentNoText     = "no_text"
	DocumentAnalyzing  = "analyzing"
	DocumentAnalyzed   = "analyzed"
	DocumentStateError = "error"
	DocumentCancelled  = "cancelled"
	DocumentMissing    = "missing"
	DocumentImported   = "imported"
	DocumentStale      = "stale"
)

// DocumentStates returns the visible processing states in UI order.
func DocumentStates() []string {
	return []string{DocumentUploaded, DocumentExtracting, DocumentReady, DocumentNoText,
		DocumentAnalyzing, DocumentAnalyzed, DocumentStateError, DocumentCancelled,
		DocumentMissing, DocumentImported, DocumentStale}
}

// Document is the API-facing description of one input document. Extracted text is served by the
// separate text endpoint so listing several CVs does not return all of their content.
type Document struct {
	ID               string         `json:"documentId"`
	ProfileID        string         `json:"profileId"`
	Kind             string         `json:"kind"`
	OriginalName     string         `json:"originalName"`
	SourceURL        string         `json:"sourceUrl,omitempty"`
	ContentHash      string         `json:"contentHash,omitempty"`
	ProcessingState  string         `json:"processingState"`
	HasText          bool           `json:"hasText"`
	TextChars        int            `json:"textChars"`
	HasOriginal      bool           `json:"hasOriginal"`
	AnalysisMetadata map[string]any `json:"analysisMetadata,omitempty"`
	Error            string         `json:"error,omitempty"`
	CreatedAt        string         `json:"createdAt"`
	UpdatedAt        string         `json:"updatedAt"`
	FactCounts       map[string]int `json:"factCounts"`
}

// DocumentError reports a document that cannot perform the requested action in its current state.
type DocumentError struct {
	State   string
	Message string
}

func (e *DocumentError) Error() string { return e.Message }

// AnalyzerUnavailableError reports that the configured extraction stage cannot be used. The
// message is suitable for the UI; no fallback provider is attempted.
type AnalyzerUnavailableError struct {
	Reason string
}

func (e *AnalyzerUnavailableError) Error() string { return e.Reason }

// Option configures the profile service's document support.
type Option func(*Service)

// WithDirs supplies the authoritative data-directory resolver for document storage.
func WithDirs(d dirs.Dirs) Option { return func(s *Service) { s.dirs = d } }

// WithDocxReader supplies the DOCX text reader. Production uses NewNodeDocxReader; tests may stub it.
func WithDocxReader(r extract.DocxReader) Option { return func(s *Service) { s.docx = r } }

// activeRun is one in-flight document analysis. Cancelling the context asks the configured
// provider adapter to stop; the run id guards against publishing a late result.
type activeRun struct {
	runID  string
	cancel context.CancelFunc
}

func (s *Service) addRun(documentID, runID string, cancel context.CancelFunc) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if s.runs == nil {
		s.runs = map[string]activeRun{}
	}
	s.runs[documentID] = activeRun{runID: runID, cancel: cancel}
}

func (s *Service) takeRun(documentID, runID string) (activeRun, bool) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	run, ok := s.runs[documentID]
	if !ok || run.runID != runID {
		return activeRun{}, false
	}
	delete(s.runs, documentID)
	return run, true
}

// ListDocuments returns the profile's documents, importing configured legacy candidate inputs
// as metadata once. It never re-reads local file text and never fetches remote profiles.
func (s *Service) ListDocuments() ([]*Document, error) {
	current, err := s.Get()
	if err != nil {
		return nil, err
	}
	docs, err := s.store.ListProfileDocuments(current.ID)
	if err != nil {
		return nil, err
	}
	if err := s.importLegacyDocuments(current.ID, docs); err != nil {
		return nil, err
	}
	docs, err = s.store.ListProfileDocuments(current.ID)
	if err != nil {
		return nil, err
	}
	facts, err := s.store.ListProfileFacts(current.ID, "")
	if err != nil {
		return nil, err
	}
	counts := factCountsByDocument(facts)
	out := make([]*Document, 0, len(docs))
	for _, rec := range docs {
		if rec.ProcessingState == store.ProfileDocumentDeleted {
			continue
		}
		out = append(out, documentFromRecord(rec, counts[rec.ID]))
	}
	return out, nil
}

// GetDocument returns one document description.
func (s *Service) GetDocument(id string) (*Document, error) {
	rec, err := s.store.GetProfileDocument(id)
	if err != nil {
		return nil, err
	}
	facts, err := s.store.ListProfileFacts(rec.ProfileID, id)
	if err != nil {
		return nil, err
	}
	return documentFromRecord(rec, factCounts(facts)), nil
}

// DocumentText returns the stored extracted text or a clear no-text error.
func (s *Service) DocumentText(id string) (string, error) {
	rec, err := s.store.GetProfileDocument(id)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(rec.ExtractedText) == "" {
		msg := "No extracted text for this document"
		if rec.Error != "" {
			msg = rec.Error
		}
		return "", &DocumentError{State: rec.ProcessingState, Message: msg}
	}
	return rec.ExtractedText, nil
}

// DocumentFilePath resolves the original file path under the data directory.
func (s *Service) DocumentFilePath(id string) (string, string, error) {
	rec, err := s.store.GetProfileDocument(id)
	if err != nil {
		return "", "", err
	}
	if rec.StoragePath == "" {
		return "", "", &DocumentError{State: rec.ProcessingState, Message: "This document has no local original file"}
	}
	path, err := s.dirs.Under(rec.StoragePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve document path: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		return "", "", &DocumentError{State: rec.ProcessingState, Message: "Original file is missing under the data directory"}
	}
	return path, rec.OriginalName, nil
}

// CreateDocument stores an uploaded original under data/evidence/profile-documents/<id>/ and
// extracts its text synchronously (no model call). The original client filename is metadata only;
// the managed path uses a fixed original.<ext> name.
func (s *Service) CreateDocument(filename string, content []byte) (*Document, error) {
	cfg := s.cfg.Get()
	if len(content) == 0 {
		return nil, fmt.Errorf("document content is empty")
	}
	max := cfg.Runtime.EffectiveProfileMaxDocumentBytes()
	if len(content) > max {
		return nil, fmt.Errorf("document is %d bytes; configured limit is %d", len(content), max)
	}
	name := filepath.Base(strings.TrimSpace(filename))
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		return nil, fmt.Errorf("filename is required")
	}
	kind := extract.Kind(name)
	if !allowedDocumentType(cfg.Runtime.EffectiveProfileDocumentTypes(), kind) {
		return nil, fmt.Errorf("document type %q is not accepted; configured types: %s", kind,
			strings.Join(cfg.Runtime.EffectiveProfileDocumentTypes(), ", "))
	}

	current, err := s.Get()
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	rel := filepath.Join(managedDocumentDir(id), "original."+kind)
	abs, err := s.dirs.Under(rel)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return nil, fmt.Errorf("create document directory: %w", err)
	}
	tmp := abs + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return nil, fmt.Errorf("write document: %w", err)
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("publish document: %w", err)
	}
	sum := sha256.Sum256(content)
	rec := &store.ProfileDocumentRecord{
		ID:              id,
		ProfileID:       current.ID,
		Kind:            kind,
		OriginalName:    name,
		StoragePath:     rel,
		ContentHash:     hex.EncodeToString(sum[:]),
		ProcessingState: DocumentExtracting,
	}
	if err := s.store.CreateProfileDocument(rec); err != nil {
		_ = os.Remove(abs)
		return nil, err
	}
	// Text extraction is local and model-free. A no-text/error state is returned with the
	// document so the client still receives its id and can inspect or retry it.
	_ = s.extractDocumentText(rec, "upload")
	return s.GetDocument(id)
}

// RetryDocument re-runs local text extraction and then analysis. It never applies facts to the
// profile; any proposals still require explicit confirmation.
func (s *Service) RetryDocument(id string) (*Document, error) {
	rec, err := s.store.GetProfileDocument(id)
	if err != nil {
		return nil, err
	}
	if rec.ProcessingState == DocumentAnalyzing {
		return nil, &DocumentError{State: rec.ProcessingState, Message: "Analysis is already running"}
	}
	if rec.StoragePath != "" || strings.TrimSpace(rec.ExtractedText) == "" {
		if err := s.extractDocumentText(rec, "retry"); err != nil {
			return nil, err
		}
	}
	rec, err = s.store.GetProfileDocument(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rec.ExtractedText) == "" {
		return nil, &DocumentError{State: rec.ProcessingState, Message: "No extractable text; provide a text/OCR export"}
	}
	return s.AnalyzeDocument(id)
}

// extractDocumentText reads the original and stores its text/state. It is idempotent and safe to
// call from upload, retry and analysis start.
func (s *Service) extractDocumentText(rec *store.ProfileDocumentRecord, reason string) error {
	if rec.StoragePath == "" {
		msg := "This document has no local original to read"
		if rec.SourceURL != "" {
			msg = "Remote profile has no export; configure an owner-provided export. Authenticated scraping is not performed."
		}
		return s.store.UpdateProfileDocumentState(rec.ID, DocumentMissing, "", msg)
	}
	path, err := s.dirs.Under(rec.StoragePath)
	if err != nil {
		return s.store.UpdateProfileDocumentState(rec.ID, DocumentStateError, "", "Document path is outside the data directory")
	}
	if _, err := os.Stat(path); err != nil {
		return s.store.UpdateProfileDocumentState(rec.ID, DocumentMissing, "", "Original file is missing under the data directory")
	}
	cfg := s.cfg.Get()
	timeout := time.Duration(cfg.Runtime.EffectiveProfileTextTimeoutMS()) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	text, err := extract.Text(ctx, path, s.docx)
	if err == nil && strings.TrimSpace(text) == "" {
		err = fmt.Errorf("no extractable text; provide an OCR/text export")
	}
	if err != nil {
		state := DocumentStateError
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("text extraction timed out after %s", timeout)
		}
		if strings.Contains(err.Error(), "no extractable text") {
			state = DocumentNoText
		}
		return s.store.UpdateProfileDocumentState(rec.ID, state, "", err.Error())
	}
	// Re-read the file only to hash the exact original bytes; the text comes from the shared
	// reader. This is one read per explicit extraction, not on every profile GET.
	raw, readErr := os.ReadFile(path)
	hash := rec.ContentHash
	if readErr == nil {
		sum := sha256.Sum256(raw)
		hash = hex.EncodeToString(sum[:])
	}
	meta := map[string]any{"extraction": reason, "extractedAt": nowISO()}
	metaJSON, _ := json.Marshal(meta)
	if err := s.store.UpdateProfileDocumentText(rec.ID, strings.TrimRight(text, "\x00"), hash, DocumentReady, string(metaJSON), ""); err != nil {
		return err
	}
	return nil
}

// AnalyzeDocument starts one asynchronous analysis run. A normal GET/PUT never starts a model
// call; only this explicit action does. The run is guarded by an id so a cancelled run publishes
// no facts.
func (s *Service) AnalyzeDocument(id string) (*Document, error) {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	rec, err := s.store.GetProfileDocument(id)
	if err != nil {
		return nil, err
	}
	if rec.ProcessingState == DocumentAnalyzing {
		if _, ok := s.activeRun(id); ok {
			return nil, &DocumentError{State: rec.ProcessingState, Message: "Analysis is already running"}
		}
		// A previous process left the state analyzing with no live run; supersede it.
	}
	if strings.TrimSpace(rec.ExtractedText) == "" {
		if err := s.extractDocumentText(rec, "analyze"); err != nil {
			return nil, err
		}
		rec, err = s.store.GetProfileDocument(id)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(rec.ExtractedText) == "" {
		return nil, &DocumentError{State: rec.ProcessingState, Message: "No extractable text; provide a text/OCR export"}
	}
	cfg := s.cfg.Get()
	providerName, provider, err := extractionProvider(cfg)
	if err != nil {
		return nil, &AnalyzerUnavailableError{Reason: err.Error()}
	}
	runID := uuid.NewString()
	meta := map[string]any{
		"runId": runID, "startedAt": nowISO(), "provider": providerName,
		"model": provider.Model, "phase": "analysis", "contentHash": rec.ContentHash,
	}
	metaJSON, _ := json.Marshal(meta)
	if err := s.store.StartProfileDocumentAnalysis(id, string(metaJSON)); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.addRun(id, runID, cancel)
	go s.runAnalysis(ctx, rec, runID, providerName, provider)
	return s.GetDocument(id)
}

// CancelDocumentAnalysis cancels the in-flight provider call when this process owns the run and
// marks the document cancelled. A late provider response is discarded by the run-id guard.
func (s *Service) CancelDocumentAnalysis(id string) (*Document, error) {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	rec, err := s.store.GetProfileDocument(id)
	if err != nil {
		return nil, err
	}
	if rec.ProcessingState != DocumentAnalyzing {
		return nil, &DocumentError{State: rec.ProcessingState, Message: "No analysis is running for this document"}
	}
	if run, ok := s.activeRun(id); ok {
		run.cancel()
	}
	meta := map[string]any{"cancelledAt": nowISO(), "phase": "analysis"}
	metaJSON, _ := json.Marshal(meta)
	if err := s.store.UpdateProfileDocumentState(id, DocumentCancelled, string(metaJSON),
		"Analysis cancelled; no facts were published."); err != nil {
		return nil, err
	}
	return s.GetDocument(id)
}

// DocumentDeletion is what DeleteDocument removed, as the API reports it.
type DocumentDeletion struct {
	DocumentID       string `json:"documentId"`
	Deleted          bool   `json:"deleted"`
	FileRemoved      bool   `json:"fileRemoved"`
	TextRemoved      bool   `json:"textRemoved"`
	ProposalsRemoved int    `json:"proposalsRemoved"`
	FactsKept        int    `json:"factsKept"`
}

// DeleteDocument removes one document: its extracted text, its unaccepted proposals and, when
// the application stored it, the original file. Facts the owner already accepted stay in the
// profile and are listed as coming from a deleted file. A running analysis is cancelled first.
// A configured file that the owner keeps in the data directory is forgotten but never deleted.
func (s *Service) DeleteDocument(id string) (*DocumentDeletion, error) {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if run, ok := s.activeRun(id); ok {
		run.cancel()
		s.runsMu.Lock()
		delete(s.runs, id)
		s.runsMu.Unlock()
	}
	rec, err := s.store.GetProfileDocument(id)
	if err != nil {
		return nil, err
	}
	managed := managedDocumentDir(id)
	managedFile := rec.StoragePath != "" &&
		strings.HasPrefix(filepath.Clean(rec.StoragePath), managed+string(filepath.Separator))
	// A file the owner configured is remembered as removed so the importer does not bring it back.
	configured := ""
	if rec.StoragePath != "" && !managedFile {
		configured = "file:" + filepath.Clean(rec.StoragePath)
	}
	result, err := s.store.DeleteProfileDocument(id, configured)
	if err != nil {
		return nil, err
	}
	out := &DocumentDeletion{
		DocumentID:       id,
		Deleted:          true,
		TextRemoved:      strings.TrimSpace(result.Removed.ExtractedText) != "",
		ProposalsRemoved: result.ProposalsRemoved,
		FactsKept:        result.FactsKept,
	}
	if managedFile {
		dir, err := s.dirs.Under(managed)
		if err == nil {
			err = os.RemoveAll(dir)
		}
		if err != nil {
			// The record is already gone; the leftover file is reported, not hidden.
			fmt.Fprintf(os.Stderr, "profile document %s: remove stored file: %v\n", id, err)
		} else {
			out.FileRemoved = true
		}
	}
	return out, nil
}

// managedDocumentDir is the directory, relative to the data directory, where CreateDocument
// keeps the original of one uploaded document.
func managedDocumentDir(id string) string {
	return filepath.Join("evidence", "profile-documents", id)
}

func (s *Service) activeRun(documentID string) (activeRun, bool) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	run, ok := s.runs[documentID]
	return run, ok
}

func (s *Service) runAnalysis(ctx context.Context, rec *store.ProfileDocumentRecord, runID, providerName string, provider providerConfig) {
	defer func() {
		if run, ok := s.takeRun(rec.ID, runID); ok {
			run.cancel()
		}
	}()
	facts, metadata, err := s.extractFacts(ctx, rec, runID, providerName, provider)
	if err != nil {
		state := DocumentStateError
		message := err.Error()
		if ctx.Err() != nil {
			state = DocumentCancelled
			message = "Analysis cancelled; no facts were published."
		}
		meta := map[string]any{
			"runId": runID, "finishedAt": nowISO(), "phase": "analysis", "error": message,
			"errorKind": analysisErrorKind(err), "provider": providerName, "model": provider.Model,
			"contentHash": rec.ContentHash,
		}
		if ctx.Err() != nil {
			meta["cancelled"] = true
		}
		metaJSON, _ := json.Marshal(meta)
		_, _ = s.store.CompleteProfileDocumentAnalysis(rec.ID, runID, state, string(metaJSON), message, nil)
		return
	}
	if facts == nil {
		facts = []*store.ProfileFactRecord{}
	}
	// Mark facts whose value is already present in the confirmed profile. The user does not
	// need to act on those; they are stored as profile_match and not shown as decisions.
	if len(facts) > 0 {
		if current, getErr := s.Get(); getErr == nil {
			for _, f := range facts {
				if matchFactToProfile(current.Data, f) {
					f.State = FactMatches
				}
			}
		}
		// If Get() fails the facts remain "proposed"; the matching step is best-effort.
	}
	metaJSON, _ := json.Marshal(metadata)
	published, err := s.store.CompleteProfileDocumentAnalysis(rec.ID, runID, DocumentAnalyzed, string(metaJSON), "", facts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "profile analysis %s publication failed: %v\n", rec.ID, err)
		failureMeta, _ := json.Marshal(map[string]any{
			"runId": runID, "finishedAt": nowISO(), "phase": "analysis", "error": err.Error(),
			"errorKind": "publication", "contentHash": rec.ContentHash,
		})
		_, _ = s.store.FailProfileDocumentAnalysis(rec.ID, runID, string(failureMeta),
			"Analysis publication failed: "+err.Error())
		return
	}
	if !published {
		// Cancelled/superseded runs are intentionally silent. A same-run refusal means the stored
		// source changed while the provider call was in flight; make that visible as an error.
		failureMeta, _ := json.Marshal(map[string]any{
			"runId": runID, "finishedAt": nowISO(), "phase": "analysis",
			"contentHash": rec.ContentHash, "error": "stored source changed during analysis",
			"errorKind": "source_changed",
		})
		_, _ = s.store.FailProfileDocumentAnalysis(rec.ID, runID, string(failureMeta),
			"Analysis result discarded because the stored source changed.")
	}
}

// providerConfig is the small subset the analysis needs; it keeps the store/domain package free
// of a second provider stack while assess owns transport.
type providerConfig struct {
	Type         string
	Model        string
	Endpoint     string
	APIKeyEnv    string
	TimeoutMS    int
	ContextChars int
	Think        *bool
}

func extractionProvider(cfg *config.Config) (string, providerConfig, error) {
	name := cfg.Providers.Stages.Extraction
	if name == "" {
		return "", providerConfig{}, fmt.Errorf("extraction provider is not configured; set stages.extraction in providers.yaml")
	}
	cp, ok := cfg.Providers.Providers[name]
	if !ok {
		return "", providerConfig{}, fmt.Errorf("extraction provider %q is not defined in providers.yaml", name)
	}
	timeout := cp.TimeoutMS
	if limit := cfg.Runtime.EffectiveProfileExtractionTimeoutMS(); limit > 0 && (timeout <= 0 || limit < timeout) {
		timeout = limit
	}
	// Extraction is a schema-constrained JSON task. A reasoning/thinking model spends its
	// configured token budget on a hidden trace before emitting the answer; measured on
	// qwen3.5:9b, ~2,300 reasoning tokens took about two minutes and exceeded the 120 s call
	// timeout. Ask the ollama adapter to skip thinking for extraction while leaving other
	// stages (assessment/preparation) on the provider's default.
	noThinking := false
	return name, providerConfig{
		Type: cp.Type, Model: cp.Model, Endpoint: cp.Endpoint, APIKeyEnv: cp.APIKeyEnv,
		TimeoutMS: timeout, ContextChars: cp.ContextChars, Think: &noThinking,
	}, nil
}

func (s *Service) importLegacyDocuments(profileID string, existing []*store.ProfileDocumentRecord) error {
	cfg := s.cfg.Get()
	byStorage := map[string]bool{}
	bySource := map[string]bool{}
	byLegacy := map[string]bool{}
	for _, rec := range existing {
		if rec.StoragePath != "" {
			byStorage[rec.StoragePath] = true
			byLegacy["file:"+filepath.Clean(rec.StoragePath)] = true
		}
		if rec.SourceURL != "" {
			bySource[rec.SourceURL] = true
		}
		var meta map[string]any
		if rec.AnalysisMetadata != "" && json.Unmarshal([]byte(rec.AnalysisMetadata), &meta) == nil {
			if id, ok := meta["legacySource"].(string); ok && id != "" {
				byLegacy[id] = true
			}
		}
	}
	for _, originalRel := range cfg.Candidate.Files {
		rel := filepath.Clean(originalRel)
		if rel == "." || rel == "" {
			continue
		}
		sourceID := "file:" + rel
		if byLegacy[sourceID] || byStorage[rel] {
			continue
		}
		if filepath.IsAbs(rel) {
			if err := s.insertLegacyMissing(profileID, filepath.Base(rel), "", "Absolute paths are not allowed; configure a path relative to the data directory", sourceID, originalRel); err != nil {
				return err
			}
			byLegacy[sourceID] = true
			continue
		}
		path, err := s.dirs.Under(rel)
		if err != nil {
			if err := s.insertLegacyMissing(profileID, filepath.Base(rel), "", err.Error(), sourceID, originalRel); err != nil {
				return err
			}
			byLegacy[sourceID] = true
			continue
		}
		if _, statErr := os.Stat(path); statErr != nil {
			if err := s.insertLegacyMissing(profileID, filepath.Base(rel), "", "Configured file is missing under the data directory", sourceID, originalRel); err != nil {
				return err
			}
			byLegacy[sourceID] = true
			continue
		}
		rec := &store.ProfileDocumentRecord{
			ID:               uuid.NewString(),
			ProfileID:        profileID,
			Kind:             extract.Kind(rel),
			OriginalName:     filepath.Base(rel),
			StoragePath:      rel,
			ProcessingState:  DocumentImported,
			AnalysisMetadata: legacyMetadata(sourceID, originalRel),
		}
		if err := s.store.CreateProfileDocument(rec); err != nil {
			return err
		}
		byStorage[rel] = true
		byLegacy[sourceID] = true
	}

	var cached *candidate.CandidateContext
	if raw, err := s.store.Context(); err == nil && raw != nil {
		if cc, err := candidate.MapToContext(raw); err == nil {
			cached = cc
		}
	}
	for _, p := range cfg.Candidate.Profiles {
		sourceID := "profile:" + p.URL
		if p.URL == "" || bySource[p.URL] || byLegacy[sourceID] {
			continue
		}
		var src *candidate.EvidenceSource
		if cached != nil {
			for i := range cached.Sources {
				if cached.Sources[i].ID == p.ID {
					src = &cached.Sources[i]
					break
				}
			}
		}
		rec := &store.ProfileDocumentRecord{
			ID:           uuid.NewString(),
			ProfileID:    profileID,
			Kind:         "remote_profile",
			OriginalName: p.ID,
			SourceURL:    p.URL,
		}
		if src != nil {
			rec.ContentHash = src.Hash
			rec.ExtractedText = src.Text
			rec.Error = src.Error
			rec.ProcessingState = DocumentImported
			switch src.Status {
			case "available":
				if strings.TrimSpace(src.Text) == "" {
					rec.ProcessingState = DocumentMissing
				}
			case "stale":
				rec.ProcessingState = DocumentStale
			default:
				rec.ProcessingState = DocumentMissing
			}
			meta := map[string]any{
				"retrievedAt": src.RetrievedAt, "category": src.Category, "scope": src.Scope,
				"legacySource": sourceID, "configuredValue": p.URL,
			}
			encoded, _ := json.Marshal(meta)
			rec.AnalysisMetadata = string(encoded)
		} else if p.Export != "" {
			export, err := s.dirs.Under(filepath.Clean(p.Export))
			if err == nil {
				if _, statErr := os.Stat(export); statErr == nil {
					rec.Kind = "remote_profile_export"
					rec.StoragePath = filepath.Clean(p.Export)
					rec.ProcessingState = DocumentImported
				}
			}
			if rec.StoragePath == "" {
				rec.ProcessingState = DocumentMissing
				rec.Error = "Configured export is missing under the data directory"
			}
		} else {
			rec.ProcessingState = DocumentMissing
			rec.Error = "Remote profile not collected; refresh explicitly. Authenticated scraping is not performed."
		}
		if rec.AnalysisMetadata == "" {
			rec.AnalysisMetadata = legacyMetadata(sourceID, p.URL)
		}
		if err := s.store.CreateProfileDocument(rec); err != nil {
			return err
		}
		bySource[p.URL] = true
		byLegacy[sourceID] = true
	}
	return nil
}

func (s *Service) insertLegacyMissing(profileID, name, storage, message, sourceID, configuredValue string) error {
	rec := &store.ProfileDocumentRecord{
		ID:               uuid.NewString(),
		ProfileID:        profileID,
		Kind:             extract.Kind(name),
		OriginalName:     name,
		StoragePath:      storage,
		ProcessingState:  DocumentMissing,
		Error:            message,
		AnalysisMetadata: legacyMetadata(sourceID, configuredValue),
	}
	return s.store.CreateProfileDocument(rec)
}

// legacyMetadata preserves the configured source identity/value as metadata only. Invalid or
// absolute configured paths never become a machine storage path.
func legacyMetadata(sourceID, configuredValue string) string {
	meta, _ := json.Marshal(map[string]any{
		"legacySource": sourceID, "configuredValue": configuredValue,
	})
	return string(meta)
}

func allowedDocumentType(types []string, kind string) bool {
	for _, t := range types {
		if strings.TrimPrefix(strings.ToLower(strings.TrimSpace(t)), ".") == kind {
			return true
		}
	}
	return false
}

func documentFromRecord(rec *store.ProfileDocumentRecord, counts map[string]int) *Document {
	d := &Document{
		ID:              rec.ID,
		ProfileID:       rec.ProfileID,
		Kind:            rec.Kind,
		OriginalName:    rec.OriginalName,
		SourceURL:       rec.SourceURL,
		ContentHash:     rec.ContentHash,
		ProcessingState: rec.ProcessingState,
		HasText:         strings.TrimSpace(rec.ExtractedText) != "",
		TextChars:       len(strings.TrimSpace(rec.ExtractedText)),
		HasOriginal:     rec.StoragePath != "" && rec.ProcessingState != DocumentMissing,
		Error:           rec.Error,
		CreatedAt:       rec.CreatedAt,
		UpdatedAt:       rec.UpdatedAt,
		FactCounts:      counts,
	}
	if d.FactCounts == nil {
		d.FactCounts = map[string]int{}
	}
	if rec.AnalysisMetadata != "" {
		var meta map[string]any
		if json.Unmarshal([]byte(rec.AnalysisMetadata), &meta) == nil {
			d.AnalysisMetadata = meta
		}
	}
	return d
}

func factCountsByDocument(facts []*store.ProfileFactRecord) map[string]map[string]int {
	out := map[string]map[string]int{}
	for _, f := range facts {
		if f.DocumentID == nil {
			continue
		}
		if out[*f.DocumentID] == nil {
			out[*f.DocumentID] = map[string]int{}
		}
		out[*f.DocumentID][f.State]++
	}
	return out
}

func factCounts(facts []*store.ProfileFactRecord) map[string]int {
	out := map[string]int{}
	for _, f := range facts {
		out[f.State]++
	}
	return out
}

func nowISO() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }
