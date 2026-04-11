package kleiogithub

// shortSHA returns up to the first 8 characters of a Git SHA for logging (avoids panics on short strings).
func shortSHA(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}
