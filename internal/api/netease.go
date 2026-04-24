package api

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ropean/muze/internal/models"
)

const (
	neteaseBase = "https://music.163.com"

	// AES encryption constants (from Meting v1.5.11)
	neteaseNonce = "0CoJUm6Qyw8W8jud"
	neteaseIV    = "0102030405060708"

	// RSA 1024-bit public key (decimal, from Meting v1.5.11)
	neteaseRSAMod = "157794750267131502212476817800345498121872783333389747424011531025366277535262539913701806290766479189477533597854989606803194253978660329941980786072432806427833685472618792592200595694346872951301770580765135349259590167490536138082469680638514416594216629258349130257685001248172188325316586707301643237607"
	neteaseRSAExp = 65537

	// iPhone client simulation (from Meting curlset() — Netease case)
	neteaseCookie = "appver=8.2.30; os=iPhone OS; osver=15.0; EVNSM=1.0.0; buildver=2206; channel=distribution; machineid=iPhone13.3"
	neteaseUA     = "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 CloudMusic/0.1.1 NeteaseMusic/8.2.30"

	// X-Real-IP random range (112.31.0.0 – 112.31.255.255)
	neteaseIPMin = 1884815360
	neteaseIPMax = 1884890111

	// CGG fallback API for URL resolution
	neteaseCGGBase = "https://api-v2.cenguigui.cn/api/netease/music_v1.php"
)

// Netease implements MusicSource and LyricsSource for 网易云音乐.
type Netease struct {
	client *http.Client
	rng    *rand.Rand
}

// NewNetease creates a Netease API client.
func NewNetease() *Netease {
	return &Netease{
		client: &http.Client{Timeout: 15 * time.Second},
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Name returns the source identifier.
func (n *Netease) Name() string { return "netease" }

// searchPage fetches a single page of raw search results.
// It tries /api/cloudsearch/pc first (returns picUrl directly), then falls back
// to the older /api/search/get which works in more restricted environments.
func (n *Netease) searchPage(keyword string, page, perPage int) (songs []models.Song, songCount int, hasMore bool, err error) {
	params := url.Values{}
	params.Set("s", keyword)
	params.Set("type", "1")
	params.Set("limit", strconv.Itoa(perPage))
	params.Set("offset", strconv.Itoa((page-1)*perPage))

	// Try the richer cloudsearch endpoint first (returns picUrl in al field).
	if songs, songCount, hasMore, err = n.searchCloudSearch(params); err == nil {
		return
	}
	// Fall back to the older, more permissive endpoint.
	return n.searchLegacy(params)
}

// searchCloudSearch uses /api/cloudsearch/pc with browser headers.
// Returns picUrl directly; uses browser UA to avoid DENY responses.
func (n *Netease) searchCloudSearch(params url.Values) ([]models.Song, int, bool, error) {
	body, err := n.browserGet("/api/cloudsearch/pc?" + params.Encode())
	if err != nil {
		return nil, 0, false, err
	}

	type qualityInfo struct {
		BR   int `json:"br"`
		Size int `json:"size"`
	}
	var resp struct {
		Result struct {
			Songs []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Ar   []struct {
					Name string `json:"name"`
				} `json:"ar"`
				Al struct {
					Name   string `json:"name"`
					PicURL string `json:"picUrl"`
					PicStr string `json:"pic_str"`
				} `json:"al"`
				Sq *qualityInfo `json:"sq"` // lossless (FLAC)
				H  *qualityInfo `json:"h"`  // high (320k MP3)
				L  *qualityInfo `json:"l"`  // low (128k MP3)
			} `json:"songs"`
			SongCount int `json:"songCount"`
		} `json:"result"`
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, false, fmt.Errorf("netease cloudsearch decode: %w", err)
	}
	if resp.Code != 200 {
		return nil, 0, false, fmt.Errorf("netease cloudsearch api error: code=%d", resp.Code)
	}

	page, _ := strconv.Atoi(params.Get("offset"))
	perPage, _ := strconv.Atoi(params.Get("limit"))
	offset := page // offset = (page-1)*perPage, already encoded in params

	out := make([]models.Song, 0, len(resp.Result.Songs))
	for _, s := range resp.Result.Songs {
		artists := make([]string, len(s.Ar))
		for i, a := range s.Ar {
			artists[i] = a.Name
		}
		idStr := strconv.Itoa(s.ID)

		// Pick best available quality for metadata display (sq > h > l).
		var br, size int
		switch {
		case s.Sq != nil && s.Sq.BR > 0:
			br, size = s.Sq.BR, s.Sq.Size
		case s.H != nil && s.H.BR > 0:
			br, size = s.H.BR, s.H.Size
		case s.L != nil && s.L.BR > 0:
			br, size = s.L.BR, s.L.Size
		}

		out = append(out, models.Song{
			Title:   s.Name,
			Artist:  strings.Join(artists, " / "),
			Album:   strings.TrimSpace(s.Al.Name),
			Source:  "netease",
			URLID:   idStr,
			PicID:   s.Al.PicStr,
			PicURL:  s.Al.PicURL,
			LyricID: idStr,
			BR:      br,
			Size:    size,
		})
	}

	fetched := offset + len(out)
	_ = perPage
	realHasMore := len(out) > 0 && fetched < resp.Result.SongCount

	return out, resp.Result.SongCount, realHasMore, nil
}

// searchLegacy uses the older /api/search/get endpoint with iPhone headers.
// This is more permissive than cloudsearch/pc but doesn't return picUrl.
func (n *Netease) searchLegacy(params url.Values) ([]models.Song, int, bool, error) {
	body, err := n.get("/api/search/get?" + params.Encode())
	if err != nil {
		return nil, 0, false, fmt.Errorf("netease search: %w", err)
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
					Name  string `json:"name"`
					PicID int64  `json:"picId"`
				} `json:"album"`
			} `json:"songs"`
			SongCount int  `json:"songCount"`
			HasMore   bool `json:"hasMore"`
		} `json:"result"`
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, false, fmt.Errorf("netease search decode: %w", err)
	}
	if resp.Code != 200 {
		return nil, 0, false, fmt.Errorf("netease search api error: code=%d", resp.Code)
	}

	offset, _ := strconv.Atoi(params.Get("offset"))
	perPage, _ := strconv.Atoi(params.Get("limit"))

	out := make([]models.Song, 0, len(resp.Result.Songs))
	for _, s := range resp.Result.Songs {
		artists := make([]string, len(s.Artists))
		for i, a := range s.Artists {
			artists[i] = a.Name
		}
		idStr := strconv.Itoa(s.ID)
		out = append(out, models.Song{
			Title:   s.Name,
			Artist:  strings.Join(artists, " / "),
			Album:   strings.TrimSpace(s.Album.Name),
			Source:  "netease",
			URLID:   idStr,
			PicID:   strconv.FormatInt(s.Album.PicID, 10),
			LyricID: idStr,
		})
	}

	_ = perPage
	fetched := offset + len(out)
	realHasMore := len(out) > 0 && fetched < resp.Result.SongCount

	return out, resp.Result.SongCount, realHasMore, nil
}

// Search queries Netease Cloud Music for the given keyword.
func (n *Netease) Search(keyword string, opts SearchOptions) ([]models.Song, int, bool, error) {
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.Page == 0 {
		opts.Page = 1
	}
	return n.searchPage(keyword, opts.Page, opts.PerPage)
}

// GetURL resolves a playable download URL for the given song ID.
// It tries the official weapi first (with requested quality), then falls back
// to the CGG third-party API if the official API returns an empty URL.
func (n *Netease) GetURL(id string, opts ...URLOptions) (models.URLResult, error) {
	var o URLOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	br := neteasebrForQuality(o.Quality)
	payload := map[string]any{
		"ids": []string{id},
		"br":  br,
	}

	body, err := n.weapi("/weapi/song/enhance/player/url", payload)
	if err != nil {
		// weapi network failure — go straight to fallback
		return n.getURLFallback(id, o.Quality)
	}

	var resp struct {
		Data []struct {
			URL string `json:"url"`
			UF  *struct {
				URL string `json:"url"`
			} `json:"uf"`
			Size int `json:"size"`
			BR   int `json:"br"`
			Code int `json:"code"`
		} `json:"data"`
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Code != 200 || len(resp.Data) == 0 {
		return n.getURLFallback(id, o.Quality)
	}

	d := resp.Data[0]
	resolvedURL := d.URL
	if d.UF != nil && d.UF.URL != "" {
		resolvedURL = d.UF.URL
	}
	if resolvedURL == "" {
		// Song blocked/VIP — try fallback before giving up
		return n.getURLFallback(id, o.Quality)
	}

	return models.URLResult{
		URL:     resolvedURL,
		Size:    d.Size,
		BR:      d.BR,
		Quality: o.Quality,
		Source:  "netease",
		ID:      id,
	}, nil
}

// getURLFallback resolves via the CGG third-party API.
func (n *Netease) getURLFallback(id, quality string) (models.URLResult, error) {
	level := neteaseCGGLevel(quality)
	apiURL := fmt.Sprintf("%s?id=%s&level=%s", neteaseCGGBase, url.QueryEscape(id), level)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return models.URLResult{}, fmt.Errorf("netease fallback: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://music.163.com/")

	resp, err := n.client.Do(req)
	if err != nil {
		return models.URLResult{}, fmt.Errorf("netease fallback: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var cgg struct {
		Code int `json:"code"`
		Data struct {
			URL  string `json:"url"`
			BR   int    `json:"br"`
			Size int    `json:"size"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &cgg); err != nil || cgg.Code != 200 || cgg.Data.URL == "" {
		return models.URLResult{}, fmt.Errorf("netease: song id=%s not available (official blocked, fallback failed)", id)
	}

	return models.URLResult{
		URL:     cgg.Data.URL,
		Size:    cgg.Data.Size,
		BR:      cgg.Data.BR,
		Quality: quality,
		Source:  "netease",
		ID:      id,
	}, nil
}

// GetLyrics fetches the LRC-format lyrics for the given song ID.
// This implements the optional LyricsSource interface.
func (n *Netease) GetLyrics(id string) (string, error) {
	params := url.Values{}
	params.Set("id", id)
	params.Set("lv", "-1")
	params.Set("kv", "-1")
	params.Set("tv", "-1")

	body, err := n.get("/api/song/lyric?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("netease lyrics: %w", err)
	}

	var resp struct {
		Lrc struct {
			Lyric string `json:"lyric"`
		} `json:"lrc"`
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("netease lyrics decode: %w", err)
	}
	if resp.Code != 200 {
		return "", fmt.Errorf("netease lyrics api error: code=%d", resp.Code)
	}
	return resp.Lrc.Lyric, nil
}

// neteasebrForQuality maps a quality string to a Netease bitrate value (bps).
func neteasebrForQuality(q string) int {
	switch q {
	case "flac":
		return 999000
	case "128k":
		return 128000
	default: // "320k" or ""
		return 320000
	}
}

// neteaseCGGLevel maps a quality string to a CGG API level name.
func neteaseCGGLevel(q string) string {
	switch q {
	case "flac":
		return "hires"
	case "128k":
		return "standard"
	default:
		return "exhigh"
	}
}

// weapi sends an AES+RSA-encrypted POST — mirrors Meting's netease_AESCBC().
//
// Encryption flow (matching Meting v1.5.11):
//  1. Generate 16-char random hex skey
//  2. AES-128-CBC encrypt JSON body with nonce key → base64 string S1
//  3. AES-128-CBC encrypt S1 bytes with skey → base64 string S2  (params)
//  4. Reverse skey bytes, treat as big-endian int, compute m^65537 mod N (encSecKey)
func (n *Netease) weapi(path string, params map[string]any) ([]byte, error) {
	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	skey := n.randomHex(16)

	enc1, err := aesCBCEncrypt(jsonBody, []byte(neteaseNonce), []byte(neteaseIV))
	if err != nil {
		return nil, err
	}
	b64step1 := base64.StdEncoding.EncodeToString(enc1)

	enc2, err := aesCBCEncrypt([]byte(b64step1), []byte(skey), []byte(neteaseIV))
	if err != nil {
		return nil, err
	}
	encParams := base64.StdEncoding.EncodeToString(enc2)

	encSecKey := neteaseRSA([]byte(skey))

	form := url.Values{}
	form.Set("params", encParams)
	form.Set("encSecKey", encSecKey)

	req, err := http.NewRequest("POST", neteaseBase+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	n.setHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// browserGet performs a GET request with standard browser headers.
// Used for the public search API which rejects CloudMusic app headers.
func (n *Netease) browserGet(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", neteaseBase+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://music.163.com/")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 && body[0] != '{' {
		return nil, fmt.Errorf("unexpected response (DENY?): %.50q", string(body))
	}
	return body, nil
}

// get performs a GET request with Netease iPhone simulation headers.
func (n *Netease) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", neteaseBase+path, nil)
	if err != nil {
		return nil, err
	}
	n.setHeaders(req)

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// setHeaders applies the iPhone CloudMusic client simulation headers.
func (n *Netease) setHeaders(req *http.Request) {
	req.Header.Set("Referer", "https://music.163.com/")
	req.Header.Set("Cookie", neteaseCookie)
	req.Header.Set("User-Agent", neteaseUA)
	req.Header.Set("X-Real-IP", n.randomChineseIP())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,gl;q=0.6,zh-TW;q=0.4")
	req.Header.Set("Connection", "keep-alive")
}

// aesCBCEncrypt encrypts plaintext using AES-128-CBC with PKCS7 padding.
func aesCBCEncrypt(plaintext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plaintext = pkcs7Pad(plaintext, block.BlockSize())
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)
	return ciphertext, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

// neteaseRSA mirrors Meting's RSA section of netease_AESCBC().
func neteaseRSA(skey []byte) string {
	rev := make([]byte, len(skey))
	for i, b := range skey {
		rev[len(skey)-1-i] = b
	}
	m := new(big.Int).SetBytes(rev)
	N, _ := new(big.Int).SetString(neteaseRSAMod, 10)
	e := big.NewInt(neteaseRSAExp)
	result := new(big.Int).Exp(m, e, N)
	return fmt.Sprintf("%0256x", result)
}

func (n *Netease) randomHex(length int) string {
	b := make([]byte, length/2)
	for i := range b {
		b[i] = byte(n.rng.Intn(256))
	}
	return fmt.Sprintf("%x", b)
}

func (n *Netease) randomChineseIP() string {
	ip := int64(neteaseIPMin) + n.rng.Int63n(int64(neteaseIPMax-neteaseIPMin+1))
	return fmt.Sprintf("%d.%d.%d.%d", (ip>>24)&0xff, (ip>>16)&0xff, (ip>>8)&0xff, ip&0xff)
}
