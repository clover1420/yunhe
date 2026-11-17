package app

import (
	"slices"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/features/playlistedit"
)

// Editing the listener's playlists lives in lilt/internal/ui/features/playlistedit; the
// app keeps its dialogs and these names while its views still live here.
type nameDialog = playlistedit.NameDialog
type deleteDialog = playlistedit.DeleteDialog

// These edit playlists with QQ Music; tests replace them.
var (
	createPlaylist     = qqmusic.CreatePlaylist
	renamePlaylist     = qqmusic.RenamePlaylist
	addToPlaylist      = qqmusic.AddToPlaylist
	removeFromPlaylist = qqmusic.RemoveFromPlaylist
	deletePlaylist     = qqmusic.DeletePlaylist
	unfavoritePlaylist = qqmusic.UnfavoritePlaylist
)

func (a *app) playlistTools() playlistedit.Deps {
	return playlistedit.Deps{
		Profile:   func() *qqmusic.Profile { return a.profile },
		Mine:      &a.mine,
		Saved:     &a.saved,
		Playlists: &a.playlists,
		Naming:    &a.naming,
		Deleting:  &a.deleting,
		Post:      a.post,
		Toast:     &a.toast,
		ShowMain:  a.showMain,
		OpenLogin: a.openLogin,
		OpenPlaylist: func(id int64) {
			a.open(route{Page: pagePlaylist, ID: id})
		},
		ReloadPlaylist: func(id int64) {
			if a.route.Page == pagePlaylist && a.route.ID == id {
				a.loadPlaylist(id, false)
			}
		},
		LeaveRemoved: func(id int64) {
			// Leave no way back to its page, and leave it for the page before.
			gone := func(r route) bool { return r.Page == pagePlaylist && r.ID == id }
			a.history = slices.DeleteFunc(a.history, gone)
			a.forward = slices.DeleteFunc(a.forward, gone)
			if gone(a.route) {
				r := route{Page: pageHome}
				if n := len(a.history); n > 0 {
					r, a.history = a.history[n-1], a.history[:n-1]
				}
				a.show(r)
			}
		},
		Create:     func(name string) (*qqmusic.Playlist, error) { return createPlaylist(name) },
		Rename:     func(id int64, name string) error { return renamePlaylist(id, name) },
		Add:        func(pid, id int64) error { return addToPlaylist(pid, id) },
		Remove:     func(pid, id int64) error { return removeFromPlaylist(pid, id) },
		Delete:     func(id int64) error { return deletePlaylist(id) },
		Unfavorite: func(id int64) error { return unfavoritePlaylist(id) },

		Cache:            func() *player.Cache { return a.pl.Cache },
		Invalidate:       a.invalidatePlaylist,
		MarkSynced:       a.markPlaylistSynced,
		PutPlaylist:      a.putPlaylist,
		SavePlaylist:     a.savePlaylistCache,
		SaveUserPlaylists: a.saveUserPlaylistsCache,
	}
}

func (a *app) editable(id int64) *qqmusic.Playlist {
	return playlistedit.Editable(a.playlistTools(), id)
}

func (a *app) savedPlaylist(id int64) *qqmusic.Playlist {
	return playlistedit.Saved(a.playlistTools(), id)
}

func (a *app) loadMine(uid int64) {
	playlistedit.LoadMine(a.playlistTools(), uid)
}

func (a *app) newPlaylist(tr *qqmusic.Track) {
	playlistedit.New(a.playlistTools(), tr)
}

func (a *app) startRename(pl *qqmusic.Playlist) {
	playlistedit.StartRename(a.playlistTools(), pl)
}

func (a *app) submitName() {
	playlistedit.SubmitName(a.playlistTools())
}

func (a *app) confirmDelete(pl *qqmusic.Playlist) {
	playlistedit.ConfirmDelete(a.playlistTools(), pl)
}

func (a *app) confirmUnfavorite(pl *qqmusic.Playlist) {
	playlistedit.ConfirmUnfavorite(a.playlistTools(), pl)
}

func (a *app) deleteChosen() {
	playlistedit.DeleteChosen(a.playlistTools())
}

func (a *app) finishPlaylistRemoval(pl qqmusic.Playlist, saved bool, index int, err error) {
	playlistedit.FinishRemoval(a.playlistTools(), pl, saved, index, err)
}

func (a *app) playlistItems(m *ui.Menu, tr *qqmusic.Track, pid int64) {
	playlistedit.Items(a.playlistTools(), m, tr, pid)
}
