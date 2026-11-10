// Package settingsui owns the settings, as a pane over the main window, as
// a window of their own, and as the settings page.
package settingsui

import (
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
	"lilt/internal/ui/features/lyricsui"
)

// PageGap is the room between the bar over a page and its content.
const PageGap = 12

// The destinations down the settings' side.
const (
	General = iota
	Appearance
	Playback
	About
)

var categories = [...]struct {
	label string
	icon  *ui.SVG
}{
	{"常规", appui.IconSettings},
	{"外观", appui.IconSun},
	{"播放与歌词", appui.IconHeadset},
	{"关于", appui.IconInfo},
}

// AppIcons are where the app's icon shows on macOS, with their names.
var AppIcons = []struct{ ID, Label string }{
	{"dock-and-menubar", "程序坞与菜单栏"},
	{"dock", "仅程序坞"},
	{"menubar", "仅菜单栏"},
}

// QualityLevels are the stream levels QQ Music names a quality by.
var QualityLevels = []struct{ Level, Label string }{
	{"standard", "标准（128k）"},
	{"exhigh", "极高（320k）"},
	{"lossless", "无损（FLAC）"},
}

// VolumeBalances are the ways the app balances loudness.
var VolumeBalances = []struct{ ID, Label string }{
	{"off", "关闭"},
	{"quiet", "安静"},
	{"standard", "标准"},
	{"high", "高音量"},
}

// CacheLimits are the sizes the song cache may take, in megabytes.
var CacheLimits = []struct {
	MB    int
	Label string
}{
	{0, "关闭"},
	{1024, "1 GB"},
	{2048, "2 GB"},
	{5120, "5 GB"},
	{10240, "10 GB"},
}

// ProgressEffects are the looks the progress bar may take.
var ProgressEffects = []struct{ ID, Label string }{
	{"standard", "标准"},
	{"smoke", "烟雾"},
	{"squiggly", "波浪"},
}

// Ink holds the colors of what is on a pane.
type Ink struct {
	Text, Muted ui.Color
}

// Font names a font the interface may use.
type Font struct {
	ID, Label string
}

// Deps are what the settings need from the app: the settings by pointer,
// the controls' state, and what only the app can do by callback.
type Deps struct {
	Settings *settings.Settings
	Palette  theme.Palette
	Window   *mygo.Window

	Open   *bool
	Tab    *int
	Pane   *ui.Handle
	Scroll *[4]ui.ScrollState
	Dirty  *bool

	Notch       bool
	Updates     bool
	CacheMoving bool
	// TaskbarLyrics shows the taskbar panel's material choice.
	TaskbarLyrics bool

	Theme                    func(*ui.Context) theme.Palette
	Ink                      func(onDark bool) Ink
	PaneLook                 func(ui.Element, float32, bool) ui.Element
	Insets                   func(*ui.Context) (top, bottom float32)
	StartMark                func(*ui.Context)
	EndMark                  func(*ui.Context)
	MeasureCache             func()
	ShowMain                 func()
	Redraw                   func()
	Changed                  func()
	CanHideOnClose           func() bool
	ChangeAppIcon            func()
	CacheUsage               func() string
	CacheUsed                func() int64
	CacheDirRow              func(*ui.Context)
	ApplyCacheLimit          func()
	ClearCache               func()
	RefreshGain              func()
	SetLyricsMode            func(string)
	SetTaskbarLyricsMaterial func(string)
	Fonts                    func() []Font
	LoadFont                 func()

	CheckForUpdates    func()
	LastUpdateCheck    func() time.Time
	UpdateChecks       func() bool
	SetUpdateChecks    func(bool)
	UpdateDownloads    func() bool
	SetUpdateDownloads func(bool)
	Version            func() string
}

// Panel is the settings, a pane over the window while they are open.
func Panel(c *ui.Context, d Deps) {
	if !*d.Open {
		return
	}
	w, h := c.Size()
	ink := d.Ink(false)
	ui.DialogBase(c.Key("settings"), d.Open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.28))
		panel.Width(min(860, w-48)).Height(max(h-96, 360)).TextColor(ink.Text).Label("设置").TransformOrigin(0, 1)
		d.PaneLook(panel, 26, false)
		panel.Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.96, Y: 12}, Exit: &ui.Motion{Scale: 0.98}})
		panel.Children(func() {
			ui.Row(c).Height(60).Padding(0, 16, 0, 28).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "设置").FontSize(20).FontWeight(700).Grow(1)
				if appui.IconButton(c.Key("close"), appui.IconX, 16, ink.Muted, "关闭").Clicked() {
					*d.Open = false
				}
			})
			ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Padding(0, 0, 12, 12).Children(func() {
				inner(c, d)
			})
		})
	})
}

// View is the settings as a window of their own.
func View(c *ui.Context, d Deps) {
	p := d.Theme(c)
	c.Root().Background(p.Sidebar)
	inner(c, d)
}

// Page is the settings page, within the window of the pages.
func Page(c *ui.Context, d Deps) {
	top, bottom := d.Insets(c)
	ui.Box(c.Key("settings-page")).Fill().Background(d.Palette.Bg).Padding(top+PageGap, 0, bottom, 0).Children(func() {
		d.StartMark(c)
		defer d.EndMark(c)
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			sidebar(c, d)
			ui.Box(c).Width(0.5).Shrink(0).Background(d.Palette.Hairline)
			body(c, d)
		})
	})
}

// inner lays the destinations beside what they show.
func inner(c *ui.Context, d Deps) {
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		sidebar(c, d)
		ui.Box(c).Width(0.5).Shrink(0).Background(theme.For(c.Theme()).Hairline)
		body(c, d)
	})
}

// sidebar is the list of destinations, which stays while one of them
// scrolls.
func sidebar(c *ui.Context, d Deps) {
	p := theme.For(c.Theme())
	ui.Column(c.Key("settings-nav")).Width(168).Shrink(0).Padding(16, 12, 16, 4).Gap(4).Label("设置分类").Children(func() {
		for i, category := range categories {
			selected := *d.Tab == i
			button := ui.ButtonBase(c.Key(category.label)).Height(38).PaddingX(10).Gap(10).Radius(10).
				AlignItems(ui.Center).Justify(ui.Start).Label("设置分类：" + category.label).Selected(selected)
			bg, color, weight := ui.Transparent, p.Muted, 500
			if selected {
				bg, color, weight = p.Press, p.Text, 600
			} else if button.Hovered() {
				bg, color = p.Hover, p.Text
			}
			button.Background(button.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond})).TextColor(color)
			button.Children(func() {
				ui.Icon(c, category.icon).Size(17, 17)
				ui.Text(c, category.label).FontSize(13.5).FontWeight(weight)
			})
			if button.Clicked() {
				*d.Tab = i
			}
		}
	})
}

// body scrolls the rows of the destination chosen, and keeps its place
// between visits.
func body(c *ui.Context, d Deps) {
	ui.Scroll(c.Key(categories[*d.Tab].label)).Bind(d.Pane).
		TrackScroll(&d.Scroll[*d.Tab]).Grow(1).Shrink(1).MinWidth(0).
		Padding(8, 24, 24, 24).Gap(8).Children(func() {
		content(c, d)
	})
}

func content(c *ui.Context, d Deps) {
	p := theme.For(c.Theme())
	s := d.Settings

	// A group of the destination showing, and none of the others: their rows
	// are not built, so that no control of another page takes a key.
	group := func(cat int, title string, rows func()) {
		if *d.Tab != cat {
			return
		}
		ui.Text(c, title).FontSize(12).FontWeight(600).TextColor(p.Muted).Margin(14, 4, 4, 4)
		ui.Column(c).Radius(20).Background(p.Bg).Border(1, p.Hairline).Dividers(1, p.Hairline).Children(rows)
	}
	row := func(label, desc string, control func()) {
		ui.Row(c).Padding(12, 16).Gap(16).AlignItems(ui.Center).MinHeight(48).Children(func() {
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Children(func() {
				ui.Text(c, label).FontSize(13.5).FontWeight(500)
				if desc != "" {
					ui.Text(c, desc).FontSize(12).TextColor(p.Muted).LineHeight(1.4)
				}
			})
			control()
		})
	}

	group(General, "通用", func() {
		if d.CanHideOnClose() {
			row("关闭主窗口时", "", func() {
				i := 0
				if s.QuitOnClose {
					i = 1
				}
				keepRunning := "最小化到托盘"
				switch runtime.GOOS {
				case "darwin":
					keepRunning = "保持后台运行"
				case "linux":
					keepRunning = "隐藏到托盘"
				}
				if appui.Segmented(c.Key("close-action"), &i, keepRunning, "退出应用") {
					s.QuitOnClose = i == 1
					s.Save()
				}
			})
			// Only on macOS: elsewhere the icon is always in the notification
			// area.
			if runtime.GOOS == "darwin" {
				row("应用图标", "显示在菜单栏时，关闭窗口后可从菜单栏图标打开", func() {
					labels := make([]string, len(AppIcons))
					chosen := AppIcons[0].Label
					for i, ic := range AppIcons {
						labels[i] = ic.Label
						if ic.ID == s.AppIcon {
							chosen = ic.Label
						}
					}
					if ui.Select(c.Key("app-icon"), &chosen, labels).Label("应用图标").Width(150).Changed() {
						for _, ic := range AppIcons {
							if ic.Label == chosen {
								s.AppIcon = ic.ID
							}
						}
						d.Changed()
						d.ChangeAppIcon()
					}
				})
			}
		}

		// Only on macOS, the one with a notch player.
		if d.Notch {
			row("显示刘海播放器", "在屏幕顶部的刘海处显示迷你播放器，鼠标悬停即可展开", func() {
				if ui.Switch(c.Key("notch"), &s.Notch).Label("显示刘海播放器").Changed() {
					d.Changed()
				}
			})
			row("播放时显示实时活动", "在刘海两侧显示封面与音频波形，可能遮挡刘海旁的菜单栏图标", func() {
				if ui.Switch(c.Key("activity"), &s.NotchActivity).Label("播放时显示实时活动").Disabled(!s.Notch).Changed() {
					d.Changed()
				}
			})
			row("在无刘海的屏幕上常驻显示", "关闭时，将鼠标移至屏幕顶部中央即可唤出，不会遮挡菜单栏", func() {
				if ui.Switch(c.Key("virtual"), &s.NotchVirtual).Label("在无刘海的屏幕上常驻显示").Disabled(!s.Notch).Changed() {
					d.Changed()
				}
			})
		}

		row("音质", "按 QQ 音乐账号权限播放 MP3 或 FLAC，下一首生效", func() {
			labels := make([]string, len(QualityLevels))
			chosen := QualityLevels[1].Label
			for i, q := range QualityLevels {
				labels[i] = q.Label
				if q.Level == s.Quality {
					chosen = q.Label
				}
			}
			if ui.Select(c.Key("quality"), &chosen, labels).Label("音质").Width(150).Changed() {
				for _, q := range QualityLevels {
					if q.Label == chosen {
						s.Quality = q.Level
					}
				}
				d.Changed()
			}
		})
		row("音量平衡", "", func() {
			labels := make([]string, len(VolumeBalances))
			chosen := VolumeBalances[2].Label
			for i, balance := range VolumeBalances {
				labels[i] = balance.Label
				if balance.ID == s.VolumeBalanceMode() {
					chosen = balance.Label
				}
			}
			if ui.Select(c.Key("volume-balance"), &chosen, labels).Label("选择音量平衡").Width(150).Changed() {
				for _, balance := range VolumeBalances {
					if balance.Label == chosen {
						s.SetVolumeBalance(balance.ID)
					}
				}
				d.RefreshGain()
				s.Save()
				d.Redraw()
			}
		})
		row("播放操作淡入淡出", "", func() {
			if ui.Switch(c.Key("playback-fade"), &s.PlaybackFade).Label("切换播放操作淡入淡出").Changed() {
				s.Save()
				d.Redraw()
			}
		})
		row("间奏时显示热评", "前奏、间奏与尾奏时，歌词处换成这首歌的热门评论", func() {
			if ui.Switch(c.Key("hot-comments"), &s.HotComments).Label("切换热评").Changed() {
				s.Save()
				d.Redraw()
			}
		})

		row("歌曲缓存", d.CacheUsage(), func() {
			labels := make([]string, len(CacheLimits))
			chosen := CacheLimits[2].Label
			for i, l := range CacheLimits {
				labels[i] = l.Label
				if l.MB == s.CacheMB {
					chosen = l.Label
				}
			}
			if ui.Select(c.Key("cache"), &chosen, labels).Label("歌曲缓存").Width(150).Disabled(d.CacheMoving).Changed() {
				for _, l := range CacheLimits {
					if l.Label == chosen {
						s.CacheMB = l.MB
					}
				}
				d.Changed()
				d.ApplyCacheLimit()
			}
		})
		d.CacheDirRow(c)
		row("清除缓存", "", func() {
			if ui.Button(c.Key("clear-cache"), "清除").Disabled(d.CacheMoving || d.CacheUsed() <= 0).Clicked() {
				d.ClearCache()
			}
		})
	})

	group(Appearance, "外观与玻璃", func() {
		row("主题模式", "", func() {
			modes := []string{"system", "light", "dark"}
			i := 0
			for k, m := range modes {
				if m == s.Appearance {
					i = k
				}
			}
			if appui.Segmented(c.Key("appearance"), &i, "跟随系统", "浅色", "深色") {
				s.Appearance = modes[i]
				d.Changed()
			}
		})
		row("主题色跟随封面", "按钮与高亮使用当前歌曲封面的颜色，而不是默认红", func() {
			if ui.Switch(c.Key("accent-cover"), &s.AccentCover).Label("主题色跟随封面").Changed() {
				d.Changed()
			}
		})
		if runtime.GOOS == "windows" {
			row("侧边栏材质", "", func() {
				i := 0
				if s.SidebarVibrancy() == mygo.VibrancyMica {
					i = 1
				}
				if appui.Segmented(c.Key("sidebar-material"), &i, "亚克力", "Mica") {
					s.SidebarMaterial = []string{"acrylic", "mica"}[i]
					if d.Window != nil {
						d.Window.SetVibrancy(s.SidebarVibrancy())
					}
					s.Save()
					d.Redraw()
				}
			})
		}

		row("字体", "界面文字所用的字体，随系统所装字体而定", func() {
			fonts := d.Fonts()
			labels := make([]string, len(fonts))
			chosen := fonts[0].Label
			for i, f := range fonts {
				labels[i] = f.Label
				if f.ID == s.Font {
					chosen = f.Label
				}
			}
			if ui.Select(c.Key("font"), &chosen, labels).Label("字体").Width(200).Changed() {
				for _, f := range fonts {
					if f.Label == chosen {
						s.Font = f.ID
					}
				}
				d.LoadFont()
				d.Changed()
			}
		})
		row("液态玻璃播放栏", "播放栏浮起为一条玻璃胶囊，页面从它底下滚过", func() {
			if ui.Switch(c.Key("liquid-glass"), &s.LiquidGlass).Label("液态玻璃播放栏").Changed() {
				d.Changed()
			}
		})
		glassRow := func(label, desc string, v *float64) {
			row(label, desc, func() {
				valueSlider(c.Key(label), d, label, label, v, s.LiquidGlass)
			})
		}
		glassRow("玻璃透明度", "越高越能看清背后的内容，文字会更难读", &s.GlassClear)
		glassRow("折射强度", "胶囊边缘弯折背后内容的程度", &s.GlassBend)
		row("界面玻璃", "菜单、弹层与对话框使用磨砂玻璃", func() {
			if ui.Switch(c.Key("glass-panels"), &s.GlassPanels).Label("界面玻璃").Changed() {
				d.Changed()
			}
		})
		row("歌词光晕", "正在唱的那一行，在封面的颜色里发光", func() {
			if ui.Switch(c.Key("np-light"), &s.NpLight).Label("歌词光晕").Changed() {
				d.Changed()
			}
		})
		row("光晕强度", "越高越亮，字也越容易糊在一起", func() {
			valueSlider(c.Key("glow-level"), d, "光晕强度", "光晕强度", &s.GlowLevel, s.NpLight)
		})
	})

	group(Playback, "歌词与播放", func() {
		if lyricsui.SupportsTaskbarLyrics() {
			row("外部歌词", "", func() {
				i := 0
				if lyricsui.Mode(*s) == lyricsui.TaskbarMode {
					i = 1
				}
				if appui.Segmented(c.Key("lyrics-mode"), &i, "桌面歌词", "任务栏歌词") {
					mode := lyricsui.DesktopMode
					if i == 1 {
						mode = lyricsui.TaskbarMode
					}
					d.SetLyricsMode(mode)
				}
			})
		}
		if lyricsui.SupportsDesktopLyrics() && lyricsui.Mode(*s) == lyricsui.DesktopMode {
			row("桌面歌词行数", "双行仅针对中文歌", func() {
				i := 0
				if s.DesktopLyricsDoubleLine {
					i = 1
				}
				if appui.Segmented(c.Key("desktop-lyrics-lines"), &i, "单行", "双行") {
					s.DesktopLyricsDoubleLine = i == 1
					s.Save()
					d.Redraw()
				}
			})
			row("桌面歌词颜色", "", func() {
				desktopLyricsColorControl(c, d)
			})
		}
		if d.TaskbarLyrics {
			row("任务栏面板材质", "", func() {
				i := 0
				if s.TaskbarLyricsMaterial == lyricsui.TaskbarMaterialWhite {
					i = 1
				}
				if appui.Segmented(c.Key("taskbar-lyrics-material"), &i, "亚克力", "白色") {
					d.SetTaskbarLyricsMaterial([]string{lyricsui.TaskbarMaterialAcrylic, lyricsui.TaskbarMaterialWhite}[i])
				}
			})
		}
		row("进度条特效", "", func() {
			chosen := ProgressEffects[0].Label
			labels := make([]string, len(ProgressEffects))
			for k, effect := range ProgressEffects {
				labels[k] = effect.Label
				if effect.ID == s.ProgressEffectMode() {
					chosen = effect.Label
				}
			}
			if ui.Select(c.Key("progress-effect"), &chosen, labels).Label("选择进度条特效").Width(150).Changed() {
				for _, effect := range ProgressEffects {
					if effect.Label == chosen {
						s.SetProgressEffect(effect.ID)
					}
				}
				s.Save()
				d.Redraw()
			}
		})
		row("烟雾背景", "播放页的背景化作封面色调的烟雾，缓缓升起，随低音起伏；仍在打磨，会多用一些电量", func() {
			if ui.Switch(c.Key("fog"), &s.Fog).Label("烟雾背景").Changed() {
				d.Changed()
			}
		})
	})

	group(About, "关于与更新", func() {
		row("当前版本 "+d.Version(), "", func() {
			if d.Updates {
				if ui.Button(c.Key("check-updates"), "检查更新").Clicked() {
					d.CheckForUpdates()
				}
			}
		})
		if d.Updates {
			desc := ""
			if t := d.LastUpdateCheck(); !t.IsZero() {
				desc = "上次检查：" + appui.RelativeTime(t, time.Now())
			}
			row("自动检查更新", desc, func() {
				on := d.UpdateChecks()
				if ui.Switch(c.Key("update-checks"), &on).Label("自动检查更新").Changed() {
					d.SetUpdateChecks(on)
				}
			})
			row("自动下载并安装", "", func() {
				on := d.UpdateDownloads()
				if ui.Switch(c.Key("update-downloads"), &on).Label("自动下载并安装").Disabled(!d.UpdateChecks()).Changed() {
					d.SetUpdateDownloads(on)
				}
			})
		}
	})
}

// valueSlider is a setting that goes from 0 to 1: a thin bar in the accent
// over the page's press shade, with the value beside it, dimmed while the
// switch it belongs to is off.
func valueSlider(c *ui.Context, d Deps, key, label string, v *float64, on bool) {
	// The settings window draws its own palette, which the app's may not be,
	// but its accent follows the whole app's, cover included.
	p := theme.For(c.Theme())
	if d.Palette.Accent.A > 0 {
		p.Accent = d.Palette.Accent
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		s := ui.SliderBase(c.Key(key), v, 0, 1).Width(150).Height(22).Step(0.01).FocusRing(false).Label(label).
			Disabled(!on).Cursor(ui.CursorPointer)
		if s.Changed() {
			*d.Dirty = true
			d.Redraw()
		}
		if *d.Dirty && !s.Pressed() {
			*d.Dirty = false
			d.Settings.Save()
		}
		held := s.Pressed()
		thick := float32(4)
		if s.Hovered() || held {
			thick = 5
		}
		thick = s.AnimateTo("thick", thick, ui.Snappy)
		at := float32(*v)
		hot := held || s.Hovered() || s.FocusVisible()
		track, fill, knob := p.Press, p.Accent, theme.Paper
		if !on {
			// Off, the slider says so by its colors rather than by a layer.
			track, fill, knob = p.Hover, p.Faint, p.Hover
		}
		s.Draw(func(pel *ui.Painter, r ui.Rect) {
			y := r.Y + (r.H-thick)/2
			pel.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: thick}, track, thick/2)
			pel.Fill(ui.Rect{X: r.X, Y: y, W: max(thick, r.W*at), H: thick}, fill, thick/2)
			if hot {
				pel.Fill(ui.Rect{X: r.X + r.W*at - 6, Y: r.Y + (r.H-12)/2, W: 12, H: 12}, knob, 6)
			}
		})
		ui.Textf(c, "%d%%", int(100*(*v))).FontSize(12).FontFeatures("tnum").TextColor(p.Muted).Width(38)
	})
}
