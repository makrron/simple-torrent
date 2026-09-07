# 🔍 Torrent Search Engine & Provider Architecture

This document describes the design, architecture, and extension guidelines for **SimpleTorrent**'s built-in torrent search engine.

---

## 📌 Architecture Overview

SimpleTorrent features a modular, high-performance torrent search engine located in the [`search/`](file:///home/makrron/Documentos/github-repos/simple-torrent/search) package:

```
search/
├── search.go            # Common types (Result, ItemDetail) & embedded file system
├── engine.go            # Search engine manager & HTTP handler for /search routes
├── client.go            # Resilient HTTP client with modern browser headers, TLS, & timeouts
├── extractor_html.go    # HTML parser using goquery (CSS selectors, @attr, regex pipelines)
├── extractor_json.go    # JSON API parser with dot-notation field mapping & auto-magnet builder
├── provider.go          # Provider interface and default provider executor with mirror failover
├── search_test.go       # Comprehensive automated test suite
└── providers/           # Directory with individual JSON provider definitions
    ├── 1337x.json
    ├── audiobookbay.json
    ├── eztv.json
    ├── limetorrents.json
    ├── nyaa.json
    ├── tpb.json
    ├── torrentgalaxy.json
    └── yts.json
```

### Key Capabilities & Modernizations
1. **Zero Monoliths**: Every search site is defined in its own modular file under [`search/providers/<id>.json`](file:///home/makrron/Documentos/github-repos/simple-torrent/search/providers/). Adding a new site requires editing **only one file**.
2. **Compile-time Embedding**: Provider definitions are embedded directly into the Go binary via Go 1.16+ `embed.FS`. No runtime external files are required, but external overrides remain supported via `ScraperURL`.
3. **Dual Engine (HTML + JSON APIs)**:
   - **HTML Web Scraping**: Powered by `goquery` with jQuery-style CSS selectors, attribute extraction, regex extraction, and fallback selectors.
   - **JSON REST APIs**: Direct mapping of structured JSON responses (e.g. YTS / EZTV / Jackett APIs), immune to HTML layout changes and significantly faster.
4. **Automatic Mirror Failover**: Each provider can specify multiple `mirrors`. If a primary mirror fails (HTTP 403, 500, or network timeout), the engine automatically tries alternative mirrors in sequence before reporting failure.
5. **Modern Browser Fingerprint**:
   - Modern rotating desktop User-Agents (Chrome 128+, Firefox 129+).
   - Realistic Client Hints (`sec-ch-ua`, `Sec-Fetch-Dest`, `Sec-Fetch-Mode`, `Accept-Language`).
   - Built-in Gzip/Deflate decompression.
   - InsecureSkipVerify for TLS proxies and self-signed mirror certificates.
   - Safe per-request context timeouts (15s) to avoid UI blocking.
6. **100% Backward Compatible**: Retains complete compatibility with the SimpleTorrent Angular web UI and legacy `scraper-config.json` specifications.

---

## 🛠️ Provider Definition Schema

Each provider definition in [`search/providers/`](file:///home/makrron/Documentos/github-repos/simple-torrent/search/providers/) is a JSON file conforming to the following structure:

### 1. HTML Web Scraping Provider (`type: "html"`)

```json
{
  "id": "tpb",
  "name": "The Pirate Bay",
  "type": "html",
  "url": "https://tpb.party/search/{{query}}/{{page:1}}/7/0",
  "mirrors": [
    "https://tpb.party/search/{{query}}/{{page:1}}/7/0",
    "https://thepiratebay0.org/search/{{query}}/{{page:1}}/7/0",
    "https://pirateproxy.live/search/{{query}}/{{page:1}}/7/0"
  ],
  "list": "#searchResult > tbody > tr",
  "result": {
    "name": [
      "td:nth-child(2) a",
      "a.detLink"
    ],
    "path": [
      "td:nth-child(2) a",
      "@href"
    ],
    "magnet": [
      "a[href^='magnet:']",
      "@href"
    ],
    "size": [
      "td:nth-child(5)",
      "font.detDesc",
      "/Size ([^,]+)/"
    ],
    "seeds": "td:nth-child(6)",
    "peers": "td:nth-child(7)"
  },
  "item_url": "https://tpb.party{{item}}",
  "item_mirrors": [
    "https://tpb.party{{item}}",
    "https://thepiratebay0.org{{item}}"
  ],
  "item_result": {
    "magnet": [
      "a[href^='magnet:']",
      "@href"
    ]
  }
}
```

#### Field Rules Explanation:
* **URL placeholders**:
  - `{{query}}`: URL-encoded search keyword.
  - `{{page:1}}`: 1-based pagination number (`1, 2, 3...`).
  - `{{page:0}}`: 0-based pagination number (`0, 1, 2...`).
* **Selector formats**:
  - **Simple text**: `"td.title a"` extracts `.Text()` trimmed of the element.
  - **Attribute extraction**: `["td.title a", "@href"]` or shorthand `"td.title a@href"`.
  - **Regex capture**: `["font.desc", "/Size ([^,]+)/"]` extracts the first capture group.
  - **Regex substitution**: `[".title", "s/[\\n\\t]+//g"]` replaces matches before returning.
  - **Fallback list**: An array of string selectors, e.g. `["td:nth-child(2) a", "a.detLink"]`. Evaluated in order until a match returns non-empty text.

---

### 2. JSON REST API Provider (`type: "json"`)

```json
{
  "id": "yts",
  "name": "YTS (YIFY)",
  "type": "json",
  "url": "https://yts.mx/api/v2/list_movies.json?query_term={{query}}&page={{page:1}}",
  "mirrors": [
    "https://yts.mx/api/v2/list_movies.json?query_term={{query}}&page={{page:1}}",
    "https://yts.nz/api/v2/list_movies.json?query_term={{query}}&page={{page:1}}"
  ],
  "root": "data.movies",
  "result": {
    "name": "title_long",
    "url": "url",
    "size": "torrents.0.size",
    "seeds": "torrents.0.seeds",
    "peers": "torrents.0.peers",
    "torrent": "torrents.0.url",
    "hash": "torrents.0.hash"
  }
}
```

* **`root`**: Dot-separated path to the JSON array of items (e.g. `data.movies` or `torrents`).
* **`hash`**: When an infohash is specified, SimpleTorrent automatically builds a full `magnet:?xt=urn:btih:...` link complete with standard public trackers.

---

## 🚀 How to Add a New Search Provider in 3 Steps

### Step 1: Create a Provider JSON File
Add a new file in `search/providers/<site_id>.json`. For example, `search/providers/rutracker.json`.

Fill in the site name, search URL structure, row selector, and result fields.

### Step 2: Run Unit Tests
Verify that all embedded providers load and pass automated validation:
```bash
# Using local Go
go test ./search -v

# Using Docker
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine go test ./search -v
```

### Step 3: Build & Test Live in SimpleTorrent
```bash
# Build Docker image
docker build -t simple-torrent:latest .

# Run container
docker run -d --name st-test -p 3000:3000 simple-torrent:latest

# Query the new provider API
curl -s "http://127.0.0.1:3000/search/<site_id>?query=ubuntu&page=1" | jq .

# Stop container
docker rm -f st-test
```

Commit the new JSON file and open a Pull Request!

---

## 🤝 Community Requests via GitHub Issues

Non-developer community members can request new search sites or report broken ones without writing Go code!

1. **Request New Site**: Use our structured [New Provider Request Template](https://github.com/makrron/simple-torrent/issues/new?template=torrent_site_request.yml).
2. **Report Broken Provider**: Use our [Report Broken Provider Template](https://github.com/makrron/simple-torrent/issues/new?template=broken_search_provider.yml).
