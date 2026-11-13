---
estimate: 6
---
# SET-2 — plan

**Author:** coordinator
**Created:** 2026-11-03

## Statement of work

The server reads all configuration once at startup into `deps.Cfg` and never refreshes it, so saving a config file through the Configuration tab changes the file on disk but leaves the provider panel, the smoke check and the inference workers running on the config captured at startup; to the owner this reads as "the save stored the previous version". This card makes a saved config take effect on the running server with no restart: after a successful `config.Save`, the server reloads the configuration from disk and publishes the new `*config.Config` behind synchronization, because `deps.Cfg` is read concurrently by the assessment and preparation workers and a bare field assignment would be a data race. The port stays the one documented exception: a changed port keeps serving on the old listener and the save reports that this single change needs a restart, rather than failing. It does not change config validation, the backup behaviour, the file formats, or the request shapes.

## Steps

### Phase 1 — Publish config changes without a race

- [x] 1.1 — Make the in-memory config accessible through a synchronized accessor (mutex or atomic pointer) that both the HTTP handlers and the workers read, replacing direct `deps.Cfg` field reads on the hot paths; check: a race-detector test (`go test -race`) reads the config from several goroutines while it is swapped, with no race reported.
- [x] 1.2 — Add a reload that reads the config from disk and publishes it through that accessor; check: a test mutates a config file, calls reload, and the accessor returns the new value.

### Phase 2 — Reload on save

- [x] 2.1 — After a successful `config.Save`, trigger the reload so the next `/api/providers`, smoke check and worker inference use the saved config; check: a test saves a changed provider model via the save path and the provider diagnostics then report the new model without a restart.
- [x] 2.2 — Treat a changed port as the documented exception: keep the current listener, and report through the save response that the port change needs a restart, without failing the save; check: a test saves a runtime config with a different port and gets a success response carrying the restart-required notice, with the server still listening on the original port.

### Phase 3 — Live verification

- [x] 3.1 — Against the running demo on the owner's real data dir, edit the ollama provider model to a bare installed ID, save, and confirm the provider panel and a smoke check reflect it with no restart; check: a recorded run shows the provider status changing after save alone.

## Acceptance criteria

- A config saved through the Configuration tab takes effect on the running server with no restart: the provider panel, smoke checks and inference workers use the new config.
- The in-memory config is read and swapped under synchronization, with a `-race` test covering concurrent reads during a swap.
- A changed port is reported as needing a restart and does not fail the save; the server keeps serving on the original port.
- Config validation, backup behaviour, file formats and request shapes are unchanged.
