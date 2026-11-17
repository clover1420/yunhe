package nowplaying

import (
	"math"
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/media/lyrics"
	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// lyricWordWake waits through genuine word gaps instead of keeping a 25 Hz loop.
func lyricWordWake(line lyrics.Line, position time.Duration) time.Duration {
	if !line.Timed {
		return 0
	}
	var next time.Duration
	for _, word := range line.Words {
		delay := time.Duration(0)
		if position < word.Start {
			delay = word.Start - position
		} else if position < word.End {
			delay = min(40*time.Millisecond, word.End-position)
		}
		if delay > 0 && (next == 0 || delay < next) {
			next = delay
		}
	}
	return next
}

// LyricsPane shows the lyrics of the song playing, the line sung lit, and
// brings it a third of the way down as it changes. Under a hot comment they
// nearly go, as Now Playing keeps it.
func LyricsPane(c *ui.Context, d Deps, tr *qqmusic.Track, h float32) {
	pl := d.Player
	lines, loaded := pl.Lyrics()
	ly := Lyrics{Player: pl, Settings: d.Settings, Scroll: d.LyricScroll, Glide: d.LyricGlide, At: d.LyricAt, Glow: d.GlowColor}
	if len(lines) == 0 {
		col := ui.Column(c).Grow(1)
		if loaded {
			col.Center()
		}
		dim, _ := d.QuietShown(c, col, tr, lines, loaded)
		QuietUnder(col, dim)
		col.Children(func() {
			if !loaded {
				d.Skeleton(c)
				return
			}
			ui.Text(c, "纯音乐，请欣赏").FontSize(24).FontWeight(700).TextColor(white.Alpha(0.4))
		})
		d.QuietCard(c, dim, h*0.18)
		return
	}
	pos := pl.Position()
	cur := lyrics.Index(lines, pos)
	// Redraw as the next line begins.
	if pl.Playing() && cur+1 < len(lines) {
		c.After(lines[cur+1].At - pos)
	}
	if *d.LyricTrack != tr.ID {
		*d.LyricTrack, *d.LyricAt = tr.ID, -2
		*d.LyricScroll, *d.LyricGlide, *d.LyricBrowse = ui.ScrollState{}, Glide{}, false
	}
	lag, resumed := lyricStep(c, d)
	// Redraw as the next line begins, and as the size popover changes it.
	size := float32(LyricsFontSize(d.Settings))
	font := d.Settings.LyricsFont
	if font == "" {
		font = c.Theme().Font
	}
	if *d.LyricSize != size || *d.LyricFont != font {
		if *d.LyricSize != 0 {
			*d.LyricSizeAt = c.Now()
		}
		*d.LyricSize, *d.LyricFont, *d.LyricAt = size, font, -2
	}
	// Builds can repeat before layout commits. Keep waiting for that entire
	// frame, so following uses the new geometry rather than a previous pass.
	resized := !(*d.LyricSizeAt).IsZero() && (*d.LyricSizeAt).Equal(c.Now())

	scroll := ui.Scroll(c.Key("lyrics")).Grow(1).TrackScroll(d.LyricScroll).Padding(0, 8, h*0.5, 0)
	dim, quiet := d.QuietShown(c, scroll, tr, lines, loaded)
	if dim > 0.98 {
		// Under the hot comment, gone, not drawn faint.
		scroll.Invisible()
	}
	defer d.QuietCard(c, dim, h*0.18)
	ly.Font, ly.Browse = font, d.LyricBrowse
	TimedLines(c, ly, scroll, lines, pos, cur, lag, LinesStyle{Font: font, Size: size, TranslationSize: 17 * size / LyricsFontSizeDefault, Weight: 800, LineHeight: 1.25, Quiet: quiet, Relayout: resized || resumed}, nil)
}

// lyricStep steps the glide of the lyrics or captions, and leaves the line
// sung to the listener once they scroll.
func lyricStep(c *ui.Context, d Deps) (float32, bool) {
	if c.WindowHidden() {
		if !*d.LyricBrowse {
			*d.LyricResume = true
		}
		return 0, true
	}
	if *d.LyricResume {
		*d.LyricResume = false
		if !*d.LyricBrowse {
			// Restoration can replace the scroll offset without any user
			// input. Do not interpret that change as browsing. Follow using
			// the restored layout's committed bounds in the next frame.
			*d.LyricGlide = Glide{set: d.LyricScroll.Y}
			*d.LyricAt = -2
			*d.LyricResumeAt = c.Now()
		}
	}
	if !(*d.LyricResumeAt).IsZero() && (*d.LyricResumeAt).Equal(c.Now()) {
		return 0, true
	}
	lag, scrolled := d.LyricGlide.Step(c, d.LyricScroll)
	if scrolled {
		*d.LyricBrowse = true
	}
	return lag, false
}

// lyricsFollowButton floats at the foot of the lyrics while the listener
// reads them on their own, its arrow pointing up or down to the line sung:
// a click brings it back, and the lyrics follow it again.
func lyricsFollowButton(c *ui.Context, up bool) ui.Element {
	in := &ui.Motion{Y: 10, Scale: 0.96}
	label := "回到下方的当前歌词"
	if up {
		label = "回到上方的当前歌词"
	}
	var b ui.Element
	ui.Row(c).Absolute().Left(0).Right(8).Bottom(24).Justify(ui.Center).PassThrough().
		Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: in, Exit: in}).Children(func() {
		// A capsule of frosted light glass: it reads over any line under
		// it, which still shows through.
		b = ui.ButtonBase(c).Height(40).PaddingX(20).Gap(8).Radius(999).AlignItems(ui.Center).FocusRing(false).
			Label(label).Tooltip(label).TextColor(white).
			Material(glass.Blur{Radius: 20}).Border(1, white.Alpha(0.28)).Shadow(0, 8, 24, -6, theme.Shade(7))
		bg := white.Alpha(0.22)
		if b.Hovered() || b.FocusVisible() {
			bg = white.Alpha(0.3)
		}
		pose := ui.Pose{}
		if b.Pressed() {
			bg, pose.Scale = white.Alpha(0.36), 0.96
		}
		// The material takes the background's place: the light goes over it.
		fill := b.AnimateColor("bg", bg, ui.Animation{Duration: 140 * time.Millisecond})
		b.Draw(func(p *ui.Painter, r ui.Rect) { p.Fill(r, fill, r.H/2) }).AnimatePose(pose, ui.Snappy)
		b.Children(func() {
			// The arrow turns over as the line sung passes where the
			// listener reads.
			turn := float32(0)
			if up {
				turn = 180
			}
			ui.Box(c.Key("arrow")).Size(16, 16).AnimatePose(ui.Pose{Rotate: turn}, ui.Snappy).Children(func() {
				ui.Icon(c, appui.IconArrowDown).Size(16, 16)
			})
			ui.Text(c, "回到当前").FontSize(15).FontWeight(700).SingleLine()
		})
	})
	return b
}

// Lyrics is what the timed lines need of the app they draw in.
type Lyrics struct {
	Player   *player.Player
	Settings *settings.Settings
	Scroll   *ui.ScrollState
	Glide    *Glide
	At       *int
	Glow     *ui.Color
	Font     string
	Browse   *bool // the listener reads the lyrics on their own
}

// LinesStyle is how TimedLines sets its lines: the size, weight and
// leading of their type; quiet nearly hides them, under a hot comment.
type LinesStyle struct {
	Font            string
	Size            float32
	TranslationSize float32
	Weight          int
	LineHeight      float32
	Quiet           bool
	Relayout        bool // wait for the new size's committed bounds before following
}

// TimedLines fills scroll with lines that follow the track playing at
// pos: the line at cur lit, and brought to a third of the way down once
// it changes, but none with cur -1 before the first; a click on one seeks
// to it. The lines are held back by lag, how far the glide to the line
// lit has still to go (Glide). foot, if set, follows the lines.
func TimedLines(c *ui.Context, l Lyrics, scroll ui.Element, lines []lyrics.Line, pos time.Duration, cur int, lag float32, st LinesStyle, foot func()) {
	pl := l.Player
	if st.Font == "" {
		st.Font = c.Theme().Font
	}
	var active ui.Element
	seeked := false
	scroll.Children(func() {
		ui.Column(c).Gap(6).Translate(0, -lag).Children(func() {
			for i, line := range lines {
				b := ui.ButtonBase(c.Key(i)).Column().AlignItems(ui.Start).Padding(10, 12).Radius(14).Gap(6).FocusRing(false).Label(line.Text)
				on := i == cur
				alpha := float32(0.28)
				if on {
					alpha = 1
				} else if b.Hovered() {
					alpha = 0.55
				}
				if st.Quiet {
					// Nearly gone under a hot comment.
					alpha *= QuietFaded
				}
				if b.Hovered() {
					b.Background(white.Alpha(0.06))
				}
				scale := float32(0.94)
				if on {
					scale = 1
				}
				b.TransformOrigin(0, 0.5).AnimatePose(ui.Pose{Scale: scale}, ui.Animation{Spring: ui.SpringOptions{Duration: 600 * time.Millisecond, Bounce: 0.1}})
				b.Children(func() {
					// The line fades as a pose, in frames that only paint, rather
					// than building the view at the display's rate as it does.
					text := ui.Column(c).AlignItems(ui.Start).Gap(6)
					text.AnimatePose(ui.Pose{Fade: 1 - alpha}, ui.Animation{Duration: 380 * time.Millisecond})
					text.Children(func() {
						if on && line.Timed && len(line.Words) > 0 {
							// The line sung lights up word by word as it is sung.
							ui.RichText(c.Key("words")).Font(st.Font).FontSize(st.Size).FontWeight(st.Weight).LineHeight(st.LineHeight).LetterSpacing(-0.4).Children(func() {
								for wi, wd := range line.Words {
									lit := wd.Progress(pos)
									ui.Text(c.Key(wi), wd.Text).TextColor(white.Alpha(0.38 + 0.62*lit))
								}
							})
							if pl.Playing() {
								// Once a word ends, sleep until another starts instead of
								// rebuilding unchanged lyrics through a gap or the outro.
								if next := lyricWordWake(line, pos); next > 0 {
									c.After(next)
								}
							}
						} else {
							ui.Text(c, line.Text).Font(st.Font).FontSize(st.Size).FontWeight(st.Weight).LineHeight(st.LineHeight).LetterSpacing(-0.4).TextColor(white)
						}
						if line.Translation != "" {
							size := st.TranslationSize
							if size == 0 {
								size = 17
							}
							ui.Text(c, line.Translation).Font(st.Font).FontSize(size).FontWeight(600).TextColor(white.Alpha(0.7))
						}
						// The line sung again in light, over it and blurred.
						if on && l.glows() {
							l.lineGlow(c, line, pos, st)
						}
					})
				})
				if b.Clicked() {
					pl.Seek(line.At)
					seeked = true
				}
				if on {
					active = b
				}
			}
		})
		if foot != nil {
			foot()
		}
	})
	// followY is the offset that brings the line sung a third of the way
	// down, and whether the line is laid out for it yet.
	followY := func() (float32, bool) {
		if !active.Valid() {
			return 0, true
		}
		// Where the glide is going, without where it still is.
		box, lb := scroll.Bounds(), active.LayoutBounds()
		if lb.H <= 0 || box.H <= 0 {
			return 0, false
		}
		return min(max(0, lb.Y-box.Y+l.Scroll.Y-box.H*0.3), l.Scroll.MaxY), true
	}
	if l.Browse != nil && *l.Browse && !st.Quiet {
		y, _ := followY()
		if lyricsFollowButton(c.Key("lyrics-follow"), y < l.Scroll.Y).Clicked() {
			*l.Browse, *l.At = false, -2
		}
	}
	if seeked {
		// A line chosen is sung from: the lyrics follow it again, once the
		// next frame finds it lit.
		if l.Browse != nil {
			*l.Browse, *l.At = false, -2
		}
		c.AnimationFrame()
		return
	}
	if st.Relayout {
		c.AnimationFrame()
		return
	}
	// Bring the line sung to a third of the way down, once it changed,
	// unless the listener scrolled away to read on their own.
	if cur != *l.At && (l.Browse == nil || !*l.Browse) {
		target, ok := followY()
		if !ok {
			return // not laid out yet: the next frame scrolls
		}
		*l.At = cur
		l.Glide.To(c, l.Scroll, target)
	}
}

// Glide scrolls the lyrics to the line sung, as a critically damped
// spring, as ScrollTo's would. ScrollTo moves the offset in every frame,
// and MyGo builds the view again in the frame after each that moved it,
// and its spring creeps over the last pixel for half a second: with a line
// following another every second or two, most of a song. So the offset
// goes to the line at once, in one frame, and the lines are held back by
// what the glide has still to go, its lag, which settles to nothing within
// half a pixel.
type Glide struct {
	on bool
	// The lag t after start: (a + b·t)·e^(−ωt), in DIPs.
	a, b  float64
	start time.Time
	// The offset it goes to, set in the next frame (pending), with the lag
	// that holds the lines back from it: the layout of the frame that
	// chose it takes the offset too, the lines built without the lag.
	target  float32
	pending bool
	set     float32 // the offset it set
}

// lyricOmega is the spring's ω: a line or two away, it comes within half a
// pixel in about 800 ms.
const lyricOmega = 10

// NewGlide returns a glide at rest at offset, so that a change of layout is
// not taken for the listener scrolling.
func NewGlide(offset float32) Glide { return Glide{set: offset} }

// at returns the lag at now, and how fast it changes, in DIPs a second.
func (g *Glide) at(now time.Time) (lag, v float64) {
	if !g.on {
		return 0, 0
	}
	t := now.Sub(g.start).Seconds()
	e := math.Exp(-lyricOmega * t)
	return (g.a + g.b*t) * e, (g.b - lyricOmega*(g.a+g.b*t)) * e
}

// To scrolls s to target from the next frame, gliding from where it
// shows, at the speed it went if it was on its way.
func (g *Glide) To(c *ui.Context, s *ui.ScrollState, target float32) {
	now := c.Now()
	lag, v := g.at(now)
	shown := float64(s.Y) + lag
	g.target, g.pending = target, true
	g.on = !c.Preferences().ReduceMotion
	g.start, g.a = now, shown-float64(target)
	g.b = v + lyricOmega*g.a
	c.AnimationFrame()
}

// Step returns the lag of the lines in this frame, and asks for the next
// while there is one. The listener scrolling stops the glide where it
// shows, and step reports it: the offset moved from the one it set, other
// than by the lyrics growing shorter under it.
func (g *Glide) Step(c *ui.Context, s *ui.ScrollState) (lag float32, scrolled bool) {
	if g.pending {
		s.Y, g.set, g.pending = g.target, g.target, false
	}
	moved := s.Y != g.set
	scrolled = moved && !(s.Y == s.MaxY && g.set > s.MaxY)
	if !g.on {
		g.set = s.Y
		return 0, scrolled
	}
	at, v := g.at(c.Now())
	if moved {
		// The offset takes what showed.
		s.Y, g.on = max(0, min(s.Y+float32(at), s.MaxY)), false
		g.set = s.Y
		return 0, scrolled
	}
	if math.Abs(at) < 0.5 && math.Abs(v) < 20 {
		g.on = false
		return 0, false
	}
	c.AnimationFrame()
	return float32(at), false
}

// QuietFaded is how much of the lyrics shows under a hot comment.
const QuietFaded = 0.06

// QuietUnder fades e, the lyrics, under a hot comment as far in as dim,
// and hides it once the comment is all in.
func QuietUnder(e ui.Element, dim float32) ui.Element {
	if dim > 0.98 {
		e.Invisible()
	} else if dim > 0.01 {
		e.Opacity(1 - (1-QuietFaded)*dim)
	}
	return e
}

// Now Playing's glow (settings.NpLight), kept as faint as iOS keeps its
// light: what is lit reads by the dark around it, and little more is added.
// The line sung has a soft aura as far as it has been sung, and a word held
// for a second or more swells and settles, as Apple Music's do.

// How strongly a line sung glows, and how far a word held swells, in the
// alpha of their light.
const (
	lineAura  = 0.62 // a line
	wordSwell = 0.45 // a word held, at the height of it
)

// wordHeld returns how much word wd swells at pos, from 0: as AMLL, which
// follows Apple Music's lyrics, has it. Only a word held a second or more
// does, the more the longer it is held, and the last of a line the most: up
// as it begins, and back to nothing by its end, so that a line at rest and
// the many short words of one have no swell at all.
func wordHeld(wd lyrics.Word, last bool, pos time.Duration) float32 {
	held := (wd.End - wd.Start).Seconds()
	if held < 1 {
		return 0
	}
	v := held / 3
	if v > 1 {
		v = math.Sqrt(v)
	} else {
		v = v * v * v
	}
	v *= 0.5
	span := held
	if last {
		v *= 1.5
		span *= 1.2
	}
	t := (pos - wd.Start).Seconds() / span
	if t <= 0 || t >= 1 {
		return 0
	}
	// Quickly up to the middle, and slowly down.
	if t < 0.5 {
		u := 1 - t*2
		return float32(min(v, 0.8) * (1 - u*u))
	}
	u := (t - 0.5) * 2
	return float32(min(v, 0.8) * (1 - u*u*(3-2*u)))
}

// glowGain is how far the listener has turned the glow up, 1 as it was tuned.
func (l Lyrics) glowGain() float32 { return float32(l.Settings.GlowLevel) * 2 }

// glows reports whether Now Playing lights the line sung.
func (l Lyrics) glows() bool { return l.Settings.NpLight && l.glowGain() > 0.01 }

// lineGlow is the line sung again in light, blurred around it, as far as it
// has been sung where it has words: those to come are left out of it, for
// nothing to glow ahead of the singer. A word's color keeps a little alpha:
// at none it would fall back to the color of the text around it, in full, as
// a span does.
func (l Lyrics) lineGlow(c *ui.Context, line lyrics.Line, pos time.Duration, st LinesStyle) {
	gain := l.glowGain()
	aura := lineAura * gain
	var glow ui.Element
	if len(line.Words) > 0 {
		if pos < line.Words[0].Start {
			return
		}
		glow = ui.RichText(c.Key("glow")).Children(func() {
			for wi, wd := range line.Words {
				if pos < wd.Start {
					break
				}
				v := aura * wd.Progress(pos)
				if line.Timed {
					v += wordSwell * gain * wordHeld(wd, wi == len(line.Words)-1, pos)
				}
				ui.Text(c.Key(wi), wd.Text).TextColor(l.Glow.Alpha(max(min(v, 1), 0.02)))
			}
		})
	} else {
		glow = ui.Text(c.Key("glow"), line.Text).TextColor(l.Glow.Alpha(max(min(aura, 1), 0.02)))
	}
	glow.Absolute().Top(0).Left(0).Right(0).Font(st.Font).FontSize(st.Size).FontWeight(st.Weight).LineHeight(st.LineHeight).LetterSpacing(-0.4).
		Blur(st.Size * 0.22).PassThrough()
}
