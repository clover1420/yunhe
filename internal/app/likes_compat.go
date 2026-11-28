package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/features/likes"
)

// The hearts and the song menu live in lilt/internal/ui/features/likes; the app keeps
// the listener's like maps and these names while its views still live here.

// These like or unlike a song or an episode with NetEase, and ask which
// episodes the listener likes; tests replace them.
var (
	likeSong      = qqmusic.Like
	likeProgram   = qqmusic.LikeProgram
	programsLiked = qqmusic.ProgramsLiked
)

// copyText puts text on the clipboard; tests replace it.
var copyText = func(text string) error { return mygo.Clipboard.Write(transfer.TextData(text)) }

// likesDeps gathers what the hearts and the song menu need, once a frame.
func (a *app) likesDeps() likes.Deps {
	return likes.Deps{
		Player:         a.pl,
		Palette:        a.pal,
		Profile:        func() *qqmusic.Profile { return a.profile },
		Liked:          &a.liked,
		Programs:       &a.likedPrograms,
		Asked:          &a.askedPrograms,
		LikeGen:        &a.likeGen,
		ProgramLikeGen: &a.programLikeGen,
		Post:           a.post,
		Toast:          &a.toast,
		LikeSong: func(id int64, like bool) error {
			return likeSong(id, like)
		},
		LikeProgram: func(id int64, like bool) error {
			return likeProgram(id, like)
		},
		ProgramsLiked: func(ids []int64) (map[int64]bool, error) {
			return programsLiked(ids)
		},
		OpenLogin: a.openLogin,
		ShowMain:  a.showMain,
		AfterSongLike: func() {
			// The liked songs' playlist changed: load it anew when shown.
			for _, pl := range a.mine {
				if pl.SpecialType == 5 {
					a.invalidatePlaylist(pl.ID)
					if a.route.Page == pagePlaylist && a.route.ID == pl.ID {
						a.loadPlaylist(pl.ID, false)
					}
				}
			}
		},
		Copy:          func(text string) error { return copyText(text) },
		OpenArtist:    a.openArtist,
		OpenAlbum:     func(id int64) { a.openPage(route{Page: pageAlbum, ID: id}) },
		OpenPodcast:   func(id int64) { a.openPage(route{Page: pagePodcast, ID: id}) },
		PlaylistItems: func(m *ui.Menu, tr *qqmusic.Track, pid int64) { a.playlistItems(m, tr, pid) },
	}
}

func (a *app) toggleLike(tr *qqmusic.Track) {
	likes.Toggle(a.likesDeps(), tr)
}

func (a *app) likeCurrent() {
	likes.LikeCurrent(a.likesDeps())
}

func (a *app) askProgramLikes(ids []int64) {
	likes.AskPrograms(a.likesDeps(), ids)
}

func (a *app) programLikesAnswered(uid int64, asked []int64, liked map[int64]bool) {
	likes.ProgramsAnswered(a.likesDeps(), uid, asked, liked)
}

func (a *app) setProgramLiked(id int64, like bool) {
	likes.SetProgramLiked(a.likesDeps(), id, like)
}

func (a *app) likeButton(c *ui.Context, tr *qqmusic.Track, size float32, color ui.Color, surface ...buttonSurface) ui.Element {
	return likes.Button(c, a.likesDeps(), tr, size, color, surface...)
}

func (a *app) songMenu(c *ui.Context, tr *qqmusic.Track, size float32, color ui.Color, surface ...buttonSurface) ui.Element {
	return likes.SongMenu(c, a.likesDeps(), tr, size, color, surface...)
}

func (a *app) songItems(tr *qqmusic.Track, pid int64) func(m *ui.Menu) {
	return likes.SongItems(a.likesDeps(), tr, pid)
}

// queueSongItems extends songItems with queue removal for the playback queue list.
func (a *app) queueSongItems(tr *qqmusic.Track, queueIndex int) func(m *ui.Menu) {
	base := a.songItems(tr, 0)
	return func(m *ui.Menu) {
		base(m)
		m.Separator()
		if m.Item("移除歌曲").Chosen() {
			a.pl.Remove(queueIndex)
			if a.pl.Current() == nil {
				a.nowPlaying = false
				a.queueOpen = false
			}
		}
	}
}
