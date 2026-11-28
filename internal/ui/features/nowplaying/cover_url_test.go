package nowplaying

import (
	"strings"
	"testing"
)

// The cover is the original unless a thumbnail size is chosen.
func TestCoverURL(t *testing.T) {
	pic := "https://y.gtimg.cn/music/photo_new/T002R500x500M000album.jpg?max_age=2592000"
	for _, tc := range []struct{ quality, want string }{
		{"", "T002M000album"},
		{"original", "T002M000album"},
		{"800", "T002R800x800M000album"},
		{"500", "T002R500x500M000album"},
		{"300", "T002R300x300M000album"},
	} {
		if got := coverURL(pic, tc.quality); !strings.Contains(got, tc.want) {
			t.Errorf("coverURL(%q) = %q, want it to contain %q", tc.quality, got, tc.want)
		}
	}
}
