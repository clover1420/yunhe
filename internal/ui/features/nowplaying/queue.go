package nowplaying

import (
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// QueueDeps are what the queue's list needs of the app it draws in.
type QueueDeps struct {
	Player      *player.Player
	Cover       func(*ui.Context, string, int, float32, float32) ui.Element
	ArtistLinks func(*ui.Context, *qqmusic.Track, ui.Color, ui.Color, ...bool) ui.Element
	SongItems      func(*qqmusic.Track, int64) func(*ui.Menu)
	QueueSongItems func(*qqmusic.Track, int) func(*ui.Menu)
}

// QueuePane lists the whole queue in play order, the track playing marked
// among the others: a click plays one there, and the list stays as it is.
func QueuePane(c *ui.Context, d Deps) {
	QueueTracks(c, queueDeps(d), d.QueueList, d.QueueAt, theme.Palette{
		Text: white, Muted: white.Alpha(0.55), Faint: white.Alpha(0.4),
		Hover: white.Alpha(0.08), Press: white.Alpha(0.12),
	}).Padding(8, 0, 40, 0)
}

func queueDeps(d Deps) QueueDeps {
	return QueueDeps{Player: d.Player, Cover: d.Cover, ArtistLinks: d.ArtistLinks, SongItems: d.SongItems, QueueSongItems: d.QueueSongItems}
}

// QueueTracks shares playback and menus between the page and its floating queue.
func QueueTracks(c *ui.Context, q QueueDeps, list *ui.ListState, lastAt *int, p theme.Palette) ui.Element {
	pl := q.Player
	order, at := pl.PlayOrder()
	// Opened, it shows the track playing at its top; after, it follows it
	// as little as keeps it in view.
	if at >= 0 && at != *lastAt {
		if *lastAt < 0 {
			list.ScrollTo(at, ui.Start)
		} else {
			list.ScrollIntoView(at)
		}
		*lastAt = at
	}
	list.Key = func(row int) any { return order[row] }
	return ui.List(c.Key("queue"), list, len(order), func(r int) {
		i := order[r]
		tr := &pl.Queue()[i]
		playing := r == at
		menu := q.SongItems(tr, 0)
		if q.QueueSongItems != nil {
			menu = q.QueueSongItems(tr, i)
		}
		row := ui.ButtonBase(c.Key(i)).Height(58).PaddingX(10).Gap(14).Radius(14).AlignItems(ui.Center).FocusRing(false).Label(tr.Name).
			ContextMenu(menu)
		switch {
		case playing:
			row.Background(p.Press)
		case row.Hovered():
			row.Background(p.Hover)
		}
		if !tr.Playable() {
			row.Opacity(0.4)
		}
		row.Children(func() {
			q.Cover(c, tr.Album.PicURL, 96, 42, 7)
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				ui.Text(c, tr.Name).FontSize(14).FontWeight(600).SingleLine()
				q.ArtistLinks(c, tr, p.Muted, p.Text).FontSize(12.5).SingleLine()
			})
			ui.Text(c, appui.Clock(tr.Length())).FontSize(12).FontFeatures("tnum").TextColor(p.Faint)
		})
		if row.Clicked() && tr.Playable() {
			if playing {
				pl.TogglePlay()
			} else {
				pl.Jump(i)
			}
		}
	}).Grow(1).MinHeight(0).TextColor(p.Text).Gap(2)
}
