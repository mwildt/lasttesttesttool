package utils

import (
	"net/http"
	"regexp"
	"strings"
)

func parsePattern(pattern string) (method, path string) {
	parts := strings.SplitN(pattern, "::", 2)
	if len(parts) == 1 {
		return "*", parts[0]
	}
	return parts[0], parts[1]
}

func matchPattern(pattern string, path string) bool {
	rePattern := "^" + regexp.QuoteMeta(pattern) + "$"
	rePattern = strings.ReplaceAll(rePattern, "\\*\\*", ".*") // ** = alles inkl. /
	rePattern = strings.ReplaceAll(rePattern, "\\*", "[^/]*") // * = alles außer /
	rePattern = strings.ReplaceAll(rePattern, "\\?", ".")     // ? = ein Zeichen
	re := regexp.MustCompile(rePattern)
	return re.MatchString(path)
}

func Match(pattern string, request *http.Request) bool {
	method, path := parsePattern(pattern)
	if method != "*" && request.Method != method {
		return false
	}
	return matchPattern(path, request.URL.Path)
}
