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

	"github.com/ropean/music-dl-cn/internal/models"
)

const (
	neteaseBase = "https://music.163.com"

	// AES encryption constants (from Meting v1.5.11)
	neteaseNonce = "0CoJUm6Qyw8W8jud"
	neteaseIV    = "0102030405060708"

	// RSA 1024-bit public key (decimal, as in Meting PHP source)
	// modulus N (decimal)
	neteaseRSAMod = "157794750267131502212476817800345498121872783333389747424011531025366277535262539913701806290766479189477533597854989606803194253978660329941980786072432806427833685472618792592200595694346872951301770580765135349259590167490536138082469680638514416594216629258349130257685001248172188325316586707301643237607"
	// public exponent e
	neteaseRSAExp = 65537

	// iPhone client simulation (from Meting curlset() — Netease case)
	neteaseCookie = "appver=8.2.30; os=iPhone OS; osver=15.0; EVNSM=1.0.0; buildver=2206; channel=distribution; machineid=iPhone13.3"
	neteaseUA     = "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 CloudMusic/0.1.1 NeteaseMusic/8.2.30"

	// X-Real-IP random range (112.31.0.0 – 112.31.255.255)
	neteaseIPMin = 1884815360
	neteaseIPMax = 1884890111
)

// Netease implements MusicSource for 网易云音乐.
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

// Search queries Netease Cloud Music for the given keyword.
// Uses the public GET /api/search/get endpoint (no encryption needed).
func (n *Netease) Search(keyword string, opts SearchOptions) ([]models.Song, int, bool, error) {
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

	songs := make([]models.Song, 0, len(resp.Result.Songs))
	for _, s := range resp.Result.Songs {
		artists := make([]string, len(s.Artists))
		for i, a := range s.Artists {
			artists[i] = a.Name
		}
		idStr := strconv.Itoa(s.ID)
		songs = append(songs, models.Song{
			Title:   s.Name,
			Artist:  strings.Join(artists, " / "),
			Source:  "netease",
			URLID:   idStr,
			PicID:   strconv.FormatInt(s.Album.PicID, 10),
			LyricID: idStr, // Netease uses song id for lyric lookup
		})
	}
	return songs, resp.Result.SongCount, resp.Result.HasMore, nil
}

// GetURL resolves a playable download URL for the given song ID.
// Uses POST /weapi/song/enhance/player/url with AES+RSA encryption,
// identical to Meting PHP v1.5.11 netease_AESCBC flow.
func (n *Netease) GetURL(id string) (models.URLResult, error) {
	payload := map[string]any{
		"ids": []string{id},
		"br":  320000,
	}

	body, err := n.weapi("/weapi/song/enhance/player/url", payload)
	if err != nil {
		return models.URLResult{}, fmt.Errorf("netease url: %w", err)
	}

	var resp struct {
		Data []struct {
			URL  string `json:"url"`
			UF   *struct {
				URL string `json:"url"`
			} `json:"uf"`
			Size int `json:"size"`
			BR   int `json:"br"`
			Code int `json:"code"`
		} `json:"data"`
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return models.URLResult{}, fmt.Errorf("netease url decode: %w", err)
	}
	if resp.Code != 200 {
		return models.URLResult{}, fmt.Errorf("netease url api error: code=%d", resp.Code)
	}
	if len(resp.Data) == 0 {
		return models.URLResult{}, fmt.Errorf("netease url: no data returned for id=%s", id)
	}

	d := resp.Data[0]
	// Meting checks uf.url first (alternate URL), then falls back to url
	resolvedURL := d.URL
	if d.UF != nil && d.UF.URL != "" {
		resolvedURL = d.UF.URL
	}
	if resolvedURL == "" {
		return models.URLResult{}, fmt.Errorf("netease url: song id=%s is not available (code=%d)", id, d.Code)
	}
	return models.URLResult{
		URL:    resolvedURL,
		Size:   d.Size,
		BR:     d.BR,
		Source: "netease",
		ID:     id,
	}, nil
}

// weapi sends an AES+RSA-encrypted POST — mirrors Meting's netease_AESCBC().
//
// Encryption flow (exactly matching Meting PHP v1.5.11):
//  1. Generate 16-char random hex skey (= bin2hex(random_bytes(8)))
//  2. AES-128-CBC encrypt JSON body with nonce key → base64 string S1
//  3. AES-128-CBC encrypt S1 bytes with skey → base64 string S2  (params)
//  4. Reverse skey bytes, treat as big-endian int, compute m^65537 mod N (encSecKey)
func (n *Netease) weapi(path string, params map[string]any) ([]byte, error) {
	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	skey := n.randomHex(16) // 16 hex chars = 16 ASCII bytes (matches PHP getRandomHex(16))

	// Step 1: AES-CBC with fixed nonce → raw bytes → base64
	enc1, err := aesCBCEncrypt(jsonBody, []byte(neteaseNonce), []byte(neteaseIV))
	if err != nil {
		return nil, err
	}
	b64step1 := base64.StdEncoding.EncodeToString(enc1)

	// Step 2: AES-CBC with skey, encrypting the base64 string from step 1 → raw bytes → base64
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

// setHeaders applies the iPhone CloudMusic client simulation headers,
// matching Meting PHP curlset() for the netease case.
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

// neteaseRSA mirrors Meting PHP's RSA section of netease_AESCBC():
//
//	strrev → str2hex → bchexdec → bcpowmod(skey, 65537, N) → bcdechex → str_pad 256
func neteaseRSA(skey []byte) string {
	// reverse bytes (PHP: strrev)
	rev := make([]byte, len(skey))
	for i, b := range skey {
		rev[len(skey)-1-i] = b
	}

	// treat reversed bytes as big-endian big integer
	// equivalent to PHP: bchexdec(str2hex(reversed)) for ASCII bytes
	m := new(big.Int).SetBytes(rev)

	N, _ := new(big.Int).SetString(neteaseRSAMod, 10) // decimal modulus from PHP
	e := big.NewInt(neteaseRSAExp)
	result := new(big.Int).Exp(m, e, N)

	// pad to 256 hex chars (128 bytes = 1024-bit RSA output), matches PHP str_pad($s, 256, '0', STR_PAD_LEFT)
	return fmt.Sprintf("%0256x", result)
}

// randomHex returns n random lowercase hex characters, matching PHP getRandomHex(n):
//
//	bin2hex(random_bytes(n/2))
func (n *Netease) randomHex(length int) string {
	b := make([]byte, length/2)
	for i := range b {
		b[i] = byte(n.rng.Intn(256))
	}
	return fmt.Sprintf("%x", b)
}

// randomChineseIP returns a random IP from range 112.31.0.0–112.31.255.255,
// matching PHP: long2ip(mt_rand(1884815360, 1884890111)).
func (n *Netease) randomChineseIP() string {
	ip := int64(neteaseIPMin) + n.rng.Int63n(int64(neteaseIPMax-neteaseIPMin+1))
	return fmt.Sprintf("%d.%d.%d.%d", (ip>>24)&0xff, (ip>>16)&0xff, (ip>>8)&0xff, ip&0xff)
}
