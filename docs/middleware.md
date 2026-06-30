# Middleware

Gapi uses standard Go middleware:

```go
type Middleware func(http.Handler) http.Handler
```

Use app-level middleware:

```go
app.Use(middleware.Recover())
app.Use(middleware.RequestID())
```

Use group-level middleware:

```go
api := app.Group("/api/v1")
api.Use(middleware.SecureHeaders())
```

Use route-level middleware:

```go
gapi.Get[In, Out](app, "/admin", Handler, gapi.Use(AdminOnly()))
```

Built-in middleware lives in `github.com/gapi-org/gapi/middleware`.

Included helpers cover recovery, request IDs, timeouts, body limits, security headers, CORS, API key auth, and bearer token auth.
