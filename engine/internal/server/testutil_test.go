package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/server"
	"github.com/welife-os/welife-os/engine/internal/testutil"
)

// newTestApp creates a fully initialized Server backed by a real SQLite
// database and a mock Ollama LLM. Call the returned cleanup function to
// release resources.
func newTestApp(t *testing.T) (*server.Server, func()) {
	t.Helper()
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))

	app, err := server.New(testutil.NewServerConfig(t, ollama.URL))
	if err != nil {
		ollama.Close()
		t.Fatalf("newTestApp: %v", err)
	}

	cleanup := func() {
		_ = app.Shutdown(t.Context())
		ollama.Close()
	}
	return app, cleanup
}

// doJSON sends an HTTP request with an optional JSON body and returns the
// response recorder. body can be nil, a string, []byte, or any
// JSON-serializable value.
func doJSON(t *testing.T, app *server.Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	switch v := body.(type) {
	case nil:
		// no body
	case string:
		reader = strings.NewReader(v)
	case []byte:
		reader = bytes.NewReader(v)
	case io.Reader:
		reader = v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("doJSON: marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	app.Handler().ServeHTTP(rec, req)
	return rec
}

// assertStatus fails the test if the recorder's status code doesn't match want.
func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, want, rec.Body.String())
	}
}

// assertBodyContains fails the test if the response body does not contain substr.
func assertBodyContains(t *testing.T, rec *httptest.ResponseRecorder, substr string) {
	t.Helper()
	if !strings.Contains(rec.Body.String(), substr) {
		t.Fatalf("body %q does not contain %q", rec.Body.String(), substr)
	}
}

// decodeJSON unmarshals the response body into a value of type T.
func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decodeJSON: %v; body: %s", err, rec.Body.String())
	}
	return v
}
