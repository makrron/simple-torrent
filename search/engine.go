package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EngineOptions configures the search Engine.
type EngineOptions struct {
	ProxyURL string
	Timeout  time.Duration
	Debug    bool
}

// Engine manages search providers and serves HTTP requests.
type Engine struct {
	opts      EngineOptions
	client    *HTTPClient
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewEngine creates and initializes a new search Engine.
func NewEngine(opts EngineOptions) *Engine {
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}

	e := &Engine{
		opts:      opts,
		client:    NewHTTPClient(opts.ProxyURL, opts.Timeout),
		providers: make(map[string]Provider),
	}

	// Automatically load embedded providers
	if err := e.LoadEmbeddedProviders(); err != nil {
		log.Printf("[search] warning: failed loading embedded providers: %v", err)
	}

	return e
}

// LoadEmbeddedProviders loads all JSON provider definitions embedded in the binary.
func (e *Engine) LoadEmbeddedProviders() error {
	entries, err := fs.ReadDir(EmbeddedProviders, "providers")
	if err != nil {
		return err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := EmbeddedProviders.ReadFile("providers/" + entry.Name())
		if err != nil {
			log.Printf("[search] error reading embedded provider %s: %v", entry.Name(), err)
			continue
		}

		configs, err := ParseProvidersConfig(data)
		if err != nil {
			log.Printf("[search] error parsing provider %s: %v", entry.Name(), err)
			continue
		}

		for _, cfg := range configs {
			if cfg.ID == "" {
				cfg.ID = strings.TrimSuffix(entry.Name(), ".json")
			}
			e.RegisterConfig(cfg)
			count++
		}
	}

	log.Printf("[search] loaded %d embedded search providers", count)
	return nil
}

// LoadConfigData parses and registers providers from a byte slice (JSON/YAML or legacy map).
func (e *Engine) LoadConfigData(data []byte) error {
	configs, err := ParseProvidersConfig(data)
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		e.RegisterConfig(cfg)
	}

	return nil
}

// LoadFromURL downloads remote provider configurations from an HTTP(S) URL.
func (e *Engine) LoadFromURL(confURL string) error {
	if !strings.HasPrefix(confURL, "http://") && !strings.HasPrefix(confURL, "https://") {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	data, status, err := e.client.Get(ctx, confURL, nil)
	if err != nil {
		return fmt.Errorf("failed fetching search config from %s: %w", confURL, err)
	}

	if status != http.StatusOK {
		return fmt.Errorf("search config URL %s returned status %d", confURL, status)
	}

	if err := e.LoadConfigData(data); err != nil {
		return fmt.Errorf("failed parsing remote search config: %w", err)
	}

	log.Printf("[search] loaded search providers from %s", confURL)
	return nil
}

// RegisterConfig adds or updates a provider from a ProviderConfig.
func (e *Engine) RegisterConfig(cfg *ProviderConfig) {
	if cfg == nil || cfg.ID == "" {
		return
	}

	p := NewDefaultProvider(cfg, e.client, e.opts.Debug)
	e.mu.Lock()
	e.providers[cfg.ID] = p
	e.mu.Unlock()
}

// GetProvider retrieves a provider by ID.
func (e *Engine) GetProvider(id string) (Provider, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.providers[id]
	return p, ok
}

// ProvidersMap returns a map representation suitable for JSON encoding in /api/searchproviders.
// It also exposes any virtual <id>/item endpoints to maintain full backward compatibility with the UI.
func (e *Engine) ProvidersMap() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]interface{})
	for id, p := range e.providers {
		cfg := p.Config()
		result[id] = map[string]interface{}{
			"name":   cfg.Name,
			"url":    cfg.URL,
			"list":   cfg.List,
			"result": cfg.Result,
		}

		if cfg.ItemURL != "" {
			result[id+"/item"] = map[string]interface{}{
				"name":   cfg.Name + " (Item)",
				"url":    cfg.ItemURL,
				"result": cfg.ItemResult,
			}
		}
	}

	return result
}

// ServeHTTP implements http.Handler for /search endpoints.
// Patterns handled:
// - GET /{provider}?query={q}&page={p}
// - GET /{provider}/item?item={path}
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	providerID := parts[0]

	if providerID == "" {
		// List all providers as JSON
		json.NewEncoder(w).Encode(e.ProvidersMap())
		return
	}

	isItem := len(parts) > 1 && parts[1] == "item"
	provider, exists := e.GetProvider(providerID)
	if !exists {
		http.Error(w, fmt.Sprintf(`{"error":"search provider '%s' not found"}`, providerID), http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), e.opts.Timeout)
	defer cancel()

	if isItem {
		itemPath := r.URL.Query().Get("item")
		if itemPath == "" {
			http.Error(w, `{"error":"missing 'item' query parameter"}`, http.StatusBadRequest)
			return
		}

		detail, err := provider.GetItem(ctx, itemPath)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
			return
		}

		json.NewEncoder(w).Encode(detail)
		return
	}

	// Search request
	query := r.URL.Query().Get("query")
	pageStr := r.URL.Query().Get("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	results, err := provider.Search(ctx, query, page)
	if err != nil {
		log.Printf("[search][%s] query='%s' page=%d FAILED: %v", providerID, query, page, err)
		w.Header().Set("X-Search-Error", err.Error())
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
		return
	}

	log.Printf("[search][%s] query='%s' page=%d returned %d results", providerID, query, page, len(results))
	json.NewEncoder(w).Encode(results)
}

// SetDebug toggles debug logging.
func (e *Engine) SetDebug(debug bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.opts.Debug = debug
}
