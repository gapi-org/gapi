# Testing

Gapi includes a small `testing` helper package for `httptest` workflows.

```go
var body HelloOut

gapitest.New(app).
	GET("/hello").
	Expect(t).
	Status(http.StatusOK).
	Decode(&body)
```

The package also includes an OpenAPI snapshot helper:

```go
gapitest.AssertOpenAPIMatchesSnapshot(t, app, "testdata/openapi.golden.json")
```

Set `UPDATE_SNAPSHOTS=1` to refresh snapshots.
