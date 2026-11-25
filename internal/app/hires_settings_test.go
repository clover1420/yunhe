package app

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestHiResSettingsPersist(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	// The quality row lives on the playback category.
	a.settingsTab = int(settingsPlayback)
	a.settings.Path = filepath.Join(t.TempDir(), "settings.json")
	tt := ui.NewTester(a.settingsView, 900, 2000)
	tt.Frame()
	for _, choice := range []struct{ label, level string }{{"Hi-Res（FLAC）", "hires"}, {"无损（FLAC）", "lossless"}, {"Hi-Res（FLAC）", "hires"}} {
		if err := tt.Click("选择音质"); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click(choice.label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		b, err := os.ReadFile(a.settings.Path)
		if err != nil {
			t.Fatal(err)
		}
		var saved settings
		if err := json.Unmarshal(b, &saved); err != nil {
			t.Fatal(err)
		}
		if a.settings.Quality != choice.level || saved.Quality != choice.level {
			t.Fatalf("live=%s saved=%s want=%s", a.settings.Quality, saved.Quality, choice.level)
		}
	}
	if dir := os.Getenv("NM_FRAMES"); dir != "" {
		if err := tt.Click("选择音质"); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "hires-settings.png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}
