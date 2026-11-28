// Package suggest owns the search field and what a query may be after as it
// is typed: the suggestions, and the searches before.
package suggest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/navigation"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// delay is how long typing pauses before suggestions are asked for.
const delay = 220 * time.Millisecond

// State is what a search may be after, as its query is typed.
type State struct {
	Query   string // what they are for
	Items   []Suggestion
	Loading bool
	Failed  bool
	Timer   *time.Timer // waiting for typing to pause
	Cancel  context.CancelFunc
	Gen     uint64 // survives dismissal so an old reply cannot match a later identical query
}

// Suggestion is a page a query may be after: a song's search, or an artist's,
// album's or playlist's page.
type Suggestion struct {
	Key          string // its option in the popup
	Name, Detail string
	Kind         string
	Route        navigation.Route
}

// Deps are what the search field needs from the app it runs in: the query,
// its focus and history by pointer, and the pages that open elsewhere by
// callback.
type Deps struct {
	Palette theme.Palette

	Query         *string
	Box           *ui.Handle
	History       func() []string
	HistoryScroll *ui.ScrollState
	State         *State

	Post func(func())
	// PaneSolid styles the field's popup, which floats over the sidebar:
	// the glass of the app's other panes cannot sample the system's
	// material there, and would tint the popup gray.
	PaneSolid func(e ui.Element, radius float32, onDark bool) ui.Element

	Fetch func(ctx context.Context, q string) (qqmusic.Suggestions, error)

	Search       func(q string)
	SearchFor    func(r navigation.Route)
	Remember     func(q string)
	Open         func(r navigation.Route)
	ClearHistory func()
}

// Ask asks for suggestions for q, once typing pauses.
func Ask(d Deps, q string) {
	s := d.State
	if q == s.Query {
		return
	}
	Dismiss(d)
	s.Query, s.Loading = q, true
	ctx, cancel := context.WithCancel(context.Background())
	s.Cancel = cancel
	gen, fetch := s.Gen, d.Fetch
	s.Timer = time.AfterFunc(delay, func() {
		res, err := fetch(ctx, q)
		d.Post(func() {
			Finish(d, gen, q, res, err)
		})
	})
}

// Finish applies a fetch result on the main thread.
func Finish(d Deps, gen uint64, q string, res qqmusic.Suggestions, err error) {
	s := d.State
	if gen != s.Gen || q != s.Query {
		return
	}
	if s.Cancel != nil {
		s.Cancel()
	}
	s.Timer, s.Cancel, s.Loading, s.Failed = nil, nil, false, err != nil
	s.Items = List(res)
}

// Dismiss stops what was asked for, and forgets it.
func Dismiss(d Deps) {
	s := d.State
	if s.Timer != nil {
		s.Timer.Stop()
	}
	if s.Cancel != nil {
		s.Cancel()
	}
	*s = State{Gen: s.Gen + 1}
}

// List is the first ten suggestions, the kinds in the order QQ Music ranks
// them.
func List(res qqmusic.Suggestions) []Suggestion {
	byKind := map[string][]Suggestion{}
	for _, t := range res.Songs {
		key := Key("song", t.ID, t.MID)
		if key == "" {
			continue
		}
		byKind["songs"] = append(byKind["songs"], Suggestion{key, t.Name, t.ArtistNames(), "单曲",
			navigation.Route{Page: navigation.Search, Query: t.Name, Kind: qqmusic.SearchSongs}})
	}
	for _, ar := range res.Artists {
		if ar.ID <= 0 {
			continue
		}
		byKind["artists"] = append(byKind["artists"], Suggestion{Key("artist", ar.ID, ar.MID), ar.Name, "", "歌手",
			navigation.Route{Page: navigation.Artist, ID: ar.ID}})
	}
	for _, al := range res.Albums {
		if al.ID <= 0 {
			continue
		}
		byKind["albums"] = append(byKind["albums"], Suggestion{Key("album", al.ID, al.MID), al.Name, al.ArtistNames(), "专辑",
			navigation.Route{Page: navigation.Album, ID: al.ID}})
	}
	for _, pl := range res.Playlists {
		byKind["playlists"] = append(byKind["playlists"], Suggestion{fmt.Sprint("playlist:", pl.ID), pl.Name, pl.Creator.Nickname, "歌单",
			navigation.Route{Page: navigation.Playlist, ID: pl.ID}})
	}
	var out []Suggestion
	for _, kind := range append(res.Order, "songs", "artists", "albums", "playlists") {
		out = append(out, byKind[kind]...)
		delete(byKind, kind) // listed once, if also in the order
	}
	return out[:min(len(out), 10)]
}

// Key is the identity of a suggestion of kind.
func Key(kind string, id int64, mid string) string {
	if mid != "" {
		return kind + ":mid:" + mid
	}
	if id > 0 {
		return fmt.Sprint(kind, ":id:", id)
	}
	return ""
}

// Field is the search field atop the sidebar, which suggests what its query
// may be after as it is typed, and, empty, lists the searches before.
func Field(c *ui.Context, d Deps) {
	p := d.Palette
	field := ui.Row(c).Height(34).PaddingX(12).Gap(7).Radius(17).AlignItems(ui.Center).Margin(0, 0, 14, 0)
	if c.Focused(*d.Box) {
		field.Background(p.Bg).Shadow(0, 0, 0, 3, p.Accent.Alpha(0.25)).Border(1, p.Accent.Alpha(0.6))
	} else {
		field.Background(p.Hover).Border(1, p.Hairline)
	}
	field.Children(func() {
		ui.Icon(c, appui.IconSearch).Size(14, 14).TextColor(p.Muted)
		cb := ui.ComboboxBase(c.Key("search"), d.Query)
		in := cb.Input.Bind(d.Box).Grow(1).MinWidth(0).
			Placeholder("搜索歌曲、歌手、专辑").FontSize(13).Label("搜索")
		q := strings.TrimSpace(*d.Query)
		s := d.State
		switch {
		case in.Focused() && cb.Open() && q != "":
			Ask(d, q)
		case s.Query != "":
			Dismiss(d)
		}
		// Empty, it lists the searches before, if any.
		if q == "" && len(d.History()) == 0 {
			cb.SetOpen(false)
		}
		cb.Popup(func(panel ui.Element) {
			panel.AttachTo(field, ui.AnchorBottomStart, ui.AnchorTopStart).
				Width(field.Bounds().W).Padding(6).Margin(8, 0, 0, 0)
			d.PaneSolid(panel, 14, false)
			option := func(key, label string) ui.Element {
				row := cb.Item(key).MinHeight(36).Padding(6, 10).Gap(10).Radius(8).AlignItems(ui.Center).Label(label)
				if row.Highlighted() {
					row.Background(p.Hover)
				}
				return row
			}
			if q == "" {
				panel.Label("搜索历史")
				ui.Row(c).Padding(2, 4, 2, 10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "搜索历史").FontSize(12).TextColor(p.Muted).Grow(1)
					appui.IconButton(c.Key("clear"), appui.IconTrash, 14, p.Muted, "清空搜索历史").KeepFocus().OnClick(d.ClearHistory)
				})
				_, windowHeight := c.Size()
				const popupChromeHeight = 80 // header, padding, and space around the popup
				const historyMaxHeight = 8 * 36
				height := min(float32(len(d.History()))*36, historyMaxHeight, max(float32(36), windowHeight-field.Bounds().Y-field.Bounds().H-popupChromeHeight))
				scroll := ui.Scroll(c.Key("search-history")).Height(height).Padding(0, 12, 0, 0).TrackScroll(d.HistoryScroll)
				if !scroll.Hovered() {
					scroll.DrawOver(func(painter *ui.Painter, r ui.Rect) {
						painter.Scrollbar(ui.Scrollbar{Box: r, View: r.H, Content: r.H + d.HistoryScroll.MaxY, Offset: d.HistoryScroll.Y}, ui.ScrollbarState{})
					})
				}
				scroll.Children(func() {
					for _, h := range d.History() {
						option("history:"+h, h).Children(func() {
							ui.Icon(c, appui.IconSearch).Size(13, 13).TextColor(p.Muted)
							ui.Text(c, h).FontSize(13).SingleLine().Shrink(1).MinWidth(0)
						})
					}
				})
				return
			}
			panel.Label("搜索建议")
			// The query first, which Enter searches as typed.
			option("query", "搜索“"+q+"”").Children(func() {
				ui.Icon(c, appui.IconSearch).Size(13, 13).TextColor(p.Muted)
				ui.Text(c, "搜索“"+q+"”").FontSize(13).FontWeight(500).SingleLine().Shrink(1).MinWidth(0)
			})
			if s.Query == q {
				for _, sg := range s.Items {
					option(sg.Key, sg.Kind+"："+sg.Name).Children(func() {
						ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(c, sg.Name).FontSize(13).SingleLine()
							if sg.Detail != "" {
								ui.Text(c, sg.Detail).FontSize(11.5).TextColor(p.Muted).SingleLine()
							}
						})
						ui.Text(c, sg.Kind).FontSize(11.5).TextColor(p.Faint)
					})
				}
			}
		})
		if key, ok := cb.Chosen(); ok {
			cb.SetOpen(false)
			Choose(d, key, q)
		} else if in.Submitted() && q != "" {
			cb.SetOpen(false)
			d.Search(q)
		}
	})
}

// Choose goes where the option chosen in the search field's popup goes.
func Choose(d Deps, key, q string) {
	if h, ok := strings.CutPrefix(key, "history:"); ok {
		*d.Query = h
		d.Search(h)
		return
	}
	if key == "query" {
		d.Search(q)
		return
	}
	for _, sg := range d.State.Items {
		if sg.Key != key {
			continue
		}
		if sg.Route.Page == navigation.Search {
			*d.Query = sg.Route.Query
			d.SearchFor(sg.Route)
			return
		}
		d.Remember(q)
		Dismiss(d)
		d.Open(sg.Route)
		return
	}
}
