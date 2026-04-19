package api

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ropean/music-provider-cn/internal/models"
)

// Registry holds all registered MusicSource implementations and provides
// concurrent multi-source search and URL resolution.
type Registry struct {
	sources map[string]MusicSource
}

// NewRegistry returns a Registry pre-populated with all built-in sources.
func NewRegistry() *Registry {
	r := &Registry{sources: make(map[string]MusicSource)}
	r.Register(NewNetease())
	return r
}

// Register adds a source. Panics if a source with the same name is already registered.
func (r *Registry) Register(s MusicSource) {
	r.sources[s.Name()] = s
}

// Names returns all registered source names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.sources))
	for n := range r.sources {
		names = append(names, n)
	}
	return names
}

// resolve maps a comma-separated sources string to MusicSource slice.
// Empty/missing input returns all registered sources.
func (r *Registry) resolve(sources string) ([]MusicSource, error) {
	if sources == "" {
		all := make([]MusicSource, 0, len(r.sources))
		for _, s := range r.sources {
			all = append(all, s)
		}
		return all, nil
	}
	names := strings.Split(sources, ",")
	out := make([]MusicSource, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		s, ok := r.sources[name]
		if !ok {
			return nil, fmt.Errorf("unknown source: %q", name)
		}
		out = append(out, s)
	}
	return out, nil
}

// SearchRequest carries all parameters for a search call.
type SearchRequest struct {
	Keyword string
	Sources string // comma-separated, empty = all
	Page    int
	Limit   int
}

// Search fans out to all requested sources concurrently and merges results.
func (r *Registry) Search(req SearchRequest) (models.SearchResult, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 30
	}

	srcs, err := r.resolve(req.Sources)
	if err != nil {
		return models.SearchResult{}, err
	}

	type result struct {
		songs   []models.Song
		total   int
		hasMore bool
		err     error
	}

	ch := make(chan result, len(srcs))
	var wg sync.WaitGroup
	for _, s := range srcs {
		wg.Add(1)
		go func(src MusicSource) {
			defer wg.Done()
			songs, total, hasMore, err := src.Search(req.Keyword, SearchOptions{
				Page:    req.Page,
				PerPage: req.Limit,
			})
			ch <- result{songs, total, hasMore, err}
		}(s)
	}
	wg.Wait()
	close(ch)

	var allSongs []models.Song
	totalCount := 0
	hasMore := false
	var firstErr error
	for res := range ch {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		allSongs = append(allSongs, res.songs...)
		totalCount += res.total
		if res.hasMore {
			hasMore = true
		}
	}

	// Return partial results + error if some sources failed but others succeeded.
	if len(allSongs) == 0 && firstErr != nil {
		return models.SearchResult{}, firstErr
	}

	sourceNames := make([]string, len(srcs))
	for i, s := range srcs {
		sourceNames[i] = s.Name()
	}

	return models.SearchResult{
		Songs: allSongs,
		Meta: models.SearchMeta{
			Keyword: req.Keyword,
			Sources: sourceNames,
			Page:    req.Page,
			Limit:   req.Limit,
			Total:   totalCount,
			HasMore: hasMore,
		},
	}, nil
}

// GetURL resolves a URL for the given source and track ID.
func (r *Registry) GetURL(source, id string) (models.URLResult, error) {
	s, ok := r.sources[source]
	if !ok {
		return models.URLResult{}, fmt.Errorf("unknown source: %q", source)
	}
	return s.GetURL(id)
}
