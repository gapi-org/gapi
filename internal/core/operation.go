package core

// Operation stores metadata used by routing and OpenAPI generation.
type Operation struct {
	OperationID  string
	Summary      string
	Description  string
	Tags         []string
	Status       int
	middlewares  []Middleware
	dependencies []DependencyResolver
	security     []string
}

// OperationOption configures route metadata.
type OperationOption func(*Operation)

// OperationID sets the OpenAPI operationId.
func OperationID(id string) OperationOption {
	return func(operation *Operation) {
		operation.OperationID = id
	}
}

// Summary sets the OpenAPI summary.
func Summary(summary string) OperationOption {
	return func(operation *Operation) {
		operation.Summary = summary
	}
}

// Description sets the OpenAPI description.
func Description(description string) OperationOption {
	return func(operation *Operation) {
		operation.Description = description
	}
}

// Tags sets OpenAPI tags.
func Tags(tags ...string) OperationOption {
	return func(operation *Operation) {
		operation.Tags = append([]string(nil), tags...)
	}
}

// Status sets the success status code.
func Status(status int) OperationOption {
	return func(operation *Operation) {
		operation.Status = status
	}
}

// Use adds middleware to a route or group.
func Use(middlewares ...Middleware) OperationOption {
	return func(operation *Operation) {
		operation.middlewares = append(operation.middlewares, middlewares...)
	}
}

// Security adds an OpenAPI security requirement by scheme name.
func Security(names ...string) OperationOption {
	return func(operation *Operation) {
		operation.security = append(operation.security, names...)
	}
}
