package generator

import "unicode"

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

func pluralize(name string) string {
	if name == "" {
		return name
	}
	return toSnakeCase(name) + "s"
}
