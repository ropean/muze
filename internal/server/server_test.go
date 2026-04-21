package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/server"
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
	_ = json.NewDecoder(resp.Body).Decode(&body)
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

func TestSearch_MethodNotAllowed(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/search?q=test", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /search: expected 405, got %d", resp.StatusCode)
	}
}

func TestURL_MethodNotAllowed(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/url?source=netease&id=123", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /url: expected 405, got %d", resp.StatusCode)
	}
}

func TestSearch_DefaultPagination(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/search?q=%E5%88%98%E5%BE%B7%E5%8D%8E&sources=netease")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Meta struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
		} `json:"meta"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Meta.Page != 1 {
		t.Errorf("default page: expected 1, got %d", result.Meta.Page)
	}
	if result.Meta.Limit != 30 {
		t.Errorf("default limit: expected 30, got %d", result.Meta.Limit)
	}
}

func TestURL_Live(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// First search for a song to get a valid ID
	searchResp, err := http.Get(ts.URL + "/search?q=%E5%88%98%E5%BE%B7%E5%8D%8E&sources=netease&limit=5")
	if err != nil {
		t.Fatal(err)
	}
	defer searchResp.Body.Close()

	var searchResult struct {
		Songs []struct {
			URLID  string `json:"url_id"`
			Source string `json:"source"`
		} `json:"songs"`
	}
	_ = json.NewDecoder(searchResp.Body).Decode(&searchResult)
	if len(searchResult.Songs) == 0 {
		t.Skip("no search results, skipping URL test")
	}

	song := searchResult.Songs[0]
	resp, err := http.Get(ts.URL + "/url?source=" + song.Source + "&id=" + song.URLID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// URL may fail for VIP songs, but the endpoint should not error internally
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadGateway {
		t.Errorf("/url: expected 200 or 502, got %d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusOK {
		var urlResult map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&urlResult)
		for _, key := range []string{"url", "size", "br", "source", "id"} {
			if _, ok := urlResult[key]; !ok {
				t.Errorf("url response missing key %q", key)
			}
		}
		url, _ := urlResult["url"].(string)
		if !strings.HasPrefix(url, "http") {
			t.Errorf("url should start with http, got %q", url)
		}
	}
}

func TestHealth_ResponseBody(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("health status: expected 'ok', got %q", body["status"])
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
