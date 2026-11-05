# GO-10 — Add Elastic License 2.0 (source-available)

**Epic:** go-migration
**Created:** 2026-11-05
**Status:** founded (plan.md written; awaiting "Execute the plan")

## Why this work exists

The project is meant to be source-available under the Elastic License 2.0: anyone may download, modify and self-host it for their own use, but not offer it to third parties as a hosted or managed service. That is written in the roadmap as a decision, but there is no `LICENSE` file in the repository, so the terms are not actually attached to the code. This card attaches them: it adds the Elastic License 2.0 as the real licence, references it from the README and the cross-tool `AGENTS.md`, and sets the frontend package's licence field. It is the licence slice of GO-6, split out and done now; the packaging and documentation parts of GO-6 stay deferred.

## Must not touch

The application's behaviour (this card adds a licence and notices, no code). The packaging, documentation and developer-setup scope of GO-6. The coordinator-gated public-repo push (GO-1). Any pre-existing third-party licence notices.

## Notes

The licence decision is already made (Elastic License 2.0, source-available, not "open source"); this card only makes it real in the tree. The licence text must be the canonical Elastic License 2.0, written verbatim rather than paraphrased. The copyright holder on the line is the owner (to be confirmed by the coordinator before merge).
