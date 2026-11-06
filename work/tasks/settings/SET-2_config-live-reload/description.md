# SET-2 — Config live reload: a saved config takes effect without a server restart

**Epic:** settings
**Created:** 2026-11-03
**Status:** founded

## Why this work exists

The server loads all configuration once at startup (`config.Load` in `cmd/sifty/serve.go`) into `deps.Cfg`, and never reads it again. Saving a config file through the Configuration tab (`PUT /api/config/{name}` → `config.Save`) writes the file on disk but does not refresh the in-memory copy. As a result everything that drives behaviour — the provider-availability panel, the smoke check, and the assessment and preparation inference workers — keeps using the config captured at startup. Only the editor textarea reflects a save, because `GET /api/config/{name}` re-reads the file.

To the owner this looks like "saving stores the previous version": the file is correct, but the running system ignores it. The Configuration tab's own text even promises that only a port change needs a restart, which is currently false for every setting. This card makes a saved config take effect on the running server immediately, so editing a provider's model (for example fixing the ollama model ID) works without restarting.

## Must not touch

- The validation and backup behaviour of `config.Save` (it already validates and writes a timestamped backup). The config file formats and the `GET`/`PUT` request shapes. The port, which genuinely cannot change without a restart — reloading must not attempt to rebind the listener.

## Notes

- `deps.Cfg` is a `*config.Config` read concurrently by the running assessment and preparation workers, so the reload must publish the new config without a data race — swap it behind synchronization (a mutex or an atomic pointer), not a bare field assignment.
- The natural trigger is a successful `config.Save`: reload from disk and publish the new `*config.Config`, so the next `/api/providers` call, smoke check and worker inference see it.
- Port is the one documented exception. If a reload changes the port, keep serving on the old port and surface that a restart is required for that specific change, rather than failing the save.
- Verify live against the running demo: edit the ollama provider's model to a bare installed ID, save, and confirm the provider panel and a smoke check reflect the new model with no restart.
