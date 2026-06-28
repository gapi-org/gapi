package gapitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type Client struct {
	handler http.Handler
}

type Request struct {
	t       *testing.T
	client  Client
	method  string
	path    string
	headers http.Header
	cookies []*http.Cookie
	body    []byte
	err     error
}

type Response struct {
	t        *testing.T
	Recorder *httptest.ResponseRecorder
}

func New(handler http.Handler) Client {
	return Client{handler: handler}
}

func (client Client) GET(path string) Request {
	return client.request(http.MethodGet, path)
}

func (client Client) POST(path string) Request {
	return client.request(http.MethodPost, path)
}

func (client Client) PATCH(path string) Request {
	return client.request(http.MethodPatch, path)
}

func (client Client) DELETE(path string) Request {
	return client.request(http.MethodDelete, path)
}

func (client Client) request(method, path string) Request {
	return Request{
		client:  client,
		method:  method,
		path:    path,
		headers: http.Header{},
	}
}

func (request Request) Header(name, value string) Request {
	request.headers.Set(name, value)
	return request
}

func (request Request) Cookie(cookie *http.Cookie) Request {
	request.cookies = append(request.cookies, cookie)
	return request
}

func (request Request) JSON(value any) Request {
	body, err := json.Marshal(value)
	if err != nil {
		request.err = err
		return request
	}
	request.body = body
	request.headers.Set("Content-Type", "application/json")
	return request
}

func (request Request) Expect(t *testing.T) Response {
	request.t = t
	if request.err != nil {
		t.Fatalf("build request: %v", request.err)
	}
	httpReq := httptest.NewRequest(request.method, request.path, bytes.NewReader(request.body))
	for name, values := range request.headers {
		for _, value := range values {
			httpReq.Header.Add(name, value)
		}
	}
	for _, cookie := range request.cookies {
		httpReq.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	request.client.handler.ServeHTTP(recorder, httpReq)
	return Response{t: t, Recorder: recorder}
}

func (response Response) Status(status int) Response {
	if response.Recorder.Code != status {
		response.t.Fatalf("expected status %d, got %d with body %s", status, response.Recorder.Code, response.Recorder.Body.String())
	}
	return response
}

func (response Response) Decode(target any) Response {
	if err := json.Unmarshal(response.Recorder.Body.Bytes(), target); err != nil {
		response.t.Fatalf("decode JSON response: %v", err)
	}
	return response
}

func (response Response) BodyString(want string) Response {
	if got := response.Recorder.Body.String(); got != want {
		response.t.Fatalf("expected body %q, got %q", want, got)
	}
	return response
}

func AssertOpenAPIMatchesSnapshot(t *testing.T, handler http.Handler, snapshotPath string) {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected OpenAPI status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	got := recorder.Body.Bytes()
	if os.Getenv("UPDATE_SNAPSHOTS") == "1" {
		if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
			t.Fatalf("create snapshot directory: %v", err)
		}
		if err := os.WriteFile(snapshotPath, got, 0o644); err != nil {
			t.Fatalf("write snapshot: %v", err)
		}
		return
	}

	want, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read snapshot %s: %v", snapshotPath, err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("OpenAPI snapshot mismatch for %s", snapshotPath)
	}
}
