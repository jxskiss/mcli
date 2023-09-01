package mcli

import (
	"strings"
)

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

func trimPrefix(s, prefix string) string {
	s = strings.TrimPrefix(s, prefix)
	return strings.TrimSpace(s)
}

// reverse passed slice using generics
func reverse[T any](original []T) (reversed []T) {
	reversed = make([]T, len(original))
	copy(reversed, original)

	for i := len(reversed)/2 - 1; i >= 0; i-- {
		tmp := len(reversed) - 1 - i
		reversed[i], reversed[tmp] = reversed[tmp], reversed[i]
	}

	return
}
