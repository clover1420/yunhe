package pages

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/navigation"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// SearchMine is the kind of search that looks through the songs of the
// listener's own playlists rather than asking QQ Music.
const SearchMine qqmusic.SearchType = -1

// SearchKinds are the kinds of search, as their tabs show them.
var SearchKinds = []struct {
	Kind        qqmusic.SearchType
	Label, Unit string
}{
	{qqmusic.SearchSongs, "单曲", "首单曲"},
	{qqmusic.SearchArtists, "歌手", "位歌手"},
	{qqmusic.SearchAlbums, "专辑", "张专辑"},
	{SearchMine, "我的歌单", "首歌曲"},
}

// SearchInfo is what a search found, as far as its pages have loaded.
type SearchInfo struct {
	Songs    []qqmusic.Track
	Artists  []qqmusic.ArtistProfile
	Albums   []qqmusic.AlbumProfile
	Users    []qqmusic.User
	Podcasts []qqmusic.Podcast
	Total    int
	Len      int
	Loaded   bool
	Loading  bool
	More     bool
	Err      string
}

// SearchDeps are what a search's page needs from the app it runs in.
type SearchDeps struct {
	Palette theme.Palette
	Route   *navigation.Route
	Sidebar float32

	Info func() SearchInfo
	More func()
	Show func(navigation.Route)

	Tracks        func(*ui.Context, []qqmusic.Track, func(*ui.Context), func(*ui.Context))
	ScrollPage    func(*ui.Context, func())
	Skeleton      func(*ui.Context, func(*ui.Context))
	CardsSkeleton func(*ui.Context)
	PageFoot      func(*ui.Context, func())
	Empty         func(*ui.Context, string)

	ArtistCard func(*ui.Context, *qqmusic.ArtistProfile)
	AlbumGrid  func(*ui.Context, []qqmusic.AlbumProfile)
	UserRow    func(*ui.Context, *qqmusic.User)
	PodcastRow func(*ui.Context, *qqmusic.Podcast)
}

// SearchPage shows what a search found, of the kind searched for.
func SearchPage(c *ui.Context, d SearchDeps) {
	info := d.Info()
	r := d.Route
	header := func(c *ui.Context) { searchHeader(c, d, info) }
	foot := func(c *ui.Context) {
		d.PageFoot(c, d.More)
		if info.Loaded && info.Len == 0 {
			d.Empty(c, "没有找到相关结果")
		}
	}
	if r.Kind == qqmusic.SearchSongs || r.Kind == SearchMine {
		if !info.Loaded && info.Loading {
			d.Skeleton(c, header)
			return
		}
		d.Tracks(c, info.Songs, header, foot)
		return
	}
	d.ScrollPage(c, func() {
		header(c)
		switch r.Kind {
		case qqmusic.SearchArtists:
			ui.Grid(c).Columns(Columns(c, d.Sidebar, 132)).GapX(20).GapY(28).Children(func() {
				for i := range info.Artists {
					d.ArtistCard(c.Key(info.Artists[i].ID), &info.Artists[i])
				}
			})
		case qqmusic.SearchAlbums:
			d.AlbumGrid(c, info.Albums)
		case qqmusic.SearchUsers:
			for i := range info.Users {
				d.UserRow(c.Key(info.Users[i].UserID), &info.Users[i])
			}
		case qqmusic.SearchPodcasts:
			for i := range info.Podcasts {
				d.PodcastRow(c.Key(info.Podcasts[i].ID), &info.Podcasts[i])
			}
		}
		if !info.Loaded && info.Loading {
			d.CardsSkeleton(c)
			return
		}
		d.PageFoot(c, d.More)
		if info.Loaded && info.Len == 0 {
			d.Empty(c, "没有找到相关结果")
		}
	})
}

// searchHeader is what was searched for, how many it found, and the tabs
// of what it may look for instead.
func searchHeader(c *ui.Context, d SearchDeps, info SearchInfo) {
	r := d.Route
	sub := "搜索"
	at := 0
	for i, k := range SearchKinds {
		if k.Kind == r.Kind {
			at = i
			if info.Loaded {
				sub = fmt.Sprintf("搜索 · 找到 %d %s", info.Total, k.Unit)
			}
		}
	}
	Heading(c, d.Palette, "“"+r.Query+"”", sub)
	ui.Row(c).Margin(0, 0, 18, 0).Children(func() {
		labels := make([]string, len(SearchKinds))
		for i, k := range SearchKinds {
			labels[i] = k.Label
		}
		if appui.Segmented(c.Key("kind"), &at, labels...) {
			next := *r
			next.Kind = SearchKinds[at].Kind
			d.Show(next)
		}
	})
}
