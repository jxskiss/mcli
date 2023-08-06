package mcli

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

// Find returns the smallest index i at which x == a[i],
// or len(a) if there is no such index.
func find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

// Contains tells whether a contains x.
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// Remove by name
func remove[T comparable](l []T, item T) []T {
	// fmt.Println(l, item)
	for i, other := range l {
		// fmt.Println(i, item, other)
		if other == item {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}
