/* Injected only by bin/sift-preview's local proxy. Production bundles never include it. */
(() => {
  if (window.__siftyPreviewInstalled) return;
  window.__siftyPreviewInstalled = true;

  const style = document.createElement('style');
  style.textContent = `
    #sifty-preview-banner { position: fixed; left: 12px; bottom: 12px; z-index: 2147483647;
      max-width: min(720px, calc(100vw - 24px)); font: 12px/1.45 ui-monospace, SFMono-Regular,
      Menlo, Consolas, monospace; color: #e6edf3; background: #161b22; border: 1px solid #30363d;
      border-radius: 8px; box-shadow: 0 8px 24px rgba(0,0,0,.45); padding: 8px 10px;
      display: flex; gap: 8px; align-items: flex-start; }
    #sifty-preview-banner[data-state="running"] { border-color: #238636; }
    #sifty-preview-banner[data-state="update"] { border-color: #9e6a03; }
    #sifty-preview-banner[data-state="failed"] { border-color: #da3633; }
    #sifty-preview-banner[data-state="down"] { border-color: #da3633; }
    #sifty-preview-banner .dot { width: 8px; height: 8px; border-radius: 50%; margin-top: 5px;
      background: #8b949e; flex: 0 0 auto; }
    #sifty-preview-banner[data-state="running"] .dot { background: #3fb950; }
    #sifty-preview-banner[data-state="update"] .dot { background: #d29922; }
    #sifty-preview-banner[data-state="failed"] .dot,
    #sifty-preview-banner[data-state="down"] .dot { background: #f85149; }
    #sifty-preview-banner .body { flex: 1 1 auto; min-width: 0; }
    #sifty-preview-banner .title { font-weight: 700; letter-spacing: .04em; }
    #sifty-preview-banner .meta { color: #8b949e; white-space: nowrap; overflow: hidden;
      text-overflow: ellipsis; }
    #sifty-preview-banner .error { color: #ffb3ad; white-space: pre-wrap; overflow-wrap: anywhere;
      margin-top: 4px; display: none; max-height: 10em; overflow: auto; }
    #sifty-preview-banner[data-expanded="1"] .error { display: block; }
    #sifty-preview-banner button { font: inherit; color: #e6edf3; background: #21262d;
      border: 1px solid #30363d; border-radius: 5px; padding: 2px 8px; cursor: pointer; }
    #sifty-preview-banner button:hover { background: #30363d; }
  `;
  document.head.appendChild(style);

  const banner = document.createElement('div');
  banner.id = 'sifty-preview-banner';
  banner.dataset.state = 'down';
  banner.innerHTML = '<span class="dot"></span><div class="body">' +
    '<div class="title">SIFTY PREVIEW</div><div class="meta">connecting…</div>' +
    '<div class="error"></div></div><button type="button" hidden>Reload</button>';
  const title = banner.querySelector('.title');
  const meta = banner.querySelector('.meta');
  const error = banner.querySelector('.error');
  const reload = banner.querySelector('button');
  reload.addEventListener('click', () => location.reload());
  banner.querySelector('.body').addEventListener('click', () => {
    if (error.textContent) banner.dataset.expanded = banner.dataset.expanded === '1' ? '0' : '1';
  });

  function mount() {
    if (!document.body) return setTimeout(mount, 20);
    document.body.appendChild(banner);
  }
  mount();

  function short(value) {
    return (value || '?').slice(0, 12);
  }

  let loadedGeneration = null;
  let edited = false;
  document.addEventListener('input', () => { edited = true; }, true);

  function render(status) {
    const requested = status.requested || {};
    const served = status.served || {};
    const build = status.build || {};
    const state = status.state || 'unknown';
    const pending = requested.fingerprint && served.fingerprint &&
      requested.fingerprint !== served.fingerprint;
    if (!loadedGeneration && served.generation) loadedGeneration = served.generation;
    const updated = loadedGeneration && served.generation &&
      loadedGeneration !== served.generation;
    const failed = state === 'failed';
    const down = !served.generation && !failed;
    banner.dataset.state = failed ? 'failed' : down ? 'down' : pending || updated ? 'update' : 'running';
    title.textContent = failed ? 'SIFTY PREVIEW · BUILD FAILED'
      : down ? 'SIFTY PREVIEW · STARTING'
      : updated ? 'SIFTY PREVIEW · UPDATE AVAILABLE'
      : pending ? 'SIFTY PREVIEW · BUILDING UPDATE'
      : 'SIFTY PREVIEW · SERVING CURRENT SOURCES';
    const dirty = requested.dirty ? ' dirty' : '';
    const servedText = served.generation
      ? `serving ${served.generation} (${short(served.commit)}${served.dirty ? ' dirty' : ''})`
      : 'no healthy generation yet';
    meta.textContent = `${short(requested.commit)}${dirty} · ${servedText}` +
      (build.count ? ` · builds ${build.count}` : '');
    error.textContent = build.lastError ?
      `${build.lastStage || 'build'}: ${build.lastError}` : '';
    reload.hidden = !updated;
    if (updated && !failed && !edited) location.reload();
  }

  async function poll() {
    try {
      const response = await fetch('/__preview/status', {cache: 'no-store'});
      if (!response.ok) throw new Error(`status HTTP ${response.status}`);
      render(await response.json());
    } catch (err) {
      banner.dataset.state = 'down';
      title.textContent = 'SIFTY PREVIEW · UNREACHABLE';
      meta.textContent = String(err && err.message || err);
    }
  }
  poll();
  setInterval(poll, 2000);
})();
