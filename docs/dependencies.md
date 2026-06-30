# Dependency Injection

Gapi supports request-scoped dependency injection with explicit named dependencies.

```go
type CurrentUser struct {
	Email string
}

type In struct {
	User CurrentUser `dep:"currentUser"`
}

app.Provide(gapi.Dep("currentUser", func(ctx context.Context, r *http.Request) (CurrentUser, error) {
	return CurrentUser{Email: r.Header.Get("X-User-Email")}, nil
}))
```

Dependencies are resolved before binding and validation. Resolved values are cached for the request, so repeated fields using the same dependency name share one resolver call.

Auth helpers such as `gapi.BearerTokenDep`, `gapi.APIKeyHeaderDep`, and `gapi.APIKeyQueryDep` are available for common credential injection.
