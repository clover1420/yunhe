package app

import (
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
)

const sidebarWidth = 232

func (a *app) view(c *ui.Context) {
	p := a.applyTheme(c)

	if a.cacheMoving {
		a.cacheMigrationView(c)
		return
	}

	a.shortcuts(c)

	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
	root := ui.Box(c).Fill()
	// The settings, a pane over the page, keep Space for their own controls.
	settingsFocused := a.settingsOpen || a.route.Page == pageSettings && !a.nowPlaying && a.settingsPane.FocusWithin(c)
	if !c.Focused(a.searchBox) && !settingsFocused && !a.lyricsSizeOpen && !a.lyricsSizeTrigger.Focused(c) {
		// Space plays and pauses wherever the focus is, as in Music. On
		// the container around everything, it comes before a button the
		// pointer focused, which would take Space as a click; on the
		// window, it would come after.
		root.OnShortcut(0, ui.KeySpace, a.pl.TogglePlay)
		// And where nothing has the focus, as once Now Playing covers the
		// button that had it.
		c.OnShortcut(0, ui.KeySpace, a.pl.TogglePlay)
	}
	npOpen, npSettled := a.npState(c)
	if npOpen || a.pl.Current() == nil {
		a.queueOpen = false
	}
	if !npOpen || a.npPane != paneStage {
		a.stageStyleOpen = false
	}
	if !npOpen || a.npPane != paneLyrics {
		a.lyricsSizeOpen = false
	}
	root.Children(func() {
		page := ui.Row(c).Fill().AlignItems(ui.Stretch)
		page.Children(func() {
			if npOpen && npSettled {
				// Covered by Now Playing: not built, which would cost each
				// of its frames as much again as the stage itself. Where
				// it is scrolled is the app's (pageStates, navScroll),
				// so it shows again as it was.
				return
			}
			side := ui.Column(c).Width(sidebarWidth)
			if !c.Vibrancy() {
				side.Background(p.Sidebar)
			} else if a.acrylicSidebar(c) {
				// Neutral frost quiets the desktop's color while leaving the
				// system acrylic visible beneath.
				side.Background(neutral(p.Dark, 2).Alpha(0.5))
			}
			side.Children(func() { a.sidebar(c) })
			// Keyed, so that a popover of the sidebar opening or closing
			// keeps what the page and its bars hold, their edges' fades.
			// The page and the sidebar are told apart by their tint alone,
			// as Cloudy's are: no rule runs between them.
			ui.Box(c.Key("main")).Grow(1).Background(p.Bg).Children(func() {
				a.content(c)
				// The bars float over the page, which scrolls under them:
				// positioned, they paint over it.
				a.pageTop(c)
				a.playerBar(c)
			})
		})
		a.nowPlayingLayer(c, npOpen, npSettled)
		a.splash(c)
	})
	a.settingsPanel(c)
	a.loginDialog(c)
	a.playlistNameDialog(c)
	a.deletePlaylistDialog(c)
	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}
	a.syncWindows()
}

func (a *app) shortcuts(c *ui.Context) {
	// The side buttons of a mouse, the keys of keyboards that have them
	// and swipes between pages, and the keys of Music.
	c.OnShortcut(0, ui.KeyBack, a.back)
	c.OnShortcut(0, ui.KeyForward, a.goForward)
	c.OnShortcut(ui.Cmd, ui.KeyBracketLeft, a.back)
	c.OnShortcut(ui.Cmd, ui.KeyBracketRight, a.goForward)
	c.OnShortcut(ui.Cmd, ui.KeyF, func() {
		a.nowPlaying = false
		a.searchBox.Focus(c)
	})
	if a.nowPlaying {
		c.OnShortcut(0, ui.KeyEscape, func() { a.nowPlaying = false })
		// ⌘ and a number choose Now Playing's tab of that place.
		if a.pl.Current() != nil {
			for i, t := range a.npTabs() {
				c.OnShortcut(ui.Cmd, ui.Key1+ui.Key(i), func() { a.choosePane(t.pane) })
			}
		}
	}
}

// go opens a page, remembering the one before for the back button, and
// closes Now Playing over it, though it is the page open.
func (a *app) open(r route) {
	a.nowPlaying = false
	a.queueOpen = false
	if r == a.route {
		return
	}
	a.history, a.forward = append(a.history, a.route), nil
	a.show(r)
}

// show shows the page of r.
func (a *app) show(r route) {
	a.queueOpen = false
	a.pageDir = pageLevel(r) - pageLevel(a.route)
	// A tab of the page open is not another page: it changes in place, so
	// that the control chosen stays where the pointer left it.
	a.pageIn = r.Page != a.route.Page || r.ID != a.route.ID
	a.route = r
	if r.Page == pageSearch {
		a.query, a.searched = r.Query, r
	}
	a.load(r)
}

// load loads what the page of r shows, unless it has.
func (a *app) load(r route) {
	switch r.Page {
	case pageHome:
		a.freshenHome()
	case pagePlaylist:
		a.loadPlaylist(r.ID, false)
		if r.Tab == 1 && a.isLikedPlaylist(r.ID) {
			a.loadLikedAlbums(false)
		}
	case pageSearch:
		a.loadSearch(r, false)
	case pageArtist:
		a.loadArtist(r.ID)
	case pageAlbum:
		a.loadAlbum(r.ID)
	case pageUser:
		a.loadUser(r.ID)
	case pagePodcast:
		a.loadPodcast(r.ID)
	case pagePodcasts:
		a.loadPodcastTab()
	case pagePurchased:
		a.loadPurchased(r.Tab, false)
	case pageSettings:
		a.measureCache()
	}
}

// back goes to the page before, as the back button of a mouse does, or
// closes Now Playing, which covers the pages.
func (a *app) back() {
	if a.nowPlaying {
		a.nowPlaying = false
		return
	}
	if len(a.history) == 0 {
		return
	}
	a.forward = append(a.forward, a.route)
	r := a.history[len(a.history)-1]
	a.history = a.history[:len(a.history)-1]
	a.show(r)
}

// goForward goes to the page back left, as the forward button of a mouse
// does.
func (a *app) goForward() {
	if len(a.forward) == 0 {
		return
	}
	a.history = append(a.history, a.route)
	r := a.forward[len(a.forward)-1]
	a.forward = a.forward[:len(a.forward)-1]
	a.nowPlaying = false
	a.show(r)
}

func (a *app) sidebar(c *ui.Context) {
	// Under the window buttons where they sit over the sidebar, as the
	// traffic lights do; elsewhere, as close to the top as to the sides.
	top := float32(14)
	if c.TitleBar().Left > 0 {
		top = topBar(c) + 4
	}
	// Where nothing in it takes the press, the sidebar moves the window, as
	// the title bar it reaches up into would.
	ui.Column(c).Fill().Padding(top, 14, 14, 14).Gap(4).DragWindow().Children(func() {
		a.searchField(c)

		ui.Scroll(c.Key("nav")).TrackScroll(&a.navScroll).Grow(1).Margin(0, -6).Padding(0, 6).Gap(4).Children(func() {
			a.navItem(c, iconHome, "为你推荐", route{Page: pageHome})
			a.navItem(c, iconDaily, "每日推荐", route{Page: pageDaily})
			a.navItem(c, iconRadio, "电台", route{Page: pageFM})
			a.navItem(c, iconCharts, "排行榜", route{Page: pageCharts})
			a.navItem(c, iconPurchased, "已购音乐", route{Page: pagePurchased, Tab: purchasedAlbums})
			if a.searched.Query != "" {
				a.navItem(c, iconSearch, "搜索结果", a.searched)
			}
			a.myPlaylists(c, "我的歌单", a.mine, true, &a.mineCollapsed)
			a.myPlaylists(c, "收藏的歌单", a.saved, false, &a.savedCollapsed)
		})

		a.navItem(c, iconSettings, "设置", route{Page: pageSettings})
		a.accountRow(c)
	})
}

func (a *app) navItem(c *ui.Context, icon *ui.SVG, label string, r route) {
	p := a.pal
	selected := a.route.Page == r.Page && !a.nowPlaying
	b := ui.ButtonBase(c.Key(label)).Height(34).PaddingX(9).Gap(10).Radius(8).AlignItems(ui.Center).Justify(ui.Start).FocusRing(false)
	a.styleSidebarItem(c, b, selected)
	color := p.Text
	if selected {
		color = p.Accent
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(17, 17).TextColor(color)
		ui.Text(c, label).FontSize(13.5).FontWeight(500).TextColor(p.Text)
	})
	if b.Clicked() {
		if r.Page == pageSettings {
			a.openSettings()
		} else {
			a.open(r)
		}
	}
}

// styleSidebarItem gives navigation and playlists the same selection cue:
// the row a press darkens, as Cloudy's sidebar marks the page open.
func (a *app) styleSidebarItem(c *ui.Context, b ui.Element, selected bool) {
	p := a.pal
	bg := ui.Transparent
	switch {
	case selected:
		bg = p.Press
	case b.Hovered():
		bg = p.Hover
	}
	b.Selected(selected).
		Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
}

// sidebarSurface lets the desktop's material show through quiet neutral
// controls. Other window materials keep their existing surfaces.
func (a *app) sidebarSurface(c *ui.Context, fallback ui.Color, opacity float32) ui.Color {
	if !a.acrylicSidebar(c) {
		return fallback
	}
	tint := ink
	if a.pal.Dark {
		tint = paper
	}
	return tint.Alpha(opacity)
}

func (a *app) acrylicSidebar(c *ui.Context) bool {
	return runtime.GOOS == "windows" && a.settings.SidebarVibrancy() == mygo.VibrancyAcrylic && c.Vibrancy()
}

// insets returns the room the bars over a page take at its top and its
// bottom: pages pad their content by it, and scroll under the bars.
func (a *app) insets(c *ui.Context) (top, bottom float32) {
	top = topBar(c)
	if runtime.GOOS == "darwin" {
		// The edge reaches a little below the traffic lights' strip.
		top += topEdgePad
	}
	return top, playerBarHeight
}

// topEdgePad is the room the edge over a page leaves below the strip, on
// macOS.
const topEdgePad = 6

// buttonsHeight is the height of the window buttons off macOS, which fill
// the edge over a page.
const buttonsHeight = topBarHeight + topEdgePad

// topBar returns the height of the strip along the top of the window, as
// tall as the window buttons and the room above and below them, which
// what sits in it centers on.
func topBar(c *ui.Context) float32 {
	if runtime.GOOS != "darwin" {
		// The whole edge, which the buttons fill: what sits in the strip
		// centers on their glyphs.
		return buttonsHeight
	}
	return max(c.TitleBar().Height, topBarHeight)
}

// topBarHeight is the least height of the strip along the top.
const topBarHeight = 40

// startMark marks where a page's content starts, first in what scrolls:
// once it has moved up, content is under the bar at the top.
func (a *app) startMark(c *ui.Context) {
	b := ui.Box(c.Key("start-mark")).Height(0).Bounds()
	if b.W > 0 {
		top, _ := a.insets(c)
		a.underTop = b.Y < top+pageGap-0.5
	}
}

// endMark marks where a page's content ends, last in what scrolls: while it
// is below the top of the player, content is under the player.
func (a *app) endMark(c *ui.Context) {
	b := ui.Box(c.Key("end-mark")).Height(0).Bounds()
	if b.W > 0 {
		_, h := c.Size()
		_, bottom := a.insets(c)
		a.underBottom = b.Y > h-bottom+0.5
	}
}

// pageTop is the strip over the top of a page, frosting what scrolls under
// it as macOS 26's hard scroll edge does: it drags the window, and holds
// the back button once there is a page to go back to.
func (a *app) pageTop(c *ui.Context) {
	top, _ := a.insets(c)
	edge := ui.Box(c.Key("top-edge")).Absolute().Top(0).Left(0).Right(0).Height(top).DragWindow().
		Material(glass.ScrollEdge{Hard: true, Background: a.pal.Bg})
	edge.Opacity(edgeShown(edge, a.underTop))
	if len(a.history) > 0 || len(a.forward) > 0 {
		ui.Row(c).Absolute().Top(0).Left(20).Height(topBar(c)).AlignItems(ui.Center).Children(func() {
			back := iconButton(c.Key("back"), iconBack, 18, a.pal.Muted, "返回").Disabled(len(a.history) == 0)
			if len(a.history) == 0 {
				back.Opacity(0.35)
			}
			if back.Clicked() {
				a.back()
			}
			if len(a.forward) > 0 && iconButton(c.Key("forward"), iconForward, 18, a.pal.Muted, "前进").Clicked() {
				a.goForward()
			}
		})
	}
}

func (a *app) content(c *ui.Context) {
	// Pages mark where their content is, as they scroll.
	a.underTop, a.underBottom = false, false
	// Keyed by its route, the page before leaves as this one comes.
	page := ui.Box(c.Key(keyOf(a.route))).Absolute().Top(0).Left(0).Right(0).Bottom(0)
	a.pageMotion(c, page)
	page.Children(func() {
		switch a.route.Page {
		case pageHome:
			a.homePage(c)
		case pageDaily:
			a.dailyPage(c)
		case pageCharts:
			a.chartsPage(c)
		case pagePlaylist:
			a.playlistPage(c)
		case pageSearch:
			a.searchPage(c)
		case pageFM:
			a.fmPage(c)
		case pageArtist:
			a.artistPage(c)
		case pageAlbum:
			a.albumPage(c)
		case pageUser:
			a.userPage(c)
		case pagePodcast:
			a.podcastPage(c)
		case pagePodcasts:
			a.podcastsPage(c)
		case pagePurchased:
			a.purchasedPage(c)
		case pageSettings:
			a.settingsPage(c)
		case pageSongShelf:
			a.songShelfPage(c)
		}
	})
}

// centered shows a message, with a retry button if retry is set: what a
// page shows in place of what failed to load.
func (a *app) centered(c *ui.Context, err string, retry func()) {
	top, bottom := a.insets(c)
	ui.Column(c).Grow(1).Padding(top, 0, bottom, 0).Center().Gap(12).Children(func() {
		ui.Text(c, err).TextColor(a.pal.Muted).FontSize(13)
		if retry != nil && ui.Button(c, "重试").Clicked() {
			retry()
		}
	})
}

// Loading what the pages show.

func (a *app) loadHome() {
	a.homeLoading, a.homeErr = true, ""
	a.loadSongShelves()
	signedIn := a.profile != nil
	at := time.Now()
	go func() {
		featured, err1 := qqmusic.FeaturedPlaylists(30)
		charts, err2 := qqmusic.Charts()
		daily, _ := qqmusic.DailySongs()
		var rec []qqmusic.Playlist
		if signedIn {
			rec, _ = qqmusic.RecommendedPlaylists()
		}
		a.post(func() {
			a.homeLoading = false
			if err1 != nil && err2 != nil {
				// Loading the new day's again, what shows stays, to try
				// once more.
				if len(a.featured) == 0 && len(a.charts) == 0 {
					a.homeErr = "无法连接 QQ 音乐，请检查网络"
				}
				return
			}
			a.featured, a.charts, a.daily, a.recPlaylists = featured, charts, daily, rec
			a.homeAt = at
		})
	}()
}

// freshenHome loads the home page again once NetEase has picked the day's
// songs anew since it last did: the app may stay open overnight.
func (a *app) freshenHome() {
	if !a.homeLoading && !a.homeAt.IsZero() && dailyRenewed(a.homeAt, time.Now()) {
		a.loadHome()
	}
}

// beijing is NetEase's time, without daylight saving.
var beijing = time.FixedZone("CST", 8*60*60)

// dailyRenewed reports whether NetEase has picked new daily songs between
// loaded and now. It does at 6:00, Beijing time.
func dailyRenewed(loaded, now time.Time) bool {
	b := now.In(beijing)
	six := time.Date(b.Year(), b.Month(), b.Day(), 6, 0, 0, 0, beijing)
	if b.Before(six) {
		six = six.AddDate(0, 0, -1)
	}
	return loaded.Before(six)
}

func (a *app) loadPlaylist(id int64, thenPlay bool) {
	if pl := a.playlists[id]; pl != nil {
		if thenPlay {
			a.pl.PlayQueue(pl.Tracks, firstPlayable(pl.Tracks))
		}
		return
	}
	if a.loadingPl[id] {
		return
	}
	a.loadingPl[id] = true
	delete(a.plErr, id)
	go func() {
		pl, err := qqmusic.PlaylistDetail(id)
		a.post(func() {
			delete(a.loadingPl, id)
			if err != nil {
				a.plErr[id] = "歌单加载失败"
				return
			}
			a.playlists[id] = pl
			if thenPlay {
				a.pl.PlayQueue(pl.Tracks, firstPlayable(pl.Tracks))
			}
		})
	}()
}

func firstPlayable(tracks []qqmusic.Track) int {
	for i := range tracks {
		if tracks[i].Playable() {
			return i
		}
	}
	return 0
}
