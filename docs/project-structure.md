# Project Structure

Gapi uses a facade layout: the repository root exposes the public `github.com/gapi-org/gapi` API, while framework implementation lives under `internal/core`.

```text
.
├── gapi.go                # Small public facade and stable import surface
├── middleware/            # Optional public middleware package
├── testing/               # Optional public testing helpers
├── tests/                 # Integration, fuzz, and benchmark tests
├── internal/              # Private implementation packages
│   ├── binding/           # Request binding internals
│   ├── core/              # App, router, responses, DI, security, errors
│   ├── docsui/            # Hosted docs UI HTML
│   ├── openapi/           # OpenAPI schema/spec generation
│   └── validate/          # Validation internals
├── cmd/gapi/              # CLI entrypoint
├── examples/              # Runnable example apps
├── docs/                  # User documentation
└── .github/workflows/     # CI
```

## Why The Root Is Small

The root package is the public import path:

```go
import "github.com/gapi-org/gapi"
```

Only `gapi.go` stays there. It re-exports the stable public API from `internal/core`, so users keep a clean import while contributors see a cleaner repository layout.

## Package Boundaries

- `gapi`: public facade used by applications.
- `middleware`: optional `net/http` middleware.
- `testing`: optional test helpers built on `httptest`.
- `internal/core`: private framework runtime implementation.
- `internal/*`: other private implementation details that users cannot import.
- `tests`: black-box integration tests for the public package.
- `cmd/gapi`: CLI commands for scaffolding, OpenAPI export, route listing, and client scaffold generation.

This keeps the repository open-source friendly while preserving a simple install and import story.
