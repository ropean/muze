# music-provider-cn

Go service and CLI for searching Chinese music platforms and resolving playback URLs. It implements the [music-adapter source provider contract](../music-adapter/docs/source-provider-guide.md): HTTP (`serve`) or CLI (`search` / `url`).

## Commands (like npm scripts)

Go uses the `go` toolchain plus a small `Makefile` so you have one place for common workflows (similar to `package.json` scripts in Node):

| Task | Makefile | Plain `go` |
|------|----------|------------|
| Build | `make build` | `go build -o music-provider-cn .` |
| Test | `make test` | `go test -race ./...` |
| Format | `make fmt` | `gofmt -s -w .` and `go fmt ./...` |
| Lint | `make lint` | `go vet ./...` |
| Stricter lint | `make lint-full` | requires [golangci-lint](https://golangci-lint.run/) |

Other common patterns in the ecosystem: [Mage](https://magefile.org/) (Go-based task files), [Task](https://taskfile.dev/) (YAML runner, like Make with nicer syntax), or shell scripts — there is no single built-in `package.json` equivalent; **`go` + `Makefile` is the most common convention**.

## CLI

```bash
go build -o music-provider-cn .
./music-provider-cn search "keyword" [--page N] [--limit N] [--sources netease,tencent]
./music-provider-cn url netease <id>
./music-provider-cn serve [--port 8010]
```

## HTTP

`serve` exposes:

- `GET /search?q=...&page=&limit=&sources=` — each song follows the [contract](../music-adapter/docs/source-provider-guide.md): required ids, title, artist, etc.; **`album`**, and optional **`br` / `size`** when the platform search API exposes bitrate (bps) or file size (bytes). Netease search currently does not, so those fields are omitted; **`GET /url`** always returns `br` and `size` when available.
- `GET /url?source=&id=`
- `GET /health`

## Docker

```bash
docker build -t music-provider-cn .
docker run --rm -p 8010:8010 music-provider-cn
```

## Adapter integration

[music-adapter](../music-adapter/) registers this project twice: channel `music-provider-cn-http` (forwards to this HTTP server) and `music-provider-cn-cli` (spawns the same binary per request). See `music-adapter/docs/channel-registry-guide.md`.
