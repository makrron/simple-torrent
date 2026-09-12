package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEngine_EmbeddedProviders(t *testing.T) {
	engine := NewEngine(EngineOptions{
		Timeout: 5 * time.Second,
		Debug:   false,
	})

	expectedProviders := []string{"tpb", "yts", "nyaa", "1337x", "eztv", "lt", "abb", "torrentgalaxy", "torrentclaw"}
	for _, id := range expectedProviders {
		p, exists := engine.GetProvider(id)
		if !exists {
			t.Errorf("expected embedded provider '%s' to be registered", id)
			continue
		}
		if p.Name() == "" {
			t.Errorf("provider '%s' has empty name", id)
		}
	}

	pMap := engine.ProvidersMap()
	if len(pMap) < len(expectedProviders) {
		t.Errorf("expected at least %d providers in map, got %d", len(expectedProviders), len(pMap))
	}
}

func TestExtractHTMLResults(t *testing.T) {
	htmlContent := `
	<html>
	<body>
		<table class="table-list">
			<tbody>
				<tr>
					<td class="coll-1"><a href="/item/101">Ubuntu 24.04 Desktop AMD64</a></td>
					<td class="coll-2">1500</td>
					<td class="coll-3">50</td>
					<td class="coll-4">5.8 GB</td>
					<td><a href="magnet:?xt=urn:btih:1234567890abcdef">Magnet</a></td>
				</tr>
				<tr>
					<td class="coll-1"><a href="/item/102">Debian 12 Netinst</a></td>
					<td class="coll-2">800</td>
					<td class="coll-3">20</td>
					<td class="coll-4">650 MB</td>
					<td><a href="magnet:?xt=urn:btih:abcdef1234567890">Magnet</a></td>
				</tr>
			</tbody>
		</table>
	</body>
	</html>
	`

	rules := map[string]interface{}{
		"name":   ".coll-1 a",
		"path":   []interface{}{".coll-1 a", "@href"},
		"seeds":  ".coll-2",
		"peers":  ".coll-3",
		"size":   ".coll-4",
		"magnet": []interface{}{"a[href^='magnet:']", "@href"},
	}

	results, err := ExtractHTMLResults([]byte(htmlContent), "table.table-list tbody tr", rules)
	if err != nil {
		t.Fatalf("unexpected error extracting HTML: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	first := results[0]
	if first.Name != "Ubuntu 24.04 Desktop AMD64" {
		t.Errorf("expected name 'Ubuntu 24.04 Desktop AMD64', got '%s'", first.Name)
	}
	if first.Seeds != "1500" {
		t.Errorf("expected seeds '1500', got '%s'", first.Seeds)
	}
	if first.Peers != "50" {
		t.Errorf("expected peers '50', got '%s'", first.Peers)
	}
	if first.Size != "5.8 GB" {
		t.Errorf("expected size '5.8 GB', got '%s'", first.Size)
	}
	if first.Path != "/item/101" {
		t.Errorf("expected path '/item/101', got '%s'", first.Path)
	}
	if first.Magnet != "magnet:?xt=urn:btih:1234567890abcdef" {
		t.Errorf("expected magnet 'magnet:?xt=urn:btih:1234567890abcdef', got '%s'", first.Magnet)
	}
}

func TestExtractJSONResults(t *testing.T) {
	jsonContent := `{
		"status": "ok",
		"data": {
			"movie_count": 1,
			"movies": [
				{
					"title_long": "Inception (2010)",
					"url": "https://yts.mx/movies/inception-2010",
					"torrents": [
						{
							"size": "2.2 GB",
							"seeds": 520,
							"peers": 40,
							"url": "https://yts.mx/torrent/download/1234",
							"hash": "ABCD1234EF5678"
						}
					]
				}
			]
		}
	}`

	rules := map[string]interface{}{
		"name":    "title_long",
		"url":     "url",
		"size":    "torrents.0.size",
		"seeds":   "torrents.0.seeds",
		"peers":   "torrents.0.peers",
		"torrent": "torrents.0.url",
		"hash":    "torrents.0.hash",
	}

	results, err := ExtractJSONResults([]byte(jsonContent), "data.movies", rules)
	if err != nil {
		t.Fatalf("unexpected error extracting JSON: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Name != "Inception (2010)" {
		t.Errorf("expected name 'Inception (2010)', got '%s'", r.Name)
	}
	if r.Size != "2.2 GB" {
		t.Errorf("expected size '2.2 GB', got '%s'", r.Size)
	}
	if r.Seeds != "520" {
		t.Errorf("expected seeds '520', got '%s'", r.Seeds)
	}
	if r.Peers != "40" {
		t.Errorf("expected peers '40', got '%s'", r.Peers)
	}
	if r.InfoHash != "ABCD1234EF5678" {
		t.Errorf("expected hash 'ABCD1234EF5678', got '%s'", r.InfoHash)
	}
	if r.Magnet == "" || !r.MagnetStartsWith("magnet:?xt=urn:btih:ABCD1234EF5678") {
		t.Errorf("expected auto-constructed magnet link starting with hash, got: %s", r.Magnet)
	}
}

// Helper method on Result for test readability
func (r *Result) MagnetStartsWith(prefix string) bool {
	return len(r.Magnet) >= len(prefix) && r.Magnet[:len(prefix)] == prefix
}

func TestExtractJSONResults_NestedTorrentClaw(t *testing.T) {
	jsonContent := `{
		"total": 1,
		"results": [
			{
				"id": 67092,
				"title": "Arrival",
				"contentUrl": "/movies/arrival-2016-67092",
				"torrents": [
					{
						"rawTitle": "Arrival.2016.UHD.2160p.BluRay.x265.HDR.DTS-HFMA.7.1-DTOne",
						"magnetUrl": "magnet:?xt=urn:btih:d0f7925fcba5ed1fbf8b1916533147fd0dccde6a&dn=Arrival",
						"torrentUrl": "/api/v1/torrent/d0f7925fcba5ed1fbf8b1916533147fd0dccde6a",
						"sizeBytes": 9005906433,
						"seeders": 158,
						"leechers": 47,
						"infoHash": "d0f7925fcba5ed1fbf8b1916533147fd0dccde6a"
					}
				]
			}
		]
	}`

	rules := map[string]interface{}{
		"name":     []interface{}{"rawTitle", "_parent.title"},
		"size":     "sizeBytes",
		"seeds":    "seeders",
		"peers":    "leechers",
		"magnet":   "magnetUrl",
		"torrent":  "torrentUrl",
		"infohash": "infoHash",
		"path":     "_parent.contentUrl",
	}

	results, err := ExtractJSONResults([]byte(jsonContent), "results.*.torrents", rules)
	if err != nil {
		t.Fatalf("unexpected error extracting TorrentClaw JSON: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Name != "Arrival.2016.UHD.2160p.BluRay.x265.HDR.DTS-HFMA.7.1-DTOne" {
		t.Errorf("unexpected name: %s", r.Name)
	}
	if r.Size != "8.39 GB" {
		t.Errorf("expected size '8.39 GB', got '%s'", r.Size)
	}
	if r.Seeds != "158" {
		t.Errorf("expected seeds '158', got '%s'", r.Seeds)
	}
	if r.Peers != "47" {
		t.Errorf("expected peers '47', got '%s'", r.Peers)
	}
	if r.InfoHash != "d0f7925fcba5ed1fbf8b1916533147fd0dccde6a" {
		t.Errorf("expected infohash 'd0f7925fcba5ed1fbf8b1916533147fd0dccde6a', got '%s'", r.InfoHash)
	}
	if !r.MagnetStartsWith("magnet:?xt=urn:btih:d0f7925fcba5ed1fbf8b1916533147fd0dccde6a") {
		t.Errorf("unexpected magnet URL: %s", r.Magnet)
	}
	if r.Path != "/movies/arrival-2016-67092" {
		t.Errorf("expected path '/movies/arrival-2016-67092', got '%s'", r.Path)
	}
}

func TestExtractHTMLItemDetail_TorrentClaw(t *testing.T) {
	htmlContent := `
	<html>
	<body>
		<div id="torrent-d0f7925fcba5ed1fbf8b1916533147fd0dccde6a" data-torrent-hash="d0f7925fcba5ed1fbf8b1916533147fd0dccde6a">
			<button>Download</button>
		</div>
	</body>
	</html>
	`

	rules := map[string]interface{}{
		"infohash": []interface{}{"div[data-torrent-hash]", "@data-torrent-hash"},
	}

	detail, err := ExtractHTMLItemDetail([]byte(htmlContent), rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.InfoHash != "d0f7925fcba5ed1fbf8b1916533147fd0dccde6a" {
		t.Errorf("expected infohash 'd0f7925fcba5ed1fbf8b1916533147fd0dccde6a', got '%s'", detail.InfoHash)
	}
	if detail.Magnet == "" || !strings.HasPrefix(detail.Magnet, "magnet:?xt=urn:btih:d0f7925fcba5ed1fbf8b1916533147fd0dccde6a") {
		t.Errorf("expected auto-constructed magnet link, got: %s", detail.Magnet)
	}
}

func TestProvider_MirrorFallback(t *testing.T) {
	// Server 1 (fails with 500)
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server down", http.StatusInternalServerError)
	}))
	defer s1.Close()

	// Server 2 (succeeds with results)
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"name": "Fallback Torrent Success", "seeds": "100"}]`)
	}))
	defer s2.Close()

	client := NewHTTPClient("", 2*time.Second)
	cfg := &ProviderConfig{
		ID:   "fallback_test",
		Name: "Fallback Test Provider",
		Type: "json",
		URL:  s1.URL + "/search?q={{query}}",
		Mirrors: []string{
			s1.URL + "/search?q={{query}}",
			s2.URL + "/search?q={{query}}",
		},
		Root: "",
		Result: map[string]interface{}{
			"name":  "name",
			"seeds": "seeds",
		},
	}

	p := NewDefaultProvider(cfg, client, true)
	results, err := p.Search(context.Background(), "linux", 1)
	if err != nil {
		t.Fatalf("expected search to succeed via fallback mirror, got: %v", err)
	}

	if len(results) != 1 || results[0].Name != "Fallback Torrent Success" {
		t.Fatalf("unexpected results from mirror fallback: %+v", results)
	}
}

func TestEngine_ServeHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"name": "Arch Linux ISO", "size": "900MB", "seeds": "300"}]`)
	}))
	defer ts.Close()

	engine := NewEngine(EngineOptions{
		Timeout: 5 * time.Second,
	})

	engine.RegisterConfig(&ProviderConfig{
		ID:   "mock",
		Name: "Mock Provider",
		Type: "json",
		URL:  ts.URL + "/search?q={{query}}",
		Root: "",
		Result: map[string]interface{}{
			"name":  "name",
			"size":  "size",
			"seeds": "seeds",
		},
	})

	// Perform HTTP GET /mock?query=arch
	req := httptest.NewRequest(http.MethodGet, "/mock?query=arch", nil)
	rr := httptest.NewRecorder()
	engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var results []Result
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed decoding json response: %v", err)
	}

	if len(results) != 1 || results[0].Name != "Arch Linux ISO" {
		t.Fatalf("expected Arch Linux ISO, got: %+v", results)
	}
}

func TestParseLegacyConfig(t *testing.T) {
	legacyJSON := `{
		"legacy_site": {
			"name": "Legacy Site",
			"url": "https://legacy.example.com/search/{{query}}",
			"list": "div.row",
			"result": {
				"name": "h3.title"
			}
		},
		"legacy_site/item": {
			"name": "Legacy Site (Item)",
			"url": "https://legacy.example.com/item/{{item}}",
			"result": {
				"magnet": "a.magnet@href"
			}
		}
	}`

	configs, err := ParseProvidersConfig([]byte(legacyJSON))
	if err != nil {
		t.Fatalf("unexpected error parsing legacy config: %v", err)
	}

	if len(configs) != 1 {
		t.Fatalf("expected 1 merged config, got %d", len(configs))
	}

	c := configs[0]
	if c.ID != "legacy_site" || c.Name != "Legacy Site" {
		t.Errorf("unexpected ID or Name: %s, %s", c.ID, c.Name)
	}
	if c.ItemURL != "https://legacy.example.com/item/{{item}}" {
		t.Errorf("expected merged ItemURL, got: %s", c.ItemURL)
	}
}

func TestLiveProvidersAudit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network audit in short mode")
	}

	engine := NewEngine(EngineOptions{
		Timeout: 10 * time.Second,
		Debug:   true,
	})

	queryMap := map[string]string{
		"tpb":           "ubuntu",
		"yts":           "matrix",
		"nyaa":          "naruto",
		"1337x":         "ubuntu",
		"eztv":          "house",
		"lt":            "ubuntu",
		"abb":           "ubuntu",
		"torrentgalaxy": "ubuntu",
		"torrentclaw":   "la llegada",
	}

	for id, query := range queryMap {
		p, exists := engine.GetProvider(id)
		if !exists {
			t.Logf("Provider %s does not exist", id)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		results, err := p.Search(ctx, query, 1)
		cancel()

		if err != nil {
			t.Logf("❌ [%s] FAIL (query: '%s'): %v", id, query, err)
		} else {
			t.Logf("✅ [%s] SUCCESS (query: '%s'): %d results found (first: %s)", id, query, len(results), func() string {
				if len(results) > 0 {
					return results[0].Name
				}
				return "NONE"
			}())
		}
	}
}


