package nowplaying

import (
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/lyrics"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
	"lilt/internal/ui/base/transport"
)

// The stage shows lyrics as Folia's classic visualizer does: one line at a
// time, its words scattered a little, each springing in as it is sung,
// lighting up with a glow, then settling and drifting. Everything follows
// from where playback is, so seeking shows the same frame playing would.

// The stage sets its lyrics in one of the styles of Folia's visualizers
// (github.com/chthollyphile/folia-major), named as Folia names them: 流光,
// its classic (stage.go); 云阶, which stacks a line in staggered rows,
// each springing in as it is sung; and 倾诉, which breaks a line into a
// few large lines, one of them leaning, its characters swelling as they
// are sung.
type StageStyle struct{ ID, Label string }

var StageStyles = []StageStyle{
	{"", "流光"},
	{"partita", "云阶"},
	{"tilt", "倾诉"},
}

// wordShape is where a word sits and leans, the same for a line each time;
// last says it ends its line.
type wordShape struct {
	x, y, rotate, scale, drift float32
	margin                     float32
	last                       bool
}

// springStep is how far a critically-ish damped spring (stiffness 200,
// damping 20, Folia's) has gone towards its target t seconds after it set
// off: it overshoots a little and settles in about half a second.
func springStep(t float32) float32 {
	if t <= 0 {
		return 0
	}
	e := float32(math.Exp(-10 * float64(t)))
	return 1 - e*(float32(math.Cos(10*float64(t)))+float32(math.Sin(10*float64(t))))
}

func clamp01(v float32) float32 { return min(1, max(0, v)) }

func lerp(a, b, t float32) float32 { return a + (b-a)*t }

func easeOutCubic(t float32) float32 {
	t = clamp01(t)
	return 1 - (1-t)*(1-t)*(1-t)
}

// stageShapes lays out the words of a line at font size f.
func stageShapes(c *ui.Context, l *lyrics.Line, f float32) []wordShape {
	seed := float64(l.At / time.Millisecond)
	shapes := make([]wordShape, len(l.Words))
	widths := make([]float32, len(l.Words))
	for i, w := range l.Words {
		widths[i], _ = c.MeasureText(0, ui.Span{Text: w.Text, Size: f, Weight: 800})
	}
	for i := range l.Words {
		ws := seed + float64(i)
		shapes[i] = wordShape{
			x:      (transport.LineRandom(ws, 1) - 0.5) * 36,
			y:      (transport.LineRandom(ws, 2) - 0.5) * 36,
			rotate: (transport.LineRandom(ws, 3) - 0.5) * 10,
			scale:  1.1 + transport.LineRandom(ws, 4)*0.2,
			drift:  (transport.LineRandom(ws, 5) - 0.5) * 10,
		}
	}
	spaceWords(shapes, widths, f*0.08)
	if len(shapes) > 0 {
		shapes[len(shapes)-1].last = true
	}
	return shapes
}

// spaceWords leaves room after each word, so that a word swelling as it
// is sung does not run into the next: half of each one's growth, and how
// far it is moved towards the next, as Folia spaces them, and gap.
func spaceWords(shapes []wordShape, widths []float32, gap float32) {
	for i := range shapes {
		grow := widths[i] * (shapes[i].scale*1.4 - 1) / 2
		next, toward := float32(0), float32(0)
		if i+1 < len(shapes) {
			next = widths[i+1] * (shapes[i+1].scale*1.4 - 1) / 2
			toward = shapes[i].x - shapes[i+1].x
		}
		shapes[i].margin = max(2*gap, grow+next+toward+gap)
	}
}

// Stage shows the lyrics of tr on the whole stage.
func Stage(c *ui.Context, d Deps, tr *qqmusic.Track, accent ui.Color) {
	pl := d.Player
	lines, loaded := pl.Lyrics()
	w, h := c.Size()
	f := min(64, max(36, w*0.045))
	box := ui.Box(c.Key("stage")).Grow(1)
	d.Motion.Rise(box, 1)
	// In a quiet part a hot comment takes the middle, the lyrics nearly
	// gone under it.
	dim, quiet := d.QuietShown(c, box, tr, lines, loaded)
	if len(lines) == 0 {
		box.Children(func() {
			QuietUnder(ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center(), dim).Children(func() {
				if !loaded {
					d.StageSkeleton(c, f, w)
					return
				}
				ui.Text(c, "纯音乐，请欣赏").FontSize(28).FontWeight(800).TextColor(white.Alpha(0.4))
			})
			d.StageQuiet(c, dim, f, w)
		})
		return
	}
	pos := pl.Position()
	cur := lyrics.Index(lines, pos)
	// Under a hot comment there is nothing to move: the quiet asks for
	// the frame where it ends.
	if pl.Playing() && !(quiet && dim > 0.98) {
		// 倾诉 shows a part of a line a little before it is sung.
		lead := 80 * time.Millisecond
		if d.Settings.StageStyle == "tilt" {
			lead = tiltLead
		}
		stageNextFrame(c, lines, cur, pos, lead)
	}

	box.Children(func() {
		defer d.StageQuiet(c, dim, f, w)
		QuietUnder(ui.Box(c.Key("sung")).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough(), dim).Children(func() {
			// The line before goes as the new one comes.
			if cur > 0 {
				gone := float32((pos - lines[cur].At).Seconds()) / 0.4
				if gone < 1 {
					stageLine(c, d, &lines[cur-1], pos, f, w, h, 1-easeOutCubic(gone), accent)
				}
			}
			switch {
			case cur < 0:
				interlude(c, pos, 0, lines[0].At)
			case cur+1 < len(lines) && pos > lines[cur].End+time.Second && lines[cur+1].At-lines[cur].End > 4*time.Second:
				// A long break before the next line.
				interlude(c, pos, lines[cur].End, lines[cur+1].At)
			default:
				stageLine(c, d, &lines[cur], pos, f, w, h, 1, accent)
			}

			// The translation, and the line coming next, along the foot.
			ui.Column(c).Absolute().Left(0).Right(0).Bottom(24).AlignItems(ui.Center).Gap(8).PassThrough().Children(func() {
				if cur >= 0 && lines[cur].Translation != "" {
					ui.Text(c.Key("translation"), lines[cur].Translation).FontSize(18).FontWeight(600).TextColor(white.Alpha(0.72)).TextAlign(ui.Center)
				}
				if cur+1 < len(lines) {
					ui.Text(c.Key("next"), lines[cur+1].Text).FontSize(14).TextColor(white.Alpha(0.32)).TextAlign(ui.Center).SingleLine()
				}
			})
		})
	})
}

// stageNextFrame asks for the next frame of the stage as soon as it
// changes: at once while a word springs in or its glow changes, a few
// times a second while words only drift, and else when the next word is
// due, lead before it. Most of a song is the slow kind.
func stageNextFrame(c *ui.Context, lines []lyrics.Line, cur int, pos, lead time.Duration) {
	if cur < 0 {
		if len(lines) > 0 {
			c.After(min(100*time.Millisecond, lines[0].At-pos))
		}
		return
	}
	next, drifting := time.Duration(-1), false
	for i := max(cur-1, 0); i <= min(cur+1, len(lines)-1); i++ {
		for _, w := range lines[i].Words {
			start, end := w.Start-lead, max(w.End, w.Start+150*time.Millisecond)
			switch {
			case pos >= start && pos < max(start+700*time.Millisecond, end+900*time.Millisecond):
				// Springing in, filling, or its glow fading.
				c.AnimationFrame()
				return
			case pos >= end && pos < end+5*time.Second:
				drifting = true
			case pos < start && (next < 0 || start-pos < next):
				next = start - pos
			}
		}
	}
	wait := 100 * time.Millisecond // the drift, and the breathing of a break
	if !drifting && next > 0 {
		wait = min(next, 250*time.Millisecond)
	}
	c.After(wait)
}

// stageLine shows the words of l as they are at pos, the whole line at
// opacity fade.
func stageLine(c *ui.Context, d Deps, l *lyrics.Line, pos time.Duration, f, w, h, fade float32, accent ui.Color) {
	switch d.Settings.StageStyle {
	case "partita":
		partitaLine(c, d, l, pos, f, h, fade, accent)
		return
	case "tilt":
		tiltLine(c, l, pos, w, fade, accent, d.Settings.StageSparks)
		return
	}
	seed := float64(l.At / time.Millisecond)
	shapes := stageShapes(c, l, f)
	// Where the line sits: to one side or the middle, a little up or down.
	justify := []ui.Align{ui.Start, ui.Center, ui.End}[int(seed)%3]
	lift := []float32{-0.14, 0, 0.12}[int(seed*2)%3] * h

	ui.Row(c.Key(l.At)).Absolute().Top(0).Left(0).Right(0).Bottom(0).Padding(0, w*0.1).
		Wrap().Justify(justify).AlignContent(ui.Center).AlignItems(ui.Center).Translate(0, lift).
		Opacity(fade).PassThrough().Children(func() {
		for i, word := range l.Words {
			stageWord(c.Key(i), word, shapes[i], pos, f, accent, d.Settings.StageSparks)
		}
	})
}

// stageWord shows a word as it is at pos: waiting to be sung, being sung,
// or sung.
func stageWord(c *ui.Context, word lyrics.Word, s wordShape, pos time.Duration, f float32, accent ui.Color, sparks bool) {
	const lookahead = 80 * time.Millisecond
	start := word.Start - lookahead
	end := max(word.End, word.Start+150*time.Millisecond)
	since := float32((pos - start).Seconds())
	after := float32((pos - end).Seconds())
	singing := float32(max(0.08, (end - word.Start).Seconds()))

	// Waiting: small, off to the side, turned and unseen.
	x := s.x + float32(math.Sin(float64(s.y)))*100
	y := s.y + float32(math.Cos(float64(s.x)))*50
	scale, rotate, opacity, blur := float32(0.5), s.rotate+20, float32(0), float32(10)
	color := white.Alpha(0.5)
	glow := float32(0)
	if since > 0 {
		// Sung: springing to its place, larger, sharp, filling with light.
		e := springStep(since)
		x, y = lerp(x, s.x, e), lerp(y, s.y, e)
		scale, rotate = lerp(0.5, s.scale*1.4, e), lerp(s.rotate+20, s.rotate, e)
		opacity = clamp01(since / 0.1)
		blur = 10 * (1 - clamp01(since/0.2))
		color = white.Alpha(lerp(0.5, 1, clamp01(since/singing)))
		glow = clamp01(since / max(0.08, 0.3*singing))
	}
	if after > 0 {
		// Past: back to its size, a little dimmer, drifting as it rests.
		q := easeOutCubic(after / 0.5)
		scale = lerp(s.scale*1.4, s.scale, q)
		opacity = lerp(1, 0.82, q)
		rotate = s.rotate + s.drift*clamp01(after/5)
		color = white.Alpha(lerp(1, 0.5, easeOutCubic(after/0.8)))
		glow = 1 - clamp01(after/0.9)
	}

	box := ui.Box(c).MarginEnd(s.margin).Translate(x, y).Rotate(rotate).Scale(scale, scale).Opacity(opacity)
	box.Children(func() {
		text := ui.Text(c.Key("body"), word.Text).FontSize(f).FontWeight(800).NoWrap().TextColor(color)
		if blur > 0.3 {
			text.Blur(blur)
		}
		if glow > 0.01 {
			// The glow: the word again, in light, blurred around it.
			ui.Text(c.Key("glow"), word.Text).Absolute().Top(0).Left(0).FontSize(f).FontWeight(800).NoWrap().
				TextColor(accent.Alpha(0.85 * glow)).Blur(f * 0.2).PassThrough()
		}
	})
	if sparks {
		stageSparks(box, float32((pos - word.Start).Seconds()), float64(word.Start/time.Millisecond), s.last, accent)
	}
}

// interlude shows three dots breathing through a break, filling as the
// next line nears.
func interlude(c *ui.Context, pos, from, to time.Duration) {
	progress := clamp01(float32((pos - from).Seconds() / max(0.1, (to-from).Seconds())))
	breath := 1 + 0.12*float32(math.Sin(float64(c.Now().UnixMilli())/420))
	ui.Row(c.Key("interlude")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().Gap(18).PassThrough().Children(func() {
		for i := range 3 {
			lit := clamp01(progress*3 - float32(i))
			ui.Box(c.Key(i)).Size(16, 16).Radius(8).Background(white.Alpha(0.2+0.7*lit)).Scale(breath, breath)
		}
	})
}

// stageStyleButton names the stage's style, and opens the menu of the
// others.
func stageStyleButton(c *ui.Context, d Deps) {
	label := StageStyles[0].Label
	for _, s := range StageStyles {
		if s.ID == d.Settings.StageStyle {
			label = s.Label
		}
	}
	m := ui.MenuBase(c.Key("stage-style"), d.StageStyleOpen)
	b := m.Trigger.Padding(6, 10, 6, 14).Shrink(0).Radius(999).Gap(4).AlignItems(ui.Center).
		FocusRing(false).Label("律动效果").Tooltip("律动效果")
	if w, _ := c.Size(); w < 1000 {
		b.Padding(6, 8)
	}
	appui.MediaFeedback(b, theme.Veil(8))
	angle := float32(0)
	if *d.StageStyleOpen {
		b.Background(theme.Veil(2)).TextColor(theme.Paper)
		angle = 180
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(13).FontWeight(600)
		ui.Icon(c, appui.IconDown).Size(12, 12).Rotate(b.AnimateTo("arrow", angle, ui.Animation{Duration: 140 * time.Millisecond}))
	})
	m.Popup(func(panel ui.Element) {
		// The stage is a dark media surface even when the app's pages use the light theme.
		panel.AttachTo(b, ui.AnchorBottomEnd, ui.AnchorTopEnd).Margin(6, 0, 0, 0).
			Width(176).Padding(6).Gap(2).Radius(12).Background(theme.Neutral(true, 2)).
			Border(0.5, theme.Neutral(true, 6)).Shadow(0, 8, 24, -4, theme.Shade(6)).Label("律动效果菜单")
		for _, s := range StageStyles {
			selected := d.Settings.StageStyle == s.ID
			item := stageMenuRow(m.RadioItem(selected), selected).Label(s.Label)
			item.Children(func() {
				ui.Text(c, s.Label).FontSize(13).Grow(1)
				ui.Box(c).Size(16, 16).Children(func() {
					if selected {
						ui.Icon(c, appui.IconCheck).Size(16, 16).TextColor(theme.Accent(true, 11))
					}
				})
			})
			if item.Clicked() && !selected {
				d.Settings.StageStyle = s.ID
				d.Settings.Save()
			}
		}
		ui.Box(c).Height(0.5).Margin(5, 10).Background(theme.Neutral(true, 6))
		item := stageMenuRow(m.CheckItem(d.Settings.StageSparks), false).Label("火花")
		item.Children(func() {
			ui.Text(c, "火花").FontSize(13).Grow(1)
			fill, knob, x := theme.Neutral(true, 6), theme.Neutral(true, 11), float32(2)
			if d.Settings.StageSparks {
				fill, knob, x = theme.Accent(true, 9), theme.Paper, 12
			}
			ui.Box(c).Size(26, 16).Radius(8).Background(fill).Children(func() {
				ui.Box(c).Absolute().Left(x).Top(2).Size(12, 12).Radius(6).Background(knob)
			})
		})
		if item.Clicked() {
			d.Settings.StageSparks = !d.Settings.StageSparks
			d.Settings.Save()
		}
	})
}

// stageMenuRow gives a row of the stage's menu the stage's look, in place of
// the default menu's.
func stageMenuRow(item ui.Element, selected bool) ui.Element {
	item.Height(34).PaddingX(10).Gap(12).Radius(7).AlignItems(ui.Center).
		TextColor(theme.Neutral(true, 12)).FocusRing(false)
	if selected {
		item.Background(theme.Neutral(true, 3))
	}
	if item.Highlighted() {
		item.Background(theme.Neutral(true, 4))
	}
	if item.Pressed() {
		item.Background(theme.Neutral(true, 5))
	}
	return item
}

// sparkTime is how long a word's sparks fly.
const sparkTime = 0.6

// stageSparks throws sparks off e as a word lands, since seconds after it
// is sung, seed telling one word's from another's: short streaks of the
// accent, white-hot at first, fly out from its edge, slowing, shortening
// and fading; more and farther off a line's last word. They are drawn only
// while they fly, in frames the stage draws for the word anyway.
func stageSparks(e ui.Element, since float32, seed float64, last bool, accent ui.Color) {
	if since < 0 || since >= sparkTime {
		return
	}
	t := since / sparkTime
	e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		n, reach := 9, max(r.W, r.H)*0.75
		if last {
			n, reach = 16, reach*1.6
		}
		out := 1 - (1-t)*(1-t)*(1-t)
		fade := (1 - t) * (1 - t)
		color := accent.Mix(white, 0.6*(1-t))
		for i := range n {
			angle := 2 * math.Pi * (float64(i) + float64(transport.LineRandom(seed, float64(i)))*0.7) / float64(n)
			dx, dy := float32(math.Cos(angle)), float32(math.Sin(angle))
			// From the word's edge, each at its own speed.
			edge := r.W/2*0.7*abs32(dx) + r.H/2*0.7*abs32(dy)
			head := edge + reach*(0.55+0.45*transport.LineRandom(seed, float64(i)+50))*out
			tail := max(edge, head-reach*0.3*(1-t))
			p.Line(cx+dx*tail, cy+dy*tail, cx+dx*head, cy+dy*head, 0.8+1.6*(1-t), color.Alpha(0.9*fade))
		}
	})
}

func abs32(v float32) float32 { return max(v, -v) }

// stageUnits splits words into the units 云阶 and 倾诉 break a line
// between, as spans [from, to) of its words: a word of a language spaced
// into words, or a phrase of Chinese or Japanese up to its space or
// punctuation, cut in fours when long. Punctuation stays with the word
// before it, and a contraction is one word.
func StageUnits(words []lyrics.Word) [][2]int {
	var units [][2]int
	phrase := func(from, to int) {
		// A long run of characters, in pieces of about four.
		n := (to - from + 3) / 4
		if to-from <= 6 {
			n = 1
		}
		for k := range n {
			units = append(units, [2]int{from + (to-from)*k/n, from + (to-from)*(k+1)/n})
		}
	}
	from := 0
	for i, w := range words {
		if i == from && len(units) > 0 && strings.TrimFunc(w.Text, isMark) == "" {
			// Punctuation alone, after a break: onto the unit before.
			units[len(units)-1][1] = i + 1
			from = i + 1
			continue
		}
		// An apostrophe or hyphen joins the word to the next, as in It's.
		last, _ := utf8.DecodeLastRuneInString(w.Text)
		if i+1 == len(words) || isMark(last) && !strings.ContainsRune("'\u2019-", last) {
			phrase(from, i+1)
			from = i + 1
		}
	}
	return units
}

// isMark says r breaks a line: a space or punctuation.
func isMark(r rune) bool { return unicode.IsSpace(r) || unicode.IsPunct(r) }

// wordsLength is how many characters words have, not counting spaces.
func wordsLength(words []lyrics.Word) int {
	n := 0
	for _, w := range words {
		n += utf8.RuneCountInString(strings.TrimSpace(w.Text))
	}
	return n
}

// partitaRow is a row of a line on 云阶: its words, [from, to), and where
// it sits, its guide on its left or right.
type partitaRow struct {
	from, to    int
	x, y, scale float32
	left        bool
}

// partitaRows breaks l into rows as 云阶 does, unevenly, as a hand would:
// as many as the units of its words, up to as many as fit in h.
func partitaRows(l *lyrics.Line, h float32) []partitaRow {
	units := StageUnits(l.Words)
	if len(units) == 0 {
		return nil
	}
	seed, k := float64(l.At/time.Millisecond)/1000, 0.0
	random := func() float32 { k++; return transport.LineRandom(seed, k) }
	count := min(len(units), max(1, int(h*0.65/100)))
	rows := make([]partitaRow, 0, count)
	left, unit := len(units), 0
	for i := range count {
		n := left
		if i < count-1 {
			avg := float32(left) / float32(count-i)
			n = min(int(math.Ceil(float64(avg*1.5))), max(1, int(math.Round(float64(avg+(random()-0.5)*avg)))))
		}
		n = max(1, min(n, left-(count-i-1)))
		stagger := 20 + random()*80
		r := partitaRow{
			from:  units[unit][0],
			to:    units[unit+n-1][1],
			x:     stagger,
			y:     (float32(i) - float32(count-1)/2) * 2.5,
			scale: 0.8 + random()*0.9,
			left:  i%2 == 0,
		}
		if r.left {
			r.x = -stagger
		}
		rows = append(rows, r)
		unit += n
		left -= n
	}
	return rows
}

// partitaLine shows l on 云阶 as it is at pos: its rows stacked in the
// middle, each sliding in from the side of its guide as its first word is
// due, the words in it springing in as classic's do. The whole line is at
// opacity fade, swelling a little as it goes.
func partitaLine(c *ui.Context, d Deps, l *lyrics.Line, pos time.Duration, f, h, fade float32, accent ui.Color) {
	const lookahead = 80 * time.Millisecond
	rows := partitaRows(l, h)
	if wordsLength(l.Words) > 40 {
		f *= 0.8
	}
	f *= 0.8
	grow := 1 + 0.05*(1-fade)
	ui.Column(c.Key(l.At)).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().
		Opacity(fade).Scale(grow, grow).PassThrough().Children(func() {
		for ri, row := range rows {
			words := l.Words[row.from:row.to]
			since := float32((pos - (words[0].Start - lookahead)).Seconds())
			end := float32((pos - max(words[len(words)-1].End, words[len(words)-1].Start+150*time.Millisecond)).Seconds())
			side := float32(40)
			if row.left {
				side = -40
			}
			x, scale, opacity := row.x+side, float32(0.85), float32(0)
			if since > 0 {
				e := springStep(since)
				x, scale, opacity = lerp(row.x+side, row.x, e), lerp(0.85, 1, e), clamp01(since/0.1)
			}
			// The guide, a corner under the row's end, drawn in as it
			// comes, lit while it is sung.
			guide, color, alpha := easeOutCubic(since/0.4), white, float32(0.14)
			switch {
			case end > 0:
				alpha = 0.22
			case since > 0:
				color, alpha = accent, 1
			}

			shapes := make([]wordShape, len(words))
			widths := make([]float32, len(words))
			seed := float64(words[0].Start/time.Millisecond) / 1000
			for i, w := range words {
				widths[i], _ = c.MeasureText(0, ui.Span{Text: w.Text, Size: f, Weight: 800})
				ws := seed + float64(i)*7.13
				shapes[i] = wordShape{
					x:      (transport.LineRandom(ws, 1) - 0.5) * 12,
					y:      (transport.LineRandom(ws, 2) - 0.5) * 12,
					rotate: (transport.LineRandom(ws, 3) - 0.5) * 6,
					scale:  row.scale,
					drift:  (transport.LineRandom(ws, 8) - 0.5) * 20,
				}
			}
			spaceWords(shapes, widths, f*0.1)
			shapes[len(shapes)-1].margin = 0
			shapes[len(shapes)-1].last = ri == len(rows)-1

			pad := f * (row.scale*1.4 - 1) / 2
			ui.Row(c.Key(ri)).AlignItems(ui.Center).Padding(3, 8).Margin(max(0, pad), 0, max(0, pad)+10, 0).
				Translate(x, row.y).Scale(scale, scale).Opacity(opacity).
				Draw(func(p *ui.Painter, r ui.Rect) {
					if guide <= 0 {
						return
					}
					c := color.Alpha(alpha * guide)
					bottom := r.Y + r.H
					if row.left {
						p.Line(r.X-8, bottom+16, r.X-8, bottom+16-32*guide, 1, c)
						p.Line(r.X-16, bottom+8, r.X-16+(r.W+36)*guide, bottom+8, 1, c)
					} else {
						p.Line(r.X+r.W+8, bottom+16, r.X+r.W+8, bottom+16-32*guide, 1, c)
						p.Line(r.X+r.W+16, bottom+8, r.X+r.W+16-(r.W+36)*guide, bottom+8, 1, c)
					}
				}).
				Children(func() {
					for i, w := range words {
						stageWord(c.Key(i), w, shapes[i], pos, f, accent, d.Settings.StageSparks)
					}
				})
		}
	})
}

// tiltLead is how long before its first word 倾诉 shows a part of a line.
const tiltLead = 250 * time.Millisecond

// tiltPart is a part of a line on 倾诉, its words [from, to): set
// upright, or leaning, larger when it is a short end of a line.
type TiltPart struct {
	From, To   int
	Lean, Tail bool
}

// tiltParts breaks l into the parts 倾诉 sets it in: from one to four,
// more the longer it is, splitting it where its words break nearest an
// even share of its characters, and one part in about three leaning.
// fits says whether the widest of some parts is as wide as can show:
// while it is not, the line breaks into more, up to four.
func tiltParts(l *lyrics.Line, fits func([]TiltPart) bool) []TiltPart {
	units := StageUnits(l.Words)
	if len(units) == 0 {
		return nil
	}
	seed := float64(l.At/time.Millisecond) / 1000
	chars := wordsLength(l.Words)
	score := math.Log(float64(chars)+4) / math.Log(24) * float64(transport.LineRandom(seed, 1)*0.6+0.7) * 0.75
	n := 4
	for i, at := range []float64{0.45, 1.05, 1.7} {
		if score < at {
			n = i + 1
			break
		}
	}
	var parts []TiltPart
	for {
		parts = SplitParts(l.Words, units, min(n, len(units)))
		var lean []int
		for i := range parts {
			if transport.LineRandom(seed, 100+float64(i)) < 0.35 {
				lean = append(lean, i)
			}
		}
		if len(lean) > 0 {
			parts[lean[int(transport.LineRandom(seed, 200)*float32(len(lean)))]].Lean = true
		}
		if n >= 4 || n >= len(units) || fits(parts) {
			break
		}
		n++
	}
	if k := len(parts); k > 1 && !parts[k-1].Lean {
		last, before := wordsLength(l.Words[parts[k-1].From:parts[k-1].To]), wordsLength(l.Words[parts[k-2].From:parts[k-2].To])
		parts[k-1].Tail = last <= 2 && last*2 <= before
	}
	return parts
}

// splitParts splits words into n parts at the ends of units, each break
// where the share of characters before it is nearest k/n, rather after
// punctuation than a space.
func SplitParts(words []lyrics.Word, units [][2]int, n int) []TiltPart {
	total := float32(wordsLength(words))
	before := make([]float32, len(units)+1) // characters before each unit
	for i, u := range units {
		before[i+1] = before[i] + float32(wordsLength(words[u[0]:u[1]]))
	}
	parts := make([]TiltPart, 0, n)
	from := 0 // the unit the next part starts at
	for k := 1; k < n; k++ {
		at, best := -1, float32(0)
		for cut := from + 1; cut <= len(units)-(n-k); cut++ {
			d := float32(math.Abs(float64(before[cut] - total*float32(k)/float32(n))))
			if last, _ := utf8.DecodeLastRuneInString(strings.TrimSpace(words[units[cut-1][1]-1].Text)); unicode.IsPunct(last) {
				d -= total * 0.15
			}
			if at < 0 || d < best {
				at, best = cut, d
			}
		}
		parts = append(parts, TiltPart{From: units[from][0], To: units[at-1][1]})
		from = at
	}
	return append(parts, TiltPart{From: units[from][0], To: units[len(units)-1][1]})
}

// tiltLine shows l on 倾诉 as it is at pos, its parts one under another in
// the middle of the stage, w wide: each rises in as it is due, its
// characters fading in one after another, swelling as each is sung and
// staying a little larger. The leaning part is lighter, spaced wider, in
// the accent, its characters up and down by turns. The whole line is at
// opacity fade, rising a little as it goes; with sparks, each word throws
// them as it is sung.
func tiltLine(c *ui.Context, l *lyrics.Line, pos time.Duration, w, fade float32, accent ui.Color, sparks bool) {
	size := min(w*0.06875, 90)
	avail := w * 0.85
	style := func(p TiltPart) (weight int, spacing float32) {
		if p.Lean {
			return 300, 0.15
		}
		return 400, 0.08
	}
	width := func(p TiltPart) float32 {
		weight, spacing := style(p)
		text := ""
		for _, w := range l.Words[p.From:p.To] {
			text += w.Text
		}
		text = strings.TrimSpace(text)
		tw, _ := c.MeasureText(0, ui.Span{Text: text, Size: size, Weight: weight})
		return tw + float32(utf8.RuneCountInString(text))*size*spacing
	}
	widest := func(parts []TiltPart) float32 {
		m := float32(0)
		for _, p := range parts {
			m = max(m, width(p))
		}
		return m
	}
	parts := tiltParts(l, func(parts []TiltPart) bool { return widest(parts) <= avail })
	if m := widest(parts); m > avail {
		size *= max(0.5, avail/m)
	}

	ui.Column(c.Key(l.At)).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().Gap(14).
		Translate(0, -12*(1-fade)).Opacity(fade).PassThrough().Children(func() {
		for pi, part := range parts {
			tiltPartView(c.Key(pi), l.Words[part.From:part.To], part, pos, size, accent, sparks, pi == len(parts)-1)
		}
	})
}

// tiltPartView shows a part of a line on 倾诉, its words, as they are at
// pos, last when the part ends the line.
func tiltPartView(c *ui.Context, words []lyrics.Word, part TiltPart, pos time.Duration, size float32, accent ui.Color, sparks, last bool) {
	ease := func(t float32) float32 { t = clamp01(t); return 1 - (1-t)*(1-t) }
	since := float32((pos - (words[0].Start - tiltLead)).Seconds())
	weight, spacing, gap, color, pulse := 400, float32(0.08), float32(0.25), white, float32(0.15)
	in := ease(since / 0.55)
	rise, scale, stagger := 20*(1-in), float32(1), float32(0.04)
	if part.Lean {
		weight, spacing, gap, color, pulse = 300, 0.15, 0.35, accent, 0.18
		in = ease(since / 0.6)
		rise, scale, stagger = 24*(1-in), 0.92+0.08*in, 0.05
	} else if part.Tail {
		size *= 1.18
	}
	row := ui.Row(c).AlignItems(ui.Center).Translate(0, rise).Scale(scale, scale).Opacity(in)
	row.Children(func() {
		k := 0 // characters so far, not counting spaces
		for wi, word := range words {
			text := strings.TrimRightFunc(word.Text, unicode.IsSpace)
			n := utf8.RuneCountInString(text)
			each := (word.End - word.Start) / time.Duration(max(n, 1))
			i := 0
			for _, r := range text {
				if unicode.IsSpace(r) {
					ui.Box(c.Key([2]int{wi, i})).Width(size * gap)
					i++
					continue
				}
				// Swelling as it is sung, for at least a fifth of a second
				// and at most nine tenths, then settling a little larger.
				start := word.Start + time.Duration(i)*each
				d := float32(min(max(each.Seconds(), 0.2), 0.9))
				t := float32((pos - start).Seconds())
				swell := float32(0)
				switch {
				case t < 0:
				case t <= d:
					swell = float32(math.Sin(float64(t / d * math.Pi)))
				default:
					swell = 0.25 * clamp01((t-d)/(d*1.2))
				}
				s := 1 + swell*pulse
				shown := ease((since - float32(k)*stagger) / 0.5)
				ch := ui.Text(c.Key([2]int{wi, i}), string(r)).FontSize(size).FontWeight(weight).NoWrap().
					TextColor(color).MarginEnd(size*spacing).Opacity(shown).Scale(s, s)
				if part.Lean {
					// Up and down by turns, from twice as far.
					y := size / 6 * (2 - shown)
					if k%2 == 0 {
						y = -y
					}
					ch.Italic().LineHeight(1.25).Translate(0, y)
				} else {
					ch.LineHeight(1.35)
				}
				if sparks && i == 0 {
					stageSparks(ch, t, float64(word.Start/time.Millisecond), last && wi == len(words)-1, accent)
				}
				i++
				k++
			}
			if text != word.Text && wi < len(words)-1 {
				ui.Box(c.Key([2]int{wi, -1})).Width(size * gap)
			}
		}
	})
}
