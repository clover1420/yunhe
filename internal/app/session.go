package app

import (
	"time"

	"github.com/egoist/mygo"

	"lilt/internal/core/session"
	"lilt/internal/media/qqmusic"
)

// saveSession keeps where playback is, if it moved on since last kept.
func (a *app) saveSession() {
	if a.quitting {
		defer a.sessionWrites.Wait()
	}
	s, ok := a.pl.Session()
	if !ok {
		return
	}
	a.sessionWrites.Submit(s)
}

// sessionEvery is how long after playback changed, and how often as it
// plays, the session is kept.
const sessionEvery = 30 * time.Second

// keepSession saves the session a while after playback changed, and every
// while again as it plays. Paused, nothing is due, and the app sleeps.
func (a *app) keepSession() {
	if a.sessionDue {
		return
	}
	a.sessionDue = true
	sessionAfter(sessionEvery, func() {
		a.sessionDue = false
		a.saveSession()
		if a.pl.Playing() {
			a.keepSession()
		}
	})
}

// sessionAfter runs fn on the main thread after d, without redrawing a
// window as Update would.
var sessionAfter = func(d time.Duration, fn func()) {
	time.AfterFunc(d, func() { mygo.RunOnMain(fn) })
}

// restoreSession picks up playback where it was left, paused, once the
// listener's session is back: the songs they may play depend on it.
func (a *app) restoreSession() {
	if a.restored {
		return
	}
	a.restored = true
	s, raw, ok := session.Load()
	if !ok {
		return
	}
	a.sessionWrites.MarkSaved(raw)
	qqmusic.RememberTracks(s.Queue)
	a.pl.Restore(s)
	if s.Radio != "" && a.pl.Current() != nil {
		a.pl.AttachRadio(s.Radio, a.radioFeed(s.Radio))
	}
}
