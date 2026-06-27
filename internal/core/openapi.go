package core

import (
	"encoding/json"
	"net/http"

	"github.com/gapi-org/gapi/internal/docsui"
	"github.com/gapi-org/gapi/internal/openapi"
)

func (app *App) serveDocs(w http.ResponseWriter, r *http.Request) {
	docsui.Serve(w)
}

func (app *App) serveReDoc(w http.ResponseWriter, r *http.Request) {
	docsui.ServeReDoc(w)
}

func (app *App) serveScalar(w http.ResponseWriter, r *http.Request) {
	docsui.ServeScalar(w)
}

func (app *App) serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(app.openapiSpec())
}

func (app *App) openapiSpec() map[string]any {
	routes := make([]openapi.Route, 0, len(app.routes))
	for _, route := range app.routes {
		routes = append(routes, openapi.Route{
			Method: route.method,
			Path:   route.path,
			Operation: openapi.Operation{
				OperationID: route.operation.OperationID,
				Summary:     route.operation.Summary,
				Description: route.operation.Description,
				Tags:        route.operation.Tags,
				Status:      route.operation.Status,
				Security:    route.operation.security,
			},
			Input:  route.input,
			Output: route.output,
		})
	}

	securitySchemes := make([]openapi.SecurityScheme, 0, len(app.securitySchemes))
	for _, scheme := range app.securitySchemes {
		securitySchemes = append(securitySchemes, openapi.SecurityScheme{
			Name:   scheme.Name,
			Type:   scheme.Type,
			Scheme: scheme.Scheme,
			In:     scheme.In,
		})
	}

	return openapi.Spec(openapi.Config{
		Title:           app.config.Title,
		Version:         app.config.Version,
		SecuritySchemes: securitySchemes,
	}, routes)
}
