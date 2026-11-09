// Package settings holds the listener's choices and where they are kept.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
)

// Settings are the listener's choices, kept in settings.json in the user
// data directory.
type Settings struct {
	// QuitOnClose exits the app when the main window is closed; otherwise
	// it stays available from the tray or Dock.
	QuitOnClose bool `json:"quitOnClose"`
	// Notch shows the quick player around the notch: the listener turns
	// it on.
	Notch bool `json:"notch"`
	// DesktopLyrics enables the independent floating lyrics panel.
	DesktopLyrics         bool    `json:"desktopLyrics"`
	DesktopLyricsLocked   bool    `json:"desktopLyricsLocked"`
	DesktopLyricsFontSize float32 `json:"desktopLyricsFontSize,omitempty"`
	// DesktopLyricsDoubleLine previews the next sentence for Chinese lyrics.
	DesktopLyricsDoubleLine bool `json:"desktopLyricsDoubleLine,omitempty"`
	// DesktopLyricsColor is the original lyric color; empty keeps the white palette.
	DesktopLyricsColor string `json:"desktopLyricsColor,omitempty"`
	// TaskbarLyrics embeds a separate compact lyric view in the Windows taskbar.
	TaskbarLyrics bool `json:"taskbarLyrics,omitempty"`
	// LyricsMode keeps the preferred location independently of its enabled state.
	LyricsMode string `json:"lyricsMode,omitempty"`
	// NotchActivity shows the cover and the music's levels beside the
	// notch while music plays, which hides the menu bar there.
	NotchActivity bool `json:"notchActivity"`
	// NotchVirtual keeps a notch of the app's own showing at the top of a
	// display without one; else it shows only while the pointer asks.
	NotchVirtual bool `json:"notchVirtual"`
	// NotchGreeted says the notch showed itself, once, as the first song
	// played (greet).
	NotchGreeted bool `json:"notchGreeted"`
	// Quality is the level of the streams: standard, higher, exhigh,
	// lossless or hires.
	Quality string `json:"quality"`
	// Appearance is light, dark, or system to follow the system.
	Appearance string `json:"appearance"`
	// SidebarMaterial selects mica or acrylic (also when empty) on Windows.
	SidebarMaterial string `json:"sidebarMaterial,omitempty"`
	// AccentCover takes the accent from the cover playing, else the app's red.
	AccentCover bool `json:"accentCover"`
	// Smoke makes the progress bars comets trailing smoke as music plays,
	// for a few percent of a core: the listener turns it on.
	Smoke bool `json:"smoke"`
	// ProgressEffect selects the progress bar's look. Empty keeps the old
	// Smoke choice when reading settings saved before the effect selector.
	ProgressEffect string `json:"progressEffect,omitempty"`
	// LyricsStage shows lyrics on the stage, Folia's style, in Now
	// Playing, rather than scrolling.
	LyricsStage bool `json:"lyricsStage"`
	// LyricsFontSize is the size the scrolling lyrics are set in; zero keeps
	// the original size.
	LyricsFontSize float64 `json:"lyricsFontSize,omitempty"`
	// StageStyle is how the stage sets its lyrics (stageStyles): empty for
	// Folia's classic, or the id of another of its visualizers.
	StageStyle string `json:"stageStyle,omitempty"`
	// StageSparks throws sparks off each word on the stage as it is sung,
	// whatever its style.
	StageSparks bool `json:"stageSparks,omitempty"`
	// Volume is where the volume was left, from 0 to 1.
	Volume float64 `json:"volume"`
	// PlaybackFade smooths pause and resume operations.
	PlaybackFade bool `json:"playbackFade"`
	// Normalize is the old volume-balance switch, kept so installs saved
	// before the mode selector can still be read.
	Normalize bool `json:"volumeBalance"`
	// VolumeBalance selects how far QQ's measured loudness is lifted.
	VolumeBalance string `json:"volumeBalanceMode,omitempty"`
	// HotComments shows a song's hot comments in its quiet parts, in
	// place of the lyrics (quietComment): the listener turns it on.
	HotComments bool `json:"hotComments"`
	// Fog fills Now Playing's backdrop with fog of the cover's color,
	// rising and swelling with the bass: an experiment the listener turns
	// on.
	Fog bool `json:"fog"`
	// AppIcon is where the app's icon shows on macOS (appIcons); elsewhere
	// it is always in the notification area.
	AppIcon string `json:"appIcon"`
	// Captions writes down episodes as they play, for their captions,
	// once the listener has downloaded the model.
	Captions bool `json:"captions"`
	// CacheMB is how many megabytes the songs kept on disk may take; 0
	// keeps none.
	CacheMB int `json:"cacheMB"`
	// CacheDir is the chosen parent of the dedicated Yunhe song cache;
	// empty uses the platform's cache directory.
	CacheDir string `json:"cacheDir,omitempty"`
	// PendingCacheDir is migrated and applied before playback on the next launch.
	// A pointer to an empty string means restore the platform's default.
	PendingCacheDir *string `json:"pendingCacheDir,omitempty"`
	// SearchHistory keeps submitted keywords, most recent first.
	SearchHistory []string `json:"searchHistory,omitempty"`

	// LiquidGlass floats the player bar as a capsule of glass.
	LiquidGlass bool `json:"liquidGlass"`
	// GlassClear is how clear the glass of the player bar is, from 0 to 1,
	// and GlassBend how far it bends what shows through.
	GlassClear float64 `json:"glassClear"`
	GlassBend  float64 `json:"glassBend"`
	// GlassPanels makes menus, popovers, dialogs and panels frosted glass.
	GlassPanels bool `json:"glassPanels"`

	// Font is the font the app is read in (uiFonts); empty for the default.
	Font string `json:"font,omitempty"`

	// NpLight lights the line sung in Now Playing, and GlowLevel how far the
	// listener has turned that light up, 0.5 as it was tuned.
	NpLight   bool    `json:"npLight"`
	GlowLevel float64 `json:"glowLevel"`

	// Path is where they were loaded from, and are saved to.
	Path string `json:"-"`
}

// Default returns the choices of a first launch.
func Default() Settings {
	return Settings{NotchActivity: true, Quality: "exhigh", Appearance: "system", SidebarMaterial: "acrylic", LiquidGlass: true, GlassClear: 0.6, GlassBend: 0.3, GlassPanels: true, NpLight: true, GlowLevel: 0.5, Volume: 0.8, PlaybackFade: true, Normalize: true, AppIcon: "dock-and-menubar", CacheMB: 2048, Captions: true}
}

const (
	VolumeBalanceOff      = "off"
	VolumeBalanceQuiet    = "quiet"
	VolumeBalanceStandard = "standard"
	VolumeBalanceHigh     = "high"
)

// VolumeBalanceMode returns the volume balance mode chosen.
func (s Settings) VolumeBalanceMode() string {
	switch s.VolumeBalance {
	case VolumeBalanceOff, VolumeBalanceQuiet, VolumeBalanceStandard, VolumeBalanceHigh:
		return s.VolumeBalance
	}
	if s.Normalize {
		return VolumeBalanceStandard
	}
	return VolumeBalanceOff
}

// SetVolumeBalance chooses a volume balance mode.
func (s *Settings) SetVolumeBalance(mode string) {
	s.VolumeBalance = mode
	s.Normalize = mode != VolumeBalanceOff
}

const (
	ProgressStandard = "standard"
	ProgressSmoke    = "smoke"
	ProgressSquiggly = "squiggly"
)

// ProgressEffectMode returns the progress bar's look chosen.
func (s Settings) ProgressEffectMode() string {
	switch s.ProgressEffect {
	case ProgressStandard, ProgressSmoke, ProgressSquiggly:
		return s.ProgressEffect
	}
	if s.Smoke {
		return ProgressSmoke
	}
	return ProgressStandard
}

// SetProgressEffect chooses the progress bar's look.
func (s *Settings) SetProgressEffect(effect string) {
	s.ProgressEffect = effect
	s.Smoke = effect == ProgressSmoke
}

// Path returns where the settings are kept.
func Path() (string, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// Load reads the settings, or returns the defaults.
func Load() Settings {
	s := Default()
	if path, err := Path(); err == nil {
		if b, err := os.ReadFile(path); err == nil {
			json.Unmarshal(b, &s)
		}
		s.Path = path
	}
	return s
}

// Save keeps the settings where they were loaded from; defaults that were
// never loaded, as in tests, are not saved.
func (s Settings) Save() error {
	path := s.Path
	if path == "" {
		return nil
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// SidebarVibrancy keeps the Windows material choice across launches while
// preserving the platform's sidebar material elsewhere.
func (s Settings) SidebarVibrancy() mygo.Vibrancy {
	if runtime.GOOS != "windows" {
		return mygo.VibrancySidebar
	}
	if s.SidebarMaterial == "mica" {
		return mygo.VibrancyMica
	}
	return mygo.VibrancyAcrylic
}

// ApplyAppearance makes the windows light, dark, or follow the system.
func (s Settings) ApplyAppearance() {
	switch s.Appearance {
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}
