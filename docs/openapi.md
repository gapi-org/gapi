# OpenAPI And Docs

Gapi generates OpenAPI 3.1 from registered routes, input structs, output structs, and operation metadata.

Built-in endpoints:

- `/openapi.json`
- `/docs`
- `/redoc`
- `/scalar`

Operation metadata:

```go
gapi.Post[CreateUserIn, User](
	app,
	"/users",
	CreateUser,
	gapi.OperationID("createUser"),
	gapi.Summary("Create user"),
	gapi.Description("Creates a user."),
	gapi.Tags("users"),
	gapi.Status(http.StatusCreated),
)
```

Schema generation supports common Go API shapes including structs, pointers/nullability, maps, slices, embedded structs, validation metadata, examples, formats, and `json.RawMessage`.

Recursive schema references and custom schema providers are still roadmap items.
