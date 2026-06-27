package core

import (
	"context"
	"net/http"
	"strings"
)

// SecurityScheme describes an OpenAPI security scheme.
type SecurityScheme struct {
	Name   string
	Type   string
	Scheme string
	In     string
}

// BearerAuth defines an HTTP bearer security scheme.
func BearerAuth(name string) SecurityScheme {
	return SecurityScheme{Name: name, Type: "http", Scheme: "bearer"}
}

// BasicAuth defines an HTTP basic security scheme.
func BasicAuth(name string) SecurityScheme {
	return SecurityScheme{Name: name, Type: "http", Scheme: "basic"}
}

// APIKeyHeader defines an API key security scheme read from a header.
func APIKeyHeader(name, header string) SecurityScheme {
	return SecurityScheme{Name: name, Type: "apiKey", In: "header", Scheme: header}
}

// APIKeyQuery defines an API key security scheme read from a query parameter.
func APIKeyQuery(name, query string) SecurityScheme {
	return SecurityScheme{Name: name, Type: "apiKey", In: "query", Scheme: query}
}

// APIKeyCookie defines an API key security scheme read from a cookie.
func APIKeyCookie(name, cookie string) SecurityScheme {
	return SecurityScheme{Name: name, Type: "apiKey", In: "cookie", Scheme: cookie}
}

// RegisterSecurity adds OpenAPI security schemes to the app.
func (app *App) RegisterSecurity(schemes ...SecurityScheme) {
	app.securitySchemes = append(app.securitySchemes, schemes...)
}

// BearerTokenDep injects the Authorization bearer token into a dep-tagged field.
func BearerTokenDep(name string) Dependency[string] {
	return Dep(name, func(ctx context.Context, r *http.Request) (string, error) {
		value := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(value, "Bearer ")
		if !ok || token == "" {
			return "", NewHTTPError(http.StatusUnauthorized, "Missing bearer token.")
		}
		return token, nil
	})
}

// APIKeyHeaderDep injects an API key from a header into a dep-tagged field.
func APIKeyHeaderDep(name, header string) Dependency[string] {
	return Dep(name, func(ctx context.Context, r *http.Request) (string, error) {
		value := r.Header.Get(header)
		if value == "" {
			return "", NewHTTPError(http.StatusUnauthorized, "Missing API key.")
		}
		return value, nil
	})
}

// APIKeyQueryDep injects an API key from a query parameter into a dep-tagged field.
func APIKeyQueryDep(name, query string) Dependency[string] {
	return Dep(name, func(ctx context.Context, r *http.Request) (string, error) {
		value := r.URL.Query().Get(query)
		if value == "" {
			return "", NewHTTPError(http.StatusUnauthorized, "Missing API key.")
		}
		return value, nil
	})
}
