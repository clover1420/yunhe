// Package notch owns the quick player around the camera housing at the top
// of the screen on macOS: black like the housing, it grows out of it as
// music plays and opens into the player under the pointer.
package notch

import (
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/media/art"
	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
	"lilt/internal/ui/base/transport"
	"lilt/internal/ui/features/lyricsui"
	"lilt/internal/ui/features/playerbar"
)

// Notch is the quick player around the camera housing at the top of the
// screen: black like the housing, it grows out of it as music plays (the
// cover on the left, the music's levels on the right) and opens into the
// player under the pointer. On a display without a notch it hangs from the
// top of the screen all the same.
//
// The exported fields let the app keep its listener's state and tests drive
// the notch without a window; everything else is private to this package.
type Notch struct {
	Deps Deps
	Win  *mygo.Window

	W0, H0     float32 // the housing's size
	Real       bool    // the display has one: else the notch hides until wanted
	WinW, WinH float32
	ScreenX    float32 // the window's left on the screen
	ScreenY    float32

	Open       bool // the pointer is over it: it shows the player
	Hello      bool // it shows itself, turned off, with a way to keep it
	EnteredAt  time.Time
	LastInside time.Time
	Passing    bool   // clicks pass through to what is under the window
	Watched    bool   // the pointer's moves poll
	unwatch    func() // stops them polling, nil while they don't
	Active     bool   // the app is active, when they go unseen over the menu bar
	Polling    bool   // a poll is due

	Bars  transport.Bar
	Scrub transport.State
}

// Deps are what the notch needs from the app it runs in: the player and the
// settings it reads directly, its windows' state by pointer, and the views
// that live elsewhere by callback.
type Deps struct {
	Player   *player.Player
	Settings *settings.Settings
	Art      *art.Cache
	Palette  theme.Palette

	Quitting *bool // the app is quitting: the panel may close

	SyncWindows    func()
	ShowMain       func()
	OpenNowPlaying func()
	Like           func(c *ui.Context, tr *qqmusic.Track, size float32, color ui.Color) ui.Element
	ArtistLinks    func(c *ui.Context, tr *qqmusic.Track, color, hover ui.Color) ui.Element
}

const (
	OpenW   = 468
	IdleW   = 330
	HelloW  = 340
	LyricsW = 360
	OpenH   = 178
)

var white = theme.Paper

// New makes the notch's window and starts following the pointer.
func New(d Deps) *Notch {
	n := &Notch{Deps: d}
	n.measure()
	n.WinW, n.WinH = OpenW+60, n.H0+OpenH+30
	n.place()
	n.Win = mygo.NewWindow(mygo.WindowOptions{
		Title:         mygo.App.Name() + "迷你播放器",
		Panel:         true,
		Hidden:        true,
		Frameless:     true,
		Transparent:   true,
		DisableShadow: true,
		DisableResize: true,
		X:             int(n.ScreenX),
		Y:             int(n.ScreenY),
		Width:         int(n.WinW),
		Height:        int(n.WinH),
		Content:       ui.View(n.View),
	})
	// Escape and the like close a panel: this one stays, until the app quits.
	n.Win.OnClose(func(e *mygo.CloseEvent) {
		if !*d.Quitting {
			e.PreventDefault()
		}
	})
	raiseAboveMenuBar(n.Win.NativeHandle())
	n.Win.SetIgnoreMouseEvents(true)
	n.Passing = true
	if d.Settings.Notch {
		n.Win.ShowInactive()
		pinToTop(n.Win.NativeHandle(), float64(n.WinW), float64(n.WinH))
	}
	mygo.Screen.OnDisplaysChanged(func() {
		n.measure()
		n.place()
		pinToTop(n.Win.NativeHandle(), float64(n.WinW), float64(n.WinH))
	})
	// The pointer's moves wake the notch: it polls only while one may
	// soon open or close it, and costs nothing at rest.
	n.Active = true
	mygo.App.OnDidBecomeActive(func() { n.Active = true; n.Poll() })
	mygo.App.OnDidResignActive(func() { n.Active = false })
	n.watch()
	n.Poll()
	return n
}

// watch follows the pointer's moves while the notch shows, turned on or
// greeting. Otherwise it lets them be: each, anywhere on the screen, would
// wake the app for nothing.
func (n *Notch) watch() {
	on := n.Deps.Settings.Notch || n.Hello
	switch {
	case on && n.unwatch == nil:
		n.unwatch, n.Watched = watchPointer(n.Poll)
	case !on && n.unwatch != nil:
		n.unwatch()
		n.unwatch, n.Watched = nil, false
	}
}

// measure finds the housing of the display with the menu bar.
func (n *Notch) measure() {
	n.W0, n.H0, n.Real = 200, 32, false
	if w, h, ok := notchSize(); ok {
		n.W0, n.H0, n.Real = float32(w), float32(h), true
	} else if h := menuBarHeight(); h > 0 {
		// A notch of its own, as high as the menu bar it hangs over.
		n.H0 = float32(h)
	}
	if n.Win != nil {
		n.Win.Invalidate()
	}
}

// place centers the window at the top of the display with the menu bar,
// where the pointer finds it.
func (n *Notch) place() {
	d := mygo.Screen.PrimaryDisplay()
	n.ScreenX = float32(d.Bounds.X) + (float32(d.Bounds.Width)-n.WinW)/2
	n.ScreenY = float32(d.Bounds.Y)
}

// SetEnabled shows the notch player, or hides it for good, as the settings
// say.
func (n *Notch) SetEnabled(on bool) {
	if n.Win == nil {
		return // as in tests
	}
	if on == n.Win.IsVisible() {
		n.Win.Invalidate()
		n.watch()
		return
	}
	if on {
		n.Win.ShowInactive()
		pinToTop(n.Win.NativeHandle(), float64(n.WinW), float64(n.WinH))
		n.watch()
		n.Poll()
		return
	}
	n.Open, n.Hello = false, false
	n.Win.Hide()
	n.watch()
}

// Greet shows the notch once, as the first song plays, to a listener who
// never turned it on: it grows out of the housing, or the top of a screen
// without one, with the song and a way to keep it, and goes back in.
func (n *Notch) Greet() {
	s := n.Deps.Settings
	if runtime.GOOS != "darwin" || s.Notch || s.NotchGreeted {
		return
	}
	s.NotchGreeted = true
	s.Save()
	n.Hello = true
	if n.Win == nil {
		return // as in tests
	}
	n.Win.ShowInactive()
	pinToTop(n.Win.NativeHandle(), float64(n.WinW), float64(n.WinH))
	n.watch()
	n.Poll()
	n.byeSoon(7 * time.Second)
}

// byeSoon ends the greeting after d, unless the pointer is on it then.
func (n *Notch) byeSoon(d time.Duration) {
	time.AfterFunc(d, func() {
		mygo.RunOnMain(func() {
			switch {
			case !n.Hello:
			case n.Open || !n.Passing:
				n.byeSoon(2 * time.Second)
			default:
				n.bye()
			}
		})
	})
}

// bye ends the greeting: the notch goes back into the housing and, still
// turned off, hides there.
func (n *Notch) bye() {
	n.Hello = false
	n.Win.Invalidate()
	if n.Deps.Settings.Notch {
		return
	}
	n.Open = false
	n.Passing = true
	n.Win.SetIgnoreMouseEvents(true)
	time.AfterFunc(time.Second, func() {
		mygo.RunOnMain(func() {
			if !n.Hello && !n.Deps.Settings.Notch {
				n.Win.Hide()
				n.watch()
			}
		})
	})
}

// keep turns the notch player on, from its greeting.
func (n *Notch) keep() {
	n.Hello = false
	n.Deps.Settings.Notch = true
	n.Deps.Settings.Save()
	n.SetEnabled(true)
}

// target returns the size the shape goes to.
func (n *Notch) target() (w, h float32) {
	tr := n.Deps.Player.Current()
	s := n.Deps.Settings
	switch {
	case n.Open && tr == nil:
		return IdleW, n.H0 + 72
	case n.Open:
		return OpenW, n.H0 + OpenH
	case n.Hello && tr != nil:
		return HelloW, n.H0 + 58
	case !n.Real && !s.NotchVirtual:
		// Over a menu bar without a notch, it would hide the menus and
		// icons there: nothing shows until the pointer asks for it.
		return 0, 0
	case tr != nil && s.NotchActivity && s.Notch:
		if !n.Real {
			return max(LyricsW, n.W0+2*max(0, n.H0-12)+20), n.H0
		}
		// Wings just wide enough for the cover and the levels, so they
		// hide no more of the menu bar than they must.
		return n.W0 + 2*max(0, n.H0-12) + 20, n.H0
	default:
		return n.W0, n.H0
	}
}

// CursorPoint is where the pointer is on the screen.
var CursorPoint = func() mygo.Point { return mygo.Screen.CursorScreenPoint() }

// Poll follows the pointer, which the window cannot always see: it lets
// clicks through where it draws nothing, and opens while over it.
func (n *Notch) Poll() {
	s := n.Deps.Settings
	if !s.Notch && !n.Hello {
		return
	}
	pt := CursorPoint()
	w, h := n.target()
	x := n.ScreenX + (n.WinW-w)/2
	px, py := float32(pt.X), float32(pt.Y)
	slack := float32(6)
	inside := px >= x-slack && px <= x+w+slack && py >= n.ScreenY-slack && py <= n.ScreenY+h+slack
	if !n.Real && !s.NotchVirtual && !n.Open && !n.Hello {
		// Without a notch, the pointer pressed against the top edge of
		// the screen, about its middle, opens it.
		cx := n.ScreenX + n.WinW/2
		inside = px >= cx-n.W0/2 && px <= cx+n.W0/2 && py <= n.ScreenY+1
	}
	now := time.Now()
	changed := false
	if inside {
		if n.LastInside.IsZero() || now.Sub(n.LastInside) > 100*time.Millisecond {
			n.EnteredAt = now
		}
		n.LastInside = now
		// A pointer passing by to the menu bar does not open it: one that
		// rests there does. While it greets it stays as it is, the pointer
		// on its button.
		if !n.Open && !n.Hello && now.Sub(n.EnteredAt) >= 300*time.Millisecond {
			n.Open, changed = true, true
		}
	} else if n.Open && now.Sub(n.LastInside) > 300*time.Millisecond {
		n.Open, changed = false, true
	}
	pass := !inside
	if n.Win == nil {
		n.Passing = pass
		return // as in tests
	}
	if pass != n.Passing {
		n.Passing = pass
		n.Win.SetIgnoreMouseEvents(pass)
	}
	if changed {
		n.Win.Invalidate()
	}
	n.pollAgain(inside)
}

// pollAgain polls again soon while the pointer is over the notch or it is
// open, for the times it rests and leaves; and, more slowly, while the app
// is active, when the pointer's moves outside its windows go unseen, or
// when they are never seen. Otherwise the next move polls.
func (n *Notch) pollAgain(inside bool) {
	var d time.Duration
	switch {
	case inside || n.Open:
		d = 33 * time.Millisecond
	case n.Active || !n.Watched:
		d = 100 * time.Millisecond
	}
	if d == 0 || n.Polling {
		return
	}
	n.Polling = true
	time.AfterFunc(d, func() {
		mygo.RunOnMain(func() {
			n.Polling = false
			n.Poll()
		})
	})
}

// shape returns the outline of the housing grown to w×h: straight sides
// that flare into the top edge, and a rounded foot.
func shape(cx, w, h, flare, foot float32) *ui.Path {
	const k = 0.5523 // a quarter circle's control points
	l, r := cx-w/2, cx+w/2
	p := &ui.Path{}
	p.MoveTo(l-flare, 0)
	p.CubeTo(l-flare+flare*k, 0, l, flare-flare*k, l, flare)
	p.LineTo(l, h-foot)
	p.CubeTo(l, h-foot+foot*k, l+foot-foot*k, h, l+foot, h)
	p.LineTo(r-foot, h)
	p.CubeTo(r-foot+foot*k, h, r, h-foot+foot*k, r, h-foot)
	p.LineTo(r, flare)
	p.CubeTo(r, flare-flare*k, r+flare-flare*k, 0, r+flare, 0)
	p.Close()
	return p
}

// b2f is 0 for false and 1 for true.
func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// View draws the notch in its window.
func (n *Notch) View(c *ui.Context) {
	d := n.Deps
	pl := d.Player
	tr := pl.Current()
	c.Root().Background(ui.Transparent)

	tw, th := n.target()
	spring := ui.Animation{Spring: ui.SpringOptions{Duration: 520 * time.Millisecond, Bounce: 0.28}}
	if !n.Open {
		spring.Spring.Bounce = 0.12
	}
	stage := ui.Box(c).Fill()
	// A spring overshoots: never below nothing.
	w := max(0, stage.AnimateTo("w", tw, spring))
	h := max(0, stage.AnimateTo("h", th, spring))
	opened := float32(0)
	if n.Open {
		opened = 1
	}
	open := stage.AnimateTo("open", opened, spring)
	hello := stage.AnimateTo("hello", b2f(n.Hello), spring)
	foot := 10 + 18*open
	flare := 6 + 8*open
	cx := n.WinW / 2

	stage.Draw(func(p *ui.Painter, r ui.Rect) {
		if w < 1 || h < 1 {
			return
		}
		path := shape(r.X+cx, w, h, flare, foot)
		p.FillPath(path, theme.Ink)
	})
	stage.Children(func() {
		body := ui.Box(c).Absolute().Top(0).Left(cx-w/2).Size(w, h).Radius(0, 0, foot, foot).Clip().TextColor(white)
		body.Children(func() {
			if tr == nil {
				n.idle(c, open, (w-IdleW)/2)
				return
			}
			if d.Settings.NotchActivity || !d.Settings.Notch {
				n.compact(c, tr, 1-open, !n.Real && !n.Hello)
			}
			n.helloRow(c, hello*(1-open), (w-HelloW)/2)
			if open > 0.02 {
				n.expanded(c, tr, open, (w-OpenW)/2)
			}
		})
	})
	d.SyncWindows()
}

// compact leaves real camera housings clear. Virtual notches use that
// space for the current lyric, between the cover and the levels.
func (n *Notch) compact(c *ui.Context, tr *qqmusic.Track, shown float32, showLyrics bool) {
	if shown < 0.02 {
		return
	}
	art := n.Deps.Art.Get(qqmusic.Sized(tr.Album.PicURL, 96), art.Cover)
	s := n.H0 - 12
	ui.Row(c).Absolute().Top(0).Left(0).Right(0).Height(n.H0).PaddingX(7).Gap(10).AlignItems(ui.Center).Opacity(shown).Children(func() {
		cov := ui.Box(c).Size(s, s).Shrink(0).Radius(s * 0.28)
		in := appui.ArtIn(cov, appui.Bitmap(art))
		if in < 1 {
			cov.Background(white.Alpha(0.12))
		}
		cov.Children(func() { appui.ArtImage(c, appui.Bitmap(art), s*0.28, in) })
		if showLyrics {
			n.Lyric(c.Key("compact-lyric"), tr, ui.Center).Grow(1)
		} else {
			ui.Spacer(c)
		}
		ui.Box(c).Size(s, s).Shrink(0).Center().Children(func() {
			n.levels(c, tr, 16, s*0.7, 4)
		})
	})
}

// Lyric draws the line sung, with its words lit as they are sung. It shares
// the player's real word times and retained text between the compact and
// expanded notch, and wakes this window at the next line even when the main
// window is hidden. Pausing needs no timer.
func (n *Notch) Lyric(c *ui.Context, tr *qqmusic.Track, align ui.Align) ui.Element {
	family := n.Deps.Settings.LyricsFont
	if family == "" {
		family = c.Theme().Font
	}
	return lyricsui.NotchLyric(c, n.Deps.Player, family, tr, align)
}

// levels draws the music's levels in the cover's color.
func (n *Notch) levels(c *ui.Context, tr *qqmusic.Track, w, h float32, count int) {
	d := n.Deps
	color := theme.Paper
	if art := d.Art.Get(qqmusic.Sized(tr.Album.PicURL, 96), art.Tint); art != nil && !art.Failed {
		color = art.Tint.Mix(white, 0.35)
	}
	ui.Box(c).Size(w, h).Draw(func(p *ui.Painter, r ui.Rect) {
		lv, live := d.Player.Levels()
		n.Bars.Step(p.Now(), lv, live)
		transport.DrawBars(p, r, &n.Bars, count, color, 2.5)
		if live || n.Bars.Group(0, player.Bands) > 0.01 {
			p.AnimationFrame()
		}
	})
}

func (n *Notch) expanded(c *ui.Context, tr *qqmusic.Track, open, left float32) {
	d := n.Deps
	pl := d.Player
	art := d.Art.Get(qqmusic.Sized(tr.Album.PicURL, 200), art.Cover)
	transport.TickTimes(c, pl)
	col := ui.Column(c).Absolute().Top(n.H0+6).Left(left).Width(OpenW).Padding(8, 22, 16, 22).Gap(12).Opacity(max(0, open*1.4-0.4))
	col.Translate(0, (1-open)*-10)
	col.Children(func() {
		ui.Row(c).Gap(14).AlignItems(ui.Center).Children(func() {
			cov := ui.ButtonBase(c.Key("cover")).Size(60, 60).Radius(14).FocusRing(false).Label("打开正在播放")
			lift := ui.Pose{}
			if cov.Hovered() {
				lift.Scale = 1.05
			}
			cov.AnimatePose(lift, ui.Snappy)
			in := appui.ArtIn(cov, appui.Bitmap(art))
			if in < 1 {
				cov.Background(white.Alpha(0.12))
			}
			cov.Children(func() { appui.ArtImage(c, appui.Bitmap(art), 12, in) })
			if cov.Clicked() {
				d.OpenNowPlaying()
			}
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, tr.Name).FontSize(15).FontWeight(700).SingleLine()
				if !pl.Playing() {
					d.ArtistLinks(c, tr, white.Alpha(0.55), white).FontSize(12.5).SingleLine()
				} else if pl.Loading() {
					ui.Text(c, "正在缓冲…").FontSize(12.5).TextColor(white.Alpha(0.55)).SingleLine()
				} else {
					n.Lyric(c.Key("expanded-lyric"), tr, ui.Start).FillWidth()
				}
			})
			d.Like(c.Key("like"), tr, 17, white.Alpha(0.6))
			n.levels(c, tr, 26, 22, 5)
		})
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, appui.Clock(pl.Position())).FontSize(10.5).FontFeatures("tnum").TextColor(white.Alpha(0.5)).Width(30)
			transport.Scrubber(c.Key("notch-scrub"), pl, d.Settings.ProgressEffectMode(), &n.Scrub, white.Alpha(0.18), white.Alpha(0.9), 14, ui.Color{}).Grow(1)
			ui.Text(c, "-"+appui.Clock(pl.Duration()-pl.Position())).FontSize(10.5).FontFeatures("tnum").TextColor(white.Alpha(0.5)).Width(34).TextAlign(ui.End)
		})
		ui.Box(c).Margin(-6, 0, 0, 0).Children(func() {
			playerbar.Transport(c, pl, white, white.Alpha(0.4), white, 20)
		})
	})
}

// helloRow says, under the housing, what the notch is, while it greets.
func (n *Notch) helloRow(c *ui.Context, shown, left float32) {
	if shown < 0.02 {
		return
	}
	d := n.Deps
	row := ui.Row(c).Absolute().Top(n.H0+2).Left(left).Width(HelloW).Padding(6, 14, 14, 20).Gap(12).AlignItems(ui.Center).
		Opacity(max(0, shown*1.4-0.4))
	row.Translate(0, (1-shown)*-8)
	row.Children(func() {
		title, hint := "刘海里也能控制播放", "指针停在这里，即可暂停、切歌"
		if !n.Real {
			title, hint = "屏幕顶部也能控制播放", "指针移到顶部中央，即可暂停、切歌"
		}
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(13.5).FontWeight(700).SingleLine()
			ui.Text(c, hint).FontSize(11.5).TextColor(white.Alpha(0.55)).SingleLine()
		})
		b := ui.ButtonBase(c.Key("keep")).Padding(6, 14).Radius(999).Background(d.Palette.Accent).FocusRing(false).Label("开启刘海播放器")
		b.Children(func() { ui.Text(c, "开启").FontSize(12.5).FontWeight(600) })
		if b.Clicked() {
			n.keep()
		}
	})
}

// idle is the notch with nothing to play: it opens on a way to the app.
func (n *Notch) idle(c *ui.Context, open, left float32) {
	if open < 0.02 {
		return
	}
	d := n.Deps
	ui.Row(c).Absolute().Top(n.H0+4).Left(left).Width(IdleW).Padding(10, 20, 14, 20).Gap(12).AlignItems(ui.Center).
		Opacity(max(0, open*1.4-0.4)).Children(func() {
		ui.Box(c).Size(36, 36).Radius(12).Background(d.Palette.Accent).Center().Children(func() {
			ui.Icon(c, appui.IconNote).Size(18, 18).TextColor(white)
		})
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, mygo.App.Name()).FontSize(14).FontWeight(700)
			ui.Text(c, "未在播放").FontSize(12).TextColor(white.Alpha(0.5))
		})
		b := ui.ButtonBase(c.Key("open")).Padding(6, 14).Radius(999).Background(white.Alpha(0.14)).FocusRing(false)
		b.Children(func() { ui.Text(c, "打开").FontSize(12.5).FontWeight(600) })
		if b.Clicked() {
			d.ShowMain()
		}
	})
}
