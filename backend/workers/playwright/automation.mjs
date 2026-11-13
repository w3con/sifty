// automation.mjs — shared browser-automation discipline for the Playwright inspection worker.
// Provides the fingerprint utilities, pacing, concurrency bounding, block detection and
// cooldown that the worker uses to behave like a careful human visitor.

// BROWSER_UA matches the User-Agent the Go candidate-context HTTP fetcher sends, so the browser
// and the in-process fetcher present a consistent identity to the servers they contact.
export const BROWSER_UA =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36';

// launchOptions returns the chromium.launch() argument set that suppresses automation
// fingerprints without changing headless mode.
export function launchOptions() {
  return {
    headless: true,
    args: [
      '--disable-blink-features=AutomationControlled',
      '--no-first-run',
      '--no-default-browser-check',
      '--disable-infobars',
    ],
  };
}

// contextOptions returns the browser.newContext() argument set that presents a realistic
// User-Agent, viewport and locale to the inspected site.
export function contextOptions() {
  return {
    userAgent: BROWSER_UA,
    viewport: { width: 1280, height: 800 },
    locale: 'en-US',
  };
}

// addFingerprintScripts injects an init script into every page opened in the given context.
// The script suppresses navigator.webdriver, which Playwright sets to true in headless mode.
export async function addFingerprintScripts(context) {
  await context.addInitScript(() => {
    Object.defineProperty(navigator, 'webdriver', {
      get: () => false,
      configurable: true,
    });
  });
}

// --- Pacing ---

// HostPacer enforces a minimum interval between navigations to the same host.
// Call wait(hostname) before each navigation; it sleeps until at least intervalMs (plus a random
// fraction of jitterMs) has elapsed since the previous wait() call for the same host.
export class HostPacer {
  constructor(intervalMs, jitterMs) {
    this._intervalMs = intervalMs;
    this._jitterMs = jitterMs;
    this._lastNav = new Map(); // hostname -> timestamp (ms since epoch)
  }

  async wait(hostname) {
    const last = this._lastNav.get(hostname);
    if (last != null) {
      const jitter = this._jitterMs > 0 ? Math.random() * this._jitterMs : 0;
      const elapsed = Date.now() - last;
      const remaining = this._intervalMs + jitter - elapsed;
      if (remaining > 0) {
        await new Promise((resolve) => setTimeout(resolve, remaining));
      }
    }
    this._lastNav.set(hostname, Date.now());
  }
}

// --- Concurrency bounding ---

// Semaphore limits the number of tasks executing concurrently. Acquire before starting
// an operation and release when it ends; excess callers wait in a queue.
export class Semaphore {
  constructor(max) {
    this._max = max;
    this._running = 0;
    this._queue = [];
    // peak is the highest simultaneously-running count observed; useful in tests.
    this.peak = 0;
  }

  async acquire() {
    if (this._running < this._max) {
      this._running++;
      this.peak = Math.max(this.peak, this._running);
      return;
    }
    return new Promise((resolve) => {
      this._queue.push(resolve);
    });
  }

  release() {
    this._running--;
    if (this._queue.length > 0) {
      this._running++;
      this.peak = Math.max(this.peak, this._running);
      const next = this._queue.shift();
      next();
    }
  }
}

// --- Block detection ---

// detectBlockSignal returns true when the HTTP status code or the page title indicates that the
// site's anti-bot layer has flagged the request (rate-limit, access denied, or captcha challenge).
export function detectBlockSignal(statusCode, pageTitle) {
  if (statusCode === 403 || statusCode === 429) return true;
  if (/cloudflare|captcha|challenge|just a moment|access denied|verify you are human|enable javascript and cookies/i.test(pageTitle ?? '')) return true;
  return false;
}

// --- Cooldown registry ---

// CooldownRegistry tracks hosts that returned a block signal and prevents further navigation
// to them until the configured cooldown period has elapsed.
export class CooldownRegistry {
  constructor(cooldownMs) {
    this._cooldownMs = cooldownMs;
    this._blocked = new Map(); // hostname -> timestamp (ms) when blocked
  }

  // isBlocked returns true if hostname is still on cooldown.
  isBlocked(hostname) {
    const when = this._blocked.get(hostname);
    if (when == null) return false;
    if (Date.now() - when < this._cooldownMs) return true;
    this._blocked.delete(hostname);
    return false;
  }

  // block registers hostname as blocked at the current time.
  block(hostname) {
    this._blocked.set(hostname, Date.now());
  }
}

// createDiscipline builds a shared automation-state object from a config.
// Keys missing from config fall back to the conservative defaults.
// In production the Go BrowserDiscipline is the authoritative rate-limiter; this
// per-process guard is a harmless inner layer for the case where the worker is run
// standalone (e.g. during local development or manual testing).
export function createDiscipline(config = {}) {
  const intervalMs  = config.intervalMs  ?? 5000;
  const jitterMs    = config.jitterMs    ?? 3000;
  const concurrency = config.concurrency ?? 1;
  const cooldownMs  = config.cooldownMs  ?? 300_000; // 5 minutes
  return {
    pacing:    new HostPacer(intervalMs, jitterMs),
    semaphore: new Semaphore(concurrency),
    cooldown:  new CooldownRegistry(cooldownMs),
    config: { intervalMs, jitterMs, concurrency, cooldownMs },
  };
}
