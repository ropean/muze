package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/config"
	"github.com/ropean/muze/internal/models"
)

// Server holds an http.Handler with a hot-swappable Registry.
// All handlers acquire a read-lock on s.reg so a POST /config reload is safe
// even while other requests are in flight.
type Server struct {
	mu  sync.RWMutex
	reg *api.Registry
	mux *http.ServeMux
}

// New creates a Server with the given initial registry.
func New(reg *api.Registry) *Server {
	s := &Server{reg: reg}
	s.mux = s.buildMux()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) getRegistry() *api.Registry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reg
}

// reload re-reads config.json and atomically replaces the in-memory registry.
func (s *Server) reload() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	reg := api.NewRegistry(api.RegistryOptions{
		NeteaseCookie:    cfg.NeteaseCookie,
		NeteaseCsrf:      cfg.NeteaseCsrf,
		NeteaseCookieRaw: cfg.NeteaseCookieRaw,
	})
	s.mu.Lock()
	s.reg = reg
	s.mu.Unlock()
	return nil
}

func (s *Server) buildMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/url", s.handleURL)
	mux.HandleFunc("/lyrics", s.handleLyrics)
	mux.HandleFunc("/config/cookie", s.handleConfig)
	mux.HandleFunc("/health", handleHealth)
	return mux
}

// RouteInfo describes a single endpoint for display at startup.
type RouteInfo struct {
	Method string
	Path   string
	Params []string
}

// Routes returns all registered endpoints for the startup table.
func Routes() []RouteInfo {
	return []RouteInfo{
		{"GET", "/search", []string{"q=<keyword>", "[page=1]", "[limit=50]", "[sources=netease]"}},
		{"GET", "/url", []string{"source=<src>", "id=<id>", "[quality=flac|320k|128k]"}},
		{"GET", "/lyrics", []string{"source=<src>", "id=<id>"}},
		{"GET", "/config/cookie", []string{"→ returns current cookie"}},
		{"POST", "/config/cookie", []string{`_ntes_nnid=cfdd02b7bc...`, "→ update cookie"}},
		{"GET", "/health", nil},
	}
}

// --- handlers ---

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	keyword := q.Get("q")
	if keyword == "" {
		jsonError(w, "missing required parameter: q", http.StatusBadRequest)
		return
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 30
	}
	result, err := s.getRegistry().Search(api.SearchRequest{
		Keyword: keyword,
		Sources: q.Get("sources"),
		Page:    page,
		Limit:   limit,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, result)
}

func (s *Server) handleURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	source, id := q.Get("source"), q.Get("id")
	if source == "" || id == "" {
		jsonError(w, "missing required parameters: source, id", http.StatusBadRequest)
		return
	}
	result, err := s.getRegistry().GetURL(source, id, api.URLOptions{Quality: q.Get("quality")})
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, result)
}

func (s *Server) handleLyrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	source, id := q.Get("source"), q.Get("id")
	if source == "" || id == "" {
		jsonError(w, "missing required parameters: source, id", http.StatusBadRequest)
		return
	}
	lyrics, err := s.getRegistry().GetLyrics(source, id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, models.LyricsResult{Lyrics: lyrics, Source: source, ID: id})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleConfigGet(w)
	case http.MethodPost:
		s.handleConfigPost(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleConfigGet(w http.ResponseWriter) {
	cfg, _ := config.Load()
	preview := ""
	cookieSet := cfg.NeteaseCookieRaw != ""
	if cookieSet {
		n := len(cfg.NeteaseCookieRaw)
		if n > 40 {
			n = 40
		}
		preview = cfg.NeteaseCookieRaw[:n] + "..."
	}
	writeJSON(w, map[string]any{
		"cookie_set": cookieSet,
		"preview":    preview,
	})
}

func (s *Server) handleConfigPost(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		jsonError(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	var req struct {
		NeteaseCookieRaw string `json:"netease_cookie_raw"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.NeteaseCookieRaw = strings.TrimSpace(req.NeteaseCookieRaw)
	if req.NeteaseCookieRaw == "" {
		jsonError(w, "netease_cookie_raw is required", http.StatusBadRequest)
		return
	}

	cfg, _ := config.Load()
	cfg.NeteaseCookieRaw = req.NeteaseCookieRaw
	cfg.NeteaseCookie = ""
	cfg.NeteaseCsrf = ""
	if err := config.Save(cfg); err != nil {
		jsonError(w, "save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.reload(); err != nil {
		jsonError(w, "reload registry: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{Error: msg})
}
