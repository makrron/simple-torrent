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
// It supports direct arrays as well as nested array extraction (e.g. "results.*.torrents").
func ExtractJSONResults(data []byte, rootPath string, fieldRules map[string]interface{}) ([]Result, error) {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Check if rootPath indicates nested extraction (e.g. "results.*.torrents" or "results[].torrents")
	parentPath := rootPath
	childPath := ""
	if idx := strings.Index(rootPath, ".*."); idx != -1 {
		parentPath = rootPath[:idx]
		childPath = rootPath[idx+3:]
	} else if idx := strings.Index(rootPath, "[]."); idx != -1 {
		parentPath = rootPath[:idx]
		childPath = rootPath[idx+3:]
	}

	targetNode := navigateJSONNode(raw, parentPath)
	if targetNode == nil {
		return []Result{}, nil
	}

	itemsSlice, ok := targetNode.([]interface{})
	if !ok {
		return []Result{}, nil
	}

	results := make([]Result, 0, len(itemsSlice))
	for _, item := range itemsSlice {
		parentMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if childPath != "" {
			childNode := navigateJSONNode(parentMap, childPath)
			if childSlice, isSlice := childNode.([]interface{}); isSlice && len(childSlice) > 0 {
				for _, childItem := range childSlice {
					childMap, isChildMap := childItem.(map[string]interface{})
					if !isChildMap {
						continue
					}
					if res := buildJSONResult(childMap, parentMap, fieldRules); res != nil {
						results = append(results, *res)
					}
				}
				continue
			}
		}

		if res := buildJSONResult(parentMap, nil, fieldRules); res != nil {
			results = append(results, *res)
		}
	}

	return results, nil
}

func buildJSONResult(item map[string]interface{}, parent map[string]interface{}, fieldRules map[string]interface{}) *Result {
	res := Result{
		Name:     extractJSONField(item, parent, fieldRules["name"]),
		Size:     extractJSONField(item, parent, fieldRules["size"]),
		Seeds:    extractJSONField(item, parent, fieldRules["seeds"]),
		Peers:    extractJSONField(item, parent, fieldRules["peers"]),
		Magnet:   extractJSONField(item, parent, fieldRules["magnet"]),
		Torrent:  extractJSONField(item, parent, fieldRules["torrent"]),
		URL:      extractJSONField(item, parent, fieldRules["url"]),
		Path:     extractJSONField(item, parent, fieldRules["path"]),
		InfoHash: extractJSONField(item, parent, fieldRules["infohash"]),
		Date:     extractJSONField(item, parent, fieldRules["date"]),
		Category: extractJSONField(item, parent, fieldRules["category"]),
	}

	// If infohash rule was named "hash" in result config
	if res.InfoHash == "" && fieldRules["hash"] != nil {
		res.InfoHash = extractJSONField(item, parent, fieldRules["hash"])
	}

	// Auto format size if numeric byte count (e.g. 9005906433 -> 8.39 GB)
	if sizeBytes, err := strconv.ParseInt(res.Size, 10, 64); err == nil && sizeBytes > 1024 {
		res.Size = FormatBytes(sizeBytes)
	}

	// Auto construct magnet if only infohash is present
	if res.Magnet == "" && res.InfoHash != "" {
		res.Magnet = BuildMagnetURI(res.InfoHash, res.Name, DefaultTrackers)
	}

	if res.Name != "" || res.Magnet != "" || res.Torrent != "" || res.InfoHash != "" || res.Path != "" {
		return &res
	}
	return nil
}

// extractJSONField retrieves a value by dot-notation path or slice of paths with parent fallback.
func extractJSONField(item map[string]interface{}, parent map[string]interface{}, rule interface{}) string {
	if rule == nil {
		return ""
	}

	switch v := rule.(type) {
	case string:
		return extractSingleJSONField(item, parent, v)
	case []interface{}:
		for _, p := range v {
			if s, ok := p.(string); ok {
				if val := extractSingleJSONField(item, parent, s); val != "" {
					return val
				}
			}
		}
		return ""
	case []string:
		for _, s := range v {
			if val := extractSingleJSONField(item, parent, s); val != "" {
				return val
			}
		}
		return ""
	default:
		return ""
	}
}

func extractSingleJSONField(item map[string]interface{}, parent map[string]interface{}, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	target := item
	if strings.HasPrefix(path, "_parent.") {
		target = parent
		path = strings.TrimPrefix(path, "_parent.")
	}

	if target == nil {
		return ""
	}

	node := navigateJSONNode(target, path)
	if node == nil {
		return ""
	}

	switch v := node.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
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

func navigateJSONNode(root interface{}, path string) interface{} {
	if path == "" || root == nil {
		return root
	}

	parts := strings.Split(path, ".")
	current := root

	for _, part := range parts {
		if current == nil {
			return nil
		}

		if idx, err := strconv.Atoi(part); err == nil {
			if arr, ok := current.([]interface{}); ok {
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
					continue
				}
			}
			return nil
		}

		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}

// FormatBytes formats byte count into human-readable string (e.g. "8.39 GB")
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit && exp < 5; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
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
