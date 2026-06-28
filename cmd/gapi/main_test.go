package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProjectScaffoldsFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hello-api")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"new", dir}, &stdout, &stderr); err != nil {
		t.Fatalf("new project failed: %v\nstderr: %s", err, stderr.String())
	}

	for _, name := range []string{"go.mod", "main.go", "README.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to be created: %v", name, err)
		}
	}

	mainFile, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read generated main.go: %v", err)
	}
	if !strings.Contains(string(mainFile), "gapi.Get") {
		t.Fatalf("expected generated app to register a Gapi route")
	}
}

func TestRoutesPrintsOpenAPIPaths(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.json")
	if err := os.WriteFile(specPath, []byte(testOpenAPI()), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"routes", "--file", specPath}, &stdout, &stderr); err != nil {
		t.Fatalf("routes failed: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "GET /todos listTodos") {
		t.Fatalf("unexpected routes output: %q", stdout.String())
	}
}

func TestOpenAPIExportsFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openapi.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testOpenAPI()))
	}))
	defer server.Close()

	outPath := filepath.Join(t.TempDir(), "exported.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"openapi", "--url", server.URL + "/openapi.json", "--out", outPath}, &stdout, &stderr); err != nil {
		t.Fatalf("openapi failed: %v\nstderr: %s", err, stderr.String())
	}
	exported, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exported spec: %v", err)
	}
	if !strings.Contains(string(exported), `"operationId":"listTodos"`) {
		t.Fatalf("unexpected exported spec: %s", string(exported))
	}
}

func TestGenWritesClientScaffold(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.json")
	if err := os.WriteFile(specPath, []byte(testOpenAPI()), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "client.go")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"gen", "--file", specPath, "--out", outPath, "--package", "todos"}, &stdout, &stderr); err != nil {
		t.Fatalf("gen failed: %v\nstderr: %s", err, stderr.String())
	}
	generated, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	if !strings.Contains(string(generated), "package todos") || !strings.Contains(string(generated), "func (client Client) ListTodos") {
		t.Fatalf("unexpected generated client:\n%s", string(generated))
	}
}

func TestDevDryRunPrintsCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"dev", "--dry-run"}, &stdout, &stderr); err != nil {
		t.Fatalf("dev dry-run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "go run .") {
		t.Fatalf("unexpected dry-run output: %q", stdout.String())
	}
}

func testOpenAPI() string {
	return `{"openapi":"3.1.0","paths":{"/todos":{"get":{"operationId":"listTodos","responses":{"200":{"description":"OK"}}}}}}`
}
