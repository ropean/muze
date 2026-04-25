# Changelog

All notable changes to this project will be documented in this file.

## [v0.0.3] - 2026-04-24

### Added
- Netease: search via `/api/cloudsearch/pc` to return cover art URL (`pic_url`) directly
- Netease: `GetURL` quality selection (`flac` / `320k` / `128k`)
- Netease: VIP quality support via `eapi` endpoint (`/eapi/song/enhance/player/url/v1`) with `level=lossless/exhigh/standard` — required for FLAC download with a browser cookie
- Netease: `__csrf` token auto-extracted from full cookie string when not set separately
- Netease: browser User-Agent used automatically when full cookie string is present
- Netease: lyrics support via `/api/song/lyric` (LRC format)
- HTTP server: `/lyrics` endpoint
- `url` command: `--quality` flag
- `download` command: `--quality` flag and `--lyrics` flag (saves `.lrc` alongside audio)
- `muze config` subcommand: view and update persistent settings (theme, download directory, Netease cookie) via flags or interactive TUI prompts
- `muze config list` subcommand: print current configuration without modifying it
- Interactive TUI: search results now show Format (FLAC / MP3 320k / 128k) and Size (MB) columns, extracted from search metadata — no extra API call
- Interactive TUI: columns auto-align with CJK-aware width (Chinese characters = 2 terminal columns)
- Interactive TUI: `--theme` flag; theme now applies to all CLI output (song list labels, progress lines, download summary) via a per-theme `Palette{Primary, OK, Fail, Text}`
- Interactive TUI: `--dir` and `--theme` values are saved to `config.json` and reused across sessions
- Interactive TUI: column header row displayed above the track list, offset to align with option checkboxes (theme-aware)
- Interactive TUI: `[ ✓ All ]` as first option — selecting it expands to all tracks
- Interactive TUI: confirmed selection reprinted to stderr after the TUI exits, remaining visible during download
- Interactive TUI: aggregated download progress bar (per-song atomic byte counters, 150 ms refresh); bar preserved on screen at completion
- Interactive TUI: elapsed time in download summary formatted with one decimal place (e.g. `3m19.8s`)
- Downloaded files use the correct extension from the resolved URL (`.flac` vs `.mp3`)
- Default download directory is now platform-appropriate (`~/Downloads`) instead of a relative path
- HTTP server: startup now lists all endpoints with auto-aligned columns; adding a route updates the display automatically
- HTTP server: `/config` renamed to `/config/cookie` for clearer semantics
- `make serve` target for quick local server start
- Help flag (`-h, --help`) always appears first in every command's flag list
- OpenAPI 3.1 spec (`openapi.yaml`)

### Fixed
- Config file with UTF-8 BOM (written by PowerShell) now parsed correctly; BOM stripped before JSON decode
- `install.sh`: installed binary now has `.exe` suffix on Windows
- `install.sh`: PATH warning now shows Windows-style path (`C:\...`) on Windows
- `install.sh`: version resolution, default install directory, cross-platform support
- `Makefile`: `build` target now appends `.exe` suffix automatically on Windows

## [v0.0.2] - 2026-04-21

### Added
- Interactive TUI mode (`muze` with no subcommand): multi-select songs, batch download
- `download` command: resolve and download a track by source and ID
- `self-update` command: update the binary in-place from GitHub Releases

### Fixed
- `hasMore` pagination now derived from total count rather than the API flag
- Removed redundant search URL validation

## [v0.0.1] - 2026-04-21

### Added
- Initial release
- Netease Cloud Music search and URL resolution (AES+RSA weapi encryption)
- HTTP server mode (`muze serve`) with `/search`, `/url`, `/health` endpoints
- CLI mode (`muze search`, `muze url`)
- GitHub Actions release workflow (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)

[v0.0.3]: https://github.com/ropean/muze/releases/tag/v0.0.3
[v0.0.2]: https://github.com/ropean/muze/releases/tag/v0.0.2
[v0.0.1]: https://github.com/ropean/muze/releases/tag/v0.0.1
