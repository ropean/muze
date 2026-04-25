package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/ropean/muze/internal/api"
	"github.com/ropean/muze/internal/config"
	"github.com/ropean/muze/internal/downloader"
	"github.com/ropean/muze/internal/models"

)

const downloadWorkers = 3

func runInteractive(keyword, dir, theme, quality string) error {
	cfg, _ := config.Load()

	// CLI flags override saved config; non-empty flag values are persisted.
	changed := false
	if theme != "" && theme != cfg.Theme {
		cfg.Theme = theme
		changed = true
	}
	if dir != "" && dir != cfg.Dir {
		cfg.Dir = dir
		changed = true
	}
	if changed {
		_ = config.Save(cfg)
	}

	// Resolve effective values (flag > config > default).
	effectiveDir := cfg.Dir
	if dir != "" {
		effectiveDir = dir
	}

	huhTheme, pal := resolveTheme(cfg.Theme)

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
			WithTheme(huhTheme).
			Run()
		if err != nil {
			return err
		}
	}

	reg := registry()

	arrow := lipgloss.NewStyle().Foreground(pal.Primary).Bold(true).Render("▶")
	fmt.Fprintf(os.Stderr, "\n%s Searching %q ...\n\n", arrow, keyword)

	result, err := reg.Search(api.SearchRequest{
		Keyword: keyword,
		Page:    1,
		Limit:   30,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	if len(result.Songs) == 0 {
		fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Faint(true).Render("No results found."))
		return nil
	}

	options := buildSongOptions(result.Songs, pal)

	titleStyle := lipgloss.NewStyle().Foreground(pal.Primary).Bold(true)
	var selected []int
	err = huh.NewMultiSelect[int]().
		Title(titleStyle.Render(fmt.Sprintf(
			"Found %d tracks  (space=toggle  ctrl+a=all  enter=confirm)", len(result.Songs)))).
		Options(options...).
		Height(22).
		Value(&selected).
		WithTheme(huhTheme).
		Run()
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Faint(true).Render("No tracks selected."))
		return nil
	}

	songs := make([]models.Song, len(selected))
	for i, idx := range selected {
		songs[i] = result.Songs[idx]
	}

	if effectiveDir == "" {
		effectiveDir = config.DefaultDownloadDir()
	}
	outDir := filepath.Join(effectiveDir, downloader.SanitizeFilename(keyword))

	fmt.Fprintf(os.Stderr, "\n%s Downloading %d track(s) → %s\n\n",
		lipgloss.NewStyle().Foreground(pal.OK).Bold(true).Render("▶"),
		len(songs), outDir)

	results := batchDownload(reg, songs, outDir, quality, pal)
	printSummary(results, pal)

	return nil
}

func buildSongOptions(songs []models.Song, pal Palette) []huh.Option[int] {
	titleStyle  := lipgloss.NewStyle().Foreground(pal.Text).Bold(true)
	artistStyle := lipgloss.NewStyle().Foreground(pal.Primary)
	albumStyle  := lipgloss.NewStyle().Faint(true)
	fmtStyle    := lipgloss.NewStyle().Foreground(pal.OK)
	sizeStyle   := lipgloss.NewStyle().Faint(true)
	naStyle     := lipgloss.NewStyle().Faint(true)

	opts := make([]huh.Option[int], len(songs))
	for i, s := range songs {
		fmtStr := formatLabel(s.BR)

		var meta string
		if fmtStr != "" {
			fmtTrail   := strings.Repeat(" ", 8-len(fmtStr))
			sizePadded := fmt.Sprintf("%5.1f MB", float64(s.Size)/(1024*1024))
			meta = fmtStyle.Render(fmtStr) + fmtTrail + "  " + sizeStyle.Render(sizePadded)
		} else {
			meta = naStyle.Render("--") + strings.Repeat(" ", 6) + "  " + naStyle.Render("    --   ")
		}

		title  := titleStyle.Render(padRight(truncateWidth(s.Title, 20), 20))
		artist := artistStyle.Render(padRight(truncateWidth(s.Artist, 16), 16))
		album  := albumStyle.Render(padRight(truncateWidth(s.Album, 16), 16))

		label := title + "  " + artist + "  " + album + "  " + meta
		opts[i] = huh.NewOption(label, i)
	}
	return opts
}

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

// truncateWidth truncates s to at most maxWidth terminal columns (CJK=2, others=1).
func truncateWidth(s string, maxWidth int) string {
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	const ellipsis = "…"
	ellipsisW := runewidth.StringWidth(ellipsis)
	w := 0
	for i, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > maxWidth-ellipsisW {
			return s[:i] + ellipsis
		}
		w += rw
	}
	return s
}

// padRight pads s with spaces to exactly width terminal columns.
func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

type trackJob struct {
	song models.Song
	idx  int
}

func batchDownload(reg *api.Registry, songs []models.Song, outDir, quality string, pal Palette) []downloader.Result {
	results := make([]downloader.Result, len(songs))
	jobs := make(chan trackJob, len(songs))
	var wg sync.WaitGroup

	okStyle   := lipgloss.NewStyle().Foreground(pal.OK).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(pal.Fail).Bold(true)
	dimStyle  := lipgloss.NewStyle().Faint(true)
	nameStyle := lipgloss.NewStyle().Foreground(pal.Text)

	var mu sync.Mutex

	for w := 0; w < downloadWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				s := job.song

				mu.Lock()
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s  %s",
					job.idx+1, len(songs),
					nameStyle.Render(s.Title),
					dimStyle.Render("— "+s.Artist+" ..."))
				mu.Unlock()

				urlResult, err := reg.GetURL(s.Source, s.URLID, api.URLOptions{Quality: quality})
				if err != nil {
					filename := downloader.DefaultFilename(s.Title, s.Artist, "")
					outPath := filepath.Join(outDir, filename)
					results[job.idx] = downloader.Result{Path: outPath, Err: fmt.Errorf("resolve url: %w", err)}
					mu.Lock()
					fmt.Fprintln(os.Stderr, "  "+failStyle.Render("FAIL")+"  "+dimStyle.Render(err.Error()))
					mu.Unlock()
					continue
				}

				ext := downloader.ExtFromURL(urlResult.URL)
				filename := downloader.DefaultFilename(s.Title, s.Artist, ext)
				outPath := filepath.Join(outDir, filename)

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

func printSummary(results []downloader.Result, pal Palette) {
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

	header   := lipgloss.NewStyle().Foreground(pal.Primary).Bold(true).Underline(true)
	okStyle  := lipgloss.NewStyle().Foreground(pal.OK).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(pal.Fail).Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)

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
					lipgloss.NewStyle().Foreground(pal.Fail).Render("✗"),
					dimStyle.Render(filepath.Base(r.Path)+": "+r.Err.Error()))
			}
		}
	}
}
