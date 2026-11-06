# GO-10 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-05 — 1.1 — write LICENSE at repo root

Check: `grep -n 'Elastic License 2.0\|Limitations\|Acceptance\|Patents' LICENSE`
Exit: 0

```
1:Elastic License 2.0
7:## Acceptance
18:## Limitations
32:## Patents
```


## 2026-11-05 — 1.2 — add License section to README and AGENTS.md, set package.json license field

Check: `grep -n 'Elastic License 2.0|LicenseRef-Elastic-2.0' sifty/README.md AGENTS.md app/package.json`
Exit: 1

```

```


## 2026-11-05 — 1.2 — add License section to README and AGENTS.md, set package.json license field

Check: `grep -En 'Elastic License 2.0|LicenseRef-Elastic-2.0' sifty/README.md AGENTS.md app/package.json`
Exit: 0

```
sifty/README.md:35:Sifty is source-available under the [Elastic License 2.0](../LICENSE). You may download, modify and self-host it for your own use. You may not offer it to third parties as a hosted or managed service. See `LICENSE` for the full terms.
AGENTS.md:65:This project is source-available under the Elastic License 2.0. You may download, modify and self-host it for your own use. You may not offer it to third parties as a hosted or managed service. See `LICENSE` for the full terms.
app/package.json:5:  "license": "LicenseRef-Elastic-2.0",
```


## 2026-11-05 — 1.3 — update GO-6 description to note license delivered in GO-10

Check: `grep -n 'GO-10' ai/tasks/go-migration/GO-6_self-host-packaging/description.md`
Exit: 0

```
9:The project is source-available and self-hostable. The owner sells a hosted offering for non-technical users; technical users may download, self-host and modify it for their own use, but not offer it to others as a competing commercial service. For that to work the public repository must be easy to install, well documented, and approachable for a programmer who wants to change it. The multi-runtime stack makes a bare manual install painful: a Go backend, Node workers (`.mjs`), the Python jobspy scraping library, and optionally a local model runtime (ollama). This card establishes the self-host story and the documentation set. The Elastic License 2.0 was delivered in GO-10; this card now covers packaging and docs only.
```

