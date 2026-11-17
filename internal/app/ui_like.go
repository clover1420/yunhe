package app

import (
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
)

// likedAlbumsLimit caps what the liked albums tab asks for at once.
const likedAlbumsLimit = 30

// ask these read the albums the listener likes, and like or unlike one;
// tests replace them.
var (
	readLikedAlbums = qqmusic.LikedAlbums
	readAlbumLiked  = qqmusic.AlbumLiked
	likeAlbum       = qqmusic.LikeAlbum
)

type likedAlbumData struct {
	albums []qqmusic.AlbumProfile
	page   paged
}

// Each collection keeps its own place when switching tabs or returning from a detail page.
type likedScrollState struct {
	songs  listScrollState
	albums ui.ListState
}

func (a *app) isLikedPlaylist(id int64) bool {
	return id > 0 && a.likedPlaylist() == id
}

func (a *app) loadLikedAlbums(again bool) {
	if a.profile == nil || a.likedAlbums != nil && !again {
		return
	}
	d := &likedAlbumData{}
	a.likedAlbums = d
	a.moreLikedAlbums(d)
}

func (a *app) moreLikedAlbums(d *likedAlbumData) {
	if a.profile == nil || d.page.loading {
		return
	}
	uid, offset := a.profile.UserID, len(d.albums)
	d.page.loading, d.page.err = true, ""
	go func() {
		albums, more, err := readLikedAlbums(likedAlbumsLimit, offset)
		a.post(func() {
			if a.profile == nil || a.profile.UserID != uid || a.likedAlbums != d {
				return
			}
			d.page.loading = false
			if err != nil {
				d.page.err = "收藏专辑加载失败"
				return
			}
			d.albums = append(d.albums, albums...)
			d.page.loaded, d.page.more = true, more && len(albums) > 0
		})
	}()
}

func (a *app) loadAlbumLike(d *albumData) {
	if a.profile == nil || d.album == nil || d.album.MID == "" || d.likeUID == a.profile.UserID && (d.likeLoading || d.likeKnown) {
		return
	}
	uid, mid := a.profile.UserID, d.album.MID
	d.likeUID, d.likeKnown, d.likeLoading = uid, false, true
	go func() {
		liked, err := readAlbumLiked(mid)
		a.post(func() { a.albumLikeStatusAnswered(uid, d, liked, err) })
	}()
}

func (a *app) albumLikeStatusAnswered(uid int64, d *albumData, liked bool, err error) {
	if a.profile == nil || a.profile.UserID != uid || d.likeUID != uid {
		return
	}
	d.likeLoading = false
	if err == nil {
		d.liked, d.likeKnown = liked, true
	}
}

func (a *app) albumLikeButton(c *ui.Context, al *qqmusic.AlbumProfile) {
	d := a.albums[al.ID]
	if d == nil || al.MID == "" {
		return
	}
	if a.profile != nil && (d.likeUID != a.profile.UserID || !d.likeKnown) {
		label := "重试专辑收藏状态"
		if d.likeLoading {
			label = "正在读取专辑收藏状态"
		}
		iconButton(c.Key("like-album"), iconReload, 20, a.pal.Muted, label).Disabled(d.likeLoading).
			OnClick(func() { a.loadAlbumLike(d) })
		return
	}
	icon, label, color := iconHeart, "收藏专辑", a.pal.Muted
	if a.profile != nil && d.likeUID == a.profile.UserID && d.likeKnown && d.liked {
		icon, label, color = iconHeartFill, "取消收藏专辑", accent(a.pal.Dark, 9)
	}
	iconButton(c.Key("like-album"), icon, 20, color, label).Disabled(d.likeLoading).
		OnClick(func() { a.toggleAlbumLike(d) })
}

func (a *app) toggleAlbumLike(d *albumData) {
	if a.profile == nil {
		a.openLogin()
		return
	}
	if d.album == nil || d.album.MID == "" || d.likeLoading {
		return
	}
	if d.likeUID != a.profile.UserID || !d.likeKnown {
		a.loadAlbumLike(d)
		return
	}
	uid, mid := a.profile.UserID, d.album.MID
	liked := !(d.likeUID == uid && d.likeKnown && d.liked)
	d.likeUID, d.likeLoading = uid, true
	go func() {
		err := likeAlbum(mid, liked)
		a.post(func() { a.albumLikeAnswered(uid, d, liked, err) })
	}()
}

func (a *app) albumLikeAnswered(uid int64, d *albumData, liked bool, err error) {
	if a.profile == nil || a.profile.UserID != uid || d.likeUID != uid {
		return
	}
	d.likeLoading = false
	if err != nil {
		a.toast = "专辑收藏操作失败，请稍后再试"
		return
	}
	d.liked, d.likeKnown = liked, true
	// Re-read the collection next time; its ordering belongs to QQ Music.
	a.likedAlbums = nil
	if a.route.Page == pagePlaylist && a.route.Tab == 1 && a.isLikedPlaylist(a.route.ID) {
		a.loadLikedAlbums(false)
	}
}

var readLikedIDs = qqmusic.LikedIDs

// loadLiked loads the listener's liked songs.
func (a *app) loadLiked(uid int64) {
	go func() {
		ids, err := readLikedIDs(uid)
		a.post(func() {
			if err != nil || a.profile == nil || a.profile.UserID != uid {
				return
			}
			a.liked = make(map[int64]bool, len(ids))
			for _, id := range ids {
				a.liked[id] = true
			}
		})
	}()
}
