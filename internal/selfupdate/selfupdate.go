package selfupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const repo = "ropean/muze"

// Set at build time via -ldflags "-X ...selfupdate.Version=v1.0.0".
var Version = "dev"

type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// LatestRelease fetches the latest release tag from GitHub.
func LatestRelease() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &rel, nil
}

// IsNewer reports whether the remote tag is strictly newer than the local version.
// Returns false when the local build has no version info ("dev").
func IsNewer(remote string) bool {
	if Version == "dev" || Version == "" {
		return false
	}
	return strings.TrimPrefix(remote, "v") != strings.TrimPrefix(Version, "v")
}

// AssetName returns the expected release asset name for the current platform.
func AssetName() string {
	name := fmt.Sprintf("muze-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// DownloadAsset downloads a release asset and returns a temp file path.
func DownloadAsset(tag string) (string, error) {
	asset := AssetName()
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, asset)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d (tag=%s, asset=%s)", resp.StatusCode, tag, asset)
	}

	tmp, err := os.CreateTemp("", "muze-*")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}

// ReplaceBinary atomically replaces the running binary with the file at newPath.
func ReplaceBinary(newPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current binary: %w", err)
	}

	if err := os.Chmod(newPath, 0o755); err != nil {
		return err
	}

	// Rename is atomic on the same filesystem; fall back to copy if needed.
	if err := os.Rename(newPath, exe); err != nil {
		return copyFile(newPath, exe)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
