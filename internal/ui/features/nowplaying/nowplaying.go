// Package nowplaying owns the view that covers the window with what plays:
// its cover over a backdrop of its own colors, the controls, the tabs, and
// the panes beside the cover. The panes' contents live elsewhere, and are
// called through Deps.
package nowplaying

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/media/art"
	"lilt/internal/media/captions"
	"lilt/internal/media/lyrics"
	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
	"lilt/internal/ui/base/transport"
	"lilt/internal/ui/features/playerbar"
)

var white = theme.Paper

var npSpring = ui.SpringOptions{Duration: 340 * time.Millisecond}

// The panes of Now Playing.
const (
	PaneLyrics = iota
	PaneQueue
	PaneStage
	PaneComments
	PaneProgram
	PaneCaptions
	PaneWall
)

// Deps are what Now Playing needs from the app it runs in: the player and
// the art it reads directly, the listener's state by pointer, and the panes
// that live elsewhere by callback.
type Deps struct {
	Player   *player.Player
	Settings *settings.Settings
	Art      *art.Cache
	NpScrub  *transport.State

	NowPlaying    *bool
	Open          *bool
	Motion        *appui.NowPlayingTransition
	Pane          *int
	PaneTop       *float32
	NPArt         **art.Artwork
	BackdropCache *BackdropCache
	GlowColor     *ui.Color
	CoverState    *CoverState
	Volume        *float64
	MutedFrom     *float64
	QueueList     *ui.ListState
	QueueAt       *int
	LyricScroll   *ui.ScrollState
	LyricGlide    *Glide
	LyricAt       *int
	LyricTrack    *int64
	LyricSize     *float32
	LyricSizeAt   *time.Time

	StageStyleOpen    *bool
	LyricsSizeOpen    *bool
	LyricsSizeTrigger *ui.Handle
	LyricsSizeChanged func()

	TopBar func(*ui.Context) float32

	SetButtonsTheme func(dark bool)
	ClearBackdrop   func()
	ResetPanes      func()
	ResetQueue      func()
	ResetWall       func()

	Backdrop         func(*ui.Context, *art.Artwork, ui.Color)
	CommentsFor      func(*qqmusic.Track)
	CaptionsOn       func() bool
	CaptionsShown    func() bool
	LyricsPaneChosen func() int

	PaneTabs func(*ui.Context)

	RadioChip   func(*ui.Context, string)
	ArtistLinks func(*ui.Context, *qqmusic.Track, ui.Color, ui.Color, ...bool) ui.Element
	Like        func(*ui.Context, *qqmusic.Track, float32, ui.Color, ...appui.ButtonSurface) ui.Element
	SongMenu    func(*ui.Context, *qqmusic.Track, float32, ui.Color, ...appui.ButtonSurface) ui.Element
	Cover       func(*ui.Context, string, int, float32, float32) ui.Element
	SongItems   func(*qqmusic.Track, int64) func(*ui.Menu)
	Avatar      func(*ui.Context, string, float32, ui.Color)

	Quiet         *QuietState
	QuietShown    func(*ui.Context, ui.Element, *qqmusic.Track, []lyrics.Line, bool) (float32, bool)
	QuietCard     func(*ui.Context, float32, float32)
	OpenComments  func()
	Skeleton      func(*ui.Context)
	StageQuiet    func(*ui.Context, float32, float32, float32)
	StageSkeleton func(*ui.Context, float32, float32)
	BackdropOver  func(*ui.Painter, ui.Rect, ui.Rect, func(float32) float32)

	Wall *WallState

	Comments          CommentFeed
	LoadComments      func()
	ToggleCommentLike func(int64)
	CommentSkeleton   func(*ui.Context, int)
	MoreSkeleton      func(*ui.Context)
	Accent            ui.Color

	Captions            *captions.Model
	SyncCaptions        func()
	StartModelDownload  func()
	CancelModelDownload func()
}

// State tracks Now Playing opening and closing, and reports whether it is
// open and whether it has come to rest, open or closed. Call it once a
// build, before the window's content.
func State(c *ui.Context, d Deps) (open, settled bool) {
	open = *d.NowPlaying && d.Player.Current() != nil
	if open != *d.Open {
		*d.Open = open
		// The window buttons light over its dark backdrop, and as the
		// theme over the pages (Windows).
		d.SetButtonsTheme(open)
	}
	return open, d.Motion.AtRest(open)
}

// Layer grows Now Playing out of the bar's cover and shrinks it back there,
// keeping the whole card over both the page and the sidebar. A pull follows
// the pointer, closing when let go far or fast enough.
func Layer(c *ui.Context, d Deps, open, settled bool) {
	m := d.Motion
	w, h := c.Size()
	layer := ui.Box(c.Key("now-playing-layer")).Absolute().Top(0).Left(0).Right(0).Bottom(0)
	m.Still = c.Preferences().ReduceMotion
	spring := npSpring
	_, dy, held := layer.Dragged()
	held = held && open
	if held {
		m.Pull += dy
	}
	if _, vy, ok := layer.DragEnded(); ok && open {
		if m.Pull > h*0.2 || m.Pull > 24 && vy > 700 {
			*d.NowPlaying, open = false, false
			spring.Velocity = -max(vy, 0) / h
		}
	}
	if !held {
		m.Pull = 0
	}
	m.Held = held
	target := float32(0)
	if open {
		target = 1
	}
	var t, shift float32
	if m.Still {
		// Use its alpha for a fade without depending on color animation.
		fade := layer.AnimateColor("fade", white.Alpha(target), ui.Animation{Duration: 200 * time.Millisecond})
		t = float32(fade.A) / 255
	} else {
		t = layer.AnimateTo("open", target, ui.Animation{Spring: spring})
	}
	if held {
		shift = layer.Jump("pull", ui.Rubberband(m.Pull, 0, h, h))
	} else {
		shift = layer.AnimateTo("pull", 0, ui.Animation{Spring: npSpring})
	}
	fresh, was := m.T <= 0 && t > 0, m.T
	m.T, m.Shift = t, shift
	if m.AtRest(open) != settled {
		c.AnimationFrame() // let the covered page and bar come or go
	}
	if t <= 0 && !open && was <= 0 || d.Player.Current() == nil {
		// Closed: nothing to paint or click.
		// Its backdrop goes with it: the canvas, and the GPU
		// texture it holds, would otherwise stay for the rest of
		// the session, invisible.
		d.ClearBackdrop()
		layer.Invisible()
		d.ResetPanes()
		return
	}
	if fresh {
		// Lay out the cover before its first visible frame.
		layer.Opacity(0)
		c.AnimationFrame()
	} else if m.Still {
		layer.Opacity(max(t, 0))
	}
	if !open {
		// On its way out, the window under it takes the clicks.
		layer.IgnoreInput(true)
	}
	layer.Children(func() {
		card := ui.Box(c.Key("card")).Absolute().Top(0).Left(0).Right(0).Bottom(0)
		var x, y float32
		if !m.Still {
			k := 1 - clamp01(t)
			from := m.FromRect()
			x, y = from.X*k, from.Y*k
			card.Top(y).Left(x).Right((w - from.X - from.W) * k).Bottom((h - from.Y - from.H) * k).
				Radius(appui.NPFromRadius*k + 28*clamp01(shift/120)).Clip()
			if shift != 0 {
				s := 1 - 0.12*clamp01(shift/h)
				card.TransformOrigin(0.5, 0).Scale(s, s).Translate(0, shift)
			}
		}
		card.Children(func() {
			ui.Box(c.Key("canvas")).Absolute().Top(-y).Left(-x).Size(w, h).
				Children(func() { View(c, d) })
		})
	})
}

// View fills the window with what plays: its cover over a slow drift of its
// own colors, the controls, and lyrics that follow the song.
func View(c *ui.Context, d Deps) {
	pl := d.Player
	tr := pl.Current()
	// The covered player bar stops building once this layer settles, so
	// keep our time labels ticking independently of paint-only animations.
	transport.TickTimes(c, pl)
	w, h := c.Size()
	art := d.Art.Get(qqmusic.Sized(tr.Album.PicURL, 600), art.Backdrop)
	// The colors stay the last song's while its cover loads, rather than
	// going gray between the two.
	shown := art
	if art == nil && tr.Album.PicURL != "" {
		shown = *d.NPArt
	}
	*d.NPArt = shown
	tint := theme.Neutral(true, 9)
	if shown != nil && !shown.Failed {
		tint = shown.Tint
	}
	accent := tint.Mix(white, 0.45)
	// The light of the line sung takes the cover's own color, lightened as
	// the lyrics beside it are.
	*d.GlowColor = accent

	bg := theme.Neutral(true, 1)
	if t := d.Motion.T; t < 1 && !d.Motion.Still {
		bg = bg.Alpha(smoothstep(0, 0.35, t))
	}
	root := ui.Box(c.Key("now-playing")).Absolute().Top(0).Left(0).Right(0).Bottom(0).
		Background(bg).TextColor(white)
	// Keep the controls visible while the pointer is in their detached menu
	// or size popover. The pointer is over the view whenever it is in the window.
	pointerIn := root.Hovered() || *d.StageStyleOpen || *d.LyricsSizeOpen || d.LyricsSizeTrigger.Focused(c)
	root.Children(func() {
		bloom := ui.Box(c.Key("bloom")).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough()
		if t := d.Motion.T; t < 1 && !d.Motion.Still {
			from := d.Motion.FromRect()
			s := 1 + 0.3*(1-clamp01(t))
			bloom.Opacity(smoothstep(0.05, 0.55, t)).
				TransformOrigin((from.X+from.W/2)/w, (from.Y+from.H/2)/h).Scale(s, s)
		}
		bloom.Children(func() { d.Backdrop(c, shown, tint) })
		// Over the backdrop, which is positioned too: positioned elements
		// paint over those in the flow.
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Children(func() { body(c, d, tr, art, accent, w, h, pointerIn) })
	})
}

func body(c *ui.Context, d Deps, tr *qqmusic.Track, art *art.Artwork, accent ui.Color, w, h float32, pointerIn bool) {
	pl := d.Player
	d.CommentsFor(tr)
	// An episode has what it is about and its captions in place of lyrics;
	// a song, lyrics in place of those.
	switch program := tr.ProgramID != 0; {
	case program && *d.Pane == PaneWall:
		*d.Pane = PaneQueue
	case program && (*d.Pane == PaneLyrics || *d.Pane == PaneStage):
		*d.Pane = PaneProgram
		if d.CaptionsOn() {
			*d.Pane = PaneCaptions
		}
	case program && *d.Pane == PaneCaptions && !d.CaptionsShown():
		*d.Pane = PaneProgram
	case !program && (*d.Pane == PaneProgram || *d.Pane == PaneCaptions):
		*d.Pane = d.LyricsPaneChosen()
	}
	if *d.Pane != PaneQueue {
		d.ResetQueue()
	}
	if *d.Pane != PaneWall {
		d.ResetWall()
	}
	{
		m := d.Motion
		// The top strip drags the window; the chevron closes the view,
		// showing only while the pointer is in the window, as the pane tabs do.
		m.Rise(ui.Row(c).Height(d.TopBar(c)).Justify(ui.Center).AlignItems(ui.Center).DragWindow(), 0).Children(func() {
			Reveal(ui.Box(c.Key("close-reveal")), pointerIn).Children(func() {
				if collapseButton(c.Key("close")).Clicked() {
					*d.NowPlaying = false
				}
			})
		})
		if *d.Pane == PaneStage {
			// The stage takes the whole view: the lyrics are the show, the
			// controls a slim bar at the foot.
			m.Rise(Reveal(TabsRow(c, w, h, d.TopBar(c)), pointerIn), 0).Children(func() {
				d.PaneTabs(c)
				stageStyleButton(c, d)
			})
			Stage(c, d, tr, accent)
			stageBar(c, d, tr, art, accent)
			return
		}
		if *d.Pane == PaneWall {
			// The wall takes the whole view too, under the song playing.
			WallPane(c, d, tr, art, accent, w, h, pointerIn)
			return
		}

		coverSize := npCoverSize(w, h)
		// The cover and what plays start at one edge, with the column
		// wide enough for the controls even when the cover is small.
		side := max(coverSize, 280)
		padX := npSideX(w)
		m.Rise(Reveal(TabsRow(c, w, h, d.TopBar(c)), pointerIn), 0).Children(func() {
			d.PaneTabs(c)
			if *d.Pane == PaneLyrics {
				LyricsSizeButton(c, d)
			}
		})
		ui.Row(c).Grow(1).Padding(0, padX, 44, padX).Gap(72).AlignItems(ui.Stretch).Children(func() {
			var coverTop float32
			ui.Column(c).Width(side).Justify(ui.Center).Gap(0).Children(func() {
				coverTop = ui.Box(c.Key("cover-top")).Height(0).LayoutBounds().Y
				m.Morph(c, ui.Box(c.Key("cover-morph")).Size(coverSize, coverSize).AlignSelf(ui.Start)).Children(func() {
					Cover(c, pl, d.CoverState, d.Motion, tr, art, coverSize)
				})

				m.Rise(ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(36, 0, 18, 0), 1).Children(func() {
					ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(4).Children(func() {
						if r := pl.Radio(); r != "" {
							d.RadioChip(c, r)
						}
						ui.Text(c, tr.Name).FontSize(22).FontWeight(700).LetterSpacing(-0.3).SingleLine()
						d.ArtistLinks(c, tr, white.Alpha(0.6), white, true).FontSize(15).SingleLine()
					})
					d.Like(c.Key("like"), tr, 22, white.Alpha(0.7), appui.MediaSurface)
					d.SongMenu(c.Key("more"), tr, 22, white.Alpha(0.7), appui.MediaSurface).Margin(0, -8, 0, 0)
				})
				m.Rise(transport.Scrubber(c.Key("np-scrub"), pl, d.Settings.ProgressEffectMode(), d.NpScrub, white.Alpha(0.18), white.Alpha(0.9), 16, accent.Mix(white, 0.35)), 2)
				// As tall with the quality's badge as without it: the badge goes
				// while the next song loads, and the column would move with it.
				m.Rise(ui.Row(c).Height(16).AlignItems(ui.Center), 2).Children(func() {
					ui.Text(c, appui.Clock(pl.Position())).FontSize(11).FontFeatures("tnum").TextColor(white.Alpha(0.5))
					ui.Spacer(c)
					qualityBadge(c, pl.Stream())
					ui.Spacer(c)
					status := "-" + appui.Clock(pl.Duration()-pl.Position())
					if pl.Loading() {
						status = "正在缓冲…"
					}
					ui.Text(c, status).FontSize(11).FontFeatures("tnum").TextColor(white.Alpha(0.5))
				})
				m.Rise(ui.Box(c).Margin(14, 0, 18, 0), 3).Children(func() {
					playerbar.Transport(c, pl, white, white.Alpha(0.45), accent, 28, appui.MediaSurface)
				})
				m.Rise(ui.Row(c).Justify(ui.Center), 4).Children(func() {
					playerbar.VolumeSlider(c, d.Volume, d.MutedFrom, pl, white.Alpha(0.7), white.Alpha(0.18), 180, appui.MediaSurface)
				})
			})

			pane := ui.Column(c.Key("pane")).Grow(1).Basis(0).MinWidth(0)
			m.Rise(pane, 2)
			pane.Children(func() {
				// The pane starts level with the cover beside it, but below the
				// tabs, which start level with the pane's top: they come and go
				// with the pointer, so they sit out of the flow, never over it.
				b := pane.LayoutBounds()
				if b.H > 0 && coverTop > 0 {
					if top := max(coverTop-b.Y, TabsHeight+8); top != *d.PaneTop {
						*d.PaneTop = top
						c.AnimationFrame()
					}
				} else {
					c.AnimationFrame() // not laid out yet: the next frame places it
				}
				ui.Box(c).Height(*d.PaneTop)
				switch *d.Pane {
				case PaneLyrics:
					LyricsPane(c, d, tr, h)
				case PaneComments:
					CommentsPane(c, d)
				case PaneProgram:
					ProgramPane(c, tr)
				case PaneCaptions:
					CaptionsPane(c, d, tr, h)
				default:
					QueuePane(c, d)
				}
			})
		})
	}
}

// npMaxWidth is how wide the cover and the pane beside it go, at most.
const npMaxWidth = 1280

// npCoverSize leaves room for the controls and the strip above the cover.
func npCoverSize(w, h float32) float32 {
	return min(float32(math.Min(float64(w)*0.36, 440)), h-436)
}

// npSideX is how far in from the sides of a view w wide the cover and the
// pane beside it are: in a wide window they keep together in the middle,
// rather than the pane taking all the width the cover leaves.
func npSideX(w float32) float32 {
	return max(72, (w-npMaxWidth)/2)
}

// TabsHeight is how tall the row of Now Playing's tabs is, for what is
// under it to start below it.
const TabsHeight = 28

// collapseButton uses translucent media controls over the cover's dark
// backdrop, independently of the appearance of the pages underneath it.
func collapseButton(c *ui.Context) ui.Element {
	b := ui.ButtonBase(c).Size(40, 32).Center().Radius(12).Label("收起播放页").Tooltip("收起播放页")
	appui.MediaFeedback(b, theme.Veil(10))
	b.Children(func() { ui.Icon(c, appui.IconDown).Size(20, 20) })
	return b
}

// TabsRow is the row of Now Playing's tabs, in the same place for every
// pane, above the right-hand content. Anchoring their start keeps them
// still as the comments' count changes or the stage's style button appears.
func TabsRow(c *ui.Context, w, h float32, top float32) ui.Element {
	left := npSideX(w) + max(npCoverSize(w, h), 280) + 72
	gap := float32(6)
	if w < 1000 {
		gap = 4
	}
	return ui.Row(c.Key("tabs")).Absolute().Top(top).Left(left).Right(npSideX(w)).Height(TabsHeight).Gap(gap).AlignItems(ui.Center)
}

// Reveal fades e in while the pointer is in the window, and out when it
// leaves.
func Reveal(e ui.Element, shown bool) ui.Element {
	pose := ui.Pose{Fade: 1}
	if shown {
		pose = ui.Pose{}
	}
	return e.AnimatePose(pose, ui.Animation{Duration: 200 * time.Millisecond})
}

// qualityBadge marks a lossless stream, as it plays.
func qualityBadge(c *ui.Context, st *qqmusic.Stream) {
	if st == nil || !qqmusic.Lossless(st.Level) {
		return
	}
	label := "无损"
	if st.Level == "hires" {
		label = "Hi-Res"
	}
	ui.Text(c, label).FontSize(9.5).FontWeight(800).LetterSpacing(0.6).TextColor(theme.Warm(true, 11)).
		Padding(1, 6).Radius(4).Border(1, theme.Warm(true, 8))
}
