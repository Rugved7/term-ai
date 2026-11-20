package secrets

import "strings"

// MaskAPIKey returns a masked version of an API key showing only
// the first 4 and last 4 characters.
// Example: "AIzaSyD...x7Q2" for key "AIzaSyDaBcDeFgHiJkLmNoPqRsTuVwXyZ1234567x7Q2"
func MaskAPIKey(key string) string {
	if key == "" {
		return ""
	}

	length := len(key)

	// For very short keys, mask everything
	if length <= 8 {
		return strings.Repeat("*", length)
	}

	// Show first 4 and last 4 characters
	return key[:4] + "..." + key[length-4:]
}
