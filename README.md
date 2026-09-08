<div align="center">

# ⚡ SimpleTorrent

**A lightweight, self-hosted remote torrent client with a modern Web UI.**

[![CI Status](https://github.com/makrron/simple-torrent/actions/workflows/ci.yml/badge.svg)](https://github.com/makrron/simple-torrent/actions/workflows/ci.yml)
[![Docker Delivery](https://github.com/makrron/simple-torrent/actions/workflows/docker-push.yml/badge.svg)](https://github.com/makrron/simple-torrent/actions/workflows/docker-push.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/makrron/simple-torrent?logo=docker&logoColor=white&label=Docker%20Pulls&color=0db7ed)](https://hub.docker.com/r/makrron/simple-torrent)
[![Docker Image Size](https://img.shields.io/docker/image-size/makrron/simple-torrent/latest?logo=docker&logoColor=white&label=Image%20Size&color=blue)](https://hub.docker.com/r/makrron/simple-torrent)
[![GitHub Release](https://img.shields.io/github/v/release/makrron/simple-torrent?logo=github&logoColor=white&color=blue)](https://github.com/makrron/simple-torrent/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/makrron/simple-torrent?logo=go&logoColor=white&color=00ADD8)](go.mod)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_3.0-blue.svg)](LICENSE)

<br/>

![SimpleTorrent Web UI](https://user-images.githubusercontent.com/1033514/64239393-bdbb6480-cf32-11e9-9269-d8d10e7c0dc7.png)

</div>

---

## 📖 Overview

**SimpleTorrent** is a fast, self-hosted remote torrent client written in Go (Golang). It allows you to start torrents remotely and download files directly onto your server's local disk, making them immediately retrievable or streamable over HTTP through a responsive, mobile-friendly browser interface.

Maintained with modern Go standards by **[@makrron](https://github.com/makrron)**. Originally forked from [cloud-torrent](https://github.com/jpillora/cloud-torrent) by `jpillora` and `boypt`.

---

## ✨ Key Features

| Category | Highlights |
| :--- | :--- |
| 🌐 **Modern Web Interface** | Clean, real-time responsive dashboard with live transfer speeds, ETA, peer maps, and mobile support. |
| 🔍 **Modular Search Engine** | Integrated search scrapers (YTS, 1337X, The Pirate Bay, LimeTorrents, Nyaa, TorrentGalaxy, and more) with automatic mirror failovers. |
| 📁 **Granular File Control** | Selective per-file downloading, priority queues, and instant in-browser HTTP streaming / downloads. |
| ⚡ **Lightweight & High Performance** | Native static Go binary with embedded web assets, low CPU/RAM footprint, memory-mapped I/O (`mmap`), and IPv6 support. |
| 🎛️ **Bandwidth & Automation** | Configurable upload/download rate limiters (`UploadRate`/`DownloadRate`), auto-stop seeding ratios (`SeedRatio`), and external post-completion scripts (`DoneCmd`). |
| 🧲 **Protocols & Feeds** | Full `magnet:` URI protocol handler, automated public tracker scraping/injection, and automated RSS feed subscriptions. |
| 🛡️ **Container & Homelab Ready** | Production-ready multi-architecture Docker container (`amd64`, `arm64`, `armv7`, `386`), Kubernetes health check endpoint (`/healthz`), and UmbrelOS integration. |

---

## 🔍 Built-In Torrent Search Engine

SimpleTorrent features an extensible, zero-config search engine embedded directly into the binary:

* **Pre-configured Providers**: The Pirate Bay (`tpb`), YTS (`yts`), 1337X (`1337x`), Nyaa (`nyaa`), LimeTorrents (`lt`), TorrentGalaxy (`torrentgalaxy`), and AudiobookBay (`abb`).
* **Resilient Architecture**: Automatic failover between alternative mirrors, rotating modern browser User-Agents, and Client Hints to overcome anti-bot protections.
* **Extensible & Community-Driven**: Add new providers simply by dropping a `.json` definition in `search/providers/`.
* **Full Documentation**: Check out [`docs/SEARCH_PROVIDERS.md`](docs/SEARCH_PROVIDERS.md) for the JSON schema and step-by-step contribution tutorials.

---

## 🚀 Deployment & Installation

### Option 1: Docker Compose (Recommended)

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  simple-torrent:
    image: makrron/simple-torrent:latest
    container_name: simple-torrent
    restart: unless-stopped
    ports:
      - "3000:3000"         # Web UI & HTTP downloads
      - "50007:50007"       # BitTorrent TCP (Optional for incoming peering)
      - "50007:50007/udp"   # BitTorrent UDP / DHT
    volumes:
      - ./downloads:/downloads
      - ./torrents:/torrents
      - ./config:/config    # Optional: persistent custom configuration
    environment:
      - TITLE=SimpleTorrent
```

Start the container:
```bash
docker compose up -d
```
Access the web dashboard at `http://localhost:3000`.

---

### Option 2: Docker CLI

Run directly via the `docker run` command:

```bash
docker run -d \
  --name simple-torrent \
  --restart unless-stopped \
  -p 3000:3000 \
  -v /path/to/my/downloads:/downloads \
  -v /path/to/my/torrents:/torrents \
  makrron/simple-torrent:latest
```

> [!TIP]
> **Multi-Arch Support**: The Docker Hub image [`makrron/simple-torrent:latest`](https://hub.docker.com/r/makrron/simple-torrent) natively supports `linux/amd64`, `linux/arm64` (e.g., Raspberry Pi 4/5, Apple Silicon), `linux/arm/v7`, and `linux/386`.

---

### Option 3: UmbrelOS / Umbrel App Store

SimpleTorrent is available directly within the **Umbrel App Store** for one-click installation on UmbrelOS home servers, integrating with Umbrel's shared storage and authentication proxy.

---

### Option 4: Standalone Pre-Built Binaries

Pre-compiled standalone binaries for Linux, macOS, and Windows are available for every release:

👉 **[Download the Latest Release](https://github.com/makrron/simple-torrent/releases/latest)**

Example for Linux x86_64:
```bash
curl -fsSL https://github.com/makrron/simple-torrent/releases/latest/download/simple-torrent_linux_amd64.tar.gz | tar -xz
./simple-torrent --listen :3000
```

---

### Option 5: Compile from Source

**Prerequisites**: Go 1.22+ and `git`.

```bash
# Clone the repository
git clone https://github.com/makrron/simple-torrent.git
cd simple-torrent

# Download dependencies
go mod download

# Compile static binary with version metadata
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.VERSION=$(git describe --tags)" -o simple-torrent

# Run the server
./simple-torrent --listen :3000
```

---

## ⚙️ Configuration & CLI Options

SimpleTorrent works out-of-the-box with sensible defaults, but can be configured via CLI flags, environment variables, or a configuration file (`.json`, `.yaml`, or `.toml` via Viper):

| Flag | Env Variable | Default | Description |
| :--- | :--- | :--- | :--- |
| `-l`, `--listen` | `LISTEN` | `:3000` | Address/Port to listen on (e.g., `:3000`, `0.0.0.0:8080`, or unix socket). |
| `-t`, `--title` | `TITLE` | `SimpleTorrent` | Title displayed in the Web UI header. |
| `-c`, `--config-path` | `CONFIGPATH` | `./cloud-torrent.yaml` | Path to persistent configuration file. |
| `-a`, `--auth` | `AUTH` | `""` | Basic HTTP Authentication credentials (`username:password`). |
| `--proxy-url` | `PROXY_URL` | `""` | Outgoing HTTP/SOCKS5 proxy URL for engine and search traffic. |
| `--rest-api` | `RESTAPI` | `""` | Local trusted address:port accepting `/api/` requests without auth. |
| `--req-log` | `REQLOG` | `false` | Enable detailed HTTP request access logging. |
| `--disable-mmap` | `DISABLEMMAP` | `false` | Disable memory-mapped file I/O (recommended for low-memory 32-bit systems). |
| `--debug` | `DEBUG` | `false` | Enable debug logging for search queries and server operations. |
| `--debug-torrent` | `DEBUGTORRENT`| `false` | Enable verbose debug output from the underlying torrent engine. |
| `-v`, `--version` | — | — | Print version and runtime details, then exit. |

---

## 🤝 Contributing

Contributions, bug reports, and provider additions are warmly welcomed!

* **Report a bug or broken provider**: Open an issue using our [GitHub Issue Templates](https://github.com/makrron/simple-torrent/issues/new/choose).
* **Submit code changes**: Fork the repository, create your feature branch, and submit a pull request.
* **Run tests**:
  ```bash
  go test ./... -v
  ```

---

## 📜 License & Credits

* **License**: Released under the [GNU Affero General Public License v3.0 (AGPL-3.0)](LICENSE).
* **Maintainer**: [@makrron](https://github.com/makrron)
* **Acknowledgements**:
  * Special thanks to **@boypt** for the initial SimpleTorrent fork.
  * Special thanks to **@jpillora** for the original Cloud Torrent project.
  * Powered by the robust [`anacrolix/torrent`](https://github.com/anacrolix/torrent) BitTorrent library.
