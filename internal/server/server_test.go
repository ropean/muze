package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/server"
)

// allSources is the full set of registered music sources.
var allSources = []string{"netease"}

// lyricsSources are the sources that implement LyricsSource.
var lyricsSources = []string{"netease"}

func newTestServer() *httptest.Server {
	return httptest.NewServer(server.New(api.NewRegistry()))
}

// --- Health ---

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

// --- Search validation (no network required) ---

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

// --- URL validation (no network required) ---

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

func TestURL_WithQualityParam(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/url?source=netease&quality=flac")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("/url?quality without id: expected 400, got %d", resp.StatusCode)
	}
}

// --- Lyrics validation (no network required) ---

func TestLyrics_MissingParams(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	for _, path := range []string{"/lyrics", "/lyrics?source=netease", "/lyrics?id=123"} {
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

func TestLyrics_MethodNotAllowed(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/lyrics?source=netease&id=123", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /lyrics: expected 405, got %d", resp.StatusCode)
	}
}

func TestLyrics_UnsupportedSource(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// kugou does not implement LyricsSource
	resp, err := http.Get(ts.URL + "/lyrics?source=kugou&id=123")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("/lyrics kugou: expected 502, got %d", resp.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error field in response")
	}
}

func TestLyrics_UnknownSource(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/lyrics?source=nosuchsource&id=123")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("/lyrics unknown source: expected 502, got %d", resp.StatusCode)
	}
}

// --- Live tests: default pagination ---

func TestSearch_DefaultPagination(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/search?q=%E5%88%98%E5%BE%B7%E5%8D%8E&sources=netease")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("netease unavailable: got %d", resp.StatusCode)
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

// --- Live tests: per-source search (HTTP path) ---

// songKeys are the required JSON keys in every search result song.
var songKeys = []string{"title", "artist", "source", "url_id", "url", "pic_id", "lyric_id"}

// metaKeys are the required JSON keys in every search result meta object.
var metaKeys = []string{"keyword", "sources", "page", "limit", "total", "hasMore"}

func TestSearch_Live_PerSource(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	for _, src := range allSources {
		src := src
		t.Run(src, func(t *testing.T) {
			resp, err := http.Get(ts.URL + "/search?q=%E5%88%98%E5%BE%B7%E5%8D%8E&sources=" + src + "&limit=3")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Skipf("%s search unavailable: got %d", src, resp.StatusCode)
			}

			var result struct {
				Songs []map[string]any `json:"songs"`
				Meta  map[string]any   `json:"meta"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(result.Songs) == 0 {
				t.Skipf("%s: no songs returned", src)
			}
			for _, key := range songKeys {
				if _, ok := result.Songs[0][key]; !ok {
					t.Errorf("%s song missing required key %q", src, key)
				}
			}
			for _, key := range metaKeys {
				if _, ok := result.Meta[key]; !ok {
					t.Errorf("%s meta missing required key %q", src, key)
				}
			}
			src_, _ := result.Songs[0]["source"].(string)
			if src_ != src {
				t.Errorf("%s: song.source = %q, want %q", src, src_, src)
			}
		})
	}
}

// --- Live tests: per-source URL resolution (HTTP path) ---

// urlKeys are the required JSON keys in every URL resolution response.
var urlKeys = []string{"url", "size", "br", "source", "id"}

func TestURL_Live_PerSource(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	for _, src := range allSources {
		src := src
		t.Run(src, func(t *testing.T) {
			// Step 1: search to get candidate IDs.
			searchResp, err := http.Get(ts.URL + "/search?q=%E5%88%98%E5%BE%B7%E5%8D%8E&sources=" + src + "&limit=10")
			if err != nil {
				t.Fatal(err)
			}
			var searchResult struct {
				Songs []struct {
					URLID  string `json:"url_id"`
					Source string `json:"source"`
				} `json:"songs"`
			}
			_ = json.NewDecoder(searchResp.Body).Decode(&searchResult)
			searchResp.Body.Close()

			if len(searchResult.Songs) == 0 {
				t.Skipf("%s: no search results", src)
			}

			// Step 2: try each song until one resolves.
			for _, song := range searchResult.Songs {
				resp, err := http.Get(fmt.Sprintf("%s/url?source=%s&id=%s", ts.URL, song.Source, song.URLID))
				if err != nil {
					continue
				}
				if resp.StatusCode == http.StatusOK {
					var urlResult map[string]any
					_ = json.NewDecoder(resp.Body).Decode(&urlResult)
					resp.Body.Close()

					for _, key := range urlKeys {
						if _, ok := urlResult[key]; !ok {
							t.Errorf("%s url response missing required key %q", src, key)
						}
					}
					urlStr, _ := urlResult["url"].(string)
					if !strings.HasPrefix(urlStr, "http") {
						t.Errorf("%s: url should start with http, got %q", src, urlStr)
					}
					return
				}
				resp.Body.Close()
			}
			t.Skipf("%s: no playable URL available in this environment", src)
		})
	}
}

// --- Live tests: lyrics via HTTP ---

// lyricsKeys are the required JSON keys in every lyrics response.
var lyricsKeys = []string{"lyrics", "source", "id"}

func TestLyrics_Live(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	for _, src := range lyricsSources {
		src := src
		t.Run(src, func(t *testing.T) {
			// Step 1: search to get a lyric_id.
			searchResp, err := http.Get(ts.URL + "/search?q=%E5%88%98%E5%BE%B7%E5%8D%8E&sources=" + src + "&limit=5")
			if err != nil {
				t.Fatal(err)
			}
			var searchResult struct {
				Songs []struct {
					LyricID string `json:"lyric_id"`
				} `json:"songs"`
			}
			_ = json.NewDecoder(searchResp.Body).Decode(&searchResult)
			searchResp.Body.Close()

			if len(searchResult.Songs) == 0 || searchResult.Songs[0].LyricID == "" {
				t.Skipf("%s: no lyric_id from search", src)
			}

			// Step 2: fetch lyrics.
			resp, err := http.Get(fmt.Sprintf("%s/lyrics?source=%s&id=%s", ts.URL, src, searchResult.Songs[0].LyricID))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Skipf("%s lyrics: got %d", src, resp.StatusCode)
			}

			var result map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&result)

			for _, key := range lyricsKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("%s lyrics response missing required key %q", src, key)
				}
			}
			if result["source"] != src {
				t.Errorf("%s: lyrics source = %q, want %q", src, result["source"], src)
			}
		})
	}
}

// --- Consistency: api.Registry (CLI adapter) == HTTP server ---
//
// Both CLI commands and HTTP endpoints go through api.Registry. This test
// explicitly proves they produce the same JSON shape for search and URL responses.

func TestConsistency_CLIvsHTTP_Search(t *testing.T) {
	reg := api.NewRegistry()
	ts := httptest.NewServer(server.New(reg))
	defer ts.Close()

	// CLI path: call Registry directly (same path as `muze search`).
	cliResult, err := reg.Search(api.SearchRequest{
		Keyword: "刘德华", Sources: "netease", Page: 1, Limit: 3,
	})
	if err != nil {
		t.Skipf("netease unavailable: %v", err)
	}

	// HTTP path: call through the server.
	resp, err := http.Get(ts.URL + "/search?q=%E5%88%98%E5%BE%B7%E5%8D%8E&sources=netease&limit=3")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("HTTP search unavailable: got %d", resp.StatusCode)
	}

	// Compare top-level JSON keys.
	cliBytes, _ := json.Marshal(cliResult)
	var cliMap, httpMap map[string]any
	_ = json.Unmarshal(cliBytes, &cliMap)
	_ = json.NewDecoder(resp.Body).Decode(&httpMap)

	for k := range cliMap {
		if _, ok := httpMap[k]; !ok {
			t.Errorf("HTTP response missing key %q (present in Registry/CLI path)", k)
		}
	}
	for k := range httpMap {
		if _, ok := cliMap[k]; !ok {
			t.Errorf("Registry/CLI response missing key %q (present in HTTP path)", k)
		}
	}

	// Compare song-level keys.
	cliSongs, _ := cliMap["songs"].([]any)
	httpSongs, _ := httpMap["songs"].([]any)
	if len(cliSongs) > 0 && len(httpSongs) > 0 {
		cliSong, _ := cliSongs[0].(map[string]any)
		httpSong, _ := httpSongs[0].(map[string]any)
		for k := range cliSong {
			if _, ok := httpSong[k]; !ok {
				t.Errorf("HTTP song missing key %q (present in Registry/CLI song)", k)
			}
		}
		for k := range httpSong {
			if _, ok := cliSong[k]; !ok {
				t.Errorf("Registry/CLI song missing key %q (present in HTTP song)", k)
			}
		}
	}
}

func TestConsistency_CLIvsHTTP_URL(t *testing.T) {
	reg := api.NewRegistry()
	ts := httptest.NewServer(server.New(reg))
	defer ts.Close()

	// Get a valid song ID via search.
	cliResult, err := reg.Search(api.SearchRequest{
		Keyword: "刘德华", Sources: "netease", Page: 1, Limit: 20,
	})
	if err != nil || len(cliResult.Songs) == 0 {
		t.Skip("netease search unavailable")
	}

	// Find a song whose URL resolves successfully.
	var resolvedID string
	for _, s := range cliResult.Songs {
		if _, err := reg.GetURL("netease", s.URLID, api.URLOptions{}); err == nil {
			resolvedID = s.URLID
			break
		}
	}
	if resolvedID == "" {
		t.Skip("no playable netease URL available in this environment")
	}

	// CLI path.
	cliURL, err := reg.GetURL("netease", resolvedID, api.URLOptions{})
	if err != nil {
		t.Skipf("URL unavailable: %v", err)
	}

	// HTTP path.
	resp, err := http.Get(fmt.Sprintf("%s/url?source=netease&id=%s", ts.URL, resolvedID))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("HTTP URL unavailable: got %d", resp.StatusCode)
	}

	cliBytes, _ := json.Marshal(cliURL)
	var cliMap, httpMap map[string]any
	_ = json.Unmarshal(cliBytes, &cliMap)
	_ = json.NewDecoder(resp.Body).Decode(&httpMap)

	for k := range cliMap {
		if _, ok := httpMap[k]; !ok {
			t.Errorf("HTTP URL response missing key %q (present in Registry/CLI path)", k)
		}
	}
	for k := range httpMap {
		if _, ok := cliMap[k]; !ok {
			t.Errorf("Registry/CLI URL response missing key %q (present in HTTP path)", k)
		}
	}
}
