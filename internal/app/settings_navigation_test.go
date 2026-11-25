package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
)

// Opening the guest account menu must not start login until its login action is chosen.
func TestGuestAccountMenuLogin(t *testing.T) {
	asked := make(chan struct{}, 1)
	begin := beginLoginQR
	beginLoginQR = func() (string, error) { asked <- struct{}{}; return "", errors.New("offline") }
	t.Cleanup(func() { beginLoginQR = begin })
	a := newTestApp()
	a.settings = defaultSettings()
	tt := newNavTester(a, 1000, 700)
	if err := tt.Click("账户"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.accountOpen || a.login.Open || len(asked) != 0 {
		t.Fatal("opening the account menu started login")
	}
	if !tt.Focused("登录 QQ 音乐") {
		t.Fatal("the guest menu did not focus its login action")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.accountOpen {
		t.Fatal("Escape left the guest menu open")
	}
	if err := tt.Click("账户"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("登录 QQ 音乐"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.accountOpen || !a.login.Open {
		t.Fatal("login did not replace the guest menu")
	}
	wait(t, asked, "the guest login QR")
}

// Settings from the sidebar open the pane over the window, without
// a route of their own or a window beside the app.
func TestAccountMenuOpensSettings(t *testing.T) {
	for _, signedIn := range []bool{false, true} {
		t.Run(fmt.Sprint(signedIn), func(t *testing.T) {
			a := newTestApp()
			a.settings = defaultSettings()
			if signedIn {
				a.profile = &qqmusic.Profile{UserID: 1, Nickname: "小明"}
			}
			tt := newNavTester(a, 1180, 780)
			route := a.route
			for range 2 {
				if err := tt.Click("设置"); err != nil {
					t.Fatal(err)
				}
				tt.Frame()
				if !a.settingsOpen || a.accountOpen || a.login.Open {
					t.Fatalf("settings did not open over the window: open %v", a.settingsOpen)
				}
				if a.route != route || len(a.history) != 0 {
					t.Fatalf("settings took the route: %+v, history %v", a.route, a.history)
				}
				tt.Key(0, ui.KeyEscape)
				tt.Frame()
				if a.settingsOpen {
					t.Fatal("Escape left the settings open")
				}
			}
			a.pl.Cue([]qqmusic.Track{{ID: 1, Name: "宝船"}})
			a.nowPlaying = true
			a.openSettings()
			tt.Frame()
			if a.nowPlaying || !a.settingsOpen {
				t.Fatal("opening the settings did not dismiss Now Playing")
			}
		})
	}
}

// Opening the settings leaves the page under them where it was, and closing
// them gives it back as it was.
func TestSettingsOverlayKeepsPageScroll(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	var cards []string
	for i := range 40 {
		name := fmt.Sprintf("浏览歌单%02d", i+1)
		cards = append(cards, name)
		a.charts = append(a.charts, qqmusic.Playlist{ID: int64(i + 1), Name: name})
	}
	a.open(route{Page: pageCharts})
	tt := newNavTester(a, 1200, 800)
	tt.Frame()
	tt.Scroll(1000, 550, 0, 600)
	tt.Frame()
	card, cardAt := visibleNavigationItem(t, tt, cards)
	a.openSettings()
	tt.Frame()
	if !a.settingsOpen || a.route.Page != pageCharts {
		t.Fatalf("settings did not open over the page: open %v, route %+v", a.settingsOpen, a.route)
	}
	// The pane takes the keys: Back is not a way out of it.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.settingsOpen {
		t.Fatal("Escape left the settings open")
	}
	assertNavigationPosition(t, tt, card, cardAt)
}

// Embedded settings must let Space toggle a focused switch rather than pause the playing song.
func TestSettingsPageSpaceTogglesControl(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	// The hot comments switch lives on the playback category.
	a.settingsTab = int(settingsPlayback)
	a.settings.Path = filepath.Join(t.TempDir(), "settings.json")
	a.route = route{Page: pageSettings}
	a.pl.Cue([]qqmusic.Track{{ID: 1, Name: "宝船"}})
	out := audio.NewPlayer(&silence{})
	out.SetVolume(0)
	out.Play()
	a.pl.Attach(out)
	defer out.Close()
	tt := newNavTester(a, 1180, 780)
	if err := tt.Click("切换热评"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	before := a.settings.HotComments
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	if a.settings.HotComments == before || !a.pl.Playing() {
		t.Fatal("Space was taken from the focused setting by the playback shortcut")
	}
	b, err := os.ReadFile(a.settings.Path)
	if err != nil {
		t.Fatal(err)
	}
	var saved settings
	if err := json.Unmarshal(b, &saved); err != nil || saved.HotComments != a.settings.HotComments {
		t.Fatalf("keyboard change was not saved: %v", err)
	}
}

// With NM_FRAMES set, saves embedded settings at narrow, regular and wide window sizes.
func TestSettingsPageFrames(t *testing.T) {
	dir := os.Getenv("NM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	checks, downloads, checked := true, false, 0
	defer fakeUpdater(&checks, &downloads, &checked)()
	for _, dark := range []bool{false, true} {
		for _, width := range []int{880, 1180, 1600} {
			a := newTestApp()
			a.settings = defaultSettings()
			a.tray = &mygo.Tray{}
			a.updates = true
			a.pl.Cue([]qqmusic.Track{{ID: 1, Name: "宝船", Artists: []qqmusic.Artist{{Name: "DOUDOU"}}, Duration: 200000}})
			a.open(route{Page: pageSettings})
			tt := ui.NewTester(a.view, width, 780)
			tt.SetDark(dark)
			tt.Frame()
			for _, bottom := range []bool{false, true} {
				if bottom {
					tt.Scroll(float32(width-100), 550, 0, 2000)
					tt.Frame()
				}
				name := fmt.Sprintf("settings-page-%d-dark-%t-bottom-%t.png", width, dark, bottom)
				f, err := os.Create(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, tt.Image())
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatalf("write frame: %v, %v", err, closeErr)
				}
			}
		}
	}
}
