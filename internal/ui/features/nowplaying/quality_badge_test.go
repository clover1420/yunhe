package nowplaying

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
)

func TestQualityBadgeUsesActualStreamLevel(t *testing.T) {
	st := &qqmusic.Stream{}
	tt := ui.NewTester(func(c *ui.Context) { qualityBadge(c, st) }, 200, 80)
	for _, level := range []string{"hires", "lossless", "standard"} {
		st.Level = level
		tt.Frame()
		_, hiRes := tt.Find("Hi-Res")
		_, lossless := tt.Find("无损")
		if hiRes != (level == "hires") || lossless != (level == "lossless") {
			t.Fatalf("incorrect badge for %s", level)
		}
	}
}
