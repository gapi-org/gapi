package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Response gives handlers control over status, headers, and body.
type Response[T any] struct {
	Status  int
	Headers http.Header
	Body    T
}

type responseValue interface {
	responseStatus() int
	responseHeaders() http.Header
	responseBody() any
}

type responseContentType interface {
	responseContentType() string
}

type responseNoBody interface {
	responseNoBody() bool
}

type responseRedirect interface {
	responseLocation() string
}

type responseWriter interface {
	writeResponse(http.ResponseWriter, int)
}

func (r Response[T]) responseStatus() int {
	return r.Status
}

func (r Response[T]) responseHeaders() http.Header {
	return r.Headers
}

func (r Response[T]) responseBody() any {
	return r.Body
}

// Text returns a plain-text response.
type Text struct {
	Status  int
	Headers http.Header
	Body    string
}

func (r Text) responseStatus() int          { return r.Status }
func (r Text) responseHeaders() http.Header { return r.Headers }
func (r Text) responseBody() any            { return r.Body }
func (r Text) responseContentType() string  { return "text/plain; charset=utf-8" }

// HTML returns an HTML response.
type HTML struct {
	Status  int
	Headers http.Header
	Body    string
}

func (r HTML) responseStatus() int          { return r.Status }
func (r HTML) responseHeaders() http.Header { return r.Headers }
func (r HTML) responseBody() any            { return r.Body }
func (r HTML) responseContentType() string  { return "text/html; charset=utf-8" }

// NoContent returns a response with no body.
type NoContent struct {
	Headers http.Header
}

func (r NoContent) responseStatus() int          { return http.StatusNoContent }
func (r NoContent) responseHeaders() http.Header { return r.Headers }
func (r NoContent) responseBody() any            { return nil }
func (r NoContent) responseNoBody() bool         { return true }

// Redirect returns a redirect response.
type Redirect struct {
	Status   int
	Headers  http.Header
	Location string
}

func (r Redirect) responseStatus() int {
	if r.Status == 0 {
		return http.StatusFound
	}
	return r.Status
}
func (r Redirect) responseHeaders() http.Header { return r.Headers }
func (r Redirect) responseBody() any            { return nil }
func (r Redirect) responseNoBody() bool         { return true }
func (r Redirect) responseLocation() string     { return r.Location }

// File streams a file from disk.
type File struct {
	Status      int
	Headers     http.Header
	Path        string
	ContentType string
}

func (r File) responseStatus() int          { return r.Status }
func (r File) responseHeaders() http.Header { return r.Headers }
func (r File) responseBody() any            { return nil }
func (r File) responseNoBody() bool         { return true }

func (r File) writeResponse(w http.ResponseWriter, defaultStatus int) {
	status := r.Status
	if status == 0 {
		status = defaultStatus
	}
	copyHeaders(w.Header(), r.Headers)
	contentType := r.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	file, err := os.Open(r.Path)
	if err != nil {
		writeProblem(w, Problem{Status: http.StatusNotFound, Detail: "File not found."})
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err == nil {
		w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	}
	w.WriteHeader(status)
	_, _ = io.Copy(w, file)
}

// Attachment streams a file with Content-Disposition: attachment.
type Attachment struct {
	Status      int
	Headers     http.Header
	Path        string
	Filename    string
	ContentType string
}

func (r Attachment) responseStatus() int          { return r.Status }
func (r Attachment) responseHeaders() http.Header { return r.Headers }
func (r Attachment) responseBody() any            { return nil }
func (r Attachment) responseNoBody() bool         { return true }

func (r Attachment) writeResponse(w http.ResponseWriter, defaultStatus int) {
	filename := r.Filename
	if filename == "" {
		filename = r.Path
	}
	headers := r.Headers.Clone()
	if headers == nil {
		headers = http.Header{}
	}
	headers.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(filename, `"`, `\"`)))
	File{
		Status:      r.Status,
		Headers:     headers,
		Path:        r.Path,
		ContentType: r.ContentType,
	}.writeResponse(w, defaultStatus)
}

// Stream writes an arbitrary response stream.
type Stream struct {
	Status      int
	Headers     http.Header
	ContentType string
	Body        io.ReadCloser
}

func (r Stream) responseStatus() int          { return r.Status }
func (r Stream) responseHeaders() http.Header { return r.Headers }
func (r Stream) responseBody() any            { return nil }
func (r Stream) responseNoBody() bool         { return true }

func (r Stream) writeResponse(w http.ResponseWriter, defaultStatus int) {
	status := r.Status
	if status == 0 {
		status = defaultStatus
	}
	copyHeaders(w.Header(), r.Headers)
	if r.ContentType != "" {
		w.Header().Set("Content-Type", r.ContentType)
	}
	w.WriteHeader(status)
	if r.Body == nil {
		return
	}
	defer r.Body.Close()
	_, _ = io.Copy(w, r.Body)
}

// SSEEvent is a single server-sent event.
type SSEEvent struct {
	Event string
	ID    string
	Retry int
	Data  string
}

// SSE writes server-sent events.
type SSE struct {
	Status  int
	Headers http.Header
	Events  []SSEEvent
}

func (r SSE) responseStatus() int          { return r.Status }
func (r SSE) responseHeaders() http.Header { return r.Headers }
func (r SSE) responseBody() any            { return nil }
func (r SSE) responseNoBody() bool         { return true }

func (r SSE) writeResponse(w http.ResponseWriter, defaultStatus int) {
	status := r.Status
	if status == 0 {
		status = defaultStatus
	}
	copyHeaders(w.Header(), r.Headers)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	for _, event := range r.Events {
		writeSSEEvent(w, event)
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func writeOutput(w http.ResponseWriter, out any, defaultStatus int) {
	if response, ok := out.(responseWriter); ok {
		response.writeResponse(w, defaultStatus)
		return
	}

	status := defaultStatus
	headers := http.Header(nil)
	body := out
	contentType := "application/json; charset=utf-8"
	noBody := false

	if response, ok := out.(responseValue); ok {
		if response.responseStatus() != 0 {
			status = response.responseStatus()
		}
		headers = response.responseHeaders()
		body = response.responseBody()
	}
	if response, ok := out.(responseContentType); ok {
		contentType = response.responseContentType()
	}
	if response, ok := out.(responseNoBody); ok {
		noBody = response.responseNoBody()
	}
	if response, ok := out.(responseRedirect); ok && response.responseLocation() != "" {
		w.Header().Set("Location", response.responseLocation())
	}

	for name, values := range headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	if !noBody {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(status)
	if noBody {
		return
	}
	if contentType != "application/json; charset=utf-8" {
		_, _ = w.Write([]byte(body.(string)))
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The response has already started, so there is no useful HTTP recovery path.
		return
	}
}

func copyHeaders(target, source http.Header) {
	for name, values := range source {
		for _, value := range values {
			target.Add(name, value)
		}
	}
}

func writeSSEEvent(w io.Writer, event SSEEvent) {
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	if event.Event != "" {
		fmt.Fprintf(w, "event: %s\n", event.Event)
	}
	if event.Retry > 0 {
		fmt.Fprintf(w, "retry: %d\n", event.Retry)
	}
	for _, line := range strings.Split(event.Data, "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprint(w, "\n")
}
