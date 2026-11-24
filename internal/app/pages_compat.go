package app

import (
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/navigation"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/features/pages"
)

// The shapes the pages share live in lilt/internal/ui/features/pages; the app keeps
// these names while its page bodies still live here.
const pagePad = pages.Pad

var (
	weekday = pages.Weekday
)

type listScrollState = pages.ListState

const artistPhotoSize = pages.PhotoSize

type spinState = pages.SpinState

const (
	dailyHeight = pages.DailyHeight
	dailyRadius = pages.DailyRadius
	dailyPadX   = pages.DailyPadX
	stackWidth  = pages.StackWidth
	sleeveLeft  = pages.SleeveLeft
	sleeveSize  = pages.SleeveSize
	recordSize  = pages.RecordSize
	dailyRate   = pages.DailyRate
)

func (a *app) dailyDeps() pages.DailyDeps {
	return pages.DailyDeps{
		Palette:   a.pal,
		Art:       a.art,
		Spin:      &a.dailySpin,
		Bars:      &a.dailyBars,
		Player:    a.pl,
		Daily:     a.daily,
		SignedIn:  a.profile != nil,
		Sidebar:   sidebarWidth,
		SongItems: a.songItems,
		ArtistLinks: func(c *ui.Context, tr *qqmusic.Track, color, hover ui.Color) ui.Element {
			return a.artistLinks(c, tr, color, hover)
		},
		CoverFill:   a.coverFill,
		BarsFn:      a.nowPlayingBars,
		Login:       a.openLogin,
		Open:        func(r navigation.Route) { a.open(r) },
		HomeLoading: a.homeLoading,
		Centered:    a.centered,
		LoadHome:    a.loadHome,
		Skeleton: func(c *ui.Context, header func(*ui.Context)) {
			a.trackSkeleton(c, a.pageShimmer(c), true, header)
		},
		SkeletonHeader: func(c *ui.Context) {
			s := a.pageShimmer(c)
			s.Text(c, 13, 1.3).Width(240)
			ui.Row(c).Gap(10).Margin(8, 0, 0, 0).Children(func() {
				s.Bone(c, 18).Size(112, 36)
				s.Bone(c, 18).Size(112, 36)
			})
		},
		Tracks: func(c *ui.Context, tracks []qqmusic.Track, header func(*ui.Context)) {
			a.trackRows(c, tracks, true, header)
		},
	}
}

func (a *app) likedDeps() pages.LikedDeps {
	return pages.LikedDeps{
		Route:  &a.route,
		Player: a.pl,
		Card:   a.pagesDeps(),
		Album:  a.albumDeps(),
		Mine:   a.mine,
		Playlist: func(id int64) (*qqmusic.Playlist, bool, string) {
			return a.playlists[id], a.loadingPl[id], a.plErr[id]
		},
		Albums: func() pages.LikedAlbumsInfo {
			d := a.likedAlbums
			if d == nil {
				return pages.LikedAlbumsInfo{}
			}
			return pages.LikedAlbumsInfo{
				Albums: d.albums, Loaded: d.page.loaded,
				Loading: d.page.loading, More: d.page.more, Err: d.page.err,
			}
		},
		State: func() pages.LikedScroll {
			return pages.LikedScroll{Songs: &a.likedScroll.songs, Albums: &a.likedScroll.albums}
		},
		ListPage: a.listPageWithKey,
		TrackRows: func(c *ui.Context, key any, state *pages.ListState, tracks []qqmusic.Track, header func(*ui.Context)) {
			a.trackRowsWithKey(c, key, state, tracks, true, header)
		},
		TrackSkeleton: func(c *ui.Context, header func(*ui.Context)) {
			a.trackSkeletonRows(c, a.pageShimmer(c), true, header)
		},
		CardsSkeleton: func(c *ui.Context) {
			a.playlistCardsSkeleton(c, a.pageShimmer(c), a.columns(c, 156), 156, false)
		},
		UnderTop:    &a.underTop,
		UnderBottom: &a.underBottom,
		PageFoot: func(c *ui.Context, more func()) {
			if d := a.likedAlbums; d != nil {
				a.pageFoot(c, &d.page, more)
			}
		},
		Empty:    a.empty,
		Centered: a.centered,
		Show:     func(r navigation.Route) { a.show(r) },
		LoadPlaylist: func(id int64) {
			a.loadPlaylist(id, false)
		},
		LoadAlbums: a.loadLikedAlbums,
		MoreAlbums: func() {
			if d := a.likedAlbums; d != nil {
				a.moreLikedAlbums(d)
			}
		},
	}
}

func (a *app) likedPlaylistPage(c *ui.Context, id int64) {
	pages.LikedPlaylistPage(c, a.likedDeps(), id)
}

func (a *app) playlistEditDeps() pages.PlaylistEditDeps {
	return pages.PlaylistEditDeps{
		Palette:  a.pal,
		PaneLook: a.pane,
		Name: func() pages.NameState {
			d := &a.naming
			return pages.NameState{Open: &d.Open, Name: &d.Name, Rename: d.Rename, Busy: d.Busy}
		},
		Delete: func() pages.DeleteState {
			d := &a.deleting
			return pages.DeleteState{Open: &d.Open, Name: d.Name, Saved: d.Saved}
		},
		SubmitName:   a.submitName,
		DeleteChosen: a.deleteChosen,
	}
}

func (a *app) playlistNameDialog(c *ui.Context) {
	pages.PlaylistNameDialog(c, a.playlistEditDeps())
}

func (a *app) deletePlaylistDialog(c *ui.Context) {
	pages.DeletePlaylistDialog(c, a.playlistEditDeps())
}

func (a *app) radioDeps() pages.RadioDeps {
	modes := make([]pages.RadioMode, len(fmModes))
	for i, m := range fmModes {
		modes[i] = pages.RadioMode{Mode: m.mode, Label: m.label}
	}
	return pages.RadioDeps{
		Palette:       a.pal,
		Art:           a.art,
		Player:        a.pl,
		Modes:         modes,
		FMIndex:       &a.fmMode,
		Scroll:        &a.pageState().scroll,
		Insets:        a.insets,
		StartMark:     a.startMark,
		EndMark:       a.endMark,
		Radio:         a.pl.Radio,
		RadioLabel:    radioLabel,
		RadioStarting: func() string { return a.radioStarting },
		StartingHeart: func() bool { return a.radioStarting == heartRadio },
		SignedIn:      a.profile != nil,
		OnHeart:       a.pl.Radio() == heartRadio,
		PlayFM:        a.playFM,
		ExitRadio:     a.exitRadio,
		StartHeart:    a.startHeartMode,
		ModeChanged:   a.playFM,
		Like:          func(c *ui.Context, tr *qqmusic.Track) { a.likeButton(c, tr, 20, white.Alpha(0.8)) },
		ArtistLinks:   a.artistLinks,
		CoverFill:     a.coverFill,
	}
}

func (a *app) fmPage(c *ui.Context) {
	pages.FMPage(c, a.radioDeps())
}

func (a *app) dailyCard(c *ui.Context, now time.Time) {
	pages.DailyCard(c, a.dailyDeps(), now)
}

func (a *app) dailyPage(c *ui.Context) {
	pages.DailyPage(c, a.dailyDeps())
}

func (a *app) dailyPicks(c *ui.Context, songs []qqmusic.Track) {
	pages.DailyPicks(c, a.dailyDeps(), songs)
}

const (
	artistHot    = pages.ArtistHot
	artistLatest = pages.ArtistLatest
	artistAlbums = pages.ArtistAlbums
	artistEvents = pages.ArtistEvents
)

func (a *app) artistDeps() pages.ArtistDeps {
	id := a.route.ID
	return pages.ArtistDeps{
		Palette: a.pal,
		Route:   &a.route,
		Player:  a.pl,
		Tab:     &a.artistTab,
		Info: func() pages.ArtistInfo {
			d := a.artists[id]
			if d == nil {
				return pages.ArtistInfo{}
			}
			return pages.ArtistInfo{
				Profile: d.profile, Loading: d.loading, Err: d.err,
				Latest: d.latest, LatestLoading: d.latestLoading, LatestErr: d.latestErr,
				Albums: d.albums, AlbumsLoaded: d.albumsPage.loaded,
				AlbumsLoading: d.albumsPage.loading, AlbumsMore: d.albumsPage.more,
				AlbumsErr: d.albumsPage.err,
			}
		},
		Load:       a.loadArtist,
		LoadLatest: a.loadLatest,
		LoadAlbums: a.loadAlbums,
		Centered:   a.centered,
		Skeleton: func(c *ui.Context) {
			s := a.pageShimmer(c)
			a.trackSkeleton(c, s, true, func(c *ui.Context) { a.artistHeaderSkeleton(c, s) })
		},
		Tracks: func(c *ui.Context, tracks []qqmusic.Track, header func(*ui.Context)) {
			a.trackRows(c, tracks, true, header)
		},
		ScrollPage: a.scrollPage,
		PageFoot: func(c *ui.Context, more func()) {
			if d := a.artists[id]; d != nil {
				a.pageFoot(c, &d.albumsPage, more)
			}
		},
		Empty:     a.empty,
		AlbumGrid: a.albumGrid,
		Events: func(c *ui.Context) {
			if d := a.artists[id]; d != nil {
				a.artistEvents(c, d)
			}
		},
		CoverFill: a.coverFill,
	}
}

func (a *app) artistPage(c *ui.Context) {
	pages.ArtistPage(c, a.artistDeps(), a.route.ID)
}

func (a *app) podcastDeps() pages.PodcastDeps {
	id := a.route.ID
	return pages.PodcastDeps{
		Palette: a.pal,
		Route:   &a.route,
		Player:  a.pl,
		Info: func() pages.PodcastInfo {
			d := a.podcasts[id]
			if d == nil {
				return pages.PodcastInfo{}
			}
			return pages.PodcastInfo{
				Podcast: d.podcast, Loading: d.loading, Err: d.err,
				Programs: d.programs, PageLoaded: d.page.loaded,
				PageLoading: d.page.loading, PageMore: d.page.more, PageErr: d.page.err,
			}
		},
		Load:         a.loadPodcast,
		ProgramsMore: a.loadPrograms,
		PlayProgram: func(i int) {
			if d := a.podcasts[id]; d != nil {
				a.playProgram(d, i)
			}
		},
		ToggleSubscribe: func() {
			if d := a.podcasts[id]; d != nil {
				a.toggleSubscribe(d)
			}
		},
		Centered: a.centered,
		Skeleton: func(c *ui.Context) {
			s := a.pageShimmer(c)
			a.trackSkeleton(c, s, true, func(c *ui.Context) { a.playlistHeaderSkeleton(c, s) })
		},
		ScrollPage: a.scrollPage,
		PageFoot: func(c *ui.Context, more func()) {
			if d := a.podcasts[id]; d != nil {
				a.pageFoot(c, &d.page, more)
			}
		},
		Empty:     a.empty,
		Cover:     a.cover,
		CoverFill: a.coverFill,
		Open:      func(r navigation.Route) { a.open(r) },
		Link:      a.link,
		Bars:      a.nowPlayingBars,
		Like:      func(c *ui.Context, tr *qqmusic.Track) { a.likeButton(c, tr, 15, a.pal.Muted) },
		Liked:     func(id int64) bool { return a.likedPrograms[id] },
	}
}

func (a *app) podcastsDeps() pages.PodcastsDeps {
	t := a.podcastTab
	return pages.PodcastsDeps{
		Palette: a.pal,
		Sidebar: sidebarWidth,
		Info: func() pages.PodcastsInfo {
			return pages.PodcastsInfo{
				Loaded: t.hotPage.loaded, Loading: t.loading, Err: t.err,
				Subs: t.subs, Picks: t.picks, Hot: t.hot,
				HotMore: t.hotPage.more, HotLoading: t.hotPage.loading, HotErr: t.hotPage.err,
			}
		},
		Scroll:    &a.pageState().scroll,
		Insets:    a.insets,
		StartMark: a.startMark,
		EndMark:   a.endMark,
		Centered:  a.centered,
		LoadTab:   a.loadPodcastTab,
		LoadMore:  a.loadHotPodcasts,
		Skeleton: func(c *ui.Context) {
			s := a.pageShimmer(c)
			cols := a.columns(c, 150)
			a.section(c, "精选播客").Margin(0, 0, 14, 0)
			a.playlistCardsSkeleton(c, s, cols, 150, true)
			a.section(c, "热门播客")
			a.playlistCardsSkeleton(c, s, cols*2, 150, true)
		},
		Card: a.podcastCard,
		PageFoot: func(c *ui.Context, more func()) {
			a.pageFoot(c, &t.hotPage, more)
		},
	}
}

func (a *app) podcastPage(c *ui.Context) {
	pages.PodcastPage(c, a.podcastDeps(), a.route.ID)
}

func (a *app) podcastsPage(c *ui.Context) {
	pages.PodcastsPage(c, a.podcastsDeps())
}

func (a *app) podcastRow(c *ui.Context, pod *qqmusic.Podcast) {
	pages.PodcastRow(c, a.podcastDeps(), pod)
}

func (a *app) programRow(c *ui.Context, d *podcastData, i int, cur *qqmusic.Track) {
	dep := a.podcastDeps()
	dep.PlayProgram = func(i int) { a.playProgram(d, i) }
	pages.ProgramRow(c, dep, pages.PodcastInfo{Podcast: d.podcast, Programs: d.programs}, i)
}

func (a *app) podcastCard(c *ui.Context, pod *qqmusic.Podcast) {
	pages.PodcastCard(c, a.pagesDeps(), pod, func() { a.playPodcast(pod.ID) })
}

func (a *app) shelfPager(shelf *qqmusic.SongShelf) pages.ShelfPager {
	if a.shelfPagers == nil {
		a.shelfPagers = map[*qqmusic.SongShelf]*songShelfPager{}
	}
	pager := a.shelfPagers[shelf]
	if pager == nil {
		pager = &songShelfPager{}
		a.shelfPagers[shelf] = pager
	}
	return pager
}

func (a *app) shelfDeps() pages.ShelfDeps {
	return pages.ShelfDeps{
		Palette: a.pal,
		Card:    a.pagesDeps(),
		Player:  a.pl,
		Info: func() pages.ShelfInfo {
			info := pages.ShelfInfo{SignedIn: a.profile != nil}
			if a.songShelves != nil {
				info.Shelves = a.songShelves.shelves
				info.Loading = a.songShelves.loading
				info.Err = a.songShelves.err
			}
			return info
		},
		Pager:     a.shelfPager,
		Columns:   a.songShelfColumns,
		CoverFill: a.coverFill,
		Like: func(c *ui.Context, tr *qqmusic.Track, size float32, color ui.Color) {
			a.likeButton(c, tr, size, color)
		},
		SongItems: a.songItems,
		Bars:      a.nowPlayingBars,
		Action: func(c *ui.Context, label, icon string) ui.Element {
			var svg *ui.SVG
			switch icon {
			case "user":
				svg = iconUser
			case "reload":
				svg = iconReload
			}
			return a.shelfAction(c, svg, label)
		},
		Login:    a.openLogin,
		Reload:   a.loadSongShelves,
		Skeleton: func(c *ui.Context) { a.songShelfSkeleton(c, a.pageShimmer(c)) },
		Tracks: func(c *ui.Context, tracks []qqmusic.Track, header func(*ui.Context)) {
			a.trackRows(c, tracks, true, header)
		},
		PlayShelf: a.playSongShelf,
		PlaySong:  a.playShelfSong,
		Centered:  a.centered,
	}
}

var shelfHasPlayable = pages.ShelfHasPlayable

func (a *app) songShelfSection(c *ui.Context, start, end int) {
	pages.SongShelfSection(c, a.shelfDeps(), start, end)
}

func (a *app) songShelfRow(c *ui.Context, shelf *qqmusic.SongShelf, i int) {
	pages.SongShelfRow(c, a.shelfDeps(), shelf, i)
}

func (a *app) songShelf(c *ui.Context, shelf *qqmusic.SongShelf) {
	pages.SongShelfCard(c, a.shelfDeps(), shelf)
}

func (a *app) songShelfPage(c *ui.Context) {
	pages.SongShelfPage(c, a.shelfDeps(), a.route.Shelf)
}

func (a *app) searchDeps() pages.SearchDeps {
	return pages.SearchDeps{
		Palette: a.pal,
		Route:   &a.route,
		Sidebar: sidebarWidth,
		Info: func() pages.SearchInfo {
			d := a.searches[a.route]
			if d == nil {
				return pages.SearchInfo{}
			}
			r := &d.res
			return pages.SearchInfo{
				Songs: r.Songs, Artists: r.Artists, Albums: r.Albums, Users: r.Users, Podcasts: r.Podcasts,
				Total: r.Total, Len: r.Len(), Loaded: d.page.loaded, Loading: d.page.loading,
				More: d.page.more, Err: d.page.err,
			}
		},
		More: func() {
			if d := a.searches[a.route]; d != nil {
				a.moreResults(a.route, d)
			}
		},
		Show: func(r navigation.Route) { a.show(r) },
		Tracks: func(c *ui.Context, songs []qqmusic.Track, header, foot func(*ui.Context)) {
			a.trackRows(c, songs, true, header, foot)
		},
		ScrollPage: a.scrollPage,
		Skeleton: func(c *ui.Context, header func(*ui.Context)) {
			a.trackSkeleton(c, a.pageShimmer(c), true, header)
		},
		CardsSkeleton: func(c *ui.Context) {
			a.playlistCardsSkeleton(c, a.pageShimmer(c), a.columns(c, 156), 156, false)
		},
		PageFoot: func(c *ui.Context, more func()) {
			if d := a.searches[a.route]; d != nil {
				a.pageFoot(c, &d.page, more)
			}
		},
		Empty:      a.empty,
		ArtistCard: a.artistCard,
		AlbumGrid:  a.albumGrid,
		UserRow:    a.userRow,
		PodcastRow: a.podcastRow,
	}
}

func (a *app) userDeps() pages.UserDeps {
	id := a.route.ID
	return pages.UserDeps{
		Palette: a.pal,
		Sidebar: sidebarWidth,
		Own:     a.profile != nil && a.profile.UserID == id,
		Info: func() pages.UserInfo {
			d := a.users[id]
			if d == nil {
				return pages.UserInfo{}
			}
			return pages.UserInfo{
				User: d.user, Playlists: d.playlists, Loading: d.loading, Err: d.err,
				Artists: d.artists, Podcasts: d.podcasts,
				ArtistsShown: d.artistsShown, PodcastsShown: d.podcastsShown,
			}
		},
		Load:     a.loadUser,
		Centered: a.centered,
		Skeleton: func(c *ui.Context) {
			s := a.pageShimmer(c)
			a.scrollPage(c, func() {
				a.playlistHeaderSkeleton(c, s)
				a.playlistCardsSkeleton(c, s, a.columns(c, 168), 168, false)
			})
		},
		ScrollPage: a.scrollPage,
		Badge: func(c *ui.Context) {
			if d := a.users[id]; d != nil {
				a.membershipBadge(c, id, d)
			}
		},
		Empty: a.empty,
		CardsSkeleton: func(c *ui.Context) {
			a.playlistCardsSkeleton(c, a.pageShimmer(c), a.columns(c, 168), 168, false)
		},
		PlaylistCard: a.playlistCard,
		ArtistCard:   a.artistCard,
		PodcastRow:   a.podcastRow,
		Cover:        a.cover,
		CoverFill:    a.coverFill,
		Open:         func(r navigation.Route) { a.open(r) },
	}
}

func (a *app) purchasedDeps() pages.PurchasedDeps {
	return pages.PurchasedDeps{
		Palette:   a.pal,
		Route:     &a.route,
		SignedIn:  a.profile != nil,
		OpenLogin: a.openLogin,
		Insets:    a.insets,
		Info: func(tab int) pages.PurchasedInfo {
			d := a.purchased[tab]
			if d == nil {
				return pages.PurchasedInfo{}
			}
			return pages.PurchasedInfo{
				Songs: d.songs, Albums: d.albums,
				Loaded: d.page.loaded, Loading: d.page.loading, More: d.page.more, Err: d.page.err,
			}
		},
		State: func(tab int) *pages.ListState { return &a.purchasedScroll[tab] },
		ListPage: func(c *ui.Context, key any, state *ui.ListState, body func()) {
			a.listPageWithKey(c, key, state, body)
		},
		Skeleton: func(c *ui.Context, header func(*ui.Context)) {
			a.trackSkeletonRows(c, a.pageShimmer(c), true, header)
		},
		CardsSkeleton: func(c *ui.Context) {
			a.playlistCardsSkeleton(c, a.pageShimmer(c), a.columns(c, 156), 156, false)
		},
		PageFoot: func(c *ui.Context, more func()) {
			if d := a.purchased[a.route.Tab]; d != nil {
				a.pageFoot(c, &d.page, more)
			}
		},
		Empty: a.empty,
		TrackRows: func(c *ui.Context, key any, state *pages.ListState, tracks []qqmusic.Track, covers bool, header, foot func(*ui.Context)) {
			a.trackRowsWithKey(c, key, state, tracks, covers, header, foot)
		},
		AlbumGrid: a.albumGrid,
		Play: func(songs []qqmusic.Track) {
			if a.pl.Shuffle() {
				a.pl.ToggleShuffle()
			}
			a.pl.PlayQueue(songs, firstPlayable(songs))
		},
		More: func(tab int) {
			if d := a.purchased[tab]; d != nil {
				a.morePurchased(tab, d)
			}
		},
		Reload: func(tab int) { a.loadPurchased(tab, true) },
		Show:   func(r navigation.Route) { a.show(r) },
	}
}

func (a *app) searchPage(c *ui.Context)               { pages.SearchPage(c, a.searchDeps()) }
func (a *app) userPage(c *ui.Context)                 { pages.UserPage(c, a.userDeps(), a.route.ID) }
func (a *app) userRow(c *ui.Context, u *qqmusic.User) { pages.UserRow(c, a.userDeps(), u) }
func (a *app) purchasedPage(c *ui.Context)            { pages.PurchasedPage(c, a.purchasedDeps()) }
func (a *app) artistCard(c *ui.Context, ar *qqmusic.ArtistProfile) {
	pages.ArtistCard(c, a.pagesDeps(), ar)
}

func (a *app) listDeps() pages.ListDeps {
	return pages.ListDeps{
		Player:      a.pl,
		Palette:     a.pal,
		Route:       &a.route,
		Liked:       a.liked,
		UnderTop:    &a.underTop,
		UnderBottom: &a.underBottom,
		Bars:        &a.rowBars,
		Sidebar:     sidebarWidth,
		Insets:      a.insets,
		StartMark:   a.startMark,
		EndMark:     a.endMark,
		SongItems:   a.songItems,
		Cover:       a.cover,
		ArtistLinks: a.artistLinks,
		Link:        a.link,
		Like:        a.likeButton,
		PlayQueue:   func(tracks []qqmusic.Track, at int) { a.pl.PlayQueue(tracks, at) },
	}
}

func (a *app) trackRows(c *ui.Context, tracks []qqmusic.Track, covers bool, header func(c *ui.Context), footer ...func(c *ui.Context)) {
	a.trackRowsWithState(c, &a.pageState().tracks, tracks, covers, header, footer...)
}

func (a *app) trackRowsWithState(c *ui.Context, state *listScrollState, tracks []qqmusic.Track, covers bool, header func(c *ui.Context), footer ...func(c *ui.Context)) {
	a.trackRowsWithKey(c, a.route, state, tracks, covers, header, footer...)
}

func (a *app) trackRowsWithKey(c *ui.Context, key any, state *listScrollState, tracks []qqmusic.Track, covers bool, header func(c *ui.Context), footer ...func(c *ui.Context)) {
	pages.TrackRows(c, a.listDeps(), key, state, tracks, covers, header, footer...)
}

func (a *app) nowPlayingBars(c *ui.Context, color ui.Color) {
	pages.NowPlayingBars(c, a.listDeps(), color)
}

func (a *app) albumDeps() pages.AlbumDeps {
	return pages.AlbumDeps{
		Palette: a.pal,
		Art:     a.art,
		Route:   &a.route,
		Info: func(id int64) pages.AlbumInfo {
			if d := a.albums[id]; d != nil {
				return pages.AlbumInfo{Profile: d.album, Loading: d.loading, Err: d.err}
			}
			return pages.AlbumInfo{}
		},
		Load:     a.loadAlbum,
		Centered: a.centered,
		Skeleton: func(c *ui.Context) {
			s := a.pageShimmer(c)
			a.trackSkeleton(c, s, false, func(c *ui.Context) { a.playlistHeaderSkeleton(c, s) })
		},
		Tracks: func(c *ui.Context, tracks []qqmusic.Track, header func(*ui.Context)) {
			a.trackRows(c, tracks, false, header)
		},
		Link:        a.link,
		MetaRow:     a.albumMetaRow,
		Description: a.collectionDescription,
		Play: func(songs []qqmusic.Track) {
			if a.pl.Shuffle() {
				a.pl.ToggleShuffle()
			}
			a.pl.PlayQueue(songs, firstPlayable(songs))
		},
		Like:    a.albumLikeButton,
		Open:    func(r navigation.Route) { a.open(r) },
		Columns: a.columns,
	}
}

func (a *app) eventDeps() pages.EventDeps {
	return pages.EventDeps{
		Palette:   a.pal,
		Cover:     a.cover,
		CoverFill: a.coverFill,
		Open:      func(r navigation.Route) { a.open(r) },
	}
}

func (a *app) event(c *ui.Context, e *qqmusic.Event) {
	pages.Event(c, a.eventDeps(), e)
}

func (a *app) albumPage(c *ui.Context) { pages.AlbumPage(c, a.albumDeps(), a.route.ID) }
func (a *app) albumGrid(c *ui.Context, albums []qqmusic.AlbumProfile) {
	pages.AlbumGrid(c, a.albumDeps(), albums)
}

func (a *app) homeDeps() pages.HomeDeps {
	return pages.HomeDeps{
		Palette:      a.pal,
		Card:         a.pagesDeps(),
		Sidebar:      sidebarWidth,
		Featured:     a.featured,
		Charts:       a.charts,
		RecPlaylists: a.recPlaylists,
		HomeLoading:  a.homeLoading,
		HomeErr:      a.homeErr,
		ProfileName: func() string {
			if a.profile != nil {
				return a.profile.Nickname
			}
			return ""
		},
		Shelves: func() (has, busy bool) {
			has = a.songShelves != nil && len(a.songShelves.shelves) > 0
			busy = a.songShelves != nil && a.songShelves.loading
			return has, busy
		},
		Scroll:          &a.pageState().scroll,
		Insets:          a.insets,
		StartMark:       a.startMark,
		EndMark:         a.endMark,
		Centered:        a.centered,
		LoadHome:        a.loadHome,
		LoadSongShelves: a.loadSongShelves,
		ShelfAction: func(c *ui.Context, label string, busy bool) ui.Element {
			return a.shelfAction(c.Key("refresh-songs"), iconReload, label).Label("更换歌曲推荐").Disabled(busy).
				Margin(0, 0, 18, 0).OnClick(a.loadSongShelves)
		},
		HomeSkeleton: func(c *ui.Context) { a.homeSkeleton(c, a.pageShimmer(c)) },
		DailyCard:    func(c *ui.Context, now time.Time) { a.dailyCard(c, now) },
		SongShelves: func(c *ui.Context, key string, from, to int) {
			a.songShelfSection(c.Key(key), from, to)
		},
		SquareEntry: a.squareEntry,
	}
}

func (a *app) chartsDeps() pages.ChartsDeps {
	return pages.ChartsDeps{
		Palette:     a.pal,
		Card:        a.pagesDeps(),
		Sidebar:     sidebarWidth,
		Charts:      a.charts,
		HomeLoading: a.homeLoading,
		HomeErr:     a.homeErr,
		Scroll:      &a.pageState().scroll,
		Insets:      a.insets,
		StartMark:   a.startMark,
		EndMark:     a.endMark,
		Centered:    a.centered,
		LoadHome:    a.loadHome,
		Skeleton: func(c *ui.Context) {
			s := a.pageShimmer(c)
			a.chartCardsSkeleton(c, s)
			a.section(c, "全部榜单")
			a.playlistCardsSkeleton(c, s, a.columns(c, 150)*2, 150, true)
		},
	}
}

func (a *app) playlistDeps() pages.PlaylistDeps {
	return pages.PlaylistDeps{
		Palette:   a.pal,
		Route:     &a.route,
		Player:    a.pl,
		Card:      a.pagesDeps(),
		Liked:     a.isLikedPlaylist,
		LikedPage: func(c *ui.Context, id int64) { a.likedPlaylistPage(c, id) },
		Info: func(id int64) pages.PlaylistInfo {
			if pl := a.playlists[id]; pl != nil {
				return pages.PlaylistInfo{Playlist: pl}
			}
			return pages.PlaylistInfo{Loading: a.loadingPl[id], Err: a.plErr[id]}
		},
		Load:     func(id int64) { a.loadPlaylist(id, false) },
		Centered: a.centered,
		Skeleton: func(c *ui.Context) {
			s := a.pageShimmer(c)
			a.trackSkeleton(c, s, true, func(c *ui.Context) { a.playlistHeaderSkeleton(c, s) })
		},
		Tracks: func(c *ui.Context, tracks []qqmusic.Track, header func(*ui.Context)) {
			a.trackRows(c, tracks, true, header)
		},
		Editable: func(id int64) bool { return a.editable(id) != nil },
		Rename: func(id int64) {
			if p := a.editable(id); p != nil {
				a.startRename(p)
			}
		},
		Delete: func(id int64) {
			if p := a.editable(id); p != nil {
				a.confirmDelete(p)
			}
		},
		Saved: func(id int64) bool { return a.savedPlaylist(id) != nil },
		Unfavorite: func(id int64) {
			if p := a.savedPlaylist(id); p != nil {
				a.confirmUnfavorite(p)
			}
		},
	}
}

func (a *app) homePage(c *ui.Context)   { pages.HomePage(c, a.homeDeps()) }
func (a *app) chartsPage(c *ui.Context) { pages.ChartsPage(c, a.chartsDeps()) }
func (a *app) playlistPage(c *ui.Context) {
	pages.PlaylistPage(c, a.playlistDeps(), a.route.ID)
}

func (a *app) pagesDeps() pages.Deps {
	return pages.Deps{
		Palette:     a.pal,
		Art:         a.art,
		Open:        func(r navigation.Route) { a.open(r) },
		Load:        func(id int64, play bool) { a.loadPlaylist(id, play) },
		Description: a.collectionDescription,
	}
}

func (a *app) columns(c *ui.Context, min float32) int {
	return pages.Columns(c, sidebarWidth, min)
}

func (a *app) section(c *ui.Context, title string) ui.Element {
	return pages.Section(c, title)
}

func (a *app) playlistCard(c *ui.Context, pl *qqmusic.Playlist) {
	pages.PlaylistCard(c, a.pagesDeps(), pl)
}

func (a *app) coverFill(c *ui.Context, url string, px int, radius float32) {
	pages.CoverFill(c, a.art, a.pal, url, px, radius)
}
