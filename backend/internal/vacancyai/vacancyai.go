// Package vacancyai holds the two model passes of the two-score model (docs/specs/quick-match.md
// section 14): the vacancy-only extraction of requirements and conditions, and the comparison of
// those requirements with the confirmed profile. Each pass is one prompt, one typed schema and
// one validation step on top of assess.InferInto, so no second provider stack exists. The passes
// produce persisted facts and verdicts; the evaluator that turns them into numbers stays pure and
// lives elsewhere.
package vacancyai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/w3con/sifty/internal/assess"
)

// InferFunc is the provider call boundary. The default is assess.InferInto; tests inject a stub
// so no test performs a live provider request.
type InferFunc func(ctx context.Context, p assess.Provider, requestText string, schema any, out any) error

var (
	// ErrInvalidOutput marks a response that parsed but could not be used.
	ErrInvalidOutput = errors.New("model output rejected")
	// ErrContextOverflow means the request is larger than the provider's configured budget; the
	// call is not made.
	ErrContextOverflow = errors.New("request exceeds the provider context budget")
	// ErrNoText means there is no vacancy text to analyse.
	ErrNoText = errors.New("vacancy has no text to analyse")
)

// Call describes the physical provider calls one pass made. It is returned even when the pass
// fails, so a caller can account for the time spent.
type Call struct {
	ProviderName string
	ProviderType string
	Model        string
	Attempts     int
	RequestChars int
	// Duration is the wall time of the whole pass including retries; LastDuration is the final call.
	Duration     time.Duration
	LastDuration time.Duration
}

// Defaults used when a pass is built without explicit limits.
const (
	defaultAttempts = 3
	defaultBackoff  = 2 * time.Second
)

// pass is the shared machinery of both passes: one provider, a call boundary and the retry policy.
type pass struct {
	providerName string
	provider     assess.Provider
	infer        InferFunc
	attempts     int
	backoff      time.Duration
}

func (p pass) inferFunc() InferFunc {
	if p.infer != nil {
		return p.infer
	}
	return assess.InferInto
}

// run sends one request and hands the raw JSON answer to accept. A provider error, an answer that
// is not JSON and an answer accept refuses are all retried up to the attempt limit; a cancelled
// context and an oversized request are not. The last raw answer is returned even on failure so a
// diagnostic can show what the model said.
func (p pass) run(ctx context.Context, request string, schema any, accept func(raw json.RawMessage) error) (json.RawMessage, Call, error) {
	call := Call{
		ProviderName: p.providerName, ProviderType: p.provider.Type, Model: p.provider.Model,
		RequestChars: len(request),
	}
	if len(request) > p.provider.ContextChars {
		return nil, call, fmt.Errorf("%w: %d > %d characters", ErrContextOverflow, len(request), p.provider.ContextChars)
	}
	attempts := p.attempts
	if attempts <= 0 {
		attempts = defaultAttempts
	}
	started := time.Now()
	var raw json.RawMessage
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		call.Attempts = attempt
		raw = nil
		callStarted := time.Now()
		err = p.inferFunc()(ctx, p.provider, request, schema, &raw)
		call.LastDuration = time.Since(callStarted)
		if err == nil {
			err = accept(raw)
		}
		call.Duration = time.Since(started)
		if err == nil {
			return raw, call, nil
		}
		if ctx.Err() != nil {
			return raw, call, ctx.Err()
		}
		if attempt < attempts {
			select {
			case <-time.After(time.Duration(attempt) * p.backoff):
			case <-ctx.Done():
				return raw, call, ctx.Err()
			}
		}
	}
	return raw, call, fmt.Errorf("after %d attempts: %w", attempts, err)
}

func digest(parts ...any) string {
	h := sha256.New()
	for _, part := range parts {
		blob, _ := json.Marshal(part)
		h.Write(blob)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:10]
}
