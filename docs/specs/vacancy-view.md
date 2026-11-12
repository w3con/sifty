# Vacancy view behavior specification

Status: **agreed with the owner, not implemented** (2026-11-12). Source: the owner's requirements of 2026-11-12 (section 1). The owner says the layout drawn for him earlier is not mockup "B" of 2026-11-09; that drawing was not found in the saved files, and the exact visual layout is to be confirmed with the owner before L.5b. Card: INBOX-2 (step L.5 rework). Related: [Quick Match](quick-match.md) §15 (sections by code), [Profiles](profiles.md).

## 1. Purpose

The owner scans many vacancies quickly. The eye looks for two things: **what they want from the candidate** and **what they offer**. The section view and the Key facts card exist for this quick look. The full text stays available, but it is not the point of the section view.

Owner, 2026-11-12: «Не было идеи вытащить все секции из вакансии. Было важно 2 секции: что хотят от кандидата, и что предлагают. Это то, на что ложится глаз когда пользователь быстро просмотривает вакансию. Разбивка по секциям - это как для того, чтобы быстро посмотреть. Для этой же цели делался блок Key Facts.»

## 2. Listing text: Original and By section

- **Original** shows the whole listing in its own order with its headings, lists, numbering, bold and italics, styled only by Sifty's CSS (sanitised structured form).
- **By section** shows a fixed set of groups, not one block per heading:
  1. **What they ask** — the listing's Requirements and Nice to have sections merged, in their original order; preferred items keep their marker.
  2. **What they offer** — Compensation, Benefits, Perks, Conditions (location, work mode, hours, contract) merged.
  3. **What you'll do** — Role / Responsibilities merged. *(Open: the owner named two groups; the mockup had this third one. Kept third until the owner decides.)*
  4. **Other text** — everything else, in original order, collapsed by default, labelled "not classified · shown in full".
- Each group carries a small state label: "recognised" (found by headings) or "not found in listing" with "Not stated". Two source headings of the same class never produce two groups.
- By section is available when at least one of the first two groups is recognised; otherwise only Original.
- Sections and their classes come from code (Quick Match §15.2 step 2); no model is needed to show this view.

## 3. Key facts

The Key facts card lists: Compensation, Work mode, Location, Employment type, Seniority, Posted, Expires, Source; each value with its origin in small text ("from listing data", "from the Compensation section", "read by the model", "applicant location requirement"); a missing value is "Not stated".

Values come from, in this order: structured source fields; code reading the classified sections (for example a salary range in What they offer: "$150,000 and $250,000" → Compensation USD 150k–250k / year); the model's extracted conditions (Quick Match §14.5, quotes validated). Key facts and Fits you use the same merged facts, so a salary stated in the text is never "Not stated" in Key facts while the text shows it.

## 4. Out of scope

Editing the listing; translating it; rewriting it with a model.
