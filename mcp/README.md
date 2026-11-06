# Sifty MCP adapter (not implemented)

This directory is the tracked placeholder for the future standalone TypeScript
HTTP adapter that will expose the Go backend over the Model Context Protocol.

No implementation is authorized by the 2026-11 structural relocation. The
existing plan and decisions are in
`work/tasks/go-migration/GO-9_mcp-server/`; implementation remains that card's
scope. The intended shape is a TypeScript process that talks to the Go HTTP API
first, adding any missing backend operation to that API rather than duplicating
product logic here.
