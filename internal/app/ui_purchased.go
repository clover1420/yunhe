package app

import (
	"slices"
	"time"

	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
)

const (
	purchasedAlbums = iota
	purchasedSongs
	purchasedLimit      = 30
	purchasedRefreshTTL = 5 * time.Minute
)

// These read the current account's purchases; tests replace them.
var (
	readPurchasedSongs  = qqmusic.PurchasedSongs
	readPurchasedAlbums = qqmusic.PurchasedAlbums
)

type purchasedData struct {
	songs      []qqmusic.Track
	albums     []qqmusic.AlbumProfile
	page       paged
	syncedAt   time.Time
	stale      bool
	refreshing bool
}

func (a *app) loadPurchased(tab int, again bool) {
	if a.profile == nil {
		return
	}
	if d := a.purchased[tab]; d != nil && !again {
		if d.stale && !d.refreshing {
			a.refreshPurchased(tab, d)
		}
		return
	}
	d := &purchasedData{}
	a.purchased[tab] = d
	if !again && a.pl != nil && a.pl.Cache != nil {
		a.loadPurchasedWithCache(tab, d)
		return
	}
	a.morePurchased(tab, d)
}

func (a *app) loadPurchasedWithCache(tab int, d *purchasedData) {
	uid, cache, epoch := a.profile.UserID, a.pl.Cache, a.purchasedEpoch
	d.page.loading = true
	go func() {
		if tab == purchasedSongs {
			if songs, more, cachedAt, ok := cache.LookupPurchasedSongs(uid); ok {
				qqmusic.RememberTracks(songs)
				a.post(func() {
					if a.profile == nil || a.profile.UserID != uid || a.purchasedEpoch != epoch || a.purchased[tab] != d {
						return
					}
					d.page.loading = false
					d.songs = songs
					d.page.loaded, d.page.more, d.page.err = true, more, ""
					d.syncedAt = cachedAt
					if time.Since(cachedAt) >= purchasedRefreshTTL {
						d.stale = true
						a.refreshPurchased(tab, d)
					}
				})
				return
			}
		} else {
			if albums, more, cachedAt, ok := cache.LookupPurchasedAlbums(uid); ok {
				qqmusic.RememberAlbums(albums)
				a.post(func() {
					if a.profile == nil || a.profile.UserID != uid || a.purchasedEpoch != epoch || a.purchased[tab] != d {
						return
					}
					d.page.loading = false
					d.albums = albums
					d.page.loaded, d.page.more, d.page.err = true, more, ""
					d.syncedAt = cachedAt
					if time.Since(cachedAt) >= purchasedRefreshTTL {
						d.stale = true
						a.refreshPurchased(tab, d)
					}
				})
				return
			}
		}
		var songs []qqmusic.Track
		var albums []qqmusic.AlbumProfile
		var more bool
		var err error
		var encoded []byte
		if tab == purchasedSongs {
			songs, more, err = readPurchasedSongs(purchasedLimit, 0)
			if err == nil {
				encoded, _ = player.MarshalPurchasedSongs(songs, more && len(songs) > 0)
			}
		} else {
			albums, more, err = readPurchasedAlbums(purchasedLimit, 0)
			if err == nil {
				encoded, _ = player.MarshalPurchasedAlbums(albums, more && len(albums) > 0)
			}
		}
		a.post(func() {
			if a.profile == nil || a.profile.UserID != uid || a.purchasedEpoch != epoch || a.purchased[tab] != d {
				return
			}
			d.page.loading = false
			if err != nil {
				d.page.err = "已购音乐加载失败"
				return
			}
			d.songs = songs
			d.albums = albums
			d.page.loaded, d.page.more = true, more && len(songs)+len(albums) > 0
			d.syncedAt = time.Now()
			d.stale = false
			if len(encoded) > 0 {
				if tab == purchasedSongs {
					go cache.KeepPurchasedSongsData(uid, encoded)
				} else {
					go cache.KeepPurchasedAlbumsData(uid, encoded)
				}
			}
		})
	}()
}

func (a *app) refreshPurchased(tab int, d *purchasedData) {
	if a.profile == nil || d.refreshing || d.page.loading {
		return
	}
	d.refreshing = true
	uid, epoch := a.profile.UserID, a.purchasedEpoch
	var cache *player.Cache
	if a.pl != nil {
		cache = a.pl.Cache
	}
	go func() {
		var songs []qqmusic.Track
		var albums []qqmusic.AlbumProfile
		var more bool
		var err error
		var encoded []byte
		if tab == purchasedSongs {
			songs, more, err = readPurchasedSongs(purchasedLimit, 0)
			if err == nil && cache != nil {
				encoded, _ = player.MarshalPurchasedSongs(songs, more && len(songs) > 0)
			}
		} else {
			albums, more, err = readPurchasedAlbums(purchasedLimit, 0)
			if err == nil && cache != nil {
				encoded, _ = player.MarshalPurchasedAlbums(albums, more && len(albums) > 0)
			}
		}
		a.post(func() {
			d.refreshing = false
			if a.profile == nil || a.profile.UserID != uid || a.purchasedEpoch != epoch || a.purchased[tab] != d || err != nil {
				return
			}
			d.songs = songs
			d.albums = albums
			d.page.loaded, d.page.more, d.page.err = true, more && len(songs)+len(albums) > 0, ""
			d.syncedAt = time.Now()
			d.stale = false
			if cache != nil && len(encoded) > 0 {
				if tab == purchasedSongs {
					go cache.KeepPurchasedSongsData(uid, encoded)
				} else {
					go cache.KeepPurchasedAlbumsData(uid, encoded)
				}
			}
		})
	}()
}

func (a *app) morePurchased(tab int, d *purchasedData) {
	if a.profile == nil || d.page.loading {
		return
	}
	d.page.loading, d.page.err = true, ""
	uid, epoch := a.profile.UserID, a.purchasedEpoch
	var cache *player.Cache
	if a.pl != nil {
		cache = a.pl.Cache
	}
	offset := len(d.songs) + len(d.albums)
	go func() {
		var songs []qqmusic.Track
		var albums []qqmusic.AlbumProfile
		var more bool
		var err error
		if tab == purchasedSongs {
			songs, more, err = readPurchasedSongs(purchasedLimit, offset)
		} else {
			albums, more, err = readPurchasedAlbums(purchasedLimit, offset)
		}
		a.post(func() {
			if a.profile == nil || a.profile.UserID != uid || a.purchasedEpoch != epoch || a.purchased[tab] != d {
				d.page.loading = false
				return
			}
			d.page.loading = false
			if err != nil {
				d.page.err = "已购音乐加载失败"
				return
			}
			d.songs = append(d.songs, songs...)
			d.albums = append(d.albums, albums...)
			d.page.loaded, d.page.more = true, more && len(songs)+len(albums) > 0
			d.syncedAt = time.Now()
			if cache != nil {
				if tab == purchasedSongs {
					snap := slices.Clone(d.songs)
					hasMore := d.page.more
					go cache.KeepPurchasedSongs(uid, snap, hasMore)
				} else {
					snap := slices.Clone(d.albums)
					hasMore := d.page.more
					go cache.KeepPurchasedAlbums(uid, snap, hasMore)
				}
			}
		})
	}()
}
