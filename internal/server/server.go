package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/models"
)

// routeDef describes a single HTTP endpoint.
type routeDef struct {
	method  string
	path    string
	params  []string // display-only param tokens, each becomes its own aligned column
	handler func(*api.Registry) http.HandlerFunc
}

var routeDefs = []routeDef{
	{"GET", "/search", []string{"q=<keyword>", "[page=1]", "[limit=50]", "[sources=netease]"}, handleSearch},
	{"GET", "/url", []string{"source=<src>", "id=<id>", "[quality=flac|320k|128k]"}, handleURL},
	{"GET", "/lyrics", []string{"source=<src>", "id=<id>"}, handleLyrics},
	{"GET", "/health", nil, func(_ *api.Registry) http.HandlerFunc { return handleHealth }},
}

// RouteInfo is a single route summary for display at startup.
type RouteInfo struct {
	Method string
	Path   string
	Params []string
}

// Routes returns the registered endpoints for display.
func Routes() []RouteInfo {
	out := make([]RouteInfo, len(routeDefs))
	for i, r := range routeDefs {
		out[i] = RouteInfo{r.method, r.path, r.params}
	}
	return out
}

// New returns an http.Handler that implements the provider contract.
func New(reg *api.Registry) http.Handler {
	mux := http.NewServeMux()
	for _, r := range routeDefs {
		mux.HandleFunc(r.path, r.handler(reg))
	}
	return mux
}

func handleSearch(reg *api.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		result, err := reg.Search(api.SearchRequest{
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
}

func handleURL(reg *api.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		source := q.Get("source")
		id := q.Get("id")
		if source == "" || id == "" {
			jsonError(w, "missing required parameters: source, id", http.StatusBadRequest)
			return
		}

		result, err := reg.GetURL(source, id, api.URLOptions{Quality: q.Get("quality")})
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	}
}

func handleLyrics(reg *api.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		source := q.Get("source")
		id := q.Get("id")
		if source == "" || id == "" {
			jsonError(w, "missing required parameters: source, id", http.StatusBadRequest)
			return
		}

		lyrics, err := reg.GetLyrics(source, id)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, models.LyricsResult{Lyrics: lyrics, Source: source, ID: id})
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
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
