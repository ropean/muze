package api_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/models"
)

func TestNeteaseSearch_LiuDehua(t *testing.T) {
	n := api.NewNetease()
	songs, total, hasMore, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(songs) == 0 {
		t.Fatal("expected at least one result, got none")
	}

	out, _ := json.MarshalIndent(songs, "", "  ")
	fmt.Printf("=== Netease Search: 刘德华 (total=%d hasMore=%v) ===\n%s\n\n", total, hasMore, out)

	first := songs[0]
	if first.URLID == "" {
		t.Error("song URLID is empty")
	}
	if first.Title == "" {
		t.Error("song Title is empty")
	}
	if first.Artist == "" {
		t.Error("song Artist is empty")
	}
	if first.Source != "netease" {
		t.Errorf("expected source=netease, got %s", first.Source)
	}
	if first.LyricID == "" {
		t.Error("lyric_id is empty")
	}
	if first.Album == "" {
		t.Error("album is empty — netease search should include album name when present")
	}
	if total == 0 {
		t.Error("total should be > 0")
	}
}

func TestNeteaseGetURL(t *testing.T) {
	n := api.NewNetease()

	songs, _, _, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(songs) == 0 {
		t.Skip("no search results, skipping URL test")
	}

	var lastErr error
	for _, s := range songs {
		t.Logf("Trying URL for: %s (id=%s)", s.Title, s.URLID)
		result, err := n.GetURL(s.URLID)
		if err != nil {
			lastErr = err
			t.Logf("  -> not available: %v", err)
			continue
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("=== Netease GetURL ===\n%s\n\n", out)

		if !strings.HasPrefix(result.URL, "http") {
			t.Errorf("URL does not start with http: %s", result.URL)
		}
		if result.Source != "netease" {
			t.Errorf("expected source=netease, got %s", result.Source)
		}
		if result.ID == "" {
			t.Error("id is empty")
		}
		return
	}
	t.Skipf("all %d songs require login/VIP in this environment (last: %v)", len(songs), lastErr)
}

func TestRegistrySearch(t *testing.T) {
	reg := api.NewRegistry()
	result, err := reg.Search(api.SearchRequest{
		Keyword: "刘德华",
		Sources: "netease",
		Page:    1,
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Registry.Search failed: %v", err)
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("=== Registry Search Result ===\n%s\n\n", out)

	if len(result.Songs) == 0 {
		t.Error("expected songs in result")
	}
	if result.Meta.Keyword != "刘德华" {
		t.Errorf("meta.keyword mismatch: %s", result.Meta.Keyword)
	}
	if result.Meta.Page != 1 {
		t.Errorf("meta.page mismatch: %d", result.Meta.Page)
	}
	if result.Meta.Limit != 5 {
		t.Errorf("meta.limit mismatch: %d", result.Meta.Limit)
	}
	if len(result.Meta.Sources) == 0 {
		t.Error("meta.sources is empty")
	}
}

func TestRegistrySearch_UnknownSource(t *testing.T) {
	reg := api.NewRegistry()
	_, err := reg.Search(api.SearchRequest{
		Keyword: "test",
		Sources: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
}

func TestRegistryGetURL_UnknownSource(t *testing.T) {
	reg := api.NewRegistry()
	_, err := reg.GetURL("nonexistent", "123")
	if err == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
}

func TestSongShape(t *testing.T) {
	// Verify Song JSON keys match the provider contract exactly.
	s := models.Song{
		Title:   "Test",
		Artist:  "Artist",
		Album:   "Album",
		Source:  "netease",
		URLID:   "123",
		PicID:   "456",
		LyricID: "123",
		BR:      320000,
		Size:    1000,
	}
	b, _ := json.Marshal(s)
	var m map[string]any
	json.Unmarshal(b, &m)

	for _, key := range []string{"title", "artist", "album", "source", "url_id", "url", "pic_id", "lyric_id", "br", "size"} {
		if _, ok := m[key]; !ok {
			t.Errorf("Song JSON missing required key: %q", key)
		}
	}
}

func TestURLResultShape(t *testing.T) {
	r := models.URLResult{URL: "http://x", Size: 100, BR: 320000, Source: "netease", ID: "1"}
	b, _ := json.Marshal(r)
	var m map[string]any
	json.Unmarshal(b, &m)

	for _, key := range []string{"url", "size", "br", "source", "id"} {
		if _, ok := m[key]; !ok {
			t.Errorf("URLResult JSON missing required key: %q", key)
		}
	}
}
