package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondOK(t *testing.T) {
	w := httptest.NewRecorder()
	respondOK(w, map[string]any{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Msg != "ok" {
		t.Errorf("expected msg 'ok', got '%s'", resp.Msg)
	}
}

func TestRespondErr(t *testing.T) {
	w := httptest.NewRecorder()
	respondErr(w, http.StatusBadRequest, "bad request")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 1 {
		t.Errorf("expected code 1, got %d", resp.Code)
	}
	if resp.Msg != "bad request" {
		t.Errorf("expected msg 'bad request', got '%s'", resp.Msg)
	}
}

func TestParseExchange(t *testing.T) {
	cases := []struct {
		in   string
		want uint8
		err  bool
	}{
		{"sh", 1, false},
		{"sz", 0, false},
		{"bj", 2, false},
		{"SH", 1, false},
		{"xx", 0, true},
	}
	for _, c := range cases {
		ex, err := parseExchange(c.in)
		if c.err {
			if err == nil {
				t.Errorf("parseExchange(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseExchange(%q) unexpected error: %v", c.in, err)
		}
		if ex.Uint8() != c.want {
			t.Errorf("parseExchange(%q) = %d, want %d", c.in, ex.Uint8(), c.want)
		}
	}
}

func TestQueryUint16Default(t *testing.T) {
	req := httptest.NewRequest("GET", "/?count=100", nil)
	if got := queryUint16Default(req, "count", 50); got != 100 {
		t.Errorf("expected 100, got %d", got)
	}
	req2 := httptest.NewRequest("GET", "/", nil)
	if got := queryUint16Default(req2, "count", 50); got != 50 {
		t.Errorf("expected default 50, got %d", got)
	}
}

func TestRotateHosts(test *testing.T) {
	hosts := []string{"host-a", "host-b", "host-c"}

	assertStrings := func(got, want []string) {
		test.Helper()
		if len(got) != len(want) {
			test.Fatalf("rotateHosts() = %#v, want %#v", got, want)
		}
		for index := range want {
			if got[index] != want[index] {
				test.Fatalf("rotateHosts()[%d] = %q, want %q", index, got[index], want[index])
			}
		}
	}

	assertStrings(rotateHosts(hosts, 0), []string{"host-a", "host-b", "host-c"})
	assertStrings(rotateHosts(hosts, 1), []string{"host-b", "host-c", "host-a"})
	assertStrings(rotateHosts(hosts, 4), []string{"host-b", "host-c", "host-a"})
}

func TestHealthAndReadinessRoutes(test *testing.T) {
	apiServer := &Server{}
	mux := http.NewServeMux()
	apiServer.registerRoutes(mux)

	tests := []struct {
		path       string
		wantStatus int
	}{
		{path: "/", wantStatus: http.StatusOK},
		{path: "/ready", wantStatus: http.StatusServiceUnavailable},
		{path: "/unknown", wantStatus: http.StatusNotFound},
	}

	for _, testCase := range tests {
		test.Run(testCase.path, func(test *testing.T) {
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			if recorder.Code != testCase.wantStatus {
				test.Errorf("GET %s status = %d, want %d", testCase.path, recorder.Code, testCase.wantStatus)
			}
		})
	}
}

func TestDocRoute(test *testing.T) {
	apiServer := &Server{}
	mux := http.NewServeMux()
	apiServer.registerRoutes(mux)

	request := httptest.NewRequest(http.MethodGet, "/doc", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		test.Fatalf("GET /doc status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		test.Errorf("GET /doc Content-Type = %q, want %q", got, "text/html; charset=utf-8")
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=300" {
		test.Errorf("GET /doc Cache-Control = %q, want %q", got, "public, max-age=300")
	}
	if !strings.Contains(recorder.Body.String(), "TDX HTTP API Reference") {
		test.Error("GET /doc body is missing API document title")
	}
}
