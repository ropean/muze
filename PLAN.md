# muze roadmap

## Goals

- Single cross-platform binary (Linux / macOS / Windows)
- CLI: search, multi-select, download, JSON-friendly output where applicable
- HTTP `serve` mode for adapter integration

## Layout

```
muze/
├── cmd/
│   └── root.go              # cobra CLI entry
├── internal/
│   ├── api/                 # platform APIs
│   │   ├── interface.go     # MusicSource interface
│   │   ├── netease.go
│   │   ├── kugou.go
│   │   └── tencent.go
│   ├── server/              # HTTP handlers (serve)
│   └── models/
├── main.go
├── go.mod
└── PLAN.md
```

## Dependencies

| Package | Role |
|---------|------|
| `cobra` | CLI commands and flags |
| stdlib `net/http`, `encoding/json`, `crypto/*` | HTTP, JSON, crypto |

## Feature matrix (target)

| Feature | Status |
|---------|--------|
| Search (netease / kugou / tencent) | per-source |
| Concurrent queries | goroutines |
| Download + progress | Phase 4 |
| JSON output (`search` / `url`) | stdout JSON |
| HTTP `/search`, `/url`, `/health` | `serve` |

## JSON examples

```bash
muze search "keyword" --sources netease
muze url netease <id>
```

## Phases

### Phase 1 — skeleton
- Go module, layout, models

### Phase 2 — Netease API
- Search + URL resolution, encryption aligned with Meting v1.5.11 reference

### Phase 3 — Kugou + Tencent
- Same pattern as Netease per platform

### Phase 4 — downloader

CLI command: `muze download <source> <id> [--out <path>]`

#### Workflow

1. Call `GetURL(source, id)` to resolve a playable URL (same as `muze url`)
2. If URL resolution fails → exit with error (the track is unavailable)
3. Stream the file via HTTP GET with progress display
4. Save to `--out` path; default filename: `<title> - <artist>.mp3`

#### Features

| Feature | Detail |
|---------|--------|
| URL resolution | On-demand via `GetURL`; no pre-validation during search |
| Output path | `--out <path>` flag; defaults to `<title> - <artist>.mp3` in cwd |
| Progress bar | Real-time display: percentage, downloaded/total size, speed |
| Retry | 1 retry on transient network errors (timeout, 5xx) |
| Overwrite guard | Skip if file exists; `--force` to overwrite |

#### Implementation

- `internal/downloader/downloader.go` — core download logic (HTTP stream, retry, progress callback)
- `cmd/download.go` — cobra command wiring

#### Default filename

`title` and `artist` come from the search result (`Song` struct). The caller already has
them before requesting a download, so they are passed directly — no extra API call needed.

### Phase 5 — interactive CLI
- Full TUI flow where applicable

### Phase 6 — release builds
- Makefile / CI binaries for major OSes
