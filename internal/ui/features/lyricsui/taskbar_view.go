package lyricsui

import (
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
	"lilt/internal/media/art"
	"lilt/internal/media/lyrics"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// The taskbar has its own compact layout, but reads the same playback timeline
// and real word times as the floating lyrics. It never owns a playback clock.
func (d *Taskbar) View(c *ui.Context) {
	c.Root().Background(ui.Transparent)
	if !d.Settings.TaskbarLyrics || !d.Visible || d.Player.Current() == nil {
		return
	}
	position := d.Player.Position()
	content := d.Content(position)
	width, height := c.Size()
	rightAligned := d.Placement.RightAligned
	left := float32(TaskbarCoverInset)
	if url := d.Player.Current().Album.PicURL; url != "" && d.Settings.ShowTaskbarLyricsCover() {
		size := min(float32(34), max(1, height-4))
		if d.CoverWin == nil {
			d.Cover(c, url, size, height, rightAligned)
		}
		left += size + TaskbarCoverGap
	}
	root := ui.Box(c.Key("taskbar-lyrics")).Fill().Clip()
	state := ui.Local(root, "presentation", func() desktopLyricViewState { return desktopLyricViewState{} })
	track := d.Player.Current()
	playing := d.Player.Playing()
	state.step(track.ID, track.MID, d.Revision, content, position, c.Now(), playing, c.Preferences().ReduceMotion || !playing)
	var scrolling bool
	family := d.Settings.LyricsFont
	if family == "" {
		family = c.Theme().Font
	}
	if content.text != "" && (content.index >= 0 || d.Player.Instrumental()) {
		reveal := state.rowReveal(c, content.activeRow, content.secondaryLine.Timed)
		layout := makeTaskbarLyricLayoutWithFont(c, content, position, d.Dark, height, reveal, family)
		available := max(1, width-left-6)
		activeLine := content.line
		activeWidth := layout.width
		if content.activeRow == 1 {
			activeLine = content.secondaryLine
			activeWidth = layout.secondaryWidth
		} else if !layout.preview && layout.secondaryWidth > activeWidth {
			activeWidth = layout.secondaryWidth
		}
		scrolling = activeWidth > available && !activeLine.Timed && activeLine.End > position && position >= activeLine.At
		offset := float32(0)
		var previous taskbarLyricLayout
		if !state.changed.IsZero() {
			elapsed := max(0, c.Now().Sub(state.changed))
			progress := min(1, float32(elapsed)/float32(DesktopFade))
			eased := 1 - (1-progress)*(1-progress)*(1-progress)
			offset = height * (1 - eased)
			previous = makeTaskbarLyricLayoutWithFont(c, state.previous, position, d.Dark, height, 1, family)
			c.After(min(time.Second/60, DesktopFade-elapsed))
		}
		root.Draw(func(p *ui.Painter, r ui.Rect) {
			viewport := ui.Rect{X: r.X + left, Y: r.Y, W: available, H: r.H}
			if rightAligned {
				viewport.X = r.X + float32(TaskbarCoverInset)
			}
			// Move whole blocks through a fixed viewport, keeping the two lines
			// together and preventing outgoing text from drawing over the cover.
			if offset > 0 {
				previous.draw(p, viewport, d.Dark, offset-height, rightAligned)
			}
			layout.draw(p, viewport, d.Dark, offset, rightAligned)
		})
		if playing {
			next := desktopLyricWake(content, position)
			// LRC has no real word times. Scroll an overflowing sentence only while
			// its own interval plays, then stop; short lines stay entirely idle.
			if scrolling && (next <= 0 || next > 40*time.Millisecond) {
				next = min(40*time.Millisecond, activeLine.End-position)
			}
			if next > 0 {
				c.After(next)
			}
		}
	} else if playing {
		if next := desktopLyricWake(content, position); next > 0 {
			c.After(next)
		}
	}
}

// Request the current album directly from the shared cache. A pending new cover
// uses its own placeholder rather than keeping the previous song's artwork.
func (d *Taskbar) Cover(c *ui.Context, url string, size, height float32, rightAligned bool) {
	cover := ui.ButtonBase(c.Key("taskbar-cover")).Absolute().Top((height-size)/2).
		Size(size, size).Radius(min(6, size/2)).Background(theme.Neutral(d.Dark, 3)).Center().
		Label("歌曲控制").Cursor(ui.CursorPointer).OnClick(d.ToggleControls)
	if rightAligned {
		cover.Right(TaskbarCoverInset)
	} else {
		cover.Left(TaskbarCoverInset)
	}
	d.CoverButton(c, cover, url, size)
}

func (d *Taskbar) CoverView(c *ui.Context) {
	c.Root().Background(ui.Transparent)
	tr := d.Player.Current()
	if !d.Settings.TaskbarLyrics || !d.Settings.ShowTaskbarLyricsCover() || !d.Visible || tr == nil || tr.Album.PicURL == "" {
		return
	}
	w, h := c.Size()
	size := min(w, h)
	cover := ui.ButtonBase(c.Key("taskbar-cover")).Fill().Radius(min(6, size/2)).
		Background(theme.Neutral(d.Dark, 3)).Center().Label("歌曲控制").
		Cursor(ui.CursorPointer).OnClick(d.ToggleControls)
	d.CoverButton(c, cover, tr.Album.PicURL, size)
}

var taskbarControlsIcon = appui.StrokeIcon(`<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M3 9h18M8 13h7M8 16h3"/><circle cx="16.5" cy="16" r="1.5"/>`)

func (d *Taskbar) CoverButton(c *ui.Context, cover ui.Element, url string, size float32) {
	active := float32(0)
	if cover.Hovered() || cover.FocusVisible() || d.ControlsOpen {
		active = 1
	}
	active = cover.AnimateTo("interaction", active, ui.Animation{Duration: 120 * time.Millisecond})
	overlay := theme.Shade(7)
	if cover.Pressed() {
		overlay = theme.Shade(9)
	}
	cover.Children(func() {
		d.CoverImage(c, url, size)
		if active > 0 {
			ui.Box(c.Key("control-overlay")).Absolute().Fill().Radius(min(6, size/2)).
				Background(overlay).Opacity(active).Center().Children(func() {
				ui.Icon(c, taskbarControlsIcon).Size(size*0.55, size*0.55).TextColor(theme.Veil(12))
			})
		}
	})
}

func (d *Taskbar) CoverImage(c *ui.Context, url string, size float32) {
	bmp := bitmap(d.Art.Get(qqmusic.Sized(url, 96), art.Cover))
	if bmp != nil {
		ui.Image(c, bmp).Fill().Fit(ui.Cover).Radius(min(6, size/2))
	} else {
		ui.Icon(c, appui.IconNote).Size(size*0.4, size*0.4).TextColor(theme.Neutral(d.Dark, 9))
	}
}

func (d *Taskbar) Content(position time.Duration) desktopLyricContent {
	if d.Player.Current() == nil {
		return desktopLyricContent{index: -1}
	}
	lines, loaded := d.Player.Lyrics()
	if loaded && d.Player.Instrumental() {
		return desktopLyricContent{index: -1, text: "纯音乐，请欣赏"}
	}
	return lyricAt(lines, loaded, position, true, d.translationPolicy.get(lines, TaskbarTranslation))
}

// Han lyrics without a translation use current/next even when they contain none
// of the simplified forms recognized by the desktop's conservative policy.
// Preserve distinct translations and Japanese/Korean originals; credits do not
// establish the language of the sung text.
func TaskbarTranslation(lines []lyrics.Line) bool {
	if !DesktopTranslation(lines) {
		return false
	}
	han := false
	for _, line := range lines {
		if lyrics.IsCredit(line.Text) {
			continue
		}
		if text := desktopLyricComparable(line.Translation); text != "" && text != desktopLyricComparable(line.Text) {
			return true
		}
		for _, r := range line.Text {
			if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
				return true
			}
			han = han || unicode.Is(unicode.Han, r)
		}
	}
	return !han
}

type taskbarLyricLayout struct {
	text, outline, secondary, secondaryOutline    *ui.TextLayout
	width, secondaryWidth, height, originalHeight float32
	progressX, progress                           float32
	secondaryProgressX, secondaryProgress         float32
	secondaryAlpha                                float32
	preview                                       bool
	activeRow                                     int
	words, secondaryWords                         *lyricWordGeometry
	wordColors, secondaryWordColors               []ui.Color
}

func MakeLayout(c *ui.Context, content desktopLyricContent, position time.Duration, dark bool, height float32) taskbarLyricLayout {
	return makeTaskbarLyricLayoutAt(c, content, position, dark, height, 1)
}

func makeTaskbarLyricLayoutAt(c *ui.Context, content desktopLyricContent, position time.Duration, dark bool, height float32, rowReveal float32) taskbarLyricLayout {
	return makeTaskbarLyricLayoutWithFont(c, content, position, dark, height, rowReveal, c.Theme().Font)
}

func makeTaskbarLyricLayoutWithFont(c *ui.Context, content desktopLyricContent, position time.Duration, dark bool, height float32, rowReveal float32, family string) taskbarLyricLayout {
	if content.text == "" {
		return taskbarLyricLayout{}
	}
	layout := retainedTaskbarLyricLayout(c, content, dark, height, family)
	layout.activeRow = content.activeRow
	color := theme.Neutral(dark, 12)
	if layout.words != nil {
		layout.wordColors = make([]ui.Color, len(content.line.Words))
		if content.activeRow == 1 {
			for i := range content.line.Words {
				layout.wordColors[i] = color.Alpha(DesktopSungAlpha)
			}
			layout.progress = 1
			layout.progressX = layout.width
		} else {
			for i, word := range content.line.Words {
				brightness := DesktopRestAlpha + (DesktopSungAlpha-DesktopRestAlpha)*word.Progress(position)
				layout.wordColors[i] = color.Alpha(brightness)
			}
			for i, word := range content.line.Words {
				if position < word.Start {
					break
				}
				left, right := layout.words.left[i], layout.words.right[i]
				layout.progressX = left + (right-left)*word.Progress(position)
			}
			if layout.width > 0 {
				layout.progress = min(1, max(0, layout.progressX/layout.width))
			}
		}
	} else if content.activeRow == 1 {
		layout.progress = 1
		layout.progressX = layout.width
	} else if content.line.End > content.line.At {
		layout.progress = min(1, max(0, float32(position-content.line.At)/float32(content.line.End-content.line.At)))
		layout.progressX = layout.width * layout.progress
	}
	if layout.preview && layout.secondary != nil {
		if layout.secondaryWords != nil {
			layout.secondaryWordColors = make([]ui.Color, len(content.secondaryLine.Words))
			if content.activeRow == 1 {
				for i, word := range content.secondaryLine.Words {
					brightness := DesktopRestAlpha + (DesktopSungAlpha-DesktopRestAlpha)*word.Progress(position)
					layout.secondaryWordColors[i] = color.Alpha(brightness)
				}
				for i, word := range content.secondaryLine.Words {
					if position < word.Start {
						break
					}
					left, right := layout.secondaryWords.left[i], layout.secondaryWords.right[i]
					layout.secondaryProgressX = left + (right-left)*word.Progress(position)
				}
				if layout.secondaryWidth > 0 {
					layout.secondaryProgress = min(1, max(0, layout.secondaryProgressX/layout.secondaryWidth))
				}
			} else {
				for i := range content.secondaryLine.Words {
					layout.secondaryWordColors[i] = color.Alpha(DesktopRestAlpha)
				}
			}
		} else if content.activeRow == 1 {
			layout.secondaryAlpha = DesktopRestAlpha + (DesktopSungAlpha-DesktopRestAlpha)*rowReveal
			if content.secondaryLine.End > content.secondaryLine.At {
				layout.secondaryProgress = min(1, max(0, float32(position-content.secondaryLine.At)/float32(content.secondaryLine.End-content.secondaryLine.At)))
				layout.secondaryProgressX = layout.secondaryWidth * layout.secondaryProgress
			}
		} else {
			layout.secondaryAlpha = DesktopRestAlpha
		}
	}
	return layout
}

func taskbarLyricSingleLine(text string) string {
	return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ", "\t", " ").Replace(text)
}

// Keep the sung word visible and leave short sentences left aligned. The clamp is
// shared by the original and translation so seeking never leaves stale scroll.
func taskbarLyricOffset(width, available, progressX float32) float32 {
	if width <= available {
		return 0
	}
	return -min(width-available, max(0, progressX-available*0.55))
}

// Right-aligned taskbars anchor short lines to the notification area and keep
// the sung word inside the viewport when the line is wider than it.
func taskbarLyricDrawX(x, available, width, progressX float32, rightAligned bool) float32 {
	if width <= available {
		if rightAligned {
			return x + available - width
		}
		return x
	}
	return x + taskbarLyricOffset(width, available, progressX)
}

func (layout taskbarLyricLayout) draw(p *ui.Painter, r ui.Rect, dark bool, offsetY float32, rightAligned bool) {
	if layout.text == nil {
		return
	}
	y := r.Y + (r.H-layout.height)/2 + offsetY
	x := taskbarLyricDrawX(r.X, r.W, layout.width, layout.progressX, rightAligned)
	outline := theme.Neutral(dark, 1).Alpha(0.8)
	p.Clip(r, 0, func() {
		// A fine contrasting edge remains legible on translucent taskbars.
		for _, offset := range []ui.Point{{X: -0.5}, {X: 0.5}, {Y: -0.5}, {Y: 0.5}} {
			p.TextLayout(layout.outline, x+offset.X, y+offset.Y, outline)
		}
		p.TextLayoutColors(layout.text, x, y, theme.Neutral(dark, 12), layout.wordColors)
		if layout.secondary != nil {
			w, progressX := layout.secondaryWidth, layout.secondaryWidth*layout.progress
			color := theme.Neutral(dark, 12)
			if layout.preview {
				progressX = layout.secondaryProgressX
				alpha := layout.secondaryAlpha
				if alpha <= 0 {
					alpha = DesktopRestAlpha
				}
				color = color.Alpha(alpha)
			}
			x = taskbarLyricDrawX(r.X, r.W, w, progressX, rightAligned)
			y += layout.originalHeight + 1
			for _, offset := range []ui.Point{{X: -0.5}, {X: 0.5}, {Y: -0.5}, {Y: 0.5}} {
				p.TextLayout(layout.secondaryOutline, x+offset.X, y+offset.Y, outline)
			}
			p.TextLayoutColors(layout.secondary, x, y, color, layout.secondaryWordColors)
		}
	})
}
