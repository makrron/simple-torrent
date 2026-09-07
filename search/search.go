package search

import (
	"embed"
)

// Result represents a single torrent search result.
type Result struct {
	Name     string `json:"name,omitempty"`
	Size     string `json:"size,omitempty"`
	Seeds    string `json:"seeds,omitempty"`
	Peers    string `json:"peers,omitempty"`
	Magnet   string `json:"magnet,omitempty"`
	Torrent  string `json:"torrent,omitempty"`
	URL      string `json:"url,omitempty"`
	Path     string `json:"path,omitempty"`
	InfoHash string `json:"infohash,omitempty"`
	Tracker  string `json:"tracker,omitempty"`
	Date     string `json:"date,omitempty"`
	Category string `json:"category,omitempty"`
}

// ItemDetail represents the detail data resolved for a specific torrent item.
type ItemDetail struct {
	Magnet   string `json:"magnet,omitempty"`
	Torrent  string `json:"torrent,omitempty"`
	InfoHash string `json:"infohash,omitempty"`
	Tracker  string `json:"tracker,omitempty"`
}

//go:embed providers/*.json
var EmbeddedProviders embed.FS
