package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ropean/music-dl-cn/internal/models"
)

const (
	neteaseBase = "https://music.163.com"
	neteaseUA   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// Netease implements MusicSource for 网易云音乐.
type Netease struct {
	client *http.Client
}

// NewNetease creates a Netease API client.
func NewNetease() *Netease {
	return &Netease{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Search queries Netease Cloud Music for the given keyword.
func (n *Netease) Search(keyword string, opts SearchOptions) ([]models.Song, error) {
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.Page == 0 {
		opts.Page = 1
	}

	params := url.Values{}
	params.Set("s", keyword)
	params.Set("type", "1")
	params.Set("limit", strconv.Itoa(opts.PerPage))
	params.Set("offset", strconv.Itoa((opts.Page-1)*opts.PerPage))

	body, err := n.get("/api/search/get?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("netease search: %w", err)
	}

	var resp struct {
		Result struct {
			Songs []struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name string `json:"name"`
				} `json:"album"`
				Fee int `json:"fee"`
			} `json:"songs"`
		} `json:"result"`
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("netease search decode: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("netease search api error: code=%d", resp.Code)
	}

	songs := make([]models.Song, 0, len(resp.Result.Songs))
	for _, s := range resp.Result.Songs {
		artists := make([]string, len(s.Artists))
		for i, a := range s.Artists {
			artists[i] = a.Name
		}
		songs = append(songs, models.Song{
			ID:     strconv.Itoa(s.ID),
			Name:   s.Name,
			Artist: strings.Join(artists, " / "),
			Album:  s.Album.Name,
			Source: "netease",
		})
	}
	return songs, nil
}

// GetURL resolves a playable download URL for the given song ID.
func (n *Netease) GetURL(id string) (string, error) {
	params := url.Values{}
	params.Set("id", id)
	params.Set("ids", "["+id+"]")
	params.Set("br", "320000")

	body, err := n.get("/api/song/enhance/player/url?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("netease url: %w", err)
	}

	var resp struct {
		Data []struct {
			URL  string `json:"url"`
			Code int    `json:"code"`
		} `json:"data"`
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("netease url decode: %w", err)
	}
	if resp.Code != 200 {
		return "", fmt.Errorf("netease url api error: code=%d", resp.Code)
	}
	if len(resp.Data) == 0 {
		return "", fmt.Errorf("netease url: no data for id=%s", id)
	}
	if resp.Data[0].Code == 404 || resp.Data[0].URL == "" {
		return "", fmt.Errorf("netease url: song id=%s is not available (may require login or VIP)", id)
	}

	return resp.Data[0].URL, nil
}

// get performs a GET request to the Netease API.
func (n *Netease) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", neteaseBase+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", neteaseUA)
	req.Header.Set("Referer", "https://music.163.com")
	req.Header.Set("Cookie", "appver=8.0.0; os=pc;")

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
