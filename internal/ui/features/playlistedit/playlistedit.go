// Package playlistedit owns the listener's own playlists: making them,
// renaming them, deleting them, and adding songs to them or taking songs out.
package playlistedit

import (
	"errors"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
)

// Deps are what editing playlists needs from the app it runs in: the
// listener's playlists and dialogs by pointer, and the views and routes that
// live elsewhere by callback.
type Deps struct {
	Profile   func() *qqmusic.Profile
	Mine      *[]qqmusic.Playlist
	Saved     *[]qqmusic.Playlist
	Playlists *map[int64]*qqmusic.Playlist

	Naming   *NameDialog
	Deleting *DeleteDialog

	Post  func(func())
	Toast *string

	ShowMain  func()
	OpenLogin func()

	OpenPlaylist   func(id int64)
	ReloadPlaylist func(id int64)
	LeaveRemoved   func(id int64)

	Create     func(name string) (*qqmusic.Playlist, error)
	Rename     func(id int64, name string) error
	Add        func(pid, id int64) error
	Remove     func(pid, id int64) error
	Delete     func(id int64) error
	Unfavorite func(id int64) error

	Cache             func() *player.Cache
	Invalidate        func(id int64)
	MarkSynced        func(id int64)
	PutPlaylist       func(*qqmusic.Playlist)
	SavePlaylist      func(*qqmusic.Playlist)
	SaveUserPlaylists func()
}

// NameDialog asks for a playlist's name: a new one's, or one's new name.
type NameDialog struct {
	Open   bool
	Name   string
	Rename int64          // the playlist renamed, or 0 to make one
	Add    *qqmusic.Track // the song to add to the playlist made, if any
	Busy   bool           // NetEase is making the playlist
}

// DeleteDialog asks whether to delete an owned playlist or unfavorite a saved one.
type DeleteDialog struct {
	Open  bool
	ID    int64
	Name  string
	Saved bool
}

// Editable returns the listener's playlist id if they may edit its songs and
// name, or nil: the liked songs' are their hearts'.
func Editable(d Deps, id int64) *qqmusic.Playlist {
	for i := range *d.Mine {
		if (*d.Mine)[i].ID == id && (*d.Mine)[i].SpecialType != 5 {
			return &(*d.Mine)[i]
		}
	}
	return nil
}

// Saved returns the listener's saved playlist id, or nil.
func Saved(d Deps, id int64) *qqmusic.Playlist {
	for i := range *d.Saved {
		if (*d.Saved)[i].ID == id {
			return &(*d.Saved)[i]
		}
	}
	return nil
}

// LoadMine loads the playlists the listener uid made and saved.
func LoadMine(d Deps, uid int64) {
	var cache *player.Cache
	if d.Cache != nil {
		cache = d.Cache()
	}
	go func() {
		if cache != nil {
			if made, saved, ok := cache.LookupUserPlaylists(uid); ok {
				for _, p := range made {
					qqmusic.RememberOwnedPlaylist(p.ID)
				}
				d.Post(func() {
					if d.Profile() != nil && d.Profile().UserID == uid && *d.Mine == nil && *d.Saved == nil {
						*d.Mine, *d.Saved = made, saved
					}
				})
			}
		}
		made, saved, err := qqmusic.UserPlaylists(uid)
		if err == nil && cache != nil {
			cache.KeepUserPlaylists(uid, made, saved)
		}
		d.Post(func() {
			if err == nil && d.Profile() != nil && d.Profile().UserID == uid {
				*d.Mine, *d.Saved = made, saved
			}
		})
	}()
}

// Changed forgets what was loaded of playlist id, which NetEase changed:
// loaded anew if shown, and the sidebar's covers with it.
func Changed(d Deps, id int64) {
	d.Invalidate(id)
	d.ReloadPlaylist(id)
	if p := d.Profile(); p != nil {
		LoadMine(d, p.UserID)
	}
}

// New asks for the name of a playlist to make, then adds tr to it if set.
// Signed out, it asks the listener to sign in.
func New(d Deps, tr *qqmusic.Track) {
	if d.Profile() == nil {
		d.ShowMain()
		d.OpenLogin()
		return
	}
	*d.Naming = NameDialog{Open: true, Add: tr}
}

// StartRename asks for a new name for the listener's playlist pl.
func StartRename(d Deps, pl *qqmusic.Playlist) {
	*d.Naming = NameDialog{Open: true, Name: pl.Name, Rename: pl.ID}
}

// SubmitName makes the playlist named, or renames one: at once in the app,
// as NetEase may refuse, when renaming.
func SubmitName(d Deps) {
	n := d.Naming
	name := strings.TrimSpace(n.Name)
	if name == "" || n.Busy {
		return
	}
	if n.Rename != 0 {
		id := n.Rename
		n.Open = false
		pl := Editable(d, id)
		if pl == nil || pl.Name == name {
			return
		}
		old := pl.Name
		setName(d, id, name)
		go func() {
			err := d.Rename(id, name)
			d.Post(func() {
				if err != nil {
					setName(d, id, old)
					*d.Toast = "重命名失败，请稍后再试"
					return
				}
				d.SaveUserPlaylists()
			})
		}()
		return
	}
	n.Busy = true
	add := n.Add
	go func() {
		pl, err := d.Create(name)
		d.Post(func() {
			d.Naming.Busy = false
			if err != nil {
				*d.Toast = "创建歌单失败，请稍后再试"
				return
			}
			d.Naming.Open = false
			// Under the liked songs, as NetEase lists the newest.
			at := 0
			if len(*d.Mine) > 0 && (*d.Mine)[0].SpecialType == 5 {
				at = 1
			}
			*d.Mine = slices.Insert(*d.Mine, at, *pl)
			pl.Tracks = nil
			d.MarkSynced(pl.ID)
			d.PutPlaylist(pl)
			d.SavePlaylist(pl)
			d.SaveUserPlaylists()
			d.OpenPlaylist(pl.ID)
			if add != nil {
				AddTo(d, pl.ID, add)
			}
		})
	}()
}

// ConfirmDelete asks whether to delete the listener's playlist pl.
func ConfirmDelete(d Deps, pl *qqmusic.Playlist) {
	*d.Deleting = DeleteDialog{Open: true, ID: pl.ID, Name: pl.Name}
}

// ConfirmUnfavorite asks whether to unfavorite a saved playlist pl.
func ConfirmUnfavorite(d Deps, pl *qqmusic.Playlist) {
	*d.Deleting = DeleteDialog{Open: true, ID: pl.ID, Name: pl.Name, Saved: true}
}

// DeleteChosen removes the confirmed playlist from the library and leaves
// its page; QQ Music may refuse, in which case the sidebar entry comes back.
func DeleteChosen(d Deps) {
	id := d.Deleting.ID
	saved := d.Deleting.Saved
	d.Deleting.Open = false
	lists, remove := d.Mine, d.Delete
	if saved {
		lists, remove = d.Saved, d.Unfavorite
	}
	i := slices.IndexFunc(*lists, func(pl qqmusic.Playlist) bool { return pl.ID == id })
	if i < 0 || !saved && (*lists)[i].SpecialType == 5 {
		return
	}
	pl := (*lists)[i]
	*lists = slices.Delete(slices.Clone(*lists), i, i+1)
	d.LeaveRemoved(id)
	go func() {
		err := remove(id)
		d.Post(func() { FinishRemoval(d, pl, saved, i, err) })
	}()
}

// FinishRemoval commits the server result or restores the removed library
// entry.
func FinishRemoval(d Deps, pl qqmusic.Playlist, saved bool, index int, err error) {
	if err != nil {
		lists := d.Mine
		*d.Toast = "删除失败，请稍后再试"
		if saved {
			lists = d.Saved
			*d.Toast = "取消收藏失败，请稍后再试"
		}
		*lists = slices.Insert(*lists, min(index, len(*lists)), pl)
		return
	}
	d.Invalidate(pl.ID)
	d.SaveUserPlaylists()
	*d.Toast = "已删除歌单"
	if saved {
		*d.Toast = "已取消收藏"
	}
}

func setName(d Deps, id int64, name string) {
	if pl := Editable(d, id); pl != nil {
		pl.Name = name
	}
	if pl := (*d.Playlists)[id]; pl != nil {
		pl.Name = name
		d.SavePlaylist(pl)
	}
}

// AddTo adds tr to the listener's playlist pid; a toast says how it went.
func AddTo(d Deps, pid int64, tr *qqmusic.Track) {
	id := tr.ID
	go func() {
		err := d.Add(pid, id)
		d.Post(func() {
			switch {
			case errors.Is(err, qqmusic.ErrInPlaylist):
				*d.Toast = "歌曲已在歌单中"
			case err != nil:
				*d.Toast = "添加失败，请稍后再试"
			default:
				*d.Toast = "已添加到歌单"
				Changed(d, pid)
			}
		})
	}()
}

// RemoveFrom takes tr out of the listener's playlist pid, at once in the
// app and then with NetEase, which may refuse: the song comes back then.
func RemoveFrom(d Deps, pid int64, tr *qqmusic.Track) {
	id := tr.ID
	pl := (*d.Playlists)[pid]
	var before []qqmusic.Track
	if pl != nil {
		// A new list: the player may be playing the old one.
		before = pl.Tracks
		pl.Tracks = slices.DeleteFunc(slices.Clone(before), func(t qqmusic.Track) bool { return t.ID == id })
	}
	go func() {
		err := d.Remove(pid, id)
		d.Post(func() {
			if err != nil {
				if pl != nil && (*d.Playlists)[pid] == pl {
					pl.Tracks = before
				}
				*d.Toast = "移除失败，请稍后再试"
				return
			}
			// The songs listed are right: only the sidebar's covers may
			// have changed.
			if pl != nil && (*d.Playlists)[pid] == pl {
				d.SavePlaylist(pl)
			}
			if p := d.Profile(); p != nil {
				LoadMine(d, p.UserID)
			}
		})
	}()
}

// Items are the items of a song's menu that put it in the listener's
// playlists, or take it out of pid, the one it is listed in, if set.
func Items(d Deps, m *ui.Menu, tr *qqmusic.Track, pid int64) {
	if d.Profile() == nil {
		return
	}
	m.Separator()
	m.Submenu("添加到歌单", func(m *ui.Menu) {
		if m.Item("新建歌单…").Chosen() {
			New(d, tr)
		}
		var to []*qqmusic.Playlist
		for i := range *d.Mine {
			if pl := &(*d.Mine)[i]; pl.SpecialType != 5 && pl.ID != pid {
				to = append(to, pl)
			}
		}
		if len(to) > 0 {
			m.Separator()
		}
		for _, pl := range to {
			if m.Item(pl.Name).Chosen() {
				AddTo(d, pl.ID, tr)
			}
		}
	})
	if Editable(d, pid) != nil && m.Item("从歌单中移除").Chosen() {
		RemoveFrom(d, pid, tr)
	}
}
