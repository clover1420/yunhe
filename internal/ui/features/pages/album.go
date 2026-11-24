package pages

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/navigation"
	"lilt/internal/media/art"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// AlbumInfo is what the app has of an album.
type AlbumInfo struct {
	Profile *qqmusic.AlbumProfile
	Loading bool
	Err     string
}

// AlbumDeps are what an album's page needs from the app it runs in.
type AlbumDeps struct {
	Palette theme.Palette
	Art     *art.Cache
	Route   *navigation.Route

	Info        func(id int64) AlbumInfo
	Load        func(id int64)
	Centered    func(*ui.Context, string, func())
	Skeleton    func(*ui.Context)
	Tracks      func(*ui.Context, []qqmusic.Track, func(*ui.Context))
	Link        func(*ui.Context, string, ui.Color, ui.Color, navigation.Route)
	MetaRow     func(*ui.Context, *qqmusic.AlbumProfile)
	Description func(*ui.Context, string)
	Play        func([]qqmusic.Track)
	Like        func(*ui.Context, *qqmusic.AlbumProfile)
	Open        func(navigation.Route)
	Columns     func(*ui.Context, float32) int
}

// AlbumPage shows an album: its header, then its songs.
func AlbumPage(c *ui.Context, d AlbumDeps, id int64) {
	info := d.Info(id)
	if info.Profile == nil {
		if !info.Loading {
			d.Centered(c, info.Err, func() { d.Load(id) })
			return
		}
		d.Skeleton(c)
		return
	}
	d.Tracks(c, info.Profile.Songs, func(c *ui.Context) { AlbumHeader(c, d, info.Profile) })
}

// AlbumHeader is an album's cover, its name, artists, release and actions.
func AlbumHeader(c *ui.Context, d AlbumDeps, al *qqmusic.AlbumProfile) {
	p := d.Palette
	ui.Row(c).Gap(28).AlignItems(ui.End).Padding(0, 0, 26, 0).Children(func() {
		ui.Box(c).Size(212, 212).Radius(14).Shadow(0, 18, 40, -10, p.CardShade).Children(func() {
			CoverFill(c, d.Art, p, al.PicURL, 600, 14)
		})
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, "专辑").FontSize(12).FontWeight(600).TextColor(p.Accent)
			ui.Text(c, al.Name).FontSize(30).FontWeight(800).LetterSpacing(-0.6).MaxLines(2).LineHeight(1.2)
			d.MetaRow(c, al)
			d.Description(c, al.Description)
			ui.Row(c).Gap(10).Margin(10, 0, 0, 0).Children(func() {
				if appui.PillButton(c.Key("play"), appui.IconPlay, "播放", p.Accent, p.OnAccent).Clicked() {
					d.Play(al.Songs)
				}
				d.Like(c, al)
			})
		})
	})
}

// AlbumGrid shows albums as cards, each opening its page.
func AlbumGrid(c *ui.Context, d AlbumDeps, albums []qqmusic.AlbumProfile) {
	ui.Grid(c).Columns(d.Columns(c, 156)).GapX(20).GapY(28).Children(func() {
		for i := range albums {
			AlbumCard(c.Key(albums[i].ID), d, &albums[i])
		}
	})
}

// AlbumCard is an album's cover, lifting under the pointer, over its name
// and the year it came out.
func AlbumCard(c *ui.Context, d AlbumDeps, al *qqmusic.AlbumProfile) {
	p := d.Palette
	card := ui.ButtonBase(c).Column().Gap(10).AlignItems(ui.Stretch).FocusRing(false).Label(al.Name)
	hovered := card.Hovered()
	card.Children(func() {
		box := ui.Box(c).AspectRatio(1).Radius(14)
		lift := ui.Pose{}
		if hovered {
			lift = ui.Pose{Y: -3}
			box.Shadow(0, 12, 28, -6, p.CardShade)
		} else {
			box.Shadow(0, 2, 8, -2, p.CardShade)
		}
		box.AnimatePose(lift, ui.Snappy)
		box.Children(func() { CoverFill(c, d.Art, p, al.PicURL, 400, 10) })
		ui.Text(c, al.Name).FontSize(13).FontWeight(500).MaxLines(2).LineHeight(1.35)
		var meta []string
		if d.Route.Page != navigation.Artist {
			meta = append(meta, al.ArtistNames())
		}
		if al.PublishTime > 0 {
			meta = append(meta, time.UnixMilli(al.PublishTime).Format("2006"))
		}
		ui.Text(c, strings.Join(meta, " · ")).FontSize(12).TextColor(p.Muted).SingleLine().Margin(-6, 0, 0, 0)
	})
	if card.Clicked() {
		d.Open(navigation.Route{Page: navigation.Album, ID: al.ID})
	}
}
