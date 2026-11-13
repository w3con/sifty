# Move Sift to another Mac

The owner chooses the new directory and installed model. The coordinator stops the old writer and creates the final private archive. The restore command verifies and installs that snapshot without starting Sift.

## Source export (coordinator)

Leave Assessment explicitly Off. Wait for collection, preparation and inference to finish. Verify and stop the old catalogue listener before exporting; do not run two writers after switching. From the source `app` directory run `node ops/transfer/export.mjs --frozen`. This writes a versioned archive, `.sha256` and readable `.manifest.json` under ignored `.local/transfers/`. Transfer the archive and checksum privately. They contain candidate documents, saved answers and full Git history. Browser sessions, CLI credentials, local inference history, schedules, logs and old backups are excluded. Configured local inputs and referenced technical records must exist or export fails.

Do not restart the old source after switching to the target. If abandoning the migration, leave the target stopped and run `npm start` from the old source's `app` directory. No automatic shutdown, startup or scheduler changes are made by these tools.

## Target prerequisites and restore

Install Node >=22.13, Git and PDF text extraction: `brew install node poppler`. If Git is missing, install Apple's command-line tools with `xcode-select --install`. Ollama must already be installed; no model is installed by this workflow.

In the directory containing the archive and checksum, run `shasum -a 256 -c <archive>.sha256`. Unpack the trusted archive into a new, empty staging directory with `tar -xzf <archive> -C <staging-directory>`. Do not unpack directly over an existing project. From that staging directory:

```sh
node restore.mjs --verify-only
mkdir -p "$HOME/Dev"
node restore.mjs --target "$HOME/Dev/fcto" --no-install
```

The target must not exist, even if empty. The default target is `$HOME/Dev/fcto`. `./restore.command` accepts the same flags. Restore checks the entire payload, file modes, safe relative links, database integrity/foreign keys/content hashes and explicit Assessment Off. It creates offline Git checkouts at the original revisions, preserves GitHub origins and the pinned shared-tooling checkout, then applies the complete working-tree snapshot, including uncommitted migration tools. The project returns to its recorded attached branch (for example, `main`), with its named GitHub upstream and archived remote-tracking tip when configured, so normal `git pull` works after re-establishing GitHub access. A source branch without an upstream stays without one; a genuinely detached source stays detached. The shared-tooling submodule stays detached at its pin. Branch/upstream metadata is validated and a branch/tracking change during export refuses capture. No network Git clone is needed. A failed restore removes only its own temporary installation. `--verify-only` creates no target; `--install` optionally runs `npm ci` and `npm run build` during restore, but does not install Chromium or start anything.

Then from the restored `app` directory:

```sh
npm ci
npm run build
npx playwright install chromium
npx tsx ops/transfer/ollama.ts --list
npx tsx ops/transfer/ollama.ts --model 'EXACT_ID_FROM_THE_LIST'
npm start
```

Start Ollama separately if discovery says it is unavailable. The helper defaults to `http://127.0.0.1:11434`; `--endpoint http://127.0.0.1:PORT` permits an explicit loopback-only endpoint. Stop Sift and all inference first. The helper requires a restore receipt, a free configured catalogue port, idle databases and Assessment Off. It validates an installed exact model ID and a bounded structured-output smoke using synthetic text before atomically updating only assessment/extraction routing and the Ollama model/endpoint. The smoke uses the existing Ollama provider timeout, capped at 120 seconds (currently the full configured 120 seconds), to allow the first cold model load. It does not rewrite the normal provider timeout or context budget. Previous provider configuration is backed up beside itself. Existing timeout/context limits and provider alternatives are preserved; no paid fallback is introduced. A smoke verifies structured connectivity, not quality or model context capacity. Keep Assessment Off until you are ready to assess jobs with the selected model.

Open `http://localhost:4317`. Browser and provider CLI logins must be established anew. The default strong application-drafting provider remains `openai/gpt-6.1-sol` and needs its separate OpenCode CLI installation/login, or explicitly choose Ollama under Application → More actions. Assessment/extraction selection does not change that drafting default. No source provider configuration, OAuth, model download or schedule is transferred/changed by the helper.

## Coordinator verification

Before handing off, unpack the real archive under a new `.local/tests/` directory and use its standalone restore with `--target` another absent directory there and `--no-install`. This validates actual source snapshots and complete table hashes without opening Store, starting another server or invoking any provider. Save archive/checksum, manifest counts, exact commands and results in the task card. Target macOS execution remains a separate owner check.
