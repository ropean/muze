# HTTP API Changelog

Breaking changes are marked **[BREAKING]**.  
Additive changes (new optional fields / new endpoints) are marked **[ADDITIVE]**.  
Additive changes are safe for existing callers but should be reviewed so they can opt in.

---

## v0.0.3

### `GET /url` — quality selection

**[ADDITIVE]** New optional query parameter: `quality`

| | Before (v0.0.2) | After (v0.0.3) |
|---|---|---|
| `quality` param | not accepted | `flac` \| `320k` \| `128k` (optional, default: `320k`) |

**[ADDITIVE]** New field in response body: `quality`

```jsonc
// v0.0.2 response
{
  "url":    "https://...",
  "size":   9116778,
  "br":     320000,
  "source": "netease",
  "id":     "418603842"
}

// v0.0.3 response — quality field added when requested
{
  "url":     "https://...",
  "size":    9116778,
  "br":      320000,
  "quality": "flac",        // present only when ?quality= was passed; omitted otherwise
  "source":  "netease",
  "id":      "418603842"
}
```

**Quality resolution behaviour:**
- `flac` → attempts lossless via eapi; falls back to 320k if the track has no lossless version or the cookie lacks VIP entitlement
- `320k` → high-quality MP3 (default)
- `128k` → standard-quality MP3
- Any other value or omitted → treated as `320k`

**VIP note:** lossless resolution requires a valid Netease browser cookie with VIP entitlement (`MUSIC_U` + `__csrf`). Configure it via `POST /config/cookie` (see below) or in `config.json`. Without a cookie the response falls back to the highest freely available quality.

---

### `GET /search` — `pic_url` added to song objects

**[ADDITIVE]** New field `pic_url` in each song object within `songs[]`:

```jsonc
// v0.0.2 song object
{
  "title":    "稻香",
  "artist":   "周杰伦",
  "album":    "魔杰座",
  "source":   "netease",
  "url_id":   "186016",
  "url":      null,
  "pic_id":   "...",
  "lyric_id": "186016",
  "br":       320000,
  "size":     9116778
}

// v0.0.3 song object — pic_url added
{
  "title":    "稻香",
  "artist":   "周杰伦",
  "album":    "魔杰座",
  "source":   "netease",
  "url_id":   "186016",
  "url":      null,
  "pic_id":   "...",
  "pic_url":  "http://p2.music.126.net/...jpg",  // direct artwork URL; omitted if unavailable
  "lyric_id": "186016",
  "br":       320000,
  "size":     9116778
}
```

`pic_url` is omitted (field absent, not null) when the upstream API does not return a direct URL for the track. Do not assume presence.

---

### `GET /lyrics` — new endpoint

**[ADDITIVE]** New endpoint.

**Request**

```
GET /lyrics?source=<source>&id=<id>
```

| Parameter | Required | Description |
|---|---|---|
| `source` | yes | Provider name (currently `netease` only) |
| `id` | yes | Track ID (same value as `lyric_id` from `/search`) |

**Response — 200 OK**

```json
{
  "lyrics": "[00:18.210]还记得你说家是唯一的城堡...\n...",
  "source": "netease",
  "id":     "186016"
}
```

`lyrics` is a plain string in LRC format (timestamps + lines separated by `\n`). May be an empty string if the track has no lyrics.

**Error responses**

| Status | Condition |
|---|---|
| 400 | `source` or `id` missing |
| 502 | Upstream API error or track not found |

---

### `GET /config/cookie` — new endpoint

**[ADDITIVE]** New endpoint. Returns whether a Netease cookie is currently active in the running server.

**Request**

```
GET /config/cookie
```

No parameters.

**Response — 200 OK**

```json
{
  "cookie_set": true,
  "preview":    "_ntes_nnid=cfdd02b7bc..."
}
```

| Field | Type | Description |
|---|---|---|
| `cookie_set` | bool | `true` if a non-empty cookie is loaded |
| `preview` | string | First 40 characters of the cookie string followed by `...`; empty string when `cookie_set` is `false` |

---

### `POST /config/cookie` — new endpoint

**[ADDITIVE]** New endpoint. Updates the Netease browser cookie and hot-reloads the registry — no server restart required. The new cookie is also persisted to `config.json` on disk.

**Request**

```
POST /config/cookie
Content-Type: application/json

{
  "netease_cookie_raw": "_ntes_nnid=cfdd02b7bc...; MUSIC_U=...; __csrf=..."
}
```

| Field | Required | Description |
|---|---|---|
| `netease_cookie_raw` | yes | Full browser cookie string. Must be non-empty. Leading/trailing whitespace is trimmed. |

**Response — 200 OK**

```json
{ "status": "ok" }
```

**Error responses**

| Status | Body | Condition |
|---|---|---|
| 400 | `{"error":"invalid JSON: ..."}` | Request body is not valid JSON |
| 400 | `{"error":"netease_cookie_raw is required"}` | Field is missing or blank |
| 500 | `{"error":"save config: ..."}` | Disk write failed |
| 500 | `{"error":"reload registry: ..."}` | In-memory reload failed (cookie written to disk but not yet active) |

**Concurrency guarantee:** the reload is atomic under a write lock. In-flight requests on other endpoints complete against the old registry; requests arriving after the lock is released use the new cookie.

---

## v0.0.2 (reference baseline)

Endpoints available in v0.0.2:

| Method | Path | Notes |
|---|---|---|
| `GET` | `/search` | `q`, `page`, `limit`, `sources` params |
| `GET` | `/url` | `source`, `id` params only; no quality selection |
| `GET` | `/health` | Always `{"status":"ok"}` |

All endpoints introduced in v0.0.3 (`/lyrics`, `/config/cookie`) and all new fields (`quality` on `/url`, `pic_url` on `/search`) were absent in v0.0.2.
