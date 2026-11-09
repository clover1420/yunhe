package app

import (
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/skeleton"
	accountpkg "lilt/internal/ui/features/account"
)

// The account and its login live in lilt/internal/ui/features/account; the app keeps
// the login's state and these names while its views still live here.
type loginState = accountpkg.State

// accountRestoreState is how restoring the saved login is going.
type accountRestoreState struct {
	gen     int
	loading bool
	failed  bool
	boot    bool
}

// beginLoginQR is replaced by UI tests so retry actions never contact QQ.
var beginLoginQR = qqmusic.QRKey
var beginWechatLoginQR = qqmusic.WechatQRKey

// Restoration tests isolate credential reads and the page loads that follow a recovered account.
var restoreLoginSession = qqmusic.RestoreSession
var readLoginAccount = qqmusic.Account

// applyLoginProfile is assigned in init: taking the method at package scope
// would form an initialization cycle through the login helpers.
var applyLoginProfile func(*app, *qqmusic.Profile)

func init() { applyLoginProfile = (*app).signedIn }

// boot signs in with the session in the keychain, if any, then loads the
// home page for whoever is listening.
func (a *app) boot() {
	a.homeLoading = true
	a.startAccountRestore(true)
}

// startAccountRestore reads saved credentials at boot; retries use the retained in-memory session.
func (a *app) startAccountRestore(boot bool) {
	if a.accountRestore.loading {
		return
	}
	a.accountRestore.gen++
	a.accountRestore.loading = true
	a.accountRestore.boot = boot
	gen := a.accountRestore.gen
	go func() {
		var prof *qqmusic.Profile
		var err error
		if !boot || restoreLoginSession() {
			prof, err = readLoginAccount()
		}
		a.post(func() {
			if !a.acceptAccountRestore(gen, prof, err) {
				return
			}
			if boot || err == nil {
				applyLoginProfile(a, a.profile)
				a.accountRestore.failed = err != nil
			}
		})
	}()
}

// acceptAccountRestore applies only the active request; failure keeps the session available to retry.
func (a *app) acceptAccountRestore(gen int, prof *qqmusic.Profile, err error) bool {
	if gen != a.accountRestore.gen || !a.accountRestore.loading {
		return false
	}
	a.accountRestore.loading = false
	a.accountRestore.boot = false
	a.accountRestore.failed = err != nil
	if err == nil {
		a.profile = prof
	}
	return true
}

// signedIn shows what is personal to prof, or to nobody.
func (a *app) signedIn(prof *qqmusic.Profile) {
	a.accountRestore = accountRestoreState{gen: a.accountRestore.gen + 1}
	a.profile = prof
	// Playback picks up where it was left, once the session that lets
	// the listener play their songs is back.
	a.restoreSession()
	a.mine, a.saved, a.recPlaylists, a.liked = nil, nil, nil, nil
	a.clearSongShelves()
	a.likedPrograms, a.askedPrograms = nil, nil
	a.likedAlbums = nil
	a.likedScroll = likedScrollState{}
	// What the listener may play, and follows, changed: load the pages
	// again.
	a.playlists = map[int64]*qqmusic.Playlist{}
	a.plErr = map[int64]string{}
	a.artists, a.albums, a.users, a.podcasts = map[int64]*artistData{}, map[int64]*albumData{}, map[int64]*userData{}, map[int64]*podcastData{}
	a.podcastTab = &podcastTab{}
	a.purchased = [2]*purchasedData{}
	a.purchasedScroll = [2]listScrollState{}
	a.searches = map[route]*searchData{}
	a.load(a.route)
	a.loadHome()
	if prof == nil {
		return
	}
	a.loadLiked(prof.UserID)
	a.loadMine(prof.UserID)
}

// accountDeps gathers what the account needs, once a frame.
func (a *app) accountDeps() accountpkg.Deps {
	return accountpkg.Deps{
		Palette:     a.pal,
		Art:         a.art,
		Login:       &a.login,
		AccountOpen: &a.accountOpen,
		Profile:     func() *qqmusic.Profile { return a.profile },
		Restore:     accountpkg.RestoreState{Loading: a.accountRestore.loading, Failed: a.accountRestore.failed},
		RetryRestore: func() {
			a.startAccountRestore(false)
		},
		Post:     a.post,
		SignedIn: a.signedIn,
		BeginQR:  func() (string, error) { return beginLoginQR() },
		BeginWechatQR: func() (string, error) {
			return beginWechatLoginQR()
		},
		Pane: a.pane,
		Line: func() ui.Color { return a.inkOn(false).line },
		QRSkeleton: func(c *ui.Context, pad float32) ui.Element {
			return skeleton.QR(c, shimmer{Base: shade(3), Sheen: shade(1)}.In(c), pad)
		},
		Sidebar:   a.sidebarSurface,
		Avatar:    a.avatar,
		OpenLogin: a.openLogin,
		OpenUser:  func(id int64) { a.open(route{Page: pageUser, ID: id}) },
		SignOut:   a.signOut,
	}
}

func (a *app) openLogin() { a.openLoginChannel(false) }

func (a *app) openLoginChannel(wechat bool) {
	if a.accountRestore.loading && a.accountRestore.boot {
		// Choosing a fresh login must still finish startup if the QR is later dismissed.
		applyLoginProfile(a, nil)
	}
	a.accountRestore.gen++
	a.accountRestore.loading = false
	a.accountRestore.boot = false
	accountpkg.Start(a.accountDeps(), wechat)
}

// acceptLoginQR applies a worker result on the main thread before polling starts.
func (a *app) acceptLoginQR(gen int, key string, code *ui.Bitmap, err error) bool {
	return accountpkg.Accept(a.accountDeps(), gen, key, code, err)
}

func (a *app) signOut() {
	accountpkg.Close(a.accountDeps())
	qqmusic.Logout()
	a.signedIn(nil)
}

func (a *app) loginDialog(c *ui.Context) {
	accountpkg.Dialog(c, a.accountDeps())
}

func (a *app) accountRow(c *ui.Context) {
	accountpkg.Row(c, a.accountDeps())
}
