package search

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ProviderConfig defines configuration for a single torrent search provider.
type ProviderConfig struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type,omitempty"` // "html" (default) or "json"
	URL         string                 `json:"url"`
	Mirrors     []string               `json:"mirrors,omitempty"`
	List        string                 `json:"list,omitempty"` // CSS selector for rows (HTML)
	Root        string                 `json:"root,omitempty"` // JSON path for results array (JSON API)
	Result      map[string]interface{} `json:"result,omitempty"`
	ItemURL     string                 `json:"item_url,omitempty"`
	ItemMirrors []string               `json:"item_mirrors,omitempty"`
	ItemResult  map[string]interface{} `json:"item_result,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
}

// GetSearchURLs returns all search URLs to attempt (primary followed by mirrors), with query and page substituted.
func (p *ProviderConfig) GetSearchURLs(query string, page int) []string {
	allURLs := []string{}
	if p.URL != "" {
		allURLs = append(allURLs, p.URL)
	}
	for _, m := range p.Mirrors {
		if m != p.URL && m != "" {
			allURLs = append(allURLs, m)
		}
	}

	result := make([]string, 0, len(allURLs))
	for _, rawURL := range allURLs {
		formatted := formatURL(rawURL, query, page)
		result = append(result, formatted)
	}
	return result
}

// GetItemURLs returns item URLs to attempt (primary followed by mirrors), with item path substituted.
func (p *ProviderConfig) GetItemURLs(itemPath string) []string {
	allURLs := []string{}
	if p.ItemURL != "" {
		allURLs = append(allURLs, p.ItemURL)
	}
	for _, m := range p.ItemMirrors {
		if m != p.ItemURL && m != "" {
			allURLs = append(allURLs, m)
		}
	}

	result := make([]string, 0, len(allURLs))
	for _, rawURL := range allURLs {
		formatted := strings.ReplaceAll(rawURL, "{{item}}", itemPath)
		result = append(result, formatted)
	}
	return result
}

func formatURL(rawURL string, query string, page int) string {
	res := rawURL
	escapedQuery := query
	if strings.Contains(res, "{{query}}") {
		if strings.Contains(res, "?") {
			escapedQuery = url.QueryEscape(query)
		} else if strings.Contains(escapedQuery, " ") {
			escapedQuery = strings.ReplaceAll(escapedQuery, " ", "+")
		}
		res = strings.ReplaceAll(res, "{{query}}", escapedQuery)
	}

	// Handle page substitution
	// {{page:0}} means 0-indexed page
	// {{page:1}} means 1-indexed page
	if strings.Contains(res, "{{page:0}}") {
		pageVal := page
		if pageVal > 0 {
			pageVal--
		}
		res = strings.ReplaceAll(res, "{{page:0}}", fmt.Sprintf("%d", pageVal))
	} else if strings.Contains(res, "{{page:1}}") {
		res = strings.ReplaceAll(res, "{{page:1}}", fmt.Sprintf("%d", page))
	} else if strings.Contains(res, "{{page}}") {
		res = strings.ReplaceAll(res, "{{page}}", fmt.Sprintf("%d", page))
	}

	return res
}

// ParseProvidersConfig parses either a single ProviderConfig, a list of ProviderConfig,
// or a legacy map[string]ProviderConfig (e.g. from scraper-config.json).
func ParseProvidersConfig(data []byte) ([]*ProviderConfig, error) {
	// Try single provider
	var single ProviderConfig
	if err := json.Unmarshal(data, &single); err == nil && single.Name != "" && single.URL != "" {
		if single.Type == "" {
			single.Type = "html"
		}
		return []*ProviderConfig{&single}, nil
	}

	// Try slice of providers
	var list []*ProviderConfig
	if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 && list[0].Name != "" {
		for _, p := range list {
			if p.Type == "" {
				p.Type = "html"
			}
		}
		return list, nil
	}

	// Try legacy map[string]json.RawMessage
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil, fmt.Errorf("failed to parse providers config: %w", err)
	}

	providersMap := make(map[string]*ProviderConfig)
	itemMap := make(map[string]*ProviderConfig)

	for k, v := range rawMap {
		var p ProviderConfig
		if err := json.Unmarshal(v, &p); err != nil {
			continue
		}
		p.ID = k
		if strings.HasSuffix(k, "/item") {
			baseID := strings.TrimSuffix(k, "/item")
			itemMap[baseID] = &p
		} else {
			if p.Type == "" {
				p.Type = "html"
			}
			providersMap[k] = &p
		}
	}

	// Merge item handlers into main providers
	for id, itemP := range itemMap {
		if mainP, exists := providersMap[id]; exists {
			mainP.ItemURL = itemP.URL
			mainP.ItemResult = itemP.Result
			if len(itemP.Mirrors) > 0 {
				mainP.ItemMirrors = itemP.Mirrors
			}
		} else {
			// Standalone item handler or provider
			itemP.Type = "html"
			providersMap[id+"/item"] = itemP
		}
	}

	res := make([]*ProviderConfig, 0, len(providersMap))
	for _, p := range providersMap {
		res = append(res, p)
	}
	return res, nil
}
