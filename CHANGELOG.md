# Changelog

All notable changes to this project will be documented in this file.

## [v0.0.3] - 2026-04-24

### Added
- Netease: search via `/api/cloudsearch/pc` to return cover art URL (`pic_url`) directly
- Netease: `GetURL` quality selection (`flac` / `320k` / `128k`)
- Netease: lyrics support via `/api/song/lyric` (LRC format)
- HTTP server: `/lyrics` endpoint
- `url` command: `--quality` flag
- `download` command: `--quality` flag and `--lyrics` flag (saves `.lrc` alongside audio)
- Interactive TUI: search results now show Format (FLAC / MP3 320k / 128k) and Size (MB) columns, extracted from search metadata — no extra API call
- Interactive TUI: columns auto-align with CJK-aware width (Chinese characters = 2 terminal columns)
- Interactive TUI: `--theme` flag to select UI theme (`base16` / `tech` / `charm` / `dracula` / `catppuccin`)
- Interactive TUI: `--dir` and `--theme` values are saved to `config.json` and reused across sessions
- HTTP server: startup now lists all endpoints with auto-aligned columns; adding a route updates the display automatically
- `make serve` target for quick local server start
- OpenAPI 3.1 spec (`openapi.yaml`)

### Fixed
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
