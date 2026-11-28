package app

import (
	"context"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/navigation"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/features/suggest"
)

// The search field and its suggestions live in lilt/internal/ui/features/suggest; the
// app keeps its query and these names while its views still live here.
type suggestState = suggest.State
type suggestion = suggest.Suggestion

var suggestionList = suggest.List

func (a *app) suggestDeps() suggest.Deps {
	return suggest.Deps{
		Palette:       a.pal,
		Query:         &a.query,
		Box:           &a.searchBox,
		History:       func() []string { return a.settings.SearchHistory },
		HistoryScroll: &a.historyScroll,
		State:         &a.suggestions,
		Post:          a.post,
		PaneSolid:     a.paneSolid,
		Fetch: func(ctx context.Context, q string) (qqmusic.Suggestions, error) {
			return searchSuggest(ctx, q)
		},
		Search:       a.search,
		SearchFor:    a.searchFor,
		Remember:     a.rememberSearch,
		Open:         func(r navigation.Route) { a.open(r) },
		ClearHistory: a.clearSearchHistory,
	}
}

func (a *app) suggest(q string) {
	suggest.Ask(a.suggestDeps(), q)
}

func (a *app) finishSuggestions(gen uint64, q string, res qqmusic.Suggestions, err error) {
	suggest.Finish(a.suggestDeps(), gen, q, res, err)
}

func (a *app) dismissSuggestions() {
	suggest.Dismiss(a.suggestDeps())
}

func (a *app) searchField(c *ui.Context) {
	suggest.Field(c, a.suggestDeps())
}
