package qqmusic

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The home feed and the recommendation streams decode QQ's replies and send
// their parameter names as the pages spell them.
func TestRecommendExtras(t *testing.T) {
	params := make(map[string]map[string]any)
	comms := make(map[string]map[string]any)
	withQQServer(t, func(w http.ResponseWriter, r *http.Request) {
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
		case "get_recommend_feed":
			rpcReply(w, map[string]any{
				"retcode": 0, "prompt": "P", "load_mark": 1, "d_num": 2,
				"v_shelf": []any{map[string]any{
					"id": 207, "title_template": "Hello {String}", "title_content": "World",
					"v_niche": []any{map[string]any{"v_card": []any{
						map[string]any{
							"id": "42", "type": 200, "title": "Song", "cover": "sc",
							"songInfo": map[string]any{"id": 42, "mid": "m42", "name": "Song", "interval": 180,
								"singer": []any{map[string]any{"id": 1, "mid": "s1", "name": "S"}},
								"album":  map[string]any{"id": 2, "mid": "al", "name": "A"}},
						},
						map[string]any{"id": "9001", "type": 500, "subtype": 0, "jumptype": 10014, "title": "PL", "cover": "pc", "cnt": 2085000},
						map[string]any{"id": "7", "type": 300, "subtype": 311, "jumptype": 10012, "title": "MV", "vid": "v1", "pic": "mp",
							"singer": []any{map[string]any{"id": 1, "mid": "s1", "name": "S"}}},
						map[string]any{"id": "5", "type": 100, "title": "Album", "cover": "ac", "albumid": 5, "albummid": "am", "albumname": "Album"},
						map[string]any{"id": "0", "type": 500, "subtype": 510, "jumptype": 10014, "title": "Daily"},
					}}},
				}},
			})
		case "get_radio_track":
			rpcReply(w, map[string]any{"tracks": []any{map[string]any{
				"id": 9, "mid": "m9", "name": "N", "interval": 180,
				"singer": []any{map[string]any{"id": 1, "mid": "s1", "name": "S"}},
			}}})
		case "GetRadarSong":
			rpcReply(w, map[string]any{
				"VecSongs": []any{map[string]any{"Track": map[string]any{
					"id": 10, "mid": "m10", "name": "R", "interval": 120,
					"singer": []any{map[string]any{"id": 1, "mid": "s1", "name": "S"}},
					"album":  map[string]any{"id": 2, "mid": "al", "name": "A"},
				}}},
				"RecommendSongIds": []any{10, 11}, "BaseSongIds": []any{3},
				"HasMore": true, "Toast": "T", "TimeStamp": 123, "VideoCards": map[string]any{"a": 1},
			})
		case "get_new_song_info":
			rpcReply(w, map[string]any{
				"lanlist": []any{map[string]any{"lan": "内地", "type": 1}},
				"lan":     "内地", "ret_msg": "ok", "type": 1,
				"songlist": []any{map[string]any{
					"id": 11, "mid": "m11", "name": "NS", "interval": 200,
					"album": map[string]any{"id": 2, "mid": "al", "name": "A"},
				}},
				"songTagInfoList": []any{map[string]any{"id": 1, "tagid": 2, "tag": "Tag", "link": "L", "from_type": 1}},
			})
		case "GetRecommendFeed":
			rpcReply(w, map[string]any{
				"List": []any{map[string]any{"Playlist": map[string]any{"basic": map[string]any{
					"tid": 88, "title": "RL", "cover": map[string]any{"default_url": "u"},
					"song_cnt": 3, "play_cnt": 100, "creator": map[string]any{"nick": "U"},
				}}}},
				"HasMore": true, "FromLimit": 25, "Msg": "m",
			})
		default:
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
		}
	})
	setSession(Session{MusicID: "10001", MusicKey: "fixture-key", LoginType: 2})

	feed, err := HomeFeed(1, 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Shelves) != 1 || feed.Shelves[0].Title != "Hello World" || len(feed.Shelves[0].Cards) != 5 {
		t.Fatalf("HomeFeed = %+v", feed)
	}
	cards := feed.Shelves[0].Cards
	if cards[0].Kind != FeedCardSong || cards[0].Track == nil || cards[0].Track.MID != "m42" {
		t.Errorf("home feed song = %+v", cards[0])
	}
	if cards[1].Kind != FeedCardPlaylist || cards[1].Playlist == nil || cards[1].Playlist.ID != 9001 || cards[1].Playlist.Cover() != "pc" || cards[1].Playlist.PlayCount != 2085000 {
		t.Errorf("home feed playlist = %+v", cards[1])
	}
	if cards[2].Kind != FeedCardVideo || cards[2].Video == nil || cards[2].Video.VID != "v1" || cards[2].Video.Cover != "mp" {
		t.Errorf("home feed video = %+v", cards[2])
	}
	if cards[3].Kind != FeedCardAlbum || cards[3].Album == nil || cards[3].Album.MID != "am" {
		t.Errorf("home feed album = %+v", cards[3])
	}
	if cards[4].Kind != FeedCardOther {
		t.Errorf("home feed navigation card = %+v", cards[4])
	}
	if p := params["get_recommend_feed"]; p["direction"] != float64(0) || p["page"] != float64(1) || p["s_num"] != float64(0) || len(p["v_cache"].([]any)) != 0 {
		t.Errorf("HomeFeed sent %v", p)
	}
	if comms["get_recommend_feed"]["ct"] != float64(20) {
		t.Errorf("HomeFeed used comm %v", comms["get_recommend_feed"])
	}

	tracks, err := GuessRecommend()
	if err != nil || len(tracks) != 1 || tracks[0].MID != "m9" {
		t.Fatalf("GuessRecommend = %+v, %v", tracks, err)
	}
	if p := params["get_radio_track"]; p["id"] != float64(99) || p["num"] != float64(5) || p["from"] != float64(0) || p["scene"] != float64(0) || len(p["song_ids"].([]any)) != 0 {
		t.Errorf("GuessRecommend sent %v", p)
	}

	radar, err := RadarRecommend(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(radar.Songs) != 1 || radar.Songs[0].MID != "m10" || len(radar.RecommendSongIDs) != 2 || radar.BaseSongIDs[0] != 3 ||
		!radar.HasMore || radar.Timestamp != 123 || radar.Toast != "T" || len(radar.VideoCards) == 0 {
		t.Fatalf("RadarRecommend = %+v", radar)
	}
	if p := params["GetRadarSong"]; p["Page"] != float64(2) || p["ReqType"] != float64(0) || len(p["FavSongs"].([]any)) != 0 || len(p["EntranceSongs"].([]any)) != 0 {
		t.Errorf("RadarRecommend sent %v", p)
	}

	fresh, err := RecommendNewSongs(1)
	if err != nil || len(fresh.Songs) != 1 || fresh.Songs[0].MID != "m11" || fresh.Lan != "内地" || fresh.Type != 1 || len(fresh.Tags) != 1 || fresh.Tags[0].TagID != 2 || len(fresh.LanList) != 1 {
		t.Fatalf("RecommendNewSongs = %+v, %v", fresh, err)
	}
	if p := params["get_new_song_info"]; p["type"] != float64(1) {
		t.Errorf("RecommendNewSongs sent %v", p)
	}
	if _, err := RecommendNewSongs(0); err != nil {
		t.Fatal(err)
	}
	if p := params["get_new_song_info"]; p["type"] != float64(5) {
		t.Errorf("RecommendNewSongs default sent %v", p)
	}

	lists, err := RecommendSonglists(1, 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists.Songlists) != 1 || lists.Songlists[0].ID != 88 || lists.Songlists[0].Name != "RL" || lists.Songlists[0].Cover() != "u" ||
		lists.Songlists[0].TrackCount != 3 || lists.Songlists[0].PlayCount != 100 || lists.Songlists[0].Creator.Nickname != "U" || !lists.HasMore || lists.From != 25 {
		t.Fatalf("RecommendSonglists = %+v", lists)
	}
	if p := params["GetRecommendFeed"]; p["From"] != float64(0) || p["Size"] != float64(25) {
		t.Errorf("RecommendSonglists sent %v", p)
	}
	if _, err := RecommendSonglists(2, 25); err != nil {
		t.Fatal(err)
	}
	if p := params["GetRecommendFeed"]; p["From"] != float64(25) {
		t.Errorf("RecommendSonglists page 2 sent %v", p)
	}
}
