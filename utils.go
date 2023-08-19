package mcli

import "strings"

func clip[S ~[]E, E any](s S) S {
	return s[:len(s):len(s)]
}

func subSlice[S ~[]E, E any](s S, i, j int) S {
	if j < 0 {
		j = len(s) + j
	}
	if i >= j {
		return nil
	}
	return s[i:j]
}

// find returns the smallest index i at which x == a[i],
// or -1 if there is no such index.
func find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

// contains tells whether a contains x.
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func trimPrefix(s, prefix string) string {
	s = strings.TrimPrefix(s, prefix)
	return strings.TrimSpace(s)
}
