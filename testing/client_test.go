package gapitest_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gapi-org/gapi"
	gapitest "github.com/gapi-org/gapi/testing"
)

func TestClientGETStatusAndDecode(t *testing.T) {
	type output struct {
		Message string `json:"message"`
	}

	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	gapi.Get[struct{}, output](app, "/hello", func(ctx context.Context, in struct{}) (output, error) {
		return output{Message: "hello"}, nil
	})

	var body output
	gapitest.New(app).
		GET("/hello").
		Expect(t).
		Status(http.StatusOK).
		Decode(&body)

	if body.Message != "hello" {
		t.Fatalf("expected decoded message, got %q", body.Message)
	}
}

func TestClientBodyString(t *testing.T) {
	app := gapi.New(gapi.Config{Title: "Test API", Version: "0.3.0"})
	gapi.Get[struct{}, gapi.Text](app, "/plain", func(ctx context.Context, in struct{}) (gapi.Text, error) {
		return gapi.Text{Body: "plain text"}, nil
	})

	gapitest.New(app).
		GET("/plain").
		Expect(t).
		Status(http.StatusOK).
		BodyString("plain text")
}
