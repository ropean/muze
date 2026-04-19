package models

// Song is one track in a search result, matching the provider contract.
type Song struct {
	Title   string  `json:"title"`
	Artist  string  `json:"artist"`
	Album   string  `json:"album"` // empty if the platform does not expose it
	Source  string  `json:"source"`
	URLID   string  `json:"url_id"`
	URL     *string `json:"url"` // nullable — populated after URL resolution
	PicID   string  `json:"pic_id"`
	LyricID string  `json:"lyric_id"`
	// BR and Size are optional on search: include when the upstream search API exposes them (bps, bytes).
	BR   int `json:"br,omitempty"`
	Size int `json:"size,omitempty"`
}

// SearchMeta holds pagination / context info for a search response.
type SearchMeta struct {
	Keyword string   `json:"keyword"`
	Sources []string `json:"sources"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
	Total   int      `json:"total"`
	HasMore bool     `json:"hasMore"`
}

// SearchResult is the top-level search response envelope.
type SearchResult struct {
	Songs []Song     `json:"songs"`
	Meta  SearchMeta `json:"meta"`
}

// URLResult is the URL resolution response.
type URLResult struct {
	URL    string `json:"url"`
	Size   int    `json:"size"` // bytes
	BR     int    `json:"br"`   // bps
	Source string `json:"source"`
	ID     string `json:"id"`
}

// ErrorResponse is returned on failure.
type ErrorResponse struct {
	Error string `json:"error"`
}
