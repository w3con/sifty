# NAVIGATION-3 — executor handback

Steps 1.1 through 3.1 are complete and committed on branch `work/NAVIGATION-3`. Step 3.2 (staging verification) is for the coordinator.

## What was built

**Collection→Search — OR/AND expression editor (steps 1.1, 1.2)**
`app/web/Collection.tsx` now loads `GET /api/search/expressions` when the Search sub-section opens. The editor shows each expression group as a card with a type selector (OR / AND), a list of terms with per-term Remove buttons, and an add-term input. Groups can be added (OR or AND) and removed. An "Unsaved changes" status indicator appears whenever local state diverges from the last saved state. Clicking Save calls `PUT /api/search/expressions` and updates the saved baseline. Groups with no terms are silently filtered before saving.

**Collection→Search — Exclusion words (step 2.1)**
The former "Stop-words" section is now labelled "Exclusion words" with updated copy: "An exclusion word hides every role whose title or company name contains it". The underlying API (stop-words endpoints) is unchanged; only the presentation changed.

**Collection→Sources — Editable sites (step 2.2)**
The Sources sub-section now shows a site list loaded from `GET /api/search/sites`. Each site has an Enabled checkbox (calls `PATCH /api/search/sites/{name}`) and a Remove button (calls `DELETE /api/search/sites/{name}`). An add-site form with name and optional URL fields calls `POST /api/search/sites`. The existing coverage display (Sources.tsx component) is retained below the site management controls.

**Settings — Dismissed & blocked panel removed (step 2.3)**
The "Dismissed & blocked" nav button and panel are gone from Settings.tsx. The block-rule list, source-block control and dismissed-roles list are all removed. Settings now has three panels: Context, Configuration, Queue. The imports `listBlockRules`, `listSources`, `addSourceBlock`, `removeBlockRule` were removed from `api.ts` as well.

**Footer removed; queue moved to Settings→Queue (step 2.4)**
The always-on `<footer className="queue-bar">` is removed from main.tsx. The `.sift-app` grid was updated from three rows to two. Settings.tsx Queue panel now shows the Assess vacancies toggle (passed as props from App), queue counts (running / pending / failed / missing desc. / current), and stage model info (loaded from `GET /api/stage-providers` when the Queue panel opens). The assessment toggle props (`assessmentOn`, `savingAssessment`, `assessmentError`, `toggleAssessment`) are passed from App to Settings.

**Build**
`tsc --noEmit` exits 0; `vite build` produces 55 modules with no warnings. Output recorded in log.md under step 3.1.

## Next step

Step 3.2 is open. The coordinator should deploy staging and verify that:
- Search expressions can be added, modified and persist across a reload.
- Exclusion words can be added and removed and persist.
- Source sites can be added, toggled and removed and persist.
