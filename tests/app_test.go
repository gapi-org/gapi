package gapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kushagra1122/gapi"
	"github.com/Kushagra1122/gapi/middleware"
)

func TestTypedGetBindsPathAndQueryAndReturnsJSON(t *testing.T) {
	type input struct {
		ID      int  `path:"id"`
		Verbose bool `query:"verbose"`
	}

	type output struct {
		ID      int  `json:"id"`
		Verbose bool `json:"verbose"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.1.0"})
	gapi.Get[input, output](app, "/things/{id}", func(ctx context.Context, in input) (output, error) {
		return output{ID: in.ID, Verbose: in.Verbose}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/things/42?verbose=true", nil)
	res := httptest.NewRecorder()

	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", res.Code, res.Body.String())
	}

	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	var body output
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not JSON: %v", err)
	}
	if body != (output{ID: 42, Verbose: true}) {
		t.Fatalf("unexpected response body: %#v", body)
	}
}

func TestPostBindsJSONBodyAndUsesConfiguredStatus(t *testing.T) {
	type createBody struct {
		Title string `json:"title"`
	}

	type input struct {
		Body createBody `body:""`
	}

	type output struct {
		Title string `json:"title"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.1.0"})
	gapi.Post[input, output](
		app,
		"/todos",
		func(ctx context.Context, in input) (output, error) {
			return output{Title: in.Body.Title}, nil
		},
		gapi.Status(http.StatusCreated),
	)

	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"write tests"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	app.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d with body %s", res.Code, res.Body.String())
	}

	var body output
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not JSON: %v", err)
	}
	if body.Title != "write tests" {
		t.Fatalf("expected title to be bound from JSON body, got %q", body.Title)
	}
}

func TestResponseHelpers(t *testing.T) {
	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	gapi.Get[struct{}, gapi.Text](app, "/text", func(ctx context.Context, in struct{}) (gapi.Text, error) {
		return gapi.Text{Body: "hello"}, nil
	})
	gapi.Get[struct{}, gapi.NoContent](app, "/empty", func(ctx context.Context, in struct{}) (gapi.NoContent, error) {
		return gapi.NoContent{}, nil
	})
	gapi.Get[struct{}, gapi.Redirect](app, "/redirect", func(ctx context.Context, in struct{}) (gapi.Redirect, error) {
		return gapi.Redirect{Location: "/text"}, nil
	})

	textReq := httptest.NewRequest(http.MethodGet, "/text", nil)
	textRes := httptest.NewRecorder()
	app.ServeHTTP(textRes, textReq)
	if textRes.Code != http.StatusOK || textRes.Body.String() != "hello" {
		t.Fatalf("unexpected text response: %d %q", textRes.Code, textRes.Body.String())
	}

	emptyReq := httptest.NewRequest(http.MethodGet, "/empty", nil)
	emptyRes := httptest.NewRecorder()
	app.ServeHTTP(emptyRes, emptyReq)
	if emptyRes.Code != http.StatusNoContent || emptyRes.Body.Len() != 0 {
		t.Fatalf("unexpected no-content response: %d %q", emptyRes.Code, emptyRes.Body.String())
	}

	redirectReq := httptest.NewRequest(http.MethodGet, "/redirect", nil)
	redirectRes := httptest.NewRecorder()
	app.ServeHTTP(redirectRes, redirectReq)
	if redirectRes.Code != http.StatusFound || redirectRes.Header().Get("Location") != "/text" {
		t.Fatalf("unexpected redirect response: %d headers=%#v", redirectRes.Code, redirectRes.Header())
	}
}

func TestAdvancedResponseHelpers(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "hello.txt")
	if err := os.WriteFile(filePath, []byte("file body"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.5.0"})
	gapi.Get[struct{}, gapi.File](app, "/file", func(ctx context.Context, in struct{}) (gapi.File, error) {
		return gapi.File{Path: filePath, ContentType: "text/plain; charset=utf-8"}, nil
	})
	gapi.Get[struct{}, gapi.Attachment](app, "/download", func(ctx context.Context, in struct{}) (gapi.Attachment, error) {
		return gapi.Attachment{Path: filePath, Filename: "download.txt"}, nil
	})
	gapi.Get[struct{}, gapi.Stream](app, "/stream", func(ctx context.Context, in struct{}) (gapi.Stream, error) {
		return gapi.Stream{ContentType: "text/plain; charset=utf-8", Body: io.NopCloser(strings.NewReader("stream body"))}, nil
	})
	gapi.Get[struct{}, gapi.SSE](app, "/events", func(ctx context.Context, in struct{}) (gapi.SSE, error) {
		return gapi.SSE{Events: []gapi.SSEEvent{{Event: "ready", Data: "ok"}}}, nil
	})

	fileReq := httptest.NewRequest(http.MethodGet, "/file", nil)
	fileRes := httptest.NewRecorder()
	app.ServeHTTP(fileRes, fileReq)
	if fileRes.Code != http.StatusOK || fileRes.Body.String() != "file body" {
		t.Fatalf("unexpected file response: %d %q", fileRes.Code, fileRes.Body.String())
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "/download", nil)
	downloadRes := httptest.NewRecorder()
	app.ServeHTTP(downloadRes, downloadReq)
	if got := downloadRes.Header().Get("Content-Disposition"); got != `attachment; filename="download.txt"` {
		t.Fatalf("unexpected content disposition: %q", got)
	}

	streamReq := httptest.NewRequest(http.MethodGet, "/stream", nil)
	streamRes := httptest.NewRecorder()
	app.ServeHTTP(streamRes, streamReq)
	if streamRes.Body.String() != "stream body" {
		t.Fatalf("unexpected stream response: %q", streamRes.Body.String())
	}

	eventsReq := httptest.NewRequest(http.MethodGet, "/events", nil)
	eventsRes := httptest.NewRecorder()
	app.ServeHTTP(eventsRes, eventsReq)
	if got := eventsRes.Header().Get("Content-Type"); got != "text/event-stream; charset=utf-8" {
		t.Fatalf("unexpected SSE content type: %q", got)
	}
	if !strings.Contains(eventsRes.Body.String(), "event: ready\n") || !strings.Contains(eventsRes.Body.String(), "data: ok\n\n") {
		t.Fatalf("unexpected SSE body: %q", eventsRes.Body.String())
	}
}

func TestBindsHeaderCookieAndDefaultQueryValues(t *testing.T) {
	type input struct {
		TraceID string `header:"X-Trace-ID"`
		Session string `cookie:"session"`
		Limit   int    `query:"limit" default:"25"`
	}
	type output struct {
		TraceID string `json:"traceId"`
		Session string `json:"session"`
		Limit   int    `json:"limit"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	gapi.Get[input, output](app, "/bound", func(ctx context.Context, in input) (output, error) {
		return output{TraceID: in.TraceID, Session: in.Session, Limit: in.Limit}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/bound", nil)
	req.Header.Set("X-Trace-ID", "trace-123")
	req.AddCookie(&http.Cookie{Name: "session", Value: "session-abc"})
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", res.Code, res.Body.String())
	}
	var body output
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not JSON: %v", err)
	}
	if body != (output{TraceID: "trace-123", Session: "session-abc", Limit: 25}) {
		t.Fatalf("unexpected response body: %#v", body)
	}
}

func TestValidationErrorsReturnProblemJSON(t *testing.T) {
	type createBody struct {
		Title string `json:"title" validate:"required,min=3"`
		Email string `json:"email" validate:"required,email"`
	}
	type input struct {
		Body createBody `body:""`
	}
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	gapi.Post[input, output](app, "/validate", func(ctx context.Context, in input) (output, error) {
		return output{OK: true}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(`{"title":"go","email":"not-email"}`))
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d with body %s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/problem+json") {
		t.Fatalf("expected problem JSON content type, got %q", got)
	}
	var problem struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Status int    `json:"status"`
		Errors []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem response was not JSON: %v", err)
	}
	if problem.Status != http.StatusUnprocessableEntity || len(problem.Errors) != 2 {
		t.Fatalf("unexpected validation problem: %#v", problem)
	}
	if problem.Errors[0].Field != "body.title" {
		t.Fatalf("expected body field path, got %#v", problem.Errors)
	}
}

type customValidatedBody struct {
	Value string `json:"value"`
}

func (body customValidatedBody) ValidateGapi() []gapi.FieldError {
	if body.Value != "allowed" {
		return []gapi.FieldError{{
			Field:   "value",
			Message: "must be allowed",
			Code:    "custom",
		}}
	}
	return nil
}

func TestCustomValidatorReturnsProblemJSON(t *testing.T) {
	type input struct {
		Body customValidatedBody `body:""`
	}
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.4.0"})
	gapi.Post[input, output](app, "/custom-validate", func(ctx context.Context, in input) (output, error) {
		return output{OK: true}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/custom-validate", strings.NewReader(`{"value":"denied"}`))
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d with body %s", res.Code, res.Body.String())
	}
	var problem gapi.Problem
	if err := json.Unmarshal(res.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem response was not JSON: %v", err)
	}
	if len(problem.Errors) != 1 || problem.Errors[0].Field != "body.value" || problem.Errors[0].Code != "custom" {
		t.Fatalf("unexpected custom validation problem: %#v", problem)
	}
}

func TestBindingErrorsReturnStructuredProblemJSON(t *testing.T) {
	type input struct {
		ID int `path:"id"`
	}
	type output struct {
		ID int `json:"id"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	gapi.Get[input, output](app, "/items/{id}", func(ctx context.Context, in input) (output, error) {
		return output{ID: in.ID}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/not-an-int", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %s", res.Code, res.Body.String())
	}
	var problem struct {
		Status int `json:"status"`
		Errors []struct {
			Field string `json:"field"`
			Code  string `json:"code"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem response was not JSON: %v", err)
	}
	if problem.Status != http.StatusBadRequest || len(problem.Errors) != 1 || problem.Errors[0].Field != "path.id" {
		t.Fatalf("unexpected binding problem: %#v", problem)
	}
}

func TestHandlerErrorWritesProblemJSON(t *testing.T) {
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.1.0"})
	gapi.Get[struct{}, output](app, "/fail", func(ctx context.Context, in struct{}) (output, error) {
		return output{}, errors.New("database unavailable")
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	res := httptest.NewRecorder()

	app.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.Code)
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/problem+json") {
		t.Fatalf("expected problem JSON content type, got %q", got)
	}

	var problem struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem response was not JSON: %v", err)
	}
	if problem.Status != http.StatusInternalServerError || problem.Title == "" || problem.Type == "" {
		t.Fatalf("unexpected problem response: %#v", problem)
	}
}

func TestHTTPErrorMapsHandlerErrorToProblemJSON(t *testing.T) {
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.4.0"})
	gapi.Get[struct{}, output](app, "/missing", func(ctx context.Context, in struct{}) (output, error) {
		return output{}, gapi.NewHTTPError(http.StatusNotFound, "todo not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d with body %s", res.Code, res.Body.String())
	}
	var problem gapi.Problem
	if err := json.Unmarshal(res.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem response was not JSON: %v", err)
	}
	if problem.Detail != "todo not found" || problem.Title != "Not Found" {
		t.Fatalf("unexpected problem response: %#v", problem)
	}
}

func TestNotFoundWritesProblemJSON(t *testing.T) {
	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.4.0"})

	req := httptest.NewRequest(http.MethodGet, "/missing-route", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d with body %s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/problem+json") {
		t.Fatalf("expected problem JSON content type, got %q", got)
	}
}

func TestOpenAPIAndDocsEndpointsAreGenerated(t *testing.T) {
	type input struct {
		ID int `path:"id"`
	}

	type output struct {
		ID int `json:"id"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.1.0"})
	gapi.Get[input, output](
		app,
		"/things/{id}",
		func(ctx context.Context, in input) (output, error) {
			return output{ID: in.ID}, nil
		},
		gapi.OperationID("getThing"),
		gapi.Summary("Get a thing"),
		gapi.Tags("things"),
	)

	openAPIReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIRes := httptest.NewRecorder()
	app.ServeHTTP(openAPIRes, openAPIReq)

	if openAPIRes.Code != http.StatusOK {
		t.Fatalf("expected OpenAPI status 200, got %d", openAPIRes.Code)
	}

	var spec map[string]any
	if err := json.Unmarshal(openAPIRes.Body.Bytes(), &spec); err != nil {
		t.Fatalf("OpenAPI response was not JSON: %v", err)
	}
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1.0, got %#v", spec["openapi"])
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths object in OpenAPI spec")
	}
	if _, ok := paths["/things/{id}"]; !ok {
		t.Fatalf("expected registered path in OpenAPI spec, got %#v", paths)
	}

	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRes := httptest.NewRecorder()
	app.ServeHTTP(docsRes, docsReq)

	if docsRes.Code != http.StatusOK {
		t.Fatalf("expected docs status 200, got %d", docsRes.Code)
	}
	if got := docsRes.Body.String(); !strings.Contains(got, "/openapi.json") {
		t.Fatalf("expected docs page to reference /openapi.json, got %q", got)
	}

	for _, path := range []string{"/redoc", "/scalar"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("expected %s status 200, got %d", path, res.Code)
		}
		if got := res.Body.String(); !strings.Contains(got, "/openapi.json") {
			t.Fatalf("expected %s page to reference /openapi.json, got %q", path, got)
		}
	}
}

func TestOpenAPIIncludesSecuritySchemesAndRequirements(t *testing.T) {
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	app.RegisterSecurity(gapi.BearerAuth("bearerAuth"))
	gapi.Get[struct{}, output](
		app,
		"/secure",
		func(ctx context.Context, in struct{}) (output, error) {
			return output{OK: true}, nil
		},
		gapi.Security("bearerAuth"),
	)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	var spec map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &spec); err != nil {
		t.Fatalf("OpenAPI response was not JSON: %v", err)
	}
	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatalf("expected components object in OpenAPI spec")
	}
	securitySchemes, ok := components["securitySchemes"].(map[string]any)
	if !ok {
		t.Fatalf("expected securitySchemes object in OpenAPI spec")
	}
	if _, ok := securitySchemes["bearerAuth"]; !ok {
		t.Fatalf("expected bearerAuth security scheme, got %#v", securitySchemes)
	}
}

func TestOpenAPISchemaIncludesAdvancedTypes(t *testing.T) {
	type Embedded struct {
		TraceID string `json:"trace_id" doc:"Trace ID"`
	}
	type output struct {
		Embedded
		Name     *string         `json:"name" validate:"required" example:"Ada"`
		Metadata json.RawMessage `json:"metadata"`
		Labels   map[string]int  `json:"labels"`
		Tags     []string        `json:"tags"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.5.0"})
	gapi.Get[struct{}, output](app, "/schema", func(ctx context.Context, in struct{}) (output, error) {
		return output{}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	var spec map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &spec); err != nil {
		t.Fatalf("OpenAPI response was not JSON: %v", err)
	}
	paths := spec["paths"].(map[string]any)
	path := paths["/schema"].(map[string]any)
	getOperation := path["get"].(map[string]any)
	responses := getOperation["responses"].(map[string]any)
	okResponse := responses["200"].(map[string]any)
	content := okResponse["content"].(map[string]any)
	jsonContent := content["application/json"].(map[string]any)
	schema := jsonContent["schema"].(map[string]any)
	properties := schema["properties"].(map[string]any)

	if _, ok := properties["trace_id"]; !ok {
		t.Fatalf("expected embedded struct field to be flattened into schema: %#v", properties)
	}
	nameSchema := properties["name"].(map[string]any)
	if types, ok := nameSchema["type"].([]any); !ok || len(types) != 2 || types[0] != "string" || types[1] != "null" {
		t.Fatalf("expected nullable string schema, got %#v", nameSchema)
	}
	if nameSchema["example"] != "Ada" {
		t.Fatalf("expected example to be preserved, got %#v", nameSchema)
	}
	metadataSchema := properties["metadata"].(map[string]any)
	if metadataSchema["description"] == nil {
		t.Fatalf("expected raw JSON schema to be documented, got %#v", metadataSchema)
	}
}

func TestHeadAndOptionsHelpersRegisterRoutes(t *testing.T) {
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	gapi.Head[struct{}, output](app, "/health", func(ctx context.Context, in struct{}) (output, error) {
		return output{OK: true}, nil
	})
	gapi.Options[struct{}, output](app, "/health", func(ctx context.Context, in struct{}) (output, error) {
		return output{OK: true}, nil
	})

	headReq := httptest.NewRequest(http.MethodHead, "/health", nil)
	headRes := httptest.NewRecorder()
	app.ServeHTTP(headRes, headReq)
	if headRes.Code != http.StatusOK {
		t.Fatalf("expected HEAD status 200, got %d", headRes.Code)
	}

	optionsReq := httptest.NewRequest(http.MethodOptions, "/health", nil)
	optionsRes := httptest.NewRecorder()
	app.ServeHTTP(optionsRes, optionsReq)
	if optionsRes.Code != http.StatusOK {
		t.Fatalf("expected OPTIONS status 200, got %d", optionsRes.Code)
	}
}

func TestGroupRegistersPrefixedRouteAndOpenAPIPath(t *testing.T) {
	type output struct {
		Message string `json:"message"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.2.0"})
	api := app.Group("/api/v1")
	gapi.Get[struct{}, output](api, "/hello", func(ctx context.Context, in struct{}) (output, error) {
		return output{Message: "hello"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", res.Code, res.Body.String())
	}

	openAPIReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIRes := httptest.NewRecorder()
	app.ServeHTTP(openAPIRes, openAPIReq)

	var spec map[string]any
	if err := json.Unmarshal(openAPIRes.Body.Bytes(), &spec); err != nil {
		t.Fatalf("OpenAPI response was not JSON: %v", err)
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths object in OpenAPI spec")
	}
	if _, ok := paths["/api/v1/hello"]; !ok {
		t.Fatalf("expected grouped path in OpenAPI spec, got %#v", paths)
	}
}

func TestMiddlewareCompositionOrder(t *testing.T) {
	type output struct {
		OK bool `json:"ok"`
	}

	var order []string
	record := func(name string) gapi.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+":before")
				next.ServeHTTP(w, r)
				order = append(order, name+":after")
			})
		}
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.2.0"})
	app.Use(record("app"))
	api := app.Group("/api", gapi.Use(record("group")))
	gapi.Get[struct{}, output](
		api,
		"/order",
		func(ctx context.Context, in struct{}) (output, error) {
			order = append(order, "handler")
			return output{OK: true}, nil
		},
		gapi.Use(record("route")),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/order", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", res.Code, res.Body.String())
	}
	want := []string{
		"app:before",
		"group:before",
		"route:before",
		"handler",
		"route:after",
		"group:after",
		"app:after",
	}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected middleware order: got %#v want %#v", order, want)
	}
}

func TestDependencyInjectionBindsDepFields(t *testing.T) {
	type currentUser struct {
		Email string `json:"email"`
	}
	type input struct {
		User currentUser `dep:"currentUser"`
	}
	type output struct {
		Email string `json:"email"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	app.Provide(gapi.Dep("currentUser", func(ctx context.Context, r *http.Request) (currentUser, error) {
		return currentUser{Email: r.Header.Get("X-User-Email")}, nil
	}))
	gapi.Get[input, output](app, "/me", func(ctx context.Context, in input) (output, error) {
		return output{Email: in.User.Email}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-User-Email", "ada@example.com")
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", res.Code, res.Body.String())
	}
	var body output
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not JSON: %v", err)
	}
	if body.Email != "ada@example.com" {
		t.Fatalf("expected injected user email, got %q", body.Email)
	}
}

func TestDependencyInjectionCachesPerRequest(t *testing.T) {
	type token struct {
		Value string
	}
	type input struct {
		First  token `dep:"token"`
		Second token `dep:"token"`
	}
	type output struct {
		First  string `json:"first"`
		Second string `json:"second"`
	}

	calls := 0
	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.4.0"})
	app.Provide(gapi.Dep("token", func(ctx context.Context, r *http.Request) (token, error) {
		calls++
		return token{Value: "cached"}, nil
	}))
	gapi.Get[input, output](app, "/cached", func(ctx context.Context, in input) (output, error) {
		return output{First: in.First.Value, Second: in.Second.Value}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/cached", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", res.Code, res.Body.String())
	}
	if calls != 1 {
		t.Fatalf("expected dependency to resolve once, got %d calls", calls)
	}
}

func TestBearerTokenDependencyInjectsToken(t *testing.T) {
	type input struct {
		Token string `dep:"token"`
	}
	type output struct {
		Token string `json:"token"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.5.0"})
	app.Provide(gapi.BearerTokenDep("token"))
	gapi.Get[input, output](app, "/auth-token", func(ctx context.Context, in input) (output, error) {
		return output{Token: in.Token}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/auth-token", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", res.Code, res.Body.String())
	}
	var body output
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not JSON: %v", err)
	}
	if body.Token != "secret" {
		t.Fatalf("expected injected bearer token, got %q", body.Token)
	}
}

func TestBearerTokenDependencyMissingTokenReturnsUnauthorized(t *testing.T) {
	type input struct {
		Token string `dep:"token"`
	}
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.5.0"})
	app.Provide(gapi.BearerTokenDep("token"))
	gapi.Get[input, output](app, "/auth-token", func(ctx context.Context, in input) (output, error) {
		return output{OK: true}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/auth-token", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d with body %s", res.Code, res.Body.String())
	}
}

func TestRecoverMiddlewareWritesProblemJSON(t *testing.T) {
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.2.0"})
	app.Use(middleware.Recover())
	gapi.Get[struct{}, output](app, "/panic", func(ctx context.Context, in struct{}) (output, error) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.Code)
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/problem+json") {
		t.Fatalf("expected problem JSON content type, got %q", got)
	}
}

func TestRequestIDMiddlewareSetsHeaderAndContext(t *testing.T) {
	type output struct {
		RequestID string `json:"requestId"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.2.0"})
	app.Use(middleware.RequestID())
	gapi.Get[struct{}, output](app, "/request-id", func(ctx context.Context, in struct{}) (output, error) {
		return output{RequestID: middleware.RequestIDFromContext(ctx)}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	req.Header.Set("X-Request-ID", "test-request")
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if got := res.Header().Get("X-Request-ID"); got != "test-request" {
		t.Fatalf("expected response request ID header, got %q", got)
	}
	var body output
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not JSON: %v", err)
	}
	if body.RequestID != "test-request" {
		t.Fatalf("expected request ID in context, got %q", body.RequestID)
	}
}

func TestBodyLimitMiddlewareRejectsLargeJSONBody(t *testing.T) {
	type input struct {
		Body struct {
			Title string `json:"title"`
		} `body:""`
	}
	type output struct {
		Title string `json:"title"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.2.0"})
	app.Use(middleware.BodyLimit(8))
	gapi.Post[input, output](app, "/limited", func(ctx context.Context, in input) (output, error) {
		return output{Title: in.Body.Title}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/limited", strings.NewReader(`{"title":"too large"}`))
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/problem+json") {
		t.Fatalf("expected problem JSON content type, got %q", got)
	}
}

func TestTimeoutMiddlewareReturnsServiceUnavailable(t *testing.T) {
	type output struct {
		OK bool `json:"ok"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.2.0"})
	app.Use(middleware.Timeout(time.Nanosecond))
	gapi.Get[struct{}, output](app, "/slow", func(ctx context.Context, in struct{}) (output, error) {
		time.Sleep(20 * time.Millisecond)
		return output{OK: true}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d with body %s", res.Code, res.Body.String())
	}
}
