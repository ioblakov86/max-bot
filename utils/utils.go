package utils

// MaxInt returns the maximum of two integers
func MaxInt(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}