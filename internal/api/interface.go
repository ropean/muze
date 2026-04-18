package api

import "github.com/ropean/music-dl-cn/internal/models"

// SearchOptions controls search behaviour.
type SearchOptions struct {
	Page    int
	PerPage int
}

// MusicSource is the common interface every platform must implement.
type MusicSource interface {
	Search(keyword string, opts SearchOptions) ([]models.Song, error)
	GetURL(id string) (string, error)
}
