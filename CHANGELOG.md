# Changelog

## Unreleased

## v0.1.0 - 2026-07-03

### Added

- Typed route helpers for `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, and `OPTIONS`.
- Request binding from path, query, header, cookie, and JSON body.
- Struct-tag validation for `required`, `min`, `max`, `len`, `email`, `uuid`, `oneof`, `regexp`, and `enum`.
- RFC 9457-style Problem Details responses with field-level validation errors.
- OpenAPI 3.1 generation and docs UIs at `/docs`, `/redoc`, and `/scalar`.
- Route groups and standard `net/http` middleware support.
- Optional middleware package with recovery, request IDs, timeout, body limits, secure headers, CORS, API key auth, and bearer auth.
- Dependency injection with request caching, auth dependency helpers, and OpenAPI security metadata.
- Response helpers for JSON, text, HTML, no-content, redirects, files, attachments, streams, and SSE.
- Testing helper package, fuzz coverage, benchmark coverage, Hello example, Todo example, and GitHub Actions CI.
- `gapi` CLI for scaffolding, route listing, OpenAPI export, linting, and client scaffold generation.

### Alpha Limitations

- CLI commands are intentionally small and do not yet provide live reload.
- Client generation is scaffold-level and not a full typed SDK.
- Router adapters for chi, gin, and echo are roadmap items.
- Full dependency override graphs and advanced OAuth/JWT helpers are roadmap items.
- Full JSON Schema coverage for recursive references, custom schema providers, and complex generics is not complete.
