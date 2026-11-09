// Package account owns the listener's account: signing in with a QQ or
// WeChat QR code, the row at the foot of the sidebar that says who is
// signed in, and the menu it drops.
package account

import (
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/art"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// State is a QR code login under way.
type State struct {
	Open     bool
	Gen      int // counts logins, to stop the polls of closed ones
	Key      string
	Code     *ui.Bitmap
	Status   int // a QR state, or 0 while the code loads
	Scanner  string
	Avatar   string
	Err      string
	Finished bool
	Wechat   bool
}

// Deps are what the account needs from the app it runs in: the login's
// state and the profile by pointer, the app's state by function, and the
// views that live elsewhere by callback.
type Deps struct {
	Palette theme.Palette
	Art     *art.Cache
	Login   *State

	AccountOpen *bool
	Profile     func() *qqmusic.Profile
	Restore     RestoreState

	Post         func(func())
	SignedIn     func(*qqmusic.Profile)
	RetryRestore func()

	BeginQR       func() (string, error)
	BeginWechatQR func() (string, error)

	Pane       func(e ui.Element, radius float32, onDark bool) ui.Element
	Line       func() ui.Color
	QRSkeleton func(c *ui.Context, pad float32) ui.Element
	Sidebar    func(c *ui.Context, fallback ui.Color, opacity float32) ui.Color
	Avatar     func(c *ui.Context, url string, size float32, empty ui.Color)
	OpenLogin  func()
	OpenUser   func(id int64)
	SignOut    func()
}

// RestoreState is how restoring the saved login is going.
type RestoreState struct {
	Loading bool
	Failed  bool
}

// Start begins a login with a new QR code, from QQ or WeChat.
func Start(d Deps, wechat bool) {
	l := d.Login
	qqmusic.CancelQR(l.Key)
	*l = State{Open: true, Gen: l.Gen + 1, Wechat: wechat}
	gen := l.Gen
	begin := d.BeginQR
	if wechat {
		begin = d.BeginWechatQR
	}
	go func() {
		key, err := begin()
		var code *ui.Bitmap
		if err == nil {
			var png []byte
			png, err = qqmusic.QRImage(key)
			if err == nil {
				code, err = ui.DecodeBitmap(png)
				if err != nil {
					err = &qqmusic.LoginError{Step: "qr-image", Err: err}
				}
			}
		}
		d.Post(func() {
			if Accept(d, gen, key, code, err) {
				poll(d, gen, key)
			}
		})
	}()
}

// Accept applies a worker result on the main thread before polling starts.
func Accept(d Deps, gen int, key string, code *ui.Bitmap, err error) bool {
	l := d.Login
	if gen != l.Gen || !l.Open {
		qqmusic.CancelQR(key)
		return false
	}
	if err != nil {
		l.Err = qqmusic.LoginErrorMessage(err)
		return false
	}
	l.Key, l.Code, l.Status, l.Err = key, code, qqmusic.QRWaiting, ""
	return true
}

// poll asks every two seconds whether the code was scanned, until the login
// ends or the dialog closes.
func poll(d Deps, gen int, key string) {
	go func() {
		for {
			time.Sleep(2 * time.Second)
			code, msg, name, avatar, err := qqmusic.QRCheck(key)
			var prof *qqmusic.Profile
			if code == qqmusic.QRConfirmed {
				prof, err = qqmusic.Account()
				if err == nil {
					err = qqmusic.SaveSession()
				}
			}
			done := make(chan bool, 1)
			d.Post(func() {
				l := d.Login
				if gen != l.Gen || !l.Open {
					done <- true
					return
				}
				switch {
				case err != nil:
					l.Err = qqmusic.LoginErrorMessage(err)
				case code == qqmusic.QRConfirmed:
					if prof == nil {
						l.Err = "未能获取 QQ 音乐账号，请刷新二维码"
						break
					}
					l.Status, l.Finished = code, true
					d.SignedIn(prof)
					closeSoon(d, gen)
				case code == qqmusic.QRExpired || code == qqmusic.QRWaiting || code == qqmusic.QRScanned:
					l.Status = code
					if name != "" {
						l.Scanner, l.Avatar = name, avatar
					}
				default:
					l.Err = msg
					if l.Err == "" {
						l.Err = "登录失败，请稍后再试"
					}
				}
				done <- l.Status == qqmusic.QRExpired || l.Finished || l.Err != ""
			})
			if <-done {
				return
			}
		}
	}()
}

// closeSoon closes the dialog a moment after it showed the welcome.
func closeSoon(d Deps, gen int) {
	go func() {
		time.Sleep(1200 * time.Millisecond)
		d.Post(func() {
			if gen == d.Login.Gen {
				d.Login.Open = false
			}
		})
	}()
}

// Close stops the login under way and its polls.
func Close(d Deps) {
	qqmusic.CancelQR(d.Login.Key)
	d.Login.Gen++
}

// Dialog shows the selected QQ or WeChat QR code.
func Dialog(c *ui.Context, d Deps) {
	p := d.Palette
	l := d.Login
	ui.DialogBase(c.Key("login"), &l.Open, func(backdrop, panel ui.Element) {
		backdrop.Background(theme.Ink.Alpha(0.35))
		panel.Width(340).Padding(28, 28, 26, 28).Gap(6).AlignItems(ui.Center)
		d.Pane(panel, 20, false)
		panel.Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.94}, Exit: &ui.Motion{Scale: 0.96}})
		channel := "QQ"
		if l.Wechat {
			channel = "微信"
		}
		title, sub := "扫码登录", "打开"+channel+"，扫一扫登录"
		switch {
		case l.Finished:
			title, sub = "登录成功", "欢迎回来，"+profileName(d)
		case l.Err != "":
			title, sub = "登录失败", l.Err
		case l.Status == qqmusic.QRScanned:
			title, sub = "扫描成功", "请在手机上确认登录"
		case l.Status == qqmusic.QRExpired:
			sub = "二维码已失效"
		}
		ui.Text(c, title).FontSize(20).FontWeight(700)
		if !l.Finished {
			kind := 0
			if l.Wechat {
				kind = 1
			}
			if appui.Segmented(c.Key("login-channel"), &kind, "QQ", "微信") {
				Start(d, kind == 1)
			}
		}
		ui.Text(c, sub).Width(284).MaxLines(3).FontSize(13).TextColor(p.Muted).TextAlign(ui.Center).Margin(0, 0, 14, 0)

		ui.Box(c).Size(212, 212).Radius(14).Background(theme.Paper).Border(1, p.Hairline).Children(func() {
			if l.Code != nil {
				code := l.Code
				opacity := float32(1)
				if l.Status == qqmusic.QRScanned || l.Status == qqmusic.QRExpired || l.Finished || l.Err != "" {
					opacity = 0.12
				}
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Opacity(opacity).
					Children(func() { ui.Image(c, code).Fill().Fit(ui.Contain) })
			} else if l.Err == "" {
				// On white whatever the appearance, as the code is.
				d.QRSkeleton(c, 14).Absolute().Top(0).Left(0).Right(0).Bottom(0)
			}
			ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().Gap(10).Children(func() {
				switch {
				case l.Status == qqmusic.QRExpired || l.Err != "":
					if appui.PillButton(c.Key("refresh"), appui.IconRepeat, "刷新二维码", p.Accent, p.OnAccent).Clicked() {
						Start(d, l.Wechat)
					}
				case l.Finished || l.Status == qqmusic.QRScanned:
					if bmp := appui.Bitmap(d.Art.Get(qqmusic.Sized(l.Avatar, 160), art.Cover)); bmp != nil {
						pic := ui.Box(c).Size(72, 72).Radius(36).Shadow(0, 6, 16, -4, theme.Ink.Alpha(0.3))
						in := appui.ArtIn(pic, bmp)
						pic.Children(func() { appui.ArtImage(c, bmp, 36, in) })
					}
					if l.Scanner != "" {
						ui.Text(c, l.Scanner).FontSize(14).FontWeight(600).TextColor(theme.Neutral(false, 12))
					}
				}
			})
		})
	})
	if !l.Open {
		qqmusic.CancelQR(l.Key)
	}
}

func profileName(d Deps) string {
	if p := d.Profile(); p != nil {
		return p.Nickname
	}
	return ""
}

// Row sits at the foot of the sidebar: who is signed in, with a dropdown to
// open their page or sign out, or a way to sign in.
func Row(c *ui.Context, d Deps) {
	p := d.Palette
	open := d.AccountOpen
	b := ui.ButtonBase(c.Key("account")).Height(40).PaddingX(8).Gap(10).Radius(10).AlignItems(ui.Center).Justify(ui.Start).FocusRing(false)
	if b.Hovered() || *open {
		b.Background(d.Sidebar(c, p.Hover, 0.035))
	}
	if d.Profile() == nil {
		status := "未登录"
		if d.Restore.Loading {
			status = "正在恢复登录"
		} else if d.Restore.Failed {
			status = "登录恢复失败"
		}
		b.Label("账户").Expanded(*open).Children(func() {
			ui.Box(c).Size(26, 26).Radius(13).Background(p.Press).Center().Children(func() {
				ui.Icon(c, appui.IconUser).Size(14, 14).TextColor(p.Muted)
			})
			ui.Text(c, status).FontSize(13).FontWeight(500).TextColor(p.Muted).Grow(1)
			ui.Icon(c, appui.IconUpDown).Size(13, 13).TextColor(p.Faint)
		})
		if b.Clicked() {
			*open = !*open
		}
		menu(c, d, b)
		return
	}
	prof := d.Profile()
	b.Label(prof.Nickname).Expanded(*open).Children(func() {
		d.Avatar(c, prof.AvatarURL, 26, p.Press)
		ui.Text(c, prof.Nickname).FontSize(13).FontWeight(600).SingleLine().Grow(1).Shrink(1).MinWidth(0)
		ui.Icon(c, appui.IconUpDown).Size(13, 13).TextColor(p.Faint)
	})
	if b.Clicked() {
		*open = !*open
	}
	menu(c, d, b)
}

// menu drops up from the account row, as wide as it, in the colors of the
// volume popover: the listener's own page and signing out.
func menu(c *ui.Context, d Deps, row ui.Element) {
	open := d.AccountOpen
	ui.PopoverBase(c.Key("account-menu"), row, open, func(panel ui.Element) {
		panel.AttachTo(row, ui.AnchorTopStart, ui.AnchorBottomStart).Margin(0, 0, 6, 0).Padding(5).Gap(1)
		d.Pane(panel, 14, false)
		panel.Role(ui.RoleMenu).FocusGroup(ui.Vertical).Label("账户").TransformOrigin(0.5, 1).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.96, Y: 4}, Exit: &ui.Motion{Scale: 0.98}})
		if w := row.Bounds().W; w > 0 {
			panel.Width(w)
		}
		prof := d.Profile()
		if prof == nil {
			r := d.Restore
			if r.Failed || r.Loading {
				retry := menuItem(c, d.Palette, appui.IconRepeat, "重试恢复登录", "").Disabled(r.Loading)
				if !r.Loading {
					retry.AutoFocus()
				}
				if retry.Clicked() {
					*open = false
					d.RetryRestore()
				}
			}
			label := "登录 QQ 音乐"
			if r.Failed {
				label = "重新扫码登录"
			}
			scan := menuItem(c, d.Palette, appui.IconUser, label, "")
			if !r.Failed || r.Loading {
				scan.AutoFocus()
			}
			if scan.Clicked() {
				*open = false
				d.OpenLogin()
			}
		} else {
			if menuItem(c, d.Palette, appui.IconUser, "我的主页", "").AutoFocus().Clicked() {
				*open = false
				d.OpenUser(prof.UserID)
			}
		}
		if prof != nil {
			ui.Box(c).Height(0.5).Margin(4, 8).Background(d.Line())
			if menuItem(c, d.Palette, appui.IconSignOut, "退出登录", "").Clicked() {
				*open = false
				d.SignOut()
			}
		}
	})
}

// menuItem is an item of a dropdown: an icon, its label, and the shortcut
// that does the same, if any.
func menuItem(c *ui.Context, p theme.Palette, icon *ui.SVG, label, shortcut string) ui.Element {
	b := ui.ButtonBase(c.Key(label)).Role(ui.RoleMenuItem).Label(label).Height(30).PaddingX(8).Gap(9).Radius(8).
		AlignItems(ui.Center).Justify(ui.Start).FocusRing(false)
	if b.Hovered() || b.FocusVisible() {
		b.Background(p.Press)
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(p.Muted)
		ui.Text(c, label).FontSize(13).Grow(1)
		if shortcut != "" {
			ui.Text(c, shortcut).FontSize(12).TextColor(p.Faint)
		}
	})
	return b
}
