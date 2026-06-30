# Request Binding

Gapi binds request data into typed input structs.

```go
type In struct {
	ID      int    `path:"id"`
	Limit   int    `query:"limit" default:"20"`
	TraceID string `header:"X-Trace-ID"`
	Session string `cookie:"session"`
	Body    Body   `body:""`
}
```

Supported sources:

- `path`
- `query`
- `header`
- `cookie`
- `body` for JSON bodies

Forms and file uploads are roadmap items.
