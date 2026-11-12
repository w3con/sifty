# Vacancy view behavior specification

Status: **agreed with the owner, not implemented** (2026-11-12; design approved). Source: the owner's requirements of 2026-11-12 (section 1). The owner says the layout drawn for him earlier is not mockup "B" of 2026-11-09; that drawing was not found in the saved files, and the exact visual layout is to be confirmed with the owner before L.5b. Card: INBOX-2 (step L.5 rework). Related: [Quick Match](quick-match.md) §15 (sections by code), [Profiles](profiles.md).

## 1. Purpose

The owner scans many vacancies quickly. The eye looks for two things: **what they want from the candidate** and **what they offer**. The section view and the Key facts card exist for this quick look. The full text stays available, but it is not the point of the section view.

Owner, 2026-11-12: «Не было идеи вытащить все секции из вакансии. Было важно 2 секции: что хотят от кандидата, и что предлагают. Это то, на что ложится глаз когда пользователь быстро просмотривает вакансию. Разбивка по секциям - это как для того, чтобы быстро посмотреть. Для этой же цели делался блок Key Facts.»

## 2. Layout (agreed design, 2026-11-12)

The approved design is the canvas "Sifty — vacancy view" (https://claude.ai/artifact/NV9n28fNGjTVruEqPsoRkp, artboard "Vacancy — quick look"); owner: «Мне нравится этот дизайн.» Built with the Sifty design system. Read it before building; this section states its rules.

Top to bottom in the work area:

1. **Header**: company label, title, meta line (published, age, source, original listing), actions Block company / Dismiss / Select.
2. **Fits you band** (one card): the Fits you number with its bucket pill on the left; on the right one **tile per goal of the active profile** (owner: seniority only where a profile has that goal — «сеньорность надо убрать - это не для всех профилей нужно»). Today: Money, Location, Work mode, Contract. A tile = icon, the vacancy's value large, one short line comparing it with the goal; the tile's fill is the verdict: matches (`ok-bg`, ✓), does not match (`warn-bg`, ✕), not stated (neutral, ?). This band **replaces** the separate Key facts and Quick match cards.
3. **Two cards side by side**:
   - **What they ask**: header with counts ✓ / ✕ / ?; groups Required and Nice to have; each requirement a row with its mark: ✓ in your profile (code: skill vocabulary, years from dates, your ✓), ✕ in "Skills you do not have", ? not in your profile with ✓ / ✕ buttons (a skill goes to Skills / Skills you do not have; any other requirement to "Confirmed by you" as "You have: …" / "You do not have: …"; the same requirement text elsewhere is then marked without asking); evidence in small text on the right ("17 years", "Go · Rust").
   - **What they offer**: tiles with icons for what the listing offers: salary range, equity, benefits, time off, and further items from a small vocabulary; a stated value or "not stated" under each.
4. **Full listing**: open, no Show button, the whole text as written with its headings, lists and emphasis (the sanitised structured form). There is no By section / Original switch any more: the two cards above are the section view, the full listing is the original.

When the listing has no recognised Requirements or Offer section, the corresponding card says so in one line and the full listing stays below.

## 3. Where the values come from

Tiles, What they offer and Fits you read the same merged facts, in this order: structured source fields; code reading the classified sections (salary ranges, work mode, location, contract words, the offer vocabulary); the model's extracted conditions (quotes validated). A value stated in the text is never "not stated" in a tile.
Requirements: the model's extraction (Quick Match §14.5) with verdicts by code and by the owner (§15.1); the model's matching pass stays off.

**Money (owner, 2026-11-12, «Пересчёт по курсу из настроек»):** a vacancy's range is converted to the profile's currency with exchange rates kept in configuration (edited by hand, no network); the tile shows the original range and «≈ €…» and a ✓ / ✕ against the minimum; a range without a tax note counts as gross; a period other than a year is converted (month × 12, hour × 2 080), else not compared. Money is read from the Compensation/Benefits sections and the model's conditions only, not from the whole text.
**Location:** shown without a verdict colour until the profile holds where the owner may work (SETTINGS-8) and regions can be compared (ASSESSMENT-8).
**What they ask** lists the model's extracted requirements whether or not a Requirements section was recognised; the card says «not found» only when there is no extraction or it is empty.

Fill rates are measured before building (INBOX-2 L.5r0); the design must hold when most tiles say "not stated".

## 4. Out of scope

Editing the listing; translating it; rewriting it with a model.
