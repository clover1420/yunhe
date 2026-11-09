package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/features/pages"
)

// The record follows song changes using the player bar's cached cover, with a daily fallback.
func TestDailyRecordCurrentCover(t *testing.T) {
	a := newTestApp()
	tracks := []qqmusic.Track{
		{ID: 1, Album: qqmusic.Album{PicURL: "https://y.gtimg.cn/music/photo_new/T002R500x500M000first.jpg"}},
		{ID: 2, Album: qqmusic.Album{PicURL: "https://y.gtimg.cn/music/photo_new/T002R500x500M000second.jpg"}},
		{ID: 3},
	}
	now := time.Unix(1000, 0)
	a.art.SetClock(func() time.Time { return now })
	for _, tr := range tracks[:2] {
		a.art.Add(qqmusic.Sized(tr.Album.PicURL, 120), &artwork{})
	}
	// Keep the sleeve and the old label size local too, so a regression cannot fetch.
	for _, px := range []int{240, 400} {
		a.art.Add(qqmusic.Sized(tracks[0].Album.PicURL, px), &artwork{})
	}
	tt := ui.NewTester(func(c *ui.Context) {
		pages.DailyStack(c, a.dailyDeps(), tracks[:1], false, 1)
	}, stackWidth, sleeveSize)
	for _, tc := range []struct {
		name string
		at   int
		want int
	}{
		{"no current song", -1, 0},
		{"first song", 0, 0},
		{"changed song while paused", 1, 1},
		{"missing current cover", 2, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.at >= 0 {
				a.pl.Cue(tracks[tc.at : tc.at+1])
			}
			now = now.Add(time.Second)
			tt.Frame()
			for i, tr := range tracks[:2] {
				art := a.art.Peek(qqmusic.Sized(tr.Album.PicURL, 120))
				if got := art.Used.Equal(now); got != (i == tc.want) {
					t.Errorf("song %d cover used=%t, want %t", i, got, i == tc.want)
				}
			}
			if a.art.Busy() != 0 {
				t.Fatal("record requested an uncached image despite the player cover being cached")
			}
		})
	}
}
