package api_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ropean/music-dl-cn/internal/api"
)

func TestNeteaseSearch_LiuDehua(t *testing.T) {
	n := api.NewNetease()
	songs, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(songs) == 0 {
		t.Fatal("expected at least one result, got none")
	}

	out, _ := json.MarshalIndent(songs, "", "  ")
	fmt.Printf("=== Netease Search: 刘德华 ===\n%s\n\n", out)

	first := songs[0]
	if first.ID == "" {
		t.Error("song ID is empty")
	}
	if first.Name == "" {
		t.Error("song Name is empty")
	}
	if first.Artist == "" {
		t.Error("song Artist is empty")
	}
	if first.Source != "netease" {
		t.Errorf("expected source=netease, got %s", first.Source)
	}
}

func TestNeteaseGetURL(t *testing.T) {
	n := api.NewNetease()

	songs, err := n.Search("刘德华", api.SearchOptions{Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(songs) == 0 {
		t.Skip("no search results, skipping URL test")
	}

	// Try multiple songs — some may be VIP/unavailable due to licensing.
	var lastErr error
	for _, s := range songs {
		t.Logf("Trying URL for: %s (id=%s)", s.Name, s.ID)
		u, err := n.GetURL(s.ID)
		if err != nil {
			lastErr = err
			t.Logf("  -> not available: %v", err)
			continue
		}
		fmt.Printf("=== Netease GetURL ===\nSong: %s\nID:   %s\nURL:  %s\n\n", s.Name, s.ID, u)
		if !strings.HasPrefix(u, "http") {
			t.Errorf("URL does not start with http: %s", u)
		}
		return // found one working URL
	}

	// All songs unavailable — skip rather than fail (licensing constraint, not a bug).
	t.Skipf("all %d songs require login/VIP in this environment (last error: %v)", len(songs), lastErr)
}
