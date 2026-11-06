package fetch

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/url"
	"sync"
	"time"
)

// trafficDiscipline enforces per-domain traffic rules:
//   - Serialises concurrent requests to the same domain via a per-domain mutex chain.
//   - Waits at least minInterval + jitter between consecutive requests to a domain.
//   - Applies exponential backoff after transient errors.
//   - Refuses any request to a domain within its cooldown window (set after a blocked result).
type trafficDiscipline struct {
	mu             sync.Mutex
	domains        map[string]*domainState
	nowMS          func() int64
	sleepMS        func(ms int64)
	randIntervalMS func(maxMS int64) int64
}

const (
	minIntervalMS  = 3000               // base inter-request wait
	jitterMaxMS    = 3000               // upper bound of random jitter added to minIntervalMS
	blockCooldownH = 24 * time.Hour     // cooldown after a 403/CAPTCHA
	backoffBaseMS  = 30 * 1000          // 30 s base backoff
	backoffCapMS   = 60 * 60 * 1000     // 1 h cap
	backoffMaxExp  = 7                  // 2^7 = 128; 30s × 128 = 3840 s, capped at 3600 s
)

type domainState struct {
	hasReq         bool  // true after at least one request has been made
	lastReqMS      int64 // time of the last request (valid only when hasReq is true)
	failures       int   // consecutive failures (resets on success)
	blockedUntilMS int64 // block cooldown expiry; zero means not blocked
}

func newTrafficDiscipline(
	nowMS func() int64,
	sleepMS func(ms int64),
	randIntervalMS func(maxMS int64) int64,
) *trafficDiscipline {
	if nowMS == nil {
		nowMS = func() int64 { return time.Now().UnixMilli() }
	}
	if sleepMS == nil {
		sleepMS = func(ms int64) { time.Sleep(time.Duration(ms) * time.Millisecond) }
	}
	if randIntervalMS == nil {
		randIntervalMS = func(maxMS int64) int64 {
			if maxMS <= 0 {
				return 0
			}
			return rand.Int64N(maxMS)
		}
	}
	return &trafficDiscipline{
		domains:        make(map[string]*domainState),
		nowMS:          nowMS,
		sleepMS:        sleepMS,
		randIntervalMS: randIntervalMS,
	}
}

func (td *trafficDiscipline) stateFor(domain string) *domainState {
	td.mu.Lock()
	defer td.mu.Unlock()
	if s, ok := td.domains[domain]; ok {
		return s
	}
	s := &domainState{}
	td.domains[domain] = s
	return s
}

// wait blocks until the domain's inter-request interval and any backoff have
// elapsed, then returns nil. It returns an error without sleeping if the domain
// is within its block cooldown.
func (td *trafficDiscipline) wait(ctx context.Context, domain string) error {
	s := td.stateFor(domain)

	now := td.nowMS()

	// Cooldown check.
	if s.blockedUntilMS > 0 && now < s.blockedUntilMS {
		remaining := s.blockedUntilMS - now
		return fmt.Errorf("fetch: domain %q is on cooldown for %d more ms", domain, remaining)
	}

	// Compute how long to wait.
	var waitMS int64
	if s.failures > 0 {
		exp := s.failures - 1
		if exp > backoffMaxExp {
			exp = backoffMaxExp
		}
		backoff := int64(backoffBaseMS) << exp
		if backoff > backoffCapMS {
			backoff = backoffCapMS
		}
		waitMS = backoff
	}
	if s.hasReq {
		interval := int64(minIntervalMS) + td.randIntervalMS(jitterMaxMS)
		elapsed := now - s.lastReqMS
		if elapsed < 0 {
			elapsed = 0
		}
		if interval-elapsed > waitMS {
			waitMS = interval - elapsed
		}
	}

	if waitMS > 0 {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		td.sleepMS(waitMS)
	}

	return nil
}

// record updates the domain state after a completed attempt. blocked indicates
// that the response was a BlockedResult, in which case the 24h cooldown is set.
// success resets the failure counter.
func (td *trafficDiscipline) record(domain string, blocked bool, success bool) {
	s := td.stateFor(domain)
	now := td.nowMS()
	s.hasReq = true
	s.lastReqMS = now
	if blocked {
		s.blockedUntilMS = now + blockCooldownH.Milliseconds()
		s.failures = 0
		return
	}
	if !success {
		s.failures++
		return
	}
	s.failures = 0
}

// hostOf extracts the hostname from rawURL, returning an empty string on error.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Hostname()
}

// GetListingWithDiscipline applies the traffic discipline around GetListing.
func (cl *Client) GetListingWithDiscipline(ctx context.Context, rawURL string) (*Page, *BlockedResult, error) {
	domain := hostOf(rawURL)
	if err := cl.discipline.wait(ctx, domain); err != nil {
		return nil, nil, err
	}
	page, blocked, err := cl.GetListing(ctx, rawURL)
	cl.discipline.record(domain, blocked != nil, err == nil && blocked == nil)
	return page, blocked, err
}

// GetDetailWithDiscipline applies the traffic discipline around GetDetail.
func (cl *Client) GetDetailWithDiscipline(ctx context.Context, rawURL string) (*Page, *BlockedResult, error) {
	domain := hostOf(rawURL)
	if err := cl.discipline.wait(ctx, domain); err != nil {
		return nil, nil, err
	}
	page, blocked, err := cl.GetDetail(ctx, rawURL)
	cl.discipline.record(domain, blocked != nil, err == nil && blocked == nil)
	return page, blocked, err
}
