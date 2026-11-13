// Playwright worker: the Go core's browser-automation sidecar for Stage 2 form inspection, over the
// protocol in docs/worker-protocol.md. The Go core has no browser binding, so every rendered-DOM
// read happens here on the official Playwright. The core invokes `inspectForm` with either a live
// `url` to navigate, or `html` to inspect directly (used by the parity check, so the real rendering
// and field-reading run without network access). The response is the questionnaire the core stores.
//
// The field-reading function readRenderedFields is serialized into each inspected frame and runs in
// the page; it mirrors the current TypeScript application's rendered-fields reader.
import process from 'node:process';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import { launchOptions, contextOptions, addFingerprintScripts, detectBlockSignal, createDiscipline } from './automation.mjs';

const hash = (s) => createHash('sha256').update(s).digest('hex');
const now = () => new Date().toISOString();

// readRenderedFields runs in the page. It collects the visible, accessible questionnaire controls
// and returns one descriptor per control. Keep every browser-only helper inside this function so it
// serializes cleanly into frame.evaluate.
function readRenderedFields() {
  const controls = 'input,textarea,select,[role=combobox],[role=radio],[role=checkbox]';
  const ignored = 'script,style,template,[hidden],[aria-hidden="true"],[inert]';
  const noise = 'input,textarea,select,button,[role=status],[role=alert],.application-field,.resume-upload-failure,.resume-upload-working,.resume-upload-success,.dropdown-container,[data-qa="privacy-link"]';
  const normalize = (t) => t.replace(/\s+/g, ' ').trim();
  const visible = (el) => { for (let node = el; node; node = node.parentElement) { const css = getComputedStyle(node); if (node.matches(ignored) || css.display === 'none' || ['hidden', 'collapse'].includes(css.visibility)) return false; } return true; };
  function text(el, exclude = noise) {
    if (!el || !visible(el)) return '';
    const pieces = []; let length = 0, visited = 0;
    function walk(node) {
      if (++visited > 4000 || length > 16000) return;
      if (node.nodeType === Node.TEXT_NODE) { pieces.push(node.textContent ?? ''); length += (node.textContent ?? '').length; return; }
      if (!(node instanceof Element) || !visible(node) || node.matches(exclude)) return;
      for (const child of node.childNodes) walk(child);
      pieces.push(' ');
    }
    walk(el); return normalize(pieces.join(''));
  }
  const rootElement = (el, id) => el.getRootNode().getElementById(id);
  const refs = (el, attribute) => (el.getAttribute(attribute) ?? '').split(/\s+/).filter(Boolean).map((id) => text(rootElement(el, id))).filter(Boolean).join(' ');
  function nativeLabel(el) {
    const aria = refs(el, 'aria-labelledby') || normalize(el.getAttribute('aria-label') ?? ''); if (aria) return aria;
    return Array.from(el.labels ?? []).map((label) => {
      const members = Array.from(label.querySelectorAll(controls)).filter((c) => c.type !== 'hidden');
      if (members.length > 1) return '';
      const markers = Array.from(label.querySelectorAll('.application-label')).filter((m) => m.closest('label') === label);
      return markers.length === 1 ? text(markers[0]) : text(label);
    }).filter(Boolean).join(' ') || (['radio', 'checkbox'].includes(el.getAttribute('role') ?? '') ? text(el) : '');
  }
  const generic = (label) => !label || /^(?:type (?:your )?(?:response|answer)(?: here)?|enter (?:your )?(?:response|answer|text)(?: here)?|your (?:response|answer)|answer here|select(?: an? option)?|choose(?: an? option)?)[\s.!…]*$/i.test(label);
  function localQuestion(el) {
    const question = el.closest('.application-question'); if (!question) return { label: '', scope: null };
    for (let parent = el.parentElement, depth = 0; parent && depth < 6; parent = parent.parentElement, depth++) {
      if (!question.contains(parent)) break;
      const labels = Array.from(parent.querySelectorAll('.application-label')).filter((label) => label.closest('.application-question') === question && visible(label) && text(label));
      const members = Array.from(parent.querySelectorAll(controls)).filter((c) => c.type !== 'hidden' && c.closest('.application-question') === question);
      const grouped = members.length > 0 && members.every((c) => ['radio', 'checkbox'].includes(c.type || c.getAttribute('role') || '') && c.name === el.name);
      if (labels.length === 1 && (members.length === 1 || grouped)) return { label: text(labels[0]), scope: parent };
      if (parent === question) break;
    }
    return { label: '', scope: question };
  }
  const elements = []; function visit(root) { elements.push(...root.querySelectorAll(controls)); for (const el of root.querySelectorAll('*')) if (el.shadowRoot) visit(el.shadowRoot); } visit(document);
  return elements.filter((el) => !el.closest('header,nav,footer,[role="navigation"],[role="search"]')
    && !['hidden', 'submit', 'button', 'reset', 'password', 'search'].includes(el.type)
    && !(!el.closest('form,[role="dialog"]') && /^(?:search\b|i[’']m looking for|select language$)/i.test(nativeLabel(el) || el.getAttribute('placeholder') || ''))
    && (el.type === 'file' || (el.getClientRects().length > 0 && visible(el)))).map((el) => {
      const input = el, role = el.getAttribute('role'), type = el.tagName === 'SELECT' ? 'select' : el.tagName === 'TEXTAREA' ? 'textarea' : role === 'combobox' ? 'select' : role === 'radio' ? 'radio' : role === 'checkbox' ? 'checkbox' : ['radio', 'checkbox', 'file'].includes(input.type) ? input.type : 'text';
      const legacy = normalize(Array.from(input.labels ?? []).map((label) => label.textContent).join(' ') || el.getAttribute('aria-label') || (el.getAttribute('aria-labelledby') ?? '').split(' ').map((id) => document.getElementById(id)?.textContent ?? '').join(' ') || el.getAttribute('placeholder') || el.getAttribute('name') || el.id);
      const controlName = input.name || el.id || legacy, standard = nativeLabel(el), local = localQuestion(el), grouped = ['radio', 'checkbox'].includes(type);
      const fieldset = el.closest('fieldset'), legend = fieldset ? Array.from(fieldset.children).find((c) => c.tagName === 'LEGEND') ?? null : null;
      const group = grouped ? (text(legend) || local.label) : '';
      const placeholder = normalize(el.getAttribute('placeholder') ?? ''), raw = group || (!generic(standard) ? standard : '') || local.label || (!generic(placeholder) ? placeholder : '');
      const labelStatus = !raw || raw.length > 8000 ? 'uncertain' : 'resolved';
      const label = labelStatus === 'resolved' ? raw.replace(/\s*[*✱✳]+\s*$/, '') : 'Question wording unavailable — review the source';
      const options = type === 'select' ? Array.from(el.options ?? []).map((o) => normalize(o.text)).filter(Boolean) : grouped && !generic(standard) ? [standard.replace(/\s*[*✱✳]+\s*$/, '')] : [];
      const helperNodes = local.scope ? Array.from(local.scope.querySelectorAll('.application-description,.description')).filter((n) => n.closest('.application-question') === el.closest('.application-question')) : [];
      const instructions = [...new Set([refs(el, 'aria-describedby'), ...helperNodes.map((n) => text(n))].filter((t) => t && t !== label))].join('\n');
      const warnings = [];
      if (labelStatus === 'uncertain') warnings.push('Question wording could not be resolved from a visible associated label or supported local question container. Review the source or add the question manually; no answer will be generated for this control.');
      if (grouped && !options.length) warnings.push('Visible option wording could not be resolved without interaction. Supply the choices manually.');
      if (instructions.length > 12000) warnings.push('Helper instructions exceed the read bound; review their full wording on the source.');
      return { label, labelStatus, instructions: instructions.slice(0, 12000), type, options, required: input.required || el.getAttribute('aria-required') === 'true' || /[*✱✳]\s*$/.test(raw), maxLength: input.maxLength > 0 ? input.maxLength : null, accept: input.accept ?? '', controlName, warnings };
    });
}

// question applies the field-schema defaults the current application's questionSchema applies, so a
// control descriptor becomes a stored question identically to the TypeScript path.
function question(item, frameUrl, language) {
  const { warnings, ...rest } = item;
  const q = {
    id: '', label: rest.label, instructions: rest.instructions, labelStatus: rest.labelStatus,
    type: rest.type, options: rest.options ?? [], required: !!rest.required,
    maxLength: rest.maxLength ?? null, accept: rest.accept ?? '', sourceUrl: frameUrl,
    origin: 'site', language, signature: '', controlName: rest.controlName ?? '',
  };
  q.id = hash(`${frameUrl}|${q.controlName}|${q.type}`);
  return q;
}

async function renderedQuestionnaire(page) {
  const fields = [];
  const gaps = ['Visible accessible controls only; hidden conditional and gated later steps have not been inspected. No fields were filled or submitted.'];
  const language = (await page.locator('html').getAttribute('lang')) || 'und';
  const host = (() => { try { return new URL(page.url()).hostname; } catch { return ''; } })();
  if (await page.locator('input[type=password]').count()) {
    return { url: page.url(), capturedAt: now(), language, status: 'login-required',
      family: /linkedin\.com$/.test(host) ? 'LinkedIn public job / account gate' : 'Application login gate',
      fields: [], gaps: [...gaps, 'Authentication/registration controls detected. Establish login in the owner browser and retry; account fields are not application questions.'] };
  }
  const frames = page.frames().filter((frame) => frame.url() && !/^https?:\/\/(?:accounts\.google\.com|(?:www\.)?google\.com\/recaptcha|li\.protechts\.net)/i.test(frame.url()));
  if (frames.length > 12) gaps.push('Additional embedded frames exceeded the inspection bound; their questionnaire coverage is unverified.');
  for (const frame of frames.slice(0, 12)) {
    try {
      const items = await frame.evaluate(`(${readRenderedFields.toString()})()`);
      for (const item of items) {
        for (const warning of item.warnings) gaps.push(`${item.controlName || 'Unidentified control'}: ${warning}`);
        const q = question(item, frame.url(), language);
        const existing = fields.find((f) => f.id === q.id);
        if (existing && ['radio', 'checkbox'].includes(q.type)) {
          existing.options = [...new Set([...existing.options, ...q.options])];
          existing.required ||= q.required;
        } else {
          fields.push(q);
        }
      }
    } catch (e) {
      gaps.push(`Frame unavailable: ${frame.url()}: ${String(e)}`);
    }
  }
  for (const file of [...fields]) {
    if (file.type === 'file' && /cover.?letter|lettre.?motivation/i.test(`${file.controlName} ${file.label}`)) {
      fields.push({ ...file, id: `${file.id}-text-draft`, type: 'textarea', label: language.startsWith('fr') ? 'Lettre de motivation (brouillon)' : 'Cover letter (editable draft for attachment)', controlName: `${file.controlName}:draft`, accept: '' });
      gaps.push('Cover-letter attachment includes a separate editable text draft; no cover-letter file is generated or uploaded.');
    }
  }
  for (const field of fields) field.signature = hash(JSON.stringify(field));
  const bodyText = ((await page.locator('body').innerText().catch(() => '')) || '').slice(0, 60000);
  const blocked = /verify you are human|just a moment|access denied|captcha/i.test(await page.title());
  const status = blocked ? 'blocked' : fields.length ? 'partial' : 'unsupported';
  if (!fields.length) gaps.push('No accessible questionnaire controls found. Establish the owner login, retry, or paste questions manually.');
  if (/next|continue|step \d|suivant|étape|weiter/i.test(bodyText)) gaps.push('Possible multi-step form: later gated steps are unverified.');
  if (fields.some((f) => f.type === 'select' && !f.options.length)) gaps.push('Custom dropdown options are not exposed without interaction; supply choices manually.');
  const family = /greenhouse|boards\.greenhouse/.test(host) ? 'Greenhouse rendered controls' : /lever\.co/.test(host) ? 'Lever rendered controls' : /ashbyhq/.test(host) ? 'Ashby accessible controls' : 'Generic rendered HTML/accessibility controls';
  return { url: page.url(), capturedAt: now(), language, status, gaps, fields: blocked ? [] : fields, family };
}

// clickApplyControl tries to find and click a visible Apply button/link that sits outside
// an application form (i.e. a landing-page CTA that reveals the form or navigates to it),
// then waits a bounded time for form inputs to appear. Falls through without error when no
// such control is found or inputs do not appear, so generic inline forms are read unchanged.
//
// ATS landing pages (Ashby, Greenhouse, Lever) are built as React SPAs, so the Apply
// button may take several seconds to render after the initial HTML load. The click may
// trigger a full-page navigation (Ashby navigates to …/application) or open a modal.
// Both are handled by waiting for form inputs after the interaction.
async function clickApplyControl(page) {
  // If the page already has visible application inputs, no click-through is needed.
  const inputsAlreadyVisible = await page
    .locator('input:not([type=hidden]):not([type=submit]):not([type=button]):not([type=reset])')
    .first().isVisible().catch(() => false);
  if (inputsAlreadyVisible) return;

  // Look for a prominent Apply button or link. ATS pages take several seconds to render
  // their controls after JS executes, so use an extended bounded wait.
  const applyLocator = page
    .locator('button, a, [role="button"]')
    .filter({ hasText: /^\s*apply\b/i })
    .first();

  try {
    // Extended timeout: React SPAs may take several seconds to render the Apply button.
    await applyLocator.waitFor({ state: 'visible', timeout: 8000 });
    // Click and wait for any resulting navigation to reach domcontentloaded.
    await Promise.all([
      page.waitForLoadState('domcontentloaded', { timeout: 8000 }).catch(() => {}),
      applyLocator.click(),
    ]);
  } catch {
    return; // No visible Apply button within the bounded wait; read the page as-is.
  }

  // Wait for the application form inputs to render on the resulting page.
  // This handles both modal-based forms and full-page navigations.
  try {
    await page.waitForSelector(
      'input:not([type=hidden]):not([type=submit]):not([type=button]):not([type=reset])',
      { state: 'visible', timeout: 8000 });
  } catch {
    // Inputs did not appear; proceed with whatever is now rendered.
  }
}

// _defaultDiscipline is used when the worker is run standalone (not invoked by Go).
// In production the Go BrowserDiscipline is the outer rate-limiter; this inner guard
// is a harmless safeguard for standalone use.
const _defaultDiscipline = createDiscipline();

// inspectFormWith is the testable core: params is the same as inspectForm, discipline
// carries the shared automation state (pacing, semaphore, cooldown).
export async function inspectFormWith(params, discipline) {
  const url = typeof params.url === 'string' ? params.url : null;
  const hostname = url ? (() => { try { return new URL(url).hostname; } catch { return null; } })() : null;

  // If this host is on cooldown after a previous block, return immediately without navigating.
  if (hostname && discipline.cooldown && discipline.cooldown.isBlocked(hostname)) {
    return {
      url: url, capturedAt: now(), language: 'und', status: 'backed-off', fields: [],
      gaps: [`Host ${hostname} is on cooldown after a previous block; retry after the cooldown lapses.`],
      family: 'Unknown',
    };
  }

  // Enforce a minimum interval before navigating to this host again (pacing).
  if (hostname && discipline.pacing) {
    await discipline.pacing.wait(hostname);
  }

  // Limit global concurrency: at most discipline.config.concurrency browsers navigate at once.
  if (discipline.semaphore) await discipline.semaphore.acquire();
  try {
    const browser = await chromium.launch(launchOptions());
    try {
      const context = await browser.newContext(contextOptions());
      await addFingerprintScripts(context);
      // Block non-GET/HEAD requests to prevent form submissions, with a targeted exemption
      // for POST requests to data-loading API endpoints (GraphQL, REST data APIs). ATS systems
      // such as Ashby fetch form questions via POST to their GraphQL API; blocking those
      // prevents the form from rendering. We allow POST only to paths that look like data APIs
      // (/api/, /graphql, /query/) while still blocking PUT/DELETE/PATCH and other POSTs.
      // Service-worker registration (which can persist state across runs) is always blocked.
      await context.route('**/*', async (route) => {
        const method = route.request().method();
        if (['GET', 'HEAD'].includes(method)) { route.continue(); return; }
        const resourceType = route.request().resourceType();
        if (resourceType === 'serviceworker') { try { await route.abort('blockedbyclient'); } catch { /* closing */ } return; }
        if (method === 'POST') {
          const url = route.request().url();
          const isDataApi = /\/(?:api|graphql|query)\b/i.test(new URL(url).pathname);
          if (isDataApi) { route.continue(); return; }
        }
        try { await route.abort('blockedbyclient'); } catch { /* closing */ }
      });
      const page = await context.newPage();
      if (typeof params.html === 'string') {
        await page.setContent(params.html, { waitUntil: 'domcontentloaded' });
      } else if (url) {
        // Navigate, catching download and non-HTML destinations so they become honest states
        // rather than raw errors.
        let navResponse = null;
        let navError = null;
        try {
          navResponse = await page.goto(url, { waitUntil: 'domcontentloaded' });
        } catch (e) {
          navError = String(e && e.message ? e.message : e);
        }
        // Detect block signals at the HTTP response level (403 / 429).
        const statusCode = navResponse?.status() ?? null;
        if (hostname && detectBlockSignal(statusCode, '')) {
          if (discipline.cooldown) discipline.cooldown.block(hostname);
          return {
            url, capturedAt: now(), language: 'und', status: 'backed-off', fields: [],
            gaps: [`Access blocked by ${hostname} (HTTP ${statusCode}); host put on cooldown.`],
            family: 'Unknown',
          };
        }
        if (navError) {
          const isDownloadOrNonHtml = /download is starting|net::ERR_ABORTED/i.test(navError);
          if (isDownloadOrNonHtml) {
            return {
              url, capturedAt: now(), language: 'und',
              status: 'no-applicable-form', fields: [],
              gaps: ['Navigation returned a download or non-HTML page; apply via the original listing.'],
              family: 'Unknown',
            };
          }
          throw new Error(navError);
        }
        // Detect block signals at the page level (challenge / captcha title).
        const pageTitle = await page.title().catch(() => '');
        if (hostname && detectBlockSignal(null, pageTitle)) {
          if (discipline.cooldown) discipline.cooldown.block(hostname);
          return {
            url, capturedAt: now(), language: 'und', status: 'backed-off', fields: [],
            gaps: [`Challenge or captcha page detected on ${hostname}; host put on cooldown.`],
            family: 'Unknown',
          };
        }
      } else {
        throw new Error('inspectForm requires params.html or params.url');
      }

      // Detect the ATS family. An explicit params.atsFamily overrides hostname detection (used
      // by test fixtures served as inline HTML, which have no real hostname).
      const host = (() => { try { return new URL(page.url()).hostname; } catch { return ''; } })();
      const detectedFamily = params.atsFamily ||
        (/ashbyhq\.com/.test(host) ? 'ashby' :
          /greenhouse\.io|boards\.greenhouse\.io/.test(host) ? 'greenhouse' :
            /lever\.co/.test(host) ? 'lever' : null);

      // For known ATS families, click through the Apply control to reveal the application form.
      if (detectedFamily) {
        await clickApplyControl(page);
      }

      return await renderedQuestionnaire(page);
    } finally {
      await browser.close();
    }
  } finally {
    if (discipline.semaphore) discipline.semaphore.release();
  }
}

// inspectForm is the public API for the stdin/stdout worker entry point.
export async function inspectForm(params) {
  return inspectFormWith(params, _defaultDiscipline);
}

// Run the stdin/stdout listener only when this file is the main entry point (not when imported
// by tests or other modules).
if (fileURLToPath(import.meta.url) === process.argv[1]) {
  let input = '';
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', (c) => { input += c; });
  process.stdin.on('end', async () => {
    try {
      const req = JSON.parse(input);
      if (req.method !== 'inspectForm') throw new Error(`unknown method ${req.method}`);
      const result = await inspectForm(req.params ?? {});
      process.stdout.write(JSON.stringify({ ok: true, result }));
    } catch (e) {
      process.stdout.write(JSON.stringify({ ok: false, error: String(e && e.message ? e.message : e) }));
    }
  });
}
