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
		result, err := n.GetURL(s.URLID, api.URLOptions{})
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

func TestNeteaseGetURL_WithQuality(t *testing.T) {
	n := api.NewNetease()
	songs, _, _, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 20})
	if err != nil {
		t.Skipf("search unavailable: %v", err)
	}
	if len(songs) == 0 {
		t.Skip("no search results")
	}

	for _, s := range songs {
		result, err := n.GetURL(s.URLID, api.URLOptions{Quality: "320k"})
		if err != nil {
			t.Logf("320k not available for %s: %v", s.URLID, err)
			continue
		}
		if result.URL == "" {
			t.Error("URL is empty")
		}
		if result.Quality != "320k" {
			t.Errorf("expected Quality=320k, got %q", result.Quality)
		}
		return
	}
	t.Skip("no songs with 320k quality available in this environment")
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
	_, err := reg.GetURL("nonexistent", "123", api.URLOptions{})
	if err == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
}

func TestNeteaseGetLyrics_Live(t *testing.T) {
	n := api.NewNetease()
	songs, _, _, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 5})
	if err != nil || len(songs) == 0 {
		t.Skip("search unavailable for lyrics test")
	}

	lyrics, err := n.GetLyrics(songs[0].LyricID)
	if err != nil {
		t.Skipf("lyrics not available: %v", err)
	}
	t.Logf("lyrics length: %d chars", len(lyrics))
}

func TestRegistryGetLyrics_UnknownSource(t *testing.T) {
	reg := api.NewRegistry()
	_, err := reg.GetLyrics("nonexistent", "123")
	if err == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
}

func TestRegistryGetLyrics_UnsupportedSource(t *testing.T) {
	reg := api.NewRegistry()
	// kugou does not implement LyricsSource
	_, err := reg.GetLyrics("kugou", "123")
	if err == nil {
		t.Fatal("expected error for source that does not support lyrics, got nil")
	}
}

func TestURLOptions_DefaultQuality(t *testing.T) {
	opts := api.URLOptions{}
	if opts.Quality != "" {
		t.Errorf("zero URLOptions should have empty Quality, got %q", opts.Quality)
	}
}

func TestSongShape(t *testing.T) {
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
	_ = json.Unmarshal(b, &m)

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
	_ = json.Unmarshal(b, &m)

	for _, key := range []string{"url", "size", "br", "source", "id"} {
		if _, ok := m[key]; !ok {
			t.Errorf("URLResult JSON missing required key: %q", key)
		}
	}
}

func TestNeteaseSearch_PicURL(t *testing.T) {
	n := api.NewNetease()
	songs, _, _, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 10})
	if err != nil {
		t.Skipf("search unavailable: %v", err)
	}
	if len(songs) == 0 {
		t.Skip("no results")
	}
	found := false
	for _, s := range songs {
		if s.PicURL != "" {
			found = true
			if !strings.HasPrefix(s.PicURL, "http") {
				t.Errorf("PicURL does not start with http: %s", s.PicURL)
			}
			break
		}
	}
	if !found {
		t.Error("expected at least one song with non-empty PicURL from cloudsearch")
	}
}

func TestNeteaseSearch_Pagination(t *testing.T) {
	n := api.NewNetease()
	page1, total, _, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 10})
	if err != nil {
		t.Skipf("search unavailable: %v", err)
	}
	if total < 11 {
		t.Skip("not enough results to test pagination")
	}
	page2, _, _, err := n.Search("刘德华", api.SearchOptions{Page: 2, PerPage: 10})
	if err != nil {
		t.Skipf("page 2 unavailable: %v", err)
	}
	if len(page2) == 0 {
		t.Error("page 2 returned no results")
	}
	if len(page1) > 0 && len(page2) > 0 && page1[0].URLID == page2[0].URLID {
		t.Error("page 1 and page 2 returned the same first song")
	}
}

func TestNeteaseGetURL_Flac(t *testing.T) {
	n := api.NewNetease()
	songs, _, _, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 20})
	if err != nil {
		t.Skipf("search unavailable: %v", err)
	}
	for _, s := range songs {
		result, err := n.GetURL(s.URLID, api.URLOptions{Quality: "flac"})
		if err != nil {
			continue
		}
		if result.URL == "" {
			t.Error("URL is empty")
		}
		if result.Quality != "flac" {
			t.Errorf("expected Quality=flac, got %q", result.Quality)
		}
		t.Logf("flac URL obtained for %s (br=%d size=%d)", s.Title, result.BR, result.Size)
		return
	}
	t.Skip("no songs with flac available in this environment")
}

func TestNeteaseGetURL_128k(t *testing.T) {
	n := api.NewNetease()
	songs, _, _, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 20})
	if err != nil {
		t.Skipf("search unavailable: %v", err)
	}
	for _, s := range songs {
		result, err := n.GetURL(s.URLID, api.URLOptions{Quality: "128k"})
		if err != nil {
			continue
		}
		if result.URL == "" {
			t.Error("URL is empty")
		}
		if result.Quality != "128k" {
			t.Errorf("expected Quality=128k, got %q", result.Quality)
		}
		t.Logf("128k URL obtained for %s (br=%d size=%d)", s.Title, result.BR, result.Size)
		return
	}
	t.Skip("no songs with 128k available in this environment")
}

func TestRegistryGetLyrics_Netease(t *testing.T) {
	reg := api.NewRegistry()
	songs, err := reg.Search(api.SearchRequest{Keyword: "刘德华", Sources: "netease", Page: 1, Limit: 5})
	if err != nil || len(songs.Songs) == 0 {
		t.Skip("search unavailable for lyrics registry test")
	}
	lyrics, err := reg.GetLyrics("netease", songs.Songs[0].LyricID)
	if err != nil {
		t.Skipf("lyrics not available via registry: %v", err)
	}
	if lyrics == "" {
		t.Error("lyrics is empty")
	}
	t.Logf("lyrics length: %d chars", len(lyrics))
}
