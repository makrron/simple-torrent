############################
# STEP 1 build executable binary
############################
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git make build-base

WORKDIR /src

# Copy module definition files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build executable binary with CGO disabled (Pure Go)
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.VERSION=$(git describe --tags 2>/dev/null || echo 'v1.4.0-dev')" -o /usr/local/bin/cloud-torrent

############################
# STEP 2 build lightweight final image
############################
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

# Create default directories for downloads and torrent watcher
RUN mkdir -p /downloads /torrents

COPY --from=builder /usr/local/bin/cloud-torrent /usr/local/bin/cloud-torrent
RUN ln -s /usr/local/bin/cloud-torrent /usr/local/bin/simple-torrent

VOLUME ["/downloads", "/torrents"]

EXPOSE 3000 50007 50007/udp

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

ENTRYPOINT ["cloud-torrent"]

