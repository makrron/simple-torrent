![screenshot](https://user-images.githubusercontent.com/1033514/64239393-bdbb6480-cf32-11e9-9269-d8d10e7c0dc7.png)

![Build Status](https://github.com/makrron/simple-torrent/workflows/Docker%20Build%20%26%20Push/badge.svg) 

**SimpleTorrent** is a self-hosted remote torrent client, written in Go (golang). Start torrents remotely and download sets of files on the local disk of the server, which are then retrievable or streamable via HTTP.

Maintained by **[@makrron](https://github.com/makrron)**. Originally based on [cloud-torrent](https://github.com/jpillora/cloud-torrent) by `jpillora` and `boypt`.

# Features

* Individual file download control
* Run external program on tasks completion: `DoneCmd`
* Stops task when seeding ratio reached: `SeedRatio`
* Download/Upload speed limiter: `UploadRate`/`DownloadRate`
* Detailed transfer stats in web UI.
* Torrent Watcher
* K8s/docker health-check endpoint `/healthz`
* Extra trackers from external source
* Protocol Handler for `magnet:`
* Magnet RSS subscribing supported
* Flexible config file accepts multiple formats (.json/.yaml/.toml via Viper)

Also:
* Single binary
* Cross platform
* **Modular Torrent Search Engine** (HTML scraping + JSON APIs, mirror failover, easily extensible)
* Real-time updates
* Mobile-friendly
* Fast content server
* IPv6 out of the box
* Updated torrent engine built on `anacrolix/torrent`

## 🔍 Torrent Search Engine
SimpleTorrent includes a modular search engine with built-in support for popular torrent indexes (The Pirate Bay, YTS, 1337X, Nyaa, EZTV, LimeTorrents, TorrentGalaxy, AudiobookBay).

* **Extensible**: Add new search sites by simply adding a JSON file in `search/providers/`.
* **Community Driven**: Request new sites or report issues via [GitHub Issue Templates](https://github.com/makrron/simple-torrent/issues/new/choose).
* **Documentation**: See [`docs/SEARCH_PROVIDERS.md`](docs/SEARCH_PROVIDERS.md) for architecture, schema details, and extension tutorials.

# Install

## Binary

See [the latest release](https://github.com/makrron/simple-torrent/releases/latest).

## Docker

```bash
docker run -d -p 3000:3000 -p 50007:50007 -p 50007:50007/udp -v /path/to/my/downloads:/srv/downloads -v /path/to/my/torrents:/srv/torrents makrron/simple-torrent:latest
```

When running as a container, keep in mind:
* You need to expose your torrent incoming port (50007 by default) if you want to seed (`-p 50007:50007`). Also, forward the port on your router.
* Automatic port forwarding on your router via UPnP IGD will not work unless run in `host` mode (`--net=host`).

It's recommended to run via Docker Compose:

```yaml
version: '3.8'

services:
  simple-torrent:
    image: makrron/simple-torrent:latest
    container_name: simple-torrent
    restart: unless-stopped
    ports:
      - "3000:3000"
      - "50007:50007"
      - "50007:50007/udp"
    volumes:
      - ./downloads:/srv/downloads
      - ./torrents:/srv/torrents
```

## Source Build

**Requirement**
- Golang (Go 1.18+)

```bash
$ git clone https://github.com/makrron/simple-torrent.git
$ cd simple-torrent
$ CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.VERSION=$(git describe --tags)" -o simple-torrent
```

# Credits 
* Maintained by [@makrron](https://github.com/makrron)
* Credits to @boypt for original SimpleTorrent.
* Credits to @jpillora for Cloud Torrent.
* Credits to @anacrolix for torrent engine.
