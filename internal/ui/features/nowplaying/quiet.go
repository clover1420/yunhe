package nowplaying

import (
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/lyrics"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
)

// Through the quiet parts of a song, its intro, an interlude or its outro,
// the lyrics dim, and one of its hot comments (热评) rises in where they
// were, down the lyrics pane or in the middle of the stage, with who wrote
// it and how many liked it, giving way as the singing comes back. A long
// quiet shows a few, one after another; a song without lyrics shows them all
// along. A click on one opens the comments. The listener turns them on
// (settings.HotComments).

const (
	quietLeast = 8 * time.Second         // the shortest quiet a comment shows in
	quietLead  = 1200 * time.Millisecond // into the quiet before it shows
	quietTail  = 1000 * time.Millisecond // before the singing, it is gone
	quietSlot  = 10 * time.Second        // about how long each shows in a long quiet
	quietRunes = 140                     // the longest comment shown
)

// QuietState is the quiet parts of the song Now Playing shows, and the
// comment shown in one, kept as it fades away.
type QuietState struct {
	Track  int64
	Lines  int // how many lines they were found among
	Loaded bool
	Quiets []lyrics.Quiet
	Cm     *qqmusic.Comment
}

// QuietComment returns the hot comment to show at pos, nil for none, and
// how long until that changes, 0 for never.
func QuietComment(d Deps, tr *qqmusic.Track, lines []lyrics.Line, loaded bool, pos time.Duration) (*qqmusic.Comment, time.Duration) {
	if !d.Settings.HotComments {
		return nil, 0
	}
	qs := d.Quiet
	if qs.Track != tr.ID || qs.Lines != len(lines) || qs.Loaded != loaded || qs.Quiets == nil {
		qs.Track, qs.Lines, qs.Loaded = tr.ID, len(lines), loaded
		qs.Quiets = lyrics.Quiets(lines, tr.Length(), quietLeast)
		if len(lines) == 0 && loaded && tr.Length() >= quietLeast {
			qs.Quiets = []lyrics.Quiet{{From: 0, To: tr.Length()}}
		}
		if qs.Quiets == nil {
			qs.Quiets = []lyrics.Quiet{}
		}
	}
	var hot []*qqmusic.Comment
	if cs := d.Comments; cs != nil {
		h := cs.Hot()
		for i := range h {
			if n := utf8.RuneCountInString(h[i].Content); n > 0 && n <= quietRunes {
				hot = append(hot, &h[i])
			}
		}
	}
	if len(hot) == 0 {
		return nil, 0
	}
	shown := 0 // the comments the quiets before showed
	for _, q := range qs.Quiets {
		from, to := q.From+quietLead, q.To-quietTail
		if pos < from {
			return nil, from - pos
		}
		// As many as fit about a slot each, sharing the quiet evenly.
		slots := max(1, int((to-from+quietSlot/2)/quietSlot))
		each := (to - from) / time.Duration(slots)
		if pos < to {
			slot := min(int((pos-from)/each), slots-1)
			next := to
			if slot+1 < slots {
				next = from + time.Duration(slot+1)*each
			}
			return hot[(shown+slot)%len(hot)], next - pos
		}
		shown += slots
	}
	return nil, 0
}

// QuietShown returns how far a hot comment has come in on el, the lyrics,
// from 0 to 1, keeping it to show as it goes, and whether one shows now.
func QuietShown(c *ui.Context, d Deps, el ui.Element, tr *qqmusic.Track, lines []lyrics.Line, loaded bool) (float32, bool) {
	pl := d.Player
	cm, next := QuietComment(d, tr, lines, loaded, pl.Position())
	if pl.Playing() && next > 0 {
		c.After(next + 20*time.Millisecond)
	}
	to := float32(0)
	if cm != nil {
		to = 1
	}
	dim := el.AnimateTo("quiet", to, ui.Animation{Duration: 600 * time.Millisecond})
	if cm != nil {
		d.Quiet.Cm = cm
	}
	return dim, cm != nil
}

// QuietCard shows the hot comment kept, top down the pane, as far in as
// dim.
func QuietCard(c *ui.Context, d Deps, dim, top float32) {
	hotCard(c, d, dim, 22, func(card ui.Element) { card.Absolute().Top(top).Left(0).Right(24) })
}

// StageQuiet shows the hot comment kept in the middle of the stage, w wide,
// its lines sung at f, as far in as dim.
func StageQuiet(c *ui.Context, d Deps, dim, f, w float32) {
	if dim <= 0.01 || d.Quiet.Cm == nil {
		return
	}
	ui.Column(c.Key("quiet-stage")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().PassThrough().Children(func() {
		hotCard(c, d, dim, min(34, max(24, f*0.55)), func(card ui.Element) { card.Width(min(760, w*0.6)) })
	})
}

// hotCard shows the hot comment kept, its words at size, as far in as dim;
// place puts it where it goes.
func hotCard(c *ui.Context, d Deps, dim, size float32, place func(ui.Element)) {
	cm := d.Quiet.Cm
	if dim <= 0.01 || cm == nil {
		return
	}
	card := ui.ButtonBase(c.Key("quiet")).Column().AlignItems(ui.Start).
		Padding(16, 12).Gap(14).Radius(14).FocusRing(false).Label("查看评论")
	place(card)
	// Each comment rises in on its own as it takes the last one's place.
	seen := ui.Local(card, "seen", func() int64 { return 0 })
	if *seen != cm.ID {
		*seen = cm.ID
		card.Jump("in", 0)
	}
	in := card.AnimateTo("in", 1, ui.Animation{Spring: ui.SpringOptions{Duration: 700 * time.Millisecond, Bounce: 0.1}})
	card.Opacity(min(dim, max(0, in))).Translate(0, (1-in)*16)
	if card.Hovered() {
		card.Background(white.Alpha(0.06))
	}
	card.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(white.Alpha(0.5)).Children(func() {
			ui.Icon(c, appui.IconComment).Size(14, 14)
			ui.Text(c, "热评").FontSize(12.5).FontWeight(700).LetterSpacing(0.6)
		})
		ui.Text(c, cm.Content).FontSize(size).FontWeight(600).LineHeight(1.5).TextColor(white.Alpha(0.92)).MaxLines(6)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			d.Avatar(c, cm.User.AvatarURL, 22, white.Alpha(0.1))
			ui.Text(c, cm.User.Nickname).FontSize(13).TextColor(white.Alpha(0.6)).SingleLine().Shrink(1).MinWidth(0)
			if cm.LikedCount > 0 {
				ui.Row(c).Gap(4).AlignItems(ui.Center).Margin(0, 0, 0, 6).TextColor(white.Alpha(0.45)).Children(func() {
					ui.Icon(c, appui.IconThumb).Size(13, 13)
					ui.Text(c, appui.Count(cm.LikedCount)).FontSize(12).FontFeatures("tnum")
				})
			}
		})
	})
	if card.Clicked() {
		d.OpenComments()
	}
}
