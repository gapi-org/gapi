# Contributing To Gapi

Thanks for your interest in contributing to Gapi.

Gapi is a FastAPI-inspired Go framework. The project values small, focused changes that preserve the typed-handler developer experience and standard `net/http` compatibility.

## Before You Start

1. Read the [README](README.md) to understand the project goals.
2. Read [docs/project-structure.md](docs/project-structure.md) to understand the repository layout.
3. Check [docs/roadmap.md](docs/roadmap.md) for planned work.

## Development Setup

```bash
git clone https://github.com/gapi-org/gapi.git
cd gapi
go test ./...
```

The project currently has no third-party runtime dependencies.

## Running Checks

Before opening a pull request, run:

```bash
gofmt -w .
go test ./...
go vet ./...
```

If you change public behavior, add or update tests. Integration tests live in `tests/`.

## Pull Request Guidelines

- Keep changes focused and easy to review.
- Preserve the public import path: `github.com/gapi-org/gapi`.
- Prefer standard library APIs and idiomatic `net/http`.
- Avoid adding dependencies unless they are clearly necessary.
- Update docs when changing user-facing behavior.
- Include tests for bug fixes and new features.

## API Stability

Gapi is currently alpha. Public APIs may change before v1, but changes should still be intentional and documented.

## Reporting Bugs

When reporting a bug, include:

- Go version
- Gapi version or commit
- Minimal reproduction
- Expected behavior
- Actual behavior

## Feature Requests

Feature requests are welcome, especially when they improve:

- typed handler ergonomics
- OpenAPI accuracy
- validation and binding behavior
- testing workflows
- `net/http` compatibility
