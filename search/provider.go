package search

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// Provider is the common interface implemented by all search providers.
type Provider interface {
	ID() string
	Name() string
	Config() *ProviderConfig
	Search(ctx context.Context, query string, page int) ([]Result, error)
	GetItem(ctx context.Context, itemPath string) (*ItemDetail, error)
}

// DefaultProvider implements Provider using ProviderConfig and HTTPClient.
type DefaultProvider struct {
	config *ProviderConfig
	client *HTTPClient
	debug  bool
}

// NewDefaultProvider creates a new provider instance.
func NewDefaultProvider(config *ProviderConfig, client *HTTPClient, debug bool) *DefaultProvider {
	return &DefaultProvider{
		config: config,
		client: client,
		debug:  debug,
	}
}

func (p *DefaultProvider) ID() string {
	return p.config.ID
}

func (p *DefaultProvider) Name() string {
	return p.config.Name
}

func (p *DefaultProvider) Config() *ProviderConfig {
	return p.config
}

// Search executes query against provider URLs (trying primary and fallback mirrors).
func (p *DefaultProvider) Search(ctx context.Context, query string, page int) ([]Result, error) {
	urls := p.config.GetSearchURLs(query, page)
	if len(urls) == 0 {
		return nil, fmt.Errorf("no search URLs configured for provider %s", p.config.ID)
	}

	var lastErr error
	for i, u := range urls {
		if p.debug || i > 0 {
			log.Printf("[search][%s] trying mirror %d/%d: %s", p.config.ID, i+1, len(urls), u)
		}

		body, statusCode, err := p.client.Get(ctx, u, p.config.Headers)
		if err != nil {
			log.Printf("[search][%s] mirror %s failed: %v", p.config.ID, u, err)
			lastErr = err
			continue
		}

		if statusCode >= http.StatusBadRequest {
			lastErr = fmt.Errorf("HTTP %d from %s", statusCode, u)
			log.Printf("[search][%s] mirror %s returned HTTP status %d", p.config.ID, u, statusCode)
			continue
		}

		var results []Result
		if p.config.Type == "json" {
			results, err = ExtractJSONResults(body, p.config.Root, p.config.Result)
		} else {
			results, err = ExtractHTMLResults(body, p.config.List, p.config.Result)
		}

		if err != nil {
			lastErr = err
			log.Printf("[search][%s] parsing error from %s: %v", p.config.ID, u, err)
			continue
		}

		if p.debug {
			log.Printf("[search][%s] returned %d results from %s", p.config.ID, len(results), u)
		}

		// Success!
		return results, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all mirrors failed for provider %s: %w", p.config.ID, lastErr)
	}

	return []Result{}, nil
}

// GetItem resolves item details (magnet, torrent, infohash) for a specific item.
func (p *DefaultProvider) GetItem(ctx context.Context, itemPath string) (*ItemDetail, error) {
	urls := p.config.GetItemURLs(itemPath)
	if len(urls) == 0 {
		return nil, fmt.Errorf("no item URLs configured for provider %s", p.config.ID)
	}

	var lastErr error
	for _, u := range urls {
		body, statusCode, err := p.client.Get(ctx, u, p.config.Headers)
		if err != nil {
			lastErr = err
			continue
		}

		if statusCode >= http.StatusBadRequest {
			lastErr = fmt.Errorf("HTTP %d from %s", statusCode, u)
			continue
		}

		var detail *ItemDetail
		detail, err = ExtractHTMLItemDetail(body, p.config.ItemResult)
		if err != nil {
			lastErr = err
			continue
		}

		if detail != nil && (detail.Magnet != "" || detail.Torrent != "" || detail.InfoHash != "") {
			return detail, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed resolving item for %s: %w", p.config.ID, lastErr)
	}

	return nil, fmt.Errorf("no valid item details extracted for %s", itemPath)
}
