package mcli

import (
	"strings"
)

func trimPrefix(s string, prefix string) string {
	s = strings.TrimPrefix(s, prefix)
	return strings.TrimSpace(s)
}
