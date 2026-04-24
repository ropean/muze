package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/downloader"
	"github.com/ropean/muze/internal/models"
)

const downloadWorkers = 3

// Tech color palette
var (
	colorCyan   = lipgloss.Color("#00D7FF")
	colorGreen  = lipgloss.Color("#00FF87")
	colorDim    = lipgloss.Color("#4A4A6A")
	colorWhite  = lipgloss.Color("#E0E0FF")
	colorRed    = lipgloss.Color("#FF5555")
	colorYellow = lipgloss.Color("#FFD700")
)

func runInteractive(keyword, dir string) error {
	if keyword == "" {
		err := huh.NewInput().
			Title("Search keyword").
			Placeholder("e.g. 不想长大").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("keyword cannot be empty")
				}
				return nil
			}).
			Value(&keyword).
			Run()
		if err != nil {
			return err
		}
	}

	reg := api.NewRegistry()

	searchLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("▶")
	fmt.Fprintf(os.Stderr, "\n%s Searching %q ...\n\n", searchLabel, keyword)

	result, err := reg.Search(api.SearchRequest{
		Keyword: keyword,
		Page:    1,
		Limit:   30,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	if len(result.Songs) == 0 {
		fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Foreground(colorYellow).Render("No results found."))
		return nil
	}

	options := buildSongOptions(result.Songs)

	titleStyle := lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	var selected []int
	err = huh.NewMultiSelect[int]().
		Title(titleStyle.Render(fmt.Sprintf("Found %d tracks  (space=toggle  ctrl+a=all  enter=confirm)", len(result.Songs)))).
		Options(options...).
		Height(22).
		Value(&selected).
		Run()
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Foreground(colorDim).Render("No tracks selected."))
		return nil
	}

	songs := make([]models.Song, len(selected))
	for i, idx := range selected {
		songs[i] = result.Songs[idx]
	}

	baseDir := dir
	if baseDir == "" {
		baseDir = filepath.Join(".", "downloads")
	}
	outDir := filepath.Join(baseDir, downloader.SanitizeFilename(keyword))

	fmt.Fprintf(os.Stderr, "\n%s Downloading %d track(s) → %s\n\n",
		lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("▶"),
		len(songs), outDir)

	results := batchDownload(reg, songs, outDir)
	printSummary(results)

	return nil
}

func buildSongOptions(songs []models.Song) []huh.Option[int] {
	titleStyle  := lipgloss.NewStyle().Foreground(colorWhite).Bold(true)
	artistStyle := lipgloss.NewStyle().Foreground(colorCyan)
	albumStyle  := lipgloss.NewStyle().Foreground(colorDim)
	fmtStyle    := lipgloss.NewStyle().Foreground(colorGreen)
	sizeStyle   := lipgloss.NewStyle().Foreground(colorDim)
	naStyle     := lipgloss.NewStyle().Foreground(colorDim)

	opts := make([]huh.Option[int], len(songs))
	for i, s := range songs {
		fmtStr  := formatLabel(s.BR)
		sizeStr := sizeLabel(s.Size)

		var meta string
		if fmtStr != "" {
			meta = fmtStyle.Render(fmtStr) + "  " + sizeStyle.Render(sizeStr)
		} else {
			meta = naStyle.Render("--")
		}

		label := fmt.Sprintf("%-36s  %-20s  %-18s  %s",
			titleStyle.Render(truncate(s.Title, 18)),
			artistStyle.Render(truncate(s.Artist, 12)),
			albumStyle.Render(truncate(s.Album, 12)),
			meta,
		)
		opts[i] = huh.NewOption(label, i)
	}
	return opts
}

// formatLabel returns a human-readable format string derived from bitrate.
func formatLabel(br int) string {
	switch {
	case br >= 800000:
		return "FLAC"
	case br >= 300000:
		return "MP3 320k"
	case br >= 180000:
		return "MP3 192k"
	case br >= 100000:
		return "MP3 128k"
	case br > 0:
		return "MP3"
	default:
		return ""
	}
}

// sizeLabel returns size in MB with one decimal place.
func sizeLabel(bytes int) string {
	if bytes <= 0 {
		return "--"
	}
	mb := float64(bytes) / (1024 * 1024)
	return fmt.Sprintf("%.1f MB", mb)
}

// truncate shortens s to max runes, appending … if cut.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

type trackJob struct {
	song models.Song
	idx  int
}

func batchDownload(reg *api.Registry, songs []models.Song, outDir string) []downloader.Result {
	results := make([]downloader.Result, len(songs))
	jobs := make(chan trackJob, len(songs))
	var wg sync.WaitGroup

	okStyle   := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	dimStyle  := lipgloss.NewStyle().Foreground(colorDim)
	nameStyle := lipgloss.NewStyle().Foreground(colorWhite)

	var mu sync.Mutex

	for w := 0; w < downloadWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				s := job.song
				filename := downloader.DefaultFilename(s.Title, s.Artist)
				outPath := filepath.Join(outDir, filename)

				mu.Lock()
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s  %s",
					job.idx+1, len(songs),
					nameStyle.Render(s.Title),
					dimStyle.Render("— "+s.Artist+" ..."))
				mu.Unlock()

				urlResult, err := reg.GetURL(s.Source, s.URLID, api.URLOptions{})
				if err != nil {
					res := downloader.Result{Path: outPath, Err: fmt.Errorf("resolve url: %w", err)}
					results[job.idx] = res
					mu.Lock()
					fmt.Fprintln(os.Stderr, "  "+failStyle.Render("FAIL")+"  "+dimStyle.Render(err.Error()))
					mu.Unlock()
					continue
				}

				res := downloader.DownloadWithResult(downloader.Options{
					URL:     urlResult.URL,
					OutPath: outPath,
					Force:   true,
				})
				results[job.idx] = res

				mu.Lock()
				if res.Err != nil {
					fmt.Fprintln(os.Stderr, "  "+failStyle.Render("FAIL")+"  "+dimStyle.Render(res.Err.Error()))
				} else {
					fmt.Fprintln(os.Stderr, "  "+okStyle.Render("OK")+"  "+
						dimStyle.Render(fmt.Sprintf("%s  %s",
							downloader.FormatBytes(res.Size),
							res.Duration.Round(time.Millisecond))))
				}
				mu.Unlock()
			}
		}()
	}

	for i, s := range songs {
		jobs <- trackJob{song: s, idx: i}
	}
	close(jobs)
	wg.Wait()

	return results
}

func printSummary(results []downloader.Result) {
	var success, fail int
	var totalSize int64
	var totalDur time.Duration

	for _, r := range results {
		if r.Err != nil {
			fail++
		} else {
			success++
			totalSize += r.Size
			totalDur += r.Duration
		}
	}

	header  := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Underline(true)
	okStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(colorDim)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, header.Render("Download Summary"))
	fmt.Fprintf(os.Stderr, "  Total    %d\n", len(results))
	fmt.Fprintf(os.Stderr, "  Success  %s\n", okStyle.Render(fmt.Sprintf("%d", success)))
	if fail > 0 {
		fmt.Fprintf(os.Stderr, "  Failed   %s\n", errStyle.Render(fmt.Sprintf("%d", fail)))
	}
	fmt.Fprintf(os.Stderr, "  Size     %s\n", dimStyle.Render(downloader.FormatBytes(totalSize)))
	fmt.Fprintf(os.Stderr, "  Elapsed  %s\n", dimStyle.Render(totalDur.Round(time.Millisecond).String()))

	if fail > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, errStyle.Render("Failed:"))
		for _, r := range results {
			if r.Err != nil {
				fmt.Fprintf(os.Stderr, "  %s  %s\n",
					lipgloss.NewStyle().Foreground(colorRed).Render("✗"),
					dimStyle.Render(filepath.Base(r.Path)+": "+r.Err.Error()))
			}
		}
	}
}
