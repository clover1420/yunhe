package nowplaying

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/lyrics"
	"lilt/internal/media/qqmusic"
	"lilt/internal/media/speech"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// CaptionsPane shows the episode playing's captions as lyrics show, in
// Now Playing: written down on the device as it plays. Without the model,
// it offers to download it.
func CaptionsPane(c *ui.Context, d Deps, tr *qqmusic.Track, h float32) {
	pl := d.Player
	cs := d.Captions
	if !cs.Ready() {
		modelCard(c, d)
		return
	}
	if !d.Settings.Captions {
		captionsNote(c, "字幕已关闭", "开启字幕", func() {
			d.Settings.Captions = true
			d.Settings.Save()
			d.SyncCaptions()
		})
		return
	}
	// Until the player says the episode changed, what the app has is the
	// last one's: none of it shows.
	var lines []lyrics.Line
	if cs.Program == tr.ProgramID {
		lines = cs.Lines
	}
	pos := pl.Position()
	if len(lines) == 0 {
		switch {
		case cs.Program != tr.ProgramID:
			captionsNote(c, "正在识别…", "", nil)
		case cs.Failed == cs.Program:
			captionsNote(c, "字幕生成失败", "", nil)
		case cs.Written():
			captionsNote(c, "这期节目没有识别到语音", "", nil)
		default:
			captionsNote(c, "正在识别…", "", nil)
		}
		return
	}
	cur := lyrics.Index(lines, pos)
	// A line stays lit through a pause, but not where nothing is written
	// down yet: after a seek ahead, until the words there are.
	if cur >= 0 && pos >= lines[cur].End && cs.Writing(pos) {
		cur = -1
	}
	if pl.Playing() {
		if i := lyrics.Index(lines, pos); i+1 < len(lines) {
			c.After(lines[i+1].At - pos + 20*time.Millisecond)
		}
	}
	if *d.LyricTrack != tr.ID {
		*d.LyricTrack, *d.LyricAt = tr.ID, -2
		*d.LyricScroll, *d.LyricGlide, *d.LyricBrowse = ui.ScrollState{}, Glide{}, false
	}
	lag, resumed := lyricStep(c, d)
	scroll := ui.Scroll(c.Key("captions")).Grow(1).TrackScroll(d.LyricScroll).Padding(0, 8, h*0.5, 0)
	ly := Lyrics{Player: pl, Settings: d.Settings, Scroll: d.LyricScroll, Glide: d.LyricGlide, At: d.LyricAt, Glow: d.GlowColor, Font: stageFamily(c, d), Browse: d.LyricBrowse}
	TimedLines(c, ly, scroll, lines, pos, cur, lag, LinesStyle{Font: stageFamily(c, d), Size: 24, Weight: 700, LineHeight: 1.4, Relayout: resumed}, func() {
		if cs.Work != nil {
			ui.Text(c, "正在识别…").FontSize(14).FontWeight(600).TextColor(white.Alpha(0.35)).Padding(16, 12)
		}
	})
}

// captionsNote says how the captions are, in the middle of the pane, with
// a button if there is something to do.
func captionsNote(c *ui.Context, text, button string, act func()) {
	ui.Column(c).Grow(1).Center().Gap(18).Children(func() {
		ui.Text(c, text).FontSize(22).FontWeight(700).TextColor(white.Alpha(0.4))
		if button != "" && appui.PillButton(c.Key("act"), nil, button, white.Alpha(0.14), white).Clicked() {
			act()
		}
	})
}

// modelCard offers to download the model that writes captions, and shows
// how far its download is.
func modelCard(c *ui.Context, d Deps) {
	m := &d.Captions.Model
	ui.Column(c).Grow(1).Justify(ui.Center).Gap(14).Padding(0, 12, 48, 12).MaxWidth(440).Children(func() {
		ui.Text(c, "为播客生成字幕").FontSize(24).FontWeight(800).LetterSpacing(-0.3)
		ui.Text(c, fmt.Sprintf("下载语音识别模型（约 %s）后，节目会在本机边播边转写成字幕，像歌词一样随播放滚动，点一句即可跳到那里。音频不会上传。", appui.Megabytes(speech.ModelSize()))).
			FontSize(14).LineHeight(1.6).TextColor(white.Alpha(0.6))
		if m.Downloading {
			share := float32(0)
			if m.Total > 0 {
				share = float32(m.Done) / float32(m.Total)
			}
			ui.Box(c).Height(6).Radius(3).Background(white.Alpha(0.14)).Margin(6, 0, 0, 0).Children(func() {
				ui.Box(c).Height(6).Radius(3).WidthPercent(share * 100).Background(white.Alpha(0.85))
			})
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("正在下载 %s / %s", appui.Megabytes(m.Done), appui.Megabytes(m.Total))).FontSize(12.5).FontFeatures("tnum").TextColor(white.Alpha(0.5))
				ui.Spacer(c)
				if appui.PillButton(c.Key("cancel"), nil, "取消", white.Alpha(0.14), white).Clicked() {
					d.CancelModelDownload()
				}
			})
			return
		}
		if m.Err != "" {
			ui.Text(c, m.Err).FontSize(13).TextColor(theme.Accent(true, 11))
		}
		label := "下载模型"
		if m.Err != "" {
			label = "重试"
		}
		ui.Row(c).Margin(6, 0, 0, 0).Children(func() {
			if appui.PillButton(c.Key("download"), nil, label, white, theme.Neutral(false, 12)).Clicked() {
				d.StartModelDownload()
			}
		})
	})
}
