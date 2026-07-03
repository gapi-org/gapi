# Gapi

Gapi is a FastAPI-inspired Go backend framework built around typed handlers, automatic OpenAPI generation, and idiomatic `net/http`.

The goal is simple:

> Write one typed Go handler, and let the framework handle request binding, validation, OpenAPI generation, docs UI, response serialization, dependency injection, and standard error responses.

Gapi is core-complete for an initial public alpha. APIs may still evolve before v1.

## Why Gapi

Most Go web frameworks are router-first. Gapi is type-first:

- Define input/output structs once.
- Bind path, query, header, cookie, and body data automatically.
- Validate requests from struct tags and custom validators.
- Generate OpenAPI 3.1 and docs automatically.
- Keep standard `net/http` compatibility.

This gives Go projects a FastAPI-like developer experience without leaving the Go ecosystem.

## Project Status

Gapi is ready for early users and feedback, but it is not v1-stable yet.

- Suitable for experiments, prototypes, internal tools, and early adopters.
- Public APIs may change before v1.
- Production use should pin versions and review changelog updates.

## Install

```bash
go get github.com/gapi-org/gapi
```

## Quickstart

```go
package main

import (
	"context"
	"net/http"

	"github.com/gapi-org/gapi"
	"github.com/gapi-org/gapi/middleware"
)

type GetUserIn struct {
	ID        int    `path:"id" validate:"min=1" doc:"User ID"`
	Verbose  bool   `query:"verbose" default:"false"`
	RequestID string `header:"X-Request-ID"`
}

type User struct {
	ID    int    `json:"id" doc:"User ID"`
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email" format:"email"`
}

func GetUser(ctx context.Context, in GetUserIn) (User, error) {
	return User{ID: in.ID, Name: "Ada Lovelace", Email: "ada@example.com"}, nil
}

func main() {
	app := gapi.New(gapi.Config{Title: "User API", Version: "0.1.0"})
	app.Use(middleware.Recover())
	app.Use(middleware.RequestID())

	gapi.Get[GetUserIn, User](
		app,
		"/users/{id}",
		GetUser,
		gapi.Summary("Get a user"),
		gapi.Tags("users"),
	)

	http.ListenAndServe(":8080", app)
}
```

Generated endpoints:

- `GET /users/{id}`
- `GET /openapi.json`
- `GET /docs`
- `GET /redoc`
- `GET /scalar`

## CLI

Install the CLI with:

```bash
go install github.com/gapi-org/gapi/cmd/gapi@latest
```

Useful commands:

```bash
gapi new hello-api
gapi routes --file openapi.json
gapi openapi --url http://localhost:8080/openapi.json --out openapi.json
gapi gen --file openapi.json --out client.go
gapi lint
```

## Current Features

- `net/http` compatible `App` implementing `http.Handler`.
- Typed handlers: `func(context.Context, In) (Out, error)`.
- Route helpers: `Get`, `Post`, `Put`, `Patch`, `Delete`, `Head`, `Options`.
- Route groups and app/group/route middleware.
- Binding tags: `path`, `query`, `header`, `cookie`, `body`.
- Validation tags: `required`, `min`, `max`, `len`, `email`, `uuid`, `oneof`, `regexp`, `enum`.
- JSON response serialization, `gapi.Response[T]`, text, HTML, redirects, files, attachments, streams, and SSE.
- RFC 9457-style Problem Details responses.
- OpenAPI 3.1 generation and hosted docs UIs.
- Dependency injection with `gapi.Dep`, `app.Provide`, `gapi.Require`, request caching, and `dep` fields.
- OpenAPI security scheme metadata plus bearer/API-key helpers.
- Optional middleware package.
- Testing helper package.
- CLI for scaffolding, route listing, OpenAPI export, and client scaffold generation.
- Hello and Todo examples.
- 

## Alpha Limitations

- Generated clients are currently scaffold-level, not full typed SDKs.
- chi/gin/echo router adapters are roadmap items.
- Full dependency override graphs and advanced OAuth/JWT helpers are roadmap items.
- Recursive JSON Schema references and custom schema providers are roadmap items.
- APIs may change before v1.

## Packages

- `github.com/gapi-org/gapi`: core framework API.
- `github.com/gapi-org/gapi/middleware`: optional middleware.
- `github.com/gapi-org/gapi/testing`: httptest helpers.

## Documentation

See [`docs/`](docs/) for quickstart, validation, middleware, OpenAPI, dependency injection, testing, project structure, and roadmap notes.

## Contributing

Contributions are welcome. Start with [`CONTRIBUTING.md`](CONTRIBUTING.md), read the project structure guide in [`docs/project-structure.md`](docs/project-structure.md), and run `go test ./...` before opening a pull request.

## Security

Please report vulnerabilities privately using the process in [`SECURITY.md`](SECURITY.md).

## License

Gapi is released under the [MIT License](LICENSE).

