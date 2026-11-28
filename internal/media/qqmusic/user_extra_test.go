package qqmusic

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Liked songs page by offset and send the account as enc_host_uin.
func TestUserFavoriteSongsPaging(t *testing.T) {
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
		if request.Req.Method != "CgiGetDiss" {
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
			return
		}
		param = request.Req.Param
		rpcReply(w, map[string]any{
			"code": 0, "subcode": 0, "msg": "",
			"total_song_num": 30, "hasmore": 1,
			"songlist": []any{map[string]any{
				"id": 9, "mid": "m9", "name": "N", "interval": 180,
				"singer": []any{map[string]any{"id": 1, "mid": "s1", "name": "S"}},
				"album":  map[string]any{"id": 2, "mid": "al", "name": "A"},
			}},
		})
	})

	page, err := UserFavoriteSongs("euin", 2, 10)
	if err != nil || page.Total != 30 || !page.More || len(page.Songs) != 1 || page.Songs[0].MID != "m9" {
		t.Fatalf("UserFavoriteSongs = %+v, %v", page, err)
	}
	if param["disstid"] != float64(0) || param["dirid"] != float64(201) || param["song_begin"] != float64(10) || param["song_num"] != float64(10) {
		t.Errorf("UserFavoriteSongs sent %v", param)
	}
	if param["enc_host_uin"] != "euin" || param["tag"] != float64(1) || param["userinfo"] != float64(1) || param["orderlist"] != float64(1) {
		t.Errorf("UserFavoriteSongs sent %v", param)
	}
}

// Removing a saved playlist sends the account's encrypted UIN and reports a
// failed playlist id rather than success.
func TestRemoveFavoritePlaylist(t *testing.T) {
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
		if request.Req.Module != "music.musicasset.PlaylistFavWrite" || request.Req.Method != "CancelFavPlaylist" {
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
			return
		}
		params = append(params, request.Req.Param)
		ids, _ := request.Req.Param["v_playlistId"].([]any)
		id := 0
		if len(ids) == 1 {
			id = int(ids[0].(float64))
		}
		switch id {
		case 901:
			rpcReply(w, map[string]any{"result": 0, "v_failedPlaylistId": []any{}})
		case 902:
			rpcReply(w, map[string]any{"result": 0, "v_failedPlaylistId": []any{902}})
		default:
			rpcReply(w, map[string]any{"result": 1, "v_failedPlaylistId": []any{}})
		}
	})
	setSession(Session{MusicID: "10001", MusicKey: "fake", LoginType: 2, EncryptUin: "enc"})

	if ok, err := RemoveFavoritePlaylist(901); err != nil || !ok {
		t.Fatalf("RemoveFavoritePlaylist(901) = %v, %v", ok, err)
	}
	if ok, err := RemoveFavoritePlaylist(902); err != nil || ok {
		t.Fatalf("RemoveFavoritePlaylist(902) = %v, %v", ok, err)
	}
	if ok, err := RemoveFavoritePlaylist(903); err != nil || ok {
		t.Fatalf("RemoveFavoritePlaylist(903) = %v, %v", ok, err)
	}
	if len(params) != 3 || params[0]["uin"] != "enc" {
		t.Fatalf("RemoveFavoritePlaylist sent %v", params)
	}
}
