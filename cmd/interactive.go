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

	fmt.Fprintf(os.Stderr, "\nSearching for %q ...\n\n", keyword)

	result, err := reg.Search(api.SearchRequest{
		Keyword: keyword,
		Page:    1,
		Limit:   30,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	if len(result.Songs) == 0 {
		fmt.Fprintln(os.Stderr, "No results found.")
		return nil
	}

	options := buildSongOptions(result.Songs)

	var selected []int
	err = huh.NewMultiSelect[int]().
		Title(fmt.Sprintf("Found %d tracks (space=toggle, ctrl+a=all, enter=confirm)", len(result.Songs))).
		Options(options...).
		Height(20).
		Value(&selected).
		Run()
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, "No tracks selected.")
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

	fmt.Fprintf(os.Stderr, "\nDownloading %d tracks to %s\n\n", len(songs), outDir)

	results := batchDownload(reg, songs, outDir)
	printSummary(results)

	return nil
}

func buildSongOptions(songs []models.Song) []huh.Option[int] {
	titleStyle := lipgloss.NewStyle().Bold(true)
	artistStyle := lipgloss.NewStyle().Faint(true)
	albumStyle := lipgloss.NewStyle().Faint(true).Italic(true)

	opts := make([]huh.Option[int], len(songs))
	for i, s := range songs {
		label := fmt.Sprintf("%s  %s", titleStyle.Render(s.Title), artistStyle.Render(s.Artist))
		if s.Album != "" {
			label += fmt.Sprintf("  %s", albumStyle.Render("["+s.Album+"]"))
		}
		opts[i] = huh.NewOption(label, i)
	}
	return opts
}

type trackJob struct {
	song models.Song
	idx  int
}

func batchDownload(reg *api.Registry, songs []models.Song, outDir string) []downloader.Result {
	results := make([]downloader.Result, len(songs))
	jobs := make(chan trackJob, len(songs))
	var wg sync.WaitGroup

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle := lipgloss.NewStyle().Faint(true)

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
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s - %s ... ",
					job.idx+1, len(songs), s.Title, s.Artist)
				mu.Unlock()

				urlResult, err := reg.GetURL(s.Source, s.URLID, api.URLOptions{})
				if err != nil {
					res := downloader.Result{Path: outPath, Err: fmt.Errorf("resolve url: %w", err)}
					results[job.idx] = res
					mu.Lock()
					fmt.Fprintln(os.Stderr, failStyle.Render("FAIL")+" "+dimStyle.Render(err.Error()))
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
					fmt.Fprintln(os.Stderr, failStyle.Render("FAIL")+" "+dimStyle.Render(res.Err.Error()))
				} else {
					fmt.Fprintln(os.Stderr, successStyle.Render("OK")+
						" "+dimStyle.Render(fmt.Sprintf("%s in %s",
						downloader.FormatBytes(res.Size), res.Duration.Round(time.Millisecond))))
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

	headerStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle := lipgloss.NewStyle().Faint(true)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, headerStyle.Render("Download Summary"))
	fmt.Fprintf(os.Stderr, "  Total:    %d tracks\n", len(results))
	fmt.Fprintf(os.Stderr, "  Success:  %s\n", okStyle.Render(fmt.Sprintf("%d", success)))
	if fail > 0 {
		fmt.Fprintf(os.Stderr, "  Failed:   %s\n", errStyle.Render(fmt.Sprintf("%d", fail)))
	}
	fmt.Fprintf(os.Stderr, "  Size:     %s\n", downloader.FormatBytes(totalSize))
	fmt.Fprintf(os.Stderr, "  Elapsed:  %s\n", dimStyle.Render(totalDur.Round(time.Millisecond).String()))

	if fail > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, errStyle.Render("Failed tracks:"))
		for _, r := range results {
			if r.Err != nil {
				fmt.Fprintf(os.Stderr, "  - %s: %s\n",
					filepath.Base(r.Path), dimStyle.Render(r.Err.Error()))
			}
		}
	}
}
