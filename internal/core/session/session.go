// Package session keeps where playback was left, in session.json in the user
// data directory: on quitting, and every little while as it plays, in case
// the app does not get to quit.
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo"

	"lilt/internal/media/player"
)

// Path returns where the session is kept.
func Path() (string, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

// Load reads the session last kept, its raw JSON, and whether there was one.
func Load() (player.Session, string, bool) {
	path, err := Path()
	if err != nil {
		return player.Session{}, "", false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return player.Session{}, "", false
	}
	var s player.Session
	if json.Unmarshal(b, &s) != nil {
		return player.Session{}, "", false
	}
	return s, string(b), true
}

// Encode returns the session as the JSON kept on disk.
func Encode(s player.Session) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Write keeps the session's raw JSON atomically.
func Write(raw string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path+".part", []byte(raw), 0o600); err != nil {
		return err
	}
	return os.Rename(path+".part", path)
}

// Writer keeps snapshots in the background: one worker writes them in order,
// replacing queued work with the latest snapshot. It does not need the main
// thread, so shutdown can wait for it.
type Writer struct {
	mu      sync.Mutex
	pending *player.Session
	done    chan struct{}
	saved   string
}

// Submit queues s to be kept, replacing any snapshot not yet written.
func (w *Writer) Submit(s player.Session) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = &s
	if w.done == nil {
		w.done = make(chan struct{})
		go w.run()
	}
}

// Wait blocks until the queued snapshot, if any, has been written.
func (w *Writer) Wait() {
	w.mu.Lock()
	done := w.done
	w.mu.Unlock()
	if done != nil {
		<-done
	}
}

// MarkSaved records raw as the session on disk, so it is not written again.
func (w *Writer) MarkSaved(raw string) {
	w.mu.Lock()
	w.saved = raw
	w.mu.Unlock()
}

func (w *Writer) run() {
	for {
		w.mu.Lock()
		job, saved := w.pending, w.saved
		if job == nil {
			close(w.done)
			w.done = nil
			w.mu.Unlock()
			return
		}
		w.pending = nil
		w.mu.Unlock()

		b, err := json.Marshal(job)
		if err != nil || string(b) == saved {
			continue
		}
		if Write(string(b)) != nil {
			continue
		}
		w.mu.Lock()
		w.saved = string(b)
		w.mu.Unlock()
	}
}
