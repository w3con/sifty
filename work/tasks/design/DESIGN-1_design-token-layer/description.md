# DESIGN-1 — Design-token layer and style cleanup

**Epic:** design
**Created:** 2026-11-06
**Status:** founded

## Why this work exists

Before a real design pass, the styling has to be editable from one place. Today the look is scattered: the frontend defines only 6 CSS custom properties and one `:root`, with no theme support, while roughly 44 distinct colours are hardcoded across the CSS files and a further 6 hex values plus 13 inline `style={{…}}` blocks sit directly in the components (notably `Collection.tsx` and `Settings.tsx`). A designer (or a design tool) would have to hunt those values across CSS and components instead of editing one palette. This card introduces a design-token layer — colours, spacing, radii and typography as CSS custom properties on `:root`, structured so a light/dark theme can be defined — replaces the hardcoded values with those tokens, and pulls the inline component styles into token-based CSS classes. It is a refactor, not a redesign: the application must look the same after it as before. The actual visual redesign happens afterwards with the design tool, which will then edit tokens rather than scattered values.

## Must not touch

Component behaviour and logic — only the styling is reorganised, not what the components do. The markup structure beyond moving inline `style={{…}}` into classes. The backend, the build wiring and the data. Dynamic inline styles that genuinely depend on runtime values may stay inline if they cannot be expressed as a static class, but static presentational inline styles come out. The visual result: appearance stays identical unless a near-duplicate colour is deliberately unified, and every such unification is listed.

## Notes

Owner request (2026-11-06): create a card for design tokens as the step that makes "design separated from code" true, before the design pass.

Dependency and sequencing: this card operates on the frontend after it has moved to `sifty/web` (card GO-11). It must be handed to an executor only once GO-11 is merged, because it edits the same CSS and component files at their new location.

Grounding read by the coordinator (on main, before the move):
- The 6 CSS files (`style.css` ~12 KB, `polish.css`, `role.css`, `application.css`, `application-draft.css`, `queue-control.css`) are imported by the components via `import './x.css'`. After GO-11 they live under `sifty/web`.
- ~44 distinct hex colours across the CSS; 6 distinct hex colours inside `.tsx`; 13 inline `style={{…}}` blocks (7 in `Collection.tsx`, 4 in `Settings.tsx`, 1 each in `Icon.tsx` and `RoleContent.tsx`).
- Only 6 `--custom-properties` exist today, one `:root`, and no `prefers-color-scheme` / `data-theme` handling.
- Fonts are already tokenisable: IBM Plex Sans/Mono and Unbounded via `@fontsource`.

Coordinator note: keep the token set small and named by role (for example surface/background, text, muted, accent, border, danger, and a web3 accent), not by raw colour; map each current colour to the nearest role token; record any intentional merge of near-duplicates. Define the tokens so a dark/light theme can be added later (tokens on `:root`, overridable under a media query and a `data-theme` attribute) without requiring it to be wired to a UI toggle in this card.
