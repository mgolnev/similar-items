package config

import "testing"

func TestNormalizeFeedURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", DefaultFeedURL},
		{"  ", DefaultFeedURL},
		{" https://example.com/feed.xml ", "https://example.com/feed.xml"},
		{"https://example.com/custom.xml", "https://example.com/custom.xml"},
	}
	for _, c := range cases {
		if got := NormalizeFeedURL(c.in); got != c.want {
			t.Fatalf("NormalizeFeedURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
