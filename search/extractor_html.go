package search

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ExtractHTMLResults parses an HTML document and returns a slice of Results based on provider config.
func ExtractHTMLResults(htmlData []byte, listSelector string, fieldRules map[string]interface{}) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	results := []Result{}
	rows := doc.Find(listSelector)
	rows.Each(func(i int, s *goquery.Selection) {
		res := Result{
			Name:     extractField(s, fieldRules["name"]),
			Size:     extractField(s, fieldRules["size"]),
			Seeds:    extractField(s, fieldRules["seeds"]),
			Peers:    extractField(s, fieldRules["peers"]),
			Magnet:   extractField(s, fieldRules["magnet"]),
			Torrent:  extractField(s, fieldRules["torrent"]),
			URL:      extractField(s, fieldRules["url"]),
			Path:     extractField(s, fieldRules["path"]),
			InfoHash: extractField(s, fieldRules["infohash"]),
			Tracker:  extractField(s, fieldRules["tracker"]),
			Date:     extractField(s, fieldRules["date"]),
			Category: extractField(s, fieldRules["category"]),
		}

		if res.Path == "" && res.URL != "" && strings.HasPrefix(res.URL, "/") {
			res.Path = res.URL
		}

		// Only include result if it has at least a name or magnet/path
		if res.Name != "" || res.Magnet != "" || res.Torrent != "" || res.Path != "" {
			results = append(results, res)
		}
	})

	return results, nil
}

// ExtractHTMLItemDetail parses an HTML document for item detail fields (magnet, torrent, infohash, tracker).
func ExtractHTMLItemDetail(htmlData []byte, fieldRules map[string]interface{}) (*ItemDetail, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse item HTML: %w", err)
	}

	root := doc.Selection
	detail := &ItemDetail{
		Magnet:   extractField(root, fieldRules["magnet"]),
		Torrent:  extractField(root, fieldRules["torrent"]),
		InfoHash: extractField(root, fieldRules["infohash"]),
		Tracker:  extractField(root, fieldRules["tracker"]),
	}

	return detail, nil
}

// extractField resolves a field rule on a selection.
// Rule can be:
// - string: CSS selector or attribute shorthand (e.g. "a@href" or "font.detDesc")
// - []interface{}: sequence of transformations or fallback selectors
func extractField(sel *goquery.Selection, rule interface{}) string {
	if rule == nil || sel == nil {
		return ""
	}

	switch v := rule.(type) {
	case string:
		return evaluateStringRule(sel, v)
	case []interface{}:
		return evaluateSliceRule(sel, v)
	case []string:
		// Convert to []interface{}
		converted := make([]interface{}, len(v))
		for i, s := range v {
			converted[i] = s
		}
		return evaluateSliceRule(sel, converted)
	default:
		return ""
	}
}

func evaluateStringRule(sel *goquery.Selection, rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return ""
	}

	// Attribute shorthand: "selector@attr"
	if atIdx := strings.LastIndex(rule, "@"); atIdx > 0 && !strings.Contains(rule[atIdx:], " ") && !strings.Contains(rule[atIdx:], "]") {
		cssSelector := strings.TrimSpace(rule[:atIdx])
		attrName := strings.TrimSpace(rule[atIdx+1:])
		target := sel
		if cssSelector != "" {
			target = sel.Find(cssSelector)
		}
		val, exists := target.Attr(attrName)
		if exists {
			return strings.TrimSpace(val)
		}
		return ""
	}

	// Just a CSS selector: extract text
	sub := sel.Find(rule)
	if sub.Length() > 0 {
		return strings.TrimSpace(sub.First().Text())
	}
	return ""
}

func evaluateSliceRule(sel *goquery.Selection, parts []interface{}) string {
	if len(parts) == 0 {
		return ""
	}

	// Check if this is a list of alternative fallback selectors (all strings starting with selector-like chars, not @ or /)
	isAlternativeList := true
	for _, p := range parts {
		str, ok := p.(string)
		if !ok || strings.HasPrefix(str, "@") || (strings.HasPrefix(str, "/") && strings.HasSuffix(str, "/")) || strings.HasPrefix(str, "s/") {
			isAlternativeList = false
			break
		}
	}

	if isAlternativeList {
		for _, p := range parts {
			val := evaluateStringRule(sel, p.(string))
			if val != "" {
				return val
			}
		}
		return ""
	}

	// Otherwise, it's a pipeline: [selector, operation, operation, ...]
	currentSel := sel
	currentVal := ""
	hasVal := false

	for i, part := range parts {
		str, ok := part.(string)
		if !ok {
			continue
		}
		str = strings.TrimSpace(str)

		if i == 0 {
			if strings.HasPrefix(str, "@") {
				// extract attribute from currentSel
				attrName := strings.TrimPrefix(str, "@")
				val, exists := currentSel.Attr(attrName)
				if exists {
					currentVal = strings.TrimSpace(val)
					hasVal = true
				}
			} else if strings.HasPrefix(str, "/") && strings.HasSuffix(str, "/") {
				// regex on current selection text
				text := currentSel.Text()
				currentVal = applyRegexExtract(text, str)
				hasVal = true
			} else {
				// selector
				currentSel = currentSel.Find(str)
			}
			continue
		}

		// Subsequent pipeline operations
		if strings.HasPrefix(str, "@") {
			attrName := strings.TrimPrefix(str, "@")
			val, exists := currentSel.Attr(attrName)
			if exists {
				currentVal = strings.TrimSpace(val)
				hasVal = true
			}
		} else if strings.HasPrefix(str, "/") && strings.HasSuffix(str, "/") {
			text := currentVal
			if !hasVal {
				text = currentSel.Text()
			}
			currentVal = applyRegexExtract(text, str)
			hasVal = true
		} else if strings.HasPrefix(str, "s/") {
			text := currentVal
			if !hasVal {
				text = currentSel.Text()
			}
			currentVal = applyRegexReplace(text, str)
			hasVal = true
		} else {
			// Sub-selector
			currentSel = currentSel.Find(str)
		}
	}

	if !hasVal && currentSel != nil && currentSel.Length() > 0 {
		currentVal = strings.TrimSpace(currentSel.First().Text())
	}

	return currentVal
}

func applyRegexExtract(text string, pattern string) string {
	rawPattern := pattern[1 : len(pattern)-1]
	re, err := regexp.Compile(rawPattern)
	if err != nil {
		return text
	}

	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	} else if len(matches) == 1 {
		return strings.TrimSpace(matches[0])
	}
	return ""
}

func applyRegexReplace(text string, replaceExpr string) string {
	// Format: s/find/replace/flags (e.g. s/[\n\t]+//g)
	parts := strings.Split(replaceExpr, "/")
	if len(parts) < 3 {
		return text
	}
	find := parts[1]
	replace := parts[2]

	re, err := regexp.Compile(find)
	if err != nil {
		return text
	}
	return strings.TrimSpace(re.ReplaceAllString(text, replace))
}
