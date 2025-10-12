package generator

import (
	"strings"
	"unicode"
)

var commonInitialisms = map[string]struct{}{
	"ACL":   {},
	"API":   {},
	"ASCII": {},
	"CPU":   {},
	"CSS":   {},
	"DNS":   {},
	"EOF":   {},
	"GUID":  {},
	"HTML":  {},
	"HTTP":  {},
	"HTTPS": {},
	"ID":    {},
	"IP":    {},
	"JSON":  {},
	"LHS":   {},
	"QPS":   {},
	"RAM":   {},
	"RHS":   {},
	"RPC":   {},
	"SLA":   {},
	"SMTP":  {},
	"SQL":   {},
	"SSH":   {},
	"TCP":   {},
	"TLS":   {},
	"TTL":   {},
	"UDP":   {},
	"UI":    {},
	"UID":   {},
	"UUID":  {},
	"URI":   {},
	"URL":   {},
	"UTF8":  {},
	"VM":    {},
	"XML":   {},
}

func exportName(name string) string {
	return camelCaseName(name, true)
}

func lowerCamel(name string) string {
	return camelCaseName(name, false)
}

func camelCaseName(name string, upperFirst bool) string {
	if name == "" {
		return ""
	}
	snake := strings.ReplaceAll(toSnakeCase(name), "-", "_")
	raw := strings.Split(snake, "_")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for i, part := range parts {
		upper := strings.ToUpper(part)
		if i == 0 && !upperFirst {
			if _, ok := commonInitialisms[upper]; ok {
				b.WriteString(strings.ToLower(upper))
			} else {
				b.WriteString(strings.ToLower(part))
			}
			continue
		}
		if _, ok := commonInitialisms[upper]; ok {
			b.WriteString(upper)
			continue
		}
		b.WriteString(capitalizeSegment(part))
	}
	return b.String()
}

func capitalizeSegment(segment string) string {
	if segment == "" {
		return segment
	}
	runes := []rune(strings.ToLower(segment))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
