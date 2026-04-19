package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ropean/music-provider-cn/internal/api"
	"github.com/ropean/music-provider-cn/internal/server"
)

func newTestServer() *httptest.Server {
	return httptest.NewServer(server.New(api.NewRegistry()))
}

func TestHealth(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("health: expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("health: expected application/json, got %q", ct)
	}
}

func TestSearch_MissingKeyword(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/search")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error field in response")
	}
}

func TestSearch_UnknownSource(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/search?q=test&sources=invalid_source")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for unknown source")
	}
}

func TestURL_MissingParams(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	for _, path := range []string{"/url", "/url?source=netease", "/url?id=123"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", path, resp.StatusCode)
		}
	}
}

func TestSearch_Live(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/search?q=%E5%88%98%E5%BE%B7%E5%8D%8E&sources=netease&limit=3")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Songs []map[string]any `json:"songs"`
		Meta  map[string]any   `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Songs) == 0 {
		t.Error("expected songs in response")
	}
	// Verify required song keys
	for _, key := range []string{"title", "artist", "source", "url_id", "url", "pic_id", "lyric_id"} {
		if _, ok := result.Songs[0][key]; !ok {
			t.Errorf("song missing key %q", key)
		}
	}
	// Verify meta keys
	for _, key := range []string{"keyword", "sources", "page", "limit", "total", "hasMore"} {
		if _, ok := result.Meta[key]; !ok {
			t.Errorf("meta missing key %q", key)
		}
	}
}
