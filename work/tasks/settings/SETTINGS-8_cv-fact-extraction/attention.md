# SETTINGS-8 — attention

Items that block handover until resolved. An open item is `- [ ]`; a resolved one is `- [x]`. While any item is open, `work-handover` refuses.

<!-- - [ ] Example: confirm the data directory path with the owner. -->

## Known limitations (for the owner; informational, none blocks the card)

- The number of skills found in one analysis of the same CV varies between runs. The owner's exact CV gave 23, 10 and 27 skills in three runs with the local model; the owner's threshold is 15, so one run in three was below it. The cause is the local model's sampling, not validation. Re-running the analysis gives a different count; nothing is lost permanently because skipped or missing skills can be added by hand or by re-running.
- A LinkedIn source stores the page that the fetch returned, which for a public view is mostly a sign-in page (cookie and privacy text, almost no profile data). Analysis of it is correct but finds no facts. Recommendation to the owner: export the LinkedIn profile as a PDF and upload that file as a profile document instead of using the LinkedIn address.
- Facts that equal the stored profile are recorded as "matches your profile" and are not shown in the interface; this is the owner's decision.
