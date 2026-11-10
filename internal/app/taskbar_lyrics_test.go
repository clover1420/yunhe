package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/features/lyricsui"
)

// taskbarLyricsViewFixture is a taskbar lyric host for the current test app.
func taskbarLyricsViewFixture() (*app, *taskbarLyrics) {
	a, _ := desktopLyricsViewFixture()
	a.settings.TaskbarLyrics = true
	d := lyricsui.NewTaskbar(a.lyricsDeps())
	d.Visible, d.Dark = true, true
	return a, d
}

// Changing the taskbar panel material persists both choices without closing controls or committing a pending seek.
func TestTaskbarLyricsMaterialSettings(t *testing.T) {
	if !supportsTaskbarLyrics() {
		t.Skip("taskbar lyrics are Windows-only")
	}
	a, d := taskbarControlsFixture()
	a.settings.LyricsMode = lyricsTaskbar
	a.settings.Path = filepath.Join(t.TempDir(), "settings.json")
	d.ControlsOpen = true
	d.Scrub = scrubState{V: 120, Held: true, Active: true}
	tt := ui.NewTester(a.settingsView, 560, 2400)
	tt.Frame()
	// The taskbar material sits under playback and lyrics, not the opening page.
	if err := tt.Click("播放与歌词"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, choice := range []struct {
		label, material string
		vibrancy        mygo.Vibrancy
	}{
		{"白色", taskbarMaterialWhite, mygo.VibrancyNone},
		{"亚克力", taskbarMaterialAcrylic, mygo.VibrancyAcrylic},
		{"白色", taskbarMaterialWhite, mygo.VibrancyNone},
	} {
		if choice.material == taskbarMaterialAcrylic {
			// Move within the focused taskbar selector; the sidebar also has an Acrylic segment.
			tt.Key(0, ui.KeyLeft)
		} else if err := tt.Click(choice.label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		restored := readDesktopLyricsEntrySettings(t, a)
		if a.settings.TaskbarLyricsMaterial != choice.material || restored.TaskbarLyricsMaterial != choice.material || taskbarControlsVibrancy(&restored) != choice.vibrancy {
			t.Fatalf("material not applied/saved: live=%q saved=%q backdrop=%q", a.settings.TaskbarLyricsMaterial, restored.TaskbarLyricsMaterial, taskbarControlsVibrancy(&restored))
		}
		if !d.ControlsOpen || !d.Scrub.Held || !d.Scrub.Active || !a.settings.TaskbarLyrics || a.pl.Position() != 40*time.Second {
			t.Fatal("material change closed controls, changed lyrics mode, or disturbed the pending seek")
		}
	}
}

// Choosing a lyric position swaps the active owner, saves the mode, and stops the previous taskbar monitor.
func TestTaskbarLyricsSettingsModeSwitch(t *testing.T) {
	if !supportsTaskbarLyrics() {
		t.Skip("taskbar lyrics are Windows-only")
	}
	a := newTestApp()
	a.settings = defaultSettings()
	// The rows of the category they are on.
	a.settingsTab = int(settingsPlayback)
	a.settings.Path = filepath.Join(t.TempDir(), "settings.json")
	a.setDesktopLyricsEnabled(true)
	tt := ui.NewTester(a.settingsView, 560, 2200)
	tt.Frame()
	stopped := 0
	for _, choice := range []struct {
		label, mode string
		taskbar     bool
	}{
		{"任务栏歌词", "taskbar", true},
		{"桌面歌词", "desktop", false},
		{"任务栏歌词", "taskbar", true},
		{"桌面歌词", "desktop", false},
	} {
		if err := tt.Click(choice.label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		restored := readDesktopLyricsEntrySettings(t, a)
		if lyricsMode(a.settings) != choice.mode || lyricsMode(restored) != choice.mode {
			t.Fatalf("position %q was not applied and saved: live=%q saved=%q", choice.label, lyricsMode(a.settings), lyricsMode(restored))
		}
		if a.settings.TaskbarLyrics != choice.taskbar || restored.TaskbarLyrics != choice.taskbar || a.settings.DesktopLyrics == choice.taskbar || restored.DesktopLyrics == choice.taskbar {
			t.Fatalf("position %q did not exclusively enable its lyrics: live=%+v saved=%+v", choice.label, a.settings, restored)
		}
		if (a.taskbarLyrics != nil) != choice.taskbar {
			t.Fatalf("taskbar owner lifecycle did not follow position %q", choice.label)
		}
		if choice.taskbar {
			a.taskbarLyrics.Stop = func() { stopped++ }
		}
	}
	if stopped != 2 {
		t.Fatalf("disabling did not stop each monitor: stopped=%d", stopped)
	}
	a.setLyricsMode("taskbar")
	b := newTestApp()
	b.settings = readDesktopLyricsEntrySettings(t, a)
	normalizeLyrics(&b.settings)
	b.syncTaskbarLyrics()
	if b.taskbarLyrics == nil {
		t.Fatal("restored preference did not recreate taskbar lyrics")
	}
}

// Choosing a taskbar position while lyrics are off saves the preference without opening either lyric window.
func TestTaskbarLyricsSettingsModeWhileDisabled(t *testing.T) {
	if !supportsTaskbarLyrics() {
		t.Skip("taskbar lyrics are Windows-only")
	}
	a := newTestApp()
	a.settings = defaultSettings()
	// The rows of the category they are on.
	a.settingsTab = int(settingsPlayback)
	a.settings.Path = filepath.Join(t.TempDir(), "settings.json")
	tt := ui.NewTester(a.settingsView, 560, 2200)
	tt.Frame()
	for _, label := range []string{"任务栏歌词", "桌面歌词", "任务栏歌词"} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if lyricsEnabled(a.settings) || a.desktopLyrics != nil || a.taskbarLyrics != nil {
			t.Fatalf("choosing %q opened disabled lyrics", label)
		}
	}
	restored := readDesktopLyricsEntrySettings(t, a)
	if lyricsMode(restored) != "taskbar" || lyricsEnabled(restored) {
		t.Fatal("disabled taskbar preference did not survive saving")
	}
	player := desktopLyricsEntryTester(a)
	player.Frame()
	if err := player.Click("任务栏歌词"); err != nil {
		t.Fatal(err)
	}
	player.Frame()
	if !a.settings.TaskbarLyrics || a.settings.DesktopLyrics || a.taskbarLyrics == nil {
		t.Fatal("player switch did not open the saved taskbar mode")
	}
}

// Paused seeks invalidate visible taskbar lyrics, while a hidden taskbar requests no redraw.
func TestTaskbarLyricsPausedSeekRedraw(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]qqmusic.Track{{ID: 1, Duration: 60000}})
	out := audio.NewPlayer(&silence{})
	defer out.Close()
	a.pl.Attach(out)
	a.pl.TogglePlay()
	a.settings.TaskbarLyrics = true
	a.taskbarLyrics = lyricsui.NewTaskbar(a.lyricsDeps())
	a.taskbarLyrics.Win = &mygo.Window{}
	a.taskbarLyrics.Visible = true
	defer func(f func(*mygo.Window)) { invalidateWindow = f }(invalidateWindow)
	seen := 0
	invalidateWindow = func(w *mygo.Window) {
		if w == a.taskbarLyrics.Win {
			seen++
		}
	}
	defer func(f func(time.Duration, func())) { sessionAfter = f }(sessionAfter)
	sessionAfter = func(time.Duration, func()) {}
	a.pl.Changed = a.playerChanged
	a.pl.Seek(12 * time.Second)
	if seen == 0 || a.taskbarLyrics.Revision == 0 {
		t.Fatal("paused seek did not refresh taskbar lyrics")
	}
	a.taskbarLyrics.Visible = false
	seen = 0
	a.pl.Seek(20 * time.Second)
	if seen != 0 {
		t.Fatal("hidden taskbar lyrics still invalidate")
	}
}
