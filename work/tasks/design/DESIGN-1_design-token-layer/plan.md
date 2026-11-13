---
estimate: 10
---
# DESIGN-1 — plan

**Author:** coordinator
**Created:** 2026-11-06

## Statement of work

Introduce a design-token layer for the frontend (now at `sifty/web`) and move the scattered styling onto it, as a pure refactor that leaves the app looking identical. Define a small, role-named set of CSS custom properties on `:root` — colours (surface/background, text, muted, accent, border, danger, web3 accent), plus spacing, radii and typography — structured so a light/dark theme can be defined later without wiring a UI toggle now. Replace the ~44 hardcoded hex colours across the CSS files with those tokens, mapping each to the nearest role token and recording any deliberate unification of near-duplicates. Remove the static presentational inline `style={{…}}` blocks and the 6 hex colours from the components into token-based CSS classes, leaving inline only styles that genuinely depend on runtime values. Verify `tsc --noEmit` and `vite build` pass and that the rendered app is visually unchanged. Depends on GO-11 (frontend moved to `sifty/web`) being merged.

## Steps

### Phase 1 — Token layer

*Gate: tokens are defined on `:root` and `tsc --noEmit` / `vite build` pass.*

- [ ] 1.1 — Define the role-named token set (colour, spacing, radius, typography) as CSS custom properties on `:root` in a single tokens stylesheet imported first; structure colours so a `prefers-color-scheme` / `data-theme` override can be added later.
- [ ] 1.2 — Build an explicit mapping from each current hex colour to a token (listed in the card log), unifying near-duplicates only deliberately and recording each merge.

### Phase 2 — Apply tokens, remove hardcoded values

*Gate: no raw hex colour remains in the CSS or in `.tsx` (except any documented exception); `tsc --noEmit` and `vite build` pass.*

- [ ] 2.1 — Replace hardcoded hex colours across the CSS files with the tokens.
- [ ] 2.2 — Move the static inline `style={{…}}` blocks and the hex colours out of the components into token-based CSS classes; leave only genuinely dynamic inline styles, each noted.

### Phase 3 — Verify visual parity

*Gate: `tsc --noEmit` and `vite build` pass; appearance unchanged.*

- [ ] 3.1 — Run `tsc --noEmit` and `vite build`; record output.
- [ ] 3.2 — (Coordinator) On staging, confirm the app looks the same as before the refactor (spot-check the main screens), and that the documented colour unifications are the only intended differences.

## Acceptance criteria

- A role-named design-token set (colours, spacing, radii, typography) is defined on `:root` in one place, structured so a light/dark theme can be added later.
- No raw hex colour remains in the CSS files or in the `.tsx` components, except documented exceptions; the components carry no static presentational inline `style={{…}}` (only genuinely dynamic ones remain, each noted).
- The card log holds the colour→token mapping and lists every deliberate unification of near-duplicate colours.
- The app is visually unchanged apart from those listed unifications; `tsc --noEmit` and `vite build` pass.
