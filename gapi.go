package gapi

import (
	"context"
	"net/http"

	"github.com/gapi-org/gapi/internal/core"
)

type Config = core.Config
type Handler[In any, Out any] = core.Handler[In, Out]
type Middleware = core.Middleware
type App = core.App
type Router = core.Router
type Group = core.Group

type Operation = core.Operation
type OperationOption = core.OperationOption

type Dependency[T any] = core.Dependency[T]
type DependencyResolver = core.DependencyResolver

type SecurityScheme = core.SecurityScheme

type Response[T any] = core.Response[T]
type Text = core.Text
type HTML = core.HTML
type NoContent = core.NoContent
type Redirect = core.Redirect
type File = core.File
type Attachment = core.Attachment
type Stream = core.Stream
type SSEEvent = core.SSEEvent
type SSE = core.SSE

type Problem = core.Problem
type FieldError = core.FieldError
type HTTPError = core.HTTPError
type Validator = core.Validator

func New(config Config) *App { return core.New(config) }

func Get[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	core.Get(router, path, handler, opts...)
}

func Post[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	core.Post(router, path, handler, opts...)
}

func Put[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	core.Put(router, path, handler, opts...)
}

func Patch[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	core.Patch(router, path, handler, opts...)
}

func Delete[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	core.Delete(router, path, handler, opts...)
}

func Options[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	core.Options(router, path, handler, opts...)
}

func Head[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	core.Head(router, path, handler, opts...)
}

func OperationID(id string) OperationOption          { return core.OperationID(id) }
func Summary(summary string) OperationOption         { return core.Summary(summary) }
func Description(description string) OperationOption { return core.Description(description) }
func Tags(tags ...string) OperationOption            { return core.Tags(tags...) }
func Status(status int) OperationOption              { return core.Status(status) }
func Use(middlewares ...Middleware) OperationOption  { return core.Use(middlewares...) }
func Security(names ...string) OperationOption       { return core.Security(names...) }

func Dep[T any](name string, resolver func(context.Context, *http.Request) (T, error)) Dependency[T] {
	return core.Dep(name, resolver)
}

func Require(dependencies ...DependencyResolver) OperationOption {
	return core.Require(dependencies...)
}

func BearerAuth(name string) SecurityScheme           { return core.BearerAuth(name) }
func BasicAuth(name string) SecurityScheme            { return core.BasicAuth(name) }
func APIKeyHeader(name, header string) SecurityScheme { return core.APIKeyHeader(name, header) }
func APIKeyQuery(name, query string) SecurityScheme   { return core.APIKeyQuery(name, query) }
func APIKeyCookie(name, cookie string) SecurityScheme { return core.APIKeyCookie(name, cookie) }

func BearerTokenDep(name string) Dependency[string] { return core.BearerTokenDep(name) }
func APIKeyHeaderDep(name, header string) Dependency[string] {
	return core.APIKeyHeaderDep(name, header)
}
func APIKeyQueryDep(name, query string) Dependency[string] { return core.APIKeyQueryDep(name, query) }

func NewHTTPError(status int, detail string) error { return core.NewHTTPError(status, detail) }
