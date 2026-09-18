package config

import "strings"

// DefaultFeedURL is intentionally empty: every installation must explicitly
// configure its own product feed through FEED_URL.
const DefaultFeedURL = ""

// NormalizeFeedURL trims the externally configured feed URL.
func NormalizeFeedURL(url string) string {
	return strings.TrimSpace(url)
}
