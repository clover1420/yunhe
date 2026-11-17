// Package navigation names the pages the app shows and the route to one.
package navigation

import "lilt/internal/media/qqmusic"

// Page is a kind of page the app shows.
type Page uint8

const (
	Home Page = iota
	Daily
	Charts
	Playlist
	Search
	FM
	Artist
	Album
	User
	Podcast
	Podcasts
	Purchased
	Settings
	SongShelf
	Square
)

// Route is the page shown, and what it shows.
type Route struct {
	Page Page
	// ID is the playlist, artist, album, user or podcast the page shows.
	ID int64
	// Query and Kind are what a search page searched for.
	Query string
	Kind  qqmusic.SearchType
	// Tab selects singles or albums on the purchased music page.
	Tab int
	// Shelf is the recommendation detail's snapshot, kept even after a refresh.
	Shelf *qqmusic.SongShelf
}
