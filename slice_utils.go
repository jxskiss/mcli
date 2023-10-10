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
func contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
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

func removeCommandName(args []string, cmdName string) []string {
	cmdWords := strings.Fields(cmdName)
	i := 0
	for ; i < len(cmdWords) && i < len(args); i++ {
		if cmdWords[i] != args[i] {
			break
		}
	}
	return args[i:]
}
