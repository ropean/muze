package models

// Song represents a single music track returned from any source.
type Song struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Artist  string `json:"artist"`
	Album   string `json:"album"`
	Source  string `json:"source"`
	URL     string `json:"url,omitempty"`
	Size    string `json:"size,omitempty"`
	Bitrate int    `json:"bitrate,omitempty"`
}

// SearchResult wraps a slice of songs for JSON output.
type SearchResult struct {
	Songs []Song `json:"songs"`
}

// URLResult wraps a single resolved URL for JSON output.
type URLResult struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	URL    string `json:"url"`
}
