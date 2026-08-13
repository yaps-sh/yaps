package httpheaders

import "strings"

// ETagMatches reports whether any of the If-None-Match header values in
// values matches etag, honoring weak (W/) prefixes and the "*" wildcard.
func ETagMatches(values []string, etag string) bool {
	want := strings.Trim(etag, "\"")
	for _, v := range values {
		for _, tag := range strings.Split(v, ",") {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if tag == "*" {
				return true
			}
			t := strings.TrimPrefix(tag, "W/")
			t = strings.TrimPrefix(t, "w/")
			t = strings.Trim(t, "\"")
			if t == want {
				return true
			}
		}
	}
	return false
}