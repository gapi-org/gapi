# Gapi

Gapi is an alpha Go backend framework inspired by FastAPI's developer experience and built around idiomatic `net/http`.

The goal is simple:

> Write one typed Go handler, and let the framework handle request binding, validation, OpenAPI generation, docs UI, response serialization, dependency injection, and standard error responses.

Gapi is early alpha software. The core workflow is usable, but APIs may still change before v1.

## Install

```bash
go get github.com/Kushagra1122/gapi
```

## Quickstart

```go
package main

import (
	"context"
	"net/http"

	"github.com/Kushagra1122/gapi"
	"github.com/Kushagra1122/gapi/middleware"
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

## Current Features

- `net/http` compatible `App` implementing `http.Handler`.
- Typed handlers: `func(context.Context, In) (Out, error)`.
- Route helpers: `Get`, `Post`, `Put`, `Patch`, `Delete`, `Head`, `Options`.
- Route groups and app/group/route middleware.
- Binding tags: `path`, `query`, `header`, `cookie`, `body`.
- Validation tags: `required`, `min`, `max`, `len`, `email`, `uuid`, `oneof`, `regexp`, `enum`.
- JSON response serialization and `gapi.Response[T]`.
- RFC 9457-style Problem Details responses.
- OpenAPI 3.1 generation and hosted docs UIs.
- Basic dependency injection with `gapi.Dep`, `app.Provide`, `gapi.Require`, and `dep` fields.
- Basic OpenAPI security scheme metadata.
- Optional middleware package.
- Testing helper package.
- Hello and Todo examples.

## Alpha Limitations

- Code generation is planned after alpha.
- chi/gin/echo router adapters are planned after alpha.
- CLI is intentionally basic.
- Full DI graphs, dependency overrides, and advanced auth middleware are not complete.
- Full JSON Schema edge cases are not complete.
- APIs may change before v1.

## Packages

- `github.com/Kushagra1122/gapi`: core framework API.
- `github.com/Kushagra1122/gapi/middleware`: optional middleware.
- `github.com/Kushagra1122/gapi/testing`: httptest helpers.

## Documentation

See [`docs/`](docs/) for quickstart, validation, middleware, OpenAPI, dependency injection, testing, and roadmap notes.

