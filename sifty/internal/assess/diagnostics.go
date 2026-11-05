package assess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ProviderStatus is the diagnostic result for one configured provider.
type ProviderStatus struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Model  string `json:"model"`
	Status string `json:"status"`
}

// Diagnostics checks the availability of each configured provider and returns a status list.
// It matches the TypeScript providerDiagnostics function.
func Diagnostics(providers map[string]Provider) []ProviderStatus {
	var result []ProviderStatus
	for name, p := range providers {
		status := checkProvider(p)
		result = append(result, ProviderStatus{Name: name, Type: p.Type, Model: p.Model, Status: status})
	}
	return result
}

func checkProvider(p Provider) string {
	switch p.Type {
	case "claude":
		return checkClaude(p)
	case "opencode":
		return checkOpencode(p)
	case "ollama", "compatible":
		return checkLocal(p)
	default:
		return fmt.Sprintf("unknown provider type %q", p.Type)
	}
}

func checkClaude(p Provider) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", "auth", "status")
	cmd.Env = isolatedEnv()
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run() // exit 1 when not logged in — still parses
	var status struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &status); err != nil {
		return fmt.Sprintf("CLI unavailable: %v", err)
	}
	if status.LoggedIn {
		return "Subscription login available; inference smoke still required"
	}
	return "Not logged in: run claude auth login manually"
}

func checkOpencode(p Provider) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "opencode", "--version")
	cmd.Env = isolatedEnv()
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("CLI unavailable: %v", err)
	}
	version := strings.TrimSpace(out.String())
	return fmt.Sprintf("CLI %s available; run provider-check to verify current subscription/model", version)
}

func checkLocal(p Provider) string {
	if p.Endpoint == "" {
		return "Configure loopback endpoint"
	}
	u, err := url.Parse(p.Endpoint)
	if err != nil || !loopbackHost(u.Hostname()) {
		return "Configure loopback endpoint"
	}

	var resolvedKey string
	if p.APIKeyEnv != "" {
		resolvedKey = os.Getenv(p.APIKeyEnv)
		if resolvedKey == "" {
			return fmt.Sprintf("Set %s", p.APIKeyEnv)
		}
	}

	var tagsURL string
	if p.Type == "ollama" {
		tagsURL = strings.TrimRight(p.Endpoint, "/") + "/api/tags"
	} else {
		tagsURL = strings.TrimRight(p.Endpoint, "/") + "/models"
	}

	client := &http.Client{Timeout: 2500 * time.Millisecond}
	req, err := http.NewRequest("GET", tagsURL, nil)
	if err != nil {
		return fmt.Sprintf("Local runtime unavailable at %s: %v. Start an installed runtime and set its actual model ID in config/providers.yaml; queue is paused, no paid fallback.", p.Endpoint, err)
	}
	if resolvedKey != "" {
		req.Header.Set("authorization", "Bearer "+resolvedKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Local runtime unavailable at %s: %v. Start an installed runtime and set its actual model ID in config/providers.yaml; queue is paused, no paid fallback.", p.Endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Sprintf("Local runtime unavailable at %s: HTTP %d. Start an installed runtime and set its actual model ID in config/providers.yaml; queue is paused, no paid fallback.", p.Endpoint, resp.StatusCode)
	}

	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"` // ollama
		Data []struct {
			ID string `json:"id"`
		} `json:"data"` // compatible
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)

	var models []string
	if p.Type == "ollama" {
		for _, m := range body.Models {
			models = append(models, m.Name)
		}
	} else {
		for _, m := range body.Data {
			models = append(models, m.ID)
		}
	}

	for _, m := range models {
		if m == p.Model {
			return "Installed model available"
		}
	}
	if len(models) == 0 {
		return fmt.Sprintf("Set config/providers.yaml model to an installed ID. Installed: none")
	}
	return fmt.Sprintf("Set config/providers.yaml model to an installed ID. Installed: %s", strings.Join(models, ", "))
}

// smokePayload is the trivial JSON structure sent for a connectivity smoke check.
var smokePayload = map[string]any{
	"score":   0,
	"summary": "Minimal provider connectivity check. No candidate data supplied.",
	"evidence": []any{},
	"gaps":    []any{"This is a smoke check, not an assessment."},
}

// SmokeCheck runs a trivial inference call against the named provider and returns the parsed
// assessment model output. It matches the TypeScript providerSmoke function.
// The inferenceFn parameter is optional; if nil, the real Infer function is used.
func SmokeCheck(p Provider, inferenceFn Inference) (RawModel, error) {
	if inferenceFn == nil {
		inferenceFn = Infer
	}
	payloadJSON, _ := json.Marshal(smokePayload)
	prompt := fmt.Sprintf("Return only this JSON, no tools: %s", string(payloadJSON))
	return inferenceFn(p, prompt, map[string]any{
		"type":  "object",
		"title": "SmokeResult",
		"properties": map[string]any{
			"score":    map[string]any{"type": "number"},
			"summary":  map[string]any{"type": "string"},
			"evidence": map[string]any{"type": "array"},
			"gaps":     map[string]any{"type": "array"},
		},
		"required": []string{"score", "summary", "evidence", "gaps"},
	})
}
