package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"lilt/internal/ui/features/settingsui"
)

// The settings live in lilt/internal/ui/features/settingsui; the app keeps its state
// and these names while its other views still live here.
const (
	settingsGeneral    = settingsui.General
	settingsAppearance = settingsui.Appearance
	settingsPlayback   = settingsui.Playback
	settingsAbout      = settingsui.About
)

const (
	pageGap = settingsui.PageGap
)

var progressEffects = settingsui.ProgressEffects

func (a *app) settingsPanel(c *ui.Context) { settingsui.Panel(c, a.settingsDeps()) }
func (a *app) settingsView(c *ui.Context)  { settingsui.View(c, a.settingsDeps()) }
func (a *app) settingsPage(c *ui.Context)  { settingsui.Page(c, a.settingsDeps()) }

// settingsDeps gathers what the settings need, once a build.
func (a *app) settingsDeps() settingsui.Deps {
	return settingsui.Deps{
		Settings:                 &a.settings,
		Palette:                  a.pal,
		Window:                   a.win,
		Open:                     &a.settingsOpen,
		Tab:                      &a.settingsTab,
		Pane:                     &a.settingsPane,
		Scroll:                   &a.settingsScroll,
		Dirty:                    &a.settingsDirty,
		Notch:                    a.notch != nil,
		Updates:                  a.updates,
		CacheMoving:              a.cacheMoving,
		Theme:                    a.applyTheme,
		Ink:                      a.settingsInk,
		PaneLook:                 a.pane,
		Insets:                   a.insets,
		StartMark:                a.startMark,
		EndMark:                  a.endMark,
		MeasureCache:             a.measureCache,
		ShowMain:                 a.showMain,
		Redraw:                   a.redraw,
		Changed:                  a.changed,
		CanHideOnClose:           a.canHideOnClose,
		ChangeAppIcon:            a.changeAppIcon,
		CacheUsage:               a.cacheUsage,
		CacheUsed:                func() int64 { return a.cacheUsed },
		CacheDirRow:              a.cacheDirectoryRow,
		ApplyCacheLimit:          a.applyCacheLimit,
		ClearCache:               a.clearCache,
		RefreshGain:              func() { a.pl.RefreshGain() },
		SetLyricsMode:            a.setLyricsMode,
		TaskbarLyrics:            supportsTaskbarLyrics() && lyricsMode(a.settings) == "taskbar",
		SetTaskbarLyricsMaterial: a.setTaskbarLyricsMaterial,
		Fonts:                    a.settingsFonts,
		LoadFont:                 a.loadFont,
		CheckForUpdates:          checkForUpdates,
		LastUpdateCheck:          lastUpdateCheck,
		UpdateChecks:             updateChecks,
		SetUpdateChecks:          setUpdateChecks,
		UpdateDownloads:          updateDownloads,
		SetUpdateDownloads:       setUpdateDownloads,
		Version:                  func() string { return mygo.App.Version() },
	}
}

func (a *app) settingsInk(onDark bool) settingsui.Ink {
	ink := a.inkOn(onDark)
	return settingsui.Ink{Text: ink.text, Muted: ink.muted}
}

func (a *app) settingsFonts() []settingsui.Font {
	fonts := a.fontChoices()
	out := make([]settingsui.Font, len(fonts))
	for i, f := range fonts {
		out[i] = settingsui.Font{ID: f.id, Label: f.label}
	}
	return out
}
