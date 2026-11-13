# Vacancy view — specification

Status: draft by OPS-11, 2026-11-13; not yet approved by the owner.

## 1. Purpose

The owner scans many vacancies quickly. The view shows, for one selected vacancy, what it demands from the candidate and what it offers, together with how well the vacancy fits the active profile. Owner, 2026-11-12: «Не было идеи вытащить все секции из вакансии. Было важно 2 секции: что хотят от кандидата, и что предлагают. Это то, на что ложится глаз когда пользователь быстро просматривает вакансию. Разбивка по секциям - это как для того, чтобы быстро посмотреть. Для этой же цели делался блок Key Facts.» The Fits you band and the two demand/offer cards serve the quick look; the full listing is available for deeper reading.

## 2. Scope

This spec covers what the owner sees when one vacancy is open: the header, the Fits you band, the What they ask card, the What they offer card, and the full listing. It does not cover: how vacancies are collected and their text stored ([collection.md](collection.md)); how the Fits you number is computed and which goals it weighs ([quick-match.md](quick-match.md), §§1–15); how profile goals are defined and edited ([profiles.md](profiles.md)). Exchange rates and vocabulary files are configured in §4 of this spec. Out of scope for this feature: editing the listing text, translating it, or rewriting it with a model.

## 3. Rules

### Header

1. The header shows the company label, the vacancy title, a meta line (publication date, age, source, and a link to the original listing), and three actions: Block company, Dismiss, Select. [implemented]

### Layout

2. The view shows, top to bottom: the header, the Fits you band, the What they ask card and the What they offer card side by side, and the full listing open below. The approved design is the canvas "Sifty — vacancy view" (https://claude.ai/artifact/NV9n28fNGjTVruEqPsoRkp, artboard "Vacancy — quick look"); read it before building. Owner, 2026-11-12: «Мне нравится этот дизайн.» The design is built with the Sifty design system tokens. [not implemented — INBOX-2 (on branch)]

3. The full listing is always open; there is no Show button, no By section toggle, and no Original toggle. [not implemented — INBOX-2 (on branch)]

4. The full listing renders the stored structured form of the vacancy text — headings h2–h4, bulleted and numbered lists, paragraphs, bold and italic — in the page typography; no raw HTML tags or character entities are visible in the rendered output. When the structured form is absent the view falls back to the existing plain-text line renderer. [not implemented — INBOX-2 (on branch)]

### Fits you band

5. The Fits you band is a card spanning the full width of the work area; it replaces the former Key facts strip and the Quick Match cards. [not implemented — INBOX-2 (on branch)]

6. The left side of the band shows the Fits you number and its bucket pill, together with the Refresh and Assess actions. [not implemented — INBOX-2 (on branch)]

7. The right side of the band shows one tile per active profile goal; today the goals are Money, Location, Work mode, and Contract. Seniority is not a goal of the band. Owner, 2026-11-12: «сеньорность надо убрать - это не для всех профилей нужно» [not implemented — INBOX-2 (on branch)]

8. Each tile shows an icon, the vacancy's value large, one short comparison line, and a background fill that is the verdict: matches (ok-bg, ✓), does not match (warn-bg, ✕), or not stated or not compared (neutral, ?). [not implemented — INBOX-2 (on branch)]

9. The Money tile shows the original salary range, an approximate euro equivalent formatted as «≈ €…», and a verdict (✓, ?, or ✕) against the profile's minimum. Owner, 2026-11-12: «Пересчёт по курсу из настроек» The range is read only from the Compensation and Benefits sections of the vacancy and from the model's extracted compensation condition; it is not read from the whole listing text. When no value is found in any layer, the tile shows "not stated". [not implemented — INBOX-2 (on branch)]

10. The Money verdict has three states: ✓ when the lower bound of the converted range meets or exceeds the profile minimum; ? when the minimum falls inside the range; ✕ when the upper bound falls below the minimum. The Fits you number uses the same three-state rule and the same conversion, so the tile and the number never disagree on the money verdict. [not implemented — INBOX-2 (on branch)]

11. The Location tile shows the geography verdict of Fits you (covered ✓ / not covered ✕ / unknown ?), the same verdict the Fits you number uses for the owner's work-authorisation and location restrictions. [not implemented — no card]

12. The Work mode tile shows the vacancy's work mode; a value derived from the unclassified text only is shown as a weak hint and does not receive a verdict colour. [not implemented — INBOX-2 (on branch)]

13. The Contract tile shows the contract type detected by explicit words in the classified sections only; bare employment-type words (full-time, part-time) are not a contract value. [not implemented — INBOX-2 (on branch)]

### What they ask

14. The What they ask card shows the model's extracted requirements whether or not the listing has a recognised Requirements section; it shows "No requirements extracted yet" in one line only when the extraction is absent or empty. [not implemented — INBOX-2 (on branch)]

15. The card header shows counts of requirements marked ✓, ✕, and ?; requirements are grouped into Required and Nice to have. [not implemented — INBOX-2 (on branch)]

16. Each requirement row shows the requirement text, a verdict mark, and evidence in small text to the right (a profile value such as "17 years", or a model quote). [not implemented — INBOX-2 (on branch)]

17. A row already decided shows only its mark; a row not yet decided shows ✓ and ✕ buttons; a button that cannot act is not shown. A row decided ✓ means the requirement is confirmed in the owner's profile by code (skill vocabulary, years from dates) or by the owner's own action; ✕ means the requirement appears in "Skills you do not have" or in a "you do not have" confirmed fact. [not implemented — INBOX-2 (on branch)]

18. When the owner presses ✓ on an undecided row, the routing depends on whether the requirement text looks like a skill name: a name that appears in the skill vocabulary, or a word sequence of at most three words where no word ends in -ing, is a skill name and is added to Skills (a comma-separated list of skill names is split into separate entries); anything else goes to the Confirmed by you list as "You have: <requirement text>". Owner, 2026-11-13 (correction of earlier behaviour): «Coding, debugging, reviewing» → Confirmed by you; «TypeScript, Node» → two skills added; «Go or Rust» → the two alternatives are offered to choose from. [not implemented — INBOX-2 (on branch)]

19. When the owner presses ✕ on an undecided row: for a skill name the skill is added to Skills you do not have; for anything else "You do not have: <requirement text>" is recorded in Confirmed by you. [not implemented — INBOX-2 (on branch)]

20. The Confirmed by you list is in the About me section of the profile page, next to Skills you do not have; each entry shows its text and a remove button. (profiles.md §5; INBOX-2 L.5r6) [not implemented — INBOX-2 (on branch)]

21. The same requirement text, normalised (case-insensitive, whitespace collapsed, trailing punctuation stripped), in a different vacancy is marked ✓ or ✕ from the recorded confirmed fact without asking the owner again. [not implemented — INBOX-2 (on branch)]

22. After any ✓ or ✕ action, the open vacancy's rows update at once without reloading the catalogue; the profile revision increments and Fits you is recomputed for that vacancy; because Fits you weighs goals, not single requirements, the number usually does not change. [not implemented — INBOX-2 (on branch)]

### What they offer

23. The What they offer card shows tiles with icons for each item the listing offers: the salary range, equity, benefits, time off, and further items from a vocabulary in configuration; each tile shows a stated value or "not stated". The card reads from the Compensation and Benefits sections only; when neither section is recognised the card says so in one line. [not implemented — INBOX-2 (on branch)]

### Values and their origin

24. The Fits you band tiles and the What they offer card read the same merged vacancy facts, resolved in this order: structured source fields; code reading the classified sections; the model's extracted conditions (validated against the source text). A value stated in the listing is never "not stated" in a tile. [not implemented — INBOX-2 (on branch)]

25. Money conversion: the vacancy's salary range is converted to the profile's currency using exchange rates from configuration (hand-edited, no network); a range without a tax note counts as gross; month × 12, hour × 2 080; a currency without a configured rate or an unrecognised period is shown as the original value but marked ? and not compared against the goal. [not implemented — INBOX-2 (on branch)]

26. Requirements in What they ask: the model's extraction (quick-match.md §14.5) with verdicts by code and by the owner (§15.1); the model's matching pass stays off, so only code verdicts and owner-confirmed facts decide rows. [not implemented — INBOX-2 (on branch)]

27. The view must remain usable when most tiles say "not stated"; fill rates were measured at INBOX-2 L.5r0 and confirmed that money is absent for about 66 % of vacancies and contract for about 78 %. [not implemented — INBOX-2 (on branch)]

## 4. Configuration

- **Exchange rates** (`config/vacancy-view.yaml`, key `exchange_rates`): a map from currency code to EUR, hand-edited, no network; the installed defaults are placeholder values and must be confirmed by the owner before the first release that shows money verdicts.
- **Offer vocabulary** (`config/vacancy-view.yaml`, key `offer_items`): the items the What they offer card looks for (equity, benefits, time off, etc.), each with a display label and a not-stated label.
- **Contract words** (`config/vacancy-view.yaml`, key `contract_kinds`): the explicit phrases that identify a contract type (e.g., "permanent role", "fixed-term contract", "1099"); each entry is matched with word-boundary rules to prevent false matches on words like "government contractor".
- **Skill vocabulary** (`backend/internal/config/skills.default.yaml`, embedded in binary; user override at `config/skills.yaml`): the list of known skill names; two additional values govern the ✓ routing: `max_skill_name_words` (default 3) and `verb_endings` (default ["ing"]); `modifier_words` lists -ing words that act as adjectives within a phrase and do not mark the whole phrase as an activity.

## 5. Questions for the owner

1. The INBOX-2 branch adds a ? state (minimum inside the range) and a day rate × 260 to the agreed ✓/✕ money rule — accept both?
2. Both prerequisites are released; the Location tile shows the geography verdict — confirm.

## 6. Open cards

- INBOX-2: implements rules 2–10, 12–27 (new layout, HTML full listing, Fits you band, What they ask and What they offer cards, Confirmed by you, unified money function); on branch `work/INBOX-2`, not in main.
- ASSESSMENT-4: adds a Dismiss action per requirement row in What they ask; the row design must accommodate three actions (✓, ✕, Dismiss) for a ? row alongside the existing two.
- ASSESSMENT-10: more restriction label forms resolved; the Location tile geography verdict reads the same comparison (in progress).
- SETTINGS-5: defines the full set of active profile goals; determines which tiles the band shows (rule 7).
