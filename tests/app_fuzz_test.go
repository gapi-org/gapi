package gapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gapi-org/gapi"
)

func FuzzPathAndQueryBinding(f *testing.F) {
	type input struct {
		ID    int    `path:"id"`
		Limit int    `query:"limit"`
		Name  string `query:"name"`
	}
	type output struct {
		ID    int    `json:"id"`
		Limit int    `json:"limit"`
		Name  string `json:"name"`
	}

	app := gapi.New(gapi.Config{Title: "Fuzz API", Version: "0.5.0"})
	gapi.Get[input, output](app, "/things/{id}", func(ctx context.Context, in input) (output, error) {
		return output{ID: in.ID, Limit: in.Limit, Name: in.Name}, nil
	})

	f.Add("42", "10", "ada")
	f.Add("not-int", "10", "ada")
	f.Add("42", "not-int", "ada")
	f.Fuzz(func(t *testing.T, id, limit, name string) {
		target := "/things/" + url.PathEscape(id) + "?limit=" + url.QueryEscape(limit) + "&name=" + url.QueryEscape(name)
		req := httptest.NewRequest(http.MethodGet, target, nil)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)

		if res.Code < 200 || res.Code >= 500 {
			t.Fatalf("unexpected status %d for id=%q limit=%q name=%q", res.Code, id, limit, name)
		}
	})
}
