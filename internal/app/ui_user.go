package app

import (
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
)

// userDetail and userPlaylists load a user and their playlists from
// NetEase, and followedArtists and subscribedPodcasts what the listener
// follows; tests replace them.
var (
	userDetail           = qqmusic.UserDetail
	userPlaylists        = qqmusic.UserPlaylists
	followedArtists      = qqmusic.FollowedArtists
	subscribedPodcasts   = qqmusic.SubscribedPodcasts
	userMembership       = qqmusic.UserMembership
	userMembershipExpiry = qqmusic.UserMembershipExpiry
	userMembershipDates  = qqmusic.UserMembershipDates
)

// userData is what the app has of a user: who they are, and the playlists
// they made that anyone may see. Of the listener's own page, it has the
// artists they follow and the podcasts they subscribe to too, which
// NetEase shows no one else.
type userData struct {
	user      *qqmusic.User
	playlists []qqmusic.Playlist
	loading   int // how many of the two are on their way
	err       string

	artists       []qqmusic.ArtistProfile
	podcasts      []qqmusic.Podcast
	artistsShown  bool // whether artists has loaded once
	podcastsShown bool
	subsLoading   int
	membership    qqmusic.Membership
	memberLoading bool
	memberErr     bool
	memberExpiry  string
	expiryLoading bool
	expiryErr     bool
	greenExpiry   string
	superExpiry   string
}

func (a *app) loadUser(id int64) {
	d := a.users[id]
	if d == nil {
		d = &userData{}
		a.users[id] = d
	}
	if a.profile != nil && a.profile.UserID == id {
		a.loadSubscriptions(d)
		a.loadMembership(id, d)
	}
	if d.user != nil || d.loading > 0 {
		return
	}
	d.loading, d.err = 2, ""
	go func() {
		u, err := userDetail(id)
		a.post(func() {
			d.loading--
			if err != nil {
				d.err = "用户加载失败"
				return
			}
			d.user = u
		})
	}()
	go func() {
		made, _, err := userPlaylists(id)
		a.post(func() {
			d.loading--
			if err == nil {
				d.playlists = publicPlaylists(made, id)
			}
		})
	}()
}

func (a *app) loadMembership(id int64, d *userData) {
	if d.memberLoading {
		return
	}
	d.memberLoading, d.memberErr = true, false
	fetch := userMembership
	go func() {
		level, err := fetch()
		a.post(func() { a.acceptMembership(id, d, level, err) })
	}()
}

func (a *app) acceptMembership(id int64, d *userData, level qqmusic.Membership, err error) {
	if a.profile == nil || a.profile.UserID != id || a.users[id] != d {
		return
	}
	d.memberLoading, d.memberErr = false, err != nil
	if err != nil {
		a.loadMembershipDates(id, d)
	}
	if err == nil {
		if d.membership != level {
			d.memberExpiry = ""
		}
		d.membership = level
		d.expiryErr = false
		if level == qqmusic.MembershipGreen || level == qqmusic.MembershipSuper {
			a.loadMembershipExpiry(id, d, level)
		}
	}
}

func (a *app) loadMembershipDates(id int64, d *userData) {
	if d.expiryLoading {
		return
	}
	d.expiryLoading, d.expiryErr = true, false
	fetch := userMembershipDates
	go func() {
		green, super, err := fetch()
		a.post(func() {
			if a.profile == nil || a.profile.UserID != id || a.users[id] != d {
				return
			}
			d.expiryLoading, d.expiryErr = false, err != nil
			if !d.memberErr && (d.membership == qqmusic.MembershipGreen || d.membership == qqmusic.MembershipSuper) {
				a.loadMembershipExpiry(id, d, d.membership)
				return
			}
			if err == nil {
				d.greenExpiry, d.superExpiry = green, super
			}
		})
	}()
}

func (a *app) loadMembershipExpiry(id int64, d *userData, level qqmusic.Membership) {
	if d.expiryLoading {
		return
	}
	d.expiryLoading, d.expiryErr = true, false
	fetch := userMembershipExpiry
	go func() {
		expiry, err := fetch(level)
		a.post(func() { a.acceptMembershipExpiry(id, d, level, expiry, err) })
	}()
}

func (a *app) acceptMembershipExpiry(id int64, d *userData, level qqmusic.Membership, expiry string, err error) {
	if a.profile == nil || a.profile.UserID != id || a.users[id] != d {
		return
	}
	d.expiryLoading = false
	if d.memberErr {
		a.loadMembershipDates(id, d)
		return
	}
	if d.membership != level {
		if d.membership == qqmusic.MembershipGreen || d.membership == qqmusic.MembershipSuper {
			a.loadMembershipExpiry(id, d, d.membership)
		}
		return
	}
	d.expiryErr = err != nil
	if err == nil {
		d.memberExpiry = expiry
	}
}

func (a *app) membershipBadge(c *ui.Context, id int64, d *userData) {
	p := a.pal
	label, ink, background := "会员状态未知", p.Muted, neutral(p.Dark, 3)
	switch d.membership {
	case qqmusic.MembershipSuper:
		label, ink, background = "超级会员", warm(p.Dark, 11), warm(p.Dark, 3)
	case qqmusic.MembershipGreen:
		label, ink, background = "绿钻会员", accent(p.Dark, 11), accent(p.Dark, 3)
	case qqmusic.MembershipFree:
		label = "免费账号"
	}
	if d.memberErr {
		label, ink, background = "会员状态暂不可用", p.Muted, neutral(p.Dark, 3)
	} else if d.memberLoading && d.membership == "" {
		label = "正在查询会员状态"
	}
	ui.Row(c).Shrink(1).Wrap().AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Box(c).Padding(3, 8).Radius(4).Background(background).Children(func() {
			ui.Text(c, label).FontSize(11).FontWeight(500).TextColor(ink).SingleLine()
		})
		if !d.memberErr && !d.expiryErr && d.memberExpiry != "" {
			ui.Text(c, d.memberExpiry+" 到期").FontSize(12).TextColor(p.Muted).SingleLine()
		}
		if d.memberErr && ui.ButtonBase(c).Label("重试会员状态").Padding(4, 6).Children(func() {
			ui.Text(c, "重试").FontSize(12).TextColor(p.Accent)
		}).Clicked() {
			a.loadMembership(id, d)
		}
		if d.memberErr && !d.expiryErr {
			if d.greenExpiry != "" {
				ui.Text(c, "绿钻有效期至 "+d.greenExpiry).FontSize(12).TextColor(p.Muted).SingleLine()
			}
			if d.superExpiry != "" {
				ui.Text(c, "超级会员有效期至 "+d.superExpiry).FontSize(12).TextColor(p.Muted).SingleLine()
			}
		}
		if !d.memberErr && d.expiryErr && ui.ButtonBase(c).Label("重试到期时间").Padding(4, 6).Children(func() {
			ui.Text(c, "到期时间获取失败，重试").FontSize(12).TextColor(p.Accent)
		}).Clicked() {
			a.loadMembershipExpiry(id, d, d.membership)
		}
	})
}

// loadSubscriptions loads the artists the listener follows and the
// podcasts they subscribe to, each time their page shows: they follow and
// subscribe on other pages, and these keep up. What they had shows while
// the lists are on their way.
func (a *app) loadSubscriptions(d *userData) {
	if d.subsLoading > 0 {
		return
	}
	d.subsLoading = 2
	go func() {
		ars, err := followedArtists()
		a.post(func() {
			d.subsLoading--
			if err == nil {
				d.artists, d.artistsShown = ars, true
			}
		})
	}()
	go func() {
		pods, err := subscribedPodcasts()
		a.post(func() {
			d.subsLoading--
			if err == nil {
				d.podcasts, d.podcastsShown = pods, true
			}
		})
	}()
}

// publicPlaylists are the playlists of lists a user made that anyone may
// see: their own, as made lists those they saved too, but not their
// private ones.
func publicPlaylists(lists []qqmusic.Playlist, uid int64) []qqmusic.Playlist {
	var out []qqmusic.Playlist
	for _, pl := range lists {
		if pl.Creator.UserID == uid && pl.Privacy != 10 {
			out = append(out, pl)
		}
	}
	return out
}
