package utils

import (
	"strings"
	"unicode"
)

// Converts header names to camelCase for DynamoDB attribute names
func ToCamelCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	words := strings.Fields(strings.ReplaceAll(s, "_", " "))
	if len(words) == 0 {
		return ""
	}

	result := strings.ToLower(words[0])
	for _, w := range words[1:] {
		if w == "" {
			continue
		}
		runes := []rune(strings.ToLower(w))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		result += string(runes)
	}

	return result
}
