package gapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gapi-org/gapi"
)

func BenchmarkTypedRoute(b *testing.B) {
	type input struct {
		ID int `path:"id"`
	}
	type output struct {
		ID int `json:"id"`
	}

	app := gapi.New(gapi.Config{Title: "Benchmark API", Version: "0.5.0"})
	gapi.Get[input, output](app, "/things/{id}", func(ctx context.Context, in input) (output, error) {
		return output{ID: in.ID}, nil
	})
	req := httptest.NewRequest(http.MethodGet, "/things/42", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", res.Code)
		}
	}
}

func BenchmarkOpenAPI(b *testing.B) {
	type output struct {
		Message string `json:"message"`
	}

	app := gapi.New(gapi.Config{Title: "Benchmark API", Version: "0.5.0"})
	for _, path := range []string{"/a", "/b", "/c", "/d"} {
		gapi.Get[struct{}, output](app, path, func(ctx context.Context, in struct{}) (output, error) {
			return output{Message: "ok"}, nil
		})
	}
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", res.Code)
		}
	}
}
