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
	"regexp"
	"strings"
	"time"
)

// Infer is the real provider-configurable model call, dispatching by provider type exactly as the
// current application does: the opencode and claude command-line tools over an isolated
// subprocess, and ollama or an OpenAI-compatible endpoint over loopback HTTP. It is the default
// Inference; tests inject a stub instead.
func Infer(p Provider, requestText string, schema any) (RawModel, error) {
	return InferContext(context.Background(), p, requestText, schema)
}

// InferContext is Infer with a caller-owned cancellation context. It exists so a profile
// document analysis can be cancelled while the configured provider call is in flight; the
// provider adapters already use exec.CommandContext and http.NewRequestWithContext.
func InferContext(ctx context.Context, p Provider, requestText string, schema any) (RawModel, error) {
	raw, err := inferRaw(ctx, p, requestText, schema)
	if err != nil {
		return RawModel{}, err
	}
	var m RawModel
	if err := decodeModelJSON(raw, &m); err != nil {
		return RawModel{}, err
	}
	return m, nil
}

// InferInto decodes a provider response into an alternate schema/decoder through the same
// provider adapters. The fit-assessment RawModel is one decoder; profile fact extraction is
// another. No second provider stack is introduced.
func InferInto(ctx context.Context, p Provider, requestText string, schema any, out any) error {
	raw, err := inferRaw(ctx, p, requestText, schema)
	if err != nil {
		return err
	}
	return decodeModelJSON(raw, out)
}

func inferRaw(ctx context.Context, p Provider, requestText string, schema any) (string, error) {
	if len(requestText) > p.ContextChars {
		return "", fmt.Errorf("context overflow: %d > %d; increase budget", len(requestText), p.ContextChars)
	}
	switch p.Type {
	case "opencode":
		return inferOpencode(ctx, p, requestText)
	case "claude":
		return inferClaude(ctx, p, requestText, schema)
	case "ollama", "compatible":
		return inferHTTP(ctx, p, requestText, schema)
	default:
		return "", fmt.Errorf("unknown provider type %q", p.Type)
	}
}

func isolatedEnv() []string {
	keys := []string{"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TMPDIR", "LANG", "TERM",
		"XDG_DATA_HOME", "XDG_CACHE_HOME"}
	var env []string
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

var fenceRe = regexp.MustCompile("(?s)^```(?:json)?\\s*|\\s*```$")

func decodeModelJSON(text string, out any) error {
	trimmed := strings.TrimSpace(text)
	trimmed = fenceRe.ReplaceAllString(trimmed, "")
	if err := json.Unmarshal([]byte(trimmed), out); err != nil {
		return fmt.Errorf("model output is not valid JSON: %w", err)
	}
	return nil
}

func runCLI(ctx context.Context, name string, args []string, stdin string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("%s: %w (%s)", name, err, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}

func inferOpencode(ctx context.Context, p Provider, requestText string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()
	env := append(isolatedEnv(),
		"OPENCODE_DISABLE_PROJECT_CONFIG=1",
		"OPENCODE_DISABLE_EXTERNAL_SKILLS=1",
		"OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1",
	)
	out, err := runCLI(ctx, "opencode",
		[]string{"run", "--pure", "--format", "json", "--model", p.Model, "--agent", "assessor"},
		requestText, env)
	if err != nil {
		return "", err
	}
	var text strings.Builder
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var ev struct {
			Type string `json:"type"`
			Part struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"part"`
			Error any `json:"error"`
		}
		if json.Unmarshal([]byte(line), &ev) != nil {
			continue
		}
		if ev.Type == "error" {
			return "", fmt.Errorf("opencode error: %v", ev.Error)
		}
		if strings.Contains(ev.Type, "tool") || ev.Part.Type == "tool" {
			return "", fmt.Errorf("provider attempted a tool call; assessment refused")
		}
		if ev.Type == "text" {
			text.WriteString(ev.Part.Text)
		}
	}
	if text.Len() == 0 {
		return "", fmt.Errorf("no text from opencode; verify subscription login and model")
	}
	return text.String(), nil
}

func inferClaude(ctx context.Context, p Provider, requestText string, schema any) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()
	schemaJSON, _ := json.Marshal(schema)
	out, err := runCLI(ctx, "claude",
		[]string{"-p", "--output-format", "json", "--model", p.Model, "--tools", "",
			"--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
			"--disable-slash-commands", "--no-session-persistence",
			"--permission-mode", "dontAsk", "--no-chrome", "--json-schema", string(schemaJSON)},
		requestText, isolatedEnv())
	if err != nil {
		return "", err
	}
	var res struct {
		IsError          bool            `json:"is_error"`
		StructuredOutput json.RawMessage `json:"structured_output"`
		Result           string          `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return "", fmt.Errorf("claude output is not valid JSON: %w", err)
	}
	if res.IsError {
		return "", fmt.Errorf("claude inference unavailable; check claude auth status")
	}
	if len(res.StructuredOutput) > 0 && json.Valid(res.StructuredOutput) {
		return string(res.StructuredOutput), nil
	}
	return res.Result, nil
}

func loopbackHost(h string) bool {
	return h == "127.0.0.1" || h == "localhost" || h == "[::1]" || h == "::1"
}

func inferHTTP(ctx context.Context, p Provider, requestText string, schema any) (string, error) {
	if p.Endpoint == "" {
		return "", fmt.Errorf("configure local endpoint")
	}
	u, err := url.Parse(p.Endpoint)
	if err != nil {
		return "", err
	}
	if !loopbackHost(u.Hostname()) {
		return "", fmt.Errorf("local inference endpoint must be loopback")
	}
	ollama := p.Type == "ollama"
	endpoint := strings.TrimRight(p.Endpoint, "/")
	if ollama {
		endpoint += "/api/chat"
	} else {
		endpoint += "/chat/completions"
	}
	payload := map[string]any{
		"model":    p.Model,
		"messages": []map[string]string{{"role": "user", "content": requestText}},
		"stream":   false,
	}
	if ollama {
		// Set the context window to match the configured budget. Using 1 token ≈ 4 chars is a
		// conservative estimate that accommodates multi-byte characters and CJK text. The upper
		// bound (128 k tokens) prevents requesting a window that most local models and hosts
		// cannot hold even when the configured budget is very large.
		const (
			charsPerToken = 4
			maxNumCtx     = 131072 // 128 k tokens — safe upper bound for local models
		)
		numCtx := p.ContextChars / charsPerToken
		if numCtx > maxNumCtx {
			numCtx = maxNumCtx
		}
		payload["options"] = map[string]any{"num_ctx": numCtx}
		payload["format"] = schema
		if p.Think != nil {
			payload["think"] = *p.Think
		}
	} else {
		payload["response_format"] = map[string]any{
			"type":        "json_schema",
			"json_schema": map[string]any{"name": "result", "schema": schema, "strict": true},
		}
	}
	body, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	if p.APIKeyEnv != "" {
		key := os.Getenv(p.APIKeyEnv)
		if key == "" {
			return "", fmt.Errorf("missing environment reference %s", p.APIKeyEnv)
		}
		req.Header.Set("authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := readLimited(resp.Body, 2<<20)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("local provider HTTP %d", resp.StatusCode)
	}
	if ollama {
		var b struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return "", err
		}
		return b.Message.Content, nil
	}
	var b struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		return "", err
	}
	if len(b.Choices) == 0 {
		return "", fmt.Errorf("local provider returned no choices")
	}
	return b.Choices[0].Message.Content, nil
}

func readLimited(r interface{ Read([]byte) (int, error) }, limit int) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if len(buf) > limit {
			return nil, fmt.Errorf("provider output exceeds limit")
		}
		if err != nil {
			return buf, nil
		}
	}
}
