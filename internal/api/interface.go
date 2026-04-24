package api

import "github.com/ropean/muze/internal/models"

// SearchOptions controls search behaviour.
type SearchOptions struct {
	Page    int
	PerPage int
}

// URLOptions controls URL resolution behaviour.
type URLOptions struct {
	Quality string // "flac" | "320k" | "128k" | "" (default: highest available)
}

// MusicSource is the common interface every platform must implement.
type MusicSource interface {
	// Name returns the source identifier string (e.g. "netease").
	Name() string
	// Search returns matched songs plus total count and hasMore flag.
	Search(keyword string, opts ...SearchOptions) (songs []models.Song, total int, hasMore bool, err error)
	// GetURL resolves a full URLResult for the given track ID.
	GetURL(id string, opts ...URLOptions) (models.URLResult, error)
}

// LyricsSource is an optional interface for sources that support lyric fetching.
type LyricsSource interface {
	GetLyrics(id string) (string, error)
}
