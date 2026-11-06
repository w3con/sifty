# ASM-2 — Set ollama context window on inference calls

**Epic:** assessment
**Created:** 2026-11-04
**Status:** founded
**Short card:** steps and acceptance criteria live here.

## Why this work exists

When the backend calls ollama it sends only the model, messages, stream flag and response format; it never sets `num_ctx`, the model's context window. Ollama then uses its own default window for the model, which is small, and silently truncates any prompt longer than it. The backend's own context budget (`context_chars`) is therefore an illusion above that default: raising it feeds text that ollama drops, and the model receives a cut-off prompt, which can produce malformed output. The fix is to tell ollama how large a window to use, matched to the configured budget, so the prompt the backend assembles is the prompt the model actually sees.

## Notes

- The ollama request is built in `internal/assess/infer.go`; the payload lacks an `options` block.
- Set `options.num_ctx` from the provider's `context_chars` converted to a token budget (a conservative chars-per-token factor), bounded so it does not exceed what the model and host can hold.
- This is a backend inference fix; it does not change which model is used or the assessment logic.

## Steps

- [x] 1.1 — Pass `options.num_ctx` on the ollama inference request, derived from the provider's configured context budget with a safe bound; check: a recorded ollama request carries `options.num_ctx` matching the configured budget, and a prompt within the budget is not truncated.

## Acceptance criteria

- The ollama inference call sets `num_ctx` from the configured context budget, so prompts within the budget reach the model intact rather than being silently truncated by ollama's default window.

## Must not touch

- The non-ollama (OpenAI-compatible) inference path and the assessment logic.
