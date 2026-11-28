package qqmusic

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

// Reordering sends every song's new position and leaves QQ's 80095 rejection
// as an error, as the songs are then not all in the playlist.
func TestReorderPlaylistSongsRejection(t *testing.T) {
	var param map[string]any
	withQQServer(t, func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Req struct {
				Module, Method string
				Param          map[string]any
			} `json:"req_0"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Req.Method != "SeqSonglist" {
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
		}
		param = request.Req.Param
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "req_0": map[string]any{"code": 80095}})
	})
	setSession(Session{MusicID: "10001", MusicKey: "fake", LoginType: 2})

	ok, err := ReorderPlaylistSongs(12, 900, []SongRef{{ID: 1, Type: 1}, {ID: 2, Type: 3}, {ID: 3, Type: 5}})
	if ok || err == nil {
		t.Fatalf("ReorderPlaylistSongs = %v, %v", ok, err)
	}
	var protocol *ProtocolError
	if !errors.As(err, &protocol) || protocol.Code != 80095 {
		t.Fatalf("ReorderPlaylistSongs error = %v, want 80095", err)
	}
	if param["dirId"] != float64(12) || param["tid"] != float64(900) {
		t.Errorf("ReorderPlaylistSongs sent %v", param)
	}
	songs, _ := param["v_songPair"].([]any)
	want := []float64{0, 1, 5}
	if len(songs) != len(want) {
		t.Fatalf("ReorderPlaylistSongs sent %v", param["v_songPair"])
	}
	for i, entry := range songs {
		if entry.(map[string]any)["songType"] != want[i] {
			t.Errorf("song %d type = %v, want %v", i, entry.(map[string]any)["songType"], want[i])
		}
	}
}

// Adding songs already in a playlist reports false without an error, and the
// request keeps bFmtUtf8 as a JSON boolean.
func TestAddPlaylistSongsAlreadyListed(t *testing.T) {
	var params []map[string]any
	withQQServer(t, func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Req struct {
				Module, Method string
				Param          map[string]any
			} `json:"req_0"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Req.Method != "AddSonglist" {
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
		}
		params = append(params, request.Req.Param)
		songs, _ := request.Req.Param["v_songInfo"].([]any)
		if len(songs) == 1 && songs[0].(map[string]any)["songId"] == float64(1) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "req_0": map[string]any{"code": 80092}})
			return
		}
		rpcReply(w, map[string]any{"retCode": 0})
	})
	setSession(Session{MusicID: "10001", MusicKey: "fake", LoginType: 2})

	if ok, err := AddPlaylistSongs(201, 0, []SongRef{{ID: 1}}); err != nil || ok {
		t.Fatalf("AddPlaylistSongs(listed) = %v, %v", ok, err)
	}
	if ok, err := AddPlaylistSongs(201, 0, []SongRef{{ID: 2}}); err != nil || !ok {
		t.Fatalf("AddPlaylistSongs = %v, %v", ok, err)
	}
	if params[0]["bFmtUtf8"] != true {
		t.Errorf("AddPlaylistSongs sent bFmtUtf8 %v, want JSON true", params[0]["bFmtUtf8"])
	}
}
