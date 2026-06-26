package core

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gapi-org/gapi/internal/binding"
	"github.com/gapi-org/gapi/internal/validate"
)

// Router is implemented by App and Group route targets.
type Router interface {
	root() *App
	routePrefix() string
	routeMiddleware() []Middleware
}

// Group registers routes under a shared path prefix and middleware stack.
type Group struct {
	app         *App
	prefix      string
	middlewares []Middleware
}

type route struct {
	method    string
	path      string
	operation Operation
	input     reflect.Type
	output    reflect.Type
}

func (app *App) root() *App {
	return app
}

func (app *App) routePrefix() string {
	return ""
}

func (app *App) routeMiddleware() []Middleware {
	return append([]Middleware(nil), app.middlewares...)
}

// Use appends middleware to routes registered after this call.
func (app *App) Use(middlewares ...Middleware) {
	app.middlewares = append(app.middlewares, middlewares...)
}

// Group creates a route group below the app.
func (app *App) Group(prefix string, opts ...OperationOption) *Group {
	return newGroup(app, "", nil, prefix, opts...)
}

func (group *Group) root() *App {
	return group.app
}

func (group *Group) routePrefix() string {
	return group.prefix
}

func (group *Group) routeMiddleware() []Middleware {
	middlewares := group.app.routeMiddleware()
	middlewares = append(middlewares, group.middlewares...)
	return middlewares
}

// Use appends middleware to routes registered on the group after this call.
func (group *Group) Use(middlewares ...Middleware) {
	group.middlewares = append(group.middlewares, middlewares...)
}

// Group creates a nested route group.
func (group *Group) Group(prefix string, opts ...OperationOption) *Group {
	return newGroup(group.app, group.prefix, group.middlewares, prefix, opts...)
}

func newGroup(app *App, parentPrefix string, parentMiddleware []Middleware, prefix string, opts ...OperationOption) *Group {
	operation := Operation{}
	for _, opt := range opts {
		opt(&operation)
	}
	middlewares := append([]Middleware(nil), parentMiddleware...)
	middlewares = append(middlewares, operation.middlewares...)
	return &Group{
		app:         app,
		prefix:      joinPaths(parentPrefix, prefix),
		middlewares: middlewares,
	}
}

// Get registers a typed GET operation.
func Get[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	register(router, http.MethodGet, path, handler, opts...)
}

// Post registers a typed POST operation.
func Post[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	register(router, http.MethodPost, path, handler, opts...)
}

// Put registers a typed PUT operation.
func Put[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	register(router, http.MethodPut, path, handler, opts...)
}

// Patch registers a typed PATCH operation.
func Patch[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	register(router, http.MethodPatch, path, handler, opts...)
}

// Delete registers a typed DELETE operation.
func Delete[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	register(router, http.MethodDelete, path, handler, opts...)
}

// Options registers a typed OPTIONS operation.
func Options[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	register(router, http.MethodOptions, path, handler, opts...)
}

// Head registers a typed HEAD operation.
func Head[In any, Out any](router Router, path string, handler Handler[In, Out], opts ...OperationOption) {
	register(router, http.MethodHead, path, handler, opts...)
}

func register[In any, Out any](router Router, method, path string, handler Handler[In, Out], opts ...OperationOption) {
	operation := Operation{Status: http.StatusOK}
	for _, opt := range opts {
		opt(&operation)
	}

	app := router.root()
	fullPath := joinPaths(router.routePrefix(), path)
	inType := reflect.TypeOf((*In)(nil)).Elem()
	outType := reflect.TypeOf((*Out)(nil)).Elem()
	binder := binding.Compile(inType)
	validator := validate.Compile(inType)
	dependencies := dependencyMap(app.dependencies, operation.dependencies)
	dependencyPlan := compileDependencyPlan(inType)

	app.routes = append(app.routes, route{
		method:    method,
		path:      fullPath,
		operation: operation,
		input:     inType,
		output:    outType,
	})

	pattern := method + " " + fullPath
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in In
		if err := dependencyPlan.resolve(r.Context(), r, reflect.ValueOf(&in).Elem(), dependencies); err != nil {
			if problem, ok := problemFromError(err); ok {
				writeProblem(w, problem)
				return
			}
			writeProblem(w, Problem{
				Type:   "about:blank",
				Title:  "Internal Server Error",
				Status: http.StatusInternalServerError,
				Detail: err.Error(),
			})
			return
		}
		if err := binder.Bind(r, reflect.ValueOf(&in).Elem()); err != nil {
			bindingErr, ok := err.(binding.Error)
			if ok {
				writeProblem(w, Problem{
					Type:   "https://gapi.dev/problems/binding-error",
					Title:  "Bad Request",
					Status: http.StatusBadRequest,
					Detail: "Request binding failed.",
					Errors: []FieldError{{
						Field:   bindingErr.Field,
						Message: bindingErr.Message,
						Code:    bindingErr.Code,
					}},
				})
				return
			}
			writeProblem(w, Problem{
				Type:   "about:blank",
				Title:  "Bad Request",
				Status: http.StatusBadRequest,
				Detail: err.Error(),
			})
			return
		}
		if err := validator.Validate(reflect.ValueOf(&in).Elem()); err != nil {
			validationErr, ok := err.(validate.Error)
			if !ok {
				writeProblem(w, Problem{
					Type:   "about:blank",
					Title:  "Unprocessable Entity",
					Status: http.StatusUnprocessableEntity,
					Detail: err.Error(),
				})
				return
			}
			writeProblem(w, Problem{
				Type:   "https://gapi.dev/problems/validation-error",
				Title:  "Validation failed",
				Status: http.StatusUnprocessableEntity,
				Detail: "Request validation failed.",
				Errors: validationFieldErrors(validationErr.Fields),
			})
			return
		}
		if fields := validateCustom(reflect.ValueOf(&in).Elem()); len(fields) > 0 {
			writeProblem(w, Problem{
				Type:   "https://gapi.dev/problems/validation-error",
				Title:  "Validation failed",
				Status: http.StatusUnprocessableEntity,
				Detail: "Request validation failed.",
				Errors: fields,
			})
			return
		}

		out, err := handler(r.Context(), in)
		if err != nil {
			if problem, ok := problemFromError(err); ok {
				writeProblem(w, problem)
				return
			}
			writeProblem(w, Problem{
				Type:   "about:blank",
				Title:  "Internal Server Error",
				Status: http.StatusInternalServerError,
				Detail: http.StatusText(http.StatusInternalServerError),
			})
			return
		}

		writeOutput(w, out, operation.Status)
	})
	middlewares := router.routeMiddleware()
	middlewares = append(middlewares, operation.middlewares...)
	app.mux.Handle(pattern, applyMiddleware(baseHandler, middlewares))
}

func applyMiddleware(handler http.Handler, middlewares []Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func joinPaths(prefix, path string) string {
	if prefix == "" {
		if path == "" {
			return "/"
		}
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}
	if path == "" || path == "/" {
		return prefix
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
}

func validationFieldErrors(fields []validate.FieldError) []FieldError {
	errors := make([]FieldError, 0, len(fields))
	for _, field := range fields {
		errors = append(errors, FieldError{
			Field:   field.Field,
			Message: field.Message,
			Code:    field.Code,
		})
	}
	return errors
}
