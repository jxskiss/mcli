package mcli

var isExampleTest = false
var isTesting = false

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
