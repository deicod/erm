package generator

import (
	"sort"
	"strings"
	"unicode"
)

var identifierSanitizer = strings.NewReplacer("-", "_", " ", "_")

func toSnakeCase(in string) string {
	if in == "" {
		return in
	}
	runes := []rune(in)
	out := make([]rune, 0, len(runes)*2)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func normalizeIdentifier(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	cleaned := identifierSanitizer.Replace(trimmed)
	cleaned = strings.Trim(cleaned, "_")
	if cleaned == "" {
		return ""
	}
	return toSnakeCase(cleaned)
}

func pluralize(name string) string {
	if name == "" {
		return name
	}
	word := toSnakeCase(name)
	if word == "" {
		return word
	}

	irregular := map[string]string{
		"person": "people",
		"man":    "men",
		"woman":  "women",
		"child":  "children",
		"tooth":  "teeth",
		"foot":   "feet",
		"mouse":  "mice",
		"goose":  "geese",
	}
	if plural, ok := irregular[word]; ok {
		return plural
	}

	alreadyPlural := []string{"ies", "ses", "xes", "zes", "ches", "shes"}
	for _, suffix := range alreadyPlural {
		if strings.HasSuffix(word, suffix) {
			return word
		}
	}

	if strings.HasSuffix(word, "y") && len(word) > 1 {
		prev := rune(word[len(word)-2])
		if !strings.ContainsRune("aeiou", prev) {
			return word[:len(word)-1] + "ies"
		}
	}

	if strings.HasSuffix(word, "fe") {
		return word[:len(word)-2] + "ves"
	}
	if strings.HasSuffix(word, "f") && len(word) > 1 && word[len(word)-2] != 'f' {
		return word[:len(word)-1] + "ves"
	}

	if strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") {
		return word + "es"
	}

	if strings.HasSuffix(word, "s") {
		if strings.HasSuffix(word, "ss") || strings.HasSuffix(word, "us") || strings.HasSuffix(word, "is") {
			return word + "es"
		}
		return word
	}

	if strings.HasSuffix(word, "o") && len(word) > 1 && !strings.ContainsRune("aeiou", rune(word[len(word)-2])) {
		return word + "es"
	}

	return word + "s"
}

func defaultJoinTableName(left, right string) string {
	parts := []string{pluralize(left), pluralize(right)}
	sort.Strings(parts)
	return strings.Join(parts, "_")
}
