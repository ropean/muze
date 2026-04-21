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
| Download + progress | planned |
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
- HTTP streaming, retries

### Phase 5 — interactive CLI
- Full TUI flow where applicable

### Phase 6 — release builds
- Makefile / CI binaries for major OSes
