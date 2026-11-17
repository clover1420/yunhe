package lyricsui

import (
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/lyrics"
	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/theme"
)

var (
	white          = theme.Paper
	notchLyricRest = white.Alpha(0.42)
	notchLyricSung = white.Alpha(0.96)
	notchLyricGold = ui.Hex("#ffb224")
)

// NotchLyric shares the player's real word times and retained text between
// the compact and expanded notch. Plain LRC remains line-timed, never
// estimated.
func NotchLyric(c *ui.Context, pl *player.Player, family string, tr *qqmusic.Track, align ui.Align) ui.Element {
	position := pl.Position()
	lines, loaded := pl.Lyrics()
	content := lyricAt(lines, loaded, position, false, false)
	if content.text == "" && content.index < 0 && tr != nil {
		content.text = tr.Name
	}
	if pl.Playing() && !pl.Loading() {
		if next := desktopLyricWake(content, position); next > 0 {
			c.After(next)
		}
	}
	if !content.line.Timed || len(content.line.Words) == 0 {
		return ui.Text(c, content.text).MinWidth(0).Font(family).FontSize(12.5).FontWeight(800).
			LetterSpacing(-0.4).TextColor(white.Alpha(0.68)).TextAlign(align).SingleLine()
	}
	layout := makeNotchLyricLayout(c, content.line, position, family)
	return ui.Box(c).MinWidth(0).Height(layout.height).Role(ui.RoleText).Label(content.text).
		Draw(func(p *ui.Painter, r ui.Rect) { layout.draw(p, r, align) })
}

type notchLyricLayout struct {
	words                    *lyricWordGeometry
	colors                   []ui.Color
	fills                    []ui.Rect
	width, height, progressX float32
}

func makeNotchLyricLayout(c *ui.Context, line lyrics.Line, position time.Duration, family string) notchLyricLayout {
	font := ui.Font{Family: family, Size: 12.5, Weight: 800}
	words := retainedSpacedLyricWords(c, line.Words, font, ui.TextShape{}, true, -0.4)
	layout := notchLyricLayout{words: words, colors: make([]ui.Color, len(line.Words))}
	layout.width, layout.height = words.layout.Size()
	for i, word := range line.Words {
		layout.colors[i] = notchLyricRest
		if position >= word.End {
			layout.colors[i] = notchLyricSung
		}
		if position < word.Start {
			continue
		}
		left, right := words.left[i], words.right[i]
		progress := word.Progress(position)
		edge := left + (right-left)*progress
		layout.progressX = edge
		if position < word.End && progress > 0 {
			// Carets may run right to left. Clip the fill in that direction
			// while preserving the sentence's original shaping and spacing.
			layout.fills = append(layout.fills, ui.Rect{
				X: min(left, edge), W: max(left, edge) - min(left, edge), H: layout.height,
			})
		}
	}
	return layout
}

func (layout notchLyricLayout) offset(available float32, align ui.Align) float32 {
	if layout.width > available {
		// Follow the sung word through long lines, including paused seeks.
		return taskbarLyricOffset(layout.width, available, layout.progressX)
	}
	if align == ui.Center {
		return (available - layout.width) / 2
	}
	return 0
}

func (layout notchLyricLayout) draw(p *ui.Painter, r ui.Rect, align ui.Align) {
	x := r.X + layout.offset(r.W, align)
	y := r.Y + (r.H-layout.height)/2
	p.Clip(r, 0, func() {
		p.TextLayoutColors(layout.words.layout, x, y, white, layout.colors)
		for _, fill := range layout.fills {
			fill.X, fill.Y = fill.X+x, y
			p.Clip(fill, 0, func() { p.TextLayout(layout.words.layout, x, y, notchLyricGold) })
		}
	})
}
