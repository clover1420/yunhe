// Package likes owns the listener's hearts: liking and unliking songs and
// episodes, the hearts themselves, and the menu of a song.
package likes

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

// Deps are what the hearts and the song menu need from the app they run in:
// the player, the listener's like maps and state by pointer, and the views
// that live elsewhere by callback.
type Deps struct {
	Player  *player.Player
	Palette theme.Palette

	Profile  func() *qqmusic.Profile
	Liked    *map[int64]bool
	Programs *map[int64]bool
	Asked    *map[int64]bool
	// LikeGen and ProgramLikeGen count the clicks of each heart, so a reply
	// to an old one does not undo a newer.
	LikeGen        *map[int64]int
	ProgramLikeGen *map[int64]int

	Post  func(func())
	Toast *string

	LikeSong      func(id int64, like bool) error
	LikeProgram   func(id int64, like bool) error
	ProgramsLiked func(ids []int64) (map[int64]bool, error)

	OpenLogin     func()
	ShowMain      func()
	AfterSongLike func()

	Copy          func(text string) error
	OpenArtist    func(id int64)
	OpenAlbum     func(id int64)
	OpenPodcast   func(id int64)
	PlaylistItems func(m *ui.Menu, tr *qqmusic.Track, pid int64)
}

// Toggle likes a song or an episode, or unlikes it, at once in the app and
// then with NetEase, which may refuse: the heart goes back then. Signed out,
// it asks the listener to sign in.
func Toggle(d Deps, tr *qqmusic.Track) {
	if d.Profile() == nil {
		d.ShowMain()
		d.OpenLogin()
		return
	}
	if tr.ProgramID != 0 {
		toggleProgram(d, tr.ProgramID)
		return
	}
	if *d.Liked == nil {
		*d.Liked = map[int64]bool{}
	}
	if *d.LikeGen == nil {
		*d.LikeGen = map[int64]int{}
	}
	uid := d.Profile().UserID
	id, like := tr.ID, !(*d.Liked)[tr.ID]
	gen := (*d.LikeGen)[id] + 1
	(*d.LikeGen)[id] = gen
	setLiked(d, id, like)
	go func() {
		err := d.LikeSong(id, like)
		d.Post(func() {
			if d.Profile() == nil || d.Profile().UserID != uid || (*d.LikeGen)[id] != gen {
				return
			}
			if err != nil {
				setLiked(d, id, !like)
				*d.Toast = "操作失败，请稍后再试"
				return
			}
			if d.AfterSongLike != nil {
				d.AfterSongLike()
			}
		})
	}()
}

func toggleProgram(d Deps, id int64) {
	if *d.ProgramLikeGen == nil {
		*d.ProgramLikeGen = map[int64]int{}
	}
	uid := d.Profile().UserID
	like := !(*d.Programs)[id]
	gen := (*d.ProgramLikeGen)[id] + 1
	(*d.ProgramLikeGen)[id] = gen
	SetProgramLiked(d, id, like)
	go func() {
		err := d.LikeProgram(id, like)
		d.Post(func() {
			if d.Profile() == nil || d.Profile().UserID != uid || (*d.ProgramLikeGen)[id] != gen {
				return
			}
			if err != nil {
				SetProgramLiked(d, id, !like)
				*d.Toast = "操作失败，请稍后再试"
			}
		})
	}()
}

// IsLiked reports whether the listener likes tr, a song or an episode.
func IsLiked(d Deps, tr *qqmusic.Track) bool {
	if tr.ProgramID != 0 {
		return (*d.Programs)[tr.ProgramID]
	}
	return (*d.Liked)[tr.ID]
}

// LikeCurrent likes the song or episode playing, or unlikes it, from the
// keyboard (⌘D), which shows no heart: a toast says which it did.
func LikeCurrent(d Deps) {
	tr := d.Player.Current()
	if tr == nil {
		return
	}
	Toggle(d, tr)
	switch {
	case d.Profile() == nil:
		return // asked to sign in
	case !IsLiked(d, tr):
		*d.Toast = "已取消喜欢"
	case tr.ProgramID != 0:
		*d.Toast = "已喜欢这期节目"
	default:
		*d.Toast = "已添加到我喜欢的音乐"
	}
}

func setLiked(d Deps, id int64, like bool) {
	if like {
		if *d.Liked == nil {
			*d.Liked = map[int64]bool{}
		}
		(*d.Liked)[id] = true
	} else {
		delete(*d.Liked, id)
	}
}

// SetProgramLiked says the listener likes an episode, or doesn't, over
// what NetEase may still answer of it.
func SetProgramLiked(d Deps, id int64, like bool) {
	if *d.Programs == nil {
		*d.Programs = map[int64]bool{}
	}
	(*d.Programs)[id] = like
	delete(*d.Asked, id)
}

// AskPrograms asks NetEase whether the listener likes the episodes ids, those
// not asked about yet: NetEase lists no episodes they like. An episode it
// doesn't answer of shows not liked, and isn't asked again.
func AskPrograms(d Deps, ids []int64) {
	if d.Profile() == nil {
		return
	}
	programs, asked := *d.Programs, *d.Asked
	var ask []int64
	for _, id := range ids {
		if _, known := programs[id]; !known && !asked[id] {
			ask = append(ask, id)
		}
	}
	if len(ask) == 0 {
		return
	}
	if *d.Asked == nil {
		*d.Asked = map[int64]bool{}
	}
	for _, id := range ask {
		(*d.Asked)[id] = true
	}
	uid := d.Profile().UserID
	go func() {
		liked, _ := d.ProgramsLiked(ask)
		d.Post(func() { ProgramsAnswered(d, uid, ask, liked) })
	}()
}

// ProgramsAnswered takes NetEase's answer of whether uid likes the episodes
// asked about.
func ProgramsAnswered(d Deps, uid int64, asked []int64, liked map[int64]bool) {
	if d.Profile() == nil || d.Profile().UserID != uid {
		return
	}
	for _, id := range asked {
		// A heart clicked meanwhile knows better.
		if (*d.Asked)[id] {
			SetProgramLiked(d, id, liked[id])
		}
	}
}

// Button is a heart that likes tr: filled in red once liked. Liking pops it
// in on a spring while a ring swells around it and confetti flies out;
// unliking lets it give a little.
func Button(c *ui.Context, d Deps, tr *qqmusic.Track, size float32, color ui.Color, surface ...appui.ButtonSurface) ui.Element {
	if tr.ProgramID != 0 {
		AskPrograms(d, []int64{tr.ProgramID})
	}
	liked := IsLiked(d, tr)
	icon, label := appui.IconHeart, "喜欢"
	if liked {
		icon, color, label = appui.IconHeartFill, theme.Accent(d.Palette.Dark, 9), "取消喜欢"
	}
	// The burst goes on a wrapper, which the heart's pop does not scale.
	wrap := ui.Box(c).Center()
	var b ui.Element
	wrap.Children(func() { b = appui.IconButton(c, icon, size, color, label, surface...) })
	scale := b.AnimateTo("pop", 1, ui.Animation{Spring: ui.SpringOptions{Duration: 550 * time.Millisecond, Bounce: 0.55}})
	b.Scale(scale, scale)
	burst := ui.Local(wrap, "burst", func() time.Time { return time.Time{} })
	if b.Clicked() {
		switch {
		case d.Profile() == nil:
		case liked:
			b.Jump("pop", 0.8)
		default:
			b.Jump("pop", 0.2)
			*burst = c.Now()
		}
		Toggle(d, tr)
	}
	if !burst.IsZero() {
		start, accent := *burst, theme.Accent(d.Palette.Dark, 9)
		wrap.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if drawBurst(p, r, size, p.Now().Sub(start), accent) {
				p.AnimationFrame()
			}
		})
	}
	return wrap
}

// burstColors are the confetti's colors, around the heart's red.
var burstColors = []ui.Color{
	theme.Accent(false, 9), theme.BurstOrange, theme.Warm(false, 9),
	theme.BurstPurple, theme.BurstCyan, theme.BurstPink,
}

const burstTime = 750 * time.Millisecond

// drawBurst draws the burst of a heart of size in r, at elapsed since the
// like, and reports whether it goes on.
func drawBurst(p *ui.Painter, r ui.Rect, size float32, elapsed time.Duration, accent ui.Color) bool {
	t := float32(elapsed) / float32(burstTime)
	if t >= 1 || t < 0 {
		return false
	}
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	out := func(x float32) float32 { return 1 - (1-x)*(1-x)*(1-x) } // ease out

	// The ring swells from the heart and thins away, in the first half.
	if rt := t * 2; rt < 1 {
		radius := size * (0.35 + 0.65*out(rt))
		width := size * 0.22 * (1 - rt)
		ring := (&ui.Path{}).Circle(cx, cy, radius)
		p.StrokePath(ring, width, accent.Alpha(0.9*(1-rt)))
	}

	// Two rings of confetti fly out, slowing, shrinking and fading; the
	// inner one a little behind and turned, as if thrown at once.
	for ring := range 2 {
		lag := float32(ring) * 0.08
		pt := (t - 0.12 - lag) / (0.88 - lag)
		if pt <= 0 {
			continue
		}
		pt = min(pt, 1)
		const n = 7
		dist := size * (0.55 + (1.15-0.25*float32(ring))*out(pt))
		dot := size * (0.16 - 0.04*float32(ring)) * (1 - pt*pt)
		alpha := 1 - pt*pt
		for i := range n {
			angle := (float64(i)/n + float64(ring)*0.5/n - 0.25) * 2 * math.Pi
			x := cx + dist*float32(math.Cos(angle))
			y := cy + dist*float32(math.Sin(angle))
			color := burstColors[(i+ring*3)%len(burstColors)].Alpha(alpha)
			p.Fill(ui.Rect{X: x - dot, Y: y - dot, W: 2 * dot, H: 2 * dot}, color, dot)
		}
	}
	return true
}
