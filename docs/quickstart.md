# Quickstart

Install Gapi:

```bash
go get github.com/gapi-org/gapi
```

Create a small API:

```go
app := gapi.New(gapi.Config{Title: "Hello API", Version: "0.1.0"})
gapi.Get[HelloIn, HelloOut](app, "/hello", Hello)
http.ListenAndServe(":8080", app)
```

Open:

- `http://localhost:8080/hello`
- `http://localhost:8080/openapi.json`
- `http://localhost:8080/docs`
