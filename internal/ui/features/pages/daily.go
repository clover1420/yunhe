package pages

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/navigation"
	"lilt/internal/media/art"
	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
	"lilt/internal/ui/base/transport"
)

// The daily card's measures; the stack of covers is laid out by hand.
const (
	DailyHeight = 212
	DailyRadius = 24
	DailyPadX   = 32
	StackWidth  = 300
	SleeveLeft  = 64 // where the first cover's sleeve is in the stack
	SleeveSize  = 156
	RecordSize  = 146
	DailyRate   = 60
)

// SpinState turns the daily card's record: its angle, in degrees, and how
// fast it turns, as a part of a turntable's speed, which eases up and
// glides to a stop.
type SpinState struct {
	angle, speed float64
	last         time.Time
}

// Speed is the record's current speed, 0 while it stands still, 1 while it
// turns; setting it skips the easing, as a page shown again would.
func (s *SpinState) Speed() float64     { return s.speed }
func (s *SpinState) SetSpeed(v float64) { s.speed = v }

// Step turns the record on to now, its speed easing to to, and reports
// whether it still turns.
func (s *SpinState) Step(now time.Time, to float64) bool {
	dt := now.Sub(s.last).Seconds()
	if s.last.IsZero() || dt > 0.2 {
		dt = 1.0 / 60
	}
	s.last = now
	s.speed += (to - s.speed) * min(1, 2.2*dt)
	if math.Abs(to-s.speed) < 0.002 {
		s.speed = to
	}
	// 33⅓ turns a minute.
	s.angle = math.Mod(s.angle+s.speed*dt*200, 360)
	return s.speed > 0
}

func (s *SpinState) angleNow() float32 { return float32(s.angle) }

// DailyDeps are what the daily recommendations need from the app.
type DailyDeps struct {
	Palette  theme.Palette
	Art      *art.Cache
	Spin     *SpinState
	Bars     *transport.Bar
	Player   *player.Player
	Daily    []qqmusic.Track
	SignedIn bool
	Sidebar  float32

	SongItems   func(*qqmusic.Track, int64) func(*ui.Menu)
	ArtistLinks func(*ui.Context, *qqmusic.Track, ui.Color, ui.Color) ui.Element
	CoverFill   func(*ui.Context, string, int, float32)
	BarsFn      func(*ui.Context, ui.Color)
	Login       func()
	Open        func(navigation.Route)

	HomeLoading    bool
	Centered       func(*ui.Context, string, func())
	LoadHome       func()
	Skeleton       func(*ui.Context, func(*ui.Context))
	SkeletonHeader func(*ui.Context)
	Tracks         func(*ui.Context, []qqmusic.Track, func(*ui.Context))
}

// DailyCard leads the home page: the day's songs, in the colors of their
// first cover, the day's date set huge and faint behind them. The first
// cover is a sleeve with its record peeking out, the next two behind it;
// on wide pages the first picks are listed between. Under the pointer the
// covers fan and the record slides out, turning; while the day's songs
// play it turns on, and grooves ripple out from it across the card.
func DailyCard(c *ui.Context, d DailyDeps, now time.Time) {
	songs := d.Daily
	if len(songs) == 0 {
		return
	}
	w, _ := c.Size()
	inner := w - d.Sidebar - 2*Pad
	still := c.Preferences().ReduceMotion
	cover := d.Art.Get(qqmusic.Sized(songs[0].Album.PicURL, 600), art.Backdrop)
	card := ui.ButtonBase(c.Key("daily")).Height(DailyHeight).Radius(DailyRadius).FocusRing(false).Label("每日推荐").TextColor(theme.Paper)
	hovered := card.Hovered()
	ours, playing := dailyPlaying(d)

	// The cover's colors, eased in as it loads.
	deep, rich, glow := dailyColors(cover)
	fade := ui.Animation{Duration: 600 * time.Millisecond}
	deep, rich, glow = card.AnimateColor("deep", deep, fade), card.AnimateColor("rich", rich, fade), card.AnimateColor("glow", glow, fade)
	card.Background(deep)
	lift := card.AnimateTo("lift", boolFloat(hovered), ui.Snappy)
	card.Shadow(0, 14+8*lift, 34+12*lift, -14, d.Palette.CardShade)
	// A sheen sweeps across once as the pointer comes in.
	sweep := float32(0)
	if hovered && !still {
		sweep = card.AnimateTo("sweep", 1, ui.Animation{Duration: 1100 * time.Millisecond, Ease: ui.EaseInOut})
	} else {
		card.Jump("sweep", 0)
	}

	// The record: tucked in, out a little under the pointer, out while it
	// plays.
	out := float32(0)
	switch {
	case ours:
		out = 1
	case hovered:
		out = 0.6
	}
	out = card.AnimateTo("out", out, ui.Animation{Spring: ui.SpringOptions{Duration: 700 * time.Millisecond, Bounce: 0.25}})
	turn := 0.0
	switch {
	case still:
	case playing:
		turn = 1
	case hovered:
		turn = 0.12
	}
	showStack, showPicks, showDay := inner >= 620, inner >= 1040, inner >= 900
	backdrop := backdropBitmap(cover)

	card.Children(func() {
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
			if d.Spin.Step(p.Now(), turn) {
				p.AnimationFrameRate(DailyRate)
			}
			// Where the record turns: at the right, in its sleeve's stack.
			cx := r.X + r.W - DailyPadX - 120
			if showStack {
				cx = r.X + r.W - DailyPadX - StackWidth + SleeveLeft + SleeveSize/2 + recordOut(out)
			}
			drawDailyBackdrop(p, r, backdrop, deep, rich, glow, cx, d.Spin.angleNow(), sweep)
			// The day, set huge and faint after the words, cut off by the
			// foot, where the covers leave it room.
			if showDay {
				day := ui.Span{Text: fmt.Sprint(now.Day()), Size: 240, Weight: 800, LetterSpacing: -12, Features: "tnum", Color: theme.Paper.Alpha(0.055)}
				p.Clip(r, DailyRadius, func() { p.RichText(r.X+DailyPadX+300, r.Y+r.H*0.22, 0, day) })
			}
		})
		ui.Row(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Padding(0, DailyPadX).Gap(28).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(6).Children(func() {
				ui.Textf(c, "%d月%d日 %s · %s", now.Month(), now.Day(), Weekday(now), dailyNote(d)).FontSize(12).FontWeight(600).LetterSpacing(0.3).TextColor(theme.Paper.Alpha(0.66)).MaxLines(1)
				ui.Text(c, "每日推荐").FontSize(32).FontWeight(800).LetterSpacing(-0.8).LineHeight(1.15)
				ui.Text(c, dailyArtists(songs)).FontSize(13).TextColor(theme.Paper.Alpha(0.8)).MaxLines(1)
				ui.Row(c).Gap(10).Margin(12, 0, 0, 0).Children(func() {
					DailyPlayButton(c.Key("play"), d, songs, ours, playing)
					if !d.SignedIn && appui.PillButton(c.Key("login"), appui.IconUser, "登录", theme.Paper.Alpha(0.16), theme.Paper).Clicked() {
						d.Login()
					}
				})
			})
			if showPicks {
				DailyPicks(c, d, songs)
			}
			if showStack {
				DailyStack(c, d, songs, hovered && !still, out)
			}
		})
	})
	if card.Clicked() {
		d.Open(navigation.Route{Page: navigation.Daily})
	}
}

// recordOut is how far right of its sleeve's middle the record's is, as
// far out as out.
func recordOut(out float32) float32 { return 30 + 46*out }

// dailyPlaying reports whether the day's songs are what plays, and whether
// they play now rather than paused.
func dailyPlaying(d DailyDeps) (ours, playing bool) {
	q, cur := d.Player.Queue(), d.Player.Current()
	if cur == nil || len(q) != len(d.Daily) || len(q) == 0 || q[0].ID != d.Daily[0].ID || q[len(q)-1].ID != d.Daily[len(q)-1].ID {
		return false, false
	}
	return true, d.Player.Playing()
}

// dailyColors returns the card's colors for the day's first cover: a deep
// and a rich shade of its hue, a little turned, and a bright one to glow,
// at depths that white reads over whatever the cover. A gray cover, or one
// still loading, takes NetEase's red.
func dailyColors(a *art.Artwork) (deep, rich, glow ui.Color) {
	h, s := 356.0, 0.74
	if a != nil && !a.Failed {
		th, ts, _ := theme.HSV(float64(a.Tint.R)/255, float64(a.Tint.G)/255, float64(a.Tint.B)/255)
		if ts > 0.15 {
			h, s = th, min(max(ts*1.2, 0.5), 0.78)
		}
	}
	return theme.HSVColor(h, s, 0.24), theme.HSVColor(math.Mod(h+16, 360), s*0.9, 0.5), theme.HSVColor(math.Mod(h+32, 360), s*0.55, 1)
}

// drawDailyBackdrop paints the daily card in r: a gradient of the cover's
// colors, the cover itself blurred faintly through it, a glow where the
// record turns at cx, and grooves rippling out from it, which travel as
// the record turns to angle; and, as sweep goes from 0 to 1, a sheen
// crossing it.
func drawDailyBackdrop(p *ui.Painter, r ui.Rect, blurred *ui.Bitmap, deep, rich, glow ui.Color, cx, angle, sweep float32) {
	cy := r.Y + r.H/2
	p.Clip(r, DailyRadius, func() {
		p.FillGradient(r, ui.LinearGradient{Angle: 100, Oklab: true, Stops: []ui.GradientStop{
			{Color: deep, At: 0.1}, {Color: rich, At: 0.85}, {Color: rich.Mix(deep, 0.3), At: 1},
		}}, 0)
		if blurred != nil {
			p.ImageRegion(blurred, ui.Rect{W: 48, H: 48}, coverRect(r, 48, 48), ui.ImageOptions{Tint: theme.Paper.Alpha(0.28)})
		}
		// Deeper behind the words, at the left.
		p.FillGradient(r, ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
			{Color: deep.Alpha(0.85), At: 0}, {Color: deep.Alpha(0), At: 0.6},
		}}, 0)
		// The glow behind the record, and light from above.
		p.Shadow(ui.Rect{X: cx - 60, Y: cy - 60, W: 120, H: 120}, 60, 0, 0, 140, 30, glow.Alpha(0.32))
		p.Shadow(ui.Rect{X: r.X + r.W*0.3, Y: r.Y - 80, W: r.W * 0.4, H: 40}, 20, 0, 0, 120, 0, theme.Paper.Alpha(0.08))
		// Grooves rippling out, fainter as they go, travelling a groove a
		// turn by whole pixels of the screen, which the rings are drawn at.
		const gap = 18
		off := float32(math.Round(float64(angle/360*gap*p.Scale()))) / p.Scale()
		for rr := float32(RecordSize/2+6) + off; rr < r.W; rr += gap {
			a := 0.05 * (1 - (rr-RecordSize/2)/(r.H*1.6))
			if a <= 0.004 {
				break
			}
			// Faded in as each leaves the record.
			a *= min(1, (rr-RecordSize/2-6)/gap)
			p.Stroke(ui.Rect{X: cx - rr, Y: cy - rr, W: 2 * rr, H: 2 * rr}, theme.Paper.Alpha(a), rr, 1)
		}
		if sweep > 0 && sweep < 1 {
			at := -0.25 + 1.5*sweep
			p.FillGradient(r, ui.LinearGradient{Angle: 110, Stops: []ui.GradientStop{
				{Color: theme.Paper.Alpha(0), At: clamp01(at - 0.12)},
				{Color: theme.Paper.Alpha(0.13 * (1 - sweep*0.5)), At: clamp01(at)},
				{Color: theme.Paper.Alpha(0), At: clamp01(at + 0.12)},
			}}, 0)
		}
	})
	// A hairline of light around its edge, as on glass.
	p.Stroke(r, theme.Paper.Alpha(0.1), DailyRadius, 1)
}

// dailyArtists names the first few singers of songs, and how many songs.
func dailyArtists(songs []qqmusic.Track) string {
	var names []string
	seen := map[string]bool{}
	for i := range songs {
		if len(songs[i].Artists) == 0 {
			continue
		}
		n := songs[i].Artists[0].Name
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
		if len(names) == 3 {
			break
		}
	}
	if len(names) == 0 {
		return fmt.Sprintf("今日 %d 首", len(songs))
	}
	return fmt.Sprintf("%s 等 %d 首", strings.Join(names, "、"), len(songs))
}

// DailyPlayButton plays the day's songs; while they play it says so, its
// bars dancing, and pauses them; paused, it goes on.
func DailyPlayButton(c *ui.Context, d DailyDeps, songs []qqmusic.Track, ours, playing bool) {
	ink := theme.Neutral(false, 12)
	b := ui.ButtonBase(c).Height(38).PaddingX(20).Gap(8).Radius(999).AlignItems(ui.Center).TextColor(ink).
		Shadow(0, 6, 16, -6, ink.Alpha(0.35))
	fill := theme.Paper
	if b.Hovered() {
		fill = theme.Neutral(false, 3)
	}
	b.Background(b.AnimateColor("bg", fill, ui.Animation{Duration: 120 * time.Millisecond}))
	pose := ui.Pose{}
	if b.Pressed() {
		pose.Scale = 0.95
	}
	b.AnimatePose(pose, ui.Snappy)
	label := "播放"
	// As wide hovered as not, its icon and its four characters: narrower,
	// it would leave the pointer, and flicker between the two.
	switch {
	case playing && b.Hovered():
		label = "暂停播放"
	case playing:
		label = "正在播放"
	case ours:
		label = "继续播放"
	}
	b.Label(label)
	b.Children(func() {
		if playing && !b.Hovered() {
			accent := d.Palette.Accent
			ui.Box(c).Size(14, 13).Draw(func(p *ui.Painter, r ui.Rect) {
				lv, live := d.Player.Levels()
				d.Bars.Step(p.Now(), lv, live)
				transport.DrawBars(p, r, d.Bars, 4, accent, 2.5)
				if live || d.Bars.Group(0, player.Bands) > 0.01 {
					p.AnimationFrameRate(DailyRate)
				}
			})
		} else {
			icon := appui.IconPlay
			if playing {
				icon = appui.IconPause
			}
			ui.Icon(c, icon).Size(14, 14)
		}
		ui.Text(c, label).FontSize(14).FontWeight(650).SingleLine()
	})
	if b.Clicked() {
		if ours {
			d.Player.TogglePlay()
		} else {
			d.Player.PlayQueue(songs, FirstPlayable(songs))
		}
	}
}

// DailyPicks lists the day's first songs. The current song pauses or
// resumes; another starts the day's queue from that song.
func DailyPicks(c *ui.Context, d DailyDeps, songs []qqmusic.Track) {
	cur := d.Player.Current()
	ui.Column(c.Key("picks")).Width(236).Shrink(0).Gap(2).Children(func() {
		for i := range min(3, len(songs)) {
			tr := &songs[i]
			row := ui.ButtonBase(c.Key(tr.ID)).Row().Gap(12).Padding(7, 12).Radius(14).AlignItems(ui.Center).FocusRing(false).Label(tr.Name).
				ContextMenu(d.SongItems(tr, 0))
			bg := theme.Paper.Alpha(0)
			if row.Hovered() {
				bg = theme.Paper.Alpha(0.1)
			}
			row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
			current := cur != nil && cur.ID == tr.ID
			row.Children(func() {
				ui.Box(c).Width(18).Center().Children(func() {
					if current {
						d.BarsFn(c, theme.Paper)
					} else {
						ui.Textf(c, "%02d", i+1).FontSize(12).FontWeight(700).FontFeatures("tnum").TextColor(theme.Paper.Alpha(0.45))
					}
				})
				ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(1).Children(func() {
					ui.Text(c, tr.Name).FontSize(13).FontWeight(600).MaxLines(1)
					d.ArtistLinks(c, tr, theme.Paper.Alpha(0.6), theme.Paper).FontSize(11.5).MaxLines(1)
				})
			})
			if row.Clicked() {
				if current {
					d.Player.TogglePlay()
				} else {
					d.Player.PlayQueue(songs, i)
				}
			}
		}
	})
}

// DailyStack is the day's first cover, as a sleeve with its record out as
// far as out, the two covers after it behind; fanned, they spread.
func DailyStack(c *ui.Context, d DailyDeps, songs []qqmusic.Track, fanned bool, out float32) {
	spring := ui.Animation{Spring: ui.SpringOptions{Duration: 600 * time.Millisecond, Bounce: 0.3}}
	cover := songs[0].Album.PicURL
	if tr := d.Player.Current(); tr != nil && tr.Album.PicURL != "" {
		cover = tr.Album.PicURL
	}
	// The small label shares the player bar's cover, without another image size.
	label := appui.Bitmap(d.Art.Get(qqmusic.Sized(cover, 120), art.Cover))
	ui.Box(c.Key("stack")).Size(StackWidth, SleeveSize).Shrink(0).Children(func() {
		// The covers behind, the farthest first.
		for i := min(2, len(songs)-1); i >= 1; i-- {
			size := float32(SleeveSize - 18*i)
			pose := ui.Pose{Rotate: -5 * float32(i), Y: 2}
			if fanned {
				pose = ui.Pose{Rotate: -9 * float32(i), X: -10 * float32(i), Y: -2}
			}
			cov := ui.Box(c.Key(i)).Absolute().Left(SleeveLeft-30*float32(i)).Top((SleeveSize-size)/2).Size(size, size).Radius(12).
				Opacity(1-0.12*float32(i)).Shadow(0, 10, 22, -6, theme.Ink.Alpha(0.45))
			cov.AnimatePose(pose, spring)
			cov.Children(func() { d.CoverFill(c, songs[i].Album.PicURL, 240, 10) })
		}
		// The record, behind its sleeve.
		mid := float32(SleeveLeft + SleeveSize/2)
		ui.Box(c.Key("record")).Absolute().Left(mid+recordOut(out)-RecordSize/2).Top((SleeveSize-RecordSize)/2).
			Size(RecordSize, RecordSize).Radius(RecordSize/2).Shadow(0, 8, 20, -4, theme.Ink.Alpha(0.5)).
			Draw(func(p *ui.Painter, r ui.Rect) {
				drawRecord(p, r, label, d.Spin.angleNow())
				// Each moving Draw must request its own repaint; the backdrop's
				// request alone leaves this record cached between view updates.
				if d.Spin.speed > 0 {
					p.AnimationFrameRate(DailyRate)
				}
			})
		// The sleeve.
		sleeve := ui.Box(c.Key("sleeve")).Absolute().Left(SleeveLeft).Top(0).Size(SleeveSize, SleeveSize).Radius(14).
			Shadow(0, 16, 30, -8, theme.Ink.Alpha(0.5))
		sleeve.AnimatePose(ui.Pose{Rotate: -2 * boolFloat(fanned), Y: -3 * boolFloat(fanned)}, spring)
		sleeve.Children(func() {
			d.CoverFill(c, songs[0].Album.PicURL, 400, 12)
			// Light on its gloss, and its edge.
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Radius(14).PassThrough().Border(1, theme.Paper.Alpha(0.14)).
				LinearGradient(ui.LinearGradient{Angle: 145, Stops: []ui.GradientStop{
					{Color: theme.Paper.Alpha(0.22), At: 0}, {Color: theme.Paper.Alpha(0), At: 0.42},
					{Color: theme.Ink.Alpha(0), At: 0.7}, {Color: theme.Ink.Alpha(0.18), At: 1},
				}})
		})
	})
}

// drawRecord paints a vinyl record in r, its label the cover bmp, turned
// to angle: grooves catching the light, which stays where it is as the
// record turns, round a label that turns with it.
func drawRecord(p *ui.Painter, r ui.Rect, bmp *ui.Bitmap, angle float32) {
	dd := r.W
	cx, cy := r.X+dd/2, r.Y+dd/2
	p.Fill(r, theme.Neutral(true, 1), dd/2)
	for i, rr := 0, dd*0.21; rr < dd*0.48; i, rr = i+1, rr+2.2 {
		a := float32(0.035)
		if i%3 == 0 {
			a = 0.07
		}
		p.Stroke(ui.Rect{X: cx - rr, Y: cy - rr, W: 2 * rr, H: 2 * rr}, theme.Paper.Alpha(a), rr, 0.7)
	}
	// The light on it, in two opposite fans.
	p.FillConicGradient(r, ui.ConicGradient{Angle: 20, Stops: []ui.GradientStop{
		{Color: theme.Paper.Alpha(0), At: 0}, {Color: theme.Paper.Alpha(0.13), At: 0.08}, {Color: theme.Paper.Alpha(0), At: 0.2},
		{Color: theme.Paper.Alpha(0), At: 0.5}, {Color: theme.Paper.Alpha(0.13), At: 0.58}, {Color: theme.Paper.Alpha(0), At: 0.7},
		{Color: theme.Paper.Alpha(0), At: 1},
	}}, dd/2)
	p.Stroke(r, theme.Paper.Alpha(0.1), dd/2, 1)
	// The label, turning: clipped still, its cover turning inside.
	lr := dd * 0.18
	label := ui.Rect{X: cx - lr, Y: cy - lr, W: 2 * lr, H: 2 * lr}
	p.Fill(label, theme.Accent(false, 9), lr)
	if bmp != nil {
		p.Clip(label, lr, func() {
			p.Transform(ui.Rotation(angle), cx, cy, func() { p.Image(bmp, label, ui.Cover) })
		})
	}
	p.Stroke(label, theme.Ink.Alpha(0.35), lr, 1.5)
	// The spindle hole.
	p.Fill(ui.Rect{X: cx - 2.5, Y: cy - 2.5, W: 5, H: 5}, theme.Neutral(true, 1), 2.5)
}

// dailyNote says, shortly, how the day's songs are picked.
func dailyNote(d DailyDeps) string {
	if !d.SignedIn {
		return "登录后为你专属生成"
	}
	return "QQ 音乐每日推荐"
}

// dailyBlurb says whose taste the day's songs follow.
func dailyBlurb(d DailyDeps) string {
	if !d.SignedIn {
		return "登录 QQ 音乐查看每日推荐"
	}
	return "根据你的音乐口味生成"
}

// DailyArt is the cover of the day's songs: their first cover, blurred,
// with the date set large over it.
func DailyArt(c *ui.Context, d DailyDeps, now time.Time, size float32) {
	var a *art.Artwork
	if len(d.Daily) > 0 {
		a = d.Art.Get(qqmusic.Sized(d.Daily[0].Album.PicURL, 600), art.Backdrop)
	}
	radius := size * 0.07
	blurred := backdropBitmap(a)
	box := ui.Box(c).Size(size, size).Radius(radius).Background(theme.Neutral(true, 6)).Shadow(0, 18, 40, -10, d.Palette.CardShade)
	in := appui.ArtIn(box, blurred)
	box.Children(func() {
		appui.ArtImage(c, blurred, radius, in)
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Radius(radius).
			LinearGradient(ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
				{Color: theme.Ink.Alpha(0), At: 0.35},
				{Color: theme.Ink.Alpha(0.4), At: 1},
			}})
		ui.Column(c).Absolute().Left(size * 0.09).Bottom(size * 0.08).TextColor(theme.Paper).Children(func() {
			ui.Textf(c, "%d", now.Day()).FontSize(size * 0.44).FontWeight(300).LetterSpacing(-size * 0.02).FontFeatures("tnum").LineHeight(1)
			ui.Textf(c, "%d月 · %s", now.Month(), Weekday(now)).FontSize(max(11, size*0.075)).FontWeight(600).TextColor(theme.Paper.Alpha(0.85))
		})
	})
}

// DailyPage shows the whole day's songs.
func DailyPage(c *ui.Context, d DailyDeps) {
	loading := len(d.Daily) == 0
	if loading && !d.HomeLoading {
		d.Centered(c, "暂时没有推荐", d.LoadHome)
		return
	}
	now := c.Now()
	songs := d.Daily
	header := func(c *ui.Context) {
		p := d.Palette
		ui.Row(c).Gap(28).AlignItems(ui.End).Padding(0, 0, 26, 0).Children(func() {
			DailyArt(c, d, now, 212)
			ui.Column(c).Gap(8).Children(func() {
				ui.Text(c, "每日推荐").FontSize(12).FontWeight(600).TextColor(p.Accent)
				ui.Text(c, "每日歌曲推荐").FontSize(30).FontWeight(800).LetterSpacing(-0.6)
				if loading {
					d.SkeletonHeader(c)
					return
				}
				ui.Text(c, fmt.Sprintf("%d 首 · %s", len(songs), dailyBlurb(d))).FontSize(13).TextColor(p.Muted)
				ui.Row(c).Gap(10).Margin(8, 0, 0, 0).Children(func() {
					if appui.PillButton(c.Key("play"), appui.IconPlay, "播放全部", p.Accent, p.OnAccent).Clicked() {
						if d.Player.Shuffle() {
							d.Player.ToggleShuffle()
						}
						d.Player.PlayQueue(songs, FirstPlayable(songs))
					}
					if appui.PillButton(c.Key("shuffle"), appui.IconShuffle, "随机播放", p.Hover.Over(p.Bg), p.Text).Clicked() {
						if !d.Player.Shuffle() {
							d.Player.ToggleShuffle()
						}
						d.Player.PlayQueue(songs, FirstPlayable(songs))
					}
				})
			})
		})
	}
	if loading {
		d.Skeleton(c, header)
		return
	}
	d.Tracks(c, songs, header)
}

// backdropBitmap is the blurred cover of a, nil while it loads.
func backdropBitmap(a *art.Artwork) *ui.Bitmap {
	if a == nil {
		return nil
	}
	return a.Backdrop
}

// coverRect returns where an image of w×h goes to cover r, centered.
func coverRect(r ui.Rect, w, h float32) ui.Rect {
	s := max(r.W/w, r.H/h)
	return ui.Rect{X: r.X + (r.W-w*s)/2, Y: r.Y + (r.H-h*s)/2, W: w * s, H: h * s}
}

func clamp01(v float32) float32 { return min(1, max(0, v)) }

func boolFloat(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
