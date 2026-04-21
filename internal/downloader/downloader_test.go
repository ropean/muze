package downloader

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDefaultFilename(t *testing.T) {
	tests := []struct {
		title, artist, want string
	}{
		{"Hello", "Adele", "Hello - Adele.mp3"},
		{"Song", "", "Song.mp3"},
		{"A/B", "C:D", "A_B - C_D.mp3"},
		{"  spaces  ", "  artist  ", "spaces - artist.mp3"},
	}
	for _, tt := range tests {
		got := DefaultFilename(tt.title, tt.artist)
		if got != tt.want {
			t.Errorf("DefaultFilename(%q, %q) = %q, want %q", tt.title, tt.artist, got, tt.want)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	input := `a/b\c:d*e?f"g<h>i|j`
	got := SanitizeFilename(input)
	if strings.ContainsAny(got, `/\:*?"<>|`) {
		t.Errorf("SanitizeFilename still contains bad chars: %q", got)
	}
}

func TestDownload_Success(t *testing.T) {
	content := "fake mp3 content for testing"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "28")
		_, _ = w.Write([]byte(content))
	}))
	defer srv.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "test.mp3")

	var lastCurrent int64
	err := Download(Options{
		URL:     srv.URL,
		OutPath: outPath,
		OnProgress: func(current, total int64) {
			lastCurrent = current
			if total != 28 {
				t.Errorf("expected total=28, got %d", total)
			}
		},
	})
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content mismatch: got %q", string(data))
	}
	if lastCurrent != 28 {
		t.Errorf("progress final current=%d, want 28", lastCurrent)
	}
}

func TestDownload_FileExists_NoForce(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "exists.mp3")
	_ = os.WriteFile(outPath, []byte("old"), 0o644)

	err := Download(Options{
		URL:     "http://unused",
		OutPath: outPath,
		Force:   false,
	})
	if err == nil {
		t.Fatal("expected error for existing file without --force")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDownload_FileExists_Force(t *testing.T) {
	content := "new content"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer srv.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "exists.mp3")
	_ = os.WriteFile(outPath, []byte("old"), 0o644)

	err := Download(Options{
		URL:     srv.URL,
		OutPath: outPath,
		Force:   true,
	})
	if err != nil {
		t.Fatalf("Download with force failed: %v", err)
	}

	data, _ := os.ReadFile(outPath)
	if string(data) != content {
		t.Errorf("expected overwritten content, got %q", string(data))
	}
}

func TestDownload_CreatesDirectory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "sub", "dir", "test.mp3")

	err := Download(Options{URL: srv.URL, OutPath: outPath})
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestDownload_404_NoRetry(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := Download(Options{
		URL:     srv.URL,
		OutPath: filepath.Join(dir, "test.mp3"),
	})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("404 should not retry, got %d attempts", hits)
	}
}

func TestDownload_500_Retries(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := Download(Options{
		URL:     srv.URL,
		OutPath: filepath.Join(dir, "test.mp3"),
	})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if atomic.LoadInt32(&hits) != 2 {
		t.Errorf("expected 2 attempts, got %d", hits)
	}
}

func TestDownload_500_BothFail(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := Download(Options{
		URL:     srv.URL,
		OutPath: filepath.Join(dir, "test.mp3"),
	})
	if err == nil {
		t.Fatal("expected error when both attempts fail")
	}
	if atomic.LoadInt32(&hits) != 2 {
		t.Errorf("expected 2 attempts for 5xx, got %d", hits)
	}
}

func TestDownload_UnknownContentLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Flush forces chunked transfer encoding so Content-Length is unknown.
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("chunk1"))
		flusher.Flush()
		_, _ = w.Write([]byte("chunk2"))
		flusher.Flush()
	}))
	defer srv.Close()

	dir := t.TempDir()
	var sawNegativeTotal bool
	err := Download(Options{
		URL:     srv.URL,
		OutPath: filepath.Join(dir, "test.mp3"),
		OnProgress: func(current, total int64) {
			if total == -1 {
				sawNegativeTotal = true
			}
		},
	})
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if !sawNegativeTotal {
		t.Error("expected total=-1 for chunked response, but never saw it")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
