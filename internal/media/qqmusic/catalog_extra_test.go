package qqmusic

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The singer, video, chart and album calls decode QQ's replies and send the
// fields' names as their pages spell them.
func TestCatalogExtras(t *testing.T) {
	params := make(map[string]map[string]any)
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
		params[request.Req.Method] = request.Req.Param
		singer := map[string]any{"singer_id": 1, "singer_mid": "s1", "singer_name": "S"}
		switch request.Req.Method {
		case "GetSingerList", "GetSingerListIndex":
			rpcReply(w, map[string]any{
				"singerlist": []any{singer}, "hotlist": []any{singer}, "total": 10,
				"tags": map[string]any{"area": []any{map[string]any{"id": 200, "name": "内地"}}},
			})
		case "GetSingerDetail":
			rpcReply(w, map[string]any{"singer_list": []any{map[string]any{
				"basic_info": map[string]any{"singer_id": 1, "singer_mid": "s1", "name": "S"},
				"ex_info":    map[string]any{"desc": "D", "area": "内地"},
				"wiki":       "W", "pic": map[string]any{"pic": "P"},
			}}})
		case "GetSimilarSingerList":
			rpcReply(w, map[string]any{"singerlist": []any{singer}})
		case "GetSingerSongList":
			rpcReply(w, map[string]any{"totalNum": 1, "songList": []any{map[string]any{
				"songInfo": map[string]any{"id": 9, "mid": "m9", "name": "N", "interval": 180},
			}}})
		case "GetAlbumList":
			rpcReply(w, map[string]any{"total": 1, "albumList": []any{map[string]any{
				"albumID": 2, "albumMid": "al", "albumName": "A", "totalNum": 10,
			}}})
		case "GetSingerMvTag":
			rpcReply(w, map[string]any{"list": []any{map[string]any{"id": 3, "name": "官方"}}})
		case "GetSingerMvList", "GetAllocMvInfo":
			rpcReply(w, map[string]any{"total": 1, "list": []any{map[string]any{
				"id": 77, "vid": "v1", "title": "MV", "picurl": "p", "playcnt": 9,
				"duration": 200, "pubdate": 1, "singers": []any{singer},
			}}})
		case "GetHomepageTabDetail":
			rpcReply(w, map[string]any{
				"TabID": "song_sing", "HasMore": 1,
				"SongTab":  map[string]any{"List": []any{map[string]any{"id": 9, "mid": "m9", "name": "N"}}},
				"AlbumTab": map[string]any{"AlbumList": []any{map[string]any{"albumID": 2, "albumMid": "al", "albumName": "A"}}},
				"VideoTab": map[string]any{"VideoList": []any{map[string]any{"id": 77, "vid": "v1", "title": "MV"}}},
			})
		case "get_video_info_batch":
			rpcReply(w, map[string]any{"v1": map[string]any{
				"vid": "v1", "sid": 77, "name": "MV", "cover_pic": "p", "duration": 200,
				"playcnt": 9, "pubdate": 1, "isfav": 1, "desc": "D", "uploader_nick": "U",
				"singers": []any{singer}, "related_songs": []int{1, 2},
			}})
		case "GetMvUrls":
			rpcReply(w, map[string]any{"v1": map[string]any{
				"mp4": []any{map[string]any{"url": []string{"http://a"}, "vkey": "k", "expire": 1}},
				"hls": []any{}, "duration": 10,
			}})
		case "GetAll":
			rpcReply(w, map[string]any{"group": []any{map[string]any{
				"groupId": 4, "groupName": "华语", "toplist": []any{map[string]any{
					"topId": 4, "title": "飙升", "song": []any{map[string]any{"rank": 1, "songId": 1, "title": "T", "singerName": "S"}},
				}},
			}}})
		case "GetDetail":
			rpcReply(w, map[string]any{
				"data":         map[string]any{"topId": 4, "title": "飙升", "totalNum": 1},
				"songInfoList": []any{map[string]any{"id": 9, "mid": "m9", "name": "N"}},
			})
		case "get_new_album_info":
			rpcReply(w, map[string]any{"total": 1, "albums": []any{map[string]any{
				"albumID": 2, "albumMid": "al", "albumName": "A", "singers": []any{singer},
			}}})
		case "GetAlbumDetail":
			rpcReply(w, map[string]any{
				"basicInfo": map[string]any{"albumID": 2, "albumMid": "al", "albumName": "A", "desc": "D", "language": 1, "genre": 2, "albumType": 3, "wikiurl": "W"},
				"company":   map[string]any{"name": "C"},
				"singer":    map[string]any{"singerList": []any{singer}},
			})
		case "GetAlbumSongList":
			rpcReply(w, map[string]any{"totalNum": 1, "songList": []any{map[string]any{
				"songInfo": map[string]any{"id": 9, "mid": "m9", "name": "N"},
			}}})
		default:
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
		}
	})

	category, err := SingersByCategory(-100, -100, -100)
	if err != nil || len(category.Singers) != 1 || len(category.Tags["area"]) != 1 {
		t.Fatalf("SingersByCategory = %+v, %v", category, err)
	}
	page, err := SingersByIndex(-100, -100, -100, -100, 2, 80)
	if err != nil || page.Total != 10 || len(page.Singers) != 1 {
		t.Fatalf("SingersByIndex = %+v, %v", page, err)
	}
	if _, ok := params["GetSingerListIndex"]["num"]; ok {
		t.Error("SingersByIndex sent num, which QQ's index does not take")
	}
	if params["GetSingerListIndex"]["sin"] != float64(80) {
		t.Errorf("SingersByIndex sent sin %v, want 80", params["GetSingerListIndex"]["sin"])
	}
	details, err := SingerDetails([]string{"s1"})
	if err != nil || len(details) != 1 || details[0].Description != "D" || len(details[0].Pics) != 1 {
		t.Fatalf("SingerDetails = %+v, %v", details, err)
	}
	similar, err := SimilarArtists("s1", 0)
	if err != nil || len(similar) != 1 || similar[0].MID != "s1" {
		t.Fatalf("SimilarArtists = %+v, %v", similar, err)
	}
	songs, total, err := SingerSongs("s1", 1, 1, 10)
	if err != nil || total != 1 || len(songs) != 1 || songs[0].MID != "m9" {
		t.Fatalf("SingerSongs = %+v, %d, %v", songs, total, err)
	}
	albums, total, err := SingerAlbums("s1", 1, 1, 10, -1)
	if err != nil || total != 1 || len(albums) != 1 || albums[0].Name != "A" {
		t.Fatalf("SingerAlbums = %+v, %d, %v", albums, total, err)
	}
	tags, err := SingerMVTags("s1")
	if err != nil || len(tags) != 1 || tags[0].Name != "官方" {
		t.Fatalf("SingerMVTags = %+v, %v", tags, err)
	}
	mvs, total, err := SingerMVs("s1", 1, 1, 10, -1)
	if err != nil || total != 1 || len(mvs) != 1 || mvs[0].ID != 77 || mvs[0].Duration != 200 {
		t.Fatalf("SingerMVs = %+v, %d, %v", mvs, total, err)
	}
	if params["GetSingerMvList"]["count"] != float64(10) {
		t.Errorf("SingerMVs sent count %v, want 10", params["GetSingerMvList"]["count"])
	}
	tab, err := SingerTab("s1", "song_sing", 1, 10, 0)
	if err != nil || !tab.HasMore || len(tab.Songs) != 1 || len(tab.Albums) != 1 || len(tab.Videos) != 1 {
		t.Fatalf("SingerTab = %+v, %v", tab, err)
	}
	if params["GetHomepageTabDetail"]["PageNum"] != float64(0) {
		t.Errorf("SingerTab sent PageNum %v, want 0", params["GetHomepageTabDetail"]["PageNum"])
	}
	video, err := MVDetails([]string{"v1"})
	if err != nil || video["v1"].Description != "D" || len(video["v1"].RelatedSongs) != 2 {
		t.Fatalf("MVDetails = %+v, %v", video, err)
	}
	links, err := MVLinks([]string{"v1"})
	if err != nil || links["v1"].MP4[0].URL != "http://a" || links["v1"].Duration != 10 {
		t.Fatalf("MVLinks = %+v, %v", links, err)
	}
	if params["GetMvUrls"]["vids"] == nil || params["GetMvUrls"]["request_type"] != float64(10003) {
		t.Errorf("MVLinks sent %v", params["GetMvUrls"])
	}
	all, total, err := MVList(15, 7, 0, 1, 10)
	if err != nil || total != 1 || len(all) != 1 || all[0].VID != "v1" {
		t.Fatalf("MVList = %+v, %d, %v", all, total, err)
	}
	charts, err := TopCategories()
	if err != nil || len(charts) != 1 || len(charts[0].Lists) != 1 || charts[0].Lists[0].Songs[0].Name != "T" {
		t.Fatalf("TopCategories = %+v, %v", charts, err)
	}
	chart, err := TopListDetail(4, 1, 10, true)
	if err != nil || chart.Info.Title != "飙升" || len(chart.Songs) != 1 || chart.Songs[0].MID != "m9" {
		t.Fatalf("TopListDetail = %+v, %v", chart, err)
	}
	if params["GetDetail"]["withTags"] != true {
		t.Errorf("TopListDetail sent withTags %v, want true", params["GetDetail"]["withTags"])
	}
	fresh, total, err := NewAlbums(1, 1, 20)
	if err != nil || total != 1 || len(fresh) != 1 || fresh[0].Name != "A" || len(fresh[0].Artists) != 1 {
		t.Fatalf("NewAlbums = %+v, %d, %v", fresh, total, err)
	}
	album, err := AlbumInfoByMID("al")
	if err != nil || album.Description != "D" || album.Company != "C" || len(album.Artists) != 1 {
		t.Fatalf("AlbumInfoByMID = %+v, %v", album, err)
	}
	if _, ok := params["GetAlbumDetail"]["albumMId"]; !ok {
		t.Error("AlbumInfoByMID did not send albumMId")
	}
	albumSongs, total, err := AlbumSongs("al", 1, 10)
	if err != nil || total != 1 || len(albumSongs) != 1 {
		t.Fatalf("AlbumSongs = %+v, %d, %v", albumSongs, total, err)
	}
}
