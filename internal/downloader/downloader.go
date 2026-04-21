package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProgressFunc is called periodically during download with current and total
// byte counts. total may be -1 if the server did not send Content-Length.
type ProgressFunc func(current, total int64)

// Options controls download behaviour.
type Options struct {
	URL        string
	OutPath    string // full destination path
	Force      bool   // overwrite existing file
	OnProgress ProgressFunc
}

// SanitizeFilename replaces characters that are problematic in file paths.
func SanitizeFilename(name string) string {
	r := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return strings.TrimSpace(r.Replace(name))
}

// DefaultFilename builds "<title> - <artist>.mp3" with sanitised characters.
func DefaultFilename(title, artist string) string {
	title = SanitizeFilename(title)
	artist = SanitizeFilename(artist)
	if artist == "" {
		return title + ".mp3"
	}
	return title + " - " + artist + ".mp3"
}

// Download fetches opts.URL and writes it to opts.OutPath.
// It retries once on transient errors (timeout, 5xx).
func Download(opts Options) error {
	if !opts.Force {
		if _, err := os.Stat(opts.OutPath); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", opts.OutPath)
		}
	}

	if err := os.MkdirAll(filepath.Dir(opts.OutPath), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second)
		}
		lastErr = doDownload(opts)
		if lastErr == nil {
			return nil
		}
		if !isTransient(lastErr) {
			return lastErr
		}
	}
	return fmt.Errorf("download failed after retry: %w", lastErr)
}

func doDownload(opts Options) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(opts.URL)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return &serverError{code: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	f, err := os.Create(opts.OutPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(opts.OutPath)
		}
	}()

	total := resp.ContentLength // -1 if unknown
	var reader io.Reader = resp.Body
	if opts.OnProgress != nil {
		reader = &progressReader{
			r:        resp.Body,
			total:    total,
			onUpdate: opts.OnProgress,
		}
	}

	_, err = io.Copy(f, reader)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return f.Close()
}

type serverError struct {
	code int
}

func (e *serverError) Error() string {
	return fmt.Sprintf("server error: %d", e.code)
}

func isTransient(err error) bool {
	if _, ok := err.(*serverError); ok {
		return true
	}
	if os.IsTimeout(err) {
		return true
	}
	return false
}

// FormatBytes returns a human-readable byte size string (e.g. "1.5 MB").
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMG"[exp])
}

type progressReader struct {
	r        io.Reader
	current  int64
	total    int64
	onUpdate ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.current += int64(n)
	pr.onUpdate(pr.current, pr.total)
	return n, err
}
