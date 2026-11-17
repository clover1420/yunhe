package pages

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// HomeDeps are what the home page needs from the app it runs in.
type HomeDeps struct {
	Palette theme.Palette
	Card    Deps
	Sidebar float32

	Featured, Charts, RecPlaylists []qqmusic.Playlist
	HomeLoading                    bool
	HomeErr                        string
	ProfileName                    func() string
	Shelves                        func() (has, busy bool)
	Scroll                         *ui.ScrollState

	Insets    func(*ui.Context) (top, bottom float32)
	StartMark func(*ui.Context)
	EndMark   func(*ui.Context)

	Centered        func(*ui.Context, string, func())
	LoadHome        func()
	LoadSongShelves func()
	ShelfAction     func(*ui.Context, string, bool) ui.Element
	HomeSkeleton    func(*ui.Context)
	DailyCard       func(*ui.Context, time.Time)
	SongShelves     func(*ui.Context, string, int, int)
	// SquareEntry links to the playlist square, at the head of the featured lists.
	SquareEntry func(*ui.Context)
}

// HomePage is what the app opens on: the day's card, songs picked for the
// listener, and playlists.
func HomePage(c *ui.Context, d HomeDeps) {
	hasShelves, _ := d.Shelves()
	// Keep the daily card's first-load space until the rest of Home answers.
	loading := len(d.Featured) == 0 && len(d.Charts) == 0 && (d.HomeLoading || !hasShelves)
	if loading && !d.HomeLoading {
		d.Centered(c, d.HomeErr, d.LoadHome)
		return
	}
	now := c.Now()
	top, bottom := d.Insets(c)
	ui.Scroll(c.Key("home")).TrackScroll(d.Scroll).Grow(1).Padding(top+PageGap, Pad, bottom+48, Pad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		d.StartMark(c)
		defer d.EndMark(c)
		hello := appui.Greeting(now)
		if name := d.ProfileName(); name != "" {
			hello += "，" + name
		}
		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				Heading(c, d.Palette, hello, fmt.Sprintf("%d月%d日 %s", now.Month(), now.Day(), Weekday(now)))
			})
			if d.ProfileName() != "" {
				_, busy := d.Shelves()
				label := "换一批"
				if busy {
					label = "更新中…"
				}
				d.ShelfAction(c, label, busy)
			}
		})
		if loading {
			d.HomeSkeleton(c)
			return
		}
		d.DailyCard(c, now)
		d.SongShelves(c, "song-shelves", 0, 1)

		if len(d.RecPlaylists) > 0 {
			Section(c, "今日为你推荐")
			ui.Grid(c).Columns(Columns(c, d.Sidebar, 168)).GapX(20).GapY(28).Children(func() {
				for i := range d.RecPlaylists[:min(len(d.RecPlaylists), Columns(c, d.Sidebar, 168)*2)] {
					PlaylistCard(c.Key(d.RecPlaylists[i].ID), d.Card, &d.RecPlaylists[i])
				}
			})
		}

		d.SongShelves(c, "more-song-shelves", 1, 3)

		ui.Row(c).Margin(34, 0, 14, 0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "精选歌单").FontSize(19).FontWeight(700).LetterSpacing(-0.3).Grow(1)
			if d.SquareEntry != nil {
				d.SquareEntry(c)
			}
		})
		ui.Grid(c).Columns(Columns(c, d.Sidebar, 168)).GapX(20).GapY(28).Children(func() {
			for i := range d.Featured {
				PlaylistCard(c.Key(d.Featured[i].ID), d.Card, &d.Featured[i])
			}
		})
	})
}

// ChartsDeps are what the charts page needs from the app it runs in.
type ChartsDeps struct {
	Palette theme.Palette
	Card    Deps
	Sidebar float32

	Charts      []qqmusic.Playlist
	HomeLoading bool
	HomeErr     string
	Scroll      *ui.ScrollState

	Insets    func(*ui.Context) (top, bottom float32)
	StartMark func(*ui.Context)
	EndMark   func(*ui.Context)

	Centered func(*ui.Context, string, func())
	LoadHome func()
	Skeleton func(*ui.Context)
}

// ChartsPage lists the charts, the official ones over the rest.
func ChartsPage(c *ui.Context, d ChartsDeps) {
	loading := len(d.Charts) == 0
	if loading && !d.HomeLoading {
		d.Centered(c, d.HomeErr, d.LoadHome)
		return
	}
	top, bottom := d.Insets(c)
	ui.Scroll(c.Key("charts")).TrackScroll(d.Scroll).Grow(1).Padding(top+PageGap, Pad, bottom+48, Pad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		d.StartMark(c)
		defer d.EndMark(c)
		Heading(c, d.Palette, "排行榜", "每日更新")
		if loading {
			d.Skeleton(c)
			return
		}
		official := d.Charts[:min(4, len(d.Charts))]
		cols := min(4, max(2, Columns(c, d.Sidebar, 260)))
		ui.Grid(c).Columns(cols).Gap(20).Children(func() {
			for i := range official {
				ChartCard(c.Key(official[i].ID), d.Card, &official[i])
			}
		})
		if len(d.Charts) > 4 {
			Section(c, "全部榜单")
			rest := d.Charts[4:]
			ui.Grid(c).Columns(Columns(c, d.Sidebar, 150)).GapX(20).GapY(28).Children(func() {
				for i := range rest {
					PlaylistCard(c.Key(rest[i].ID), d.Card, &rest[i])
				}
			})
		}
	})
}
