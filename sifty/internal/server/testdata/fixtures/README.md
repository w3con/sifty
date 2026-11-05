# API Fixtures

One JSON file per endpoint. Files with `"source": "synthetic"` were generated from the
documented API contract shapes (backend-api-contract.md) and should be replaced with files
captured from the running TypeScript application using `sifty/scripts/capture-fixtures`.

## Format

```json
{
  "description": "human-readable summary",
  "method": "GET",
  "path": "/api/...",
  "requestBody": null,
  "responseStatus": 200,
  "responseBodyShape": "object|array|null",
  "requiredFields": ["field1", "field2"],
  "source": "synthetic|captured"
}
```

`responseBody` may be a complete response when `source` is `captured`.
`requiredFields` and `responseBodyShape` are used by the replay harness when only a synthetic
fixture is available: the harness checks the shape and required fields rather than doing an
exact diff.

## Capture

```sh
cd sifty
go run scripts/capture-fixtures/main.go --base-url http://127.0.0.1:4317 --out internal/server/testdata/fixtures
```
