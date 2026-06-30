# Validation

Gapi validates inputs after binding and before calling the handler.

```go
type CreateTodoBody struct {
	Title string `json:"title" validate:"required,min=1,max=120"`
	Email string `json:"email" validate:"email"`
	Role  string `json:"role" validate:"oneof=admin user viewer"`
}
```

Supported rules:

- `required`
- `min`
- `max`
- `len`
- `email`
- `uuid`
- `oneof`
- `regexp`
- `enum`

Validation errors return `422 application/problem+json` with field-level errors.

Types can also implement `ValidateGapi() []gapi.FieldError` for custom validation.
