package qqmusic

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Trending words, quick search and typed search decode QQ's replies and send
// their parameter names as the pages spell them.
func TestSearchExtras(t *testing.T) {
	params := make(map[string]map[string]any)
	comms := make(map[string]map[string]any)
	var quickQuery url.Values
	withQQServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			quickQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{
				"song": map[string]any{"count": 1, "itemlist": []any{map[string]any{
					"docid": "d1", "id": "1", "mid": "m1", "name": "N", "singer": "S", "pic": "p",
				}}},
				"singer": map[string]any{"count": 1, "itemlist": []any{map[string]any{
					"docid": "d2", "id": "2", "mid": "s2", "name": "S2",
				}}},
				"album": map[string]any{"count": 1, "itemlist": []any{map[string]any{
					"docid": "d3", "id": "3", "mid": "a3", "name": "A3",
				}}},
				"mv": map[string]any{"count": 1, "itemlist": []any{map[string]any{
					"vid": "v1", "name": "MV",
				}}},
			}})
			return
		}
		var request struct {
			Comm map[string]any `json:"comm"`
			Req  struct {
				Module, Method string
				Param          map[string]any
			} `json:"req_0"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		params[request.Req.Method] = request.Req.Param
		comms[request.Req.Method] = request.Comm
		switch request.Req.Method {
		case "GetHotkeyForQQMusicMobile":
			rpcReply(w, map[string]any{
				"hotkey_time": "t", "track_list_id": 7,
				"vec_hotkey": []any{map[string]any{
					"hotkey_id": "h1", "query": "周杰伦", "title": "周杰伦", "score": "99.5",
					"kind": 1, "type": 2, "source": 3, "need_top": 1, "subpos": 4, "song_type": 5,
					"direct_id": 123, "jump_tab": "0", "jump_url": "u", "cover_pic_url": "c",
					"pic_url": "p", "description": "d",
				}},
			})
		case "DoSearchForQQMusicMobile":
			rpcReply(w, map[string]any{
				"meta": map[string]any{"searchid": "sid", "perpage": 10, "nextpage": 3, "estimate_sum": 100, "sum": 25},
				"body": map[string]any{
					"item_song": []any{map[string]any{
						"id": 101, "mid": "m101", "title": "Song", "interval": 180,
						"singer": []any{map[string]any{"id": 1, "mid": "s1", "name": "S"}},
						"album":  map[string]any{"id": 2, "mid": "al", "name": "A"},
						"file":   map[string]any{"media_mid": "mm"},
					}},
					"singer": []any{map[string]any{
						"id": 1, "mid": "s1", "name": "S", "singerPic": "sp", "songNum": 9, "albumNum": 3,
					}},
					"item_album": []any{map[string]any{
						"albumID": 2, "albumMid": "al", "albumName": "A", "albumPic": "ap",
						"song_count": 4, "publicTime": "2020-01-02",
						"singer_list": []any{map[string]any{"id": 1, "mid": "s1", "name": "S"}},
					}},
					"item_songlist": []any{map[string]any{
						"tid": 88, "title": "PL", "cover": map[string]any{"default_url": "u"},
						"song_cnt": 3, "play_cnt": 100, "creator": map[string]any{"nick": "U"},
					}},
					"item_mv": []any{map[string]any{
						"id": 77, "vid": "v1", "title": "MV", "pic": "p", "play_count": 9,
						"duration": 200, "publish_date": "2020-01-01",
						"singerid": 1, "singermid": "s1", "singername": "S",
					}},
					"item_user": []any{map[string]any{"uin": 123, "nick": "U", "avatar": "a"}},
					"item_audio": []any{map[string]any{
						"albumID": 3, "albumMid": "a2", "albumName": "AA",
					}},
				},
			})
		default:
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
		}
	})
	setSession(Session{MusicID: "10001", MusicKey: "fixture-key", LoginType: 2})

	hotkeys, err := SearchHotkeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(hotkeys.Items) != 1 || hotkeys.Items[0].Query != "周杰伦" || hotkeys.Items[0].Score != "99.5" ||
		hotkeys.Items[0].DirectID != 123 || hotkeys.Items[0].NeedTop != 1 || hotkeys.TrackListID != "7" || hotkeys.HotkeyTime != "t" {
		t.Fatalf("SearchHotkeys = %+v", hotkeys)
	}
	id, _ := params["GetHotkeyForQQMusicMobile"]["search_id"].(string)
	if len(id) != 18 || strings.Trim(id, "0123456789") != "" {
		t.Errorf("SearchHotkeys sent search_id %q", id)
	}

	quick, err := QuickSearch("周杰伦")
	if err != nil {
		t.Fatal(err)
	}
	if quick.Songs.Count != 1 || quick.Songs.Items[0].MID != "m1" || quick.Singers.Items[0].MID != "s2" ||
		quick.Albums.Items[0].MID != "a3" || quick.Videos.Count != 1 || quick.Videos.Items[0].VID != "v1" {
		t.Fatalf("QuickSearch = %+v", quick)
	}
	if quickQuery.Get("key") != "周杰伦" {
		t.Errorf("QuickSearch sent key %q", quickQuery.Get("key"))
	}

	page, err := SearchByType("周杰伦", 0, 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if page.SearchID != "sid" || page.NextPage != 3 || !page.HasMore || page.Total != 25 || page.PerPage != 10 {
		t.Fatalf("SearchByType page = %+v", page)
	}
	if len(page.Songs) != 1 || page.Songs[0].MID != "m101" || page.Songs[0].MediaMID != "mm" || page.Songs[0].Duration != 180000 {
		t.Errorf("SearchByType songs = %+v", page.Songs)
	}
	if len(page.Artists) != 1 || page.Artists[0].MID != "s1" || page.Artists[0].PicURL != "sp" || page.Artists[0].MusicSize != 9 {
		t.Errorf("SearchByType artists = %+v", page.Artists)
	}
	if len(page.Albums) != 1 || page.Albums[0].MID != "al" || page.Albums[0].PicURL != "ap" || len(page.Albums[0].Artists) != 1 {
		t.Errorf("SearchByType albums = %+v", page.Albums)
	}
	if len(page.Playlists) != 1 || page.Playlists[0].ID != 88 || page.Playlists[0].Cover() != "u" || page.Playlists[0].Creator.Nickname != "U" {
		t.Errorf("SearchByType playlists = %+v", page.Playlists)
	}
	if len(page.MVs) != 1 || page.MVs[0].VID != "v1" || page.MVs[0].PlayCount != 9 || page.MVs[0].PublishDate != "2020-01-01" || page.MVs[0].SingerMID != "s1" {
		t.Errorf("SearchByType videos = %+v", page.MVs)
	}
	if len(page.Users) != 1 || page.Users[0].UserID != 123 || page.Users[0].Nickname != "U" {
		t.Errorf("SearchByType users = %+v", page.Users)
	}
	if len(page.AudioAlbums) != 1 || page.AudioAlbums[0].MID != "a2" {
		t.Errorf("SearchByType audio albums = %+v", page.AudioAlbums)
	}
	p := params["DoSearchForQQMusicMobile"]
	if p["query"] != "周杰伦" || p["search_type"] != float64(0) || p["num_per_page"] != float64(10) ||
		p["page_num"] != float64(2) || p["highlight"] != float64(1) || p["grp"] != float64(1) {
		t.Errorf("SearchByType sent %v", p)
	}
	if _, ok := p["searchid"].(string); !ok {
		t.Errorf("SearchByType did not send searchid: %v", p)
	}
	if _, ok := p["selectors"].(map[string]any); !ok {
		t.Errorf("SearchByType did not send selectors: %v", p)
	}
	if _, ok := p["vec_selectors"].([]any); !ok {
		t.Errorf("SearchByType did not send vec_selectors: %v", p)
	}
	if comms["DoSearchForQQMusicMobile"]["ct"] != float64(11) || comms["DoSearchForQQMusicMobile"]["cv"] != float64(14090008) ||
		comms["DoSearchForQQMusicMobile"]["qq"] != "10001" {
		t.Errorf("SearchByType used comm %v", comms["DoSearchForQQMusicMobile"])
	}
}
