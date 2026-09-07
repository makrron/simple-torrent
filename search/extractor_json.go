package search

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DefaultTrackers used to construct magnets when only infohash is available
var DefaultTrackers = []string{
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.openbittorrent.com:80",
	"udp://tracker.coppersurfer.tk:6969",
	"udp://glotorrents.pw:6969/announce",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://torrent.gresille.org:80/announce",
	"udp://p4p.arenabg.com:1337",
}

// ExtractJSONResults parses a JSON response and returns search Results.
func ExtractJSONResults(data []byte, rootPath string, fieldRules map[string]interface{}) ([]Result, error) {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	targetNode := raw
	if rootPath != "" {
		parts := strings.Split(rootPath, ".")
		for _, part := range parts {
			if m, ok := targetNode.(map[string]interface{}); ok {
				targetNode = m[part]
			} else {
				targetNode = nil
				break
			}
		}
	}

	itemsSlice, ok := targetNode.([]interface{})
	if !ok {
		return []Result{}, nil
	}

	results := make([]Result, 0, len(itemsSlice))
	for _, item := range itemsSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		res := Result{
			Name:     extractJSONField(itemMap, fieldRules["name"]),
			Size:     extractJSONField(itemMap, fieldRules["size"]),
			Seeds:    extractJSONField(itemMap, fieldRules["seeds"]),
			Peers:    extractJSONField(itemMap, fieldRules["peers"]),
			Magnet:   extractJSONField(itemMap, fieldRules["magnet"]),
			Torrent:  extractJSONField(itemMap, fieldRules["torrent"]),
			URL:      extractJSONField(itemMap, fieldRules["url"]),
			Path:     extractJSONField(itemMap, fieldRules["path"]),
			InfoHash: extractJSONField(itemMap, fieldRules["infohash"]),
			Date:     extractJSONField(itemMap, fieldRules["date"]),
			Category: extractJSONField(itemMap, fieldRules["category"]),
		}

		// If infohash rule was named "hash" in result config
		if res.InfoHash == "" && fieldRules["hash"] != nil {
			res.InfoHash = extractJSONField(itemMap, fieldRules["hash"])
		}

		// Auto construct magnet if only infohash is present
		if res.Magnet == "" && res.InfoHash != "" {
			res.Magnet = BuildMagnetURI(res.InfoHash, res.Name, DefaultTrackers)
		}

		if res.Name != "" || res.Magnet != "" || res.Torrent != "" || res.InfoHash != "" {
			results = append(results, res)
		}
	}

	return results, nil
}

// extractJSONField retrieves a value by dot-notation path from a map.
func extractJSONField(item map[string]interface{}, rule interface{}) string {
	if rule == nil {
		return ""
	}

	path, ok := rule.(string)
	if !ok {
		return ""
	}

	parts := strings.Split(path, ".")
	var current interface{} = item

	for _, part := range parts {
		if current == nil {
			return ""
		}

		// Check if part is an array index
		if idx, err := strconv.Atoi(part); err == nil {
			if arr, ok := current.([]interface{}); ok {
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
					continue
				}
			}
			return ""
		}

		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return ""
		}
	}

	if current == nil {
		return ""
	}

	switch v := current.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		// Check if whole integer
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// BuildMagnetURI creates a magnet link from infohash, name, and trackers.
func BuildMagnetURI(hash string, name string, trackers []string) string {
	var sb strings.Builder
	sb.WriteString("magnet:?xt=urn:btih:")
	sb.WriteString(hash)
	if name != "" {
		sb.WriteString("&dn=")
		sb.WriteString(url.QueryEscape(name))
	}
	for _, tr := range trackers {
		sb.WriteString("&tr=")
		sb.WriteString(url.QueryEscape(tr))
	}
	return sb.String()
}
