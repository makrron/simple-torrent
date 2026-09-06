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

COPY --from=builder /usr/local/bin/cloud-torrent /usr/local/bin/cloud-torrent

EXPOSE 3000 50007 50007/udp

ENTRYPOINT ["cloud-torrent"]
