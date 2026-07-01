package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		usage(stdout)
		return nil
	}

	switch args[0] {
	case "new":
		if len(args) < 2 {
			return fmt.Errorf("usage: gapi new <name>")
		}
		return newProject(stdout, args[1])
	case "dev":
		return dev(args[1:], stdout, stderr)
	case "routes":
		return routes(args[1:], stdout)
	case "openapi":
		return exportOpenAPI(args[1:], stdout)
	case "gen":
		return gen(args[1:], stdout)
	case "lint":
		return lint(stdout, stderr)
	default:
		usage(stdout)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `gapi is the CLI for Gapi projects.

Usage:
  gapi new <name>
  gapi dev [--dry-run]
  gapi routes --file openapi.json
  gapi openapi [--url http://localhost:8080/openapi.json] [--out openapi.json]
  gapi gen --file openapi.json --out client.go [--package client]
  gapi lint`)
}

func newProject(stdout io.Writer, name string) error {
	dir := filepath.Clean(name)
	if dir == "." || dir == string(filepath.Separator) {
		return fmt.Errorf("invalid project name %q", name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	module := filepath.Base(dir)
	files := map[string]string{
		"go.mod":    "module " + module + "\n\ngo 1.25.3\n\nrequire github.com/Kushagra1122/gapi latest\n",
		"main.go":   starterMain(),
		"README.md": "# " + module + "\n\nGenerated with `gapi new`.\n\nRun:\n\n```bash\ngo mod tidy\ngo run .\n```\n",
	}
	for name, contents := range files {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", path)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			return err
		}
	}

	fmt.Fprintf(stdout, "Created Gapi project in %s\n", dir)
	return nil
}

func starterMain() string {
	return `package main

import (
	"context"
	"net/http"

	"github.com/Kushagra1122/gapi"
	"github.com/Kushagra1122/gapi/middleware"
)

type HelloIn struct {
	Name string ` + "`query:\"name\" default:\"world\" doc:\"Name to greet\"`" + `
}

type HelloOut struct {
	Message string ` + "`json:\"message\"`" + `
}

func Hello(ctx context.Context, in HelloIn) (HelloOut, error) {
	return HelloOut{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
	app := gapi.New(gapi.Config{Title: "Hello API", Version: "0.1.0"})
	app.Use(middleware.Recover())

	gapi.Get[HelloIn, HelloOut](app, "/hello", Hello, gapi.Summary("Say hello"))

	http.ListenAndServe(":8080", app)
}
`
}

func lint(stdout, stderr io.Writer) error {
	cmd := exec.Command("go", "test", "./...")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	text := strings.TrimSpace(output.String())
	if text != "" {
		fmt.Fprintln(stdout, text)
	}
	if err != nil {
		fmt.Fprintln(stderr, "gapi lint failed")
		return err
	}
	fmt.Fprintln(stdout, "gapi lint passed")
	return nil
}

type openAPIDoc struct {
	Paths map[string]map[string]openAPIOperation `json:"paths"`
}

type openAPIOperation struct {
	OperationID string `json:"operationId"`
}

func dev(args []string, stdout, stderr io.Writer) error {
	if hasFlag(args, "--dry-run") {
		fmt.Fprintln(stdout, "go run .")
		return nil
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func routes(args []string, stdout io.Writer) error {
	specPath := flagValue(args, "--file", "")
	if specPath == "" {
		return fmt.Errorf("usage: gapi routes --file openapi.json")
	}
	doc, err := readOpenAPIFile(specPath)
	if err != nil {
		return err
	}
	for _, route := range openAPIRoutes(doc) {
		fmt.Fprintf(stdout, "%s %s %s\n", strings.ToUpper(route.method), route.path, route.operation.OperationID)
	}
	return nil
}

func exportOpenAPI(args []string, stdout io.Writer) error {
	url := flagValue(args, "--url", "http://localhost:8080/openapi.json")
	outPath := flagValue(args, "--out", "")
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("fetch %s: %s", url, res.Status)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if outPath == "" {
		_, err = stdout.Write(body)
		return err
	}
	if err := os.WriteFile(outPath, body, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Wrote OpenAPI document to %s\n", outPath)
	return nil
}

func gen(args []string, stdout io.Writer) error {
	specPath := flagValue(args, "--file", "")
	outPath := flagValue(args, "--out", "")
	packageName := flagValue(args, "--package", "client")
	if specPath == "" || outPath == "" {
		return fmt.Errorf("usage: gapi gen --file openapi.json --out client.go [--package client]")
	}
	doc, err := readOpenAPIFile(specPath)
	if err != nil {
		return err
	}
	source := generateClient(packageName, doc)
	if err := os.WriteFile(outPath, []byte(source), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Wrote client scaffold to %s\n", outPath)
	return nil
}

type openAPIRoute struct {
	path      string
	method    string
	operation openAPIOperation
}

func readOpenAPIFile(path string) (openAPIDoc, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return openAPIDoc{}, err
	}
	var doc openAPIDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return openAPIDoc{}, err
	}
	return doc, nil
}

func openAPIRoutes(doc openAPIDoc) []openAPIRoute {
	var routes []openAPIRoute
	for path, pathItem := range doc.Paths {
		for method, operation := range pathItem {
			routes = append(routes, openAPIRoute{
				path:      path,
				method:    method,
				operation: operation,
			})
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].path == routes[j].path {
			return routes[i].method < routes[j].method
		}
		return routes[i].path < routes[j].path
	})
	return routes
}

func generateClient(packageName string, doc openAPIDoc) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "package %s\n\n", packageName)
	builder.WriteString("import \"net/http\"\n\n")
	builder.WriteString("type Client struct {\n\tBaseURL string\n\tHTTPClient *http.Client\n}\n\n")
	for _, route := range openAPIRoutes(doc) {
		name := exportedName(route.operation.OperationID)
		if name == "" {
			name = exportedName(route.method + "_" + strings.Trim(route.path, "/"))
		}
		fmt.Fprintf(&builder, "func (client Client) %s() (*http.Request, error) {\n", name)
		fmt.Fprintf(&builder, "\treturn http.NewRequest(%q, client.BaseURL+%q, nil)\n", strings.ToUpper(route.method), route.path)
		builder.WriteString("}\n\n")
	}
	return builder.String()
}

func exportedName(value string) string {
	var builder strings.Builder
	upperNext := true
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			upperNext = true
			continue
		}
		if upperNext {
			builder.WriteRune(unicode.ToUpper(r))
			upperNext = false
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func flagValue(args []string, name, fallback string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return fallback
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}
