# muze — Design Notes for Claude

## Architecture

### Two modes, one registry

All business logic goes through `api.Registry`. Both CLI and HTTP modes call the same registry methods — the only difference is how results are presented.

- **HTTP mode** (`muze serve`): fast path. Search returns raw results with no URL resolution. Callers resolve URLs on demand via `GET /url`.
- **CLI mode** (`muze` interactive TUI): richer path. Search displays Size and Format from search result metadata. URL is resolved once, right before each download. If resolution fails, the track is marked as failed.

### Search: no URL resolution

Search endpoints and methods must never call `GetURL` internally. Search is a pure metadata operation. This keeps HTTP responses fast and avoids unnecessary upstream API calls.

### CLI: Size and Format from search metadata

Netease's cloudsearch API returns quality metadata (`sq`, `h`, `l`) for each song directly in the search response. The CLI reads Size and BR from these fields to display `Format` (FLAC / 320k / 128k) and `Size (MB)` in the selection list — no extra API call needed.

### CLI: URL resolution at download time

When the user confirms a selection, `GetURL` is called once per track immediately before the download starts. If resolution fails (VIP, geo-blocked, expired), the track is marked as failed in the download summary. There is no pre-flight URL check.

## Sources

Currently only **netease** is registered. Other sources (kugou, kuwo, qq) were explored but their APIs are unreliable from outside China and depend on fragile third-party proxies (cenguigui.cn, vkeys.cn) — not included for now.

## API Compatibility

**Rule: never break existing callers when evolving interfaces.**

- Optional parameters on any interface method must use variadic form: `opts ...FooOptions`. Callers that pass nothing still compile; callers that pass a value still compile. Default values are applied inside the implementation (not the interface): `Search` defaults to Page=1, PerPage=50.
- Adding a new field to an existing options struct is always safe (zero value = old behavior).
- Removing or renaming a method, or changing required parameter types, requires updating every caller in the same commit.

This applies to every future change, not just `GetURL`.

## LyricsSource

`GetLyrics` is an optional interface. Only sources that implement it appear in `/lyrics`. Currently: netease only.
