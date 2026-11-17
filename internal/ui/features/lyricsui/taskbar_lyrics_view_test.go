package lyricsui

import (
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/media/lyrics"
	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/theme"
)

// taskbarLyricsFixture is a taskbar lyric host with a paused, cued track.
func taskbarLyricsFixture() (*player.Player, *settings.Settings, *Taskbar) {
	p := player.New(func(func()) {})
	s := settings.Default()
	s.TaskbarLyrics = true
	p.Cue([]qqmusic.Track{{ID: 7, Name: "测试歌曲", Duration: 60000}})
	p.SetLyrics(lyrics.Build("[00:01]我们一起听音乐", "", ""))
	p.Seek(2 * time.Second)
	d := NewTaskbar(Deps{Player: p, Settings: &s})
	d.Visible, d.Dark = true, true
	return p, &s, d
}

// Paused seeks move an overflowing timed sentence to the sung word and back without a stale scroll offset.
func TestTaskbarLyricsPausedSeek(t *testing.T) {
	p, _, d := taskbarLyricsFixture()
	p.SetLyrics(lyrics.Build("", "", "[1000,10000]我们一起走过漫长道路(1000,5000)直到音乐带我们走向远方(6000,5000)"))
	var offset float32
	tt := ui.NewTester(func(c *ui.Context) {
		content := d.Content(p.Position())
		layout := MakeLayout(c, content, p.Position(), d.Dark, 42)
		offset = taskbarLyricOffset(layout.width, 108, layout.progressX)
		d.View(c)
	}, 120, 42)
	p.Seek(time.Second)
	tt.Frame()
	start := offset
	p.Seek(10 * time.Second)
	tt.Frame()
	if offset >= start || offset >= 0 {
		t.Fatalf("forward seek did not reveal later words: start=%v after=%v", start, offset)
	}
	p.Seek(time.Second)
	tt.Frame()
	if offset != start || p.Playing() {
		t.Fatalf("paused backward seek retained scroll or resumed audio: offset=%v want=%v playing=%t", offset, start, p.Playing())
	}
}

// The taskbar used to omit Chinese previews and shrink translations; retain equally sized rows even on a compact taskbar.
func TestTaskbarLyricsTwoLineContent(t *testing.T) {
	for _, tc := range []struct {
		name, lrc, translation, preview, translated string
	}{
		{"Chinese preview", "[00:01]我们一起听音乐\n[00:10]让歌声陪伴远方", "", "让歌声陪伴远方", ""},
		// 快乐的扑满 contains none of the old simplified-character whitelist.
		{"Chinese without whitelist characters", "[00:44.16]那个样子\n[00:48.39]友情 在积蓄", "", "友情 在积蓄", ""},
		{"traditional Chinese without translation", "[00:01]山河歲月\n[00:10]風雨同行", "", "風雨同行", ""},
		{"Han with duplicate translation", "[00:01]那个样子\n[00:10]友情 在积蓄", "[00:01]那个，样子。", "友情 在积蓄", ""},
		{"Han with distinct translation", "[00:01]学校 国体\n[00:10]Next line", "[00:01]学校与国体", "", "学校与国体"},
		{"Chinese with Japanese credit", "[00:00]作曲：たなか\n[00:01]那个样子\n[00:10]友情 在积蓄", "", "友情 在积蓄", ""},
		{"Japanese without translation", "[00:01]愛\n[00:10]君をのせて", "", "", ""},
		{"Korean without translation", "[00:01]사랑해\n[00:10]다시", "", "", ""},
		{"foreign translation", "[00:01]Stay with me\n[00:10]Through the night", "[00:01]留在我身边\n[00:10]陪伴漫漫长夜", "", "留在我身边"},
		{"foreign without translation", "[00:01]Stay with me\n[00:10]Through the night", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, _, d := taskbarLyricsFixture()
			p.SetLyrics(lyrics.Build(tc.lrc, tc.translation, ""))
			lines, _ := p.Lyrics()
			for _, line := range lines {
				if !lyrics.IsCredit(line.Text) {
					p.Seek(line.At + time.Second)
					break
				}
			}
			content := d.Content(p.Position())
			if content.preview != tc.preview || content.translation != tc.translated {
				t.Fatalf("wrong secondary sentence: preview=%q translation=%q", content.preview, content.translation)
			}
			for _, height := range []int{28, 42} {
				tt := ui.NewTester(func(c *ui.Context) {
					layout := MakeLayout(c, content, p.Position(), d.Dark, float32(height))
					if tc.preview != "" || tc.translated != "" {
						if layout.secondary == nil {
							t.Fatal("compact layout discarded the second sentence")
						}
						_, secondaryHeight := layout.secondary.Size()
						// Latin and Han glyph metrics differ by a fraction of a pixel;
						// drawing already spaces both rows by originalHeight.
						if math.Abs(float64(secondaryHeight-layout.originalHeight)) > 0.5 || layout.height > float32(height) {
							t.Fatalf("rows differ in size or exceed taskbar: first=%v second=%v total=%v available=%d", layout.originalHeight, secondaryHeight, layout.height, height)
						}
					} else if layout.secondary != nil {
						t.Fatal("foreign song without translation borrowed the next sentence")
					}
					d.View(c)
				}, 280, height)
				tt.Frame()
				if !taskbarFrameHasInk(tt.Image()) {
					t.Fatal("two-line layout stopped painting")
				}
			}
			p.Seek(lines[len(lines)-1].At + time.Second)
			after := d.Content(p.Position())
			if tc.preview != "" {
				if after.activeRow != 1 || after.text != content.text || after.preview != content.preview {
					t.Fatalf("seek to second row of pair did not keep both rows in place: %+v", after)
				}
			} else if after.preview != "" || after.text == content.text {
				t.Fatalf("seek kept an old preview or current line: %+v", after)
			}
		})
	}
}

// Instrumentals used to be filtered into missing lyrics and never painted in the taskbar.
func TestTaskbarLyricsInstrumental(t *testing.T) {
	p, _, d := taskbarLyricsFixture()
	p.SetInstrumentalLyrics()
	p.Seek(0)
	content := d.Content(0)
	if content.text != "纯音乐，请欣赏" || content.index != -1 || content.preview != "" || content.translation != "" || desktopLyricWake(content, 0) != 0 {
		t.Fatalf("instrumental lost its static presentation: %+v", content)
	}
	tt := ui.NewTester(d.View, 280, 42)
	tt.Frame()
	if !taskbarFrameHasInk(tt.Image()) {
		t.Fatal("instrumental status was not painted")
	}
	p.SetLyrics(nil)
	tt.Frame()
	if p.Instrumental() || taskbarFrameHasInk(tt.Image()) {
		t.Fatal("missing lyrics retained the previous instrumental status")
	}
}

// A new track without lyrics, the prelude, and a hidden/disabled host must erase the old song's pixels.
func TestTaskbarLyricsClearsUnavailableContent(t *testing.T) {
	for _, reason := range []string{"missing", "prelude", "hidden", "disabled", "no-track"} {
		t.Run(reason, func(t *testing.T) {
			p, s, d := taskbarLyricsFixture()
			tt := ui.NewTester(d.View, 280, 42)
			tt.Frame()
			if !taskbarFrameHasInk(tt.Image()) {
				t.Fatal("fixture did not paint lyrics")
			}
			switch reason {
			case "missing":
				p.SetLyrics(nil)
			case "prelude":
				p.Seek(0)
			case "hidden":
				d.Visible = false
			case "disabled":
				s.TaskbarLyrics = false
			case "no-track":
				p2 := player.New(func(func()) {})
				s2 := settings.Default()
				s2.TaskbarLyrics = true
				d.Deps = Deps{Player: p2, Settings: &s2}
			}
			tt.Frame()
			if taskbarFrameHasInk(tt.Image()) {
				t.Fatalf("%s left old lyrics or an opaque background", reason)
			}
		})
	}
}

func taskbarFrameHasInk(frame image.Image) bool {
	for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
		for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
			if _, _, _, alpha := frame.At(x, y).RGBA(); alpha != 0 {
				return true
			}
		}
	}
	return false
}

// Export compact Chinese, translated, long and small-height taskbar frames for visual inspection.
func TestTaskbarLyricsFrames(t *testing.T) {
	dir := os.Getenv("NM_FRAMES")
	if dir == "" {
		t.Skip("set NM_FRAMES to inspect taskbar lyrics")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, scene := range []struct {
		name, lrc, translation, qrc string
		width, height               int
		dark                        bool
		position                    time.Duration
	}{
		{"chinese-dark", "", "", "[1000,9000]我们一起(1000,4000)听音乐(5000,5000)\n[12000,4000]让歌声陪伴远方(12000,4000)", 280, 42, true, 3 * time.Second},
		{"chinese-light", "", "", "[1000,9000]我们一起(1000,4000)听音乐(5000,5000)\n[12000,4000]让歌声陪伴远方(12000,4000)", 280, 42, false, 3 * time.Second},
		{"chinese-compact-light", "[00:01]我们一起听音乐\n[00:12]让歌声陪伴远方", "", "", 240, 28, false, 3 * time.Second},
		{"chinese-unrecognized-light", "[00:44.16]那个样子\n[00:48.39]友情 在积蓄", "", "", 280, 42, false, 45 * time.Second},
		{"instrumental-light", "[00:00:00]此歌曲为没有填词的纯音乐，请您欣赏", "", "", 280, 42, false, 0},
		{"translation-dark", "[00:01]君をのせて 空へ続く道", "[00:01]伴随着你，走向天空的路", "", 280, 42, true, 3 * time.Second},
		{"long-dark", "", "", "[1000,10000]我们一起走过漫长道路(1000,5000)直到音乐带我们走向远方(6000,5000)\n[20000,4000]让歌声陪伴远方(20000,4000)", 160, 42, true, 8 * time.Second},
		{"compact-light", "[00:01]Stay with me through the night", "[00:01]留在我身边", "", 240, 28, false, 3 * time.Second},
	} {
		t.Run(scene.name, func(t *testing.T) {
			p, _, d := taskbarLyricsFixture()
			d.Dark = scene.dark
			p.SetLyrics(lyrics.Build(scene.lrc, scene.translation, scene.qrc))
			if lyrics.IsInstrumental(scene.lrc) {
				p.SetInstrumentalLyrics()
			}
			p.Seek(scene.position)
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Box(c).Fill().Background(theme.Neutral(scene.dark, 2)).Children(func() { d.View(c) })
			}, scene.width, scene.height)
			tt.Frame()
			f, err := os.Create(filepath.Join(dir, "taskbar-lyrics-"+scene.name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, tt.Image())
			closeErr := f.Close()
			if err != nil || closeErr != nil {
				t.Fatalf("save taskbar frame: %v / %v", err, closeErr)
			}
		})
	}
}
