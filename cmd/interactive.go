package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
			Placeholder("e.g. Jay Chou").
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
	hdrStyle := lipgloss.NewStyle().Faint(true)
	// Offset the header to align with option text (selector + checkbox prefix).
	hdrOffset := strings.Repeat(" ", optionPrefixWidth(cfg.Theme))
	header := hdrStyle.Render(
		hdrOffset + padRight("Title", 20) + "  " + padRight("Artist", 16) +
			"  " + padRight("Album", 16) + "  " + padRight("Format", 8) + "  " + padRight("Size", 8))

	var selected []int
	err = huh.NewMultiSelect[int]().
		Title(titleStyle.Render(fmt.Sprintf(
			"Found %d tracks  (space=toggle  ctrl+a=all  enter=confirm)", len(result.Songs))) +
			"\n" + header).
		Options(options...).
		Height(22).
		Value(&selected).
		WithTheme(huhTheme).
		Run()
	if err != nil {
		return err
	}

	// Expand "Select All" (value = -1) to every song index.
	hasAll := false
	for _, v := range selected {
		if v == -1 {
			hasAll = true
			break
		}
	}
	if hasAll {
		selected = make([]int, len(result.Songs))
		for i := range selected {
			selected[i] = i
		}
	} else {
		// Filter out any stray -1.
		filtered := selected[:0]
		for _, v := range selected {
			if v >= 0 {
				filtered = append(filtered, v)
			}
		}
		selected = filtered
	}

	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Faint(true).Render("No tracks selected."))
		return nil
	}

	songs := make([]models.Song, len(selected))
	for i, idx := range selected {
		songs[i] = result.Songs[idx]
	}

	// Reprint selected tracks so they remain visible during download.
	dimStyle := lipgloss.NewStyle().Faint(true)
	nameStyle := lipgloss.NewStyle().Foreground(pal.Text)
	fmt.Fprintln(os.Stderr)
	for i, s := range songs {
		fmt.Fprintf(os.Stderr, "  %2d.  %s  %s\n", i+1,
			nameStyle.Render(s.Title),
			dimStyle.Render("— "+s.Artist))
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
	titleStyle := lipgloss.NewStyle().Foreground(pal.Text).Bold(true)
	artistStyle := lipgloss.NewStyle().Foreground(pal.Primary)
	albumStyle := lipgloss.NewStyle().Faint(true)
	fmtStyle := lipgloss.NewStyle().Foreground(pal.OK)
	sizeStyle := lipgloss.NewStyle().Faint(true)
	naStyle := lipgloss.NewStyle().Faint(true)
	selectAllStyle := lipgloss.NewStyle().Foreground(pal.Primary).Bold(true)

	opts := make([]huh.Option[int], 0, len(songs)+1)
	opts = append(opts, huh.NewOption(selectAllStyle.Render(padRight("[ ✓ All ]", 20)), -1))

	for i, s := range songs {
		fmtStr := formatLabel(s.BR)

		var meta string
		if fmtStr != "" {
			fmtTrail := strings.Repeat(" ", 8-len(fmtStr))
			sizePadded := fmt.Sprintf("%5.1f MB", float64(s.Size)/(1024*1024))
			meta = fmtStyle.Render(fmtStr) + fmtTrail + "  " + sizeStyle.Render(sizePadded)
		} else {
			meta = naStyle.Render("--") + strings.Repeat(" ", 6) + "  " + naStyle.Render("    --   ")
		}

		title := titleStyle.Render(padRight(truncateWidth(s.Title, 20), 20))
		artist := artistStyle.Render(padRight(truncateWidth(s.Artist, 16), 16))
		album := albumStyle.Render(padRight(truncateWidth(s.Album, 16), 16))

		label := title + "  " + artist + "  " + album + "  " + meta
		opts = append(opts, huh.NewOption(label, i))
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

	failStyle := lipgloss.NewStyle().Foreground(pal.Fail).Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	nameStyle := lipgloss.NewStyle().Foreground(pal.Text)

	// Per-song byte counters (written by download callbacks, read by progress goroutine).
	songBytes := make([]int64, len(songs))
	var completedCount int32

	// Total expected bytes from search metadata for progress percentage.
	var totalExpected int64
	for _, s := range songs {
		totalExpected += int64(s.Size)
	}

	var mu sync.Mutex
	doneCh := make(chan struct{})
	progressGone := make(chan struct{})

	// Aggregated progress bar — refreshes every 150 ms.
	go func() {
		defer close(progressGone)
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-doneCh:
				return
			case <-ticker.C:
				var downloaded int64
				for i := range songBytes {
					downloaded += atomic.LoadInt64(&songBytes[i])
				}
				done := int(atomic.LoadInt32(&completedCount))
				mu.Lock()
				printBar(os.Stderr, done, len(songs), downloaded, totalExpected, pal)
				mu.Unlock()
			}
		}
	}()

	for w := 0; w < downloadWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				s := job.song
				jobIdx := job.idx

				mu.Lock()
				clearBar(os.Stderr)
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s  %s\n",
					jobIdx+1, len(songs),
					nameStyle.Render(s.Title),
					dimStyle.Render("— "+s.Artist+" ..."))
				mu.Unlock()

				urlResult, err := reg.GetURL(s.Source, s.URLID, api.URLOptions{Quality: quality})
				if err != nil {
					filename := downloader.DefaultFilename(s.Title, s.Artist, "")
					outPath := filepath.Join(outDir, filename)
					results[jobIdx] = downloader.Result{Path: outPath, Err: fmt.Errorf("resolve url: %w", err)}
					atomic.AddInt32(&completedCount, 1)
					mu.Lock()
					clearBar(os.Stderr)
					fmt.Fprintf(os.Stderr, "       %s  %s\n",
						failStyle.Render("FAIL"), dimStyle.Render(err.Error()))
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
					OnProgress: func(current, _ int64) {
						atomic.StoreInt64(&songBytes[jobIdx], current)
					},
				})
				results[jobIdx] = res
				atomic.AddInt32(&completedCount, 1)

				if res.Err != nil {
					mu.Lock()
					clearBar(os.Stderr)
					fmt.Fprintf(os.Stderr, "       %s  %s\n",
						failStyle.Render("FAIL"), dimStyle.Render(res.Err.Error()))
					mu.Unlock()
				}
			}
		}()
	}

	for i, s := range songs {
		jobs <- trackJob{song: s, idx: i}
	}
	close(jobs)
	wg.Wait()

	close(doneCh)
	<-progressGone

	// Print final bar at 100% and leave it on screen.
	var finalDownloaded int64
	for i := range songBytes {
		finalDownloaded += atomic.LoadInt64(&songBytes[i])
	}
	mu.Lock()
	printBar(os.Stderr, len(songs), len(songs), finalDownloaded, totalExpected, pal)
	fmt.Fprintln(os.Stderr)
	mu.Unlock()

	return results
}

func printBar(w io.Writer, done, total int, downloaded, expected int64, pal Palette) {
	const barWidth = 30
	pct := 0.0
	if expected > 0 {
		pct = float64(downloaded) / float64(expected)
		if pct > 1 {
			pct = 1
		}
	}
	filled := int(pct * barWidth)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	barStr := lipgloss.NewStyle().Foreground(pal.OK).Render(bar)
	fmt.Fprintf(w, "\r\033[K  [%s] %3.0f%%  %s/%s  (%d/%d done)",
		barStr, pct*100,
		downloader.FormatBytes(downloaded),
		downloader.FormatBytes(expected),
		done, total)
}

func clearBar(w io.Writer) {
	fmt.Fprint(w, "\r\033[K")
}

// optionPrefixWidth returns the visual width of the cursor + checkbox prefix
// that huh prepends to each MultiSelect option row, so the column header can
// be indented to match.  charm uses 2-char prefixes; all other themes (which
// inherit ThemeBase) use 4-char prefixes; selector is always 2 chars wide.
func formatDuration(d time.Duration) string {
	d = d.Round(100 * time.Millisecond)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := float64(d) / float64(time.Second)
	if h > 0 {
		return fmt.Sprintf("%dh%dm%.1fs", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%.1fs", m, s)
	}
	return fmt.Sprintf("%.1fs", s)
}

func optionPrefixWidth(name string) int {
	if name == "charm" {
		return 4 // "> " (2) + "✓ " (2)
	}
	return 6 // "> " (2) + "[•] " (4)
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

	header := lipgloss.NewStyle().Foreground(pal.Primary).Bold(true).Underline(true)
	okStyle := lipgloss.NewStyle().Foreground(pal.OK).Bold(true)
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
	fmt.Fprintf(os.Stderr, "  Elapsed  %s\n", dimStyle.Render(formatDuration(totalDur)))

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
