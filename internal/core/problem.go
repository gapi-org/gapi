package core

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Problem is an RFC 9457 Problem Details response.
type Problem struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"errors,omitempty"`
}

// FieldError describes a field-level binding or validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// HTTPError maps an application error to a Problem Details response.
type HTTPError struct {
	Status int
	Detail string
	Type   string
}

func (err HTTPError) Error() string {
	if err.Detail != "" {
		return err.Detail
	}
	return http.StatusText(err.Status)
}

// NewHTTPError creates an error that handlers can return to control HTTP status.
func NewHTTPError(status int, detail string) error {
	return HTTPError{Status: status, Detail: detail}
}

func problemFromError(err error) (Problem, bool) {
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		return Problem{}, false
	}
	return Problem{
		Type:   httpErr.Type,
		Title:  http.StatusText(httpErr.Status),
		Status: httpErr.Status,
		Detail: httpErr.Detail,
	}, true
}

func writeProblem(w http.ResponseWriter, problem Problem) {
	if problem.Type == "" {
		problem.Type = "about:blank"
	}
	if problem.Title == "" {
		problem.Title = http.StatusText(problem.Status)
	}
	if problem.Status == 0 {
		problem.Status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(problem)
}
