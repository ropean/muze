package api

import "github.com/ropean/music-dl-cn/internal/models"

// SearchOptions controls search behaviour.
type SearchOptions struct {
	Page    int
	PerPage int
}

// MusicSource is the common interface every platform must implement.
type MusicSource interface {
	// Name returns the source identifier string (e.g. "netease").
	Name() string
	// Search returns matched songs plus total count and hasMore flag.
	Search(keyword string, opts SearchOptions) (songs []models.Song, total int, hasMore bool, err error)
	// GetURL resolves a full URLResult for the given track ID.
	GetURL(id string) (models.URLResult, error)
}
