# Releasing

Gapi uses semantic version tags. Public APIs may still change before v1, so releases before `v1.0.0` are alpha releases.

## Pre-Release Checklist

Run these from the repository root:

```bash
go test ./...
go test -race ./...
go vet ./...
go test ./examples/...
```

Also verify:

- `README.md` install commands point at the intended version.
- `CHANGELOG.md` has an entry for the new version.
- Examples build and use the public `github.com/gapi-org/gapi` import path.
- Generated OpenAPI still serves at `/openapi.json`.
- Docs UIs still serve at `/docs`, `/redoc`, and `/scalar`.

## Create A Release Tag

Only create tags after the checklist passes:

```bash
git tag v0.1.1
git push origin v0.1.1
```

Users can then install the library or CLI with:

```bash
go get github.com/gapi-org/gapi@v0.1.1
go install github.com/gapi-org/gapi/cmd/gapi@v0.1.1
```

## Release Notes

Release notes should include:

- New framework features.
- Breaking changes, even before v1.
- Migration steps.
- Known alpha limitations.
- Security or validation changes.

## Module Proxy

After pushing a tag, the Go module proxy may take a short time to discover it. You can verify availability with:

```bash
go list -m -versions github.com/gapi-org/gapi
```
