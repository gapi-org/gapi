package core

import (
	"bytes"
	"context"
	"net/http"
	"strings"
)

// Config describes the generated API document and built-in docs endpoints.
type Config struct {
	Title   string
	Version string
}

// Handler is the primary typed handler shape.
type Handler[In any, Out any] func(context.Context, In) (Out, error)

// Middleware is standard Go HTTP middleware.
type Middleware func(http.Handler) http.Handler

// App is a net/http-compatible application.
type App struct {
	config          Config
	mux             *http.ServeMux
	routes          []route
	middlewares     []Middleware
	dependencies    map[string]DependencyResolver
	securitySchemes []SecurityScheme
}

// New creates an application with OpenAPI and docs endpoints registered.
func New(config Config) *App {
	if config.Title == "" {
		config.Title = "Gapi API"
	}
	if config.Version == "" {
		config.Version = "0.1.0"
	}

	app := &App{
		config: config,
		mux:    http.NewServeMux(),
	}
	app.mux.HandleFunc("GET /openapi.json", app.serveOpenAPI)
	app.mux.HandleFunc("GET /docs", app.serveDocs)
	app.mux.HandleFunc("GET /redoc", app.serveReDoc)
	app.mux.HandleFunc("GET /scalar", app.serveScalar)
	return app
}

// ServeHTTP implements http.Handler.
func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	recorder := newResponseRecorder()
	app.mux.ServeHTTP(recorder, r)
	if (recorder.Code == http.StatusNotFound || recorder.Code == http.StatusMethodNotAllowed) &&
		!strings.Contains(recorder.Header().Get("Content-Type"), "application/problem+json") {
		copyHeaders(w.Header(), recorder.Header())
		writeProblem(w, Problem{
			Type:   "about:blank",
			Title:  http.StatusText(recorder.Code),
			Status: recorder.Code,
			Detail: http.StatusText(recorder.Code),
		})
		return
	}
	for name, values := range recorder.Header() {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(recorder.Code)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(recorder.Body.Bytes())
}

type responseRecorder struct {
	Code   int
	Body   *bytes.Buffer
	header http.Header
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{
		Code:   http.StatusOK,
		Body:   &bytes.Buffer{},
		header: http.Header{},
	}
}

func (recorder *responseRecorder) Header() http.Header {
	return recorder.header
}

func (recorder *responseRecorder) Write(body []byte) (int, error) {
	return recorder.Body.Write(body)
}

func (recorder *responseRecorder) WriteHeader(status int) {
	recorder.Code = status
}
